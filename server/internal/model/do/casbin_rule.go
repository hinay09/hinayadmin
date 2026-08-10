// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// CasbinRule is the golang structure of table casbin_rule for DAO operations like Where/Data.
type CasbinRule struct {
	g.Meta `orm:"table:casbin_rule, do:true"`
	Id     any // 主键
	Ptype  any // 策略类型: p / g / g2 ...
	V0     any // p:sub(role) | g:user
	V1     any // p:obj(path) | g:role
	V2     any // p:act(method) | g:domain
	V3     any // 预留字段
	V4     any // 预留字段
	V5     any // 预留字段
}
