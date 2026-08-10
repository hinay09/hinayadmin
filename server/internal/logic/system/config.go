// Package system 系统管理-全局配置业务逻辑。
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

type sConfig struct{}

func init() {
	service.RegisterConfig(NewConfig())
}

func NewConfig() *sConfig {
	return &sConfig{}
}

// List 分页列表。
func (s *sConfig) List(ctx context.Context, req *v1.ConfigListReq) (res *v1.ConfigListRes, err error) {
	q := dao.SysConfig.Ctx(ctx).Where("deleted_at IS NULL")

	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("config_key LIKE ? OR name LIKE ?", kw, kw)
	}
	if req.ConfigType != nil {
		q = q.Where("config_type", *req.ConfigType)
	}

	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	var rows []*model.ConfigItem
	if err = q.Page(req.Page, req.PageSize).Order("sort ASC, id ASC").Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	page := response.Page(rows, int64(total), req.Page, req.PageSize)
	r := v1.ConfigListRes(page)
	return &r, nil
}

// All 获取所有启用的全局配置。
func (s *sConfig) All(ctx context.Context, _ *v1.ConfigAllReq) (res *v1.ConfigAllRes, err error) {
	var rows []*model.ConfigItem
	if err = dao.SysConfig.Ctx(ctx).
		Where("status", 1).
		Where("deleted_at IS NULL").
		Order("sort ASC, id ASC").
		Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// 转为 key-value map 方便前端直接使用
	list := make(g.Map, len(rows))
	for _, item := range rows {
		list[item.ConfigKey] = g.Map{
			"value": item.ConfigValue,
			"type":  item.ConfigType,
			"name":  item.Name,
		}
	}
	return &v1.ConfigAllRes{List: list}, nil
}

// Create 新增全局配置。
func (s *sConfig) Create(ctx context.Context, req *v1.ConfigCreateReq) (res *v1.ConfigCreateRes, err error) {
	// 检查 config_key 唯一性
	cnt, _ := dao.SysConfig.Ctx(ctx).
		Where("config_key", req.ConfigKey).
		Where("deleted_at IS NULL").
		Ctx(ctx).Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeBusinessError, "配置键已存在")
	}

	id, err := dao.SysConfig.Ctx(ctx).Data(g.Map{
		"config_key":   req.ConfigKey,
		"config_value": req.ConfigValue,
		"config_type":  req.ConfigType,
		"name":         req.Name,
		"remark":       req.Remark,
		"status":       req.Status,
		"sort":         req.Sort,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.ConfigCreateRes{Id: uint64(id)}, nil
}

// Update 修改全局配置。
func (s *sConfig) Update(ctx context.Context, req *v1.ConfigUpdateReq) (res *v1.ConfigUpdateRes, err error) {
	// 检查 config_key 唯一性（排除自身）
	cnt, _ := dao.SysConfig.Ctx(ctx).
		Where("config_key", req.ConfigKey).
		Where("id != ?", req.Id).
		Where("deleted_at IS NULL").
		Ctx(ctx).Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeBusinessError, "配置键已存在")
	}

	if _, err = dao.SysConfig.Ctx(ctx).
		Where("id", req.Id).Where("deleted_at IS NULL").
		Data(g.Map{
			"config_key":   req.ConfigKey,
			"config_value": req.ConfigValue,
			"config_type":  req.ConfigType,
			"name":         req.Name,
			"remark":       req.Remark,
			"status":       req.Status,
			"sort":         req.Sort,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.ConfigUpdateRes{}, nil
}

// Delete 删除全局配置（软删除）。
func (s *sConfig) Delete(ctx context.Context, req *v1.ConfigDeleteReq) (res *v1.ConfigDeleteRes, err error) {
	if _, err = dao.SysConfig.Ctx(ctx).
		Where("id", req.Id).
		Data(g.Map{"deleted_at": gtime.Now()}).
		Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.ConfigDeleteRes{}, nil
}
