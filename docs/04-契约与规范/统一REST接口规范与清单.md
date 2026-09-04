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
- 工程开发工作空间、上下文和发布资源管理
- 节点注册、部署、发布、日志与运维聚合
- 运行态账号与角色的项目侧治理

### 4.2 认证模块 `/api/v1/auth`

| 方法   | 路径                    | 功能概要                        |
| ------ | ----------------------- | ------------------------------- |
| `GET`  | `/api/v1/auth/captcha`  | 获取登录滑块挑战（拖至最右侧）  |
| `POST` | `/api/v1/auth/login`    | 登录并设置 HttpOnly 会话 Cookie |
| `POST` | `/api/v1/auth/refresh`  | 使用刷新 Cookie 轮换会话        |
| `POST` | `/api/v1/auth/logout`   | 注销、撤销会话并清除 Cookie     |
| `PUT`  | `/api/v1/auth/password` | 当前用户修改密码                |
| `GET`  | `/api/v1/auth/me`       | 获取当前用户信息                |
| `GET`  | `/api/v1/auth/config`   | 获取登录页和应用配置            |

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

| 方法     | 路径                                                               | 功能概要                         |
| -------- | ------------------------------------------------------------------ | -------------------------------- |
| `GET`    | `/api/v1/projects`                                                 | 获取工程列表                     |
| `GET`    | `/api/v1/projects/:id/export`                                      | 导出工程源码、场景与数据中心数据 |
| `POST`   | `/api/v1/projects/import`                                          | 导入工程包                       |
| `POST`   | `/api/v1/projects`                                                 | 创建工程                         |
| `PUT`    | `/api/v1/projects/:id`                                             | 更新工程                         |
| `DELETE` | `/api/v1/projects/:id`                                             | 删除工程                         |
| `GET`    | `/api/v1/projects/:id/delete-impact`                               | 获取删除影响评估                 |
| `POST`   | `/api/v1/projects/:id/operations/:operation`                       | 执行工程运维操作                 |
| `GET`    | `/api/v1/projects/:id/runtime-users`                               | 获取运行态用户列表               |
| `POST`   | `/api/v1/projects/:id/runtime-users`                               | 创建运行态用户                   |
| `PATCH`  | `/api/v1/projects/:id/runtime-users/:runtimeUserId/status`         | 更新运行态用户状态               |
| `POST`   | `/api/v1/projects/:id/runtime-users/:runtimeUserId/reset-password` | 重置运行态用户密码               |
| `PUT`    | `/api/v1/projects/:id/runtime-users/:runtimeUserId/roles`          | 绑定运行态用户角色               |
| `GET`    | `/api/v1/projects/:id/runtime-roles`                               | 获取运行态角色列表               |
| `POST`   | `/api/v1/projects/:id/runtime-roles`                               | 创建运行态角色                   |
| `PUT`    | `/api/v1/projects/:id/runtime-roles/:roleId`                       | 更新运行态角色                   |
| `DELETE` | `/api/v1/projects/:id/runtime-roles/:roleId`                       | 删除运行态角色                   |

典型业务失败：

- `24001` 工程不存在
- `24003` 工程创建失败
- `24006` 工程运维操作失败

### 4.6.1 运维节点与运行环境 `/api/v1/ops`

运维管理员负责节点接入和运行环境管理；工程管理员只能读取可部署的运行环境，不能新增环境或调整节点归属。运行环境是用户可见的工程运行边界，接口不暴露 K3s、Pod、容器或资源池概念。

| 方法     | 路径                                                        | 功能概要                     |
| -------- | ----------------------------------------------------------- | ---------------------------- |
| `GET`    | `/api/v1/ops/node-enrollments`                              | 分页查询节点接入申请         |
| `POST`   | `/api/v1/ops/node-enrollments`                              | 创建一次性节点接入码         |
| `GET`    | `/api/v1/ops/nodes`                                         | 分页查询物理节点及环境归属   |
| `GET`    | `/api/v1/ops/runtime-environments`                          | 分页查询运行环境             |
| `POST`   | `/api/v1/ops/runtime-environments`                          | 仅按名称创建运行环境         |
| `GET`    | `/api/v1/ops/runtime-environments/:id`                      | 查询环境聚合状态             |
| `GET`    | `/api/v1/ops/runtime-environments/:id/nodes`                | 分页查询环境内物理节点       |
| `POST`   | `/api/v1/ops/runtime-environments/:id/nodes`                | 将已批准 Linux 节点加入环境  |
| `DELETE` | `/api/v1/ops/runtime-environments/:id/nodes/:nodeId`        | 从环境移除未承载服务的节点   |
| `GET`    | `/api/v1/ops/runtime-environments/:id/events`               | 分页查询环境运维事件         |

运行环境标识由服务端生成。一个物理节点最多属于一个运行环境；Windows 采集节点不加入运行环境。环境状态由节点心跳和基础服务观测派生，不能由前端写入或伪造。未部署基础服务时固定为 `uninitialized`，存在离线节点或不健康服务时为 `attention`，全部节点与基础服务健康后才为 `available`。

### 4.7 2D/3D 场景与资源库

场景与资源接口统一使用短期编辑会话。除缩略图、文件内容和 ZIP 流外，响应均使用
`code/msg/data/reqId` 包络；浏览器不得直接访问 MinIO 或传入 Provider URL。对外 Provider
标识固定为 `induforge`，内部 Provider 类型及版本仅在服务端会话、存储和构建流程中使用。
普通场景、资源和 revision 响应不返回内部 Provider、Provider 版本、入口路径或内容摘要。

| 方法             | 路径                                                                 | 功能概要                                             |
| ---------------- | -------------------------------------------------------------------- | ---------------------------------------------------- |
| `GET/PUT`        | `/api/v1/scene-provider`                                             | 查询或切换平台唯一 Provider；存在场景时禁止切换      |
| `GET/POST`       | `/api/v1/projects/{projectId}/scenes`                                | 分页查询或按唯一名称创建场景，`sceneId` 由平台生成   |
| `GET/PUT/DELETE` | `/api/v1/projects/{projectId}/scenes/{sceneId}`                      | 场景详情、公开契约与墓碑删除                         |
| `POST`           | `/api/v1/projects/{projectId}/scenes/{sceneId}/editor-session`       | 创建绑定用户、工程、场景、类型和 Provider 的短期会话 |
| `POST`           | `/api/v1/projects/{projectId}/scenes/{sceneId}/viewer-session`       | 创建固定到当前已提交 revision 的只读 Viewer 会话     |
| `POST`           | `/api/v1/projects/{projectId}/scenes/{sceneId}/commit`               | 将指定草稿版本提交为不可变 revision                  |
| `GET/POST`       | `/api/v1/scene-viewer-sessions/{sessionId}`、`/heartbeat`            | 读取 Viewer 启动信息或延长空闲有效期                 |
| `GET`            | `/api/v1/scene-viewer-sessions/{sessionId}/files/content`            | 读取 Viewer 会话固定 revision 的文件                 |
| `GET/PUT`        | `/api/v1/scene-editor-sessions/{sessionId}/files/content`            | 受控读取或保存 Provider 文件                         |
| `POST`           | `/api/v1/scene-editor-sessions/{sessionId}/import`                   | 原子导入场景 ZIP                                     |
| `POST`           | `/api/v1/scene-editor-sessions/{sessionId}/export`                   | 按当前场景依赖导出 ZIP                               |
| `GET`            | `/api/v1/scene-editor-sessions/{sessionId}/datapoints`               | 代理分页查询平台数据点                               |
| `GET`            | `/api/v1/projects/{projectId}/scene-assets`                          | 按分类、关键字、场景类型分页查询工程资源             |
| `POST`           | `/api/v1/projects/{projectId}/scene-assets/import`                   | 上传单文件或 ZIP 并创建资源首个内部代次              |
| `GET/PUT/DELETE` | `/api/v1/projects/{projectId}/scene-assets/{assetId}`                | 资源详情、重命名与归档                               |
| `POST`           | `/api/v1/projects/{projectId}/scene-assets/{assetId}/replace`        | 创建新的内部资源代次                                 |
| `GET`            | `/api/v1/projects/{projectId}/scene-assets/{assetId}/thumbnail`      | 平台代理资源缩略图                                   |
| `POST`           | `/api/v1/projects/{projectId}/scene-assets/{assetId}/editor-session` | 创建 Symbol/Component 工作副本会话                   |
| `GET`            | `/api/v1/scene-editor-sessions/{sessionId}/assets`                   | 场景工作室内分页查询兼容资源及绑定/更新状态          |
| `POST`           | `/api/v1/scene-editor-sessions/{sessionId}/assets/actions`           | 以乐观锁执行 `attach/update/detach`                  |
| `POST`           | `/api/v1/scene-editor-sessions/{sessionId}/assets/from-selection`    | 将画布选中内容保存为图形模板或业务组件               |
| `GET`            | `/api/v1/scene-editor-sessions/{sessionId}/dependencies`             | 只读分页查询场景依赖诊断                             |
| `GET/PUT`        | `/api/v1/scene-asset-editor-sessions/{sessionId}/files/content`      | 读取或保存资源工作副本                               |
| `POST`           | `/api/v1/scene-asset-editor-sessions/{sessionId}/commit`             | 发布资源工作副本为新内部代次                         |

读取资源要求 `project:read`；上传、替换、归档、编辑、挂载、更新和场景提交要求
`project:write`。上传单文件上限为 `64 MiB`；ZIP 请求上限为 `512 MiB`、最多 `10,000`
个条目，并拒绝绝对路径、路径逃逸、符号链接、重复逻辑路径和不完整依赖。

### 4.8 设计页面模块 `/api/v1/design`

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

### 4.9 设计资源模块 `/api/v1/design`

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

### 4.10 页面锁模块 `/api/v1/pages`

| 方法     | 路径                                   | 功能概要              |
| -------- | -------------------------------------- | --------------------- |
| `GET`    | `/api/v1/pages/:pageId/lock`           | 查询页面锁状态        |
| `POST`   | `/api/v1/pages/:pageId/lock`           | 申请页面锁            |
| `DELETE` | `/api/v1/pages/:pageId/lock`           | 释放页面锁            |
| `POST`   | `/api/v1/pages/:pageId/lock/heartbeat` | 续约页面锁            |
| `POST`   | `/api/v1/pages/:pageId/lock/release`   | Beacon 方式释放页面锁 |
| `DELETE` | `/api/v1/pages/:pageId/lock/force`     | 强制释放页面锁        |

### 4.11 节点与注册模块

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

### 4.12 部署模块 `/api/v1/deployments`

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

#### 正式运维工程部署 `/api/v1/ops/project-deployments`

工程列表 `GET /api/v1/projects` 的每个当前页项目包含权威 `deploymentSummary`：`deploymentCount`、`environmentCount`、`operationInProgress`、`updatedAt`、`primarySelection`、`primaryDeployment`。部署摘要使用当前页 ID 一次租户隔离批量查询，列表总查询数固定，不逐工程请求运维部署。仅且必须在未删除部署数为 1 时返回 primary；无部署为 `none/null`，多部署为 `multiple/null`，默认环境或最近更新时间均不能代表多部署工程整体。primary 的 mode 对外为 `development|production`，状态沿用部署的 desired/observed 状态；不新增 `deploying` 枚举。pending run、删除请求，或非失败/非异常部署中未失败运行引擎的期望代次大于观测代次时，表达操作进行中；`primaryDeployment.updating` 保留代次更新细分。失败/异常部署、失败服务和观测代次反向领先均不视为持续更新，不能阻止用户处理异常。placements/services 仅为现有槽位节点预填，当前制品所需引擎仍须使用开发态 requirements 或正式版本清单。不完整/失败的摘要不能伪装成零部署。

环境概览使用 `GET /api/v1/ops/runtime-environments/:id/overview` 获取全局统计与风险节点前五，口径见 [运行环境概览读模型契约](运行环境概览读模型契约.md)。不使用当前部署/节点分页推算全量指标。

统一记录入口为 `GET /api/v1/ops/records`，详细字段、权限、分页过滤和任务/事件状态边界见 [统一运维记录查询契约](统一运维记录查询契约.md)。它只读聚合现有来源，不创建或补写基础服务任务。

运维实时订阅与状态对账遵守 [统一运维实时协议](统一运维实时协议.md)。HTTP 查询保持权威快照，实时通知不替代租户与能力校验。

运行环境列表和详情中的 `projectCount` 是当前租户、当前环境下未删除部署的总量，包含停止、失败及执行中的部署；`runningDeploymentCount` 仅包含期望与观测均为 `running`、至少有一个运行引擎且所有引擎的状态和代次已收敛的部署。历史停用引擎须已停止并确认自身代次，不要求再次启动。概览“运行中”必须使用后者，不得用部署总量或占位数字替代。

| 方法 | 路径 | 功能概要 |
| ---- | ---- | -------- |
| `GET` | `/api/v1/ops/project-deployments` | 当前租户内分页查询；支持 `projectId`、`environmentId`、`search`，列表及 `total` 使用相同过滤条件 |
| `GET` | `/api/v1/ops/project-deployments/:id/runs` | 当前租户未删除部署的任务历史；`page`/`limit` 分页，默认 1/20，上限 200；按 `startedAt DESC,id DESC` |
| `GET` | `/api/v1/ops/deployment-runs/:id/events/page` | 当前租户未删除部署的指定任务事件；`page`/`limit` 同上；按 `createdAt ASC,id ASC`；旧 `/events` 不变 |
| `POST` | `/api/v1/ops/project-deployments/:id/start` | 整体启动当前制品需要的工程引擎 |
| `POST` | `/api/v1/ops/project-deployments/:id/stop` | 整体停止，不清理数据 |
| `POST` | `/api/v1/ops/project-deployments/:id/restart` | 整体重启当前制品需要的工程引擎 |
| `POST` | `/api/v1/ops/project-deployments/:id/redeploy` | 重新下发当前存储制品，不构建、不升级、不更换节点或端口、不清数据 |

新增任务历史及事件分页接口采用 `data.list`、`data.pagination {page,limit,total}`；权限与部署详情同为 `node:read`，租户从鉴权会话取得。历史项沿用 run 字段，增加 `actorDisplayName`（同租户 `full_name` 优先 `username`，无匹配为“未知用户”）及 `durationMs`（已完成时取 `completedAt-startedAt` 的非负毫秒数，未完成为 null）。不返回用户 ID、邮箱等用户详情。任务和事件查询均关联未删除部署，跨租户或不存在返回相同未找到错误。每接口固定三次 SQL，不逐任务读取用户或事件。

生命周期操作复用部署运维权限，返回 `{ deployment, run }` 于统一响应包络的 `data` 中。创建/升级与生命周期操作共享部署锁；有 `pending` 任务或正在删除时拒绝并发操作。`redeploy` 保留部署模式、固定版本及开发态 descriptor，内部 `run.operation` 沿用 `deploy`，`run.message` 和 `queued` 事件明确注明重新部署。历史版本曾需要而当前制品不再需要的引擎保持停止。结果通过 `/api/v1/ops/deployment-runs/:id` 及其 `/events` 查询；入队不代表完成。

### 4.13 发布模块 `/api/v1/publish`

| 方法     | 路径                                      | 功能概要     |
| -------- | ----------------------------------------- | ------------ |
| `POST`   | `/api/v1/publish/:projectId`              | 创建发布版本 |
| `GET`    | `/api/v1/publish/:projectId/versions`     | 获取版本列表 |
| `GET`    | `/api/v1/publish/deployment/:id`          | 获取发布详情 |
| `GET`    | `/api/v1/publish/deployment/:id/download` | 下载发布包   |
| `DELETE` | `/api/v1/publish/deployment/:id`          | 删除发布版本 |

### 4.14 日志模块 `/api/v1/logs`

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

`/connections` 是普通接入源唯一列表与详情读模型；工业采集连接和计算单元继续使用各自领域接口。

#### 连接 `/api/v1/data/projects/{projectId}/connections`

| 方法     | 路径                                                                                        | 功能概要                                                 |
| -------- | ------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/connections`                                             | 获取连接列表                                             |
| `GET`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}`                              | 获取连接详情                                             |
| `POST`   | `/api/v1/data/projects/{projectId}/connections`                                             | 创建连接                                                 |
| `PUT`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}`                              | 更新连接                                                 |
| `POST`   | `/api/v1/data/projects/{projectId}/connections/{connectionId}/secrets/reveal`               | 具备工程写权限的用户主动查看单个已保存密码；响应禁止缓存 |
| `GET`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}/delete-impact`                | 获取删除影响与阻断引用                                   |
| `DELETE` | `/api/v1/data/projects/{projectId}/connections/{connectionId}`                              | 删除连接                                                 |
| `POST`   | `/api/v1/data/projects/{projectId}/connections/test`                                        | 测试连接                                                 |
| `POST`   | `/api/v1/data/projects/{projectId}/connections/{connectionId}/test`                         | 测试已保存连接并记录脱敏摘要                             |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoint-source-options`                                | 分页搜索普通连接、工业连接和计算单元                     |
| `GET`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}/tables`                       | 获取表列表                                               |
| `GET`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}/structure` | 获取表结构                                               |
| `GET`    | `/api/v1/data/projects/{projectId}/connections/{connectionId}/tables/{tableName}/data`      | 获取表数据                                               |
| `POST`   | `/api/v1/data/projects/{projectId}/connections/{connectionId}/execute-sql`                  | 执行 SQL                                                 |

删除接入源必须先读取 `delete-impact`；存在报警、计算或历史目标引用时 `canDelete=false`，正式删除也会在事务内重新检查并拒绝竞态写入。无阻断时删除来源内配置，已生成数据点保留并标记为 `invalid`。

SQL 工作台和保存查询统一限制为 30 秒、500 行、5 MiB。达到行数或响应字节边界时返回 `truncated=true`、`truncatedBy=rows|bytes` 和 `limits`，调用方不得把截断数据解释为全量结果。

#### 工业采集连接

| 方法     | 路径                                                                                   | 功能概要                         |
| -------- | -------------------------------------------------------------------------------------- | -------------------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/collector/connections/{connectionId}/delete-impact` | 获取工业连接删除影响与阻断引用   |
| `DELETE` | `/api/v1/data/projects/{projectId}/collector/connections/{connectionId}`               | 事务删除工业连接并保留失效数据点 |

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

| 方法     | 路径                                                                    | 功能概要                                             |
| -------- | ----------------------------------------------------------------------- | ---------------------------------------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints`                          | 获取数据点列表                                       |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints/{id}`                     | 获取数据点详情                                       |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints/value`                    | 按 path 获取数据点值                                 |
| `PUT`    | `/api/v1/data/projects/{projectId}/datapoints/{id}`                     | 更新数据点                                           |
| `DELETE` | `/api/v1/data/projects/{projectId}/datapoints/{id}`                     | 删除数据点                                           |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/delete-batch`             | 批量删除数据点                                       |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/status`                   | 批量查询数据点状态                                   |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/values`                   | 批量查询数据点值                                     |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints/tags`                     | 获取工程级标签及引用数                               |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/remove-tag`               | 从工程内全部数据点移除标签                           |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/{id}/write`               | 写入数据点值                                         |
| `POST`   | `/api/v1/data/projects/{projectId}/datapoints/write-by-path`            | 按稳定 path 写入数据点值，并校验类型、范围和运行角色 |
| `PUT`    | `/api/v1/data/projects/{projectId}/datapoints/{id}/runtime-permissions` | 更新运行态权限                                       |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints/{id}/custom-attributes`   | 获取开发态自定义属性默认值                           |
| `PUT`    | `/api/v1/data/projects/{projectId}/datapoints/{id}/custom-attributes`   | 原子替换开发态自定义属性默认值                       |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints/{id}/usages`              | 获取数据点引用关系                                   |

自定义属性请求与响应均使用 `{"attributes":{"asset_code":"PUMP-001"}}`。Key 必须以小写字母开头，最长 64 个字符，后续只允许小写字母、数字、点、短横线和下划线；值始终为字符串。内置属性名冲突、格式错误和重复 Key 均按参数错误拒绝。

### 5.5 历史存储 `/api/v1/data/projects/{projectId}/history-storage`

历史存储接口只维护开发态配置，不执行历史写入、建表或历史查询。接入源、工业连接和计算单元配置自动作用于后续新增数据点或计算输出，数据点配置可覆盖来源设置。

| 方法   | 路径                                                                              | 功能概要                                   |
| ------ | --------------------------------------------------------------------------------- | ------------------------------------------ |
| `GET`  | `/api/v1/data/projects/{projectId}/history-storage/sources`                       | 分页查询接入源、工业采集和计算单元历史状态 |
| `GET`  | `/api/v1/data/projects/{projectId}/history-storage/sources/{scopeType}/{scopeId}` | 获取来源历史设置                           |
| `PUT`  | `/api/v1/data/projects/{projectId}/history-storage/sources/{scopeType}/{scopeId}` | 保存来源历史设置                           |
| `GET`  | `/api/v1/data/projects/{projectId}/history-storage/targets`                       | 获取 IF 时序库和 TDengine 目标             |
| `GET`  | `/api/v1/data/projects/{projectId}/history-storage/datapoints/{datapointId}`      | 获取单点覆盖及最终生效设置                 |
| `PUT`  | `/api/v1/data/projects/{projectId}/history-storage/datapoints/{datapointId}`      | 保存单点覆盖或恢复沿用来源                 |
| `POST` | `/api/v1/data/projects/{projectId}/history-storage/datapoints/batch-configure`    | 按 ID 或筛选条件批量设置数据点             |

`GET /sources` 查询参数为 `page`、`pageSize`、`search`、`scopeType` 和 `historyState`。`scopeType` 可取
`access_source`、`collector_connection`、`compute_unit`；`historyState` 可取 `enabled`、`disabled`。列表项固定返回
`scope`、`datapointCount`、`historyState`、`pointOverrideCount`、写入方式和目标摘要，零数据点来源也必须返回。

来源和数据点保存请求统一使用以下结构：

```json
{
  "behavior": "custom",
  "configuration": {
    "writeMode": "on_change",
    "intervalMs": null,
    "deadband": 0,
    "maxSilenceMs": 3600000,
    "offlineBehavior": "store_stale",
    "targets": [
      {
        "connectionId": "00000000-0000-0000-0000-000000000001",
        "isPrimary": true,
        "sortOrder": 0,
        "retentionDays": 30
      }
    ]
  }
}
```

- 来源 `behavior` 只允许 `off`、`custom`；数据点额外允许 `inherit`，此时删除单点覆盖。
- `every_sample` 不接受模式参数；`interval_latest` 和 `periodic_snapshot` 必须提供正数 `intervalMs`。
- `on_change` 可提供非负 `deadband`，`maxSilenceMs` 为 `null` 或正数；质量变化始终触发保存。
- 只有 `periodic_snapshot` 可将 `offlineBehavior` 设为 `skip`，默认值为 `store_stale`。
- 开启配置必须提供至少一个目标，且必须恰有一个 `isPrimary=true`；目标不可重复，只能引用本工程的
  `builtin.timeseries` 或 `tdengine` 连接。
- `retentionDays=null` 表示永久保留，否则必须为正整数；各目标独立配置保留时间。
- 关闭已有配置只改变启用状态，保留模式参数和目标；删除配置不会删除目标库中的表或历史数据。

批量请求的 `selection` 二选一：`{"mode":"ids","datapointIds":[...]}`，或
`{"mode":"filtered","filters":{...},"excludeDatapointIds":[...]}`。筛选模式由后端按数据点列表同口径执行，
不得由前端拉取全部 ID。响应 `data.updatedCount` 返回实际处理数量。

最终生效优先级为“单点覆盖 > 所属来源 > 默认关闭”。单点 `off` 是显式关闭；无单点记录表示沿用来源。
普通数据点归属由 `sourceConfig`、查询/MQTT 结构化关联和直接连接 ID 统一解析，Collector 点通过
`data_collector_points.connection_id` 归属工业采集连接。

### 5.6 MQTT 管理

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

### 5.7 项目快照与工件

| 方法  | 路径                                         | 功能概要       |
| ----- | -------------------------------------------- | -------------- |
| `GET` | `/api/v1/data/projects/{projectId}/snapshot` | 获取项目快照   |
| `PUT` | `/api/v1/data/projects/{projectId}/snapshot` | 替换项目快照   |
| `GET` | `/api/v1/data/projects/{projectId}/artifact` | 获取数据域工件 |

### 5.8 协议接入

#### 协议 Wave 1

| 方法   | 路径                                                                 | 功能概要              |
| ------ | -------------------------------------------------------------------- | --------------------- |
| `POST` | `/api/v1/data/projects/{projectId}/kafka/configs`                    | 创建 Kafka 配置       |
| `PUT`  | `/api/v1/data/projects/{projectId}/kafka/configs/{connectionId}`     | 更新 Kafka 配置与密钥 |
| `POST` | `/api/v1/data/projects/{projectId}/http/configs`                     | 创建 HTTP 配置        |
| `PUT`  | `/api/v1/data/projects/{projectId}/http/configs/{connectionId}`      | 更新 HTTP 配置        |
| `POST` | `/api/v1/data/projects/{projectId}/websocket/configs`                | 创建 WebSocket 配置   |
| `PUT`  | `/api/v1/data/projects/{projectId}/websocket/configs/{connectionId}` | 更新 WebSocket 配置   |
| `POST` | `/api/v1/data/projects/{projectId}/redis/configs`                    | 创建 Redis 配置       |
| `PUT`  | `/api/v1/data/projects/{projectId}/redis/configs/{connectionId}`     | 更新 Redis 配置与密钥 |
| `POST` | `/api/v1/data/projects/{projectId}/protocols/{connectionId}/preview` | 统一短时真实抓样      |

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

#### TDengine 与工业协议边界

| 方法   | 路径                                                                | 功能概要                 |
| ------ | ------------------------------------------------------------------- | ------------------------ |
| `POST` | `/api/v1/data/projects/{projectId}/tdengine/configs`                | 创建 TDengine 配置       |
| `PUT`  | `/api/v1/data/projects/{projectId}/tdengine/configs/{connectionId}` | 更新 TDengine 配置与密钥 |
| `POST` | `/api/v1/data/projects/{projectId}/opcda/contracts/validate`        | 校验 OPC DA 合同         |

TDengine 使用结构化 `ws/wss` 配置和独立只读运行时，支持真实连通测试、对象分页、结构、数据预览和 SQL 工作台。OPC UA、S7、Modbus 等工业协议不再使用这里的旧专用路由，统一由 `/collector/connections` 和驱动 Manifest 管理。

### 5.9 预览会话与计算

#### 预览

| 方法       | 路径                                                    | 功能概要             |
| ---------- | ------------------------------------------------------- | -------------------- |
| `POST`     | `/api/v1/data/projects/{projectId}/preview/sessions`    | 创建预览会话         |
| `GET`      | `/api/v1/data/projects/{projectId}/preview/diagnostics` | 获取预览链路诊断信息 |
| `POST`     | `/api/v1/data/preview/sessions/{sessionId}/heartbeat`   | 预览会话心跳         |
| `DELETE`   | `/api/v1/data/preview/sessions/{sessionId}`             | 关闭预览会话         |
| `GET/POST` | `/socket.io`                                            | 预览 Socket 通道     |

#### 计算单元

| 方法     | 路径                                                               | 功能概要                     |
| -------- | ------------------------------------------------------------------ | ---------------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/compute-units`                  | 查询计算单元列表             |
| `POST`   | `/api/v1/data/projects/{projectId}/compute-units`                  | 创建计算单元                 |
| `GET`    | `/api/v1/data/projects/{projectId}/compute-units/{id}`             | 查询计算单元详情             |
| `PUT`    | `/api/v1/data/projects/{projectId}/compute-units/{id}`             | 更新计算单元并同步输出数据点 |
| `DELETE` | `/api/v1/data/projects/{projectId}/compute-units/{id}`             | 删除计算单元并失效输出数据点 |
| `PATCH`  | `/api/v1/data/projects/{projectId}/compute-units/{id}/enabled`     | 切换计算单元启用状态         |
| `GET`    | `/api/v1/data/projects/{projectId}/compute-units/{id}/runs`        | 查询计算单元运行记录         |
| `POST`   | `/api/v1/data/projects/{projectId}/compute-units/{id}/run`         | 运行计算单元                 |
| `POST`   | `/api/v1/data/projects/{projectId}/compute-units/{id}/debug`       | 调试计算单元                 |
| `GET`    | `/api/v1/data/projects/{projectId}/compute-units/capabilities`     | 获取独立沙箱真实能力         |
| `POST`   | `/api/v1/data/projects/{projectId}/compute-units/schedule-preview` | 校验触发并预览未来五次执行   |
| `GET`    | `/api/v1/data/projects/{projectId}/compute-units/dependencies`     | 查询工程已安装计算依赖       |
| `POST`   | `/api/v1/data/projects/{projectId}/compute-units/dependencies`     | 在线安装工程依赖；版本可选   |
| `POST`   | `/api/v1/data/projects/{projectId}/compute-units/dependencies/import` | 离线导入 npm tgz/Python wheel |
| `DELETE` | `/api/v1/data/projects/{projectId}/compute-units/dependencies/{id}` | 卸载未被计算单元引用的依赖  |

计算脚本的开发态调试只能通过独立 `compute_sandbox` 执行。后端按 `inputBindings` 预取已声明的数据点和 SQL 查询结果，脚本仅可使用 `ctx.datapoint.get(path)`、`ctx.datapoint.meta(path)` 和 `ctx.sql.query(key)` 读取预取数据；不开放任意 SQL、HTTP、MQTT/Kafka、文件、网络或子进程。工程依赖可在线安装或离线导入 npm `.tgz` / Python `.whl`；在线版本留空时获取最新版本，最终始终保存包的实际名称、版本和导入名。保存计算单元时按代码导入语句自动记录引用，执行时仅把当前工程已引用依赖只读挂载到隔离进程。仍被计算单元引用的依赖禁止卸载。沙箱不可用时 capabilities 返回 unavailable，语法检查和试运行禁用，禁止回退到 `data_service` 宿主执行。定时与点变只保存节点运行契约，开发态不自动调度。

#### 报警开发态配置

| 方法     | 路径                                                                   | 功能概要                                                                    |
| -------- | ---------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-items`                        | 分页查询独立报警项，支持点位、类型和目录递归筛选                            |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items`                        | 创建一条普通或组合报警项                                                    |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-items/{id}`                   | 查询报警项详情                                                              |
| `PUT`    | `/api/v1/data/projects/{projectId}/alarm-items/{id}`                   | 原子更新单条报警项                                                          |
| `PATCH`  | `/api/v1/data/projects/{projectId}/alarm-items/{id}/enabled`           | 切换报警项启用状态                                                          |
| `DELETE` | `/api/v1/data/projects/{projectId}/alarm-items/{id}`                   | 删除报警项                                                                  |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/validate-draft`         | 校验名称、指纹和重叠警告                                                    |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/test-draft`             | 使用带时间戳、质量和离线状态的样本序列运行三态试算，返回活动/候选等级与延时 |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-items/{id}/contract`          | 预览 `alarm.item.v1` 契约                                                   |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/batch-create`           | 为明确点位集合原子创建独立报警项                                            |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/batch-create/validate`  | 逐点预校验整批创建草稿                                                      |
| `PUT`    | `/api/v1/data/projects/{projectId}/alarm-items/preset-config`          | 原子替换所选点位的默认报警槽位配置                                          |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/preset-config/validate` | 预校验默认报警配置及条件重叠                                                |
| `PATCH`  | `/api/v1/data/projects/{projectId}/alarm-items/batch`                  | 按 ID 或筛选结果字段掩码批量修改                                            |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/batch-delete`           | 按 ID 或筛选结果批量删除                                                    |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/export`                 | 导出所选普通报警 XLSX                                                       |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-items/import-template`        | 下载普通报警导入模板                                                        |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/import/preview`         | 校验 XLSX 并预览新增、更新和问题                                            |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/import/error-workbook`  | 下载附带逐行错误原因的工作簿                                                |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-items/import/apply`           | 校验摘要和 revision 后事务导入                                              |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-groups`                       | 分页/按父目录查询目录                                                       |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-groups/tree`                  | 查询目录完整路径                                                            |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-groups`                       | 创建目录                                                                    |
| `PUT`    | `/api/v1/data/projects/{projectId}/alarm-groups/{id}`                  | 更新目录                                                                    |
| `DELETE` | `/api/v1/data/projects/{projectId}/alarm-groups/{id}`                  | 删除空目录                                                                  |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-settings`                     | 查询工程默认通知                                                            |
| `PUT`    | `/api/v1/data/projects/{projectId}/alarm-settings`                     | 保存工程默认通知                                                            |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-level-settings`               | 查询工程报警级别与升级规则                                                  |
| `PUT`    | `/api/v1/data/projects/{projectId}/alarm-level-settings`               | 保存工程报警级别与升级规则                                                  |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-history-settings`             | 查询工程报警历史设置                                                        |
| `PUT`    | `/api/v1/data/projects/{projectId}/alarm-history-settings`             | 保存工程报警历史设置                                                        |
| `GET`    | `/api/v1/data/projects/{projectId}/alarm-channels`                     | 查询站内及外部通知渠道                                                      |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-channels`                     | 新增外部通知渠道                                                            |
| `PUT`    | `/api/v1/data/projects/{projectId}/alarm-channels/{id}`                | 更新外部通知渠道                                                            |
| `DELETE` | `/api/v1/data/projects/{projectId}/alarm-channels/{id}`                | 删除未被引用的外部通知渠道                                                  |
| `GET`    | `/api/v1/data/projects/{projectId}/datapoints/{datapointId}/alarms`    | 查询数据点的有效报警摘要                                                    |
| `POST`   | `/api/v1/data/projects/{projectId}/alarm-config-sync`                  | 原子接收节点报警配置增量回写                                                |

报警模式固定为 `point` 与 `derived`。普通报警项只关联一个数据点，报警项 ID 是稳定运行身份；同一点可拥有多种不同报警，但显示名称和触发指纹分别唯一。多点创建会生成 N 条独立报警项。`single` 必须只有一个条件，数值 `highest_matching` 可配置任意数量的高低限等级且同一时刻只选择最深越限等级。工程内置提示、警告、重要、紧急四个级别，并允许增加自定义级别；级别顺序决定严重程度。升级规则按“未确认持续时间”从源级别提升到更高的目标级别。完全重复和同名冲突不可绕过，重叠警告需携带稳定 `ackKey` 确认。Excel 只新增或按 `alarmId + revision` 更新普通报警，不根据缺失行删除，也不执行公式或宏。同步资源为 `alarm_item/group/project_settings/history_settings/channel`；报警历史无显式记录时使用“开启、保留 30 天、保存通知投递记录”的默认值。

#### 契约检查

| 方法   | 路径                                                       | 功能概要                   |
| ------ | ---------------------------------------------------------- | -------------------------- |
| `POST` | `/api/v1/data/projects/{projectId}/contract-checks/run`    | 实时执行项目数据域契约检查 |
| `GET`  | `/api/v1/data/projects/{projectId}/contract-checks/latest` | 获取最近/实时契约检查结果  |
| `GET`  | `/api/v1/data/projects/{projectId}/contract-checks/runs`   | 分页获取契约检查历史记录   |

项目 artifact v1 的 `alarms` 区块使用 `alarm.item.v1`，每个 `items[]` 就是一条稳定报警身份；普通报警直接包含数据点 ID 和路径，组合报警包含输入、别名和表达式。区块还包含目录、分级条件、通知、报警历史、渠道定义和密钥引用，不包含密钥明文或密文。`historyStorage` 始终输出有效值；snapshot replace 在单事务中往返报警项模型并拒绝旧共享结构和名称/指纹冲突。
契约检查 `run` 请求体可传 `scope`、`objectType`、`objectId`，用于收窄检查结果范围；检查项覆盖数据点状态、计算输入/输出、计算 SQL 绑定查询，以及报警项的点位、表达式、条件、通知渠道和启用状态。

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
