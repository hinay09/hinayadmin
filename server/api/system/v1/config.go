// Package v1 系统管理-全局配置接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// ============================================================
// 全局配置 (sys_config)
// ============================================================

// ConfigListReq 全局配置分页列表。
type ConfigListReq struct {
	g.Meta     `path:"/system/configs" tags:"SystemConfig" method:"get" summary:"全局配置分页列表"`
	Keyword    string `json:"keyword"  in:"query" dc:"键/名称模糊搜索"`
	ConfigType *int   `json:"configType" in:"query" dc:"配置类型筛选"`
	Page       int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize   int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// ConfigListRes 列表响应。
type ConfigListRes response.PageResult

// ConfigAllReq 获取所有启用的全局配置(前端直接使用)。
type ConfigAllReq struct {
	g.Meta `path:"/system/configs/all" tags:"SystemConfig" method:"get" summary:"全部启用配置"`
}

// ConfigAllRes 全量配置响应。
type ConfigAllRes struct {
	List any `json:"list"`
}

// ConfigCreateReq 新增全局配置。
type ConfigCreateReq struct {
	g.Meta      `path:"/system/configs" tags:"SystemConfig" method:"post" summary:"新增全局配置"`
	ConfigKey   string `json:"configKey"    v:"required|length:2,128#请输入配置键|配置键长度 2-128"`
	ConfigValue string `json:"configValue"`
	ConfigType  int    `json:"configType"   d:"0"`
	Name        string `json:"name"         v:"required|length:1,128#请输入配置名称|配置名称长度 1-128"`
	Remark      string `json:"remark"`
	Status      int    `json:"status"       d:"1"`
	Sort        int    `json:"sort"`
}

// ConfigCreateRes 新增响应。
type ConfigCreateRes struct {
	Id uint64 `json:"id"`
}

// ConfigUpdateReq 修改全局配置。
type ConfigUpdateReq struct {
	g.Meta      `path:"/system/configs/{id}" tags:"SystemConfig" method:"put" summary:"修改全局配置"`
	Id          uint64 `json:"id"           in:"path" v:"required"`
	ConfigKey   string `json:"configKey"    v:"required|length:2,128"`
	ConfigValue string `json:"configValue"`
	ConfigType  int    `json:"configType"`
	Name        string `json:"name"         v:"required|length:1,128"`
	Remark      string `json:"remark"`
	Status      int    `json:"status"`
	Sort        int    `json:"sort"`
}

// ConfigUpdateRes 修改响应。
type ConfigUpdateRes struct{}

// ConfigDeleteReq 删除全局配置。
type ConfigDeleteReq struct {
	g.Meta `path:"/system/configs/{id}" tags:"SystemConfig" method:"delete" summary:"删除全局配置"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// ConfigDeleteRes 删除响应。
type ConfigDeleteRes struct{}
