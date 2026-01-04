# 后端 API 文档

## 概述

InduForge 后端服务基于 Node.js + Express 构建，提供统一的 API 接口。

## 技术栈

- **运行时**: Node.js 18+
- **框架**: Express
- **ORM**: Sequelize
- **数据库**: MySQL / PostgreSQL / SQL Server
- **认证**: JWT
- **国际化**: i18next
- **日志**: Winston

## 项目结构

```
dev_core/
├── src/
│   ├── app.js              # Express 应用
│   ├── index.js            # 入口文件
│   ├── config/             # 配置文件
│   ├── constants/          # 常量定义
│   ├── controllers/        # 控制器
│   ├── middlewares/        # 中间件
│   ├── models/             # 数据模型
│   ├── routes/             # 路由
│   ├── services/           # 业务逻辑
│   ├── utils/              # 工具函数
│   └── locales/            # 国际化文件
├── database/               # 数据库脚本
├── scripts/                # 脚本文件
└── config/                 # 配置文件
```

## API 接口（高层）

### 基础能力

| 模块 | 路径 | 说明 |
| --- | --- | --- |
| 健康检查 | `GET /health` | 服务状态 |
| 文档 | `GET /api-docs` | Swagger（非生产环境） |
| V2 测试 | `GET /api/v2/test` | V2 示例接口 |

### 认证相关

| 路径 | 说明 |
| --- | --- |
| `GET /api/v1/auth/captcha` | 获取验证码 |
| `POST /api/v1/auth/login` | 登录 |
| `POST /api/v1/auth/refresh` | 刷新 Token |
| `POST /api/v1/auth/logout` | 退出登录 |
| `GET /api/v1/auth/me` | 获取当前用户 |
| `GET /api/v1/auth/config` | 登录配置 |

### 租户/用户/工程

| 模块 | 路径前缀 | 说明 |
| --- | --- | --- |
| 租户 | `/api/v1/tenants` | 租户管理 |
| 用户 | `/api/v1/users` | 用户管理 |
| 工程 | `/api/v1/projects` | 工程管理 |
| 日志 | `/api/v1/logs` | 系统日志 |
| 角色 | `/api/v1/roles` | 角色管理 |

### 数据中心（关系库）

| 路径 | 说明 |
| --- | --- |
| `GET /api/v1/data/projects/:projectId/connections` | 连接列表 |
| `POST /api/v1/data/projects/:projectId/connections` | 创建连接 |
| `PUT /api/v1/data/projects/:projectId/connections/:connectionId` | 更新连接 |
| `DELETE /api/v1/data/projects/:projectId/connections/:connectionId` | 删除连接 |
| `PATCH /api/v1/data/projects/:projectId/connections/:connectionId/status` | 更新连接状态 |
| `POST /api/v1/data/projects/:projectId/connections/test` | 测试连接 |
| `GET /api/v1/data/projects/:projectId/connections/:connectionId/tables` | 表列表 |
| `GET /api/v1/data/projects/:projectId/connections/:connectionId/tables/:tableName/data` | 表数据 |
| `GET /api/v1/data/projects/:projectId/connections/:connectionId/tables/:tableName/structure` | 表结构 |
| `POST /api/v1/data/projects/:projectId/connections/:connectionId/execute-sql` | 执行 SQL |
| `GET /api/v1/data/projects/:projectId/queries` | 查询列表 |
| `POST /api/v1/data/projects/:projectId/queries` | 创建查询 |
| `POST /api/v1/data/queries/:id/execute` | 执行查询 |

### 数据中心（MQTT）

| 路径 | 说明 |
| --- | --- |
| `POST /api/v1/data/projects/:projectId/mqtt/connections` | 创建连接 |
| `GET /api/v1/data/projects/:projectId/mqtt/connections` | 连接列表 |
| `GET /api/v1/data/mqtt/connections/:id` | 连接详情 |
| `PUT /api/v1/data/mqtt/connections/:id` | 更新连接 |
| `DELETE /api/v1/data/mqtt/connections/:id` | 删除连接 |
| `POST /api/v1/data/mqtt/connections/test` | 测试连接 |
| `POST /api/v1/data/mqtt/connections/:id/start` | 启动连接 |
| `POST /api/v1/data/mqtt/connections/:id/stop` | 停止连接 |
| `GET /api/v1/data/mqtt/connections/:id/status` | 获取连接状态 |
| `GET /api/v1/data/projects/:projectId/mqtt/connections/:connectionId/subscriptions` | 订阅列表 |
| `POST /api/v1/data/projects/:projectId/mqtt/connections/:connectionId/subscriptions` | 创建订阅 |
| `GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId` | 订阅详情 |
| `PUT /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId` | 更新订阅 |
| `DELETE /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId` | 删除订阅 |
| `PATCH /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/toggle` | 启停订阅 |
| `GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/messages` | 订阅消息 |
| `GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tag-groups` | 变量组列表 |
| `POST /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tag-groups` | 创建变量组 |
| `PUT /api/v1/data/mqtt/tag-groups/order` | 变量组排序 |
| `GET /api/v1/data/mqtt/tag-groups/:groupId` | 变量组详情 |
| `PUT /api/v1/data/mqtt/tag-groups/:groupId` | 更新变量组 |
| `DELETE /api/v1/data/mqtt/tag-groups/:groupId` | 删除变量组 |
| `GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags` | 变量列表 |
| `POST /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags` | 创建变量 |
| `POST /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags/batch` | 批量创建 |
| `GET /api/v1/data/projects/:projectId/mqtt/tags` | 工程内变量 |
| `GET /api/v1/data/mqtt/tags/:id` | 变量详情 |
| `PUT /api/v1/data/mqtt/tags/:id` | 更新变量 |
| `DELETE /api/v1/data/mqtt/tags/:id` | 删除变量 |
| `PATCH /api/v1/data/mqtt/tags/:id/toggle` | 启停变量 |
| `PUT /api/v1/data/mqtt/tags/order` | 变量排序 |
| `GET /api/v1/data/mqtt/tags/:id/value` | 变量值 |
| `POST /api/v1/data/mqtt/tags/values` | 批量变量值 |

### 设计中心

| 路径 | 说明 |
| --- | --- |
| `GET /api/v1/design/projects/:projectId/pages` | 页面列表 |
| `POST /api/v1/design/projects/:projectId/pages` | 创建页面 |
| `GET /api/v1/design/projects/:projectId/pages/:pageId` | 页面详情 |
| `PUT /api/v1/design/projects/:projectId/pages/:pageId` | 更新页面 |
| `DELETE /api/v1/design/projects/:projectId/pages/:pageId` | 删除页面 |
| `PATCH /api/v1/design/projects/:projectId/pages/:pageId/rename` | 重命名 |
| `PATCH /api/v1/design/projects/:projectId/pages/:pageId/move` | 移动页面 |
| `GET /api/v1/design/projects/:projectId/variables` | 全局变量 |
| `PUT /api/v1/design/projects/:projectId/variables` | 更新全局变量 |

## 认证和授权

### JWT Token

使用 JWT 进行身份认证：

```javascript
// 请求头
Authorization: Bearer <token>
```

### Token 刷新

Token 过期后可以使用刷新接口获取新 Token：

```javascript
POST /api/v1/auth/refresh
{
  "refreshToken": "<refresh_token>"
}
```

### 权限控制

基于角色的权限控制（RBAC）：

- SUPER_ADMIN: 超级管理员
- SYSTEM_ADMIN: 系统管理员
- PROJECT_ADMIN: 工程管理员
- OPS_ADMIN: 运维管理员
- USER_ADMIN: 用户管理员

## 错误处理

### 响应格式（统一）

```json
{
  "success": false,
  "errorCode": "ERROR_CODE",
  "message": "错误信息",
  "requestId": "REQ_ID"
}
```

### 常见错误码

错误码以 `dev_core/src/constants/errorCodes.js` 为准。

## 国际化

支持多语言：

```javascript
// 请求头
Accept-Language: zh-CN
```

支持的语言：
- zh-CN: 简体中文
- en-US: 英语

## 日志

使用 Winston 记录日志：

- 日志级别: error, warn, info, debug
- 日志文件: logs/app.log
- 错误日志: logs/error.log

## 配置

### 环境变量

```env
# 服务器配置
PORT=9099
NODE_ENV=development

# 数据库配置
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=123456
DB_NAME=tenant_management

# JWT 配置
JWT_SECRET=your-secret-key
JWT_EXPIRES_IN=24h
JWT_REFRESH_EXPIRES_IN=7d

# 日志配置
LOG_LEVEL=info
```

## 开发指南

### 启动开发服务器

```bash
cd dev_core
pnpm install
pnpm dev
```

### 运行测试

```bash
pnpm test
```

### 启动生产服务器

```bash
pnpm start
```

## 相关文档

- [认证与授权](./auth.md)
- [数据库初始化](./database-init.md)
- [数据连接管理](../datacenter/connections.md)
- [查询管理](../datacenter/queries.md)

---

**版本**: 2.1.0  
**最后更新**: 2025-03-08
