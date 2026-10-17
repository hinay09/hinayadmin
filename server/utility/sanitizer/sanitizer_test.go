// sanitizer_test.go 纯函数测试: XSS 向量剥离与 wangeditor 产物保留。
package sanitizer

import (
	"strings"
	"testing"
)

func TestRichTextStripsScript(t *testing.T) {
	out := RichText(`<p>正文</p><script>alert(1)</script><iframe src="https://evil.com"></iframe>`)
	if strings.Contains(out, "script") || strings.Contains(out, "iframe") {
		t.Fatalf("script/iframe 未剥离: %s", out)
	}
	if !strings.Contains(out, "<p>正文</p>") {
		t.Fatalf("正常段落被破坏: %s", out)
	}
}

func TestRichTextStripsEventHandlers(t *testing.T) {
	out := RichText(`<img src="https://cdn.example.com/a.png" onerror="alert(1)" onload="steal()">`)
	if strings.Contains(out, "onerror") || strings.Contains(out, "onload") {
		t.Fatalf("事件属性未剥离: %s", out)
	}
	if !strings.Contains(out, `src="https://cdn.example.com/a.png"`) {
		t.Fatalf("合法 https 图片被误杀: %s", out)
	}
}

func TestRichTextStripsDangerousScheme(t *testing.T) {
	out := RichText(`<a href="javascript:alert(1)">点我</a><img src="data:text/html;base64,x">`)
	if strings.Contains(out, "javascript:") || strings.Contains(out, "data:") {
		t.Fatalf("危险 scheme 未剥离: %s", out)
	}
	// 相对路径 (本站 /upload) 与 http(s) 是放行的。
	if !strings.Contains(RichText(`<img src="/upload/a.png">`), `src="/upload/a.png"`) {
		t.Fatal("相对路径图片被误杀")
	}
}

func TestRichTextKeepsMediaRef(t *testing.T) {
	out := RichText(`<img src="https://presigned.example.com/x.png" data-ref="s3://bucket/key"><video src="https://p.example.com/v.mp4" data-ref="s3://bucket/v"></video>`)
	if !strings.Contains(out, `data-ref="s3://bucket/key"`) {
		t.Fatalf("img data-ref 持久引用被剥离: %s", out)
	}
	if !strings.Contains(out, `data-ref="s3://bucket/v"`) {
		t.Fatalf("video data-ref 持久引用被剥离: %s", out)
	}
	// 非 URL 类的自定义属性不受 scheme 白名单影响。
	if !strings.Contains(out, "<video") || !strings.Contains(out, "</video>") {
		t.Fatalf("video 标签被破坏: %s", out)
	}
}

func TestRichTextStyleWhitelist(t *testing.T) {
	out := RichText(`<span style="color: rgb(225,60,57); background-image: url(javascript:alert(1))">红字</span>`)
	if !strings.Contains(out, "color: rgb(225,60,57)") && !strings.Contains(out, "color:rgb(225,60,57)") {
		t.Fatalf("安全样式 color 被误杀: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "background-image") || strings.Contains(strings.ToLower(out), "url(") {
		t.Fatalf("危险样式声明未剥离: %s", out)
	}
}

func TestRichTextKeepsWangeditorOutput(t *testing.T) {
	in := `<h2 style="text-align: center">标题</h2>` +
		`<p style="text-align: center; line-height: 1.5">居中段落 <strong>加粗</strong> <em>斜体</em> <u>下划线</u> <span style="font-size: 24px; color: rgb(51,51,51)">大字</span></p>` +
		`<blockquote>引用</blockquote><ul><li>项</li></ul><pre><code class="language-javascript">var a = 1;</code></pre>` +
		`<table><tbody><tr><th>表头</th></tr><tr><td style="width: 120px">单元格</td></tr></tbody></table>` +
		`<a href="https://example.com" target="_blank">外链</a><hr/><br/>`
	out := RichText(in)
	for _, want := range []string{
		`text-align: center`, `<strong>加粗</strong>`, `<em>斜体</em>`, `<u>下划线</u>`,
		`font-size: 24px`, "<blockquote>", "<li>项</li>", `class="language-javascript"`,
		`<td style="width: 120px">`, `href="https://example.com"`, `target="_blank"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("wangeditor 常规产物被误杀, 期望含 %q: %s", want, out)
		}
	}
}

func TestPlainText(t *testing.T) {
	out := PlainText(`<b>标题</b><script>alert(1)</script>与文本 <img src=x onerror=alert(2)>`)
	if strings.ContainsAny(out, "<>") {
		t.Fatalf("标签未剥净: %s", out)
	}
	if !strings.Contains(out, "标题") || !strings.Contains(out, "与文本") {
		t.Fatalf("文本内容被破坏: %s", out)
	}
	// 无标签输入原样返回。
	if PlainText("普通文本 123") != "普通文本 123" {
		t.Fatal("无标签文本被改变")
	}
}
