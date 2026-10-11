// Package model 租户相关上下文与展示结构。
package model

// TenantInfo 请求/任务上下文中的租户信息 (consts.CtxTenantKey 携带)。
//
// 与 LoginUser 分离独立成键的原因 (docs/multi-tenant.md §2.1): job/异步回调
// 可以在没有登录用户的情况下独立构造租户上下文; LoginUser 上的租户字段仅作冗余展示。
//
// Inject 语义 (§3.2 注入规则):
//   - true  = ORM 驱动层对该租户强制注入 tenant_id (普通租户用户 / 平台超管代运维切换后);
//   - false = 不注入 (平台超管未切换时跨租户可见; 租户创建事务等平台语境)。
type TenantInfo struct {
	Id     uint64 `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Inject bool   `json:"-"`
}

// TenantLite 租户轻量行 (任务巡检/创建向导内部使用)。
type TenantLite struct {
	Id       uint64 `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Status   int    `json:"status"`
	ExpireAt string `json:"expireAt"` // 空=永久
}
