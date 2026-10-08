package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"hinay.cn/admin/utility/mimeutil"
)

// presignPutTTL 预签名直传 URL 有效期 (客户端拿到后立即上传, 无需更长)。
const presignPutTTL = 10 * time.Minute

// s3Storage S3 兼容对象存储 (MinIO / AWS S3 / 阿里云 OSS / 腾讯云 COS 等,
// 统一走 AWS SDK 的 SigV4 兼容协议)。
type s3Storage struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	expire  time.Duration // 预签名 GET 有效期
}

func newS3(cfg *Config) (*s3Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 存储未配置: 缺少 file.storage.bucket")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("S3 存储未配置: 缺少 file.storage.access_key / file.storage.secret_key")
	}
	awscfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("初始化 S3 客户端失败: %w", err)
	}
	client := s3.NewFromConfig(awscfg, func(o *s3.Options) {
		o.Region = cfg.Region
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		// MinIO/OSS/COS 自建服务必须路径风格寻址 (virtual-host 风格会解析到错误域名)
		o.UsePathStyle = cfg.PathStyle
	})
	return &s3Storage{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  cfg.Bucket,
		expire:  cfg.PresignExpire,
	}, nil
}

func (s *s3Storage) Type() string { return TypeS3 }

func (s *s3Storage) RefURL(key string) string {
	return "s3://" + s.bucket + "/" + key
}

// contentTypeOf Content-Type 缺省时按扩展名推导。
func contentTypeOf(key, ct string) string {
	if strings.TrimSpace(ct) != "" {
		return mimeutil.Base(ct)
	}
	return mime.TypeByExtension(strings.ToLower(path.Ext(key)))
}

func (s *s3Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentTypeOf(key, contentType)),
	})
	if err != nil {
		return fmt.Errorf("上传到对象存储失败: %w", err)
	}
	return nil
}

func (s *s3Storage) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("删除对象存储文件失败: %w", err)
	}
	return nil
}

func (s *s3Storage) Stat(ctx context.Context, key string) (int64, string, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, "", fmt.Errorf("查询对象失败: %w", err)
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return size, aws.ToString(out.ContentType), nil
}

// URL 生成预签名 GET 下载地址:
//   - 非内联类型 (非图片扩展名) 通过签名的 response-content-disposition 强制
//     附件下载并指定文件名, 与本地 /upload 静态服务的 XSS 纵深防御策略对齐;
//   - 签名仅是本地计算 (SigV4 HMAC), 不产生网络请求, 可按行批量生成。
func (s *s3Storage) URL(ctx context.Context, key, filename string, inline bool) (string, error) {
	if filename == "" {
		filename = path.Base(key)
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}
	if inline {
		input.ResponseContentDisposition = aws.String("inline")
	} else {
		// RFC 5987 编码, 兼容中文文件名
		input.ResponseContentDisposition = aws.String(
			"attachment; filename*=UTF-8''" + url.PathEscape(filename))
	}
	req, err := s.presign.PresignGetObject(ctx, input, s3.WithPresignExpires(s.expire))
	if err != nil {
		return "", fmt.Errorf("生成预签名下载地址失败: %w", err)
	}
	return req.URL, nil
}

// PresignPut 生成预签名直传地址: Content-Type 参与签名, 客户端必须原样携带;
// 文件内容不经过应用服务器 (内容嗅探防线仅在服务端中转上传路径生效)。
func (s *s3Storage) PresignPut(ctx context.Context, key, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	req, err := s.presign.PresignPutObject(ctx, input, s3.WithPresignExpires(presignPutTTL))
	if err != nil {
		return "", fmt.Errorf("生成预签名上传地址失败: %w", err)
	}
	return req.URL, nil
}
