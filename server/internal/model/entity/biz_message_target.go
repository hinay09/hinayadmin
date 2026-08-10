// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// BizMessageTarget is the golang structure for table biz_message_target.
type BizMessageTarget struct {
	Id         uint64 `json:"id"         orm:"id"          description:""`                   //
	MessageId  uint64 `json:"messageId"  orm:"message_id"  description:"消息ID"`               // 消息ID
	TargetType int    `json:"targetType" orm:"target_type" description:"目标类型:2=role,3=user"` // 目标类型:2=role,3=user
	TargetId   uint64 `json:"targetId"   orm:"target_id"   description:"角色ID或用户ID"`          // 角色ID或用户ID
}
