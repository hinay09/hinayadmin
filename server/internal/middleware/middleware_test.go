package middleware

import (
	"strings"
	"testing"
)

func TestSanitizeLogBody(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string // want 为空表示期望返回空串
	}{
		{
			name:  "密码字段脱敏",
			input: `{"oldPassword":"secret123","newPassword":"newsecret456"}`,
			want:  `{"newPassword":"***","oldPassword":"***"}`,
		},
		{
			name:  "嵌套与数组递归脱敏",
			input: `{"user":{"password":"p","name":"a"},"list":[{"token":"t","id":1}]}`,
			want:  `{"list":[{"id":1,"token":"***"}],"user":{"name":"a","password":"***"}}`,
		},
		{
			name:  "非敏感字段保留",
			input: `{"nickname":"张三","email":"a@b.c"}`,
			want:  `{"email":"a@b.c","nickname":"张三"}`,
		},
		{
			name:  "非JSON输入不记录",
			input: `password=abc&x=1`,
			want:  "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeLogBody(c.input, 2048)
			if got != c.want {
				t.Fatalf("sanitizeLogBody(%s) = %s, want %s", c.input, got, c.want)
			}
		})
	}

	if got := sanitizeLogBody(strings.Repeat(`{"k":"v"},`, 500), 128); len(got) > 140 {
		t.Fatalf("truncation failed, got len %d", len(got))
	}
}

func TestParseOperation(t *testing.T) {
	cases := []struct {
		name                     string
		path, method             string
		wantRes, wantAct, wantId string
	}{
		// 资源归属: 剥离模块前缀取业务段
		{"模块前缀剥离", "/api/v1/system/users", "POST", "users", "create", ""},
		{"biz前缀剥离", "/api/v1/biz/orders", "POST", "orders", "create", ""},
		{"cms前缀剥离", "/api/v1/cms/articles", "GET", "articles", "", ""},
		{"flow前缀剥离", "/api/v1/flow/instances", "PUT", "instances", "update", ""},
		{"无前缀取第一段", "/api/v1/leaves", "POST", "leaves", "create", ""},
		{"auth本身即资源", "/api/v1/auth/profile", "PUT", "auth", "update", ""},

		// 动作映射表
		{"POST-create", "/api/v1/system/users", "POST", "users", "create", ""},
		{"PUT-update", "/api/v1/system/roles", "PUT", "roles", "update", ""},
		{"DELETE-delete", "/api/v1/system/menus", "DELETE", "menus", "delete", ""},
		{"GET无动作", "/api/v1/system/users", "GET", "users", "", ""},

		// ID 提取与动作细化
		{"ID提取", "/api/v1/system/users/123", "PUT", "users", "update", "123"},
		{"ID后子操作", "/api/v1/system/users/123/password", "PUT", "users", "update_password", "123"},
		{"ID删除", "/api/v1/system/users/123", "DELETE", "users", "delete", "123"},
		{"ID后子操作-POST", "/api/v1/leaves/9/submit", "POST", "leaves", "create_submit", "9"},
		{"ID后子操作-PUT", "/api/v1/flow/instances/3/terminate", "PUT", "instances", "update_terminate", "3"},

		// upload 覆盖
		{"上传覆盖动作", "/api/v1/system/files/upload", "POST", "files", "upload", ""},

		// 特例路径 (动作不走 method 推导)
		{"登录", "/api/v1/auth/login", "POST", "auth", "login", ""},
		{"登出", "/api/v1/auth/logout", "POST", "auth", "logout", ""},
		{"续签", "/api/v1/auth/refresh", "POST", "auth", "refresh", ""},
		{"TOTP登录", "/api/v1/auth/login/totp", "POST", "auth", "login_totp", ""},
		{"TOTP开启", "/api/v1/auth/totp/enable", "POST", "auth", "totp_enable", ""},
		{"TOTP关闭", "/api/v1/auth/totp/disable", "POST", "auth", "totp_disable", ""},
		{"头像上传", "/api/v1/auth/avatar", "POST", "auth", "upload_avatar", ""},

		// 边界
		{"空路径", "/api/v1/", "POST", "", "", ""},
		{"仅前缀", "/api/v1", "POST", "", "", ""},
		{"模块前缀后无业务段", "/api/v1/system", "POST", "system", "create", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, act, id := parseOperation(c.path, c.method)
			if res != c.wantRes || act != c.wantAct || id != c.wantId {
				t.Fatalf("parseOperation(%s, %s) = (%q, %q, %q), want (%q, %q, %q)",
					c.path, c.method, res, act, id, c.wantRes, c.wantAct, c.wantId)
			}
		})
	}
}
