// Package wechat 微信公众号对接 (占位实现, 基于 silenceper/wechat/v2)。
//
// 能力: 服务器回调验签(GET) + 消息接收/回复(POST), 消息处理为 Echo 占位
// (回复用户发送的文本; 非文本消息回复提示)。
// 配置 (sys_config, 管理员在 全局配置 页维护; 修改后自动重建客户端):
//
//	wechat.app_id           公众号 AppID (必填)
//	wechat.app_secret       公众号 AppSecret (必填)
//	wechat.token            服务器 Token (必填)
//	wechat.encoding_aes_key 消息加解密密钥 (可空, 明文模式不填)
//
// access_token 缓存默认用内存 (单实例); 多副本部署可换 redis cache (见库文档)。
package wechat

import (
	"errors"
	"strings"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/officialaccount"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/message"

	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/service"
)

func init() {
	service.RegisterWechat(&sWechat{})
}

// errNotConfigured 公众号参数未配置齐全。
var errNotConfigured = errors.New("wechat not configured: 请管理员在 全局配置 中填写 wechat.app_id / wechat.app_secret / wechat.token")

type sWechat struct {
	mu    sync.Mutex
	built string // 已构建客户端的配置指纹 (变更时重建, 保住 access_token 缓存)
	oa    *officialaccount.OfficialAccount
}

// Callback 微信回调入口 (GET 验签 / POST 消息)。
func (s *sWechat) Callback(r *ghttp.Request) {
	oa, err := s.account()
	if err != nil {
		g.Log().Warningf(r.Context(), "[wechat] %v", err)
		r.Response.WriteStatus(503, "wechat not configured")
		return
	}
	// 传入原始 request 与 responseWriter, 由库完成验签/解密/回复
	server := oa.GetServer(r.Request, r.Response.RawWriter())
	server.SetMessageHandler(func(msg *message.MixMessage) *message.Reply {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			content = "(已收到消息)"
		}
		return &message.Reply{MsgType: message.MsgTypeText, MsgData: message.NewText(content)}
	})
	if err = server.Serve(); err != nil {
		g.Log().Warningf(r.Context(), "[wechat] serve: %v", err)
		return
	}
	if err = server.Send(); err != nil {
		g.Log().Warningf(r.Context(), "[wechat] send: %v", err)
	}
}

// account 懒构建公众号客户端; 配置变更 (指纹不同) 时重建。
func (s *sWechat) account() (*officialaccount.OfficialAccount, error) {
	appID := cfgOf("wechat.app_id", "")
	secret := cfgOf("wechat.app_secret", "")
	token := cfgOf("wechat.token", "")
	aesKey := cfgOf("wechat.encoding_aes_key", "")

	if appID == "" || secret == "" || token == "" {
		return nil, errNotConfigured
	}
	fp := appID + "|" + secret + "|" + token + "|" + aesKey

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.oa != nil && s.built == fp {
		return s.oa, nil
	}
	wc := wechat.NewWechat()
	oa := wc.GetOfficialAccount(&offConfig.Config{
		AppID:          appID,
		AppSecret:      secret,
		Token:          token,
		EncodingAESKey: aesKey,
		Cache:          cache.NewMemory(), // 内存缓存 access_token; 多副本部署建议换 redis
	})
	s.oa = oa
	s.built = fp
	return oa, nil
}

// cfgOf 读 sys_config, 空/缺失回落默认。
func cfgOf(key, def string) string {
	v, err := dao.SysConfig.Ctx(nil).
		Where("config_key", key).Where("deleted_at IS NULL").
		Value("config_value")
	if err != nil || v == nil {
		return def
	}
	s := strings.TrimSpace(v.String())
	if s == "" {
		return def
	}
	return s
}
