// Package system 系统管理-登录日志业务逻辑。
package system

import (
	"context"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/tenantx"
	"hinay.cn/admin/utility/xerror"

	"github.com/gogf/gf/v2/frame/g"
)

type sLoginLog struct{}

func init() {
	service.RegisterLoginLog(NewLoginLog())
}

func NewLoginLog() *sLoginLog {
	return &sLoginLog{}
}

// Record 异步写一条登录日志 (登录/登出等链路调用, 失败仅记日志不阻断主流程)。
// 保留请求 ctx 的租户值 (WithoutCancel 去取消传播): sys_login_log 为 A 类隔离表,
// 无租户上下文的 Background 写入会被注入器 fail-close 拒绝 (docs/multi-tenant.md §4.2)。
func (s *sLoginLog) Record(ctx context.Context, entry model.LoginLogEntry) {
	if entry.TenantId == 0 {
		entry.TenantId = consts.TenantPlatformId
	}
	go func() {
		bg := context.WithoutCancel(ctx)
		if _, err := dao.SysLoginLog.Ctx(bg).Data(g.Map{
			"tenant_id":  entry.TenantId,
			"user_id":    entry.UserId,
			"username":   entry.Username,
			"status":     entry.Status,
			"message":    entry.Message,
			"ip":         entry.Ip,
			"user_agent": entry.UserAgent,
		}).Insert(); err != nil {
			g.Log().Errorf(bg, "写入登录日志失败: %v", err)
		}
	}()
}

// List 分页列表。
func (s *sLoginLog) List(ctx context.Context, req *v1.LoginLogListReq) (res *v1.LoginLogListRes, err error) {
	q := dao.SysLoginLog.Ctx(ctx)

	if req.Username != "" {
		q = q.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.Ip != "" {
		q = q.Where("ip LIKE ?", "%"+req.Ip+"%")
	}
	switch req.Status {
	case "success":
		q = q.Where("status", 1)
	case "fail":
		q = q.Where("status", 0)
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

	var rows []*model.LoginLogItem
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
	r := v1.LoginLogListRes(page)
	return &r, nil
}

// Delete 按 ID 删除登录日志。
func (s *sLoginLog) Delete(ctx context.Context, req *v1.LoginLogDeleteReq) (res *v1.LoginLogDeleteRes, err error) {
	if _, err = dao.SysLoginLog.Ctx(ctx).Where("id", req.Id).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "删除登录日志失败")
	}
	return &v1.LoginLogDeleteRes{}, nil
}
