# dev_core 协作规则

## 模块定位

- `dev_core` 是平台控制面后端，负责工程、页面、认证、发布、部署、预览聚合与平台级 API。

## 进入前先看

- `dev_core/package.json`
- `dev_core/src/index.js`
- `dev_core/src/app.js`
- `dev_core/src/routes/`
- `dev_core/src/services/`
- `docs/03-模块设计/dev_core/README.md`
- `docs/03-模块设计/dev_core/publish-deploy-api.md`

## 开发约束

- 运行时为 Node.js + Express + Sequelize，改动优先沿用现有路由层、服务层、模型层结构。
- 路由统一挂载在 `/api/v1`，仅测试接口允许放在 `/api/v2`。
- 返回统一使用 `ApiResponse`，业务错误统一使用 `AppError` 与 `ErrorCodes`。
- 参数校验优先走中间件或服务层，认证与权限校验优先放在路由层。
- 控制面数据库只通过 `scripts/bootstrap/init-core-database.js` 初始化空库；已有数据库禁止自动同步或兼容修复，结构变更直接操作开发数据库并更新最终 `core-schema.sql`。
- 查询注意避免 N+1；Socket.IO 推送优先按房间粒度广播。
- 日志中禁止输出密码、令牌等敏感字段。

## 禁止事项

- 不要发明新的响应结构。
- 不要把编辑器内部模型直接混入发布或预览聚合结构。
- 不要借当前任务顺手扩展无关后台能力。

## 验证命令

- `pnpm --dir dev_core test`

## 相关契约

- `docs/03-模块设计/dev_core/README.md`
- `docs/03-模块设计/dev_core/publish-deploy-api.md`
- `docs/04-契约与规范/跨模块契约/designer-publish-schema.md`
- `docs/04-契约与规范/跨模块契约/ifp-manifest-contract.md`
