package model

import "github.com/gogf/gf/v2/os/gtime"

// ConfigItem 全局配置列表 VO。
type ConfigItem struct {
	Id          uint64      `json:"id"`
	ConfigKey   string      `json:"configKey"`
	ConfigValue string      `json:"configValue"`
	ConfigType  int         `json:"configType"`
	Name        string      `json:"name"`
	Remark      string      `json:"remark"`
	Status      int         `json:"status"`
	Sort        int         `json:"sort"`
	CreatedAt   *gtime.Time `json:"createdAt"`
	UpdatedAt   *gtime.Time `json:"updatedAt"`
}
