// Package notify 出站通知渠道: 站内信之外的可选触达方式 (邮件/群机器人)。
//
// 挂载点: logic/message 的 SystemCreate —— 站内信落库与 SSE 推送之后按渠道
// 配置分发 (异步尽力而为, 失败仅记日志, 不影响消息本身)。
// 配置: sys_config 平台级 notify.* 键 (全局配置页维护, 修改即时生效):
//
//	notify.email.enable / .host / .port / .username / .password / .from
//	notify.webhook.dingtalk.enable / .url / .secret   (secret=加签密钥, 可空)
//	notify.webhook.wecom.enable / .url
//
// 触达语义:
//   - email 定向渠道: 逐接收人查 sys_user.email (租户上下文自动限界), 有邮箱才投递;
//   - webhook 广播渠道: 钉钉/企微群机器人, 每条消息投递一次到配置的群。
//
// 扩展: 实现 Channel 接口并注册到 channels 即可, 无需改动任何业务代码。
package notify

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/logic/configx"
)

// OutMessage 出站消息 (渠道投递视角)。
type OutMessage struct {
	Title   string
	Content string
	Level   int
	All     bool     // 全员通知 (TargetScope=1)
	UserIds []uint64 // 定向接收人 (All=false 时有效, 已按角色展开为用户)
}

// Channel 出站通知渠道。
type Channel interface {
	// Name 渠道标识 (日志/诊断用)。
	Name() string
	// Enabled 配置开关 + 必要参数是否就绪 (cfg 为 notify.* 平台配置快照)。
	Enabled(cfg map[string]string) bool
	// Send 投递一条消息 (实现自行控制单条超时, 返回 error 仅用于日志)。
	Send(ctx context.Context, cfg map[string]string, msg OutMessage) error
}

// channels 已注册渠道 (顺序即分发顺序)。
var channels = []Channel{
	&emailChannel{},
	&webhookChannel{kind: "dingtalk"},
	&webhookChannel{kind: "wecom"},
}

// channelSendTimeout 单渠道单条消息投递超时 (整体兜底, 实现内可更短)。
const channelSendTimeout = 30 * time.Second

// DispatchChannels 按平台配置把消息分发到启用的渠道 (并发投递, 尽力而为)。
// 调用方应传入不可取消的 ctx (后台语义); 接收人解析在租户上下文中进行。
func DispatchChannels(ctx context.Context, msg OutMessage) {
	cfg := configx.PlatformPrefix(ctx, "notify.")
	if len(cfg) == 0 {
		return
	}
	for _, ch := range channels {
		if !ch.Enabled(cfg) {
			continue
		}
		ch := ch
		go func() {
			cctx, cancel := context.WithTimeout(ctx, channelSendTimeout)
			defer cancel()
			if err := ch.Send(cctx, cfg, msg); err != nil {
				g.Log().Warningf(cctx, "通知渠道 %s 投递失败(尽力而为, 不重试): %v", ch.Name(), err)
			}
		}()
	}
}
