# 多租户功能开发指南

> 本文面向在本脚手架上做二次开发的工程师：如何开启多租户、新代码要遵守哪些规约、
> 租户相关能力怎么用、怎么自测。架构决策与完整设计演进见团队内部设计文档。

## 一、功能概览与开关

**隔离模型**：共享库共享表 + `tenant_id` 行级隔离（SaaS 主流形态），单实例部署。

| 能力 | 说明 |
|------|------|
| 行级隔离 | 20 张 A 类表在 ORM 驱动层强制注入 `tenant_id`，业务代码绕不过、也无需感知 |
| 权限域化 | Casbin 升级域模型，各租户策略互不可见；角色唯一键 `(tenant_id, code)` |
| 双层管理员 | 平台超管（角色 id=1，跨租户）/ 租户管理员（`tenant_admin` 角色，本租户围栏内全权） |
| 应用授权集 | 平台通过 `sys_tenant_menu` 圈定每租户可用菜单/API，变更即时双向联动 Casbin 策略 |
| 配置覆盖 | 白名单键（站点名/Logo/版权/注册开关/密码策略）支持租户覆盖，读序：租户 > 平台 |
| 代运维切换 | 平台超管携带 `X-Tenant-Id` 头即可切换数据面到目标租户（内存态，非平台超管携带被忽略+告警审计） |
| 租户生命周期 | 创建向导（六件套单事务）/ 禁用与到期逐请求拦截 / 软删（code 原子改写防重建撞码） |

**开关**：`config.yaml` → `multiTenant.enable`（默认 `false`）。**关闭时系统行为与单租户完全一致**，
老部署升级零感知；修改后需重启生效。

**关闭语义（注入层兜底）**：开关关闭时，`tenantorm` 对隔离表**恒定注入平台租户 `tenant_id=1`**，
即"关闭 = 全部限定平台租户"：单租户数据本就全在平台租户下，行为无感；而关闭前遗留的
他租户数据将不可见、其用户不可登录（跨租户同名用户名在 `tenant_id=1` 下天然唯一，登录
不会错取他租户同名列）。因此**从多租户回退单租户前必须先清理**：先在租户管理里删光
非平台租户，或至少跑预检确认没有与平台用户重名的租户用户——

```sql
SELECT username FROM sys_user WHERE deleted_at IS NULL AND tenant_id<>1
  AND username IN (SELECT username FROM sys_user WHERE deleted_at IS NULL AND tenant_id=1);
```

预检有结果时，先处理这些重名用户（改名或删除所属租户）再关开关重启，否则关闭后这些
账号将无法登录（安全方向失败，不会串号，但会"凭空消失"）。

**新装**：直接执行 `manifest/sql/init.sql`（已含多租户基线）。
**存量单租户库升级**：先执行 `manifest/sql/upgrade-multi-tenant.sql`（务必先备份，
建议直接新装重建），再开开关重启。

## 二、核心心智模型（三条）

1. **数据面 × 身份面**：租户注入只作用于"数据面"（租户拥有的业务数据）。
   会话/自助端点（userInfo/profile/password/avatar/refresh）是"身份面"——操作的是
   操作者**本人**在归属租户的记录，统一走 `contextx.HomeTenantScope(ctx)`，
   不受代运维切换影响。
2. **平台超管默认不注入**：未切换时跨租户全量可见（列表混排 + "所属租户"列区分）；
   携带 `X-Tenant-Id` 切换后数据面 = 目标租户。
3. **租户管理员 = 圈养版超管**：本租户内体验与单租户超管一致（管用户/角色/组织），
   但跨不过租户边界（注入硬隔离）、可绑菜单/API ⊆ 应用授权集、永远碰不到平台功能
   （租户管理/监控/工具/菜单·API·字典目录维护/配置平台行）。

## 三、后端开发规约（新代码必读）

### 3.1 新增业务表

1. DDL 带 `tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 1` + `KEY idx_tenant (tenant_id)`；
2. 在 `server/internal/logic/ormfill/tenantorm.go` 的 `tenantTables` 登记（A 类隔离表）；
3. 有 `create_id/update_id` 审计列的表同时在 `ormfill.go` 的 `fillTables` 登记；
4. 唯一键必须租户化：`(tenant_id, col)`（全局唯一业务号需含租户短码或按租户独立序列）；
5. 建表语句保留 `DROP TABLE IF EXISTS` 前缀（幂等重建，防旧库结构残留）。

隔离由驱动层注入自动完成，业务 CRUD 代码**零改动**。

### 3.2 禁止事项（`make tenant-lint` 静态检查）

| 禁止 | 原因 |
|------|------|
| 手写 `tenant_id` 条件（`Where("tenant_id", ...)`） | 租户条件只由注入器管理；需要"写入目标租户内"的存在性预检用 `contextx.ScopedToDataTenant(ctx)` |
| `.Cache()` 查询缓存 | 缓存 key 不含租户维度，会串数据 |
| `tx.Exec` / `.Exec(` / `.Raw(` 手写 DML 引用隔离表 | 无安全注入点，`DoExec` 直接拒绝（fail-close）；改走 dao 结构化方法 |
| 新增外键约束 | 全库无外键方针，关联完整性由应用层 + 注入保证 |
| 业务代码写 `tenant_id == x` 分支 | 租户差异只能走 配置覆盖 / 应用授权集，防代码腐烂 |
| Redis/进程内缓存键无租户维度 | 新键必须带租户维度或证明数据无租户边界 |

### 3.3 SQL 形态注意（隔离表）

- 复杂形态（UNION / 子查询引用隔离表 / 派生表 / WITH / 自连接 / 语句含分号）会被
  注入器 **fail-close 拒绝**——这不是 bug，是保护；改走 dao 结构化方法或拆分语句；
- JOIN 只允许主表为隔离表 + 对端为非隔离表（如 `biz_message` LEFT JOIN `biz_message_read`）；
  A 类表互 JOIN 一律拒绝（列名歧义，请显式拆分）；
- 慢查询归因：注入器在语句头部追加 `/*tenant:N*/` 注释，Slow Query Log 可按租户定位。

### 3.4 配置读取（sys_config）

一律走 `internal/logic/configx`，**不要**直接 `dao.SysConfig` 查询（租户上下文直查会"丢失"平台行）：

```go
configx.Value(ctx, tenantLite, "sys.name", def)   // 白名单键: 租户行 > 平台行
configx.Bool(ctx, tenantLite, key, def)
configx.PlatformValue(ctx, key, def)               // 平台级键 (ai.*/wechat.*/file.storage.* 等)
configx.PlatformPrefix(ctx, "file.storage.")       // 平台级前缀批量
```

租户管理员维护覆盖行的入口是"租户设置"页（`/system/tenant-configs`，仅白名单键）；
平台干预走租户管理的"重置租户配置"。

### 3.5 异步 / 定时任务上下文

隔离表在无租户上下文时的写入会被 fail-close。规则：**一切异步边界显式携带租户**。

- 定时任务：`sys_job.tenant_id` 自动生效——执行时按该列构造租户 ctx；
  禁用/到期租户的任务自动跳过并写 job_log 原因；
- 平台级巡检任务（需覆盖全部租户）：用 `job.SweepTenants(ctx, func(ctx, tid) error {...})`
  逐租户构造注入 ctx 执行，单租户失败不阻断整轮；
- 业务回调（审批引擎 `BizListener`）：回调签名带 `ctx`，回写业务表必须透传该 ctx，
  不要自行 `context.Background()`；
- 其他后台 goroutine 写隔离表：从请求 ctx 取 `contextx.TenantId(ctx)` 落到载荷，
  消费端 `contextx.WithTenant` 还原；或用 `context.WithoutCancel(ctx)` 保值去取消。

### 3.6 权限与角色

- Casbin 域 = 租户 ID（p 行 `v3`、g 行 `v2`）；`casbinx` 全部接口自动从 ctx 取域；
- 管理员判定：`contextx.IsPlatformAdmin(ctx)`（平台超管）/ `IsTenantAdmin(ctx)`（租户管理员）/
  `IsManagerAdmin(ctx)`（两者之一，"管理员可见"类场景用这个，租户管理员天然限界本租户）；
- 平台专属功能（租户管理/监控/gencode/菜单·API·字典目录维护）守卫用 `IsPlatformAdmin`；
- 租户角色的可绑菜单/API 由写侧校验自动圈定（`system/role.go`），无需业务代码关心。

### 3.7 文件存储

`storage.NewKey(ctx, ext)` 自动按数据面租户加 `tenant/{tid}/` 前缀（平台租户保持旧布局，
存量数据零迁移）；本地 `/upload` 静态路由是公开的，**SaaS 生产形态建议 s3 私有桶 + 预签名**
（`multiTenant.enable=true` 且本地存储时启动日志会打警告）。

## 四、前端开发规约

- **请求头**：`useRequest` 已统一注入 `X-Tenant-Id`（仅平台超管切换态的内存值）；
  手写 fetch 的旁路（如 SSE）需自行补头（参考 `useApi/ai.ts`、`MessageBell.vue`）；
- **切换态清理**：监听 `window` 的 `tenant:switched` 事件断开长连接（参考 `pages/ai/chat`）；
  切换器已内聚完成 tags 重置 + 会话/配置重拉 + 回工作台；
- **开关感知**：`configStore.multiTenant`（来自 `/configs/all` 合成键 `sys.multi_tenant`），
  关闭态不渲染租户相关 UI；
- **租户列**：平台超管混排视角的列表加"所属租户"列，模式统一为
  `showTenantCol = computed(() => userStore.isPlatformAdmin && configStore.multiTenant)`；
- **下拉收窄**：选角色/选岗位等下拉数据源（`roles/all`、`posts/all`）已按数据租户收窄，
  新增类似"全量下拉"接口时同样处理（`contextx.ScopedToDataTenant`）。

## 五、常用后端接口（平台超管）

| 接口 | 用途 |
|------|------|
| `POST /system/tenants` | 创建租户（向导三步合一：基本信息+管理员账号+应用授权预览） |
| `PUT /system/tenants/{id}` | 修改（code 不可变；禁用/到期即时拦截该租户全部会话） |
| `DELETE /system/tenants/{id}` | 软删（code 改写 `{code}#del{unix}` 防重建撞码） |
| `GET/PUT /system/tenants/{id}/menus` | 应用授权集（PUT 全量覆盖，双向联动 Casbin 即时生效） |
| `POST /system/tenants/{id}/configs/reset` | 重置租户覆盖配置（回归平台默认） |
| `GET /auth/tenant/info?code=` | 公开租户信息（仅显示名，按 IP 限流） |

登录侧：请求体 `tenantCode`（或 `X-Tenant-Code` 头）指定租户，缺省平台租户；
错误三态：50023 租户码无效 / 50024 已禁用 / 50025 已到期。

## 六、测试资产（保留可重跑）

```bash
cd server
go test ./internal/logic/ormfill/   # 注入器全形态 (允许/放行/fail-close 34 例)
go test ./internal/logic/casbinx/   # 域化匹配器与域隔离
go test ./internal/logic/configx/   # 配置覆盖读序
go test ./internal/logic/tenant/    # 授权集闭包/差集/平台专属判定
make tenant-lint                    # 工程规约四项静态检查
# 端到端冒烟 (前置: make run + initdb + sys.captcha_enable=false):
bash tools/tenant-smoke.sh          # 16 组 56 断言: 隔离/切换/授权联动/生命周期全覆盖
```

## 七、常见坑速查

| 现象 | 原因与解法 |
|------|-----------|
| 隔离表 SQL 报"多租户注入保护" | 语句形态无法安全注入（子查询引用隔离表/UNION/手写 DML）→ 改走 dao 结构化方法 |
| 后台 goroutine 写隔离表失败 | 无租户上下文 → 按本文 3.5 构造 ctx |
| 平台超管切换后"用户不存在" | 身份面端点误用数据面注入 → 会话端点走 `HomeTenantScope`（已内置，新会话端点注意） |
| sys_config 读不到值 | 直查 dao 丢了平台行 → 走 `configx` |
| 角色下拉混入他租角色 | 全量接口未收窄 → `ScopedToDataTenant` |
| 新表数据跨租户可见 | 忘记在 `tenantTables` 登记 |
