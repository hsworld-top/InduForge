# dev_ide 协作规则

## 模块定位

- `dev_ide` 是平台管理与运维前端，负责项目入口、权限视图、部署管理与节点状态展示。

## 进入前先看

- `dev_ide/package.json`
- `dev_ide/src/`
- `dev_ide/tests/`
- `docs/dev_ide/README.md`

## 开发约束

- 运行时为 Vue 3 + Vite + Pinia + Element Plus，优先沿用现有页面、路由、store 和请求封装方式。
- 组件使用 `PascalCase`，文件使用 `kebab-case`，API 文件保持 `*.api.js`。
- 认证与权限只信任后端 Token，重要操作保持前后端双重校验。
- 未登录统一跳转 `/login`，涉及角色语义调整时同步核对前后端枚举与菜单逻辑。
- 视图层逻辑优先抽到 composable 或 util，避免无意义的全量 store 更新。

## 禁止事项

- 不要绕过统一请求封装直接发接口。
- 不要把 Designer 或运行时渲染逻辑混入管理端。
- 不要在当前任务中顺手重构整套后台框架。

## 验证命令

- `pnpm --dir dev_ide test`
- `pnpm --dir dev_ide build`

## 相关契约

- `docs/dev_ide/README.md`
- `docs/contracts/runtime-health-status-contract.md`
