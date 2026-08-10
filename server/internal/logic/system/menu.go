// Package system 系统管理-菜单业务逻辑。
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

type sMenu struct{}

func NewMenu() *sMenu {
	return &sMenu{}
}

func init() {
	service.RegisterMenu(NewMenu())
}

// List 扁平列表。
func (s *sMenu) List(ctx context.Context, req *v1.MenuListReq) (res *v1.MenuListRes, err error) {
	q := dao.SysMenu.Ctx(ctx).Where("deleted_at IS NULL")
	if req.Keyword != "" {
		q = q.WhereLike("name", "%"+strings.TrimSpace(req.Keyword)+"%")
	}
	if req.Status != nil {
		q = q.Where("status", *req.Status)
	}
	var list []*model.SysMenu
	if err = q.Order("sort ASC, id ASC").Ctx(ctx).Scan(&list); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.MenuListRes{List: list}, nil
}

// Tree 树形菜单(全部启用菜单, 不按权限过滤, 用于后台维护)。
func (s *sMenu) Tree(ctx context.Context, _ *v1.MenuTreeReq) (res *v1.MenuTreeRes, err error) {
	var list []*model.SysMenu
	if err = dao.SysMenu.Ctx(ctx).
		Where("deleted_at IS NULL").
		Order("sort ASC, id ASC").Ctx(ctx).Scan(&list); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.MenuTreeRes{Tree: buildMenuTree(list, 0)}, nil
}

func buildMenuTree(list []*model.SysMenu, parentId uint64) []*model.MenuTree {
	out := make([]*model.MenuTree, 0)
	for _, m := range list {
		if m.ParentId == parentId {
			n := &model.MenuTree{SysMenu: *m}
			n.Children = buildMenuTree(list, m.Id)
			out = append(out, n)
		}
	}
	return out
}

// Detail 详情。
func (s *sMenu) Detail(ctx context.Context, req *v1.MenuDetailReq) (res *v1.MenuDetailRes, err error) {
	var m *model.SysMenu
	if err = dao.SysMenu.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").Scan(&m); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if m == nil {
		return nil, xerror.New(xerror.CodeNotFound)
	}
	return &v1.MenuDetailRes{SysMenu: m}, nil
}

// Create 新增。
func (s *sMenu) Create(ctx context.Context, req *v1.MenuCreateReq) (res *v1.MenuCreateRes, err error) {
	if req.Status == 0 {
		req.Status = consts.StatusEnabled
	}
	id, err := dao.SysMenu.Ctx(ctx).Data(g.Map{
		"parent_id": req.ParentId, "name": req.Name, "type": req.Type,
		"path": req.Path, "component": req.Component, "icon": req.Icon,
		"permission": req.Permission, "api_path": req.ApiPath, "api_method": req.ApiMethod,
		"sort": req.Sort, "visible": req.Visible, "status": req.Status,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.MenuCreateRes{Id: uint64(id)}, nil
}

// Update 修改。
func (s *sMenu) Update(ctx context.Context, req *v1.MenuUpdateReq) (res *v1.MenuUpdateRes, err error) {
	if _, err = dao.SysMenu.Ctx(ctx).Where("id", req.Id).Data(g.Map{
		"parent_id": req.ParentId, "name": req.Name, "type": req.Type,
		"path": req.Path, "component": req.Component, "icon": req.Icon,
		"permission": req.Permission, "api_path": req.ApiPath, "api_method": req.ApiMethod,
		"sort": req.Sort, "visible": req.Visible, "status": req.Status,
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.MenuUpdateRes{}, nil
}

// Delete 软删除 (含子节点检查)。
func (s *sMenu) Delete(ctx context.Context, req *v1.MenuDeleteReq) (res *v1.MenuDeleteRes, err error) {
	cnt, _ := dao.SysMenu.Ctx(ctx).Where("parent_id", req.Id).Where("deleted_at IS NULL").Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeBusinessError, "存在子菜单, 请先删除子菜单")
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(g.Map{"deleted_at": gtime.Now()}).Where("id", req.Id).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.MenuDeleteRes{}, nil
}
