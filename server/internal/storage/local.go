package storage

import (
	"context"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// localRoot 本地存储物理根目录 (与 cmd.go 的 /upload 静态服务一致)。
const localRoot = "resource/upload"

// localStorage 本地磁盘存储。
type localStorage struct{}

func newLocal() *localStorage { return &localStorage{} }

func (s *localStorage) Type() string { return TypeLocal }

// normalize 归一化 key: 兼容历史记录中的 resource/upload / /upload 前缀与反斜杠,
// 并以 "/" 前缀 Clean 阻止 ".." 路径穿越。
func (s *localStorage) normalize(key string) string {
	k := strings.ReplaceAll(key, "\\", "/")
	k = strings.TrimPrefix(k, localRoot+"/")
	k = strings.TrimPrefix(k, "/upload/")
	return strings.TrimPrefix(path.Clean("/"+k), "/")
}

// diskPath key -> 物理路径。
func (s *localStorage) diskPath(key string) string {
	return filepath.Join(localRoot, s.normalize(key))
}

func (s *localStorage) RefURL(key string) string {
	return "/upload/" + s.normalize(key)
}

func (s *localStorage) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	p := s.diskPath(key)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	dst, err := os.Create(p)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, r)
	return err
}

func (s *localStorage) Delete(_ context.Context, key string) error {
	if err := os.Remove(s.diskPath(key)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *localStorage) Stat(_ context.Context, key string) (int64, string, error) {
	fi, err := os.Stat(s.diskPath(key))
	if err != nil {
		return 0, "", err
	}
	return fi.Size(), mime.TypeByExtension(strings.ToLower(path.Ext(key))), nil
}

// URL 本地存储无过期概念, 返回站内静态服务相对路径 (忽略 filename/inline,
// 由 /upload 路由按扩展名决定内联或附件下载)。
func (s *localStorage) URL(_ context.Context, key, _ string, _ bool) (string, error) {
	return s.RefURL(key), nil
}

// PresignPut 本地存储不支持预签名。
func (s *localStorage) PresignPut(_ context.Context, _, _ string) (string, error) {
	return "", ErrUnsupportedPresign
}
