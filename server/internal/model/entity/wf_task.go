// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// WfTask is the golang structure for table wf_task.
type WfTask struct {
	Id           uint64      `json:"id"           orm:"id"            description:"主键ID"`                            // 主键ID
	InstanceId   uint64      `json:"instanceId"   orm:"instance_id"   description:"实例ID"`                            // 实例ID
	NodeId       string      `json:"nodeId"       orm:"node_id"       description:"节点ID(树内唯一)"`                      // 节点ID(树内唯一)
	NodeName     string      `json:"nodeName"     orm:"node_name"     description:"节点名称(冗余)"`                        // 节点名称(冗余)
	NodeType     int         `json:"nodeType"     orm:"node_type"     description:"节点类型:1=审批,2=抄送"`                  // 节点类型:1=审批,2=抄送
	SignType     int         `json:"signType"     orm:"sign_type"     description:"签核方式:1=或签,2=会签"`                  // 签核方式:1=或签,2=会签
	AssigneeId   uint64      `json:"assigneeId"   orm:"assignee_id"   description:"处理人ID"`                           // 处理人ID
	AssigneeName string      `json:"assigneeName" orm:"assignee_name" description:"处理人昵称(冗余)"`                       // 处理人昵称(冗余)
	Status       int         `json:"status"       orm:"status"        description:"状态:1=待办,2=已同意,3=已驳回,4=已转出,5=已作废"` // 状态:1=待办,2=已同意,3=已驳回,4=已转出,5=已作废
	Comment      string      `json:"comment"      orm:"comment"       description:"审批意见"`                            // 审批意见
	ReceiveTime  *gtime.Time `json:"receiveTime"  orm:"receive_time"  description:"到达时间"`                            // 到达时间
	ActedAt      *gtime.Time `json:"actedAt"      orm:"acted_at"      description:"处理时间"`                            // 处理时间
	CreatedAt    *gtime.Time `json:"createdAt"    orm:"created_at"    description:"创建时间"`                            // 创建时间
	UpdatedAt    *gtime.Time `json:"updatedAt"    orm:"updated_at"    description:"更新时间"`                            // 更新时间
}
