// Package flow 自由审批流业务逻辑 — 审批任务动作。
package flow

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"

	v1 "hinay.cn/admin/api/flow/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/xerror"
)

// TaskApprove 同意: 或签首签即过节点, 会签须全部同意后过节点并推进。
func (s *sFlow) TaskApprove(ctx context.Context, in *v1.FlowTaskApproveReq) (res *v1.FlowTaskApproveRes, err error) {
	act := &taskAction{taskId: in.Id, comment: in.Comment}
	inst, def, node, notify, err := s.loadAndValidateTask(ctx, act)
	if err != nil {
		return nil, err
	}

	err = dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 实例行锁串行化并发动作 (或签双击/会签并发), 锁后复核实例仍在运行
		if e := lockInstanceRow(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := ensureInstanceStatus(ctx, tx, inst.Id, instStatusRunning); e != nil {
			return e
		}
		// 锁后重读签核方式: 任务在锁前加载, 并发加签可能已把本节点从或签升级为会签
		signType, e := taskSignTypeInTx(ctx, tx, act.task.Id)
		if e != nil {
			return e
		}
		// 任务置已同意 (仅当仍为待办, 并发已处理则报错回滚)
		if e := completeTask(ctx, tx, act.task.Id, taskStatusApproved, act.comment); e != nil {
			return e
		}
		if e := writeRecord(ctx, tx, inst.Id, act.task.Id, act.task.NodeId, act.task.NodeName,
			"approve", act.operatorId, act.operatorName, act.comment); e != nil {
			return e
		}
		// 会签: 还有其他待办则继续等待
		if signType == taskSignAll {
			cnt, e := tx.Model(dao.WfTask.Table()).Ctx(ctx).
				Where("instance_id", inst.Id).
				Where("node_id", act.task.NodeId).
				Where("status", taskStatusPending).
				Count()
			if e != nil {
				return e
			}
			if cnt > 0 {
				return nil
			}
		}
		// 或签: 其余待办作废
		if signType == taskSignAny {
			if e := voidPendingTasks(ctx, tx, inst.Id, act.task.NodeId, act.task.Id); e != nil {
				return e
			}
		}
		// 节点完成, 推进到下一节点 (自选审批人取实例快照: 发起时选的人存在实例行上)
		selfSelects, e := tx.Model(dao.WfInstance.Table()).Ctx(ctx).
			Where("id", inst.Id).Value("self_selects")
		if e != nil {
			return e
		}
		ss, e := parseSelfSelects(gconv.String(selfSelects))
		if e != nil {
			return e
		}
		return advanceFlow(ctx, tx, def, inst, node.Child, ss, notify)
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	notify.flush(ctx)
	fireEvent(inst.FlowKey, inst.Status, inst.Id, inst.BizId)
	return &v1.FlowTaskApproveRes{}, nil
}

// TaskReject 驳回: 节点级动作 —— 驳回者的待办置已驳回, 同节点其余待办作废。
// 回退目标二选一: 默认退回发起人 (发起人可修改后重新提交或撤销);
// targetNodeId 非空时退回到该**已审批节点**重新处理 (实例保持运行, 从该节点起依次重审)。
// 或签/会签语义一致: 会签任一成员驳回即该节点驳回, 走同一条驳回路径。
func (s *sFlow) TaskReject(ctx context.Context, in *v1.FlowTaskRejectReq) (res *v1.FlowTaskRejectRes, err error) {
	act := &taskAction{taskId: in.Id, comment: in.Comment, wantComment: true}
	inst, _, node, notify, err := s.loadAndValidateTask(ctx, act)
	if err != nil {
		return nil, err
	}

	// 驳回到指定节点: 当前节点作废, 目标节点重新生成待办, 实例保持运行
	if in.TargetNodeId != "" {
		if err = rejectToNode(ctx, inst, act, in.TargetNodeId, notify); err != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, err)
		}
		notify.flush(ctx)
		return &v1.FlowTaskRejectRes{}, nil
	}

	err = dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if e := lockInstanceRow(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := ensureInstanceStatus(ctx, tx, inst.Id, instStatusRunning); e != nil {
			return e
		}
		if e := completeTask(ctx, tx, act.task.Id, taskStatusRejected, act.comment); e != nil {
			return e
		}
		if e := writeRecord(ctx, tx, inst.Id, act.task.Id, act.task.NodeId, act.task.NodeName,
			"reject", act.operatorId, act.operatorName, act.comment); e != nil {
			return e
		}
		// 实例其余待办作废, 原已同意的审批置已失效, 整体退回发起人 (不置 finished_at, 流程未结束)
		if e := voidPendingTasks(ctx, tx, inst.Id, "", 0); e != nil {
			return e
		}
		if e := invalidateApprovedTasks(ctx, tx, inst.Id); e != nil {
			return e
		}
		if _, e := tx.Model(dao.WfInstance.Table()).Ctx(ctx).Where("id", inst.Id).Data(g.Map{
			"status":           instStatusReturned,
			"current_node_ids": "",
		}).Update(); e != nil {
			return e
		}
		inst.Status = instStatusReturned
		return nil
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// 通知发起人被驳回 (尽力而为)
	notify.add(inst.StartUserId,
		fmt.Sprintf("申请被驳回: %s", inst.Title),
		fmt.Sprintf("「%s」的节点「%s」被 %s 驳回: %s。请到 审批中心-我发起的 修改后重新提交, 或撤销申请。",
			inst.Title, node.Name, act.operatorName, act.comment))
	notify.flush(ctx)
	fireEvent(inst.FlowKey, inst.Status, inst.Id, inst.BizId)
	return &v1.FlowTaskRejectRes{}, nil
}

// rejectToNode 驳回到指定已审批节点: 当前节点任务作废 (会签同伴的待办一并作废),
// 原已同意的审批整单置已失效, 目标节点沿用原处理人重新生成待办 (历史出现过会签则统一会签);
// 实例保持运行, 目标节点通过后沿原链路依次重审 (加签功能之前既定的节点级回退逻辑)。
func rejectToNode(ctx context.Context, inst *entity.WfInstance, act *taskAction, targetNodeId string, notify *notifySink) error {
	if targetNodeId == act.task.NodeId {
		return xerror.New(xerror.CodeParamInvalid, "不能驳回到当前节点")
	}
	// 目标节点在本实例的历史处理任务 (已同意或已失效均可用; 处理人与签核方式沿用)
	var hist []*entity.WfTask
	if err := dao.WfTask.Ctx(ctx).
		Where("instance_id", inst.Id).
		Where("node_id", targetNodeId).
		Where("node_type", taskNodeTypeApprove).
		WhereIn("status", []int{taskStatusApproved, taskStatusInvalid}).
		Order("id ASC").Scan(&hist); err != nil {
		return err
	}
	if len(hist) == 0 {
		return xerror.New(xerror.CodeParamInvalid, "目标节点未被审批过, 不能驳回")
	}
	targetName := hist[0].NodeName
	return dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 实例行锁串行化 + 状态复核 (与同意/撤销同一锁序)
		if e := lockInstanceRow(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := ensureInstanceStatus(ctx, tx, inst.Id, instStatusRunning); e != nil {
			return e
		}
		// 本任务置已驳回
		if e := completeTask(ctx, tx, act.task.Id, taskStatusRejected, act.comment); e != nil {
			return e
		}
		if e := writeRecord(ctx, tx, inst.Id, act.task.Id, act.task.NodeId, act.task.NodeName,
			"reject", act.operatorId, act.operatorName, act.comment); e != nil {
			return e
		}
		// 当前节点其余待办作废 (会签同伴)
		if e := voidPendingTasks(ctx, tx, inst.Id, act.task.NodeId, act.task.Id); e != nil {
			return e
		}
		// 原已同意的审批 (含目标节点自身) 置已失效: 目标节点重新生成待办后从头重审
		if e := invalidateApprovedTasks(ctx, tx, inst.Id); e != nil {
			return e
		}
		// 系统记录: 退回到目标节点
		if e := writeRecord(ctx, tx, inst.Id, 0, targetNodeId, targetName,
			"back", 0, "系统", "驳回到「"+targetName+"」, 重新处理"); e != nil {
			return e
		}
		// 目标节点重新生成待办 (去重处理人)。签核口径: 该节点历史任务只要出现过会签,
		// 重生成的一律会签 —— 否则或签/会签混合时, 或签任务先同意即作废同伴, 击穿加签升级过的会签
		nodeSign := taskSignAny
		for _, h := range hist {
			if h.SignType == taskSignAll {
				nodeSign = taskSignAll
				break
			}
		}
		seen := map[uint64]bool{}
		for _, h := range hist {
			if seen[h.AssigneeId] {
				continue
			}
			seen[h.AssigneeId] = true
			if _, e := tx.Model(dao.WfTask.Table()).Ctx(ctx).Data(g.Map{
				"instance_id":   inst.Id,
				"node_id":       h.NodeId,
				"node_name":     h.NodeName,
				"node_type":     taskNodeTypeApprove,
				"sign_type":     nodeSign,
				"assignee_id":   h.AssigneeId,
				"assignee_name": h.AssigneeName,
				"status":        taskStatusPending,
			}).Insert(); e != nil {
				return e
			}
			notify.add(h.AssigneeId,
				fmt.Sprintf("审批被退回: %s", inst.Title),
				fmt.Sprintf("「%s」被 %s 驳回到节点「%s」, 请到 审批中心-我的审批 重新处理。",
					inst.Title, act.operatorName, targetName))
		}
		// 实例保持运行, 当前节点指向目标节点
		if _, e := tx.Model(dao.WfInstance.Table()).Ctx(ctx).Where("id", inst.Id).Data(g.Map{
			"current_node_ids": targetNodeId,
		}).Update(); e != nil {
			return e
		}
		return nil
	})
}

// TaskTransfer 转办: 将我的待办审批任务转给指定人处理。
// 原任务置已转出 (状态4), 目标人生成同节点新待办; 会签同伴不受影响。
func (s *sFlow) TaskTransfer(ctx context.Context, in *v1.FlowTaskTransferReq) (res *v1.FlowTaskTransferRes, err error) {
	act := &taskAction{taskId: in.Id, comment: in.Comment}
	inst, _, _, notify, err := s.loadAndValidateTask(ctx, act)
	if err != nil {
		return nil, err
	}
	if in.TargetUserId == act.operatorId {
		return nil, xerror.New(xerror.CodeParamInvalid, "不能转办给自己")
	}
	var target *entity.SysUser
	if err = dao.SysUser.Ctx(ctx).
		Where("id", in.TargetUserId).Where("status", 1).Where("deleted_at IS NULL").
		Scan(&target); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if target == nil {
		return nil, xerror.New(xerror.CodeParamInvalid, "转办对象不存在或已被禁用")
	}
	targetName := pickName(target.Nickname, target.Username)

	err = dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if e := lockInstanceRow(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := ensureInstanceStatus(ctx, tx, inst.Id, instStatusRunning); e != nil {
			return e
		}
		// 锁后重读签核方式: 并发加签可能已把本节点升级为会签, 新任务须跟随当前口径
		signType, e := taskSignTypeInTx(ctx, tx, act.task.Id)
		if e != nil {
			return e
		}
		// 对方已是本节点待办审批人则拒绝: 同节点同人两行待办会要求其同意两次, 时间线也随之重复
		dup, e := tx.Model(dao.WfTask.Table()).Ctx(ctx).
			Where("instance_id", inst.Id).
			Where("node_id", act.task.NodeId).
			Where("node_type", taskNodeTypeApprove).
			Where("status", taskStatusPending).
			Where("assignee_id", target.Id).
			Count()
		if e != nil {
			return e
		}
		if dup > 0 {
			return xerror.New(xerror.CodeParamInvalid, "对方已是该节点待办审批人, 无需转办")
		}
		if e := completeTask(ctx, tx, act.task.Id, taskStatusTransferred, act.comment); e != nil {
			return e
		}
		// receive_time 沿用原任务: 与本轮同伴同轮聚合 (流程图/时间线口径)
		if _, e = tx.Model(dao.WfTask.Table()).Ctx(ctx).Data(g.Map{
			"instance_id":   inst.Id,
			"node_id":       act.task.NodeId,
			"node_name":     act.task.NodeName,
			"node_type":     taskNodeTypeApprove,
			"sign_type":     signType,
			"assignee_id":   target.Id,
			"assignee_name": targetName,
			"status":        taskStatusPending,
			"receive_time":  act.task.ReceiveTime,
		}).Insert(); e != nil {
			return e
		}
		comment := "转办给 " + targetName
		if act.comment != "" {
			comment += ": " + act.comment
		}
		return writeRecord(ctx, tx, inst.Id, act.task.Id, act.task.NodeId, act.task.NodeName,
			"transfer", act.operatorId, act.operatorName, comment)
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	notify.add(target.Id,
		fmt.Sprintf("审批转办: %s", inst.Title),
		fmt.Sprintf("%s 将「%s」的节点「%s」审批任务转办给您%s, 请到 审批中心-我的审批 处理。",
			act.operatorName, inst.Title, act.task.NodeName, transferNote(act.comment)))
	notify.flush(ctx)
	return &v1.FlowTaskTransferRes{}, nil
}

// transferNote 转办通知中的说明片段。
func transferNote(comment string) string {
	if comment == "" {
		return ""
	}
	return " (说明: " + comment + ")"
}

// loadAnchorTask 加签/减签的任务锚: 待办审批任务 + 运行中实例。
// 与 loadAndValidateTask 的区别: 节点处理人本人**或管理员**均可操作 (加签/减签
// 是节点级动作, 不要求操作人就是该任务的处理人); 不解析节点树。
func (s *sFlow) loadAnchorTask(ctx context.Context, taskId uint64) (
	act *taskAction, inst *entity.WfInstance, notify *notifySink, err error) {

	act = &taskAction{taskId: taskId}
	act.operatorId = contextx.UserId(ctx)
	if act.operatorId == 0 {
		return nil, nil, nil, xerror.New(xerror.CodeUnauthorized)
	}
	user := contextx.LoginUser(ctx)
	act.operatorName = pickName(user.Nickname, user.Username)

	if err = dao.WfTask.Ctx(ctx).Where("id", taskId).Scan(&act.task); err != nil {
		return nil, nil, nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if act.task == nil {
		return nil, nil, nil, xerror.New(xerror.CodeNotFound, "任务不存在")
	}
	if act.task.NodeType != taskNodeTypeApprove {
		return nil, nil, nil, xerror.New(xerror.CodeParamInvalid, "仅审批任务支持加签/减签")
	}
	if act.task.Status != taskStatusPending {
		return nil, nil, nil, xerror.New(xerror.CodeParamInvalid, "任务已处理, 无法加签/减签")
	}
	if inst, err = loadInstance(ctx, act.task.InstanceId); err != nil {
		return nil, nil, nil, err
	}
	if inst.Status != instStatusRunning {
		return nil, nil, nil, xerror.New(xerror.CodeParamInvalid, "流程已结束")
	}
	if act.task.AssigneeId != act.operatorId && !contextx.IsAdmin(ctx) {
		return nil, nil, nil, xerror.New(xerror.CodeForbidden, "仅节点处理人或管理员可加签/减签")
	}
	return act, inst, &notifySink{sender: act.operatorId}, nil
}

// TaskAppend 加签: 以我的待办所在节点为锚, 追加必要审批人。
// 语义 (钉钉式"同时加签"): 节点全部待办转为会签 —— 原处理人与新加人均须同意,
// 节点才通过 (任一驳回=该节点驳回, 按所选目标回退); 新任务沿用本轮 receive_time, 流程图聚合同一轮次。
// 已是该节点待办/本轮已同意过的人不再加入 (会签下重复加签会要求其二次同意)。
func (s *sFlow) TaskAppend(ctx context.Context, in *v1.FlowTaskAppendReq) (res *v1.FlowTaskAppendRes, err error) {
	act, inst, notify, err := s.loadAnchorTask(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	act.comment = in.Comment
	// 加签对象解析 (启用用户去重; 已是该节点待办/本轮已同意的人在事务内过滤)
	users, err := lookupUsers(ctx, in.UserIds, "加签对象")
	if err != nil {
		return nil, err
	}
	var added []approverUser

	err = dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if e := lockInstanceRow(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := ensureInstanceStatus(ctx, tx, inst.Id, instStatusRunning); e != nil {
			return e
		}
		// 锁后取节点当前待办: 节点可能已被并发同意推进到下一节点
		pend, e := nodePendingTasks(ctx, tx, inst.Id, act.task.NodeId)
		if e != nil {
			return e
		}
		if len(pend) == 0 {
			return xerror.New(xerror.CodeParamInvalid, "该节点已处理完成, 无法加签")
		}
		pendingIds := map[uint64]bool{}
		for _, t := range pend {
			pendingIds[t.AssigneeId] = true
		}
		// 本轮已同意过该节点的人也跳过: 会签下部分同意后再加签其本人, 会要求其二次同意
		var acted []*entity.WfTask
		if e = tx.Model(dao.WfTask.Table()).Ctx(ctx).
			Where("instance_id", inst.Id).
			Where("node_id", act.task.NodeId).
			Where("node_type", taskNodeTypeApprove).
			Where("status", taskStatusApproved).
			Scan(&acted); e != nil {
			return e
		}
		doneIds := map[uint64]bool{}
		for _, t := range acted {
			doneIds[t.AssigneeId] = true
		}
		added = make([]approverUser, 0, len(users))
		for _, u := range users {
			if !pendingIds[u.Id] && !doneIds[u.Id] {
				added = append(added, u)
			}
		}
		if len(added) == 0 {
			return xerror.New(xerror.CodeParamInvalid, "所选用户均已是该节点审批人或已同意过该节点")
		}
		// 节点转为会签: 本轮全部待办改 sign_type=2 (既有待办也须同意)
		if _, e = tx.Model(dao.WfTask.Table()).Ctx(ctx).
			Where("instance_id", inst.Id).
			Where("node_id", act.task.NodeId).
			Where("node_type", taskNodeTypeApprove).
			Where("status", taskStatusPending).
			Data("sign_type", taskSignAll).Update(); e != nil {
			return e
		}
		// 新增待办: receive_time 沿用本轮, 与既有待办同轮聚合
		for _, u := range added {
			if _, e = tx.Model(dao.WfTask.Table()).Ctx(ctx).Data(g.Map{
				"instance_id":   inst.Id,
				"node_id":       act.task.NodeId,
				"node_name":     act.task.NodeName,
				"node_type":     taskNodeTypeApprove,
				"sign_type":     taskSignAll,
				"assignee_id":   u.Id,
				"assignee_name": u.Name,
				"status":        taskStatusPending,
				"receive_time":  pend[0].ReceiveTime,
			}).Insert(); e != nil {
				return e
			}
		}
		comment := "加签: " + joinNames(approverNames(added)) + " 加入审批 (节点转为会签)"
		if act.comment != "" {
			comment += "; " + act.comment
		}
		return writeRecord(ctx, tx, inst.Id, act.task.Id, act.task.NodeId, act.task.NodeName,
			"append", act.operatorId, act.operatorName, comment)
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	for _, u := range added {
		notify.add(u.Id,
			fmt.Sprintf("审批加签: %s", inst.Title),
			fmt.Sprintf("%s 在「%s」的节点「%s」加签您为审批人, 请到 审批中心-我的审批 处理。",
				act.operatorName, inst.Title, act.task.NodeName))
	}
	notify.flush(ctx)
	return &v1.FlowTaskAppendRes{}, nil
}

// TaskReduce 减签: 从当前节点移除指定待办审批人 (任务置已作废, 至少保留一人)。
// 仅移除待办任务; 已同意的不回退 (会签已投的票有效); 被移除人收到待办作废通知。
func (s *sFlow) TaskReduce(ctx context.Context, in *v1.FlowTaskReduceReq) (res *v1.FlowTaskReduceRes, err error) {
	act, inst, notify, err := s.loadAnchorTask(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	act.comment = in.Comment
	var removed []*entity.WfTask

	err = dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if e := lockInstanceRow(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := ensureInstanceStatus(ctx, tx, inst.Id, instStatusRunning); e != nil {
			return e
		}
		pend, e := nodePendingTasks(ctx, tx, inst.Id, act.task.NodeId)
		if e != nil {
			return e
		}
		if len(pend) == 0 {
			return xerror.New(xerror.CodeParamInvalid, "该节点已处理完成, 无法减签")
		}
		removeIds := map[uint64]bool{}
		for _, uid := range in.UserIds {
			removeIds[uid] = true
		}
		removed = make([]*entity.WfTask, 0, len(in.UserIds))
		for _, t := range pend {
			if removeIds[t.AssigneeId] {
				removed = append(removed, t)
			}
		}
		if len(removed) == 0 {
			return xerror.New(xerror.CodeParamInvalid, "所选用户均不是该节点的待办审批人")
		}
		if len(removed) == len(pend) {
			return xerror.New(xerror.CodeParamInvalid, "不能移除全部待办审批人, 至少保留一人")
		}
		ids := make([]uint64, 0, len(removed))
		names := make([]string, 0, len(removed))
		for _, t := range removed {
			ids = append(ids, t.Id)
			names = append(names, t.AssigneeName)
		}
		if _, e = tx.Model(dao.WfTask.Table()).Ctx(ctx).
			WhereIn("id", ids).
			Where("status", taskStatusPending).
			Data("status", taskStatusVoid).Update(); e != nil {
			return e
		}
		comment := "减签: 移除 " + joinNames(names)
		if act.comment != "" {
			comment += "; " + act.comment
		}
		return writeRecord(ctx, tx, inst.Id, act.task.Id, act.task.NodeId, act.task.NodeName,
			"reduce", act.operatorId, act.operatorName, comment)
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 被移除人收到待办作废通知 (不再静默消失)
	for _, t := range removed {
		notify.add(t.AssigneeId,
			fmt.Sprintf("审批减签: %s", inst.Title),
			fmt.Sprintf("%s 在「%s」的节点「%s」将你移出审批人, 该待办已作废, 无需再处理。",
				act.operatorName, inst.Title, act.task.NodeName))
	}
	notify.flush(ctx)
	return &v1.FlowTaskReduceRes{}, nil
}

// nodePendingTasks 取节点当前待办审批任务 (事务内, 锁后调用)。
func nodePendingTasks(ctx context.Context, tx gdb.TX, instanceId uint64, nodeId string) ([]*entity.WfTask, error) {
	var rows []*entity.WfTask
	if err := tx.Model(dao.WfTask.Table()).Ctx(ctx).
		Where("instance_id", instanceId).
		Where("node_id", nodeId).
		Where("node_type", taskNodeTypeApprove).
		Where("status", taskStatusPending).
		Order("id ASC").Scan(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// approverNames 审批人列表转昵称列表。
func approverNames(users []approverUser) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		out = append(out, u.Name)
	}
	return out
}

// TaskRead 抄送已读。
func (s *sFlow) TaskRead(ctx context.Context, in *v1.FlowTaskReadReq) (res *v1.FlowTaskReadRes, err error) {
	uid := contextx.UserId(ctx)
	var t *entity.WfTask
	if err = dao.WfTask.Ctx(ctx).Where("id", in.Id).Scan(&t); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if t == nil {
		return nil, xerror.New(xerror.CodeNotFound, "任务不存在")
	}
	if t.AssigneeId != uid {
		return nil, xerror.New(xerror.CodeForbidden, "非任务处理人")
	}
	if t.NodeType != taskNodeTypeCC {
		return nil, xerror.New(xerror.CodeParamInvalid, "仅抄送任务需要标记已读")
	}
	if t.Status != taskStatusPending {
		return &v1.FlowTaskReadRes{}, nil // 幂等
	}
	if _, err = dao.WfTask.Ctx(ctx).Where("id", in.Id).Where("status", taskStatusPending).Data(g.Map{
		"status": taskStatusApproved, "acted_at": gtime.Now(), // 抄送行复用状态: 2=已阅
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.FlowTaskReadRes{}, nil
}

// TaskCount 我的待办/待阅数量。
func (s *sFlow) TaskCount(ctx context.Context, in *v1.FlowTaskCountReq) (res *v1.FlowTaskCountRes, err error) {
	uid := contextx.UserId(ctx)
	todo, err := dao.WfTask.Ctx(ctx).
		Where("assignee_id", uid).
		Where("node_type", taskNodeTypeApprove).
		Where("status", taskStatusPending).
		Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	cc, err := dao.WfTask.Ctx(ctx).
		Where("assignee_id", uid).
		Where("node_type", taskNodeTypeCC).
		Where("status", taskStatusPending).
		Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.FlowTaskCountRes{Todo: int(todo), Cc: int(cc)}, nil
}

// ============================================================
// 内部工具
// ============================================================

// completeTask 条件更新任务状态 (仅当仍为待办时生效)。
// 0 行受影响说明任务已被并发事务处理 (双击/或签抢先), 报错回滚避免双推进。
func completeTask(ctx context.Context, tx gdb.TX, taskId uint64, status int, comment string) error {
	res, err := tx.Model(dao.WfTask.Table()).Ctx(ctx).
		Where("id", taskId).
		Where("status", taskStatusPending).
		// wf_task.comment VARCHAR(500): 拼接后的意见可能超长 (加签/减签/转办), 按列长截断防 INSERT 失败
		Data(g.Map{"status": status, "comment": truncateRunes(comment, 500), "acted_at": gtime.Now()}).
		Update()
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return xerror.New(xerror.CodeParamInvalid, "任务已被处理, 请刷新后重试")
	}
	return nil
}

// taskSignTypeInTx 事务内 (实例行锁后) 重读任务行的签核方式。
// 任务在事务外加载, 加锁前节点可能已被并发加签从或签升级为会签,
// 同意/转办的分支决策不能依赖锁前快照 (否则或签旧值会作废同伴并击穿会签)。
func taskSignTypeInTx(ctx context.Context, tx gdb.TX, taskId uint64) (int, error) {
	v, err := tx.Model(dao.WfTask.Table()).Ctx(ctx).Where("id", taskId).Value("sign_type")
	if err != nil {
		return 0, err
	}
	st := v.Int()
	if st != taskSignAny && st != taskSignAll {
		return 0, xerror.New(xerror.CodeBusinessError, "任务不存在或签核方式异常")
	}
	return st, nil
}

// taskAction 一次任务动作的入参。
type taskAction struct {
	taskId       uint64
	comment      string
	wantComment  bool // 驳回时必填意见
	task         *entity.WfTask
	operatorId   uint64
	operatorName string
}

// loadAndValidateTask 加载并校验任务归属/状态, 返回实例、定义、节点与通知队列。
func (s *sFlow) loadAndValidateTask(ctx context.Context, act *taskAction) (
	inst *entity.WfInstance, def *entity.WfDefinition, node *Node, notify *notifySink, err error) {

	act.operatorId = contextx.UserId(ctx)
	if act.operatorId == 0 {
		return nil, nil, nil, nil, xerror.New(xerror.CodeUnauthorized)
	}
	user := contextx.LoginUser(ctx)
	act.operatorName = pickName(user.Nickname, user.Username)

	if act.wantComment && gconv.String(act.comment) == "" {
		return nil, nil, nil, nil, xerror.New(xerror.CodeParamInvalid, "请填写驳回意见")
	}

	if err = dao.WfTask.Ctx(ctx).Where("id", act.taskId).Scan(&act.task); err != nil {
		return nil, nil, nil, nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if act.task == nil {
		return nil, nil, nil, nil, xerror.New(xerror.CodeNotFound, "任务不存在")
	}
	if act.task.AssigneeId != act.operatorId {
		return nil, nil, nil, nil, xerror.New(xerror.CodeForbidden, "非任务处理人")
	}
	if act.task.Status != taskStatusPending {
		return nil, nil, nil, nil, xerror.New(xerror.CodeParamInvalid, "任务已处理")
	}
	if act.task.NodeType != taskNodeTypeApprove {
		return nil, nil, nil, nil, xerror.New(xerror.CodeParamInvalid, "抄送任务无审批动作")
	}

	if inst, err = loadInstance(ctx, act.task.InstanceId); err != nil {
		return nil, nil, nil, nil, err
	}
	if inst.Status != instStatusRunning {
		return nil, nil, nil, nil, xerror.New(xerror.CodeParamInvalid, "流程已结束")
	}
	if def, err = loadDefinition(ctx, inst.DefinitionId); err != nil {
		return nil, nil, nil, nil, err
	}
	// 节点树优先取实例快照 (定义后续修改不影响在途实例)
	flowConf := inst.FlowConf
	if flowConf == "" {
		flowConf = def.FlowConf
	}
	var root *Node
	if root, err = parseFlowConf(flowConf); err != nil {
		return nil, nil, nil, nil, xerror.Wrap(xerror.CodeParamInvalid, err)
	}
	node = findNode(root, act.task.NodeId)
	if node == nil {
		return nil, nil, nil, nil, xerror.New(xerror.CodeBusinessError, "流程定义与任务不匹配 (节点缺失)")
	}
	return inst, def, node, &notifySink{sender: act.operatorId}, nil
}
