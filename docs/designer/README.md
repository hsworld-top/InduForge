# designer 概览

## 模块定位
- `designer` 是低代码页面设计器，负责页面、组件、绑定、表达式与工程导出。

## 当前已实现
- 编辑器内核、页面模型、命令与历史、页面锁。
- 组件树、属性面板、事件面板、绑定面板。
- 数据服务与表达式基础能力。

## 当前边界
- 负责设计态与发布态导出，不负责部署执行。
- 运行时页面解释不在本模块内部完成。

## 当前重点
- 冻结发布态 Schema。
- 区分编辑态模型与运行态导出模型。
- 与 `publishService` 和 `runtime_engine` 对齐契约。

## 关联文档
- [产品定义](../产品定义.md)
- [详细设计](../详细设计.md)
- [designer.task](../ai-packages/tasks/designer.task.md)
- [refactor/README](./refactor/README.md)
