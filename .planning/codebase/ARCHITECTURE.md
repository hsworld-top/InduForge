# ARCHITECTURE

## 架构风格
- 仓库采用多应用单仓（multi-app monorepo）结构，而不是严格的 pnpm workspace 根工作区。
- 平台控制面与数据聚合集中在 `dev_core`，多个前端应用按业务域拆分。
- 运行时执行层与控制面解耦，`runtime/node_agent` 可独立部署在节点侧。

## 分层视图
- 展示层：
- `dev_ide`（平台管理）
- `datacenter`（数据中心）
- `designer`（低代码设计器）
- `runtime/node_agent_front`（节点本地运维）
- 控制层：
- `dev_core/src/routes/v1/*.js`（版本化 API）
- `dev_core/src/middlewares/*.js`（认证、审计、国际化、请求ID）
- 领域服务层：
- `dev_core/src/services/*.js`（发布、部署、数据连接、MQTT、设计资产等）
- 数据访问层：
- Sequelize 模型与关联：`dev_core/src/models/index.js`
- 驱动抽象层：`dev_core/src/services/drivers/*.js`
- 执行层：
- 进程/容器执行器：`runtime/node_agent/internal/agent/executor/*.go`

## 后端启动与路由装配
- 入口：`dev_core/src/index.js`。
- 应用构建：`dev_core/src/app.js`。
- 路由注册：`dev_core/src/routes/register.js`。
- API 版本：
- v1：业务主路由（`/api/v1`）
- v2：测试路由（`/api/v2/test`，`dev_core/src/routes/v2/index.js`）

## 核心业务流
- 发布流：
- `publishService.publish`（`dev_core/src/services/publishService.js`）
- 校验工程 -> 汇总页面/数据点 -> 生成 manifest -> 打包 IFP -> 记录 Deployment
- 部署流：
- `deploymentService.deployToNodes/deployDevToNodesByProject`（`dev_core/src/services/deploymentService.js`）
- 创建 NodeDeployment/NodeCommand -> 节点心跳取指令 -> 节点回传状态
- 节点执行流：
- `runtime/node_agent/cmd/main.go` 启动 HTTP 服务
- `APIHandler` 处理项目部署/启停/回滚（`runtime/node_agent/internal/web/handler/api.go`）
- `Orchestrator` 调用 `Executor` 完成执行（`runtime/node_agent/internal/agent/orchestrator/orchestrator.go`）

## 实时通信流
- MQTT 输入：`MqttProtocol` -> `MqttService.handleMqttMessage`。
- 消息分发：`socketService.broadcastMqttMessage`、数据点广播、运维广播。
- 前端消费：通过 socket.io 房间订阅（项目、订阅、数据点、运维租户维度）。

## 设计器内部架构
- 主状态：`designer/src/stores/editor-store.ts`（文档、历史、选区、页面、锁）。
- 文档模型：`designer/src/editor-core/document/*`。
- 命令系统：`designer/src/editor-core/commands/*`。
- 壳层编排：`designer/src/ui/shell/DesignerView.vue`。

## 运行时模式
- NodeAgent 支持 `offline` 与 `online` 模式（`runtime/node_agent/cmd/main.go` + `internal/pkg/config`）。
- online 模式通过中心心跳接收 deploy/start/stop/restart/rollback 指令（`heartbeat_runtime.go`）。
- 离线模式提供本地工程管理 API 与本地前端。

## 架构特征总结
- 优点：模块边界清晰、运行时可独立部署、前后端职责分层明确。
- 当前成本：接口风格不完全统一、历史代码与新规范并存、跨模块契约仍需持续收敛。
