# MQTT 实现说明

本文档面向内部开发，描述当前 MQTT 能力边界与关键实现点（高层）。

## 当前能力

- MQTT 连接：创建、编辑、删除、测试、启停、状态查询
- 主题订阅：创建、编辑、删除、启停、消息实时查看
- 变量管理：变量组/变量 CRUD、排序、启停、值解析
- 变量监控：实时值展示、更新时间、质量状态
- 实时推送：Socket.IO 推送连接状态、订阅状态、消息与变量值
- 数据点：MQTT 变量和订阅自动生成对应数据点

## 前端组件

| 组件                     | 说明                                     |
| ------------------------ | ---------------------------------------- |
| `MqttSubscriptionList`   | 订阅列表管理，支持创建、编辑、删除、启停 |
| `MqttSubscriptionDialog` | 订阅创建/编辑对话框                      |
| `MqttMessageViewer`      | 订阅消息实时查看器                       |
| `MqttTagList`            | 变量列表管理，支持分组、CRUD、排序       |
| `MqttTagGroupDialog`     | 变量组创建/编辑对话框                    |
| `MqttTagDialog`          | 变量创建/编辑对话框                      |
| `MqttTagMonitor`         | 变量实时监控面板                         |
| `TagItem`                | 单个变量展示组件                         |

## 变量自动发现与批量导入（规划中）

支持多种方式获取变量定义，实现变量动态管理：

| 方式                 | 说明                                               |
| -------------------- | -------------------------------------------------- |
| **MQTT 自动发现**    | 通过 Tag/Data 双主题模式，自动发现变量并解析实时值 |
| **RESTful API 拉取** | 从第三方系统接口同步变量列表，支持定时同步         |
| **CSV 文件导入**     | 批量导入离线变量清单，支持 Excel 格式              |

详细设计请参考：[变量自动发现与批量导入](./mqtt-auto-discovery.md)

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

| 事件                       | 说明         |
| -------------------------- | ------------ |
| `mqtt:message`             | 订阅主题消息 |
| `mqtt:subscription:status` | 订阅状态变更 |
| `mqtt:connection:status`   | 连接状态变更 |
| `mqtt:tag:value`           | 变量值更新   |

## 数据点集成

MQTT 变量和订阅自动生成对应数据点：

| 操作     | 数据点行为                                         |
| -------- | -------------------------------------------------- |
| 创建变量 | 自动创建数据点 `mqtt.{连接名}.{变量组名}.{变量名}` |
| 创建订阅 | 自动创建数据点 `mqtt.{连接名}.{订阅名}`            |
| 修改变量 | 自动更新数据点属性                                 |
| 删除变量 | 标记数据点为失效                                   |

设计器可通过数据点统一接口使用 MQTT 变量数据。

详细设计请参考：[数据点方案设计](./datapoint-design.md)

## 后续规划

- 变量回写能力
- 协议扩展（OPC UA、Modbus 等）
- 变量自动发现与批量导入
