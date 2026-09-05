# dev_ide 协作规则

## 模块定位

- `dev_ide` 是平台管理与运维前端，负责项目入口、权限视图、部署管理与节点状态展示。

## 按需参考

以下路径均相对仓库根目录，仅在任务涉及对应内容时查阅。

- 管理端职责与页面边界：`docs/03-模块设计/dev_ide/README.md`。
- 认证与请求处理：`dev_ide/src/utils/request.ts`；权限与接口契约以根目录规则引用的正式文档为准。
- 运维节点与部署交互：`docs/04-契约与规范/跨模块契约/node-agent-runtime-protocol.md`；展示被托管组件健康字段时，再查 `docs/04-契约与规范/跨模块契约/runtime-health-status-contract.md`。

## 开发约束

- 运行时为 Vue 3 + Vite + Pinia + Element Plus，优先沿用现有页面、路由、store 和请求封装方式。
- Vue 组件及其文件使用 `PascalCase`，API 文件使用 `*.api.ts`；其他文件沿用所在目录的命名方式。
- 浏览器认证使用同源 `HttpOnly` Cookie，不在前端存储或传递 JWT；权限以后端为准，重要操作保持前后端双重校验。
- 未登录统一跳转 `/login`，涉及角色语义调整时同步核对前后端枚举与菜单逻辑。
- 视图层逻辑优先抽到 composable 或 util，避免无意义的全量 store 更新。

## 禁止事项

- 不要绕过统一请求封装直接发接口。
- 不要把 Designer 或运行时渲染逻辑混入管理端。
- 不要在当前任务中顺手重构整套后台框架。

## 验证命令

- 从仓库根目录按变更面选用，不要求每次全跑；测试命令可追加相关测试文件路径。
- TypeScript 改动：`pnpm --dir dev_ide typecheck`。
- `pnpm --dir dev_ide test`
- `pnpm --dir dev_ide build`
