// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish to manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/leave/v1"
)

type (
	ILeave interface {
		// LeaveList 请假申请分页列表 (admin 看全部, 其余仅本人)。
		LeaveList(ctx context.Context, in *v1.LeaveListReq) (res *v1.LeaveListRes, err error)
		// LeaveCreate 新增请假申请 (草稿, 未发起审批)。
		LeaveCreate(ctx context.Context, in *v1.LeaveCreateReq) (res *v1.LeaveCreateRes, err error)
		// LeaveUpdate 修改请假申请 (仅未发起/被退回/已撤销)。
		LeaveUpdate(ctx context.Context, in *v1.LeaveUpdateReq) (res *v1.LeaveUpdateRes, err error)
		// LeaveDelete 删除请假申请 (审批中/退回态需先撤销)。
		LeaveDelete(ctx context.Context, in *v1.LeaveDeleteReq) (res *v1.LeaveDeleteRes, err error)
		// LeaveSubmit 提交审批 (公共弹窗触发): 草稿走 StartForBiz, 退回/撤销后复用实例重提。
		LeaveSubmit(ctx context.Context, in *v1.LeaveSubmitReq) (res *v1.LeaveSubmitRes, err error)
		// LeaveCancel 撤销审批 (发起人; flow_status 由 OnCanceled 回调写回)。
		LeaveCancel(ctx context.Context, in *v1.LeaveCancelReq) (res *v1.LeaveCancelRes, err error)
	}
)

var (
	localLeave ILeave
)

func Leave() ILeave {
	if localLeave == nil {
		panic("implement not found for interface ILeave, forgot register?")
	}
	return localLeave
}

func RegisterLeave(i ILeave) {
	localLeave = i
}
