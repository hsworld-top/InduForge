# 数据库设计（概览）

本文档面向内部开发，提供 InduForge 主要业务表的高层结构与关系说明。

> **详细设计**：完整的数据库 Schema 设计请参阅 [高层设计 - 数据库设计](./高层设计.md#9-数据库设计)。

## 核心实体

| 模块 | 主要表 | 说明 |
| --- | --- | --- |
| 租户与用户 | `tenants`, `users` | 多租户与用户体系 |
| 工程 | `projects`, `project_members` | 工程元数据与成员 |
| 日志 | `logs` | 系统日志 |
| 字典 | `dictionaries`, `dictionary_items` | 数据字典 |
| 设计器 | `design_pages`, `design_assets` | 页面树、资源管理 |
| 数据中心 | `data_connections`, `data_points` | 连接主表、数据点 |
| 关系库配置 | `data_relational_configs` | 数据库连接配置 |
| MQTT 配置 | `data_mqtt_configs` | MQTT 连接配置 |
| MQTT 订阅 | `data_mqtt_subscriptions` | 订阅主题 |
| MQTT 变量组 | `data_mqtt_tag_groups` | Tag 分组 |
| MQTT 变量 | `data_mqtt_tags` | Tag 定义与解析规则 |
| 数据查询 | `data_queries`, `data_sql_configs` | 查询定义与 SQL 配置 |
| 告警系统 | `alarm_rules`, `alarm_records` | 告警规则与记录 |
| 发布部署 | `deployments`, `deployment_pages` | 发布版本与页面快照 |

## 关键关系

```
tenants ──1:N──> users
tenants ──1:N──> projects
projects ──1:N──> data_connections
projects ──1:N──> design_pages
projects ──1:N──> data_points
projects ──1:N──> deployments

data_connections ──1:1──> data_relational_configs
data_connections ──1:1──> data_mqtt_configs
data_connections ──1:N──> data_mqtt_subscriptions
data_mqtt_subscriptions ──1:N──> data_mqtt_tags
data_mqtt_tag_groups ──1:N──> data_mqtt_tags

data_queries ──1:N──> data_query_logs
deployments ──1:N──> deployment_pages
```

## 设计要点

- `data_connections.type` 使用 `relational|mqtt|websocket|opcua|modbus|http|s7`
- `data_connections.category` 用于区分数据库/消息/协议/API
- MQTT 变量值不落库，实时值通过 Redis + Socket.IO 推送
- `data_points` 作为统一抽象层，支持 `status: active/invalid`
- `design_pages` 支持多端视图（`logicalId`, `target`, `isDefaultTarget`）

## 相关文档

- [高层设计 - 数据库设计](./高层设计.md#9-数据库设计) - 完整表结构
- [init.sql](../dev_core/database/init.sql) - 数据库初始化脚本

---

**版本**: 3.0.0  
**最后更新**: 2026-01
