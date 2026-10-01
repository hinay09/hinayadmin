// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/model"
)

type (
	IOnline interface {
		// Register 登录成功后注册在线会话 (尽力而为: 失败仅记日志, 不影响登录)。
		Register(ctx context.Context, token string, sess model.OnlineSession)
		// Touch 刷新会话活跃时间。
		// 节流: 每个会话 consts.OnlineTouchIntervalSec 内只真正写一次,
		// 由 Auth 中间件异步调用, 任何失败静默忽略 (心跳属于遥测数据, 不允许影响请求)。
		Touch(ctx context.Context, token string)
		// Remove 会话登出时剔除 (黑名单由调用方负责)。
		Remove(ctx context.Context, token string)
		// List 在线用户分页列表 (按最近活跃倒序), 惰性清理已过期会话。
		List(ctx context.Context, req *v1.OnlineListReq) (res *v1.OnlineListRes, err error)
		// Kick 强制下线: 会话 token 写入黑名单并从在线列表剔除。
		Kick(ctx context.Context, req *v1.OnlineKickReq) (res *v1.OnlineKickRes, err error)
	}
)

var (
	localOnline IOnline
)

func Online() IOnline {
	if localOnline == nil {
		panic("implement not found for interface IOnline, forgot register?")
	}
	return localOnline
}

func RegisterOnline(i IOnline) {
	localOnline = i
}
