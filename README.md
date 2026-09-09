# Hinay Admin

通用后台管理系统脚手架, 前后端分离架构。

- 后端: Go 1.26 + GoFrame v2.10 + MySQL + Redis + JWT + Casbin v2
- 前端: Nuxt 4 + Vue 3 + Element Plus + Pinia + ECharts (TypeScript)

## 功能特性

- 完整的 RBAC 权限模型 (用户 / 角色 / 菜单 / API 资源 四套件)
- JWT 登录鉴权 + Token 自动续签 (401 静默刷新) + Redis 黑名单退出
- Casbin **双维度** 权限校验 (菜单维度 `menu:<id>` + API 维度 `path/method`)
- 动态侧边栏菜单 + 前端 `v-permission` 按钮级权限
- 图标可视化选择器 (Element Plus 全图标库, 支持搜索)
- 消息中心: 系统通知 (全员/角色/用户) + 私信 + 收件箱 + 已读/未读统计
- 个人中心: 资料修改 + 密码修改 + 头像上传
- 字典管理 (类型 + 数据项, 支持批量排序)
- 文件管理 (上传 / 列表 / MIME 过滤)
- 全局配置 (键值对, 多类型支持: 文本 / 数字 / 布尔 / JSON)
- 操作日志 (自动记录 POST/PUT/DELETE)
- 统一软删除规范: 所有核心表带 `deleted_at`
- 统一响应协议 `{ code, message, data }`
- 跨域 / RequestId 链路追踪 / 鉴权 / 权限四级中间件
- GoFrame 工程化分层: `api -> controller -> service -> logic -> dao` (gen ctrl / gen dao / gen service)

## 目录结构

```
hinay-admin/
├── server/                              # Go 后端 (GoFrame v2)
│   ├── api/                             # API 接口契约 (Req/Res + g.Meta 路由)
│   │   ├── auth/v1                      #   登录/登出/当前用户/菜单/个人中心
│   │   ├── message/v1                   #   消息中心 (系统通知 + 私信 + 收件箱)
│   │   └── system/v1                    #   用户 / 角色 / 菜单 / API 资源 / 字典 / 文件 / 配置 / 操作日志
│   ├── internal/
│   │   ├── cmd                          # 启动入口 + 路由分组
│   │   ├── controller/                  # 控制器层 (单方法薄透传, gf gen ctrl 生成)
│   │   │   ├── auth                     #   鉴权
│   │   │   ├── message                  #   消息
│   │   │   └── system                   #   系统管理全模块
│   │   ├── service/                     # 业务接口层 (IAuth/IMessage/IUser/IRole/IMenu/IApi/IDict/IFile/IConfig/IAuditLog)
│   │   ├── logic/                       # 业务实现层 (sXxx + init 注册到 service)
│   │   │   ├── auth                     #   登录/登出/菜单树/个人资料/改密/头像
│   │   │   ├── system                   #   用户/角色/菜单/API/字典/文件/配置/审计日志
│   │   │   ├── message                  #   消息通知 (含收件箱权限路由)
│   │   │   └── casbinx                  #   Casbin enforcer + 策略读写
│   │   ├── dao/                         # 数据访问对象 (gf gen dao 自动生成, 禁止手改)
│   │   ├── model/                       # 实体 + DO + 业务 DTO
│   │   ├── middleware                   # CORS / RequestId / Auth / Casbin
│   │   ├── consts                       # 常量与上下文 Key
│   │   └── packed                       # 注册 MySQL/Redis 驱动
│   ├── utility/
│   │   ├── jwtx                         # JWT 签发与校验
│   │   ├── password                     # bcrypt
│   │   ├── response                     # 统一响应 / 分页结构
│   │   ├── contextx                     # ctx 取登录用户/Token 工具
│   │   └── xerror                       # 错误码与包装
│   ├── manifest/
│   │   ├── config/config.yaml           # 运行时配置 (server/db/redis/jwt/casbin)
│   │   └── sql/init.sql                 # 数据库 DDL + 种子 + 初始策略
│   └── Makefile                         # run / initdb / vet / fmt / tidy
└── web_src/                             # Nuxt 4 前端
    ├── app/
    │   ├── app.vue                      # 根组件
    │   ├── layouts/                     # default(主骨架) / blank(登录)
    │   ├── pages/                       # 页面路由
    │   │   ├── login.vue                #   登录
    │   │   ├── dashboard.vue            #   仪表盘
    │   │   ├── profile.vue              #   个人中心 (资料+头像+密码)
    │   │   └── system/                  #   系统管理
    │   │       ├── users/index.vue      #     用户管理
    │   │       ├── roles/index.vue      #     角色管理
    │   │       ├── menus/index.vue      #     菜单管理 (含图标选择器)
    │   │       ├── apis/index.vue       #     API 管理
    │   │       ├── dicts/index.vue      #     字典管理
    │   │       ├── files/index.vue      #     文件管理
    │   │       ├── configs/index.vue    #     全局配置
    │   │       └── audit-logs/index.vue #     操作日志
    │   └── message/                     #   消息中心
    │       ├── system/index.vue         #     系统通知
    │       └── private/index.vue        #     私信
    │   ├── stores/user.ts               # Pinia 用户/菜单/权限
    │   ├── composables/
    │   │   ├── useRequest.ts            # 统一请求封装 (token 注入 / 401 静默续期)
    │   │   └── useApi.ts                # auth / system / message API hook
    │   ├── middleware/auth.global.ts    # 路由守卫
    │   └── plugins/
    │       ├── element-icons.ts         # 全局注册 @element-plus/icons-vue (侧边栏动态图标)
    │       ├── permission.ts            # v-permission 按钮级权限指令
    │       └── echarts.client.ts        # ECharts 客户端注册
    └── nuxt.config.ts                   # modules + nitro.devProxy
```

## 快速开始

### 1. 准备环境

- Go >= 1.26 (见 `server/go.mod`)
- Node.js >= 18 (推荐 20)
- MySQL >= 8.0 (兼容 5.7+)
- Redis >= 5.0

### 2. 初始化数据库

修改 `server/manifest/config/config.yaml` 中的 `database.default` 与 `redis.default` 连接参数, 然后执行:

```bash
cd server
make initdb DB_USER=root DB_PASS=yourpass DB_NAME=hinay_admin
# 或手动:
mysql -uroot -p hinay_admin < manifest/sql/init.sql
```

`init.sql` 会创建以下表并写入种子数据:

| 表 | 说明 |
| --- | --- |
| `sys_user` | 系统用户 (含 `deleted_at` 软删) |
| `sys_role` | 系统角色 |
| `sys_menu` | 菜单/按钮 (`type` 1=目录 2=菜单 3=按钮) |
| `sys_api` | API 资源 (用于角色 API 维度授权) |
| `casbin_rule` | Casbin 策略 (p=权限, g=用户-角色) |
| `biz_message` | 消息主表 (系统通知 + 私信) |
| `biz_message_target` | 系统通知定向目标 (角色/用户) |
| `biz_message_read` | 消息已读关系 (含个人删除 `hidden`) |
| `sys_config` | 全局配置 (文本/数字/布尔/JSON 多类型) |
| `sys_dict_type` | 字典类型 |
| `sys_dict_data` | 字典数据项 (支持排序) |
| `sys_file` | 文件管理 |
| `sys_audit_log` | 操作审计日志 |

### 3. 启动后端

```bash
cd server
go mod tidy
make run                  # 或 go run main.go
# 默认监听 :8000, API 前缀 /api/v1, Swagger: /swagger, OpenAPI: /api.json
```

### 4. 启动前端

```bash
cd web_src
yarn install              # 或 npm install / pnpm install
yarn dev
# http://localhost:3000  (/api 已通过 nitro.devProxy 反代到 :8000)
```

> **本地预览生产构建时需要指定后端地址**: `nitro.devProxy` 仅在 `yarn dev` 模式生效,
> `yarn preview` / `node .output/server/index.mjs` 没有 `/api` 代理, 需通过环境变量指向后端:
>
> ```bash
> NUXT_PUBLIC_API_BASE=http://127.0.0.1:8000/api/v1 node .output/server/index.mjs
> ```
>
> Docker 部署无需此变量, 由 nginx 统一反代 `/api` 到后端。

### 5. 默认账号

```
账号: admin
密码: 123456
```

> 内置 `admin` 用户默认绑定 `admin` 角色, 不可禁用、不可解绑超管角色、不可删除。

## Docker Compose 部署

一键启动 MySQL、Redis、Go 后端、Nuxt 前端四个服务, 无需手动安装依赖。

### 1. 准备环境变量

```bash
cp .env.example .env
```

按需修改 `.env` 中的密码和密钥:

```bash
MYSQL_ROOT_PASSWORD=<强密码>                    # MySQL root 密码
MYSQL_DATABASE=hinay_admin                      # 数据库名
JWT_SECRET=<openssl rand -hex 32 生成>           # JWT 密钥, 必填且至少 32 位随机字符
```

### 2. 构建并启动

```bash
docker compose up -d --build
```

首次启动会自动:
- 拉取 MySQL 8.0、Redis 7、Node.js 22、Go 1.26 等基础镜像
- 编译 Go 后端二进制
- 构建 Nuxt 前端产物
- 初始化数据库 (执行 `init.sql`, 创建表和种子数据)

### 3. 访问

| 服务 | 地址 | 说明 |
| --- | --- | --- |
| 前端 | http://localhost | Nginx 反向代理入口 |
| 后端 API | http://127.0.0.1:8000/api/v1 | Go API 服务 (仅绑定回环, 供本机调试) |
| MySQL | localhost:3306 | 数据库 |
| Redis | localhost:6379 | 缓存 |

默认账号: `admin` / `123456` (仅开发环境提示, 首次部署后请立即修改默认密码)

### 4. 常用命令

```bash
# 查看服务状态
docker compose ps

# 查看日志 (所有服务)
docker compose logs -f

# 查看单个服务日志
docker compose logs -f server
docker compose logs -f web

# 停止所有服务
docker compose down

# 停止并删除数据卷 (清空数据库)
docker compose down -v

# 重新构建某个服务
docker compose build server
docker compose up -d server
```

### 5. 文件说明

| 文件 | 说明 |
| --- | --- |
| `docker-compose.yml` | 服务编排, 定义四个服务及依赖关系 |
| `.env` / `.env.example` | 环境变量 (密码、密钥等) |
| `server/Dockerfile` | Go 后端多阶段构建 |
| `server/manifest/config/config.docker.yaml` | Docker 环境专用配置 (host 为容器名, 密码通过环境变量注入) |
| `web_src/Dockerfile` | Nuxt 前端多阶段构建 (Node + Nginx) |
| `web_src/nginx.conf` | Nginx 配置, `/api` 和 `/upload` 转发到后端 |

> 生产部署时, 请务必修改 `.env` 中的 `MYSQL_ROOT_PASSWORD` 和 `JWT_SECRET`。

## 配置说明

`server/manifest/config/config.yaml` 主要项:

```yaml
server:
  address: ":8000"
  openapiPath: /api.json
  swaggerPath: /swagger
  dumpRouterMap: true

database:
  default:
    type: mysql
    host: 127.0.0.1
    port: "3306"
    user: root
    pass: "yourpass"
    name: hinay_admin
    charset: utf8mb4

redis:
  default:
    address: 127.0.0.1:6379
    db: 0

jwt:
  secret:    "hinay-admin-please-change-me"
  expireSec: 86400          # token 过期时间(秒) 24h
  issuer:    "hinay-admin"
  header:    "Authorization"

casbin:
  enable: true
  model: |
    [request_definition]
    r = sub, obj, act
    [policy_definition]
    p = sub, obj, act
    [role_definition]
    g = _, _
    [policy_effect]
    e = some(where (p.eft == allow))
    [matchers]
    m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
```

> Casbin 模型使用 `g(r.sub, p.sub)`, 因此 `sub` 直接传 `username`, 由 g 策略自动解析其角色。

## 接口规范

- 鉴权: HTTP Header 加 `Authorization: Bearer <token>`
- 响应: `{ "code": 0, "message": "ok", "data": ... }`, 业务错误 `code` 非 0
- 错误码: 见 [`server/utility/xerror/xerror.go`](server/utility/xerror/xerror.go)
- 鉴权失败统一返回 401 (`40100`), 无权限返回 403 (`40300`); 前端 401 自动尝试静默续签 Token
- 分页响应统一 `response.PageResult { list, total, page, pageSize }`

### 主要接口一览

| 模块 | 路径 | 方法 | 说明 |
| --- | --- | --- | --- |
| 鉴权 | `/api/v1/auth/public-key` | GET | 获取一次性登录加密公钥 (公开, 单 IP 限流) |
| 鉴权 | `/api/v1/auth/login` | POST | 登录 (公开, 密码须用公钥 RSA 加密后提交) |
| 鉴权 | `/api/v1/auth/refresh` | POST | 刷新 Token |
| 鉴权 | `/api/v1/auth/logout` | POST | 登出 (Token 加 Redis 黑名单) |
| 鉴权 | `/api/v1/auth/userInfo` | GET | 当前用户信息 |
| 鉴权 | `/api/v1/auth/menus` | GET | 当前用户菜单 + 权限码 |
| 鉴权 | `/api/v1/auth/profile` | GET / PUT | 个人中心详情 / 修改资料 |
| 鉴权 | `/api/v1/auth/password` | PUT | 修改密码 |
| 鉴权 | `/api/v1/auth/avatar` | POST | 上传头像 |
| 用户 | `/api/v1/system/users` | GET / POST | 列表 / 新增 |
| 用户 | `/api/v1/system/users/{id}` | GET / PUT / DELETE | 详情 / 修改 / 软删 |
| 用户 | `/api/v1/system/users/{id}/password` | PUT | 重置密码 |
| 角色 | `/api/v1/system/roles` (含 `/all`) | CRUD | 列表 / 全量 / CRUD |
| 角色 | `/api/v1/system/roles/{id}/menus` | GET / PUT | 获取/分配菜单权限 |
| 角色 | `/api/v1/system/roles/{id}/apis` | GET / PUT | 获取/分配 API 权限 |
| 菜单 | `/api/v1/system/menus` (含 `/tree`) | CRUD | 扁平/树形/CRUD |
| API 资源 | `/api/v1/system/apis` | CRUD | API 接口资源管理 |
| 字典类型 | `/api/v1/system/dict-types` (含 `/all`) | CRUD | 字典类型管理 |
| 字典数据项 | `/api/v1/system/dict-types/{typeId}/items` | CRUD | 字典数据项管理 (含排序) |
| 字典全量 | `/api/v1/system/dicts/all` | GET | 全量字典数据 (按类型分组) |
| 文件 | `/api/v1/system/files` | GET | 文件列表 |
| 文件 | `/api/v1/system/files/upload` | POST | 上传文件 |
| 文件 | `/api/v1/system/files/{id}` | DELETE | 删除文件 |
| 全局配置 | `/api/v1/system/configs` (含 `/all`) | CRUD | 全局配置管理 |
| 操作日志 | `/api/v1/system/audit-logs` | GET | 操作日志列表 |
| 消息 | `/api/v1/message` | GET | 管理员消息列表 |
| 消息 | `/api/v1/message/system` | POST | 发布系统通知 |
| 消息 | `/api/v1/message/private` | POST | 发送私信 |
| 消息 | `/api/v1/message/inbox` | GET | 我的收件箱 |
| 消息 | `/api/v1/message/inbox/{id}` | GET / DELETE | 阅读详情(自动已读) / 个人删除 |
| 消息 | `/api/v1/message/{id}/read` | PUT | 标记单条已读 |
| 消息 | `/api/v1/message/read-all` | PUT | 全部标记已读 |
| 消息 | `/api/v1/message/unread-count` | GET | 未读消息数量 |

## 鉴权与权限模型

请求链路 (见 [`internal/cmd/cmd.go`](server/internal/cmd/cmd.go) 与 [`internal/middleware/middleware.go`](server/internal/middleware/middleware.go)):

```
CORS -> RequestId -> MiddlewareHandlerResponse -> Auth -> Casbin -> Controller
```

- **Auth**: 解析 JWT, 校验 Redis 黑名单, 写入 LoginUser 到 ctx; 命中 `publicPaths` (如 `/auth/login`) 直接放行。
- **Casbin**: 校验 `(username, path, method)`; 超管 (`admin` 角色) 全放行; 命中 `authWhitelist` (如 `/auth/menus`、`/auth/userInfo`、`/auth/profile`、`/auth/password`、`/auth/logout`) 已登录即可访问, 不进入策略检查。
- **双维度授权**:
  - 菜单维度: `p, <roleCode>, menu:<menuId>, *`  → 控制可见菜单与按钮权限码
  - API 维度: `p, <roleCode>, <apiPath>, <method>` → 控制实际 HTTP 接口
  - 用户-角色: `g, <username>, <roleCode>` → 在 g 策略中维护
- **前端**:
  - 登录后调用 `/auth/menus` 拉取 `{ menus, permissions }`
  - 路由按 `menus` 动态注入, 按钮用 `<el-button v-permission="'system:user:create'">`
  - 401 响应自动触发静默 Token 续签, 续签失败跳转登录页
  - 侧边栏菜单图标通过全局注册 `@element-plus/icons-vue` 动态渲染

## 二次开发

> 项目遵循 GoFrame v2 强规范化实践: 严禁 `g.DB().Model("xxx")` 形式硬编码表名, 数据访问统一走 `dao.Xxx.Ctx(ctx)`。详见 [memory: GoFrame v2项目开发强制规范]。

新增业务模块的标准步骤 (以 `message` 模块为参考):

1. **DDL**: 在 `server/manifest/sql/init.sql` 增加表与索引, 字段含 `deleted_at`, 顺便加初始菜单/权限码/API 资源。
2. **生成 dao/model**: 配好数据库后执行 `gf gen dao`, 自动生成 `internal/dao/<table>.go` 与 `internal/model/{do,entity}/<table>.go`。
3. **API 契约**: 在 `server/api/<module>/v1/` 编写 `XxxReq/XxxRes`, 用 `g.Meta` 声明 path/method/tags/summary, 加 `v` 校验规则。
4. **service 接口**: 在 `server/internal/service/<module>.go` 定义 `IXxx interface { Method(ctx, *v1.XxxReq) (*v1.XxxRes, error) }` + `localXxx` 单例 + `Xxx()` 取值 + `RegisterXxx(i IXxx)`。
5. **logic 实现**: 在 `server/internal/logic/<module>/` 写 `sXxx struct{}` 实现接口, `init()` 注册 `service.RegisterXxx(NewXxx())`, 数据库访问走 `dao.Xxx.Ctx(ctx)`。
6. **controller**: 执行 `gf gen ctrl` 生成 `internal/controller/<module>/<module>_v1_*.go`, 每个方法单行透传 `return service.Xxx().Method(ctx, req)`。
7. **绑定路由**: 在 `internal/cmd/cmd.go` 的 `sec.Bind(...)` 中加入 `<module>.NewV1()` (公开接口在 `middleware.publicPaths` 白名单短路)。
8. **前端**: `composables/useApi.ts` 新增 hook + `pages/<module>/index.vue` 页面。
9. **授权**: 通过菜单管理与角色管理为对应角色分配菜单权限码与 API 权限即可生效。

## 常用命令

```bash
# 后端
cd server
make run            # go run main.go
make initdb         # 初始化数据库
make vet            # go vet ./...
make fmt            # gofmt -s -w .
make tidy           # go mod tidy

# 前端
cd web_src
yarn dev            # 开发
yarn build          # 生产构建 (产物 .output/)
yarn generate       # 静态生成
```

## License

MIT
