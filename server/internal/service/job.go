// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

// Package service 定时任务服务接口。
package service

import (
	"context"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/model"
)

type IJob interface {
	// Start 应用启动时加载启用任务加入调度 (幂等)。
	Start(ctx context.Context) error
	// RegisterHandler 注册任务处理器 (业务模块在 init() 中调用)。
	RegisterHandler(name string, fn model.JobHandler)
	// Handlers 已注册处理器名称列表。
	Handlers() []string
	// List 任务分页列表。
	List(ctx context.Context, req *v1.JobListReq) (res *v1.JobListRes, err error)
	// Create 新增任务 (默认暂停)。
	Create(ctx context.Context, req *v1.JobCreateReq) (res *v1.JobCreateRes, err error)
	// Update 修改任务配置。
	Update(ctx context.Context, req *v1.JobUpdateReq) (res *v1.JobUpdateRes, err error)
	// Delete 删除任务。
	Delete(ctx context.Context, req *v1.JobDeleteReq) (res *v1.JobDeleteRes, err error)
	// ChangeStatus 一键启动/暂停。
	ChangeStatus(ctx context.Context, req *v1.JobChangeStatusReq) (res *v1.JobChangeStatusRes, err error)
	// Run 立即执行一次。
	Run(ctx context.Context, req *v1.JobRunReq) (res *v1.JobRunRes, err error)
	// LogList 执行日志分页列表。
	LogList(ctx context.Context, req *v1.JobLogListReq) (res *v1.JobLogListRes, err error)
}

var localJob IJob

func Job() IJob {
	if localJob == nil {
		panic("implement not found for interface IJob, forgot register?")
	}
	return localJob
}

func RegisterJob(i IJob) {
	localJob = i
}
