// Package casbinx Casbin enforcer 单例与策略加载 (域化多租户模型)。
//
// 域模型 (docs/multi-tenant.md §5): 原生 domain 模式, r/p 定义为 sub, dom, obj, act。
// 域 = 租户ID 十进制字符串 ('1'=平台域)。roleId 为 DB 自增跨租户唯一, 域化防撞号放串:
//   - p 行: v0=角色ID, v1=obj(path 或 menu:<菜单ID>), v2=act, v3=域(租户ID);
//   - g 行: v0=用户ID, v1=角色ID, v2=域(租户ID)。
//
// 列映射隔离在 loadRuleLines: p 行发 `p, v0, v3, v1, v2`; g 行发 `g, v0, v1, v2`。
// 域取自 ctx 租户上下文 (consts.CtxTenantKey), 无租户上下文默认平台域 '1'。
package casbinx

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
)

var (
	enforcer *casbin.Enforcer
	mu       sync.RWMutex
)

// ApiPolicy API 策略结构。
type ApiPolicy struct {
	Path   string
	Method string
}

// domainOf 从 ctx 提取 Casbin 域 (=租户ID 十进制字符串)。
// 不经 contextx (会形成循环依赖), 与 ormfill 同法直读 CtxTenantKey;
// 无租户上下文 (启动期/公开接口前置逻辑) 默认平台域 '1'。
func domainOf(ctx context.Context) string {
	if ctx != nil {
		if v := ctx.Value(consts.CtxTenantKey); v != nil {
			if t, ok := v.(*model.TenantInfo); ok && t != nil && t.Id > 0 {
				return strconv.FormatUint(t.Id, 10)
			}
		}
	}
	return "1"
}

// memAdapter 简易内存 adapter, 仅支持 LoadPolicy。
// 业务侧的策略增删通过直接写 casbin_rule 表后调用 Reload 实现。
type memAdapter struct {
	lines []string
}

// LoadPolicy 启动/重载时把预取的 casbin_rule 行文本灌进 model (逐行走标准解析)。
func (a *memAdapter) LoadPolicy(m casbinmodel.Model) error {
	for _, line := range a.lines {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		if err := persist.LoadPolicyLine(line, m); err != nil {
			return err
		}
	}
	return nil
}

// 以下四个写方法均为只读空实现: 策略变更走业务侧直接写表 + Reload, 不经过 adapter。
func (a *memAdapter) SavePolicy(_ casbinmodel.Model) error                       { return nil }
func (a *memAdapter) AddPolicy(_, _ string, _ []string) error                    { return nil }
func (a *memAdapter) RemovePolicy(_, _ string, _ []string) error                 { return nil }
func (a *memAdapter) RemoveFilteredPolicy(_, _ string, _ int, _ ...string) error { return nil }

// Get 获取(并按需加载)全局 enforcer。
func Get(ctx context.Context) (*casbin.Enforcer, error) {
	mu.RLock()
	if enforcer != nil {
		defer mu.RUnlock()
		return enforcer, nil
	}
	mu.RUnlock()
	return Reload(ctx)
}

// Reload 重新从 casbin_rule 表加载策略, 用于角色权限变更后即时生效。
// 全量内存策略随租户数线性增长 —— 千租户内无压力, 超出再按域分区加载 (二期)。
func Reload(ctx context.Context) (*casbin.Enforcer, error) {
	mu.Lock()
	defer mu.Unlock()

	modelText, err := g.Cfg().Get(ctx, "casbin.model")
	if err != nil {
		return nil, err
	}
	m, err := casbinmodel.NewModelFromString(modelText.String())
	if err != nil {
		return nil, err
	}

	lines, err := loadRuleLines(ctx, m)
	if err != nil {
		return nil, err
	}
	a := &memAdapter{lines: lines}
	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		return nil, err
	}
	enforcer = e
	return enforcer, nil
}

// loadRuleLines 从 casbin_rule 表读取所有策略, 拼接为标准 policy 行文本切片 (域化列映射)。
// 仅保留当前 model 中已声明的 ptype (如 p / g / g2),
// 避免历史数据中未在 model 声明的 ptype 导致 NewEnforcer 报
// "missing required section g" 之类错误。
// 列映射: p 行发 `p, v0(角色ID), v3(域), v1(obj), v2(act)`; g 行发 `g, v0(用户ID), v1(角色ID), v2(域)`。
// 域列为空的历史行回填平台域 '1' (存量库兼容, 升级脚本也会显式回填)。
func loadRuleLines(ctx context.Context, m casbinmodel.Model) ([]string, error) {
	res, err := dao.CasbinRule.Ctx(ctx).All()
	if err != nil {
		return nil, err
	}
	allowed := map[string]struct{}{}
	for sec, asts := range m {
		if sec != "p" && sec != "g" {
			continue
		}
		for key := range asts {
			allowed[key] = struct{}{}
		}
	}
	val := func(r gdb.Record, k string) string { return strings.TrimSpace(r[k].String()) }
	out := make([]string, 0, len(res))
	for _, r := range res {
		ptype := val(r, "ptype")
		if ptype == "" {
			continue
		}
		if _, ok := allowed[ptype]; !ok {
			g.Log().Warningf(ctx, "casbin: skip unknown ptype %q (not declared in model)", ptype)
			continue
		}
		v0, v1, v2, v3 := val(r, "v0"), val(r, "v1"), val(r, "v2"), val(r, "v3")
		var line string
		switch ptype {
		case "p":
			dom := v3
			if dom == "" {
				dom = "1"
			}
			line = fmt.Sprintf("p, %s, %s, %s, %s", v0, dom, v1, v2)
		case "g":
			dom := v2
			if dom == "" {
				dom = "1"
			}
			line = fmt.Sprintf("g, %s, %s, %s", v0, v1, dom)
		default:
			line = ptype
			for _, k := range []string{"v0", "v1", "v2", "v3", "v4", "v5"} {
				v := val(r, k)
				if v == "" {
					break
				}
				line += ", " + v
			}
		}
		out = append(out, line)
	}
	return out, nil
}

// Enforce 校验 sub(userId)/域(租户)/obj(path)/act(method) 是否允许。
// g 策略以 用户ID-角色ID-域 关联, 因此 sub 传用户ID(十进制字符串),
// 用户名/角色code 的变更不影响策略匹配。
// 当 enforcer 返回 false 时，追加 DB 直查兜底，避免因 Reload 并发窗口导致的误拒。
func Enforce(ctx context.Context, userId uint64, obj, act string) (bool, error) {
	sub := strconv.FormatUint(userId, 10)
	dom := domainOf(ctx)
	e, err := Get(ctx)
	if err != nil {
		return false, err
	}
	ok, err := e.Enforce(sub, dom, obj, act)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}

	// DB 兜底: 规避 Get() 返回旧 enforcer 导致的并发竞争（UserCreate Reload 尚未刷新生效）
	count, cerr := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "p").
		Where("v0 IN (SELECT v1 FROM casbin_rule WHERE ptype='g' AND v0=? AND v2=?)", sub, dom).
		Where("v1", obj).
		WhereIn("v2", []string{act, "*"}).
		Where("v3", dom).
		Count()
	if cerr != nil {
		g.Log().Warningf(ctx, "casbin db fallback query failed: %v", cerr)
		return false, nil
	}
	if count > 0 {
		g.Log().Infof(ctx, "[casbin] db fallback hit: sub=%s dom=%s obj=%s act=%s (enforcer returned false)", sub, dom, obj, act)
		return true, nil
	}
	return false, nil
}

// GetUserRoles 通过 Casbin g 策略获取用户在当前域的所有角色ID
// (g 行 v1 即角色ID, v2=域过滤)。
func GetUserRoles(ctx context.Context, userId uint64) ([]uint64, error) {
	res, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "g").
		Where("v0", strconv.FormatUint(userId, 10)).
		Where("v2", domainOf(ctx)).
		Fields("v1").
		Ctx(ctx).All()
	if err != nil {
		return nil, err
	}
	roles := make([]uint64, 0, len(res))
	for _, r := range res {
		if id, e := strconv.ParseUint(strings.TrimSpace(r["v1"].String()), 10, 64); e == nil && id > 0 {
			roles = append(roles, id)
		}
	}
	return roles, nil
}

// SetUserRoles 设置用户在当前域的所有角色（先清除旧的再设置新的, g 行以
// 用户ID/角色ID/域 关联; 跨域 g 行不受影响）。
func SetUserRoles(ctx context.Context, userId uint64, roleIds []uint64) error {
	uid := strconv.FormatUint(userId, 10)
	dom := domainOf(ctx)
	err := dao.CasbinRule.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := dao.CasbinRule.Ctx(ctx).
			Where("ptype", "g").
			Where("v0", uid).
			Where("v2", dom).
			Delete(); err != nil {
			return err
		}
		rows := make([]g.Map, 0, len(roleIds))
		seen := make(map[uint64]struct{}, len(roleIds))
		for _, rid := range roleIds {
			if rid == 0 {
				continue
			}
			// 去重: 客户端重复提交同一角色会撞唯一键导致整个事务失败
			if _, dup := seen[rid]; dup {
				continue
			}
			seen[rid] = struct{}{}
			rows = append(rows, g.Map{"ptype": "g", "v0": uid, "v1": strconv.FormatUint(rid, 10), "v2": dom})
		}
		if len(rows) > 0 {
			if _, err := dao.CasbinRule.Ctx(ctx).Data(rows).Insert(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if _, rerr := Reload(ctx); rerr != nil {
		return rerr
	}
	return nil
}

// GetRoleMenus 获取角色在指定域关联的菜单ID列表（从 p 策略中 obj 以 "menu:" 开头的提取）。
func GetRoleMenus(ctx context.Context, roleId uint64) ([]int64, error) {
	res, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "p").
		Where("v0", strconv.FormatUint(roleId, 10)).
		Where("v3", domainOf(ctx)).
		WhereLike("v1", "menu:%").
		Fields("v1").
		Ctx(ctx).All()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(res))
	for _, r := range res {
		s := strings.TrimPrefix(r["v1"].String(), "menu:")
		id, e := strconv.ParseInt(s, 10, 64)
		if e == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// SetRoleMenus 设置角色在当前域的菜单权限策略 (p 行 v0 为角色ID, v3 为域)。
// 注意: 本函数不做授权集校验 —— 租户域的应用授权集圈定由 system/role.go 调用侧
// 执行 (§7.1 写入侧校验), casbinx 保持纯策略原语。
func SetRoleMenus(ctx context.Context, roleId uint64, menuIds []int64) error {
	rid := strconv.FormatUint(roleId, 10)
	dom := domainOf(ctx)
	err := dao.CasbinRule.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := dao.CasbinRule.Ctx(ctx).
			Where("ptype", "p").
			Where("v0", rid).
			Where("v3", dom).
			WhereLike("v1", "menu:%").
			Delete(); err != nil {
			return err
		}
		rows := make([]g.Map, 0, len(menuIds))
		seen := make(map[int64]struct{}, len(menuIds))
		for _, mid := range menuIds {
			if _, dup := seen[mid]; dup {
				continue
			}
			seen[mid] = struct{}{}
			rows = append(rows, g.Map{"ptype": "p", "v0": rid, "v1": fmt.Sprintf("menu:%d", mid), "v2": "*", "v3": dom})
		}
		if len(rows) > 0 {
			if _, err := dao.CasbinRule.Ctx(ctx).Data(rows).Insert(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if _, rerr := Reload(ctx); rerr != nil {
		return rerr
	}
	return nil
}

// GetRoleApis 获取角色在当前域的 API 策略。
func GetRoleApis(ctx context.Context, roleId uint64) ([]ApiPolicy, error) {
	res, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "p").
		Where("v0", strconv.FormatUint(roleId, 10)).
		Where("v3", domainOf(ctx)).
		WhereNotLike("v1", "menu:%").
		Fields("v1,v2").
		Ctx(ctx).All()
	if err != nil {
		return nil, err
	}
	apis := make([]ApiPolicy, 0, len(res))
	for _, r := range res {
		path := strings.TrimSpace(r["v1"].String())
		method := strings.TrimSpace(r["v2"].String())
		if path != "" {
			apis = append(apis, ApiPolicy{Path: path, Method: method})
		}
	}
	return apis, nil
}

// SetRoleApis 设置角色在当前域的 API 策略 (p 行 v0 为角色ID, v3 为域)。
// 注意: 应用授权集校验 (API 的 sys_api.menu_id ∈ 授权集闭包) 在 system/role.go 调用侧。
func SetRoleApis(ctx context.Context, roleId uint64, apis []ApiPolicy) error {
	rid := strconv.FormatUint(roleId, 10)
	dom := domainOf(ctx)
	err := dao.CasbinRule.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := dao.CasbinRule.Ctx(ctx).
			Where("ptype", "p").
			Where("v0", rid).
			Where("v3", dom).
			WhereNotLike("v1", "menu:%").
			Delete(); err != nil {
			return err
		}
		rows := make([]g.Map, 0, len(apis))
		type key struct{ p, m string }
		seen := make(map[key]struct{}, len(apis))
		for _, a := range apis {
			if a.Path == "" || a.Method == "" {
				continue
			}
			k := key{a.Path, a.Method}
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			rows = append(rows, g.Map{"ptype": "p", "v0": rid, "v1": a.Path, "v2": a.Method, "v3": dom})
		}
		if len(rows) > 0 {
			if _, err := dao.CasbinRule.Ctx(ctx).Data(rows).Insert(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if _, rerr := Reload(ctx); rerr != nil {
		return rerr
	}
	return nil
}

// RemoveRolePolicies 删除角色在当前域的所有策略（p 按角色ID+域匹配;
// g 行中该角色作为角色的记录按 v1=角色ID+v2=域 匹配, 跨域不受影响）。
func RemoveRolePolicies(ctx context.Context, roleId uint64) error {
	rid := strconv.FormatUint(roleId, 10)
	dom := domainOf(ctx)
	err := dao.CasbinRule.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := dao.CasbinRule.Ctx(ctx).
			Where("ptype", "p").
			Where("v0", rid).
			Where("v3", dom).
			Delete(); err != nil {
			return err
		}
		if _, err := dao.CasbinRule.Ctx(ctx).
			Where("ptype", "g").
			Where("v1", rid).
			Where("v2", dom).
			Delete(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if _, rerr := Reload(ctx); rerr != nil {
		return rerr
	}
	return nil
}

// RemoveRolePoliciesDomain 删除指定域内角色的全部 p 策略 (菜单+API),
// 供授权收缩联动按域清理 (§7.1), 不触碰 g 行。
func RemoveRolePoliciesDomain(ctx context.Context, roleId uint64, dom string) error {
	rid := strconv.FormatUint(roleId, 10)
	if _, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "p").
		Where("v0", rid).
		Where("v3", dom).
		Delete(); err != nil {
		return err
	}
	return nil
}
