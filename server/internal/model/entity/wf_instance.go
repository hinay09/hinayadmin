// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// WfInstance is the golang structure for table wf_instance.
type WfInstance struct {
	Id             uint64      `json:"id"              orm:"id"                description:"主键ID"`                              // 主键ID
	DefinitionId   uint64      `json:"definitionId"    orm:"definition_id"     description:"定义版本行ID"`                           // 定义版本行ID
	FlowKey        string      `json:"flowKey"         orm:"flow_key"           description:"流程标识(冗余)"`                         // 流程标识(冗余)
	FlowName       string      `json:"flowName"        orm:"flow_name"          description:"流程名称(冗余)"`                         // 流程名称(冗余)
	BizId          uint64      `json:"bizId"           orm:"biz_id"             description:"业务关联ID(0=审批中心直接发起)"`               // 业务关联ID(0=审批中心直接发起)
	Title          string      `json:"title"           orm:"title"              description:"申请标题"`                             // 申请标题
	FormData       string      `json:"formData"        orm:"form_data"          description:"提交的表单数据 JSON"`                     // 提交的表单数据 JSON
	FormConf       string      `json:"formConf"         orm:"form_conf"           description:"表单定义快照(发起时从定义复制)"`               // 表单定义快照(发起时从定义复制)
	FlowConf       string      `json:"flowConf"         orm:"flow_conf"           description:"节点树快照(发起时从定义复制, 驳回重提沿用)"`        // 节点树快照(发起时从定义复制, 驳回重提沿用)
	CurrentNodeIds string      `json:"currentNodeIds"  orm:"current_node_ids"   description:"当前活跃节点ID(逗号分隔)"`                   // 当前活跃节点ID(逗号分隔)
	Status         int         `json:"status"          orm:"status"             description:"状态:1=运行中,2=已通过,3=已驳回,4=已撤销,5=已终止"` // 状态:1=运行中,2=已通过,3=已驳回,4=已撤销,5=已终止
	StartUserId    uint64      `json:"startUserId"     orm:"start_user_id"      description:"发起人ID"`                            // 发起人ID
	StartUserName  string      `json:"startUserName"   orm:"start_user_name"    description:"发起人昵称(冗余)"`                        // 发起人昵称(冗余)
	FinishedAt     *gtime.Time `json:"finishedAt"      orm:"finished_at"        description:"结束时间"`                             // 结束时间
	CreateId       uint64      `json:"createId"        orm:"create_id"          description:"创建人ID(ORM自动填充)"`                   // 创建人ID(ORM自动填充)
	UpdateId       uint64      `json:"updateId"        orm:"update_id"          description:"最后修改人ID(ORM自动填充)"`                 // 最后修改人ID(ORM自动填充)
	CreatedAt      *gtime.Time `json:"createdAt"       orm:"created_at"         description:"创建时间"`                             // 创建时间
	UpdatedAt      *gtime.Time `json:"updatedAt"       orm:"updated_at"         description:"更新时间"`                             // 更新时间
	DeletedAt      *gtime.Time `json:"deletedAt"       orm:"deleted_at"         description:"删除时间(软删)"`                         // 删除时间(软删)
}
