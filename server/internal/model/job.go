// Package model 定时任务模型。
package model

import (
	"context"

	"github.com/gogf/gf/v2/os/gtime"
)

// JobHandler 定时任务处理器签名。
// params 为任务配置的原始字符串(通常为 JSON), 由处理器自行解析。
type JobHandler func(ctx context.Context, params string) error

// SysJob 数据库实体 (sys_job)。
type SysJob struct {
	Id        uint64      `json:"id"        orm:"id"`
	Name      string      `json:"name"      orm:"name"`
	Handler   string      `json:"handler"   orm:"handler"`
	CronExpr  string      `json:"cronExpr"  orm:"cron_expr"`
	Params    string      `json:"params"    orm:"params"`
	Status    int         `json:"status"    orm:"status"`
	Remark    string      `json:"remark"    orm:"remark"`
	CreateId  uint64      `json:"createId" orm:"create_id"`
	UpdateId  uint64      `json:"updateId" orm:"update_id"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

// JobItem 任务列表/详情 VO。
type JobItem struct {
	Id        uint64      `json:"id"`
	Name      string      `json:"name"`
	Handler   string      `json:"handler"`
	CronExpr  string      `json:"cronExpr"`
	Params    string      `json:"params"`
	Status    int         `json:"status"`
	Remark    string      `json:"remark"`
	CreatedAt *gtime.Time `json:"createdAt"`
	UpdatedAt *gtime.Time `json:"updatedAt"`
}

// JobLogItem 任务执行日志 VO。
type JobLogItem struct {
	Id         uint64      `json:"id"`
	JobId      uint64      `json:"jobId"`
	JobName    string      `json:"jobName"`
	Handler    string      `json:"handler"`
	Params     string      `json:"params"`
	Status     int         `json:"status"`
	Output     string      `json:"output"`
	DurationMs int64       `json:"durationMs"`
	CreatedAt  *gtime.Time `json:"createdAt"`
}
