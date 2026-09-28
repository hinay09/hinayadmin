// Package job 定时任务调度: 基于 gcron 的任务注册/启停/立即执行与执行日志。
//
// 调度设计:
//   - 每个任务对应一个 gcron 命名条目 (job:<id>), 采用 AddSingleton 防止上一次执行未结束时重复触发;
//   - 启动时 (cmd 调用 Start) 从 sys_job 加载启用任务加入调度;
//   - 网页端的启停/修改/删除实时作用于调度器, 无需重启服务;
//   - 执行结果异步写入 sys_job_log。
//
// 业务模块注册处理器: 在自身 init() 中调用 service.Job().RegisterHandler("名称", 处理函数)。
package job

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sJob struct {
	cron *gcron.Cron

	mu       sync.RWMutex
	handlers map[string]model.JobHandler
}

func init() {
	service.RegisterJob(NewJob())
}

func NewJob() *sJob {
	s := &sJob{
		cron:     gcron.New(),
		handlers: make(map[string]model.JobHandler),
	}
	s.registerBuiltinHandlers()
	return s
}

// jobName gcron 中任务条目的命名: 固定前缀 + 任务ID, 便于按名 Remove/Start/Stop。
func jobName(id uint64) string {
	return fmt.Sprintf("job:%d", id)
}

// RegisterHandler 包级处理器注册入口。
// 业务模块 import 本包后在自身 init() 中调用: Go 语言保证被依赖包先完成初始化,
// 规避经由 service.Job() 注册的初始化顺序陷阱。
// 处理器返回的字符串将作为执行输出写入 sys_job_log.output。
func RegisterHandler(name string, fn model.JobHandler) {
	service.Job().RegisterHandler(name, fn)
}

// ---------------------------------------------------------------------------
// 处理器注册表
// ---------------------------------------------------------------------------

// RegisterHandler 注册任务处理器 (业务模块在 init() 中调用)。
// 处理器返回的字符串将作为执行输出写入 sys_job_log.output。
func (s *sJob) RegisterHandler(name string, fn model.JobHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[name] = fn
}

// Handlers 已注册处理器名称 (排序后返回)。
func (s *sJob) Handlers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.handlers))
	for n := range s.handlers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// invoke 调用处理器, 未注册时返回明确错误。
func (s *sJob) invoke(ctx context.Context, name, params string) (string, error) {
	s.mu.RLock()
	fn, ok := s.handlers[name]
	s.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("处理器 %q 未注册", name)
	}
	return fn(ctx, params)
}

// registerBuiltinHandlers 脚手架内置处理器: 示例 + 日志清理自维护。
func (s *sJob) registerBuiltinHandlers() {
	s.RegisterHandler("demo.echo", func(ctx context.Context, params string) (string, error) {
		// 双通道: 应用日志(服务端排障/采集) + 返回摘要(执行日志表/网页端)
		g.Log().Infof(ctx, "定时任务示例输出: %s", params)
		return "echo: " + params, nil
	})
	// 按天清理登录日志: params 形如 {"days": 90}
	s.RegisterHandler("job.cleanLoginLog", func(ctx context.Context, params string) (string, error) {
		return cleanLogBefore(ctx, "sys_login_log", params, 90)
	})
	// 按天清理操作日志: params 形如 {"days": 180}
	s.RegisterHandler("job.cleanAuditLog", func(ctx context.Context, params string) (string, error) {
		return cleanLogBefore(ctx, "sys_audit_log", params, 180)
	})
	// 按天清理自身执行日志: params 形如 {"days": 180}
	s.RegisterHandler("job.cleanJobLog", func(ctx context.Context, params string) (string, error) {
		return cleanLogBefore(ctx, "sys_job_log", params, 180)
	})
}

// daysParam 解析 {"days": N} 形式的参数, 缺省/非法时回退默认值。
func daysParam(params string, defaultDays int) int {
	if strings.TrimSpace(params) == "" {
		return defaultDays
	}
	var p struct {
		Days int `json:"days"`
	}
	if err := json.Unmarshal([]byte(params), &p); err != nil || p.Days <= 0 {
		return defaultDays
	}
	return p.Days
}

// cleanLogBefore 删除 N 天前的日志行 (通用内置清理), 返回执行摘要。
// 硬编码表名说明: 这里操作的均为本脚手架固定日志表, 且表名不允许来自外部参数,
// 不存在注入面。
func cleanLogBefore(ctx context.Context, table, params string, defaultDays int) (string, error) {
	days := daysParam(params, defaultDays)
	before := gtime.Now().AddDate(0, 0, -days).Format("Y-m-d H:i:s")
	res, err := g.DB().Model(table).Ctx(ctx).Where("created_at < ?", before).Delete()
	if err != nil {
		return "", err
	}
	n, _ := res.RowsAffected()
	g.Log().Infof(ctx, "清理 %s %d 天前日志完成, 删除 %d 行", table, days, n)
	return fmt.Sprintf("清理 %s %d 天前日志, 删除 %d 行", table, days, n), nil
}

// ---------------------------------------------------------------------------
// 调度生命周期
// ---------------------------------------------------------------------------

// Start 应用启动时加载启用任务加入调度 (cmd 调用, 幂等)。
func (s *sJob) Start(ctx context.Context) error {
	var rows []*model.SysJob
	err := dao.SysJob.Ctx(ctx).
		Where("status", consts.StatusEnabled).
		Where("deleted_at IS NULL").
		Scan(&rows)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err = s.schedule(ctx, row); err != nil {
			// 单个任务表达式非法不阻断启动, 记录后继续
			g.Log().Errorf(ctx, "定时任务 %s(%d) 调度失败: %v", row.Name, row.Id, err)
		}
	}
	g.Log().Infof(ctx, "定时任务调度已启动, 共 %d 个启用任务", len(rows))
	return nil
}

// schedule 将任务加入/重新加入调度 (先移除旧条目, 语义为"以最新配置覆盖")。
func (s *sJob) schedule(ctx context.Context, row *model.SysJob) error {
	name := jobName(row.Id)
	s.cron.Remove(name)
	if row.Status != consts.StatusEnabled {
		return nil
	}
	if _, err := s.cron.AddSingleton(ctx, row.CronExpr, func(ctx context.Context) {
		s.runJob(row.Id, row.Name, row.Handler, row.Params)
	}, name); err != nil {
		return err
	}
	return nil
}

// unschedule 从调度器移除任务条目。
func (s *sJob) unschedule(id uint64) {
	s.cron.Remove(jobName(id))
}

// runJob 执行一次任务并异步落执行日志。
func (s *sJob) runJob(id uint64, name, handler, params string) {
	ctx := context.Background()
	start := time.Now()
	out, err := s.invoke(ctx, handler, params)
	duration := time.Since(start).Milliseconds()

	// 成功与失败都记录执行输出: 成功=处理器返回的摘要, 失败=错误原因
	output, status := strings.TrimSpace(out), consts.JobLogStatusSuccess
	if err != nil {
		status = consts.JobLogStatusFail
		if output == "" {
			output = err.Error()
		} else {
			output = output + "; " + err.Error()
		}
		// 失败同步写应用日志 (执行日志表入库失败时仍有服务侧痕迹)
		g.Log().Errorf(ctx, "定时任务[%s#%d]执行失败: %v", name, id, err)
	}
	output = truncate(output, 1000)
	if _, lerr := dao.SysJobLog.Ctx(ctx).Data(g.Map{
		"job_id":      id,
		"job_name":    name,
		"handler":     handler,
		"params":      params,
		"status":      status,
		"output":      output,
		"duration_ms": duration,
	}).Insert(); lerr != nil {
		g.Log().Errorf(ctx, "写入任务执行日志失败: %v", lerr)
	}
}

// truncate 超长错误信息截断, 防止异常输出撑爆日志表。
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// ---------------------------------------------------------------------------
// CRUD 接口
// ---------------------------------------------------------------------------

// List 任务分页列表。
func (s *sJob) List(ctx context.Context, req *v1.JobListReq) (res *v1.JobListRes, err error) {
	q := dao.SysJob.Ctx(ctx)
	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("name LIKE ? OR handler LIKE ?", kw, kw)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*model.SysJob
	if err = q.Page(req.Page, req.PageSize).Order("id ASC").Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	items := make([]*model.JobItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, &model.JobItem{
			Id: r.Id, Name: r.Name, Handler: r.Handler, CronExpr: r.CronExpr,
			Params: r.Params, Status: r.Status, Remark: r.Remark,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	page := response.Page(items, int64(total), req.Page, req.PageSize)
	out := v1.JobListRes(page)
	return &out, nil
}

// validateJobInput 校验创建/更新的公共入参: 表达式合法性 + 处理器已注册 + params JSON 合法。
func (s *sJob) validateJobInput(ctx context.Context, cronExpr, handler, params string) error {
	if err := validateCronPattern(ctx, cronExpr); err != nil {
		return xerror.New(xerror.CodeParamInvalid, "cron 表达式不合法: "+err.Error())
	}
	if !s.handlerExists(handler) {
		return xerror.New(xerror.CodeParamInvalid, "处理器未注册: "+handler)
	}
	if strings.TrimSpace(params) != "" && !json.Valid([]byte(params)) {
		return xerror.New(xerror.CodeParamInvalid, "任务参数必须是合法 JSON")
	}
	return nil
}

// validateCronPattern 校验 gcron 表达式: 借助一次性 Cron 实例 Add 后立即 Remove,
// 闭包为空函数, 即使极端情况下被触发也无副作用。
func validateCronPattern(ctx context.Context, pattern string) error {
	c := gcron.New()
	if _, err := c.Add(ctx, pattern, func(_ context.Context) {}, "validate"); err != nil {
		return err
	}
	c.Remove("validate")
	return nil
}

func (s *sJob) handlerExists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.handlers[name]
	return ok
}

// Create 新增任务 (默认暂停, 由用户在页面一键启动)。
func (s *sJob) Create(ctx context.Context, req *v1.JobCreateReq) (res *v1.JobCreateRes, err error) {
	if err = s.validateJobInput(ctx, req.CronExpr, req.Handler, req.Params); err != nil {
		return nil, err
	}
	if _, err = dao.SysJob.Ctx(ctx).Data(g.Map{
		"name":      req.Name,
		"handler":   req.Handler,
		"cron_expr": req.CronExpr,
		"params":    req.Params,
		"status":    consts.StatusDisabled,
		"remark":    req.Remark,
	}).Insert(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "新增任务失败")
	}
	return &v1.JobCreateRes{}, nil
}

// Update 修改任务配置, 启用中的任务以最新配置重新调度。
func (s *sJob) Update(ctx context.Context, req *v1.JobUpdateReq) (res *v1.JobUpdateRes, err error) {
	if err = s.validateJobInput(ctx, req.CronExpr, req.Handler, req.Params); err != nil {
		return nil, err
	}
	row, err := s.getJob(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if _, err = dao.SysJob.Ctx(ctx).Where("id", req.Id).Data(g.Map{
		"name":      req.Name,
		"handler":   req.Handler,
		"cron_expr": req.CronExpr,
		"params":    req.Params,
		"remark":    req.Remark,
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改任务失败")
	}
	row.Name, row.Handler, row.CronExpr, row.Params, row.Remark = req.Name, req.Handler, req.CronExpr, req.Params, req.Remark
	if rerr := s.schedule(ctx, row); rerr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, rerr, "重新调度失败")
	}
	return &v1.JobUpdateRes{}, nil
}

// Delete 删除任务 (软删) 并移出调度器。
func (s *sJob) Delete(ctx context.Context, req *v1.JobDeleteReq) (res *v1.JobDeleteRes, err error) {
	if _, err = dao.SysJob.Ctx(ctx).Where("id", req.Id).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "删除任务失败")
	}
	s.unschedule(req.Id)
	return &v1.JobDeleteRes{}, nil
}

// ChangeStatus 一键启动/暂停。
func (s *sJob) ChangeStatus(ctx context.Context, req *v1.JobChangeStatusReq) (res *v1.JobChangeStatusRes, err error) {
	row, err := s.getJob(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if _, err = dao.SysJob.Ctx(ctx).Where("id", req.Id).Data(g.Map{"status": req.Status}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改任务状态失败")
	}
	row.Status = req.Status
	if rerr := s.schedule(ctx, row); rerr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, rerr, "调度失败")
	}
	return &v1.JobChangeStatusRes{}, nil
}

// Run 立即执行一次 (异步, 结果见执行日志)。
func (s *sJob) Run(ctx context.Context, req *v1.JobRunReq) (res *v1.JobRunRes, err error) {
	row, err := s.getJob(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	go s.runJob(row.Id, row.Name, row.Handler, row.Params)
	return &v1.JobRunRes{}, nil
}

// LogList 执行日志分页列表。
func (s *sJob) LogList(ctx context.Context, req *v1.JobLogListReq) (res *v1.JobLogListRes, err error) {
	q := dao.SysJobLog.Ctx(ctx)
	if req.JobId > 0 {
		q = q.Where("job_id", req.JobId)
	}
	switch req.Status {
	case "success":
		q = q.Where("status", consts.JobLogStatusSuccess)
	case "fail":
		q = q.Where("status", consts.JobLogStatusFail)
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
	var rows []*model.JobLogItem
	if err = q.Page(req.Page, req.PageSize).Order("id DESC").Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	page := response.Page(rows, int64(total), req.Page, req.PageSize)
	out := v1.JobLogListRes(page)
	return &out, nil
}

// getJob 按 ID 查任务, 不存在时报业务错误。
func (s *sJob) getJob(ctx context.Context, id uint64) (*model.SysJob, error) {
	var row *model.SysJob
	err := dao.SysJob.Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Scan(&row)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if row == nil {
		return nil, xerror.New(xerror.CodeBusinessError, "任务不存在")
	}
	return row, nil
}
