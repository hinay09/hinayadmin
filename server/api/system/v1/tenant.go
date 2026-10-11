// Package v1 系统管理-租户接口 (平台超管专属, docs/multi-tenant.md §7/§7.1)。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// TenantListReq 租户分页列表。
type TenantListReq struct {
	g.Meta   `path:"/system/tenants" tags:"SystemTenant" method:"get" summary:"租户分页列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"名称/标识/联系人模糊"`
	Status   *int   `json:"status"   in:"query"`
	Page     int    `json:"page"     in:"query" d:"1"`
	PageSize int    `json:"pageSize" in:"query" d:"10"`
}

// TenantListRes 列表响应。
type TenantListRes response.PageResult

// TenantListAllReq 启用租户全量 (平台超管切换器下拉)。
type TenantListAllReq struct {
	g.Meta `path:"/system/tenants/all" tags:"SystemTenant" method:"get" summary:"启用租户全量"`
}

// TenantListAllRes 全量。
type TenantListAllRes struct {
	List any `json:"list"`
}

// TenantDetailReq 详情。
type TenantDetailReq struct {
	g.Meta `path:"/system/tenants/{id}" tags:"SystemTenant" method:"get" summary:"租户详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// TenantDetailRes 详情响应。
type TenantDetailRes struct {
	Tenant any `json:"tenant"`
}

// TenantCreateReq 创建租户 (向导三步合一提交, 服务端单事务完成 §7 六件套):
// 租户行/组织根/克隆租户管理员角色/管理员账号(强制改密)/应用授权集物化/管理员 g 行。
// 明确不含: 默认岗位/流程定义克隆/配置覆盖行。
type TenantCreateReq struct {
	g.Meta        `path:"/system/tenants" tags:"SystemTenant" method:"post" summary:"创建租户(向导)"`
	Name          string   `json:"name"          v:"required|length:2,64" dc:"租户名称"`
	Code          string   `json:"code"          v:"required|length:2,32|regex:^[a-z][a-z0-9-]*$" dc:"租户标识(登录码, 全局唯一, 创建后不可变)"`
	Contact       string   `json:"contact"       v:"length:0,64" dc:"联系人"`
	Plan          string   `json:"plan"          v:"length:0,32" d:"standard" dc:"套餐(展示标签)"`
	ExpireAt      string   `json:"expireAt"      dc:"到期时间(空=永久)"`
	Remark        string   `json:"remark"`
	AdminUsername string   `json:"adminUsername" v:"required|length:2,32" dc:"租户管理员登录账号"`
	AdminNickname string   `json:"adminNickname" v:"length:0,32" dc:"租户管理员昵称(缺省同账号)"`
	AdminPassword string   `json:"adminPassword" v:"length:0,32" dc:"初始密码(空=123456, 首次登录强制改密)"`
	MenuIds       []uint64 `json:"menuIds" dc:"初始应用授权集(目录/页面级, 缺省=基础协作包模板)"`
}

// TenantCreateRes 创建响应。
type TenantCreateRes struct {
	Id          uint64 `json:"id" dc:"租户ID"`
	AdminUserId uint64 `json:"adminUserId" dc:"租户管理员用户ID"`
}

// TenantUpdateReq 修改租户 (code 不可变 —— 租户码是登录入口/命名空间依据)。
type TenantUpdateReq struct {
	g.Meta   `path:"/system/tenants/{id}" tags:"SystemTenant" method:"put" summary:"修改租户"`
	Id       uint64 `json:"id"       in:"path" v:"required"`
	Name     string `json:"name"     v:"required|length:2,64"`
	Contact  string `json:"contact"  v:"length:0,64"`
	Plan     string `json:"plan"     v:"length:0,32"`
	ExpireAt string `json:"expireAt" dc:"到期时间(空=永久)"`
	Status   int    `json:"status"   v:"in:0,1" dc:"1=启用 0=禁用"`
	Remark   string `json:"remark"`
}

// TenantUpdateRes 修改响应。
type TenantUpdateRes struct{}

// TenantDeleteReq 删除租户 (软删; 事务内 code 改写 {code}#del{unix} 防同码重建撞 uk;
// 平台租户 id=1 不可删)。
type TenantDeleteReq struct {
	g.Meta `path:"/system/tenants/{id}" tags:"SystemTenant" method:"delete" summary:"删除租户"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// TenantDeleteRes 删除响应。
type TenantDeleteRes struct{}

// TenantGetMenusReq 获取租户应用授权集。
type TenantGetMenusReq struct {
	g.Meta `path:"/system/tenants/{id}/menus" tags:"SystemTenant" method:"get" summary:"获取租户应用授权集"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// TenantGetMenusRes 授权集响应。
type TenantGetMenusRes struct {
	MenuIds []uint64 `json:"menuIds"`
}

// TenantSetMenusReq 设置租户应用授权集 (全量重写 + 双向联动 casbin:
// 收缩删除域内越界 p 行即时生效; 扩张自动追加 tenant_admin 角色的 p 行,
// 平台一授权、租户管理员立刻可见可用)。
type TenantSetMenusReq struct {
	g.Meta  `path:"/system/tenants/{id}/menus" tags:"SystemTenant" method:"put" summary:"设置租户应用授权集"`
	Id      uint64   `json:"id"      in:"path" v:"required"`
	MenuIds []uint64 `json:"menuIds" dc:"目录/页面级菜单ID (按钮随父页面闭包)"`
}

// TenantSetMenusRes 响应。
type TenantSetMenusRes struct{}

// TenantResetConfigsReq 一键重置租户覆盖配置 (删除该租户全部 sys_config 覆盖行,
// 回归平台默认 —— 平台干预违规站点名/Logo/水印文案的手段)。
type TenantResetConfigsReq struct {
	g.Meta `path:"/system/tenants/{id}/configs/reset" tags:"SystemTenant" method:"post" summary:"重置租户覆盖配置"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// TenantResetConfigsRes 响应。
type TenantResetConfigsRes struct{}
