# Allen-Bradley EtherNet/IP CIP 采集调试驱动设计

## 目标

基于 HSL Demo 12.9.0 的 `AllenBradleyNet` 接入 Logix 系列 EtherNet/IP CIP 标签访问，覆盖连接测试、长连接、主动断开和点位读取。

## 驱动边界

- `driverId`：`allen-bradley.ethernet-ip`
- `protocolFamily`：`allen-bradley`
- 默认端口：`44818`
- 适用 Demo 中 `AllenBradleyNet` 支持的 ControlLogix、CompactLogix、GuardLogix、SoftLogix 和 Logix Emulate。
- 首期不混入 `AllenBradleyPcccNet`、`AllenBradleyConnectedCipNet`、MicroCIP 或 SLC，它们后续作为独立驱动接入。

## 连接配置

- `host`
- `port`，默认 `44818`
- `slot`，默认 `0`
- `messageRouter`，可选，例如 `1.15.2.18.1.12`
- `connectTimeoutMs`，默认 `5000`
- `receiveTimeoutMs`，默认 `5000`
- `contextCheck`，默认 `false`
- `readArrayUseSegment`，默认 `true`
- `dataFormat`，默认 `DCBA`

自定义消息路由必须是偶数个 `0..255` 数字段，以点号分隔。设置后由路由定义覆盖常规机架槽位路径。

## 点位地址与类型

地址直接使用 PLC 标签，例如 `Temperature`、`ArrayTag[10]`、`Program:MainProgram.LocalTag`，也允许 HSL 支持的 `slot=2;TagName` 地址前缀。地址去除首尾空格，拒绝空值、内部空白和超过 512 字符的内容。

支持 `bool`、`int8`、`uint8`、`int16`、`uint16`、`int32`、`uint32`、`int64`、`uint64`、`float32`、`float64`、`string`、`bytes`、`datetime`。数组元素数上限为 `65535`；字符串使用 PLC 标签内的实际长度读取。

## 会话与错误

- 连接成功后复用同一 EtherNet/IP 会话。
- 批量调试逐点读取，单个标签失败不影响同批其他标签。
- HSL 错误转换为稳定的 `ALLEN_BRADLEY_*` 错误码，并保留底层错误消息。
- 离开工业采集界面仍由现有通用会话生命周期关闭连接。

## 前后端

- 新增 Schema 版本 `1` 的协议目录。
- Data Service 增加标签地址、元素数量保存校验和地址文本格式化。
- Datacenter 继续复用通用连接表单和变量配置，只补中文名称。

## 首期不做

- 写入标签。
- 标签浏览与结构体递归枚举。
- 多标签 CIP 合并读取。
- PCCC、Connected CIP、MicroCIP 和 SLC。
