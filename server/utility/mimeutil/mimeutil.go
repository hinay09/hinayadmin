// Package mimeutil MIME 类型工具。
package mimeutil

import (
	"net/http"
	"strings"
)

// Base 去除 DetectContentType 结果中的 charset 等参数,
// 如 "text/html; charset=utf-8" -> "text/html", 便于与白/黑名单精确比较。
func Base(m string) string {
	if i := strings.IndexByte(m, ';'); i >= 0 {
		m = m[:i]
	}
	return strings.TrimSpace(m)
}

// Detect 读取数据开头的真实内容类型(已去除参数部分)。
func Detect(head []byte) string {
	return Base(http.DetectContentType(head))
}
