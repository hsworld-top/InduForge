# 数据中心 MQTT 接入源工作台设计

> 日期：2026-05-21  
> 范围：`datacenter/` 新版接入源工作台中的 MQTT 查看消息、变量管理与旧 MQTT 入口清理。  
> 目标：以新版接入源工作台为唯一 MQTT 操作入口，完成订阅、消息查看、变量管理、实时监控和资源清理闭环。

## 1. 背景

`datacenter` 已有新版接入源工作台，并在 `AccessSourceWorkbench.vue` 中将 `type=mqtt` 路由到 `MqttWorkbenchPanel.vue`。当前仓库还保留旧版 `DataCenterNew.vue` 中的 MQTT tab 逻辑，以及 `components/mqtt/` 下的消息查看、订阅管理、变量管理和监控组件。

本轮不再维护旧入口兼容。旧版实现只作为行为参考：可复用的 API、socket、preview session、弹窗和业务校验继续复用；不符合新版工作台风格或生命周期要求的部分允许修改、调整或重做。完成后旧版 MQTT tab 逻辑应清理，避免两套入口并存。

## 2. 成功标准

- 从新版接入源列表打开 MQTT 接入源后，工作台默认展示订阅管理。
- 点击订阅行、右侧快捷动作或订阅管理表格中的“查看消息”，打开该订阅的消息查看页签。
- 消息查看页签能加载历史消息，并通过 preview session + MQTT socket 接收实时消息。
- 点击“变量管理”后打开变量管理页签，支持变量分组、变量 CRUD、启停变量和实时值监控。
- 创建、更新、删除或启停变量后，变量列表和监控区同步更新。
- MQTT 变量继续生成 `mqtt.tag` 数据点，数据点列表可以按既有逻辑读取。
- 关闭消息页签、切换接入源或卸载工作台时，实时消息订阅、变量订阅和 preview session 被清理。
- UI 与新版 datacenter 工作台家族化风格一致，使用 `--dc-*` 变量、紧凑工具栏、左树中台右详情布局和图标按钮。
- `pnpm --dir datacenter build` 通过。

## 3. 不做范围

- 不新增后端接口。
- 不做 Kafka、HTTP、WebSocket、Redis 的真实 preview 行为。
- 不保留旧 `DataCenterNew.vue` 的 MQTT tab 兼容入口。
- 不引入新的组件库、图标库或全局状态方案。
- 不重写 MQTT 连接、订阅、Tag、数据点等已有业务接口。
- 不实现运行态 MQTT 消费，仅覆盖开发态工作台 preview。
- 不把 MQTT 业务 API 封装成跨协议抽象。

## 4. 整体架构

### 4.1 新版工作台入口

`AccessSourceWorkbench.vue` 继续按协议类型选择子工作台。MQTT 类型进入 `components/access-source/workbench/MqttWorkbenchPanel.vue`，再挂载 `components/mqtt/MqttWorkbench.vue`。

`MqttWorkbench.vue` 是本轮 MQTT 的唯一业务容器，负责：

- 接收 `projectId` 与当前 MQTT 接入源。
- 创建和维护 preview session。
- 启动 MQTT 预览连接。
- 加载订阅树。
- 管理工作台页签。
- 在页签关闭、工作台卸载时清理 socket 订阅。
- 组织消息查看、变量管理和右侧 inspector。

### 4.2 复用与重做边界

继续复用：

- `data.api.ts` 中 MQTT 订阅、消息、变量组、变量和当前值接口。
- `usePreviewSession` 的 preview session 生命周期。
- `useMqttSocket` 的共享 socket、消息订阅和 tag 订阅能力。
- `MqttSubscriptionDialog`、`MqttTagDialog`、`MqttTagGroupDialog` 中已存在的表单校验和提交逻辑。
- `useMqttTagSync` 在同一订阅内同步变量变化。

允许修改或重做：

- `MqttMessageViewer.vue` 的视觉、布局和消息渲染。
- `MqttTagList.vue`、`MqttTagMonitor.vue` 的工作台外观和组件职责。
- `TagItem.vue` 的视觉结构。
- `DataCenterNew.vue` 中所有旧 MQTT tab 入口和清理逻辑。

## 5. 可复用公共组件

本轮只抽取明确会被后续协议 preview 复用、且不包含 MQTT 业务语义的组件。公共组件应放在 `datacenter/src/components/workbench/` 或现有同等公共目录下，命名保持 PascalCase。

建议抽取：

- `WorkbenchStreamToolbar`：搜索、数量限制、清空、刷新、连接状态等通用消息流工具栏。
- `WorkbenchStreamMessageList`：消息列表、空态、加载态、复制、JSON 格式化、时间显示。
- `WorkbenchStatusPill`：工作台内紧凑状态标签，统一 success、info、warning、danger 视觉。

不抽取：

- MQTT 订阅树。
- MQTT Tag 解析配置。
- MQTT 变量组。
- MQTT 数据点同步。
- MQTT socket 事件名与订阅参数。

抽取原则：公共组件只接收普通数据和事件，不调用 MQTT API，不持有 preview session，不知道 `subscriptionId` 或 `tagId` 的业务含义。

## 6. 交互设计

### 6.1 布局

MQTT 工作台采用三栏结构：

- 左侧 explorer：Broker 摘要、返回接入源、订阅管理、刷新、启动预览、订阅搜索、订阅树。
- 中间 main：页签栏和当前功能面板。
- 右侧 inspector：当前 Broker 状态、选中订阅详情、快捷动作。

小屏幕下隐藏右侧 inspector；更窄屏幕下左侧 explorer 堆叠到上方。布局不使用营销式大卡片，保持工作台密度。

### 6.2 订阅管理

默认页签为“订阅管理”。订阅管理负责创建、编辑、删除订阅，并向工作台抛出：

- `subscription-select`
- `view-messages`
- `manage-tags`
- `subscription-deleted`

删除订阅后，工作台关闭对应消息页签和变量页签，并刷新左侧订阅树。

### 6.3 查看消息

打开消息页签前，工作台先确保 preview session 存在，再调用启动 MQTT 连接接口。启动失败时保持当前页签不变，并展示错误消息。

消息页签功能：

- 加载当前订阅历史消息。
- 接收实时消息并插入列表顶部。
- 支持按 Topic 或 Payload 搜索。
- 支持限制显示条数，默认显示 100 条，内存最多保留 1000 条。
- 支持复制 payload。
- 支持 JSON 自动格式化与原文查看。
- 支持清空当前前端消息列表。
- 显示连接状态、消息总数、最后消息时间。

消息列表只负责展示和本地交互；实时订阅由工作台统一建立和清理，避免组件卸载顺序导致 socket 泄漏。

### 6.4 变量管理

变量管理页签以当前订阅为上下文，左右分栏：

- 左侧变量配置：变量组、未分组变量、变量 CRUD、变量启停、搜索。
- 右侧实时监控：启用变量列表、当前值、质量、更新时间、解析错误。

变量配置面板操作成功后通过 `useMqttTagSync(subscriptionId)` 通知监控面板刷新。监控面板订阅启用变量的实时值，变量停用或删除时立即取消对应 tag 订阅。

### 6.5 标签页规则

页签 id 规则：

- 订阅管理：`mqtt-subscriptions-${connection.id}`
- 消息查看：`mqtt-messages-${subscription.id}`
- 变量管理：`mqtt-tags-${subscription.id}`

同一个订阅的同类页签只打开一个。再次点击时激活已有页签。

## 7. 数据流

### 7.1 消息查看

1. 用户点击“查看消息”。
2. `MqttWorkbench` 调用 `ensureSession()`。
3. `MqttWorkbench` 调用 `startMqttConnection(projectId, connectionId)`。
4. `MqttWorkbench` 打开消息页签。
5. 消息面板调用 `getMqttSubscriptionMessages(projectId, subscriptionId)` 加载历史消息。
6. `MqttWorkbench` 通过 `subscribeMessages(subscriptionId, handler)` 订阅实时消息。
7. handler 将 `data.message` 转交消息面板公开方法。
8. 页签关闭时调用对应 cleanup。

### 7.2 变量管理

1. 用户点击“变量管理”。
2. `MqttWorkbench` 确保 preview session 和 MQTT 连接可用。
3. 工作台打开变量管理页签。
4. 变量配置面板调用变量组和变量接口加载配置。
5. 监控面板加载启用变量。
6. 监控面板通过 `subscribeTag(tagId)` 订阅实时值。
7. 变量配置变化后发送 tag sync 事件。
8. 监控面板收到 sync 后刷新或局部更新。
9. 页签关闭或组件卸载时取消全部 tag 订阅。

## 8. 错误处理

- preview session 创建失败：展示“预览会话不可用”，不打开目标页签。
- MQTT 启动失败：展示后端错误消息，不打开需要实时能力的页签。
- 历史消息加载失败：保留空态，展示错误消息，允许用户重试。
- 变量或变量组加载失败：保留面板，展示错误消息，允许刷新。
- 实时 socket 未连接：面板显示“未连接”，但不阻断历史消息和变量配置查看。
- JSON 格式化失败：展示原始 payload，不抛出 UI 错误。
- 删除订阅后：无论关联页签是否存在，都执行幂等关闭。

所有 API 错误消息通过 `getApiErrorMessage` 解析。

## 9. 旧入口清理

`DataCenterNew.vue` 中旧 MQTT 专属逻辑应删除或收敛，包括：

- MQTT 消息查看器和变量管理 tab 模板分支。
- `MqttMessageViewer`、`MqttTagList`、`MqttTagMonitor` 的旧入口 import。
- `mqttMessageViewerRefs`、`mqttSubscriptionListRefs` 中只服务旧 MQTT tab 的引用管理。
- `openMqttMessageViewer`、`openMqttTagManager`、`handleMqttSubscriptionView`、`handleMqttSubscriptionManage` 等旧 tab 打开函数。
- 旧 tab 关闭时的 MQTT 特殊 unsubscribe 逻辑。

如果仍有连接树右键菜单使用这些事件，应改为打开新版接入源工作台，或删除事件绑定，保证用户从接入源工作台完成 MQTT 操作。

## 10. 验证

### 静态验证

- 运行 `pnpm --dir datacenter build`。
- 如果新增或调整纯函数、composable、消息格式化逻辑，补充或更新 vitest，并运行对应测试。

### 手动验收路径

- 打开 MQTT 接入源，默认出现订阅管理页签。
- 新建订阅后左侧订阅树与订阅管理表格同步刷新。
- 对订阅点击“查看消息”，历史消息加载，实时消息进入列表。
- 关闭消息页签后再次推送消息，不再写入已关闭页签。
- 对订阅点击“变量管理”，创建变量并启用，监控区出现实时值。
- 停用或删除变量后，监控区不再订阅该变量。
- 删除订阅后，对应消息页签和变量页签关闭。
- 返回接入源列表再重新打开 MQTT 工作台，preview session 不残留旧状态。

## 11. 风险与控制

- 旧组件存在 Tailwind 与新版 `--dc-*` 风格混用风险。控制方式：本轮改造进入工作台的可见组件样式，保留表单业务逻辑。
- `useMqttSocket` 为共享连接，多个面板同时订阅时必须依赖 cleanup 和引用计数。控制方式：实时订阅集中由工作台或监控面板建立，关闭时逐项释放。
- `usePreviewSession` 卸载时会销毁 session。控制方式：MQTT 工作台作为 session 生命周期边界，不让子组件自行创建独立 session。
- 删除旧 MQTT tab 逻辑可能影响旧连接树右键入口。控制方式：右键入口统一导向新版工作台，删除只服务旧 tab 的事件链。

