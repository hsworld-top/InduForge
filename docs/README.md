# InduForge 项目文档

欢迎查阅 InduForge 低代码平台的完整文档。本文档库包含了项目的架构设计、开发指南、API 文档等内容。

## 📚 文档目录

### 🆕 核心设计文档（推荐）

- **[高层设计](./高层设计.md)** - 平台整体架构设计（**入口文档**）

  - 数据隔离三态模型
  - 发布与部署流程
  - 数据库设计
  - 安全与可靠性

- **[数据库设计概览](./database-design.md)** - 数据库表结构快速索引

### 设计中心文档

- **[设计中心概述](./designer/README.md)** - Designer 模块总览

#### 重构计划（进行中）

- **[重构计划](./designer/refactor/README.md)** - 完整重构计划与里程碑
- **[编辑器内核](./designer/refactor/editor-core.md)** - DocumentModel、Command、History、页面锁
- **[Schema 设计](./designer/refactor/schema-design.md)** - 规范化工程 Schema（v2）
- **[组件清单](./designer/refactor/component-manifest.md)** - Component Manifest 规范
- **[数据绑定 v2](./designer/refactor/data-binding-v2.md)** - 三态隔离、Binding 结构
- **[变量系统](./designer/refactor/vars-system.md)** - 页面级/全局变量
- **[表达式引擎](./designer/refactor/expression-engine.md)** - 上下文变量、内置函数
- **[布局系统](./designer/refactor/layout-system.md)** - Flex/Free/Grid、Constraints
- **[渲染架构](./designer/refactor/rendering.md)** - 设计态/运行态同构渲染
- **[设计态交互](./designer/refactor/design-interaction.md)** - 工具栏、属性面板、Canvas 绘图
- **[动作系统](./designer/refactor/action-system.md)** - 完整动作类型、控制流
- **[动画系统](./designer/refactor/animation-system.md)** - 状态驱动动画
- **[验证系统](./designer/refactor/validation-system.md)** - 表单验证规则
- **[发布流水线](./designer/refactor/publish-pipeline.md)** - 校验、编译、打包、部署
- **[运行时引擎](./designer/refactor/runtime-engine.md)** - DataService、Watchdog
- **[多端适配](./designer/refactor/multi-view.md)** - Multi-View 模型
- **[权限系统](./designer/refactor/permissions.md)** - 组件权限 + 动作权限
- **[国际化与主题](./designer/refactor/i18n-theme.md)** - i18n、主题切换组件
- **[最佳实践](./designer/refactor/best-practices.md)** - 性能优化、安全考虑

#### 其他文档

- **[组件开发指南](./designer/component-development.md)** - 自定义组件开发
- **[开发历程](./designer/development-history.md)** - 里程碑记录与变更摘要

### 数据中心文档

- **[数据中心概述](./datacenter/README.md)** - DataCenter 模块总览
- **[数据点方案设计](./datacenter/datapoint-design.md)** - 数据点统一抽象设计
- **[数据点改造实施](./datacenter/datapoint-implementation.md)** - 数据点功能改造计划
- **[数据连接管理](./datacenter/connections.md)** - 数据库连接配置
- **[查询管理](./datacenter/queries.md)** - SQL 查询管理
- **[MQTT 实现说明](./datacenter/mqtt-implementation.md)** - 实现现状与后续规划
- **[变量自动发现与批量导入](./datacenter/mqtt-auto-discovery.md)** - MQTT/API/CSV 多种变量来源

### IDE 前端文档

- **[dev_ide 概述](./dev_ide/README.md)** - 项目管理与运维中心
  - 工程管理、发布流程
  - 用户管理、权限控制
  - 节点管理、部署运维
  - **工程共享机制**（private/shared 可见性）
  - **工程成员管理**（OWNER/ADMIN/DEVELOPER/VIEWER 角色）

### NodeAgent 系统文档

- **[NodeAgent 系统架构](./README_nodeagent.md)** - 系统整体架构和组件说明
- **[NodeAgent 后端文档](./node_agent/README.md)** - Go 后端服务详细功能
- **[NodeAgent Front 文档](./node_agent_front/README.md)** - Vue 前端管理界面详解

### 后端文档

- **[后端 API 文档](./backend/README.md)** - API 接口说明
- **[认证与授权](./backend/auth.md)** - 用户认证和权限控制
- **[数据库初始化](./backend/database-init.md)** - 数据库初始化指南
- **[WebSocket 协议](./backend/websocket.md)** - Socket.IO 事件与消息格式
- **[发布部署 API](./backend/publish-deploy-api.md)** - 发布、节点、部署接口

### 历史参考文档

> 以下文档为历史版本，部分内容可能已过时。

- **[DSL 设计规范 v2.0](./dsl-design.md)** - 低代码平台 DSL 规范（被 Schema v2 替代）
- **[迁移计划](./migration-plan.md)** - 从 KingPortal 迁移指南（已归档）

## 🚀 快速导航

### 我是新手，想快速了解项目

1. 阅读 [项目 README](../README.md)
2. 查看 [高层设计](./高层设计.md)
3. 了解 [Designer 重构计划](./designer/refactor/README.md)

### 我要开发自定义组件

1. 阅读 [组件开发指南](./designer/component-development.md)
2. 参考 [Schema 设计](./designer/refactor/schema-design.md)
3. 查看 [数据绑定 v2](./designer/refactor/data-binding-v2.md)

### 我要配置数据源

1. 阅读 [数据绑定 v2](./designer/refactor/data-binding-v2.md)
2. 参考 [数据中心概述](./datacenter/README.md)
3. 查看 [数据连接管理](./datacenter/connections.md)

### 我要了解发布与部署

1. 阅读 [发布流水线](./designer/refactor/publish-pipeline.md)
2. 查看 [运行时引擎](./designer/refactor/runtime-engine.md)
3. 参考 [发布部署 API](./backend/publish-deploy-api.md)
4. 了解 [dev_ide 运维管理](./dev_ide/README.md#5-运维管理待开发)

### 我要开发实时数据功能

1. 阅读 [WebSocket 协议](./backend/websocket.md)
2. 参考 [MQTT 实现](./datacenter/mqtt-implementation.md)
3. 查看 [数据绑定 v2](./designer/refactor/data-binding-v2.md)

### 我要部署项目

1. 阅读 [项目 README - 部署指南](../README.md#部署指南)
2. 参考 [数据库初始化](./backend/database-init.md)
3. 配置 Nginx（参考 `../nginx/nginx.conf`）

## 📖 文档说明

### 文档结构

```
docs/
├── README.md                           # 本文档
├── 高层设计.md                          # 平台整体架构设计 ⭐
├── database-design.md                  # 数据库设计概览
├── dsl-design.md                       # DSL 设计规范（历史）
├── migration-plan.md                   # 迁移计划（历史）
├── dev_ide/                            # 🆕 IDE 前端文档
│   └── README.md                       # 项目管理与运维中心
├── designer/                           # 设计中心文档
│   ├── README.md                       # 概述
│   ├── refactor/                       # 🆕 重构文档
│   │   ├── README.md                   # 重构计划
│   │   ├── editor-core.md              # 编辑器内核
│   │   ├── schema-design.md            # Schema 设计
│   │   ├── data-binding-v2.md          # 数据绑定 v2
│   │   ├── layout-system.md            # 布局系统
│   │   ├── design-interaction.md       # 设计态交互
│   │   ├── publish-pipeline.md         # 发布流水线
│   │   ├── runtime-engine.md           # 运行时引擎
│   │   └── i18n-theme.md               # 国际化与主题
│   ├── component-development.md        # 组件开发指南
│   └── development-history.md          # 开发历程
├── datacenter/                         # 数据中心文档
│   ├── README.md                       # 概述
│   ├── datapoint-design.md             # 数据点方案设计
│   ├── datapoint-implementation.md     # 数据点改造实施
│   ├── connections.md                  # 数据连接管理
│   ├── queries.md                      # 查询管理
│   ├── mqtt-implementation.md          # MQTT 实现说明
│   └── mqtt-auto-discovery.md          # MQTT 自动发现设计
└── backend/                            # 后端文档
    ├── README.md                       # 概述
    ├── auth.md                         # 认证与授权
    ├── database-init.md                # 数据库初始化
    ├── websocket.md                    # 🆕 WebSocket 协议
    └── publish-deploy-api.md           # 🆕 发布部署 API
```

### 文档版本

- **版本**: 3.0.0
- **最后更新**: 2026-01
- **维护者**: InduForge Team

### 文档贡献

欢迎贡献文档！请遵循以下规范：

1. 使用 Markdown 格式
2. 保持文档结构清晰
3. 添加必要的代码示例
4. 更新文档索引
5. 提交 Pull Request

## 🔗 相关资源

### 外部文档

- [Vue 3 文档](https://vuejs.org/)
- [Pinia 文档](https://pinia.vuejs.org/)
- [Element Plus 文档](https://element-plus.org/)
- [ECharts 文档](https://echarts.apache.org/)
- [Day.js 文档](https://day.js.org/)

### 项目资源

- [GitHub 仓库](#)
- [在线演示](#)
- [问题反馈](#)

## 📞 获取帮助

如有问题或建议：

1. 查看 [常见问题](../README.md#常见问题)
2. 搜索现有文档
3. 提交 Issue
4. 联系开发团队

---

**文档状态**: ✅ 完整  
**覆盖率**: 95%+  
**语言**: 简体中文
