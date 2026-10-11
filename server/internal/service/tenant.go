// ================================================================================
// 租户服务接口 (人工维护, 风格对齐 service/system.go 的 gf 生成物)。
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/system/v1"

	"hinay.cn/admin/internal/model"
)

type (
	// ITenant 租户生命周期与应用授权集 (平台超管专属能力, docs/multi-tenant.md §7/§7.1)。
	ITenant interface {
		// List 租户分页列表 (平台视角, 全租户混排)。
		List(ctx context.Context, req *v1.TenantListReq) (res *v1.TenantListRes, err error)
		// ListAll 启用租户全量 (平台切换器下拉)。
		ListAll(ctx context.Context, req *v1.TenantListAllReq) (res *v1.TenantListAllRes, err error)
		// Detail 租户详情。
		Detail(ctx context.Context, req *v1.TenantDetailReq) (res *v1.TenantDetailRes, err error)
		// Create 创建租户 (向导六件套单事务: 租户行/组织根/克隆租户管理员角色/
		// 管理员账号(强制改密)/授权集物化/管理员 g 行)。
		Create(ctx context.Context, req *v1.TenantCreateReq) (res *v1.TenantCreateRes, err error)
		// Update 修改租户 (code 不可变; 平台租户不可禁用)。
		Update(ctx context.Context, req *v1.TenantUpdateReq) (res *v1.TenantUpdateRes, err error)
		// Delete 删除租户 (软删 + 事务内 code 改写 {code}#del{unix} 防撞 uk)。
		Delete(ctx context.Context, req *v1.TenantDeleteReq) (res *v1.TenantDeleteRes, err error)
		// ResetConfigs 一键重置租户覆盖配置 (删租户行回归平台默认)。
		ResetConfigs(ctx context.Context, req *v1.TenantResetConfigsReq) (res *v1.TenantResetConfigsRes, err error)
		// GetMenus 获取租户应用授权集 (目录/页面级原值)。
		GetMenus(ctx context.Context, req *v1.TenantGetMenusReq) (res *v1.TenantGetMenusRes, err error)
		// SetMenus 全量重写授权集 + 双向联动 casbin (收缩删越界 p 行/扩张追加 tenant_admin p 行),
		// 同一事务, Reload 在事务提交之后。
		SetMenus(ctx context.Context, req *v1.TenantSetMenusReq) (res *v1.TenantSetMenusRes, err error)

		// ---- 跨模块辅助 (不经 api 契约, 供 auth/menus 过滤、角色授权校验、任务巡检调用) ----

		// AuthorizedMenuIds 授权集原始行 (目录/页面级)。
		AuthorizedMenuIds(ctx context.Context, tenantId uint64) ([]uint64, error)
		// EffectiveMenuClosure 有效闭包: 授权行 ∪ 子孙(按钮随父闭包) ∪ 祖先(菜单树完整性);
		// 平台租户(id=1)整体豁免返回 nil (=全集)。
		EffectiveMenuClosure(ctx context.Context, tenantId uint64) (map[uint64]struct{}, error)
		// TenantAdminRoleId 定位租户的 tenant_admin 克隆角色 ID (扩张联动目标)。
		TenantAdminRoleId(ctx context.Context, tenantId uint64) (uint64, error)
		// ActiveTenantLites 全部启用且未到期的租户 (平台级任务逐租户巡检)。
		ActiveTenantLites(ctx context.Context) ([]*model.TenantLite, error)
		// LoadLiteById 按ID装载租户轻量行 (任务 ctx 构造)。
		LoadLiteById(ctx context.Context, id uint64) (*model.TenantLite, error)
		// LoadLiteByCode 按租户码装载租户轻量行 (登录入口解析/公开租户信息)。
		LoadLiteByCode(ctx context.Context, code string) (*model.TenantLite, error)
	}
)

var (
	localTenant ITenant
)

func Tenant() ITenant {
	if localTenant == nil {
		panic("implement not found for interface ITenant, forgot register?")
	}
	return localTenant
}

func RegisterTenant(i ITenant) {
	localTenant = i
}
