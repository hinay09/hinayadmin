// Package system 系统管理-操作日志业务逻辑。
package system

import (
	"context"
	"strings"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sAuditLog struct{}

func init() {
	service.RegisterAuditLog(NewAuditLog())
}

func NewAuditLog() *sAuditLog {
	return &sAuditLog{}
}

// List 分页列表。
func (s *sAuditLog) List(ctx context.Context, req *v1.AuditLogListReq) (res *v1.AuditLogListRes, err error) {
	q := dao.SysAuditLog.Ctx(ctx)

	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("username LIKE ? OR action LIKE ? OR resource LIKE ?", kw, kw, kw)
	}
	if req.Action != "" {
		q = q.Where("action", req.Action)
	}
	if req.StartAt != "" {
		q = q.Where("created_at >= ?", req.StartAt)
	}
	if req.EndAt != "" {
		q = q.Where("created_at <= ?", req.EndAt)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	var rows []*model.AuditLogItem
	if err = q.Page(req.Page, req.PageSize).Order("id DESC").Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	page := response.Page(rows, int64(total), req.Page, req.PageSize)
	r := v1.AuditLogListRes(page)
	return &r, nil
}
