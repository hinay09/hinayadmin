// Package v1 系统管理-用户接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"hinay.cn/admin/utility/response"
)

// UserListReq 用户分页列表。
type UserListReq struct {
	g.Meta   `path:"/system/users" tags:"SystemUser" method:"get" summary:"用户分页列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"账号/昵称模糊"`
	Status   *int   `json:"status"   in:"query" dc:"状态过滤"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// UserVO 列表/详情视图。
type UserVO struct {
	Id        uint64      `json:"id"`
	Username  string      `json:"username"`
	Nickname  string      `json:"nickname"`
	Avatar    string      `json:"avatar"`
	Email     string      `json:"email"`
	Phone     string      `json:"phone"`
	OrgId     uint64      `json:"orgId"`
	OrgName   string      `json:"orgName"`
	Status    int         `json:"status"`
	Remark    string      `json:"remark"`
	Roles     []string    `json:"roles"`
	CreatedAt *gtime.Time `json:"createdAt"`
}

// UserListRes 列表响应。
type UserListRes response.PageResult

// UserDetailReq 详情。
type UserDetailReq struct {
	g.Meta `path:"/system/users/{id}" tags:"SystemUser" method:"get" summary:"用户详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// UserDetailRes 详情响应。
type UserDetailRes struct {
	*UserVO
}

// UserCreateReq 新增。
type UserCreateReq struct {
	g.Meta   `path:"/system/users" tags:"SystemUser" method:"post" summary:"新增用户"`
	Username string   `json:"username" v:"required|length:2,32#请输入账号|账号长度 2-32"`
	Password string   `json:"password" v:"required|length:6,32#请输入密码|密码长度 6-32"`
	Nickname string   `json:"nickname" v:"required#请输入昵称"`
	Email    string   `json:"email"    v:"email#邮箱格式不正确"`
	Phone    string   `json:"phone"    v:"phone#手机号格式不正确"`
	OrgId    uint64   `json:"orgId"`
	Status   int      `json:"status"   v:"in:0,1#状态取值仅能为 0/1" d:"1"`
	Remark   string   `json:"remark"`
	RoleIds  []uint64 `json:"roleIds"`
}

// UserCreateRes 新增响应。
type UserCreateRes struct {
	Id uint64 `json:"id"`
}

// UserUpdateReq 修改。
type UserUpdateReq struct {
	g.Meta   `path:"/system/users/{id}" tags:"SystemUser" method:"put" summary:"修改用户"`
	Id       uint64   `json:"id"       in:"path" v:"required"`
	Nickname string   `json:"nickname" v:"required#请输入昵称"`
	Email    string   `json:"email"    v:"email#邮箱格式不正确"`
	Phone    string   `json:"phone"    v:"phone#手机号格式不正确"`
	OrgId    uint64   `json:"orgId"`
	Status   int      `json:"status"   v:"in:0,1#状态取值仅能为 0/1"`
	Remark   string   `json:"remark"`
	RoleIds  []uint64 `json:"roleIds"`
}

// UserUpdateRes 修改响应。
type UserUpdateRes struct{}

// UserDeleteReq 删除。
type UserDeleteReq struct {
	g.Meta `path:"/system/users/{id}" tags:"SystemUser" method:"delete" summary:"删除用户"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// UserDeleteRes 删除响应。
type UserDeleteRes struct{}

// UserResetPwdReq 重置密码。
type UserResetPwdReq struct {
	g.Meta   `path:"/system/users/{id}/password" tags:"SystemUser" method:"put" summary:"重置密码"`
	Id       uint64 `json:"id"       in:"path" v:"required"`
	Password string `json:"password" v:"required|length:6,32"`
}

// UserResetPwdRes 重置密码响应。
type UserResetPwdRes struct{}
