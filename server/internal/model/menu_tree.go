package model

// MenuTree 含 children 的树形菜单。
type MenuTree struct {
	SysMenu
	Children []*MenuTree `json:"children"`
}
