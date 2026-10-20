// ================================================================================
// 租户套餐服务接口 (人工维护, 风格对齐 service/tenant.go)。
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/system/v1"
)

type (
	// ITenantPackage 租户套餐: 应用授权集的模板复用层 (平台超管专属, docs/multi-tenant.md §7/§7.2)。
	// 套餐 = 目录/页面级菜单 ID 集的命名快照; 创建租户时选中预填(可微调),
	// 修改套餐不直接影响已绑定租户 —— 生效走 Sync 显式同步 (SetMenus 同路径联动)。
	ITenantPackage interface {
		// List 套餐分页列表 (含每套餐绑定租户数)。
		List(ctx context.Context, req *v1.TenantPackageListReq) (res *v1.TenantPackageListRes, err error)
		// ListAll 启用套餐全量 (创建向导/应用授权抽屉下拉, 含授权集模板)。
		ListAll(ctx context.Context, req *v1.TenantPackageListAllReq) (res *v1.TenantPackageListAllRes, err error)
		// Detail 套餐详情 (含解析后的授权集模板与绑定租户数)。
		Detail(ctx context.Context, req *v1.TenantPackageDetailReq) (res *v1.TenantPackageDetailRes, err error)
		// Create 新增套餐 (授权集经 validateGrantSet 校验)。
		Create(ctx context.Context, req *v1.TenantPackageCreateReq) (res *v1.TenantPackageCreateRes, err error)
		// Update 修改套餐 (仅改模板, 不触碰已绑定租户的授权集)。
		Update(ctx context.Context, req *v1.TenantPackageUpdateReq) (res *v1.TenantPackageUpdateRes, err error)
		// Delete 删除套餐 (软删 + 事务内 name 改写 {name}#del{unix} 防撞 uk; 使用中禁止)。
		Delete(ctx context.Context, req *v1.TenantPackageDeleteReq) (res *v1.TenantPackageDeleteRes, err error)
		// Sync 同步套餐至全部绑定租户 (逐租户重写授权集 + casbin 双向联动, 整体完成后 Reload 一次)。
		Sync(ctx context.Context, req *v1.TenantPackageSyncReq) (res *v1.TenantPackageSyncRes, err error)
	}
)

var (
	localTenantPackage ITenantPackage
)

func TenantPackage() ITenantPackage {
	if localTenantPackage == nil {
		panic("implement not found for interface ITenantPackage, forgot register?")
	}
	return localTenantPackage
}

func RegisterTenantPackage(i ITenantPackage) {
	localTenantPackage = i
}
