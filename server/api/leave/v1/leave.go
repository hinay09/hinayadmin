// Package v1 请假申请接口契约 —— 业务型流程审批 Demo。
//
// 业务与流程分离: 业务增删改是业务自己的事 (草稿态, 发起审批前自由操作);
// 「提交审批」是独立动作 —— 前端复用公共弹窗 <FlowSubmitDialog flow-key="biz_leave"
// :show-form="false" :show-title="false">, 表单数据由后端从业务表组装,
// 弹窗产出 (含"发起人自选"节点的 selfSelects) 通过本模块透传给 flow.StartForBiz;
// 审批状态由 flow.RegisterBizListener 回调写回 flow_status (见 logic/leave)。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"hinay.cn/admin/utility/response"
)

// LeaveItem 请假申请条目。
type LeaveItem struct {
	Id           uint64      `json:"id"`
	LeaveType    int         `json:"leaveType"   dc:"请假类型:1=事假,2=病假,3=年假,4=调休,5=其他"`
	StartDate    string      `json:"startDate"   dc:"开始日期 YYYY-MM-DD"`
	EndDate      string      `json:"endDate"     dc:"结束日期 YYYY-MM-DD"`
	Days         float64     `json:"days"        dc:"请假天数"`
	Reason       string      `json:"reason"`
	FlowStatus   int         `json:"flowStatus"  dc:"审批状态:0=审批中,1=已通过,2=被退回,3=已撤销,4=已终止"`
	FlowInstance uint64      `json:"flowInstance" dc:"流程实例ID,0=未发起(草稿)"`
	CreateId     uint64      `json:"createId"`
	CreateName   string      `json:"createName"  dc:"申请人"`
	CreatedAt    *gtime.Time `json:"createdAt"`
}

// LeaveListReq 请假申请分页列表 (admin 看全部, 其余仅本人)。
type LeaveListReq struct {
	g.Meta     `path:"/leaves" tags:"Leave" method:"get" summary:"请假申请列表"`
	LeaveType  *int `json:"leaveType"  in:"query" dc:"请假类型"`
	FlowStatus *int `json:"flowStatus" in:"query" dc:"审批状态"`
	Mine       *int `json:"mine"       in:"query" dc:"admin视角:1=只看我发起的"`
	Page       int  `json:"page"       in:"query" d:"1"`
	PageSize   int  `json:"pageSize"   in:"query" d:"10"`
}

// LeaveListRes 列表响应。
type LeaveListRes response.PageResult

// LeaveCreateReq 新增请假申请 (保存为草稿, 不发起审批)。
type LeaveCreateReq struct {
	g.Meta    `path:"/leaves" tags:"Leave" method:"post" summary:"新增请假申请"`
	LeaveType int     `json:"leaveType" v:"required|in:1,2,3,4,5#请选择请假类型|请假类型仅能为 1-5"`
	StartDate string  `json:"startDate" v:"required|date-format:Y-m-d#请选择开始日期|日期格式 YYYY-MM-DD"`
	EndDate   string  `json:"endDate"   v:"required|date-format:Y-m-d#请选择结束日期|日期格式 YYYY-MM-DD"`
	Days      float64 `json:"days"      v:"required|min:0.5|max:99.5#请填写请假天数|至少 0.5 天|最多 99.5 天"`
	Reason    string  `json:"reason"    v:"required|length:1,200#请填写请假事由|事由长度 1-200"`
}

// LeaveCreateRes 新增响应。
type LeaveCreateRes struct {
	Id uint64 `json:"id"`
}

// LeaveUpdateReq 修改请假申请 (仅未发起/被退回/已撤销的单子)。
type LeaveUpdateReq struct {
	g.Meta    `path:"/leaves/{id}" tags:"Leave" method:"put" summary:"修改请假申请"`
	Id        uint64  `json:"id"        in:"path" v:"required"`
	LeaveType int     `json:"leaveType" v:"required|in:1,2,3,4,5#请选择请假类型|请假类型仅能为 1-5"`
	StartDate string  `json:"startDate" v:"required|date-format:Y-m-d#请选择开始日期|日期格式 YYYY-MM-DD"`
	EndDate   string  `json:"endDate"   v:"required|date-format:Y-m-d#请选择结束日期|日期格式 YYYY-MM-DD"`
	Days      float64 `json:"days"      v:"required|min:0.5|max:99.5#请填写请假天数|至少 0.5 天|最多 99.5 天"`
	Reason    string  `json:"reason"    v:"required|length:1,200#请填写请假事由|事由长度 1-200"`
}

// LeaveUpdateRes 修改响应。
type LeaveUpdateRes struct{}

// LeaveDeleteReq 删除请假申请 (仅未发起/已结束的单子, 审批中请先撤销)。
type LeaveDeleteReq struct {
	g.Meta `path:"/leaves/{id}" tags:"Leave" method:"delete" summary:"删除请假申请"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// LeaveDeleteRes 删除响应。
type LeaveDeleteRes struct{}

// LeaveSubmitReq 提交审批 (由公共弹窗 FlowSubmitDialog 触发):
// 草稿首次发起走 StartForBiz; 被退回/已撤销复用原实例重新提交 (流程从头重走, 历史保留)。
// selfSelects 为弹窗收集的"发起人自选"节点审批人 (流程定义含自选节点时弹窗自动渲染选人 UI,
// 业务页无需关心), 无自选节点可传空。
type LeaveSubmitReq struct {
	g.Meta      `path:"/leaves/{id}/submit" tags:"Leave" method:"post" summary:"提交审批"`
	Id          uint64              `json:"id" in:"path" v:"required"`
	SelfSelects map[string][]uint64 `json:"selfSelects" dc:"自选节点审批人 {nodeId: [userId]}"`
}

// LeaveSubmitRes 提交响应。
type LeaveSubmitRes struct {
	FlowInstance uint64 `json:"flowInstance" dc:"流程实例ID"`
}

// LeaveCancelReq 撤销审批 (发起人, 运行中或被退回态); 状态经 OnCanceled 回调写回。
type LeaveCancelReq struct {
	g.Meta `path:"/leaves/{id}/cancel" tags:"Leave" method:"post" summary:"撤销审批"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// LeaveCancelRes 撤销响应。
type LeaveCancelRes struct{}
