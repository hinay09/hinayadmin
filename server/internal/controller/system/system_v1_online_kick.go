package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) OnlineKick(ctx context.Context, req *v1.OnlineKickReq) (res *v1.OnlineKickRes, err error) {
	return service.Online().Kick(ctx, req)
}
