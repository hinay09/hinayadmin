// Package ai — 流式输出特殊 token 过滤。
//
// 背景: 部分模型 (DeepSeek 系列等) 会在输出中泄漏 <｜end▁of▁sentence｜> / <|im_end|>
// 等特殊 token, 且可能被截断成多段 (甚至夹着损坏字节) 分布在不同 chunk 中,
// 必须带缓冲拼接判定后再下发前端, 否则界面出现 <｜ 之类的乱码片段。
//
// 规则 (对累积缓冲自左向右扫描):
//  1. 锚定首位的完整特殊 token  → 剥离;
//  2. "<｜/<|" 开头、未闭合、其后紧跟下一个 '<' 且 ≤48 字 → 视为被截断的垃圾, 整段吸收;
//  3. 已遇 '>' 或换行仍未构成特殊 token → '<' 为普通字符, 放行;
//  4. "<｜/<|" 开头、无终止线索且 ≤48 字 → 扣留, 等待后续 chunk 拼接判定;
//  5. 其余 (如 "a < b") → 正常放行。
package ai

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// specialTokenRe 完整特殊 token: <｜xxx｜> / <|xxx|> (内部不含 界符/>/换行, ≤48 字)。
var specialTokenRe = regexp.MustCompile(`<[｜|][^｜|>\n]{0,48}[｜|]>`)

// streamFilter 流式过滤器: feed 逐段输入, 返回可安全下发文本; 流结束调用 flush。
type streamFilter struct {
	buf string
}

// feed 输入一段增量, 返回当前可安全下发的文本 (可能为空: 全部被扣留待拼接)。
func (f *streamFilter) feed(delta string) string {
	f.buf += delta
	var out strings.Builder
	for f.buf != "" {
		if f.buf[0] != '<' {
			// 发出至下一个 '<'
			j := strings.IndexByte(f.buf, '<')
			if j < 0 {
				out.WriteString(f.buf)
				f.buf = ""
				break
			}
			out.WriteString(f.buf[:j])
			f.buf = f.buf[j:]
			continue
		}
		// 完整特殊 token (锚定首位)
		if m := specialTokenRe.FindStringIndex(f.buf); m != nil && m[0] == 0 {
			f.buf = f.buf[m[1]:]
			continue
		}
		isBarPrefix := len(f.buf) > 1 && (f.buf[1] == 0xEF || f.buf[1] == '|') // '｜' UTF-8 首字节或 '|'
		nxt := strings.IndexByte(f.buf[1:], '<')                               // 下一个 '<'
		if nxt >= 0 {
			nxt++
		}
		stop := strings.IndexAny(f.buf[1:], ">\n") // 最早的 '>' 或换行
		if stop >= 0 {
			stop++
		}
		// 被截断的特殊 token 后紧跟下一个 '<': 整段吸收为垃圾
		if isBarPrefix && nxt > 0 && (stop < 0 || nxt < stop) && utf8.RuneCountInString(f.buf[1:nxt]) <= 48 {
			f.buf = f.buf[nxt:]
			continue
		}
		// 已闭合/换行仍未构成特殊 token: '<' 为普通字符
		if stop > 0 && (nxt < 0 || stop < nxt) {
			out.WriteByte('<')
			f.buf = f.buf[1:]
			continue
		}
		// 无终止线索: 疑似未完的特殊 token → 扣留; 否则放行 '<'
		if isBarPrefix && utf8.RuneCountInString(f.buf) <= 49 {
			return out.String()
		}
		out.WriteByte('<')
		f.buf = f.buf[1:]
	}
	return out.String()
}

// flush 流结束: 剩余缓冲 (含始终未闭合的 '<') 按普通文本下发。
func (f *streamFilter) flush() string {
	out := f.buf
	f.buf = ""
	return out
}
