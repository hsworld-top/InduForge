# CONVENTIONS

## 全仓公共约束（来自根 AGENTS）
- 沟通与注释使用中文。
- 根目录统一 `.env`，包管理器统一 `pnpm`。
- 后端 API 前缀 `/api/v1`，健康检查 `/health`。
- 后端错误/响应建议统一：`ApiResponse` + `AppError` + `ErrorCodes`。
- 时间库统一 `dayjs`，字符串格式 `YYYY-MM-DD HH:mm:ss`。
- 数据库初始化应通过 `dev_core/scripts/init-database.js`，禁止 `sequelize.sync()`。

## 模块级命名约定
- Vue 组件名：`PascalCase`（见各模块 AGENTS）。
- 文件名：通常 `kebab-case`。
- composable：`useXxx`（如 `datacenter/src/composables/*`）。
- API 文件：`*.api.js`（`dev_ide/src/api/*.api.js`）。

## 后端实现模式
- 路由层 -> 服务层 -> 模型层：
- 示例：`dev_core/src/routes/v1/deployment.js` -> `dev_core/src/services/deploymentService.js` -> `dev_core/src/models/*`。
- 权限模型：`authenticate`、`checkProjectAccess`、`requireCapability`（`dev_core/src/middlewares/auth.js`）。
- 版本化路由：`dev_core/src/routes/register.js` 挂载 v1/v2。

## 前端实现模式
- 统一 axios 请求封装 + token 刷新：
- `dev_ide/src/utils/request.js`
- `datacenter/src/utils/request.js`
- `designer/src/utils/request.ts`
- 路由守卫处理登录态与页面跳转（如 `dev_ide/src/router/index.js`、`designer/src/router/index.ts`）。
- 大型页面采用懒加载和模块分割（如 `designer/src/ui/shell/DesignerView.vue`）。

## 编辑器工程模式（designer）
- Store 统一调度（`designer/src/stores/editor-store.ts`）。
- 内核命令模式（`designer/src/editor-core/commands/*`）。
- 页面 Schema 归一化与持久化动作拆分到 `stores/editor/*` 子模块。

## 代码质量工具
- ESLint Flat Config：`dev_ide/eslint.config.js`、`datacenter/eslint.config.js`。
- designer 使用 `@antfu/eslint-config`：`designer/eslint.config.mjs`。
- 格式化统一 `prettier`（各模块脚本中定义）。

## 观察到的偏差（与“应有规范”对比）
- `dev_core/src/routes/v1/data.js`、`dev_core/src/routes/v1/node-register.js` 中存在直接 `res.json` 返回，未统一走 `ApiResponse`。
- `dev_core/src/middlewares/auth.js` 在 development 下存在无 token 自动注入默认管理员逻辑。
- 局部文件出现重复定义/重复调用（例如 `dev_core/src/services/socketService.js` 中 `setupDataPointSubscription` 定义重复）。

## 结论
- 仓库总体遵循“分层 + 统一请求封装 + 角色权限”模式。
- 目前处于“新规范与历史实现并存”阶段，建议后续逐步收敛响应格式、权限策略和重复代码段。
