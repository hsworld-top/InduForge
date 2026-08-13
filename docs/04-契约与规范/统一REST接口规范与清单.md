# 统一REST接口规范与清单

## 1. 适用范围

本文档覆盖当前仓库中两个对外 HTTP 服务的正式接口：

- `dev_core`
- `data_service`

目标：

- 统一响应格式
- 明确业务失败与技术异常边界
- 提供模块化接口清单与功能概要
- 为前端适配、联调和后续开发提供单一参考

## 2. 统一响应格式

### 2.1 JSON 接口统一包络

```json
{
  "code": 0,
  "msg": "操作成功",
  "data": null,
  "reqId": "req-example-1"
}
```

### 2.2 判定规则

- `code === 0`：成功
- `HTTP 2xx && code !== 0`：业务失败
- `HTTP 4xx/5xx && code !== 0`：技术异常

### 2.3 文件流例外

以下接口允许成功时直接返回文件流，不强制使用 JSON 包络：

- `dev_core` 的租户资产流式读取
- `dev_core` 的发布包下载

这类接口一旦返回错误，仍必须使用统一 JSON 包络。

## 3. HTTP 状态码使用约定

### 3.1 业务失败

适用场景：

- 权限不足
- 资源不存在
- 名称重复
- 状态不允许
- 业务参数不满足约束

统一约定：

- `HTTP 2xx`
- `code != 0`

### 3.2 技术异常

适用场景：

- 鉴权链路失败
- 请求体无法解析
- 数据库、缓存、对象存储、下游服务异常
- 未捕获异常

统一约定：

- `HTTP 4xx/5xx`
- `code != 0`

## 4. dev_core 接口清单

### 4.1 模块功能概要

`dev_core` 是平台控制面后端，负责：

- 认证与租户治理
- 用户、角色、工程管理
- Designer 页面与资源管理
- 节点注册、部署、发布、日志与运维聚合
- 运行态账号与角色的项目侧治理

### 4.2 认证模块 `/api/v1/auth`

| 方法   | 路径                    | 功能概要             |
| ------ | ----------------------- | -------------------- |
| `GET`  | `/api/v1/auth/captcha`  | 获取登录滑块挑战（拖至最右侧） |
| `POST` | `/api/v1/auth/login`    | 登录并设置 HttpOnly 会话 Cookie |
| `POST` | `/api/v1/auth/refresh`  | 使用刷新 Cookie 轮换会话 |
| `POST` | `/api/v1/auth/logout`   | 注销、撤销会话并清除 Cookie |
| `PUT`  | `/api/v1/auth/password` | 当前用户修改密码     |
| `GET`  | `/api/v1/auth/me`       | 获取当前用户信息     |
| `GET`  | `/api/v1/auth/config`   | 获取登录页和应用配置 |

浏览器调用认证接口时由同源 `HttpOnly` Cookie 承载会话，响应体不返回访问令牌或刷新令牌；浏览器前端不得保存 JWT。

典型业务失败：

- `10010` 滑块验证错误或过期
- `10007` 用户名或密码错误
- `10008` 缺少租户编码
- `10009` 租户编码无效

典型技术异常：

- `500/30001` 系统内部错误

### 4.3 租户模块 `/api/v1/tenants`

| 方法     | 路径                                              | 功能概要                       |
| -------- | ------------------------------------------------- | ------------------------------ |
| `GET`    | `/api/v1/tenants/assets`                          | 读取租户 Logo/背景图文件流     |
| `GET`    | `/api/v1/tenants/current/dashboard-notes`         | 获取当前租户仪表盘共享便签列表 |
| `POST`   | `/api/v1/tenants/current/dashboard-notes`         | 新增当前租户仪表盘共享便签     |
| `PUT`    | `/api/v1/tenants/current/dashboard-notes/:noteId` | 更新当前租户仪表盘共享便签     |
| `DELETE` | `/api/v1/tenants/current/dashboard-notes/:noteId` | 删除当前租户仪表盘共享便签     |
| `GET`    | `/api/v1/tenants`                                 | 获取租户列表                   |
| `POST`   | `/api/v1/tenants`                                 | 创建租户及默认管理员           |
| `PUT`    | `/api/v1/tenants/:id`                             | 更新租户信息                   |
| `DELETE` | `/api/v1/tenants/:id`                             | 删除租户                       |
| `POST`   | `/api/v1/tenants/:id/upload`                      | 上传租户 Logo/背景图           |

#### 当前租户仪表盘共享便签

`GET /api/v1/tenants/current/dashboard-notes`

- 鉴权：必须登录。
- 租户范围：只读取当前登录用户所属租户。
- 成功响应 `data.notes`：

```json
[
  {
    "id": "b7e2aaf8-67ff-42df-90f7-57cc66fcb2f8",
    "content": "便签正文",
    "createdAt": "2026-04-24T12:00:00.000Z",
    "createdBy": "550e8400-e29b-41d4-a716-446655440002",
    "createdByName": "系统管理员",
    "updatedAt": "2026-04-24T12:00:00.000Z",
    "updatedBy": "550e8400-e29b-41d4-a716-446655440002",
    "updatedByName": "系统管理员"
  }
]
```

`POST /api/v1/tenants/current/dashboard-notes`

- 鉴权：必须登录。
- 租户范围：只写入当前登录用户所属租户。
- 请求体：

```json
{
  "content": "便签正文，最多 2000 个字符"
}
```

- 成功响应 `data.note` 返回新增便签。

`PUT /api/v1/tenants/current/dashboard-notes/:noteId`

- 鉴权：必须登录。
- 租户范围：只更新当前登录用户所属租户下指定便签。
- 请求体：

```json
{
  "content": "便签正文，最多 2000 个字符"
}
```

- 成功响应 `data.note` 返回更新后的便签。
- 写入语义：只覆盖指定单条便签内容，后提交内容覆盖前一版本。

`DELETE /api/v1/tenants/current/dashboard-notes/:noteId`

- 鉴权：必须登录。
- 租户范围：只删除当前登录用户所属租户下指定便签。
- 成功响应 `data.deletedId` 返回被删除的便签 ID。

典型业务失败：

- `20001` 业务参数非法
- `11004` 租户不匹配或缺少当前租户上下文
- `22001` 租户不存在
- `22002` 租户编码重复

典型技术异常：

- `404/21001` 资产文件不存在
- `500/30003` 对象存储异常

### 4.4 用户模块 `/api/v1/users`

| 方法     | 路径                         | 功能概要               |
| -------- | ---------------------------- | ---------------------- |
| `GET`    | `/api/v1/users`              | 获取用户列表           |
| `POST`   | `/api/v1/users`              | 创建用户               |
| `PUT`    | `/api/v1/users/:id`          | 更新用户信息           |
| `PUT`    | `/api/v1/users/:id/password` | 重置或修改指定用户密码 |
| `DELETE` | `/api/v1/users/:id`          | 删除用户               |

典型业务失败：

- `23001` 用户不存在
- `23002` 用户名重复
- `21003` 禁止删除自己
- `21004` 禁止删除超级管理员

### 4.5 角色模块 `/api/v1/roles`

| 方法     | 路径                      | 功能概要                     |
| -------- | ------------------------- | ---------------------------- |
| `GET`    | `/api/v1/roles`           | 获取角色能力清单             |
| `POST`   | `/api/v1/roles`           | 占位接口，统一返回方法不允许 |
| `PUT`    | `/api/v1/roles`           | 占位接口，统一返回方法不允许 |
| `DELETE` | `/api/v1/roles`           | 占位接口，统一返回方法不允许 |
| `GET`    | `/api/v1/roles/me`        | 获取当前用户角色与能力       |
| `PUT`    | `/api/v1/roles/users/:id` | 更新指定用户角色             |

### 4.6 工程模块 `/api/v1/projects`

| 方法     | 路径                                                               | 功能概要                            |
| -------- | ------------------------------------------------------------------ | ----------------------------------- |
| `GET`    | `/api/v1/projects`                                                 | 获取工程列表                        |
| `GET`    | `/api/v1/projects/:id/export`                                      | 导出工程及 Designer/DataCenter 数据 |
| `POST`   | `/api/v1/projects/import`                                          | 导入工程包                          |
| `POST`   | `/api/v1/projects`                                                 | 创建工程                            |
| `PUT`    | `/api/v1/projects/:id`                                             | 更新工程                            |
| `DELETE` | `/api/v1/projects/:id`                                             | 删除工程                            |
| `GET`    | `/api/v1/projects/:id/delete-impact`                               | 获取删除影响评估                    |
| `POST`   | `/api/v1/projects/:id/operations/:operation`                       | 执行工程运维操作                    |
| `GET`    | `/api/v1/projects/:id/runtime-users`                               | 获取运行态用户列表                  |
| `POST`   | `/api/v1/projects/:id/runtime-users`                               | 创建运行态用户                      |
| `PATCH`  | `/api/v1/projects/:id/runtime-users/:runtimeUserId/status`         | 更新运行态用户状态                  |
| `POST`   | `/api/v1/projects/:id/runtime-users/:runtimeUserId/reset-password` | 重置运行态用户密码                  |
| `PUT`    | `/api/v1/projects/:id/runtime-users/:runtimeUserId/roles`          | 绑定运行态用户角色                  |
| `GET`    | `/api/v1/projects/:id/runtime-roles`                               | 获取运行态角色列表                  |
| `POST`   | `/api/v1/projects/:id/runtime-roles`                               | 创建运行态角色                      |
| `PUT`    | `/api/v1/projects/:id/runtime-roles/:roleId`                       | 更新运行态角色                      |
| `DELETE` | `/api/v1/projects/:id/runtime-roles/:roleId`                       | 删除运行态角色                      |

典型业务失败：

- `24001` 工程不存在
- `24003` 工程创建失败
- `24006` 工程运维操作失败

### 4.7 设计页面模块 `/api/v1/design`

| 方法     | 路径                                                       | 功能概要       |
| -------- | ---------------------------------------------------------- | -------------- |
| `GET`    | `/api/v1/design/projects/:projectId/pages`                 | 获取项目页面树 |
| `GET`    | `/api/v1/design/projects/:projectId/pages/:pageId`         | 获取页面详情   |
| `POST`   | `/api/v1/design/projects/:projectId/pages`                 | 创建页面或目录 |
| `PUT`    | `/api/v1/design/projects/:projectId/pages/:pageId`         | 更新页面信息   |
| `DELETE` | `/api/v1/design/projects/:projectId/pages/:pageId`         | 删除页面或目录 |
| `PATCH`  | `/api/v1/design/projects/:projectId/pages/:pageId/content` | 更新页面内容   |
| `PATCH`  | `/api/v1/design/projects/:projectId/pages/order`           | 调整页面顺序   |
| `GET`    | `/api/v1/design/projects/:projectId/entry`                 | 获取入口配置   |
| `PUT`    | `/api/v1/design/projects/:projectId/entry`                 | 更新入口配置   |
| `GET`    | `/api/v1/design/projects/:projectId/settings`              | 获取设计设置   |
| `PUT`    | `/api/v1/design/projects/:projectId/settings`              | 更新设计设置   |

### 4.8 设计资源模块 `/api/v1/design`

| 方法     | 路径                                                      | 功能概要           |
| -------- | --------------------------------------------------------- | ------------------ |
| `GET`    | `/api/v1/design/projects/:projectId/assets`               | 获取资源列表       |
| `GET`    | `/api/v1/design/projects/:projectId/assets/:assetId`      | 获取资源详情       |
| `POST`   | `/api/v1/design/projects/:projectId/assets`               | 上传资源           |
| `PATCH`  | `/api/v1/design/projects/:projectId/assets/:assetId`      | 更新资源元信息     |
| `DELETE` | `/api/v1/design/projects/:projectId/assets/:assetId`      | 删除资源           |
| `GET`    | `/api/v1/design/projects/:projectId/assets/:assetId/file` | 下载或预览资源文件 |
| `POST`   | `/api/v1/design/projects/:projectId/folders`              | 创建资源目录       |
| `PATCH`  | `/api/v1/design/projects/:projectId/folders/:folderId`    | 更新目录           |
| `POST`   | `/api/v1/design/projects/:projectId/assets/move`          | 批量移动资源       |
| `DELETE` | `/api/v1/design/projects/:projectId/folders/:folderId`    | 删除资源目录       |

### 4.9 页面锁模块 `/api/v1/pages`

| 方法     | 路径                                   | 功能概要              |
| -------- | -------------------------------------- | --------------------- |
| `GET`    | `/api/v1/pages/:pageId/lock`           | 查询页面锁状态        |
| `POST`   | `/api/v1/pages/:pageId/lock`           | 申请页面锁            |
| `DELETE` | `/api/v1/pages/:pageId/lock`           | 释放页面锁            |
| `POST`   | `/api/v1/pages/:pageId/lock/heartbeat` | 续约页面锁            |
| `POST`   | `/api/v1/pages/:pageId/lock/release`   | Beacon 方式释放页面锁 |
| `DELETE` | `/api/v1/pages/:pageId/lock/force`     | 强制释放页面锁        |

### 4.10 节点与注册模块

#### `/api/v1/node-register`

| 方法   | 路径                                            | 功能概要         |
| ------ | ----------------------------------------------- | ---------------- |
| `POST` | `/api/v1/node-register/register-with-auth`      | 节点带鉴权注册   |
| `GET`  | `/api/v1/node-register/:nodeId/approval-status` | 查询节点审批状态 |

#### `/api/v1/nodes`

| 方法     | 路径                                      | 功能概要     |
| -------- | ----------------------------------------- | ------------ |
| `POST`   | `/api/v1/nodes/:nodeId/heartbeat`         | 节点心跳     |
| `POST`   | `/api/v1/nodes/:nodeId/deployment-status` | 上报部署状态 |
| `POST`   | `/api/v1/nodes/:nodeId/offline`           | 节点离线     |
| `GET`    | `/api/v1/nodes`                           | 节点列表     |
| `GET`    | `/api/v1/nodes/:nodeId`                   | 节点详情     |
| `PUT`    | `/api/v1/nodes/:nodeId`                   | 更新节点     |
| `DELETE` | `/api/v1/nodes/:nodeId`                   | 删除节点     |
| `PUT`    | `/api/v1/nodes/:nodeId/approve`           | 审批节点     |
| `PUT`    | `/api/v1/nodes/:nodeId/reject`            | 拒绝节点     |

### 4.11 部署模块 `/api/v1/deployments`

| 方法     | 路径                                                  | 功能概要           |
| -------- | ----------------------------------------------------- | ------------------ |
| `GET`    | `/api/v1/deployments/project/:projectId`              | 获取项目部署记录   |
| `GET`    | `/api/v1/deployments/:id`                             | 获取部署详情       |
| `POST`   | `/api/v1/deployments/project/:projectId/deploy-dev`   | DEV 模式部署       |
| `POST`   | `/api/v1/deployments/:id/deploy`                      | RELEASE 部署       |
| `POST`   | `/api/v1/deployments/node-deployment/:id/start`       | 启动实例           |
| `POST`   | `/api/v1/deployments/node-deployment/:id/stop`        | 停止实例           |
| `POST`   | `/api/v1/deployments/node-deployment/:id/restart`     | 重启实例           |
| `DELETE` | `/api/v1/deployments/node-deployment/:id`             | 撤销部署           |
| `GET`    | `/api/v1/deployments/project/:projectId/runtime-mode` | 查询运行模式       |
| `POST`   | `/api/v1/deployments/:id/rollback`                    | 回滚部署           |
| `GET`    | `/api/v1/deployments/project/:projectId/nodes`        | 获取可部署节点列表 |
| `GET`    | `/api/v1/deployments/node/:nodeId/history`            | 获取节点部署历史   |

### 4.12 发布模块 `/api/v1/publish`

| 方法     | 路径                                      | 功能概要     |
| -------- | ----------------------------------------- | ------------ |
| `POST`   | `/api/v1/publish/:projectId`              | 创建发布版本 |
| `GET`    | `/api/v1/publish/:projectId/versions`     | 获取版本列表 |
| `GET`    | `/api/v1/publish/deployment/:id`          | 获取发布详情 |
| `GET`    | `/api/v1/publish/deployment/:id/download` | 下载发布包   |
| `DELETE` | `/api/v1/publish/deployment/:id`          | 删除发布版本 |

### 4.13 日志模块 `/api/v1/logs`

| 方法  | 路径                             | 功能概要         |
| ----- | -------------------------------- | ---------------- |
| `GET` | `/api/v1/logs`                   | 获取系统日志列表 |
| `GET` | `/api/v1/logs/stats`             | 获取日志统计     |
| `GET` | `/api/v1/logs/recent-activities` | 获取近期活动     |

#### 仪表盘最近活动

`GET /api/v1/logs/recent-activities?limit=5`

- 鉴权：必须登录，不要求系统日志管理权限。
- 租户范围：只读取当前登录用户所属租户。
- 只返回执行成功的业务写操作，包括新增、更新、删除以及启停、审批、发布部署等动作。
- 不返回查询请求、登录刷新等认证请求或系统日志清理操作；完整请求审计仍通过系统日志接口查询。
- 成功响应 `data.activities`，单项包含 `id`、`action`、`resource`、`path`、`createdAt` 和 `user`。

## 5. data_service 接口清单

### 5.1 模块功能概要

`data_service` 是数据域服务，负责：

- 数据连接管理
- 查询与数据点能力
- MQTT 接入与消息/Tag 管理
- 协议接入（Kafka/HTTP/WebSocket/Redis 等）
- 预览会话与项目快照
- 计算单元运行

### 5.2 健康检查

| 方法  | 路径      | 功能概要           |
| ----- | --------- | ------------------ |
| `GET` | `/health` | 数据域服务健康检查 |

### 5.3 连接管理 `/api/v1/data/projects/{projectId}/connections`

#### 接入源 `/api/v1/data/projects/{projectId}/access-sources`

| 方法  | 路径                                                                  | 功能概要           |
| ----- | --------------------------------------------------------------------- | ------------------ |
| `GET` | `/api/v1/data/projects/{projectId}/access-sources`                    | 获取统一接入源列表 |
| `GET` | `/api/v1/data/projects/{projectId}/access-sources/{sourceId}`         | 获取接入源详情     |
| `GET` | `/api/v1/data/projects/{projectId}/access-sources/{sourceId}/records` | 获取接入源最近记录 |

#### 原始连接 `/api/v1/data/projects/{projectId}/connections`

| 方法     | 路径                                                                                        | 功能概要     |
| -------- | ------------------------------------------------------------------------------------------- | ------------ |
| `GET`    | `/api/v1/data/projects/{projectId}/connections`                                             | 获取连接列表 |
| `POST`   | `/api/v1/data/projects/{projectId}/connections`                                             | 创建连接     |
| `PUT`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}`                              | 更新连接     |
| `DELETE` | `/api/v1/data/projects/{projectId}/connections/{connectionId}`                              | 删除连接     |
| `POST`   | `/api/v1/data/projects/{projectId}/connections/test`                                        | 测试连接     |
| `PATCH`  | `/api/v1/data/projects/{projectId}/connections/{connectionId}/status`                       | 更新连接状态 |
| `GET`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}/tables`                       | 获取表列表   |
| `GET`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}/structure` | 获取表结构   |
| `GET`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}/data`      | 获取表数据   |
| `POST`   | `/api/v1/data/projects/{projectId}/connections/{connectionId}/execute-sql`                  | 执行 SQL     |

### 5.4 查询与数据点

#### 查询 `/api/v1/data`

| 方法     | 路径                                        | 功能概要     |
| -------- | ------------------------------------------- | ------------ |
| `GET`    | `/api/v1/data/projects/{projectId}/queries` | 获取查询列表 |
| `POST`   | `/api/v1/data/projects/{projectId}/queries` | 创建查询     |
| `POST`   | `/api/v1/data/queries/{id}/execute`         | 执行查询     |
| `PUT`    | `/api/v1/data/queries/{id}`                 | 更新查询     |
| `DELETE` | `/api/v1/data/queries/{id}`                 | 删除查询     |

#### 数据点 `/api/v1/data`

| 方法     | 路径                                                                    | 功能概要             |
| -------- | ----------------------------------------------------------------------- | -------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints`                          | 获取数据点列表       |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints/{id}`                     | 获取数据点详情       |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints/value`                    | 按 path 获取数据点值 |
| `PUT`    | `/api/v1/data/projects/{projectId}/datapoints/{id}`                     | 更新数据点           |
| `DELETE` | `/api/v1/data/projects/{projectId}/datapoints/{id}`                     | 删除数据点           |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/delete-batch`             | 批量删除数据点       |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/status`                   | 批量查询数据点状态   |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/values`                   | 批量查询数据点值     |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/{id}/write`               | 写入数据点值         |
| `PUT`    | `/api/v1/data/projects/{projectId}/datapoints/{id}/runtime-permissions` | 更新运行态权限       |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints/{id}/usages`              | 获取数据点引用关系   |

### 5.5 MQTT 管理

| 方法     | 路径                                                                               | 功能概要              |
| -------- | ---------------------------------------------------------------------------------- | --------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/connections`                               | 获取 MQTT 连接列表    |
| `POST`   | `/api/v1/data/projects/{projectId}/mqtt/connections`                               | 创建 MQTT 连接        |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}`                | 获取 MQTT 连接详情    |
| `PUT`    | `/api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}`                | 更新 MQTT 连接        |
| `DELETE` | `/api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}`                | 删除 MQTT 连接        |
| `POST`   | `/api/v1/data/projects/{projectId}/mqtt/connections/test`                          | 测试 MQTT 连接配置    |
| `POST`   | `/api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/start`          | 启动 MQTT 连接        |
| `POST`   | `/api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/stop`           | 停止 MQTT 连接        |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/status`         | 获取 MQTT 连接状态    |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/subscriptions`  | 获取订阅列表          |
| `POST`   | `/api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/subscriptions`  | 创建订阅              |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}`            | 获取订阅详情          |
| `PUT`    | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}`            | 更新订阅              |
| `DELETE` | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}`            | 删除订阅              |
| `PATCH`  | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/toggle`     | 启停订阅              |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/messages`   | 获取消息列表          |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tag-groups` | 获取 Tag 组           |
| `POST`   | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tag-groups` | 创建 Tag 组           |
| `PUT`    | `/api/v1/data/projects/{projectId}/mqtt/tag-groups/order`                          | 调整 Tag 组排序       |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/tag-groups/{groupId}`                      | 获取 Tag 组详情       |
| `PUT`    | `/api/v1/data/projects/{projectId}/mqtt/tag-groups/{groupId}`                      | 更新 Tag 组           |
| `DELETE` | `/api/v1/data/projects/{projectId}/mqtt/tag-groups/{groupId}`                      | 删除 Tag 组           |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tags`       | 获取订阅下的 Tag 列表 |
| `POST`   | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tags`       | 创建 Tag              |
| `POST`   | `/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/tags/batch` | 批量创建 Tag          |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/tags`                                      | 获取项目级 Tag 列表   |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/tags/{tagId}`                              | 获取 Tag 详情         |
| `PUT`    | `/api/v1/data/projects/{projectId}/mqtt/tags/{tagId}`                              | 更新 Tag              |
| `DELETE` | `/api/v1/data/projects/{projectId}/mqtt/tags/{tagId}`                              | 删除 Tag              |
| `PATCH`  | `/api/v1/data/projects/{projectId}/mqtt/tags/{tagId}/toggle`                       | 启停 Tag              |
| `PUT`    | `/api/v1/data/projects/{projectId}/mqtt/tags/order`                                | 调整 Tag 排序         |
| `GET`    | `/api/v1/data/projects/{projectId}/mqtt/tags/{tagId}/value`                        | 获取 Tag 当前值       |
| `POST`   | `/api/v1/data/projects/{projectId}/mqtt/tags/values`                               | 批量获取 Tag 值       |

### 5.6 项目快照与工件

| 方法  | 路径                                         | 功能概要       |
| ----- | -------------------------------------------- | -------------- |
| `GET` | `/api/v1/data/projects/{projectId}/snapshot` | 获取项目快照   |
| `PUT` | `/api/v1/data/projects/{projectId}/snapshot` | 替换项目快照   |
| `GET` | `/api/v1/data/projects/{projectId}/artifact` | 获取数据域工件 |

### 5.7 协议接入

#### 协议 Wave 1

| 方法   | 路径                                                                     | 功能概要            |
| ------ | ------------------------------------------------------------------------ | ------------------- |
| `POST` | `/api/v1/data/projects/{projectId}/kafka/configs`                        | 创建 Kafka 配置     |
| `GET`  | `/api/v1/data/projects/{projectId}/kafka/configs/{connectionId}/preview` | 预览 Kafka Topic    |
| `POST` | `/api/v1/data/projects/{projectId}/http/configs`                         | 创建 HTTP 配置      |
| `POST` | `/api/v1/data/projects/{projectId}/websocket/configs`                    | 创建 WebSocket 配置 |
| `POST` | `/api/v1/data/projects/{projectId}/redis/configs`                        | 创建 Redis 配置     |
| `POST` | `/api/v1/data/projects/{projectId}/protocols/{connectionId}/preview`     | 统一短时真实抓样    |

Kafka 工作台接口：

| 方法     | 路径                                                                              | 功能概要                |
| -------- | --------------------------------------------------------------------------------- | ----------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/kafka/sources/{connectionId}/topic-groups`     | 获取 Kafka Topic 分组   |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/sources/{connectionId}/topic-groups`     | 创建 Kafka Topic 分组   |
| `PUT`    | `/api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}`                  | 更新 Kafka Topic 分组   |
| `DELETE` | `/api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}`                  | 删除 Kafka Topic 分组   |
| `GET`    | `/api/v1/data/projects/{projectId}/kafka/sources/{connectionId}/topic-mappings`   | 获取 Kafka Topic 映射   |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/sources/{connectionId}/topic-mappings`   | 创建 Kafka Topic 映射   |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/sources/{connectionId}/preview`          | 短时预览 Kafka 接入源   |
| `PUT`    | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}`              | 更新 Kafka Topic 映射   |
| `DELETE` | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}`              | 删除 Kafka Topic 映射   |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/preview`      | 短时预览 Kafka Topic    |
| `GET`    | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields`       | 获取 Kafka 字段映射     |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields`       | 创建 Kafka 字段映射     |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields/batch` | 批量创建 Kafka 字段映射 |
| `PUT`    | `/api/v1/data/projects/{projectId}/kafka/fields/{fieldId}`                        | 更新 Kafka 字段映射     |
| `DELETE` | `/api/v1/data/projects/{projectId}/kafka/fields/{fieldId}`                        | 删除 Kafka 字段映射     |
| `PATCH`  | `/api/v1/data/projects/{projectId}/kafka/fields/{fieldId}/toggle`                 | 启停 Kafka 字段映射     |

统一协议 preview 请求体：

```json
{
  "limit": 10,
  "timeoutMs": 5000,
  "options": {}
}
```

统一协议 preview 响应 `data` 字段：

```json
{
  "protocol": "http",
  "connectionId": "uuid",
  "status": "ok",
  "schema": {},
  "samples": [],
  "rawPayload": "",
  "diagnostics": {},
  "durationMs": 12,
  "truncated": false
}
```

`limit` 默认 10、最大 100；`timeoutMs` 默认 5000、最大 30000。服务端只持久化接入源预览摘要，不保存 `samples/rawPayload` 原始数据。

#### 协议 Wave 2

| 方法   | 路径                                                         | 功能概要           |
| ------ | ------------------------------------------------------------ | ------------------ |
| `POST` | `/api/v1/data/projects/{projectId}/opcua/configs`            | 创建 OPC UA 配置   |
| `POST` | `/api/v1/data/projects/{projectId}/s7/configs`               | 创建 S7 配置       |
| `POST` | `/api/v1/data/projects/{projectId}/modbus/configs`           | 创建 Modbus 配置   |
| `POST` | `/api/v1/data/projects/{projectId}/tdengine/configs`         | 创建 TDengine 配置 |
| `POST` | `/api/v1/data/projects/{projectId}/opcda/contracts/validate` | 校验 OPC DA 合同   |

Wave 2 当前只交付平台侧配置和 artifact 契约：`opcua/s7/modbus/tdengine` 保存连接配置，`opcda` 校验合约字段；工业协议真实连接、轮询采集和运行态诊断由节点侧运行器负责。

### 5.8 预览会话与计算

#### 预览

| 方法       | 路径                                                    | 功能概要             |
| ---------- | ------------------------------------------------------- | -------------------- |
| `POST`     | `/api/v1/data/projects/{projectId}/preview/sessions`    | 创建预览会话         |
| `GET`      | `/api/v1/data/projects/{projectId}/preview/diagnostics` | 获取预览链路诊断信息 |
| `POST`     | `/api/v1/data/preview/sessions/{sessionId}/heartbeat`   | 预览会话心跳         |
| `DELETE`   | `/api/v1/data/preview/sessions/{sessionId}`             | 关闭预览会话         |
| `GET/POST` | `/socket.io`                                            | 预览 Socket 通道     |

#### 计算单元

| 方法     | 路径                                                           | 功能概要                     |
| -------- | -------------------------------------------------------------- | ---------------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/compute-units`              | 查询计算单元列表             |
| `POST`   | `/api/v1/data/projects/{projectId}/compute-units`              | 创建计算单元                 |
| `GET`    | `/api/v1/data/projects/{projectId}/compute-units/{id}`         | 查询计算单元详情             |
| `PUT`    | `/api/v1/data/projects/{projectId}/compute-units/{id}`         | 更新计算单元并同步输出数据点 |
| `DELETE` | `/api/v1/data/projects/{projectId}/compute-units/{id}`         | 删除计算单元并失效输出数据点 |
| `PATCH`  | `/api/v1/data/projects/{projectId}/compute-units/{id}/enabled` | 切换计算单元启用状态         |
| `GET`    | `/api/v1/data/projects/{projectId}/compute-units/{id}/runs`    | 查询计算单元运行记录         |
| `POST`   | `/api/v1/data/projects/{projectId}/compute-units/{id}/run`     | 运行计算单元                 |
| `POST`   | `/api/v1/data/projects/{projectId}/compute-units/{id}/debug`   | 调试计算单元                 |

计算脚本支持开发态 `ctx` SDK。后端会根据 `inputBindings` 预取已声明的数据点和 SQL 查询结果，脚本内可使用 `ctx.datapoint.get(path)`、`ctx.datapoint.meta(path)`、`ctx.sql.query(key)`；未命中预取结果时，`ctx.sql.query(key, params)` 可按已声明别名或项目内查询 ID 动态回调后端执行，JavaScript 脚本需要 `await ctx.sql.query(...)`。`ctx.mqtt.publish(source, topic, payload)` 仅在计算单元显式开启 `sideEffects.mqttPublish.enabled=true` 且命中 `sources/sourceIds` 与 `topics` 白名单时真实发布，否则只返回未接受的 side effect。

#### 报警规则

| 方法     | 路径                                                                 | 功能概要               |
| -------- | -------------------------------------------------------------------- | ---------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-rules`                      | 查询报警规则列表       |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-rules`                      | 创建报警规则           |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-rules/{id}`                 | 查询报警规则详情       |
| `PUT`    | `/api/v1/data/projects/{projectId}/alarm-rules/{id}`                 | 更新报警规则           |
| `DELETE` | `/api/v1/data/projects/{projectId}/alarm-rules/{id}`                 | 删除报警规则           |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-rules/{id}/validate-target` | 校验规则目标数据点     |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-rules/{id}/test`            | 使用样本值试算报警规则 |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-rules/{id}/contract`        | 预览运行态报警契约     |

报警规则支持 `threshold`、`range`、`expression`。`expression` 当前为开发态受控子集，支持 `value > 10`、`value <= 20`、`value == "ON"` 以及 `&&` / `||` 组合。

#### 契约检查

| 方法   | 路径                                                       | 功能概要                   |
| ------ | ---------------------------------------------------------- | -------------------------- |
| `POST` | `/api/v1/data/projects/{projectId}/contract-checks/run`    | 实时执行项目数据域契约检查 |
| `GET`  | `/api/v1/data/projects/{projectId}/contract-checks/latest` | 获取最近/实时契约检查结果  |
| `GET`  | `/api/v1/data/projects/{projectId}/contract-checks/runs`   | 分页获取契约检查历史记录   |

项目 artifact v1 已包含连接、查询、数据点、MQTT、Phase 1 协议、计算单元和报警规则区块；snapshot replace 会同步写回计算单元和报警规则。
契约检查 `run` 请求体可传 `scope`、`objectType`、`objectId`，用于收窄检查结果范围；检查项覆盖数据点状态、计算输入/输出、计算 SQL 绑定查询、报警目标数据点。每次 `run` 会写入 `data_contract_check_runs`，`latest` 优先返回最近一次历史结果，`runs` 使用 `data.list` 与 `data.pagination` 返回历史分页。

## 6. 开发约束

后续新增接口时，必须满足：

- JSON 接口统一返回 `code/msg/data/reqId`
- 业务失败统一 `HTTP 2xx + code!=0`
- 技术异常统一 `HTTP 4xx/5xx + code!=0`
- 文件流接口成功时可直接返回流，但错误必须回到统一包络
- 错误码必须引用 [统一错误码枚举表](./统一错误码枚举表.md)

## 7. 联调建议

前端联调时建议统一按以下顺序处理：

1. 先判断是否命中 `4xx/5xx`
2. 若是 `2xx`，再判断 `code === 0`
3. 若 `code !== 0`，直接使用 `msg` 作为业务提示，并可按 `code` 做细分处理
4. 日志、埋点、排障统一记录 `reqId`
