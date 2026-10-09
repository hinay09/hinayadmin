// Package ai — 对话实现 (eino react agent / eino-ext openai, OpenAI 兼容接口流式输出)。
package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	ub "github.com/cloudwego/eino/utils/callbacks"

	v1 "hinay.cn/admin/api/ai/v1"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/xerror"
)

const (
	maxTurns       = 30              // 组装上下文时保留的最近消息条数 (含工具步骤行, 控制长度/成本)
	maxContentRune = 8000            // 单条消息内容上限 (字符)
	usageWaitMax   = 2 * time.Second // 收尾等待用量回调排空的上限 (防御挂死)
)

// chatEmitter 多通道下发: 正文 / 思考过程 (推理模型) / 工具调用步骤, 由控制器实现为 SSE 帧。
// agent 可能并行执行工具 (即使顺序执行, 步骤事件也在工具 goroutine 上发出), 故所有下发加锁串行化。
type chatEmitter struct {
	mu        sync.Mutex
	content   func(delta string) error
	reasoning func(delta string) error
	toolStart func(step *v1.AiToolStep) error
	toolEnd   func(step *v1.AiToolStep) error
}

// emitContent 正文增量 (用户断连等发送失败会中断对话, 向上冒泡)。
func (e *chatEmitter) emitContent(text string) error {
	if e.content == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.content(text)
}

// emitReasoning 思考增量 (发送失败向上冒泡)。
func (e *chatEmitter) emitReasoning(text string) error {
	if e.reasoning == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.reasoning(text)
}

// emitToolStart/emitToolEnd 工具步骤事件 (发送失败不阻断对话)。
func (e *chatEmitter) emitToolStart(step *v1.AiToolStep) {
	if e.toolStart == nil {
		return
	}
	e.mu.Lock()
	err := e.toolStart(step)
	e.mu.Unlock()
	_ = err
}

func (e *chatEmitter) emitToolEnd(step *v1.AiToolStep) {
	if e.toolEnd == nil {
		return
	}
	e.mu.Lock()
	err := e.toolEnd(step)
	e.mu.Unlock()
	_ = err
}

// tokenUsage 一轮对话的累计 token 用量 (agent 内多次模型调用逐次累加; 每次调用都全额计费)。
type tokenUsage struct {
	prompt     int
	completion int
	total      int
}

// usageCollector 并发安全的用量累加器: agent 每次模型调用的流式回调在尾帧回报 TokenUsage
// (eino-ext openai 流式请求自动携带 include_usage), 由排空 goroutine 写入, 主循环收尾读取。
type usageCollector struct {
	mu    sync.Mutex
	wg    sync.WaitGroup
	usage tokenUsage
}

// add 累加一次模型调用的用量。
func (c *usageCollector) add(t *model.TokenUsage) {
	if t == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.usage.prompt += t.PromptTokens
	c.usage.completion += t.CompletionTokens
	c.usage.total += t.TotalTokens
}

// snapshot 取当前累计值。
func (c *usageCollector) snapshot() tokenUsage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.usage
}

// drainAsync 排空模型回调的输出流副本, 累计其中的 token 用量 (尾帧携带)。
// 必须在独立 goroutine 中消费: 回调是同步派发的, 阻塞消费会反压卡死主流;
// 而不消费的副本也必须排空到底 (或 Close), 否则管道广播会永久阻塞上游写入。
func (c *usageCollector) drainAsync(sr *schema.StreamReader[*model.CallbackOutput]) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer sr.Close() // 排空/中断后释放副本 (幂等)
		for {
			out, err := sr.Recv()
			if err != nil {
				return
			}
			if out != nil && out.TokenUsage != nil {
				c.add(out.TokenUsage)
			}
		}
	}()
}

// wait 主流结束后等待全部排空 (末轮用量可能在主循环见 EOF 后才写入), 限时防御。
func (c *usageCollector) wait() {
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(usageWaitMax):
	}
}

// ChatStream AI 对话: 逐段回调增量内容与思考过程。
// 前端仅上送本轮 message + sessionId, 历史由服务端会话记忆 (MySQL) 提供;
// 思考来源: ① reasoning_content 独立字段 (DeepSeek-R1 等, eino 映射到 msg.ReasoningContent);
//
//	② content 内联 <think>...</think> 标签 (vLLM 自部署网关, thinkSplitter 拆分)。
//
// 工具循环由 react agent 驱动 (模型原生 function calling), 工具步骤事件由 stepTool 装饰器发出。
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
	msgs := buildMessages(cfg, history, in.Message)

	temperature := float32(cfg.temperature)
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      cfg.apiKey,
		BaseURL:     cfg.baseURL,
		Model:       cfg.model,
		Temperature: &temperature,
	})
	if err != nil {
		return xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// react agent: 工具定义/执行走 stepTool 装饰器 (事件+落库素材); 顺序执行对齐旧循环语义
	recorder := &stepRecorder{}
	tools, err := buildTools(e, recorder)
	if err != nil {
		return xerror.Wrap(xerror.CodeBusinessError, err)
	}
	runner, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: cm,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               toolsOf(tools),
			UnknownToolsHandler: unknownToolStep(e, recorder), // 幻觉工具: 错误结果回填, 不中断对话
			ExecuteSequentially: true,
		},
		MaxStep: agentMaxStep,
	})
	if err != nil {
		return xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// token 用量: 模型节点回调流式尾帧回报, 逐次累加 (排空 goroutine, 见 drainAsync)
	usage := &usageCollector{}
	cb := ub.NewHandlerHelper().ChatModel(&ub.ModelCallbackHandler{
		OnEndWithStreamOutput: func(c context.Context, _ *callbacks.RunInfo,
			sr *schema.StreamReader[*model.CallbackOutput]) context.Context {
			usage.drainAsync(sr)
			return c
		},
	}).Handler()

	// 两条流水线: 思考流(特殊token过滤) / 正文流(特殊token过滤 + think 标签拆分)
	rFilter := &streamFilter{}
	cFilter := &streamFilter{}
	splitter := &thinkSplitter{}
	var finalAnswer strings.Builder // 回写会话用的完整正文 (思考不入上下文)

	emitContent := func(text string) error {
		if out := cFilter.feed(text); out != "" {
			finalAnswer.WriteString(out)
			return e.emitContent(out)
		}
		return nil
	}
	emitReasoning := func(text string) error {
		if out := rFilter.feed(text); out != "" {
			return e.emitReasoning(out)
		}
		return nil
	}
	handleContent := func(text string) error {
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

	// 流式消费: 正文与思考增量分别过各自的流水线下发; 工具轮次由 agent 内部消化,
	// 只有最终回答轮的消息会流到这里
	sr, serr := runner.Stream(ctx, msgs, agent.WithComposeOptions(compose.WithCallbacks(cb)))
	if serr != nil {
		return xerror.Wrap(xerror.CodeBusinessError, fmt.Errorf("AI 接口调用失败: %w", serr))
	}
	for {
		msg, rerr := sr.Recv()
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			err = rerr
			break
		}
		if msg == nil {
			continue
		}
		if r := msg.ReasoningContent; r != "" {
			if eerr := emitReasoning(r); eerr != nil {
				err = eerr
				break
			}
		}
		if c := msg.Content; c != "" {
			if eerr := handleContent(c); eerr != nil {
				err = eerr
				break
			}
		}
	}
	sr.Close() // 中途出错时释放主流 (完整消费后为幂等空操作)

	// 流结束: 冲刷拆分器与过滤器残余 (尽力而为)
	if ferr := flushAll(handleContent, emitContent, emitReasoning, splitter, cFilter, rFilter); ferr != nil && err == nil {
		err = ferr
	}
	if err != nil {
		return xerror.Wrap(xerror.CodeBusinessError, fmt.Errorf("AI 接口调用失败: %w", err))
	}

	usage.wait() // 等回调流副本排空, 保证末轮用量已计入

	// 本轮成功: user + 工具步骤(role=tool) + assistant(正文, 带 token 用量) 持久化入库
	// (思考过程不入库); 首轮 (会话无历史) 以首条用户消息作为会话标题
	firstTitle := ""
	if len(history) == 0 {
		firstTitle = in.Message
	}
	tu := usage.snapshot()
	saveTurn(ctx, sessionId, in.Message, finalAnswer.String(), firstTitle, recorder.all(), tu)
	// 成功收尾: 下发本轮累计 token 用量 (网关未回报为 0, 前端隐藏不展示)
	if emit.Usage != nil {
		_ = emit.Usage(&v1.AiTokenUsage{Prompt: tu.prompt, Completion: tu.completion, Total: tu.total})
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

// buildMessages 组装模型消息 (eino schema.Message): 可选系统提示词 + 会话历史 + 本轮新消息。
func buildMessages(cfg *aiConfig, history []sessionMessage, message string) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(history)+2)
	if cfg.systemPrompt != "" {
		msgs = append(msgs, schema.SystemMessage(cfg.systemPrompt))
	}
	for _, m := range history {
		// 工具步骤行 → 以 AI 文本消息回放 (模型跨轮记得自己调用过什么、拿到过什么)
		if m.Role == "tool" {
			if step := parseToolStep(m.Content); step != nil {
				msgs = append(msgs, schema.AssistantMessage(clip(stepContextText(step)), nil))
			}
			continue
		}
		switch m.Role {
		case "user":
			msgs = append(msgs, schema.UserMessage(clip(m.Content)))
		case "assistant":
			msgs = append(msgs, schema.AssistantMessage(clip(m.Content), nil))
		case "system":
			msgs = append(msgs, schema.SystemMessage(clip(m.Content)))
		}
		// 其余角色: 历史里的坏角色直接丢弃, 不阻断对话
	}
	msgs = append(msgs, schema.UserMessage(clip(message)))

	// 截断: 保留首条 system + 最近 maxTurns 条
	if len(msgs) > maxTurns+1 {
		kept := msgs[len(msgs)-maxTurns:]
		if msgs[0].Role == schema.System {
			kept = append([]*schema.Message{msgs[0]}, kept...)
		}
		msgs = kept
	}
	return msgs
}

// clip 内容裁剪: 去空白 + 截断至单条上限。
func clip(content string) string {
	c := strings.TrimSpace(content)
	if r := []rune(c); len(r) > maxContentRune {
		return string(r[:maxContentRune])
	}
	return c
}

// toolsOf []tool.InvokableTool → []tool.BaseTool (ToolsNodeConfig.Tools 字段类型)。
func toolsOf(invokables []tool.InvokableTool) []tool.BaseTool {
	base := make([]tool.BaseTool, 0, len(invokables))
	for _, it := range invokables {
		base = append(base, it)
	}
	return base
}
