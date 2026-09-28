// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

// Package service 在线会话服务接口。
package service

import (
	"context"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/model"
)

type IOnline interface {
	// Register 登录成功后注册在线会话 (尽力而为)。
	Register(ctx context.Context, token string, sess model.OnlineSession)
	// Touch 刷新会话活跃时间 (节流, 由 Auth 中间件异步调用)。
	Touch(ctx context.Context, token string)
	// Remove 会话登出时剔除。
	Remove(ctx context.Context, token string)
	// List 在线用户分页列表 (按最近活跃倒序, 惰性清理过期会话)。
	List(ctx context.Context, req *v1.OnlineListReq) (res *v1.OnlineListRes, err error)
	// Kick 强制下线指定会话。
	Kick(ctx context.Context, req *v1.OnlineKickReq) (res *v1.OnlineKickRes, err error)
}

var localOnline IOnline

func Online() IOnline {
	if localOnline == nil {
		panic("implement not found for interface IOnline, forgot register?")
	}
	return localOnline
}

func RegisterOnline(i IOnline) {
	localOnline = i
}
