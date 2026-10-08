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

// FilePresignReq 获取预签名直传地址 (S3 兼容存储模式)。
// 上传走预签名 PUT 直传对象存储, 文件内容不经过应用服务器;
// 本地存储时返回 mode=server, 前端回落到普通上传接口。
type FilePresignReq struct {
	g.Meta       `path:"/system/files/presign" tags:"SystemFile" method:"post" summary:"获取预签名上传地址"`
	OriginalName string `json:"originalName" v:"required" dc:"原始文件名"`
	ContentType  string `json:"contentType"              dc:"文件MIME类型(浏览器File.type), 参与签名"`
	Size         int64  `json:"size"                     dc:"文件大小(字节)"`
}

// FilePresignRes 预签名响应。
type FilePresignRes struct {
	Mode        string `json:"mode"        dc:"presign=预签名直传 / server=回落普通上传"`
	UploadUrl   string `json:"uploadUrl"   dc:"预签名上传地址(直传对象存储, 不带鉴权头)"`
	Method      string `json:"method"      dc:"HTTP方法, 固定 PUT"`
	Key         string `json:"key"         dc:"对象存储key, 确认接口原样带回"`
	ContentType string `json:"contentType" dc:"上传时必须原样携带的Content-Type头"`
	ExpireSec   int    `json:"expireSec"   dc:"上传地址有效期(秒)"`
}

// FilePresignConfirmReq 预签名直传完成后的确认: 服务端校验对象已存在后落库。
type FilePresignConfirmReq struct {
	g.Meta       `path:"/system/files/presign/confirm" tags:"SystemFile" method:"post" summary:"预签名上传确认"`
	Key          string `json:"key"          v:"required" dc:"预签名响应返回的对象key"`
	OriginalName string `json:"originalName" v:"required" dc:"原始文件名"`
	ContentType  string `json:"contentType"              dc:"上传时使用的Content-Type"`
}

// FilePresignConfirmRes 确认响应, 结构同普通上传响应。
type FilePresignConfirmRes = FileUploadRes
