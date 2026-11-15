// Package sms 阿里云短信 (dysmsapi SendSms, RPC 风格 API 签名, 标准库实现)。
//
// 签名算法 (SignatureVersion 1.0 / SignatureMethod HMAC-SHA1):
//  1. 公共参数 + 业务参数按 key 排序, 逐对 RFC3986 百分号编码后拼 k=v&…
//  2. StringToSign = "POST" + "&" + enc("/") + "&" + enc(query)
//  3. Signature = base64(HMAC-SHA1(StringToSign, AccessKeySecret + "&"))
//
// 应答 Code=="OK" 视为成功 (其余如 isv.BUSINESS_LIMIT_CONTROL 原样上抛)。
package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// aliyunEndpoint API 入口 (国际站换 dysmsapi.ap-southeast-1.aliyuncs.com)。
const aliyunEndpoint = "https://dysmsapi.aliyuncs.com/"

// aliyunAPIVersion SendSms 的 RPC 版本号。
const aliyunAPIVersion = "2017-05-25"

type aliyunProvider struct{}

func (a *aliyunProvider) Name() string { return "aliyun" }

func (a *aliyunProvider) Send(ctx context.Context, cfg map[string]string, phone, code string) error {
	keyID := strings.TrimSpace(cfg["sms.aliyun.access_key_id"])
	secret := strings.TrimSpace(cfg["sms.aliyun.access_key_secret"])
	signName := strings.TrimSpace(cfg["sms.sign_name"])
	templateCode := strings.TrimSpace(cfg["sms.template_id"])
	if keyID == "" || secret == "" || signName == "" || templateCode == "" {
		return fmt.Errorf("阿里云短信未配置完整 (需 access_key/secret/sign_name/template_id)")
	}

	params := map[string]string{
		"Action":           "SendSms",
		"Version":          aliyunAPIVersion,
		"AccessKeyId":      keyID,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   randNonce(),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Format":           "JSON",
		"PhoneNumbers":     strings.TrimPrefix(normalizePhone(phone), "+"),
		"SignName":         signName,
		"TemplateCode":     templateCode,
		"TemplateParam":    fmt.Sprintf(`{"code":%q}`, code),
	}
	query := signAliyunQuery(params, secret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, aliyunEndpoint, strings.NewReader(query))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("阿里云短信 HTTP %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		BizId     string `json:"BizId"`
		RequestId string `json:"RequestId"`
	}
	if jerr := json.Unmarshal(body, &out); jerr != nil {
		return fmt.Errorf("阿里云短信应答解析失败: %s", string(body))
	}
	if out.Code != "OK" {
		return fmt.Errorf("阿里云短信发送失败: %s %s", out.Code, out.Message)
	}
	return nil
}

// signAliyunQuery 排序 + 编码 + 签名, 返回可直接作为请求体的最终 query (含 Signature)。
func signAliyunQuery(params map[string]string, accessKeySecret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(aliyunEncode(k))
		b.WriteByte('=')
		b.WriteString(aliyunEncode(params[k]))
	}
	sortedQuery := b.String()
	stringToSign := "POST&" + aliyunEncode("/") + "&" + aliyunEncode(sortedQuery)
	mac := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return sortedQuery + "&Signature=" + aliyunEncode(signature)
}

// aliyunEncode 阿里云 RPC 规范的 RFC3986 百分号编码:
// 空格→%20、*→%2A、~ 为非保留字符不编码, 其余按大写十六进制。
func aliyunEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
