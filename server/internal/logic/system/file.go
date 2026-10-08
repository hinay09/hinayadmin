// Package system 系统管理-文件业务逻辑。
package system

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/internal/storage"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/mimeutil"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sFile struct{}

func init() {
	service.RegisterFile(NewFile())
}

func NewFile() *sFile {
	return &sFile{}
}

// maxUploadSize 单文件大小上限(20MB, 与 nginx client_max_body_size 保持一致)。
const maxUploadSize = 20 << 20

// allowedUploadExts 上传扩展名白名单。
// 安全约束: 排除 html/htm/svg/xml 等可被浏览器当页面执行的类型(存储型 XSS 载体),
// 以及 php/jsp/sh 等可执行脚本(防止反代/托管环境开启脚本解析导致 RCE)。
var allowedUploadExts = map[string]bool{
	// 图片
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true, ".ico": true,
	// 文档
	".pdf": true, ".txt": true, ".csv": true, ".md": true,
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	// 压缩包
	".zip": true, ".rar": true, ".7z": true, ".tar": true, ".gz": true,
	// 音视频
	".mp3": true, ".mp4": true, ".wav": true, ".mov": true,
	// 数据
	".json": true,
}

// dangerousContentTypes 内容嗅探识别为这些类型的文件直接拒绝
// (真实内容是页面/脚本的文件, 即使扩展名在白名单内也不允许)。
var dangerousContentTypes = map[string]bool{
	"text/html":             true,
	"image/svg+xml":         true,
	"application/xhtml+xml": true,
	"text/xml":              true,
	"application/xml":       true,
}

// presignKeyPattern 预签名 key 形态: {yyyymmdd}/{guid}{ext},
// 确认接口只认本服务签发的 key, 防止对桶内任意对象"认领落库"。
var presignKeyPattern = regexp.MustCompile(`^[0-9]{8}/[0-9a-z]{20,64}(\.[a-z0-9]{1,10})$`)

// List 分页列表。
// S3 存储的记录 (url=s3://...) 出参改写为带有效期的预签名 GET 地址
// (SigV4 签名是本地计算, 批量生成无网络开销); 本地记录保持 /upload 相对路径。
func (s *sFile) List(ctx context.Context, req *v1.FileListReq) (res *v1.FileListRes, err error) {
	q := dao.SysFile.Ctx(ctx).Where("deleted_at IS NULL")

	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("original_name LIKE ?", kw)
	}
	if req.MimeType != "" {
		q = q.Where("mime_type LIKE ?", req.MimeType+"%")
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	var rows []*model.FileItem
	if err = q.Page(req.Page, req.PageSize).Order("id DESC").Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	var s3st storage.Storage
	for _, row := range rows {
		if !strings.HasPrefix(row.Url, "s3://") {
			continue
		}
		if s3st == nil {
			if st, serr := storage.ForURL(ctx, row.Url); serr != nil {
				g.Log().Warningf(ctx, "resolve s3 storage failed, keep ref url: %v", serr)
				break
			} else {
				s3st = st
			}
		}
		if u, uerr := s3st.URL(ctx, row.Path, row.OriginalName, storage.IsInlineExt(row.Extension)); uerr == nil {
			row.Url = u
		}
	}

	page := response.Page(rows, int64(total), req.Page, req.PageSize)
	r := v1.FileListRes(page)
	return &r, nil
}

// Upload 上传文件 (服务端中转)。
// 校验链: 扩展名白名单 -> 大小上限 -> 文件头内容嗅探(拒绝页面/脚本类真实内容)。
// 本地存储写磁盘 resource/upload, S3 存储经 AWS SDK 中转写入对象存储。
func (s *sFile) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (res *v1.FileUploadRes, err error) {
	// 审计: 暂存文件摘要, OperationLog 记录入库 (校验失败也留痕)
	contextx.SetAuditUpload(ctx, fmt.Sprintf("%s (%d bytes, %s)",
		filepath.Base(header.Filename), header.Size, header.Header.Get("Content-Type")))
	originalName := filepath.Base(header.Filename)
	ext := strings.ToLower(filepath.Ext(originalName))
	if !allowedUploadExts[ext] {
		return nil, xerror.New(xerror.CodeBusinessError, "不支持的文件类型: "+ext)
	}
	if header.Size > maxUploadSize {
		return nil, xerror.New(xerror.CodeBusinessError, "文件大小不能超过 20MB")
	}

	// 内容嗅探: 读文件头识别真实类型, 拒绝伪装扩展名的页面/脚本内容
	var mimeType string
	head := make([]byte, 512)
	n, _ := file.Read(head)
	mimeType = mimeutil.Detect(head[:n])
	if dangerousContentTypes[mimeType] {
		return nil, xerror.New(xerror.CodeBusinessError, "文件内容包含可执行页面, 已拒绝")
	}
	if _, serr := file.Seek(0, io.SeekStart); serr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, serr, "读取文件失败")
	}

	st, err := storage.Current(ctx)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "文件存储未就绪")
	}

	key := storage.NewKey(ext)
	if err = st.Put(ctx, key, file, header.Size, mimeType); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	id, err := s.insertFileRecord(ctx, key, originalName, st.RefURL(key), header.Size, mimeType, ext)
	if err != nil {
		// 落库失败回收已写入对象 (尽力而为)
		go func() { _ = st.Delete(context.Background(), key) }()
		return nil, err
	}

	return &v1.FileUploadRes{
		Id:           uint64(id),
		Name:         filepath.Base(key),
		OriginalName: originalName,
		Url:          s.viewURL(ctx, st, key, originalName, ext),
		Size:         uint64(header.Size),
		Extension:    ext,
	}, nil
}

// Presign 获取预签名直传地址 (S3 存储模式的默认上传路径)。
// 直传不经服务端, 无法做文件头内容嗅探: 以签名约束 Content-Type (客户端必须
// 原样携带), 声明类型命中黑名单直接拒绝; 实际大小由确认接口 HeadObject 把关。
func (s *sFile) Presign(ctx context.Context, req *v1.FilePresignReq) (res *v1.FilePresignRes, err error) {
	originalName := filepath.Base(req.OriginalName)
	ext := strings.ToLower(filepath.Ext(originalName))
	if !allowedUploadExts[ext] {
		return nil, xerror.New(xerror.CodeBusinessError, "不支持的文件类型: "+ext)
	}
	if req.Size > maxUploadSize {
		return nil, xerror.New(xerror.CodeBusinessError, "文件大小不能超过 20MB")
	}
	contentType := mimeutil.Base(strings.TrimSpace(req.ContentType))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if dangerousContentTypes[contentType] {
		return nil, xerror.New(xerror.CodeBusinessError, "不支持的文件类型: "+contentType)
	}

	st, err := storage.Current(ctx)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "文件存储未就绪")
	}
	// 本地存储无预签名能力, 前端回落到普通上传接口
	if st.Type() == storage.TypeLocal {
		return &v1.FilePresignRes{Mode: "server"}, nil
	}

	key := storage.NewKey(ext)
	uploadURL, err := st.PresignPut(ctx, key, contentType)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.FilePresignRes{
		Mode:        "presign",
		UploadUrl:   uploadURL,
		Method:      "PUT",
		Key:         key,
		ContentType: contentType,
		ExpireSec:   600,
	}, nil
}

// PresignConfirm 预签名直传确认: HeadObject 校验对象确实存在且未超限后落库。
func (s *sFile) PresignConfirm(ctx context.Context, req *v1.FilePresignConfirmReq) (res *v1.FilePresignConfirmRes, err error) {
	key := strings.TrimSpace(req.Key)
	if !presignKeyPattern.MatchString(key) {
		return nil, xerror.New(xerror.CodeBusinessError, "非法的文件key")
	}
	ext := strings.ToLower(filepath.Ext(key))
	if !allowedUploadExts[ext] {
		return nil, xerror.New(xerror.CodeBusinessError, "不支持的文件类型: "+ext)
	}

	st, err := storage.Current(ctx)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "文件存储未就绪")
	}
	if st.Type() == storage.TypeLocal {
		return nil, xerror.New(xerror.CodeBusinessError, "当前为本地存储, 不支持预签名上传")
	}

	size, objCT, err := st.Stat(ctx, key)
	if err != nil {
		return nil, xerror.New(xerror.CodeBusinessError, "文件尚未上传完成或已过期, 请重新上传")
	}
	if size > maxUploadSize {
		go func() { _ = st.Delete(context.Background(), key) }()
		return nil, xerror.New(xerror.CodeBusinessError, "文件大小不能超过 20MB")
	}

	// 幂等: 网络重试导致重复确认时直接返回已有记录
	var exist *model.FileItem
	if err = dao.SysFile.Ctx(ctx).
		Where("path", key).Where("deleted_at IS NULL").
		Order("id DESC").Scan(&exist); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	originalName := filepath.Base(req.OriginalName)
	if exist != nil {
		return &v1.FilePresignConfirmRes{
			Id:           exist.Id,
			Name:         exist.Name,
			OriginalName: exist.OriginalName,
			Url:          s.viewURL(ctx, st, key, exist.OriginalName, ext),
			Size:         exist.Size,
			Extension:    exist.Extension,
		}, nil
	}

	mimeType := mimeutil.Base(strings.TrimSpace(objCT))
	if mimeType == "" {
		mimeType = mimeutil.Base(strings.TrimSpace(req.ContentType))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	id, err := s.insertFileRecord(ctx, key, originalName, st.RefURL(key), size, mimeType, ext)
	if err != nil {
		return nil, err
	}
	return &v1.FilePresignConfirmRes{
		Id:           uint64(id),
		Name:         filepath.Base(key),
		OriginalName: originalName,
		Url:          s.viewURL(ctx, st, key, originalName, ext),
		Size:         uint64(size),
		Extension:    ext,
	}, nil
}

// Delete 删除文件（软删记录）。
func (s *sFile) Delete(ctx context.Context, req *v1.FileDeleteReq) (res *v1.FileDeleteRes, err error) {
	// 查询文件记录
	var f *model.SysFile
	if err = dao.SysFile.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").Scan(&f); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if f == nil {
		return nil, xerror.New(xerror.CodeNotFound, "文件不存在")
	}
	// 软删记录
	if _, err = dao.SysFile.Ctx(ctx).
		Where("id", req.Id).Data(g.Map{"deleted_at": gtime.Now()}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 尝试删除物理对象（非阻塞, 按记录所在存储路由: 本地文件/对象存储）
	st, serr := storage.ForURL(ctx, f.Url)
	if serr != nil {
		g.Log().Warningf(ctx, "resolve storage for file %d failed: %v", req.Id, serr)
		return &v1.FileDeleteRes{}, nil
	}
	go func() {
		bg := context.Background()
		if derr := st.Delete(bg, f.Path); derr != nil {
			g.Log().Warningf(bg, "delete file object %s failed: %v", f.Path, derr)
		}
	}()
	return &v1.FileDeleteRes{}, nil
}

// insertFileRecord 写入 sys_file 记录。
func (s *sFile) insertFileRecord(ctx context.Context, key, originalName, refURL string,
	size int64, mimeType, ext string) (int64, error) {
	userId := contextx.UserId(ctx)
	id, err := dao.SysFile.Ctx(ctx).Data(g.Map{
		"name":          filepath.Base(key),
		"original_name": originalName,
		"path":          key,
		"url":           refURL,
		"size":          size,
		"mime_type":     mimeType,
		"extension":     ext,
		"user_id":       userId,
	}).InsertAndGetId()
	if err != nil {
		return 0, xerror.Wrap(xerror.CodeBusinessError, err, "保存文件记录失败")
	}
	return id, nil
}

// viewURL 生成展示/下载用的即时访问地址 (S3 为带有效期的预签名 URL)。
func (s *sFile) viewURL(ctx context.Context, st storage.Storage, key, originalName, ext string) string {
	u, err := st.URL(ctx, key, originalName, storage.IsInlineExt(ext))
	if err != nil {
		g.Log().Warningf(ctx, "generate file view url failed: %v", err)
		return st.RefURL(key)
	}
	return u
}
