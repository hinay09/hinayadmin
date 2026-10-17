// Package sanitizer HTML XSS 过滤收口 (bluemonday) —— 富文本字段入库前在这里
// 白名单清洗, 纯文本字段在这里剥标签。当前使用方: CMS 文章/栏目 (logic/cms)。
//
// 约定: 过滤发生在 logic 层写库之前, 且必须先于其它正文变换 (CMS 的
// normalizeContentImgs 归一) —— 过滤剥离非白名单 scheme 的 src 后再由
// data-ref 补回, 安全过滤与引用归一互不腐蚀。
package sanitizer

import (
	"regexp"

	"github.com/microcosm-cc/bluemonday"
)

// styleProps 内联样式属性白名单 —— 刻意不含 background(-image)/list-style-image/
// behavior/filter 等可挂载 url()/expression() 的属性 (bluemonday 按 CSS 声明逐条
// 过滤, 不在名单上的声明整体丢弃), 覆盖 wangeditor 的常规输出 (颜色/字号/对齐等)。
var styleProps = []string{
	// 颜色 / 字体
	"color", "background-color",
	"font-size", "font-weight", "font-style", "font-family",
	// 文本排版
	"text-align", "text-indent", "text-decoration", "text-transform",
	"line-height", "letter-spacing", "word-spacing", "white-space",
	// 盒模型 (表格/媒体尺寸与间距)
	"width", "height", "max-width", "min-height",
	"margin", "margin-top", "margin-right", "margin-bottom", "margin-left",
	"padding", "padding-top", "padding-right", "padding-bottom", "padding-left",
	"border", "border-top", "border-right", "border-bottom", "border-left",
	"border-radius", "border-color", "border-width", "border-style",
	// 其它安全表现属性
	"display", "vertical-align", "overflow", "list-style-type",
}

var (
	// languageClassRe 代码块高亮 class: <pre><code class="language-javascript">
	languageClassRe = regexp.MustCompile(`^language-[a-zA-Z0-9+#_-]*$`)
	// targetRe 链接打开方式仅允许新窗口一种取值。
	targetRe = regexp.MustCompile(`^_blank$`)
)

var (
	richPolicy  = buildRichTextPolicy()
	plainPolicy = bluemonday.StrictPolicy()
)

// buildRichTextPolicy 在 UGCPolicy (面向用户生成内容的白名单) 基础上补齐
// wangeditor 富文本产物的表达面: 内联样式、媒体标签、上传持久引用 data-ref、
// 代码块语言 class 与链接 target。URL 类属性 (href/src/poster) 仍受
// UGCPolicy 的 scheme 白名单 (http/https/mailto + 相对路径) 约束。
func buildRichTextPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	// 常规元素补齐 (UGCPolicy 未放行的补上, 已放行的重复 Allow 无副作用)。
	p.AllowElements("span", "div", "section", "u", "s")

	// 媒体: 图片/视频/音频, data-ref 为文件上传持久引用
	// (入库归一/出站重写依赖, 见 logic/cms normalizeContentImgs)。
	p.AllowAttrs("src", "alt", "width", "height", "data-ref").OnElements("img")
	p.AllowAttrs("src", "poster", "preload", "controls", "width", "height", "data-ref").OnElements("video")
	p.AllowAttrs("src", "controls", "preload").OnElements("audio")
	p.AllowAttrs("src", "type").OnElements("source")

	// 代码块高亮 class。
	p.AllowAttrs("class").Matching(languageClassRe).OnElements("code")

	// 链接新窗口打开 (wangeditor 链接可配 target)。
	p.AllowAttrs("target").Matching(targetRe).OnElements("a")

	// 内联样式: 属性名白名单 (见 styleProps 注释)。
	p.AllowStyles(styleProps...).OnElements(
		"span", "p", "div", "section", "h1", "h2", "h3", "h4", "h5", "h6",
		"li", "td", "th", "blockquote", "pre", "a", "strong", "em", "u", "s",
	)

	return p
}

// RichText 富文本清理: 白名单过滤 HTML —— script/iframe、事件属性 (onerror 等)、
// 危险 URL scheme (javascript: 等)、非白名单 style 声明全部剥离, 保留 wangeditor
// 常规产物。无标签文本原样保留。
func RichText(s string) string {
	return richPolicy.Sanitize(s)
}

// PlainText 纯文本清理: 剥除全部标签 (标签体文本保留), 用于标题/摘要/备注等
// 不应含 HTML 的字段 —— 标题里的 <script> 会变成可见文本而非可执行代码。
func PlainText(s string) string {
	return plainPolicy.Sanitize(s)
}
