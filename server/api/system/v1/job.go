// Package v1 系统管理-定时任务接口。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/response"
)

// JobListReq 定时任务分页列表。
type JobListReq struct {
	g.Meta   `path:"/system/jobs" tags:"SystemJob" method:"get" summary:"定时任务分页列表"`
	Keyword  string `json:"keyword"  in:"query" dc:"任务名/处理器模糊"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// JobListRes 列表响应。
type JobListRes response.PageResult

// JobCreateReq 新增定时任务。
type JobCreateReq struct {
	g.Meta   `path:"/system/jobs" tags:"SystemJob" method:"post" summary:"新增定时任务"`
	Name     string `v:"required#请输入任务名称" json:"name"     dc:"任务名称"`
	Handler  string `v:"required#请选择处理器" json:"handler"   dc:"处理器名称(需已注册)"`
	CronExpr string `v:"required#请输入cron表达式" json:"cronExpr" dc:"cron表达式(6位: 秒 分 时 日 月 周)"`
	Params   string `json:"params"  dc:"任务参数(JSON, 可空)"`
	Remark   string `json:"remark"  dc:"备注"`
}

// JobCreateRes 新增响应。
type JobCreateRes struct{}

// JobUpdateReq 修改定时任务。
type JobUpdateReq struct {
	g.Meta   `path:"/system/jobs/:id" tags:"SystemJob" method:"put" summary:"修改定时任务"`
	Id       uint64 `v:"min:1" in:"path" dc:"任务ID"`
	Name     string `v:"required#请输入任务名称" json:"name"     dc:"任务名称"`
	Handler  string `v:"required#请选择处理器" json:"handler"   dc:"处理器名称"`
	CronExpr string `v:"required#请输入cron表达式" json:"cronExpr" dc:"cron表达式"`
	Params   string `json:"params"  dc:"任务参数(JSON, 可空)"`
	Remark   string `json:"remark"  dc:"备注"`
}

// JobUpdateRes 修改响应。
type JobUpdateRes struct{}

// JobDeleteReq 删除定时任务。
type JobDeleteReq struct {
	g.Meta `path:"/system/jobs/:id" tags:"SystemJob" method:"delete" summary:"删除定时任务"`
	Id     uint64 `v:"min:1" in:"path" dc:"任务ID"`
}

// JobDeleteRes 删除响应。
type JobDeleteRes struct{}

// JobChangeStatusReq 启动/暂停任务 (一键启停)。
type JobChangeStatusReq struct {
	g.Meta `path:"/system/jobs/:id/status" tags:"SystemJob" method:"put" summary:"启动/暂停任务"`
	Id     uint64 `v:"min:1" in:"path" dc:"任务ID"`
	Status int    `v:"in:0,1#状态取值不合法" json:"status" dc:"目标状态:1=启动,0=暂停"`
}

// JobChangeStatusRes 启停响应。
type JobChangeStatusRes struct{}

// JobRunReq 立即执行一次任务。
type JobRunReq struct {
	g.Meta `path:"/system/jobs/:id/run" tags:"SystemJob" method:"post" summary:"立即执行一次"`
	Id     uint64 `v:"min:1" in:"path" dc:"任务ID"`
}

// JobRunRes 立即执行响应。
type JobRunRes struct{}

// JobLogListReq 任务执行日志分页列表。
type JobLogListReq struct {
	g.Meta   `path:"/system/jobs/logs" tags:"SystemJob" method:"get" summary:"任务执行日志列表"`
	JobId    uint64 `json:"jobId"    in:"query" dc:"任务ID过滤"`
	Status   string `json:"status"   in:"query" dc:"结果过滤: success/fail" v:"in:,success,fail"`
	StartAt  string `json:"startAt"  in:"query" dc:"开始时间"`
	EndAt    string `json:"endAt"    in:"query" dc:"结束时间"`
	Page     int    `json:"page"     in:"query" d:"1"  dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"页大小"`
}

// JobLogListRes 列表响应。
type JobLogListRes response.PageResult

// JobHandlersReq 已注册处理器名称列表 (供新增/编辑任务下拉选择)。
type JobHandlersReq struct {
	g.Meta `path:"/system/jobs/handlers" tags:"SystemJob" method:"get" summary:"已注册处理器列表"`
}

// JobHandlersRes 处理器列表响应。
type JobHandlersRes struct {
	Handlers []string `json:"handlers" dc:"处理器名称列表"`
}
