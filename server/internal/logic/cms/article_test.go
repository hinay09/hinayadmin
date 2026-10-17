// article_test.go 纯函数测试: 富文本正文 过滤 → 图片引用归一 的组合行为
// (不触库, 只测字符串变换)。
package cms

import (
	"strings"
	"testing"

	"hinay.cn/admin/utility/sanitizer"
)

func TestNormalizeContentImgs(t *testing.T) {
	// 常规链路: 编辑器产物 src=预签名 URL + data-ref=持久引用 → 归一为持久引用。
	src := `<p><img src="https://presigned.example.com/x.png" data-ref="s3://b/k" alt="x"></p>`
	out := normalizeContentImgs(sanitizer.RichText(src))
	if !strings.Contains(out, `src="s3://b/k"`) || !strings.Contains(out, `data-ref="s3://b/k"`) {
		t.Fatalf("预签名 src 未归一为持久引用: %s", out)
	}

	// 直接 API 塞 s3:// src: sanitizer 剥掉非白名单 scheme 的 src, 归一从 data-ref 补回。
	src2 := `<img src="s3://b/k2" data-ref="s3://b/k2">`
	out2 := normalizeContentImgs(sanitizer.RichText(src2))
	if !strings.Contains(out2, `src="s3://b/k2"`) {
		t.Fatalf("被剥掉的 src 未从 data-ref 补回: %s", out2)
	}
	if strings.Count(out2, `src="s3://b/k2"`) != 1 {
		t.Fatalf("src 补回重复: %s", out2)
	}

	// 无 data-ref 的外链媒体原样保留 (http/https 不被误杀)。
	src3 := `<img src="https://cdn.example.com/foreign.png">`
	out3 := normalizeContentImgs(sanitizer.RichText(src3))
	if !strings.Contains(out3, `src="https://cdn.example.com/foreign.png"`) {
		t.Fatalf("外链图片被误杀: %s", out3)
	}

	// 已是持久引用 (src == data-ref): 不重复改写。
	src4 := `<img src="s3://b/k" data-ref="s3://b/k">`
	out4 := normalizeContentImgs(sanitizer.RichText(src4))
	if strings.Count(out4, "s3://b/k") != 2 {
		t.Fatalf("已归一内容被改写: %s", out4)
	}
}
