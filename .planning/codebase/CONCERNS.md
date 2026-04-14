# CONCERNS

## 高优先级关注点

### 1. 响应格式不统一
- 规范要求后端接口统一 `ApiResponse`。
- 但在 `dev_core/src/routes/v1/data.js`、`dev_core/src/routes/v1/node-register.js` 可见直接 `res.json({...})` 与 `res.status(...).json({...})`。
- 风险：前端错误处理分支复杂化，跨模块契约不一致，难以做统一网关与审计。

### 2. 开发环境认证旁路
- `dev_core/src/middlewares/auth.js` 在 development 且无 token 时注入默认 `SYSTEM_ADMIN` 用户。
- 风险：本地调试与真实权限模型偏离，易掩盖鉴权缺陷；若环境变量配置错误可能引入越权风险。

### 3. Socket 服务重复代码段
- `dev_core/src/services/socketService.js` 中 `setupDataPointSubscription` 存在重复定义，`setupEventHandlers` 也重复调用了该方法。
- 风险：维护成本升高，后续修复易出现“一处改动另一处遗漏”。

## 中优先级关注点

### 4. NodeAgent Makefile 目标一致性问题
- `runtime/node_agent/Makefile` 的 `dev` 目标中命令块重复。
- `help` 文本提到 `build-windows-console`，但文件中不存在该目标。
- 风险：新同学执行命令时预期与实际不一致。

### 5. 进程执行器日志覆盖行为
- `runtime/node_agent/internal/agent/executor/process.go` 在 `Start` 中使用 `os.Create(logFile)`。
- 这会在每次启动时覆盖旧日志。
- 风险：运维排障时历史日志丢失。

### 6. 发布资产打包不完整
- `dev_core/src/services/publishService.js` 中明确有 `TODO: 添加 assets 目录`。
- 风险：IFP 制品可能缺少运行所需静态资源，导致部署后行为与设计器预期不一致。

## 低优先级关注点

### 7. 文档与编码一致性
- 根 `README.md` 在当前终端读取出现明显乱码（疑似编码/终端码页不一致）。
- 风险：新成员快速上手体验下降，且自动化文档处理可能失败。

### 8. 环境模板包含强默认口令
- `.env_example` 中出现示例型强默认值（如 `DB_PASSWORD=123456`、`REDIS_PASSWORD=Rootali1`）。
- 风险：被误用于真实环境时造成安全隐患。

## 建议处理顺序
- 第一步：统一 `dev_core` 路由返回结构到 `ApiResponse`。
- 第二步：清理 `socketService` 重复定义并补回归测试。
- 第三步：修正 NodeAgent Makefile 与日志追加策略。
- 第四步：完善 IFP 资产打包、清理模板密码与 README 编码。
