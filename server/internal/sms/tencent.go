// Package sms 腾讯云短信 (SMS v3 SendSms, TC3-HMAC-SHA256 签名, 标准库实现)。
//
// TC3 签名链:
//  1. CanonicalRequest = POST\n/\n\ncontent-type:application/json\nhost:{host}\n\n
//     content-type;host\nsha256hex(body)
//  2. StringToSign = "TC3-HMAC-SHA256\n{timestamp}\n{date}/sms/tc3_request\n" + sha256hex(CanonicalRequest)
//  3. SecretDate=HMAC("TC3"+SecretKey, date) → SecretService=HMAC(date, "sms")
//     → SecretSigning=HMAC(service, "tc3_request") → Signature=hex(HMAC(signing, StringToSign))
//
// 应答 Response.SendStatusSet[0].Code=="Ok" 视为成功。
package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// tencentHost 短信 v3 API 域名。
const tencentHost = "sms.tencentcloudapi.com"

type tencentProvider struct{}

func (t *tencentProvider) Name() string { return "tencent" }

func (t *tencentProvider) Send(ctx context.Context, cfg map[string]string, phone, code string) error {
	secretID := strings.TrimSpace(cfg["sms.tencent.secret_id"])
	secretKey := strings.TrimSpace(cfg["sms.tencent.secret_key"])
	sdkAppID := strings.TrimSpace(cfg["sms.tencent.sdk_app_id"])
	signName := strings.TrimSpace(cfg["sms.sign_name"])
	templateID := strings.TrimSpace(cfg["sms.template_id"])
	if secretID == "" || secretKey == "" || sdkAppID == "" || signName == "" || templateID == "" {
		return fmt.Errorf("腾讯云短信未配置完整 (需 secret_id/secret_key/sdk_app_id/sign_name/template_id)")
	}

	payload, err := json.Marshal(map[string]any{
		"PhoneNumberSet":   []string{normalizePhone(phone)},
		"SmsSdkAppId":      sdkAppID,
		"SignName":         signName,
		"TemplateId":       templateID,
		"TemplateParamSet": []string{code},
	})
	if err != nil {
		return err
	}

	ts := time.Now().Unix()
	auth := tc3Authorization(secretID, secretKey, payload, ts)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+tencentHost, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", tencentHost)
	req.Header.Set("X-TC-Action", "SendSms")
	req.Header.Set("X-TC-Version", "2021-01-11")
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("Authorization", auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("腾讯云短信 HTTP %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Response struct {
			Error *struct {
				Code, Message string
			} `json:"Error"`
			SendStatusSet []struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"SendStatusSet"`
			RequestId string `json:"RequestId"`
		} `json:"Response"`
	}
	if jerr := json.Unmarshal(body, &out); jerr != nil {
		return fmt.Errorf("腾讯云短信应答解析失败: %s", string(body))
	}
	if out.Response.Error != nil {
		return fmt.Errorf("腾讯云短信发送失败: %s %s",
			out.Response.Error.Code, out.Response.Error.Message)
	}
	if len(out.Response.SendStatusSet) == 0 || out.Response.SendStatusSet[0].Code != "Ok" {
		msg := "未知应答"
		if len(out.Response.SendStatusSet) > 0 {
			msg = out.Response.SendStatusSet[0].Code + " " + out.Response.SendStatusSet[0].Message
		}
		return fmt.Errorf("腾讯云短信发送失败: %s", msg)
	}
	return nil
}

// tc3Authorization 组装 Authorization 头 (纯函数, 便于单测锁定签名串构造)。
func tc3Authorization(secretID, secretKey string, payload []byte, timestamp int64) string {
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	hashedPayload := sha256Hex(payload)
	canonicalRequest := strings.Join([]string{
		"POST", "/",
		"",                              // canonical query (空)
		"content-type:application/json", // canonical headers (小写, 排序)
		"host:" + tencentHost,
		"",
		"content-type;host",
		hashedPayload,
	}, "\n")
	stringToSign := strings.Join([]string{
		"TC3-HMAC-SHA256",
		fmt.Sprintf("%d", timestamp),
		date + "/sms/tc3_request",
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signature := hex.EncodeToString(tc3HMAC(
		tc3HMAC(tc3HMAC(tc3HMAC([]byte("TC3"+secretKey), date), "sms"), "tc3_request"),
		stringToSign,
	))
	return fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s/sms/tc3_request, SignedHeaders=content-type;host, Signature=%s",
		secretID, date, signature)
}

// tc3HMAC HMAC-SHA256。
func tc3HMAC(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

// sha256Hex 十六进制摘要。
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
