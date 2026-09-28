package message

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	v1 "hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/logic/notify"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/xerror"
)

// MessageEvents 消息事件流 (SSE 长连接)。
// 行为: 连接建立即下发 hello(当前未读数) -> 新消息事件实时推送 -> 25s 心跳注释帧保活。
// 断开: 客户端断连时请求 ctx 结束, select 退出并注销订阅。
func (c *ControllerV1) MessageEvents(ctx context.Context, req *v1.MessageEventsReq) (res *v1.MessageEventsRes, err error) {
	r := g.RequestFromCtx(ctx)
	user := contextx.LoginUser(ctx)
	if user == nil {
		return nil, nil
	}

	// 先占连接配额 (每用户/全局上限), 超限在写流响应头之前以 429 拒绝,
	// 客户端据此停止重连并回落轮询 (MessageBell)。
	ch, cancel, serr := notify.Subscribe(user.UserId)
	if serr != nil {
		r.Response.WriteStatus(http.StatusTooManyRequests, g.Map{
			"code":    xerror.CodeTooManyReq.Code(),
			"message": serr.Error(),
			"data":    nil,
		})
		return nil, nil
	}
	defer cancel()

	// SSE 响应头; X-Accel-Buffering 告知 nginx 禁用代理缓冲
	r.Response.Header().Set("Content-Type", "text/event-stream")
	r.Response.Header().Set("Cache-Control", "no-cache")
	r.Response.Header().Set("X-Accel-Buffering", "no")

	write := func(event string, payload any) {
		data, _ := json.Marshal(payload)
		r.Response.Write(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data))
		r.Response.Flush()
	}

	// 连接建立: 立即下发当前未读数, 客户端无需再发一次查询
	if cnt, cerr := service.Message().UnreadCount(ctx, &v1.MessageUnreadCountReq{}); cerr == nil {
		write("hello", cnt)
	}

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	// 单连接最长存活 1 小时: 到点收流, 客户端退避重连。
	// 目的: 周期性回收僵尸连接(客户端断开未被感知的场景), 释放连接配额。
	maxAge := time.NewTimer(time.Hour)
	defer maxAge.Stop()
	streamCtx := r.Context()
	for {
		select {
		case <-streamCtx.Done():
			return nil, nil
		case <-maxAge.C:
			return nil, nil
		case ev := <-ch:
			write("message", ev)
		case <-heartbeat.C:
			// SSE 注释帧, 仅保活不下发业务事件
			r.Response.Write(": ping\n\n")
			r.Response.Flush()
		}
	}
}
