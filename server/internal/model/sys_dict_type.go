package model

import "github.com/gogf/gf/v2/os/gtime"

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
