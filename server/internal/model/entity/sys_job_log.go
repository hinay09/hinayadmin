// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysJobLog is the golang structure for table sys_job_log.
type SysJobLog struct {
	Id         uint64      `json:"id"         orm:"id"          description:"日志ID"`                 // 日志ID
	JobId      uint64      `json:"jobId"      orm:"job_id"      description:"任务ID"`                 // 任务ID
	JobName    string      `json:"jobName"    orm:"job_name"    description:"任务名称(冗余, 删除任务后日志仍可读)"` // 任务名称(冗余, 删除任务后日志仍可读)
	Handler    string      `json:"handler"    orm:"handler"     description:"处理器名称"`                // 处理器名称
	Params     string      `json:"params"     orm:"params"      description:"任务参数(JSON)"`           // 任务参数(JSON)
	Status     int         `json:"status"     orm:"status"      description:"结果:1=成功,0=失败"`         // 结果:1=成功,0=失败
	Output     string      `json:"output"     orm:"output"      description:"执行输出/失败原因"`            // 执行输出/失败原因
	DurationMs int         `json:"durationMs" orm:"duration_ms" description:"耗时(毫秒)"`               // 耗时(毫秒)
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"  description:"创建时间"`                 // 创建时间
}
