// Package casbinx 匹配器回归测试。
//
// casbin.model 的匹配器必须同时支持两种路径参数风格 (两者取或):
//   - ":id"  冒号风格  —— message 组 sys_api (/api/v1/message/inbox/:id), keyMatch2 支持
//   - "{id}" 花括号风格 —— flow 组 sys_api (/api/v1/flow/tasks/{id}/approve), keyMatch3 支持
// 中间件以**实际请求 URL** (如 /api/v1/flow/instances/42) 做 Enforce,
// 任一风格失效都会导致带参数接口对非 admin 角色全量 403 (p012 测试账号曾踩中)。
// 本测试内联模型文本, 与 manifest/config/config.yaml 的 casbin.model 必须保持一致。
package casbinx

import (
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

const dualStyleModelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && (keyMatch2(r.obj, p.obj) || keyMatch3(r.obj, p.obj)) && (r.act == p.act || p.act == "*")
`

func TestMatcherDualPathStyles(t *testing.T) {
	m, err := model.NewModelFromString(dualStyleModelText)
	if err != nil {
		t.Fatalf("解析模型失败: %v", err)
	}
	a := &memAdapter{lines: []string{
		"p, r1, /api/v1/flow/instances/{id}, GET",       // 花括号风格 (flow 组)
		"p, r1, /api/v1/flow/tasks/{id}/approve, POST",  // 花括号风格
		"p, r1, /api/v1/message/inbox/:id, GET",         // 冒号风格 (message 组)
		"p, r1, /api/v1/auth/*, *",                      // 通配
		"p, r1, /api/v1/flow/tasks/count, GET",          // 精确
		"g, u1, r1",
	}}
	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		t.Fatalf("构建 enforcer 失败: %v", err)
	}

	cases := []struct {
		name string
		obj  string
		act  string
		want bool
	}{
		{"花括号风格匹配实际请求", "/api/v1/flow/instances/42", "GET", true},
		{"花括号风格匹配动作接口", "/api/v1/flow/tasks/88/approve", "POST", true},
		{"冒号风格匹配实际请求", "/api/v1/message/inbox/9", "GET", true},
		{"通配符匹配", "/api/v1/auth/refresh", "POST", true},
		{"精确路径匹配", "/api/v1/flow/tasks/count", "GET", true},
		{"方法不符拒绝", "/api/v1/flow/instances/42", "DELETE", false},
		{"未授权路径拒绝", "/api/v1/system/roles/all", "GET", false},
		{"字面相同按精确匹配放行(无害, 真实请求不含花括号)", "/api/v1/flow/tasks/{id}/approve", "POST", true},
		{"无角色用户拒绝", "/api/v1/flow/instances/42", "GET", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.name == "无角色用户拒绝" {
				got, err := e.Enforce("nobody", c.obj, c.act)
				if err != nil || got {
					t.Errorf("Enforce(nobody, %s, %s) = %v, err=%v, 期望 false", c.obj, c.act, got, err)
				}
				return
			}
			got, err := e.Enforce("u1", c.obj, c.act)
			if err != nil {
				t.Fatalf("Enforce 出错: %v", err)
			}
			if got != c.want {
				t.Errorf("Enforce(u1, %s, %s) = %v, 期望 %v", c.obj, c.act, got, c.want)
			}
		})
	}
}
