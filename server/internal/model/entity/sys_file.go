// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysFile is the golang structure for table sys_file.
type SysFile struct {
	Id           uint64      `json:"id"           orm:"id"            description:"ID"`       // ID
	Name         string      `json:"name"         orm:"name"          description:"存储文件名"`    // 存储文件名
	OriginalName string      `json:"originalName" orm:"original_name" description:"原始文件名"`    // 原始文件名
	Path         string      `json:"path"         orm:"path"          description:"存储路径"`     // 存储路径
	Url          string      `json:"url"          orm:"url"           description:"访问URL"`    // 访问URL
	Size         uint64      `json:"size"         orm:"size"          description:"文件大小(字节)"` // 文件大小(字节)
	MimeType     string      `json:"mimeType"     orm:"mime_type"     description:"MIME类型"`   // MIME类型
	Extension    string      `json:"extension"    orm:"extension"     description:"文件扩展名"`    // 文件扩展名
	UserId       uint64      `json:"userId"       orm:"user_id"       description:"上传用户ID"`   // 上传用户ID
	CreatedAt    *gtime.Time `json:"createdAt"    orm:"created_at"    description:"创建时间"`     // 创建时间
	DeletedAt    *gtime.Time `json:"deletedAt"    orm:"deleted_at"    description:"删除时间(软删)"` // 删除时间(软删)
}
