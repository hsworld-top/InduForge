# 数据连接管理

本文描述数据中心开发态连接的最终模型。连接配置用于工作台预览、查询建模、字段映射和发布契约；节点运行连接不在本模块维护。

## 统一读模型

`GET /api/v1/data/projects/{projectId}/connections` 是普通接入源唯一的列表读模型，服务端负责分页、搜索、排序和生成点计数。响应将状态拆成四个互不混用的维度：

- `enabled`：配置是否启用。
- `configurationState: ready|incomplete`：必填配置是否完整。
- `testCapability: supported|unsupported`：是否支持来源级连通测试。
- `lastTest: not_tested|succeeded|failed`：最近一次已保存连接测试摘要。

预览会话、MQTT 临时连接和 WebSocket 开发态会话状态只在对应工作台返回，不写入连接主表、快照或 Artifact。HTTP、WebSocket 的地址属于请求或会话，不支持来源级测试。

## 连接与密钥

普通连接包含内置关系库、时序库、实时库和消息库，以及 MySQL、PostgreSQL、SQL Server、MQTT、Kafka、HTTP、WebSocket、Redis 和 TDengine。协议配置使用各自结构化明细；专用类型只能由专用更新接口修改。

密码、Token、敏感 Header、证书和私钥统一保存到 `data_connection_secrets`，使用 AES-GCM 加密。读取 API 仅返回 `secretStatus`；更新时：

- `secrets` 表示新增或替换。
- `clearSecretKeys` 表示显式删除。
- 未提交某个密钥表示保持原值。

快照只往返密文与密钥版本，Artifact 只输出密钥引用，不输出明文。

MySQL、PostgreSQL 和 SQL Server 使用统一的 `sslConfig`：`mode` 支持禁用、优先、仅加密、验证 CA 和完整验证，具体可选项按驱动能力收敛；`ca`、`cert`、`key` 只作为密钥输入，不进入普通连接 JSON。运行时分别转换为 MySQL 驱动 TLS 注册配置、pgx TLS 配置和 SQL Server TDS TLS 配置。编辑时未提交证书内容表示保留现有密钥。

## 输出映射

SQL 查询、HTTP 请求、WebSocket 会话和实时 Key 使用规范化 `data_source_output_mappings`。一条映射稳定关联一个生成数据点，保存输出键、显示名称、规范数据类型、单位、精度和选择器：

- `whole`：保存整个结果；SQL 整包为 `array`，HTTP/WS/Redis 整包按实际结构为 `object` 或 `array`。
- `column`：SQL 单行字段；查询返回零行或多行时读取失败。
- `path`：字符串/非负整数路径片段，例如 `["data", "items", 0, "value.with.dot"]`。

映射修改保持映射 ID 和数据点 ID 稳定。删除前检查报警、计算和历史引用；无引用时生成点标记为 `invalid`。类型变化会重新校验下游兼容性。

Kafka 字段、MQTT 映射和工业采集点使用各自规范表，但数据类型和路径片段语义与上述映射一致。所有可增长对象均由服务端分页或按父目录加载。

## 开发态测试边界

- 未保存配置使用创建向导测试接口；已保存连接使用连接 ID 测试接口。
- 测试与预览是独立外部操作，不与配置保存伪装为同一数据库事务。
- TDengine 使用官方 WebSocket Go 驱动和独立只读 SQL 适配器。
- SQL 工作台限制 30 秒、500 行和 5 MiB，并返回是否截断及原因。
- 工业连接由开发调试代理测试；未选代理、代理离线、缺驱动或契约不匹配时入口禁用并显示原因。
