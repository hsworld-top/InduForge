# datacenter 模块规则

## 模块定位

- `datacenter` 是数据接入与数据语义建模前端，负责数据库连接、查询、MQTT、变量组与数据点交互。

## 处理该模块时先看哪里

- `datacenter/package.json`
- `datacenter/src/`
- `docs/datacenter/README.md`
- `docs/datacenter/queries.md`
- `docs/datacenter/mqtt-implementation.md`
- `docs/ai-packages/datacenter-ai-package.md`

## 必须遵守

- 技术栈为 Vue 3 + Vite + Pinia + Element Plus + Monaco，图标统一使用 `unplugin-icons`。
- 组件用 `PascalCase`，文件用 `kebab-case`，composable 用 `useXxx`。
- Vue SFC 顺序保持 `template/script/style`。
- 所有 API 调用必须走统一 `request` 封装，Token 与租户信息仅从 Storage 读取。
- 连接模型统一为 `type=relational|mqtt`，具体数据库类型从 `config.dbType` 读取。
- WebSocket 订阅在组件卸载或切换时及时解绑，避免深度 watch 造成额外开销。

## 不要做

- 不要直接在组件里堆叠复杂业务逻辑，优先拆到 composable。
- 不要突破模块边界去实现运行时渲染。
- 不要为当前任务扩大量新连接器类型。

## 验证命令

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter lint`

## 相关参考文档

- `docs/datacenter/README.md`
- `docs/ai-packages/tasks/datacenter.task.md`
