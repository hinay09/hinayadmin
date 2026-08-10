// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// CasbinRule is the golang structure for table casbin_rule.
type CasbinRule struct {
	Id    uint64 `json:"id"    orm:"id"    description:"主键"`                       // 主键
	Ptype string `json:"ptype" orm:"ptype" description:"策略类型: p / g / g2 ..."`     // 策略类型: p / g / g2 ...
	V0    string `json:"v0"    orm:"v0"    description:"p:sub(role) | g:user"`     // p:sub(role) | g:user
	V1    string `json:"v1"    orm:"v1"    description:"p:obj(path) | g:role"`     // p:obj(path) | g:role
	V2    string `json:"v2"    orm:"v2"    description:"p:act(method) | g:domain"` // p:act(method) | g:domain
	V3    string `json:"v3"    orm:"v3"    description:"预留字段"`                     // 预留字段
	V4    string `json:"v4"    orm:"v4"    description:"预留字段"`                     // 预留字段
	V5    string `json:"v5"    orm:"v5"    description:"预留字段"`                     // 预留字段
}
