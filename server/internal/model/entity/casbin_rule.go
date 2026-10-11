// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// CasbinRule is the golang structure for table casbin_rule.
type CasbinRule struct {
	Id    uint64 `json:"id"    orm:"id"    description:"主键"`                             // 主键
	Ptype string `json:"ptype" orm:"ptype" description:"策略类型: p / g / g2 ..."`           // 策略类型: p / g / g2 ...
	V0    string `json:"v0"    orm:"v0"    description:"p:sub(角色ID) | g:用户ID"`           // p:sub(角色ID) | g:用户ID
	V1    string `json:"v1"    orm:"v1"    description:"p:obj(path/menu:菜单ID) | g:角色ID"` // p:obj(path/menu:菜单ID) | g:角色ID
	V2    string `json:"v2"    orm:"v2"    description:"p:act(method) | g:domain(租户ID)"` // p:act(method) | g:domain(租户ID)
	V3    string `json:"v3"    orm:"v3"    description:"p:domain(租户ID)"`                 // p:domain(租户ID)
	V4    string `json:"v4"    orm:"v4"    description:"预留字段"`                           // 预留字段
	V5    string `json:"v5"    orm:"v5"    description:"预留字段"`                           // 预留字段
}
