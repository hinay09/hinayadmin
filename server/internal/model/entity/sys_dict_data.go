// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysDictData is the golang structure for table sys_dict_data.
type SysDictData struct {
	Id        uint64      `json:"id"        orm:"id"         description:"ID"`                      // ID
	TypeId    uint64      `json:"typeId"    orm:"type_id"    description:"字典类型ID(关联sys_dict_type)"` // 字典类型ID(关联sys_dict_type)
	DictLabel string      `json:"dictLabel" orm:"dict_label" description:"字典标签(展示名)"`               // 字典标签(展示名)
	DictValue string      `json:"dictValue" orm:"dict_value" description:"字典键值"`                    // 字典键值
	Sort      int         `json:"sort"      orm:"sort"       description:"排序"`                      // 排序
	Status    int         `json:"status"    orm:"status"     description:"状态:1=启用,0=禁用"`            // 状态:1=启用,0=禁用
	Remark    string      `json:"remark"    orm:"remark"     description:"备注"`                      // 备注
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`                    // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`                    // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`                // 删除时间(软删)
}
