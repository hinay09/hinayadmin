// Package middleware HTTP 中间件集合。
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/guid"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/jwtx"
	"hinay.cn/admin/utility/xerror"
)

// corsAllowOrigins 允许跨域的来源白名单, 从配置 cors.allowOrigins 读取。
// 不配置则不下发任何 CORS 头(仅同源可用); 不支持 "*" 通配, 不允许携带 Cookie 凭据。
var (
	corsAllowOrigins []string
	corsOnce         sync.Once
)

// loadCorsConfig 懒加载 CORS 白名单配置(仅加载一次)。
func loadCorsConfig(ctx context.Context) {
	corsOnce.Do(func() {
		corsAllowOrigins = []string{}
		if v, err := g.Cfg().Get(ctx, "cors.allowOrigins"); err == nil && v != nil {
			if origins := v.Strings(); len(origins) > 0 {
				corsAllowOrigins = origins
			}
		}
	})
}

// CORS 跨域中间件: 基于显式来源白名单, 拒绝反射任意 Origin。
func CORS(r *ghttp.Request) {
	loadCorsConfig(r.Context())
	origin := r.Header.Get("Origin")
	if origin != "" {
		for _, allowed := range corsAllowOrigins {
			if origin == allowed {
				h := r.Response.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
				h.Set("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization,X-Requested-With,X-Request-Id")
				h.Set("Access-Control-Max-Age", "86400")
				// 认证走 Authorization 请求头而非 Cookie, 因此不设置 Allow-Credentials
				break
			}
		}
	}
	// 直接响应预检请求, 避免落到路由层 404
	if r.Method == "OPTIONS" {
		r.Response.WriteStatus(http.StatusNoContent)
		r.ExitAll()
		return
	}
	r.Middleware.Next()
}

// SecurityHeaders 为所有 API 响应附加基础安全响应头。
func SecurityHeaders(r *ghttp.Request) {
	h := r.Response.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "no-referrer")
	r.Middleware.Next()
}

// RequestId 注入 RequestId 到 ctx 与响应头, 便于链路追踪。
func RequestId(r *ghttp.Request) {
	id := r.Header.Get("X-Request-Id")
	if id == "" {
		id = guid.S()
	}
	// 双写: ghttp request 级 + 标准 context.WithValue
	r.SetCtxVar(consts.CtxRequestIdName, id)
	r.SetCtx(context.WithValue(r.Context(), consts.CtxRequestIdKey, id))
	r.Response.Header().Set("X-Request-Id", id)
	start := time.Now()
	r.Middleware.Next()
	g.Log().Infof(r.Context(),
		"[HTTP] %s %s %d %s",
		r.Method, r.URL.Path, r.Response.Status, time.Since(start),
	)
}

// Auth 解析 JWT, 注入 LoginUser 到 ctx; 校验 Redis 黑名单。
//
// 并发安全说明:
//   - 每个 HTTP 请求都拥有独立的 *ghttp.Request 与 ctx, 此处创建的 user 是请求作用域内的局部变量,
//     不会被其他请求 goroutine 看到, 也不存在跨请求交叉污染。
//   - 同时写入 ghttp request 级与标准 context.WithValue, 使 ctx 在传递到后端 goroutine 后仍可提取。
func Auth(r *ghttp.Request) {
	// 公开接口: 登录不需 token, 直接放行。
	if _, ok := publicPaths[r.URL.Path]; ok {
		r.Middleware.Next()
		return
	}
	ctx := r.Context()
	header := r.Header.Get(consts.AuthHeader)
	if header == "" || !strings.HasPrefix(header, consts.AuthScheme) {
		writeUnauthorized(r)
		return
	}
	tokenStr := strings.TrimPrefix(header, consts.AuthScheme)

	// 超长 token 直接拒绝, 规避解析放大攻击(CVE-2025-30204 类)与异常内存占用
	if len(tokenStr) > 8192 {
		writeUnauthorized(r)
		return
	}

	// Redis 黑名单 (登出后 token 立即失效)。
	// fail-close: Redis 异常时拒绝请求而非放行, 避免已登出的 token 在故障期间继续生效。
	exists, err := g.Redis().Exists(ctx, consts.JWTBlacklistPrefix+tokenStr)
	if err != nil {
		g.Log().Errorf(ctx, "jwt blacklist check failed (redis unavailable): %v", err)
		writeUnauthorizedWithMessage(r, "服务暂时不可用, 请稍后重试")
		return
	}
	if exists > 0 {
		writeUnauthorized(r)
		return
	}

	claims, err := jwtx.Parse(ctx, tokenStr)
	if err != nil {
		writeUnauthorized(r)
		return
	}
	// 请求作用域内独立创建的指针, 不与其他 goroutine 共享。
	user := &model.LoginUser{
		UserId:   claims.UserId,
		Username: claims.Username,
	}
	// 双写: ghttp request 级 (兼容现有逻辑) + 类型安全 context.WithValue (极推荐)
	r.SetCtxVar(consts.CtxUserKeyName, user)
	r.SetCtxVar(consts.CtxJwtTokenKeyName, tokenStr)
	newCtx := context.WithValue(ctx, consts.CtxUserKey, user)
	newCtx = context.WithValue(newCtx, consts.CtxJwtTokenKey, tokenStr)
	r.SetCtx(newCtx)
	r.Middleware.Next()
}

// publicPaths 公开接口白名单: Auth 与 Casbin 中间件均跳过。
var publicPaths = map[string]struct{}{
	"/api/v1/auth/login":      {},
	"/api/v1/auth/public-key": {},
}

// authWhitelist 基础会话类接口白名单。
// 这些接口仅返回"当前已登录用户自身"的会话信息(菜单/资料/登出),
// 任何已登录用户都应能访问, 不再进入 Casbin RBAC 策略校验,
// 避免因为策略遗漏 (如某用户没有绑定任何角色) 而被锁死, 导致登录后
// 拿不到菜单/用户信息从而被前端 reset() 踢回登录页。
var authWhitelist = map[string]struct{}{
	"/api/v1/auth/menus":    {},
	"/api/v1/auth/userInfo": {},
	"/api/v1/auth/logout":   {},
	"/api/v1/auth/profile":  {},
	"/api/v1/auth/password": {},
	"/api/v1/auth/refresh":  {},
	"/api/v1/auth/avatar":   {},
	// 全局配置读取(仅启用项): 前端布局(站点名/Logo/页脚版权)对任何已登录用户展示,
	// 若走逐角色 Casbin 授权, 未配置该 API 策略的普通用户会在进布局时 403。
	// 放入白名单 = 登录即可读; 匿名访问仍被 Auth 中间件拦截。
	"/api/v1/system/configs/all": {},
}

// isAuthWhitelisted 判断给定路径是否命中已登录用户白名单。
// 注意: 只做显式路径枚举, 不做前缀匹配, 避免未来新增敏感接口被自动绕过 RBAC。
func isAuthWhitelisted(path string) bool {
	_, ok := authWhitelist[path]
	return ok
}

// Casbin 基于用户 + 路径 + 方法做策略校验。
// 新 model 使用 g(r.sub, p.sub) 自动解析角色, sub 直接传 username。
func Casbin(r *ghttp.Request) {
	// 公开接口: 登录直接放行。
	if _, ok := publicPaths[r.URL.Path]; ok {
		r.Middleware.Next()
		return
	}
	ctx := r.Context()
	user := contextx.LoginUser(ctx)
	if user == nil {
		writeUnauthorized(r)
		return
	}
	// 超管全放行
	if contextx.IsAdmin(ctx) {
		r.Middleware.Next()
		return
	}
	path := r.URL.Path
	method := r.Method
	// 基础会话类接口白名单: 已登录即可访问, 不进入 Casbin 校验
	if isAuthWhitelisted(path) {
		r.Middleware.Next()
		return
	}
	ok, err := casbinx.Enforce(ctx, user.Username, path, method)
	if err != nil {
		g.Log().Errorf(ctx, "casbin enforce error: %v", err)
	}
	if !ok {
		r.Response.WriteStatus(http.StatusForbidden, g.Map{
			"code":    xerror.CodeForbidden.Code(),
			"message": xerror.CodeForbidden.Message(),
			"data":    nil,
		})
		r.ExitAll()
		return
	}
	r.Middleware.Next()
}

func writeUnauthorized(r *ghttp.Request) {
	writeUnauthorizedWithMessage(r, xerror.CodeUnauthorized.Message())
}

func writeUnauthorizedWithMessage(r *ghttp.Request, message string) {
	r.Response.WriteStatus(http.StatusUnauthorized, g.Map{
		"code":    xerror.CodeUnauthorized.Code(),
		"message": message,
		"data":    nil,
	})
	r.ExitAll()
}

// OperationLog 操作日志中间件: 自动记录写操作(POST/PUT/DELETE)到操作日志。
//
// 挂载位置要求: 必须注册在 ghttp.MiddlewareHandlerResponse 之外层(先注册),
// 使 defer 阶段能读到最终响应(业务码/消息/HTTP状态)。
// 每条记录包含: 操作人、动作、资源、HTTP方法与路径、成功/失败结果与原因、
// 耗时、RequestId、脱敏后的请求体、IP 与 UA。
//
// 覆盖范围:
//   - 成功与失败的业务操作(业务码 != 0 也会记录, 含失败原因)
//   - 401/403 等经 ExitAll 中断的请求(通过 defer 在 panic 展开时兜底记录)
//   - 登录/登出/续签事件(action=login/logout/refresh), 含登录失败的尝试账号
func OperationLog(r *ghttp.Request) {
	// 仅记录写操作
	method := r.Method
	if method != "POST" && method != "PUT" && method != "DELETE" {
		r.Middleware.Next()
		return
	}

	path := r.URL.Path
	clientIp := r.GetClientIp()
	userAgent := r.Header.Get("User-Agent")
	requestId := r.GetCtxVar(consts.CtxRequestIdName, "").String()
	start := time.Now()

	// 请求体在 pre 阶段读取并脱敏(截断保护, 最多 2048 字节)
	// 跳过 multipart/form-data(文件上传), 避免触发 body too large
	detail := ""
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "multipart/") {
		if body := r.GetBodyString(); body != "" {
			detail = sanitizeLogBody(body, 2048)
		}
	}

	defer func() {
		// defer 在 ExitAll panic 展开时依然执行, 保证被拒绝的请求也有审计记录
		var userId uint64
		username := ""
		if user := contextx.LoginUser(r.Context()); user != nil {
			userId, username = user.UserId, user.Username
		}
		// 未认证场景(登录失败/无效token)尝试从请求体提取账号, 便于排查暴力尝试
		if username == "" && detail != "" {
			username = jsonField(detail, "username")
		}

		resource, action, resourceId := parseOperation(path, method)
		switch path {
		case "/api/v1/auth/login":
			resource, action = "auth", "login"
		case "/api/v1/auth/logout":
			resource, action = "auth", "logout"
		case "/api/v1/auth/refresh":
			resource, action = "auth", "refresh"
		}

		statusCode := r.Response.Status
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		code, message := parseResponseResult(r.Response.Buffer())
		durationMs := time.Since(start).Milliseconds()

		// 异步写入数据库: 使用全新的 background context, 不依赖已结束的请求 ctx
		go func() {
			ctx := context.Background()
			_, err := dao.SysAuditLog.Ctx(ctx).Data(g.Map{
				"user_id":     userId,
				"username":    username,
				"action":      action,
				"resource":    resource,
				"resource_id": resourceId,
				"method":      method,
				"path":        path,
				"status_code": statusCode,
				"code":        code,
				"message":     truncateForColumn(message, 512),
				"duration_ms": durationMs,
				"request_id":  requestId,
				"detail":      detail,
				"ip":          clientIp,
				"user_agent":  truncateForColumn(userAgent, 512),
			}).Insert()
			if err != nil {
				g.Log().Warningf(ctx, "OperationLog insert failed: %v", err)
			}
		}()
	}()

	r.Middleware.Next()
}

// parseResponseResult 从响应体解析业务码与消息({code,message,...} 结构)。
func parseResponseResult(buf []byte) (code int, message string) {
	if len(buf) == 0 {
		return 0, ""
	}
	var res struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(buf, &res); err != nil {
		return 0, ""
	}
	return res.Code, res.Message
}

// jsonField 从 JSON 字符串中提取指定字符串字段值, 解析失败返回空。
func jsonField(body, field string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return ""
	}
	if v, ok := m[field].(string); ok {
		return truncateForColumn(v, 64)
	}
	return ""
}

// truncateForColumn 按 rune 截断到 max 字符, 避免超出列宽导致写入失败。
func truncateForColumn(s string, max int) string {
	if len(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

// sensitiveKeyPattern 审计日志中需要脱敏的字段名(不区分大小写)。
var sensitiveKeyPattern = regexp.MustCompile(`(?i)password|passwd|secret|token|authorization`)

// sanitizeLogBody 对请求体做敏感字段脱敏后返回, 超过 maxLen 截断。
// 仅接受可解析的 JSON 对象/数组; 解析失败(可能含非 JSON 敏感数据)时保守地不记录。
func sanitizeLogBody(body string, maxLen int) string {
	var payload interface{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return ""
	}
	redactSensitiveValue(payload)
	out, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	if len(out) > maxLen {
		return string(out[:maxLen]) + "...(truncated)"
	}
	return string(out)
}

// redactSensitiveValue 递归遍历已解析的 JSON 数据, 将敏感 key 的值替换为掩码。
func redactSensitiveValue(node interface{}) {
	switch v := node.(type) {
	case map[string]interface{}:
		for k, val := range v {
			if sensitiveKeyPattern.MatchString(k) {
				v[k] = "***"
			} else {
				redactSensitiveValue(val)
			}
		}
	case []interface{}:
		for _, item := range v {
			redactSensitiveValue(item)
		}
	}
}

// parseOperation 从 URL path + HTTP method 解析资源名与操作类型。
func parseOperation(path, method string) (resource, action, resourceId string) {
	// 去掉 /api/v1 前缀
	p := strings.TrimPrefix(path, "/api/v1")
	p = strings.TrimLeft(p, "/")

	parts := strings.Split(p, "/")
	if len(parts) == 0 {
		return "", "", ""
	}

	switch method {
	case "POST":
		action = "create"
	case "PUT":
		action = "update"
	case "DELETE":
		action = "delete"
	}

	// 资源名: 第一段
	if len(parts) > 0 {
		resource = parts[0]
	}

	// 特殊处理: URL 末段为 "upload" 时, 操作类型覆盖为 upload
	if len(parts) > 0 && parts[len(parts)-1] == "upload" && method == "POST" {
		action = "upload"
	}

	// 尝试提取 ID
	// /system/users/123  -> resourceId=123
	// /system/users/123/password -> resourceId=123
	if len(parts) >= 2 {
		// 最后一段或倒数第二段可能是 ID
		for i, part := range parts {
			if isNumeric(part) {
				resourceId = part
				// 如果 ID 后面还有路径段, 细化 action 描述
				if i < len(parts)-1 && parts[i+1] != "" {
					action = action + "_" + parts[i+1]
				}
				break
			}
		}
	}

	if resource == "" {
		resource = parts[0]
	}

	return
}

// isNumeric 判断字符串是否为纯数字。
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
