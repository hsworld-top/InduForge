# InduForge 文档中心

## 1. 使用原则
- 当前代码、数据库、接口与已实现能力优先于历史文档。
- `docs/` 中的核心设计文档为当前正式版本。
- `docs/archive/` 中的内容仅作为历史参考，不作为开发依据。

## 2. 核心文档入口
### 平台级正式文档
- [产品决策确认版](./产品决策确认版.md)
- [文档审计报告](./文档审计报告.md)
- [模块现状分析](./模块现状分析.md)
- [产品共创分析](./产品共创分析.md)
- [产品定义](./产品定义.md)
- [数据库设计](./database-design.md)
- [高层设计](./高层设计.md)
- [详细设计](./详细设计.md)
- [开发计划](./开发计划.md)
- [开发态预览专项计划](./开发态预览专项计划.md)
- [运行时闭环专项计划](./运行时闭环专项计划.md)
- [发布态 Schema 契约](./contracts/designer-publish-schema.md)
- [IFP Manifest 契约](./contracts/ifp-manifest-contract.md)
- [NodeAgent 启动协议](./contracts/node-agent-runtime-protocol.md)
- [Runtime 健康状态协议](./contracts/runtime-health-status-contract.md)
- [AI 分发包索引](./ai-packages/README.md)
- [模块上下文摘要](./模块上下文摘要.md)
- [待确认事项](./待确认事项.md)

### 模块概览文档
- [后端概览](./backend/README.md)
- [IDE 管理端概览](./dev_ide/README.md)
- [数据中心概览](./datacenter/README.md)
- [设计器概览](./designer/README.md)
- [NodeAgent 总览](./README_nodeagent.md)
- [NodeAgent 后端概览](./node_agent/README.md)
- [NodeAgent Front 概览](./node_agent_front/README.md)

## 3. 当前文档组织
```text
docs/
  README.md
  产品决策确认版.md
  文档审计报告.md
  模块现状分析.md
  产品共创分析.md
  产品定义.md
  database-design.md
  高层设计.md
  详细设计.md
  开发计划.md
  开发态预览专项计划.md
  运行时闭环专项计划.md
  contracts/
  ai-packages/
  模块上下文摘要.md
  待确认事项.md
  backend/
  datacenter/
  designer/
  dev_ide/
  node_agent/
  node_agent_front/
  archive/
```

## 4. 文档维护规则
- 平台级能力变更后，优先更新上述核心文档。
- 模块文档用于承接模块边界、现状、任务和接口说明。
- 明确废弃或过期的文档一律迁移到 `docs/archive/`。
- 未完成核准的规划内容，不应直接写成“当前已实现”。

## 5. 当前重点
- 当前平台的第一优先级已调整为“开发态预览优先”，运行态闭环后置。
- 后续所有模块开发应以 `产品定义`、`高层设计`、`详细设计`、`开发计划` 和 `docs/ai-packages/tasks/` 下的模块任务文档为共同输入。
