// Package system 系统管理-全局配置业务逻辑。
package system

import (
	"context"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/configx"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/tenantx"
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

// All 获取所有生效配置 (docs/multi-tenant.md D5 覆盖读序的前端消费口)。
// 多租户开启时:
//   - 平台租户用户: 全量平台行 (含敏感项脱敏, 行为同旧版);
//   - 租户用户: 仅白名单键 (租户覆盖值 > 平台默认值), 基础设施配置 (storage/ai/
//     wechat 等) 不下发 —— 前端 config store 零改动自动生效 (§8.5)。
//
// 无论何种视角, 追加合成键 sys.multi_tenant (布尔) —— 开关状态前端感知 (§9)。
func (s *sConfig) All(ctx context.Context, _ *v1.ConfigAllReq) (res *v1.ConfigAllRes, err error) {
	tid := contextx.TenantId(ctx)
	tenantView := tenantx.Enabled(ctx) && tid != consts.TenantPlatformId && tid != 0

	qctx := contextx.WithoutTenant(ctx)
	q := dao.SysConfig.Ctx(qctx).
		Where("status", 1).
		Where("deleted_at IS NULL")
	if tenantView {
		q = q.WhereIn("tenant_id", []uint64{consts.TenantPlatformId, tid})
	} else {
		q = q.Where("tenant_id", consts.TenantPlatformId)
	}
	var rows []*model.ConfigItem
	if err = q.Order("sort ASC, id ASC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// 分桶: 平台行全集 + 租户覆盖行 (configx.buildEffectiveAll 同一读序语义)
	meta := make(map[string][]configx.CfgRowMeta, len(rows))
	for _, item := range rows {
		tidRow := uint64(consts.TenantPlatformId)
		if item.TenantId > 0 {
			tidRow = item.TenantId
		}
		meta[item.ConfigKey] = append(meta[item.ConfigKey], configx.CfgRowMeta{
			TenantId: tidRow, Status: 1, Value: item.ConfigValue, Type: item.ConfigType, Name: item.Name,
		})
	}
	base := uint64(consts.TenantPlatformId)
	if tenantView {
		base = tid
	}
	merged := configx.BuildEffectiveAll(meta, base)

	// 转为 key-value map: 租户视角仅白名单键
	list := make(g.Map, len(merged)+1)
	for key, m := range merged {
		if tenantView && !configx.OverrideAllowed(key) {
			continue
		}
		v := m.Value
		if isSensitiveConfigKey(key) {
			v = maskConfigValue(v)
		}
		list[key] = g.Map{"value": v, "type": m.Type, "name": m.Name}
	}
	list["sys.multi_tenant"] = g.Map{"value": strconv.FormatBool(tenantx.Enabled(ctx)), "type": 2, "name": "多租户"}
	return &v1.ConfigAllRes{List: list}, nil
}

// Create 新增全局配置。
func (s *sConfig) Create(ctx context.Context, req *v1.ConfigCreateReq) (res *v1.ConfigCreateRes, err error) {
	// 检查 config_key 唯一性
	// 多租户 (D5): 租户管理员仅可创建白名单键的覆盖行; 非白名单键为平台级配置,
	// 仅平台超管可维护。注入器按 ctx 归属写 tenant_id (平台超管未切换=平台行)。
	if verr := guardConfigWrite(ctx, req.ConfigKey); verr != nil {
		return nil, verr
	}
	cnt, _ := dao.SysConfig.Ctx(contextx.ScopedToDataTenant(ctx)).
		Where("config_key", req.ConfigKey).
		Where("deleted_at IS NULL").
		Count()
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
	cnt, _ := dao.SysConfig.Ctx(contextx.ScopedToDataTenant(ctx)).
		Where("config_key", req.ConfigKey).
		Where("id != ?", req.Id).
		Where("deleted_at IS NULL").
		Count()
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

	// 多租户: 行归属校验 (租户管理员仅可改本租户覆盖行且限白名单键;
	// 平台行 (tenant_id=1) 仅平台侧可改 —— 租户管理员视角下行不可见, 显式复核)
	if verr := guardConfigRowWrite(ctx, req.Id, req.ConfigKey); verr != nil {
		return nil, verr
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

// guardConfigWrite 配置写入键守卫 (docs/multi-tenant.md D5):
// 租户域仅开放白名单键 (租户覆盖行); 非白名单键仅平台超管。
func guardConfigWrite(ctx context.Context, key string) error {
	if !tenantx.Enabled(ctx) {
		return nil
	}
	tid := contextx.TenantId(ctx)
	if tid == consts.TenantPlatformId || tid == 0 {
		return nil // 平台侧 (平台租户用户/平台超管未切换): 维护平台行
	}
	if contextx.IsPlatformAdmin(ctx) {
		return nil // 平台超管代运维切换态: 视同租户管理员规则
	}
	if !configx.OverrideAllowed(key) {
		return xerror.New(xerror.CodeForbidden, "该配置为平台级配置, 租户不可维护")
	}
	return nil
}

// guardConfigRowWrite 配置行写入归属校验: 行必须属于当前数据租户 (或平台行由平台侧操作)。
func guardConfigRowWrite(ctx context.Context, id uint64, key string) error {
	if !tenantx.Enabled(ctx) {
		return nil
	}
	row, err := dao.SysConfig.Ctx(contextx.WithoutTenant(ctx)).
		Where("id", id).Where("deleted_at IS NULL").Fields("tenant_id, config_key").One()
	if err != nil || row.IsEmpty() {
		return nil // 存在性/注入归属由后续更新语义兜底
	}
	rowTid := row["tenant_id"].Uint64()
	tid := contextx.TenantId(ctx)
	platformSide := tid == consts.TenantPlatformId || tid == 0 || contextx.IsPlatformAdmin(ctx)
	if rowTid == consts.TenantPlatformId {
		if !platformSide {
			return xerror.New(xerror.CodeForbidden, "平台配置行仅平台侧可修改")
		}
		return nil
	}
	// 租户覆盖行: 本租户管理员可改 (白名单键校验), 平台干预走租户管理"重置配置"
	if rowTid != tid && !platformSide {
		return xerror.New(xerror.CodeForbidden, "不可修改其他租户的配置")
	}
	if !platformSide {
		return guardConfigWrite(ctx, key)
	}
	return nil
}
