// Package crudgen 模板定义。 (CLI 与网页端代码生成共用)
package crudgen

// ---------------------------------------------------------------------------
// 模板内容
// ---------------------------------------------------------------------------

const apiTmpl = `// Package v1 {{.Title}}接口。 (crudgen 生成)
package v1

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gtime"

	"hinay.cn/admin/utility/response"
)

// I{{.Entity}}V1 控制器接口 (gf gen ctrl 约定)。
type I{{.Entity}}V1 interface {
	{{.Entity}}List(ctx context.Context, req *{{.Entity}}ListReq) (res *{{.Entity}}ListRes, err error)
	{{.Entity}}Create(ctx context.Context, req *{{.Entity}}CreateReq) (res *{{.Entity}}CreateRes, err error)
	{{.Entity}}Update(ctx context.Context, req *{{.Entity}}UpdateReq) (res *{{.Entity}}UpdateRes, err error)
	{{.Entity}}Delete(ctx context.Context, req *{{.Entity}}DeleteReq) (res *{{.Entity}}DeleteRes, err error)
	{{.Entity}}BatchDelete(ctx context.Context, req *{{.Entity}}BatchDeleteReq) (res *{{.Entity}}BatchDeleteRes, err error)
{{- if .HasStatusCol}}
	{{.Entity}}BatchStatus(ctx context.Context, req *{{.Entity}}BatchStatusReq) (res *{{.Entity}}BatchStatusRes, err error)
{{- end}}
	{{.Entity}}Export(ctx context.Context, req *{{.Entity}}ExportReq) (res *{{.Entity}}ExportRes, err error)
	{{.Entity}}ImportTemplate(ctx context.Context, req *{{.Entity}}ImportTemplateReq) (res *{{.Entity}}ImportTemplateRes, err error)
	{{.Entity}}Import(ctx context.Context, req *{{.Entity}}ImportReq) (res *{{.Entity}}ImportRes, err error)
}

// {{.Entity}}ListReq 分页列表。
type {{.Entity}}ListReq struct {
	g.Meta   {{bt}}path:"/{{.Mod}}" tags:"{{.Tag}}" method:"get" summary:"{{.Title}}分页列表"{{bt}}
	Keyword  string {{bt}}json:"keyword" in:"query" dc:"关键词模糊"{{bt}}
	Page     int    {{bt}}json:"page"     in:"query" d:"1"  dc:"页码"{{bt}}
	PageSize int    {{bt}}json:"pageSize" in:"query" d:"10" dc:"页大小"{{bt}}
}

// {{.Entity}}Item 列表项。
type {{.Entity}}Item struct {
	Id        uint64      {{bt}}json:"id"{{bt}}
{{- range .Columns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"{{bt}}
{{- end}}
	CreatedAt *gtime.Time {{bt}}json:"createdAt"{{bt}}
}

// {{.Entity}}ListRes 列表响应。
type {{.Entity}}ListRes response.PageResult

// {{.Entity}}CreateReq 新增。
type {{.Entity}}CreateReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}" tags:"{{.Tag}}" method:"post" summary:"新增{{.Title}}"{{bt}}
{{- range .FormColumns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"{{if .Required}} v:"required#{{label .}}不能为空"{{end}}{{bt}}
{{- end}}
}

// {{.Entity}}CreateRes 新增响应。
type {{.Entity}}CreateRes struct{}

// {{.Entity}}UpdateReq 修改。
type {{.Entity}}UpdateReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}/:id" tags:"{{.Tag}}" method:"put" summary:"修改{{.Title}}"{{bt}}
	Id     uint64 {{bt}}json:"id" in:"path" v:"min:1"{{bt}}
{{- range .FormColumns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"{{if .Required}} v:"required#{{label .}}不能为空"{{end}}{{bt}}
{{- end}}
}

// {{.Entity}}UpdateRes 修改响应。
type {{.Entity}}UpdateRes struct{}

// {{.Entity}}DeleteReq 删除 (软删)。
type {{.Entity}}DeleteReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}/:id" tags:"{{.Tag}}" method:"delete" summary:"删除{{.Title}}"{{bt}}
	Id     uint64 {{bt}}json:"id" in:"path" v:"min:1"{{bt}}
}

// {{.Entity}}DeleteRes 删除响应。
type {{.Entity}}DeleteRes struct{}

// {{.Entity}}BatchDeleteReq 批量删除 (软删)。
type {{.Entity}}BatchDeleteReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}/batch-delete" tags:"{{.Tag}}" method:"post" summary:"批量删除{{.Title}}"{{bt}}
	Ids    []uint64 {{bt}}json:"ids" v:"required#请选择要删除的记录"{{bt}}
}

// {{.Entity}}BatchDeleteRes 批量删除响应。
type {{.Entity}}BatchDeleteRes struct{}
{{if .HasStatusCol}}
// {{.Entity}}BatchStatusReq 批量启用/停用。
type {{.Entity}}BatchStatusReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}/batch-status" tags:"{{.Tag}}" method:"post" summary:"批量启用/停用{{.Title}}"{{bt}}
	Ids    []uint64 {{bt}}json:"ids" v:"required#请选择要操作的记录"{{bt}}
	{{.StatusCol.GoName}} int {{bt}}json:"{{camel .StatusCol.GoName}}" v:"required|in:0,1#请选择记录|状态取值仅能为 0/1"{{bt}}
}

// {{.Entity}}BatchStatusRes 批量状态响应。
type {{.Entity}}BatchStatusRes struct{}
{{end}}
// {{.Entity}}ExportReq 列表导出 (xlsx 附件, 过滤条件与列表一致, 不分页)。
type {{.Entity}}ExportReq struct {
	g.Meta  {{bt}}path:"/{{.Mod}}/export" tags:"{{.Tag}}" method:"get" summary:"{{.Title}}列表导出"{{bt}}
	Keyword string {{bt}}json:"keyword" in:"query" dc:"关键词模糊"{{bt}}
}

// {{.Entity}}ExportRes 导出响应 (内容直接写入响应流)。
type {{.Entity}}ExportRes struct{}

// {{.Entity}}ImportTemplateReq 导入模板下载 (xlsx 附件, 表头与 Import 严格一致)。
type {{.Entity}}ImportTemplateReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}/import-template" tags:"{{.Tag}}" method:"get" summary:"{{.Title}}导入模板下载"{{bt}}
}

// {{.Entity}}ImportTemplateRes 模板下载响应 (内容直接写入响应流)。
type {{.Entity}}ImportTemplateRes struct{}

// {{.Entity}}ImportReq 导入 (xlsx 文件, 表头与模板一致)。
type {{.Entity}}ImportReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}/import" tags:"{{.Tag}}" method:"post" summary:"{{.Title}}导入"{{bt}}
	File   *ghttp.UploadFile {{bt}}json:"file" type:"file" v:"required#请选择导入文件" dc:"xlsx 文件"{{bt}}
}

// {{.Entity}}ImportRes 导入结果 (行级错误逐条返回)。
type {{.Entity}}ImportRes struct {
	SuccessCount int      {{bt}}json:"successCount" dc:"成功导入条数"{{bt}}
	FailCount    int      {{bt}}json:"failCount"    dc:"失败条数"{{bt}}
	Errors       []string {{bt}}json:"errors"       dc:"失败明细 (最多返回前 50 条)"{{bt}}
}
`

const apiAliasTmpl = `// Package {{.Mod}} {{.Title}}控制器接口别名。 (crudgen 生成)
package {{.Mod}}

import v1 "hinay.cn/admin/api/{{.Mod}}/v1"

// I{{.Entity}}V1 与 v1 接口等价 (gf gen ctrl 的包级别名约定)。
type I{{.Entity}}V1 = v1.I{{.Entity}}V1
`

const ctrlNewTmpl = `// {{.Entity}} 控制器注册。 (crudgen 生成)
package {{.Mod}}

import (
	"hinay.cn/admin/api/{{.Mod}}"
)

type ControllerV1 struct{}

func NewV1() {{.Mod}}.I{{.Entity}}V1 {
	return &ControllerV1{}
}
`

const ctrlMethodTmpl = `package {{.Mod}}

import (
	"context"

	v1 "hinay.cn/admin/api/{{.Mod}}/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) {{.Entity}}{{.Act}}(ctx context.Context, req *v1.{{.Entity}}{{.Act}}Req) (res *v1.{{.Entity}}{{.Act}}Res, err error) {
	return service.{{.Entity}}().{{.Act}}(ctx, req)
}
`

const svcTmpl = `// Package service {{.Title}}服务接口。 (crudgen 生成)
package service

import (
	"context"

	v1 "hinay.cn/admin/api/{{.Mod}}/v1"
)

type I{{.Entity}} interface {
	// List 分页列表。
	List(ctx context.Context, req *v1.{{.Entity}}ListReq) (res *v1.{{.Entity}}ListRes, err error)
	// Create 新增。
	Create(ctx context.Context, req *v1.{{.Entity}}CreateReq) (res *v1.{{.Entity}}CreateRes, err error)
	// Update 修改。
	Update(ctx context.Context, req *v1.{{.Entity}}UpdateReq) (res *v1.{{.Entity}}UpdateRes, err error)
	// Delete 删除 (软删)。
	Delete(ctx context.Context, req *v1.{{.Entity}}DeleteReq) (res *v1.{{.Entity}}DeleteRes, err error)
	// BatchDelete 批量删除 (软删)。
	BatchDelete(ctx context.Context, req *v1.{{.Entity}}BatchDeleteReq) (res *v1.{{.Entity}}BatchDeleteRes, err error)
{{- if .HasStatusCol}}
	// BatchStatus 批量启用/停用。
	BatchStatus(ctx context.Context, req *v1.{{.Entity}}BatchStatusReq) (res *v1.{{.Entity}}BatchStatusRes, err error)
{{- end}}
	// Export 列表导出 (xlsx 附件)。
	Export(ctx context.Context, req *v1.{{.Entity}}ExportReq) (res *v1.{{.Entity}}ExportRes, err error)
	// ImportTemplate 导入模板下载。
	ImportTemplate(ctx context.Context, req *v1.{{.Entity}}ImportTemplateReq) (res *v1.{{.Entity}}ImportTemplateRes, err error)
	// Import 导入 (行级错误不中断整体)。
	Import(ctx context.Context, req *v1.{{.Entity}}ImportReq) (res *v1.{{.Entity}}ImportRes, err error)
}

var local{{.Entity}} I{{.Entity}}

func {{.Entity}}() I{{.Entity}} {
	if local{{.Entity}} == nil {
		panic("implement not found for interface I{{.Entity}}, forgot register?")
	}
	return local{{.Entity}}
}

func Register{{.Entity}}(i I{{.Entity}}) {
	local{{.Entity}} = i
}
`

const logicTmpl = `// Package {{.Mod}} {{.Title}}业务逻辑。 (crudgen 生成)
package {{.Mod}}

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/{{.Mod}}/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/excelx"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type s{{.Entity}} struct{}

func init() {
	service.Register{{.Entity}}(New{{.Entity}}())
}

func New{{.Entity}}() *s{{.Entity}} {
	return &s{{.Entity}}{}
}

{{if .HasScope}}// scopedQuery 组装列表/导出共用的过滤条件 + 组织数据权限。
func (s *s{{.Entity}}) scopedQuery(ctx context.Context, keyword string) (*gdb.Model, error) {
	q := dao.{{.Entity}}.Ctx(ctx).Where("deleted_at IS NULL")
	if keyword != "" {
		kw := "%" + strings.TrimSpace(keyword) + "%"
		// 括号分组: 避免 OR 条件逃逸出软删过滤
		conds := make([]string, 0, {{len .StringCols}})
		args := make([]any, 0, {{len .StringCols}})
{{- range .StringCols}}
		conds = append(conds, "{{.Name}} LIKE ?")
		args = append(args, kw)
{{- end}}
		if len(conds) > 0 {
			q = q.Where("("+strings.Join(conds, " OR ")+")", args...)
		}
	}
	// 数据权限: 组织维度过滤 (表含 org_id 列自动套用, -no-scope 可关闭)
	q, derr := service.DataScope().Apply(ctx, q, "org_id", "create_id")
	if derr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, derr, "数据权限计算失败")
	}
	return q, nil
}
{{else}}// filterQuery 组装列表/导出共用的过滤条件。
func (s *s{{.Entity}}) filterQuery(ctx context.Context, keyword string) *gdb.Model {
	q := dao.{{.Entity}}.Ctx(ctx).Where("deleted_at IS NULL")
	if keyword != "" {
		kw := "%" + strings.TrimSpace(keyword) + "%"
		// 括号分组: 避免 OR 条件逃逸出软删过滤
		conds := make([]string, 0, {{len .StringCols}})
		args := make([]any, 0, {{len .StringCols}})
{{- range .StringCols}}
		conds = append(conds, "{{.Name}} LIKE ?")
		args = append(args, kw)
{{- end}}
		if len(conds) > 0 {
			q = q.Where("("+strings.Join(conds, " OR ")+")", args...)
		}
	}
	return q
}
{{end}}
// List 分页列表。
func (s *s{{.Entity}}) List(ctx context.Context, req *v1.{{.Entity}}ListReq) (res *v1.{{.Entity}}ListRes, err error) {
{{- if .HasScope}}
	q, err := s.scopedQuery(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}
{{- else}}
	q := s.filterQuery(ctx, req.Keyword)
{{- end}}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*model.{{.Entity}}
	if err = q.Page(req.Page, req.PageSize).Order("id DESC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	items := make([]*v1.{{.Entity}}Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, &v1.{{.Entity}}Item{
			Id: r.Id,
{{- range .Columns}}
			{{.GoName}}: r.{{.GoName}},
{{- end}}
			CreatedAt: r.CreatedAt,
		})
	}
	page := response.Page(items, int64(total), req.Page, req.PageSize)
	out := v1.{{.Entity}}ListRes(page)
	return &out, nil
}

// Create 新增 (create_id/update_id 由 ormfill 自动填充)。
func (s *s{{.Entity}}) Create(ctx context.Context, req *v1.{{.Entity}}CreateReq) (res *v1.{{.Entity}}CreateRes, err error) {
	_, err = dao.{{.Entity}}.Ctx(ctx).Data(g.Map{
{{- range .FormColumns}}
		"{{.Name}}": req.{{.GoName}},
{{- end}}
	}).Insert()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "新增失败")
	}
	return &v1.{{.Entity}}CreateRes{}, nil
}

// Update 修改。
func (s *s{{.Entity}}) Update(ctx context.Context, req *v1.{{.Entity}}UpdateReq) (res *v1.{{.Entity}}UpdateRes, err error) {
	_, err = dao.{{.Entity}}.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").
		Data(g.Map{
{{- range .FormColumns}}
			"{{.Name}}": req.{{.GoName}},
{{- end}}
		}).Update()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改失败")
	}
	return &v1.{{.Entity}}UpdateRes{}, nil
}

// Delete 删除 (软删)。
func (s *s{{.Entity}}) Delete(ctx context.Context, req *v1.{{.Entity}}DeleteReq) (res *v1.{{.Entity}}DeleteRes, err error) {
	_, err = dao.{{.Entity}}.Ctx(ctx).Where("id", req.Id).Delete()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "删除失败")
	}
	return &v1.{{.Entity}}DeleteRes{}, nil
}

// BatchDelete 批量删除 (软删)。
func (s *s{{.Entity}}) BatchDelete(ctx context.Context, req *v1.{{.Entity}}BatchDeleteReq) (res *v1.{{.Entity}}BatchDeleteRes, err error) {
	if _, err = dao.{{.Entity}}.Ctx(ctx).WhereIn("id", req.Ids).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "批量删除失败")
	}
	return &v1.{{.Entity}}BatchDeleteRes{}, nil
}
{{if .HasStatusCol}}
// BatchStatus 批量启用/停用。
func (s *s{{.Entity}}) BatchStatus(ctx context.Context, req *v1.{{.Entity}}BatchStatusReq) (res *v1.{{.Entity}}BatchStatusRes, err error) {
	if _, err = dao.{{.Entity}}.Ctx(ctx).WhereIn("id", req.Ids).
		Data(g.Map{"{{.StatusCol.Name}}": req.{{.StatusCol.GoName}}}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "批量操作失败")
	}
	return &v1.{{.Entity}}BatchStatusRes{}, nil
}
{{end}}
// ---------------------------------------------------------------------------
// Excel 导入导出 (范式见 utility/excelx 与 logic/system/user.go)
// ---------------------------------------------------------------------------

// {{camel .Entity}}ExportMaxRows 导出行数上限: 防全量导出拖垮内存, 更大数据量应走异步导出。
const {{camel .Entity}}ExportMaxRows = 10000

// {{camel .Entity}}ImportMaxRows 导入行数上限。
const {{camel .Entity}}ImportMaxRows = 1000

// {{camel .Entity}}ImportHeaders 导入模板/解析共用表头 (导入按下标取列, 须与模板一致)。
var {{camel .Entity}}ImportHeaders = []string{
{{- range $i, $c := .FormColumns}}
	"{{label $c}}{{if or $c.Required (eq $c.GoName $.FirstGoName)}}*{{end}}",
{{- end}}
}

// statusText 状态列导出文案。
func statusText(v int) string {
	if v == 1 {
		return "启用"
	}
	return "禁用"
}

// cellOf 任意列值转 Excel 单元格 (时间列格式化, 其余原样)。
func cellOf(v any) any {
	if t, ok := v.(*gtime.Time); ok && t != nil {
		return t.Format("Y-m-d H:i:s")
	}
	return v
}

// Export 列表导出 (过滤条件与列表一致, 不分页; 内容直接写入响应流下载)。
func (s *s{{.Entity}}) Export(ctx context.Context, req *v1.{{.Entity}}ExportReq) (res *v1.{{.Entity}}ExportRes, err error) {
{{- if .HasScope}}
	q, err := s.scopedQuery(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}
{{- else}}
	q := s.filterQuery(ctx, req.Keyword)
{{- end}}
	var rows []*model.{{.Entity}}
	if err = q.Limit({{camel .Entity}}ExportMaxRows).Order("id ASC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	headers := []string{"ID"{{range .Columns}}, "{{label .}}"{{end}}, "创建时间"}
	data := make([][]any, 0, len(rows))
	for _, r := range rows {
		data = append(data, []any{
			r.Id,
{{- range .Columns}}
			{{- if eq .Kind "switch"}}
			statusText(r.{{.GoName}}),
			{{- else}}
			cellOf(r.{{.GoName}}),
			{{- end}}
{{- end}}
			cellOf(r.CreatedAt),
		})
	}
	content, berr := excelx.Build("{{.Title}}", headers, data)
	if berr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, berr, "生成导出文件失败")
	}
	excelx.WriteToResponse(ctx, fmt.Sprintf("{{.Title}}_%s.xlsx", gtime.Now().Format("YmdHis")), content)
	return &v1.{{.Entity}}ExportRes{}, nil
}

// ImportTemplate 导入模板下载 (与 Import 的解析表头严格一致)。
func (s *s{{.Entity}}) ImportTemplate(ctx context.Context, req *v1.{{.Entity}}ImportTemplateReq) (res *v1.{{.Entity}}ImportTemplateRes, err error) {
	content, berr := excelx.Build("{{.Title}}导入", {{camel .Entity}}ImportHeaders, nil)
	if berr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, berr, "生成模板失败")
	}
	excelx.WriteToResponse(ctx, "{{.Title}}导入模板.xlsx", content)
	return &v1.{{.Entity}}ImportTemplateRes{}, nil
}

// Import 导入: 逐行校验, 行级错误不中断整体导入。
func (s *s{{.Entity}}) Import(ctx context.Context, req *v1.{{.Entity}}ImportReq) (res *v1.{{.Entity}}ImportRes, err error) {
	// 审计: 暂存导入文件摘要, OperationLog 记录入库
	contextx.SetAuditUpload(ctx, fmt.Sprintf("%s (%d bytes, %s)",
		filepath.Base(req.File.Filename), req.File.Size, req.File.Header.Get("Content-Type")))
	f, oerr := req.File.Open()
	if oerr != nil {
		return nil, xerror.Wrap(xerror.CodeParamInvalid, oerr, "读取上传文件失败")
	}
	defer f.Close()
	content, rerr := io.ReadAll(f)
	if rerr != nil {
		return nil, xerror.Wrap(xerror.CodeParamInvalid, rerr, "读取上传文件失败")
	}
	rows, perr := excelx.Read(content)
	if perr != nil {
		return nil, xerror.New(xerror.CodeParamInvalid, "无法解析 xlsx 文件, 请使用下载的模板填写")
	}
	if len(rows) < 2 {
		return nil, xerror.New(xerror.CodeParamInvalid, "文件没有数据行")
	}

	// 表头一致性校验 (忽略尾随空列差异)
	for i, want := range {{camel .Entity}}ImportHeaders {
		if i >= len(rows[0]) || strings.TrimSpace(rows[0][i]) == "" {
			continue
		}
		if strings.TrimSuffix(rows[0][i], "*") != strings.TrimSuffix(want, "*") &&
			strings.TrimSpace(rows[0][i]) != want {
			return nil, xerror.New(xerror.CodeParamInvalid, fmt.Sprintf("表头第 %d 列应为「%s」", i+1, want))
		}
	}

	res = &v1.{{.Entity}}ImportRes{Errors: []string{}}
	cell := func(cols []string, i int) string {
		if i < len(cols) {
			return strings.TrimSpace(cols[i])
		}
		return ""
	}

	for i := 1; i < len(rows); i++ {
		if res.SuccessCount+res.FailCount >= {{camel .Entity}}ImportMaxRows {
			res.Errors = append(res.Errors, fmt.Sprintf("已达单次导入上限 %d 行, 其余未处理", {{camel .Entity}}ImportMaxRows))
			break
		}
		cols := rows[i]
		rowno := i + 1
		fail := func(reason string) {
			res.FailCount++
			if len(res.Errors) < 50 {
				res.Errors = append(res.Errors, fmt.Sprintf("第 %d 行: %s", rowno, reason))
			}
		}
		// 整行为空跳过
		allEmpty := true
		for _, c := range cols {
			if strings.TrimSpace(c) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}

		data := g.Map{}
{{- range $i, $c := .FormColumns}}
		// {{label $c}}
		{
			s := cell(cols, {{$i}})
		{{- if eq $c.Kind "switch"}}
			if s == "" {
				data["{{$c.Name}}"] = 1 // 默认启用
			} else {
				v, cerr := strconv.Atoi(s)
				if cerr != nil || (v != 0 && v != 1) {
					fail("{{label $c}}须为 0/1")
					continue
				}
				data["{{$c.Name}}"] = v
			}
		{{- else if eq $c.GoType "string"}}
			{{- if or $c.Required (eq $c.GoName $.FirstGoName)}}
			if s == "" {
				fail("{{label $c}}不能为空")
				continue
			}
			{{- end}}
			data["{{$c.Name}}"] = s
		{{- else if eq $c.GoType "int"}}
			{{- if $c.Required}}
			if s == "" {
				fail("{{label $c}}不能为空")
				continue
			}
			{{- end}}
v := 0
			if s != "" {
				p, cerr := strconv.Atoi(s)
				if cerr != nil {
					fail("{{label $c}}须为数字")
					continue
				}
				v = p
			}
			data["{{$c.Name}}"] = v
		{{- else if eq $c.GoType "int64"}}
			{{- if $c.Required}}
			if s == "" {
				fail("{{label $c}}不能为空")
				continue
			}
			{{- end}}
var v int64
			if s != "" {
				p, cerr := strconv.ParseInt(s, 10, 64)
				if cerr != nil {
					fail("{{label $c}}须为数字")
					continue
				}
				v = p
			}
			data["{{$c.Name}}"] = v
		{{- else if eq $c.GoType "uint64"}}
			{{- if $c.Required}}
			if s == "" {
				fail("{{label $c}}不能为空")
				continue
			}
			{{- end}}
var v uint64
			if s != "" {
				p, cerr := strconv.ParseUint(s, 10, 64)
				if cerr != nil {
					fail("{{label $c}}须为数字ID")
					continue
				}
				v = p
			}
			data["{{$c.Name}}"] = v
		{{- else if eq $c.GoType "float64"}}
			{{- if $c.Required}}
			if s == "" {
				fail("{{label $c}}不能为空")
				continue
			}
			{{- end}}
var v float64
			if s != "" {
				p, cerr := strconv.ParseFloat(s, 64)
				if cerr != nil {
					fail("{{label $c}}须为数字")
					continue
				}
				v = p
			}
			data["{{$c.Name}}"] = v
		{{- else}}
			// 时间类列: 文本原样交给 ORM 解析
			if s != "" {
				data["{{$c.Name}}"] = s
			}
		{{- end}}
		}
{{- end}}

		if _, ierr := dao.{{.Entity}}.Ctx(ctx).Data(data).Insert(); ierr != nil {
			fail("写入失败: " + ierr.Error())
			continue
		}
		res.SuccessCount++
	}
	return res, nil
}
`

const modelTmpl = `package model

import "github.com/gogf/gf/v2/os/gtime"

// {{.Entity}} 数据库实体 ({{.Table}})。 (crudgen 生成)
type {{.Entity}} struct {
	Id        uint64      {{bt}}json:"id"        orm:"id"{{bt}}
{{- range .Columns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"  orm:"{{.Name}}"{{bt}}
{{- end}}
	CreatedAt *gtime.Time {{bt}}json:"createdAt" orm:"created_at"{{bt}}
	UpdatedAt *gtime.Time {{bt}}json:"updatedAt" orm:"updated_at"{{bt}}
}
`

const entityTmpl = `// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// {{.Entity}} is the golang structure for table {{.Table}}.
type {{.Entity}} struct {
	Id        uint64      {{bt}}json:"id"        orm:"id"         description:"ID"{{bt}}
{{- range .Columns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"  orm:"{{.Name}}"  description:"{{.Comment}}"{{bt}}
{{- end}}
	CreatedAt *gtime.Time {{bt}}json:"createdAt" orm:"created_at" description:"创建时间"{{bt}}
	UpdatedAt *gtime.Time {{bt}}json:"updatedAt" orm:"updated_at" description:"更新时间"{{bt}}
	DeletedAt *gtime.Time {{bt}}json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"{{bt}}
}
`

const doTmpl = `// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// {{.Entity}} is the golang structure of table {{.Table}} for DAO operations like Where/Data.
type {{.Entity}} struct {
	g.Meta    {{bt}}orm:"table:{{.Table}}, do:true"{{bt}}
	Id        any         // ID
{{- range .Columns}}
	{{.GoName}} any         // {{.Comment}}
{{- end}}
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删)
}
`

const daoTmpl = `// =================================================================================
// This file is auto-generated by the GoFrame CLI tool. You may modify it as needed.
// =================================================================================

package dao

import (
	"hinay.cn/admin/internal/dao/internal"
)

// {{camel .Entity}}Dao is the data access object for the table {{.Table}}.
type {{camel .Entity}}Dao struct {
	*internal.{{.Entity}}Dao
}

var (
	// {{.Entity}} is a globally accessible object for table {{.Table}} operations.
	{{.Entity}} = {{camel .Entity}}Dao{internal.New{{.Entity}}Dao()}
)

// Add your custom methods and functionality below.
`

const daoInternalTmpl = `// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// {{.Entity}}Dao is the data access object for the table {{.Table}}.
type {{.Entity}}Dao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  {{.Entity}}Columns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// {{.Entity}}Columns defines and stores column names for the table {{.Table}}.
type {{.Entity}}Columns struct {
	Id        string // ID
{{- range .Columns}}
	{{.GoName}} string // {{.Comment}}
{{- end}}
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间
	DeletedAt string // 删除时间(软删)
}

// {{camel .Entity}}Columns holds the columns for the table {{.Table}}.
var {{camel .Entity}}Columns = {{.Entity}}Columns{
	Id:        "id",
{{- range .Columns}}
	{{.GoName}}: "{{.Name}}",
{{- end}}
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// New{{.Entity}}Dao creates and returns a new DAO object for table data access.
func New{{.Entity}}Dao(handlers ...gdb.ModelHandler) *{{.Entity}}Dao {
	return &{{.Entity}}Dao{
		group:    "default",
		table:    "{{.Table}}",
		columns:  {{camel .Entity}}Columns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *{{.Entity}}Dao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *{{.Entity}}Dao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *{{.Entity}}Dao) Columns() {{.Entity}}Columns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *{{.Entity}}Dao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO.
func (dao *{{.Entity}}Dao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
func (dao *{{.Entity}}Dao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
`

const webApiTmpl = `/**
 * {{.Title}} API。 (crudgen 生成)
 */
import { useRequest } from '~/composables/useRequest'

export function use{{.Entity}}Api() {
  const r = useRequest()
  const base = '/{{.Mod}}'
  return {
    list: (params: { keyword?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(base + '/' + id, data),
    remove: (id: number) => r.del(base + '/' + id),
    batchRemove: (ids: number[]) => r.post(base + '/batch-delete', { ids }),
{{- if .HasStatusCol}}
    batchStatus: (ids: number[], {{camel .StatusCol.GoName}}: number) => r.post(base + '/batch-status', { ids, {{camel .StatusCol.GoName}} }),
{{- end}}
    export: (params?: { keyword?: string }) => r.download(base + '/export', params),
    importTemplate: () => r.download(base + '/import-template'),
    import: (file: File) => {
      const fd = new FormData()
      fd.append('file', file)
      return r.post<{ successCount: number; failCount: number; errors: string[] }>(base + '/import', fd)
    },
  }
}
`

const webPageTmpl = `<script setup lang="ts">
/**
 * {{.Title}} (crudgen 生成)
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Edit, Delete, Download, Upload } from '@element-plus/icons-vue'
import { use{{.Entity}}Api{{range .RelApis}}, use{{pascal .}}Api{{end}} } from '~/composables/useApi'
import type { ProColumn } from '~/components/ProTable.vue'

definePageMeta({ title: '{{.Title}}' })
defineOptions({ name: '{{.Mod}}' })

const api = use{{.Entity}}Api()
const tableRef = ref<InstanceType<typeof ProTable>>()
const query = reactive({ keyword: '' })

const columns: ProColumn[] = [
{{- range .ListColumns}}
  { prop: '{{camel .GoName}}', label: '{{label .}}'{{if eq .Kind "switch"}}, tag: true, width: 90{{end}}{{if eq .Kind "dict"}}, dict: '{{.DictCode}}'{{end}} },
{{- end}}
]
{{if or .FormColumns .HasStatusCol}}
// ---- 字典/关联下拉数据源 ----
{{- range .FormColumns}}
{{- if eq .Kind "dict"}}
const {{camel .GoName}}Options = useDictOptions().options('{{.DictCode}}')
{{- else if eq .Kind "relation"}}
{{- if .RelGuess}}
// TODO: 「{{label .}}」关联下拉未声明数据源 —— 列注释写「关联:{{.RelMod}}」后重新生成即自动接线,
// 或在此自行加载: {{camel .GoName}}Options.value = (await use{{pascal .RelMod}}Api().all()).list || []
const {{camel .GoName}}Options = ref<any[]>([])
{{- else}}
const {{camel .GoName}}Options = ref<any[]>([])
{{- end}}
{{- end}}
{{- end}}
{{- end}}
// ---- 新增/编辑 ----
const dialogVisible = ref(false)
const submitting = ref(false)
const editingId = ref(0)
const formRef = ref<FormInstance>()
const form = reactive({
{{- range .FormColumns}}
  {{camel .GoName}}: {{if eq .Kind "switch"}}1{{else if eq .Kind "number"}}0{{else if eq .Kind "relation"}}0{{else if eq .Kind "dict"}}''{{else}}''{{end}},
{{- end}}
})
const rules: FormRules = {
{{- range .FormColumns}}
{{- if .Required}}
  {{camel .GoName}}: [{ required: true, message: '必填', trigger: 'blur' }],
{{- end}}
{{- end}}
}

function openCreate() {
  editingId.value = 0
  Object.assign(form, {
{{- range .FormColumns}}
    {{camel .GoName}}: {{if eq .Kind "switch"}}1{{else if eq .Kind "number"}}0{{else if eq .Kind "relation"}}0{{else}}''{{end}},
{{- end}}
  })
  dialogVisible.value = true
}

function openEdit(row: any) {
  editingId.value = row.id
  Object.assign(form, {
{{- range .FormColumns}}
    {{camel .GoName}}: row.{{camel .GoName}}{{if eq .Kind "switch"}} ?? 1{{end}},
{{- end}}
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (editingId.value) {
      await api.update(editingId.value, { ...form })
      ElMessage.success('修改成功')
    }
    else {
      await api.create({ ...form })
      ElMessage.success('新增成功')
    }
    dialogVisible.value = false
    tableRef.value?.reload()
  }
  finally {
    submitting.value = false
  }
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确认删除该条记录吗?', '提示', { type: 'warning' })
  await api.remove(row.id)
  ElMessage.success('已删除')
  tableRef.value?.reload()
}

// ---- 批量操作 ----
const selected = computed(() => tableRef.value?.selection || [])

async function handleBatchRemove() {
  const ids = selected.value.map((r: any) => r.id)
  if (!ids.length) return
  await ElMessageBox.confirm({{bt}}确认删除选中的 ${ids.length} 条记录吗?{{bt}}, '提示', { type: 'warning' })
  await api.batchRemove(ids)
  ElMessage.success('已删除')
  tableRef.value?.reload()
}
{{if .HasStatusCol}}
async function handleBatchStatus({{camel .StatusCol.GoName}}: number) {
  const ids = selected.value.map((r: any) => r.id)
  if (!ids.length) return
  await api.batchStatus(ids, {{camel .StatusCol.GoName}})
  ElMessage.success('操作成功')
  tableRef.value?.reload()
}
{{end}}
// ---- 导出 / 导入 ----
const exporting = ref(false)

async function handleExport() {
  exporting.value = true
  try {
    await api.export({ keyword: query.keyword || undefined })
  }
  finally {
    exporting.value = false
  }
}

const importVisible = ref(false)
const importing = ref(false)
const importFile = ref<File | null>(null)
const fileInputRef = ref<HTMLInputElement>()
const importResult = ref<{ successCount: number; failCount: number; errors: string[] } | null>(null)

function openImport() {
  importFile.value = null
  importResult.value = null
  importVisible.value = true
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  importFile.value = input.files?.[0] || null
}

async function handleImport() {
  if (!importFile.value) {
    ElMessage.warning('请先选择文件')
    return
  }
  importing.value = true
  try {
    importResult.value = await api.import(importFile.value)
    if (importResult.value.successCount > 0) tableRef.value?.reload()
  }
  finally {
    importing.value = false
  }
}

onMounted(() => {
{{- range .FormColumns}}
{{- if and (eq .Kind "relation") (not .RelGuess)}}
  use{{pascal .RelMod}}Api().all().then((res: any) => {
    {{camel .GoName}}Options.value = res.list || []
  })
{{- end}}
{{- end}}
})
</script>

<template>
  <div class="page">
    <el-card>
      <QueryForm :query="query" @search="() => tableRef?.search()" @reset="() => { query.keyword = ''; tableRef?.search() }">
        <el-form-item label="关键词">
          <el-input v-model="query.keyword" placeholder="模糊搜索" clearable style="width:200px" @keyup.enter="() => tableRef?.search()" />
        </el-form-item>
      </QueryForm>

      <ProTable ref="tableRef" :fetch="api.list" :columns="columns" :query="query" selectable>
        <template #toolbar>
          <el-button v-permission="'{{.PermPrefix}}:create'" type="success" :icon="Plus" @click="openCreate">新增</el-button>
          <el-button v-permission="'{{.PermPrefix}}:batch-delete'" type="danger" plain :icon="Delete" :disabled="!selected.length" @click="handleBatchRemove">批量删除</el-button>
{{- if .HasStatusCol}}
          <el-button v-permission="'{{.PermPrefix}}:batch-status'" type="warning" plain :disabled="!selected.length" @click="handleBatchStatus(1)">批量启用</el-button>
          <el-button v-permission="'{{.PermPrefix}}:batch-status'" type="info" plain :disabled="!selected.length" @click="handleBatchStatus(0)">批量停用</el-button>
{{- end}}
          <el-button v-permission="'{{.PermPrefix}}:export'" :loading="exporting" :icon="Download" @click="handleExport">导出</el-button>
          <el-button v-permission="'{{.PermPrefix}}:import'" :icon="Upload" @click="openImport">导入</el-button>
        </template>
        <template #actions="{ row }">
          <el-button v-permission="'{{.PermPrefix}}:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
          <el-button v-permission="'{{.PermPrefix}}:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
        </template>
      </ProTable>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑{{.Title}}' : '新增{{.Title}}'" width="560px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
{{- range .FormColumns}}
{{- if eq .Kind "switch"}}
        <el-form-item label="{{label .}}">
          <el-radio-group v-model="form.{{camel .GoName}}">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
{{- else if eq .Kind "textarea"}}
        <el-form-item label="{{label .}}" prop="{{camel .GoName}}">
          <el-input v-model="form.{{camel .GoName}}" type="textarea" :rows="3" />
        </el-form-item>
{{- else if eq .Kind "number"}}
        <el-form-item label="{{label .}}" prop="{{camel .GoName}}">
          <el-input-number v-model="form.{{camel .GoName}}" :min="0" style="width:100%" />
        </el-form-item>
{{- else if eq .Kind "date"}}
        <el-form-item label="{{label .}}" prop="{{camel .GoName}}">
          <el-date-picker
            v-model="form.{{camel .GoName}}"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            placeholder="选择时间"
            clearable
            style="width:100%"
          />
        </el-form-item>
{{- else if eq .Kind "dict"}}
        <el-form-item label="{{label .}}" prop="{{camel .GoName}}">
          <el-select v-model="form.{{camel .GoName}}" clearable style="width:100%">
            <el-option v-for="o in {{camel .GoName}}Options" :key="o.dictValue" :value="o.dictValue" :label="o.dictLabel" />
          </el-select>
        </el-form-item>
{{- else if eq .Kind "relation"}}
        <el-form-item label="{{label .}}" prop="{{camel .GoName}}">
          <el-select v-model="form.{{camel .GoName}}" clearable filterable style="width:100%">
            <el-option v-for="o in {{camel .GoName}}Options" :key="o.id" :value="o.id" :label="o.name || o.title || o.id" />
          </el-select>
        </el-form-item>
{{- else}}
        <el-form-item label="{{label .}}" prop="{{camel .GoName}}">
          <el-input v-model="form.{{camel .GoName}}" />
        </el-form-item>
{{- end}}
{{- end}}
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 导入弹窗 -->
    <el-dialog v-model="importVisible" title="导入{{.Title}}" width="520px">
      <el-alert type="info" :closable="false" style="margin-bottom:12px">
        <template #title>
          请先
          <el-link type="primary" style="vertical-align:baseline" @click="api.importTemplate()">下载导入模板</el-link>
          , 按模板列顺序填写后上传
        </template>
      </el-alert>
      <input ref="fileInputRef" type="file" accept=".xlsx" style="display:none" @change="onFileChange">
      <el-button :icon="Upload" @click="fileInputRef?.click()">
        {{printf "{{ importFile ? importFile.name : '选择 xlsx 文件' }}"}}
      </el-button>

      <div v-if="importResult" class="import-result">
        <el-alert
          :type="importResult.failCount === 0 ? 'success' : 'warning'"
          :closable="false"
          :title="'导入完成: 成功 ' + importResult.successCount + ' 条, 失败 ' + importResult.failCount + ' 条'"
        />
        <ul v-if="importResult.errors.length" class="import-errors">
          <li v-for="(msg, i) in importResult.errors" :key="i">{{printf "{{ msg }}"}}</li>
        </ul>
      </div>

      <template #footer>
        <el-button @click="importVisible = false">关闭</el-button>
        <el-button type="primary" :loading="importing" @click="handleImport">开始导入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page {
  padding: 0;
}

.import-result {
  margin-top: 12px;
}

.import-errors {
  margin: 8px 0 0 4px;
  padding-left: 18px;
  max-height: 160px;
  overflow-y: auto;
  font-size: 13px;
  color: var(--el-color-danger);
  line-height: 1.8;
}
</style>
`

const sqlTmpl = `-- ============================================================
-- {{.Title}} (crudgen 生成: 菜单/按钮/API 种子)
-- 表: {{.Table}}
-- 幂等: 菜单按本模块 ID 块显式 DELETE+INSERT (sys_menu 无自然唯一键,
--       重复执行可覆盖旧值); API 依赖 sys_api uk_path_method 唯一键,
--       INSERT IGNORE 可重复执行。
-- ============================================================

-- 菜单 (顶级, id={{.MenuId}}): 先清本模块 ID 块再插入
DELETE FROM sys_menu WHERE id IN ({{range $i, $id := .MenuIds}}{{if $i}}, {{end}}{{$id}}{{end}});
INSERT INTO sys_menu (id, parent_id, name, type, path, component, icon, permission, sort, visible, status) VALUES
  ({{.MenuId}}, 0, '{{.Title}}', 2, '/{{.Mod}}', '{{.Mod}}/index', 'Document', '{{.PermPrefix}}:list', 200, 1, 1);

-- 按钮权限
INSERT INTO sys_menu (id, parent_id, name, type, path, component, icon, permission, sort, visible, status) VALUES
  ({{.Btns.Create}}, {{.MenuId}}, '{{.Title}}新增', 3, '', '', '', '{{.PermPrefix}}:create', 1, 0, 1),
  ({{.Btns.Update}}, {{.MenuId}}, '{{.Title}}修改', 3, '', '', '', '{{.PermPrefix}}:update', 2, 0, 1),
  ({{.Btns.Delete}}, {{.MenuId}}, '{{.Title}}删除', 3, '', '', '', '{{.PermPrefix}}:delete', 3, 0, 1),
  ({{.Btns.BatchDelete}}, {{.MenuId}}, '{{.Title}}批量删除', 3, '', '', '', '{{.PermPrefix}}:batch-delete', 4, 0, 1){{if .HasStatusCol}},
  ({{.Btns.BatchStatus}}, {{.MenuId}}, '{{.Title}}批量启用停用', 3, '', '', '', '{{.PermPrefix}}:batch-status', 5, 0, 1){{end}},
  ({{.Btns.Export}}, {{.MenuId}}, '{{.Title}}导出', 3, '', '', '', '{{.PermPrefix}}:export', 6, 0, 1),
  ({{.Btns.Import}}, {{.MenuId}}, '{{.Title}}导入', 3, '', '', '', '{{.PermPrefix}}:import', 7, 0, 1);

-- API 资源
INSERT IGNORE INTO sys_api (path, method, group_name, description) VALUES
  ('/api/v1/{{.Mod}}',                'GET',    '{{.Title}}', '{{.Title}}列表'),
  ('/api/v1/{{.Mod}}',                'POST',   '{{.Title}}', '新增{{.Title}}'),
  ('/api/v1/{{.Mod}}/:id',            'PUT',    '{{.Title}}', '修改{{.Title}}'),
  ('/api/v1/{{.Mod}}/:id',            'DELETE', '{{.Title}}', '删除{{.Title}}'),
  ('/api/v1/{{.Mod}}/batch-delete',   'POST',   '{{.Title}}', '批量删除{{.Title}}'){{if .HasStatusCol}},
  ('/api/v1/{{.Mod}}/batch-status',   'POST',   '{{.Title}}', '批量启用停用{{.Title}}'){{end}},
  ('/api/v1/{{.Mod}}/export',         'GET',    '{{.Title}}', '{{.Title}}列表导出'),
  ('/api/v1/{{.Mod}}/import-template','GET',    '{{.Title}}', '{{.Title}}导入模板下载'),
  ('/api/v1/{{.Mod}}/import',         'POST',   '{{.Title}}', '{{.Title}}导入');

-- Casbin: 内置超管角色(id=1) 不种 p 策略 (API 中间件全放行 + 菜单全量下发),
-- 新菜单默认仅超管可见, 其他角色在「角色管理-分配权限」中按需勾选。
`
