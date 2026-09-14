package model

import "github.com/gogf/gf/v2/os/gtime"

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
