// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/flow/v1"
)

type (
	IFlow interface {
		// DefinitionList 流程定义分页列表。
		DefinitionList(ctx context.Context, in *v1.FlowDefinitionListReq) (res *v1.FlowDefinitionListRes, err error)
		// DefinitionUsable 可发起的流程: 每个 flow_key 取最新已发布且未停用版本。
		DefinitionUsable(ctx context.Context, in *v1.FlowDefinitionUsableReq) (res *v1.FlowDefinitionUsableRes, err error)
		// DefinitionDetail 定义详情。
		DefinitionDetail(ctx context.Context, in *v1.FlowDefinitionDetailReq) (res *v1.FlowDefinitionDetailRes, err error)
		// DefinitionCreate 新增流程定义 (草稿)。
		DefinitionCreate(ctx context.Context, in *v1.FlowDefinitionCreateReq) (res *v1.FlowDefinitionCreateRes, err error)
		// DefinitionUpdate 修改流程定义 (仅草稿)。
		DefinitionUpdate(ctx context.Context, in *v1.FlowDefinitionUpdateReq) (res *v1.FlowDefinitionUpdateRes, err error)
		// DefinitionDelete 删除流程定义 (仅未发布过的草稿; 发布过的版本被实例引用, 不允许删除)。
		DefinitionDelete(ctx context.Context, in *v1.FlowDefinitionDeleteReq) (res *v1.FlowDefinitionDeleteRes, err error)
		// DefinitionPublish 发布: 原地生效 —— 同一行 version+1 且 status=1。
		// 单行模型说明: 发布后可继续编辑 (保存对新发起即时生效); 在途实例走自身快照, 不受影响。
		DefinitionPublish(ctx context.Context, in *v1.FlowDefinitionPublishReq) (res *v1.FlowDefinitionPublishRes, err error)
		// DefinitionDisable 停用一个已发布版本 (不可再发起; 在途实例不受影响)。
		DefinitionDisable(ctx context.Context, in *v1.FlowDefinitionDisableReq) (res *v1.FlowDefinitionDisableRes, err error)
		// DesignerOptions 设计器选项: 启用用户 + 启用角色。
		DesignerOptions(ctx context.Context, in *v1.FlowDesignerOptionsReq) (res *v1.FlowDesignerOptionsRes, err error)
		// InstanceStart 发起流程: 校验定义/表单 → 建实例(携带表单/节点树快照) → 从发起节点推进。
		InstanceStart(ctx context.Context, in *v1.FlowInstanceStartReq) (res *v1.FlowInstanceStartRes, err error)
		// InstanceList 实例列表 (todo/done/mine/ccme 四视角 + all 管理员全局视角)。
		InstanceList(ctx context.Context, in *v1.FlowInstanceListReq) (res *v1.FlowInstanceListRes, err error)
		// InstanceDetail 实例详情: 表单/节点树取实例快照 (存量旧数据兜底定义行) + 任务 + 时间线。
		InstanceDetail(ctx context.Context, in *v1.FlowInstanceDetailReq) (res *v1.FlowInstanceDetailRes, err error)
		// InstanceCancel 发起人撤销流程 (运行中或退回待重提均可撤销)。
		InstanceCancel(ctx context.Context, in *v1.FlowInstanceCancelReq) (res *v1.FlowInstanceCancelRes, err error)
		// InstanceWithdraw 发起人撤回流程 (尚无审批人同意时收回, 待修改后重提)。
		InstanceWithdraw(ctx context.Context, in *v1.FlowInstanceWithdrawReq) (res *v1.FlowInstanceWithdrawRes, err error)
		// InstanceResubmit 重新提交: 仅退回态(6)/已撤销(4)且发起人可操作 —— 撤销后仍可改表单再次发起。
		// 沿用实例快照的流程定义 (可顺带修改表单数据); 重新提交流程从头重走, 历史轮次保留。
		InstanceResubmit(ctx context.Context, in *v1.FlowInstanceResubmitReq) (res *v1.FlowInstanceResubmitRes, err error)
		// InstanceTerminate 管理员终止流程: 运行中实例立即结束, 待办作废、原同意置已失效。
		InstanceTerminate(ctx context.Context, in *v1.FlowInstanceTerminateReq) (res *v1.FlowInstanceTerminateRes, err error)
		// InstanceUrge 发起人催办: 通知实例当前全部待办审批人 (同实例 10 分钟内限一次)。
		InstanceUrge(ctx context.Context, in *v1.FlowInstanceUrgeReq) (res *v1.FlowInstanceUrgeRes, err error)
		// TaskApprove 同意: 或签首签即过节点, 会签须全部同意后过节点并推进。
		TaskApprove(ctx context.Context, in *v1.FlowTaskApproveReq) (res *v1.FlowTaskApproveRes, err error)
		// TaskReject 驳回: 节点级动作 —— 驳回者的待办置已驳回, 同节点其余待办作废。
		// 回退目标二选一: 默认退回发起人 (发起人可修改后重新提交或撤销);
		// targetNodeId 非空时退回到该**已审批节点**重新处理 (实例保持运行, 从该节点起依次重审)。
		// 或签/会签语义一致: 会签任一成员驳回即该节点驳回, 走同一条驳回路径。
		TaskReject(ctx context.Context, in *v1.FlowTaskRejectReq) (res *v1.FlowTaskRejectRes, err error)
		// TaskTransfer 转办: 将我的待办审批任务转给指定人处理。
		// 原任务置已转出 (状态4), 目标人生成同节点新待办; 会签同伴不受影响。
		TaskTransfer(ctx context.Context, in *v1.FlowTaskTransferReq) (res *v1.FlowTaskTransferRes, err error)
		// TaskDelegate 委派: 将我的待办交被委托人先行处理, 其提交意见后回到本人终审。
		// 原任务置已委派(状态7)挂起, 被委托人生成 delegate_from_id 指向原任务的代办待办。
		TaskDelegate(ctx context.Context, in *v1.FlowTaskDelegateReq) (res *v1.FlowTaskDelegateRes, err error)
		// TaskDelegateResolve 委派处理: 被委托人提交处理意见, 任务回到原审批人终审。
		TaskDelegateResolve(ctx context.Context, in *v1.FlowTaskDelegateResolveReq) (res *v1.FlowTaskDelegateResolveRes, err error)
		// TaskAppend 加签: 以我的待办所在节点为锚, 追加必要审批人。
		// 语义 (钉钉式"同时加签"): 节点全部待办转为会签 —— 原处理人与新加人均须同意,
		// 节点才通过 (任一驳回=该节点驳回, 按所选目标回退); 新任务沿用本轮 receive_time, 流程图聚合同一轮次。
		// 已是该节点待办/本轮已同意过的人不再加入 (会签下重复加签会要求其二次同意)。
		TaskAppend(ctx context.Context, in *v1.FlowTaskAppendReq) (res *v1.FlowTaskAppendRes, err error)
		// TaskReduce 减签: 从当前节点移除指定待办审批人 (任务置已作废, 至少保留一人)。
		// 仅移除待办任务; 已同意的不回退 (会签已投的票有效); 被移除人收到待办作废通知。
		TaskReduce(ctx context.Context, in *v1.FlowTaskReduceReq) (res *v1.FlowTaskReduceRes, err error)
		// TaskRead 抄送已读。
		TaskRead(ctx context.Context, in *v1.FlowTaskReadReq) (res *v1.FlowTaskReadRes, err error)
		// TaskCount 我的待办/待阅数量。
		TaskCount(ctx context.Context, in *v1.FlowTaskCountReq) (res *v1.FlowTaskCountRes, err error)
	}
)

var (
	localFlow IFlow
)

func Flow() IFlow {
	if localFlow == nil {
		panic("implement not found for interface IFlow, forgot register?")
	}
	return localFlow
}

func RegisterFlow(i IFlow) {
	localFlow = i
}
