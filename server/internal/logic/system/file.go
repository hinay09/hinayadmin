// Package system 系统管理-文件业务逻辑。
package system

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/guid"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
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

// uploadDir 上传文件存储目录。
const uploadDir = "resource/upload"

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

// ensureDir 确保目录存在。
func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// List 分页列表。
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

	page := response.Page(rows, int64(total), req.Page, req.PageSize)
	r := v1.FileListRes(page)
	return &r, nil
}

// Upload 上传文件。
// 校验链: 扩展名白名单 -> 大小上限 -> 文件头内容嗅探(拒绝页面/脚本类真实内容)。
func (s *sFile) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (res *v1.FileUploadRes, err error) {
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

	// 生成唯一文件名
	saveName := fmt.Sprintf("%s%s", guid.S(), ext)

	dateDir := time.Now().Format("20060102")
	saveDir := filepath.Join(uploadDir, dateDir)
	if err = ensureDir(saveDir); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "创建目录失败")
	}
	savePath := filepath.Join(saveDir, saveName)

	// 写入文件
	dst, err := os.Create(savePath)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "保存文件失败")
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "写入文件失败")
	}

	fileSize := header.Size
	url := fmt.Sprintf("/upload/%s/%s", dateDir, saveName)

	userId := contextx.UserId(ctx)
	id, err := dao.SysFile.Ctx(ctx).Data(g.Map{
		"name":          saveName,
		"original_name": originalName,
		"path":          savePath,
		"url":           url,
		"size":          fileSize,
		"mime_type":     mimeType,
		"extension":     ext,
		"user_id":       userId,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "保存文件记录失败")
	}

	return &v1.FileUploadRes{
		Id:           uint64(id),
		Name:         saveName,
		OriginalName: originalName,
		Url:          url,
		Size:         uint64(fileSize),
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
	// 尝试删除物理文件（非阻塞）
	go func() {
		_ = os.Remove(f.Path)
	}()
	return &v1.FileDeleteRes{}, nil
}
