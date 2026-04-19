# NodeAgent 总览

## 模块定位
- NodeAgent 是节点侧控制面，负责节点初始化、中心绑定、命令执行、部署落地与本地运维入口。

## 当前已实现
- 节点注册与审批协同。
- 心跳与命令轮询。
- 部署包下载、执行与状态回传主干。
- 本地前端管理页面与静态资源嵌入。

## 当前边界
- NodeAgent 是运行时宿主，不负责解释低代码页面 Schema。
- 真正的页面运行能力依赖 RuntimeEngine。

## 当前重点
- 与 RuntimeEngine 建立稳定启动协议。
- 规范版本目录、探活与回滚策略。
- 统一本地状态和平台状态。

## 关联文档
- [node_agent/README](../../../../docs/node_agent/README.md)
- [node_agent_front/README](../../../../docs/node_agent_front/README.md)
- [详细设计](../../../../docs/详细设计.md)
- [runtime_node_agent.task](../../../ai-packages/tasks/runtime_node_agent.task.md)
