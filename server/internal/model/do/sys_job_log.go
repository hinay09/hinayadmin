// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysJobLog is the golang structure of table sys_job_log for DAO operations like Where/Data.
type SysJobLog struct {
	g.Meta     `orm:"table:sys_job_log, do:true"`
	Id         any         // 日志ID
	JobId      any         // 任务ID
	JobName    any         // 任务名称(冗余, 删除任务后日志仍可读)
	Handler    any         // 处理器名称
	Params     any         // 任务参数(JSON)
	Status     any         // 结果:1=成功,0=失败
	Output     any         // 执行输出/失败原因
	DurationMs any         // 耗时(毫秒)
	CreatedAt  *gtime.Time // 创建时间
}
