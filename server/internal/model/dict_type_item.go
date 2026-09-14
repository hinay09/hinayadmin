package model

import "github.com/gogf/gf/v2/os/gtime"

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
