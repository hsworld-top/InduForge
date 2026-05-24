# 数据连接管理

本文档面向内部开发，概述 DataCenter 的连接类型、配置结构与当前实现范围。

## 连接分类

- **关系数据库（relational）**：MySQL / PostgreSQL / SQL Server
- **消息连接（mqtt）**：MQTT Broker

连接主表为 `data_connections`，具体配置存储在：

- 关系库：`data_relational_configs`
- MQTT：`data_mqtt_configs`

## 连接模型（高层）

### data_connections

- `type`: `relational` 或 `mqtt`
- `category`: `database` / `message`
- `status`: `connected` / `disconnected` / `error` / `unknown`

### 关系库配置（data_relational_configs）

共性字段：

- `dbType`: `mysql` / `postgresql` / `sqlserver`
- `host` / `port` / `database` / `username` / `password`
- `queryTimeout`、`connectionTimeout`、`charset`
- SSL 字段（PostgreSQL 为主）：`ssl`, `sslConfig`, `sslMode`, `sslCa`, `sslCert`, `sslKey`

### MQTT 配置（data_mqtt_configs）

- `brokerUrl`、`protocol`（`mqtt`/`mqtts`/`ws`/`wss`）
- `port`、`clientId`、`username`、`password`
- `keepalive`、`cleanSession`、`qos`
- `reconnectPeriod`、`connectTimeout`
- `will`、`sslConfig`

## 当前实现能力

- 连接列表与 CRUD
- 连接测试
- 连接状态更新
- MQTT 连接启停与状态查询
