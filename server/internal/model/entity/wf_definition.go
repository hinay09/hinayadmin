// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// WfDefinition is the golang structure for table wf_definition.
type WfDefinition struct {
	Id        uint64      `json:"id"        orm:"id"         description:"主键ID"`                     // 主键ID
	FlowKey   string      `json:"flowKey"   orm:"flow_key"   description:"流程标识(同标识共用一组版本, 如 leave)"` // 流程标识(同标识共用一组版本, 如 leave)
	Name      string      `json:"name"      orm:"name"       description:"流程名称"`                     // 流程名称
	FormConf  string      `json:"formConf"  orm:"form_conf"  description:"表单字段定义 JSON"`              // 表单字段定义 JSON [{key,label,type,options,required}]
	FlowConf  string      `json:"flowConf"  orm:"flow_conf"  description:"节点树定义 JSON"`               // 节点树定义 JSON {id,type,name,child,...}
	Version   int         `json:"version"   orm:"version"    description:"版本:0=草稿,>=1=已发布版本号"`       // 版本:0=草稿,>=1=已发布版本号
	Status    int         `json:"status"    orm:"status"     description:"状态:0=草稿,1=已发布,2=已停用"`      // 状态:0=草稿,1=已发布,2=已停用
	Remark    string      `json:"remark"    orm:"remark"     description:"备注"`                       // 备注
	CreateId  uint64      `json:"createId"  orm:"create_id"  description:"创建人ID(ORM自动填充)"`           // 创建人ID(ORM自动填充)
	UpdateId  uint64      `json:"updateId"  orm:"update_id"  description:"最后修改人ID(ORM自动填充)"`         // 最后修改人ID(ORM自动填充)
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`                     // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`                     // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`                 // 删除时间(软删)
}
