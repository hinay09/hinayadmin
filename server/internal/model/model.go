// Package model 业务通用 DTO/VO/Entity 定义。
// 为避免引入 gf gen dao 步骤, 这里集中维护核心实体与查询结构。
package model

import "github.com/gogf/gf/v2/os/gtime"

// LoginUser 上下文中的登录用户信息。
type LoginUser struct {
	UserId   uint64   `json:"userId"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar"`
	Email    string   `json:"email"`
	Phone    string   `json:"phone"`
	Roles    []string `json:"roles"`
}

// SysUser 数据库实体。
type SysUser struct {
	Id        uint64      `json:"id"        orm:"id"`
	Username  string      `json:"username"  orm:"username"`
	Password  string      `json:"-"         orm:"password"`
	Nickname  string      `json:"nickname"  orm:"nickname"`
	Avatar    string      `json:"avatar"    orm:"avatar"`
	Email     string      `json:"email"     orm:"email"`
	Phone     string      `json:"phone"     orm:"phone"`
	OrgId     uint64      `json:"orgId"     orm:"org_id"`
	Status    int         `json:"status"    orm:"status"`
	Remark    string      `json:"remark"    orm:"remark"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

// SysOrg 组织机构实体。
type SysOrg struct {
	Id        uint64      `json:"id"        orm:"id"`
	ParentId  uint64      `json:"parentId"  orm:"parent_id"`
	Name      string      `json:"name"      orm:"name"`
	Leader    string      `json:"leader"    orm:"leader"`
	Phone     string      `json:"phone"     orm:"phone"`
	Email     string      `json:"email"     orm:"email"`
	Sort      int         `json:"sort"      orm:"sort"`
	Status    int         `json:"status"    orm:"status"`
	Remark    string      `json:"remark"    orm:"remark"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

// OrgTree 含 children 的树形组织。
type OrgTree struct {
	SysOrg
	Children []*OrgTree `json:"children"`
}

// SysRole 角色实体。
type SysRole struct {
	Id        uint64      `json:"id"        orm:"id"`
	Name      string      `json:"name"      orm:"name"`
	Code      string      `json:"code"      orm:"code"`
	Sort      int         `json:"sort"      orm:"sort"`
	Status    int         `json:"status"    orm:"status"`
	Remark    string      `json:"remark"    orm:"remark"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

// SysMenu 菜单实体。
type SysMenu struct {
	Id         uint64      `json:"id"         orm:"id"`
	ParentId   uint64      `json:"parentId"   orm:"parent_id"`
	Name       string      `json:"name"       orm:"name"`
	Type       int         `json:"type"       orm:"type"`
	Path       string      `json:"path"       orm:"path"`
	Component  string      `json:"component"  orm:"component"`
	Icon       string      `json:"icon"       orm:"icon"`
	Permission string      `json:"permission" orm:"permission"`
	Sort       int         `json:"sort"       orm:"sort"`
	Visible    int         `json:"visible"    orm:"visible"`
	Status     int         `json:"status"     orm:"status"`
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"`
	UpdatedAt  *gtime.Time `json:"updatedAt"  orm:"updated_at"`
}

// MenuTree 含 children 的树形菜单。
type MenuTree struct {
	SysMenu
	Children []*MenuTree `json:"children"`
}

// 消息通知: 类型与范围枚举。
const (
	MessageTypeSystem  = 1 // 系统通知
	MessageTypePrivate = 2 // 私信通知

	MessageScopeNone = 0 // 私信专用
	MessageScopeAll  = 1 // 全员
	MessageScopeRole = 2 // 指定角色
	MessageScopeUser = 3 // 指定用户

	MessageTargetTypeRole = 2
	MessageTargetTypeUser = 3
)

// BizMessage 消息通知主表实体。
type BizMessage struct {
	Id          uint64      `json:"id"          orm:"id"`
	Type        int         `json:"type"        orm:"type"`
	Title       string      `json:"title"       orm:"title"`
	Content     string      `json:"content"     orm:"content"`
	Level       int         `json:"level"       orm:"level"`
	SenderId    uint64      `json:"senderId"    orm:"sender_id"`
	TargetScope int         `json:"targetScope" orm:"target_scope"`
	ReceiverId  uint64      `json:"receiverId"  orm:"receiver_id"`
	Status      int         `json:"status"      orm:"status"`
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"`
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"`
}

// BizMessageTarget 系统通知定向目标。
type BizMessageTarget struct {
	Id         uint64 `json:"id"         orm:"id"`
	MessageId  uint64 `json:"messageId"  orm:"message_id"`
	TargetType int    `json:"targetType" orm:"target_type"`
	TargetId   uint64 `json:"targetId"   orm:"target_id"`
}

// BizMessageRead 消息已读关系。
type BizMessageRead struct {
	Id        uint64      `json:"id"        orm:"id"`
	MessageId uint64      `json:"messageId" orm:"message_id"`
	UserId    uint64      `json:"userId"    orm:"user_id"`
	ReadAt    *gtime.Time `json:"readAt"    orm:"read_at"`
	Hidden    int         `json:"hidden"    orm:"hidden"`
}

// MessageItem 消息列表/详情 VO (面向接口)。
type MessageItem struct {
	Id              uint64      `json:"id"`
	Type            int         `json:"type"`
	Title           string      `json:"title"`
	Content         string      `json:"content"`
	Level           int         `json:"level"`
	Status          int         `json:"status"`
	SenderId        uint64      `json:"senderId"`
	SenderName      string      `json:"senderName"`
	TargetScope     int         `json:"targetScope"`
	ReceiverId      uint64      `json:"receiverId"`
	ReceiverName    string      `json:"receiverName"`
	TargetRoleIds   []uint64    `json:"targetRoleIds"`
	TargetUserIds   []uint64    `json:"targetUserIds"`
	TargetRoleNames []string    `json:"targetRoleNames"`
	IsRead          bool        `json:"isRead"`
	CreatedAt       *gtime.Time `json:"createdAt"`
	UpdatedAt       *gtime.Time `json:"updatedAt"`
}

// SysAuditLog 操作日志实体。
type SysAuditLog struct {
	Id         uint64      `json:"id"         orm:"id"`
	UserId     uint64      `json:"userId"     orm:"user_id"`
	Username   string      `json:"username"   orm:"username"`
	Action     string      `json:"action"     orm:"action"`
	Resource   string      `json:"resource"   orm:"resource"`
	ResourceId string      `json:"resourceId" orm:"resource_id"`
	Detail     string      `json:"detail"     orm:"detail"`
	Ip         string      `json:"ip"         orm:"ip"`
	UserAgent  string      `json:"userAgent"  orm:"user_agent"`
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"`
}

// SysDictType 字典类型实体。
type SysDictType struct {
	Id        uint64      `json:"id"        orm:"id"`
	TypeCode  string      `json:"typeCode"  orm:"type_code"`
	TypeName  string      `json:"typeName"  orm:"type_name"`
	Status    int         `json:"status"    orm:"status"`
	Remark    string      `json:"remark"    orm:"remark"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

// SysDictData 字典数据实体。
type SysDictData struct {
	Id        uint64      `json:"id"         orm:"id"`
	TypeId    uint64      `json:"typeId"     orm:"type_id"`
	DictLabel string      `json:"dictLabel"  orm:"dict_label"`
	DictValue string      `json:"dictValue"  orm:"dict_value"`
	Sort      int         `json:"sort"       orm:"sort"`
	Status    int         `json:"status"     orm:"status"`
	Remark    string      `json:"remark"     orm:"remark"`
	CreatedAt *gtime.Time `json:"createdAt"  orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt"  orm:"updated_at"`
}

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

// AuditLogItem 操作日志列表 VO。
type AuditLogItem struct {
	Id         uint64      `json:"id"`
	UserId     uint64      `json:"userId"`
	Username   string      `json:"username"`
	Action     string      `json:"action"`
	Resource   string      `json:"resource"`
	ResourceId string      `json:"resourceId"`
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	StatusCode int         `json:"statusCode"`
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	DurationMs int64       `json:"durationMs"`
	RequestId  string      `json:"requestId"`
	Detail     string      `json:"detail"`
	Ip         string      `json:"ip"`
	UserAgent  string      `json:"userAgent"`
	CreatedAt  *gtime.Time `json:"createdAt"`
}

// DictTypeItem 字典类型列表 VO。
type DictTypeItem struct {
	Id        uint64      `json:"id"`
	TypeCode  string      `json:"typeCode"`
	TypeName  string      `json:"typeName"`
	Status    int         `json:"status"`
	Remark    string      `json:"remark"`
	CreatedAt *gtime.Time `json:"createdAt"`
	UpdatedAt *gtime.Time `json:"updatedAt"`
}

// DictDataItem 字典数据列表 VO。
type DictDataItem struct {
	Id        uint64      `json:"id"`
	TypeId    uint64      `json:"typeId"`
	DictLabel string      `json:"dictLabel"`
	DictValue string      `json:"dictValue"`
	Sort      int         `json:"sort"`
	Status    int         `json:"status"`
	Remark    string      `json:"remark"`
	CreatedAt *gtime.Time `json:"createdAt"`
	UpdatedAt *gtime.Time `json:"updatedAt"`
}

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

// SysConfig 全局配置实体。
type SysConfig struct {
	Id          uint64      `json:"id"          orm:"id"`
	ConfigKey   string      `json:"configKey"   orm:"config_key"`
	ConfigValue string      `json:"configValue" orm:"config_value"`
	ConfigType  int         `json:"configType"  orm:"config_type"`
	Name        string      `json:"name"        orm:"name"`
	Remark      string      `json:"remark"      orm:"remark"`
	Status      int         `json:"status"      orm:"status"`
	Sort        int         `json:"sort"        orm:"sort"`
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"`
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"`
}

// ConfigItem 全局配置列表 VO。
type ConfigItem struct {
	Id          uint64      `json:"id"`
	ConfigKey   string      `json:"configKey"`
	ConfigValue string      `json:"configValue"`
	ConfigType  int         `json:"configType"`
	Name        string      `json:"name"`
	Remark      string      `json:"remark"`
	Status      int         `json:"status"`
	Sort        int         `json:"sort"`
	CreatedAt   *gtime.Time `json:"createdAt"`
	UpdatedAt   *gtime.Time `json:"updatedAt"`
}
