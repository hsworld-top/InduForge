# 数据库设计

## 1. 现有表结构分析

### 当前已实现核心实体

- 平台域：`tenants`、`users`、`projects`、`logs`
- 设计域：`design_pages`、`design_project_settings`、`design_assets`、`design_asset_folders`
- 数据域：`data_connections`、`data_queries`、`data_points`、`data_mqtt_*`、`data_relational_configs`
- 发布运维域：`deployments`、`nodes`、`node_deployments`、`node_commands`

### 现状判断

- 现有表覆盖面已经足够支撑本期，不建议大规模推翻。
- 重点应放在字段语义补强和约束统一。

## 2. 核心实体

### 工程实体

- 工程 `projects`
- 页面 `design_pages`
- 工程设置 `design_project_settings`

### 数据实体

- 连接 `data_connections`
- 查询 `data_queries`
- 数据点 `data_points`

### 运维实体

- 发布版本 `deployments`
- 节点 `nodes`
- 节点部署 `node_deployments`
- 节点命令 `node_commands`

## 3. 表设计建议

### 3.1 `projects`

建议继续作为工程主表，保留：

- 名称、描述、颜色标签
- 租户归属
- 创建人/更新时间
- 工程级变量入口

建议明确：

- `entryConfig` 的结构边界
- 导入导出时的来源版本追踪

### 3.2 `design_pages`

建议继续作为页面树主表，保留：

- 页面层级
- 页面类型
- 页面排序
- 页面 Schema 内容

建议补强：

- `schemaVersion`
- 页面根节点与入口一致性校验
- 页面锁辅助字段和超时策略说明

### 3.3 `design_project_settings`

建议作为工程级设置聚合表，统一承载：

- 全局变量
- 全局脚本
- 国际化资源
- 主题配置
- 工程入口相关元数据

### 3.4 `data_points`

建议明确字段：

- `sourceType`
- `sourceId`
- `path`
- `status`
- `dataType`
- `lastSeenAt`

索引建议：

- `(projectId, path)` 唯一或半唯一约束
- `(projectId, status)` 普通索引
- `(projectId, sourceType, sourceId)` 普通索引

### 3.5 `deployments`

建议明确字段语义：

- `mode`
- `status`
- `artifactUrl`
- `artifactHash`
- `artifactSize`
- `manifest`
- `errorMessage`
- `buildLog`
- `completedAt`

建议新增或补充：

- `runtimeVersion`
- `schemaVersion`
- `entrySnapshot`
- `assetSummary`

### 3.6 `node_deployments`

建议明确：

- 一个节点与一个工程在同一时刻只保留一条有效部署关系
- `mode`、`status`、`runtimeConfig`、`deployLog` 为核心字段

建议增加：

- `lastHeartbeatAt`
- `runtimeHealth`
- `lastStatusReason`

### 3.7 `node_commands`

建议作为节点命令队列表长期保留，补强以下字段语义：

- 命令来源
- 命令类型
- 重试次数
- 超时秒数
- 结果错误摘要

## 4. 字段、主键、索引、约束建议

- 所有主业务表继续使用 UUID 主键
- `deployments(projectId, version)` 保持强唯一语义
- `node_deployments(nodeId, projectId)` 建议保持单活约束
- `data_points(projectId, path)` 建议增加唯一性检查
- `nodes(tenantId, name)` 应保持租户内名称唯一

## 5. 表关系

- 一个租户有多个用户、工程、节点
- 一个工程有多个页面、连接、查询、数据点、发布版本
- 一个发布版本可对应多个节点部署
- 一个节点部署可关联多个命令

## 6. 数据兼容与迁移建议

### 当前已实现

- 现有表结构已可用

### 共创后建议目标

- 采取增量迁移，不做破坏式重构
- 对字段补充优先采用可空新增 + 兼容回填方式
- 对唯一约束相关表，先清历史脏数据后再收紧约束

## 7. 待确认事项

- `design_project_settings` 是否统一收口国际化与主题资源
- Runtime 运行状态是否需要独立持久化表
- 是否需要新增制品元信息独立表而不是继续聚合在 `deployments.manifest`
