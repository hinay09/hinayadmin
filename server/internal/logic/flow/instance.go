// Package flow 自由审批流业务逻辑 — 流程实例。
package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"

	v1 "hinay.cn/admin/api/flow/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

// InstanceStart 发起流程: 校验定义/表单 → 建实例(携带表单/节点树快照) → 从发起节点推进。
func (s *sFlow) InstanceStart(ctx context.Context, in *v1.FlowInstanceStartReq) (res *v1.FlowInstanceStartRes, err error) {
	uid := contextx.UserId(ctx)
	if uid == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	user := contextx.LoginUser(ctx)
	userName := pickName(user.Nickname, user.Username)

	def, err := loadDefinition(ctx, in.DefinitionId)
	if err != nil {
		return nil, err
	}
	if def.Status != defStatusPublished || def.Version < 1 {
		return nil, xerror.New(xerror.CodeParamInvalid, "仅已发布流程可发起")
	}

	inst, notify, err := startInstanceTx(ctx, def, in.Title, in.FormData, in.SelfSelects, uid, userName, 0)
	if err != nil {
		return nil, err // startInstanceTx 内已按 xerror 语义包装
	}
	notify.flush(ctx)
	fireEvent(def.FlowKey, inst.Status, inst.Id, inst.BizId)
	return &v1.FlowInstanceStartRes{Id: inst.Id}, nil
}

// startInstanceTx 发起实例公共实现 (API 发起与业务模块 StartForBiz 共用):
// 表单必填校验 → 写实例(含 form_conf/flow_conf 快照) → 提交记录 → 从根节点推进。
func startInstanceTx(ctx context.Context, def *entity.WfDefinition, title string,
	formData map[string]any, selfSelects map[string][]uint64,
	uid uint64, userName string, bizId uint64) (*entity.WfInstance, *notifySink, error) {

	fields, err := parseFormConf(def.FormConf)
	if err != nil {
		return nil, nil, xerror.New(xerror.CodeParamInvalid, err.Error())
	}
	if formData == nil {
		formData = map[string]any{}
	}
	for _, f := range fields {
		if !f.Required {
			continue
		}
		v, ok := formData[f.Key]
		if !ok || isEmptyValue(v) {
			return nil, nil, xerror.New(xerror.CodeParamInvalid, "请填写「"+f.Label+"」")
		}
	}
	root, err := parseFlowConf(def.FlowConf)
	if err != nil {
		return nil, nil, xerror.New(xerror.CodeParamInvalid, err.Error())
	}
	// 自选节点完整性前置校验: 深层自选节点若发起时未选人, 推进到它时才解析失败,
	// 前一节点的同意动作会永远报错回滚, 流程卡死中途 (前端提交弹窗有同样校验, 这里兜底 API 直调)。
	if err = validateSelfSelects(root, selfSelects); err != nil {
		return nil, nil, xerror.New(xerror.CodeParamInvalid, err.Error())
	}
	formJSON, err := json.Marshal(formData)
	if err != nil {
		return nil, nil, xerror.Wrap(xerror.CodeParamInvalid, err)
	}
	// 自选审批人整份快照落实例: 发起时引擎只走到第一个审批节点,
	// 深层自选节点要等后续推进才解析, 届时从这里读取。
	selfJSON, err := marshalSelfSelects(selfSelects)
	if err != nil {
		return nil, nil, xerror.Wrap(xerror.CodeParamInvalid, err)
	}

	notify := &notifySink{sender: uid}
	inst := &entity.WfInstance{
		FlowKey:       def.FlowKey,
		BizId:         bizId,
		FlowName:      def.Name,
		Title:         title,
		FormData:      string(formJSON),
		Status:        instStatusRunning,
		StartUserId:   uid,
		StartUserName: userName,
	}
	err = dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		data := g.Map{
			"definition_id":    def.Id,
			"flow_key":         def.FlowKey,
			"biz_id":           bizId,
			"flow_name":        def.Name,
			"title":            title,
			"form_data":        string(formJSON),
			"form_conf":        def.FormConf, // 快照: 在途实例不随定义后续修改变化
			"flow_conf":        def.FlowConf,
			"current_node_ids": "",
			"status":           instStatusRunning,
			"start_user_id":    uid,
			"start_user_name":  userName,
		}
		if selfJSON != "" {
			data["self_selects"] = selfJSON
		}
		id, ie := tx.Model(dao.WfInstance.Table()).Ctx(ctx).Data(data).InsertAndGetId()
		if ie != nil {
			return ie
		}
		inst.Id = uint64(id)
		if ie = writeRecord(ctx, tx, inst.Id, 0, "start", "发起人", "submit", uid, userName, "提交申请"); ie != nil {
			return ie
		}
		return advanceFlow(ctx, tx, def, inst, root, selfSelects, notify)
	})
	if err != nil {
		return nil, nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return inst, notify, nil
}

// InstanceList 实例列表 (todo/done/mine/ccme 四视角 + all 管理员全局视角)。
func (s *sFlow) InstanceList(ctx context.Context, in *v1.FlowInstanceListReq) (res *v1.FlowInstanceListRes, err error) {
	uid := contextx.UserId(ctx)
	q := dao.WfInstance.Ctx(ctx).Where("deleted_at IS NULL")
	myTask := map[uint64]uint64{} // instanceId -> 我的任务ID

	switch in.Scope {
	case "todo":
		tasks, te := myTasks(ctx, uid, taskNodeTypeApprove, []int{taskStatusPending})
		if te != nil {
			return nil, te
		}
		if len(tasks) == 0 {
			return emptyInstancePage(in), nil
		}
		for _, t := range tasks {
			myTask[t.InstanceId] = t.Id
		}
		q = q.WhereIn("id", instanceIds(tasks)).Where("status", instStatusRunning)
	case "done":
		// 已失效(被退回)的审批也算"处理过", 保留在已办列表
		tasks, te := myTasks(ctx, uid, taskNodeTypeApprove, []int{taskStatusApproved, taskStatusRejected, taskStatusInvalid})
		if te != nil {
			return nil, te
		}
		if len(tasks) == 0 {
			return emptyInstancePage(in), nil
		}
		q = q.WhereIn("id", instanceIds(tasks))
	case "ccme":
		tasks, te := myTasks(ctx, uid, taskNodeTypeCC, []int{taskStatusPending, taskStatusApproved})
		if te != nil {
			return nil, te
		}
		if len(tasks) == 0 {
			return emptyInstancePage(in), nil
		}
		for _, t := range tasks {
			myTask[t.InstanceId] = t.Id
		}
		q = q.WhereIn("id", instanceIds(tasks))
	case "mine":
		q = q.Where("start_user_id", uid)
		if in.Status != nil {
			q = q.Where("status", *in.Status)
		}
	case "all":
		// 管理员全局视角 (流程实例管理页): 跨用户列出全部实例
		if !contextx.IsAdmin(ctx) {
			return nil, xerror.New(xerror.CodeForbidden, "仅管理员可查看全部流程实例")
		}
		if in.Status != nil {
			q = q.Where("status", *in.Status)
		}
		if flowKey := strings.TrimSpace(in.FlowKey); flowKey != "" {
			q = q.Where("flow_key", flowKey)
		}
	default:
		return nil, xerror.New(xerror.CodeParamInvalid, "不支持的列表视角: "+in.Scope)
	}

	if in.Keyword != "" {
		kw := "%" + strings.TrimSpace(in.Keyword) + "%"
		q = q.Where("(title LIKE ? OR flow_name LIKE ?)", kw, kw)
	}

	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*entity.WfInstance
	if err = q.Order("id DESC").Page(in.Page, in.PageSize).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 当前节点名一次批量取回, 避免逐行查询 (N+1)
	nodeNames := currentNodeNamesBatch(ctx, rows)
	list := make([]*v1.FlowInstanceItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, instItem(r, myTask[r.Id], nodeNames[r.Id]))
	}
	page := response.Page(list, int64(total), in.Page, in.PageSize)
	return (*v1.FlowInstanceListRes)(&page), nil
}

// InstanceDetail 实例详情: 表单/节点树取实例快照 (存量旧数据兜底定义行) + 任务 + 时间线。
func (s *sFlow) InstanceDetail(ctx context.Context, in *v1.FlowInstanceDetailReq) (res *v1.FlowInstanceDetailRes, err error) {
	uid := contextx.UserId(ctx)
	inst, err := loadInstance(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	formConf, flowConf := inst.FormConf, inst.FlowConf
	if formConf == "" || flowConf == "" {
		def, derr := loadDefinition(ctx, inst.DefinitionId)
		if derr != nil {
			return nil, derr
		}
		if formConf == "" {
			formConf = def.FormConf
		}
		if flowConf == "" {
			flowConf = def.FlowConf
		}
	}

	var tasks []*entity.WfTask
	if err = dao.WfTask.Ctx(ctx).Where("instance_id", inst.Id).Order("id ASC").Scan(&tasks); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var records []*entity.WfRecord
	if err = dao.WfRecord.Ctx(ctx).Where("instance_id", inst.Id).Order("id ASC").Scan(&records); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	taskItems := make([]*v1.FlowTaskItem, 0, len(tasks))
	var myPending, myCc uint64
	for _, t := range tasks {
		taskItems = append(taskItems, taskItem(t))
		if t.AssigneeId == uid && t.Status == taskStatusPending {
			if t.NodeType == taskNodeTypeApprove {
				myPending = t.Id
			} else {
				myCc = t.Id
			}
		}
	}
	recordItems := make([]*v1.FlowRecordItem, 0, len(records))
	for _, r := range records {
		recordItems = append(recordItems, recordItem(r))
	}
	// 可驳回目标: 按节点聚合取最新一条任务, 最新状态为已同意/已失效的节点可被驳回
	// (已失效=退回后原同意不再计入, rejectToNode 后端同样接受该状态, 前后端口径一致);
	// 按节点首次到达顺序展示, 供驳回弹窗选择。
	latestTask := map[string]*entity.WfTask{}
	nodeOrder := make([]string, 0)
	for _, t := range tasks {
		if t.NodeType != taskNodeTypeApprove {
			continue
		}
		if _, ok := latestTask[t.NodeId]; !ok {
			nodeOrder = append(nodeOrder, t.NodeId)
		}
		if prev, ok := latestTask[t.NodeId]; !ok || t.Id > prev.Id {
			latestTask[t.NodeId] = t
		}
	}
	rejectTargets := make([]*v1.FlowRejectTarget, 0, len(nodeOrder))
	for _, nodeId := range nodeOrder {
		t := latestTask[nodeId]
		if t.Status == taskStatusApproved || t.Status == taskStatusInvalid {
			rejectTargets = append(rejectTargets, &v1.FlowRejectTarget{NodeId: t.NodeId, NodeName: t.NodeName})
		}
	}
	return &v1.FlowInstanceDetailRes{
		Instance:        instItem(inst, 0, currentNodeNames(ctx, inst)),
		FormConf:        formConf,
		FormData:        inst.FormData,
		FlowConf:        flowConf,
		Tasks:           taskItems,
		Records:         recordItems,
		MyPendingTaskId: myPending,
		MyCcTaskId:      myCc,
		RejectTargets:   rejectTargets,
		CanCancel:       (inst.Status == instStatusRunning || inst.Status == instStatusReturned) && inst.StartUserId == uid,
		CanResubmit:     (inst.Status == instStatusReturned || inst.Status == instStatusCanceled) && inst.StartUserId == uid,
		PrevSelfSelects: prevSelfSelects(flowConf, tasks),
	}, nil
}

// InstanceCancel 发起人撤销流程 (运行中或退回待重提均可撤销)。
func (s *sFlow) InstanceCancel(ctx context.Context, in *v1.FlowInstanceCancelReq) (res *v1.FlowInstanceCancelRes, err error) {
	uid := contextx.UserId(ctx)
	if uid == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	user := contextx.LoginUser(ctx)
	userName := pickName(user.Nickname, user.Username)

	inst, err := loadInstance(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if inst.Status != instStatusRunning && inst.Status != instStatusReturned {
		return nil, xerror.New(xerror.CodeParamInvalid, "流程已结束, 无法撤销")
	}
	if inst.StartUserId != uid {
		return nil, xerror.New(xerror.CodeForbidden, "仅发起人可撤销")
	}

	err = dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if te := lockInstanceRow(ctx, tx, inst.Id); te != nil {
			return te
		}
		// 锁后复核状态: 加载与加锁之间可能已被并发撤销/驳回
		if te := ensureInstanceStatus(ctx, tx, inst.Id, instStatusRunning, instStatusReturned); te != nil {
			return te
		}
		if te := voidPendingTasks(ctx, tx, inst.Id, "", 0); te != nil {
			return te
		}
		// 原已同意的审批同步置已失效 (流程图/列表不再显示绿色已通过)
		if te := invalidateApprovedTasks(ctx, tx, inst.Id); te != nil {
			return te
		}
		if te := finishInstance(ctx, tx, inst, instStatusCanceled); te != nil {
			return te
		}
		return writeRecord(ctx, tx, inst.Id, 0, "", "", "cancel", uid, userName, "发起人撤销申请")
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	fireEvent(inst.FlowKey, inst.Status, inst.Id, inst.BizId)
	return &v1.FlowInstanceCancelRes{}, nil
}

// InstanceResubmit 重新提交: 仅退回态(6)/已撤销(4)且发起人可操作 —— 撤销后仍可改表单再次发起。
// 沿用实例快照的流程定义 (可顺带修改表单数据); 重新提交流程从头重走, 历史轮次保留。
func (s *sFlow) InstanceResubmit(ctx context.Context, in *v1.FlowInstanceResubmitReq) (res *v1.FlowInstanceResubmitRes, err error) {
	uid := contextx.UserId(ctx)
	if uid == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	inst, err := loadInstance(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if inst.Status != instStatusReturned && inst.Status != instStatusCanceled {
		return nil, xerror.New(xerror.CodeParamInvalid, "当前状态不可重新提交 (仅被驳回或已撤销)")
	}
	if inst.StartUserId != uid {
		return nil, xerror.New(xerror.CodeForbidden, "仅发起人可重新提交")
	}

	// 流程树/表单定义取实例快照 (兜底定义行, 兼容迁移前数据)
	formConf, flowConf := inst.FormConf, inst.FlowConf
	if formConf == "" || flowConf == "" {
		def, derr := loadDefinition(ctx, inst.DefinitionId)
		if derr != nil {
			return nil, derr
		}
		if formConf == "" {
			formConf = def.FormConf
		}
		if flowConf == "" {
			flowConf = def.FlowConf
		}
	}
	root, err := parseFlowConf(flowConf)
	if err != nil {
		return nil, xerror.New(xerror.CodeParamInvalid, err.Error())
	}

	// 表单更新 (传了 FormData 则整份替换并校验必填)
	formData := inst.FormData
	if in.FormData != nil {
		fields, fe := parseFormConf(formConf)
		if fe != nil {
			return nil, xerror.New(xerror.CodeParamInvalid, fe.Error())
		}
		for _, f := range fields {
			if !f.Required {
				continue
			}
			v, ok := in.FormData[f.Key]
			if !ok || isEmptyValue(v) {
				return nil, xerror.New(xerror.CodeParamInvalid, "请填写「"+f.Label+"」")
			}
		}
		b, me := json.Marshal(in.FormData)
		if me != nil {
			return nil, xerror.Wrap(xerror.CodeParamInvalid, me)
		}
		formData = string(b)
	}

	// 自选审批人: 传了则整份替换并落快照; 未传则沿用实例已存快照 (发起时写入)
	selfSelects := in.SelfSelects
	selfJSON := ""
	if selfSelects != nil {
		if selfJSON, err = marshalSelfSelects(selfSelects); err != nil {
			return nil, xerror.Wrap(xerror.CodeParamInvalid, err)
		}
	} else {
		selfSelects, err = daoSelfSelects(ctx, inst.Id)
		if err != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, err)
		}
	}
	// 与发起同口径: 自选节点全部选齐才放行重提 (沿用快照时兜底校验存量数据)
	if err = validateSelfSelects(root, selfSelects); err != nil {
		return nil, xerror.New(xerror.CodeParamInvalid, err.Error())
	}

	notify := &notifySink{sender: uid}
	rerun := &entity.WfInstance{
		Id: inst.Id, FlowKey: inst.FlowKey, BizId: inst.BizId, FlowName: inst.FlowName,
		Title: inst.Title, FormData: formData, Status: instStatusRunning,
		StartUserId: uid, StartUserName: inst.StartUserName,
	}
	defSnap := &entity.WfDefinition{Id: inst.DefinitionId, FlowKey: inst.FlowKey, Name: inst.FlowName, FlowConf: flowConf}
	err = dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if e := lockInstanceRow(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := ensureInstanceStatus(ctx, tx, inst.Id, instStatusReturned, instStatusCanceled); e != nil {
			return e
		}
		data := g.Map{"status": instStatusRunning, "current_node_ids": ""}
		if in.FormData != nil {
			data["form_data"] = formData
		}
		if selfJSON != "" {
			data["self_selects"] = selfJSON
		}
		if _, e := tx.Model(dao.WfInstance.Table()).Ctx(ctx).Where("id", inst.Id).Data(data).Update(); e != nil {
			return e
		}
		// 兜底: 清理残留的已同意审批 (旧数据/边界场景), 新一轮从头重审
		if e := invalidateApprovedTasks(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := writeRecord(ctx, tx, inst.Id, 0, "start", "发起人", "resubmit", uid, inst.StartUserName, "修改后重新提交"); e != nil {
			return e
		}
		return advanceFlow(ctx, tx, defSnap, rerun, root, selfSelects, notify)
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	notify.flush(ctx)
	if rerun.Status != instStatusRunning {
		fireEvent(inst.FlowKey, rerun.Status, inst.Id, inst.BizId)
	}
	return &v1.FlowInstanceResubmitRes{}, nil
}

// InstanceTerminate 管理员终止流程: 运行中实例立即结束, 待办作废、原同意置已失效。
func (s *sFlow) InstanceTerminate(ctx context.Context, in *v1.FlowInstanceTerminateReq) (res *v1.FlowInstanceTerminateRes, err error) {
	uid := contextx.UserId(ctx)
	if uid == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	if !contextx.IsAdmin(ctx) {
		return nil, xerror.New(xerror.CodeForbidden, "仅管理员可终止流程")
	}
	user := contextx.LoginUser(ctx)
	userName := pickName(user.Nickname, user.Username)

	inst, err := loadInstance(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if inst.Status != instStatusRunning {
		return nil, xerror.New(xerror.CodeParamInvalid, "仅运行中的流程可终止")
	}

	err = dao.WfInstance.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if e := lockInstanceRow(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := ensureInstanceStatus(ctx, tx, inst.Id, instStatusRunning); e != nil {
			return e
		}
		if e := voidPendingTasks(ctx, tx, inst.Id, "", 0); e != nil {
			return e
		}
		if e := invalidateApprovedTasks(ctx, tx, inst.Id); e != nil {
			return e
		}
		if e := finishInstance(ctx, tx, inst, instStatusTerminated); e != nil {
			return e
		}
		comment := strings.TrimSpace(in.Comment)
		if comment == "" {
			comment = "管理员终止流程"
		}
		return writeRecord(ctx, tx, inst.Id, 0, "", "", "terminate", uid, userName, comment)
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	fireEvent(inst.FlowKey, inst.Status, inst.Id, inst.BizId)
	return &v1.FlowInstanceTerminateRes{}, nil
}

// urgeCooldown 同一实例两次催办的最小间隔。
const urgeCooldown = 10 * time.Minute

// InstanceUrge 发起人催办: 通知实例当前全部待办审批人 (同实例 10 分钟内限一次)。
func (s *sFlow) InstanceUrge(ctx context.Context, in *v1.FlowInstanceUrgeReq) (res *v1.FlowInstanceUrgeRes, err error) {
	uid := contextx.UserId(ctx)
	if uid == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	inst, err := loadInstance(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if inst.StartUserId != uid {
		return nil, xerror.New(xerror.CodeForbidden, "仅发起人可催办")
	}
	if inst.Status != instStatusRunning {
		return nil, xerror.New(xerror.CodeParamInvalid, "流程不在运行中, 无需催办")
	}

	// 限流: 最近一条催办记录仍在冷却期内则拒绝
	var lastUrge *entity.WfRecord
	if err = dao.WfRecord.Ctx(ctx).
		Where("instance_id", inst.Id).
		Where("action", "urge").
		Order("id DESC").Limit(1).Scan(&lastUrge); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if lastUrge != nil && lastUrge.CreatedAt != nil && time.Since(lastUrge.CreatedAt.Time) < urgeCooldown {
		return nil, xerror.New(xerror.CodeTooManyReq, "催办过于频繁, 请稍后再试 (10 分钟一次)")
	}

	// 当前待办审批人 (按人去重)
	var pend []*entity.WfTask
	if err = dao.WfTask.Ctx(ctx).
		Where("instance_id", inst.Id).
		Where("node_type", taskNodeTypeApprove).
		Where("status", taskStatusPending).
		Scan(&pend); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if len(pend) == 0 {
		return nil, xerror.New(xerror.CodeParamInvalid, "当前没有待处理的审批节点")
	}

	comment := strings.TrimSpace(in.Comment)
	if comment == "" {
		comment = "发起人催办, 请尽快处理"
	}
	// wf_record.comment VARCHAR(500), 按列长截断防 INSERT 失败
	comment = truncateRunes(comment, 500)
	user := contextx.LoginUser(ctx)
	if _, err = dao.WfRecord.Ctx(ctx).Data(g.Map{
		"instance_id":   inst.Id,
		"task_id":       0,
		"node_id":       "",
		"node_name":     "",
		"action":        "urge",
		"operator_id":   uid,
		"operator_name": pickName(user.Nickname, user.Username),
		"comment":       comment,
	}).Insert(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	notify := &notifySink{sender: uid}
	seen := map[uint64]bool{}
	for _, t := range pend {
		if seen[t.AssigneeId] {
			continue
		}
		seen[t.AssigneeId] = true
		notify.add(t.AssigneeId,
			fmt.Sprintf("催办提醒: %s", inst.Title),
			fmt.Sprintf("发起人 %s 催办: 「%s」的节点「%s」等待您审批。%s",
				inst.StartUserName, inst.Title, t.NodeName, urgeNote(comment)))
	}
	notify.flush(ctx)
	return &v1.FlowInstanceUrgeRes{}, nil
}

// urgeNote 催办通知中的说明片段。
func urgeNote(comment string) string {
	if comment == "" {
		return ""
	}
	return "催办说明: " + comment
}

// ============================================================
// 内部工具
// ============================================================

// isEmptyValue 表单必填校验的空值判定: 空串/空数组/空 map 均视为未填写
// (附件字段值为 JSON 数组, gconv.String([]) 会得到 "[]" 非空串, 不能直接按字符串判)。
func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	if a, ok := v.([]any); ok {
		return len(a) == 0
	}
	if m, ok := v.(map[string]any); ok {
		return len(m) == 0
	}
	return false
}

// marshalSelfSelects 自选审批人序列化 (空集返回空串, 落库时不写该列)。
func marshalSelfSelects(m map[string][]uint64) (string, error) {
	if len(m) == 0 {
		return "", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// parseSelfSelects 反序列化实例上的自选审批人快照。
func parseSelfSelects(s string) (map[string][]uint64, error) {
	if s == "" {
		return nil, nil
	}
	out := map[string][]uint64{}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// daoSelfSelects 读实例持久化的自选审批人快照 (事务外)。
func daoSelfSelects(ctx context.Context, instanceId uint64) (map[string][]uint64, error) {
	v, err := dao.WfInstance.Ctx(ctx).Where("id", instanceId).Value("self_selects")
	if err != nil {
		return nil, err
	}
	return parseSelfSelects(gconv.String(v))
}

// loadInstance 按 ID 取未删除实例。
func loadInstance(ctx context.Context, id uint64) (*entity.WfInstance, error) {
	var m *entity.WfInstance
	if err := dao.WfInstance.Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Scan(&m); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if m == nil {
		return nil, xerror.New(xerror.CodeNotFound, "流程实例不存在")
	}
	return m, nil
}

// myTasks 取当前用户指定类型/状态的任务。
func myTasks(ctx context.Context, uid uint64, nodeType int, statuses []int) ([]*entity.WfTask, error) {
	var rows []*entity.WfTask
	if err := dao.WfTask.Ctx(ctx).
		Where("assignee_id", uid).
		Where("node_type", nodeType).
		WhereIn("status", statuses).
		Order("id DESC").
		Limit(500). // 防御性上限: 超大待办场景分页语义由实例侧兜底
		Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return rows, nil
}

// instanceIds 任务列表转实例 ID 去重集合。
func instanceIds(tasks []*entity.WfTask) []uint64 {
	seen := map[uint64]bool{}
	out := make([]uint64, 0, len(tasks))
	for _, t := range tasks {
		if !seen[t.InstanceId] {
			seen[t.InstanceId] = true
			out = append(out, t.InstanceId)
		}
	}
	return out
}

// currentNodeNames 实例当前待审批节点名 (顿号分隔, 非运行中为空)。
func currentNodeNames(ctx context.Context, inst *entity.WfInstance) string {
	return currentNodeNamesBatch(ctx, []*entity.WfInstance{inst})[inst.Id]
}

// currentNodeNamesBatch 批量取一组实例的当前待审批节点名 (列表页一次查询, 避免 N+1)。
// 只查运行中实例的待办任务, 非运行中实例不在结果中 (取值为空串)。
func currentNodeNamesBatch(ctx context.Context, rows []*entity.WfInstance) map[uint64]string {
	out := make(map[uint64]string, len(rows))
	running := make([]uint64, 0, len(rows))
	for _, r := range rows {
		if r.Status == instStatusRunning {
			running = append(running, r.Id)
		}
	}
	if len(running) == 0 {
		return out
	}
	var tasks []*entity.WfTask
	if err := dao.WfTask.Ctx(ctx).
		WhereIn("instance_id", running).
		Where("node_type", taskNodeTypeApprove).
		Where("status", taskStatusPending).
		Order("id ASC").Scan(&tasks); err != nil {
		return out
	}
	// 按实例聚合节点名并按首次到达顺序去重
	order := map[uint64][]string{}
	seen := map[uint64]map[string]bool{}
	for _, t := range tasks {
		if seen[t.InstanceId] == nil {
			seen[t.InstanceId] = map[string]bool{}
		}
		if seen[t.InstanceId][t.NodeName] {
			continue
		}
		seen[t.InstanceId][t.NodeName] = true
		order[t.InstanceId] = append(order[t.InstanceId], t.NodeName)
	}
	for id, names := range order {
		out[id] = joinNames(names)
	}
	return out
}

// validateSelfSelects 发起/重提前校验: 节点树中每个"发起人自选"审批节点都已选审批人。
// 缺失时若不拦截, 深层自选节点会在流程推进到它时解析失败, 前一节点的同意永远报错, 实例卡死中途。
func validateSelfSelects(root *Node, selfSelects map[string][]uint64) error {
	for cur := root; cur != nil; cur = cur.Child {
		if cur.Type == nodeTypeApprover && cur.ApproverType == approverTypeSelfSelect && len(selfSelects[cur.Id]) == 0 {
			return fmt.Errorf("节点「%s」为发起人自选, 请在发起时选择审批人", cur.Name)
		}
		if cur.Type == nodeTypeCondition {
			for _, br := range cur.Branches {
				if br.Child == nil {
					continue
				}
				if err := validateSelfSelects(br.Child, selfSelects); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// collectSelfSelectIds 收集节点树中所有"发起人自选"节点的 ID。
func collectSelfSelectIds(root *Node) map[string]bool {
	out := map[string]bool{}
	var walk func(n *Node)
	walk = func(n *Node) {
		for cur := n; cur != nil; cur = cur.Child {
			if cur.Type == nodeTypeApprover && cur.ApproverType == approverTypeSelfSelect {
				out[cur.Id] = true
			}
			if cur.Type == nodeTypeCondition {
				for _, br := range cur.Branches {
					if br.Child != nil {
						walk(br.Child)
					}
				}
			}
		}
	}
	walk(root)
	return out
}

// prevSelfSelects 取上一轮自选节点的实际处理人, 作为重新提交弹窗的默认选择。
func prevSelfSelects(flowConf string, tasks []*entity.WfTask) map[string][]uint64 {
	root, err := parseFlowConf(flowConf)
	if err != nil || root == nil {
		return nil
	}
	selfIds := collectSelfSelectIds(root)
	if len(selfIds) == 0 {
		return nil
	}
	out := map[string][]uint64{}
	seen := map[string]map[uint64]bool{}
	for _, t := range tasks {
		if !selfIds[t.NodeId] || t.Status == taskStatusVoid || t.AssigneeId == 0 {
			continue
		}
		if seen[t.NodeId] == nil {
			seen[t.NodeId] = map[uint64]bool{}
		}
		if seen[t.NodeId][t.AssigneeId] {
			continue
		}
		seen[t.NodeId][t.AssigneeId] = true
		out[t.NodeId] = append(out[t.NodeId], t.AssigneeId)
	}
	return out
}

// instItem 实例转 API 条目。
func instItem(r *entity.WfInstance, taskId uint64, currentNodes string) *v1.FlowInstanceItem {
	return &v1.FlowInstanceItem{
		Id:            r.Id,
		DefinitionId:  r.DefinitionId,
		FlowKey:       r.FlowKey,
		BizId:         r.BizId,
		FlowName:      r.FlowName,
		Title:         r.Title,
		Status:        r.Status,
		StartUserId:   r.StartUserId,
		StartUserName: r.StartUserName,
		CurrentNodes:  currentNodes,
		TaskId:        taskId,
		CreatedAt:     r.CreatedAt,
		FinishedAt:    r.FinishedAt,
	}
}

// taskItem 任务转 API 条目。
func taskItem(t *entity.WfTask) *v1.FlowTaskItem {
	return &v1.FlowTaskItem{
		Id:           t.Id,
		InstanceId:   t.InstanceId,
		NodeId:       t.NodeId,
		NodeName:     t.NodeName,
		NodeType:     t.NodeType,
		SignType:     t.SignType,
		AssigneeId:   t.AssigneeId,
		AssigneeName: t.AssigneeName,
		Status:       t.Status,
		Comment:      t.Comment,
		ReceiveTime:  t.ReceiveTime,
		ActedAt:      t.ActedAt,
	}
}

// recordItem 记录转 API 条目。
func recordItem(r *entity.WfRecord) *v1.FlowRecordItem {
	return &v1.FlowRecordItem{
		Id:           r.Id,
		NodeName:     r.NodeName,
		Action:       r.Action,
		OperatorId:   r.OperatorId,
		OperatorName: r.OperatorName,
		Comment:      r.Comment,
		CreatedAt:    r.CreatedAt,
	}
}

// emptyInstancePage 空结果页。
func emptyInstancePage(in *v1.FlowInstanceListReq) *v1.FlowInstanceListRes {
	page := response.Page([]any{}, 0, in.Page, in.PageSize)
	return (*v1.FlowInstanceListRes)(&page)
}
