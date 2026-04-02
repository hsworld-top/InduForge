# dev_core 模块规则

## 模块定位

- `dev_core` 是平台控制面后端，负责工程、页面、数据连接、发布与预览聚合能力。

## 处理该模块时先看哪里

- `dev_core/package.json`
- `dev_core/src/index.js`
- `dev_core/src/routes/`
- `dev_core/src/services/`
- `docs/backend/README.md`
- `docs/contracts/designer-publish-schema.md`
- `docs/ai-packages/dev_core-ai-package.md`

## 必须遵守

- 路由统一挂在 `/api/v1`，测试接口才允许放在 `/api/v2`。
- 返回统一使用 `ApiResponse`，业务错误统一使用 `AppError` 与 `ErrorCodes`。
- 参数校验优先走 `validate` 中间件或服务层校验，认证与权限检查优先放在路由层。
- 数据库初始化只能走 `scripts/init-database.js`，禁止 `sequelize.sync()`。
- 查询注意避免 N+1；Socket.IO 推送优先房间粒度，不要全局广播。
- 日志中禁止输出密码、令牌等敏感字段。

## 不要做

- 不要直接发明新的响应结构。
- 不要把编辑器内部模型混入发布或预览聚合结构。
- 不要为当前任务顺手扩展无关后台功能。

## 验证命令

- `pnpm --dir dev_core test`

## 相关参考文档

- `docs/backend/publish-deploy-api.md`
- `docs/contracts/ifp-manifest-contract.md`
- `docs/ai-packages/tasks/dev_core.task.md`
