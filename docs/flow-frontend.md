# 审批流前端对接指南

> 适用: 业务模块**前端页面**需要 发起审批 / 展示审批状态 / 列表标记"我的待办" / 内嵌审批操作的场景。
> **新业务审批模块开发请直接按 [flow-template.md](flow-template.md) 标准模板编码**
> (业务与流程分离 + 公共弹窗提交的端到端手册); 本文是前端资源与机制的完整参考。
> 后端接入(`StartForBiz`/状态回调/建表冗余)见 [flow-integration.md](flow-integration.md), 配合阅读。

## 一、先分清两种发起方式

| | 人工发起 | 业务发起 (业务模块标准做法) |
|---|---|---|
| 入口 | 「流程定义 → 发起流程」弹窗, 或任何页面调 `api.instStart()` | 业务页面提交**业务自己的 API** |
| 链路 | `POST /flow/instances` (前端直连) | 业务后端落表 → `flow.StartForBiz(flowKey, ..., bizId, ...)` |
| bizId | 恒为 0 | 业务单 ID (回调定位业务记录的绑定键) |
| 表单数据 | 只存在流程实例快照里 | 业务表一份 + 流程快照一份 |
| 适用 | 行政类轻流程 (请假条、用章申请, 无业务表) | 请假/报销/合同等**有业务数据和状态机**的模块 |

**关键约束: HTTP 发起接口 `POST /flow/instances` 没有 `bizId` 参数。**
业务流程不要用 `api.instStart()` 发起——实例会与业务单失去关联, 状态回调也定位不到业务记录。
业务前端只负责: ①把表单提交给业务 API; ②展示业务 API 返回的审批状态。

## 二、前端资源总览

| 资源 | 位置 | 用途 |
|---|---|---|
| `/flow/definitions` | `app/pages/flow/definitions/` | 流程设计(表单+节点)/发布/版本管理/人工发起 |
| `/flow/center` | `app/pages/flow/center/` | 我的审批: 待我审批 / 我已审批 / 我发起的 / 抄送我的 |
| `/flow/instances` | `app/pages/flow/instances/` | **实例管理 (管理员)**: 全部用户审批记录 + 动态加签/减签/中止 |
| `/flow/detail?id=N` | `app/pages/flow/detail/` | 实例详情: 审批操作/退回重提/撤销/流程图/时间线 (菜单不可见, 供跳转) |
| `<FlowSubmitDialog>` | `app/components/flow/FlowSubmitDialog.vue` | **提交审批公共弹窗** (选流程/填表单/选审批人) |
| `<FlowHistoryDialog>` | `app/components/flow/FlowHistoryDialog.vue` | **审批历史公共弹窗** (流转记录+审批任务表格) |
| `<FlowDiagramDialog>` | `app/components/flow/FlowDiagramDialog.vue` | **流程图公共弹窗** (按实例渲染带进度的流程图) |
| `<FlowDiagram>` | `app/components/flow/FlowDiagram.vue` | 只读流程图组件 (弹窗内部也用它, 可直接内嵌页面) |
| `useFlowApi()` | `app/composables/useApi/flow.ts` | 全部流程接口 + 状态映射表 + 状态聚合辅助函数 |
| `<FormRender>` | `app/components/flow/FormRender.vue` | 按 formConf 渲染动态表单 (可编辑/只读两态) |
| `collectSelfSelectNodes()` | 同 useApi/flow.ts | 从节点树收集"发起人自选"节点 (发起时要选人) |

## 三、`useFlowApi()` 接口速查

| 方法 | 说明 | 返回要点 |
|---|---|---|
| `defUsable()` | 可发起的流程 (每个 flowKey 取最新已发布版) | 含 `formConf`/`flowConf`, 供发起弹窗渲染 |
| `instStart({ definitionId, title, formData, selfSelects })` | 人工发起 (**无 bizId, 业务流程勿用**) | `{ id }` 实例ID |
| `instList({ scope, keyword, status, flowKey, page, pageSize })` | 实例列表 (四个人视角 + 管理员 `all`) | 行含 `bizId`/`taskId`/`status`/`currentNodes` |
| `instDetail(id)` | 实例详情 | 见下方"详情驱动字段" |
| `instCancel(id)` | 发起人撤销, 未处理待办全部作废 | - |
| `instResubmit(id, { formData, selfSelects })` | 退回态修改后重提 (流程从头重走) | - |
| `instTerminate(id, comment)` | **管理员**终止流程 (仅运行中, 后端校验 admin) | - |
| `instUrge(id, comment)` | 发起人催办, 通知全部待办审批人 | 同实例 10 分钟限一次 (后端限流) |
| `taskApprove(taskId, comment)` | 同意 | comment 可空 |
| `taskReject(taskId, comment, targetNodeId?)` | 驳回 (节点级: 退回发起人或指定已审批节点) | **comment 后端必填校验, 前端先拦** |
| `taskTransfer(taskId, targetUserId, comment)` | 转办: 待办转给他人 (原任务置已转出) | 目标人收到通知 |
| `taskAppend(taskId, userIds, comment)` | 加签: 当前节点追加必要审批人 (**节点转为会签**) | 候选人需过滤已在节点待办中的; taskId 锚=我的待办, 管理员可传当前节点任一待办 |
| `taskReduce(taskId, userIds, comment)` | 减签: 移除节点待办审批人 (至少保留一人) | 候选=当前节点全部待办审批人; 被移除人收到作废通知 |
| `taskRead(taskId)` | 抄送已阅 | - |
| `taskCount()` | 我的数量 | `{ todo, cc }` 角标用 |
| `options()` | 用户/角色/岗位选项 | 自选审批人下拉用 |

**`instList` 的 scope 语义** (服务端按当前登录用户过滤):

| scope | 内容 | 行内 `taskId` |
|---|---|---|
| `todo` | 待**我**审批的运行中实例 | 我的待办任务ID → 可直接 `taskApprove/taskReject` |
| `done` | 我处理过的实例 | 恒 0 |
| `mine` | 我发起的 (支持 `status` 筛选) | 恒 0 |
| `ccme` | 抄送**我**的 (待阅+已阅) | 我的抄送任务ID → 可 `taskRead` |
| `all` | **全部用户**的实例 (仅管理员, 后端校验; 支持 `status`/`flowKey` 筛选) | 恒 0 |

**`instDetail(id)` 的详情驱动字段** (详情页全部交互由它们驱动, 业务页跳转前不需要预判):

| 字段 | 含义 | 驱动的 UI |
|---|---|---|
| `myPendingTaskId` | 当前用户待办任务ID (0=无) | 「待你审批」操作卡: 同意/驳回/转办 |
| `myCcTaskId` | 当前用户待阅任务ID | 打开详情自动已阅 |
| `canResubmit` | 退回态**或已撤销**且当前用户是发起人 | 重提卡: 改表单+`instResubmit`/撤销 |
| `canCancel` | 运行中且当前用户是发起人 | 撤销/催办按钮: `instCancel`/`instUrge` |
| `prevSelfSelects` | 上轮自选审批人 `{nodeId: [userId]}` | 重提弹窗默认值 |

终止 (`instTerminate`) 不由详情字段驱动: 仅运行中 + 管理员 (`useUserStore().isAdmin`, 后端同样校验)。

## 四、业务页面接入 (标准三件套)

前提: 后端已按 [flow-integration.md](flow-integration.md) 接入, 业务表冗余了
`flow_status`(0=审批中,1=已通过,2=被退回,3=已撤销) 和 `flow_instance`(实例ID) 两列,
业务列表接口把这两列原样返回。

### 1. 发起: 提交按钮只调业务 API

```ts
// 业务列表页: 提交审批 = 调业务自己的接口 (同样封装进 useApi, 如 useLeaveApi().submit),
// 后端负责落表 + StartForBiz, 并写入 flow_status=0 / flow_instance=N
async function submit(row: LeaveRow) {
  await useLeaveApi().submit(row.id)
  ElMessage.success('已提交审批')
  load()
}
```

### 2. 状态列: 业务冗余列 + 映射 tag

```ts
/** 业务表冗余的审批状态 (回调写入, 与实例 status 是两套编码, 勿混用) */
const bizFlowStatus: Record<number, { text: string, tag: string }> = {
  0: { text: '审批中', tag: 'primary' },
  1: { text: '已通过', tag: 'success' },
  2: { text: '被退回', tag: 'danger' },
  3: { text: '已撤销', tag: 'info' },
}
```

```html
<el-table-column label="审批状态" width="100">
  <template #default="{ row }">
    <el-tag v-if="row.flowInstance" :type="bizFlowStatus[row.flowStatus]?.tag || 'info'">
      {{ bizFlowStatus[row.flowStatus]?.text || '未知' }}
    </el-tag>
    <span v-else>-</span>
  </template>
</el-table-column>
```

### 3. 操作列: 「查看审批」跳详情页

审批/驳回/重提/撤销等**全部动作都在 `/flow/detail` 页完成**——详情页按当前用户身份
(`myPendingTaskId`/`canResubmit`/`canCancel`) 自动渲染对应操作区, 业务页不需要也不应该自己实现。

```html
<el-table-column label="操作" width="180">
  <template #default="{ row }">
    <el-button v-if="!row.flowInstance" link type="primary" @click="submit(row)">提交审批</el-button>
    <!-- 被退回: 详情页里发起人可改表单重提, 业务页直接引导过去即可 -->
    <el-button v-if="row.flowInstance" link type="primary"
      @click="navigateTo({ path: '/flow/detail', query: { id: String(row.flowInstance) } })">
      {{ row.flowStatus === 2 ? '去修改重提' : '查看审批' }}
    </el-button>
  </template>
</el-table-column>
```

> 「我的审批」(`/flow/center`) 里同样能看到业务流程的待办, 详情页展示的就是发起时的表单快照。

## 五、业务列表标记「我的待办」

审批人打开业务列表时, 通常希望直接看出"哪些单子等我审批"。两种做法:

### 方式 A: 后端联查 (推荐, 列表大/分页多时)

业务 List 接口为本页每行多算两个字段 `myTodo bool` + `flowInstance`:

```go
// 业务 logic List 里 (示例: 请假):
uid := contextx.UserId(ctx)
// ① 本页业务单对应的实例 (flow_key + biz_id)
var insts []*entity.WfInstance
_ = dao.WfInstance.Ctx(ctx).Where("flow_key", "biz_leave").
    WhereIn("biz_id", pageIds).Where("deleted_at IS NULL").
    Order("id DESC").Scan(&insts)
instByBiz := map[uint64]*entity.WfInstance{}
ids := make([]uint64, 0)
for _, i := range insts { instByBiz[i.BizId] = i; ids = append(ids, i.Id) }
// ② 我在这些实例上的待办 (审批类节点)
var tasks []*entity.WfTask
_ = dao.WfTask.Ctx(ctx).WhereIn("instance_id", ids).
    Where("assignee_id", uid).Where("status", 1).Where("node_type", 1).Scan(&tasks)
todoInst := map[uint64]bool{}
for _, t := range tasks { todoInst[t.InstanceId] = true }
// ③ 组装行: MyTodo = todoInst[instByBiz[bizId].Id]
```

### 方式 B: 前端聚合 (零后端改动)

待办列表本身带 `bizId`, 前端按 `flowKey + bizId` 匹配业务行:

```ts
// app/composables/useFlowTodos.ts
import { ref } from 'vue'
import { useFlowApi, type FlowInstanceItem } from '~/composables/useApi/flow'

/** 拉当前用户在指定流程上的待办, 按 bizId 索引 (业务列表打"待我审批"标记用) */
export function useFlowTodos(flowKey: string) {
  const api = useFlowApi()
  const todos = ref(new Map<number, FlowInstanceItem>())

  async function loadTodos() {
    const next = new Map<number, FlowInstanceItem>()
    const res = await api.instList({ scope: 'todo', pageSize: 100 }) // 个人待办一般不多; 超过 100 条需翻页
    for (const it of res.list || []) {
      if (it.flowKey === flowKey) next.set(it.bizId, it)             // 只认本业务的流程
    }
    todos.value = next
  }
  const isTodo = (bizId: number) => todos.value.has(bizId)
  const todoOf = (bizId: number) => todos.value.get(bizId)           // 含 taskId, 可行内审批

  return { todos, loadTodos, isTodo, todoOf }
}
```

```html
<el-table-column label="审批状态" width="150">
  <template #default="{ row }">
    <el-tag v-if="row.flowInstance" :type="bizFlowStatus[row.flowStatus]?.tag">
      {{ bizFlowStatus[row.flowStatus]?.text }}
    </el-tag>
    <el-tag v-if="isTodo(row.id)" type="danger" effect="dark" style="margin-left:4px">待我审批</el-tag>
  </template>
</el-table-column>
```

审批动线二选一: 跳 `/flow/detail?id=` (推荐, 带意见框/确认/完整时间线), 或见下一节行内快捷操作。

## 六、内嵌审批操作按钮 (可选)

> 一般**不建议**在业务页重复实现审批 UI——审批中心与详情页已完整覆盖(意见输入、驳回必填校验、
> 会签/或签、退回重提)。仅当交互上强依赖行内操作(如运营工作台)再这样做。

行内操作的前提是拿到 `taskId`, 两个来源: `useFlowTodos().todoOf(bizId).taskId` (列表行),
或 `api.instDetail(id).myPendingTaskId` (详情)。

```ts
const api = useFlowApi()
const { loadTodos } = useFlowTodos('biz_leave')

/** 行内同意 */
async function quickApprove(row: LeaveRow) {
  const t = todoOf(row.id)
  if (!t?.taskId) return
  await ElMessageBox.confirm(`同意「${row.title}」?`, '审批确认', { type: 'info' })
  await api.taskApprove(t.taskId, '')          // 同意意见可空
  ElMessage.success('已同意')
  loadTodos(); load()
}

/** 行内驳回: 意见必填 (后端同样强校验, 前端先拦体验更好) */
async function quickReject(row: LeaveRow) {
  const t = todoOf(row.id)
  if (!t?.taskId) return
  const { value } = await ElMessageBox.prompt('驳回意见 (必填)', '驳回', {
    inputType: 'textarea', inputValidator: v => !!v?.trim() || '请填写驳回意见',
  })
  await api.taskReject(t.taskId, value)
  ElMessage.success('已驳回, 已退回发起人')
  loadTodos(); load()
}
```

### 待办角标

```ts
const counts = ref({ todo: 0, cc: 0 })
async function loadCounts() { counts.value = await api.taskCount() }
// 挂顶栏图标 `el-badge :value="counts.todo"`, 或进入页面时刷新; 用法可参照 flow/center 页
```

## 七、业务页三大公共组件 (提交审批 / 审批历史 / 流程图)

业务页的标准三按钮对应三个公共弹窗组件, 显式 import 后即可用:

### 1. 提交审批 `<FlowSubmitDialog>`

```html
<el-button type="primary" @click="submitVisible = true">提交审批</el-button>
<FlowSubmitDialog v-model="submitVisible" flow-key="biz_leave"
  :show-form="false" :show-title="false"
  :handler="payload => useLeaveApi().submit(row.id, payload.selfSelects)"
  @submitted="load" />
```

- `flow-key`: 绑定业务流程, 自动选中最新已发布定义 (不传则显示流程下拉, 即人工发起模式);
- `handler`: 业务提交钩子——组件完成"选审批人/填表单/校验"后把
  `{ definitionId, title, formData, selfSelects }` 交给业务 API, 由后端调 `StartForBiz`
  (返回实例 ID 则会随 `submitted` 事件透传, 便于跳详情); **不传 handler 则直接 `instStart` 人工发起**;
- `:show-form="false"`: 业务表单由页面自己维护时只选审批人; `:show-title="false"` 隐藏标题输入。
- 组件内部会校验: 流程必选、必填字段、每个"发起人自选"节点已选人。

### 2. 审批历史 `<FlowHistoryDialog>`

```html
<el-button link type="primary" @click="historyId = row.flowInstance">审批历史</el-button>
<FlowHistoryDialog v-model="historyVisible" :instance-id="historyId" />
```

弹窗内两个页签: 流转记录表 (时间/动作/操作人/节点/意见) + 审批任务表
(含会签/或签、抄送已阅语义), 顶部有标题与实例状态摘要。

### 3. 流程图 `<FlowDiagramDialog>`

```html
<el-button link type="primary" @click="diagramId = row.flowInstance">流程图</el-button>
<FlowDiagramDialog v-model="diagramVisible" :instance-id="diagramId" />
```

按实例快照渲染钉钉式纵向流程图: 审批/抄送卡片随运行状态着色
(已通过绿/进行中蓝/已驳回红/未经过灰虚线), 条件分支横向铺开且**实际命中的分支高亮**,
终点胶囊显示流程结果。组件 `<FlowDiagram>` 也可直接内嵌页面 (详情页的"流程进度"就是它)。

> 三个弹窗都只需要实例 ID (业务表的 `flow_instance` 冗余列), 无需引入其它依赖。

## 八、权限码 (按钮级)

| 权限码 | 控制点 | 谁需要 |
|---|---|---|
| `flow:definition:list/create/update/delete/publish` | 流程定义页各按钮 | 流程管理员 |
| `flow:instance:list/start/cancel` | 我的审批: 列表/发起/撤销 | 普通用户 (start/cancel 发起人场景) |
| `flow:instance:manage` | 实例管理页可见性 (菜单 9050) | 管理员 (数据侧 scope=all 后端再校验 admin) |
| `flow:task:handle` | 同意/驳回按钮 | 审批人 |

- 前端用 `v-permission="'flow:task:handle'"` 指令控制显隐; `admin` 角色自动放行。
- **审批人只需要 `flow:task:handle` + 菜单可见**, 不需要业务模块的任何权限——审批动作发生在审批中心,
  服务端按任务归属校验, 拿别人的 taskId 操作会报错。
- 菜单/按钮在 `manifest/sql/init.sql` 模块段 9000 号段, 走「系统管理→角色管理」勾选分配。

## 九、常见问题 (前端向)

- **业务发起后去哪看进度**: 业务列表「查看审批」跳 `/flow/detail?id={flow_instance}`;
  审批人在 `/flow/center` 待办页处理。两处看到的是同一实例。
- **被退回后业务数据怎么改**: 详情页的重提只更新**流程快照**里的表单数据——
  业务表**不会**自动同步(回调只有 通过/退回/撤销 三种, 没有重提回调)。
  若业务字段也需修改, 业务页要提供自己的编辑入口, 或接受"流程快照为准"的口径。
- **两套状态编码别混用**: 业务冗余 `flow_status`(0-3, 回调写入) ≠ 实例 `status`(1/2/4/5/6)。
  前者用业务页自己的映射表; 后者用 `flowInstStatusMap`(useApi/flow.ts 导出)。
- **发起接口报"仅已发布流程可发起"**: 流程定义还是草稿/已停用, 找管理员在流程定义页「发布」。
- **自选审批人没传会怎样**: 发起直接报错(实例不创建), 前端在提交前用 `collectSelfSelectNodes`
  校验每个自选节点已选人, 参照 flow/definitions 页 `handleStart` 的写法。
