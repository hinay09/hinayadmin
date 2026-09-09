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
