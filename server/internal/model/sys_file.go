package model

import "github.com/gogf/gf/v2/os/gtime"

// SysFile 文件实体。
type SysFile struct {
	Id           uint64      `json:"id"           orm:"id"`
	Name         string      `json:"name"         orm:"name"`
	OriginalName string      `json:"originalName" orm:"original_name"`
	Path         string      `json:"path"         orm:"path"`
	Url          string      `json:"url"          orm:"url"`
	Size         uint64      `json:"size"         orm:"size"`
	MimeType     string      `json:"mimeType"     orm:"mime_type"`
	Extension    string      `json:"extension"    orm:"extension"`
	UserId       uint64      `json:"userId"       orm:"user_id"`
	CreatedAt    *gtime.Time `json:"createdAt"    orm:"created_at"`
}
