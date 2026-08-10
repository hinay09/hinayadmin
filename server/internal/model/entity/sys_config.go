// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysConfig is the golang structure for table sys_config.
type SysConfig struct {
	Id          uint64      `json:"id"          orm:"id"           description:"ID"`                         // ID
	ConfigKey   string      `json:"configKey"   orm:"config_key"   description:"配置键(唯一)"`                    // 配置键(唯一)
	ConfigValue string      `json:"configValue" orm:"config_value" description:"配置值"`                        // 配置值
	ConfigType  int         `json:"configType"  orm:"config_type"  description:"配置类型:0=文本,1=数字,2=布尔,3=JSON"` // 配置类型:0=文本,1=数字,2=布尔,3=JSON
	Name        string      `json:"name"        orm:"name"         description:"配置名称(中文说明)"`                 // 配置名称(中文说明)
	Remark      string      `json:"remark"      orm:"remark"       description:"备注"`                         // 备注
	Status      int         `json:"status"      orm:"status"       description:"状态:1=启用,0=禁用"`               // 状态:1=启用,0=禁用
	Sort        int         `json:"sort"        orm:"sort"         description:"排序"`                         // 排序
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"   description:"创建时间"`                       // 创建时间
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"   description:"更新时间"`                       // 更新时间
	DeletedAt   *gtime.Time `json:"deletedAt"   orm:"deleted_at"   description:"删除时间(软删)"`                   // 删除时间(软删)
}
