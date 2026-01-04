# 数据库设计（概览）

本文档面向内部开发，提供 InduForge 主要业务表的高层结构与关系说明。详细字段以 `dev_core/database/init.sql` 为准。

## 核心实体

| 模块 | 主要表 | 说明 |
| --- | --- | --- |
| 租户与用户 | `tenants`, `users` | 多租户与用户体系 |
| 工程 | `projects` | 工程元数据 |
| 日志 | `logs` | 系统日志 |
| 设计器 | `design_pages` | 页面树与 Schema |
| 数据中心 | `data_connections` | 连接主表（关系库/MQTT 等） |
| 关系库配置 | `data_relational_configs` | 数据库连接配置 |
| MQTT 配置 | `data_mqtt_configs` | MQTT 连接配置 |
| MQTT 订阅 | `data_mqtt_subscriptions` | 订阅主题 |
| MQTT 变量组 | `data_mqtt_tag_groups` | Tag 分组 |
| MQTT 变量 | `data_mqtt_tags` | Tag 定义与解析规则 |
| 数据查询 | `data_queries` | 查询定义 |
| 查询日志 | `data_query_logs` | 查询执行记录 |
| SQL 配置 | `data_sql_configs` | 预置 SQL 配置 |

## 关键关系

- `tenants` 1:N `users`
- `tenants` 1:N `projects`
- `projects` 1:N `data_connections`
- `data_connections` 1:1 `data_relational_configs`
- `data_connections` 1:1 `data_mqtt_configs`
- `data_connections` 1:N `data_mqtt_subscriptions`
- `projects` 1:N `data_mqtt_subscriptions`
- `data_mqtt_subscriptions` 1:N `data_mqtt_tags`
- `data_mqtt_tag_groups` 1:N `data_mqtt_tags`
- `projects` 1:N `data_queries`
- `data_queries` 1:N `data_query_logs`
- `projects` 1:N `design_pages`

## 设计要点

- `data_connections.type` 使用 `relational|mqtt|websocket|opcua|modbus|http|s7`
- `data_connections.category` 用于区分数据库/消息/协议/API
- MQTT 变量值不落库，实时值通过 Redis + Socket.IO 推送

