// Package v1 系统管理-字典接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// ============================================================
// 字典类型 (sys_dict_type)
// ============================================================

// DictTypeListReq 字典类型分页列表。
type DictTypeListReq struct {
	g.Meta   `path:"/system/dict-types" tags:"SystemDictType" method:"get" summary:"字典类型分页列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"编码/名称模糊搜索"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// DictTypeListRes 类型列表响应。
type DictTypeListRes response.PageResult

// DictTypeAllReq 获取所有启用的字典类型。
type DictTypeAllReq struct {
	g.Meta `path:"/system/dict-types/all" tags:"SystemDictType" method:"get" summary:"获取所有字典类型"`
}

// DictTypeAllRes 全量类型响应。
type DictTypeAllRes struct {
	List any `json:"list"`
}

// DictTypeCreateReq 新增字典类型。
type DictTypeCreateReq struct {
	g.Meta   `path:"/system/dict-types" tags:"SystemDictType" method:"post" summary:"新增字典类型"`
	TypeCode string `json:"typeCode" v:"required|length:2,64#请输入类型编码|类型编码长度 2-64"`
	TypeName string `json:"typeName" v:"required|length:1,128#请输入类型名称|类型名称长度 1-128"`
	Status   int    `json:"status" d:"1"`
	Remark   string `json:"remark"`
}

// DictTypeCreateRes 新增类型响应。
type DictTypeCreateRes struct {
	Id uint64 `json:"id"`
}

// DictTypeUpdateReq 修改字典类型。
type DictTypeUpdateReq struct {
	g.Meta   `path:"/system/dict-types/{id}" tags:"SystemDictType" method:"put" summary:"修改字典类型"`
	Id       uint64 `json:"id"       in:"path" v:"required"`
	TypeCode string `json:"typeCode" v:"required|length:2,64"`
	TypeName string `json:"typeName" v:"required|length:1,128"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

// DictTypeUpdateRes 修改类型响应。
type DictTypeUpdateRes struct{}

// DictTypeDeleteReq 删除字典类型。
type DictTypeDeleteReq struct {
	g.Meta `path:"/system/dict-types/{id}" tags:"SystemDictType" method:"delete" summary:"删除字典类型"`
	Id     uint64 `json:"id" in:"path" v:"required"`
}

// DictTypeDeleteRes 删除类型响应。
type DictTypeDeleteRes struct{}

// ============================================================
// 字典数据项 (sys_dict_data)
// ============================================================

// DictDataListReq 字典数据项分页列表。
type DictDataListReq struct {
	g.Meta   `path:"/system/dict-types/{typeId}/items" tags:"SystemDict" method:"get" summary:"字典数据项列表"`
	TypeId   uint64 `json:"typeId"   in:"path" v:"required"`
	Keyword  string `json:"keyword"  in:"query" dc:"标签/值模糊搜索"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"50" dc:"页大小"`
}

// DictDataListRes 数据项列表响应。
type DictDataListRes response.PageResult

// DictDataCreateReq 新增字典数据项。
type DictDataCreateReq struct {
	g.Meta    `path:"/system/dict-types/{typeId}/items" tags:"SystemDict" method:"post" summary:"新增字典数据项"`
	TypeId    uint64 `json:"typeId"    in:"path" v:"required"`
	DictLabel string `json:"dictLabel" v:"required|length:1,128#请输入字典标签|字典标签长度 1-128"`
	DictValue string `json:"dictValue" v:"length:0,255"`
	Sort      int    `json:"sort"`
	Status    int    `json:"status" d:"1"`
	Remark    string `json:"remark"`
}

// DictDataCreateRes 新增数据项响应。
type DictDataCreateRes struct {
	Id uint64 `json:"id"`
}

// DictDataUpdateReq 修改字典数据项。
type DictDataUpdateReq struct {
	g.Meta    `path:"/system/dict-types/{typeId}/items/{id}" tags:"SystemDict" method:"put" summary:"修改字典数据项"`
	TypeId    uint64 `json:"typeId"    in:"path" v:"required"`
	Id        uint64 `json:"id"        in:"path" v:"required"`
	DictLabel string `json:"dictLabel" v:"required|length:1,128"`
	DictValue string `json:"dictValue"`
	Sort      int    `json:"sort"`
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
}

// DictDataUpdateRes 修改数据项响应。
type DictDataUpdateRes struct{}

// DictDataDeleteReq 删除字典数据项。
type DictDataDeleteReq struct {
	g.Meta `path:"/system/dict-types/{typeId}/items/{id}" tags:"SystemDict" method:"delete" summary:"删除字典数据项"`
	TypeId uint64 `json:"typeId" in:"path" v:"required"`
	Id     uint64 `json:"id"     in:"path" v:"required"`
}

// DictDataDeleteRes 删除数据项响应。
type DictDataDeleteRes struct{}

// DictDataSortItem 排序项。
type DictDataSortItem struct {
	Id   uint64 `json:"id"`
	Sort int    `json:"sort"`
}

// DictDataSortReq 批量排序字典数据项。
type DictDataSortReq struct {
	g.Meta `path:"/system/dict-types/{typeId}/items/sort" tags:"SystemDict" method:"put" summary:"批量排序字典数据项"`
	TypeId uint64             `json:"typeId" in:"path" v:"required"`
	Items  []DictDataSortItem `json:"items" v:"required"`
}

// DictDataSortRes 排序响应。
type DictDataSortRes struct{}

// ============================================================
// 兼容接口: 全量字典数据(按类型分组)
// ============================================================

// DictAllReq 获取所有启用的字典数据（按类型分组）。
type DictAllReq struct {
	g.Meta `path:"/system/dicts/all" tags:"SystemDict" method:"get" summary:"获取所有字典数据(按类型分组)"`
}

// DictAllRes 全量响应。
type DictAllRes struct {
	List any `json:"list"`
}
