// Package configx sys_config 读取统一入口 (docs/multi-tenant.md D5 覆盖读序)。
//
// 读序: 白名单键 → 租户行 > 平台行 (tenant_id=1); 其余键 (storage/ai/wechat 等
// 基础设施配置) 仅平台级 —— 租户上下文不可见平台行, 必须经本包以平台视角读取,
// 否则注入器会把直查限定在租户行内导致配置"丢失"。
//
// 白名单键集 (仅开放租户可覆盖): 站点名/Logo/版权/注册开关/密码策略;
// sys.captcha_enable 保持平台级 (人机挑战是平台侧防滥用设施)。
package configx

import (
	"context"
	"sort"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/utility/contextx"
)

// overridePrefixes 允许租户覆盖的键前缀 (D5 白名单键集)。
var overridePrefixes = []string{
	"sys.name", "sys.logo", "sys.copyright", "sys.allow_register", "sys.password.",
}

// OverrideAllowed 判定键是否开放租户覆盖 (纯函数, 单测覆盖)。
func OverrideAllowed(key string) bool {
	for _, p := range overridePrefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

// cfgRow 配置行骨架 (纯函数 pick 用)。
type cfgRow struct {
	TenantId uint64
	Status   int
	Value    string
}

// pickMerged 覆盖读序纯函数: 候选行按 tenant_id 降序逐级回落, 取首个
// "启用且值非空白"的行; 租户行禁用/空白视同未覆盖, 回落平台行;
// 全部不可用返回 def (第二个返回值 false)。
func pickMerged(rows []cfgRow, def string) (string, bool) {
	sorted := make([]cfgRow, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TenantId > sorted[j].TenantId })
	for _, r := range sorted {
		if r.Status != consts.StatusEnabled {
			continue
		}
		if strings.TrimSpace(r.Value) == "" {
			continue
		}
		return r.Value, true
	}
	return def, false
}

// Value 覆盖读序读取: tenant 为当前数据租户 (可为 nil = 平台视角)。
// 白名单键合并租户/平台两行, 其余键仅平台行; 查询以平台视角 (屏蔽注入)。
func Value(ctx context.Context, tenant *model.TenantLite, key, def string) string {
	tid := uint64(consts.TenantPlatformId)
	if tenant != nil && tenant.Id > 0 {
		tid = tenant.Id
	}
	qctx := contextx.WithoutTenant(ctx)
	q := dao.SysConfig.Ctx(qctx).
		Where("config_key", key).
		Where("status", consts.StatusEnabled).
		Where("deleted_at IS NULL").
		Fields("config_value, tenant_id").
		Order("tenant_id ASC")
	if !OverrideAllowed(key) || tid == consts.TenantPlatformId {
		q = q.Where("tenant_id", consts.TenantPlatformId)
	} else {
		q = q.WhereIn("tenant_id", []uint64{consts.TenantPlatformId, tid})
	}
	res, err := q.All()
	if err != nil {
		g.Log().Warningf(qctx, "configx read %s failed: %v", key, err)
		return def
	}
	rows := make([]cfgRow, 0, len(res))
	for _, r := range res {
		rows = append(rows, cfgRow{TenantId: r["tenant_id"].Uint64(), Status: consts.StatusEnabled, Value: r["config_value"].String()})
	}
	if v, ok := pickMerged(rows, def); ok {
		return v
	}
	return def
}

// Bool 布尔读取 ("true"/"1")。
func Bool(ctx context.Context, tenant *model.TenantLite, key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(Value(ctx, tenant, key, "")))
	switch v {
	case "true", "1":
		return true
	case "false", "0":
		return false
	}
	return def
}

// PlatformValue 平台行读取 (非白名单键与平台语境统一入口)。
func PlatformValue(ctx context.Context, key, def string) string {
	return Value(ctx, nil, key, def)
}

// PlatformBool 平台行布尔读取。
func PlatformBool(ctx context.Context, key string, def bool) bool {
	return Bool(ctx, nil, key, def)
}

// PlatformPrefix 平台行前缀批量读取 (storage 等基础设施配置装载), 仅启用行。
// 返回 map[config_key]config_value。
func PlatformPrefix(ctx context.Context, prefix string) map[string]string {
	qctx := contextx.WithoutTenant(ctx)
	res, err := dao.SysConfig.Ctx(qctx).
		Where("config_key LIKE ?", prefix+"%").
		Where("tenant_id", consts.TenantPlatformId).
		Where("status", consts.StatusEnabled).
		Where("deleted_at IS NULL").
		Fields("config_key, config_value").All()
	if err != nil {
		g.Log().Warningf(qctx, "configx prefix read %s failed: %v", prefix, err)
		return map[string]string{}
	}
	out := make(map[string]string, len(res))
	for _, r := range res {
		out[r["config_key"].String()] = r["config_value"].String()
	}
	return out
}

// BuildEffectiveAll /configs/all 合并视图 (导出供 system/config 组装) (纯函数部分在 buildEffectiveAll):
//   - 白名单键: 租户行覆盖平台行 (含未覆盖键的平台默认值);
//   - 其余启用键: 平台值;
//   - 平台租户/平台视角: 全量平台行。
//
// rowsIn 为 (key -> {tenantId, status, value, type, name}) 全量候选。
func BuildEffectiveAll(rowsIn map[string][]CfgRowMeta, tid uint64) map[string]CfgRowMeta {
	out := make(map[string]CfgRowMeta, len(rowsIn))
	for key, rows := range rowsIn {
		var platform, tenantRow *CfgRowMeta
		for i := range rows {
			r := rows[i]
			if r.TenantId == consts.TenantPlatformId && r.Status == consts.StatusEnabled {
				platform = &rows[i]
			}
			if tid != consts.TenantPlatformId && r.TenantId == tid && r.Status == consts.StatusEnabled {
				tenantRow = &rows[i]
			}
		}
		// 平台行兜底
		if platform != nil {
			out[key] = *platform
		}
		// 白名单键租户覆盖
		if tenantRow != nil && OverrideAllowed(key) {
			out[key] = *tenantRow
		}
	}
	return out
}

// CfgRowMeta /configs/all 出参行 (键的候选行元数据)。
type CfgRowMeta struct {
	TenantId uint64
	Status   int
	Value    string
	Type     int
	Name     string
}
