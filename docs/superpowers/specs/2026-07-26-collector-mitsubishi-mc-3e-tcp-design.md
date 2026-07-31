# Mitsubishi MC 3E TCP 采集调试驱动设计

## 目标

接入 HSL Demo 已实现的三菱 MC 3E Binary TCP 客户端，在 DevAgent 中完成连接测试、长连接、断开和点位读取，并通过协议目录自动进入 Data Service 与前端的通用连接、变量配置和当前页调试流程。

## 采用方案

- 协议族：`mitsubishi`。
- 驱动标识：`mitsubishi.mc-3e-tcp`。
- 底层实现：`HslCommunication.Profinet.Melsec.MelsecMcNet`。
- 地址直接使用三菱设备地址字符串，例如 `D100`、`M0`、`X0`、`Y0`、`W100`、`D100.3`。
- 首期不解析设备区进制和连续地址，不做错误概率较高的自动合并；同一会话复用 TCP 连接并逐点读取。
- Schema 基线版本保持 `1`，不增加历史兼容分支。

## 连接配置

- `host`：设备 IP 或主机名，必填。
- `port`：MC 协议监听端口，默认 `6000`。
- `networkNumber`：网络号，默认 `0`，范围 `0-255`。
- `networkStationNumber`：网络站号，默认 `0`，范围 `0-255`。
- `targetIoStation`：目标模块 I/O 号，默认 `1023`（`0x03FF`），范围 `0-65535`。
- `connectTimeoutMs`：连接和接收超时，默认 `5000`，范围 `100-120000`。

## 点位地址

地址 Schema 只包含 `address` 字符串。驱动去除首尾空格并拒绝空值、空白字符和超过 128 字符的地址。

支持 `bool`、整数、浮点、字符串和字节数组。单点读取按 MC 3E 批量读取边界限制：bool 最多 `7168` 个元素；其他类型最多占用 `960` 个字寄存器。

## HSL 适配

新增 `IHslMelsecMcClient`、工厂、连接参数和读取请求。HSL 类型不进入平台公共契约。适配层负责连接、关闭、数据类型读取、异常归一化和串行化访问。

## 前后端接入

协议目录新增 Manifest、连接 Schema、地址 Schema 和 UI Schema。Data Service 继续使用通用协议目录、连接保存和调试任务能力；前端继续使用 Schema 表单和通用变量表格，不新增协议专属工作台。

## 不包含

- MC ASCII、MC UDP、A1E、A3C、FX Links 和串口协议。
- 写入、设备浏览、PLC 型号读取和远程控制。
- 地址自动合并或随机读取优化。
- Runtime Collector 接入。
