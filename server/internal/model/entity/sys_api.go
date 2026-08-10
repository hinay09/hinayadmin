// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysApi is the golang structure for table sys_api.
type SysApi struct {
	Id          uint64      `json:"id"          orm:"id"          description:""`                            //
	Path        string      `json:"path"        orm:"path"        description:"API路径"`                       // API路径
	Method      string      `json:"method"      orm:"method"      description:"HTTP方法(GET/POST/PUT/DELETE)"` // HTTP方法(GET/POST/PUT/DELETE)
	GroupName   string      `json:"groupName"   orm:"group_name"  description:"分组名称"`                        // 分组名称
	Description string      `json:"description" orm:"description" description:"接口描述"`                        // 接口描述
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"  description:""`                            //
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"  description:""`                            //
	DeletedAt   *gtime.Time `json:"deletedAt"   orm:"deleted_at"  description:"删除时间(软删)"`                    // 删除时间(软删)
}
