// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// WfDefinition is the golang structure of table wf_definition for DAO operations like Where/Data.
type WfDefinition struct {
	g.Meta    `orm:"table:wf_definition, do:true"`
	Id        any         // 主键ID
	FlowKey   any         // 流程标识(同标识共用一组版本, 如 leave)
	Name      any         // 流程名称
	FormConf  any         // 表单字段定义 JSON [{key,label,type,options,required}]
	FlowConf  any         // 节点树定义 JSON {id,type,name,child,...}
	Version   any         // 版本:0=草稿,>=1=已发布版本号
	Status    any         // 状态:0=草稿,1=已发布,2=已停用
	Remark    any         // 备注
	CreateId  any         // 创建人ID(ORM自动填充)
	UpdateId  any         // 最后修改人ID(ORM自动填充)
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删)
}
