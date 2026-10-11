// Package system 系统管理-API资源业务逻辑。
package system

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/tenantx"
	"hinay.cn/admin/utility/xerror"
)

type sApi struct{}

func NewApi() *sApi {
	return &sApi{}
}

func init() {
	service.RegisterApi(NewApi())
}

// List 获取 API 列表（支持分组筛选、分页）。
func (s *sApi) List(ctx context.Context, req *v1.ApiListReq) (res *v1.ApiListRes, err error) {
	q := dao.SysApi.Ctx(ctx).Where("deleted_at IS NULL")
	if req.GroupName != "" {
		q = q.Where("group_name LIKE ?", "%"+strings.TrimSpace(req.GroupName)+"%")
	}
	if req.Path != "" {
		q = q.Where("path LIKE ?", "%"+strings.TrimSpace(req.Path)+"%")
	}

	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	type row struct {
		Id          int64  `orm:"id"`
		Path        string `orm:"path"`
		Method      string `orm:"method"`
		GroupName   string `orm:"group_name"`
		Description string `orm:"description"`
	}
	var rows []row
	if err = q.Page(req.Page, req.PageSize).Order("group_name ASC, id ASC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list := make([]v1.ApiItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, v1.ApiItem{
			Id:          r.Id,
			Path:        r.Path,
			Method:      r.Method,
			GroupName:   r.GroupName,
			Description: r.Description,
		})
	}
	pageRes := response.Page(list, int64(total), req.Page, req.PageSize)
	res = (*v1.ApiListRes)(&pageRes)
	return
}

// All 全量 API（不分页，用于权限分配等场景）。
// 多租户 (§7.1 可绑列表过滤): 非平台租户用户仅返回归属菜单 ∈ 授权集闭包的
// API (menu_id=0 的平台/公共接口不可见 —— 会话类接口本就免 Casbin 无需绑定)。
func (s *sApi) All(ctx context.Context, _ *v1.ApiAllReq) (res *v1.ApiAllRes, err error) {
	type row struct {
		Id          int64  `orm:"id"`
		Path        string `orm:"path"`
		Method      string `orm:"method"`
		GroupName   string `orm:"group_name"`
		Description string `orm:"description"`
		MenuId      uint64 `orm:"menu_id"`
	}
	q := dao.SysApi.Ctx(ctx).Where("deleted_at IS NULL")
	var closure map[uint64]struct{}
	if tenantx.Enabled(ctx) {
		var cerr error
		closure, cerr = service.Tenant().EffectiveMenuClosure(ctx, contextx.TenantId(ctx))
		if cerr == nil && closure != nil {
			ids := make([]uint64, 0, len(closure))
			for id := range closure {
				ids = append(ids, id)
			}
			q = q.WhereIn("menu_id", ids)
		}
	}
	var rows []row
	if err = q.Order("group_name ASC, id ASC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list := make([]v1.ApiItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, v1.ApiItem{
			Id:          r.Id,
			Path:        r.Path,
			Method:      r.Method,
			GroupName:   r.GroupName,
			Description: r.Description,
		})
	}
	return &v1.ApiAllRes{List: list}, nil
}

// Create 创建 API (D4: API 目录为平台统一发布, 仅平台超管)。
func (s *sApi) Create(ctx context.Context, req *v1.ApiCreateReq) (res *v1.ApiCreateRes, err error) {
	if verr := guardPlatformCatalog(ctx); verr != nil {
		return nil, verr
	}
	if _, err = dao.SysApi.Ctx(ctx).Data(g.Map{
		"path": req.Path, "method": req.Method,
		"group_name": req.GroupName, "description": req.Description,
	}).Insert(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.ApiCreateRes{}, nil
}

// Update 更新 API (D4: API 目录为平台统一发布, 仅平台超管)。
func (s *sApi) Update(ctx context.Context, req *v1.ApiUpdateReq) (res *v1.ApiUpdateRes, err error) {
	if verr := guardPlatformCatalog(ctx); verr != nil {
		return nil, verr
	}
	if _, err = dao.SysApi.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").
		Data(g.Map{
			"path": req.Path, "method": req.Method,
			"group_name": req.GroupName, "description": req.Description,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.ApiUpdateRes{}, nil
}

// Delete 删除 API (D4: API 目录为平台统一发布, 仅平台超管)。
func (s *sApi) Delete(ctx context.Context, req *v1.ApiDeleteReq) (res *v1.ApiDeleteRes, err error) {
	if verr := guardPlatformCatalog(ctx); verr != nil {
		return nil, verr
	}
	if _, err = dao.SysApi.Ctx(ctx).Data(g.Map{"deleted_at": gtime.Now()}).Where("id", req.Id).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.ApiDeleteRes{}, nil
}
