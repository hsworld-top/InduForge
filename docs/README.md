# InduForge 项目文档

欢迎查阅 InduForge 低代码平台的完整文档。本文档库包含了项目的架构设计、开发指南、API 文档等内容。

## 📚 文档目录

### 核心设计文档

- **[DSL 设计规范](./dsl-design.md)** - 低代码平台 DSL 完整规范
  - Page Schema 页面结构
  - Component Schema 组件结构
  - DataSource Schema 数据源
  - Action Schema 动作系统
  - Expression 表达式系统
  - 完整示例和最佳实践

- **[数据库设计](./database-design.md)** - 关键表结构与关系概览

- **[迁移计划](./migration-plan.md)** - 从 KingPortal 迁移指南
  - 现状分析
  - 技术栈对比
  - 迁移策略
  - 时间规划

### 设计中心文档

- **[设计中心概述](./designer/README.md)** - Designer 模块总览
- **[数据绑定系统](./designer/data-binding.md)** - 数据绑定完整指南
- **[数据绑定架构](./designer/data-binding-architecture.md)** - 架构设计说明
- **[Canvas 渲染引擎](./designer/canvas-engine.md)** - Konva 渲染引擎
- **[组件开发指南](./designer/component-development.md)** - 自定义组件开发
- **[开发历程](./designer/development-history.md)** - 里程碑记录与变更摘要

### 数据中心文档

- **[数据中心概述](./datacenter/README.md)** - DataCenter 模块总览
- **[数据连接管理](./datacenter/connections.md)** - 数据库连接配置
- **[查询管理](./datacenter/queries.md)** - SQL 查询管理（当前能力与限制）
- **[MQTT 实现说明](./datacenter/mqtt-implementation.md)** - 实现现状与后续规划

### 后端文档

- **[后端 API 文档](./backend/README.md)** - API 接口说明（高层）
- **[认证与授权](./backend/auth.md)** - 用户认证和权限控制
- **[数据库初始化](./backend/database-init.md)** - 数据库初始化指南

## 🚀 快速导航

### 我是新手，想快速了解项目
1. 阅读 [项目 README](../README.md)
2. 查看 [DSL 设计规范](./dsl-design.md)
3. 参考 [数据绑定系统快速开始](./designer/data-binding.md#快速开始)

### 我要开发自定义组件
1. 阅读 [组件开发指南](./designer/component-development.md)
2. 参考 [DSL 组件结构](./dsl-design.md#3-component-schema-组件结构)
3. 查看现有组件示例

### 我要配置数据源
1. 阅读 [数据绑定系统](./designer/data-binding.md)
2. 参考 [DSL 数据源规范](./dsl-design.md#4-datasource-schema-数据源)
3. 查看 [数据连接管理](./datacenter/connections.md)

### 我要部署项目
1. 阅读 [项目 README - 部署指南](../README.md#部署指南)
2. 参考 [数据库初始化](./backend/database-init.md)
3. 配置 Nginx（参考 `../nginx/nginx.conf`）

### 我要了解架构设计
1. 阅读 [项目 README - 整体架构](../README.md#整体架构)
2. 查看 [数据绑定架构](./designer/data-binding-architecture.md)
3. 参考 [迁移计划](./migration-plan.md)

## 📖 文档说明

### 文档结构

```
docs/
├── README.md                           # 本文档
├── dsl-design.md                       # DSL 设计规范
├── database-design.md                  # 数据库设计（概览）
├── migration-plan.md                   # 迁移计划
├── designer/                           # 设计中心文档
│   ├── README.md                       # 概述
│   ├── data-binding.md                 # 数据绑定系统
│   ├── data-binding-architecture.md    # 数据绑定架构
│   ├── canvas-engine.md                # Canvas 渲染引擎
│   ├── component-development.md        # 组件开发指南
│   └── development-history.md          # 开发历程
├── datacenter/                         # 数据中心文档
│   ├── README.md                       # 概述
│   ├── connections.md                  # 数据连接管理
│   ├── queries.md                      # 查询管理
│   └── mqtt-implementation.md          # MQTT 实现说明
└── backend/                            # 后端文档
    ├── README.md                       # 概述
    ├── auth.md                         # 认证与授权
    └── database-init.md                # 数据库初始化
```

### 文档版本

- **版本**: 2.1.0
- **最后更新**: 2025-03-08
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
- [Konva.js 文档](https://konvajs.org/)
- [ECharts 文档](https://echarts.apache.org/)

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
**覆盖率**: 90%+  
**语言**: 简体中文
