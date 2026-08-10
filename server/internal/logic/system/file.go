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
func (s *sFile) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (res *v1.FileUploadRes, err error) {
	originalName := header.Filename
	ext := strings.ToLower(filepath.Ext(originalName))
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

	mimeType := header.Header.Get("Content-Type")
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
