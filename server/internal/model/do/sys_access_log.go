// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysAccessLog is the golang structure of table sys_access_log for DAO operations like Where/Data.
type SysAccessLog struct {
	g.Meta     `orm:"table:sys_access_log, do:true"`
	Id         any         // ID
	TenantId   any         // 租户ID(记录归属, 平台可查全量)
	UserId     any         // 用户ID(匿名请求为0)
	Username   any         // 用户名(匿名请求为空)
	Method     any         // HTTP方法
	Path       any         // 请求路径
	Query      any         // URL查询串(GET筛选条件)
	StatusCode any         // HTTP状态码
	Code       any         // 业务码(0=成功; 二进制/超大响应不解析记0)
	DurationMs any         // 耗时(毫秒)
	Slow       any         // 慢接口标记:1=耗时超过阈值(accessLog.slowMs)
	RequestId  any         // 请求ID(链路追踪)
	Ip         any         // IP地址
	UserAgent  any         // User-Agent
	CreatedAt  *gtime.Time // 创建时间
}
