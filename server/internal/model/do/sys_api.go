// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysApi is the golang structure of table sys_api for DAO operations like Where/Data.
type SysApi struct {
	g.Meta      `orm:"table:sys_api, do:true"`
	Id          any         //
	Path        any         // API路径
	Method      any         // HTTP方法(GET/POST/PUT/DELETE)
	GroupName   any         // 分组名称
	Description any         // 接口描述
	CreatedAt   *gtime.Time //
	UpdatedAt   *gtime.Time //
	DeletedAt   *gtime.Time // 删除时间(软删)
}
