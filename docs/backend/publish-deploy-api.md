# 发布部署 API

## 1. 文档定位
- 本文档描述 `dev_core` 当前的发布、部署、节点相关接口职责与下一阶段对齐方向。
- 详细产品和流程设计以 [详细设计](../详细设计.md) 与 [开发计划](../开发计划.md) 为准。

## 2. 当前已实现
### 发布相关
- 已存在项目发布入口。
- 已支持聚合页面、数据点、连接与查询并生成版本包。
- 已记录发布版本元数据。

### 部署相关
- 已支持创建部署单。
- 已支持向节点下发启动、停止、重启、回滚类命令。
- 已支持部署状态与错误信息回传。

### 节点相关
- 已支持节点注册、审批、心跳、命令轮询、执行回执。

## 3. 共创后建议目标
### 发布接口应补强
- 明确 `manifest.json` 结构。
- 增加资源清单与版本校验信息。
- 输出阻断错误与警告的区分结果。

### 部署接口应补强
- 统一部署状态枚举：`PENDING`、`DOWNLOADING`、`EXTRACTING`、`STARTING`、`RUNNING`、`FAILED`、`STOPPED`、`ROLLED_BACK`。
- 明确回滚和失败重试策略。

### 节点接口应补强
- 明确 Runtime 健康检查上报字段。
- 增加运行版本、最后错误、探活结果等字段。

## 4. 推荐阅读
- [高层设计](../高层设计.md)
- [详细设计](../详细设计.md)
- [runtime_node_agent.task](../ai-packages/tasks/runtime_node_agent.task.md)
- [runtime_engine.task](../ai-packages/tasks/runtime_engine.task.md)
