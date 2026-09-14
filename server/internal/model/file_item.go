package model

import "github.com/gogf/gf/v2/os/gtime"

// FileItem 文件列表 VO。
type FileItem struct {
	Id           uint64      `json:"id"`
	Name         string      `json:"name"`
	OriginalName string      `json:"originalName"`
	Path         string      `json:"path"`
	Url          string      `json:"url"`
	Size         uint64      `json:"size"`
	MimeType     string      `json:"mimeType"`
	Extension    string      `json:"extension"`
	UserId       uint64      `json:"userId"`
	CreatedAt    *gtime.Time `json:"createdAt"`
}
