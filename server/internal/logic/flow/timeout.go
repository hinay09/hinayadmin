// Package flow 审批超时扫描 (定时任务处理器, 注册名 flow.timeoutScan)。
//
// 节点超时配置 (approver 节点, 随 flow_conf 快照存于实例):
//
//	timeoutHours: 办理期限(小时), 0=不限; 引擎生成任务时物化为 wf_task.due_time;
//	timeoutAction: remind=逾期提醒(默认) / transfer=自动转办 / approve=自动通过;
//	timeoutTransfer: 自动转办目标用户ID (transfer 必填)。
//
// 扫描语义 (每轮批量处理逾期任务, 单任务失败不阻断整轮):
//   - remind: 逾期通知审批人, 每任务一次 (wf_record action=timeoutRemind 去重);
//   - transfer: 系统转办给指定人 (复用转办语义: 原任务置已转出), 新任务不再计时
//     (due_time 置空, 防无限转办循环);
//   - approve: 系统代为同意并走完整推进 (或签作废同伴/会签计数/下一节点),
//     与人工同意共用事务内核 approveInTx;
//   - 被委派任务 (delegate_from_id>0) 不做自动转办/自动通过 —— 代办后的终审权
//     属于原审批人, 系统不能越权代决; 一律降级为提醒 (提醒对象即被委托人)。
//
// 与并发动作的赛跑: 全部走实例行锁 + 条件更新 (completeTask), 先落者赢,
// 后到者在复核/条件更新处报错回滚, 扫描循环按"跳过"记数, 下轮自然消失。
package flow

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/job"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/utility/xerror"
)

func init() {
	// sys_job 种子行在 manifest/sql/init.sql (默认每 10 分钟), 网页端可调频率/暂停
	job.RegisterHandler("flow.timeoutScan", scanTimeoutTasks)
}

// timeoutScanBatch 每轮扫描的批量上限 (防单轮过载, 余量下轮继续)。
const timeoutScanBatch = 200

// scanTimeoutTasks 超时扫描入口 (job 处理器): 找出逾期待办并按节点策略处理。
func scanTimeoutTasks(ctx context.Context, params string) (string, error) {
	var tasks []*entity.WfTask
	if err := dao.WfTask.Ctx(ctx).
		Where("node_type", taskNodeTypeApprove).
		Where("status", taskStatusPending).
		Where("due_time IS NOT NULL AND due_time <= ?", gtime.Now()).
		Order("due_time ASC").
		Limit(timeoutScanBatch).
		Scan(&tasks); err != nil {
		return "", err
	}
	if len(tasks) == 0 {
		return "无逾期待办", nil
	}

	var reminded, transferred, approved, skipped int
	for _, t := range tasks {
		switch handleTimeout(ctx, t) {
		case "remind":
			reminded++
		case "transfer":
			transferred++
		case "approve":
			approved++
		default:
			skipped++
		}
	}
	summary := fmt.Sprintf("逾期待办 %d: 提醒 %d, 自动转办 %d, 自动通过 %d, 跳过 %d",
		len(tasks), reminded, transferred, approved, skipped)
	g.Log().Infof(ctx, "flow.timeoutScan: %s", summary)
	return summary, nil
}

// handleTimeout 处理单个逾期任务, 返回动作结果 (remind/transfer/approve/skip)。
// 出错记日志并按 skip 记数, 不阻断整轮扫描。
func handleTimeout(ctx context.Context, t *entity.WfTask) string {
	inst, err := loadInstance(ctx, t.InstanceId)
	if err != nil || inst.Status != instStatusRunning {
		return "skip" // 实例不存在/已结束: 待办已被作废, 等下轮条件自然过滤
	}
	node, err := nodeOfInstance(ctx, inst, t.NodeId)
	if err != nil || node == nil {
		return "skip" // 快照中找不到节点 (脏数据): 不猜策略
	}

	action := node.TimeoutAction
	if action == "" {
		action = timeoutActionRemind
	}
	// 被委派的代办任务不可代决/换手, 一律降级为提醒
	if t.DelegateFromId > 0 && action != timeoutActionRemind {
		action = timeoutActionRemind
	}

	switch action {
	case timeoutActionTransfer:
		if node.TimeoutTransfer == 0 {
			return "skip" // 配置脏数据 (发布校验已拦, 快照存量兜底)
		}
		if err := timeoutTransfer(ctx, inst, t, node); err != nil {
			g.Log().Warningf(ctx, "flow.timeoutScan transfer task %d failed: %v", t.Id, err)
			return "skip"
		}
		return "transfer"
	case timeoutActionApprove:
		if err := timeoutApprove(ctx, inst, t, node); err != nil {
			g.Log().Warningf(ctx, "flow.timeoutScan approve task %d failed: %v", t.Id, err)
			return "skip"
		}
		return "approve"
	default: // remind
		if err := timeoutRemind(ctx, inst, t, node); err != nil {
			g.Log().Warningf(ctx, "flow.timeoutScan remind task %d failed: %v", t.Id, err)
			return "skip"
		}
		return "remind"
	}
}

// nodeOfInstance 取实例快照节点树中的节点 (存量旧数据兜底定义行)。
func nodeOfInstance(ctx context.Context, inst *entity.WfInstance, nodeId string) (*Node, error) {
	flowConf := inst.FlowConf
	if flowConf == "" {
		def, err := loadDefinition(ctx, inst.DefinitionId)
		if err != nil {
			return nil, err
		}
		flowConf = def.FlowConf
	}
	root, err := parseFlowConf(flowConf)
	if err != nil {
		return nil, err
	}
	return findNode(root, nodeId), nil
}

// timeoutRemind 逾期提醒: 通知任务处理人, 每任务一次 (wf_record timeoutRemind 去重)。
func timeoutRemind(ctx context.Context, inst *entity.WfInstance, t *entity.WfTask, node *Node) error {
	cnt, err := dao.WfRecord.Ctx(ctx).
		Where("task_id", t.Id).
		Where("action", "timeoutRemind").
		Count()
	if err != nil {
		return err
	}
	if cnt > 0 {
		return nil // 已提醒过
	}
	if _, err = dao.WfRecord.Ctx(ctx).Data(g.Map{
		"instance_id":   inst.Id,
		"task_id":       t.Id,
		"node_id":       t.NodeId,
		"node_name":     t.NodeName,
		"action":        "timeoutRemind",
		"operator_id":   0,
		"operator_name": "系统",
		"comment": truncateRunes(fmt.Sprintf("审批超时提醒: 超过「%s」办理期限 %d 小时",
			node.Name, node.TimeoutHours), 500),
	}).Insert(); err != nil {
		return err
	}
	notify := &notifySink{}
	notify.add(t.AssigneeId,
		fmt.Sprintf("审批超时: %s", inst.Title),
		fmt.Sprintf("「%s」的节点「%s」已超过 %d 小时办理期限, 请尽快到 审批中心-我的审批 处理。",
			inst.Title, t.NodeName, node.TimeoutHours))
	notify.flush(ctx)
	return nil
}

// timeoutTransfer 超时自动转办: 系统把逾期任务转给节点配置的接办人。
// 复用转办语义 (原任务置已转出); 新任务 due_time 置空 —— 不再重复计时, 防无限循环。
func timeoutTransfer(ctx context.Context, inst *entity.WfInstance, t *entity.WfTask, node *Node) error {
	users, err := lookupUsers(ctx, []uint64{node.TimeoutTransfer}, "节点「"+node.Name+"」超时转办对象")
	if err != nil {
		return err
	}
	target := users[0]

	txErr := dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if e := lockInstanceRow(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := ensureInstanceStatus(ctx, tx, inst.Id, instStatusRunning); e != nil {
			return e
		}
		signType, e := taskSignTypeInTx(ctx, tx, t.Id)
		if e != nil {
			return e
		}
		// 对方已是本节点待办审批人则放弃转办 (配置指向了已在办的人);
		// 已委派挂起(状态7)的原审批人也算在内, 其代办回归后同样构成两票
		dup, e := tx.Model(dao.WfTask.Table()).Ctx(ctx).
			Where("instance_id", inst.Id).
			Where("node_id", t.NodeId).
			Where("node_type", taskNodeTypeApprove).
			WhereIn("status", []int{taskStatusPending, taskStatusDelegated}).
			Where("assignee_id", target.Id).
			Count()
		if e != nil {
			return e
		}
		if dup > 0 {
			return xerror.New(xerror.CodeBusinessError, "转办对象已是该节点待办审批人")
		}
		if e := completeTask(ctx, tx, t.Id, taskStatusTransferred, "超时自动转办给 "+target.Name); e != nil {
			return e
		}
		if _, e = tx.Model(dao.WfTask.Table()).Ctx(ctx).Data(g.Map{
			"instance_id":   inst.Id,
			"node_id":       t.NodeId,
			"node_name":     t.NodeName,
			"node_type":     taskNodeTypeApprove,
			"sign_type":     signType,
			"assignee_id":   target.Id,
			"assignee_name": target.Name,
			"status":        taskStatusPending,
			"receive_time":  t.ReceiveTime,
			"due_time":      nil, // 不再计时: 防同一任务被反复转办
		}).Insert(); e != nil {
			return e
		}
		return writeRecord(ctx, tx, inst.Id, t.Id, t.NodeId, t.NodeName,
			"timeoutTransfer", 0, "系统", "超时自动转办给 "+target.Name)
	})
	if txErr != nil {
		return txErr
	}
	notify := &notifySink{}
	notify.add(target.Id,
		fmt.Sprintf("审批超时转办: %s", inst.Title),
		fmt.Sprintf("「%s」的节点「%s」审批超时, 系统转办给您处理, 请到 审批中心-我的审批 尽快办理。",
			inst.Title, t.NodeName))
	notify.flush(ctx)
	return nil
}

// timeoutApprove 超时自动通过: 系统代为同意并推进, 与人工同意共用事务内核。
func timeoutApprove(ctx context.Context, inst *entity.WfInstance, t *entity.WfTask, node *Node) error {
	act := &taskAction{
		taskId:       t.Id,
		comment:      "超时自动通过",
		task:         t,
		operatorId:   0,
		operatorName: "系统",
	}
	notify := &notifySink{}
	err := dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		return approveInTx(ctx, tx, act, inst, defSnapOf(inst), node, notify, "timeoutApprove")
	})
	if err != nil {
		return err
	}
	notify.flush(ctx)
	if inst.Status != instStatusRunning {
		fireEvent(inst.FlowKey, inst.Status, inst.Id, inst.BizId)
	}
	return nil
}

// defSnapOf 由实例行构造推进用的定义快照 (advanceFlow 只走节点树, 定义行仅兜底)。
func defSnapOf(inst *entity.WfInstance) *entity.WfDefinition {
	return &entity.WfDefinition{Id: inst.DefinitionId, FlowKey: inst.FlowKey, Name: inst.FlowName, FlowConf: inst.FlowConf}
}
