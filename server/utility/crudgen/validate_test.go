package crudgen

import (
	"strings"
	"testing"
)

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

func TestDetectKind(t *testing.T) {
	cases := []struct {
		name, col, sqlType, comment string
		wantKind, wantDict, wantRel string
		wantGuess                   bool
	}{
		{"状态列", "status", "tinyint", "状态", "switch", "", "", false},
		{"后缀状态列", "audit_status", "tinyint", "审核状态", "switch", "", "", false},
		{"状态列字典指令升级", "status", "tinyint", "状态(字典:common_status)", "dict", "common_status", "", false},
		{"备注", "remark", "varchar", "备注", "textarea", "", "", false},
		{"字典指令", "order_type", "varchar", "订单类型(字典:order_type)", "dict", "order_type", "", false},
		{"全角冒号字典", "level", "int", "等级(字典：user_level)", "dict", "user_level", "", false},
		{"显式关联", "user_id", "bigint", "用户(关联:user)", "relation", "", "user", false},
		{"裸id猜测关联", "category_id", "bigint", "分类ID", "relation", "", "category", true},
		{"金额", "amount", "decimal", "金额", "number", "", "", false},
		{"整型", "count", "int", "数量", "number", "", "", false},
		{"日期时间", "meet_time", "datetime", "开始时间", "date", "", "", false},
		{"日期", "birth", "date", "生日", "date", "", "", false},
		{"普通文本", "title", "varchar", "标题", "input", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, dict, rel, guess := detectKind(c.col, c.sqlType, c.comment)
			if kind != c.wantKind || dict != c.wantDict || rel != c.wantRel || guess != c.wantGuess {
				t.Fatalf("detectKind(%s,%s,%s) = (%q,%q,%q,%v), want (%q,%q,%q,%v)",
					c.col, c.sqlType, c.comment, kind, dict, rel, guess, c.wantKind, c.wantDict, c.wantRel, c.wantGuess)
			}
		})
	}
}

// orderInputs 特性全覆盖的列输入 (org_id/状态/字典/关联/金额)。
var orderInputs = []ColumnInput{
	{Name: "id", DataType: "bigint", ColumnType: "bigint unsigned"},
	{Name: "title", DataType: "varchar", ColumnType: "varchar(128)", Comment: "订单标题"},
	{Name: "user_id", DataType: "bigint", ColumnType: "bigint unsigned", Comment: "下单用户ID(关联:user)"},
	{Name: "org_id", DataType: "bigint", ColumnType: "bigint unsigned", Comment: "所属组织ID"},
	{Name: "order_type", DataType: "varchar", ColumnType: "varchar(32)", Comment: "订单类型(字典:order_type)"},
	{Name: "amount", DataType: "decimal", ColumnType: "decimal(12,2)", Comment: "金额"},
	{Name: "status", DataType: "tinyint", ColumnType: "tinyint", Comment: "状态"},
	{Name: "remark", DataType: "varchar", ColumnType: "varchar(255)", Comment: "备注"},
	{Name: "create_id", DataType: "bigint", ColumnType: "bigint unsigned"},
	{Name: "created_at", DataType: "datetime", ColumnType: "datetime"},
}

func TestBuildScopeAndButtons(t *testing.T) {
	// 含 org_id + status: 默认开数据权限, 7 个按钮 (含 batch-status)
	m, err := Build(BuildOptions{Table: "biz_order", Mod: "order", Title: "订单管理", Inputs: orderInputs, MenuMaxId: 999})
	if err != nil {
		t.Fatal(err)
	}
	if !m.HasScope {
		t.Error("含 org_id 列且未传 NoScope, HasScope 应为 true")
	}
	if !m.HasStatusCol || m.StatusCol == nil || m.StatusCol.Name != "status" {
		t.Error("应识别 status 列为批量状态列")
	}
	if len(m.MenuIds) != 8 || m.MenuIds[0] != 1000 {
		t.Errorf("MenuIds 应为 菜单+7按钮 共 8 个, got %v", m.MenuIds)
	}
	if m.Btns.Import != m.ButtonBase+6 || m.Btns.Export != m.ButtonBase+5 || m.Btns.BatchStatus != m.ButtonBase+4 {
		t.Errorf("按钮 ID 布局异常: %+v", m.Btns)
	}
	if len(m.RelApis) != 1 || m.RelApis[0] != "user" {
		t.Errorf("显式关联应收集 user, got %v", m.RelApis)
	}

	// NoScope: 强制关闭
	m2, err := Build(BuildOptions{Table: "biz_order", Mod: "order", Title: "订单管理", Inputs: orderInputs, MenuMaxId: 999, NoScope: true})
	if err != nil {
		t.Fatal(err)
	}
	if m2.HasScope {
		t.Error("NoScope=true 时 HasScope 应为 false")
	}

	// 无 org_id 且无状态列: 不开 scope, 无 batch-status (6 个按钮)
	plain := []ColumnInput{orderInputs[1], orderInputs[2], orderInputs[4], orderInputs[7]}
	m3, err := Build(BuildOptions{Table: "biz_notice", Mod: "notice", Title: "公告", Inputs: plain, MenuMaxId: 999})
	if err != nil {
		t.Fatal(err)
	}
	if m3.HasScope {
		t.Error("无 org_id 列时 HasScope 应为 false")
	}
	if m3.HasStatusCol || m3.Btns.BatchStatus != 0 {
		t.Error("无状态列不应生成 batch-status 按钮")
	}
	if len(m.MenuIds) != len(m3.MenuIds)+1 {
		t.Errorf("无状态列应少 1 个按钮 ID, got %d vs %d", len(m3.MenuIds), len(m.MenuIds))
	}
}

func TestParseSQLDecimalColumn(t *testing.T) {
	// 回归: DECIMAL(12,2) 类型内含逗号, 旧正则会整列丢失
	const ddl = "CREATE TABLE IF NOT EXISTS `biz_x` (\n  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n  `amount` DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '金额',\n  `title` VARCHAR(64) NOT NULL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB;"
	inputs, _, err := ParseSQL(ddl, "biz_x")
	if err != nil {
		t.Fatal(err)
	}
	var hasAmount bool
	for _, in := range inputs {
		if in.Name == "amount" {
			hasAmount = true
			if in.Comment != "金额" {
				t.Errorf("amount 注释解析错误: %q", in.Comment)
			}
		}
	}
	if !hasAmount {
		t.Fatal("DECIMAL(12,2) 列被丢弃 (逗号截断回归)")
	}
}

// TestRenderSeedSQLIdempotent 锁定种子 SQL 的幂等块: 菜单 DELETE+INSERT 清 ID 块,
// API INSERT IGNORE (依赖 sys_api uk_path_method), 可重复执行; 批量/导入导出按钮与 API 齐全。
func TestRenderSeedSQLIdempotent(t *testing.T) {
	m, err := Build(BuildOptions{Table: "biz_order", Mod: "order", Title: "订单管理", Inputs: orderInputs, MenuMaxId: 999})
	if err != nil {
		t.Fatal(err)
	}
	files, err := Render(m)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	var sql string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".sql") {
			sql = f.Content
			break
		}
	}
	if sql == "" {
		t.Fatal("Render 未产出种子 SQL 文件")
	}
	// 菜单: DELETE 覆盖 菜单+全部按钮 ID 块 (含状态列时的 7 个按钮), 再 INSERT
	wantDelete := "DELETE FROM sys_menu WHERE id IN (1000, 10001, 10002, 10003, 10004, 10005, 10006, 10007);"
	if !strings.Contains(sql, wantDelete) {
		t.Errorf("菜单 ID 块清理缺失, want %q\ngot:\n%s", wantDelete, sql)
	}
	for _, perm := range []string{"order:order:batch-delete", "order:order:batch-status", "order:order:export", "order:order:import"} {
		if !strings.Contains(sql, perm) {
			t.Errorf("按钮权限 %s 缺失", perm)
		}
	}
	for _, api := range []string{"/api/v1/order/batch-delete", "/api/v1/order/batch-status", "/api/v1/order/export", "/api/v1/order/import-template", "/api/v1/order/import"} {
		if !strings.Contains(sql, api) {
			t.Errorf("API 种子 %s 缺失", api)
		}
	}
	if strings.Contains(sql, "INSERT INTO sys_menu") && strings.Contains(sql, "INSERT IGNORE INTO sys_menu") {
		t.Error("sys_menu 不应使用 INSERT IGNORE (无自然唯一键, 需 DELETE+INSERT 覆盖旧值)")
	}
	// API: INSERT IGNORE 幂等
	if !strings.Contains(sql, "INSERT IGNORE INTO sys_api") {
		t.Errorf("sys_api 应使用 INSERT IGNORE 依赖 uk_path_method 幂等\ngot:\n%s", sql)
	}
}

func TestApplyKindOverride(t *testing.T) {
	for in, want := range map[string]string{
		"input":  "input",
		"date":   "date",
		"":       "input", // 空=不覆盖
		"hack":   "input", // 白名单外忽略
		"select": "input", // 非法类型忽略
	} {
		if got := applyKindOverride("input", in); got != want {
			t.Fatalf("applyKindOverride(input, %q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildKindRequiredOverride(t *testing.T) {
	t1 := true
	inputs := []ColumnInput{
		{Name: "title", DataType: "varchar", ColumnType: "varchar(128)", Comment: "标题"},
		{Name: "owner_id", DataType: "bigint", ColumnType: "bigint unsigned", Comment: "负责人(关联:post)"},
		{Name: "meet_time", DataType: "datetime", ColumnType: "datetime", Comment: "时间"},
	}
	// title 改 textarea + 必填; owner_id 关联被覆盖成 number (relation 附带字段应清空)
	inputs[0].Kind = "textarea"
	inputs[0].Required = &t1
	inputs[1].Kind = "number"
	m, err := Build(BuildOptions{Table: "biz_demo2", Mod: "demo2", Title: "演示二", Inputs: inputs, MenuMaxId: 999})
	if err != nil {
		t.Fatal(err)
	}
	var title, owner, meet *Column
	for i := range m.Columns {
		switch m.Columns[i].Name {
		case "title":
			title = &m.Columns[i]
		case "owner_id":
			owner = &m.Columns[i]
		case "meet_time":
			meet = &m.Columns[i]
		}
	}
	if title == nil || title.Kind != "textarea" || !title.Required {
		t.Errorf("title 覆盖失效: %+v", title)
	}
	if owner == nil || owner.Kind != "number" || owner.RelMod != "" || owner.RelGuess {
		t.Errorf("owner_id 覆盖为 number 应清空 relation 附带字段: %+v", owner)
	}
	if meet == nil || meet.Kind != "date" {
		t.Errorf("datetime 应推导 date: %+v", meet)
	}

	// 无显式必填时: 首个 input 列兜底必填
	inputs[0].Required = nil
	inputs[0].Kind = ""
	m2, err := Build(BuildOptions{Table: "biz_demo2", Mod: "demo2", Title: "演示二", Inputs: inputs, MenuMaxId: 999})
	if err != nil {
		t.Fatal(err)
	}
	if !m2.Columns[0].Required {
		t.Error("无显式必填时首列应兜底必填")
	}
}
