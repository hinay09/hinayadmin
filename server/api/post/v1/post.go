// Package v1 岗位管理接口契约。
// 岗位是审批人解析的依据: "指定岗位"直接按岗位找人;
// "部门主管"取发起人组织(含上级组织)挂主管岗(post_kind=2)的用户。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"hinay.cn/admin/utility/response"
)

// PostItem 岗位条目。
type PostItem struct {
	Id       uint64      `json:"id"`
	PostCode string      `json:"postCode" dc:"岗位编码"`
	PostName string      `json:"postName" dc:"岗位名称"`
	PostKind int         `json:"postKind"  dc:"1=普通岗,2=主管岗"`
	Sort     int         `json:"sort"`
	Status   int         `json:"status"`
	Remark   string      `json:"remark"`
	Members  int64       `json:"members" dc:"成员数"`
	UpdateAt *gtime.Time `json:"updateAt"`
}

// PostListReq 岗位分页列表。
type PostListReq struct {
	g.Meta   `path:"/system/posts" tags:"Post" method:"get" summary:"岗位分页列表"`
	Keyword  string `json:"keyword" in:"query" dc:"编码/名称"`
	Status   *int   `json:"status"   in:"query"`
	Page     int    `json:"page"     in:"query" d:"1"`
	PageSize int    `json:"pageSize" in:"query" d:"10"`
}

// PostListRes 列表响应。
type PostListRes response.PageResult

// PostAllReq 启用岗位全量 (设计器/下拉用)。
type PostAllReq struct {
	g.Meta `path:"/system/posts/all" tags:"Post" method:"get" summary:"启用岗位全量"`
}

// PostAllRes 全量响应。
type PostAllRes struct {
	List []*PostItem `json:"list"`
}

// PostCreateReq 新增岗位。
type PostCreateReq struct {
	g.Meta   `path:"/system/posts" tags:"Post" method:"post" summary:"新增岗位"`
	PostCode string `json:"postCode" v:"required|length:1,32#请输入岗位编码|编码长度 1-32"`
	PostName string `json:"postName" v:"required|length:1,64#请输入岗位名称|名称长度 1-64"`
	PostKind int    `json:"postKind" v:"required|in:1,2#请选择岗位类型|岗位类型仅能为 1/2"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status" d:"1"`
	Remark   string `json:"remark"`
}

// PostCreateRes 新增响应。
type PostCreateRes struct {
	Id uint64 `json:"id"`
}

// PostUpdateReq 修改岗位。
type PostUpdateReq struct {
	g.Meta   `path:"/system/posts/{id}" tags:"Post" method:"put" summary:"修改岗位"`
	Id       uint64 `json:"id" in:"path" v:"required"`
	PostCode string `json:"postCode" v:"required|length:1,32#请输入岗位编码|编码长度 1-32"`
	PostName string `json:"postName" v:"required|length:1,64#请输入岗位名称|名称长度 1-64"`
	PostKind int    `json:"postKind" v:"required|in:1,2#请选择岗位类型|岗位类型仅能为 1/2"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

// PostUpdateRes 修改响应。
type PostUpdateRes struct{}

// PostDeleteReq 删除岗位 (同时清理挂岗关系)。
type PostDeleteReq struct {
	g.Meta `path:"/system/posts/{id}" tags:"Post" method:"delete" summary:"删除岗位"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// PostDeleteRes 删除响应。
type PostDeleteRes struct{}

// PostMemberItem 岗位成员。
type PostMemberItem struct {
	Id       uint64      `json:"id" dc:"挂岗关系ID"`
	UserId   uint64      `json:"userId"`
	UserName string      `json:"userName"`
	OrgId    uint64      `json:"orgId"`
	OrgName  string      `json:"orgName" dc:"不限定组织时为空"`
	CreateAt *gtime.Time `json:"createAt"`
}

// PostMemberListReq 岗位成员列表。
type PostMemberListReq struct {
	g.Meta `path:"/system/posts/{id}/members" tags:"Post" method:"get" summary:"岗位成员列表"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// PostMemberListRes 成员列表响应。
type PostMemberListRes struct {
	List []*PostMemberItem `json:"list"`
}

// PostMemberAddReq 添加岗位成员 (谁在哪个组织担任该岗位; orgId=0 不限定组织)。
type PostMemberAddReq struct {
	g.Meta `path:"/system/posts/{id}/members" tags:"Post" method:"post" summary:"添加岗位成员"`
	Id     uint64 `json:"id"     in:"path" v:"required"`
	UserId uint64 `json:"userId" v:"required#请选择用户"`
	OrgId  uint64 `json:"orgId"  dc:"组织ID, 0=不限定组织"`
}

// PostMemberAddRes 添加响应。
type PostMemberAddRes struct {
	Id uint64 `json:"id" dc:"挂岗关系ID"`
}

// PostMemberRemoveReq 移除岗位成员 (按挂岗关系ID)。
type PostMemberRemoveReq struct {
	g.Meta `path:"/system/posts/members/{relId}" tags:"Post" method:"delete" summary:"移除岗位成员"`
	RelId  uint64 `json:"relId" in:"path" v:"required"`
}

// PostMemberRemoveRes 移除响应。
type PostMemberRemoveRes struct{}
