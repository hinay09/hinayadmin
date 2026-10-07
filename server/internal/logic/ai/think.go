// Package ai — 内联思考标签拆分器。
//
// 背景: 推理模型的思考过程有两类到达方式——
//  1. 独立字段 reasoning_content (DeepSeek-R1 官方 API / DashScope 兼容模式等): 由
//     langchaingo 的 WithStreamingReasoningFunc 直接回调, 无需本拆分器;
//  2. 内联在 content 里的 <think>...</think> 标签 (vLLM/SGLang 自部署 R1 系常见):
//     需要把标签内文本路由到思考通道。
//
// 标签可能被拆在多个 chunk 边界, 用前缀扣留法处理; 流结束时仍在标签内则剩余全部归思考。
package ai

import "strings"

const (
	thinkOpen  = "<think>"
	thinkClose = "</think>"
)

// thinkSplitter 流式拆分: feed 逐段输入, 输出 (正文, 思考) 两通道文本。
type thinkSplitter struct {
	inThink bool
	buf     string // 扣留的可疑标签前缀 (可能是被 chunk 截断的 <think>/</think> 头部)
}

// feed 输入一段增量, 返回本轮可下发的正文与思考文本。
func (t *thinkSplitter) feed(delta string) (content, reasoning string) {
	s := t.buf + delta
	t.buf = ""
	var c, r strings.Builder
	for {
		tag := thinkClose
		if !t.inThink {
			tag = thinkOpen
		}
		if i := strings.Index(s, tag); i >= 0 {
			if t.inThink {
				r.WriteString(s[:i])
			} else {
				c.WriteString(s[:i])
			}
			s = s[i+len(tag):]
			t.inThink = !t.inThink
			continue
		}
		// 尾部疑似被截断的标签前缀 → 扣留待下一段
		if h := partialTagSuffix(s, tag); h > 0 {
			head := s[:len(s)-h]
			if t.inThink {
				r.WriteString(head)
			} else {
				c.WriteString(head)
			}
			t.buf = s[len(s)-h:]
		} else if t.inThink {
			r.WriteString(s)
		} else {
			c.WriteString(s)
		}
		break
	}
	return c.String(), r.String()
}

// flush 流结束: 扣留残余按当前通道归属下发 (未闭合 </think> 时剩余视为思考)。
func (t *thinkSplitter) flush() (content, reasoning string) {
	left := t.buf
	t.buf = ""
	if left == "" {
		return "", ""
	}
	if t.inThink {
		return "", left
	}
	return left, ""
}

// partialTagSuffix 返回 s 尾部与 tag 前缀相同的最大长度 (0=无可疑; 不含完整 tag 场景, 调用前已先行匹配)。
func partialTagSuffix(s, tag string) int {
	max := len(tag) - 1
	if max > len(s) {
		max = len(s)
	}
	for n := max; n > 0; n-- {
		if strings.HasSuffix(s, tag[:n]) {
			return n
		}
	}
	return 0
}
