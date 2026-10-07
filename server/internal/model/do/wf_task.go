// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// WfTask is the golang structure of table wf_task for DAO operations like Where/Data.
type WfTask struct {
	g.Meta       `orm:"table:wf_task, do:true"`
	Id           any         // 主键ID
	InstanceId   any         // 实例ID
	NodeId       any         // 节点ID(树内唯一)
	NodeName     any         // 节点名称(冗余)
	NodeType     any         // 节点类型:1=审批,2=抄送
	SignType     any         // 签核方式:1=或签,2=会签
	AssigneeId   any         // 处理人ID
	AssigneeName any         // 处理人昵称(冗余)
	Status       any         // 状态:1=待办,2=已同意,3=已驳回,4=已转出,5=已作废
	Comment      any         // 审批意见
	ReceiveTime  *gtime.Time // 到达时间
	ActedAt      *gtime.Time // 处理时间
	CreatedAt    *gtime.Time // 创建时间
	UpdatedAt    *gtime.Time // 更新时间
}
