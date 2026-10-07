# Hinay Admin

通用后台管理系统脚手架，基于前后端分离架构构建。

## 技术栈

- **后端**: Go 1.26 + GoFrame v2.10 + MySQL + Redis + JWT + Casbin v2
- **前端**: Nuxt 4 + Vue 3 + Element Plus + Pinia + ECharts (TypeScript)

## 功能特性

- **权限管理**: 完整 RBAC 模型（用户/角色/菜单/API 资源），Casbin 双维度权限校验
- **认证鉴权**: JWT 登录 + Token 自动续签 + Redis 黑名单退出
- **数据权限**: 组织数据权限控制（全部/自定义/本部门/仅本人）
- **消息中心**: 系统通知 + 私信 + SSE 实时推送
- **定时任务**: gcron 调度，网页端配置管理
- **审批流**: 可视化流程设计器，支持会签/或签/转办/委派/加签等
- **AI 对话**: OpenAI 兼容接口，会话持久化
- **代码生成**: CLI + 网页版 CRUD 代码生成器
- **审计日志**: 操作日志自动记录，登录日志管理

## 快速开始

### 环境要求

- Go >= 1.26
- Node.js >= 18 (推荐 20)
- MySQL >= 8.0
- Redis >= 5.0

### 初始化数据库

```bash
cd server
# 修改 manifest/config/config.yaml 中的数据库配置
make initdb DB_USER=root DB_PASS=yourpass DB_NAME=hinay_admin
```

### 启动后端

```bash
cd server
go mod tidy
make run
# 默认监听 :8000
```

### 启动前端

```bash
cd web_src
yarn install
yarn dev
# 访问 http://localhost:3000
```

### 默认账号

```
账号: admin
密码: 123456
```

## Docker Compose 部署

```bash
# 准备环境变量
cp .env.example .env
# 修改 MYSQL_ROOT_PASSWORD 和 JWT_SECRET

# 启动所有服务
docker compose up -d --build

# 访问 http://localhost
```

## 目录结构

```
hinay-admin/
├── server/                    # Go 后端
│   ├── api/                   # API 接口契约
│   ├── internal/              # 核心业务代码
│   │   ├── controller/        # 控制器层
│   │   ├── service/           # 业务接口层
│   │   ├── logic/             # 业务实现层
│   │   ├── dao/               # 数据访问层
│   │   ├── model/             # 数据模型
│   │   └── middleware/        # 中间件
│   ├── utility/               # 工具函数
│   └── manifest/              # 配置和 SQL
└── web_src/                   # Nuxt 前端
    ├── app/
    │   ├── pages/             # 页面路由
    │   ├── composables/       # 组合式 API
    │   ├── stores/            # Pinia 状态管理
    │   └── components/        # 组件
    └── nuxt.config.ts
```

## 接口规范

- 鉴权: `Authorization: Bearer <token>`
- 响应格式: `{ "code": 0, "message": "ok", "data": ... }`
- API 前缀: `/api/v1`

##  License

Apache License 2.0