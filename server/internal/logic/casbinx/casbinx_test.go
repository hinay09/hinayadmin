// Package casbinx 匹配器回归测试。
//
// casbin.model 的匹配器必须同时支持两种路径参数风格 (两者取或):
//   - ":id"  冒号风格 —— message 组 sys_api (/api/v1/message/inbox/:id), keyMatch2 支持
//   - "{id}" 花括号风格 —— flow 组 sys_api (/api/v1/flow/tasks/{id}/approve), keyMatch3 支持
//
// 中间件以**实际请求 URL** (如 /api/v1/flow/instances/42) 做 Enforce,
// 任一风格失效都会导致带参数接口对非 admin 角色全量 403 (p012 测试账号曾踩中)。
//
// 多租户域化: 模型为 sub/dom/obj/act 四元组, g 三元组按域匹配 ——
// 不同租户域的角色策略互不可见 ( roleId 跨租户唯一但策略隔离)。
// 本测试内联模型文本, 与 manifest/config/config.yaml 的 casbin.model 必须保持一致。
package casbinx

import (
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

const domainModelText = `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && (keyMatch2(r.obj, p.obj) || keyMatch3(r.obj, p.obj)) && (r.act == p.act || p.act == "*")
`

func TestMatcherDomainDualPathStyles(t *testing.T) {
	m, err := model.NewModelFromString(domainModelText)
	if err != nil {
		t.Fatalf("解析模型失败: %v", err)
	}
	// 域化策略: u1 在平台域(1)持角色 r1; u2 在租户域(7)持角色 r2 (策略路径同 r1)
	a := &memAdapter{lines: []string{
		"p, r1, 1, /api/v1/flow/instances/{id}, GET", // 花括号风格 (flow 组)
		"p, r1, 1, /api/v1/flow/tasks/{id}/approve, POST",
		"p, r1, 1, /api/v1/message/inbox/:id, GET",        // 冒号风格 (message 组)
		"p, r1, 1, /api/v1/auth/*, *",                     // 通配
		"p, r1, 1, /api/v1/flow/tasks/count, GET",         // 精确
		"p, r2, 7, /api/v1/flow/tasks/{id}/approve, POST", // 租户域 7 的同路径策略
		"g, u1, r1, 1",
		"g, u2, r2, 7",
	}}
	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		t.Fatalf("构建 enforcer 失败: %v", err)
	}

	cases := []struct {
		name string
		sub  string
		dom  string
		obj  string
		act  string
		want bool
	}{
		{"花括号风格匹配实际请求", "u1", "1", "/api/v1/flow/instances/42", "GET", true},
		{"花括号风格匹配动作接口", "u1", "1", "/api/v1/flow/tasks/88/approve", "POST", true},
		{"冒号风格匹配实际请求", "u1", "1", "/api/v1/message/inbox/9", "GET", true},
		{"通配符匹配", "u1", "1", "/api/v1/auth/refresh", "POST", true},
		{"精确路径匹配", "u1", "1", "/api/v1/flow/tasks/count", "GET", true},
		{"方法不符拒绝", "u1", "1", "/api/v1/flow/instances/42", "DELETE", false},
		{"未授权路径拒绝", "u1", "1", "/api/v1/system/roles/all", "GET", false},
		{"无角色用户拒绝", "nobody", "1", "/api/v1/flow/instances/42", "GET", false},
		// 域隔离: 平台域策略不作用于租户域, 反之亦然
		{"域不匹配拒绝(平台用户进租户域)", "u1", "7", "/api/v1/flow/instances/42", "GET", false},
		{"域不匹配拒绝(租户用户进平台域)", "u2", "1", "/api/v1/flow/tasks/9/approve", "POST", false},
		{"租户域内策略放行", "u2", "7", "/api/v1/flow/tasks/9/approve", "POST", true},
		{"租户域内无该策略拒绝", "u2", "7", "/api/v1/message/inbox/9", "GET", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := e.Enforce(c.sub, c.dom, c.obj, c.act)
			if err != nil {
				t.Fatalf("Enforce 出错: %v", err)
			}
			if got != c.want {
				t.Errorf("Enforce(%s, %s, %s, %s) = %v, 期望 %v", c.sub, c.dom, c.obj, c.act, got, c.want)
			}
		})
	}
}
