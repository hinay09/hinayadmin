// Package storage 文件存储抽象。
//
// 本地磁盘 (local, 默认) 与 S3 兼容对象存储 (s3: MinIO / AWS S3 / 阿里云 OSS /
// 腾讯云 COS 等, 统一走 AWS SDK) 两套实现, 存储类型与连接参数配置在 sys_config
// (file.storage.* 键, 由 系统管理-全局配置 页维护), 修改后无需重启 —— 客户端按
// 配置指纹懒重建。
//
// S3 模式下上传/下载默认走预签名接口: 上传为预签名 PUT 直传 (文件不经过应用
// 服务器, 见 logic/system 的 presign 流程), 下载地址为带有效期的预签名 GET URL。
package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/guid"

	"hinay.cn/admin/internal/dao"
)

// 存储类型标识 (sys_config: file.storage.type)。
const (
	TypeLocal = "local"
	TypeS3    = "s3"
)

// cfgPrefix file.storage.* 配置键前缀。
const cfgPrefix = "file.storage."

// ErrUnsupportedPresign 本地存储不支持预签名直传。
var ErrUnsupportedPresign = fmt.Errorf("本地存储不支持预签名上传, 请使用服务端上传接口")

// Storage 文件存储接口。
//
// key 为对象相对路径 (如 20260108/0cefgz8l3kc6kw1x2f3g4h5i6j7k8.png);
// url 列在 sys_file 中持久化的是 RefURL (稳定引用), 展示/下载用的
// 临时地址 (预签名) 由 URL 按需生成, 不落库。
type Storage interface {
	// Type 存储类型标识 (local/s3)。
	Type() string
	// RefURL 写入 sys_file.url 的持久引用:
	// local 为 /upload/<key> 站内相对路径; s3 为 s3://<bucket>/<key>。
	RefURL(key string) string
	// Put 写入对象 (服务端中转上传路径)。
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Delete 删除对象。
	Delete(ctx context.Context, key string) error
	// Stat 查询对象元信息, 不存在时返回错误。
	Stat(ctx context.Context, key string) (size int64, contentType string, err error)
	// URL 生成访问地址: local 返回站内 /upload 相对路径; s3 返回预签名 GET URL。
	// 非内联类型携带 attachment 响应头 (与本地静态服务的 XSS 纵深防御策略一致)。
	URL(ctx context.Context, key, filename string, inline bool) (string, error)
	// PresignPut 生成预签名直传 URL (仅 s3 支持), 签名 Content-Type,
	// 客户端上传时必须原样携带同名头。
	PresignPut(ctx context.Context, key, contentType string) (string, error)
}

// inlineExts 允许浏览器内联展示的扩展名, 其余类型一律作为附件下载
// (阻止上传内容被浏览器当页面执行, 存储型 XSS 纵深防御; 与扩展名白名单
// 配合见 logic/system/file.go)。
var inlineExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true, ".ico": true,
}

// IsInlineExt 扩展名是否允许浏览器内联展示。
func IsInlineExt(ext string) bool {
	return inlineExts[strings.ToLower(ext)]
}

// NewKey 生成对象相对路径: {yyyymmdd}/{guid}{ext}, 布局与历史本地存储一致,
// 便于本地模式存量数据与对象存储互认。
func NewKey(ext string) string {
	return fmt.Sprintf("%s/%s%s", gtime.Now().Format("Ymd"), guid.S(), ext)
}

// Config file.storage.* 配置快照。
type Config struct {
	Type          string        // local | s3
	Bucket        string        // 桶名
	Endpoint      string        // 服务地址, 如 http://minio:9000; 空 = SDK 默认 (AWS)
	Region        string        // 地域, MinIO 默认 us-east-1
	AccessKey     string        // 访问密钥 ID
	SecretKey     string        // 访问密钥 Secret
	PathStyle     bool          // 路径风格寻址 (MinIO/OSS/COS 需开启)
	PresignExpire time.Duration // 预签名 GET 有效期
}

// loadConfig 一次读取全部 file.storage.* 配置, 未配置项回落默认值。
func loadConfig(ctx context.Context) (*Config, error) {
	rows, err := dao.SysConfig.Ctx(ctx).
		Where("config_key LIKE ?", cfgPrefix+"%").
		Where("status", 1).
		Where("deleted_at IS NULL").
		All()
	if err != nil {
		return nil, err
	}
	m := g.MapStrStr{}
	for _, row := range rows {
		m[row["config_key"].String()] = strings.TrimSpace(row["config_value"].String())
	}

	// 默认: 本地存储; 预签名有效期 60 分钟; 路径风格 (自建 MinIO 为主场景)。
	cfg := &Config{
		Type:          TypeLocal,
		Region:        "us-east-1",
		PathStyle:     true,
		PresignExpire: 60 * time.Minute,
	}
	if v := m[cfgPrefix+"type"]; v == TypeS3 {
		cfg.Type = TypeS3
	}
	cfg.Bucket = m[cfgPrefix+"bucket"]
	cfg.Endpoint = m[cfgPrefix+"endpoint"]
	cfg.AccessKey = m[cfgPrefix+"access_key"]
	cfg.SecretKey = m[cfgPrefix+"secret_key"]
	if v := m[cfgPrefix+"region"]; v != "" {
		cfg.Region = v
	}
	if v := m[cfgPrefix+"path_style"]; v != "" {
		cfg.PathStyle = v != "false" && v != "0"
	}
	if mins := g.NewVar(m[cfgPrefix+"presign_expire"]).Int(); mins > 0 {
		cfg.PresignExpire = time.Duration(mins) * time.Minute
	}
	return cfg, nil
}

// fingerprint 配置指纹, 变化时重建客户端。
func (c *Config) fingerprint() string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%v|%s",
		c.Type, c.Bucket, c.Endpoint, c.Region, c.AccessKey, c.SecretKey, c.PathStyle, c.PresignExpire)
}

var (
	mu        sync.RWMutex
	cached    Storage
	cachedFp  string
	localOnce Storage
)

// Local 本地磁盘存储单例 (无论当前配置如何, 历史本地文件的删除/访问都走它)。
func Local() Storage {
	if localOnce == nil {
		localOnce = newLocal()
	}
	return localOnce
}

// Current 返回当前存储实现: 按 sys_config 解析, 客户端按配置指纹缓存,
// 配置变更后的下一次调用自动重建。
func Current(ctx context.Context) (Storage, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		// 数据库抖动时回落已缓存实例, 避免短暂故障导致上传/下载不可用
		mu.RLock()
		s := cached
		mu.RUnlock()
		if s != nil {
			return s, nil
		}
		return nil, err
	}
	fp := cfg.fingerprint()
	mu.RLock()
	s := cached
	old := cachedFp
	mu.RUnlock()
	if s != nil && old == fp {
		return s, nil
	}

	ns, err := build(cfg)
	if err != nil {
		return nil, err
	}
	mu.Lock()
	cached = ns
	cachedFp = fp
	mu.Unlock()
	return ns, nil
}

// build 按配置构建存储实现。
func build(cfg *Config) (Storage, error) {
	switch cfg.Type {
	case TypeLocal:
		return Local(), nil
	case TypeS3:
		return newS3(cfg)
	default:
		return nil, fmt.Errorf("未知的文件存储类型: %s (可选 local/s3)", cfg.Type)
	}
}

// ForURL 按文件记录的持久引用 (sys_file.url) 推断其所在存储:
// "/upload/..." 为本地, "s3://bucket/key" 为对象存储 (连接参数仍读 file.storage.*,
// 即便当前类型已切回 local, 存量 S3 文件的删除/取链接依旧可用)。
func ForURL(ctx context.Context, fileURL string) (Storage, error) {
	if strings.HasPrefix(fileURL, "s3://") {
		cfg, err := loadConfig(ctx)
		if err != nil {
			return nil, err
		}
		return newS3(cfg)
	}
	return Local(), nil
}

// ViewURL 将持久引用转换为可直接给浏览器渲染的即时地址 (如头像 <img src>):
// 本地 /upload/... 原样返回; s3://... 生成内联预签名 GET URL, 失败时原样返回
// (引用值入库、展示地址出参, 出口处统一调用, 见 auth/user 的 avatar 字段)。
func ViewURL(ctx context.Context, refURL string) string {
	if !strings.HasPrefix(refURL, "s3://") {
		return refURL
	}
	st, err := ForURL(ctx, refURL)
	if err != nil {
		return refURL
	}
	u, err := st.URL(ctx, KeyOfRefURL(refURL), "", true)
	if err != nil {
		return refURL
	}
	return u
}

// KeyOfRefURL 从持久引用中提取对象 key ("s3://bucket/key" -> "key"),
// 本地引用原样返回 (local 实现内部会归一化 resource/upload 前缀)。
func KeyOfRefURL(fileURL string) string {
	if strings.HasPrefix(fileURL, "s3://") {
		if i := strings.Index(fileURL[5:], "/"); i >= 0 {
			return fileURL[5+i+1:]
		}
	}
	return fileURL
}

// compile-time 接口实现约束。
var (
	_ Storage = (*localStorage)(nil)
	_ Storage = (*s3Storage)(nil)
)
