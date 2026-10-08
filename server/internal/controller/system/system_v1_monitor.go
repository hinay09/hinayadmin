package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) MonitorServer(ctx context.Context, req *v1.MonitorServerReq) (res *v1.MonitorServerRes, err error) {
	return service.Monitor().Server(ctx, req)
}

func (c *ControllerV1) MonitorMysql(ctx context.Context, req *v1.MonitorMysqlReq) (res *v1.MonitorMysqlRes, err error) {
	return service.Monitor().Mysql(ctx, req)
}

func (c *ControllerV1) MonitorRedis(ctx context.Context, req *v1.MonitorRedisReq) (res *v1.MonitorRedisRes, err error) {
	return service.Monitor().Redis(ctx, req)
}
