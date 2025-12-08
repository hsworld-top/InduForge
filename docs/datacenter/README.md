# 数据中心 (DataCenter) 文档

## 概述

数据中心是 InduForge 平台的数据管理模块，提供数据连接管理、SQL 查询、数据预览等功能。

## 核心功能

### 1. 数据连接管理
- 支持多种数据库类型
  - MySQL
  - PostgreSQL
  - SQL Server
- 连接配置和测试
- 连接池管理
- 连接状态监控

### 2. SQL 查询管理
- 查询编辑器（Monaco Editor）
- SQL 语法高亮
- 查询执行和结果预览
- 查询历史记录
- 参数化查询

### 3. 数据预览
- 表结构查看
- 数据浏览
- 分页加载
- 数据导出

### 4. 点位订阅（工业场景）
- PLC 数据订阅
- 实时数据推送
- 数据缓存
- 订阅管理

## 技术架构

```
DataCenter
├── src/
│   ├── api/              # API 接口
│   ├── components/       # Vue 组件
│   ├── constants/        # 常量定义
│   ├── router/           # 路由配置
│   ├── utils/            # 工具函数
│   └── views/            # 页面视图
├── public/               # 静态资源
└── vite.config.js        # Vite 配置
```

## 快速开始

### 启动数据中心

```bash
cd InduForge/datacenter
pnpm install
pnpm dev
```

访问: http://localhost:9092/datacenter/

### 创建数据连接

1. 点击"新建连接"
2. 选择数据库类型
3. 填写连接信息
4. 测试连接
5. 保存

### 创建查询

1. 点击"新建查询"
2. 选择数据连接
3. 编写 SQL 查询
4. 运行查询
5. 保存查询

## 文档导航

- **[数据连接管理](./connections.md)** - 数据库连接配置和管理
- **[查询管理](./queries.md)** - SQL 查询编辑和执行

## API 接口

### 连接管理

```
GET    /api/v1/data/projects/:projectId/connections
POST   /api/v1/data/projects/:projectId/connections
PUT    /api/v1/data/projects/:projectId/connections/:id
DELETE /api/v1/data/projects/:projectId/connections/:id
POST   /api/v1/data/projects/:projectId/connections/:id/test
```

### 查询管理

```
GET    /api/v1/data/projects/:projectId/queries
POST   /api/v1/data/projects/:projectId/queries
PUT    /api/v1/data/projects/:projectId/queries/:id
DELETE /api/v1/data/projects/:projectId/queries/:id
POST   /api/v1/data/queries/:id/execute
```

### 数据操作

```
POST   /api/v1/data/projects/:projectId/connections/:id/execute-sql
GET    /api/v1/data/projects/:projectId/connections/:id/tables
GET    /api/v1/data/projects/:projectId/connections/:id/tables/:tableName/data
```

## 使用场景

### 场景1: 数据报表
1. 创建数据连接
2. 编写查询 SQL
3. 在 Designer 中配置数据源
4. 绑定到表格组件

### 场景2: 实时监控
1. 创建 PLC 连接
2. 配置点位订阅
3. 在 Designer 中配置数据源
4. 绑定到仪表盘组件

### 场景3: 数据分析
1. 创建数据连接
2. 编写聚合查询
3. 在 Designer 中配置数据源
4. 绑定到图表组件

## 常见问题

### Q: 如何配置数据库连接？
A: 参考 [数据连接管理](./connections.md) 文档。

### Q: 如何编写参数化查询？
A: 使用 `?` 占位符，在查询配置中定义参数。

### Q: 如何导出查询结果？
A: 在查询结果页面点击"导出"按钮，选择格式（CSV/Excel）。

### Q: 如何优化查询性能？
A: 
1. 添加合适的索引
2. 限制返回行数
3. 避免 SELECT *
4. 使用查询缓存

## 相关资源

- [后端 API 文档](../backend/README.md)
- [设计中心文档](../designer/README.md)
- [DSL 设计规范](../dsl-design.md)
- [Monaco Editor 文档](https://microsoft.github.io/monaco-editor/)

---

**版本**: 2.0.0  
**最后更新**: 2025-12-08
