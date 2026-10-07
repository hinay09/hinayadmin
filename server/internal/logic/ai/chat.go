// Package ai — 对话实现 (langchaingo / llms/openai, OpenAI 兼容接口流式输出)。
package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"

	v1 "hinay.cn/admin/api/ai/v1"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/xerror"
)

const (
	maxTurns       = 30   // 组装上下文时保留的最近消息条数 (含工具步骤行, 控制长度/成本)
	maxContentRune = 8000 // 单条消息内容上限 (字符)
)

// chatEmitter 多通道下发: 正文 / 思考过程 (推理模型) / 工具调用步骤, 由控制器实现为 SSE 帧。
type chatEmitter struct {
	content   func(delta string) error
	reasoning func(delta string) error
	toolStart func(step *v1.AiToolStep) error
	toolEnd   func(step *v1.AiToolStep) error
}

// emitToolStart/emitToolEnd 工具步骤事件 (通道可空, 发送失败不阻断对话)。
func (e *chatEmitter) emitToolStart(step *v1.AiToolStep) {
	if e.toolStart != nil {
		_ = e.toolStart(step)
	}
}

func (e *chatEmitter) emitToolEnd(step *v1.AiToolStep) {
	if e.toolEnd != nil {
		_ = e.toolEnd(step)
	}
}

// tokenUsage 一轮对话的累计 token 用量 (多轮工具循环逐次累加; 每轮都全额计费)。
type tokenUsage struct {
	prompt     int
	completion int
	total      int
}

// addUsage 累加一次生成的用量 (langchaingo 把 usage 放在首 choice 的 GenerationInfo, 流式亦有值)。
func (u *tokenUsage) addUsage(resp *llms.ContentResponse) {
	if resp == nil || len(resp.Choices) == 0 {
		return
	}
	gi := resp.Choices[0].GenerationInfo
	u.prompt += toInt(gi["PromptTokens"])
	u.completion += toInt(gi["CompletionTokens"])
	u.total += toInt(gi["TotalTokens"])
}

// toInt 安全取整数 (其他 provider 的 GenerationInfo 值类型未必是 int, 不断言失败panic)。
func toInt(v any) int {
	if n, ok := v.(int); ok {
		return n
	}
	return 0
}

// ChatStream AI 对话: 逐段回调增量内容与思考过程。
// 前端仅上送本轮 message + sessionId, 历史由服务端会话记忆 (Redis) 提供;
// 思考来源: ① reasoning_content 独立字段 (DeepSeek-R1 等, langchaingo 回调);
//
//	② content 内联 <think>...</think> 标签 (vLLM 自部署网关, thinkSplitter 拆分)。
func (s *sAi) ChatStream(ctx context.Context, in *v1.AiChatReq, emit *v1.AiStreamEmit) error {
	e := &chatEmitter{content: emit.Content, reasoning: emit.Reasoning,
		toolStart: emit.ToolStart, toolEnd: emit.ToolEnd}
	cfg, err := loadAiConfig(ctx)
	if err != nil {
		return err
	}

	// 会话上下文 (MySQL 持久化; 读取失败自动降级无记忆)
	sessionId := strings.TrimSpace(in.SessionId)
	history := loadChatHistory(ctx, contextx.UserId(ctx), sessionId)
	msgs, err := buildMessages(cfg, history, in.Message)
	if err != nil {
		return err
	}

	llm, err := openai.New(
		openai.WithToken(cfg.apiKey),
		openai.WithBaseURL(cfg.baseURL),
		openai.WithModel(cfg.model),
	)
	if err != nil {
		return xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// 两条流水线: 思考流(特殊token过滤) / 正文流(特殊token过滤 + think 标签拆分)
	rFilter := &streamFilter{}
	cFilter := &streamFilter{}
	splitter := &thinkSplitter{}
	var finalAnswer strings.Builder // 回写会话用的完整正文 (思考不入上下文)

	emitContent := func(text string) error {
		if out := cFilter.feed(text); out != "" {
			finalAnswer.WriteString(out)
			return e.content(out)
		}
		return nil
	}
	emitReasoning := func(text string) error {
		if e.reasoning == nil {
			return nil
		}
		if out := rFilter.feed(text); out != "" {
			return e.reasoning(out)
		}
		return nil
	}
	handleContent := func(text string) error {
		if isToolCallChunk(text) {
			return nil // 工具调用增量 (适配层塞进正文通道的 JSON), 不是正文, 整段拦下
		}
		c, r := splitter.feed(text)
		if r != "" {
			if err := emitReasoning(r); err != nil {
				return err
			}
		}
		if c != "" {
			return emitContent(c)
		}
		return nil
	}

	// 生成循环: 模型若请求工具 (function calling) 则本地执行 (发步骤事件) 并把结果回填,
	// 再继续生成, 最多 maxToolRounds 轮; 之后不再携带工具, 强制以正文收尾。
	// 全程累计工具步骤与 token 用量, 成功后随本轮消息落库。
	tools := toolSpecs()
	var toolSteps []v1.AiToolStep
	usage := tokenUsage{}
	for round := 0; ; round++ {
		opts := []llms.CallOption{
			llms.WithTemperature(cfg.temperature),
			llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
				if len(chunk) == 0 {
					return nil
				}
				return handleContent(string(chunk))
			}),
			llms.WithStreamingReasoningFunc(func(_ context.Context, reasoningChunk, _ []byte) error {
				if len(reasoningChunk) == 0 {
					return nil
				}
				return emitReasoning(string(reasoningChunk))
			}),
		}
		if round < maxToolRounds {
			opts = append(opts, llms.WithTools(tools), llms.WithToolChoice("auto"))
		}
		var resp *llms.ContentResponse
		resp, err = llm.GenerateContent(ctx, msgs, opts...)
		if err != nil {
			break
		}
		usage.addUsage(resp)
		calls := toolCallsOf(resp)
		if len(calls) == 0 || round >= maxToolRounds {
			break // 正文已生成 (或预算用尽), 结束循环
		}
		msgs, toolSteps = appendToolMessages(ctx, msgs, calls, e)
	}
	// 流结束: 冲刷拆分器与过滤器残余 (尽力而为)
	if ferr := flushAll(handleContent, emitContent, emitReasoning, splitter, cFilter, rFilter); ferr != nil && err == nil {
		err = ferr
	}
	if err != nil {
		return xerror.Wrap(xerror.CodeBusinessError, fmt.Errorf("AI 接口调用失败: %w", err))
	}

	// 本轮成功: user + 工具步骤(role=tool) + assistant(正文, 带 token 用量) 持久化入库
	// (思考过程不入库); 首轮 (会话无历史) 以首条用户消息作为会话标题
	firstTitle := ""
	if len(history) == 0 {
		firstTitle = in.Message
	}
	saveTurn(ctx, sessionId, in.Message, finalAnswer.String(), firstTitle, toolSteps, usage)
	// 成功收尾: 下发本轮累计 token 用量 (网关未回报为 0, 前端隐藏不展示)
	if emit.Usage != nil {
		_ = emit.Usage(&v1.AiTokenUsage{Prompt: usage.prompt, Completion: usage.completion, Total: usage.total})
	}
	return nil
}

// flushAll 依次冲刷: 拆分器残余 → 两通道过滤器残余。
func flushAll(handleContent func(string) error, emitContent, emitReasoning func(string) error,
	splitter *thinkSplitter, cFilter, rFilter *streamFilter) error {
	if c, r := splitter.flush(); c != "" || r != "" {
		if r != "" {
			if err := emitReasoning(r); err != nil {
				return err
			}
		}
		if c != "" {
			if err := emitContent(c); err != nil {
				return err
			}
		}
	}
	if out := rFilter.flush(); out != "" {
		if err := emitReasoning(out); err != nil {
			return err
		}
	}
	if out := cFilter.flush(); out != "" {
		return emitContent(out)
	}
	return nil
}

// AiConfig 配置状态 (供前端提示)。
func (s *sAi) AiConfig(ctx context.Context, in *v1.AiConfigReq) (res *v1.AiConfigRes, err error) {
	key := strings.TrimSpace(cfgOf(ctx, cfgKeyAPIKey, ""))
	return &v1.AiConfigRes{
		Configured: key != "",
		BaseUrl:    strings.TrimSpace(cfgOf(ctx, cfgKeyBaseURL, defaultBaseURL)),
		Model:      strings.TrimSpace(cfgOf(ctx, cfgKeyModel, defaultModel)),
		KeyMasked:  maskKey(key),
	}, nil
}

// AiHistory 会话历史 (全量消息, 恢复对话界面; 思考过程不入历史)。
// 工具步骤行以 role=tool + 解析后的步骤详情返回, 前端折叠渲染到其后紧邻的 assistant 气泡。
func (s *sAi) AiHistory(ctx context.Context, in *v1.AiHistoryReq) (res *v1.AiHistoryRes, err error) {
	msgs, err := loadFullHistory(ctx, contextx.UserId(ctx), strings.TrimSpace(in.SessionId))
	if err != nil {
		return nil, err
	}
	list := make([]v1.AiHistoryMessage, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "tool" {
			if step := parseToolStep(m.Content); step != nil {
				list = append(list, v1.AiHistoryMessage{Role: "tool", Tool: step})
			}
			continue // 坏行丢弃, 不阻断历史恢复
		}
		item := v1.AiHistoryMessage{Role: m.Role, Content: m.Content}
		if m.Role == "assistant" {
			item.Usage = &v1.AiTokenUsage{Prompt: m.promptTokens, Completion: m.completionTokens, Total: m.totalTokens}
		}
		list = append(list, item)
	}
	return &v1.AiHistoryRes{List: list}, nil
}

// AiSessions 当前用户的历史会话列表 (左侧历史会话栏, 按最后活跃倒序)。
func (s *sAi) AiSessions(ctx context.Context, in *v1.AiSessionsReq) (res *v1.AiSessionsRes, err error) {
	list, err := listConversations(ctx, contextx.UserId(ctx))
	if err != nil {
		return nil, err
	}
	return &v1.AiSessionsRes{List: list}, nil
}

// AiSessionDelete 删除当前用户的一个会话 (连同消息, 校验归属)。
func (s *sAi) AiSessionDelete(ctx context.Context, in *v1.AiSessionDeleteReq) (res *v1.AiSessionDeleteRes, err error) {
	if err := deleteConversation(ctx, contextx.UserId(ctx), strings.TrimSpace(in.SessionId)); err != nil {
		return nil, err
	}
	return &v1.AiSessionDeleteRes{}, nil
}

// buildMessages 组装 langchaingo 消息: 可选系统提示词 + 会话历史 + 本轮新消息。
func buildMessages(cfg *aiConfig, history []sessionMessage, message string) ([]llms.MessageContent, error) {
	msgs := make([]llms.MessageContent, 0, len(history)+2)
	if cfg.systemPrompt != "" {
		msgs = append(msgs, textMsg(llms.ChatMessageTypeSystem, cfg.systemPrompt))
	}
	for _, m := range history {
		// 工具步骤行 → 以 AI 文本消息回放 (模型跨轮记得自己调用过什么、拿到过什么)
		if m.Role == "tool" {
			if step := parseToolStep(m.Content); step != nil {
				msgs = append(msgs, textMsg(llms.ChatMessageTypeAI, clip(stepContextText(step))))
			}
			continue
		}
		rt, ok := historyRole(m.Role)
		if !ok {
			continue // 历史里的坏角色直接丢弃, 不阻断对话
		}
		msgs = append(msgs, textMsg(rt, clip(m.Content)))
	}
	msgs = append(msgs, textMsg(llms.ChatMessageTypeHuman, clip(message)))

	// 截断: 保留首条 system + 最近 maxTurns 条
	if len(msgs) > maxTurns+1 {
		head := msgs[:0]
		if msgs[0].Role == llms.ChatMessageTypeSystem {
			head = append(head, msgs[0])
		}
		msgs = append(head, msgs[len(msgs)-maxTurns:]...)
	}
	return msgs, nil
}

// historyRole 会话角色映射。
func historyRole(role string) (llms.ChatMessageType, bool) {
	switch role {
	case "user":
		return llms.ChatMessageTypeHuman, true
	case "assistant":
		return llms.ChatMessageTypeAI, true
	case "system":
		return llms.ChatMessageTypeSystem, true
	}
	return "", false
}

// clip 内容裁剪: 去空白 + 截断至单条上限。
func clip(content string) string {
	c := strings.TrimSpace(content)
	if r := []rune(c); len(r) > maxContentRune {
		return string(r[:maxContentRune])
	}
	return c
}

// textMsg 构造纯文本消息。
func textMsg(role llms.ChatMessageType, text string) llms.MessageContent {
	return llms.MessageContent{
		Role:  role,
		Parts: []llms.ContentPart{llms.TextContent{Text: text}},
	}
}
