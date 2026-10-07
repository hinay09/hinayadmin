// Package flow 自由审批流业务逻辑 — 流程定义管理。
package flow

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	v1 "hinay.cn/admin/api/flow/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

// defItem 实体转 API 条目。
func defItem(d *entity.WfDefinition) *v1.FlowDefinitionItem {
	return &v1.FlowDefinitionItem{
		Id:        d.Id,
		FlowKey:   d.FlowKey,
		Name:      d.Name,
		FormConf:  d.FormConf,
		FlowConf:  d.FlowConf,
		Version:   d.Version,
		Status:    d.Status,
		Remark:    d.Remark,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// DefinitionList 流程定义分页列表。
func (s *sFlow) DefinitionList(ctx context.Context, in *v1.FlowDefinitionListReq) (res *v1.FlowDefinitionListRes, err error) {
	q := dao.WfDefinition.Ctx(ctx).Where("deleted_at IS NULL")
	if in.Keyword != "" {
		kw := "%" + strings.TrimSpace(in.Keyword) + "%"
		// 括号分组, 防止 OR 破坏软删条件
		q = q.Where("(name LIKE ? OR flow_key LIKE ?)", kw, kw)
	}
	if in.Status != nil {
		q = q.Where("status", *in.Status)
	}
	if in.Version != nil {
		q = q.Where("version", *in.Version)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*entity.WfDefinition
	if err = q.Order("id DESC").Page(in.Page, in.PageSize).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list := make([]*v1.FlowDefinitionItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, defItem(r))
	}
	page := response.Page(list, int64(total), in.Page, in.PageSize)
	return (*v1.FlowDefinitionListRes)(&page), nil
}

// DefinitionUsable 可发起的流程: 每个 flow_key 取最新已发布且未停用版本。
func (s *sFlow) DefinitionUsable(ctx context.Context, in *v1.FlowDefinitionUsableReq) (res *v1.FlowDefinitionUsableRes, err error) {
	var rows []*entity.WfDefinition
	if err = dao.WfDefinition.Ctx(ctx).
		Where("status", defStatusPublished).
		Where("deleted_at IS NULL").
		Order("flow_key ASC, version DESC").
		Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	seen := map[string]bool{}
	list := make([]*v1.FlowDefinitionItem, 0)
	for _, r := range rows {
		if seen[r.FlowKey] {
			continue
		}
		seen[r.FlowKey] = true
		list = append(list, defItem(r))
	}
	return &v1.FlowDefinitionUsableRes{List: list}, nil
}

// DefinitionDetail 定义详情。
func (s *sFlow) DefinitionDetail(ctx context.Context, in *v1.FlowDefinitionDetailReq) (res *v1.FlowDefinitionDetailRes, err error) {
	d, err := loadDefinition(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	return &v1.FlowDefinitionDetailRes{FlowDefinitionItem: defItem(d)}, nil
}

// DefinitionCreate 新增流程定义 (草稿)。
func (s *sFlow) DefinitionCreate(ctx context.Context, in *v1.FlowDefinitionCreateReq) (res *v1.FlowDefinitionCreateRes, err error) {
	formConf, flowConf := in.FormConf, in.FlowConf
	if formConf == "" {
		formConf = defaultFormConf()
	}
	if flowConf == "" {
		flowConf = defaultFlowConf()
	}
	if err = validateConf(formConf, flowConf); err != nil {
		return nil, xerror.New(xerror.CodeParamInvalid, err.Error())
	}
	id, err := dao.WfDefinition.Ctx(ctx).Data(g.Map{
		"flow_key":  strings.TrimSpace(in.FlowKey),
		"name":      in.Name,
		"form_conf": formConf,
		"flow_conf": flowConf,
		"version":   0,
		"status":    defStatusDraft,
		"remark":    in.Remark,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.FlowDefinitionCreateRes{Id: uint64(id)}, nil
}

// DefinitionUpdate 修改流程定义 (仅草稿)。
func (s *sFlow) DefinitionUpdate(ctx context.Context, in *v1.FlowDefinitionUpdateReq) (res *v1.FlowDefinitionUpdateRes, err error) {
	// 任意状态可编辑: 发布态保存后对新发起即时生效, 在途实例走自身快照不受影响
	if _, err = loadDefinition(ctx, in.Id); err != nil {
		return nil, err
	}
	if err = validateConf(in.FormConf, in.FlowConf); err != nil {
		return nil, xerror.New(xerror.CodeParamInvalid, err.Error())
	}
	if _, err = dao.WfDefinition.Ctx(ctx).Where("id", in.Id).Data(g.Map{
		"flow_key":  strings.TrimSpace(in.FlowKey),
		"name":      in.Name,
		"form_conf": in.FormConf,
		"flow_conf": in.FlowConf,
		"remark":    in.Remark,
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.FlowDefinitionUpdateRes{}, nil
}

// DefinitionDelete 删除流程定义 (仅未发布过的草稿; 发布过的版本被实例引用, 不允许删除)。
func (s *sFlow) DefinitionDelete(ctx context.Context, in *v1.FlowDefinitionDeleteReq) (res *v1.FlowDefinitionDeleteRes, err error) {
	d, err := loadDefinition(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if d.Version > 0 {
		return nil, xerror.New(xerror.CodeParamInvalid, "仅未发布的草稿可删除; 该流程已发布过, 历史实例仍引用此定义")
	}
	if _, err = dao.WfDefinition.Ctx(ctx).Where("id", in.Id).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.FlowDefinitionDeleteRes{}, nil
}

// DefinitionPublish 发布: 原地生效 —— 同一行 version+1 且 status=1。
// 单行模型说明: 发布后可继续编辑 (保存对新发起即时生效); 在途实例走自身快照, 不受影响。
func (s *sFlow) DefinitionPublish(ctx context.Context, in *v1.FlowDefinitionPublishReq) (res *v1.FlowDefinitionPublishRes, err error) {
	d, err := loadDefinition(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if d.Status == defStatusPublished {
		return nil, xerror.New(xerror.CodeParamInvalid, "已处于发布状态 (编辑保存后对新发起即时生效, 无需重复发布)")
	}
	if err = validateConf(d.FormConf, d.FlowConf); err != nil {
		return nil, xerror.New(xerror.CodeParamInvalid, err.Error())
	}
	// flow_key 为空时按行 ID 生成稳定标识
	flowKey := strings.TrimSpace(d.FlowKey)
	if flowKey == "" {
		flowKey = "flow_" + u64str(d.Id)
	}
	nextVer := d.Version + 1
	if _, err = dao.WfDefinition.Ctx(ctx).Where("id", in.Id).Data(g.Map{
		"flow_key": flowKey,
		"status":   defStatusPublished,
		"version":  nextVer,
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.FlowDefinitionPublishRes{Id: in.Id, Version: nextVer}, nil
}

// DefinitionDisable 停用一个已发布版本 (不可再发起; 在途实例不受影响)。
func (s *sFlow) DefinitionDisable(ctx context.Context, in *v1.FlowDefinitionDisableReq) (res *v1.FlowDefinitionDisableRes, err error) {
	d, err := loadDefinition(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if d.Status != defStatusPublished {
		return nil, xerror.New(xerror.CodeParamInvalid, "仅已发布版本可停用")
	}
	if _, err = dao.WfDefinition.Ctx(ctx).Where("id", in.Id).Data(g.Map{
		"status": defStatusDisabled,
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.FlowDefinitionDisableRes{}, nil
}

// DesignerOptions 设计器选项: 启用用户 + 启用角色。
func (s *sFlow) DesignerOptions(ctx context.Context, in *v1.FlowDesignerOptionsReq) (res *v1.FlowDesignerOptionsRes, err error) {
	var users []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).
		Where("status", 1).Where("deleted_at IS NULL").
		Order("id ASC").Scan(&users); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var roles []*entity.SysRole
	if err = dao.SysRole.Ctx(ctx).
		Where("status", 1).Where("deleted_at IS NULL").
		Order("id ASC").Scan(&roles); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var posts []*entity.SysPost
	if err = dao.SysPost.Ctx(ctx).
		Where("status", 1).Where("deleted_at IS NULL").
		Order("sort ASC, id ASC").Scan(&posts); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	out := &v1.FlowDesignerOptionsRes{
		Users: make([]*v1.FlowUserOption, 0, len(users)),
		Roles: make([]*v1.FlowRoleOption, 0, len(roles)),
		Posts: make([]*v1.FlowPostOption, 0, len(posts)),
	}
	for _, u := range users {
		out.Users = append(out.Users, &v1.FlowUserOption{Id: u.Id, Nickname: u.Nickname, Username: u.Username})
	}
	for _, r := range roles {
		out.Roles = append(out.Roles, &v1.FlowRoleOption{Id: r.Id, Name: r.Name, Code: r.Code})
	}
	for _, p := range posts {
		out.Posts = append(out.Posts, &v1.FlowPostOption{Id: p.Id, Code: p.PostCode, Name: p.PostName, Kind: p.PostKind})
	}
	return out, nil
}

// loadDefinition 按 ID 取未删除定义, 不存在报 404。
func loadDefinition(ctx context.Context, id uint64) (*entity.WfDefinition, error) {
	var d *entity.WfDefinition
	if err := dao.WfDefinition.Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Scan(&d); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if d == nil {
		return nil, xerror.New(xerror.CodeNotFound, "流程定义不存在")
	}
	return d, nil
}
