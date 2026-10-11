// tenantorm.go —— 租户行级隔离注入 (docs/multi-tenant.md §3.2 定稿实现)。
//
// 与审计字段填充 (ormfill.go) 同层同风格: 在同一个 Driver 上重写驱动回调,
// SELECT/裸查询/事务全链路收口于 DoQuery, 写路径收口于 DoInsert/DoUpdate/DoDelete,
// 手写 DML (tx.Exec / DB().Exec) 收口于 DoExec 并 fail-close。
//
// 注入规则:
//   - ctx 有租户且 Inject=true (普通租户用户 / 平台超管代运维切换后) → 注入该租户;
//   - INSERT 无条件覆盖载荷中的 tenant_id (租户归属只信 ctx, 不信载荷);
//   - UPDATE/DELETE 在 condition 末尾追加 AND `tenant_id` = ? (软删转 UPDATE 天然覆盖);
//   - SELECT 为"受控改写": 白名单表 + 首关键词门禁 + 字面量遮蔽 + 顶层定位 +
//     括号内出现白名单表一律 fail-close (宁拒勿漏);
//   - 平台超管未切换 (Inject=false)、非白名单表 → 原样放行;
//   - 开关关闭 → 恒注入平台租户 (关闭 = 全部限定平台租户): 关闭前遗留的他租户
//     行不可见、其用户不可登录, 跨租户同名用户名在 tenant_id=1 下天然唯一不再歧义。
//
// fail-close 总则: 白名单表 + 需要注入 (开启态租户语境或关闭态平台语境) 但无法安全注入
// → 返回错误拒绝执行。
// 新业务表带 tenant_id 列须在 tenantTables 登记 (AGENTS.md 同步更新)。
package ormfill

import (
	"context"
	"database/sql"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/utility/tenantx"
)

// tenantTables A 类租户隔离表白名单 (均有 tenant_id 列)。
// 与 fillTables 同模式维护; sys_tenant 本身无 tenant_id 列 (平台表, API 层守卫),
// sys_tenant_menu 为显式 tenant_id 行不走注入, 两者均不在本表内。
var tenantTables = map[string]struct{}{
	"sys_user":        {},
	"sys_org":         {},
	"sys_role":        {},
	"sys_role_org":    {},
	"sys_post":        {},
	"sys_user_post":   {},
	"sys_config":      {},
	"sys_file":        {},
	"sys_audit_log":   {},
	"sys_login_log":   {},
	"sys_job":         {},
	"sys_job_log":     {},
	"wf_definition":   {},
	"wf_instance":     {},
	"wf_task":         {},
	"wf_record":       {},
	"biz_leave":       {},
	"biz_message":     {},
	"ai_conversation": {},
	"ai_chat_message": {},
	"cms_category":    {},
	"cms_article":     {},
}

const tenantColumn = "tenant_id"

// injectTenant 返回当前 ctx 的注入目标租户。
//   - 开启态: ctx 有租户且 Inject=true (普通租户用户 / 平台超管代运维切换后) → 注入该租户;
//     平台超管未切换 (Inject=false) → 放行;
//   - 关闭态: 恒注入平台租户 —— 关闭即"全部限定平台租户", 隔离兜底不因关闸而失守
//     (遗留他租户行不可见/不可写, 其用户登录查询被限定 tenant_id=1 后不再歧义)。
func injectTenant(ctx context.Context) (uint64, bool) {
	if ctx == nil || !tenantx.Enabled(ctx) {
		return consts.TenantPlatformId, true
	}
	if v := ctx.Value(consts.CtxTenantKey); v != nil {
		if t, ok := v.(*model.TenantInfo); ok && t != nil && t.Id > 0 && t.Inject {
			return t.Id, true
		}
	}
	return 0, false
}

// errTenantRefused fail-close 错误 (含语句形态说明, 引导开发者改走 ORM 结构化方法)。
func errTenantRefused(reason string) error {
	return gerror.Newf("多租户注入保护: %s —— 该 SQL 无法安全注入租户条件已拒绝执行, 请改用 dao 结构化方法或拆分语句", reason)
}

// ---------------------------------------------------------------------------
// SELECT 受控改写 (纯函数, 表驱动测试见 tenantorm_test.go)
// ---------------------------------------------------------------------------

// tenantInjectSelect 对最终 SELECT SQL 做保守租户条件改写。
// 返回改写后的 sql 与 args (租户值插入 args 头部); changed=false 表示无需改写 (原样放行);
// err != nil 表示 fail-close (拒绝执行)。
func tenantInjectSelect(sql string, args []any, tid uint64) (outSQL string, outArgs []any, changed bool, err error) {
	masked, merr := maskLiterals(sql)
	if merr != nil {
		// 引号未闭合等无法分析的形态: 保守拒绝 (该语句本身也过不了 MySQL 解析)
		return "", nil, false, errTenantRefused("SQL 字面量引号未闭合, 无法安全分析")
	}

	// 白名单表出现扫描: 无出现 → 无论何种形态均原样放行 (information_schema/VERSION 等)
	occurs := whitelistOccurrences(masked)
	if len(occurs) == 0 {
		return sql, args, false, nil
	}

	// 首关键词门禁: SELECT 进入改写; SHOW/DESC/DESCRIBE/SET 为元数据放行;
	// 其余首关键词 (WITH/EXPLAIN 等, gf ORM 不生成, 必为手写) 一律 fail-close。
	head := firstKeyword(masked)
	switch head {
	case "SELECT":
	case "SHOW", "DESC", "DESCRIBE", "SET":
		return sql, args, false, nil
	default:
		return "", nil, false, errTenantRefused("首关键词 " + head + " 不在允许改写范围 (CTE/EXPLAIN 等手写语句)")
	}

	// 异常形态 fail-close: UNION (任意位置, 含子查询内) / 语句内分号
	if indexWord(masked, "UNION", 0) >= 0 {
		return "", nil, false, errTenantRefused("语句含 UNION")
	}
	if strings.IndexByte(masked, ';') >= 0 {
		return "", nil, false, errTenantRefused("语句含分号")
	}

	// 顶层 FROM 解析主表 (去反引号/库名前缀/别名)
	fromPos := keywordAtDepth0(masked, 0, "FROM")
	if fromPos < 0 {
		return "", nil, false, errTenantRefused("未找到顶层 FROM 子句")
	}
	tok, nameStart, nameEnd, next, perr := parseFromTable(masked, fromPos+4)
	if perr != nil {
		return "", nil, false, errTenantRefused(perr.Error())
	}
	mainTable := normalizeTable(tok)
	if _, ok := tenantTables[mainTable]; !ok {
		return "", nil, false, errTenantRefused("主表 " + mainTable + " 非租户隔离表但语句引用了隔离表 (JOIN/子查询对端)")
	}

	// 白名单表名唯一出现校验: 每处出现必须恰好是顶层主表裸名本身
	// (防 A-join-A / EXISTS/IN 子查询 / 派生表 / UNION 分支 / 字段区引用漏注入)。
	for _, oc := range occurs {
		if oc.pos < nameStart || oc.pos >= nameEnd || oc.name != mainTable {
			return "", nil, false, errTenantRefused("白名单表 " + oc.name + " 出现在 JOIN/子查询/字段区 (仅允许顶层主表)")
		}
	}

	// 注入点定位 (括号深度 0, 主表 token 之后)
	var injectAt int
	var frag string
	if wherePos := keywordAtDepth0(masked, next, "WHERE"); wherePos >= 0 {
		injectAt = wherePos + len("WHERE")
		frag = " `" + tenantColumn + "`=? AND"
	} else if kwPos := earliestTailKeyword(masked, next); kwPos >= 0 {
		injectAt = kwPos // 前一字符必为空格, 无需再补前导空格
		frag = "WHERE `" + tenantColumn + "`=? "
	} else {
		injectAt = len(sql)
		frag = " WHERE `" + tenantColumn + "`=?"
	}

	// 参数顺序校验: 注入点之前 (含 SELECT 字段区) 的 ? 计数必须为 0,
	// 否则说明字段区带占位子查询, 租户值不能安全地插到 args 头部 → fail-close。
	if strings.Count(masked[:injectAt], "?") != 0 {
		return "", nil, false, errTenantRefused("注入点之前存在占位符 (字段区带占位子查询), 参数顺序无法保证")
	}

	out := sql[:injectAt] + frag + sql[injectAt:]
	// 观测归因: 语句头部追加 /*tenant:N*/ 注释, Slow Query Log / RDS 性能洞察
	// 即可按租户归因慢查询 (放行路径不加注释, 零开销)。
	out = "/*tenant:" + uitoa(tid) + "*/ " + out
	newArgs := make([]any, 0, len(args)+1)
	newArgs = append(newArgs, tid)
	newArgs = append(newArgs, args...)
	return out, newArgs, true, nil
}

// ---------------------------------------------------------------------------
// 驱动回调重写 (DoInsert/DoUpdate 在 ormfill.go 中叠加审计填充后调用本包钩子)
// ---------------------------------------------------------------------------

// DoQuery SELECT 全链路收口 (Model 查询/裸查询/事务)。
func (d *Driver) DoQuery(ctx context.Context, link gdb.Link, sql string, args ...any) (gdb.Result, error) {
	if tid, ok := injectTenant(ctx); ok {
		newSQL, newArgs, _, err := tenantInjectSelect(sql, args, tid)
		if err != nil {
			g.Log().Errorf(ctx, "tenantorm refuse: %v | sql=%.200s", err, sql)
			return nil, err
		}
		sql, args = newSQL, newArgs
	}
	return d.Driver.DoQuery(ctx, link, sql, args...)
}

// structuredExecKey 标记"本语句由 DoInsert/DoUpdate/DoDelete 结构化路径生成"。
// 背景 (2026-10-11 e2e 实测发现): gdb 的 Core.DoInsert/DoUpdate/DoDelete 在组装
// 最终 SQL 后同样汇入 DoExec 执行 —— DoExec 不能无差别 fail-close, 否则全部
// ORM 结构化写在租户语境下被误杀。打标后的内部汇聚语句放行 (租户注入已在
// DoInsert/DoUpdate/DoDelete 层完成), 未打标的外部裸调用 (tx.Exec / DB().Exec)
// 才进入 fail-close 判定。
type structuredExecKey struct{}

func withStructuredExec(ctx context.Context) context.Context {
	return context.WithValue(ctx, structuredExecKey{}, true)
}

func fromStructuredExec(ctx context.Context) bool {
	v, _ := ctx.Value(structuredExecKey{}).(bool)
	return v
}

// DoExec 手写语句收口: 租户注入语境下语句引用白名单表 → fail-close
// (INSERT…SELECT / REPLACE / ON DUPLICATE KEY / 裸 UPDATE/DELETE 无安全注入点,
// 唯一合法出路是改走 ORM 结构化方法; 平台 ctx 放行, 租户创建等事务依赖此语义)。
func (d *Driver) DoExec(ctx context.Context, link gdb.Link, sql string, args ...any) (sql.Result, error) {
	if !fromStructuredExec(ctx) {
		if tid, ok := injectTenant(ctx); ok {
			masked, merr := maskLiterals(sql)
			if merr != nil {
				return nil, errTenantRefused("SQL 字面量引号未闭合, 无法安全分析")
			}
			if len(whitelistOccurrences(masked)) > 0 {
				rerr := errTenantRefused("租户上下文禁止手写 DML 引用隔离表")
				g.Log().Errorf(ctx, "tenantorm refuse exec (tenant=%d): %v | sql=%.200s", tid, rerr, sql)
				return nil, rerr
			}
		}
	}
	return d.Driver.DoExec(ctx, link, sql, args...)
}

// DoDelete 硬删路径注入 (软删在 Model 层已转 UPDATE, 走 DoUpdate)。
func (d *Driver) DoDelete(ctx context.Context, link gdb.Link, table string, condition string, args ...any) (sql.Result, error) {
	if cond, newArgs, err := tenantCondUpdate(ctx, table, condition, args); err != nil {
		return nil, err
	} else if cond != condition || len(newArgs) != len(args) {
		condition, args = cond, newArgs
	}
	return d.Driver.DoDelete(withStructuredExec(ctx), link, table, condition, args...)
}

// tenantFillInsert INSERT 注入钩子: 白名单表每条 record 无条件写 tenant_id
// (与 create_id 同点叠加; 开启态平台语境不注入时业务显式写入的 tenant_id 生效,
// 关闭态恒定覆盖为平台租户 1)。
func tenantFillInsert(ctx context.Context, table string, list gdb.List) {
	if tid, ok := injectTenant(ctx); ok {
		if _, whitelisted := tenantTables[normalizeTable(table)]; whitelisted {
			for _, record := range list {
				record[tenantColumn] = tid
			}
		}
	}
}

// tenantCondUpdate UPDATE/DELETE 注入钩子: condition 追加 AND `tenant_id`=? (args 末尾追加)。
// condition 含 tenant_id token → fail-close: 租户条件只由注入器管理, 业务代码禁止手写。
// 注意 (2026-10-11 e2e 实测): gdb DoUpdate 以 "UPDATE %s SET %s%s" 直接拼接,
// condition 约定带前导空格 (形如 " WHERE (...)") —— 此处不得 TrimSpace,
// 空条件时须自带 " WHERE " 前缀。
func tenantCondUpdate(ctx context.Context, table, condition string, args []any) (string, []any, error) {
	tid, ok := injectTenant(ctx)
	if !ok {
		return condition, args, nil
	}
	if _, whitelisted := tenantTables[normalizeTable(table)]; !whitelisted {
		return condition, args, nil
	}
	if masked, merr := maskLiterals(condition); merr == nil && indexWord(masked, tenantColumn, 0) >= 0 {
		return "", nil, errTenantRefused("业务代码禁止手写 tenant_id 条件 (租户条件仅由注入器管理)")
	}
	if strings.TrimSpace(condition) == "" {
		return " WHERE `" + tenantColumn + "` = ?", append(args, tid), nil
	}
	return condition + " AND `" + tenantColumn + "` = ?", append(args, tid), nil
}

// ---------------------------------------------------------------------------
// 词法工具 (遮蔽/关键词定位/表名解析) —— 纯函数, tenantorm_test.go 单测覆盖
// ---------------------------------------------------------------------------

// maskLiterals 将 '...' 与 "..." 字面量内容替换为等长空格 (保留引号本身),
// 防字面量内出现 WHERE/表名等字样导致定位漂移; 引号未闭合返回错误。
// 支持 SQL 转义: 同引号双写 (” / "") 与反斜杠转义下一字符。
func maskLiterals(s string) (string, error) {
	b := []byte(s)
	var quote byte
	for i := 0; i < len(b); i++ {
		c := b[i]
		if quote == 0 {
			if c == '\'' || c == '"' {
				quote = c
			}
			continue
		}
		switch c {
		case '\\':
			if i+1 >= len(b) {
				return "", gerror.New("unterminated escape")
			}
			b[i], b[i+1] = ' ', ' '
			i++
		case quote:
			if i+1 < len(b) && b[i+1] == quote { // '' / "" 转义
				b[i], b[i+1] = ' ', ' '
				i++
			} else {
				quote = 0 // 字面量闭合
			}
		default:
			b[i] = ' '
		}
	}
	if quote != 0 {
		return "", gerror.New("unterminated literal")
	}
	return string(b), nil
}

type tableOccurrence struct {
	name string
	pos  int
}

// whitelistOccurrences 在遮蔽后的语句中扫描全部白名单表名出现位置 (词边界匹配)。
func whitelistOccurrences(masked string) []tableOccurrence {
	var out []tableOccurrence
	for name := range tenantTables {
		start := 0
		for {
			idx := indexWord(masked, name, start)
			if idx < 0 {
				break
			}
			out = append(out, tableOccurrence{name: name, pos: idx})
			start = idx + len(name)
		}
	}
	return out
}

// isIdentChar SQL 标识符字符 (字母/数字/下划线/$)。
func isIdentChar(c byte) bool {
	return c == '_' || c == '$' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// indexWord 词边界子串查找 (前后均非标识符字符才算命中, 大小写不敏感)。
func indexWord(s, word string, from int) int {
	if len(word) == 0 || from >= len(s) {
		return -1
	}
	for i := from; i+len(word) <= len(s); i++ {
		if (s[i] | 0x20) != (word[0] | 0x20) {
			continue
		}
		if !strings.EqualFold(s[i:i+len(word)], word) {
			continue
		}
		if i > 0 && isIdentChar(s[i-1]) {
			continue
		}
		if i+len(word) < len(s) && isIdentChar(s[i+len(word)]) {
			continue
		}
		return i
	}
	return -1
}

// firstKeyword 语句首关键词 (大写)。
func firstKeyword(s string) string {
	s = strings.TrimLeft(s, " \t\r\n(")
	i := 0
	for i < len(s) && isIdentChar(s[i]) {
		i++
	}
	return strings.ToUpper(s[:i])
}

// keywordAtDepth0 在 [from, len) 范围内查找首个位于括号深度 0 的词边界关键词, 返回起始位置。
func keywordAtDepth0(masked string, from int, word string) int {
	depth := 0
	for i := 0; i < len(masked); i++ {
		switch masked[i] {
		case '(':
			depth++
			continue
		case ')':
			depth--
			continue
		}
		if i < from || depth != 0 {
			continue
		}
		if i > 0 && isIdentChar(masked[i-1]) {
			continue
		}
		if i+len(word) > len(masked) || !strings.EqualFold(masked[i:i+len(word)], word) {
			continue
		}
		if i+len(word) < len(masked) && isIdentChar(masked[i+len(word)]) {
			continue
		}
		return i
	}
	return -1
}

// tailKeywords 尾部子句关键词 (注入点兜底定位): 无 WHERE 时在其前插入 WHERE。
var tailKeywords = []string{
	"ORDER BY", "GROUP BY", "HAVING", "LIMIT",
	"FOR UPDATE", "FOR SHARE", "LOCK IN SHARE MODE", "PROCEDURE",
}

// earliestTailKeyword 在 [from, len) 顶层查找最早出现的尾部子句关键词, 返回位置。
func earliestTailKeyword(masked string, from int) int {
	best := -1
	for _, w := range tailKeywords {
		if p := keywordAtDepth0(masked, from, w); p >= 0 && (best < 0 || p < best) {
			best = p
		}
	}
	return best
}

// parseFromTable 解析 FROM 后的主表 token。
// 返回: 原始 token (可含 `db`.`tbl`)、裸表名在语句中的起止位置、token 结束位置、
// 错误 (逗号多表 / 派生表)。
func parseFromTable(masked string, from int) (tok string, nameStart, nameEnd, next int, err error) {
	i := from
	for i < len(masked) && (masked[i] == ' ' || masked[i] == '\t' || masked[i] == '\n' || masked[i] == '\r') {
		i++
	}
	start := i
	for i < len(masked) {
		c := masked[i]
		if c == '`' || c == '.' || isIdentChar(c) {
			i++
			continue
		}
		break
	}
	if i == start {
		return "", 0, 0, 0, gerror.New("FROM 后无表名 (派生表/子查询)")
	}
	tok = masked[start:i]
	// 裸表名位置: 去库名前缀
	ns, ne := start, i
	if p := strings.LastIndex(tok, "."); p >= 0 {
		ns = start + p + 1
	}
	// 跳过可选别名 (AS x / x) 后探测逗号多表
	j := i
	skipSpaces := func() {
		for j < len(masked) && (masked[j] == ' ' || masked[j] == '\t' || masked[j] == '\r' || masked[j] == '\n') {
			j++
		}
	}
	skipIdent := func() {
		for j < len(masked) && isIdentChar(masked[j]) {
			j++
		}
	}
	skipSpaces()
	if j < len(masked) && isIdentChar(masked[j]) {
		if j+2 < len(masked) && strings.EqualFold(masked[j:j+2], "AS") && !isIdentChar(masked[j+2]) {
			j += 2
			skipSpaces()
		}
		skipIdent()
		skipSpaces()
	}
	if j < len(masked) && masked[j] == ',' {
		return "", 0, 0, 0, gerror.New("FROM 多表逗号形式")
	}
	return tok, ns, ne, i, nil
}

// uitoa 十进制无符号整数转字符串。
func uitoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
