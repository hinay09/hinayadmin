// Package response 统一响应辅助。
package response

// Result 统一响应结构 (与 GoFrame DefaultHandlerResponse 字段一致, 便于直接复用 ghttp.MiddlewareHandlerResponse)。
type Result struct {
	Code    int    `json:"code"    dc:"业务码, 0 表示成功"`
	Message string `json:"message" dc:"提示信息"`
	Data    any    `json:"data"    dc:"业务数据"`
}

// PageResult 分页通用响应数据。
type PageResult struct {
	List     any   `json:"list"     dc:"列表数据"`
	Total    int64 `json:"total"    dc:"记录总数"`
	Page     int   `json:"page"     dc:"当前页"`
	PageSize int   `json:"pageSize" dc:"每页大小"`
}

// Page 构造分页结果。
func Page(list any, total int64, page, pageSize int) PageResult {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return PageResult{List: list, Total: total, Page: page, PageSize: pageSize}
}
