package crudgen

import "testing"

func TestValidateModAndTitle(t *testing.T) {
	cases := []struct {
		name, mod, title string
		wantErr          bool
	}{
		{"合法", "article", "文章管理", false},
		{"mod 空走默认", "", "标题", false},
		{"路径穿越-相对", "../../evil", "x", true},
		{"路径穿越-混合", "a/../../b", "x", true},
		{"路径穿越-点", "..", "x", true},
		{"mod 大写", "Article", "x", true},
		{"mod 含横线", "my-mod", "x", true},
		{"mod 单字符", "a", "x", true},
		{"mod 数字开头", "1abc", "x", true},
		{"mod 下划线合法", "a_1", "x", false},
		{"title 单引号", "ok", "含'引号", true},
		{"title 双引号", "ok", `含"引号`, true},
		{"title 反斜杠", "ok", "含\\反斜杠", true},
		{"title 换行", "ok", "含\n换行", true},
		{"title 中文合法", "ok", "通知公告中心", false},
		{"title 过长(33字)", "ok", "一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二三四", true},
	}
	for _, c := range cases {
		err := ValidateModAndTitle(c.mod, c.title)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: mod=%q title=%q err=%v, wantErr=%v", c.name, c.mod, c.title, err, c.wantErr)
		}
	}
}
