// Package ai — 工具运行时装饰 (SSE 步骤事件 / 限时执行 / 错误转结果回填 / 步骤落库素材)。
//
// 工具定义在子包 tools (声明式, InferTool 生成); 本文件在其外包裹 stepTool 装饰器,
// 把工具执行接到本包的对话流程上: toolStart/toolEnd 事件、步骤记录、超时与容错。
package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"

	aitools "hinay.cn/admin/internal/logic/ai/tools"

	v1 "hinay.cn/admin/api/ai/v1"
)

// maxToolRounds 单轮对话内「模型请求工具 → 回填结果」的最大轮数 (防御模型反复要工具的死循环)。
// react agent 以步数计: 每轮占 2 步 (模型+工具), 末轮模型收尾, 故步数上限 = (轮数+1)*2;
// 达到上限仍未收尾 agent 会报错中断 (与旧手写循环"最后一轮强制无工具收尾"略有差异, 均为防御行为)。
const maxToolRounds = 5

// agentMaxStep react agent 步数上限 (由 maxToolRounds 推导, 12 步 = 6 次模型调用 = 最多 5 次工具)。
const agentMaxStep = (maxToolRounds + 1) * 2

// toolExecTimeout 单次工具执行超时: 挂死的工具不应拖住整轮对话 (超时以错误结果回填给模型)。
const toolExecTimeout = 15 * time.Second

// stepRecorder 收集本轮工具步骤 (落库与跨轮回放的素材)。
// agent 默认并行执行同轮的多个工具调用, 事件与记录可能来自不同 goroutine, 需并发安全。
type stepRecorder struct {
	mu    sync.Mutex
	seq   int64
	steps []v1.AiToolStep
}

// nextID 生成前端步骤条需要的稳定键 (模型侧 call id 不透传到本层, 自产自足)。
func (r *stepRecorder) nextID(name string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	return fmt.Sprintf("%s#%d", name, r.seq)
}

// add 记录一个已完成的步骤。
func (r *stepRecorder) add(step v1.AiToolStep) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps = append(r.steps, step)
}

// all 取全部步骤副本 (无步骤返回 nil)。
func (r *stepRecorder) all() []v1.AiToolStep {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.steps) == 0 {
		return nil
	}
	out := make([]v1.AiToolStep, len(r.steps))
	copy(out, r.steps)
	return out
}

// stepTool 工具装饰器: 执行前发 toolStart 事件 (前端步骤条转圈), 执行完发 toolEnd (带结果或错误),
// 并记录步骤。执行失败也转成 {"error":...} 结果文本回填, 让模型据此调整而不是中断整轮对话 —
// 不向框架返回 error (那会中止 agent 循环)。
type stepTool struct {
	tool.InvokableTool
	name string
	e    *chatEmitter
	rec  *stepRecorder
}

// InvokableRun 包裹执行: 发事件 → 限时执行 → 记录 → 发事件。
func (t *stepTool) InvokableRun(ctx context.Context, argsInJSON string, opts ...tool.Option) (string, error) {
	step := v1.AiToolStep{Id: t.rec.nextID(t.name), Name: t.name, Args: argsInJSON}
	t.e.emitToolStart(&step)

	tctx, cancel := context.WithTimeout(ctx, toolExecTimeout)
	defer cancel()
	result, err := t.InvokableTool.InvokableRun(tctx, argsInJSON, opts...)
	if err != nil {
		step.Error = err.Error()
		result = fmt.Sprintf(`{"error":%q}`, err.Error())
	} else {
		step.Result = result
	}
	t.rec.add(step)
	t.e.emitToolEnd(&step)
	return result, nil
}

// buildTools 构造本轮对话的工具列表 (注册表 tools.Set 逐项包上步骤装饰器)。
func buildTools(e *chatEmitter, rec *stepRecorder) ([]tool.InvokableTool, error) {
	raws, err := aitools.Set()
	if err != nil {
		return nil, err
	}
	wrapped := make([]tool.InvokableTool, 0, len(raws))
	for _, it := range raws {
		info, ierr := it.Info(context.Background())
		if ierr != nil {
			return nil, ierr
		}
		wrapped = append(wrapped, &stepTool{InvokableTool: it, name: info.Name, e: e, rec: rec})
	}
	return wrapped, nil
}

// unknownToolStep 模型幻觉调用未注册工具: 同样发一对步骤事件并回填错误结果,
// 不让 tools node 直接报错中断整轮对话 (对齐旧手写循环的容错语义)。
func unknownToolStep(e *chatEmitter, rec *stepRecorder) func(ctx context.Context, name, input string) (string, error) {
	return func(_ context.Context, name, input string) (string, error) {
		step := v1.AiToolStep{Id: rec.nextID(name), Name: name, Args: input,
			Error: fmt.Sprintf("未注册的工具: %s", name)}
		e.emitToolStart(&step)
		rec.add(step)
		e.emitToolEnd(&step)
		return fmt.Sprintf(`{"error":%q}`, step.Error), nil
	}
}
