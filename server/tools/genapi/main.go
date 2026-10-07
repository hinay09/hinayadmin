// genapi 扫描 api/ 目录下所有 g.Meta 路由声明, 自动生成 sys_api 种子 SQL。
//
// 背景: CRUD 代码生成器 (tools/crudgen) 产出的升级 SQL 自带 sys_api 行,
// 但**手写**的 API 契约 (如审批流的加签/减签) 此前需要手工编写 INSERT 行,
// 路径/方法/描述容易抄漏或漂移。本工具直接从 g.Meta 标签推导, 保证与代码一致。
//
// 用法 (在 server/ 下):
//
//	go run ./tools/genapi                 # 全量路由 → stdout (INSERT IGNORE, 幂等)
//	go run ./tools/genapi -pkg flow       # 只扫描 api/flow 模块
//	go run ./tools/genapi -out x.sql      # 写入文件 (stdout 同时打印)
//	go run ./tools/genapi -check          # 对照 manifest/sql 种子, 报告尚未入库的路由 (退出码 1)
//	make gen-api-sql [PKG=flow] [OUT=xx.sql] [CHECK=1]
//
// 路径参数风格沿用 g.Meta 声明的 `{id}` 花括号 (与 flow 组一致);
// Casbin 匹配器已同时支持 `:id` 与 `{id}` (见 config.yaml casbin.model)。
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// tags → sys_api.group_name 的默认映射 (可被 -map Tag=分组 覆盖/扩充)。
var defaultGroupNames = map[string]string{
	"Flow":           "审批中心",
	"Message":        "消息通知",
	"Post":           "岗位管理",
	"Auth":           "认证",
	"AI":             "AI助手",
	"SystemUser":     "用户管理",
	"SystemRole":     "角色管理",
	"SystemMenu":     "菜单管理",
	"SystemOrg":      "组织机构",
	"SystemDict":     "字典管理",
	"SystemDictType": "字典管理",
	"SystemConfig":   "全局配置",
	"SystemFile":     "文件管理",
	"SystemAuditLog": "操作日志",
	"SystemLoginLog": "登录日志",
	"SystemOnline":   "在线用户",
	"SystemJob":      "定时任务",
	"SystemGencode":  "代码生成",
}

// route 一条 g.Meta 路由声明。
type route struct {
	path, method, group, desc, file string
}

// normPathParam 统一路径参数风格用于比对: `:id` 与 `{id}` 视为等价。
func normPathParam(p string) string {
	p = regexp.MustCompile(`:[^/]+`).ReplaceAllString(p, "{p}")
	p = regexp.MustCompile(`\{[^/]+\}`).ReplaceAllString(p, "{p}")
	return p
}

func main() {
	var (
		pkg    = flag.String("pkg", "", "只扫描 api/<pkg> 模块 (空=全部)")
		prefix = flag.String("prefix", "/api/v1", "路由前缀 (与 cmd.go 的 s.Group 一致)")
		out    = flag.String("out", "", "写入 SQL 文件 (空=仅 stdout)")
		check  = flag.Bool("check", false, "对照 manifest/sql 种子报告缺失路由, 不生成 SQL")
		maps   = flag.String("map", "", "覆盖/扩充 tags→分组名映射, 逗号分隔, 如: Flow=审批中心,New=新模块")
	)
	flag.Parse()

	groupNames := map[string]string{}
	for k, v := range defaultGroupNames {
		groupNames[k] = v
	}
	if *maps != "" {
		for _, kv := range strings.Split(*maps, ",") {
			if p := strings.SplitN(strings.TrimSpace(kv), "=", 2); len(p) == 2 && p[0] != "" {
				groupNames[p[0]] = p[1]
			}
		}
	}

	root := findServerRoot()
	apiDir := filepath.Join(root, "api")
	if *pkg != "" {
		apiDir = filepath.Join(root, "api", *pkg)
		if st, err := os.Stat(apiDir); err != nil || !st.IsDir() {
			fmt.Printf("错误: 模块目录不存在: %s\n", apiDir)
			os.Exit(1)
		}
	}

	routes := collectRoutes(apiDir, *prefix, groupNames)
	if len(routes) == 0 {
		fmt.Println("错误: 未扫描到任何 g.Meta 路由 (请确认在 server/ 下执行)")
		os.Exit(1)
	}

	if *check {
		missing := checkMissing(root, routes)
		if len(missing) == 0 {
			fmt.Printf("✓ 全部 %d 条路由均已存在于 manifest/sql 种子中\n", len(routes))
			return
		}
		fmt.Printf("发现 %d 条路由尚未写入任何种子 SQL (手写 API 请补种, 生成后可直接执行):\n", len(missing))
		for _, r := range missing {
			fmt.Printf("  %-4s %-46s %s (%s)\n", r.method, r.path, r.desc, r.file)
		}
		hint := "go run ./tools/genapi"
		if *pkg != "" {
			hint += " -pkg " + *pkg
		}
		fmt.Printf("\n生成: %s\n", hint)
		os.Exit(1)
	}

	sql := renderSQL(routes)
	fmt.Print(sql)
	if *out != "" {
		if err := os.WriteFile(*out, []byte(sql), 0o644); err != nil {
			fmt.Printf("错误: 写入文件失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n→ 已写入 %s (%d 条路由)\n", *out, len(routes))
	}
}

// collectRoutes 递归解析 api 目录下所有 Go 文件, 提取内嵌 g.Meta 字段的路由声明。
func collectRoutes(dir, prefix string, groupNames map[string]string) []route {
	tagRe := regexp.MustCompile(`(\w+):"([^"]*)"`)
	var routes []route
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range st.Fields.List {
					sel, ok := field.Type.(*ast.SelectorExpr) // g.Meta (内嵌字段, Names 为空)
					if !ok || sel.Sel.Name != "Meta" {
						continue
					}
					if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "g" {
						continue
					}
					lit := field.Tag
					if lit == nil || lit.Kind != token.STRING {
						continue
					}
					meta := map[string]string{}
					for _, m := range tagRe.FindAllStringSubmatch(strings.Trim(lit.Value, "`"), -1) {
						meta[m[1]] = m[2]
					}
					p, method := meta["path"], strings.ToUpper(meta["method"])
					if p == "" || method == "" || !strings.HasPrefix(p, "/") {
						continue
					}
					if !strings.HasPrefix(p, "/api/") {
						p = prefix + p
					}
					tag := meta["tags"]
					group := tag
					if g, ok := groupNames[tag]; ok {
						group = g
					}
					desc := meta["summary"]
					if desc == "" {
						desc = method + " " + p
					}
					rel, _ := filepath.Rel(dir, path)
					routes = append(routes, route{
						path: p, method: method, group: group, desc: desc, file: rel,
					})
				}
			}
		}
		return nil
	})
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].group != routes[j].group {
			return routes[i].group < routes[j].group
		}
		if routes[i].path != routes[j].path {
			return routes[i].path < routes[j].path
		}
		return routes[i].method < routes[j].method
	})
	return routes
}

// checkMissing 扫描 manifest/sql 下所有种子 SQL, 报告 api/ 中存在但种子未收录的路由。
func checkMissing(root string, routes []route) []route {
	rowRe := regexp.MustCompile(`\('(/api/[^']+)',\s*'(GET|POST|PUT|DELETE|PATCH|\*)'`)
	exist := map[string]bool{}
	_ = filepath.Walk(filepath.Join(root, "manifest", "sql"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".sql") {
			return err
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		for _, m := range rowRe.FindAllStringSubmatch(string(data), -1) {
			exist[normPathParam(m[1])+"|"+m[2]] = true
		}
		return nil
	})
	var missing []route
	for _, r := range routes {
		if !exist[normPathParam(r.path)+"|"+r.method] {
			missing = append(missing, r)
		}
	}
	return missing
}

// renderSQL 生成对齐的 INSERT IGNORE 块 (风格与 init.sql 种子块一致)。
func renderSQL(routes []route) string {
	wp, wm, wg := 0, 0, 0
	for _, r := range routes {
		wp = max(wp, len(r.path))
		wm = max(wm, len(r.method))
		wg = max(wg, len(escape(r.group)))
	}
	var b strings.Builder
	b.WriteString("-- ------------------------------------------------------------\n")
	b.WriteString("-- sys_api 种子 (tools/genapi 扫描 api/ 的 g.Meta 路由自动生成, 勿手工改路径)\n")
	b.WriteString("-- 依赖 sys_api uk_path_method 唯一键, INSERT IGNORE 幂等可重复执行。\n")
	b.WriteString("-- ------------------------------------------------------------\n")
	b.WriteString("INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES\n")
	for i, r := range routes {
		cellP := fmt.Sprintf("'%s',", r.path)
		cellM := fmt.Sprintf("'%s',", r.method)
		cellG := fmt.Sprintf("'%s',", escape(r.group))
		end := ","
		if i == len(routes)-1 {
			end = ";"
		}
		fmt.Fprintf(&b, "  (%-*s %-*s %-*s '%s')%s\n",
			wp+3, cellP, wm+3, cellM, wg+3, cellG, escape(r.desc), end)
	}
	return b.String()
}

// escape SQL 字符串转义 (单引号翻倍)。
func escape(s string) string { return strings.ReplaceAll(s, "'", "''") }

// findServerRoot 从当前目录向上查找 go.mod (module hinay.cn/admin) 定位 server 根。
func findServerRoot() string {
	dir, _ := os.Getwd()
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(data), "module hinay.cn/admin") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			fmt.Println("错误: 未找到 server 根目录 (需在 server/ 或其子目录内执行)")
			os.Exit(1)
		}
		dir = parent
	}
}
