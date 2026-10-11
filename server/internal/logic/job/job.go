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

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/tenantx"
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

// logDao 日志表 → dao 映射 (数据库规约: 禁止裸 g.DB().Model(硬编码表名),
// docs/multi-tenant.md §3.3/§6-job; 日志清理语义 = 物理删除旧数据行)。
func logDao(table string) *gdb.Model {
	switch table {
	case "sys_login_log":
		return dao.SysLoginLog.Ctx(context.Background()).Unscoped()
	case "sys_audit_log":
		return dao.SysAuditLog.Ctx(context.Background()).Unscoped()
	case "sys_job_log":
		return dao.SysJobLog.Ctx(context.Background()).Unscoped()
	}
	return nil
}

// cleanLogBefore 删除 N 天前的日志行, 返回执行摘要。
// 多租户语义 (§6-job/§12): 任务在租户上下文只清理本租户日志 (注入器限界);
// 平台级任务 (tenant_id=1) 逐租户巡检, 保证全部租户的日志都被清理。
func cleanLogBefore(ctx context.Context, table, params string, defaultDays int) (string, error) {
	days := daysParam(params, defaultDays)
	before := gtime.Now().AddDate(0, 0, -days).Format("Y-m-d H:i:s")

	total := int64(0)
	sweepErr := SweepTenants(ctx, func(ctx context.Context, tid uint64) error {
		m := logDao(table)
		if m == nil {
			return gerror.Newf("未知日志表 %s", table)
		}
		m = m.Ctx(ctx)
		res, err := m.Where("created_at < ?", before).Delete()
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		total += n
		return nil
	})
	if sweepErr != nil {
		return "", sweepErr
	}
	g.Log().Infof(ctx, "清理 %s %d 天前日志完成, 删除 %d 行", table, days, total)
	return fmt.Sprintf("清理 %s %d 天前日志, 删除 %d 行", table, days, total), nil
}

// SweepTenants 平台级任务逐租户巡检 (导出供 flow 超时扫描等平台级处理器复用): 多租户开启时遍历全部启用且未到期租户
// (含平台租户) 逐个构造注入 ctx 执行 fn; 关闭时以平台语境执行一次。
// fn 内的 A 类表操作天然限界在当前巡检租户 (§6-job 遍历多租户的平台级任务)。
func SweepTenants(ctx context.Context, fn func(ctx context.Context, tid uint64) error) error {
	if !tenantx.Enabled(ctx) {
		return fn(ctx, consts.TenantPlatformId)
	}
	tenants, err := service.Tenant().ActiveTenantLites(ctx)
	if err != nil {
		return err
	}
	for _, t := range tenants {
		if serr := fn(contextx.WithTenant(ctx, &model.TenantInfo{Id: t.Id, Code: t.Code, Name: t.Name, Inject: true}), t.Id); serr != nil {
			g.Log().Warningf(ctx, "巡检租户 %s(%d) 失败: %v", t.Name, t.Id, serr)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 调度生命周期
// ---------------------------------------------------------------------------

// Start 应用启动时加载启用任务加入调度 (cmd 调用, 幂等)。
// sys_job 为 A 类隔离表: 以平台视角 (不注入) 加载**全租户**任务 —— 租户管理员
// 创建的任务同样需要调度, 执行时再按行内 tenant_id 构造租户上下文 (§6-job)。
func (s *sJob) Start(ctx context.Context) error {
	var rows []*model.SysJob
	err := dao.SysJob.Ctx(contextx.WithoutTenant(ctx)).
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
		s.runJob(row.Id, row.TenantId, row.Name, row.Handler, row.Params)
	}, name); err != nil {
		return err
	}
	return nil
}

// unschedule 从调度器移除任务条目。
func (s *sJob) unschedule(id uint64) {
	s.cron.Remove(jobName(id))
}

// runJob 执行一次任务并落执行日志。
// 多租户 (§6-job): 按任务行 tenant_id 构造带注入的租户上下文 —— 处理器内的
// A 类表读写自动限界在该租户; 禁用/到期租户的任务跳过并写明原因 (评审四)。
func (s *sJob) runJob(id, tenantId uint64, name, handler, params string) {
	ctx, skip := buildJobCtx(context.Background(), tenantId)
	start := time.Now()
	var out string
	var err error
	if skip != "" {
		err = gerror.New(skip)
	} else {
		out, err = s.invoke(ctx, handler, params)
	}
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
	// 执行日志随任务归属租户落库 (平台视角写入, 显式 tenant_id)
	if _, lerr := dao.SysJobLog.Ctx(contextx.WithoutTenant(ctx)).Data(g.Map{
		"tenant_id":   tenantId,
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

// buildJobCtx 定时任务租户上下文构造: 校验租户状态/到期 (docs/multi-tenant.md §6/§12),
// 失效租户返回 skip 原因 (调用方记 job_log 后跳过执行)。
func buildJobCtx(ctx context.Context, tenantId uint64) (context.Context, string) {
	if tenantId == 0 {
		tenantId = consts.TenantPlatformId
	}
	if !tenantx.Enabled(ctx) {
		return contextx.WithTenant(ctx, &model.TenantInfo{
			Id: consts.TenantPlatformId, Code: consts.TenantPlatformCode, Name: "平台租户", Inject: false,
		}), ""
	}
	t, err := service.Tenant().LoadLiteById(ctx, tenantId)
	if err != nil {
		return contextx.WithoutTenant(ctx), "租户不存在或已删除, 跳过执行"
	}
	if t.Status != consts.StatusEnabled {
		return contextx.WithoutTenant(ctx), "租户已禁用, 跳过执行"
	}
	if t.ExpireAt != "" {
		if ft, perr := gtime.StrToTime(t.ExpireAt); perr == nil && ft.Before(gtime.Now()) {
			return contextx.WithoutTenant(ctx), "租户已到期, 跳过执行"
		}
	}
	return contextx.WithTenant(ctx, &model.TenantInfo{Id: t.Id, Code: t.Code, Name: t.Name, Inject: true}), ""
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
	tnMap := map[uint64]string{}
	if tenantx.Enabled(ctx) {
		tnMap = tenantNames(ctx)
	}
	for _, r := range rows {
		items = append(items, &model.JobItem{
			Id: r.Id, TenantId: r.TenantId, TenantName: tnMap[r.TenantId],
			Name: r.Name, Handler: r.Handler, CronExpr: r.CronExpr,
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

// handlerExists 判断任务处理器是否已注册 (读锁保护, 新增/校验任务时用)。
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
	go s.runJob(row.Id, row.TenantId, row.Name, row.Handler, row.Params)
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

// tenantNames 租户名映射 (sys_tenant 平台表全量未删行; 与 system 包 loadTenantNames 同一规约,
// 独立包内联避免跨 logic 包耦合)。
func tenantNames(ctx context.Context) map[uint64]string {
	out := map[uint64]string{}
	res, err := dao.SysTenant.Ctx(contextx.WithoutTenant(ctx)).
		Where("deleted_at IS NULL").Fields("id, name").All()
	if err != nil {
		return out
	}
	for _, r := range res {
		out[r["id"].Uint64()] = r["name"].String()
	}
	return out
}
