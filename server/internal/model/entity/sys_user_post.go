// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUserPost is the golang structure for table sys_user_post.
type SysUserPost struct {
	Id        uint64      `json:"id" orm:"id" description:"主键ID"`                 // 主键ID
	UserId    uint64      `json:"userId" orm:"user_id" description:"用户ID"`        // 用户ID
	PostId    uint64      `json:"postId" orm:"post_id" description:"岗位ID"`        // 岗位ID
	OrgId     uint64      `json:"orgId" orm:"org_id" description:"组织ID(0=不限定组织)"` // 组织ID(0=不限定组织)
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`  // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`  // 更新时间

}
