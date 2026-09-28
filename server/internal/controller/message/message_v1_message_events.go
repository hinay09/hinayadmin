package message

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	v1 "hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/logic/notify"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
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

	ch, cancel := notify.Subscribe(user.UserId)
	defer cancel()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	streamCtx := r.Context()
	for {
		select {
		case <-streamCtx.Done():
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
