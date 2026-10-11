// tenantorm_test.go —— 租户注入改写器全形态表驱动测试 (docs/multi-tenant.md §11 P1 验收项)。
//
// 改写器为纯函数, 可测性最好; 本文件固化全部允许形态与 fail-close 形态,
// 后续任何对 tenantorm.go 词法规则的改动都必须保持本组用例全绿。
package ormfill

import (
	"context"
	"strings"
	"testing"

	"github.com/gogf/gf/v2/database/gdb"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/utility/tenantx"
)

// tenantCtx 构造带注入租户的测试 ctx (绕过 contextx 避免引入 casbinx 依赖链)。
func tenantCtx(tid uint64) context.Context {
	tenantx.SetEnabledForTest(true)
	return context.WithValue(context.Background(), consts.CtxTenantKey,
		&model.TenantInfo{Id: tid, Code: "t", Name: "t", Inject: true})
}

// plainCtx 无租户上下文。
func plainCtx() context.Context {
	tenantx.SetEnabledForTest(true)
	return context.Background()
}

func TestTenantInjectSelectAllowed(t *testing.T) {
	cases := []struct {
		name     string
		sql      string
		args     []any
		tid      uint64
		want     string // 期望改写后的 SQL (含 /*tenant:N*/ 前缀)
		wantArgs []any
	}{
		{
			name: "单表带WHERE",
			sql:  "SELECT * FROM sys_user WHERE id=?",
			args: []any{uint64(5)}, tid: 3,
			want:     "/*tenant:3*/ SELECT * FROM sys_user WHERE `tenant_id`=? AND id=?",
			wantArgs: []any{uint64(3), uint64(5)},
		},
		{
			name: "无WHERE串尾追加",
			sql:  "SELECT COUNT(*) FROM sys_file",
			tid:  2,
			want: "/*tenant:2*/ SELECT COUNT(*) FROM sys_file WHERE `tenant_id`=?",
		},
		{
			name: "分页LIMIT前插入",
			sql:  "SELECT id FROM biz_leave ORDER BY id DESC LIMIT 10,20",
			tid:  7,
			want: "/*tenant:7*/ SELECT id FROM biz_leave WHERE `tenant_id`=? ORDER BY id DESC LIMIT 10,20",
		},
		{
			name: "GROUP BY前插入",
			sql:  "SELECT status, COUNT(*) AS c FROM wf_task GROUP BY status",
			tid:  9,
			want: "/*tenant:9*/ SELECT status, COUNT(*) AS c FROM wf_task WHERE `tenant_id`=? GROUP BY status",
		},
		{
			name: "主表别名",
			sql:  "SELECT u.id, u.username FROM sys_user AS u WHERE u.status=?",
			args: []any{1}, tid: 4,
			want:     "/*tenant:4*/ SELECT u.id, u.username FROM sys_user AS u WHERE `tenant_id`=? AND u.status=?",
			wantArgs: []any{uint64(4), 1},
		},
		{
			name: "小写关键词与反引号表名",
			sql:  "select `id` from `sys_config` where `config_key`=?",
			args: []any{"sys.name"}, tid: 6,
			want:     "/*tenant:6*/ select `id` from `sys_config` where `tenant_id`=? AND `config_key`=?",
			wantArgs: []any{uint64(6), "sys.name"},
		},
		{
			name: "库名前缀",
			sql:  "SELECT id FROM `hinay_admin`.`sys_org` WHERE parent_id=?",
			args: []any{0}, tid: 5,
			want:     "/*tenant:5*/ SELECT id FROM `hinay_admin`.`sys_org` WHERE `tenant_id`=? AND parent_id=?",
			wantArgs: []any{uint64(5), 0},
		},
		{
			name: "JOIN C类表(已读关系)主表注入",
			sql:  "SELECT m.* FROM biz_message m LEFT JOIN biz_message_read r ON (r.message_id = m.id AND r.user_id = 5) WHERE m.deleted_at IS NULL AND m.status=1",
			tid:  8,
			want: "/*tenant:8*/ SELECT m.* FROM biz_message m LEFT JOIN biz_message_read r ON (r.message_id = m.id AND r.user_id = 5) WHERE `tenant_id`=? AND m.deleted_at IS NULL AND m.status=1",
		},
		{
			name: "WHERE内EXISTS非隔离表放行注入",
			sql:  "SELECT m.* FROM biz_message m WHERE m.deleted_at IS NULL AND EXISTS (SELECT 1 FROM biz_message_target t WHERE t.message_id=m.id)",
			tid:  8,
			want: "/*tenant:8*/ SELECT m.* FROM biz_message m WHERE `tenant_id`=? AND m.deleted_at IS NULL AND EXISTS (SELECT 1 FROM biz_message_target t WHERE t.message_id=m.id)",
		},
		{
			name: "OR条件与软删组合",
			sql:  "SELECT * FROM sys_user WHERE deleted_at IS NULL AND (nickname LIKE ? OR username LIKE ?) ORDER BY id DESC",
			args: []any{"%a%", "%b%"}, tid: 11,
			want:     "/*tenant:11*/ SELECT * FROM sys_user WHERE `tenant_id`=? AND deleted_at IS NULL AND (nickname LIKE ? OR username LIKE ?) ORDER BY id DESC",
			wantArgs: []any{uint64(11), "%a%", "%b%"},
		},
		{
			name: "字面量内含WHERE字样不漂移",
			sql:  "SELECT * FROM sys_config WHERE config_key='why WHERE matters' AND status=1",
			tid:  2,
			want: "/*tenant:2*/ SELECT * FROM sys_config WHERE `tenant_id`=? AND config_key='why WHERE matters' AND status=1",
		},
		{
			name: "字面量内含隔离表名不算出现",
			sql:  "SELECT * FROM sys_config WHERE config_key='sys_user demo' LIMIT 1",
			tid:  2,
			want: "/*tenant:2*/ SELECT * FROM sys_config WHERE `tenant_id`=? AND config_key='sys_user demo' LIMIT 1",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, gotArgs, changed, err := tenantInjectSelect(c.sql, c.args, c.tid)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !changed {
				t.Fatalf("expected rewrite, got pass-through")
			}
			if got != c.want {
				t.Fatalf("sql mismatch:\n got:  %s\n want: %s", got, c.want)
			}
			// 租户值插入 args 头部: gotArgs[0] 恒为 tid, 其余与原 args 一致
			if len(gotArgs) != len(c.args)+1 || gotArgs[0] != c.tid {
				t.Fatalf("args mismatch: got %v, want [%v]+%v", gotArgs, c.tid, c.args)
			}
			for i := range c.args {
				if gotArgs[i+1] != c.args[i] {
					t.Fatalf("args[%d] mismatch: got %v want %v", i+1, gotArgs[i+1], c.args[i])
				}
			}
		})
	}
}

func TestTenantInjectSelectPassThrough(t *testing.T) {
	cases := []struct {
		name string
		sql  string
	}{
		{"非白名单表 information_schema", "SELECT TABLE_NAME AS tname FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME"},
		{"无FROM无隔离表", "SELECT VERSION()"},
		{"SHOW元数据含隔离表名", "SHOW CREATE TABLE sys_user"},
		{"SHOW GLOBAL STATUS", "SHOW GLOBAL STATUS"},
		{"B类共享表 sys_menu", "SELECT id FROM sys_menu WHERE status=1"},
		{"B类共享表 casbin_rule", "SELECT v1 FROM casbin_rule WHERE ptype='p'"},
		{"C类跟随表 sys_user_totp", "SELECT secret FROM sys_user_totp WHERE user_id=1"},
		{"SET语句", "SET NAMES utf8mb4"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _, changed, err := tenantInjectSelect(c.sql, nil, 3)
			if err != nil {
				t.Fatalf("unexpected fail-close: %v", err)
			}
			if changed || got != c.sql {
				t.Fatalf("expected pass-through, got %s", got)
			}
		})
	}
}

func TestTenantInjectSelectFailClose(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want string // 错误信息包含的关键字
	}{
		{"WITH开头CTE", "WITH x AS (SELECT 1) SELECT * FROM sys_user", "首关键词"},
		{"EXPLAIN", "EXPLAIN SELECT * FROM sys_user", "首关键词"},
		{"UNION分支", "SELECT id FROM sys_user UNION SELECT id FROM sys_role", "UNION"},
		{"语句含分号", "SELECT * FROM sys_user WHERE id=1; ", "分号"},
		{"IN子查询引用隔离表", "SELECT * FROM sys_user WHERE org_id IN (SELECT id FROM sys_org WHERE status=1)", "sys_org"},
		{"EXISTS子查询引用隔离表", "SELECT * FROM sys_user u WHERE EXISTS (SELECT 1 FROM sys_role r WHERE r.id=u.id)", "sys_role"},
		{"派生表", "SELECT t.* FROM (SELECT * FROM sys_job WHERE status=1) t", "派生表"},
		{"A-join-A", "SELECT u.id FROM sys_user u LEFT JOIN sys_role r ON r.id=u.org_id WHERE u.id=1", "sys_role"},
		{"FROM逗号多表", "SELECT * FROM sys_user u, sys_org o WHERE u.org_id=o.id", "多表"},
		{"主表非隔离表但引用隔离表", "SELECT u.id FROM casbin_rule c LEFT JOIN sys_user u ON u.id=c.v0", "主表"},
		{"字段区占位符", "SELECT ? AS x FROM sys_config", "占位符"},
		{"字段区标量子查询", "SELECT (SELECT COUNT(*) FROM sys_user) AS c FROM sys_org", "sys_user"},
		{"引号未闭合", "SELECT * FROM sys_user WHERE name='abc", "未闭合"},
		{"A类表自连接", "SELECT a.id FROM sys_user a JOIN sys_user b ON a.id=b.id", "sys_user"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, _, err := tenantInjectSelect(c.sql, nil, 3)
			if err == nil {
				t.Fatalf("expected fail-close error, got pass")
			}
			if c.want != "" && !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), c.want)
			}
		})
	}
}

func TestTenantCondUpdate(t *testing.T) {
	ctx := tenantCtx(7)
	// 常规追加 (condition 保留前导空格形态, gdb 以 "SET %s%s" 直接拼接)
	cond, args, err := tenantCondUpdate(ctx, "`sys_user`", " WHERE (`id`=?)", []any{5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cond != " WHERE (`id`=?) AND `tenant_id` = ?" || len(args) != 2 || args[1] != uint64(7) {
		t.Fatalf("cond=%q args=%v", cond, args)
	}
	// 空条件 (防御路径): 自带 WHERE 前缀
	cond, args, err = tenantCondUpdate(ctx, "sys_file", "", nil)
	if err != nil || cond != " WHERE `tenant_id` = ?" || len(args) != 1 {
		t.Fatalf("cond=%q args=%v err=%v", cond, args, err)
	}
	// 业务手写 tenant_id 条件 → fail-close
	if _, _, err = tenantCondUpdate(ctx, "sys_user", "tenant_id=2 OR id=?", nil); err == nil {
		t.Fatalf("expected fail-close on hand-written tenant_id condition")
	}
	// 非白名单表不动
	cond, args, _ = tenantCondUpdate(ctx, "sys_menu", "id=?", []any{1})
	if cond != "id=?" || len(args) != 1 {
		t.Fatalf("non-whitelist table should be untouched: %q %v", cond, args)
	}
	// 无租户 ctx 不动
	cond, _, _ = tenantCondUpdate(plainCtx(), "sys_user", "id=?", []any{1})
	if cond != "id=?" {
		t.Fatalf("no-tenant ctx should be untouched: %q", cond)
	}
}

func TestTenantFillInsert(t *testing.T) {
	ctx := tenantCtx(9)
	list := gdb.List{{"username": "a", "tenant_id": uint64(1)}, {"username": "b"}}
	tenantFillInsert(ctx, "sys_user", list)
	if list[0]["tenant_id"] != uint64(9) || list[1]["tenant_id"] != uint64(9) {
		t.Fatalf("tenant_id should be unconditionally overwritten from ctx: %+v", list)
	}
	// 非白名单表不注入
	list2 := gdb.List{{"name": "x"}}
	tenantFillInsert(ctx, "sys_menu", list2)
	if _, ok := list2[0]["tenant_id"]; ok {
		t.Fatalf("non-whitelist table should not be injected")
	}
	// 无租户 ctx 不覆盖
	list3 := gdb.List{{"name": "x", "tenant_id": uint64(123)}}
	tenantFillInsert(plainCtx(), "sys_user", list3)
	if list3[0]["tenant_id"] != uint64(123) {
		t.Fatalf("platform ctx should keep explicit tenant_id (租户创建事务语义)")
	}
}

func TestMaskLiterals(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{`WHERE a='x' AND b="y"`, `WHERE a=' ' AND b=" "`, true},
		{`WHERE a='it''s' `, `WHERE a='     ' `, true},
		{`WHERE a='a\'b'`, `WHERE a='    '`, true},
		{"SELECT 1", "SELECT 1", true},
		{`WHERE a='abc`, "", false},
		{`WHERE a='x\`, "", false},
	}
	for _, c := range cases {
		got, err := maskLiterals(c.in)
		if c.ok && (err != nil || got != c.want) {
			t.Fatalf("maskLiterals(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
		if !c.ok && err == nil {
			t.Fatalf("maskLiterals(%q) should fail", c.in)
		}
	}
}

func TestInjectTenantSwitch(t *testing.T) {
	// 开关关闭 → 一律不注入 (单租户兼容)
	tenantx.SetEnabledForTest(false)
	disabledCtx := context.WithValue(context.Background(), consts.CtxTenantKey,
		&model.TenantInfo{Id: 3, Code: "t", Inject: true})
	if _, ok := injectTenant(disabledCtx); ok {
		t.Fatalf("disabled switch must disable injection")
	}
	tenantx.SetEnabledForTest(true)
	// Inject=false (平台超管未切换) → 不注入
	ctx := context.WithValue(context.Background(), consts.CtxTenantKey,
		&model.TenantInfo{Id: 1, Code: "platform", Inject: false})
	if _, ok := injectTenant(ctx); ok {
		t.Fatalf("platform admin (un-switched) must not be injected")
	}
	// 无租户 ctx (job 未构造) → 不注入 (后续白名单写 fail-close 兜底)
	if _, ok := injectTenant(context.Background()); ok {
		t.Fatalf("no-tenant ctx must not be injected")
	}
}
