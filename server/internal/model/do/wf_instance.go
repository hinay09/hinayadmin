// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// WfInstance is the golang structure of table wf_instance for DAO operations like Where/Data.
type WfInstance struct {
	g.Meta         `orm:"table:wf_instance, do:true"`
	Id             any         // 主键ID
	DefinitionId   any         // 定义版本行ID
	FlowKey        any         // 流程标识(冗余)
	FlowName       any         // 流程名称(冗余)
	BizId          any         // 业务关联ID(0=审批中心直接发起)
	Title          any         // 申请标题
	FormData       any         // 提交的表单数据 JSON
	FormConf       any         // 表单定义快照(发起时从定义复制)
	FlowConf       any         // 节点树快照(发起时从定义复制, 驳回重提沿用)
	CurrentNodeIds any         // 当前活跃节点ID(逗号分隔)
	Status         any         // 状态:1=运行中,2=已通过,3=已驳回,4=已撤销,5=已终止
	StartUserId    any         // 发起人ID
	StartUserName  any         // 发起人昵称(冗余)
	FinishedAt     *gtime.Time // 结束时间
	CreateId       any         // 创建人ID(ORM自动填充)
	UpdateId       any         // 最后修改人ID(ORM自动填充)
	CreatedAt      *gtime.Time // 创建时间
	UpdatedAt      *gtime.Time // 更新时间
	DeletedAt      *gtime.Time // 删除时间(软删)
}
