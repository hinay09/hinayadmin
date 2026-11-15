// Package notify SMTP 邮件渠道 (标准库实现, 零外部依赖)。
//
// 端口约定: 465 = 隐式 TLS (直连 TLS), 25/587 等明文端口在服务器支持时自动升级 STARTTLS。
// 认证: PLAIN 优先, 服务器不支持时回落 LOGIN; username/password 任一为空则免认证 (内网中继场景)。
package notify

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"hinay.cn/admin/internal/dao"
)

// emailChannel 定向邮件渠道: 逐接收人查 sys_user.email 投递。
type emailChannel struct{}

// emailMaxRecipients 单条消息最多投递的收件人数 (全员通知的兜底上限, 防误发海量邮件)。
const emailMaxRecipients = 200

func (e *emailChannel) Name() string { return "email" }

func (e *emailChannel) Enabled(cfg map[string]string) bool {
	return cfgBool(cfg["notify.email.enable"]) && strings.TrimSpace(cfg["notify.email.host"]) != ""
}

func (e *emailChannel) Send(ctx context.Context, cfg map[string]string, msg OutMessage) error {
	addrs, err := e.recipientEmails(ctx, msg)
	if err != nil {
		return err
	}
	if len(addrs) == 0 {
		return nil // 无可达收件人 (未填邮箱) = 静默跳过
	}
	from := strings.TrimSpace(cfg["notify.email.from"])
	host := strings.TrimSpace(cfg["notify.email.host"])
	port := 465
	if p, perr := strconv.Atoi(strings.TrimSpace(cfg["notify.email.port"])); perr == nil && p > 0 {
		port = p
	}
	username := strings.TrimSpace(cfg["notify.email.username"])
	password := cfg["notify.email.password"]

	subject := "[通知] " + msg.Title
	for _, to := range addrs {
		// 逐收件人独立投递 (不做群发 BCC: 各收件人邮箱互不可见)
		if err := sendOneMail(host, port, username, password, from, to, subject, mailBody(msg)); err != nil {
			return fmt.Errorf("发往 %s: %w", to, err)
		}
	}
	return nil
}

// recipientEmails 解析可达收件邮箱 (sys_user 按当前租户上下文限界)。
func (e *emailChannel) recipientEmails(ctx context.Context, msg OutMessage) ([]string, error) {
	q := dao.SysUser.Ctx(ctx).
		Where("deleted_at IS NULL").
		Where("status", 1).
		Where("email <> ''").
		Fields("email")
	if !msg.All {
		if len(msg.UserIds) == 0 {
			return nil, nil
		}
		q = q.WhereIn("id", msg.UserIds)
	}
	vars, err := q.Limit(emailMaxRecipients).Array()
	if err != nil {
		return nil, fmt.Errorf("查询收件人失败: %w", err)
	}
	out := make([]string, 0, len(vars))
	seen := map[string]struct{}{}
	for _, v := range vars {
		e := strings.TrimSpace(v.String())
		if e == "" || !strings.Contains(e, "@") {
			continue
		}
		if _, dup := seen[e]; dup {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out, nil
}

// sendOneMail 发送一封纯文本 UTF-8 邮件。
func sendOneMail(host string, port int, username, password, from, to, subject, body string) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	var conn net.Conn
	var err error
	if port == 465 {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("连接 %s: %w", addr, err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()

	// 明文端口: 服务器支持则升级 STARTTLS
	if port != 465 {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err = c.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return fmt.Errorf("STARTTLS: %w", err)
			}
		}
	}
	// 认证 (PLAIN 优先, 回落 LOGIN; 未配置账号则跳过)
	if username != "" && password != "" {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
				// 部分服务只接受 LOGIN (如旧版 Exchange): PlainAuth 失败再试一次
				if lerr := c.Auth(newLoginAuth(username, password, host)); lerr != nil {
					return fmt.Errorf("认证失败: %w", err)
				}
			}
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	if err = c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(buildMail(from, to, subject, body)); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// loginAuth LOGIN 方式认证 (标准库未内置; 少数 SMTP 服务仅支持该方式)。
type loginAuth struct{ username, password, host string }

func newLoginAuth(username, password, host string) smtp.Auth {
	return &loginAuth{username, password, host}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && server.Name != a.host {
		// 与 PlainAuth 同款防护: 明文网络下不发送凭据 (本包几乎总在 TLS 后)
		return "", nil, fmt.Errorf("非 TLS 连接拒绝 LOGIN 认证")
	}
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(string(fromServer)) {
	case "username:", "username":
		return []byte(a.username), nil
	case "password:", "password":
		return []byte(a.password), nil
	}
	return nil, fmt.Errorf("LOGIN 认证意外的服务端问询: %q", fromServer)
}

// cfgBool 配置布尔 ("true"/"1")。
func cfgBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1":
		return true
	}
	return false
}

// mailBody 邮件正文: 纯文本, 标题 + 内容 (站内信内容即纯文本)。
func mailBody(msg OutMessage) string {
	return msg.Title + "\n\n" + msg.Content
}

// buildMail 生成 RFC 5322 纯文本邮件报文 (Subject 按 RFC 2047 UTF-8 编码)。
func buildMail(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	// 正文 base64: 防 8-bit 中转服务器截断, 76 列折行
	enc := base64.StdEncoding.EncodeToString([]byte(body))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	return []byte(b.String())
}
