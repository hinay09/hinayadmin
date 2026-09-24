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

// GetUserRoles 从 Casbin 获取当前登录用户的角色列表（请求级缓存）。
// 首次调用时查询 Casbin, 后续同请求内复用结果。
func GetUserRoles(ctx context.Context) ([]string, error) {
	u := LoginUser(ctx)
	if u == nil {
		return nil, nil
	}
	return casbinx.GetUserRoles(ctx, u.Username)
}

// IsAdmin 判断当前用户是否为超管（含 admin 角色）。
func IsAdmin(ctx context.Context) bool {
	u := LoginUser(ctx)
	if u == nil {
		return false
	}
	roles, err := GetUserRoles(ctx)
	if err != nil {
		return false
	}
	return slices.Contains(roles, consts.RoleAdmin)
}
