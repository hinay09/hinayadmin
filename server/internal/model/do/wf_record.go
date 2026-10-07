// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// WfRecord is the golang structure of table wf_record for DAO operations like Where/Data.
type WfRecord struct {
	g.Meta       `orm:"table:wf_record, do:true"`
	Id           any         // 主键ID
	InstanceId   any         // 实例ID
	TaskId       any         // 关联任务ID(无则为0)
	NodeId       any         // 节点ID
	NodeName     any         // 节点名称
	Action       any         // 动作:submit/approve/reject/cancel/cc/finish
	OperatorId   any         // 操作人ID(0=系统)
	OperatorName any         // 操作人昵称(0=系统)
	Comment      any         // 备注/意见
	CreatedAt    *gtime.Time // 创建时间
	UpdatedAt    *gtime.Time // 更新时间
}
