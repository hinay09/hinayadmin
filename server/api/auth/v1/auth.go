// Package v1 鉴权相关接口定义。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"hinay.cn/admin/internal/model"
)

// PublicKeyReq 获取一次性登录加密公钥。
type PublicKeyReq struct {
	g.Meta `path:"/auth/public-key" tags:"Auth" method:"get" summary:"获取登录加密公钥"`
}

// PublicKeyRes 公钥响应。
type PublicKeyRes struct {
	KeyId     string `json:"keyId"     dc:"密钥标识, 登录时随密文一并提交"`
	PublicKey string `json:"publicKey" dc:"RSA 公钥(PEM), 用于加密登录密码"`
}

// LoginReq 登录请求。
// 密码须先用 PublicKey 接口返回的公钥加密(RSA PKCS#1 v1.5, base64),
// 每个密钥对仅可使用一次, 过期或已使用需重新获取。
type LoginReq struct {
	g.Meta   `path:"/auth/login" tags:"Auth" method:"post" summary:"登录"`
	Username string `v:"required#请输入账号" json:"username" dc:"账号"`
	Password string `v:"required#请输入密码" json:"password" dc:"RSA 加密后的密码密文(base64)"`
	KeyId    string `v:"required#缺少加密密钥标识" json:"keyId" dc:"公钥标识"`
}

// LoginRes 登录响应。
type LoginRes struct {
	Token    string           `json:"token"    dc:"JWT token"`
	ExpireAt int64            `json:"expireAt" dc:"过期时间戳(秒)"`
	UserInfo *model.LoginUser `json:"userInfo" dc:"用户信息"`
}

// RefreshReq 刷新 Token 请求 (要求当前 token 有效)。
type RefreshReq struct {
	g.Meta `path:"/auth/refresh" tags:"Auth" method:"post" summary:"刷新Token"`
}

// RefreshRes 刷新 Token 响应。
type RefreshRes struct {
	Token    string `json:"token"    dc:"新的 JWT token"`
	ExpireAt int64  `json:"expireAt" dc:"过期时间戳(秒)"`
}

// LogoutReq 登出。
type LogoutReq struct {
	g.Meta `path:"/auth/logout" tags:"Auth" method:"post" summary:"登出"`
}

// LogoutRes 登出响应。
type LogoutRes struct{}

// UserInfoReq 获取当前登录用户信息。
type UserInfoReq struct {
	g.Meta `path:"/auth/userInfo" tags:"Auth" method:"get" summary:"当前用户信息"`
}

// UserInfoRes 当前登录用户信息。
type UserInfoRes struct {
	*model.LoginUser
}

// MenuTreeReq 当前用户菜单树。
type MenuTreeReq struct {
	g.Meta `path:"/auth/menus" tags:"Auth" method:"get" summary:"当前用户菜单"`
}

// MenuTreeRes 菜单树响应。
type MenuTreeRes struct {
	Menus       []*model.MenuTree `json:"menus"       dc:"菜单树"`
	Permissions []string          `json:"permissions" dc:"权限标识列表"`
}

// ProfileReq 个人中心-获取详情。
type ProfileReq struct {
	g.Meta `path:"/auth/profile" tags:"Auth" method:"get" summary:"个人中心详情"`
}

// ProfileRes 个人中心详情。
type ProfileRes struct {
	*model.LoginUser
}

// UpdateProfileReq 修改个人基础信息。
type UpdateProfileReq struct {
	g.Meta   `path:"/auth/profile" tags:"Auth" method:"put" summary:"修改个人资料"`
	Nickname string `json:"nickname" v:"required#请输入昵称"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"    v:"email#邮箱格式不正确"`
	Phone    string `json:"phone"    v:"phone#手机号格式不正确"`
}

// UpdateProfileRes 修改个人资料响应。
type UpdateProfileRes struct{}

// ChangePasswordReq 修改密码。
type ChangePasswordReq struct {
	g.Meta      `path:"/auth/password" tags:"Auth" method:"put" summary:"修改密码"`
	OldPassword string `json:"oldPassword" v:"required#请输入原密码"`
	NewPassword string `json:"newPassword" v:"required|length:6,32#请输入新密码|密码长度 6-32"`
}

// ChangePasswordRes 修改密码响应。
type ChangePasswordRes struct{}

// UploadAvatarReq 上传头像请求。
type UploadAvatarReq struct {
	g.Meta `path:"/auth/avatar" tags:"Auth" method:"post" summary:"上传头像"`
	File   *ghttp.UploadFile `json:"file" v:"required#请选择头像文件"`
}

// UploadAvatarRes 上传头像响应。
type UploadAvatarRes struct {
	Url string `json:"url" dc:"头像访问路径"`
}
