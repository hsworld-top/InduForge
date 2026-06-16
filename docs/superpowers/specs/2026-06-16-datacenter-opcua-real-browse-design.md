# OPC UA 真实地址空间浏览设计

## 目标

OPC UA 工作台的浏览导入必须从真实 OPC UA Server 地址空间读取节点树，不再用已建模变量伪造浏览结果。已建模变量仅用于给真实浏览节点打 `modeled=true` 标记，帮助前端区分“已导入”和“可导入”。

## 范围

- 后端 `data_service` 增加 OPC UA 地址空间浏览适配器。
- `BrowseOpcua` 接口保持现有响应结构：`nodes` 与 `diagnostics`。
- 支持 `SecurityPolicy=None`，并按现有连接配置支持常见安全策略与 `None`、`Sign`、`SignAndEncrypt` 模式。
- 支持匿名认证与用户名密码认证。
- 不新增前端证书管理 UI；证书、私钥、CA 等材料优先从 `options.sslConfig` 或顶层 `sslConfig` 读取。
- 不把 OPC UA 读取值、订阅功能一并真实化。

## 数据流

1. 前端创建开发态 OPC UA 会话。
2. 前端调用 `/browse`。
3. 会话服务校验 session，读取已建模 OPC UA 节点生成 NodeId 集合。
4. 真实浏览适配器使用 session 配置连接 OPC UA Server，从 `ObjectsFolder` 开始递归浏览地址空间。
5. 适配器返回 folder/object/variable 节点；变量节点读取 `DataType` 属性并映射为前端可识别文本。
6. 会话服务按 NodeId 标记 `modeled` 并返回结果。

## 错误处理

- endpoint 为空、配置不合法、连接失败、browse 失败时返回业务错误，不静默回退为已建模变量。
- diagnostics 只描述真实浏览限制或截断信息，不再出现“当前浏览结果来自已建模变量”。
- 默认限制最大深度与节点数量，避免大地址空间导致接口长时间阻塞。配置可通过 `options.browseMaxDepth`、`options.browseMaxNodes`、`options.connectTimeoutMs` 覆盖。

## 成功标准

- `/browse` 返回的树来自 OPC UA Server。
- 已建模 NodeId 在真实节点中显示为 `modeled=true`。
- 真实 browse 失败时前端收到明确错误，不再展示旧占位结果。
- `data_service` 相关 Go 测试通过。
