// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// BizLeave is the golang structure for table biz_leave.
type BizLeave struct {
	Id           uint64      `json:"id"           orm:"id"            description:"主键ID"`                                               // 主键ID
	LeaveType    int         `json:"leaveType"    orm:"leave_type"    description:"请假类型:1=事假,2=病假,3=年假,4=调休,5=其他"`                      // 请假类型:1=事假,2=病假,3=年假,4=调休,5=其他
	StartDate    *gtime.Time `json:"startDate"    orm:"start_date"    description:"开始日期"`                                               // 开始日期
	EndDate      *gtime.Time `json:"endDate"      orm:"end_date"      description:"结束日期"`                                               // 结束日期
	Days         float64     `json:"days"         orm:"days"          description:"请假天数(0.5天粒度,申请人填报)"`                                 // 请假天数(0.5天粒度,申请人填报)
	Reason       string      `json:"reason"       orm:"reason"        description:"请假事由"`                                               // 请假事由
	FlowStatus   int         `json:"flowStatus"   orm:"flow_status"   description:"审批状态:0=审批中,1=已通过,2=被退回,3=已撤销,4=已终止(引擎回调写入;未发起时无意义)"` // 审批状态:0=审批中,1=已通过,2=被退回,3=已撤销,4=已终止(引擎回调写入;未发起时无意义)
	FlowInstance uint64      `json:"flowInstance" orm:"flow_instance" description:"流程实例ID(0=未发起/草稿)"`                                   // 流程实例ID(0=未发起/草稿)
	CreateId     uint64      `json:"createId"     orm:"create_id"     description:"创建人ID(ORM自动填充)"`                                     // 创建人ID(ORM自动填充)
	UpdateId     uint64      `json:"updateId"     orm:"update_id"     description:"最后修改人ID(ORM自动填充)"`                                   // 最后修改人ID(ORM自动填充)
	CreatedAt    *gtime.Time `json:"createdAt"    orm:"created_at"    description:"创建时间"`                                               // 创建时间
	UpdatedAt    *gtime.Time `json:"updatedAt"    orm:"updated_at"    description:"更新时间"`                                               // 更新时间
	DeletedAt    *gtime.Time `json:"deletedAt"    orm:"deleted_at"    description:"删除时间(软删)"`                                           // 删除时间(软删)
}
