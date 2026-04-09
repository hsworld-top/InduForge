# INTEGRATIONS

## 集成总览
- 平台集成可分为三层：
- 平台内部模块集成（`dev_ide` / `datacenter` / `designer` 与 `dev_core`）。
- 基础设施集成（MySQL、Redis、MinIO、MQTT）。
- 运行时集成（`dev_core` 与 `runtime/node_agent`、运维中心）。

## 前后端 API 集成
- 前端默认都以 `/api/v1` 作为基础路径：
- `dev_ide/src/utils/request.js`
- `datacenter/src/utils/request.js`
- `designer/src/utils/request.ts`
- `runtime/node_agent_front/src/api/nodeApi.js`
- 设计器 API 主要挂在 `/design`：`designer/src/services/projectApi.ts`。
- 数据中心 API 主要挂在 `/data`：`datacenter/src/api/data.api.js`。
- 平台管理端 API 主要挂在 `/auth`、`/tenant`、`/project` 等：`dev_ide/src/api/*.api.js`。

## 数据库与缓存
- 主业务库：MySQL（`dev_core/src/config/database.js`，Sequelize 连接池）。
- 多关系库查询能力：MySQL / PostgreSQL / SQL Server（`dev_core/src/services/drivers/*.js`）。
- 节点端本地库：SQLite（`runtime/node_agent/internal/agent/store/store.go`）。
- 缓存：Redis（`dev_core/src/utils/redis.js`），用于运行时状态等高频读写场景。

## 对象存储与制品
- MinIO：`dev_core/src/services/storageService.js`，维护 `ifp-artifacts`、`design-assets` bucket。
- 发布制品（IFP）打包：`dev_core/src/services/publishService.js`（zip 打包 + hash）。
- 下载入口：`/api/v1/publish/deployment/:id/download`（`dev_core/src/routes/v1/publish.js`）。

## MQTT 与实时消息
- MQTT 连接/订阅生命周期：`dev_core/src/services/mqttService.js`。
- MQTT 协议适配：`dev_core/src/services/protocols/MqttProtocol.js`。
- WebSocket 广播：`dev_core/src/services/socketService.js`（主题房间、数据点房间、运维房间）。
- 前端消费：`socket.io-client` 出现在 `dev_ide`、`datacenter`、`designer` 依赖中。

## 运行时与节点运维集成
- 控制面下发节点指令：`dev_core/src/services/deploymentService.js`（deploy/start/stop/restart/rollback）。
- 节点侧统一 API：`runtime/node_agent/internal/web/handler/routes.go`。
- 节点心跳上报：`POST /api/v1/nodes/:nodeId/heartbeat`（`dev_core/src/routes/v1/node.js`）。
- 节点部署状态回传：`POST /api/v1/nodes/:nodeId/deployment-status`（同上）。
- 节点主动下线：`POST /api/v1/nodes/:nodeId/offline`。

## 运维中心（Center）集成
- 节点代理中心 API：`runtime/node_agent/internal/web/handler/center_proxy.go`。
- 在线模式心跳与中心指令处理：`runtime/node_agent/internal/web/handler/heartbeat_runtime.go`。
- 本地管理前端注册流程：`runtime/node_agent_front/src/api/registerApi.js`。

## 认证与权限
- JWT + 刷新令牌：`dev_core/src/middlewares/auth.js`、各前端 `request` 拦截器。
- Token 黑名单与租户隔离：后端中间件 + `X-Tenant-ID` 请求头。
- 角色能力模型：`ROLE_CAPABILITIES`（`dev_core/src/middlewares/auth.js`）。

## 结论
- 本仓库的关键集成链路是：
- 业务前端 -> `dev_core` API -> MySQL/Redis/MinIO/MQTT。
- 运维中心/本地前端 -> `runtime/node_agent` -> 项目执行器（process/docker）。
- `dev_core` 与 `runtime/node_agent` 通过心跳、命令与状态回传闭环联动。
