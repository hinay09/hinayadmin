// Package flow — 业务模块编程式接入 (见 docs/pro/flow-integration.md)。
package flow

import (
	"context"

	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/utility/xerror"
)

// StartForBiz 业务模块发起流程: 按 flow_key 取最新已发布且未停用的定义,
// 以 initiatorId 作为发起人创建实例并推进。bizId 用于回调定位业务记录 (0=无关联)。
//
// 注意:
//   - 流程定义中含"发起人自选"节点时, 须由调用方传 selfSelects {nodeId: [userId]};
//   - 返回实例 ID; 表单必填校验失败/定义缺失返回 error;
//   - 状态回调走 RegisterBizListener 注册的 BizListener (事务提交后尽力调用)。
func StartForBiz(ctx context.Context, flowKey, title string, formData map[string]any,
	bizId, initiatorId uint64, selfSelects map[string][]uint64) (uint64, error) {
	if flowKey == "" {
		return 0, xerror.New(xerror.CodeParamInvalid, "flowKey 不能为空")
	}
	def, err := latestPublished(ctx, flowKey)
	if err != nil {
		return 0, err
	}
	var u *entity.SysUser
	if err = dao.SysUser.Ctx(ctx).
		Where("id", initiatorId).Where("status", 1).Where("deleted_at IS NULL").
		Scan(&u); err != nil {
		return 0, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if u == nil {
		return 0, xerror.New(xerror.CodeParamInvalid, "发起人不存在或已禁用")
	}
	inst, notify, err := startInstanceTx(ctx, def, title, formData, selfSelects,
		initiatorId, pickName(u.Nickname, u.Username), bizId)
	if err != nil {
		return 0, err
	}
	notify.flush(ctx)
	fireEvent(def.FlowKey, inst.Status, inst.Id, inst.BizId)
	return inst.Id, nil
}

// BizInstance 按 (flowKey, bizId) 查最新一条实例 (业务侧冗余流程状态用)。
func BizInstance(ctx context.Context, flowKey string, bizId uint64) (*entity.WfInstance, error) {
	if flowKey == "" || bizId == 0 {
		return nil, xerror.New(xerror.CodeParamInvalid, "flowKey/bizId 不能为空")
	}
	var m *entity.WfInstance
	if err := dao.WfInstance.Ctx(ctx).
		Where("flow_key", flowKey).Where("biz_id", bizId).Where("deleted_at IS NULL").
		Order("id DESC").Scan(&m); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if m == nil {
		return nil, xerror.New(xerror.CodeNotFound, "未找到关联的流程实例")
	}
	return m, nil
}

// latestPublished 取 flow_key 下最新已发布且未停用的定义 (单行模型下通常即该行本身)。
func latestPublished(ctx context.Context, flowKey string) (*entity.WfDefinition, error) {
	var d *entity.WfDefinition
	if err := dao.WfDefinition.Ctx(ctx).
		Where("flow_key", flowKey).
		Where("status", defStatusPublished).
		Where("version >= 1").
		Where("deleted_at IS NULL").
		Order("version DESC").Scan(&d); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if d == nil {
		return nil, xerror.New(xerror.CodeNotFound, "流程 "+flowKey+" 不存在或未发布")
	}
	return d, nil
}
