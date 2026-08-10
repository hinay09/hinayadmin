// Package system 系统管理-组织机构业务逻辑。
package system

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/xerror"
)

type sOrg struct{}

func NewOrg() *sOrg {
	return &sOrg{}
}

func init() {
	service.RegisterOrg(NewOrg())
}

// Tree 树形组织机构。
func (s *sOrg) Tree(ctx context.Context, _ *v1.OrgTreeReq) (res *v1.OrgTreeRes, err error) {
	var list []*model.SysOrg
	if err = dao.SysOrg.Ctx(ctx).
		Where("deleted_at IS NULL").
		Order("sort ASC, id ASC").Ctx(ctx).Scan(&list); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.OrgTreeRes{Tree: buildOrgTree(list, 0)}, nil
}

func buildOrgTree(list []*model.SysOrg, parentId uint64) []*model.OrgTree {
	out := make([]*model.OrgTree, 0)
	for _, m := range list {
		if m.ParentId == parentId {
			n := &model.OrgTree{SysOrg: *m}
			n.Children = buildOrgTree(list, m.Id)
			out = append(out, n)
		}
	}
	return out
}

// List 扁平列表。
func (s *sOrg) List(ctx context.Context, req *v1.OrgListReq) (res *v1.OrgListRes, err error) {
	q := dao.SysOrg.Ctx(ctx).Where("deleted_at IS NULL")
	if req.Keyword != "" {
		q = q.WhereLike("name", "%"+strings.TrimSpace(req.Keyword)+"%")
	}
	if req.Status != nil {
		q = q.Where("status", *req.Status)
	}
	var list []*model.SysOrg
	if err = q.Order("sort ASC, id ASC").Ctx(ctx).Scan(&list); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.OrgListRes{List: list}, nil
}

// Detail 详情。
func (s *sOrg) Detail(ctx context.Context, req *v1.OrgDetailReq) (res *v1.OrgDetailRes, err error) {
	var m *model.SysOrg
	if err = dao.SysOrg.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").Scan(&m); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if m == nil {
		return nil, xerror.New(xerror.CodeNotFound)
	}
	return &v1.OrgDetailRes{SysOrg: m}, nil
}

// Create 新增。
func (s *sOrg) Create(ctx context.Context, req *v1.OrgCreateReq) (res *v1.OrgCreateRes, err error) {
	if req.Status == 0 {
		req.Status = consts.StatusEnabled
	}
	id, err := dao.SysOrg.Ctx(ctx).Data(g.Map{
		"parent_id": req.ParentId,
		"name":      req.Name,
		"leader":    req.Leader,
		"phone":     req.Phone,
		"email":     req.Email,
		"sort":      req.Sort,
		"status":    req.Status,
		"remark":    req.Remark,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.OrgCreateRes{Id: uint64(id)}, nil
}

// Update 修改。
func (s *sOrg) Update(ctx context.Context, req *v1.OrgUpdateReq) (res *v1.OrgUpdateRes, err error) {
	if _, err = dao.SysOrg.Ctx(ctx).Where("id", req.Id).Data(g.Map{
		"parent_id": req.ParentId,
		"name":      req.Name,
		"leader":    req.Leader,
		"phone":     req.Phone,
		"email":     req.Email,
		"sort":      req.Sort,
		"status":    req.Status,
		"remark":    req.Remark,
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.OrgUpdateRes{}, nil
}

// Delete 软删除 (含子节点检查)。
func (s *sOrg) Delete(ctx context.Context, req *v1.OrgDeleteReq) (res *v1.OrgDeleteRes, err error) {
	cnt, _ := dao.SysOrg.Ctx(ctx).Where("parent_id", req.Id).Where("deleted_at IS NULL").Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeBusinessError, "存在子组织, 请先删除子组织")
	}
	if _, err = dao.SysOrg.Ctx(ctx).Data(g.Map{"deleted_at": gtime.Now()}).Where("id", req.Id).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.OrgDeleteRes{}, nil
}
