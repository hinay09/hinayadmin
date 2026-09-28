// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysJob is the golang structure for table sys_job.
type SysJob struct {
	Id        uint64      `json:"id"        orm:"id"         description:"任务ID"`                     // 任务ID
	Name      string      `json:"name"      orm:"name"       description:"任务名称"`                     // 任务名称
	Handler   string      `json:"handler"   orm:"handler"    description:"处理器名称"`                    // 处理器名称
	CronExpr  string      `json:"cronExpr"  orm:"cron_expr"  description:"cron表达式(6位: 秒 分 时 日 月 周)"` // cron表达式(6位: 秒 分 时 日 月 周)
	Params    string      `json:"params"    orm:"params"     description:"任务参数(JSON)"`               // 任务参数(JSON)
	Status    int         `json:"status"    orm:"status"     description:"状态:1=启动,0=暂停"`             // 状态:1=启动,0=暂停
	Remark    string      `json:"remark"    orm:"remark"     description:"备注"`                       // 备注
	CreateId  uint64      `json:"createId"   orm:"create_id"   description:"创建人ID"`                  // 创建人ID
	UpdateId  uint64      `json:"updateId"   orm:"update_id"   description:"最后修改人ID"`                // 最后修改人ID
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`                     // 创建时间
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`                     // 更新时间
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"`                 // 删除时间(软删)
}
