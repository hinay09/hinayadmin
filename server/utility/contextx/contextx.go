// Package contextx 上下文工具: 从 ctx 提取登录用户等。
//
// 并发安全设计要点:
//
//  1. GoFrame 为每个 HTTP 请求创建独立的 *ghttp.Request 与独立 ctx, 请求间完全隔离。
//  2. middleware.Auth 在请求作用域内新建 *model.LoginUser 指针, 不与其他 goroutine 共享,
//     本函数返回该指针后调用方仅读, 不写, 多用户并发不会交叉污染。
//  3. 优先从标准 context.Value 提取 (适用于 logic 层被传入后台 goroutine 的场景);
//     未命中时 fallback 到 ghttp.Request 级 GetCtxVar (处理仅有 request 未走 SetCtx 的老逻辑)。
//  4. 帮助函数不接受外部传入的 user 参数, 避免调用方误传其他请求的 user。
package contextx

import (
	"context"
	"slices"

	"github.com/gogf/gf/v2/net/ghttp"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/model"
)

// LoginUser 从上下文中提取当前请求的登录用户; 未登录返回 nil。
//
// 多用户并发安全:
//   - 不同请求传入的 ctx 互相独立, 获取的必是当前请求 middleware.Auth 写入的那个 user 指针。
//   - 本函数仅读取, 不对返回值加锁; 调用方如果需要修改, 请自行复制一份。
func LoginUser(ctx context.Context) *model.LoginUser {
	if ctx == nil {
		return nil
	}
	// 1. 优先走标准 context.Value (类型安全, 脱离 request 仍可用)
	if v := ctx.Value(consts.CtxUserKey); v != nil {
		if u, ok := v.(*model.LoginUser); ok && u != nil {
			return u
		}
	}
	// 2. fallback: ghttp.Request 级 ctx var
	r := ghttp.RequestFromCtx(ctx)
	if r == nil {
		return nil
	}
	v := r.GetCtxVar(consts.CtxUserKeyName)
	if v.IsNil() {
		return nil
	}
	u := &model.LoginUser{}
	if err := v.Struct(u); err != nil {
		return nil
	}
	return u
}

// JwtToken 提取当前请求 token 字符串。
func JwtToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(consts.CtxJwtTokenKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	r := ghttp.RequestFromCtx(ctx)
	if r == nil {
		return ""
	}
	return r.GetCtxVar(consts.CtxJwtTokenKeyName).String()
}

// UserId 便捷获取当前登录用户 ID。
func UserId(ctx context.Context) uint64 {
	if u := LoginUser(ctx); u != nil {
		return u.UserId
	}
	return 0
}

// MustLoginUser 要求必须有登录用户, 获取不到返回零值占位对象, 避免 nil panic。
// 业务使用方如果仅需读字段, 可以直接调用这个函数隐式避免空指针访问。
func MustLoginUser(ctx context.Context) *model.LoginUser {
	if u := LoginUser(ctx); u != nil {
		return u
	}
	return &model.LoginUser{}
}

// GetUserRoles 从 Casbin 获取当前登录用户的角色ID列表（g 行按用户ID+租户域关联）。
// 首次调用时查询 Casbin, 后续同请求内复用结果。
func GetUserRoles(ctx context.Context) ([]uint64, error) {
	u := LoginUser(ctx)
	if u == nil {
		return nil, nil
	}
	return casbinx.GetUserRoles(ctx, u.UserId)
}

// IsAdmin 判断当前用户是否为平台超管（拥有内置超管角色, id=1, 仅平台租户可授予）。
//
// 多租户关闭时与旧语义等价 (只有平台租户, role 1 即全部); 开启时等价于 IsPlatformAdmin。
// 新代码请按语义显式选用 IsPlatformAdmin / IsTenantAdmin / IsManagerAdmin。
func IsAdmin(ctx context.Context) bool {
	return IsPlatformAdmin(ctx)
}

// IsPlatformAdmin 平台超管: 跨租户一切操作、租户管理、gencode/monitor/系统配置。
// 中间件在 Auth 阶段已将判定结果写入 LoginUser (含代运维切换后仍保持身份判定),
// 此处优先读缓存, 非请求上下文 (无 LoginUser) 回退实时查 Casbin。
func IsPlatformAdmin(ctx context.Context) bool {
	if u := LoginUser(ctx); u != nil {
		return u.IsPlatformAdmin
	}
	roles, err := GetUserRoles(ctx)
	if err != nil {
		return false
	}
	return slices.Contains(roles, consts.RoleAdminId)
}

// IsTenantAdmin 租户管理员: 持有 code=tenant_admin 的角色 (各租户创建时克隆)。
// 租户管理员在本租户围栏内的体验与单租户超管一致, 但跨不过租户边界、
// 授权清单被应用授权集圈定、碰不到平台层功能 (docs/multi-tenant.md §2.2)。
func IsTenantAdmin(ctx context.Context) bool {
	if u := LoginUser(ctx); u != nil {
		return u.IsTenantAdmin
	}
	return false
}

// IsManagerAdmin 管理视角判定: 平台超管或租户管理员 (请假/流程等"管理员可见"场景,
// 租户注入天然限界 —— 租户管理员只在本租户内获得该视角)。
func IsManagerAdmin(ctx context.Context) bool {
	return IsPlatformAdmin(ctx) || IsTenantAdmin(ctx)
}

// WithTenant 注入租户上下文 (标准 context.WithValue 双写键之一, 与 CtxUserKey 同层)。
// inject=true 时 ORM 驱动层对该租户强制注入 tenant_id (普通用户/平台超管代运维);
// inject=false 为平台语境 (跨租户可见/租户创建事务)。
func WithTenant(ctx context.Context, info *model.TenantInfo) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, consts.CtxTenantKey, info)
}

// Tenant 提取当前上下文的租户信息; 无租户上下文返回 nil。
func Tenant(ctx context.Context) *model.TenantInfo {
	if ctx == nil {
		return nil
	}
	if v := ctx.Value(consts.CtxTenantKey); v != nil {
		if t, ok := v.(*model.TenantInfo); ok {
			return t
		}
	}
	return nil
}

// TenantId 当前有效数据租户 ID (注入目标); 无租户上下文返回 0。
// 平台超管未切换时 Auth 中间件也会写入 {Id:1, Inject:false}, 因此请求链路上恒 > 0。
func TenantId(ctx context.Context) uint64 {
	if t := Tenant(ctx); t != nil {
		return t.Id
	}
	return 0
}

// TenantInjectable 当前上下文是否需要 ORM 租户注入。
func TenantInjectable(ctx context.Context) bool {
	if t := Tenant(ctx); t != nil {
		return t.Id > 0 && t.Inject
	}
	return false
}

// ScopedToDataTenant 强制注入"当前数据租户"的查询视角。
// 场景: 平台超管未切换 (Inject=false) 时做"写入目标租户内"的存在性预检
// (如用户名/角色code/配置键查重) —— 业务代码禁止手写 tenant_id 条件
// (docs/multi-tenant.md §3.2 写路径边界), 本助手是合规的租户条件路径。
func ScopedToDataTenant(ctx context.Context) context.Context {
	t := Tenant(ctx)
	if t == nil || t.Id == 0 {
		return WithTenant(ctx, &model.TenantInfo{Id: consts.TenantPlatformId, Code: consts.TenantPlatformCode, Inject: true})
	}
	clone := *t
	clone.Inject = true
	return WithTenant(ctx, &clone)
}

// HomeTenantScope 构造"操作者归属租户"视角 (**身份面**)。
// 会话/自助端点 (userInfo/profile/password/avatar/refresh) 读写的是操作者本人
// 在归属租户的记录, 与数据面租户 (平台超管代运维切换的 X-Tenant-Id 目标) 无关 ——
// 切换态下若按数据面注入查询会"用户不存在"。Inject=false (仅定位归属域,
// 不做行级注入); 按 JWT 解出的主键查本人行, 无跨租户暴露面。
func HomeTenantScope(ctx context.Context) context.Context {
	id, code := uint64(consts.TenantPlatformId), consts.TenantPlatformCode
	if u := LoginUser(ctx); u != nil && u.TenantId > 0 {
		id, code = u.TenantId, u.TenantCode
	}
	return WithTenant(ctx, &model.TenantInfo{Id: id, Code: code, Inject: false})
}

// WithoutTenant 构造"平台视角"上下文: 显式屏蔽外层租户注入 (配置覆盖读序、
// 跨租户管理查询等场景)。与不写租户键的区别在于它能覆盖外层已注入的租户。
func WithoutTenant(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, consts.CtxTenantKey, (*model.TenantInfo)(nil))
}

// SetAuditUpload 上传逻辑暂存文件摘要到请求上下文, 供 OperationLog 审计记录。
// summary 建议格式: "文件名 (N bytes, content-type)"。非 HTTP 上下文静默忽略。
func SetAuditUpload(ctx context.Context, summary string) {
	if r := ghttp.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.CtxAuditUploadName, summary)
	}
}
