// Package system 系统管理-接口日志业务逻辑。
package system

import (
	"context"
	"strings"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/tenantx"
	"hinay.cn/admin/utility/xerror"
)

type sAccessLog struct{}

func init() {
	service.RegisterAccessLog(NewAccessLog())
}

func NewAccessLog() *sAccessLog {
	return &sAccessLog{}
}

// List 分页列表。
func (s *sAccessLog) List(ctx context.Context, req *v1.AccessLogListReq) (res *v1.AccessLogListRes, err error) {
	q := dao.SysAccessLog.Ctx(ctx)

	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where(
			"username LIKE ? OR path LIKE ? OR request_id LIKE ?",
			kw, kw, kw,
		)
	}
	if req.Username != "" {
		q = q.Where("username", req.Username)
	}
	if req.Method != "" {
		q = q.Where("method", req.Method)
	}
	if req.Slow == 1 {
		q = q.Where("slow", 1)
	}
	switch req.Result {
	case "success":
		q = q.Where("code = 0 AND status_code < 400")
	case "fail":
		q = q.Where("(code != 0 OR status_code >= 400)")
	}
	if req.MinDuration > 0 {
		q = q.Where("duration_ms >= ?", req.MinDuration)
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

	var rows []*model.AccessLogItem
	if err = q.Page(req.Page, req.PageSize).Order("id DESC").Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if tenantx.Enabled(ctx) {
		tnMap := loadTenantNames(ctx)
		for _, r := range rows {
			r.TenantName = tnMap[r.TenantId]
		}
	}

	page := response.Page(rows, int64(total), req.Page, req.PageSize)
	r := v1.AccessLogListRes(page)
	return &r, nil
}
