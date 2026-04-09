# TESTING

## 测试现状概览
- 当前仓库存在约 `50` 个测试相关文件（按文件名/目录模式统计）。
- 主要测试密度集中在 `designer`、`dev_core`、`runtime/node_agent`。
- `dev_ide` 目前是轻量脚本断言测试（`dev_ide/tests/run-tests.js`）。

## 后端（dev_core）
- 测试命令：`pnpm --dir dev_core test`（`dev_core/package.json`）。
- 框架：Jest + Supertest + fast-check（依赖见 `dev_core/package.json`）。
- 测试分布：
- `dev_core/src/dsl/__tests__/*`（属性测试、校验器测试）
- `dev_core/src/services/__tests__/*`（服务行为与响应约束）
- 附加脚本：`dev_core/scripts/test_mqtt_server.js`（联调用途）。

## 设计器（designer）
- 测试命令：
- `pnpm --dir designer test`
- `pnpm --dir designer typecheck`
- 框架：Vitest + jsdom + Vue Test Utils（`designer/vitest.config.js`、`designer/package.json`）。
- 覆盖方向：
- store 合同测试：`designer/src/stores/editor-store.contracts.test.ts`
- 页面/面板逻辑：`designer/src/ui/editors/page/.../*.test.ts`
- 工具函数与规范化逻辑：`designer/src/data/*.test.ts`、`designer/src/utils/*.test.ts`

## 数据中心（datacenter）
- 主要验证命令：`pnpm --dir datacenter build`、`pnpm --dir datacenter lint`（模块 AGENTS）。
- 未见独立系统化单测目录，更多依赖构建与手工联调验证。

## 平台管理端（dev_ide）
- 测试命令：`pnpm --dir dev_ide test`。
- 实现方式：Node 脚本断言（`dev_ide/tests/run-tests.js`），覆盖权限规则、状态聚合、i18n 键值。

## 节点执行器（runtime/node_agent）
- 测试命令：
- `make -C runtime/node_agent test`
- `make -C runtime/node_agent vet`
- 代码中存在 Go 单元测试：
- `runtime/node_agent/internal/agent/executor/process_test.go`
- `runtime/node_agent/internal/agent/orchestrator/orchestrator_test.go`
- `runtime/node_agent/internal/web/handler/api_test.go`
- 其他 pkg 层 `*_test.go`。

## 节点前端（runtime/node_agent_front）
- 主要验证命令：`pnpm --dir runtime/node_agent_front build`。
- 当前未发现同等规模自动化单测文件。

## 测试风格与模式
- 属性测试：`dev_core` 与 `designer` 都引入 fast-check 进行生成式验证。
- 合同意识：`designer` 存在 contract 测试文件命名。
- 运行时验证：Go 模块偏向包级单元测试 + Makefile 编译/检查组合。

## 覆盖风险
- 前端多模块自动化测试覆盖不均衡（`designer` 高、`datacenter/node_agent_front` 低）。
- 部分关键跨模块链路（发布 -> 部署 -> 心跳回传）偏集成行为，建议增加端到端回归脚本。

## 建议优先补强
- 为 `datacenter` 补齐 API 适配层与核心 composable 单测。
- 为 `runtime/node_agent_front` 增加初始化向导与注册流程回归测试。
- 为 `dev_core` 的路由层增加统一响应格式回归测试。
