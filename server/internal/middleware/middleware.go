// Package middleware HTTP 中间件集合。
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/guid"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/logic/pwdpolicy"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/demox"
	"hinay.cn/admin/utility/jwtx"
	"hinay.cn/admin/utility/tenantx"
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

// Auth 解析 JWT, 注入 LoginUser 与租户上下文到 ctx; 校验 Redis 黑名单、
// 账号状态、所属租户状态 (禁用/到期, docs/multi-tenant.md §2.1)。
//
// 多租户组装 (multiTenant.enable 开启时):
//   - claims.tid 为所属租户 (老 token 无该 claim 回填平台租户 1);
//   - 平台超管 (role id=1, 仅平台租户可授予) 默认不注入 (跨租户可见);
//   - 平台超管携带 X-Tenant-Id 头 → 切换注入目标租户 (代运维);
//   - 非平台超管携带 X-Tenant-Id → 忽略并写告警审计 (§12);
//   - 租户上下文写 CtxTenantKey (与 CtxUserKey 同层双写), tenantorm 据此注入。
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

	// 超长 token 直接拒绝, 规避解析放大攻击(CVE-2025-30204类)与异常内存占用
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

	// 后端兜底 (纵深防御, 不依赖前端跳转):
	//   1) 账号已禁用/删除 -> token 立即失效 (登录/续签之外的首道实时校验);
	//   2) 租户已禁用/到期 -> 403 拦截 (含会话类接口, "已登录会话全部拦截");
	//   3) 强制改密 (管理员创建/重置/导入, 或密码已过期) -> 仅放行会话类白名单接口,
	//      其余业务 API 一律拒绝 (HTTP 428/业务码 42800), 直至完成改密。
	// 会话类白名单(authWhitelist)与本中间件同文件, 含改密/用户信息/登出等。
	_, isSessionPath := authWhitelist[r.URL.Path]
	state, ok := userAccessState(newCtx, claims.UserId)
	if !ok {
		writeUnauthorized(r)
		return
	}
	// 回填昵称: JWT 只携带 UserId/Username, 业务侧 (消息发送人/审批人
	// 显示名等) 取 LoginUser.Nickname 时不再回退登录名;
	// userAccessState 本就逐请求查 sys_user, 顺带取回零额外开销。
	user.Nickname = state.nickname
	if state.tenantBlocked {
		writeBlockedWithCode(r, state.tenantCode)
		return
	}
	if !isSessionPath && state.pwdBlocked {
		writeBlockedWithCode(r, xerror.CodePwdMustChange)
		return
	}

	// ---- 多租户上下文组装 (docs/multi-tenant.md §4.1) ----
	var tenantCtx context.Context
	if tenantx.Enabled(newCtx) {
		tid := claims.TenantId
		if tid == 0 {
			tid = consts.TenantPlatformId // 老 token 兼容: 回填平台租户
		}
		user.TenantId = tid
		user.TenantCode = state.tenantCodeStr
		user.TenantName = state.tenantName

		// 先以"平台视角"(不注入)计算归属域角色: 平台超管身份判定
		homeCtx := contextx.WithTenant(newCtx, &model.TenantInfo{Id: tid, Code: state.tenantCodeStr, Inject: false})
		roles, rerr := casbinx.GetUserRoles(homeCtx, claims.UserId)
		isPlatform := rerr == nil && slices.Contains(roles, consts.RoleAdminId)
		user.IsPlatformAdmin = isPlatform
		if !isPlatform && rerr == nil && len(roles) > 0 {
			user.IsTenantAdmin = hasTenantAdminRole(homeCtx, roles)
		}

		// 平台超管代运维: X-Tenant-Id 切换注入目标租户 (仅 platform:admin 可携带)
		effective := &model.TenantInfo{Id: tid, Code: state.tenantCodeStr, Name: state.tenantName, Inject: !isPlatform}
		if switchHeader := r.Header.Get(consts.HeaderTenantId); switchHeader != "" {
			if !isPlatform {
				// 非平台超管携带: 忽略并写告警审计 (§12)
				auditTenantHeaderSpoof(newCtx, user, tid, switchHeader)
				g.Log().Warningf(newCtx, "[tenant] non-platform user %d carried %s=%s, ignored", user.UserId, consts.HeaderTenantId, switchHeader)
			} else if target, perr := strconv.ParseUint(switchHeader, 10, 64); perr != nil {
				writeBlockedWithCode(r, xerror.CodeParamInvalid)
				return
			} else if target != tid {
				if target == consts.TenantPlatformId {
					effective = &model.TenantInfo{Id: consts.TenantPlatformId, Code: consts.TenantPlatformCode, Name: state.tenantName, Inject: false}
					user.SwitchTenantInfo = nil
				} else {
					tref, terr := loadTenantForSwitch(newCtx, target)
					if terr != nil {
						writeBlockedWithCode(r, xerror.CodeNotFound)
						return
					}
					effective = &model.TenantInfo{Id: tref.Id, Code: tref.Code, Name: tref.Name, Inject: true}
					user.SwitchTenantInfo = tref
				}
			}
		}
		tenantCtx = contextx.WithTenant(newCtx, effective)
		// 日志归因: glog ctxKeys 按 "ctxTenant" 输出当前数据面租户 ID
		r.SetCtxVar(consts.CtxTenantName, effective.Id)
	} else {
		// 开关关闭: 单租户兼容 —— 平台视角 (TenantId(ctx)=1 但不注入)。
		// 平台超管判定仍需计算 (Casbin 中间件旁路依赖; admin 不种 p 行, 靠判定短路)。
		roles, rerr := casbinx.GetUserRoles(newCtx, claims.UserId)
		if rerr == nil {
			user.IsPlatformAdmin = slices.Contains(roles, consts.RoleAdminId)
		}
		tenantCtx = contextx.WithTenant(newCtx, &model.TenantInfo{
			Id: consts.TenantPlatformId, Code: consts.TenantPlatformCode, Name: "平台租户", Inject: false,
		})
	}
	r.SetCtx(tenantCtx)

	// 在线会话心跳: 异步 + 节流 (logic/online), 尽力而为, 不影响请求
	go service.Online().Touch(context.Background(), tokenStr)

	r.Middleware.Next()
}

// accessState 每请求一次的用户+租户访问状态 (join sys_tenant, §2.1 校验并入
// 现有 per-request 查询, 不新增 DB 往返)。
type accessState struct {
	nickname      string
	pwdBlocked    bool
	tenantBlocked bool
	tenantCode    gcode.Code // 拦截业务码 (租户禁用/到期)
	tenantCodeStr string     // 租户 code (展示冗余)
	tenantName    string
}

// userAccessState 查询用户当前访问状态 (每个已登录请求一次主键查询)。
// 第二返回值 false = 账号级拦截 (401)。
func userAccessState(ctx context.Context, userId uint64) (accessState, bool) {
	var st accessState
	row, err := dao.SysUser.Ctx(contextx.WithoutTenant(ctx)).As("u").
		LeftJoin("sys_tenant t", "t.id = u.tenant_id").
		Fields("u.id, u.status, u.must_change_pwd, u.pwd_updated_at, u.deleted_at, u.nickname, u.tenant_id, t.code AS tcode, t.name AS tname, t.status AS tstatus, t.expire_at AS texpire, t.deleted_at AS tdeleted").
		Where("u.id", userId).
		One()
	if err != nil {
		// 查询失败 fail-close: 拒绝而非放行
		return st, false
	}
	if row.IsEmpty() || !row["deleted_at"].IsNil() || row["status"].Int() != consts.StatusEnabled {
		// 账号已删除/禁用: token 视为失效
		return st, false
	}
	st.nickname = row["nickname"].String()
	st.tenantCodeStr = row["tcode"].String()
	st.tenantName = row["tname"].String()

	// 租户状态校验 (多租户开启时; 平台租户行恒在且不受运营操作禁用)
	if tenantx.Enabled(ctx) {
		if row["tcode"].IsNil() || !row["tdeleted"].IsNil() {
			// 租户行缺失 (升级脚本未执行/租户被物理清理): fail-close
			st.tenantBlocked, st.tenantCode = true, xerror.CodeTenantDisabled
			return st, true
		}
		if row["tstatus"].Int() != consts.StatusEnabled {
			st.tenantBlocked, st.tenantCode = true, xerror.CodeTenantDisabled
			return st, true
		}
		if !row["texpire"].IsNil() && !row["texpire"].GTime().IsZero() && row["texpire"].GTime().Before(gtime.Now()) {
			st.tenantBlocked, st.tenantCode = true, xerror.CodeTenantExpired
			return st, true
		}
	}

	// 演示模式放行强制改密拦截: 改密接口已被 demox.Guard 禁止,
	// 若仍按 428 锁定到改密页, 用户将被困在永远无法完成的改密流程 (前后端死锁)。
	st.pwdBlocked = !demox.Enabled(ctx) &&
		(row["must_change_pwd"].Int() == 1 || pwdpolicy.Expired(ctx, row["pwd_updated_at"].GTime()))
	return st, true
}

// hasTenantAdminRole 判定角色列表中是否含 code=tenant_admin 的角色 (租户管理员)。
// 以平台视角查询 (角色 ID 已来自归属域 g 行, 跨域角色 ID 天然不可能出现)。
func hasTenantAdminRole(ctx context.Context, roleIds []uint64) bool {
	if len(roleIds) == 0 {
		return false
	}
	n, err := dao.SysRole.Ctx(contextx.WithoutTenant(ctx)).
		WhereIn("id", roleIds).
		Where("code", consts.RoleTenantAdminCode).
		Where("deleted_at IS NULL").
		Count()
	return err == nil && n > 0
}

// loadTenantForSwitch 平台超管切换目标租户时装载租户行 (平台视角查询, 不校验
// 状态 —— 运维需要能进入已禁用租户排查)。
func loadTenantForSwitch(ctx context.Context, tenantId uint64) (*model.TenantRef, error) {
	row, err := dao.SysTenant.Ctx(contextx.WithoutTenant(ctx)).
		Fields("id, code, name").
		Where("id", tenantId).
		Where("deleted_at IS NULL").
		One()
	if err != nil {
		return nil, err
	}
	if row.IsEmpty() {
		return nil, gerror.New("tenant not found")
	}
	return &model.TenantRef{
		Id:   row["id"].Uint64(),
		Code: row["code"].String(),
		Name: row["name"].String(),
	}, nil
}

// auditTenantHeaderSpoof 非平台超管伪造 X-Tenant-Id 的告警审计 (§7 平台审计,
// 异步落 sys_audit_log, 租户归属=操作者所属租户, detail 记录伪造值)。
func auditTenantHeaderSpoof(ctx context.Context, user *model.LoginUser, homeTid uint64, headerVal string) {
	ctx = contextx.WithoutTenant(context.WithoutCancel(ctx))
	go func() {
		_, err := dao.SysAuditLog.Ctx(ctx).Data(g.Map{
			"tenant_id":   homeTid,
			"user_id":     user.UserId,
			"username":    user.Username,
			"action":      "tenant_header_spoof",
			"resource":    "tenant",
			"resource_id": "",
			"method":      "",
			"path":        "",
			"status_code": 0,
			"code":        0,
			"message":     "非平台超管携带 X-Tenant-Id 头, 已忽略",
			"detail":      truncateForColumn("forged X-Tenant-Id: "+headerVal, 512),
			"ip":          "",
			"user_agent":  "",
		}).Insert()
		if err != nil {
			g.Log().Warningf(ctx, "tenant header spoof audit insert failed: %v", err)
		}
	}()
}

// publicPaths 公开接口白名单: Auth 与 Casbin 中间件均跳过。
var publicPaths = map[string]struct{}{
	"/api/v1/auth/login":           {},
	"/api/v1/auth/public-key":      {},
	"/api/v1/auth/captcha":         {},
	"/api/v1/auth/login/totp":      {},
	"/api/v1/auth/register":        {},
	"/api/v1/auth/register/status": {},
	// 公开租户信息 (登录页租户名提示条): 仅返回租户显示名, 不泄露其他信息,
	// 按 IP 限流防租户码枚举 (docs/multi-tenant.md §4.3)。
	"/api/v1/auth/tenant/info": {},
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
	// 两步验证自助管理: 仅操作"当前登录用户自身"的绑定, 登录即可访问
	"/api/v1/auth/totp/setup":   {},
	"/api/v1/auth/totp/enable":  {},
	"/api/v1/auth/totp/disable": {},
	// 全局配置读取(仅启用项): 前端布局(站点名/Logo/页脚版权)对任何已登录用户展示,
	// 若走逐角色 Casbin 授权, 未配置该 API 策略的普通用户会在进布局时 403。
	// 放入白名单 = 登录即可读; 匿名访问仍被 Auth 中间件拦截。
	"/api/v1/system/configs/all": {},
	// 字典全量(仅启用项): 前端字典数据全站消费, 而字典管理菜单(14)不进任何租户
	// 基础授权包 —— 与 configs/all 同理同法 (docs/multi-tenant.md §7.1 支撑 API 段)。
	"/api/v1/system/dicts/all": {},
	// 消息事件流: 会话级实时通知, 登录即可订阅 (SSE 长连接)
	"/api/v1/message/events": {},
}

// isAuthWhitelisted 判断给定路径是否命中已登录用户白名单。
// 注意: 只做显式路径枚举, 不做前缀匹配, 避免未来新增敏感接口被自动绕过 RBAC。
func isAuthWhitelisted(path string) bool {
	_, ok := authWhitelist[path]
	return ok
}

// Casbin 基于用户ID + 路径 + 方法做策略校验。
// g 策略以 用户ID/角色ID 关联, sub 传用户ID, 用户名/角色code 变更不影响匹配。
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
	// 平台超管全放行 (docs/multi-tenant.md §2.2: 判定从 IsAdmin 换 IsPlatformAdmin;
	// 租户管理员不旁路 Casbin —— 他们的权限由域内角色策略决定)
	if contextx.IsPlatformAdmin(ctx) {
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
	ok, err := casbinx.Enforce(ctx, user.UserId, path, method)
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

// httpStatusOf 拦截业务码对应的 HTTP 状态码:
// 强制改密走 428 Precondition Required, 与通用 403 无权限区分,
// 避免前端 (error.vue / showError) 或中间层任何针对 403 的通用处理误伤。
func httpStatusOf(code gcode.Code) int {
	if code == xerror.CodePwdMustChange {
		return http.StatusPreconditionRequired
	}
	return http.StatusForbidden
}

func writeBlockedWithCode(r *ghttp.Request, code gcode.Code) {
	r.Response.WriteStatus(httpStatusOf(code), g.Map{
		"code":    code.Code(),
		"message": code.Message(),
		"data":    nil,
	})
	r.ExitAll()
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
	// multipart(文件上传)不读原始请求体(避免二进制入库),
	// 审计记录照常产生, detail 由上传逻辑经 ctx 暂存的文件摘要填充(见 defer)。
	detail := ""
	ct := r.Header.Get("Content-Type")
	isMultipart := strings.HasPrefix(ct, "multipart/")
	if !isMultipart {
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
		case "/api/v1/auth/login/totp":
			resource, action = "auth", "login_totp"
		case "/api/v1/auth/totp/enable":
			resource, action = "auth", "totp_enable"
		case "/api/v1/auth/totp/disable":
			resource, action = "auth", "totp_disable"
		}

		// multipart: detail 用上传逻辑暂存的文件摘要 (无摘要时给占位说明, 记录本身不缺)
		if isMultipart {
			if summary := r.GetCtxVar(consts.CtxAuditUploadName, "").String(); summary != "" {
				detail = truncateForColumn(summary, 512)
			} else {
				detail = "(multipart 文件上传, 无文件摘要)"
			}
		}

		statusCode := r.Response.Status
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		code, message := parseResponseResult(r.Response.Buffer())
		durationMs := time.Since(start).Milliseconds()

		// 异步写入数据库: 保留请求上下文租户信息 (WithoutCancel 去取消传播,
		// sys_audit_log 为 A 类隔离表, 无租户上下文写入会被 fail-close 拒绝)
		go func() {
			ctx := context.WithoutCancel(r.Context())
			_, err := dao.SysAuditLog.Ctx(ctx).Data(g.Map{
				"tenant_id":   contextx.TenantId(ctx),
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

// ---------------------------------------------------------------------------
// 全量 API 访问日志 (接口日志)
// ---------------------------------------------------------------------------

// accessLogSkipPaths 访问日志跳过列表: SSE 长连接存活数分钟到数小时,
// 记入访问日志会产生 "耗时几分钟" 的误导性慢接口记录
// (它是流式推送, 不是一次慢请求)。
var accessLogSkipPaths = map[string]struct{}{
	"/api/v1/message/events": {},
}

var (
	alOnce    sync.Once
	alEnabled bool
	alSlowMs  int64
)

// initAccessLogConfig 惰性初始化: 读取 accessLog 配置(修改需重启生效),
// 与全局限流 (ratelimit) 同款 once 模式。
func initAccessLogConfig(ctx context.Context) {
	alOnce.Do(func() {
		alEnabled = true
		if v, err := g.Cfg().Get(ctx, "accessLog.enable", true); err == nil {
			alEnabled = v.Bool()
		}
		alSlowMs = 1000
		if v, err := g.Cfg().Get(ctx, "accessLog.slowMs", 1000); err == nil && v.Int64() > 0 {
			alSlowMs = v.Int64()
		}
	})
}

// AccessLog 接口日志中间件: 全方法访问流水 (GET 查询亦留痕), 与
// OperationLog (只记写操作+脱敏请求体) 互补 —— 本表回答"谁在什么时候
// 调了什么接口、多快", 每条含 RequestId/用户/方法路径/HTTP状态/业务码/
// 耗时, 超过阈值 (accessLog.slowMs, 默认 1000ms) 打慢接口标记。
//
// 挂载位置要求: 与 OperationLog 相同, 必须注册在 ghttp.MiddlewareHandlerResponse
// 之外层(先注册), defer 阶段才能读到最终响应。
// 业务码解析有 64KB 响应体上限保护: 二进制导出 (xlsx) 与超大列表响应
// 不做 JSON 解析 (白耗 CPU 且无 {code,message} 结构), 记 0。
func AccessLog(r *ghttp.Request) {
	initAccessLogConfig(r.Context())
	if !alEnabled {
		r.Middleware.Next()
		return
	}
	if _, ok := accessLogSkipPaths[r.URL.Path]; ok {
		r.Middleware.Next()
		return
	}

	method := r.Method
	path := r.URL.Path
	query := truncateForColumn(r.URL.RawQuery, 512)
	clientIp := r.GetClientIp()
	userAgent := r.Header.Get("User-Agent")
	requestId := r.GetCtxVar(consts.CtxRequestIdName, "").String()
	start := time.Now()

	defer func() {
		// defer 在 ExitAll panic 展开时依然执行, 保证被 401/403/429 拦截的请求也留痕
		var userId uint64
		username := ""
		if user := contextx.LoginUser(r.Context()); user != nil {
			userId, username = user.UserId, user.Username
		}

		statusCode := r.Response.Status
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		code := 0
		if buf := r.Response.Buffer(); len(buf) > 0 && len(buf) <= 65536 {
			code, _ = parseResponseResult(buf)
		}
		durationMs := time.Since(start).Milliseconds()
		slow := 0
		if durationMs >= alSlowMs {
			slow = 1
			g.Log().Warningf(r.Context(), "[access-log] slow api: %s %s %dms (threshold %dms)", method, path, durationMs, alSlowMs)
		}

		// 异步写入: 与 OperationLog 同款 —— WithoutCancel 保留租户上下文
		// (sys_access_log 为 A 类隔离表, 注入器要求 ctx 携带租户信息)
		go func() {
			ctx := context.WithoutCancel(r.Context())
			_, err := dao.SysAccessLog.Ctx(ctx).Data(g.Map{
				"tenant_id":   contextx.TenantId(ctx),
				"user_id":     userId,
				"username":    username,
				"method":      method,
				"path":        path,
				"query":       query,
				"status_code": statusCode,
				"code":        code,
				"duration_ms": durationMs,
				"slow":        slow,
				"request_id":  requestId,
				"ip":          clientIp,
				"user_agent":  truncateForColumn(userAgent, 512),
			}).Insert()
			if err != nil {
				g.Log().Warningf(ctx, "AccessLog insert failed: %v", err)
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

// ---------------------------------------------------------------------------
// 全局限流 (按客户端 IP 的内存令牌桶)
// ---------------------------------------------------------------------------

// rlBucket 单个 IP 的令牌桶。
type rlBucket struct {
	tokens float64
	last   time.Time
}

// rateLimiter 进程内令牌桶集合; 限流为单实例内存态, 多副本部署时每副本独立配额。
type rateLimiter struct {
	mu      sync.Mutex
	rate    float64 // 每秒补充令牌数
	burst   float64 // 桶容量(突发上限)
	buckets map[string]*rlBucket
}

var (
	globalLimiter *rateLimiter
	rlOnce        sync.Once
	rlEnabled     bool
)

// initRateLimiter 惰性初始化: 读取 ratelimit 配置(修改需重启生效), 并启动过期桶清理。
func initRateLimiter(ctx context.Context) {
	rlOnce.Do(func() {
		v, err := g.Cfg().Get(ctx, "ratelimit.enable", true)
		if err == nil {
			rlEnabled = v.Bool()
		} else {
			rlEnabled = true
		}
		if !rlEnabled {
			return
		}
		rate := 100.0
		burst := 200.0
		if rv, err := g.Cfg().Get(ctx, "ratelimit.rate", 100); err == nil && rv.Int() > 0 {
			rate = float64(rv.Int())
		}
		if bv, err := g.Cfg().Get(ctx, "ratelimit.burst", 200); err == nil && bv.Int() > 0 {
			burst = float64(bv.Int())
		}
		globalLimiter = &rateLimiter{
			rate:    rate,
			burst:   burst,
			buckets: make(map[string]*rlBucket),
		}
		// 每 5 分钟清理 10 分钟未活跃的桶, 防止 IP 空间膨胀
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				globalLimiter.mu.Lock()
				for k, b := range globalLimiter.buckets {
					if time.Since(b.last) > 10*time.Minute {
						delete(globalLimiter.buckets, k)
					}
				}
				globalLimiter.mu.Unlock()
			}
		}()
	})
}

// allow 取一枚令牌, 无可用令牌返回 false。
func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		// 新桶预扣一枚, 首个请求直接放行
		l.buckets[key] = &rlBucket{tokens: l.burst - 1, last: now}
		return true
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// RateLimit 全局限流中间件: 按客户端 IP 限流, 超限返回 429。
// 配置 (yaml, 重启生效):
//
//	ratelimit:
//	  enable: true   # 关闭后完全放行
//	  rate: 100      # 每秒允许请求数(令牌补充速率)
//	  burst: 200     # 突发上限(桶容量)
func RateLimit(r *ghttp.Request) {
	initRateLimiter(r.Context())
	if !rlEnabled || globalLimiter == nil {
		r.Middleware.Next()
		return
	}
	if !globalLimiter.allow(r.GetClientIp()) {
		g.Log().Warningf(r.Context(), "rate limited: ip=%s path=%s", r.GetClientIp(), r.URL.Path)
		r.Response.WriteStatus(http.StatusTooManyRequests, g.Map{
			"code":    xerror.CodeTooManyReq.Code(),
			"message": xerror.CodeTooManyReq.Message(),
			"data":    nil,
		})
		r.ExitAll()
		return
	}
	r.Middleware.Next()
}
