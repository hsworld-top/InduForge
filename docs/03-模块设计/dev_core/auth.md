# 认证与授权

## 概述

InduForge 使用 JWT (JSON Web Token) 作为服务端会话凭据，浏览器通过同源 `HttpOnly` Cookie 使用会话；前端不读取、保存或传递 JWT。权限控制基于 RBAC (Role-Based Access Control)，登录支持滑块验证（按需触发）。

本文档只描述用户登录、角色权限和接口访问控制，不描述产品商务授权、节点运行授权和工程资产授权。离线授权设计见 [离线授权与工程资产保护设计](../../06-运维与安全/离线授权与工程资产保护设计.md)。

## 认证流程

### 1. 用户登录

```javascript
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

成功后服务端设置 `if_access`、`if_refresh` 两个同源 `HttpOnly` Cookie，响应只返回用户资料：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user": {
      "id": "user_001",
      "username": "admin",
      "name": "管理员",
      "role": "SYSTEM_ADMIN",
      "tenantId": "tenant_001"
    }
  }
}
```

### 2. 使用会话

浏览器后续同源请求自动携带 `HttpOnly` Cookie，前端不应手写 `Authorization` 请求头。服务间调用可从可信服务端提取 JWT 后构造内部 Bearer 请求头。

```javascript
GET /api/v1/users
```

### 3. 会话刷新

浏览器调用刷新接口时自动携带 `if_refresh` Cookie，服务端轮换访问与刷新 Cookie：

```javascript
POST /api/v1/auth/refresh
Content-Type: application/json
```

### 4. 用户登出

```javascript
POST /api/v1/auth/logout
```

## JWT Token

### Token 结构

JWT Token 包含三部分：

```
Header.Payload.Signature
```

### Payload 内容

```json
{
  "userId": "user_001",
  "username": "admin",
  "role": "SYSTEM_ADMIN",
  "tenantId": "tenant_001",
  "iat": 1701234567,
  "exp": 1701320967
}
```

### Token 配置

```env
JWT_SECRET=your-secret-key
JWT_EXPIRES_IN=24h
JWT_REFRESH_EXPIRES_IN=7d
```

## 角色和权限

### 角色定义

| 角色          | 说明       | 权限范围             |
| ------------- | ---------- | -------------------- |
| SUPER_ADMIN   | 超级管理员 | 所有权限             |
| SYSTEM_ADMIN  | 系统管理员 | 租户管理、系统级管理 |
| PROJECT_ADMIN | 工程管理员 | 工程管理与配置       |
| OPS_ADMIN     | 运维管理员 | 运维与日志相关       |
| USER_ADMIN    | 用户管理员 | 用户管理             |

### 权限列表

#### 用户管理

- `user:create` - 创建用户
- `user:read` - 查看用户
- `user:update` - 更新用户
- `user:delete` - 删除用户

#### 租户管理

- `tenant:create` - 创建租户
- `tenant:read` - 查看租户
- `tenant:update` - 更新租户
- `tenant:delete` - 删除租户

#### 工程管理

- `project:create` - 创建工程
- `project:read` - 查看工程
- `project:update` - 更新工程
- `project:delete` - 删除工程

#### 数据管理

- `data:connection:create` - 创建数据连接
- `data:connection:read` - 查看数据连接
- `data:connection:update` - 更新数据连接
- `data:connection:delete` - 删除数据连接
- `data:query:execute` - 执行查询

### 角色权限映射

```javascript
const rolePermissions = {
  SUPER_ADMIN: ['*'],
  SYSTEM_ADMIN: ['*'],
  PROJECT_ADMIN: ['project:*', 'data:*', 'design:*'],
  OPS_ADMIN: ['logs:*', 'project:read', 'data:read'],
  USER_ADMIN: ['user:*', 'tenant:read'],
}
```

## 权限检查

### 中间件

```javascript
// 检查是否已认证
const requireAuth = (req, res, next) => {
  const token = req.cookies.if_access
  if (!token) {
    return res.status(401).json({ error: 'Unauthorized' })
  }

  try {
    const decoded = jwt.verify(token, process.env.JWT_SECRET)
    req.user = decoded
    next()
  } catch (error) {
    return res.status(401).json({ error: 'Invalid token' })
  }
}

// 检查权限
const requirePermission = (permission) => {
  return (req, res, next) => {
    if (!hasPermission(req.user.role, permission)) {
      return res.status(403).json({ error: 'Forbidden' })
    }
    next()
  }
}
```

### 使用示例

```javascript
// 需要认证
router.get('/users', requireAuth, getUsers)

// 需要特定权限
router.post('/users', requireAuth, requirePermission('user:create'), createUser)

// 需要多个权限之一
router.get(
  '/projects',
  requireAuth,
  requireAnyPermission(['project:read', 'project:update']),
  getProjects,
)
```

## 数据隔离

### 租户隔离

每个租户的数据相互隔离：

```javascript
// 查询时自动添加租户过滤
const getProjects = async (req, res) => {
  const projects = await Project.findAll({
    where: {
      tenantId: req.user.tenantId,
    },
  })
  res.json(projects)
}
```

### 用户隔离

普通用户只能访问自己创建的资源：

```javascript
const getMyProjects = async (req, res) => {
  const projects = await Project.findAll({
    where: {
      tenantId: req.user.tenantId,
      createdBy: req.user.userId,
    },
  })
  res.json(projects)
}
```

## 安全最佳实践

### 1. 密码安全

- 使用 bcrypt 加密密码
- 密码强度要求：至少 8 位，包含大小写字母和数字
- 定期更换密码

```javascript
const bcrypt = require('bcryptjs')

// 加密密码
const hashedPassword = await bcrypt.hash(password, 10)

// 验证密码
const isValid = await bcrypt.compare(password, hashedPassword)
```

### 2. 会话安全

- 使用强随机密钥
- 设置合理的过期时间
- 使用 HTTPS 传输
- 不在 URL、`localStorage` 或跨窗口消息中传递 JWT

### 3. 防止攻击

- SQL 注入：使用参数化查询
- XSS 攻击：对用户输入进行转义
- CSRF 攻击：使用 CSRF Token
- 暴力破解：限制登录尝试次数

### 4. 日志审计

记录关键操作：

```javascript
logger.info('User login', {
  userId: user.id,
  username: user.username,
  ip: req.ip,
  timestamp: new Date(),
})
```

## 常见问题

### Q: 会话过期怎么办？

A: 浏览器调用 `/api/v1/auth/refresh`，由 `if_refresh` Cookie 自动携带刷新凭据并轮换会话。

### Q: 如何实现单点登录（SSO）？

A: 可以集成 OAuth 2.0 或 SAML 协议。

### Q: 如何实现多租户隔离？

A: 在数据库查询时自动添加 tenantId 过滤条件。

### Q: 如何降低会话被盗用风险？

A:

1. 使用 HTTPS
2. 设置短过期时间
3. 绑定 IP 地址
4. 服务端撤销访问令牌和刷新会话

## 相关资源

- [后端 API 文档](./README.md)
- [环境端口规划](../../06-运维与安全/环境端口规划.md)
- [JWT 官方文档](https://jwt.io/)
- [bcrypt 文档](https://github.com/kelektiv/node.bcrypt.js)

---

**版本**: 2.1.0
**最后更新**: 2026-08-12
