// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/auth/v1"
)

type (
	IAuth interface {
		// PublicKey 生成一次性登录加密公钥。
		// 每次调用生成全新 RSA 密钥对, 私钥存 Redis 并设置 TTL, 用后即毁;
		// 按 IP 限流防止匿名端点被刷导致密钥生成 DoS。
		PublicKey(ctx context.Context, req *v1.PublicKeyReq) (res *v1.PublicKeyRes, err error)
		// Login 用户名密码登录。
		// 密码为前端用一次性公钥加密的 RSA 密文, 服务端解密后再走 bcrypt 校验。
		// 带 IP+用户名 双维度失败计数防暴力破解: 窗口内失败超过 consts.LoginFailMax 次后临时锁定。
		Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error)
		// Refresh 使用当前有效 token 续签新 token, 旧 token 加入黑名单。
		Refresh(ctx context.Context, req *v1.RefreshReq) (res *v1.RefreshRes, err error)
		// Logout 写入 Redis 黑名单。
		Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error)
		// UserInfo 当前登录用户信息。
		UserInfo(ctx context.Context, req *v1.UserInfoReq) (res *v1.UserInfoRes, err error)
		// Profile 个人中心-获取详情。
		Profile(ctx context.Context, req *v1.ProfileReq) (res *v1.ProfileRes, err error)
		// UpdateProfile 修改当前用户基础信息 (昵称/头像/邮箱/手机)。
		UpdateProfile(ctx context.Context, req *v1.UpdateProfileReq) (res *v1.UpdateProfileRes, err error)
		// ChangePassword 修改当前用户密码 (原密码校验 + bcrypt 重新生成)。
		ChangePassword(ctx context.Context, req *v1.ChangePasswordReq) (res *v1.ChangePasswordRes, err error)
		// UploadAvatar 上传并更新当前用户头像。
		UploadAvatar(ctx context.Context, req *v1.UploadAvatarReq) (res *v1.UploadAvatarRes, err error)
		// MenuTree 当前用户可见菜单树 (排除按钮类型)。
		MenuTree(ctx context.Context, req *v1.MenuTreeReq) (res *v1.MenuTreeRes, err error)
	}
)

var (
	localAuth IAuth
)

func Auth() IAuth {
	if localAuth == nil {
		panic("implement not found for interface IAuth, forgot register?")
	}
	return localAuth
}

func RegisterAuth(i IAuth) {
	localAuth = i
}
