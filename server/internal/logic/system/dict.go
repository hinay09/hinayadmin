// Package system 系统管理-字典数据项业务逻辑。
package system

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sDict struct{}

func init() {
	service.RegisterDict(NewDict())
}

func NewDict() *sDict {
	return &sDict{}
}

// ListByType 按类型获取字典数据项分页列表。
func (s *sDict) ListByType(ctx context.Context, req *v1.DictDataListReq) (res *v1.DictDataListRes, err error) {
	q := dao.SysDictData.Ctx(ctx).
		Where("type_id", req.TypeId).
		Where("deleted_at IS NULL")

	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("dict_label LIKE ? OR dict_value LIKE ?", kw, kw)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	var rows []*model.DictDataItem
	if err = q.Page(req.Page, req.PageSize).Order("sort ASC, id ASC").Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	page := response.Page(rows, int64(total), req.Page, req.PageSize)
	r := v1.DictDataListRes(page)
	return &r, nil
}

// Create 新增字典数据项。
func (s *sDict) Create(ctx context.Context, req *v1.DictDataCreateReq) (res *v1.DictDataCreateRes, err error) {
	id, err := dao.SysDictData.Ctx(ctx).Data(g.Map{
		"type_id":    req.TypeId,
		"dict_label": req.DictLabel,
		"dict_value": req.DictValue,
		"sort":       req.Sort,
		"status":     req.Status,
		"remark":     req.Remark,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.DictDataCreateRes{Id: uint64(id)}, nil
}

// Update 修改字典数据项。
func (s *sDict) Update(ctx context.Context, req *v1.DictDataUpdateReq) (res *v1.DictDataUpdateRes, err error) {
	if _, err = dao.SysDictData.Ctx(ctx).
		Where("id", req.Id).Where("type_id", req.TypeId).Where("deleted_at IS NULL").
		Data(g.Map{
			"dict_label": req.DictLabel,
			"dict_value": req.DictValue,
			"sort":       req.Sort,
			"status":     req.Status,
			"remark":     req.Remark,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.DictDataUpdateRes{}, nil
}

// Delete 删除字典数据项。
func (s *sDict) Delete(ctx context.Context, req *v1.DictDataDeleteReq) (res *v1.DictDataDeleteRes, err error) {
	if _, err = dao.SysDictData.Ctx(ctx).
		Where("id", req.Id).Where("type_id", req.TypeId).
		Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.DictDataDeleteRes{}, nil
}

// Sort 批量排序字典数据项。
func (s *sDict) Sort(ctx context.Context, req *v1.DictDataSortReq) (res *v1.DictDataSortRes, err error) {
	if len(req.Items) == 0 {
		return &v1.DictDataSortRes{}, nil
	}
	err = dao.SysDictData.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		for _, item := range req.Items {
			if _, ie := tx.Model("sys_dict_data").Ctx(ctx).
				Where("id", item.Id).
				Where("type_id", req.TypeId).
				Data(g.Map{"sort": item.Sort}).
				Update(); ie != nil {
				return ie
			}
		}
		return nil
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.DictDataSortRes{}, nil
}

// All 获取所有启用的字典数据（按类型编码分组）。
func (s *sDict) All(ctx context.Context, _ *v1.DictAllReq) (res *v1.DictAllRes, err error) {
	// 查询所有启用的字典类型
	var types []*model.SysDictType
	if err = dao.SysDictType.Ctx(ctx).
		Where("status", 1).
		Where("deleted_at IS NULL").
		Ctx(ctx).Scan(&types); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if len(types) == 0 {
		return &v1.DictAllRes{List: g.Map{}}, nil
	}

	// 收集所有 type_id
	typeIds := make([]uint64, 0, len(types))
	typeMap := make(map[uint64]string, len(types))
	for _, t := range types {
		typeIds = append(typeIds, t.Id)
		typeMap[t.Id] = t.TypeCode
	}

	// 查询所有启用的字典数据项
	var items []*model.DictDataItem
	if err = dao.SysDictData.Ctx(ctx).
		WhereIn("type_id", typeIds).
		Where("status", 1).
		Where("deleted_at IS NULL").
		Order("type_id ASC, sort ASC, id ASC").
		Ctx(ctx).Scan(&items); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// 按 typeCode 分组
	grouped := make(g.Map)
	for _, item := range items {
		code := typeMap[item.TypeId]
		if code == "" {
			continue
		}
		if _, ok := grouped[code]; !ok {
			grouped[code] = make([]any, 0)
		}
		grouped[code] = append(grouped[code].([]any), item)
	}

	return &v1.DictAllRes{List: grouped}, nil
}
