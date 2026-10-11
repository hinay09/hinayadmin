// configx_test.go —— 覆盖读序纯函数表驱动测试 (docs/multi-tenant.md D5)。
package configx

import "testing"

func TestOverrideAllowed(t *testing.T) {
	allowed := []string{"sys.name", "sys.logo", "sys.copyright", "sys.allow_register",
		"sys.password.min_length", "sys.password.require_upper"}
	denied := []string{"sys.captcha_enable", "file.storage.type", "ai.api_key",
		"wechat.app_id", "sys", "sys.passwordx", "sysx.name"}
	for _, k := range allowed {
		if !OverrideAllowed(k) {
			t.Errorf("OverrideAllowed(%q) = false, 期望 true", k)
		}
	}
	for _, k := range denied {
		if OverrideAllowed(k) {
			t.Errorf("OverrideAllowed(%q) = true, 期望 false", k)
		}
	}
}

func TestPickMerged(t *testing.T) {
	cases := []struct {
		name string
		rows []cfgRow
		def  string
		want string
		ok   bool
	}{
		{"租户行覆盖平台行", []cfgRow{{TenantId: 1, Status: 1, Value: "平台"}, {TenantId: 7, Status: 1, Value: "租户"}}, "d", "租户", true},
		{"无租户行回落平台", []cfgRow{{TenantId: 1, Status: 1, Value: "平台"}}, "d", "平台", true},
		{"租户行禁用回落平台", []cfgRow{{TenantId: 1, Status: 1, Value: "平台"}, {TenantId: 7, Status: 0, Value: "租户"}}, "d", "平台", true},
		{"全部禁用回落默认", []cfgRow{{TenantId: 1, Status: 0, Value: "x"}, {TenantId: 7, Status: 0, Value: "y"}}, "d", "d", false},
		{"空行集回落默认", nil, "d", "d", false},
		{"租户行空白值回落平台", []cfgRow{{TenantId: 1, Status: 1, Value: "平台"}, {TenantId: 7, Status: 1, Value: "  "}}, "d", "平台", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := pickMerged(c.rows, c.def)
			if got != c.want || ok != c.ok {
				t.Fatalf("pickMerged = (%q,%v), want (%q,%v)", got, ok, c.want, c.ok)
			}
		})
	}
}

func TestBuildEffectiveAll(t *testing.T) {
	rows := map[string][]CfgRowMeta{
		"sys.name": {
			{TenantId: 1, Status: 1, Value: "平台名", Name: "系统名称"},
			{TenantId: 7, Status: 1, Value: "租户名", Name: "系统名称"},
		},
		"sys.allow_register": {
			{TenantId: 1, Status: 1, Value: "false"},
		},
		"ai.api_key": {
			{TenantId: 1, Status: 1, Value: "sk-xxx"},
			{TenantId: 7, Status: 1, Value: "should-never-appear"},
		},
		"sys.disabled_key": {
			{TenantId: 1, Status: 0, Value: "x"},
		},
	}
	// 租户 7 视角: 白名单覆盖生效, 非白名单只取平台行, 禁用键不出现
	got := BuildEffectiveAll(rows, 7)
	if got["sys.name"].Value != "租户名" {
		t.Fatalf("sys.name = %q, 期望租户覆盖值", got["sys.name"].Value)
	}
	if got["sys.allow_register"].Value != "false" {
		t.Fatalf("sys.allow_register = %q, 期望平台值", got["sys.allow_register"].Value)
	}
	if got["ai.api_key"].Value != "sk-xxx" {
		t.Fatalf("ai.api_key = %q, 期望平台值 (非白名单键租户行不可见)", got["ai.api_key"].Value)
	}
	if _, ok := got["sys.disabled_key"]; ok {
		t.Fatal("禁用键不应出现")
	}
	// 平台视角: 不覆盖
	got = BuildEffectiveAll(rows, 1)
	if got["sys.name"].Value != "平台名" {
		t.Fatalf("平台视角 sys.name = %q, 期望平台值", got["sys.name"].Value)
	}
}
