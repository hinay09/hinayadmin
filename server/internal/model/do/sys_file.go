// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysFile is the golang structure of table sys_file for DAO operations like Where/Data.
type SysFile struct {
	g.Meta       `orm:"table:sys_file, do:true"`
	Id           any         // ID
	Name         any         // 存储文件名
	OriginalName any         // 原始文件名
	Path         any         // 存储路径
	Url          any         // 访问URL
	Size         any         // 文件大小(字节)
	MimeType     any         // MIME类型
	Extension    any         // 文件扩展名
	UserId       any         // 上传用户ID
	CreatedAt    *gtime.Time // 创建时间
	DeletedAt    *gtime.Time // 删除时间(软删)
}
