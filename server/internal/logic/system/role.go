// Package system 系统管理-角色业务逻辑。
package system

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
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
		// 括号分组: 避免关键词 OR 条件逃逸出 deleted_at 过滤 (WhereOr 顶层分组陷阱)
		q = q.Where("(name LIKE ? OR code LIKE ?)", kw, kw)
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

// Detail 详情 + 已绑定菜单 ID 列表（从 Casbin 获取）+ 自定义数据范围组织 ID 列表。
func (s *sRole) Detail(ctx context.Context, req *v1.RoleDetailReq) (res *v1.RoleDetailRes, err error) {
	r, ids, err := s.detailInternal(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	orgIds := make([]uint64, 0)
	if r.DataScope == consts.DataScopeCustom {
		mappings, merr := dao.SysRoleOrg.Ctx(ctx).Fields("org_id").Where("role_id", r.Id).All()
		if merr != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, merr)
		}
		for _, m := range mappings {
			orgIds = append(orgIds, m["org_id"].Uint64())
		}
	}
	return &v1.RoleDetailRes{Role: r, MenuIds: ids, OrgIds: orgIds}, nil
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
	mids, _ := casbinx.GetRoleMenus(ctx, r.Id)
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
	if req.DataScope == 0 {
		req.DataScope = consts.DataScopeAll
	}
	id, err := dao.SysRole.Ctx(ctx).Data(g.Map{
		"name": req.Name, "code": req.Code, "sort": req.Sort,
		"status": req.Status, "remark": req.Remark, "data_scope": req.DataScope,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if err = s.replaceRoleOrgs(ctx, uint64(id), req.DataScope, req.OrgIds); err != nil {
		return nil, err
	}
	return &v1.RoleCreateRes{Id: uint64(id)}, nil
}

// Update 修改。
func (s *sRole) Update(ctx context.Context, req *v1.RoleUpdateReq) (res *v1.RoleUpdateRes, err error) {
	if req.DataScope == 0 {
		req.DataScope = consts.DataScopeAll
	}
	if _, err = dao.SysRole.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").
		Data(g.Map{
			"name": req.Name, "sort": req.Sort,
			"status": req.Status, "remark": req.Remark, "data_scope": req.DataScope,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if err = s.replaceRoleOrgs(ctx, req.Id, req.DataScope, req.OrgIds); err != nil {
		return nil, err
	}
	return &v1.RoleUpdateRes{}, nil
}

// replaceRoleOrgs 重写角色的自定义数据范围组织绑定 (dataScope=自定义 时生效)。
func (s *sRole) replaceRoleOrgs(ctx context.Context, roleId uint64, dataScope int, orgIds []uint64) error {
	if dataScope != consts.DataScopeCustom {
		// 非自定义范围: 清掉残留绑定
		if _, err := dao.SysRoleOrg.Ctx(ctx).Where("role_id", roleId).Delete(); err != nil {
			return xerror.Wrap(xerror.CodeBusinessError, err)
		}
		return nil
	}
	err := dao.SysRoleOrg.Ctx(ctx).Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := dao.SysRoleOrg.Ctx(ctx).Where("role_id", roleId).Delete(); err != nil {
			return err
		}
		if len(orgIds) == 0 {
			return nil
		}
		rows := make([]g.Map, 0, len(orgIds))
		seen := make(map[uint64]struct{}, len(orgIds))
		for _, oid := range orgIds {
			if oid == 0 {
				continue
			}
			if _, dup := seen[oid]; dup {
				continue
			}
			seen[oid] = struct{}{}
			rows = append(rows, g.Map{"role_id": roleId, "org_id": oid})
		}
		if len(rows) == 0 {
			return nil
		}
		if _, err := dao.SysRoleOrg.Ctx(ctx).Data(rows).Insert(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return nil
}

// Delete 删除 (软删 + 清理 Casbin 策略)。
func (s *sRole) Delete(ctx context.Context, req *v1.RoleDeleteReq) (res *v1.RoleDeleteRes, err error) {
	if req.Id == 1 {
		return nil, xerror.New(xerror.CodeBusinessError, "内置超级管理员角色不可删除")
	}
	cnt, cerr := dao.SysRole.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").Count()
	if cerr != nil || cnt == 0 {
		return nil, xerror.New(xerror.CodeNotFound, "角色不存在")
	}
	if _, err = dao.SysRole.Ctx(ctx).Data(g.Map{"deleted_at": gtime.Now()}).Where("id", req.Id).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 清理 Casbin 中该角色的所有策略（p 策略 + g 策略中引用该角色的, 均按角色ID）
	_ = casbinx.RemoveRolePolicies(ctx, req.Id)
	return &v1.RoleDeleteRes{}, nil
}

// isBuiltInAdmin 内置超级管理员角色判定: 该角色在校验层始终全量放行, 权限配置不生效。
func isBuiltInAdmin(id uint64) bool {
	return id == consts.RoleAdminId
}

// AssignMenus 角色绑定菜单, 通过 Casbin 策略管理。
func (s *sRole) AssignMenus(ctx context.Context, req *v1.RoleAssignMenusReq) (res *v1.RoleAssignMenusRes, err error) {
	var role *model.SysRole
	if err = dao.SysRole.Ctx(ctx).Where("id", req.Id).Scan(&role); err != nil || role == nil {
		return nil, xerror.New(xerror.CodeNotFound, "角色不存在")
	}
	if isBuiltInAdmin(role.Id) {
		return nil, xerror.New(xerror.CodeBusinessError, "内置超级管理员默认拥有全部权限, 无需也无法配置")
	}
	// 转换 menuIds 为 int64 切片
	ids := make([]int64, 0, len(req.MenuIds))
	for _, mid := range req.MenuIds {
		if mid == 0 {
			continue
		}
		ids = append(ids, int64(mid))
	}
	if err = casbinx.SetRoleMenus(ctx, role.Id, ids); err != nil {
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

// validApiMethods 允许配置的 HTTP 方法枚举。
var validApiMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "*": true,
}

// validateApiPolicy 校验单条 API 策略, 防止意外写入超宽通配或非 API 路径导致越权。
func validateApiPolicy(path, method string) error {
	path = strings.TrimSpace(path)
	method = strings.ToUpper(strings.TrimSpace(method))
	if !strings.HasPrefix(path, "/api/") {
		return xerror.New(xerror.CodeBusinessError, "API 路径必须以 /api/ 开头: "+path)
	}
	if strings.ContainsAny(path, " \t") || strings.Contains(path, "..") {
		return xerror.New(xerror.CodeBusinessError, "API 路径包含非法字符: "+path)
	}
	// 通配符只允许结尾的 "/*", 避免中间通配意外放大授权面
	if strings.Contains(path, "*") && !strings.HasSuffix(path, "/*") {
		return xerror.New(xerror.CodeBusinessError, `API 路径通配符只支持结尾 "/*": `+path)
	}
	if !validApiMethods[method] {
		return xerror.New(xerror.CodeBusinessError, "非法的 HTTP 方法: "+method)
	}
	return nil
}

// AssignApis 角色分配 API 权限。
func (s *sRole) AssignApis(ctx context.Context, req *v1.RoleAssignApisReq) (res *v1.RoleAssignApisRes, err error) {
	var role *model.SysRole
	if err = dao.SysRole.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").Scan(&role); err != nil || role == nil {
		return nil, xerror.New(xerror.CodeNotFound, "角色不存在")
	}
	if isBuiltInAdmin(role.Id) {
		return nil, xerror.New(xerror.CodeBusinessError, "内置超级管理员默认拥有全部权限, 无需也无法配置")
	}
	apis := make([]casbinx.ApiPolicy, 0, len(req.Apis))
	for _, a := range req.Apis {
		if verr := validateApiPolicy(a.Path, a.Method); verr != nil {
			return nil, verr
		}
		apis = append(apis, casbinx.ApiPolicy{
			Path:   strings.TrimSpace(a.Path),
			Method: strings.ToUpper(strings.TrimSpace(a.Method)),
		})
	}
	if err = casbinx.SetRoleApis(ctx, role.Id, apis); err != nil {
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
	apis, err := casbinx.GetRoleApis(ctx, role.Id)
	if err != nil {
		return nil, err
	}
	list := make([]v1.RoleApiItem, 0, len(apis))
	for _, a := range apis {
		list = append(list, v1.RoleApiItem{Path: a.Path, Method: a.Method})
	}
	return &v1.RoleGetApisRes{List: list}, nil
}
