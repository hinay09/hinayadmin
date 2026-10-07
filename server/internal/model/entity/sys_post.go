// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysPost is the golang structure for table sys_post.
type SysPost struct {
	Id        uint64      `json:"id" orm:"id" description:"主键ID"`                                     // 主键ID
	PostCode  string      `json:"postCode" orm:"post_code" description:"岗位编码(唯一, 如 hr, dept_leader)"` // 岗位编码(唯一, 如 hr, dept_leader)
	PostName  string      `json:"postName" orm:"post_name" description:"岗位名称"`                        // 岗位名称
	PostKind  int         `json:"postKind" orm:"post_kind" description:"岗位类型:1=普通岗,2=主管岗(部门主管解析依据)"`  // 岗位类型:1=普通岗,2=主管岗(部门主管解析依据)
	Sort      int         `json:"sort" orm:"sort" description:"排序"`                                   // 排序
	Status    int         `json:"status" orm:"status" description:"状态:1=启用,0=禁用"`                     // 状态:1=启用,0=禁用
	Remark    string      `json:"remark" orm:"remark" description:"备注"`                               // 备注
	CreateId  uint64      `json:"createId" orm:"create_id" description:"创建人ID(ORM自动填充)"`              // 创建人ID(ORM自动填充)
	UpdateId  uint64      `json:"updateId" orm:"update_id" description:"最后修改人ID(ORM自动填充)"`            // 最后修改人ID(ORM自动填充)
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`                      // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`                      // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`                  // 删除时间(软删)

}
