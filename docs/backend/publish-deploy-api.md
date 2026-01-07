# 发布部署 API 文档

本文档描述工程发布、节点管理、部署相关的后端 API 接口设计。

> **状态**: 待实现。本文档为 API 设计规范，供后续开发参考。

## 1. API 概览

### 1.1 接口分类

| 模块 | 路径前缀                               | 说明         |
| ---- | -------------------------------------- | ------------ |
| 发布 | `/api/v1/projects/:projectId/publish`  | 工程发布相关 |
| 版本 | `/api/v1/projects/:projectId/versions` | 版本管理     |
| 节点 | `/api/v1/nodes`                        | 节点管理     |
| 部署 | `/api/v1/deployments`                  | 部署管理     |

### 1.2 通用响应格式

```typescript
// 成功响应
{
  "success": true,
  "data": { ... },
  "requestId": "REQ_xxx"
}

// 错误响应
{
  "success": false,
  "errorCode": "ERROR_CODE",
  "message": "错误描述",
  "requestId": "REQ_xxx"
}
```

## 2. 发布 API

### 2.1 发布工程

触发工程发布流程，生成 IFP 包。

```
POST /api/v1/projects/:projectId/publish
```

**请求参数**

```typescript
interface PublishRequest {
  version: string; // 版本号，如 "1.0.0"
  releaseNotes?: string; // 发布说明
  includeDatacenter: boolean; // 是否包含 DataCenter 配置
  createSnapshot: boolean; // 是否创建工程快照
}
```

**请求示例**

```json
{
  "version": "1.2.0",
  "releaseNotes": "新增实时监控页面",
  "includeDatacenter": true,
  "createSnapshot": true
}
```

**响应**

```typescript
interface PublishResponse {
  deploymentId: string; // 部署记录 ID
  version: string;
  artifactUrl: string; // IFP 包下载地址
  artifactHash: string; // SHA256 哈希
  artifactSize: number; // 文件大小（字节）
  snapshotId?: string; // 快照 ID
  publishedAt: string; // 发布时间

  // 校验结果
  validation: {
    valid: boolean;
    errors: ValidationError[];
    warnings: ValidationWarning[];
  };

  // 清单信息
  manifest: {
    dataRequirements: {
      datapoints: DataPointRequirement[];
      providers: string[];
    };
    capabilities: string[];
    pages: number;
    assets: number;
  };
}
```

**响应示例**

```json
{
  "success": true,
  "data": {
    "deploymentId": "dep_xxx",
    "version": "1.2.0",
    "artifactUrl": "https://registry.example.com/projects/proj_xxx/v1.2.0.ifp",
    "artifactHash": "sha256:abc123...",
    "artifactSize": 1048576,
    "snapshotId": "snap_xxx",
    "publishedAt": "2026-01-07T14:30:00Z",
    "validation": {
      "valid": true,
      "errors": [],
      "warnings": [
        { "code": "UNUSED_ASSET", "message": "存在未使用的资源: bg.png" }
      ]
    },
    "manifest": {
      "dataRequirements": {
        "datapoints": [
          { "path": "mqtt.EMQX.温度组.temperature", "required": true }
        ],
        "providers": ["dc_main"]
      },
      "capabilities": ["mqtt", "database"],
      "pages": 5,
      "assets": 12
    }
  }
}
```

**错误码**

| 错误码                | 说明                     |
| --------------------- | ------------------------ |
| `PROJECT_NOT_FOUND`   | 工程不存在               |
| `VALIDATION_FAILED`   | 校验失败（返回详细错误） |
| `VERSION_EXISTS`      | 版本号已存在             |
| `PUBLISH_IN_PROGRESS` | 发布进行中               |
| `DATAPOINT_INVALID`   | 存在失效的数据点绑定     |

### 2.2 获取发布状态

获取正在进行的发布任务状态。

```
GET /api/v1/projects/:projectId/publish/status
```

**响应**

```typescript
interface PublishStatusResponse {
  status:
    | "idle"
    | "validating"
    | "compiling"
    | "bundling"
    | "uploading"
    | "completed"
    | "failed";
  progress: number; // 0-100
  currentStep?: string; // 当前步骤描述
  error?: string; // 失败原因
  startedAt?: string;
  completedAt?: string;
}
```

### 2.3 取消发布

取消正在进行的发布任务。

```
POST /api/v1/projects/:projectId/publish/cancel
```

## 3. 版本 API

### 3.1 获取版本列表

```
GET /api/v1/projects/:projectId/versions
```

**查询参数**

| 参数     | 类型   | 说明                      |
| -------- | ------ | ------------------------- |
| page     | number | 页码，默认 1              |
| pageSize | number | 每页数量，默认 20         |
| status   | string | 状态筛选：active/archived |

**响应**

```typescript
interface VersionListResponse {
  total: number;
  items: Version[];
}

interface Version {
  id: string;
  version: string;
  releaseNotes?: string;
  artifactUrl: string;
  artifactHash: string;
  artifactSize: number;
  status: "active" | "archived";
  publishedBy: string;
  publishedAt: string;

  // 部署统计
  deploymentCount: number;
  activeDeployments: number;
}
```

### 3.2 获取版本详情

```
GET /api/v1/projects/:projectId/versions/:version
```

**响应**

```typescript
interface VersionDetailResponse extends Version {
  manifest: Manifest;
  deployments: DeploymentSummary[];
  snapshot?: {
    id: string;
    url: string;
    hash: string;
  };
}
```

### 3.3 下载版本

```
GET /api/v1/projects/:projectId/versions/:version/download
```

返回 IFP 文件流。

### 3.4 归档版本

```
POST /api/v1/projects/:projectId/versions/:version/archive
```

归档后版本不可部署，但保留记录。

### 3.5 删除版本

```
DELETE /api/v1/projects/:projectId/versions/:version
```

仅允许删除无部署记录的版本。

## 4. 节点 API

### 4.1 生成注册码

```
POST /api/v1/nodes/registration-code
```

**请求参数**

```typescript
interface RegistrationCodeRequest {
  expiresIn?: number; // 有效期（秒），默认 86400（24小时）
  maxUses?: number; // 最大使用次数，默认 1
}
```

**响应**

```typescript
interface RegistrationCodeResponse {
  code: string; // 注册码
  expiresAt: string; // 过期时间
  maxUses: number;
  usedCount: number;
}
```

### 4.2 注册节点

NodeAgent 调用此接口完成注册。

```
POST /api/v1/nodes/register
```

**请求参数**

```typescript
interface NodeRegisterRequest {
  code: string; // 注册码
  name: string; // 节点名称
  system: {
    os: string;
    arch: string;
    nodeAgentVersion: string;
  };
}
```

**响应**

```typescript
interface NodeRegisterResponse {
  nodeId: string;
  token: string; // 节点认证 Token
  wsUrl: string; // WebSocket 连接地址
}
```

### 4.3 获取节点列表

```
GET /api/v1/nodes
```

**查询参数**

| 参数     | 类型   | 说明                         |
| -------- | ------ | ---------------------------- |
| status   | string | 状态筛选：online/offline/all |
| page     | number | 页码                         |
| pageSize | number | 每页数量                     |

**响应**

```typescript
interface NodeListResponse {
  total: number;
  items: Node[];
}

interface Node {
  id: string;
  name: string;
  status: "online" | "offline" | "unknown";
  ipAddress: string;
  port: number;
  system?: {
    cpu: number;
    memory: number;
    disk: number;
    os: string;
    nodeAgentVersion: string;
  };
  currentDeployment?: {
    projectId: string;
    projectName: string;
    version: string;
    status: "running" | "stopped" | "error";
  };
  registeredAt: string;
  lastHeartbeat: string;
}
```

### 4.4 获取节点详情

```
GET /api/v1/nodes/:nodeId
```

### 4.5 更新节点

```
PUT /api/v1/nodes/:nodeId
```

**请求参数**

```typescript
interface NodeUpdateRequest {
  name?: string;
}
```

### 4.6 删除节点

```
DELETE /api/v1/nodes/:nodeId
```

仅允许删除离线且无部署的节点。

### 4.7 获取节点日志

```
GET /api/v1/nodes/:nodeId/logs
```

**查询参数**

| 参数      | 类型   | 说明                            |
| --------- | ------ | ------------------------------- |
| projectId | string | 按工程筛选                      |
| level     | string | 日志级别：debug/info/warn/error |
| startTime | string | 开始时间                        |
| endTime   | string | 结束时间                        |
| limit     | number | 数量限制，默认 100              |

**响应**

```typescript
interface NodeLogsResponse {
  logs: LogEntry[];
}

interface LogEntry {
  timestamp: string;
  level: "debug" | "info" | "warn" | "error";
  source: "nodeagent" | "runtime" | "dataservice";
  projectId?: string;
  message: string;
  meta?: object;
}
```

## 5. 部署 API

### 5.1 部署工程到节点

```
POST /api/v1/deployments
```

**请求参数**

```typescript
interface DeployRequest {
  projectId: string;
  version: string;
  nodeIds: string[]; // 目标节点 ID 列表
  config: {
    port: number;
    uiTarget: "pc" | "bigscreen" | "mobile";
    defaultLocale: string;
    defaultTheme: string;
  };
}
```

**请求示例**

```json
{
  "projectId": "proj_xxx",
  "version": "1.2.0",
  "nodeIds": ["node_001", "node_002"],
  "config": {
    "port": 8080,
    "uiTarget": "bigscreen",
    "defaultLocale": "zh-CN",
    "defaultTheme": "dark"
  }
}
```

**响应**

```typescript
interface DeployResponse {
  deployments: DeploymentResult[];
}

interface DeploymentResult {
  nodeId: string;
  nodeName: string;
  status: "pending" | "deploying" | "success" | "failed";
  error?: string;
}
```

### 5.2 获取部署列表

```
GET /api/v1/deployments
```

**查询参数**

| 参数      | 类型   | 说明       |
| --------- | ------ | ---------- |
| projectId | string | 按工程筛选 |
| nodeId    | string | 按节点筛选 |
| status    | string | 状态筛选   |
| page      | number | 页码       |
| pageSize  | number | 每页数量   |

**响应**

```typescript
interface DeploymentListResponse {
  total: number;
  items: Deployment[];
}

interface Deployment {
  id: string;
  projectId: string;
  projectName: string;
  version: string;
  nodeId: string;
  nodeName: string;
  status: "pending" | "deploying" | "running" | "stopped" | "failed";
  config: DeployConfig;
  deployedBy: string;
  deployedAt: string;
  stoppedAt?: string;
  error?: string;
}
```

### 5.3 获取部署详情

```
GET /api/v1/deployments/:deploymentId
```

### 5.4 停止部署

```
POST /api/v1/deployments/:deploymentId/stop
```

### 5.5 重启部署

```
POST /api/v1/deployments/:deploymentId/restart
```

### 5.6 回滚部署

```
POST /api/v1/deployments/:deploymentId/rollback
```

**请求参数**

```typescript
interface RollbackRequest {
  targetVersion: string; // 目标版本
}
```

### 5.7 删除部署记录

```
DELETE /api/v1/deployments/:deploymentId
```

仅允许删除已停止的部署记录。

## 6. 数据点状态 API

### 6.1 批量查询数据点状态

```
POST /api/v1/datapoints/status
```

**请求参数**

```typescript
interface DataPointStatusRequest {
  projectId: string;
  paths: string[];
}
```

**响应**

```typescript
interface DataPointStatusResponse {
  items: DataPointStatus[];
}

interface DataPointStatus {
  path: string;
  status: "active" | "invalid" | "unknown";
  statusReason?: string;
  lastSeenAt?: string;
  lastValueAt?: string;
}
```

### 6.2 获取数据点引用

获取引用特定数据点的组件列表。

```
GET /api/v1/projects/:projectId/datapoints/:path/references
```

**响应**

```typescript
interface DataPointReferencesResponse {
  total: number;
  references: DataPointReference[];
}

interface DataPointReference {
  pageId: string;
  pageName: string;
  nodeId: string;
  nodeName: string;
  propKey: string;
  refType: "datapoint" | "expr" | "script";
}
```

## 7. 快照 API

### 7.1 获取快照列表

```
GET /api/v1/projects/:projectId/snapshots
```

### 7.2 获取快照详情

```
GET /api/v1/projects/:projectId/snapshots/:snapshotId
```

### 7.3 从快照恢复

```
POST /api/v1/projects/:projectId/snapshots/:snapshotId/restore
```

### 7.4 删除快照

```
DELETE /api/v1/projects/:projectId/snapshots/:snapshotId
```

## 8. 权限说明

### 8.1 接口权限要求

| 接口           | 所需角色                 |
| -------------- | ------------------------ |
| 发布工程       | PROJECT_ADMIN            |
| 版本管理       | PROJECT_ADMIN            |
| 生成注册码     | OPS_ADMIN                |
| 节点管理       | OPS_ADMIN                |
| 部署/停止/回滚 | OPS_ADMIN                |
| 查看部署       | PROJECT_ADMIN, OPS_ADMIN |
| 数据点状态     | 工程成员                 |
| 快照管理       | PROJECT_ADMIN            |

### 8.2 租户隔离

所有接口自动按当前用户的租户进行数据隔离。

## 9. 错误码汇总

| 错误码                      | HTTP 状态码 | 说明                 |
| --------------------------- | ----------- | -------------------- |
| `PROJECT_NOT_FOUND`         | 404         | 工程不存在           |
| `VERSION_NOT_FOUND`         | 404         | 版本不存在           |
| `NODE_NOT_FOUND`            | 404         | 节点不存在           |
| `DEPLOYMENT_NOT_FOUND`      | 404         | 部署记录不存在       |
| `VALIDATION_FAILED`         | 400         | 发布校验失败         |
| `VERSION_EXISTS`            | 409         | 版本号已存在         |
| `PUBLISH_IN_PROGRESS`       | 409         | 发布进行中           |
| `NODE_OFFLINE`              | 400         | 节点离线，无法部署   |
| `DEPLOYMENT_RUNNING`        | 400         | 部署运行中，无法删除 |
| `REGISTRATION_CODE_INVALID` | 400         | 注册码无效或已过期   |
| `REGISTRATION_CODE_USED`    | 400         | 注册码已使用         |
| `ACCESS_DENIED`             | 403         | 无权限访问           |

## 10. 实现优先级

### 第一阶段（必要功能）

- [ ] `POST /api/v1/projects/:projectId/publish` - 发布工程
- [ ] `GET /api/v1/projects/:projectId/versions` - 版本列表
- [ ] `GET /api/v1/projects/:projectId/versions/:version/download` - 下载版本

### 第二阶段（节点管理）

- [ ] `POST /api/v1/nodes/registration-code` - 生成注册码
- [ ] `POST /api/v1/nodes/register` - 注册节点
- [ ] `GET /api/v1/nodes` - 节点列表
- [ ] `GET /api/v1/nodes/:nodeId` - 节点详情

### 第三阶段（部署功能）

- [ ] `POST /api/v1/deployments` - 部署工程
- [ ] `GET /api/v1/deployments` - 部署列表
- [ ] `POST /api/v1/deployments/:deploymentId/stop` - 停止
- [ ] `POST /api/v1/deployments/:deploymentId/rollback` - 回滚

### 第四阶段（增强功能）

- [ ] `POST /api/v1/datapoints/status` - 数据点状态
- [ ] `GET /api/v1/nodes/:nodeId/logs` - 节点日志
- [ ] 快照相关接口
- [ ] 页面锁相关接口

## 8. 页面锁 API

用于支持多人开发时的页面级编辑锁定。

### 8.1 获取页面锁

```
POST /api/v1/pages/:pageId/lock
```

**响应（成功）**

```json
{
  "success": true,
  "data": {
    "pageId": "page_xxx",
    "lockedBy": "user_001",
    "lockedByName": "张三",
    "lockedAt": 1704614400000
  }
}
```

**响应（已被锁定）**

```json
{
  "success": false,
  "errorCode": "PAGE_LOCKED",
  "message": "页面已被其他用户锁定",
  "data": {
    "lockedBy": "user_002",
    "lockedByName": "李四",
    "lockedAt": 1704614300000
  }
}
```

### 8.2 释放页面锁

```
DELETE /api/v1/pages/:pageId/lock
```

**权限**: 锁持有者

### 8.3 强制释放页面锁

```
DELETE /api/v1/pages/:pageId/lock/force
```

**权限**: PROJECT_ADMIN+

### 8.4 心跳续锁

```
POST /api/v1/pages/:pageId/lock/heartbeat
```

**权限**: 锁持有者

### 8.5 查询锁状态

```
GET /api/v1/pages/:pageId/lock
```

**响应**

```json
{
  "success": true,
  "data": {
    "locked": true,
    "lockedBy": "user_001",
    "lockedByName": "张三",
    "lockedAt": 1704614400000
  }
}
```

### 8.6 批量查询页面锁状态

```
POST /api/v1/pages/lock/batch
```

**请求**

```json
{
  "projectId": "proj_xxx",
  "pageIds": ["page_001", "page_002", "page_003"]
}
```

**响应**

```json
{
  "success": true,
  "data": {
    "locks": {
      "page_001": {
        "locked": true,
        "lockedBy": "user_001",
        "lockedByName": "张三"
      },
      "page_002": { "locked": false },
      "page_003": {
        "locked": true,
        "lockedBy": "user_002",
        "lockedByName": "李四"
      }
    }
  }
}
```

---

**相关文档**：

- [发布流水线](../designer/refactor/publish-pipeline.md)
- [运行时引擎](../designer/refactor/runtime-engine.md)
- [WebSocket 协议](./websocket.md)
- [后端 API 总览](./README.md)
- [dev_ide 文档](../dev_ide/README.md)

---

**版本**: 1.0.0  
**创建日期**: 2026-01-07
