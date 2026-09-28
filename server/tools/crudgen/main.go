// crudgen 命令行入口: 核心逻辑在 utility/crudgen (与网页端代码生成共用)。
//
// 用法 (在 server/ 下):
//
//	go run ./tools/crudgen -table biz_article -mod article -title "文章管理"
//	go run ./tools/crudgen -table biz_article -title "文章管理" -dry   # 仅预览
//	make gen-crud TABLE=biz_article TITLE="文章管理"
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hinay.cn/admin/utility/crudgen"
)

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

func main() {
	var (
		table = flag.String("table", "", "目标表名 (必填, 需存在于 -sql 指向的 DDL 中)")
		mod   = flag.String("mod", "", "模块名 (go 包名兼前端目录, 默认取表名去前缀去复数)")
		title = flag.String("title", "", "中文标题 (默认表名)")
		sqlF  = flag.String("sql", "manifest/sql/init.sql", "DDL 来源 SQL 文件")
		dry   = flag.Bool("dry", false, "仅预览, 不写文件")
	)
	flag.Parse()
	if *table == "" {
		fmt.Println("用法: crudgen -table <表名> [-mod 模块名] -title <中文标题> [-sql DDL文件] [-dry]")
		os.Exit(1)
	}

	serverRoot := findServerRoot()
	sqlPath := *sqlF
	if !filepath.IsAbs(sqlPath) {
		sqlPath = filepath.Join(serverRoot, sqlPath)
	}
	sqlBytes, err := os.ReadFile(sqlPath)
	check(err)

	inputs, menuMax, err := crudgen.ParseSQL(string(sqlBytes), *table)
	check(err)

	m, err := crudgen.Build(crudgen.BuildOptions{
		Table: *table, Mod: *mod, Title: *title,
		Inputs:     inputs,
		MenuMaxId:  menuMax,
		UpgradeSeq: crudgen.NextUpgradeSeq(filepath.Join(serverRoot, "manifest/sql/upgrade")),
	})
	check(err)

	files, err := crudgen.Render(m)
	check(err)

	fmt.Printf("crudgen: 表=%s 模块=%s 实体=%s 标题=%s, 共 %d 个文件\n",
		m.Table, m.Mod, m.Entity, m.Title, len(files))
	if *dry {
		for _, f := range files {
			fmt.Printf("  ~ [dry] %s (%d bytes)\n", f.Path, len(f.Content))
		}
		return
	}

	webRoot := filepath.Clean(filepath.Join(serverRoot, "../web_src"))
	written, err := crudgen.WriteAll(serverRoot, webRoot, m, files)
	check(err)
	for _, p := range written {
		fmt.Printf("  + %s\n", p)
	}
	fmt.Println("  ~ logic.go / cmd.go / useApi/index.ts 已自动接线")
	fmt.Printf(`
完成! 后续步骤:
  1. 确认建表 DDL 已在 init.sql (或本次升级 SQL) 中, 并执行:
     mysql -u<user> -p <库名> < manifest/sql/upgrade/%s_gen_%s.sql
  2. (可选) 连库执行 gf gen dao 覆盖重新生成 model/dao (结构一致)
  3. 角色管理 -> 为目标角色分配「%s」菜单与按钮权限
  4. 重启后端生效
`, m.UpgradeSeq, m.Mod, m.Title)
}

func check(err error) {
	if err != nil {
		fmt.Println("错误:", err)
		os.Exit(1)
	}
}
