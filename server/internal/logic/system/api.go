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
	"hinay.cn/admin/utility/response"
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
func (s *sApi) All(ctx context.Context, _ *v1.ApiAllReq) (res *v1.ApiAllRes, err error) {
	type row struct {
		Id          int64  `orm:"id"`
		Path        string `orm:"path"`
		Method      string `orm:"method"`
		GroupName   string `orm:"group_name"`
		Description string `orm:"description"`
	}
	var rows []row
	if err = dao.SysApi.Ctx(ctx).Where("deleted_at IS NULL").
		Order("group_name ASC, id ASC").Scan(&rows); err != nil {
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

// Create 创建 API。
func (s *sApi) Create(ctx context.Context, req *v1.ApiCreateReq) (res *v1.ApiCreateRes, err error) {
	if _, err = dao.SysApi.Ctx(ctx).Data(g.Map{
		"path": req.Path, "method": req.Method,
		"group_name": req.GroupName, "description": req.Description,
	}).Insert(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.ApiCreateRes{}, nil
}

// Update 更新 API。
func (s *sApi) Update(ctx context.Context, req *v1.ApiUpdateReq) (res *v1.ApiUpdateRes, err error) {
	if _, err = dao.SysApi.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").
		Data(g.Map{
			"path": req.Path, "method": req.Method,
			"group_name": req.GroupName, "description": req.Description,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.ApiUpdateRes{}, nil
}

// Delete 删除 API。
func (s *sApi) Delete(ctx context.Context, req *v1.ApiDeleteReq) (res *v1.ApiDeleteRes, err error) {
	if _, err = dao.SysApi.Ctx(ctx).Data(g.Map{"deleted_at": gtime.Now()}).Where("id", req.Id).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.ApiDeleteRes{}, nil
}
