package model

import "github.com/gogf/gf/v2/os/gtime"

// SysConfig 全局配置实体。
type SysConfig struct {
	Id          uint64      `json:"id"          orm:"id"`
	ConfigKey   string      `json:"configKey"   orm:"config_key"`
	ConfigValue string      `json:"configValue" orm:"config_value"`
	ConfigType  int         `json:"configType"  orm:"config_type"`
	Name        string      `json:"name"        orm:"name"`
	Remark      string      `json:"remark"      orm:"remark"`
	Status      int         `json:"status"      orm:"status"`
	Sort        int         `json:"sort"        orm:"sort"`
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"`
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"`
}
