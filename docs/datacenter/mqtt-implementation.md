# MQTT 实现说明（当前）

本文档面向内部开发，描述当前 MQTT 能力边界与关键实现点（高层）。

## 当前能力

- MQTT 连接：创建、测试、启停、状态查询
- 主题订阅：创建、启停、消息查看
- 变量管理：变量组/变量 CRUD、排序、启停
- 实时推送：Socket.IO 推送连接状态、订阅状态、消息与变量值

## 关键表

- `data_connections`：连接主表（`type=mqtt`）
- `data_mqtt_configs`：MQTT 配置
- `data_mqtt_subscriptions`：主题订阅
- `data_mqtt_tag_groups`：变量分组
- `data_mqtt_tags`：变量定义与解析规则

## 解析模型（高层）

- `parseType`: `jsonpath` / `regex` / `script` / `fixed`
- `parseRule`: 解析规则内容
- `dataType`: `string` / `number` / `boolean` / `object` / `array`

## 实时推送事件

| 事件 | 说明 |
| --- | --- |
| `mqtt:message` | 订阅主题消息 |
| `mqtt:subscription:status` | 订阅状态变更 |
| `mqtt:connection:status` | 连接状态变更 |
| `mqtt:tag:value` | 变量值更新 |

## 数据点集成（规划中）

MQTT 变量将自动生成对应数据点：

| 操作 | 数据点行为 |
|------|-----------|
| 创建变量 | 自动创建数据点 `mqtt.{连接名}.{变量组名}.{变量名}` |
| 修改变量 | 自动更新数据点属性 |
| 删除变量 | 标记数据点为失效 |

设计器可通过数据点统一接口使用 MQTT 变量数据。

详细设计请参考：[数据点方案设计](./datapoint-design.md)

## 后续规划

- 变量回写能力
- 协议扩展（OPC UA、Modbus 等）
- 数据点自动生成

