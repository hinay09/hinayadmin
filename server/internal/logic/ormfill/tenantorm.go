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
//     WHERE 条件整体包裹 (原条件) AND tenant_id=? (租户条件在括号外, 顶层 OR
//     分支不漏过滤) + 顶层 JOIN (LEFT/INNER, 隔离表须别名化) 逐表 ON 包裹注入 +
//     gdb 分页 COUNT 包装 (SELECT COUNT(1) FROM (…) alias) 递归改写内层 +
//     其余括号内出现白名单表一律 fail-close (宁拒勿漏);
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
	"sys_access_log":  {},
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
// 返回改写后的 sql 与 args; changed=false 表示无需改写 (原样放行);
// err != nil 表示 fail-close (拒绝执行)。
//
// 支持形态: 顶层单表 + 顶层 JOIN 序列 (gdb 结构化 LeftJoin/InnerJoin 生成的
// "JOIN `tbl` [AS ]`alias` ON (cond)" 形态, 隔离表对端逐表在 ON 括号内追加
// 别名限定租户条件 —— LEFT 保持外连接语义不丢主表行, INNER 等价于 WHERE 过滤),
// 以及 gdb 对 GROUP BY 模型分页计数的固定包装 SELECT COUNT(1) FROM (…) alias
// (递归改写内层)。其余形态 (子查询/派生表/UNION/逗号多表/RIGHT JOIN 隔离表等)
// 一律 fail-close。
func tenantInjectSelect(sql string, args []any, tid uint64) (outSQL string, outArgs []any, changed bool, err error) {
	return tenantInjectSelectDepth(sql, args, tid, 0)
}

const maxWrapperDepth = 2 // COUNT 包装递归上限 (gdb 只产一层, 手写嵌套按深度熔断)

func tenantInjectSelectDepth(sql string, args []any, tid uint64, depth int) (outSQL string, outArgs []any, changed bool, err error) {
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

	// gdb 分页 COUNT 包装 (GROUP BY 模型): SELECT COUNT(1) FROM (<inner>) alias ——
	// 外层不引用任何表, 隔离只能落在内层 → 递归改写内层, 内层拒则整体拒。
	if inStart, inEnd, ok := unwrapCountWrapper(masked); ok {
		if depth >= maxWrapperDepth {
			return "", nil, false, errTenantRefused("COUNT 包装嵌套超深 (非 gdb 生成形态)")
		}
		innerOut, innerArgs, innerChanged, ierr := tenantInjectSelectDepth(sql[inStart:inEnd], args, tid, depth+1)
		if ierr != nil {
			return "", nil, false, ierr
		}
		if !innerChanged {
			return sql, args, false, nil
		}
		out := "/*tenant:" + uitoa(tid) + "*/ " + sql[:inStart] + innerOut + sql[inEnd:]
		return out, innerArgs, true, nil
	}

	// 顶层 FROM 解析主表 (去反引号/库名前缀, 保留别名)
	fromPos := keywordAtDepth0(masked, 0, "FROM")
	if fromPos < 0 {
		return "", nil, false, errTenantRefused("未找到顶层 FROM 子句")
	}
	main, perr := parseTableRef(masked, fromPos+4)
	if perr != nil {
		return "", nil, false, errTenantRefused(perr.Error())
	}
	if _, ok := tenantTables[main.name]; !ok {
		return "", nil, false, errTenantRefused("主表 " + main.name + " 非租户隔离表但语句引用了隔离表 (JOIN/子查询对端)")
	}

	// FROM 区域边界 (WHERE / 尾部子句关键词之前) 与逗号多表 fail-close
	wherePos := keywordAtDepth0(masked, main.compEnd, "WHERE")
	tailPos := earliestTailKeyword(masked, main.compEnd)
	regionEnd := len(masked)
	if wherePos >= 0 && wherePos < regionEnd {
		regionEnd = wherePos
	}
	if tailPos >= 0 && tailPos < regionEnd {
		regionEnd = tailPos
	}
	if commaAtDepth0(masked, main.next, regionEnd) {
		return "", nil, false, errTenantRefused("FROM 多表逗号形式")
	}

	// 顶层 JOIN 序列解析 (gdb 结构化形态)
	joins, jerr := parseFromJoins(masked, main.next, regionEnd)
	if jerr != nil {
		return "", nil, false, errTenantRefused(jerr.Error())
	}
	for i := range joins {
		jr := &joins[i]
		_, jr.isolated = tenantTables[jr.ref.name]
		if !jr.isolated {
			continue
		}
		// 隔离表 JOIN 的可注入形态校验: 只支持 LEFT/INNER + 显式 ON + 别名
		// (RIGHT JOIN 由对端表驱动行可见性、CROSS/无 ON/USING 无注入点、
		//  无别名时 ON 只能以裸表名引用会撞白名单出现校验)。
		switch jr.jtype {
		case "LEFT", "INNER":
		default:
			return "", nil, false, errTenantRefused("隔离表 " + jr.ref.name + " 为 " + jr.jtype + " JOIN, 无安全注入点 (仅支持 LEFT/INNER JOIN)")
		}
		if jr.usingClose >= 0 {
			return "", nil, false, errTenantRefused("隔离表 " + jr.ref.name + " 的 JOIN 使用 USING (请改用 ON 条件)")
		}
		if jr.onClose < 0 {
			return "", nil, false, errTenantRefused("隔离表 " + jr.ref.name + " 的 JOIN 缺少 ON 条件")
		}
		if jr.ref.alias == "" {
			return "", nil, false, errTenantRefused("隔离表 " + jr.ref.name + " 的 JOIN 缺少别名 (ON 以裸表名引用无法安全注入)")
		}
	}
	anyIsolatedJoin := false
	for i := range joins {
		if joins[i].isolated {
			anyIsolatedJoin = true
			break
		}
	}

	// 白名单表名出现位置校验: 每处出现必须是主表裸名或某个隔离 JOIN 表裸名本身
	// (防 EXISTS/IN 子查询 / 派生表 / UNION 分支 / 字段区 / ON 条件内裸表名引用漏注入)。
	for _, oc := range occurs {
		inMain := oc.pos >= main.compStart && oc.pos < main.compEnd
		inJoin := false
		for i := range joins {
			if joins[i].isolated && oc.pos >= joins[i].ref.compStart && oc.pos < joins[i].ref.compEnd {
				inJoin = true
				break
			}
		}
		if !inMain && !inJoin {
			return "", nil, false, errTenantRefused("白名单表 " + oc.name + " 出现在子查询/字段区/ON 条件 (仅允许顶层主表与 JOIN 表)")
		}
	}

	// 字段区 (FROM 之前) 占位符 fail-close: 注入值按位插入 args, 字段区占位子查询
	// 虽可按位拼接, 但属非 gdb 生成形态, 维持保守拒绝; JOIN ON 内占位符为 gdb 合法
	// 产物, 由按位拼接保证顺序。
	if strings.IndexByte(masked[:fromPos], '?') >= 0 {
		return "", nil, false, errTenantRefused("字段区存在占位符, 参数顺序无法保证")
	}
	// 占位符与参数数量必须一致 (按位拼接的前置不变量; gdb 生成语句天然满足,
	// 手写错配语句防越界, 交给 MySQL 也过不了)。
	if strings.Count(masked, "?") != len(args) {
		return "", nil, false, errTenantRefused("占位符与参数数量不一致, 无法安全注入")
	}

	// 注入点组装: 隔离 JOIN 逐表改写为 ON ((原条件) AND `alias`.`tenant_id`=?)
	// (整体加括号防 OR 分支漏隔离)。主表条件必须置于原 WHERE 条件整体之外 ——
	// gdb 的 WhereLike/WhereOrLike 混用链在顶层产 OR (实测关键词搜索列表页:
	// WHERE (code LIKE ?) OR (name LIKE ?) AND (type_id IN ...)), 若按头插
	// "tenant=? AND <条件>" 会因 AND 优先级高于 OR 留下未过滤的 OR 分支
	// (既有多租户隔离漏洞, 本轮 e2e 实测发现) —— 故 WHERE 分支包裹改写:
	// WHERE (<原条件>) AND `alias`.`tenant_id`=?; 无 WHERE 时注入条件为唯一
	// 条件, 无 OR 风险, 维持尾部子句前/串尾单点插入。
	// 无隔离 JOIN 对端时维持裸列形态, 有隔离 JOIN 对端时别名限定防列名歧义。
	mainQual := main.name
	if main.alias != "" {
		mainQual = main.alias
	}
	mainCol := "`" + tenantColumn + "`=?"
	if anyIsolatedJoin {
		mainCol = "`" + mainQual + "`.`" + tenantColumn + "`=?"
	}

	type injectPoint struct {
		pos  int    // 插入位置 (原始语句坐标)
		frag string // 可不含 ? (纯括号包裹点)
		k    int    // 该位置之前原有占位符个数 (args 插入下标基准)
	}
	points := make([]injectPoint, 0, 2*len(joins)+1)
	for i := range joins {
		if !joins[i].isolated {
			continue
		}
		q := joins[i].ref.alias
		// 包裹注入: ON (X) → ON ((X) AND `alias`.`tenant_id`=?)。
		// 不能直接在 X 尾部追加 AND —— gdb 原样透传业务 ON 串, X 含 OR 时
		// AND 优先级更高会留下未过租户的 OR 分支 (a=b OR c=d AND tenant ⇒
		// a=b OR (c=d AND tenant), a=b 分支漏隔离), 故整体加括号后再 AND。
		points = append(points,
			injectPoint{pos: joins[i].onOpen + 1, frag: "("},
			injectPoint{
				pos:  joins[i].onClose,
				frag: ") AND `" + q + "`.`" + tenantColumn + "`=?",
				k:    strings.Count(masked[:joins[i].onClose], "?"),
			},
		)
	}
	if wherePos >= 0 {
		tailPoint := len(sql)
		if tailPos >= 0 {
			tailPoint = tailPos
		}
		// WHERE (原条件整体) AND tenant=? —— 租户条件在括号外, 顶层 OR 分支全受控;
		// 尾部子句存在时补尾随空格 (原语句的空格落在包裹括号之前)
		frag := ") AND " + mainCol
		if tailPos >= 0 {
			frag += " "
		}
		points = append(points,
			injectPoint{pos: wherePos + len("WHERE"), frag: " ("},
			injectPoint{
				pos:  tailPoint,
				frag: frag,
				k:    strings.Count(masked[:tailPoint], "?"),
			},
		)
	} else if tailPos >= 0 {
		// 无 WHERE: 注入条件为唯一条件无 OR 风险, 尾部子句前单点插入
		// (前一字符必为空格, 无需再补前导空格)
		points = append(points, injectPoint{
			pos:  tailPos,
			frag: "WHERE " + mainCol + " ",
			k:    strings.Count(masked[:tailPos], "?"),
		})
	} else {
		points = append(points, injectPoint{
			pos:  len(sql),
			frag: " WHERE " + mainCol,
			k:    strings.Count(masked, "?"),
		})
	}

	// SQL 拼接: 注入点按构建序天然位置升序, 降序应用互不影响坐标
	out := sql
	for i := len(points) - 1; i >= 0; i-- {
		out = out[:points[i].pos] + points[i].frag + out[points[i].pos:]
	}

	// args 合并: 含 ? 的注入点按 (k, pos) 升序插入租户值 —— k 为插入点之前原有
	// 占位符个数, 同 k 时按语句位置排, 与最终 SQL 中占位符出现顺序一一对应
	// (覆盖 JOIN ON 自带占位符的语句); 纯括号包裹点不含 ? 不参与。
	newArgs := make([]any, 0, len(args)+len(joins)+1)
	oi := 0
	flush := func(upto int) {
		for ; oi < upto; oi++ {
			newArgs = append(newArgs, args[oi])
		}
	}
	for i := range points {
		if !strings.ContainsRune(points[i].frag, '?') {
			continue
		}
		flush(points[i].k)
		newArgs = append(newArgs, tid)
	}
	flush(len(args))

	// 观测归因: 语句头部追加 /*tenant:N*/ 注释, Slow Query Log / RDS 性能洞察
	// 即可按租户归因慢查询 (放行路径不加注释, 零开销)。
	out = "/*tenant:" + uitoa(tid) + "*/ " + out
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

// tableRef 解析后的表引用 (主表或 JOIN 对端)。
type tableRef struct {
	tok       string // 原始 token (可含 `db`.`tbl` 与反引号)
	name      string // 裸表名 (去反引号/库名前缀, 与 tenantTables 键同口径)
	compStart int    // 裸表名组件在语句中的起始位置 (含引导反引号)
	compEnd   int    // 裸表名组件结束位置 (含收尾反引号)
	alias     string // 别名 ("" 表示无)
	next      int    // 别名之后的位置
}

// fromClauseStopWords 表引用后不能视为别名的关键词 (MySQL 保留字本身也不能做裸别名)。
var fromClauseStopWords = map[string]struct{}{
	"WHERE": {}, "LEFT": {}, "RIGHT": {}, "INNER": {}, "CROSS": {}, "STRAIGHT": {},
	"JOIN": {}, "ON": {}, "USING": {}, "GROUP": {}, "ORDER": {}, "HAVING": {},
	"LIMIT": {}, "OFFSET": {}, "UNION": {}, "FOR": {}, "LOCK": {}, "WINDOW": {},
	"AND": {}, "OR": {}, "NOT": {}, "AS": {}, "SET": {}, "INTO": {},
}

// isSpaceMasked 空白字符 (遮蔽后语句内的排版空白)。
func isSpaceMasked(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

// readWordOrQuoted 在 i 处读取一个词 (裸标识符或 `反引号词`), 返回去除反引号的内容与结束位置。
// 非词首字符返回 ok=false (逗号/括号等结构符号)。
func readWordOrQuoted(masked string, i int) (word string, next int, ok bool) {
	if i >= len(masked) {
		return "", i, false
	}
	if masked[i] == '`' {
		j := i + 1
		for j < len(masked) && masked[j] != '`' {
			j++
		}
		if j >= len(masked) {
			return "", i, false // 反引号未闭合
		}
		return masked[i+1 : j], j + 1, true
	}
	if isIdentChar(masked[i]) {
		j := i
		for j < len(masked) && isIdentChar(masked[j]) {
			j++
		}
		return masked[i:j], j, true
	}
	return "", i, false
}

// parseTableRef 解析 FROM/JOIN 后的表引用: <token> [AS alias | alias]。
// token 支持反引号与库名前缀; 别名不被子句关键词误吞; '(' 开头 (派生表/子查询) 报错。
func parseTableRef(masked string, from int) (tableRef, error) {
	i := from
	for i < len(masked) && isSpaceMasked(masked[i]) {
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
		return tableRef{}, gerror.New("FROM/JOIN 后无表名 (派生表/子查询)")
	}
	tok := masked[start:i]
	compStart, compEnd := start, i
	if p := strings.LastIndex(tok, "."); p >= 0 {
		compStart = start + p + 1
	}
	ref := tableRef{tok: tok, name: normalizeTable(tok), compStart: compStart, compEnd: compEnd, next: i}

	j := i
	for j < len(masked) && isSpaceMasked(masked[j]) {
		j++
	}
	if j >= len(masked) {
		return ref, nil
	}
	if masked[j] != '`' && !isIdentChar(masked[j]) {
		return ref, nil // 结构符号 (逗号/括号等), 无别名
	}
	w, wEnd, ok := readWordOrQuoted(masked, j)
	if !ok {
		return tableRef{}, gerror.New("别名反引号未闭合")
	}
	if strings.EqualFold(w, "AS") {
		j = wEnd
		for j < len(masked) && isSpaceMasked(masked[j]) {
			j++
		}
		a, aEnd, aok := readWordOrQuoted(masked, j)
		if !aok {
			return tableRef{}, gerror.New("AS 后缺少别名")
		}
		ref.alias, ref.next = a, aEnd
		return ref, nil
	}
	if _, stop := fromClauseStopWords[strings.ToUpper(w)]; !stop {
		ref.alias, ref.next = w, wEnd
	}
	return ref, nil
}

// joinRef 顶层 JOIN 子句。
type joinRef struct {
	jtype      string // LEFT / INNER / RIGHT / CROSS / STRAIGHT (裸 JOIN 记 INNER)
	ref        tableRef
	onOpen     int // ON 条件左括号位置; -1 无 ON (gdb 结构化形态恒有括号)
	onClose    int // ON 条件右括号位置; -1 无 ON
	usingClose int // USING 列表右括号位置; -1 无 USING
	isolated   bool
}

// parseFromJoins 解析 FROM 区域 (主表之后) 的顶层 JOIN 序列 (gdb 结构化形态)。
// 仅接受 <TYPE> JOIN <表引用> [ON (cond) | USING (cols)]; JOIN 目标为子查询由
// parseTableRef 报错; 其他 token 一律 fail-close。
func parseFromJoins(masked string, from, regionEnd int) ([]joinRef, error) {
	var joins []joinRef
	i := from
	for {
		for i < len(masked) && isSpaceMasked(masked[i]) {
			i++
		}
		if i >= regionEnd || i >= len(masked) {
			return joins, nil
		}
		w, wEnd, ok := readWordOrQuoted(masked, i)
		if !ok || w == "" {
			return nil, gerror.New("FROM 子句无法解析 (仅允许 JOIN 序列)")
		}
		var jt string
		switch strings.ToUpper(w) {
		case "JOIN":
			jt = "INNER"
			i = wEnd
		case "LEFT", "RIGHT", "INNER", "CROSS", "STRAIGHT":
			j := wEnd
			for j < len(masked) && isSpaceMasked(masked[j]) {
				j++
			}
			w2, w2End, ok2 := readWordOrQuoted(masked, j)
			if !ok2 || !strings.EqualFold(w2, "JOIN") {
				return nil, gerror.New("JOIN 关键词形态无法解析")
			}
			jt = strings.ToUpper(w)
			i = w2End
		default:
			return nil, gerror.New("FROM 子句存在非 JOIN 形态 token (子查询/手写片段)")
		}
		ref, err := parseTableRef(masked, i)
		if err != nil {
			return nil, err
		}
		i = ref.next
		for i < len(masked) && isSpaceMasked(masked[i]) {
			i++
		}
		jr := joinRef{jtype: jt, ref: ref, onClose: -1, usingClose: -1}
		if w, wEnd, ok := readWordOrQuoted(masked, i); ok {
			switch strings.ToUpper(w) {
			case "ON":
				i = wEnd
				for i < len(masked) && isSpaceMasked(masked[i]) {
					i++
				}
				if i >= len(masked) || masked[i] != '(' {
					// gdb 结构化 JOIN 恒生成 ON (cond) 括号形态; 无括号必为手写 → fail-close
					return nil, gerror.New("ON 后缺少条件括号 (仅支持 gdb 结构化形态)")
				}
				cp := matchParen(masked, i)
				if cp < 0 {
					return nil, gerror.New("ON 条件括号不闭合")
				}
				jr.onOpen, jr.onClose = i, cp
				i = cp + 1
			case "USING":
				i = wEnd
				for i < len(masked) && isSpaceMasked(masked[i]) {
					i++
				}
				if i >= len(masked) || masked[i] != '(' {
					return nil, gerror.New("USING 后缺少列括号")
				}
				cp := matchParen(masked, i)
				if cp < 0 {
					return nil, gerror.New("USING 括号不闭合")
				}
				jr.usingClose = cp
				i = cp + 1
			}
		}
		joins = append(joins, jr)
	}
}

// matchParen 返回 open 处 '(' 的配对 ')' 位置; 不闭合返回 -1。
func matchParen(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// commaAtDepth0 检查 [from, end) 顶层 (括号深度 0) 是否出现逗号多表。
func commaAtDepth0(masked string, from, end int) bool {
	if end > len(masked) {
		end = len(masked)
	}
	depth := 0
	for i := from; i < end; i++ {
		switch masked[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				return true
			}
		}
	}
	return false
}

// unwrapCountWrapper 识别 gdb 对 GROUP BY 模型分页计数的固定包装
// (gdb_model_select.go): SELECT COUNT(1) FROM (<inner>) alias —— 外层除包装外
// 不引用任何表, 隔离条件安全落点唯一 (内层)。命中返回 inner 起止位置;
// 不匹配返回 false, 交由常规门禁 (其余派生表形态仍 fail-close)。
func unwrapCountWrapper(masked string) (innerStart, innerEnd int, ok bool) {
	i := 0
	readKw := func(w string) bool {
		for i < len(masked) && isSpaceMasked(masked[i]) {
			i++
		}
		if i+len(w) > len(masked) || !strings.EqualFold(masked[i:i+len(w)], w) {
			return false
		}
		if i+len(w) < len(masked) && isIdentChar(masked[i+len(w)]) {
			return false
		}
		i += len(w)
		return true
	}
	skipWs := func() {
		for i < len(masked) && isSpaceMasked(masked[i]) {
			i++
		}
	}
	if !readKw("SELECT") || !readKw("COUNT") {
		return 0, 0, false
	}
	skipWs()
	if i >= len(masked) || masked[i] != '(' {
		return 0, 0, false
	}
	i++
	skipWs()
	switch {
	case i < len(masked) && masked[i] == '1':
		i++
	case i < len(masked) && masked[i] == '*':
		i++
	default:
		return 0, 0, false
	}
	skipWs()
	if i >= len(masked) || masked[i] != ')' {
		return 0, 0, false
	}
	i++
	if !readKw("FROM") {
		return 0, 0, false
	}
	skipWs()
	if i >= len(masked) || masked[i] != '(' {
		return 0, 0, false
	}
	open := i
	closePos := matchParen(masked, open)
	if closePos < 0 {
		return 0, 0, false
	}
	// 尾部仅允许一个可选别名标识符
	j := closePos + 1
	skipWsAt := func(at int) int {
		for at < len(masked) && isSpaceMasked(masked[at]) {
			at++
		}
		return at
	}
	j = skipWsAt(j)
	if _, wEnd, wok := readWordOrQuoted(masked, j); wok {
		j = skipWsAt(wEnd)
	}
	if j != len(masked) {
		return 0, 0, false
	}
	if firstKeyword(masked[open+1:closePos]) != "SELECT" {
		return 0, 0, false
	}
	return open + 1, closePos, true
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
