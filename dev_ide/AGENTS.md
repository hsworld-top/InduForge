# dev_ide 模块规则

## 模块定位

- `dev_ide` 是平台管理与运维前端，负责项目入口、权限视图、部署与节点状态展示。

## 处理该模块时先看哪里

- `dev_ide/package.json`
- `dev_ide/src/`
- `dev_ide/tests/`
- `docs/dev_ide/README.md`
- `docs/contracts/runtime-health-status-contract.md`
- `docs/ai-packages/dev_ide-ai-package.md`

## 必须遵守

- 技术栈为 Vue 3 + Vite + Pinia + Element Plus，图标统一使用 `unplugin-icons`。
- 组件用 `PascalCase`，文件用 `kebab-case`，API 文件保持 `*.api.js`。
- 未登录统一跳转 `/login`，超级管理员固定进入 `/admin`。
- 角色语义要与后端一致，清理或修正 `TENANT_ADMIN` 历史残留时必须同步核对。
- 认证和权限只信任后端 Token，重要操作保持前后端双重校验。
- 视图层逻辑优先抽到 composable 或 util，避免无意义的全量 store 更新。

## 不要做

- 不要绕过统一请求封装直接发接口。
- 不要在当前任务中顺手重构整套后台框架。
- 不要把 Designer 或运行时渲染逻辑混入管理端。

## 验证命令

- `pnpm --dir dev_ide test`
- `pnpm --dir dev_ide build`

## 相关参考文档

- `docs/dev_ide/README.md`
- `docs/ai-packages/tasks/dev_ide.task.md`
