# 欧姆龙 FINS TCP/UDP 采集调试驱动设计

## 目标

基于 HSL Communication Demo 12.9.0 已提供的 `OmronFinsNet` 与 `OmronFinsUdp` 能力，接入欧姆龙 FINS TCP 和 FINS UDP。首期覆盖连接测试、长会话、主动断开和点位读取，并继续复用现有采集调试工作台。

## 驱动边界

- `omron.fins-tcp`：使用 `OmronFinsNet`，默认端口 `9600`，建立真实 TCP 长连接。
- `omron.fins-udp`：使用 `OmronFinsUdp`，协议本身无连接；打开会话时读取 CPU 状态确认 PLC 可达，后续复用同一客户端对象和 UDP 配置。
- 两个驱动共用 HSL 适配接口、连接参数模型、地址解析、点位读取会话和测试基线，避免后续维护两套类型转换逻辑。

## 连接配置

两个驱动都支持：

- `host`
- `port`，默认 `9600`
- `receiveTimeoutMs`，默认 `5000`
- `plcType`，支持 `CSCJ`、`CV`
- `readSplits`，默认 `500`，范围 `1..999`
- `gct`，默认 `2`
- `sid`，默认 `0`
- `dataFormat`，默认 `CDAB`
- `stringReverseByteWord`，默认 `true`

TCP 额外支持：

- `connectTimeoutMs`，默认 `5000`
- `receiveUntilEmpty`，默认 `false`

HSL 已自动处理 FINS 节点号，首期不暴露 `DA1`、`SA1` 等容易误配的底层字段。

## 点位地址与类型

地址直接保存 HSL 原生 FINS 地址，例如 `D100`、`C100`、`W100`、`H100`、`A100`、`D100.0`、`E0.100`。保存和 Agent 执行时去除首尾空格，并拒绝空值、内部空白字符和超过 128 字符的地址。

支持平台类型：`bool`、`int8`、`uint8`、`int16`、`uint16`、`int32`、`uint32`、`int64`、`uint64`、`float32`、`float64`、`string`、`bytes`。单变量读取换算后的协议单位不得超过 `65535`，HSL 再按 `readSplits` 自动分段。

## 错误与会话

- 连接、探测和读取错误保留 HSL 错误码与消息，并转换为稳定的 FINS 驱动错误码。
- 批量读取继续逐点执行，单点失败不影响同批其他变量。
- TCP 断开时关闭 HSL 连接；UDP 断开时释放逻辑会话和客户端资源。
- 离开工业采集界面仍由现有通用会话生命周期自动断开。

## 前后端

- 新增两个协议目录，Schema 版本保持 `1`。
- Data Service 使用通用协议目录加载，只补 FINS 地址和读取长度保存校验、地址文本格式化。
- Datacenter 继续使用通用连接表单和变量抽屉，只补中文驱动名称，不新增协议专属工作台。

## 首期不做

- 写入 PLC。
- 设备浏览。
- FINS 随机地址批量读取优化。
- PLC Run/Stop、CPU 时间和 CPU 信息等运维命令。
- 手动暴露全部 FINS 帧头字段。
