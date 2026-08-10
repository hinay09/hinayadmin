// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysAuditLog is the golang structure of table sys_audit_log for DAO operations like Where/Data.
type SysAuditLog struct {
	g.Meta     `orm:"table:sys_audit_log, do:true"`
	Id         any         // ID
	UserId     any         // 用户ID
	Username   any         // 用户名
	Action     any         // 操作类型(create/update/delete/upload/...)
	Resource   any         // 操作资源(如user/role/menu/dict/file)
	ResourceId any         // 资源标识
	Detail     any         // 详情(JSON格式)
	Ip         any         // IP地址
	UserAgent  any         // User-Agent
	CreatedAt  *gtime.Time // 创建时间
}
