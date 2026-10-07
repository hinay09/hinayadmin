// Package leave 请假申请 —— 业务型流程审批 Demo (docs/pro/flow-integration.md 示例落地)。
//
// 接入模式 (不走表单设计器):
//   - 业务数据存 biz_leave 单表, 页面只提交业务字段;
//   - LeaveSubmit 调 flow.StartForBiz(flowKey) 发起审批, 表单数据由业务表字段组装,
//     仅作流程详情页只读快照 + 条件分支求值 (days>3 加签);
//   - 审批状态 flow_status 不由业务代码推进 —— 全部来自 flow.RegisterBizListener
//     五回调 (通过/退回/撤回/撤销/终止), 业务侧只做幂等回写;
//   - 撤销/重提复用审批引擎的实例动作 (InstanceCancel/InstanceResubmit),
//     前端审批操作仍统一在「审批中心 → 我的审批」完成。
package leave

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	v1flow "hinay.cn/admin/api/flow/v1"
	v1 "hinay.cn/admin/api/leave/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/flow"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

// flowKeyLeave 绑定流程定义的「流程标识」(wf_definition.flow_key),
// 与种子 p016 的演示流程一致; 换流程只需在流程定义页改配置, 业务代码不动。
const flowKeyLeave = "biz_leave"

// 业务表冗余的审批状态编码 (与实例 status 是两套编码, 见 DDL 注释)。
const (
	flowStatusApproving = 0 // 审批中 (提交时写入)
	flowStatusApproved  = 1 // 已通过 (OnApproved 回调)
	flowStatusReturned  = 2 // 被退回 (OnReturned 回调)
	flowStatusCanceled  = 3 // 已撤销 (OnCanceled 回调)
	flowStatusTermed    = 4 // 已终止 (OnTerminated 回调)
)

func init() {
	service.RegisterLeave(&sLeave{})
	// 状态回调: 引擎事务提交后同步尽力调用, 业务侧须幂等。
	flow.RegisterBizListener(flowKeyLeave, flow.BizListener{
		OnApproved:   func(instanceId, bizId uint64) { writeBackFlowStatus(instanceId, bizId, flowStatusApproved) },
		OnReturned:   func(instanceId, bizId uint64) { writeBackFlowStatus(instanceId, bizId, flowStatusReturned) },
		OnWithdrawn:  func(instanceId, bizId uint64) { writeBackFlowStatus(instanceId, bizId, flowStatusReturned) }, // 撤回≈退回: 单据回到可修改重提态
		OnCanceled:   func(instanceId, bizId uint64) { writeBackFlowStatus(instanceId, bizId, flowStatusCanceled) },
		OnTerminated: func(instanceId, bizId uint64) { writeBackFlowStatus(instanceId, bizId, flowStatusTermed) },
	})
}

// writeBackFlowStatus 回调幂等回写: 按 (bizId, instanceId) 定位, 旧实例的迟到
// 回调不会覆盖新实例的状态 (撤销后重新发起等场景)。
func writeBackFlowStatus(instanceId, bizId uint64, status int) {
	_, err := dao.BizLeave.Ctx(context.Background()).
		Where("id", bizId).Where("flow_instance", instanceId).
		Data(g.Map{"flow_status": status}).Update()
	if err != nil {
		g.Log().Warningf(context.Background(),
			"leave flow callback write back failed (biz=%d inst=%d status=%d): %v", bizId, instanceId, status, err)
	}
}

type sLeave struct{}

// ============================================================
// 增删改查
// ============================================================

// LeaveList 请假申请分页列表: admin 看全部 (可 Mine=1 只看自己的), 其余仅本人。
func (s *sLeave) LeaveList(ctx context.Context, in *v1.LeaveListReq) (res *v1.LeaveListRes, err error) {
	uid := contextx.UserId(ctx)
	q := dao.BizLeave.Ctx(ctx).Where("deleted_at IS NULL")
	if !contextx.IsAdmin(ctx) || (in.Mine != nil && *in.Mine == 1) {
		q = q.Where("create_id", uid)
	}
	if in.LeaveType != nil {
		q = q.Where("leave_type", *in.LeaveType)
	}
	if in.FlowStatus != nil {
		q = q.Where("flow_status", *in.FlowStatus)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*entity.BizLeave
	if err = q.Order("id DESC").Page(in.Page, in.PageSize).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 申请人昵称一次批量取回, 避免逐行查询 (N+1)
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.CreateId)
	}
	names := userNames(ctx, ids...)
	list := make([]*v1.LeaveItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, leaveItem(r, names[r.CreateId]))
	}
	page := response.Page(list, int64(total), in.Page, in.PageSize)
	return (*v1.LeaveListRes)(&page), nil
}

// LeaveCreate 新增请假申请 (草稿, 不发起审批)。
func (s *sLeave) LeaveCreate(ctx context.Context, in *v1.LeaveCreateReq) (res *v1.LeaveCreateRes, err error) {
	if err = checkDateRange(in.StartDate, in.EndDate); err != nil {
		return nil, err
	}
	id, err := dao.BizLeave.Ctx(ctx).Data(s.buildData(in.LeaveType, in.StartDate, in.EndDate, in.Days, in.Reason)).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.LeaveCreateRes{Id: uint64(id)}, nil
}

// LeaveUpdate 修改请假申请: 仅本人 (admin 放行), 且未进入审批流或已退回/撤销。
func (s *sLeave) LeaveUpdate(ctx context.Context, in *v1.LeaveUpdateReq) (res *v1.LeaveUpdateRes, err error) {
	if err = checkDateRange(in.StartDate, in.EndDate); err != nil {
		return nil, err
	}
	lv, err := s.loadForEdit(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if _, err = dao.BizLeave.Ctx(ctx).Where("id", lv.Id).
		Data(s.buildData(in.LeaveType, in.StartDate, in.EndDate, in.Days, in.Reason)).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.LeaveUpdateRes{}, nil
}

// LeaveDelete 删除请假申请: 审批中/退回态不可删 (先撤销), 其余软删。
func (s *sLeave) LeaveDelete(ctx context.Context, in *v1.LeaveDeleteReq) (res *v1.LeaveDeleteRes, err error) {
	lv, err := s.loadForEdit(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if _, err = dao.BizLeave.Ctx(ctx).Where("id", lv.Id).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.LeaveDeleteRes{}, nil
}

// ============================================================
// 审批接入 (提交 / 撤销)
// ============================================================

// LeaveSubmit 提交审批 (由公共弹窗 FlowSubmitDialog 触发, :show-form=false 模式:
// 表单数据由 buildFormData 从业务表组装, 弹窗产出 selfSelects 透传引擎):
//   - 草稿 (flow_instance=0): flow.StartForBiz 发起新实例;
//   - 被退回(2)/已撤销(3): 复用原实例 InstanceResubmit, 流程从头重走、历史保留,
//     表单快照以业务表最新字段重建。
//
// 双击防线: 前端按钮 loading + 回填条件更新 (引擎实例状态机天然拒绝重复重提)。
func (s *sLeave) LeaveSubmit(ctx context.Context, in *v1.LeaveSubmitReq) (res *v1.LeaveSubmitRes, err error) {
	uid := contextx.UserId(ctx)
	lv, err := s.loadOne(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if lv.CreateId != uid {
		return nil, xerror.New(xerror.CodeForbidden, "仅申请人可提交审批")
	}

	switch {
	case lv.FlowInstance == 0: // 草稿首次发起
		user := contextx.LoginUser(ctx)
		title := fmt.Sprintf("请假申请-%s %g天", pickName(user.Nickname, user.Username), lv.Days)
		instanceId, fe := flow.StartForBiz(ctx, flowKeyLeave, title, buildFormData(lv), lv.Id, uid, in.SelfSelects)
		if fe != nil {
			// 常见失败: 流程未发布/审批人失效/自选未传; 单据保持草稿, 修正后可重试
			return nil, fe
		}
		if _, ue := dao.BizLeave.Ctx(ctx).Where("id", lv.Id).Where("flow_instance", 0).
			Data(g.Map{"flow_status": flowStatusApproving, "flow_instance": instanceId}).Update(); ue != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, ue)
		}
		return &v1.LeaveSubmitRes{FlowInstance: instanceId}, nil

	case lv.FlowStatus == flowStatusReturned || lv.FlowStatus == flowStatusCanceled:
		// 退回/撤销后重提: 引擎按实例状态校验 (仅 6=退回/4=撤销 可重提), 表单整份替换;
		// selfSelects 来自公共提交弹窗 (无自选节点时为空 map), 与发起同一语义
		if _, re := service.Flow().InstanceResubmit(ctx, &v1flow.FlowInstanceResubmitReq{
			Id:          lv.FlowInstance,
			FormData:    buildFormData(lv),
			SelfSelects: in.SelfSelects,
		}); re != nil {
			return nil, re
		}
		if _, ue := dao.BizLeave.Ctx(ctx).Where("id", lv.Id).
			Where("flow_status", lv.FlowStatus). // 防回调并发, 条件回写
			Data(g.Map{"flow_status": flowStatusApproving}).Update(); ue != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, ue)
		}
		return &v1.LeaveSubmitRes{FlowInstance: lv.FlowInstance}, nil

	case lv.FlowStatus == flowStatusApproving:
		return nil, xerror.New(xerror.CodeParamInvalid, "该单据已在审批中, 请勿重复提交")
	default:
		return nil, xerror.New(xerror.CodeParamInvalid, "该单据已审批结束, 不可再次提交")
	}
}

// LeaveCancel 撤销审批: 转调引擎 InstanceCancel (校验发起人/实例状态),
// flow_status=3 由 OnCanceled 回调写回 —— 本方法不直接改审批状态。
func (s *sLeave) LeaveCancel(ctx context.Context, in *v1.LeaveCancelReq) (res *v1.LeaveCancelRes, err error) {
	uid := contextx.UserId(ctx)
	lv, err := s.loadOne(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if lv.CreateId != uid {
		return nil, xerror.New(xerror.CodeForbidden, "仅申请人可撤销")
	}
	if lv.FlowInstance == 0 {
		return nil, xerror.New(xerror.CodeParamInvalid, "该单据尚未提交审批")
	}
	if lv.FlowStatus != flowStatusApproving && lv.FlowStatus != flowStatusReturned {
		return nil, xerror.New(xerror.CodeParamInvalid, "流程已结束, 无需撤销")
	}
	if _, err = service.Flow().InstanceCancel(ctx, &v1flow.FlowInstanceCancelReq{Id: lv.FlowInstance}); err != nil {
		return nil, err
	}
	return &v1.LeaveCancelRes{}, nil
}

// ============================================================
// 内部工具
// ============================================================

// buildData 组装业务字段列 (新增/修改共用)。
func (s *sLeave) buildData(leaveType int, startDate, endDate string, days float64, reason string) g.Map {
	return g.Map{
		"leave_type": leaveType,
		"start_date": startDate,
		"end_date":   endDate,
		"days":       days,
		"reason":     reason,
	}
}

// loadOne 按 ID 取未删除单据。
func (s *sLeave) loadOne(ctx context.Context, id uint64) (*entity.BizLeave, error) {
	var lv *entity.BizLeave
	if err := dao.BizLeave.Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Scan(&lv); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if lv == nil {
		return nil, xerror.New(xerror.CodeNotFound, "请假单不存在")
	}
	return lv, nil
}

// loadForEdit 取可编辑 (删除/修改共用) 的单据: 本人或 admin,
// 且未发起或已退回/已撤销 —— 审批中与已结束的单子不允许改删。
func (s *sLeave) loadForEdit(ctx context.Context, id uint64) (*entity.BizLeave, error) {
	uid := contextx.UserId(ctx)
	lv, err := s.loadOne(ctx, id)
	if err != nil {
		return nil, err
	}
	if lv.CreateId != uid && !contextx.IsAdmin(ctx) {
		return nil, xerror.New(xerror.CodeForbidden, "仅申请人可操作")
	}
	if lv.FlowInstance > 0 &&
		lv.FlowStatus != flowStatusReturned && lv.FlowStatus != flowStatusCanceled {
		return nil, xerror.New(xerror.CodeParamInvalid, "审批中或已结束的单据不可修改/删除 (审批中请先撤销)")
	}
	return lv, nil
}

// checkDateRange 校验日期顺序 (格式由 API 层 v 校验保证)。
func checkDateRange(start, end string) error {
	st, se := time.ParseInLocation("2006-01-02", start, time.Local)
	et, ee := time.ParseInLocation("2006-01-02", end, time.Local)
	if se != nil || ee != nil {
		return xerror.New(xerror.CodeParamInvalid, "日期格式不合法")
	}
	if et.Before(st) {
		return xerror.New(xerror.CodeParamInvalid, "结束日期不能早于开始日期")
	}
	return nil
}

// buildFormData 业务字段 → 流程表单快照: 键与流程定义 form_conf 对齐
// (审批详情页只读展示 + 条件分支按 days 求值), 数据本体仍在 biz_leave 表。
func buildFormData(lv *entity.BizLeave) map[string]any {
	return map[string]any{
		"leave_type": leaveTypeName(lv.LeaveType),
		"date_range": fmt.Sprintf("%s ~ %s", lv.StartDate.Format("Y-m-d"), lv.EndDate.Format("Y-m-d")),
		"days":       lv.Days,
		"reason":     lv.Reason,
	}
}

// leaveTypeName 类型编码 → 中文名 (与流程定义 form_conf 的 select options 对齐)。
func leaveTypeName(t int) string {
	names := map[int]string{1: "事假", 2: "病假", 3: "年假", 4: "调休", 5: "其他"}
	if n, ok := names[t]; ok {
		return n
	}
	return "其他"
}

// leaveItem 实体转条目。
func leaveItem(r *entity.BizLeave, createName string) *v1.LeaveItem {
	return &v1.LeaveItem{
		Id: r.Id, LeaveType: r.LeaveType,
		StartDate: r.StartDate.Format("Y-m-d"), EndDate: r.EndDate.Format("Y-m-d"),
		Days: r.Days, Reason: r.Reason,
		FlowStatus: r.FlowStatus, FlowInstance: r.FlowInstance,
		CreateId: r.CreateId, CreateName: createName, CreatedAt: r.CreatedAt,
	}
}

// userNames 批量取用户昵称 (列表展示申请人; 查不到回退空串)。
func userNames(ctx context.Context, ids ...uint64) map[uint64]string {
	out := map[uint64]string{}
	valid := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			valid = append(valid, id)
		}
	}
	if len(valid) == 0 {
		return out
	}
	var users []*entity.SysUser
	if err := dao.SysUser.Ctx(ctx).WhereIn("id", valid).Scan(&users); err != nil {
		return out
	}
	for _, u := range users {
		out[u.Id] = pickName(u.Nickname, u.Username)
	}
	return out
}

// pickName 昵称优先, 兜底登录名。
func pickName(nickname, username string) string {
	if nickname != "" {
		return nickname
	}
	return username
}
