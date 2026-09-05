# datacenter 协作规则

## 模块定位

- `datacenter` 是数据接入与数据语义建模前端，负责数据库连接、查询、MQTT、变量组与数据点交互。

## 按需参考

以下路径均相对仓库根目录，仅在任务涉及对应内容时查阅。

- 数据中心职责与页面边界：`docs/03-模块设计/datacenter/README.md`。
- 连接配置与密钥：`docs/03-模块设计/datacenter/connections.md`、`datacenter/src/api/schemas/connection.schema.ts`。
- 查询建模：`docs/03-模块设计/datacenter/queries.md`。
- 数据域服务边界：`docs/03-模块设计/data_service/README.md`。

## 开发约束

- 运行时为 Vue 3 + Vite + Pinia + Element Plus + Monaco，图标统一使用 `unplugin-icons`。
- Vue 组件及其文件使用 `PascalCase`，composable 使用 `useXxx`；其他文件沿用所在目录的命名方式。
- API 调用使用统一 `request` 封装和同源 `HttpOnly` Cookie，不在前端存储或传递 JWT；工程、租户及编辑代次从当前 Wujie 上下文读取，不以 Storage 旧值替代。
- 连接类型与结构化配置以连接设计和当前 API schema 为准，调整时同步核对后端契约。
- WebSocket 订阅在组件卸载或切换时及时解绑，避免深度 watch 带来额外开销。
- 复杂业务逻辑优先拆到 composable，不直接堆在组件里。

## 禁止事项

- 不要突破模块边界去实现运行时渲染。
- 不要在任务范围外新增连接器类型。
- 不要绕过现有数据域接口直接在前端拼接协议语义。

## 验证命令

- 从仓库根目录按变更面选用，不要求每次全跑；测试命令可追加相关测试文件路径。
- TypeScript 改动：`pnpm --dir datacenter typecheck`。
- `pnpm --dir datacenter test`
- `pnpm --dir datacenter build`
- `pnpm --dir datacenter lint`
