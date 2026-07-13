# 查询管理

本文档面向内部开发，描述当前查询管理能力与接口范围（高层）。

## 核心能力

- 查询定义（基于连接）
- 查询执行（参数化）
- 查询编辑与删除
- 直接执行 SQL 语句
- 执行日志记录

## 接口范围

### 查询 CRUD

| 接口                                       | 方法   | 说明                                 |
| ------------------------------------------ | ------ | ------------------------------------ |
| `/api/v1/data/projects/:projectId/queries` | GET    | 获取查询列表，支持 connectionId 过滤 |
| `/api/v1/data/projects/:projectId/queries` | POST   | 创建新查询                           |
| `/api/v1/data/queries/:id`                 | PUT    | 更新查询配置                         |
| `/api/v1/data/queries/:id`                 | DELETE | 删除查询                             |

### 查询执行

| 接口                                                                     | 方法 | 说明                            |
| ------------------------------------------------------------------------ | ---- | ------------------------------- |
| `/api/v1/data/queries/:id/execute`                                       | POST | 执行已保存的查询                |
| `/api/v1/data/projects/:projectId/connections/:connectionId/execute-sql` | POST | 直接执行 SQL 语句（支持参数化） |

## 查询配置结构

```javascript
{
  name: "设备查询",
  connectionId: "conn_xxx",
  queryType: "sql",
  config: {
    sql: "SELECT * FROM devices WHERE status = ?",
    parameters: [
      { name: "param1", type: "string", required: false, default: "active" }
    ]
  }
}
```

## 前端功能

- 查询列表展示（在连接树中按蓝色区域分组）
- 双击查询打开编辑器
- SQL 编辑器（基于 Monaco Editor）
- SQL 美化功能
- 参数化查询支持（`?` 占位符）
- 查询结果分页展示
- 右键菜单管理（查看详情/打开/删除）
