// Package system 系统管理-角色业务逻辑。
package system

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sRole struct{}

func NewRole() *sRole {
	return &sRole{}
}

func init() {
	service.RegisterRole(NewRole())
}

// List 分页列表。
func (s *sRole) List(ctx context.Context, req *v1.RoleListReq) (res *v1.RoleListRes, err error) {
	q := dao.SysRole.Ctx(ctx).Where("deleted_at IS NULL")
	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.WhereOr("name LIKE ?", kw).WhereOr("code LIKE ?", kw)
	}
	if req.Status != nil {
		q = q.Where("status", *req.Status)
	}
	total, err := q.Ctx(ctx).Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var list []*model.SysRole
	if err = q.Ctx(ctx).Page(req.Page, req.PageSize).Order("sort ASC, id DESC").Scan(&list); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	page := response.Page(list, int64(total), req.Page, req.PageSize)
	r := v1.RoleListRes(page)
	return &r, nil
}

// All 全量。
func (s *sRole) All(ctx context.Context, _ *v1.RoleAllReq) (res *v1.RoleAllRes, err error) {
	var list []*model.SysRole
	if err = dao.SysRole.Ctx(ctx).
		Where("deleted_at IS NULL").Where("status", consts.StatusEnabled).
		Order("sort ASC, id ASC").Ctx(ctx).Scan(&list); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.RoleAllRes{List: list}, nil
}

// Detail 详情 + 已绑定菜单 ID 列表（从 Casbin 获取）。
func (s *sRole) Detail(ctx context.Context, req *v1.RoleDetailReq) (res *v1.RoleDetailRes, err error) {
	r, ids, err := s.detailInternal(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.RoleDetailRes{Role: r, MenuIds: ids}, nil
}

// detailInternal 内部 helper：返回角色实体与菜单 ID 列表。
func (s *sRole) detailInternal(ctx context.Context, id uint64) (*model.SysRole, []uint64, error) {
	var r *model.SysRole
	if err := dao.SysRole.Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Scan(&r); err != nil {
		return nil, nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if r == nil {
		return nil, nil, xerror.New(xerror.CodeNotFound)
	}
	menuIds := make([]uint64, 0)
	mids, _ := casbinx.GetRoleMenus(ctx, r.Code)
	for _, mid := range mids {
		menuIds = append(menuIds, uint64(mid))
	}
	return r, menuIds, nil
}

// Create 新增。
func (s *sRole) Create(ctx context.Context, req *v1.RoleCreateReq) (res *v1.RoleCreateRes, err error) {
	cnt, _ := dao.SysRole.Ctx(ctx).Where("code", req.Code).Where("deleted_at IS NULL").Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeRoleCodeExists)
	}
	if req.Status == 0 {
		req.Status = consts.StatusEnabled
	}
	id, err := dao.SysRole.Ctx(ctx).Data(g.Map{
		"name": req.Name, "code": req.Code, "sort": req.Sort,
		"status": req.Status, "remark": req.Remark,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.RoleCreateRes{Id: uint64(id)}, nil
}

// Update 修改。
func (s *sRole) Update(ctx context.Context, req *v1.RoleUpdateReq) (res *v1.RoleUpdateRes, err error) {
	if _, err = dao.SysRole.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").
		Data(g.Map{
			"name": req.Name, "sort": req.Sort,
			"status": req.Status, "remark": req.Remark,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.RoleUpdateRes{}, nil
}

// Delete 删除 (软删 + 清理 Casbin 策略)。
func (s *sRole) Delete(ctx context.Context, req *v1.RoleDeleteReq) (res *v1.RoleDeleteRes, err error) {
	if req.Id == 1 {
		return nil, xerror.New(xerror.CodeBusinessError, "内置超级管理员角色不可删除")
	}
	// 先查角色 code
	var code string
	if err = dao.SysRole.Ctx(ctx).Where("id", req.Id).Fields("code").Scan(&code); err != nil || code == "" {
		return nil, xerror.New(xerror.CodeNotFound, "角色不存在")
	}
	if _, err = dao.SysRole.Ctx(ctx).Data(g.Map{"deleted_at": gtime.Now()}).Where("id", req.Id).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 清理 Casbin 中该角色的所有策略（p 策略 + g 策略中引用该角色的）
	_ = casbinx.RemoveRolePolicies(ctx, code)
	return &v1.RoleDeleteRes{}, nil
}

// AssignMenus 角色绑定菜单, 通过 Casbin 策略管理。
func (s *sRole) AssignMenus(ctx context.Context, req *v1.RoleAssignMenusReq) (res *v1.RoleAssignMenusRes, err error) {
	var role *model.SysRole
	if err = dao.SysRole.Ctx(ctx).Where("id", req.Id).Scan(&role); err != nil || role == nil {
		return nil, xerror.New(xerror.CodeNotFound, "角色不存在")
	}
	// 转换 menuIds 为 int64 切片
	ids := make([]int64, 0, len(req.MenuIds))
	for _, mid := range req.MenuIds {
		if mid == 0 {
			continue
		}
		ids = append(ids, int64(mid))
	}
	if err = casbinx.SetRoleMenus(ctx, role.Code, ids); err != nil {
		return nil, err
	}
	return &v1.RoleAssignMenusRes{}, nil
}

// GetMenus 获取角色已绑定的菜单 ID 列表（从 Casbin 获取）。
func (s *sRole) GetMenus(ctx context.Context, req *v1.RoleGetMenusReq) (res *v1.RoleGetMenusRes, err error) {
	_, ids, err := s.detailInternal(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.RoleGetMenusRes{MenuIds: ids}, nil
}

// AssignApis 角色分配 API 权限。
func (s *sRole) AssignApis(ctx context.Context, req *v1.RoleAssignApisReq) (res *v1.RoleAssignApisRes, err error) {
	var role *model.SysRole
	if err = dao.SysRole.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").Scan(&role); err != nil || role == nil {
		return nil, xerror.New(xerror.CodeNotFound, "角色不存在")
	}
	apis := make([]casbinx.ApiPolicy, 0, len(req.Apis))
	for _, a := range req.Apis {
		apis = append(apis, casbinx.ApiPolicy{Path: a.Path, Method: a.Method})
	}
	if err = casbinx.SetRoleApis(ctx, role.Code, apis); err != nil {
		return nil, err
	}
	return &v1.RoleAssignApisRes{}, nil
}

// GetApis 获取角色已分配的 API 权限。
func (s *sRole) GetApis(ctx context.Context, req *v1.RoleGetApisReq) (res *v1.RoleGetApisRes, err error) {
	var role *model.SysRole
	if err = dao.SysRole.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").Scan(&role); err != nil || role == nil {
		return nil, xerror.New(xerror.CodeNotFound, "角色不存在")
	}
	apis, err := casbinx.GetRoleApis(ctx, role.Code)
	if err != nil {
		return nil, err
	}
	list := make([]v1.RoleApiItem, 0, len(apis))
	for _, a := range apis {
		list = append(list, v1.RoleApiItem{Path: a.Path, Method: a.Method})
	}
	return &v1.RoleGetApisRes{List: list}, nil
}
