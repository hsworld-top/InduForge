# datacenter 协作规则

## 模块定位

- `datacenter` 是数据接入与数据语义建模前端，负责数据库连接、查询、MQTT、变量组与数据点交互。

## 进入前先看

- `datacenter/package.json`
- `datacenter/src/`
- `docs/03-模块设计/datacenter/README.md`
- `docs/03-模块设计/datacenter/queries.md`

## 开发约束

- 运行时为 Vue 3 + Vite + Pinia + Element Plus + Monaco，图标统一使用 `unplugin-icons`。
- 组件使用 `PascalCase`，文件使用 `kebab-case`，composable 使用 `useXxx`。
- 所有 API 调用必须走统一 `request` 封装，Token 与租户信息仅从 Storage 读取。
- 连接模型统一为 `type=relational|mqtt`，具体数据库类型从 `config.dbType` 读取。
- WebSocket 订阅在组件卸载或切换时及时解绑，避免深度 watch 带来额外开销。
- 复杂业务逻辑优先拆到 composable，不直接堆在组件里。

## 禁止事项

- 不要突破模块边界去实现运行时渲染。
- 不要为当前任务扩大量新的连接器类型。
- 不要绕过现有数据域接口直接在前端拼接协议语义。

## 验证命令

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter lint`

## 相关契约

- `docs/03-模块设计/datacenter/README.md`
- `docs/03-模块设计/datacenter/queries.md`
- `docs/03-模块设计/data_service/README.md`
