/**
 * 请假申请 API —— 业务型流程审批 Demo。
 * 业务数据走本模块; 审批动作(同意/驳回/重提详情)统一在审批中心完成,
 * 这里只有 提交(发起/重提) 与 撤销 两个流程入口。
 */
import { useRequest } from '~/composables/useRequest'

/** 请假类型: 1=事假,2=病假,3=年假,4=调休,5=其他 (与后端/流程表单 options 对齐) */
export const leaveTypeMap: Record<number, string> = { 1: '事假', 2: '病假', 3: '年假', 4: '调休', 5: '其他' }

/** 业务表冗余的审批状态 (引擎回调写入, 与实例 status 是两套编码, 勿混用) */
export const bizFlowStatusMap: Record<number, { text: string, tag: 'primary' | 'success' | 'danger' | 'info' | 'warning' }> = {
  0: { text: '审批中', tag: 'primary' },
  1: { text: '已通过', tag: 'success' },
  2: { text: '被退回', tag: 'danger' },
  3: { text: '已撤销', tag: 'info' },
  4: { text: '已终止', tag: 'warning' },
}

export interface LeaveItem {
  id: number
  leaveType: number
  startDate: string
  endDate: string
  days: number
  reason: string
  flowStatus: number
  flowInstance: number
  createId: number
  createName: string
  createdAt: string
}

export interface LeaveFormData {
  leaveType: number
  startDate: string
  endDate: string
  days: number
  reason: string
}

/** 行是否可编辑/删除/提交: 未发起(草稿) 或 已退回/已撤销 */
export function leaveEditable(row: Pick<LeaveItem, 'flowInstance' | 'flowStatus'>) {
  return row.flowInstance === 0 || row.flowStatus === 2 || row.flowStatus === 3
}

/** 行是否可撤销: 已发起且 审批中/被退回 */
export function leaveCancelable(row: Pick<LeaveItem, 'flowInstance' | 'flowStatus'>) {
  return row.flowInstance > 0 && (row.flowStatus === 0 || row.flowStatus === 2)
}

export function useLeaveApi() {
  const r = useRequest()
  return {
    list: (params: { leaveType?: number, flowStatus?: number, mine?: number, page?: number, pageSize?: number }) =>
      r.get<{ list: LeaveItem[], total: number }>('/leaves', params),
    create: (data: LeaveFormData) => r.post<{ id: number }>('/leaves', data),
    update: (id: number, data: LeaveFormData) => r.put(`/leaves/${id}`, data),
    remove: (id: number) => r.del(`/leaves/${id}`),
    /** 提交审批 (由公共弹窗 FlowSubmitDialog 触发): 草稿发起新实例 / 退回撤销后复用实例重提 */
    submit: (id: number, selfSelects?: Record<string, number[]>) =>
      r.post<{ flowInstance: number }>(`/leaves/${id}/submit`, { selfSelects }),
    /** 撤销审批: flow_status 由引擎 OnCanceled 回调写回 */
    cancel: (id: number) => r.post(`/leaves/${id}/cancel`),
  }
}
