// Package v1 消息通知接口契约。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/utility/response"
)

// MessageListReq 管理列表 (管理员)。
type MessageListReq struct {
	g.Meta   `path:"/message" tags:"Message" method:"get" summary:"消息管理列表"`
	Type     *int   `json:"type"     in:"query" dc:"消息类型: 1=系统,2=私信"`
	Keyword  string `json:"keyword"  in:"query"`
	Level    *int   `json:"level"    in:"query"`
	Status   *int   `json:"status"   in:"query"`
	Page     int    `json:"page"     in:"query" d:"1"`
	PageSize int    `json:"pageSize" in:"query" d:"10"`
}

// MessageListRes 列表响应。
type MessageListRes response.PageResult

// MessageInboxReq 我的收件箱。
type MessageInboxReq struct {
	g.Meta   `path:"/message/inbox" tags:"Message" method:"get" summary:"我的收件箱"`
	Type     *int   `json:"type"     in:"query" dc:"过滤类型 1=系统,2=私信; 不传=全部"`
	IsRead   *int   `json:"isRead"   in:"query" dc:"过滤已读: 0=未读,1=已读"`
	Keyword  string `json:"keyword"  in:"query"`
	Page     int    `json:"page"     in:"query" d:"1"`
	PageSize int    `json:"pageSize" in:"query" d:"10"`
}

// MessageInboxRes 收件箱响应。
type MessageInboxRes response.PageResult

// MessageDetailReq 详情 (管理员视角)。
type MessageDetailReq struct {
	g.Meta `path:"/message/{id}" tags:"Message" method:"get" summary:"消息详情"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// MessageDetailRes 详情响应。
type MessageDetailRes struct {
	*model.MessageItem
}

// MessageInboxDetailReq 收件箱阅读详情 (自动标记已读)。
type MessageInboxDetailReq struct {
	g.Meta `path:"/message/inbox/{id}" tags:"Message" method:"get" summary:"阅读消息(自动已读)"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// MessageInboxDetailRes 阅读详情响应。
type MessageInboxDetailRes struct {
	*model.MessageItem
}

// MessageSystemCreateReq 发布系统通知。
type MessageSystemCreateReq struct {
	g.Meta      `path:"/message/system" tags:"Message" method:"post" summary:"发布系统通知"`
	Title       string   `json:"title"       v:"required|length:1,128#请输入标题|标题长度 1-128"`
	Content     string   `json:"content"     v:"required#请输入内容"`
	Level       int      `json:"level"       d:"1"`
	Status      int      `json:"status"      d:"1"`
	TargetScope int      `json:"targetScope" v:"required|in:1,2,3#请选择发布范围|范围取值仅能为 1/2/3"`
	TargetIds   []uint64 `json:"targetIds"   dc:"指定角色或用户的ID集合, 范围=1时可空"`
}

// MessageSystemCreateRes 响应。
type MessageSystemCreateRes struct {
	Id uint64 `json:"id"`
}

// MessagePrivateCreateReq 发送私信。
type MessagePrivateCreateReq struct {
	g.Meta     `path:"/message/private" tags:"Message" method:"post" summary:"发送私信"`
	Title      string `json:"title"      v:"required|length:1,128"`
	Content    string `json:"content"    v:"required"`
	Level      int    `json:"level"      d:"1"`
	ReceiverId uint64 `json:"receiverId" v:"required|min:1#请选择接收人|接收人无效"`
}

// MessagePrivateCreateRes 响应。
type MessagePrivateCreateRes struct {
	Id uint64 `json:"id"`
}

// MessageDeleteReq 管理员物理(软)删除消息。
type MessageDeleteReq struct {
	g.Meta `path:"/message/{id}" tags:"Message" method:"delete" summary:"管理员删除消息"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// MessageDeleteRes 响应。
type MessageDeleteRes struct{}

// MessageInboxDeleteReq 个人删除 (从我的收件箱视角隐藏)。
type MessageInboxDeleteReq struct {
	g.Meta `path:"/message/inbox/{id}" tags:"Message" method:"delete" summary:"从我的收件箱中删除"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// MessageInboxDeleteRes 响应。
type MessageInboxDeleteRes struct{}

// MessageReadReq 标记某条为已读。
type MessageReadReq struct {
	g.Meta `path:"/message/{id}/read" tags:"Message" method:"put" summary:"标记为已读"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// MessageReadRes 响应。
type MessageReadRes struct{}

// MessageReadAllReq 标记全部为已读。
type MessageReadAllReq struct {
	g.Meta `path:"/message/read-all" tags:"Message" method:"put" summary:"全部标记为已读"`
}

// MessageReadAllRes 响应。
type MessageReadAllRes struct {
	Affected int64 `json:"affected"`
}

// MessageUnreadCountReq 未读总数。
type MessageUnreadCountReq struct {
	g.Meta `path:"/message/unread-count" tags:"Message" method:"get" summary:"未读消息数量"`
}

// MessageUnreadCountRes 响应。
type MessageUnreadCountRes struct {
	Total   int64 `json:"total"`
	System  int64 `json:"system"`
	Private int64 `json:"private"`
}
