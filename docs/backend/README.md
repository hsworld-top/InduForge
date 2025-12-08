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

## API 接口

### 认证相关

```
POST   /api/v1/auth/login           # 用户登录
POST   /api/v1/auth/logout          # 用户登出
POST   /api/v1/auth/refresh         # 刷新 Token
GET    /api/v1/auth/me              # 获取当前用户信息
```

### 用户管理

```
GET    /api/v1/users                # 获取用户列表
POST   /api/v1/users                # 创建用户
GET    /api/v1/users/:id            # 获取用户详情
PUT    /api/v1/users/:id            # 更新用户
DELETE /api/v1/users/:id            # 删除用户
```

### 租户管理

```
GET    /api/v1/tenants              # 获取租户列表
POST   /api/v1/tenants              # 创建租户
GET    /api/v1/tenants/:id          # 获取租户详情
PUT    /api/v1/tenants/:id          # 更新租户
DELETE /api/v1/tenants/:id          # 删除租户
```

### 工程管理

```
GET    /api/v1/projects             # 获取工程列表
POST   /api/v1/projects             # 创建工程
GET    /api/v1/projects/:id         # 获取工程详情
PUT    /api/v1/projects/:id         # 更新工程
DELETE /api/v1/projects/:id         # 删除工程
```

### 数据连接

```
GET    /api/v1/data/projects/:projectId/connections              # 获取连接列表
POST   /api/v1/data/projects/:projectId/connections              # 创建连接
GET    /api/v1/data/projects/:projectId/connections/:id          # 获取连接详情
PUT    /api/v1/data/projects/:projectId/connections/:id          # 更新连接
DELETE /api/v1/data/projects/:projectId/connections/:id          # 删除连接
POST   /api/v1/data/projects/:projectId/connections/:id/test     # 测试连接
```

### 查询管理

```
GET    /api/v1/data/projects/:projectId/queries                  # 获取查询列表
POST   /api/v1/data/projects/:projectId/queries                  # 创建查询
GET    /api/v1/data/projects/:projectId/queries/:id              # 获取查询详情
PUT    /api/v1/data/projects/:projectId/queries/:id              # 更新查询
DELETE /api/v1/data/projects/:projectId/queries/:id              # 删除查询
POST   /api/v1/data/queries/:id/execute                          # 执行查询
```

### 数据操作

```
POST   /api/v1/data/projects/:projectId/connections/:id/execute-sql        # 执行 SQL
GET    /api/v1/data/projects/:projectId/connections/:id/tables             # 获取表列表
GET    /api/v1/data/projects/:projectId/connections/:id/tables/:name/data  # 获取表数据
```

### 页面管理

```
GET    /api/v1/design/projects/:projectId/pages                 # 获取页面列表
POST   /api/v1/design/projects/:projectId/pages                 # 创建页面
GET    /api/v1/design/projects/:projectId/pages/:id             # 获取页面详情
PUT    /api/v1/design/projects/:projectId/pages/:id             # 更新页面
DELETE /api/v1/design/projects/:projectId/pages/:id             # 删除页面
```

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
- TENANT_ADMIN: 租户管理员
- USER: 普通用户

## 错误处理

### 错误响应格式

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "错误信息",
    "details": {}
  }
}
```

### 常见错误码

| 错误码 | HTTP 状态码 | 说明 |
|--------|------------|------|
| UNAUTHORIZED | 401 | 未授权 |
| FORBIDDEN | 403 | 禁止访问 |
| NOT_FOUND | 404 | 资源不存在 |
| VALIDATION_ERROR | 400 | 参数验证失败 |
| INTERNAL_ERROR | 500 | 服务器内部错误 |

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

### 构建生产版本

```bash
pnpm build
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

**版本**: 2.0.0  
**最后更新**: 2025-12-08
