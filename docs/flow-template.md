# 业务审批标准模板 (biz_leave 范式)

> **这是后期所有业务审批模块的标准开发范式**。凡是「有业务表 + 有自己状态机」的审批需求
> (请假/报销/合同/用章/采购…), 一律照本模板编码; 参照实现即仓库内置的请假 Demo:
> 后端 `server/internal/logic/leave` + 前端 `web_src/app/pages/biz/leave` + 种子 `p016`。
>
> 引擎机制细节见 [flow-integration.md](flow-integration.md) (后端语义),
> [flow-frontend.md](flow-frontend.md) (前端资源); 本文是端到端的**操作手册**。

## 一、模板总览: 业务与流程分离

```
业务页面                          业务后端 (logic/leave 范式)            审批引擎 (logic/flow)
  │ ① 新增/编辑/删除 草稿单据 ─────▶ biz_leave 表 (纯业务 CRUD)
  │
  │ ② 「提交审批」按钮
  │     └─ <FlowSubmitDialog :show-form=false>  (公共提交弹窗)
  │           handler(selfSelects) │
  │                               ▼
  │              POST /leaves/{id}/submit ──▶ StartForBiz(草稿发起新实例)
  │                                          或 InstanceResubmit(退回/撤销后重提)
  │                                           ──▶ 建实例+表单/节点树快照, 推进, 站内通知审批人
  │
  │ ③ 审批/驳回/转办/加签… 全部在「审批中心 → 我的审批」详情页 (业务页不实现审批 UI)
  │
  │ ④ 状态列变化: flow_status 冗余列 ←── BizListener 四回调 (引擎事务提交后尽力调用)
  │        OnApproved→1  OnReturned→2  OnCanceled→3  OnTerminated→4
```

三条铁律 (后面「规范速查」有完整版):

1. **业务表单数据只在业务表里**, 流程实例上的 form_data 只是发起/重提时的快照
   (详情页展示 + 条件分支求值), 由后端从业务列组装, 前端不重复录入;
2. **flow_status 只由回调写回**, 业务代码除了"提交时置 0"外不写审批状态;
3. **发起一律走业务后端 (StartForBiz / InstanceResubmit)**, 前端绝不用
   `POST /flow/instances` (该接口无 bizId, 实例会与业务单失联)。

## 二、第 1 步: 建业务表

单表 + 两个流程冗余列 + 标准审计三件套 (软删/审计列)。

```sql
CREATE TABLE IF NOT EXISTS `biz_leave` (           -- 替换点: 表名
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  -- ↓ 替换点: 业务列 (请假为 类型/起止日期/天数/事由)
  `leave_type` TINYINT      NOT NULL DEFAULT 1,
  `start_date` DATE         NOT NULL,
  `end_date`   DATE         NOT NULL,
  `days`       DECIMAL(5,1) NOT NULL DEFAULT 1.0,
  `reason`     VARCHAR(500) NOT NULL DEFAULT '',
  -- ↓ 流程冗余列 (两个都建, 名称约定不变)
  `flow_status`   TINYINT   NOT NULL DEFAULT 0 COMMENT '审批状态:0=审批中,1=已通过,2=被退回,3=已撤销,4=已终止(引擎回调写入;未发起时无意义)',
  `flow_instance` BIGINT UNSIGNED NOT NULL DEFAULT 0  COMMENT '流程实例ID(0=未发起/草稿)',
  -- ↓ 审计列 (照抄; 必须有 create_id/update_id 才能进 ormfill 白名单)
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_create` (`create_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='请假申请(业务审批Demo)';
```

DDL 写两处: 新增 `manifest/sql/upgrade-modules/pXXX_<mod>_demo.sql` 增量脚本 (存量库),
同内容并入 `modules_init.sql` (新装)。建表进本地库后生成 dao:

```bash
mysql ... < pXXX_xxx.sql        # 表建进开发库
gf gen dao -t biz_xxx           # 生成 dao/entity/do (internal/ 下 4 个文件)
```

**接线两处** (漏了必炸):
- `internal/logic/ormfill/ormfill.go` 的 `fillTables` 加表名 (有审计列的必须加, 否则 create_id 恒 0);
- `internal/logic/logic.go` 加 `_ "hinay.cn/admin/internal/logic/<mod>"`。

## 三、第 2 步: 建流程定义 (flow_key 是绑定键)

「审批中心 → 流程定义 → 新增」: **流程标识 (flow_key) 填业务标识** (如 `biz_leave`,
与代码常量一致, 上线后不要改), 设计节点后**发布**。

form_conf 的设计原则 (它只是**快照展示 + 条件求值**, 不承载业务数据):

| 原则 | 说明 |
|---|---|
| 字段与业务列对齐 | 审批人在详情页看到的快照即业务数据; `buildFormData` 按 key 组装 |
| required 只加条件分支要用的 | required 字段在发起/重提时被引擎强校验 (如 `days`/`reason`); 纯展示字段不 required, 避免业务列未填时发不出去 |
| 条件分支路由字段必须 required | 路由字段缺失时 `condMatch` 一律不命中 → 悄悄走默认分支 |
| 每个条件节点配默认分支 | 未命中无默认分支 → 发起直接报错 |

「用户选路」用 **required 的 select 字段 + eq 条件** 实现 (如 `urgency=紧急` 走加急分支),
不要发明"人工选分支"——选择时点固定在发起/重提, 中途改道走 驳回→重提 (重提会重新求值)。

演示流程也可以直接种 SQL (参照 p016 的 `wf_definition` 行, form_conf/flow_conf 是 JSON 文本)。

## 四、第 3 步: 后端编码 (五件套)

分层照仓库惯例: `api → controller → service → logic → dao`, controller 单行透传 (gf gen ctrl 风格)。
参照 `api/leave/` + `internal/{controller,service,logic}/leave/` 逐文件抄。

### 4.1 api 契约 (`api/<mod>/v1/<mod>.go`)

标准六个接口, 路径用业务复数名 (`/leaves`):

| 接口 | 语义 | 约束 |
|---|---|---|
| `GET /leaves` | 列表 (admin 全部 + `mine=1`, 其余仅本人 `create_id=uid`) | 返回 `flowStatus`/`flowInstance`/`createName` |
| `POST /leaves` | 新增**草稿** (纯业务字段, 不发起) | 业务字段 v 校验 |
| `PUT /leaves/{id}` | 修改 (仅 未发起/被退回/已撤销) | `loadForEdit` 统一校验 |
| `DELETE /leaves/{id}` | 删除 (同上; 审批中先撤销) | 软删 |
| `POST /leaves/{id}/submit` | **提交审批** (公共弹窗触发), body 只有 `selfSelects` | 草稿→StartForBiz; 退回/撤销→InstanceResubmit |
| `POST /leaves/{id}/cancel` | 撤销 (转调引擎 `InstanceCancel`) | flow_status 由 OnCanceled 回调写回 |

`LeaveSubmitReq` 关键字段:

```go
type LeaveSubmitReq struct {
	g.Meta      `path:"/leaves/{id}/submit" tags:"Leave" method:"post" summary:"提交审批"`
	Id          uint64              `json:"id" in:"path" v:"required"`
	SelfSelects map[string][]uint64 `json:"selfSelects" dc:"自选节点审批人 {nodeId: [userId]} (公共弹窗产出)"`
}
```

日期类业务字段用 `v:"required|date-format:Y-m-d#...#..."` 校验 (注意不是 `length:10`)。

### 4.2 logic (`internal/logic/<mod>/<mod>.go`) — 模板核心

包注释 + 常量 + init 注册 + CRUD + Submit/Cancel, 骨架如下 (完整实现见
`internal/logic/leave/leave.go`, 逐段对照):

```go
const flowKeyLeave = "biz_leave" // 与流程定义 flow_key 一致, 上线后不改

// 业务冗余状态编码 (≠ 实例 status, 两套编码勿混用)
const (
	flowStatusApproving = 0 // 审批中 (提交时写入) —— 业务代码唯一允许写的状态
	flowStatusApproved  = 1 // OnApproved 回调
	flowStatusReturned  = 2 // OnReturned 回调
	flowStatusCanceled  = 3 // OnCanceled 回调
	flowStatusTermed    = 4 // OnTerminated 回调
)

func init() {
	service.RegisterLeave(&sLeave{})
	// 状态回调: 引擎事务提交后同步尽力调用, 业务侧必须幂等
	flow.RegisterBizListener(flowKeyLeave, flow.BizListener{
		OnApproved:   func(instanceId, bizId uint64) { writeBackFlowStatus(instanceId, bizId, flowStatusApproved) },
		OnReturned:   func(instanceId, bizId uint64) { writeBackFlowStatus(instanceId, bizId, flowStatusReturned) },
		OnCanceled:   func(instanceId, bizId uint64) { writeBackFlowStatus(instanceId, bizId, flowStatusCanceled) },
		OnTerminated: func(instanceId, bizId uint64) { writeBackFlowStatus(instanceId, bizId, flowStatusTermed) },
	})
}

// 回写幂等: 按 (bizId, instanceId) 定位 —— 旧实例的迟到回调不覆盖新实例状态
func writeBackFlowStatus(instanceId, bizId uint64, status int) {
	_, err := dao.BizLeave.Ctx(context.Background()).
		Where("id", bizId).Where("flow_instance", instanceId).
		Data(g.Map{"flow_status": status}).Update()
	if err != nil { /* g.Log().Warningf ... */ }
}
```

`LeaveSubmit` 双路径 (业务侧全部流程代码就这一个函数):

```go
switch {
case lv.FlowInstance == 0: // 草稿首次发起
	// 标题可由业务字段生成, 如 fmt.Sprintf("请假申请-%s %g天", 申请人, 天数)
	instanceId, fe := flow.StartForBiz(ctx, flowKeyLeave, title,
		buildFormData(lv), lv.Id, uid, in.SelfSelects)
	if fe != nil {
		return nil, fe // 失败单据保持草稿, 修正后可重试 (实例未创建, 无残留)
	}
	// 条件回填: 防旧实例覆盖; 失败不影响流程本身
	dao.BizLeave.Ctx(ctx).Where("id", lv.Id).Where("flow_instance", 0).
		Data(g.Map{"flow_status": flowStatusApproving, "flow_instance": instanceId}).Update()

case lv.FlowStatus == flowStatusReturned || lv.FlowStatus == flowStatusCanceled:
	// 退回/撤销后重提: 复用原实例, 流程从头重走、历史保留, 表单快照整份替换
	service.Flow().InstanceResubmit(ctx, &v1flow.FlowInstanceResubmitReq{
		Id: lv.FlowInstance, FormData: buildFormData(lv), SelfSelects: in.SelfSelects,
	})
	// 条件回写 flow_status=0 (WHERE flow_status=旧值, 防回调并发)

case lv.FlowStatus == flowStatusApproving:
	return nil, xerror.New(xerror.CodeParamInvalid, "该单据已在审批中, 请勿重复提交")
}
```

`LeaveCancel` = 校验本人 + 实例在途 → `service.Flow().InstanceCancel(ctx, {Id})`,
**之后什么都不写** (回调会写 3)。

`buildFormData`: 业务列 → `map[string]any`, key 与流程 form_conf 对齐:

```go
func buildFormData(lv *entity.BizLeave) map[string]any {
	return map[string]any{
		"leave_type": leaveTypeName(lv.LeaveType), // select 字段传 option 文本, 与 form_conf 对齐
		"date_range": lv.StartDate.Format("Y-m-d") + " ~ " + lv.EndDate.Format("Y-m-d"),
		"days":       lv.Days,   // 条件分支求值字段, 数值类型
		"reason":     lv.Reason,
	}
}
```

### 4.3 编辑/删除权限校验 (loadForEdit 模式)

```go
// 仅本人 (admin 放行); 且 未发起 或 已退回/已撤销 —— 审批中与已结束不可改删
if lv.CreateId != uid && !contextx.IsAdmin(ctx) { return nil, xerror.New(xerror.CodeForbidden, "仅申请人可操作") }
if lv.FlowInstance > 0 && lv.FlowStatus != flowStatusReturned && lv.FlowStatus != flowStatusCanceled {
	return nil, xerror.New(xerror.CodeParamInvalid, "审批中或已结束的单据不可修改/删除 (审批中请先撤销)")
}
```

### 4.4 路由接线

`internal/cmd/cmd.go`: import `controller/<mod>` + `sec.Bind(<mod>.NewV1())`。

## 五、第 4 步: 前端编码 (业务页三段式)

参照 `web_src/app/pages/biz/leave/index.vue` + `composables/useApi/leave.ts`。

### 5.1 useApi (`composables/useApi/<mod>.ts`)

必须导出: list/create/update/remove/submit/cancel 六方法、
`bizFlowStatusMap` (0-4 → 文本+tag 色)、`xxxEditable`/`xxxCancelable` 行为判定:

```ts
/** 行是否可编辑/删除/提交: 未发起(草稿) 或 已退回/已撤销 */
export function leaveEditable(row: Pick<LeaveItem, 'flowInstance' | 'flowStatus'>) {
  return row.flowInstance === 0 || row.flowStatus === 2 || row.flowStatus === 3
}
/** 行是否可撤销: 已发起且 审批中/被退回 */
export function leaveCancelable(row: Pick<LeaveItem, 'flowInstance' | 'flowStatus'>) {
  return row.flowInstance > 0 && (row.flowStatus === 0 || row.flowStatus === 2)
}
```

`submit` 带弹窗产出: `submit: (id, selfSelects?) => r.post(`/leaves/${id}/submit`, { selfSelects })`。
新模块记得在 `composables/useApi/index.ts` re-export。

### 5.2 页面骨架 (四块)

1. **业务 CRUD**: 列表 + 业务字段 drawer (新增/编辑, 只服务草稿与退回改单) —— 纯业务代码;
2. **提交审批** (流程侧唯一入口, 公共弹窗):

```html
<el-button v-if="leaveEditable(row)" v-permission="'biz:leave:submit'" link type="primary"
  @click="openSubmit(row)">{{ row.flowInstance ? '重新提交' : '提交审批' }}</el-button>
...
<!-- :show-form=false —— 业务表单不重复录入, 快照由后端从业务表组装;
     流程定义将来加"发起人自选"节点, 弹窗自动渲染选人 UI, 本页零改动 -->
<FlowSubmitDialog v-model="submitVisible" flow-key="biz_leave"
  :show-form="false" :show-title="false"
  :handler="onFlowSubmit" @submitted="onSubmitted" />
```

```ts
async function onFlowSubmit(payload: FlowSubmitPayload): Promise<number | void> {
  if (!submitRow.value) return
  const res = await api.submit(submitRow.value.id, payload.selfSelects)
  return res.flowInstance // 供弹窗透传 submitted 事件 (跳详情)
}
```

3. **状态列** (两套编码勿混用, 业务列用 `bizFlowStatusMap`, 未发起显示"草稿" tag);
4. **审批动作全部跳走**: 「查看审批/去处理」→ `/flow/detail?id={flowInstance}` (详情页按
   `myPendingTaskId`/`canCancel`/`canResubmit` 自动渲染操作区); 「历史」→
   `<FlowHistoryDialog :instance-id>`; 撤销是唯一留在业务页的动作 (confirm + `api.cancel`)。

不要做: 业务页内嵌同意/驳回按钮、自己拼 `instStart`、再写一套审批时间线 ——
公共弹窗 + 详情页 + 历史弹窗已全覆盖。

## 六、第 5 步: SQL 种子 (四个块)

照 p016 抄, 全部 `INSERT IGNORE` 幂等:

1. **菜单** (新号段, 参照 9200): 目录 + 页面 (`component` 填 `biz/<mod>/index`) +
   按钮权限 `biz:<mod>:create/update/delete/submit/cancel`;
2. **sys_api**: 六接口行 (路径/方法与 api 契约一致), 用工具生成后贴进脚本:
   `make gen-api-sql PKG=<mod>`; 提交前 `make gen-api-sql PKG=<mod> CHECK=1` 校验;
3. **wf_definition** (可选, 演示流程直接种): flow_key=业务标识, form_conf/flow_conf JSON;
4. **casbin_rule**: 给演示角色授权 `menu:<id>` + `/api/v1/...` p 行 ——
   **直改 casbin_rule 必须重启服务 (或重新保存任一角色) 才生效**。

## 七、状态机与回调对照表 (业务侧视角)

| 用户动作 | 业务接口 | 引擎动作 | 实例状态 | 回调 | flow_status |
|---|---|---|---|---|---|
| 新增/编辑/删除草稿 | POST/PUT/DELETE | — (不碰流程) | — | — | — (仍 0, flow_instance=0) |
| 提交审批 (草稿) | POST {id}/submit | StartForBiz | 1 运行中 | — (运行中不触发) | 业务侧置 0 |
| 提交审批 (退回/撤销后) | 同上 | InstanceResubmit | 1 运行中 | — | 业务侧置 0 |
| 撤销 | POST {id}/cancel | InstanceCancel | 4 已撤销 | OnCanceled | **回调写 3** |
| 全部节点通过 | (审批中心) | 链尾通过 | 2 已通过 | OnApproved | **回调写 1** |
| 驳回退回发起人 | (审批中心) | TaskReject | 6 已退回 | OnReturned | **回调写 2** |
| 驳回到指定节点 | (审批中心) | TaskReject | 1 运行中 | — (无回调, 实例继续) | 不变 (0) |
| 管理员终止 | (审批中心) | InstanceTerminate | 5 已终止 | OnTerminated | **回调写 4** |

边界语义: 退回/撤销后的单子**可编辑** (改完点「重新提交」, 复用实例从头重走, 历史保留);
已通过/已终止的单子不可改删; 驳回到节点无回调 (流程仍在跑, flow_status 保持 0)。

## 八、验收清单 (冒烟必过项)

用两个测试账号 (发起人 + 审批人, 参照 p012) 走一遍:

- [ ] 草稿新增/编辑/删除正常; 提交后列表状态变「审批中」且跳详情页
- [ ] 条件分支命中正确 (构造路由字段临界值, 如 days>3)
- [ ] 或签一人同意即过节点; 会签须全员; 会签中一人驳回=节点驳回
- [ ] 全链通过后, **刷新业务列表 flow_status=1** (回调链路通)
- [ ] 驳回退回 → 业务单可编辑 → 重新提交 → 状态回「审批中」(复用实例, 历史保留)
- [ ] 撤销 → flow_status=3; 已撤销单可再编辑再提交
- [ ] 审批中的单子: 修改/删除被拒 (40000 + 明确文案); 重复提交被拒
- [ ] 非 admin 用户列表只见本人; 普通审批人只需 `flow:task:handle` + 菜单可见即可审批
- [ ] `make vet` + `yarn build` + `make gen-api-sql PKG=<mod> CHECK=1` 全过

## 九、规范速查 (Do / Don't)

| ✅ Do | ❌ Don't |
|---|---|
| 发起/重提走业务后端 (StartForBiz / InstanceResubmit) | 前端直接 `api.instStart()` (无 bizId, 实例与业务单失联) |
| flow_status 只由回调写 (提交置 0 除外) | 业务代码在同意/驳回后手写状态 |
| 回调幂等: `WHERE id=bizId AND flow_instance=instanceId` | 回调里信任"最新实例就是自己" |
| 提交按钮 = `<FlowSubmitDialog :show-form=false>` | 业务页手写提交确认框/拼流程参数 |
| 审批 UI 全部跳 `/flow/detail` 或审批中心 | 业务页内嵌同意/驳回/时间线 |
| 表单快照由后端 `buildFormData` 从业务表组装 | 前端把业务字段塞进弹窗 formData 双写 |
| 条件路由字段 required + 条件节点配默认分支 | 路由字段可空 (未选悄悄走默认) |
| 双击防线: 前端按钮 loading + 回填条件更新 | 依赖"先查再写"防并发 (有窗口) |
| 新表登记 ormfill `fillTables` + logic.go import | 漏登记 (create_id 恒 0) / 漏 import (回调不注册) |
| casbin 直改后重启服务 | 种子执行完不重启 (策略不生效, 接口 403) |
