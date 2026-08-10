// Package casbinx Casbin enforcer 单例与策略加载。
package casbinx

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/dao"
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

// memAdapter 简易内存 adapter, 仅支持 LoadPolicy。
// 业务侧的策略增删通过直接写 casbin_rule 表后调用 Reload 实现。
type memAdapter struct {
	lines []string
}

func (a *memAdapter) LoadPolicy(m model.Model) error {
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

func (a *memAdapter) SavePolicy(_ model.Model) error                             { return nil }
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
func Reload(ctx context.Context) (*casbin.Enforcer, error) {
	mu.Lock()
	defer mu.Unlock()

	modelText, err := g.Cfg().Get(ctx, "casbin.model")
	if err != nil {
		return nil, err
	}
	m, err := model.NewModelFromString(modelText.String())
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

// loadRuleLines 从 casbin_rule 表读取所有策略, 拼接为标准 policy 行文本切片。
// 仅保留当前 model 中已声明的 ptype (如 p / g / g2),
// 避免历史数据中未在 model 声明的 ptype 导致 NewEnforcer 报
// "missing required section g" 之类错误。
func loadRuleLines(ctx context.Context, m model.Model) ([]string, error) {
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
	out := make([]string, 0, len(res))
	for _, r := range res {
		ptype := strings.TrimSpace(r["ptype"].String())
		if ptype == "" {
			continue
		}
		if _, ok := allowed[ptype]; !ok {
			g.Log().Warningf(ctx, "casbin: skip unknown ptype %q (not declared in model)", ptype)
			continue
		}
		line := ptype
		for _, k := range []string{"v0", "v1", "v2", "v3", "v4", "v5"} {
			v := strings.TrimSpace(r[k].String())
			if v == "" {
				break
			}
			line += ", " + v
		}
		out = append(out, line)
	}
	return out, nil
}

// Enforce 校验 sub(username)/obj(path)/act(method) 是否允许。
// 新 model 使用 g(r.sub, p.sub) 自动解析用户角色归属, 因此 sub 直接传 username。
// 当 enforcer 返回 false 时，追加 DB 直查兜底，避免因 Reload 并发窗口导致的误拒。
func Enforce(ctx context.Context, sub, obj, act string) (bool, error) {
	e, err := Get(ctx)
	if err != nil {
		return false, err
	}
	ok, err := e.Enforce(sub, obj, act)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}

	// DB 兜底: 规避 Get() 返回旧 enforcer 导致的并发竞争（UserCreate Reload 尚未刷新生效）
	count, cerr := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "p").
		Where("v0 IN (SELECT v1 FROM casbin_rule WHERE ptype='g' AND v0=?)", sub).
		Where("v1", obj).
		Where("v2 IN (?, ?)", act, "*").
		Count()
	if cerr != nil {
		g.Log().Warningf(ctx, "casbin db fallback query failed: %v", cerr)
		return false, nil
	}
	if count > 0 {
		g.Log().Infof(ctx, "[casbin] db fallback hit: sub=%s obj=%s act=%s (enforcer returned false)", sub, obj, act)
		return true, nil
	}
	return false, nil
}

// GetUserRoles 通过 Casbin g 策略获取用户的所有角色。
func GetUserRoles(ctx context.Context, username string) ([]string, error) {
	res, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "g").
		Where("v0", username).
		Fields("v1").
		Ctx(ctx).All()
	if err != nil {
		return nil, err
	}
	roles := make([]string, 0, len(res))
	for _, r := range res {
		if v := strings.TrimSpace(r["v1"].String()); v != "" {
			roles = append(roles, v)
		}
	}
	return roles, nil
}

// AddUserRole 添加用户-角色映射 (g 策略)。
func AddUserRole(ctx context.Context, username, roleCode string) error {
	_, err := dao.CasbinRule.Ctx(ctx).Data(g.Map{
		"ptype": "g",
		"v0":    username,
		"v1":    roleCode,
	}).Insert()
	if err != nil {
		return err
	}
	_, _ = Reload(ctx)
	return nil
}

// RemoveUserRole 移除用户-角色映射 (g 策略)。
func RemoveUserRole(ctx context.Context, username, roleCode string) error {
	_, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "g").
		Where("v0", username).
		Where("v1", roleCode).
		Delete()
	if err != nil {
		return err
	}
	_, _ = Reload(ctx)
	return nil
}

// SetUserRoles 设置用户的所有角色（先清除旧的再设置新的）。
func SetUserRoles(ctx context.Context, username string, roleCodes []string) error {
	err := dao.CasbinRule.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Exec("DELETE FROM casbin_rule WHERE ptype='g' AND v0=?", username); err != nil {
			return err
		}
		for _, code := range roleCodes {
			if _, err := tx.Exec("INSERT INTO casbin_rule(ptype,v0,v1) VALUES('g',?,?)", username, code); err != nil {
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

// GetRoleMenus 获取角色关联的菜单ID列表（从 p 策略中 obj 以 "menu:" 开头的提取）。
func GetRoleMenus(ctx context.Context, roleCode string) ([]int64, error) {
	res, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "p").
		Where("v0", roleCode).
		Where("v1 LIKE ?", "menu:%").
		Fields("v1").
		Ctx(ctx).All()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(res))
	for _, r := range res {
		s := strings.TrimPrefix(r["v1"].String(), "menu:")
		id, e := strconv.ParseInt(s, 10, 64)
		if e != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// SetRoleMenus 设置角色的菜单权限策略。
func SetRoleMenus(ctx context.Context, roleCode string, menuIds []int64) error {
	err := dao.CasbinRule.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 删除该角色所有 menu: 前缀的 p 策略
		if _, err := tx.Exec("DELETE FROM casbin_rule WHERE ptype='p' AND v0=? AND v1 LIKE 'menu:%'", roleCode); err != nil {
			return err
		}
		// 批量插入新的菜单策略
		for _, mid := range menuIds {
			obj := fmt.Sprintf("menu:%d", mid)
			if _, err := tx.Exec("INSERT INTO casbin_rule(ptype,v0,v1,v2) VALUES('p',?,?,?)", roleCode, obj, "*"); err != nil {
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

// GetRoleApis 获取角色的 API 策略。
func GetRoleApis(ctx context.Context, roleCode string) ([]ApiPolicy, error) {
	res, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "p").
		Where("v0", roleCode).
		Where("v1 NOT LIKE ?", "menu:%").
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

// SetRoleApis 设置角色的 API 策略。
func SetRoleApis(ctx context.Context, roleCode string, apis []ApiPolicy) error {
	err := dao.CasbinRule.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 删除该角色所有非 menu: 前缀的 p 策略
		if _, err := tx.Exec("DELETE FROM casbin_rule WHERE ptype='p' AND v0=? AND v1 NOT LIKE 'menu:%'", roleCode); err != nil {
			return err
		}
		// 批量插入新的 API 策略
		for _, a := range apis {
			if a.Path == "" || a.Method == "" {
				continue
			}
			if _, err := tx.Exec("INSERT INTO casbin_rule(ptype,v0,v1,v2) VALUES('p',?,?,?)", roleCode, a.Path, a.Method); err != nil {
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

// RemoveRolePolicies 删除角色的所有策略（p 和 g 中涉及该角色的）。
func RemoveRolePolicies(ctx context.Context, roleCode string) error {
	err := dao.CasbinRule.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 删除 p 策略中 sub=roleCode 的记录
		if _, err := tx.Exec("DELETE FROM casbin_rule WHERE ptype='p' AND v0=?", roleCode); err != nil {
			return err
		}
		// 删除 g 策略中角色作为角色的记录 (v1=roleCode)
		if _, err := tx.Exec("DELETE FROM casbin_rule WHERE ptype='g' AND v1=?", roleCode); err != nil {
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
