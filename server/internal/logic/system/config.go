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

// sensitiveWords 敏感配置叶子名词条, 命中即视为机密值, 出参脱敏。
var sensitiveWords = map[string]struct{}{
	"secret": {}, "key": {}, "token": {}, "password": {}, "passwd": {}, "pwd": {},
}

// isSensitiveConfigKey 判断配置键是否属于敏感项 (如 ai.api_key / wechat.app_secret /
// wechat.token)。只看最后一段叶子名 (按 . 切) 再按 _/- 切词, 避免误伤
// sys.password.min_length 这类"含 password 字样但值是策略数值"的键。
func isSensitiveConfigKey(key string) bool {
	leaf := key
	if i := strings.LastIndex(key, "."); i >= 0 {
		leaf = key[i+1:]
	}
	for _, w := range strings.FieldsFunc(strings.ToLower(leaf), func(r rune) bool {
		return r == '_' || r == '-'
	}) {
		if _, ok := sensitiveWords[w]; ok {
			return true
		}
	}
	return false
}

// maskConfigValue 机密值脱敏: 保留前3后4字符, 过短全遮; 空值原样返回(表示未设置)。
func maskConfigValue(v string) string {
	if v == "" {
		return ""
	}
	r := []rune(v)
	if len(r) <= 8 {
		return "***"
	}
	return string(r[:3]) + "***" + string(r[len(r)-4:])
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

	// 敏感项 (api_key/app_secret 等) 脱敏后返回; 库中仍存明文供后端内部使用。
	for _, item := range rows {
		if isSensitiveConfigKey(item.ConfigKey) {
			item.ConfigValue = maskConfigValue(item.ConfigValue)
		}
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
		v := item.ConfigValue
		if isSensitiveConfigKey(item.ConfigKey) {
			v = maskConfigValue(v)
		}
		list[item.ConfigKey] = g.Map{
			"value": v,
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

	data := g.Map{
		"config_key":   req.ConfigKey,
		"config_value": req.ConfigValue,
		"config_type":  req.ConfigType,
		"name":         req.Name,
		"remark":       req.Remark,
		"status":       req.Status,
		"sort":         req.Sort,
	}

	// 脱敏回传保护: 编辑表单拿到的是掩码值, 未改动时原样提交;
	// 提交值与库中真实值的掩码一致则跳过 config_value, 避免掩码覆盖真实值。
	// (含键改名场景, 按新旧键任一敏感判断。)
	old, _ := dao.SysConfig.Ctx(ctx).
		Where("id", req.Id).Where("deleted_at IS NULL").
		Fields("config_key, config_value").One()
	if !old.IsEmpty() &&
		(isSensitiveConfigKey(req.ConfigKey) || isSensitiveConfigKey(old["config_key"].String())) &&
		req.ConfigValue == maskConfigValue(old["config_value"].String()) {
		delete(data, "config_value")
	}

	if _, err = dao.SysConfig.Ctx(ctx).
		Where("id", req.Id).Where("deleted_at IS NULL").
		Data(data).Update(); err != nil {
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
