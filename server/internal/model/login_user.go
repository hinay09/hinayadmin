package model

import "github.com/gogf/gf/v2/os/gtime"

// LoginUser 上下文中的登录用户信息。
type LoginUser struct {
	UserId        uint64      `json:"userId"`
	Username      string      `json:"username"`
	Nickname      string      `json:"nickname"`
	Avatar        string      `json:"avatar"`
	Email         string      `json:"email"`
	Phone         string      `json:"phone"`
	Roles         []string    `json:"roles"`
	RoleNames     []string    `json:"roleNames"`
	IsAdmin       bool        `json:"isAdmin"`
	LastLoginAt   *gtime.Time `json:"lastLoginAt"`
	LastLoginIp   string      `json:"lastLoginIp"`
	MustChangePwd bool        `json:"mustChangePwd"`
	TwoFaEnabled  bool        `json:"twoFaEnabled"`
	// ---- 多租户 (docs/multi-tenant.md §4.1): 租户字段仅作冗余展示, 注入只看 CtxTenantKey ----
	TenantId         uint64     `json:"tenantId"`
	TenantCode       string     `json:"tenantCode"`
	TenantName       string     `json:"tenantName"`
	IsPlatformAdmin  bool       `json:"isPlatformAdmin"` // 平台超管 (role id=1, 仅平台租户可授予)
	IsTenantAdmin    bool       `json:"isTenantAdmin"`   // 租户管理员 (持有 code=tenant_admin 的角色)
	SwitchTenantInfo *TenantRef `json:"switchTenant"`    // 平台超管代运维切换的目标租户 (nil=未切换)
}

// TenantRef 前端展示用租户引用 (切换器徽标/确认文案)。
type TenantRef struct {
	Id   uint64 `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}
