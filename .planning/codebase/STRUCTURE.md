# STRUCTURE

## 根目录结构
- `AGENTS.md`：仓库级 AI 协作约束。
- `README.md`：项目说明与启动指南。
- `.env` / `.env_example`：统一环境变量入口。
- `docs/`：契约与业务文档中心。
- `dev_core/`：后端控制面。
- `dev_ide/`：平台管理前端。
- `datacenter/`：数据中心前端。
- `designer/`：低代码设计器前端。
- `runtime/`：运行时共享与节点执行体系。

## dev_core
- 入口：`dev_core/src/index.js`、`dev_core/src/app.js`。
- 路由：`dev_core/src/routes/register.js`、`dev_core/src/routes/v1/*`、`dev_core/src/routes/v2/index.js`。
- 中间件：`dev_core/src/middlewares/*`。
- 模型：`dev_core/src/models/*`（集中关联在 `dev_core/src/models/index.js`）。
- 服务：`dev_core/src/services/*`（发布、部署、数据、MQTT、Socket、节点）。
- 驱动：`dev_core/src/services/drivers/*`（MySQL/PostgreSQL/SQLServer）。
- 协议：`dev_core/src/services/protocols/*`（MQTT）。
- 测试：`dev_core/src/dsl/__tests__/*`、`dev_core/src/services/__tests__/*`。

## dev_ide
- 入口：`dev_ide/src/main.js`。
- 路由：`dev_ide/src/router/index.js`。
- API：`dev_ide/src/api/*.api.js`。
- 请求封装：`dev_ide/src/utils/request.js`。
- 视图：`dev_ide/src/views/admin/*`、`dev_ide/src/views/tenant/*`、`dev_ide/src/views/auth/*`。
- 轻量测试：`dev_ide/tests/run-tests.js`。

## datacenter
- 入口：`datacenter/src/main.js`。
- 路由：`datacenter/src/router/index.js`（base 为 `/datacenter/`）。
- API：`datacenter/src/api/data.api.js`。
- 请求封装：`datacenter/src/utils/request.js`。
- 主视图：`datacenter/src/views/DataCenterNew.vue`。
- 组件域：`datacenter/src/components/connection`、`database`、`datapoint`、`mqtt`。

## designer
- 入口：`designer/src/main.ts`。
- 路由：`designer/src/router/index.ts`（base 为 `/designer/`）。
- 请求封装：`designer/src/utils/request.ts`。
- 服务：`designer/src/services/projectApi.ts`、`datacenterApi.ts`、`assetApi.ts`。
- 状态管理：`designer/src/stores/editor-store.ts`。
- 编辑器内核：`designer/src/editor-core/*`。
- UI 壳层：`designer/src/ui/shell/DesignerView.vue`。
- 测试：大量 `*.test.ts` + `designer/vitest.config.js`。

## runtime/node_agent
- 启动入口：`runtime/node_agent/cmd/main.go`。
- 业务编排：`runtime/node_agent/internal/agent/orchestrator/*`。
- 执行器：`runtime/node_agent/internal/agent/executor/*`。
- 本地存储：`runtime/node_agent/internal/agent/store/store.go`。
- API：`runtime/node_agent/internal/web/handler/*`。
- 构建与测试：`runtime/node_agent/Makefile`。

## runtime/node_agent_front
- 入口：`runtime/node_agent_front/src/main.js`。
- 路由：`runtime/node_agent_front/src/router/index.js`。
- API：`runtime/node_agent_front/src/api/nodeApi.js`、`registerApi.js`。
- 页面：`runtime/node_agent_front/src/views/*`。

## 文档与契约
- 契约集中：`docs/contracts/*`。
- 后端文档：`docs/backend/*`。
- designer/datacenter/dev_ide 专项文档在 `docs/` 下分目录维护。

## 结构观察
- 模块边界明确，目录职责命名一致。
- `designer` 体量最大（文件数最高），`dev_core` 次之，是后续优化重点区域。
