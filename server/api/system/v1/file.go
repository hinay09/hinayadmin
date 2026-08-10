// Package v1 系统管理-文件接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// FileListReq 文件分页列表。
type FileListReq struct {
	g.Meta   `path:"/system/files" tags:"SystemFile" method:"get" summary:"文件分页列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"文件名/原始名模糊"`
	MimeType string `json:"mimeType" in:"query" dc:"MIME类型过滤"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// FileListRes 列表响应。
type FileListRes response.PageResult

// FileUploadReq 上传文件。
type FileUploadReq struct {
	g.Meta `path:"/system/files/upload" tags:"SystemFile" method:"post" summary:"上传文件"`
}

// FileUploadRes 上传响应。
type FileUploadRes struct {
	Id           uint64 `json:"id"`
	Name         string `json:"name"`
	OriginalName string `json:"originalName"`
	Url          string `json:"url"`
	Size         uint64 `json:"size"`
	Extension    string `json:"extension"`
}

// FileDeleteReq 删除文件。
type FileDeleteReq struct {
	g.Meta `path:"/system/files/{id}" tags:"SystemFile" method:"delete" summary:"删除文件"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// FileDeleteRes 删除响应。
type FileDeleteRes struct{}
