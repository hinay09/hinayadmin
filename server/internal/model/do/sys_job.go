// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysJob is the golang structure of table sys_job for DAO operations like Where/Data.
type SysJob struct {
	g.Meta    `orm:"table:sys_job, do:true"`
	Id        any         // 任务ID
	Name      any         // 任务名称
	Handler   any         // 处理器名称(需已注册)
	CronExpr  any         // cron表达式(6位: 秒 分 时 日 月 周)
	Params    any         // 任务参数(JSON, 可空)
	Status    any         // 状态:1=启动,0=暂停
	Remark    any         // 备注
	CreateId  any         // 创建人ID(ORM自动填充)
	UpdateId  any         // 最后修改人ID(ORM自动填充)
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删)
}
