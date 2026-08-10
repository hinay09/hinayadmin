// Package v1 系统管理-组织机构接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/model"
)

// OrgTreeReq 组织机构树。
type OrgTreeReq struct {
	g.Meta `path:"/system/orgs/tree" tags:"SystemOrg" method:"get" summary:"组织机构树"`
}

// OrgTreeRes 组织机构树响应。
type OrgTreeRes struct {
	Tree []*model.OrgTree `json:"tree"`
}

// OrgListReq 组织机构扁平列表。
type OrgListReq struct {
	g.Meta  `path:"/system/orgs" tags:"SystemOrg" method:"get" summary:"组织机构列表"`
	Keyword string `json:"keyword" in:"query"`
	Status  *int   `json:"status"  in:"query"`
}

// OrgListRes 组织机构列表响应。
type OrgListRes struct {
	List []*model.SysOrg `json:"list"`
}

// OrgDetailReq 组织机构详情。
type OrgDetailReq struct {
	g.Meta `path:"/system/orgs/{id}" tags:"SystemOrg" method:"get" summary:"组织机构详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// OrgDetailRes 组织机构详情响应。
type OrgDetailRes struct {
	*model.SysOrg
}

// OrgCreateReq 新增组织机构。
type OrgCreateReq struct {
	g.Meta   `path:"/system/orgs" tags:"SystemOrg" method:"post" summary:"新增组织机构"`
	ParentId uint64 `json:"parentId"`
	Name     string `json:"name"     v:"required|length:1,64"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Email    string `json:"email"    v:"email#邮箱格式不正确"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"   d:"1"`
	Remark   string `json:"remark"`
}

// OrgCreateRes 新增响应。
type OrgCreateRes struct {
	Id uint64 `json:"id"`
}

// OrgUpdateReq 修改组织机构。
type OrgUpdateReq struct {
	g.Meta   `path:"/system/orgs/{id}" tags:"SystemOrg" method:"put" summary:"修改组织机构"`
	Id       uint64 `json:"id"       in:"path" v:"required"`
	ParentId uint64 `json:"parentId"`
	Name     string `json:"name"     v:"required"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

// OrgUpdateRes 修改响应。
type OrgUpdateRes struct{}

// OrgDeleteReq 删除组织机构。
type OrgDeleteReq struct {
	g.Meta `path:"/system/orgs/{id}" tags:"SystemOrg" method:"delete" summary:"删除组织机构"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// OrgDeleteRes 删除响应。
type OrgDeleteRes struct{}
