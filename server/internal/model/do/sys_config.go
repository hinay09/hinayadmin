// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysConfig is the golang structure of table sys_config for DAO operations like Where/Data.
type SysConfig struct {
	g.Meta      `orm:"table:sys_config, do:true"`
	Id          any         // ID
	ConfigKey   any         // 配置键(唯一)
	ConfigValue any         // 配置值
	ConfigType  any         // 配置类型:0=文本,1=数字,2=布尔,3=JSON
	Name        any         // 配置名称(中文说明)
	Remark      any         // 备注
	Status      any         // 状态:1=启用,0=禁用
	Sort        any         // 排序
	CreatedAt   *gtime.Time // 创建时间
	UpdatedAt   *gtime.Time // 更新时间
	DeletedAt   *gtime.Time // 删除时间(软删)
}
