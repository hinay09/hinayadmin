/**
 * 自由审批流 API。
 */
import { useRequest } from '~/composables/useRequest'

/** 流程节点 (flow_conf 树) */
export interface FlowNode {
  id: string
  type: 'start' | 'approver' | 'cc' | 'condition'
  name?: string
  child?: FlowNode | null
  /** approver/cc: user/role/selfSelect/superior/initiator */
  approverType?: string
  approverIds?: number[]
  /** approver: any=或签, all=会签 */
  signType?: string
  /** approver: 办理期限(小时), 0=不限; 逾期由定时任务按超时策略处理 */
  timeoutHours?: number
  /** approver: 超时策略 remind=提醒(默认)/transfer=自动转办/approve=自动通过 */
  timeoutAction?: string
  /** approver: 自动转办目标用户ID (timeoutAction=transfer 必填) */
  timeoutTransfer?: number
  /** condition: 分支列表 */
  branches?: FlowBranch[]
}

/** 条件分支: conditions 外层 OR、内层 AND */
export interface FlowBranch {
  id: string
  name: string
  isDefault: boolean
  conditions: FlowCond[][]
  child?: FlowNode | null
}

/** 单个条件 */
export interface FlowCond {
  field: string
  op: 'eq' | 'ne' | 'gt' | 'lt' | 'ge' | 'le' | 'in'
  value: any
}

/** 表单字段定义 (form_conf) */
export interface FlowFormField {
  key: string
  label: string
  /** input/number/textarea/select/date/image/file */
  type: 'input' | 'number' | 'textarea' | 'select' | 'date' | 'image' | 'file'
  options?: string[]
  required?: boolean
}

/** 附件/图片字段的值项 (存储为 JSON 数组) */
export interface FlowFileItem {
  name: string
  url: string
}

/** 归一化图片/附件字段值为 FlowFileItem[] (兼容字符串 URL / 对象数组 / JSON 字符串) */
export function parseFileItems(v: any): FlowFileItem[] {
  if (!v) return []
  if (Array.isArray(v)) {
    return v
      .map((x: any) => (typeof x === 'string' ? { name: x, url: x } : { name: x?.name || x?.url || '', url: x?.url || '' }))
      .filter((x: FlowFileItem) => x.url)
  }
  if (typeof v === 'string') {
    if (/^(https?:)?\/\//.test(v) || v.startsWith('/')) return [{ name: v, url: v }]
    try { return parseFileItems(JSON.parse(v)) } catch { return [] }
  }
  return []
}

/** 表单必填校验的空值判定: 空串/空数组/空 map 均视为未填写 (与后端 isEmptyValue 同口径,
 *  图片/附件字段值为数组, 只判 `v === ''` 会把空数组当成已填写) */
export function isEmptyFormValue(v: any): boolean {
  if (v === undefined || v === null) return true
  if (typeof v === 'string') return v.trim() === ''
  if (Array.isArray(v)) return v.length === 0
  if (typeof v === 'object') return Object.keys(v).length === 0
  return false
}

export interface FlowDefinitionItem {
  id: number
  flowKey: string
  name: string
  formConf: string
  flowConf: string
  version: number
  status: number
  remark: string
  createdAt: string
  updatedAt: string
}

export interface FlowUserOption { id: number, nickname: string, username: string }
export interface FlowRoleOption { id: number, name: string, code: string }
export interface FlowPostOption { id: number, code: string, name: string, kind: number }

export interface FlowInstanceItem {
  id: number
  definitionId: number
  flowKey: string
  flowName: string
  title: string
  status: number
  startUserId: number
  startUserName: string
  currentNodes: string
  /** 当前用户相关任务: ID, 0=无 */
  taskId: number
  /** 当前用户相关任务状态: 1=待办/未读, 2=已同意(抄送=已阅), 3=已驳回, 6=已失效, 0=无 */
  taskStatus: number
  /** 任务到达时间 (停留时长计算起点) */
  taskReceiveTime: string | null
  createdAt: string
  finishedAt: string | null
}

export interface FlowTaskItem {
  id: number
  instanceId: number
  nodeId: string
  nodeName: string
  nodeType: number
  signType: number
  assigneeId: number
  assigneeName: string
  /** 委派来源任务ID: >0=本人被委派的代办任务 (提交意见后回到原审批人终审) */
  delegateFromId: number
  status: number
  comment: string
  receiveTime: string
  /** 办理期限 (节点超时配置物化, null=不限) */
  dueTime: string | null
  actedAt: string | null
}

export interface FlowRecordItem {
  id: number
  nodeName: string
  action: string
  operatorId: number
  operatorName: string
  comment: string
  createdAt: string
}

/** 可驳回到的目标节点 (历史已通过的审批节点) */
export interface FlowRejectTarget {
  nodeId: string
  nodeName: string
}

export interface FlowInstanceDetail {
  instance: FlowInstanceItem
  formConf: string
  formData: string
  flowConf: string
  tasks: FlowTaskItem[]
  records: FlowRecordItem[]
  myPendingTaskId: number
  myCcTaskId: number
  rejectTargets: FlowRejectTarget[]
  canCancel: boolean
  /** 发起人可撤回 (运行中且尚无任何审批人同意) */
  canWithdraw: boolean
  canResubmit: boolean
  prevSelfSelects: Record<string, number[]>
}

export function useFlowApi() {
  const r = useRequest()
  return {
    // 流程定义
    defList: (params: { keyword?: string, status?: number, version?: number, page?: number, pageSize?: number }) =>
      r.get<{ list: FlowDefinitionItem[], total: number }>('/flow/definitions', params),
    defUsable: () =>
      r.get<{ list: FlowDefinitionItem[] }>('/flow/definitions/usable'),
    defDetail: (id: number) => r.get<FlowDefinitionItem>(`/flow/definitions/${id}`),
    defCreate: (data: { flowKey?: string, name: string, formConf?: string, flowConf?: string, remark?: string }) =>
      r.post<{ id: number }>('/flow/definitions', data),
    defUpdate: (id: number, data: { flowKey?: string, name: string, formConf: string, flowConf: string, remark?: string }) =>
      r.put(`/flow/definitions/${id}`, data),
    defRemove: (id: number) => r.del(`/flow/definitions/${id}`),
    defPublish: (id: number) => r.post<{ id: number, version: number }>(`/flow/definitions/${id}/publish`),
    defDisable: (id: number) => r.post(`/flow/definitions/${id}/disable`),
    // 设计器选项
    options: () => r.get<{ users: FlowUserOption[], roles: FlowRoleOption[], posts: FlowPostOption[] }>('/flow/designer/options'),
    // 实例
    instStart: (data: { definitionId: number, title: string, formData?: Record<string, any>, selfSelects?: Record<string, number[]> }) =>
      r.post<{ id: number }>('/flow/instances', data),
    instList: (params: { scope: 'todo' | 'done' | 'mine' | 'ccme' | 'all', keyword?: string, status?: number, flowKey?: string, taskStatus?: number, page?: number, pageSize?: number }) =>
      r.get<{ list: FlowInstanceItem[], total: number }>('/flow/instances', params),
    instDetail: (id: number) => r.get<FlowInstanceDetail>(`/flow/instances/${id}`),
    instCancel: (id: number) => r.post(`/flow/instances/${id}/cancel`),
    instWithdraw: (id: number) => r.post(`/flow/instances/${id}/withdraw`),
    instResubmit: (id: number, data: { formData?: Record<string, any>, selfSelects?: Record<string, number[]> }) =>
      r.post(`/flow/instances/${id}/resubmit`, data),
    instTerminate: (id: number, comment: string) => r.post(`/flow/instances/${id}/terminate`, { comment }),
    instUrge: (id: number, comment: string) => r.post(`/flow/instances/${id}/urge`, { comment }),
    // 任务
    taskApprove: (id: number, comment: string, opts?: { silent?: boolean }) =>
      r.post(`/flow/tasks/${id}/approve`, { comment }, opts),
    taskReject: (id: number, comment: string, targetNodeId?: string) =>
      r.post(`/flow/tasks/${id}/reject`, { comment, targetNodeId: targetNodeId || undefined }),
    taskTransfer: (id: number, targetUserId: number, comment: string) =>
      r.post(`/flow/tasks/${id}/transfer`, { targetUserId, comment }),
    taskDelegate: (id: number, targetUserId: number, comment: string) =>
      r.post(`/flow/tasks/${id}/delegate`, { targetUserId, comment }),
    taskDelegateResolve: (id: number, comment: string) =>
      r.post(`/flow/tasks/${id}/delegateResolve`, { comment }),
    taskAppend: (id: number, userIds: number[], comment: string) =>
      r.post(`/flow/tasks/${id}/append`, { userIds, comment }),
    taskReduce: (id: number, userIds: number[], comment: string) =>
      r.post(`/flow/tasks/${id}/reduce`, { userIds, comment }),
    taskRead: (id: number, opts?: { silent?: boolean }) => r.put(`/flow/tasks/${id}/read`, undefined, opts),
    taskCount: () => r.get<{ todo: number, cc: number }>('/flow/tasks/count'),
  }
}

/** 实例状态文本/颜色 */
export const flowInstStatusMap: Record<number, { text: string, tag: string }> = {
  1: { text: '运行中', tag: 'primary' },
  2: { text: '已通过', tag: 'success' },
  3: { text: '已驳回(旧)', tag: 'info' },
  4: { text: '已撤销', tag: 'info' },
  5: { text: '已终止', tag: 'warning' },
  6: { text: '被驳回', tag: 'danger' },
  7: { text: '已撤回', tag: 'warning' },
}

/** 任务状态文本/颜色 */
export const flowTaskStatusMap: Record<number, { text: string, tag: string }> = {
  1: { text: '待办', tag: 'primary' },
  2: { text: '已同意', tag: 'success' },
  3: { text: '已驳回', tag: 'danger' },
  4: { text: '已转出', tag: 'info' },
  5: { text: '已作废', tag: 'info' },
  6: { text: '已失效', tag: 'info' },
  7: { text: '已委派', tag: 'warning' },
  8: { text: '委办完成', tag: 'success' },
}

/** 流转动作文本 */
export const flowActionMap: Record<string, string> = {
  submit: '提交申请',
  resubmit: '修改后重新提交',
  approve: '同意',
  reject: '驳回',
  back: '退回到',
  cancel: '撤销',
  withdraw: '撤回',
  cc: '抄送',
  finish: '流程通过',
  transfer: '转办',
  delegate: '委派',
  delegateResolve: '委派处理',
  terminate: '终止',
  urge: '催办',
  append: '加签',
  reduce: '减签',
  timeoutRemind: '超时提醒',
  timeoutTransfer: '超时转办',
  timeoutApprove: '超时自动通过',
}

/** 收集节点树中所有"发起人自选"审批节点 (发起/重提弹窗需要动态选人) */
export function collectSelfSelectNodes(root: FlowNode | null | undefined): FlowNode[] {
  const out: FlowNode[] = []
  const walk = (n: FlowNode | null | undefined) => {
    let cur: FlowNode | null | undefined = n
    while (cur) {
      if (cur.type === 'approver' && cur.approverType === 'selfSelect') out.push(cur)
      if (cur.type === 'condition') {
        for (const b of cur.branches || []) walk(b.child)
      }
      cur = cur.child
    }
  }
  walk(root?.child ?? null)
  return out
}

/** 提交审批公共弹窗的发起参数 (人工发起直接用; 业务模式由页面转交自己的 API) */
export interface FlowSubmitPayload {
  definitionId: number
  title: string
  formData: Record<string, any>
  selfSelects: Record<string, number[]>
}

/** 节点运行状态 (流程图着色用) */
export interface FlowNodeState {
  state: 'done' | 'current' | 'rejected' | 'voided'
  users: string[]
  time: string
}

/** 由任务列表聚合各节点运行状态 (流程图/进度展示的公共依据)。
 *  多轮退回/重提/驳回后, 节点会存在多轮任务 —— 状态只看"最新一轮"
 *  (同轮任务 receive_time 相同), 旧轮次的驳回/作废/失效不覆盖最新轮结果:
 *  最新轮任一驳回→rejected; 有待办→current; 有通过→done; 否则 voided */
export function flowStatesFromTasks(tasks: FlowTaskItem[] = []): Record<string, FlowNodeState> {
  // nodeId -> receiveTime -> 该轮任务
  const rounds = new Map<string, Map<string, FlowTaskItem[]>>()
  for (const t of tasks) {
    let byTime = rounds.get(t.nodeId)
    if (!byTime) { byTime = new Map(); rounds.set(t.nodeId, byTime) }
    const key = t.receiveTime || ''
    const arr = byTime.get(key) || []
    arr.push(t)
    byTime.set(key, arr)
  }
  const out: Record<string, FlowNodeState> = {}
  for (const [nodeId, byTime] of rounds) {
    const latestKey = [...byTime.keys()].sort().pop() || ''
    const ts = byTime.get(latestKey) || []
    let state: FlowNodeState['state']
    if (ts.some(t => t.status === 3)) state = 'rejected'
    else if (ts.some(t => t.status === 1)) state = 'current'
    else if (ts.some(t => t.status === 2)) state = 'done'
    else state = 'voided'
    const acted = ts.filter(t => t.actedAt).map(t => t.actedAt as string).sort()
    out[nodeId] = {
      state,
      users: ts.map(t => t.assigneeName),
      time: (acted[acted.length - 1] || ts[0]?.receiveTime || '').slice(0, 16),
    }
  }
  return out
}

/** 实例状态 → 流程图终点样式 */
export function flowEndState(status: number): { state: 'done' | 'rejected' | 'voided' | 'warn' | 'wait', text: string } {
  if (status === 2) return { state: 'done', text: '流程通过' }
  if (status === 6) return { state: 'rejected', text: '退回发起人' }
  if (status === 7) return { state: 'voided', text: '已撤回待修改' }
  if (status === 4) return { state: 'voided', text: '已撤销' }
  if (status === 5) return { state: 'warn', text: '已终止' }
  return { state: 'wait', text: '结束' }
}
