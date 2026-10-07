// Package gencode 网页版 CRUD 代码生成 (核心在 utility/crudgen)。
//
// 安全约束:
//   - 整组接口受配置 gencode.enable 控制 (默认关闭, 生产保持关闭);
//   - 表/列信息来自 information_schema (参数化查询, 表名为占位符);
//   - 直写仅当服务实例运行于源码目录 (向上找到 go.mod) 时可用, Docker 容器内天然拒绝;
//   - 写入前逐一检查目标存在性, 任一已存在即中止, 防覆盖。
package gencode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/crudgen"
	"hinay.cn/admin/utility/xerror"
)

type sGencode struct{}

func init() {
	service.RegisterGencode(NewGencode())
}

func NewGencode() *sGencode {
	return &sGencode{}
}

// enabled 配置开关检查 (实时读取, 改配置无需重启)。
func (s *sGencode) enabled(ctx context.Context) error {
	if v, err := g.Cfg().Get(ctx, "gencode.enable", false); err != nil || !v.Bool() {
		return xerror.New(xerror.CodeForbidden, "代码生成功能未启用 (配置 gencode.enable=true 后开放)")
	}
	return nil
}

// Tables 当前库的全部业务表。
func (s *sGencode) Tables(ctx context.Context, _ *v1.GencodeTablesReq) (res *v1.GencodeTablesRes, err error) {
	if err = s.enabled(ctx); err != nil {
		return nil, err
	}
	rows, err := g.DB().GetAll(ctx,
		"SELECT TABLE_NAME AS tname, TABLE_COMMENT AS tcomment "+
			"FROM information_schema.TABLES "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE' "+
			"ORDER BY TABLE_NAME")
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list := make([]v1.GencodeTableItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, v1.GencodeTableItem{
			TableName:    r["tname"].String(),
			TableComment: r["tcomment"].String(),
		})
	}
	return &v1.GencodeTablesRes{List: list}, nil
}

// Columns 表列信息 + 默认勾选 (约定列标记 skip)。
func (s *sGencode) Columns(ctx context.Context, req *v1.GencodeColumnsReq) (res *v1.GencodeColumnsRes, err error) {
	if err = s.enabled(ctx); err != nil {
		return nil, err
	}
	inputs, err := s.loadInputs(ctx, req.Table)
	if err != nil {
		return nil, err
	}
	list := make([]v1.GencodeColumnItem, 0, len(inputs))
	businessIdx := 0
	for _, in := range inputs {
		skip := crudgen.SkipColumn(in.Name)
		item := v1.GencodeColumnItem{
			Name:       in.Name,
			DataType:   in.DataType,
			ColumnType: in.ColumnType,
			Comment:    in.Comment,
			Skip:       skip,
			InForm:     true,
		}
		if !skip {
			item.GoType = crudgen.GoTypeOf(in.DataType, in.ColumnType)
			item.Kind = crudgen.KindOf(in.Name, in.DataType)
			item.InList = businessIdx < 8
			item.InSearch = item.GoType == "string"
			businessIdx++
		}
		list = append(list, item)
	}
	return &v1.GencodeColumnsRes{List: list}, nil
}

// Preview 预览全部生成文件内容。
func (s *sGencode) Preview(ctx context.Context, req *v1.GencodePreviewReq) (res *v1.GencodePreviewRes, err error) {
	if err = s.enabled(ctx); err != nil {
		return nil, err
	}
	m, err := s.build(ctx, req.Table, req.Mod, req.Title, req.Columns)
	if err != nil {
		return nil, err
	}
	files, err := crudgen.Render(m)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	out := &v1.GencodePreviewRes{Files: make([]v1.GencodeFileItem, 0, len(files))}
	for _, f := range files {
		out.Files = append(out.Files, v1.GencodeFileItem{Path: f.Path, Content: f.Content})
	}
	return out, nil
}

// Download 打包 zip 写入响应流 (附件下载)。
func (s *sGencode) Download(ctx context.Context, req *v1.GencodeDownloadReq) (res *v1.GencodeDownloadRes, err error) {
	if err = s.enabled(ctx); err != nil {
		return nil, err
	}
	var cols []v1.GencodeColumnIn
	if strings.TrimSpace(req.Cols) != "" {
		if err = json.Unmarshal([]byte(req.Cols), &cols); err != nil {
			return nil, xerror.New(xerror.CodeParamInvalid, "cols 参数不是合法 JSON")
		}
	}
	m, err := s.build(ctx, req.Table, req.Mod, req.Title, cols)
	if err != nil {
		return nil, err
	}
	files, err := crudgen.Render(m)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	content, err := crudgen.Zip(m, files)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "打包失败")
	}
	r := g.RequestFromCtx(ctx)
	if r == nil {
		return nil, xerror.New(xerror.CodeBusinessError, "非 HTTP 上下文")
	}
	r.Response.Header().Set("Content-Type", "application/zip")
	r.Response.Header().Set("Content-Disposition",
		`attachment; filename="gencode_`+m.Mod+`.zip"`)
	r.Response.Write(content)
	return &v1.GencodeDownloadRes{}, nil
}

// Write 直写源码树 (仅开发环境: 需检测到 server 源码根)。
func (s *sGencode) Write(ctx context.Context, req *v1.GencodeWriteReq) (res *v1.GencodeWriteRes, err error) {
	if err = s.enabled(ctx); err != nil {
		return nil, err
	}
	serverRoot := findServerRoot()
	if serverRoot == "" {
		return nil, xerror.New(xerror.CodeForbidden,
			"直写模式仅支持从源码目录启动的服务实例 (当前运行环境未检测到 go.mod), 请改用 zip 下载")
	}
	m, err := s.build(ctx, req.Table, req.Mod, req.Title, req.Columns)
	if err != nil {
		return nil, err
	}
	files, err := crudgen.Render(m)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	webRoot := filepath.Clean(filepath.Join(serverRoot, "../web_src"))
	if _, werr := os.Stat(webRoot); werr != nil {
		return nil, xerror.New(xerror.CodeForbidden, "未找到前端源码目录 web_src, 请改用 zip 下载")
	}
	written, werr := crudgen.WriteAll(serverRoot, webRoot, m, files)
	if werr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, werr)
	}
	return &v1.GencodeWriteRes{Written: written}, nil
}

// ---------------------------------------------------------------------------
// 内部
// ---------------------------------------------------------------------------

// loadInputs 从 information_schema 读取列元数据。
func (s *sGencode) loadInputs(ctx context.Context, table string) ([]crudgen.ColumnInput, error) {
	if !validTableName(table) {
		return nil, xerror.New(xerror.CodeParamInvalid, "表名不合法")
	}
	rows, err := g.DB().GetAll(ctx,
		"SELECT COLUMN_NAME AS cname, DATA_TYPE AS dtype, COLUMN_TYPE AS ctype, COLUMN_COMMENT AS ccomment "+
			"FROM information_schema.COLUMNS "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? "+
			"ORDER BY ORDINAL_POSITION", table)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if len(rows) == 0 {
		return nil, xerror.New(xerror.CodeParamInvalid, "表不存在: "+table)
	}
	inputs := make([]crudgen.ColumnInput, 0, len(rows))
	for _, r := range rows {
		inputs = append(inputs, crudgen.ColumnInput{
			Name:       r["cname"].String(),
			DataType:   r["dtype"].String(),
			ColumnType: r["ctype"].String(),
			Comment:    r["ccomment"].String(),
		})
	}
	return inputs, nil
}

// build 组装渲染模型 (菜单 ID 取自库中 sys_menu 最大值)。
func (s *sGencode) build(ctx context.Context, table, mod, title string, cols []v1.GencodeColumnIn) (*crudgen.Model, error) {
	if !validTableName(table) {
		return nil, xerror.New(xerror.CodeParamInvalid, "表名不合法")
	}
	inputs, err := s.loadInputs(ctx, table)
	if err != nil {
		return nil, err
	}
	// 应用列勾选覆盖
	if len(cols) > 0 {
		byName := map[string]v1.GencodeColumnIn{}
		for _, c := range cols {
			byName[c.Name] = c
		}
		for i := range inputs {
			if c, ok := byName[inputs[i].Name]; ok {
				inputs[i].InList = c.InList
				inputs[i].InForm = c.InForm
				inputs[i].InSearch = c.InSearch
			}
		}
	}
	menuMax, _ := g.DB().Model("sys_menu").Ctx(ctx).Fields("IFNULL(MAX(id),0)").Value()
	var menuMaxId uint64
	if menuMax != nil {
		menuMaxId = menuMax.Uint64()
	}
	// 升级序号: 源码目录可读时取下一个, 否则空 (Build 内兜底)
	serverRoot := findServerRoot()
	seq := ""
	if serverRoot != "" {
		seq = crudgen.NextUpgradeSeq(filepath.Join(serverRoot, "manifest/sql/gen"))
	}
	m, err := crudgen.Build(crudgen.BuildOptions{
		Table: table, Mod: mod, Title: title,
		Inputs: inputs, MenuMaxId: menuMaxId, UpgradeSeq: seq,
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeParamInvalid, err)
	}
	return m, nil
}

// validTableName 表名白名单校验 (information_schema 参数化已防注入,
// 此处同时防目录穿越等二次使用)。
func validTableName(t string) bool {
	if t == "" || len(t) > 64 {
		return false
	}
	for _, c := range t {
		if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// findServerRoot 从当前工作目录向上查找 server 源码根 (go.mod), 找不到返回空串。
func findServerRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		data, rerr := os.ReadFile(filepath.Join(dir, "go.mod"))
		if rerr == nil && strings.Contains(string(data), "module hinay.cn/admin") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}
