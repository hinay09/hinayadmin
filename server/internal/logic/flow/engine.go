// Package flow 自由审批流业务逻辑 — 推进引擎。
//
// 引擎语义:
//   - advanceFlow 从指定节点沿主链推进, 遇审批节点解析审批人、落任务行后挂起返回;
//     遇抄送节点落待阅行并继续; 遇条件节点求值后深入命中分支; 走到链尾则整单通过。
//   - 或签: 任一人处理即过节点 (其余待办作废); 会签: 全部同意才过,
//     任一驳回=该节点驳回 (同伴待办作废), 按所选目标回退 (任意已审批节点或发起人)。
//   - 实例引用定义版本行 (发布后不可变), 定义后续修改/停用不影响在途实例。
package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"

	v1msg "hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/post"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/xerror"
)

// approverUser 解析出的审批人 (ID + 冗余昵称)。
type approverUser struct {
	Id   uint64
	Name string
}

// u64str uint64 转字符串。
func u64str(v uint64) string {
	return strconv.FormatUint(v, 10)
}

// ============================================================
// 推进
// ============================================================

// advanceFlow 从 from 节点开始推进 (事务内调用)。
// notify 收集事务内产生的通知, 由调用方在事务提交后发送。
func advanceFlow(ctx context.Context, tx gdb.TX, def *entity.WfDefinition, inst *entity.WfInstance,
	from *Node, selfSelects map[string][]uint64, notify *notifySink) error {

	cur := from
	for cur != nil {
		switch cur.Type {
		case nodeTypeStart:
			cur = cur.Child

		case nodeTypeApprover:
			users, err := resolveApprovers(ctx, cur, inst, selfSelects)
			if err != nil {
				return err
			}
			signType := taskSignAny
			if cur.SignType == "all" {
				signType = taskSignAll
			}
			for _, u := range users {
				if _, err = tx.Model(dao.WfTask.Table()).Ctx(ctx).Data(g.Map{
					"instance_id":   inst.Id,
					"node_id":       cur.Id,
					"node_name":     cur.Name,
					"node_type":     taskNodeTypeApprove,
					"sign_type":     signType,
					"assignee_id":   u.Id,
					"assignee_name": u.Name,
					"status":        taskStatusPending,
				}).Insert(); err != nil {
					return err
				}
				notify.add(u.Id,
					fmt.Sprintf("审批待办: %s", inst.Title),
					fmt.Sprintf("%s 提交的「%s」等待您审批, 请到 审批中心-我的审批 处理。", inst.StartUserName, inst.FlowName))
			}
			if _, err = tx.Model(dao.WfInstance.Table()).Ctx(ctx).Where("id", inst.Id).
				Data(g.Map{"current_node_ids": cur.Id}).Update(); err != nil {
				return err
			}
			return nil // 挂起等待审批

		case nodeTypeCC:
			users, err := resolveApprovers(ctx, cur, inst, selfSelects)
			if err != nil {
				// 抄送是非阻塞动作: 解析失败 (角色/岗位无人、成员全被禁用) 只跳过本节点,
				// 不阻断上游审批 —— 否则下游抄送配置失实会让前一节点的"同意"永远报错, 流程卡死。
				g.Log().Warningf(ctx, "flow instance %d cc node %s resolve failed, skipped: %v", inst.Id, cur.Id, err)
				if err = writeRecord(ctx, tx, inst.Id, 0, cur.Id, cur.Name, "cc", 0, "系统",
					"抄送对象解析失败, 已跳过: "+truncateRunes(err.Error(), 400)); err != nil {
					return err
				}
				cur = cur.Child
				continue
			}
			names := make([]string, 0, len(users))
			for _, u := range users {
				if _, err = tx.Model(dao.WfTask.Table()).Ctx(ctx).Data(g.Map{
					"instance_id":   inst.Id,
					"node_id":       cur.Id,
					"node_name":     cur.Name,
					"node_type":     taskNodeTypeCC,
					"sign_type":     taskSignAny,
					"assignee_id":   u.Id,
					"assignee_name": u.Name,
					"status":        taskStatusPending,
				}).Insert(); err != nil {
					return err
				}
				names = append(names, u.Name)
				notify.add(u.Id,
					fmt.Sprintf("流程抄送: %s", inst.Title),
					fmt.Sprintf("%s 提交的「%s」已抄送给您, 可在 审批中心-我的审批-抄送我的 查看。", inst.StartUserName, inst.FlowName))
			}
			if err = writeRecord(ctx, tx, inst.Id, 0, cur.Id, cur.Name, "cc", 0, "系统",
				"抄送: "+joinNames(names)); err != nil {
				return err
			}
			cur = cur.Child // 抄送不阻塞, 继续推进

		case nodeTypeCondition:
			var formData map[string]any
			if inst.FormData != "" {
				_ = json.Unmarshal([]byte(inst.FormData), &formData)
			}
			br, err := evalBranches(cur, formData)
			if err != nil {
				return err
			}
			cur = br.Child

		default:
			return fmt.Errorf("节点「%s」类型不合法: %s", cur.Name, cur.Type)
		}
	}
	// 走到链尾: 全部节点通过
	return finishInstance(ctx, tx, inst, instStatusApproved)
}

// finishInstance 结束实例 (事务内调用), 同步刷新内存状态。
func finishInstance(ctx context.Context, tx gdb.TX, inst *entity.WfInstance, status int) error {
	if _, err := tx.Model(dao.WfInstance.Table()).Ctx(ctx).Where("id", inst.Id).Data(g.Map{
		"status":           status,
		"finished_at":      gtime.Now(),
		"current_node_ids": "",
	}).Update(); err != nil {
		return err
	}
	inst.Status = status
	if status == instStatusApproved {
		return writeRecord(ctx, tx, inst.Id, 0, "", "", "finish", 0, "系统", "流程结束: 全部节点通过")
	}
	return nil
}

// lockInstanceRow 事务内对实例行加排他锁 (SELECT ... FOR UPDATE):
// 串行化同一实例的并发动作 (或签同时同意/会签并发/驳回与撤销赛跑),
// 防止双推进生成重复任务或会签计数快照读到旧值卡死节点。
// 必须是动作事务内的第一条语句, 保证锁顺序一致不产生死锁。
func lockInstanceRow(ctx context.Context, tx gdb.TX, instanceId uint64) error {
	_, err := tx.Model(dao.WfInstance.Table()).Ctx(ctx).Where("id", instanceId).LockUpdate().One()
	return err
}

// ensureInstanceStatus 锁后复核实例状态仍在允许集合内:
// 任务/实例在事务外加载, 加锁前可能已被并发动作 (撤销/驳回/终止) 改变状态。
func ensureInstanceStatus(ctx context.Context, tx gdb.TX, instanceId uint64, allowed ...int) error {
	v, err := tx.Model(dao.WfInstance.Table()).Ctx(ctx).Where("id", instanceId).Value("status")
	if err != nil {
		return err
	}
	st := v.Int()
	for _, a := range allowed {
		if st == a {
			return nil
		}
	}
	return xerror.New(xerror.CodeParamInvalid, "流程状态已变化, 请刷新后重试")
}

// voidPendingTasks 作废待办任务 (事务内调用)。
// nodeId 为空时作废实例全部待办; excludeTaskId 排除指定任务 (或签抢先场景)。
func voidPendingTasks(ctx context.Context, tx gdb.TX, instanceId uint64, nodeId string, excludeTaskId uint64) error {
	q := tx.Model(dao.WfTask.Table()).Ctx(ctx).
		Where("instance_id", instanceId).
		Where("status", taskStatusPending)
	if nodeId != "" {
		q = q.Where("node_id", nodeId)
	}
	if excludeTaskId > 0 {
		q = q.Where("id != ?", excludeTaskId)
	}
	_, err := q.Data(g.Map{"status": taskStatusVoid, "acted_at": nil}).Update()
	return err
}

// invalidateApprovedTasks 已同意的审批任务批量置"已失效" (事务内调用):
// 退回发起人/驳回到节点/撤销/重提时, 原先的同意不再计入当前轮次, 流程图与列表
// 随之同步为"已失效"; 抄送已阅不回退, 流转记录 (历史) 不动。
func invalidateApprovedTasks(ctx context.Context, tx gdb.TX, instanceId uint64) error {
	_, err := tx.Model(dao.WfTask.Table()).Ctx(ctx).
		Where("instance_id", instanceId).
		Where("node_type", taskNodeTypeApprove).
		Where("status", taskStatusApproved).
		Data(g.Map{"status": taskStatusInvalid}).Update()
	return err
}

// writeRecord 追加流转记录 (事务内调用)。
// wf_record.comment VARCHAR(500): 加签/减签/转办等会拼接昵称前缀, 按列长截断防 INSERT 失败。
func writeRecord(ctx context.Context, tx gdb.TX, instanceId, taskId uint64, nodeId, nodeName, action string,
	operatorId uint64, operatorName, comment string) error {
	_, err := tx.Model(dao.WfRecord.Table()).Ctx(ctx).Data(g.Map{
		"instance_id":   instanceId,
		"task_id":       taskId,
		"node_id":       nodeId,
		"node_name":     nodeName,
		"action":        action,
		"operator_id":   operatorId,
		"operator_name": operatorName,
		"comment":       truncateRunes(comment, 500),
	}).Insert()
	return err
}

// joinNames 逗号拼接昵称。
func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += "、"
		}
		out += n
	}
	return out
}

// ============================================================
// 审批人解析
// ============================================================

// resolveApprovers 按节点规则解析审批/抄送人列表。
func resolveApprovers(ctx context.Context, node *Node, inst *entity.WfInstance, selfSelects map[string][]uint64) ([]approverUser, error) {
	switch node.ApproverType {
	case approverTypeUser:
		return lookupUsers(ctx, node.ApproverIds, "节点「"+node.Name+"」指定的成员")
	case approverTypeRole:
		ids, err := roleUserIds(ctx, node.ApproverIds)
		if err != nil {
			return nil, err
		}
		return lookupUsers(ctx, ids, "节点「"+node.Name+"」指定角色下的用户")
	case approverTypeSelfSelect:
		ids := selfSelects[node.Id]
		if len(ids) == 0 {
			return nil, fmt.Errorf("节点「%s」为发起人自选, 请在发起时选择审批人", node.Name)
		}
		return lookupUsers(ctx, ids, "节点「"+node.Name+"」发起人选择的审批人")
	case approverTypePost:
		ids := make([]uint64, 0, len(node.ApproverIds))
		for _, pid := range node.ApproverIds {
			pu, pe := post.UsersOfPost(ctx, pid)
			if pe != nil {
				return nil, pe
			}
			for _, u := range pu {
				ids = append(ids, u.Id)
			}
		}
		users, err := lookupUsers(ctx, ids, "节点「"+node.Name+"」指定岗位下的用户")
		if err != nil {
			return nil, err
		}
		return users, nil
	case approverTypeSuperior:
		u, err := post.FindSuperior(ctx, inst.StartUserId)
		if err != nil {
			return nil, err
		}
		return []approverUser{{Id: u.Id, Name: u.Name}}, nil
	case approverTypeInitiator:
		return []approverUser{{Id: inst.StartUserId, Name: inst.StartUserName}}, nil
	default:
		return nil, fmt.Errorf("节点「%s」未设置审批人规则", node.Name)
	}
}

// lookupUsers 按 ID 集合取启用用户 (去重), 全部无效时报错。
func lookupUsers(ctx context.Context, ids []uint64, what string) ([]approverUser, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%s 为空", what)
	}
	var rows []*entity.SysUser
	if err := dao.SysUser.Ctx(ctx).
		WhereIn("id", ids).Where("status", 1).Where("deleted_at IS NULL").
		Scan(&rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s 不存在或已被禁用", what)
	}
	seen := map[uint64]bool{}
	out := make([]approverUser, 0, len(rows))
	for _, r := range rows {
		if seen[r.Id] {
			continue
		}
		seen[r.Id] = true
		out = append(out, approverUser{Id: r.Id, Name: pickName(r.Nickname, r.Username)})
	}
	return out, nil
}

// roleUserIds 角色展开为用户 ID (casbin g 行: v0=用户ID, v1=角色ID)。
func roleUserIds(ctx context.Context, roleIds []uint64) ([]uint64, error) {
	vals, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "g").
		WhereIn("v1", roleIds).
		Array("v0")
	if err != nil {
		return nil, err
	}
	seen := map[uint64]bool{}
	out := make([]uint64, 0, len(vals))
	for _, v := range vals {
		id := gconv.Uint64(v)
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}

// pickName 昵称优先, 兜底登录名。
func pickName(nickname, username string) string {
	if nickname != "" {
		return nickname
	}
	return username
}

// ============================================================
// 条件求值
// ============================================================

// evalBranches 依序评估分支 (外层 OR), 未命中走默认分支, 均无则报错。
func evalBranches(n *Node, formData map[string]any) (*Branch, error) {
	var def *Branch
	for _, br := range n.Branches {
		if br.IsDefault {
			def = br
			continue
		}
		if branchMatch(br, formData) {
			return br, nil
		}
	}
	if def != nil {
		return def, nil
	}
	return nil, fmt.Errorf("条件节点「%s」无命中分支且未配置默认分支", n.Name)
}

// branchMatch 分支命中: 条件组任一整组成立 (组内 AND)。
func branchMatch(br *Branch, formData map[string]any) bool {
	for _, group := range br.Conditions {
		all := true
		for _, c := range group {
			if !condMatch(&c, formData) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// condMatch 单条件求值: 数值按数值比较, 其余按字符串比较。
func condMatch(c *Cond, formData map[string]any) bool {
	v, ok := formData[c.Field]
	if !ok || v == nil {
		return false
	}
	switch c.Op {
	case "eq":
		return looseEqual(v, c.Value)
	case "ne":
		return !looseEqual(v, c.Value)
	case "gt", "lt", "ge", "le":
		a, okA := toFloat(v)
		b, okB := toFloat(c.Value)
		if !okA || !okB {
			return false
		}
		switch c.Op {
		case "gt":
			return a > b
		case "lt":
			return a < b
		case "ge":
			return a >= b
		default:
			return a <= b
		}
	case "in":
		target := gconv.String(v)
		for _, item := range gconv.Strings(c.Value) {
			if item == target {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// looseEqual 宽松相等: 双方可转数值则数值比较, 否则字符串比较。
func looseEqual(a, b any) bool {
	fa, okA := toFloat(a)
	fb, okB := toFloat(b)
	if okA && okB {
		return fa == fb
	}
	return gconv.String(a) == gconv.String(b)
}

// toFloat 数值转换 (非数值返回 false)。
func toFloat(v any) (float64, bool) {
	f, err := strconv.ParseFloat(gconv.String(v), 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// ============================================================
// 通知与回调 (事务提交后调用)
// ============================================================

// notifyMsg 站内私信通知。
type notifyMsg struct {
	receiver uint64
	title    string
	content  string
}

// notifySink 事务内累积、提交后发送的通知队列。
type notifySink struct {
	sender uint64
	msgs   []notifyMsg
}

// add 事务内追加一条待发通知 (不落库, 提交后由 flush 统一发送)。
func (n *notifySink) add(receiver uint64, title, content string) {
	n.msgs = append(n.msgs, notifyMsg{receiver: receiver, title: title, content: content})
}

// flush 事务提交后发送系统通知 (定向用户, 尽力而为, 失败仅记日志)。
// 用系统通知而非私信: 无"不能给自己发"的限制 (发起人自审也能收到待办提醒),
// 且后台上下文 (无登录用户, 如业务模块 StartForBiz) 同样可发。
func (n *notifySink) flush(ctx context.Context) {
	if len(n.msgs) == 0 {
		return
	}
	// 相同标题+内容合并接收人, 一次系统通知批量定向
	type batch struct {
		title, content string
		ids            []uint64
		seen           map[uint64]bool
	}
	var order []*batch
	index := map[string]*batch{}
	for _, m := range n.msgs {
		if m.receiver == 0 {
			continue
		}
		key := m.title + "\x00" + m.content
		b, ok := index[key]
		if !ok {
			b = &batch{title: m.title, content: m.content, seen: map[uint64]bool{}}
			index[key] = b
			order = append(order, b)
		}
		if b.seen[m.receiver] {
			continue
		}
		b.seen[m.receiver] = true
		b.ids = append(b.ids, m.receiver)
	}
	for _, b := range order {
		if _, err := service.Message().SystemCreate(ctx, &v1msg.MessageSystemCreateReq{
			Title:       truncateRunes(b.title, 128),
			Content:     b.content,
			Level:       1,
			Status:      1,
			TargetScope: 3,
			TargetIds:   b.ids,
		}); err != nil {
			g.Log().Warningf(ctx, "flow notify users %v failed: %v", b.ids, err)
		}
	}
	n.msgs = nil
}

// truncateRunes 按字符截断 (标题限长 128)。
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// fireEvent 触发业务回调 (尽力而为, panic 保护)。
func fireEvent(flowKey string, status int, instanceId, bizId uint64) {
	l, ok := bizListeners[flowKey]
	if !ok {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			g.Log().Warningf(context.Background(), "flow biz listener panic (%s): %v", flowKey, r)
		}
	}()
	switch status {
	case instStatusApproved:
		if l.OnApproved != nil {
			l.OnApproved(instanceId, bizId)
		}
	case instStatusReturned:
		if l.OnReturned != nil {
			l.OnReturned(instanceId, bizId)
		}
	case instStatusCanceled:
		if l.OnCanceled != nil {
			l.OnCanceled(instanceId, bizId)
		}
	case instStatusTerminated:
		if l.OnTerminated != nil {
			l.OnTerminated(instanceId, bizId)
		}
	}
}
