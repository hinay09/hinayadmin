// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish to manually maintain this interface file.
// ================================================================================

package service

import (
	"github.com/gogf/gf/v2/net/ghttp"
)

type (
	IWechat interface {
		// Callback 微信公众号服务器回调: GET 验签 + POST 消息接收与回复 (占位: Echo 回复)。
		// 绑定在根路由 /wechat/callback, 不走鉴权 (微信服务器直接调用)。
		Callback(r *ghttp.Request)
	}
)

var (
	localWechat IWechat
)

func Wechat() IWechat {
	if localWechat == nil {
		panic("implement not found for interface IWechat, forgot register?")
	}
	return localWechat
}

func RegisterWechat(i IWechat) {
	localWechat = i
}
