// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/system/v1"
)

type (
	IMonitor interface {
		// Server 服务器监控指标: CPU/内存/磁盘/主机信息/Go 运行时 (仅超管)。
		// CPU 占用为自上次调用以来的增量值, 前端轮询间隔即采样窗口。
		Server(ctx context.Context, req *v1.MonitorServerReq) (res *v1.MonitorServerRes, err error)
		// Mysql MySQL 运行状态: 服务端连接计数 + 应用侧连接池 (仅超管)。
		Mysql(ctx context.Context, req *v1.MonitorMysqlReq) (res *v1.MonitorMysqlRes, err error)
		// Redis Redis 运行状态: 版本/连接/内存/命中率/键数量 (仅超管)。
		Redis(ctx context.Context, req *v1.MonitorRedisReq) (res *v1.MonitorRedisRes, err error)
	}
)

var (
	localMonitor IMonitor
)

func Monitor() IMonitor {
	if localMonitor == nil {
		panic("implement not found for interface IMonitor, forgot register?")
	}
	return localMonitor
}

func RegisterMonitor(i IMonitor) {
	localMonitor = i
}
