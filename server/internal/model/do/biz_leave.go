// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// BizLeave is the golang structure of table biz_leave for DAO operations like Where/Data.
type BizLeave struct {
	g.Meta       `orm:"table:biz_leave, do:true"`
	Id           any         // 主键ID
	LeaveType    any         // 请假类型:1=事假,2=病假,3=年假,4=调休,5=其他
	StartDate    *gtime.Time // 开始日期
	EndDate      *gtime.Time // 结束日期
	Days         any         // 请假天数(0.5天粒度,申请人填报)
	Reason       any         // 请假事由
	FlowStatus   any         // 审批状态:0=审批中,1=已通过,2=被退回,3=已撤销,4=已终止(引擎回调写入;未发起时无意义)
	FlowInstance any         // 流程实例ID(0=未发起/草稿)
	CreateId     any         // 创建人ID(ORM自动填充)
	UpdateId     any         // 最后修改人ID(ORM自动填充)
	CreatedAt    *gtime.Time // 创建时间
	UpdatedAt    *gtime.Time // 更新时间
	DeletedAt    *gtime.Time // 删除时间(软删)
}
