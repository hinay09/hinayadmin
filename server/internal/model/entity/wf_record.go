// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// WfRecord is the golang structure for table wf_record.
type WfRecord struct {
	Id           uint64      `json:"id"           orm:"id"            description:"主键ID"`                                      // 主键ID
	InstanceId   uint64      `json:"instanceId"   orm:"instance_id"   description:"实例ID"`                                      // 实例ID
	TaskId       uint64      `json:"taskId"       orm:"task_id"       description:"关联任务ID(无则为0)"`                              // 关联任务ID(无则为0)
	NodeId       string      `json:"nodeId"       orm:"node_id"       description:"节点ID"`                                      // 节点ID
	NodeName     string      `json:"nodeName"     orm:"node_name"     description:"节点名称"`                                      // 节点名称
	Action       string      `json:"action"       orm:"action"        description:"动作:submit/approve/reject/cancel/cc/finish"` // 动作:submit/approve/reject/cancel/cc/finish
	OperatorId   uint64      `json:"operatorId"   orm:"operator_id"   description:"操作人ID(0=系统)"`                               // 操作人ID(0=系统)
	OperatorName string      `json:"operatorName" orm:"operator_name" description:"操作人昵称(0=系统)"`                               // 操作人昵称(0=系统)
	Comment      string      `json:"comment"      orm:"comment"       description:"备注/意见"`                                     // 备注/意见
	CreatedAt    *gtime.Time `json:"createdAt"    orm:"created_at"    description:"创建时间"`                                      // 创建时间
	UpdatedAt    *gtime.Time `json:"updatedAt"    orm:"updated_at"    description:"更新时间"`                                      // 更新时间
}
