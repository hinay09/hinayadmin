// Package v1 系统管理-租户套餐接口 (平台超管专属, docs/multi-tenant.md §7/§7.2)。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// TenantPackageListReq 套餐分页列表。
type TenantPackageListReq struct {
	g.Meta   `path:"/system/tenant-packages" tags:"SystemTenantPackage" method:"get" summary:"套餐分页列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"名称模糊"`
	Status   *int   `json:"status"   in:"query"`
	Page     int    `json:"page"     in:"query" d:"1"`
	PageSize int    `json:"pageSize" in:"query" d:"10"`
}

// TenantPackageListRes 列表响应。
type TenantPackageListRes response.PageResult

// TenantPackageListAllReq 启用套餐全量 (创建向导/应用授权抽屉下拉)。
type TenantPackageListAllReq struct {
	g.Meta `path:"/system/tenant-packages/all" tags:"SystemTenantPackage" method:"get" summary:"启用套餐全量"`
}

// TenantPackageListAllRes 全量 (含授权集模板, 供前端预填勾选树)。
type TenantPackageListAllRes struct {
	List any `json:"list"`
}

// TenantPackageDetailReq 详情。
type TenantPackageDetailReq struct {
	g.Meta `path:"/system/tenant-packages/{id}" tags:"SystemTenantPackage" method:"get" summary:"套餐详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// TenantPackageDetailRes 详情响应。
type TenantPackageDetailRes struct {
	Package any `json:"package"`
}

// TenantPackageCreateReq 新增套餐 (menu_ids 为目录/页面级 ID 快照, 保存前经
// 与租户授权集相同的合法性校验: 仅目录/页面级 + 非平台专属)。
type TenantPackageCreateReq struct {
	g.Meta  `path:"/system/tenant-packages" tags:"SystemTenantPackage" method:"post" summary:"新增套餐"`
	Name    string   `json:"name"    v:"required|length:2,64" dc:"套餐名称(全局唯一)"`
	MenuIds []uint64 `json:"menuIds" v:"required" dc:"授权集模板(目录/页面级, 至少一项)"`
	Status  int      `json:"status"  v:"in:0,1" d:"1" dc:"1=启用 0=停用"`
	Remark  string   `json:"remark"`
}

// TenantPackageCreateRes 新增响应。
type TenantPackageCreateRes struct {
	Id uint64 `json:"id" dc:"套餐ID"`
}

// TenantPackageUpdateReq 修改套餐 (保存新模板即完成; 已绑定租户的授权集不变,
// 需要生效走 sync 显式同步)。
type TenantPackageUpdateReq struct {
	g.Meta  `path:"/system/tenant-packages/{id}" tags:"SystemTenantPackage" method:"put" summary:"修改套餐"`
	Id      uint64   `json:"id"      in:"path" v:"required"`
	Name    string   `json:"name"    v:"required|length:2,64"`
	MenuIds []uint64 `json:"menuIds" v:"required"`
	Status  int      `json:"status"  v:"in:0,1"`
	Remark  string   `json:"remark"`
}

// TenantPackageUpdateRes 修改响应。
type TenantPackageUpdateRes struct{}

// TenantPackageDeleteReq 删除套餐 (软删; 事务内 name 改写 {name}#del{unix} 防同名校建撞 uk;
// 仍有未删租户绑定时拒绝 —— 先解绑或换绑)。
type TenantPackageDeleteReq struct {
	g.Meta `path:"/system/tenant-packages/{id}" tags:"SystemTenantPackage" method:"delete" summary:"删除套餐"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// TenantPackageDeleteRes 删除响应。
type TenantPackageDeleteRes struct{}

// TenantPackageSyncReq 套餐同步至绑定租户 (逐租户走 SetMenus 同路径: 重写授权集 +
// casbin 双向联动; 逐租户独立事务, 整体完成后 Reload 一次)。
type TenantPackageSyncReq struct {
	g.Meta `path:"/system/tenant-packages/{id}/sync" tags:"SystemTenantPackage" method:"post" summary:"套餐同步至绑定租户"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// TenantPackageSyncRes 同步响应。
type TenantPackageSyncRes struct {
	Count int `json:"count" dc:"同步的租户数"`
}
