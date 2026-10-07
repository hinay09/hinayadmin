// Package crudgen 离线 CRUD 代码生成核心 (CLI 与网页端共用)。
//
// 输入: 表的列元数据 (CLI 从 init.sql DDL 解析, 网页端从 information_schema 查询)
// 输出: 前后端完整增删改查文件集 (内存 map), 可写入磁盘 (CLI/直写) 或打包 zip 下载。
//
// 约定: 目标表需含 id 主键与 created_at/updated_at/deleted_at;
// 审计字段 create_id/update_id 由 ormfill 自动填充。
package crudgen

import (
	"archive/zip"
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// Column 单列元数据。
type Column struct {
	Name    string // snake_case 列名
	GoName  string // PascalCase
	GoType  string // Go 类型
	Comment string // 中文说明
	Kind    string // 表单类型: input/textarea/number/switch
}

// ColumnInput 列输入 (DDL 解析或 information_schema 查询结果 + 可选前端覆盖)。
type ColumnInput struct {
	Name       string
	DataType   string // SQL 类型 (varchar/bigint/...)
	ColumnType string // 完整类型定义 (含 unsigned 等)
	Comment    string
	InList     *bool // 覆盖: 是否进列表 (默认前 8 列)
	InForm     *bool // 覆盖: 是否进表单 (默认全部, switch 置尾)
	InSearch   *bool // 覆盖: 是否进关键词搜索 (默认字符串列)
}

// Model 渲染数据。
type Model struct {
	Table  string
	Entity string
	Mod    string
	Title  string
	Tag    string

	Columns     []Column // 全部业务列
	StringCols  []Column
	ListColumns []Column
	FormColumns []Column
	FirstGoName string

	MenuId     uint64
	ButtonBase uint64
	UpgradeSeq string
	PermPrefix string
}

// BuildOptions 模型构建参数。
type BuildOptions struct {
	Table      string
	Mod        string // 为空时自动推导
	Title      string // 为空时用表名
	Inputs     []ColumnInput
	MenuMaxId  uint64 // 已占用最大菜单 ID (CLI 解析 SQL / 网页查 DB)
	UpgradeSeq string // 为空时按时间戳兜底
}

// GeneratedFile 单个生成文件。
type GeneratedFile struct {
	Path    string // 相对根路径; "web:" 前缀路由到前端目录
	Content string
	IsGo    bool
}

// ---------------------------------------------------------------------------
// 命名与类型推导
// ---------------------------------------------------------------------------

// Pascal 下划线命名 → 大驼峰 (如 biz_leave → BizLeave)。
func Pascal(s string) string {
	out := ""
	for _, p := range strings.Split(s, "_") {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	return out
}

// camel 首字母转小写 (大驼峰 → 小驼峰, 其余原样)。
func camel(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// GoTypeOf SQL 类型 -> Go 类型 (导出供网页端列信息展示)。
func GoTypeOf(sqlType, columnType string) string { return goType(sqlType, columnType) }

// KindOf 列名/类型 -> 表单控件类型 (导出供网页端列信息展示)。
func KindOf(name, sqlType string) string { return formKind(name, sqlType) }

// goType SQL 类型 → Go 类型 (无符号 BIGINT → uint64, 其余整型归并 int, 时间 → *gtime.Time)。
func goType(sqlType, columnType string) string {
	unsigned := strings.Contains(strings.ToUpper(columnType), "UNSIGNED")
	switch strings.ToUpper(sqlType) {
	case "BIGINT":
		if unsigned {
			return "uint64"
		}
		return "int64"
	case "INT", "TINYINT", "SMALLINT", "MEDIUMINT":
		return "int"
	case "DECIMAL", "FLOAT", "DOUBLE":
		return "float64"
	case "DATETIME", "TIMESTAMP", "DATE":
		return "*gtime.Time"
	default:
		return "string"
	}
}

// formKind 按列名/类型启发式推导表单控件: 状态列 → 开关, 备注类 → 多行文本,
// 外键/整型 → 数字输入, 其余默认单行输入。
func formKind(name, sqlType string) string {
	switch {
	case name == "status" || strings.HasSuffix(name, "_status"):
		return "switch"
	case strings.Contains(name, "remark") || strings.Contains(name, "content") ||
		strings.Contains(name, "description") || name == "desc":
		return "textarea"
	case strings.HasSuffix(name, "_id") || strings.HasPrefix(strings.ToUpper(sqlType), "BIGINT") ||
		strings.HasPrefix(strings.ToUpper(sqlType), "INT") || strings.HasPrefix(strings.ToUpper(sqlType), "TINYINT"):
		return "number"
	default:
		return "input"
	}
}

// SkipColumn 脚手架约定列 (主键/时间戳/审计/软删), 不进业务代码。
func SkipColumn(name string) bool {
	switch name {
	case "id", "created_at", "updated_at", "deleted_at", "create_id", "update_id":
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// 模型构建
// ---------------------------------------------------------------------------

// reModName 模块名白名单: 小写字母开头, 仅小写字母/数字/下划线, 长度 2-31。
// 安全约束: mod 会拼进生成文件路径与 import 语句, 必须拒绝路径穿越类字符。
var reModName = regexp.MustCompile(`^[a-z][a-z0-9_]{1,30}$`)

// ValidateModAndTitle 校验外部输入的模块名与标题。
// title 会进入生成的 SQL/代码文本, 拒绝引号/反斜杠/换行等破坏性与注入类字符。
func ValidateModAndTitle(mod, title string) error {
	if mod != "" && !reModName.MatchString(mod) {
		return fmt.Errorf("模块名不合法: 仅允许小写字母开头的 小写字母/数字/下划线, 长度 2-31 (收到 %q)", mod)
	}
	if r := []rune(title); len(r) > 32 {
		return fmt.Errorf("中文标题过长 (最多 32 字符)")
	}
	for _, c := range title {
		if c == '\'' || c == '"' || c == '\\' || c == '\n' || c == '\r' || c < 0x20 {
			return fmt.Errorf("中文标题含非法字符 (引号/反斜杠/换行/控制字符)")
		}
	}
	return nil
}

// Build 由列输入构建渲染模型。
func Build(opts BuildOptions) (*Model, error) {
	if err := ValidateModAndTitle(opts.Mod, opts.Title); err != nil {
		return nil, err
	}
	entity := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(opts.Table, "biz_"), "sys_"), "s")
	mod := opts.Mod
	if mod == "" {
		mod = entity
	}
	title := opts.Title
	if title == "" {
		title = opts.Table
	}

	cols := make([]Column, 0, len(opts.Inputs))
	for _, in := range opts.Inputs {
		if SkipColumn(in.Name) {
			continue
		}
		cols = append(cols, Column{
			Name:    in.Name,
			GoName:  Pascal(in.Name),
			GoType:  goType(in.DataType, in.ColumnType),
			Comment: in.Comment,
			Kind:    formKind(in.Name, in.DataType),
		})
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("表 %s 未解析出业务列", opts.Table)
	}

	menuId := (opts.MenuMaxId/1000 + 1) * 1000
	seq := opts.UpgradeSeq
	if seq == "" {
		seq = fmt.Sprintf("9%03d", int(os.Getpid()%1000))
	}

	m := &Model{
		Table: opts.Table, Entity: Pascal(entity), Mod: mod,
		Title: title, Tag: Pascal(mod),
		Columns: cols, MenuId: menuId, ButtonBase: menuId*10 + 1,
		UpgradeSeq: seq, PermPrefix: mod + ":" + entity,
	}

	var switches []Column
	listCount := 0
	for _, c := range cols {
		if c.Kind == "switch" {
			switches = append(switches, c)
		} else {
			m.FormColumns = append(m.FormColumns, c)
			if m.FirstGoName == "" && c.Kind == "input" {
				m.FirstGoName = c.GoName
			}
		}
		if c.GoType == "string" {
			m.StringCols = append(m.StringCols, c)
		}
		listCount++
		if listCount <= 8 {
			m.ListColumns = append(m.ListColumns, c)
		}
	}
	m.FormColumns = append(m.FormColumns, switches...)
	if m.FirstGoName == "" {
		m.FirstGoName = cols[0].GoName
	}

	// 应用前端覆盖 (inList/inForm/inSearch)
	m.applyOverrides(opts.Inputs)
	return m, nil
}

// applyOverrides 按列名应用 UI 勾选覆盖 (列表/表单/搜索)。
func (m *Model) applyOverrides(inputs []ColumnInput) {
	byName := map[string]ColumnInput{}
	for _, in := range inputs {
		byName[in.Name] = in
	}

	// 列表覆盖
	if hasOverride(inputs, func(in ColumnInput) bool { return in.InList != nil }) {
		list := make([]Column, 0, 8)
		for _, c := range m.Columns {
			if in, ok := byName[c.Name]; ok && in.InList != nil && *in.InList {
				list = append(list, c)
			}
		}
		m.ListColumns = list
	}
	// 表单覆盖
	if hasOverride(inputs, func(in ColumnInput) bool { return in.InForm != nil }) {
		var form, sw []Column
		for _, c := range m.Columns {
			in, ok := byName[c.Name]
			on := !ok || in.InForm == nil || *in.InForm
			if !on {
				continue
			}
			if c.Kind == "switch" {
				sw = append(sw, c)
			} else {
				form = append(form, c)
			}
		}
		m.FormColumns = append(form, sw...)
		if m.FirstGoName == "" || !containsCol(m.FormColumns, m.FirstGoName) {
			m.FirstGoName = ""
			for _, c := range m.FormColumns {
				if c.Kind == "input" {
					m.FirstGoName = c.GoName
					break
				}
			}
			if m.FirstGoName == "" && len(m.FormColumns) > 0 {
				m.FirstGoName = m.FormColumns[0].GoName
			}
		}
	}
	// 搜索覆盖
	if hasOverride(inputs, func(in ColumnInput) bool { return in.InSearch != nil }) {
		sc := make([]Column, 0, 4)
		for _, c := range m.Columns {
			if c.GoType != "string" {
				continue
			}
			if in, ok := byName[c.Name]; ok && in.InSearch != nil && !*in.InSearch {
				continue
			}
			sc = append(sc, c)
		}
		m.StringCols = sc
	}
}

// hasOverride 用户列输入中是否存在满足条件的列 (生成前覆盖检查)。
func hasOverride(inputs []ColumnInput, f func(ColumnInput) bool) bool {
	for _, in := range inputs {
		if f(in) {
			return true
		}
	}
	return false
}

// containsCol 已解析列中是否含指定 Go 字段名 (重名冲突检查)。
func containsCol(cols []Column, goName string) bool {
	for _, c := range cols {
		if c.GoName == goName {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// DDL 解析 (CLI 路径)
// ---------------------------------------------------------------------------

var (
	reCreateStart = regexp.MustCompile("(?i)CREATE\\s+TABLE\\s+(?:IF\\s+NOT\\s+EXISTS\\s+)?`([\\w]+)`\\s*\\(")
	reColumn      = regexp.MustCompile("^`([\\w]+)`\\s+([A-Za-z]+)([^,]*),?$")
	reComment     = regexp.MustCompile("COMMENT\\s+'((?:[^']|'')*)'")
	reMenuRow     = regexp.MustCompile(`(?m)^\s*\((\d+),\s*\d+,`)
)

// ParseSQL 从 SQL 文本解析指定表的列输入与已占用最大菜单 ID。
func ParseSQL(sqlText, table string) ([]ColumnInput, uint64, error) {
	for _, loc := range reCreateStart.FindAllStringSubmatchIndex(sqlText, -1) {
		if sqlText[loc[2]:loc[3]] != table {
			continue
		}
		rest := sqlText[loc[1]:]
		end := strings.Index(rest, ") ENGINE")
		if end < 0 {
			return nil, 0, fmt.Errorf("表 %s 的 DDL 块不完整", table)
		}
		inputs := make([]ColumnInput, 0)
		for _, line := range strings.Split(rest[:end], "\n") {
			m := reColumn.FindStringSubmatch(strings.TrimSpace(line))
			if m == nil {
				continue
			}
			name, sqlType, attr := m[1], m[2], m[3]
			comment := ""
			if cm := reComment.FindStringSubmatch(attr); cm != nil {
				comment = strings.ReplaceAll(cm[1], "''", "'")
			}
			inputs = append(inputs, ColumnInput{
				Name: name, DataType: sqlType, ColumnType: sqlType + " " + attr, Comment: comment,
			})
		}
		return inputs, maxMenuId(sqlText), nil
	}
	return nil, 0, fmt.Errorf("SQL 中未找到表 %s 的 CREATE TABLE 语句", table)
}

// maxMenuId 扫描 SQL 种子中的菜单 INSERT 行, 取已占用最大菜单 ID (新菜单 ID 从其后排)。
func maxMenuId(sqlText string) uint64 {
	max := uint64(0)
	for _, m := range reMenuRow.FindAllStringSubmatch(sqlText, -1) {
		if v, err := strconv.ParseUint(m[1], 10, 64); err == nil && v > max {
			max = v
		}
	}
	return max
}

// NextUpgradeSeq 扫描升级脚本目录取下一个序号, 目录不可读时返回空(调用方兜底)。
func NextUpgradeSeq(upgradeDir string) string {
	entries, err := os.ReadDir(upgradeDir)
	if err != nil {
		return ""
	}
	max := 0
	for _, e := range entries {
		name := e.Name()
		if len(name) >= 4 {
			if v, err := strconv.Atoi(name[:4]); err == nil && v > max {
				max = v
			}
		}
	}
	return fmt.Sprintf("%04d", max+1)
}

// ---------------------------------------------------------------------------
// 渲染与产出
// ---------------------------------------------------------------------------

var funcMap = template.FuncMap{"bt": func() string { return "`" }, "camel": camel, "add1": func(u uint64) uint64 { return u + 1 }}

// Render 渲染全部生成文件 (CLI 与网页端一致)。
func Render(m *Model) ([]GeneratedFile, error) {
	render := func(name, body string, data any) (string, error) {
		t, err := template.New(name).Funcs(funcMap).Parse(body)
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		if err = t.Execute(&sb, data); err != nil {
			return "", err
		}
		return sb.String(), nil
	}

	var files []GeneratedFile
	add := func(rel, content string, isGo bool) error {
		if isGo {
			src, err := format.Source([]byte(content))
			if err != nil {
				return fmt.Errorf("生成 %s 后 gofmt 失败: %w", rel, err)
			}
			content = string(src)
		}
		files = append(files, GeneratedFile{Path: rel, Content: content, IsGo: isGo})
		return nil
	}

	if s, err := render("api", apiTmpl, m); err == nil {
		if err := add(fmt.Sprintf("api/%s/v1/%s.go", m.Mod, m.Mod), s, true); err != nil {
			return nil, err
		}
	}
	if s, err := render("api-alias", apiAliasTmpl, m); err == nil {
		if err := add(fmt.Sprintf("api/%s/%s.go", m.Mod, m.Mod), s, true); err != nil {
			return nil, err
		}
	}
	if err := add(fmt.Sprintf("internal/controller/%s/%s.go", m.Mod, m.Mod), "package "+m.Mod+"\n", true); err != nil {
		return nil, err
	}
	if s, err := render("ctrl-new", ctrlNewTmpl, m); err == nil {
		if err := add(fmt.Sprintf("internal/controller/%s/%s_new.go", m.Mod, m.Mod), s, true); err != nil {
			return nil, err
		}
	}
	for _, act := range []string{"List", "Create", "Update", "Delete"} {
		s, err := render("ctrl-"+act, ctrlMethodTmpl, struct {
			Model
			Act string
		}{*m, act})
		if err != nil {
			return nil, err
		}
		if err := add(fmt.Sprintf("internal/controller/%s/%s_v1_%s.go", m.Mod, m.Mod, strings.ToLower(act)), s, true); err != nil {
			return nil, err
		}
	}
	for _, tpl := range []struct{ name, body, rel string }{
		{"svc", svcTmpl, fmt.Sprintf("internal/service/%s.go", m.Mod)},
		{"logic", logicTmpl, fmt.Sprintf("internal/logic/%s/%s.go", m.Mod, m.Mod)},
		{"model", modelTmpl, fmt.Sprintf("internal/model/%s.go", m.Table)},
		{"entity", entityTmpl, fmt.Sprintf("internal/model/entity/%s.go", m.Table)},
		{"do", doTmpl, fmt.Sprintf("internal/model/do/%s.go", m.Table)},
		{"dao", daoTmpl, fmt.Sprintf("internal/dao/%s.go", m.Table)},
		{"dao-int", daoInternalTmpl, fmt.Sprintf("internal/dao/internal/%s.go", m.Table)},
		{"sql", sqlTmpl, fmt.Sprintf("manifest/sql/gen/%s_gen_%s.sql", m.UpgradeSeq, m.Mod)},
	} {
		s, err := render(tpl.name, tpl.body, m)
		if err != nil {
			return nil, err
		}
		if err := add(tpl.rel, s, false); err != nil {
			return nil, err
		}
	}
	for _, tpl := range []struct{ name, body, rel string }{
		{"web-api", webApiTmpl, fmt.Sprintf("web:app/composables/useApi/%s.ts", m.Mod)},
		{"web-page", webPageTmpl, fmt.Sprintf("web:app/pages/%s/index.vue", m.Mod)},
	} {
		s, err := render(tpl.name, tpl.body, m)
		if err != nil {
			return nil, err
		}
		if err := add(tpl.rel, s, false); err != nil {
			return nil, err
		}
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// Zip 将文件集打包为 zip (供网页端下载), 内含 README 说明接线步骤。
func Zip(m *Model, files []GeneratedFile) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	readme := fmt.Sprintf(`%s 代码生成产物 (crudgen)
==============================

目录: server/ = Go 后端, web_src/ = Nuxt 前端

合并到项目后需要手动接线 3 处:
  1. server/internal/logic/logic.go      import 增加: _ "hinay.cn/admin/internal/logic/%s"
  2. server/internal/cmd/cmd.go          import 增加 controller/%s, sec.Bind 增加 %s.NewV1()
  3. web_src/app/composables/useApi/index.ts  增加: export * from './%s'

然后:
  - 执行 server/manifest/sql/gen/%s_gen_%s.sql (建表 DDL 需自行确认已在库中)
  - 角色管理为目标角色分配「%s」菜单与按钮权限
  - 重启后端 (go build / make run)
`, m.Title, m.Mod, m.Mod, m.Mod, m.Mod, m.UpgradeSeq, m.Mod, m.Title)
	write := func(name, content string) error {
		f, err := w.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(content))
		return err
	}
	if err := write("README.txt", readme); err != nil {
		return nil, err
	}
	for _, f := range files {
		name := filepath.ToSlash(f.Path)
		if strings.HasPrefix(name, "web:") {
			name = "web_src/" + strings.TrimPrefix(name, "web:")
		} else {
			name = "server/" + name
		}
		if err := write(name, f.Content); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ---------------------------------------------------------------------------
// 磁盘写入 (CLI / 网页直写)
// ---------------------------------------------------------------------------

// WriteAll 写入源码树并自动接线 (返回写入清单; 目标已存在时报错防覆盖)。
func WriteAll(serverRoot, webRoot string, m *Model, files []GeneratedFile) ([]string, error) {
	var written []string
	for _, f := range files {
		root, rel := serverRoot, f.Path
		if strings.HasPrefix(rel, "web:") {
			root, rel = webRoot, strings.TrimPrefix(rel, "web:")
		}
		full := filepath.Join(root, rel)
		if _, err := os.Stat(full); err == nil {
			return written, fmt.Errorf("目标已存在, 中止以防覆盖: %s", full)
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(full, []byte(f.Content), 0o644); err != nil {
			return written, err
		}
		written = append(written, full)
	}
	if err := patchFile(serverRoot, "internal/logic/logic.go", "import (\n",
		fmt.Sprintf("\t_ \"hinay.cn/admin/internal/logic/%s\"\n", m.Mod)); err != nil {
		return written, err
	}
	if err := patchFile(serverRoot, "internal/cmd/cmd.go", "import (\n",
		fmt.Sprintf("\t\"hinay.cn/admin/internal/controller/%s\"\n", m.Mod)); err != nil {
		return written, err
	}
	if err := patchFile(serverRoot, "internal/cmd/cmd.go", "\t\t\tsec.Bind(\n",
		fmt.Sprintf("\t\t\t\t%s.NewV1(),\n", m.Mod)); err != nil {
		return written, err
	}
	if err := patchFile(webRoot, "app/composables/useApi/index.ts", "export * from './auth'\n",
		fmt.Sprintf("export * from './%s'\n", m.Mod)); err != nil {
		return written, err
	}
	return written, nil
}

// patchFile 在锚点后插入文本 (已存在则跳过)。
func patchFile(root, rel, anchor, insert string) error {
	full := filepath.Join(root, rel)
	data, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	src := string(data)
	if strings.Contains(src, insert) {
		return nil
	}
	idx := strings.Index(src, anchor)
	if idx < 0 {
		return fmt.Errorf("patch 锚点未找到: %s", rel)
	}
	out := src[:idx+len(anchor)] + insert + src[idx+len(anchor):]
	return os.WriteFile(full, []byte(out), 0o644)
}
