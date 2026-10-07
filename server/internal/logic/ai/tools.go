// Package ai — 工具调用 (function calling): 模型可请求调用服务端注册的工具,
// 拿到结果后继续生成正文。工具在此注册, chat.go 负责调用循环。
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "time/tzdata" // 内嵌 IANA 时区库, 保证精简容器里 LoadLocation("Asia/Shanghai") 也可用

	"github.com/tmc/langchaingo/llms"

	v1 "hinay.cn/admin/api/ai/v1"
)

// maxToolRounds 单轮对话内「模型请求工具 → 回填结果」的最大轮数 (防御模型反复要工具的死循环);
// 达到上限后最后一轮不再携带工具, 强制模型以正文收尾。
const maxToolRounds = 5

// toolExecTimeout 单次工具执行超时: 挂死的工具不应拖住整轮对话 (超时以错误结果回填给模型)。
const toolExecTimeout = 15 * time.Second

// aiTool 服务端工具: 暴露给模型的定义 (OpenAI function calling) + 本地执行器。
type aiTool struct {
	definition llms.FunctionDefinition
	run        func(ctx context.Context, args string) (string, error)
}

// toolRegistry 工具注册表 (顺序稳定); 新增工具在此追加一项。
var toolRegistry = []aiTool{
	{
		definition: llms.FunctionDefinition{
			Name:        "get_current_time",
			Description: "获取服务器当前日期时间 (含星期、时区与 Unix 时间戳)。当用户询问现在几点、今天日期、星期几, 或回答需要依赖当前时间时调用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"timezone": map[string]any{
						"type":        "string",
						"description": "IANA 时区名, 如 Asia/Shanghai、UTC; 缺省用服务器本地时区",
					},
				},
			},
		},
		run: runCurrentTime,
	},
}

// toolSpecs 暴露给模型的工具定义列表 (llms.WithTools 用)。
func toolSpecs() []llms.Tool {
	specs := make([]llms.Tool, 0, len(toolRegistry))
	for i := range toolRegistry {
		specs = append(specs, llms.Tool{Type: "function", Function: &toolRegistry[i].definition})
	}
	return specs
}

// runTool 按名分发执行; 未注册的工具/非法参数/超时均返回错误 (由调用方转成错误结果回填给模型)。
func runTool(ctx context.Context, name, args string) (string, error) {
	// 参数必须是合法 JSON (可空串 = 无参): 提前拦截模型幻觉输出, 不带进业务工具
	if s := strings.TrimSpace(args); s != "" && !json.Valid([]byte(s)) {
		return "", fmt.Errorf("工具参数不是合法 JSON: %s", clipRunes(s, 200))
	}
	for i := range toolRegistry {
		if toolRegistry[i].definition.Name == name {
			tctx, cancel := context.WithTimeout(ctx, toolExecTimeout)
			defer cancel()
			return toolRegistry[i].run(tctx, args)
		}
	}
	return "", fmt.Errorf("未注册的工具: %s", name)
}

// clipRunes 按字符截断 (错误提示兜底, 防超长参数刷屏)。
func clipRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
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

// weekdays 星期中文映射 (time.Weekday: 周日=0)。
var weekdays = [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

// toolCallsOf 从生成结果中提取模型请求的工具调用 (取首个 choice)。
func toolCallsOf(resp *llms.ContentResponse) []llms.ToolCall {
	if resp == nil || len(resp.Choices) == 0 {
		return nil
	}
	return resp.Choices[0].ToolCalls
}

// appendToolMessages 执行模型请求的工具调用: 每个调用先发 toolStart 事件 (前端步骤条转圈),
// 执行完发 toolEnd (带结果或错误), 并把 AI 消息 (携带调用请求) 与各工具结果逐一追加到消息列表,
// 供下一轮生成续写; 同时返回步骤记录供落库 (跨轮回放)。约束: tool 角色消息必须单条单结果
// (openai 适配层要求)。执行失败也以错误文本回填, 让模型据此调整而不是中断整轮对话。
func appendToolMessages(ctx context.Context, msgs []llms.MessageContent, calls []llms.ToolCall,
	e *chatEmitter) ([]llms.MessageContent, []v1.AiToolStep) {
	parts := make([]llms.ContentPart, 0, len(calls))
	for _, c := range calls {
		typ := c.Type
		if typ == "" {
			typ = "function" // 个别网关不回传 type, 回填时兜底
		}
		parts = append(parts, llms.ToolCall{ID: c.ID, Type: typ, FunctionCall: c.FunctionCall})
	}
	msgs = append(msgs, llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: parts})

	steps := make([]v1.AiToolStep, 0, len(calls))
	for i, c := range calls {
		var name, args string
		if c.FunctionCall != nil {
			name = c.FunctionCall.Name
			args = c.FunctionCall.Arguments
		}
		step := v1.AiToolStep{Id: c.ID, Name: name, Args: args}
		if step.Id == "" {
			step.Id = fmt.Sprintf("%s#%d", name, i) // 个别网关无 call id, 前端步骤条需要稳定键
		}
		e.emitToolStart(&step)

		result, err := runTool(ctx, name, args)
		if err != nil {
			step.Error = err.Error()
			result = fmt.Sprintf(`{"error":%q}`, err.Error())
		} else {
			step.Result = result
		}
		steps = append(steps, step)
		e.emitToolEnd(&step)

		msgs = append(msgs, llms.MessageContent{
			Role:  llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{llms.ToolCallResponse{ToolCallID: c.ID, Name: name, Content: result}},
		})
	}
	return msgs, steps
}

// runCurrentTime 返回当前时间; 可选 timezone 参数指定时区 (IANA 名, 缺省服务器本地)。
func runCurrentTime(_ context.Context, args string) (string, error) {
	var a struct {
		Timezone string `json:"timezone"`
	}
	// 参数可空/可为空对象, 解析失败一律按缺省时区处理
	_ = json.Unmarshal([]byte(args), &a)

	loc := time.Local
	if tz := strings.TrimSpace(a.Timezone); tz != "" {
		l, err := time.LoadLocation(tz)
		if err != nil {
			return "", fmt.Errorf("无法识别的时区 %q, 请使用 IANA 名称 (如 Asia/Shanghai、UTC)", tz)
		}
		loc = l
	}
	now := time.Now().In(loc)
	out, err := json.Marshal(map[string]any{
		"timezone": now.Location().String(),
		"datetime": now.Format("2006-01-02 15:04:05"),
		"weekday":  weekdays[now.Weekday()],
		"unix":     now.Unix(),
	})
	if err != nil {
		return "", err
	}
	return string(out), nil
}
