// Package middleware HTTP 中间件集合。
package middleware

import (
	"context"
	"net/http"
	"strings"
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

// CORS 简单跨域中间件, 开发环境与前端联调使用。
func CORS(r *ghttp.Request) {
	cors := r.Response.DefaultCORSOptions()
	cors.AllowDomain = []string{"*"}
	cors.AllowMethods = "GET,POST,PUT,DELETE,OPTIONS,PATCH"
	cors.AllowHeaders = "Origin,Content-Type,Accept,Authorization,X-Requested-With"
	r.Response.CORS(cors)
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

	// Redis 黑名单 (登出后 token 立即失效)
	exists, _ := g.Redis().Exists(ctx, consts.JWTBlacklistPrefix+tokenStr)
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
	"/api/v1/auth/login": {},
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
}

// authWhitelistPrefixes 路径前缀白名单, 用于覆盖 dashboard 等
// 一类"模块根路径下所有子路径"的场景。
var authWhitelistPrefixes = []string{
	"/api/v1/dashboard",
}

// isAuthWhitelisted 判断给定路径是否命中已登录用户白名单。
func isAuthWhitelisted(path string) bool {
	if _, ok := authWhitelist[path]; ok {
		return true
	}
	for _, p := range authWhitelistPrefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
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
	r.Response.WriteStatus(http.StatusUnauthorized, g.Map{
		"code":    xerror.CodeUnauthorized.Code(),
		"message": xerror.CodeUnauthorized.Message(),
		"data":    nil,
	})
	r.ExitAll()
}

// OperationLog 操作日志中间件: 自动记录 POST/PUT/DELETE 请求到操作日志。
// 放置在 Auth + Casbin 中间件之后, 确保只记录合法请求。
func OperationLog(r *ghttp.Request) {
	// 仅记录写操作
	method := r.Method
	if method != "POST" && method != "PUT" && method != "DELETE" {
		r.Middleware.Next()
		return
	}

	// 跳过公开接口(登录等)
	if _, ok := publicPaths[r.URL.Path]; ok {
		r.Middleware.Next()
		return
	}

	// ---- 在 Next() 之前提取所有 goroutine 所需数据 ----
	// 避免 goroutine 使用请求结束后已取消的 ctx
	user := contextx.LoginUser(r.Context())
	if user == nil {
		r.Middleware.Next()
		return
	}

	userId := user.UserId
	username := user.Username
	clientIp := r.GetClientIp()
	userAgent := r.Header.Get("User-Agent")
	path := r.URL.Path

	// 尝试读取请求体作为 detail(截断保护, 最多 2048 字节)
	// 跳过 multipart/form-data(文件上传), 避免触发 body too large
	detail := ""
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "multipart/") {
		if body := r.GetBodyString(); body != "" {
			if len(body) > 2048 {
				body = body[:2048] + "...(truncated)"
			}
			detail = body
		}
	}

	r.Middleware.Next()

	// 仅当响应成功时记录
	if r.Response.Status >= 400 {
		return
	}

	// 解析资源和操作
	resource, action, resourceId := parseOperation(path, method)

	// 异步写入数据库: 使用全新的 background context, 不依赖已结束的请求 ctx
	go func() {
		ctx := context.Background()
		_, err := dao.SysAuditLog.Ctx(ctx).Data(g.Map{
			"user_id":     userId,
			"username":    username,
			"action":      action,
			"resource":    resource,
			"resource_id": resourceId,
			"detail":      detail,
			"ip":          clientIp,
			"user_agent":  userAgent,
		}).Insert()
		if err != nil {
			g.Log().Warningf(ctx, "OperationLog insert failed: %v", err)
		}
	}()
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
