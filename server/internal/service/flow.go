// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish to manually maintain this interface file.
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
		// DefinitionUsable 可发起的流程 (每个 flow_key 最新已发布且未停用版本, 含表单/节点树配置)。
		DefinitionUsable(ctx context.Context, in *v1.FlowDefinitionUsableReq) (res *v1.FlowDefinitionUsableRes, err error)
		// DefinitionDetail 定义详情。
		DefinitionDetail(ctx context.Context, in *v1.FlowDefinitionDetailReq) (res *v1.FlowDefinitionDetailRes, err error)
		// DefinitionCreate 新增流程定义 (草稿)。
		DefinitionCreate(ctx context.Context, in *v1.FlowDefinitionCreateReq) (res *v1.FlowDefinitionCreateRes, err error)
		// DefinitionUpdate 修改流程定义 (仅草稿)。
		DefinitionUpdate(ctx context.Context, in *v1.FlowDefinitionUpdateReq) (res *v1.FlowDefinitionUpdateRes, err error)
		// DefinitionDelete 删除流程定义 (仅草稿, 软删)。
		DefinitionDelete(ctx context.Context, in *v1.FlowDefinitionDeleteReq) (res *v1.FlowDefinitionDeleteRes, err error)
		// DefinitionPublish 发布: 以草稿内容生成新版本行。
		DefinitionPublish(ctx context.Context, in *v1.FlowDefinitionPublishReq) (res *v1.FlowDefinitionPublishRes, err error)
		// DefinitionDisable 停用一个已发布版本。
		DefinitionDisable(ctx context.Context, in *v1.FlowDefinitionDisableReq) (res *v1.FlowDefinitionDisableRes, err error)
		// DesignerOptions 设计器选项 (启用用户 + 启用角色)。
		DesignerOptions(ctx context.Context, in *v1.FlowDesignerOptionsReq) (res *v1.FlowDesignerOptionsRes, err error)
		// InstanceStart 发起流程: 建实例 + 从首节点推进。
		InstanceStart(ctx context.Context, in *v1.FlowInstanceStartReq) (res *v1.FlowInstanceStartRes, err error)
		// InstanceList 实例列表 (todo/done/mine/ccme 四视角)。
		InstanceList(ctx context.Context, in *v1.FlowInstanceListReq) (res *v1.FlowInstanceListRes, err error)
		// InstanceDetail 实例详情: 表单 + 任务 + 流转时间线。
		InstanceDetail(ctx context.Context, in *v1.FlowInstanceDetailReq) (res *v1.FlowInstanceDetailRes, err error)
		// InstanceCancel 发起人撤销流程 (运行中或退回待重提均可)。
		InstanceCancel(ctx context.Context, in *v1.FlowInstanceCancelReq) (res *v1.FlowInstanceCancelRes, err error)
		// InstanceResubmit 退回后修改重新提交, 流程从头重走。
		InstanceResubmit(ctx context.Context, in *v1.FlowInstanceResubmitReq) (res *v1.FlowInstanceResubmitRes, err error)
		// InstanceTerminate 管理员终止流程 (仅运行中)。
		InstanceTerminate(ctx context.Context, in *v1.FlowInstanceTerminateReq) (res *v1.FlowInstanceTerminateRes, err error)
		// InstanceUrge 发起人催办 (10 分钟限一次)。
		InstanceUrge(ctx context.Context, in *v1.FlowInstanceUrgeReq) (res *v1.FlowInstanceUrgeRes, err error)
		// TaskApprove 同意 (或签首签生效; 会签须全部同意)。
		TaskApprove(ctx context.Context, in *v1.FlowTaskApproveReq) (res *v1.FlowTaskApproveRes, err error)
		// TaskReject 驳回 (节点级: 同伴待办作废, 可退回发起人或指定已审批节点)。
		TaskReject(ctx context.Context, in *v1.FlowTaskRejectReq) (res *v1.FlowTaskRejectRes, err error)
		// TaskTransfer 转办: 我的待办转给他人处理。
		TaskTransfer(ctx context.Context, in *v1.FlowTaskTransferReq) (res *v1.FlowTaskTransferRes, err error)
		// TaskAppend 加签: 当前节点追加必要审批人 (节点转为会签)。
		TaskAppend(ctx context.Context, in *v1.FlowTaskAppendReq) (res *v1.FlowTaskAppendRes, err error)
		// TaskReduce 减签: 移除当前节点待办审批人 (至少保留一人)。
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
