package model

// OrgTree 含 children 的树形组织。
type OrgTree struct {
	SysOrg
	Children []*OrgTree `json:"children"`
}
