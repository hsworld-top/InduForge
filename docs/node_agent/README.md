# node_agent 后端概览

## 模块定位
- Go 实现的节点后端服务，负责命令执行、部署编排、本地状态和 Web API。

## 当前已实现
- 节点初始化向导。
- 中心绑定与代理。
- 部署执行器、状态存储、日志与健康检查。
- 静态前端资源服务。

## 当前重点
- 完成 RuntimeEngine 进程托管。
- 补齐探活失败、回滚失败与错误上报策略。
- 增强主链路测试。

## 关联文档
- [NodeAgent 总览](../README_nodeagent.md)
- [详细设计](../详细设计.md)
- [runtime_node_agent.task](../ai-packages/tasks/runtime_node_agent.task.md)
