// Package ai — 会话记忆与历史持久化 (MySQL: ai_conversation / ai_chat_message)。
//
// 前端仅发送 {sessionId, message}, 历史与上下文均由服务端按 sessionId 落库:
//   - 历史记录: 全量永久保存 (查看走 /ai/history, 无过期);
//   - 上下文记忆: 每轮组装模型消息只取最近 maxTurns 条 (chat.go), 成功一轮
//     事务追加 user 行 + 工具步骤行(role=tool) + assistant 行(带 token 用量) 并刷新会话活跃时间;
//   - 读取失败降级为无记忆单轮 (不阻断对话), 持久化失败仅记日志 (本轮不入历史)。
//
// 会话归属 user_id: 非本人 sessionId 一律按"不存在"处理, 不泄露存在性。
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	v1 "hinay.cn/admin/api/ai/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/xerror"
)

const (
	historyMaxOut = 1000 // 历史查看单次返回上限 (防御超长会话, 取最近 N 条)
	convListMax   = 50   // 会话列表保留条数 (按活跃倒序)
	sessTitleMax  = 30   // 会话标题裁剪长度 (字符)
)

// sessionMessage 会话中的一条消息。
type sessionMessage struct {
	Role    string `json:"role"`    // user / assistant / tool
	Content string `json:"content"` // 正文 (思考过程不入库, 不占上下文); role=tool 时为步骤 JSON
	// token 用量列 (仅 assistant 行; 历史展示用, 上下文组装不使用)
	promptTokens     int
	completionTokens int
	totalTokens      int
}

// getConversation 按 session_id + user_id 取会话; 不存在/非本人/参数缺失 返回 nil。
func getConversation(ctx context.Context, userId uint64, sessionId string) (*entity.AiConversation, error) {
	if userId == 0 || sessionId == "" {
		return nil, nil
	}
	one, err := dao.AiConversation.Ctx(ctx).
		Where("session_id", sessionId).
		Where("user_id", userId).
		One()
	if err != nil {
		return nil, err
	}
	if one.IsEmpty() {
		return nil, nil
	}
	var conv entity.AiConversation
	if err = one.Struct(&conv); err != nil {
		return nil, err
	}
	return &conv, nil
}

// loadChatHistory 读取对话上下文: 最近 maxTurns 条 (时间正序)。
// 出错降级返回 nil (无记忆单轮, 不阻断对话)。
func loadChatHistory(ctx context.Context, userId uint64, sessionId string) []sessionMessage {
	conv, err := getConversation(ctx, userId, sessionId)
	if err != nil {
		g.Log().Warningf(ctx, "[ai] 会话读取失败, 降级为无记忆单轮: %v", err)
		return nil
	}
	if conv == nil {
		return nil
	}
	msgs, err := loadMessages(ctx, conv.Id, maxTurns)
	if err != nil {
		g.Log().Warningf(ctx, "[ai] 会话消息读取失败, 降级为无记忆单轮: %v", err)
		return nil
	}
	return msgs
}

// loadFullHistory 历史查看: 全量消息 (超长会话取最近 historyMaxOut 条, 时间正序)。
func loadFullHistory(ctx context.Context, userId uint64, sessionId string) ([]sessionMessage, error) {
	conv, err := getConversation(ctx, userId, sessionId)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "读取会话历史失败")
	}
	if conv == nil {
		return []sessionMessage{}, nil
	}
	return loadMessages(ctx, conv.Id, historyMaxOut)
}

// loadMessages 取会话最近 limit 条消息并反转为时间正序 (含 token 用量列, 供历史展示)。
func loadMessages(ctx context.Context, conversationId uint64, limit int) ([]sessionMessage, error) {
	all, err := dao.AiChatMessage.Ctx(ctx).
		Fields("role", "content", "prompt_tokens", "completion_tokens", "total_tokens").
		Where("conversation_id", conversationId).
		Order("id DESC").
		Limit(limit).
		All()
	if err != nil {
		return nil, err
	}
	msgs := make([]sessionMessage, 0, len(all))
	for i := len(all) - 1; i >= 0; i-- { // 倒序取回 → 反转为正序
		msgs = append(msgs, sessionMessage{
			Role:             all[i]["role"].String(),
			Content:          all[i]["content"].String(),
			promptTokens:     all[i]["prompt_tokens"].Int(),
			completionTokens: all[i]["completion_tokens"].Int(),
			totalTokens:      all[i]["total_tokens"].Int(),
		})
	}
	return msgs, nil
}

// saveTurn 一轮成功后持久化: 建会话 (首轮带标题) + 追加 user 行、工具步骤行 (role=tool)、
// assistant 行 (带本轮累计 token 用量) + 刷新活跃时间。工具步骤与 assistant 正文一起构成
// 下一轮的上下文回放素材 (见 chat.go buildMessages)。
// best-effort: 失败仅记日志 (本轮不进入历史, 不影响已流式返回给用户的内容)。
func saveTurn(ctx context.Context, sessionId, userMsg, assistantMsg, title string,
	steps []v1.AiToolStep, usage tokenUsage) {
	userId := contextx.UserId(ctx)
	if userId == 0 || sessionId == "" {
		return
	}
	err := dao.AiConversation.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 会话行不存在则建 (并发首轮撞唯一键 = 已建, 直接继续)
		if _, err := dao.AiConversation.Ctx(ctx).Data(g.Map{
			"session_id": sessionId,
			"user_id":    userId,
			"title":      clipTitle(title),
		}).Insert(); err != nil && !isDupEntry(err) {
			return err
		}
		conv, err := dao.AiConversation.Ctx(ctx).
			Fields("id").
			Where("session_id", sessionId).
			Where("user_id", userId).
			One()
		if err != nil {
			return err
		}
		if conv.IsEmpty() {
			// session_id 已被他人占用 (唯一键命中但归属查不到): 拒绝挂靠他人会话
			return xerror.New(xerror.CodeBusinessError, "会话不存在或已过期")
		}
		cid := conv["id"].Uint64()
		// user 行 + 工具步骤行 (同构列, 一次插入); 步骤内容为 v1.AiToolStep JSON
		rows := g.List{
			{"conversation_id": cid, "role": "user", "content": userMsg},
		}
		for _, s := range steps {
			content, merr := json.Marshal(s)
			if merr != nil {
				continue // 单步序列化失败丢弃该步, 不阻断整轮落库
			}
			rows = append(rows, g.Map{"conversation_id": cid, "role": "tool", "content": string(content)})
		}
		if _, err := dao.AiChatMessage.Ctx(ctx).Data(rows).Insert(); err != nil {
			return err
		}
		// assistant 行单独插入: 附本轮累计 token 用量 (网关未回报时为 0)
		if _, err := dao.AiChatMessage.Ctx(ctx).Data(g.Map{
			"conversation_id":   cid,
			"role":              "assistant",
			"content":           assistantMsg,
			"prompt_tokens":     usage.prompt,
			"completion_tokens": usage.completion,
			"total_tokens":      usage.total,
		}).Insert(); err != nil {
			return err
		}
		// 显式更新一行使 updated_at 随 ON UPDATE 刷新 (列表排序依据)
		_, err = dao.AiConversation.Ctx(ctx).
			Where("id", cid).
			Data(g.Map{"message_count": gdb.Raw("message_count + 2")}).
			Update()
		return err
	})
	if err != nil {
		g.Log().Warningf(ctx, "[ai] 会话持久化失败 (本轮不进入历史): %v", err)
	}
}

// listConversations 当前用户的历史会话列表 (按最后活跃倒序, 最多 convListMax 条)。
func listConversations(ctx context.Context, userId uint64) ([]v1.AiSessionItem, error) {
	if userId == 0 {
		return []v1.AiSessionItem{}, nil
	}
	all, err := dao.AiConversation.Ctx(ctx).
		Fields("session_id", "title", "updated_at").
		Where("user_id", userId).
		Order("updated_at DESC, id DESC").
		Limit(convListMax).
		All()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "读取会话列表失败")
	}
	items := make([]v1.AiSessionItem, 0, len(all))
	for _, r := range all {
		var ts int64
		if t := r["updated_at"].GTime(); t != nil {
			ts = t.Unix()
		}
		items = append(items, v1.AiSessionItem{
			SessionId: r["session_id"].String(),
			Title:     r["title"].String(),
			UpdatedAt: ts,
		})
	}
	return items, nil
}

// deleteConversation 删除当前用户的一个会话 (连同消息物理删除)。
// 物理删除而非软删: uk_session_id 与软删残留行冲突 (同 sys_user_totp 先例),
// 且会话数据为可再生对话内容, 无审计留存要求。
func deleteConversation(ctx context.Context, userId uint64, sessionId string) error {
	conv, err := getConversation(ctx, userId, sessionId)
	if err != nil {
		return xerror.Wrap(xerror.CodeBusinessError, err, "删除会话失败")
	}
	if conv == nil {
		return xerror.New(xerror.CodeBusinessError, "会话不存在或已过期")
	}
	return dao.AiConversation.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := dao.AiChatMessage.Ctx(ctx).Unscoped().
			Where("conversation_id", conv.Id).Delete(); err != nil {
			return err
		}
		_, err := dao.AiConversation.Ctx(ctx).Unscoped().
			Where("id", conv.Id).Delete()
		return err
	})
}

// clipTitle 会话标题裁剪: 压缩空白 + 截断 (按字符)。
func clipTitle(title string) string {
	c := strings.Join(strings.Fields(title), " ")
	if r := []rune(c); len(r) > sessTitleMax {
		return string(r[:sessTitleMax]) + "…"
	}
	return c
}

// isDupEntry 判断 MySQL 唯一键冲突 (并发首建会话时容忍)。
func isDupEntry(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "Error 1062")
}

// parseToolStep 解析落库的工具步骤 JSON (role=tool 行); 坏行返回 nil 由调用方丢弃。
func parseToolStep(content string) *v1.AiToolStep {
	var s v1.AiToolStep
	if json.Unmarshal([]byte(content), &s) != nil || s.Name == "" {
		return nil
	}
	return &s
}

// stepContextText 工具步骤 → 上下文回放文本: 以 AI 消息注入, 让模型跨轮记得自己调用过什么、拿到过什么。
// 不回放原生 tool 消息: 严格校验 tool 角色必须紧跟 assistant(tool_calls) 的网关会在截断边界处整轮报错。
func stepContextText(step *v1.AiToolStep) string {
	if step.Error != "" {
		return fmt.Sprintf("[调用工具 %s(%s) 失败: %s]", step.Name, step.Args, clipRunes(step.Error, 300))
	}
	return fmt.Sprintf("[调用工具 %s(%s) → 结果: %s]", step.Name, step.Args, clipRunes(step.Result, 2000))
}

// clipRunes 按字符截断 (错误提示兜底, 防超长参数刷屏)。
func clipRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
