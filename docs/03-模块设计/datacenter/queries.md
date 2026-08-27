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

开发态 SQL 执行统一受 30 秒、500 行和 5 MiB 限制。返回 `truncated=true` 时结果只用于预览，工作台必须展示截断原因；数据点、计算或其他消费方不得将其视为完整数据集。

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
- 查询默认将完整结果集定义为对象输出，计算脚本或前端脚本自行处理数据
- 查询保存与数据点生成解耦：保存只持久化查询定义，右侧“生成数据点”状态按钮显式创建或更新输出映射
- 每个查询只生成一个数据点；高级提取模式可将完整结果切换为单个结果列。JSON 内部值由 SQL 提取并设置别名后作为普通列选择
- 查询执行成功后在右侧预览结果；未生成时展示本次 SQL 结果，已生成时通过数据点 GET 接口回读真实值并展示质量状态
- 左侧查询树仅在查询存在真实输出映射和数据点路径时展示“数据点”标签，普通保存查询不展示
- 右键菜单管理（查看详情/打开/删除）

## 查询输出与数据点

- 新查询默认输出草稿为 `result`，选择器为完整结果，类型为 `object`。GET 返回稳定数据集对象：

  ```json
  {
    "fields": ["id", "name"],
    "rows": [],
    "rowCount": 0
  }
  ```

  `fields` 用于生成表头，`rows` 是数据行，`rowCount` 只表示本次返回量。平台不推断 SQL `LIMIT/OFFSET` 之前的总量；需要总页数时由用户 SQL 返回，例如使用 `COUNT(*) OVER()`。

- 未保存查询显示“保存查询后可生成”；首次生成显示“保存并生成”；输出配置改变后显示“保存并更新”；同步完成显示“已保存”。
- 完整结果集路径直接使用查询路径，例如 `db.IF关系库.test1`；提取字段时追加字段 key，例如 `db.IF关系库.test1.id`。
- `POST /queries` 未携带 `outputs` 时不得隐式补默认输出或创建数据点。
- `PUT /queries/:id` 仅在请求明确携带 `outputs` 时同步输出映射与数据点；普通查询保存必须保留现有映射且不改写数据点。
- 数据点 GET 回放使用已保存查询和输出配置；页面存在未保存修改时必须明确提示，不能把 SQL 本次预览冒充为数据点值。
