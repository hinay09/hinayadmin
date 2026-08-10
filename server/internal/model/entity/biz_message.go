// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// BizMessage is the golang structure for table biz_message.
type BizMessage struct {
	Id          uint64      `json:"id"          orm:"id"           description:""`                                //
	Type        int         `json:"type"        orm:"type"         description:"消息类型:1=系统通知,2=私信"`                // 消息类型:1=系统通知,2=私信
	Title       string      `json:"title"       orm:"title"        description:"标题"`                              // 标题
	Content     string      `json:"content"     orm:"content"      description:"内容"`                              // 内容
	Level       int         `json:"level"       orm:"level"        description:"级别:1=普通,2=重要,3=紧急"`               // 级别:1=普通,2=重要,3=紧急
	SenderId    uint64      `json:"senderId"    orm:"sender_id"    description:"发送人ID"`                           // 发送人ID
	TargetScope int         `json:"targetScope" orm:"target_scope" description:"系统通知范围:1=all,2=role,3=user;私信=0"` // 系统通知范围:1=all,2=role,3=user;私信=0
	ReceiverId  uint64      `json:"receiverId"  orm:"receiver_id"  description:"私信接收者ID"`                         // 私信接收者ID
	Status      int         `json:"status"      orm:"status"       description:"状态:1=已发布,0=草稿"`                   // 状态:1=已发布,0=草稿
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"   description:"创建时间"`                            // 创建时间
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"   description:"更新时间"`                            // 更新时间
	DeletedAt   *gtime.Time `json:"deletedAt"   orm:"deleted_at"   description:"删除时间(软删)"`                        // 删除时间(软删)
}
