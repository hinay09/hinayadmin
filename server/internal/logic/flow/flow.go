// Package flow 自由审批流业务逻辑。
package flow

import (
	"hinay.cn/admin/internal/service"
)

type sFlow struct{}

func init() {
	service.RegisterFlow(NewFlow())
}

// NewFlow 创建并注册审批流服务实例。
func NewFlow() *sFlow {
	return &sFlow{}
}
