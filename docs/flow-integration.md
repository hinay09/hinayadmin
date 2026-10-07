# 审批流业务模版对接指南

> 适用: 需要把"请假/报销/合同用印"等**业务表单**接入自由审批流的业务模块。
> **新业务审批模块开发请直接按 [flow-template.md](flow-template.md) 标准模板编码** (以 biz_leave 为
> 参照的端到端操作手册); 本文是模板背后的引擎语义与四步接入原理。
> 纯人工发起(不走业务代码)无需本文: 直接在「审批中心-流程定义」里设计表单并发布即可。
> 前端页面如何 发起/展示状态/标待办/内嵌审批按钮, 见 [flow-frontend.md](flow-frontend.md)。

## 一、整体模型

```
业务模块 (如 biz_leave)                      审批引擎 (logic/flow)
┌──────────────────┐   ①StartForBiz(flowKey, bizId, 表单, 发起人)
│ 业务表 + 状态冗余  │ ────────────────────────────────────▶ 创建实例(带快照)并推进
│                  │                                        │
│                  │   ②BizListener 回调 (事务提交后尽力调用)  │
│  flow_status     │ ◀──────────────────────────────────── │ 通过/被退回/撤销
└──────────────────┘                                        ▼
                                            系统通知(定向)提醒审批人 → 我的审批 处理
```

关键语义(务必了解):

| 语义 | 行为 |
|---|---|
| **快照** | 发起实例时复制当时的表单定义+节点树到实例上; 之后修改/停用流程定义**不影响在途实例** |
| **自选快照** | 发起/重提时的 selfSelects **整份存入实例** (`wf_instance.self_selects` 列), 引擎推进到深层"发起人自选"节点时从这里读取——不需要、也无法在审批中途补充选人 |
| **已失效** | 退回发起人/驳回到节点/撤销/重提时, 原"已同意"的审批任务自动置 `6=已失效` (撤回因时机约束不存在已同意, 不涉及) (流程图与列表同步变灰); 流转记录保留完整历史, 抄送已阅不回退 |
| **驳回** | **节点级动作 (或签/会签一致)**: 驳回者待办置已驳回, 同节点其余待办作废。回退目标二选一: 默认**退回发起人**(实例不终止, 发起人可修改表单后**重新提交**, 流程从头重走, 历史保留; **撤销后同样可重新提交**); 或**退回到指定已审批节点**——实例保持运行, 原已同意的审批整单置已失效, 目标节点沿用原审批人重新生成待办, 通过后沿原链路依次重审 (仅退回发起人才触发 OnReturned 回调, 驳回到节点无回调) |
| **或签/会签** | 或签任一人通过即过节点; 会签须全员通过; **会签中任一人驳回=该节点驳回** (同伴待办作废), 回退规则同上——可退回以前的任意已审批节点或退回发起人 |
| **转办** | 审批人可将我的待办转给他人: 原任务置 `4=已转出`, 目标人生成同节点新待办 (沿用原任务 receive_time, 与本轮同伴同轮聚合) 并收到通知; 会签同伴不受影响 |
| **委派** | 审批人可将我的待办交被委托人**代办** (区别于转办换人): 原任务置 `7=已委派`挂起, 被委托人生成 `delegate_from_id` 指向原任务的代办待办; 其通过「委派处理」提交意见后代办行置 `8=委办完成`, 任务回到原审批人做最终同意/驳回——被委托人不能直接表决/转办/再委派。或签/会签计数时该对任务整体仍算一票 |
| **撤回** | 发起人可在**尚无任何审批人同意**时收回流程 (提交错了收回改): 实例置 `7=已撤回` 待修改重提 (不触发 OnCanceled 终态回调, 而是 OnWithdrawn, 未注册时回退 OnReturned); 已有人审批则不可撤回, 只能撤销。当前待办作废并通知处理人 |
| **超时** | 审批节点可配置 `timeoutHours` 办理期限 + 超时策略 (`remind` 提醒 / `transfer` 自动转办 / `approve` 自动通过): 期限在引擎生成任务时物化为 `wf_task.due_time`, 由定时任务 `flow.timeoutScan` (sys_job 种子, 默认 10 分钟) 扫描处理。提醒每任务一次; 自动转办转给节点配置的 `timeoutTransfer` 用户且转办后不再计时; 自动通过与人工同意共用推进内核 (operator=系统); 被委派任务不代决, 一律降级为提醒 |
| **加签/减签** | 节点处理人或管理员可调整当前节点人员。**加签**: 追加必要审批人, 节点全部待办转为会签 (原处理人与新加人全员同意才过)——**或签节点加签后同样升级为会签, 此为既定语义** (加签=增加必要审批人, 而非增加可选项), 新任务沿用本轮 receive_time 与既有待办同轮聚合。**减签**: 移除指定待办审批人 (任务置 `5=已作废`, 时间线记 reduce), 至少保留一人; 已同意的不回退, 被移除人收到待办作废通知 |
| **终止** | 管理员可终止运行中实例: 待办作废、原同意置已失效、实例置 `5=已终止` 并触发 `OnTerminated` 回调 |
| **催办** | 发起人可对运行中实例催办: 通知当前全部待办审批人, 同实例 10 分钟限一次 |
| **并发安全** | 同意/驳回/撤销/撤回/重提/转办/委派/委派处理/终止/加签/减签/超时处理在事务内先对实例行 `SELECT ... FOR UPDATE` 串行化, 并以条件更新 (仅待办状态可处理) 防止双击双推进; 同意/转办对签核方式**锁后重读**——并发加签把节点升级为会签后再同意, 仍按会签语义等待全员, 不会按旧或签快照作废同伴推进 |
| **回调时机** | 状态回调在数据库事务提交后"尽力"调用(失败仅记日志), **业务侧须幂等** |
| **部门主管** | 从发起人所在组织起逐级向上找挂「主管岗」的用户(跳过发起人), 到根仍无则发起失败并报错 |

## 二、四步接入

### 第 1 步: 建流程定义并绑定 flow_key

「审批中心 → 流程定义 → 新增」: **流程标识(flow_key) 填业务标识**(如 `biz_leave`), 设计表单+流程后**发布**。
之后可继续编辑(对新发起即时生效), 建议流程标识一旦上线不要改动——它是业务代码与流程的绑定键。

### 第 2 步: 业务表增加流程状态冗余列(建议)

回调里只拿到 instanceId/bizId, 业务侧冗余状态可避免每次查实例:

```sql
ALTER TABLE biz_leave ADD COLUMN flow_status TINYINT NOT NULL DEFAULT 0
  COMMENT '审批状态:0=审批中,1=已通过,2=被退回,3=已撤销';
```

### 第 3 步: 业务代码发起流程

```go
// internal/logic/leave/leave.go (示例)
package leave

import (
    "context"

    "hinay.cn/admin/internal/dao"
    "hinay.cn/admin/internal/logic/flow"
)

// Submit 员工提交请假申请: 落业务表 → 发起审批
func Submit(ctx context.Context, leaveId, userId uint64) error {
    var lv *entity.BizLeave
    _ = dao.BizLeave.Ctx(ctx).Where("id", leaveId).Scan(&lv)

    // 发起审批: flowKey 对应流程定义的"流程标识"; bizId=业务单ID(回调定位用)
    instanceId, err := flow.StartForBiz(ctx, "biz_leave",
        lv.Title,                            // 申请标题(待办列表显示)
        map[string]any{                      // 表单数据, 键须与流程表单字段 key 一致
            "reason": lv.Reason,
            "days":   lv.Days,
        },
        leaveId,   // bizId
        userId,    // 发起人
        nil,       // selfSelects: 流程含"发起人自选"节点时传 {nodeId: [审批人ID]}
    )
    if err != nil {
        return err // 常见失败: 未发布/必填缺失/部门主管无法解析/自选未传
    }
    _, _ = dao.BizLeave.Ctx(ctx).Where("id", leaveId).
        Data(g.Map{"flow_status": 0, "flow_instance": instanceId}).Update()
    return nil
}
```

> 注意: `StartForBiz` 内部会做表单必填校验与审批人解析, 失败时业务单**不会**产生实例——
> 建议先落业务表(状态=审批中), 发起失败时把业务单标记为"发起失败"或回滚。

### 第 4 步: 注册状态回调

```go
// 同包 init() 中注册 (flow_key 与流程定义一致)
func init() {
    flow.RegisterBizListener("biz_leave", flow.BizListener{
        OnApproved: func(instanceId, bizId uint64) { // 全部节点通过
            _, _ = dao.BizLeave.Ctx(context.Background()).
                Where("id", bizId).Data(g.Map{"flow_status": 1}).Update()
        },
        OnReturned: func(instanceId, bizId uint64) { // 被驳回退回发起人(可能重提)
            _, _ = dao.BizLeave.Ctx(context.Background()).
                Where("id", bizId).Data(g.Map{"flow_status": 2}).Update()
        },
        OnWithdrawn: func(instanceId, bizId uint64) { // 发起人撤回(可重提; 撤回≈退回, 未注册时回退 OnReturned)
            _, _ = dao.BizLeave.Ctx(context.Background()).
                Where("id", bizId).Data(g.Map{"flow_status": 2}).Update()
        },
        OnCanceled: func(instanceId, bizId uint64) { // 发起人撤销
            _, _ = dao.BizLeave.Ctx(context.Background()).
                Where("id", bizId).Data(g.Map{"flow_status": 3}).Update()
        },
        OnTerminated: func(instanceId, bizId uint64) { // 管理员终止 (可选)
            _, _ = dao.BizLeave.Ctx(context.Background()).
                Where("id", bizId).Data(g.Map{"flow_status": 4}).Update()
        },
    })
}
```

## 三、辅助查询

```go
// 按 (flowKey, bizId) 取最新实例 (拿实时状态/当前节点)
inst, err := flow.BizInstance(ctx, "biz_leave", leaveId)
if err == nil {
    _ = inst.Status        // 1=运行中 2=已通过 6=已退回待重提 4=已撤销
    _ = inst.CurrentNodeIds
}
```

用户在前台看到的审批入口与普通流程完全一致: 「审批中心 → 我的审批」待办/详情页,
详情页里展示的就是发起时**快照的表单**。业务模块通常只需要一个"发起"按钮 + 状态展示。

## 四、审批人类型速查(设计器可选)

| 类型 | 解析规则 | 何时失效 |
|---|---|---|
| 指定成员 | 固定用户列表 | 用户被禁用/删除 |
| 指定角色 | casbin 角色下全部用户(运行时展开) | 角色下无启用用户 |
| 指定岗位 | 挂该岗位的启用用户(「岗位管理」维护) | 岗位无成员 |
| 发起人自选 | 发起/重提时必须选择 | 未传 selfSelects |
| 部门主管 | 发起人组织起逐级向上找挂「主管岗」者 | 无组织/到根无人挂主管岗 |
| 发起人本人 | 发起人自己审批 | - |

岗位维护入口: 「系统管理 → 岗位管理」——建岗位(勾选"主管岗"即为部门主管解析依据),
在"成员"里指派用户并选择组织(同一岗位可在不同组织各派人)。

## 五、常见问题

- **发起报"仅已发布流程可发起"**: 定义还停在草稿, 在流程定义页点「发布」。
- **发起报"部门主管无法解析"**: 发起人没挂组织, 或其组织及上级都无人挂主管岗;
  到「岗位管理」给相应组织指派主管岗成员。
- **重提会不会换审批人**: 自选节点会预填上一轮选择(可改); 其余类型按规则重新解析
  (角色/岗位成员变化会反映到新一轮)。
- **回调没触发?**: 回调在事务提交后异步尽力调用, 须保证进程存活; 业务处理请做幂等
  (以 bizId+状态为准, 重复回调不产生副作用)。

## 六、内置完整示例: 请假申请 Demo (biz_leave)

仓库自带一个按本文四步落地的**业务型审批 Demo** (p016 种子, 菜单「业务审批 → 请假申请」),
业务与流程分离的单表实现, 已沉淀为**标准审批模板** —— 新业务审批模块照
[flow-template.md](flow-template.md) 编码, 各环节位置:

| 环节 | 位置 |
|---|---|
| 业务表 (含 `flow_status`/`flow_instance` 冗余列) | `manifest/sql/upgrade-modules/p016_biz_leave_demo.sql` |
| 回调注册 (四回调写回 `flow_status`) | `server/internal/logic/leave/leave.go` 的 `init()` |
| 发起 + 退回/撤销后重提 (复用实例) | 同文件 `LeaveSubmit` (`flow.StartForBiz` / `InstanceResubmit`) |
| 撤销 (转调引擎) | 同文件 `LeaveCancel` (`InstanceCancel`, 状态由 `OnCanceled` 回调写回) |
| 前端页面 (列表/业务表单 drawer/状态列/审批跳转) | `web_src/app/pages/biz/leave/index.vue` + `composables/useApi/leave.ts` |
| 演示流程定义 (条件分支 days>3 加会签 + 抄送) | p016 种子 `wf_definition` (flow_key=`biz_leave`), 依赖 p012 测试账号 |

**业务与流程分离的提交形态** (flow-frontend.md 第七节的 `:show-form="false"` 模式):
单据增删改是业务自己的事 (草稿态, 发起前自由操作); 「提交审批」按钮打开公共弹窗
`<FlowSubmitDialog flow-key="biz_leave" :show-form="false" :show-title="false">`,
弹窗负责流程确认与自选审批人 (`payload.selfSelects` 透传 `StartForBiz`/`InstanceResubmit`——
流程定义将来加"发起人自选"节点时弹窗自动渲染选人 UI, 业务页零改动),
表单快照由后端 `buildFormData` 从业务表组装, 不在弹窗里重复录入业务字段。

体验路径: 执行 p012 + p016 种子并重启服务 → 用 `flow_demo_zhang` (密码 `Flow@Test2026`) 登录
→ 业务审批 → 请假申请 → 新增草稿 → 「提交审批」→ 分别用 `flow_demo_li` / `flow_demo_zhao` 在
「审批中心 → 我的审批」处理 → 回到请假页看状态列被回调逐次改写 (审批中 → 已通过/被退回/已撤销)。
