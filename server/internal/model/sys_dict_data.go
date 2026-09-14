package model

import "github.com/gogf/gf/v2/os/gtime"

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
