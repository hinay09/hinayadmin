// Package notify 钉钉/企微群机器人 webhook 渠道 (标准库实现, 零外部依赖)。
//
// 报文: 两者均支持 text 消息; 应答均含 errcode (0=成功)。
// 钉钉加签 (可选): notify.webhook.dingtalk.secret 配置后按官方算法
// timestamp + "\n" + secret 的 HMAC-SHA256 追加 &timestamp=&sign= 查询参数。
package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// webhookChannel 群机器人广播渠道 (kind: dingtalk / wecom)。
type webhookChannel struct {
	kind string
}

// webhookTextMax 正文最大字节数 (企微 text 上限 2048, 统一按 2000 截断)。
const webhookTextMax = 2000

// webhookHTTPTimeout 单次投递 HTTP 超时。
const webhookHTTPTimeout = 10 * time.Second

// webhookClient 投递专用客户端 (带超时, 禁止跟随重定向避免机器人地址变更后被带偏)。
var webhookClient = &http.Client{Timeout: webhookHTTPTimeout}

func (w *webhookChannel) Name() string { return "webhook_" + w.kind }

func (w *webhookChannel) Enabled(cfg map[string]string) bool {
	return cfgBool(cfg["notify.webhook."+w.kind+".enable"]) &&
		strings.TrimSpace(cfg["notify.webhook."+w.kind+".url"]) != ""
}

func (w *webhookChannel) Send(ctx context.Context, cfg map[string]string, msg OutMessage) error {
	endpoint := strings.TrimSpace(cfg["notify.webhook."+w.kind+".url"])
	if w.kind == "dingtalk" {
		if secret := strings.TrimSpace(cfg["notify.webhook.dingtalk.secret"]); secret != "" {
			endpoint = signDingTalk(endpoint, secret)
		}
	}
	payload, err := json.Marshal(map[string]any{
		"msgtype": "text",
		"text":    map[string]string{"content": truncateBytes(msg.Title+"\n"+msg.Content, webhookTextMax)},
	})
	if err != nil {
		return err
	}
	return postWebhook(ctx, endpoint, payload)
}

// signDingTalk 钉钉加签 (当前时间戳)。
func signDingTalk(endpoint, secret string) string {
	return signDingTalkAt(endpoint, secret, time.Now().UnixMilli())
}

// signDingTalkAt 钉钉加签纯函数: HMAC-SHA256(timestamp+"\n"+secret, secret) base64 后 URL 编码,
// 以 timestamp=…&sign=… 追加到 webhook 地址 (已有查询串用 & 衔接)。
func signDingTalkAt(endpoint, secret string, ts int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10) + "\n" + secret))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	sep := "&"
	if !strings.Contains(endpoint, "?") {
		sep = "?"
	}
	return endpoint + sep + "timestamp=" + strconv.FormatInt(ts, 10) + "&sign=" + sign
}

// postWebhook POST JSON 并校验平台 errcode。
func postWebhook(ctx context.Context, endpoint string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := webhookClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if jerr := json.Unmarshal(body, &out); jerr == nil && out.ErrCode != 0 {
		return fmt.Errorf("errcode=%d: %s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

// truncateBytes 按字节截断 (不切断多字节字符的尾部残字节)。
func truncateBytes(s string, max int) string {
	b := []byte(s)
	if len(b) <= max {
		return s
	}
	cut := b[:max]
	// 回退到最后一个完整 UTF-8 序列边界
	for len(cut) > 0 && !utf8RuneStart(cut[len(cut)-1]) {
		cut = cut[:len(cut)-1]
	}
	return string(cut) + "…"
}

// utf8RuneStart 字节是否为 UTF-8 序列首字节 (0xxxxxxx / 11xxxxxx)。
func utf8RuneStart(b byte) bool {
	return b&0x80 == 0 || b&0xC0 == 0xC0
}
