// Package v1 系统管理-代码生成接口 (gencode)。
//
// 网页版 CRUD 代码生成器: 表清单/列信息读取 (information_schema),
// 预览 / zip 下载 / 直写源码树。整组接口受配置开关 gencode.enable 控制 (默认关闭),
// 直写额外要求服务实例运行于源码目录 (检测 go.mod)。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// GencodeColumnIn 列配置 (预览/下载/写入口透传; 覆盖默认勾选)。
type GencodeColumnIn struct {
	Name     string `json:"name"`
	InList   *bool  `json:"inList"`
	InForm   *bool  `json:"inForm"`
	InSearch *bool  `json:"inSearch"`
}

// GencodeTablesReq 表清单。
type GencodeTablesReq struct {
	g.Meta `path:"/system/gencode/tables" tags:"SystemGencode" method:"get" summary:"可生成表清单"`
}

// GencodeTablesRes 表清单响应。
type GencodeTablesRes struct {
	List []GencodeTableItem `json:"list"`
}

// GencodeTableItem 表项。
type GencodeTableItem struct {
	TableName    string `json:"tableName"`
	TableComment string `json:"tableComment"`
}

// GencodeColumnsReq 列信息 (含默认勾选)。
type GencodeColumnsReq struct {
	g.Meta `path:"/system/gencode/columns" tags:"SystemGencode" method:"get" summary:"表列信息"`
	Table  string `json:"table" in:"query" v:"required#缺少表名" dc:"表名"`
}

// GencodeColumnsRes 列信息响应。
type GencodeColumnsRes struct {
	List []GencodeColumnItem `json:"list"`
}

// GencodeColumnItem 列项 (含推断的 Go 类型/表单控件与默认勾选)。
type GencodeColumnItem struct {
	Name       string `json:"name"`
	DataType   string `json:"dataType"`
	ColumnType string `json:"columnType"`
	Comment    string `json:"comment"`
	GoType     string `json:"goType"`
	Kind       string `json:"kind"`
	Skip       bool   `json:"skip"` // 约定列 (id/时间戳/审计), 不参与生成
	InList     bool   `json:"inList"`
	InForm     bool   `json:"inForm"`
	InSearch   bool   `json:"inSearch"`
}

// GencodePreviewReq 预览生成文件。
type GencodePreviewReq struct {
	g.Meta  `path:"/system/gencode/preview" tags:"SystemGencode" method:"post" summary:"预览生成代码"`
	Table   string              `json:"table" v:"required#缺少表名"`
	Mod     string              `json:"mod" dc:"模块名 (默认表名去前缀)"`
	Title   string              `json:"title" dc:"中文标题"`
	Columns []GencodeColumnIn   `json:"columns" dc:"列勾选覆盖 (为空用默认)"`
}

// GencodePreviewRes 预览响应。
type GencodePreviewRes struct {
	Files []GencodeFileItem `json:"files"`
}

// GencodeFileItem 生成文件。
type GencodeFileItem struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// GencodeDownloadReq zip 下载 (列配置经 cols 传 JSON 字符串)。
type GencodeDownloadReq struct {
	g.Meta `path:"/system/gencode/download" tags:"SystemGencode" method:"get" summary:"下载生成代码(zip)"`
	Table  string `json:"table"    in:"query" v:"required#缺少表名"`
	Mod    string `json:"mod"      in:"query"`
	Title  string `json:"title"    in:"query"`
	Cols   string `json:"cols"     in:"query" dc:"列勾选覆盖 JSON (可空)"`
}

// GencodeDownloadRes 下载响应 (zip 流直接写入响应)。
type GencodeDownloadRes struct{}

// GencodeWriteReq 直写源码树 (开发环境专用)。
type GencodeWriteReq struct {
	g.Meta  `path:"/system/gencode/write" tags:"SystemGencode" method:"post" summary:"生成并写入源码树"`
	Table   string            `json:"table" v:"required#缺少表名"`
	Mod     string            `json:"mod"`
	Title   string            `json:"title"`
	Columns []GencodeColumnIn `json:"columns"`
}

// GencodeWriteRes 直写响应。
type GencodeWriteRes struct {
	Written []string `json:"written" dc:"已写入文件清单"`
}
