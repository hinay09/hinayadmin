// Package v1 自由审批流接口契约。
//
// flow_conf 节点树 JSON 约定 (钉钉式纵向树):
//
//	{
//	  "id": "start", "type": "start", "name": "发起人", "child": { ... }
//	}
//	节点类型: start=发起 / approver=审批 / cc=抄送 / condition=条件分支
//	approver/cc 节点: approverType(user|role|selfSelect|superior|initiator)
//	                 + approverIds([]uint64) + signType(any=或签,all=会签)
//	condition 节点: branches[]{name,isDefault,conditions,child};
//	                 conditions 外层 OR、内层 AND, 条件 {field,op,value},
//	                 op: eq/ne/gt/lt/ge/le/in
//
// form_conf 表单字段 JSON 约定: [{key,label,type,input|number|textarea|select|date,options[],required}]
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"hinay.cn/admin/utility/response"
)

// ============================================================
// 流程定义
// ============================================================

// FlowDefinitionItem 流程定义条目。
type FlowDefinitionItem struct {
	Id        uint64      `json:"id"        dc:"定义版本行ID"`
	FlowKey   string      `json:"flowKey"   dc:"流程标识"`
	Name      string      `json:"name"`
	FormConf  string      `json:"formConf"  dc:"表单字段定义 JSON 字符串"`
	FlowConf  string      `json:"flowConf"  dc:"节点树定义 JSON 字符串"`
	Version   int         `json:"version"   dc:"版本:0=草稿,>=1=已发布版本号"`
	Status    int         `json:"status"    dc:"状态:0=草稿,1=已发布,2=已停用"`
	Remark    string      `json:"remark"`
	CreatedAt *gtime.Time `json:"createdAt"`
	UpdatedAt *gtime.Time `json:"updatedAt"`
}

// FlowDefinitionListReq 流程定义分页列表。
type FlowDefinitionListReq struct {
	g.Meta   `path:"/flow/definitions" tags:"Flow" method:"get" summary:"流程定义列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"名称/标识模糊搜索"`
	Status   *int   `json:"status"   in:"query" dc:"状态:0=草稿,1=已发布,2=已停用"`
	Version  *int   `json:"version"  in:"query" dc:"版本号精确筛选:0=草稿,N=vN"`
	Page     int    `json:"page"     in:"query" d:"1"`
	PageSize int    `json:"pageSize" in:"query" d:"10"`
}

// FlowDefinitionListRes 列表响应。
type FlowDefinitionListRes response.PageResult

// FlowDefinitionUsableReq 可发起的流程 (每个 flow_key 取最新已发布且未停用版本)。
type FlowDefinitionUsableReq struct {
	g.Meta `path:"/flow/definitions/usable" tags:"Flow" method:"get" summary:"可发起的流程列表"`
}

// FlowDefinitionUsableRes 可发起列表响应 (含表单/节点树配置, 供发起弹窗渲染)。
type FlowDefinitionUsableRes struct {
	List []*FlowDefinitionItem `json:"list"`
}

// FlowDefinitionDetailReq 定义详情。
type FlowDefinitionDetailReq struct {
	g.Meta `path:"/flow/definitions/{id}" tags:"Flow" method:"get" summary:"流程定义详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// FlowDefinitionDetailRes 详情响应。
type FlowDefinitionDetailRes struct {
	*FlowDefinitionItem
}

// FlowDefinitionCreateReq 新增流程定义 (草稿)。
type FlowDefinitionCreateReq struct {
	g.Meta   `path:"/flow/definitions" tags:"Flow" method:"post" summary:"新增流程定义"`
	FlowKey  string `json:"flowKey"  v:"length:0,64#流程标识长度 0-64"`
	Name     string `json:"name"     v:"required|length:1,128#请输入流程名称|名称长度 1-128"`
	FormConf string `json:"formConf" dc:"表单字段定义 JSON 字符串"`
	FlowConf string `json:"flowConf" dc:"节点树定义 JSON 字符串"`
	Remark   string `json:"remark"`
}

// FlowDefinitionCreateRes 新增响应。
type FlowDefinitionCreateRes struct {
	Id uint64 `json:"id"`
}

// FlowDefinitionUpdateReq 修改流程定义 (仅草稿可改)。
type FlowDefinitionUpdateReq struct {
	g.Meta   `path:"/flow/definitions/{id}" tags:"Flow" method:"put" summary:"修改流程定义(仅草稿)"`
	Id       uint64 `json:"id" in:"path" v:"required"`
	Name     string `json:"name"     v:"required|length:1,128#请输入流程名称|名称长度 1-128"`
	FlowKey  string `json:"flowKey"  v:"length:0,64#流程标识长度 0-64"`
	FormConf string `json:"formConf"`
	FlowConf string `json:"flowConf"`
	Remark   string `json:"remark"`
}

// FlowDefinitionUpdateRes 修改响应。
type FlowDefinitionUpdateRes struct{}

// FlowDefinitionDeleteReq 删除流程定义 (仅草稿可删)。
type FlowDefinitionDeleteReq struct {
	g.Meta `path:"/flow/definitions/{id}" tags:"Flow" method:"delete" summary:"删除流程定义(仅草稿)"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// FlowDefinitionDeleteRes 删除响应。
type FlowDefinitionDeleteRes struct{}

// FlowDefinitionPublishReq 发布定义: 以草稿当前内容生成新版本行 (已发布版本不可变)。
type FlowDefinitionPublishReq struct {
	g.Meta `path:"/flow/definitions/{id}/publish" tags:"Flow" method:"post" summary:"发布流程定义"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// FlowDefinitionPublishRes 发布响应。
type FlowDefinitionPublishRes struct {
	Id      uint64 `json:"id"      dc:"新版本行ID"`
	Version int    `json:"version" dc:"新版本号"`
}

// FlowDefinitionDisableReq 停用一个已发布版本 (停用后不可再发起, 在途实例不受影响)。
type FlowDefinitionDisableReq struct {
	g.Meta `path:"/flow/definitions/{id}/disable" tags:"Flow" method:"post" summary:"停用流程定义版本"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// FlowDefinitionDisableRes 停用响应。
type FlowDefinitionDisableRes struct{}

// ============================================================
// 设计器选项
// ============================================================

// FlowUserOption 用户选项。
type FlowUserOption struct {
	Id       uint64 `json:"id"`
	Nickname string `json:"nickname"`
	Username string `json:"username"`
}

// FlowRoleOption 角色选项。
type FlowRoleOption struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// FlowPostOption 岗位选项。
type FlowPostOption struct {
	Id   uint64 `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Kind int    `json:"kind" dc:"1=普通岗,2=主管岗"`
}

// FlowDesignerOptionsReq 设计器/发起弹窗的选项数据。
type FlowDesignerOptionsReq struct {
	g.Meta `path:"/flow/designer/options" tags:"Flow" method:"get" summary:"设计器选项(用户/角色)"`
}

// FlowDesignerOptionsRes 选项响应。
type FlowDesignerOptionsRes struct {
	Users []*FlowUserOption `json:"users"`
	Roles []*FlowRoleOption `json:"roles"`
	Posts []*FlowPostOption `json:"posts"`
}

// ============================================================
// 流程实例
// ============================================================

// FlowInstanceItem 实例条目。
type FlowInstanceItem struct {
	Id              uint64      `json:"id"`
	DefinitionId    uint64      `json:"definitionId"`
	FlowKey         string      `json:"flowKey"`
	BizId           uint64      `json:"bizId"    dc:"业务关联ID, 0=审批中心直接发起"`
	FlowName        string      `json:"flowName"`
	Title           string      `json:"title"`
	Status          int         `json:"status" dc:"1=运行中,2=已通过,4=已撤销,5=已终止,6=已退回待重提,7=已撤回待重提"`
	StartUserId     uint64      `json:"startUserId"`
	StartUserName   string      `json:"startUserName"`
	CurrentNodes    string      `json:"currentNodes" dc:"当前节点名(逗号分隔, 运行中才有)"`
	TaskId          uint64      `json:"taskId" dc:"当前用户的相关任务ID(待办/待阅), 0=无"`
	TaskStatus      int         `json:"taskStatus" dc:"当前用户相关任务状态: 1=待办/待阅, 2=已同意(抄送=已阅), 3=已驳回, 6=已失效, 0=无"`
	TaskReceiveTime *gtime.Time `json:"taskReceiveTime" dc:"任务到达时间(待办停留时长/待阅未读时长的计算起点)"`
	CreatedAt       *gtime.Time `json:"createdAt"`
	FinishedAt      *gtime.Time `json:"finishedAt"`
}

// FlowInstanceStartReq 发起流程。
type FlowInstanceStartReq struct {
	g.Meta       `path:"/flow/instances" tags:"Flow" method:"post" summary:"发起流程"`
	DefinitionId uint64              `json:"definitionId" v:"required#请选择流程定义"`
	Title        string              `json:"title"        v:"required|length:1,128#请输入申请标题|标题长度 1-128"`
	FormData     map[string]any      `json:"formData"     dc:"表单数据"`
	SelfSelects  map[string][]uint64 `json:"selfSelects" dc:"发起人自选审批人 {nodeId: [userId]}"`
}

// FlowInstanceStartRes 发起响应。
type FlowInstanceStartRes struct {
	Id uint64 `json:"id"`
}

// FlowInstanceListReq 实例列表 (按 scope 切换视角; all=管理员全局视角, 跨用户列出全部实例)。
type FlowInstanceListReq struct {
	g.Meta     `path:"/flow/instances" tags:"Flow" method:"get" summary:"流程实例列表"`
	Scope      string `json:"scope"    in:"query" d:"todo" v:"in:todo,done,mine,ccme,all#视角取值 todo/done/mine/ccme/all"`
	Keyword    string `json:"keyword"  in:"query"`
	Status     *int   `json:"status"   in:"query" dc:"实例状态筛选(mine/all 视角可用)"`
	FlowKey    string `json:"flowKey"  in:"query" dc:"流程标识精确筛选(仅 all 视角)"`
	TaskStatus *int   `json:"taskStatus" in:"query" dc:"任务状态筛选(仅 ccme 视角: 1=未读,2=已读)"`
	Page       int    `json:"page"     in:"query" d:"1"`
	PageSize   int    `json:"pageSize" in:"query" d:"10"`
}

// FlowInstanceListRes 列表响应。
type FlowInstanceListRes response.PageResult

// FlowTaskItem 任务条目。
type FlowTaskItem struct {
	Id             uint64      `json:"id"`
	InstanceId     uint64      `json:"instanceId"`
	NodeId         string      `json:"nodeId"`
	NodeName       string      `json:"nodeName"`
	NodeType       int         `json:"nodeType" dc:"1=审批,2=抄送"`
	SignType       int         `json:"signType" dc:"1=或签,2=会签"`
	AssigneeId     uint64      `json:"assigneeId"`
	AssigneeName   string      `json:"assigneeName"`
	DelegateFromId uint64      `json:"delegateFromId" dc:"委派来源任务ID: >0=本人被委派的代办任务, 处理后回到原审批人"`
	Status         int         `json:"status" dc:"1=待办,2=已同意,3=已驳回,4=已转出,5=已作废,6=已失效,7=已委派,8=委办完成"`
	Comment        string      `json:"comment"`
	ReceiveTime    *gtime.Time `json:"receiveTime"`
	DueTime        *gtime.Time `json:"dueTime" dc:"办理期限 (节点超时配置物化, NULL=不限)"`
	ActedAt        *gtime.Time `json:"actedAt"`
}

// FlowRecordItem 流转记录条目。
type FlowRecordItem struct {
	Id           uint64      `json:"id"`
	NodeName     string      `json:"nodeName"`
	Action       string      `json:"action" dc:"submit/resubmit/approve/reject/back/cancel/withdraw/cc/finish/transfer/delegate/delegateResolve/terminate/urge/append/reduce/timeoutRemind/timeoutTransfer/timeoutApprove"`
	OperatorId   uint64      `json:"operatorId" dc:"0=系统"`
	OperatorName string      `json:"operatorName"`
	Comment      string      `json:"comment"`
	CreatedAt    *gtime.Time `json:"createdAt"`
}

// FlowInstanceDetailReq 实例详情 (表单 + 任务 + 时间线)。
type FlowInstanceDetailReq struct {
	g.Meta `path:"/flow/instances/{id}" tags:"Flow" method:"get" summary:"流程实例详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// FlowInstanceDetailRes 详情响应。
type FlowInstanceDetailRes struct {
	Instance        *FlowInstanceItem   `json:"instance"`
	FormConf        string              `json:"formConf"`
	FormData        string              `json:"formData" dc:"提交的表单数据 JSON 字符串"`
	FlowConf        string              `json:"flowConf"`
	Tasks           []*FlowTaskItem     `json:"tasks"`
	Records         []*FlowRecordItem   `json:"records"`
	MyPendingTaskId uint64              `json:"myPendingTaskId" dc:"当前用户待办任务ID, 0=无"`
	MyCcTaskId      uint64              `json:"myCcTaskId"      dc:"当前用户待阅任务ID, 0=无"`
	RejectTargets   []*FlowRejectTarget `json:"rejectTargets"   dc:"当前待办可驳回到的目标节点 (历史已通过的审批节点)"`
	CanCancel       bool                `json:"canCancel"       dc:"当前用户是否可撤销"`
	CanWithdraw     bool                `json:"canWithdraw"     dc:"当前用户是否可撤回(发起人+运行中+尚无任何审批人同意)"`
	CanResubmit     bool                `json:"canResubmit"     dc:"当前用户是否可重新提交(退回态/已撤销/已撤回的发起人)"`
	PrevSelfSelects map[string][]uint64 `json:"prevSelfSelects" dc:"上一轮自选节点已选审批人 {nodeId: [userId]}, 重提弹窗默认值"`
}

// FlowInstanceResubmitReq 发起人修改后重新提交 (退回态/已撤销均可), 流程从头重走。
type FlowInstanceResubmitReq struct {
	g.Meta      `path:"/flow/instances/{id}/resubmit" tags:"Flow" method:"post" summary:"重新提交(退回/撤销后)"`
	Id          uint64              `json:"id" in:"path" v:"required"`
	FormData    map[string]any      `json:"formData" dc:"修改后的表单数据 (不传则沿用原数据)"`
	SelfSelects map[string][]uint64 `json:"selfSelects" dc:"自选审批人 {nodeId: [userId]}"`
}

// FlowInstanceResubmitRes 重提响应。
type FlowInstanceResubmitRes struct{}

// FlowInstanceCancelReq 发起人撤销流程。
type FlowInstanceCancelReq struct {
	g.Meta `path:"/flow/instances/{id}/cancel" tags:"Flow" method:"post" summary:"撤销流程(发起人)"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// FlowInstanceCancelRes 撤销响应。
type FlowInstanceCancelRes struct{}

// FlowInstanceWithdrawReq 发起人撤回流程: 尚无任何审批人同意时收回, 实例转入已撤回态(7)。
// 与撤销的区别: 撤销是终态(业务回调 OnCanceled); 撤回是"收回待改"——实例未结束,
// 修改表单后可重新提交 (复用 InstanceResubmit, 业务回调 OnWithdrawn, 未注册时回退 OnReturned)。
type FlowInstanceWithdrawReq struct {
	g.Meta `path:"/flow/instances/{id}/withdraw" tags:"Flow" method:"post" summary:"撤回流程(发起人,尚无审批时)"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// FlowInstanceWithdrawRes 撤回响应。
type FlowInstanceWithdrawRes struct{}

// FlowInstanceTerminateReq 终止流程 (管理员): 运行中实例立即结束, 待办作废。
type FlowInstanceTerminateReq struct {
	g.Meta  `path:"/flow/instances/{id}/terminate" tags:"Flow" method:"post" summary:"终止流程(管理员)"`
	Id      uint64 `json:"id"      in:"path" v:"required"`
	Comment string `json:"comment" dc:"终止原因"`
}

// FlowInstanceTerminateRes 终止响应。
type FlowInstanceTerminateRes struct{}

// FlowInstanceUrgeReq 发起人催办: 通知当前所有待办审批人 (同实例 10 分钟限一次)。
type FlowInstanceUrgeReq struct {
	g.Meta  `path:"/flow/instances/{id}/urge" tags:"Flow" method:"post" summary:"催办(发起人)"`
	Id      uint64 `json:"id"      in:"path" v:"required"`
	Comment string `json:"comment" dc:"催办说明 (选填)"`
}

// FlowInstanceUrgeRes 催办响应。
type FlowInstanceUrgeRes struct{}

// ============================================================
// 审批任务
// ============================================================

// FlowTaskApproveReq 同意。
type FlowTaskApproveReq struct {
	g.Meta  `path:"/flow/tasks/{id}/approve" tags:"Flow" method:"post" summary:"同意"`
	Id      uint64 `json:"id" in:"path" v:"required"`
	Comment string `json:"comment"`
}

// FlowTaskApproveRes 同意响应。
type FlowTaskApproveRes struct{}

// FlowRejectTarget 可驳回到的目标节点 (本实例历史已通过的审批节点)。
type FlowRejectTarget struct {
	NodeId   string `json:"nodeId"`
	NodeName string `json:"nodeName"`
}

// FlowTaskRejectReq 驳回: 节点级动作 (或签/会签一致, 同节点其余待办一并作废)。
// 默认退回发起人; targetNodeId 指定已审批节点则退回该节点重新处理 (实例保持运行)。
type FlowTaskRejectReq struct {
	g.Meta       `path:"/flow/tasks/{id}/reject" tags:"Flow" method:"post" summary:"驳回"`
	Id           uint64 `json:"id"           in:"path" v:"required"`
	Comment      string `json:"comment"      dc:"驳回意见 (必填)"`
	TargetNodeId string `json:"targetNodeId" dc:"驳回到的已审批节点ID, 空=退回发起人修改后重提"`
}

// FlowTaskRejectRes 驳回响应。
type FlowTaskRejectRes struct{}

// FlowTaskTransferReq 转办: 将我的待办审批任务转给他人处理 (原任务置已转出, 新任务落目标人)。
type FlowTaskTransferReq struct {
	g.Meta       `path:"/flow/tasks/{id}/transfer" tags:"Flow" method:"post" summary:"转办"`
	Id           uint64 `json:"id"           in:"path" v:"required"`
	TargetUserId uint64 `json:"targetUserId" v:"required#请选择转办对象"`
	Comment      string `json:"comment"      dc:"转办说明 (选填)"`
}

// FlowTaskTransferRes 转办响应。
type FlowTaskTransferRes struct{}

// FlowTaskDelegateReq 委派: 将我的待办交给被委托人先行处理。
// 与转办的区别: 转办后原审批人出局; 委派后被委托人仅提交处理意见 (委派处理),
// 任务回到原审批人做最终同意/驳回 —— 被委派的代办任务不能再委派/转办。
type FlowTaskDelegateReq struct {
	g.Meta       `path:"/flow/tasks/{id}/delegate" tags:"Flow" method:"post" summary:"委派(代办后回到原审批人)"`
	Id           uint64 `json:"id"           in:"path" v:"required"`
	TargetUserId uint64 `json:"targetUserId" v:"required#请选择被委托人"`
	Comment      string `json:"comment"      dc:"委派说明 (选填)"`
}

// FlowTaskDelegateRes 委派响应。
type FlowTaskDelegateRes struct{}

// FlowTaskDelegateResolveReq 委派处理: 被委托人提交处理意见, 任务回到原审批人终审。
type FlowTaskDelegateResolveReq struct {
	g.Meta  `path:"/flow/tasks/{id}/delegateResolve" tags:"Flow" method:"post" summary:"委派处理(被委托人提交意见)"`
	Id      uint64 `json:"id"      in:"path" v:"required"`
	Comment string `json:"comment" dc:"处理意见 (选填)"`
}

// FlowTaskDelegateResolveRes 委派处理响应。
type FlowTaskDelegateResolveRes struct{}

// FlowTaskAppendReq 加签: 以我的待办所在节点为锚追加必要审批人。
// 语义 (钉钉式"同时加签"): 节点全部待办转为会签 —— 原处理人与新加人均须同意, 节点才通过。
type FlowTaskAppendReq struct {
	g.Meta  `path:"/flow/tasks/{id}/append" tags:"Flow" method:"post" summary:"加签"`
	Id      uint64   `json:"id"      in:"path" v:"required"`
	UserIds []uint64 `json:"userIds" v:"required#请选择加签人"`
	Comment string   `json:"comment" dc:"加签说明 (选填)"`
}

// FlowTaskAppendRes 加签响应。
type FlowTaskAppendRes struct{}

// FlowTaskReduceReq 减签: 从当前节点移除指定待办审批人 (任务置已作废, 至少保留一人)。
type FlowTaskReduceReq struct {
	g.Meta  `path:"/flow/tasks/{id}/reduce" tags:"Flow" method:"post" summary:"减签"`
	Id      uint64   `json:"id"      in:"path" v:"required"`
	UserIds []uint64 `json:"userIds" v:"required#请选择要移除的审批人"`
	Comment string   `json:"comment" dc:"减签说明 (选填)"`
}

// FlowTaskReduceRes 减签响应。
type FlowTaskReduceRes struct{}

// FlowTaskReadReq 抄送已读。
type FlowTaskReadReq struct {
	g.Meta `path:"/flow/tasks/{id}/read" tags:"Flow" method:"put" summary:"抄送已读"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// FlowTaskReadRes 已读响应。
type FlowTaskReadRes struct{}

// FlowTaskCountReq 我的待办/待阅数量。
type FlowTaskCountReq struct {
	g.Meta `path:"/flow/tasks/count" tags:"Flow" method:"get" summary:"待办/待阅数量"`
}

// FlowTaskCountRes 数量响应。
type FlowTaskCountRes struct {
	Todo int `json:"todo"`
	Cc   int `json:"cc"`
}
