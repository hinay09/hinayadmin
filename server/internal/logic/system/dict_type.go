// Package system 系统管理-字典类型业务逻辑。
package system

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sDictType struct{}

func init() {
	service.RegisterDictType(NewDictType())
}

func NewDictType() *sDictType {
	return &sDictType{}
}

// List 字典类型分页列表。
func (s *sDictType) List(ctx context.Context, req *v1.DictTypeListReq) (res *v1.DictTypeListRes, err error) {
	q := dao.SysDictType.Ctx(ctx).Where("deleted_at IS NULL")

	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("type_code LIKE ? OR type_name LIKE ?", kw, kw)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	var rows []*model.DictTypeItem
	if err = q.Page(req.Page, req.PageSize).Order("id ASC").Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	page := response.Page(rows, int64(total), req.Page, req.PageSize)
	r := v1.DictTypeListRes(page)
	return &r, nil
}

// All 获取所有启用的字典类型。
func (s *sDictType) All(ctx context.Context, _ *v1.DictTypeAllReq) (res *v1.DictTypeAllRes, err error) {
	var rows []*model.DictTypeItem
	if err = dao.SysDictType.Ctx(ctx).
		Where("status", 1).
		Where("deleted_at IS NULL").
		Order("type_code ASC").
		Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.DictTypeAllRes{List: rows}, nil
}

// Create 新增字典类型。
func (s *sDictType) Create(ctx context.Context, req *v1.DictTypeCreateReq) (res *v1.DictTypeCreateRes, err error) {
	// 检查 type_code 唯一性
	cnt, _ := dao.SysDictType.Ctx(ctx).
		Where("type_code", req.TypeCode).
		Where("deleted_at IS NULL").
		Ctx(ctx).Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeBusinessError, "类型编码已存在")
	}
	id, err := dao.SysDictType.Ctx(ctx).Data(g.Map{
		"type_code": req.TypeCode,
		"type_name": req.TypeName,
		"status":    req.Status,
		"remark":    req.Remark,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.DictTypeCreateRes{Id: uint64(id)}, nil
}

// Update 修改字典类型。
func (s *sDictType) Update(ctx context.Context, req *v1.DictTypeUpdateReq) (res *v1.DictTypeUpdateRes, err error) {
	// 检查 type_code 唯一性（排除自身）
	cnt, _ := dao.SysDictType.Ctx(ctx).
		Where("type_code", req.TypeCode).
		Where("id != ?", req.Id).
		Where("deleted_at IS NULL").
		Ctx(ctx).Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeBusinessError, "类型编码已存在")
	}
	if _, err = dao.SysDictType.Ctx(ctx).
		Where("id", req.Id).Where("deleted_at IS NULL").
		Data(g.Map{
			"type_code": req.TypeCode,
			"type_name": req.TypeName,
			"status":    req.Status,
			"remark":    req.Remark,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.DictTypeUpdateRes{}, nil
}

// Delete 删除字典类型（有子项则拒绝删除）。
func (s *sDictType) Delete(ctx context.Context, req *v1.DictTypeDeleteReq) (res *v1.DictTypeDeleteRes, err error) {
	// 检查是否有子项
	cnt, _ := dao.SysDictData.Ctx(ctx).
		Where("type_id", req.Id).
		Where("deleted_at IS NULL").
		Ctx(ctx).Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeBusinessError, "该类型下存在字典数据项，请先删除子项")
	}
	if _, err = dao.SysDictType.Ctx(ctx).
		Where("id", req.Id).
		Data(g.Map{"deleted_at": gtime.Now()}).
		Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.DictTypeDeleteRes{}, nil
}
