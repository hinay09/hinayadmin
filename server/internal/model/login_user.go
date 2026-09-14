package model

// LoginUser 上下文中的登录用户信息。
type LoginUser struct {
	UserId   uint64   `json:"userId"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar"`
	Email    string   `json:"email"`
	Phone    string   `json:"phone"`
	Roles    []string `json:"roles"`
}
