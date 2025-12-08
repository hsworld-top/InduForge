# 查询管理

## 概述

查询管理提供 SQL 查询的编辑、执行、保存和管理功能。

## 查询编辑器

### 功能特性

- **语法高亮**: 支持 SQL 语法高亮
- **智能提示**: 表名、字段名自动补全
- **格式化**: SQL 代码格式化
- **快捷键**: 常用操作快捷键
- **多标签**: 支持多个查询同时编辑

### 快捷键

| 快捷键 | 功能 |
|--------|------|
| Ctrl+Enter | 执行查询 |
| Ctrl+S | 保存查询 |
| Ctrl+/ | 注释/取消注释 |
| Ctrl+F | 查找 |
| Ctrl+H | 替换 |
| Alt+Shift+F | 格式化代码 |

## 创建查询

### 通过界面创建

1. 点击"新建查询"
2. 选择数据连接
3. 输入查询名称
4. 编写 SQL 语句
5. 点击"运行"测试
6. 点击"保存"

### 查询示例

#### 简单查询

```sql
SELECT id, name, status, value
FROM devices
WHERE status = 1
ORDER BY id DESC
LIMIT 100
```

#### 参数化查询

```sql
SELECT id, name, status, value
FROM devices
WHERE status = ?
  AND created_at >= ?
  AND created_at <= ?
ORDER BY id DESC
```

参数配置：
```json
{
  "parameters": [
    {
      "name": "status",
      "type": "number",
      "default": 1
    },
    {
      "name": "startDate",
      "type": "date",
      "default": "2025-01-01"
    },
    {
      "name": "endDate",
      "type": "date",
      "default": "2025-12-31"
    }
  ]
}
```

#### 聚合查询

```sql
SELECT 
  status,
  COUNT(*) as count,
  AVG(value) as avg_value,
  MAX(value) as max_value,
  MIN(value) as min_value
FROM devices
GROUP BY status
```

#### 关联查询

```sql
SELECT 
  d.id,
  d.name,
  d.status,
  c.name as category_name
FROM devices d
LEFT JOIN categories c ON d.category_id = c.id
WHERE d.status = 1
```

## 执行查询

### 通过界面执行

1. 打开查询编辑器
2. 编写或选择查询
3. 点击"运行"按钮
4. 查看结果

### 通过 API 执行

```javascript
const response = await fetch('/api/v1/data/queries/QUERY_ID/execute', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer TOKEN'
  },
  body: JSON.stringify({
    parameters: {
      status: 1,
      startDate: '2025-01-01',
      endDate: '2025-12-31'
    }
  })
})

const result = await response.json()
console.log(result.data)
```

## 查询结果

### 结果展示

- 表格视图
- 分页加载
- 列排序
- 数据筛选

### 结果导出

支持导出格式：
- CSV
- Excel
- JSON

### 结果统计

- 总行数
- 执行时间
- 影响行数

## 查询管理

### 查询列表

```javascript
const response = await fetch('/api/v1/data/projects/PROJECT_ID/queries', {
  headers: {
    'Authorization': 'Bearer TOKEN'
  }
})
```

### 更新查询

```javascript
const response = await fetch('/api/v1/data/projects/PROJECT_ID/queries/QUERY_ID', {
  method: 'PUT',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer TOKEN'
  },
  body: JSON.stringify({
    name: '新名称',
    sql: 'SELECT * FROM devices WHERE status = ?'
  })
})
```

### 删除查询

```javascript
const response = await fetch('/api/v1/data/projects/PROJECT_ID/queries/QUERY_ID', {
  method: 'DELETE',
  headers: {
    'Authorization': 'Bearer TOKEN'
  }
})
```

## 参数化查询

### 参数类型

| 类型 | 说明 | 示例 |
|------|------|------|
| string | 字符串 | 'hello' |
| number | 数字 | 123 |
| boolean | 布尔值 | true |
| date | 日期 | '2025-01-01' |
| datetime | 日期时间 | '2025-01-01 12:00:00' |
| array | 数组 | [1, 2, 3] |

### 参数绑定

在 Designer 中使用表达式绑定参数：

```json
{
  "id": "ds_devices",
  "type": "dataCenter",
  "config": {
    "sourceType": "query",
    "queryId": "query_devices",
    "parameters": {
      "status": "{{ vars.filterStatus }}",
      "startDate": "{{ vars.startDate }}",
      "endDate": "{{ vars.endDate }}"
    }
  }
}
```

## 查询优化

### 性能优化建议

1. **使用索引**
   ```sql
   CREATE INDEX idx_status ON devices(status);
   CREATE INDEX idx_created_at ON devices(created_at);
   ```

2. **限制返回行数**
   ```sql
   SELECT * FROM devices LIMIT 100;
   ```

3. **避免 SELECT ***
   ```sql
   -- 不推荐
   SELECT * FROM devices;
   
   -- 推荐
   SELECT id, name, status FROM devices;
   ```

4. **使用 WHERE 过滤**
   ```sql
   SELECT * FROM devices WHERE status = 1;
   ```

5. **避免子查询**
   ```sql
   -- 不推荐
   SELECT * FROM devices WHERE id IN (SELECT device_id FROM logs);
   
   -- 推荐
   SELECT DISTINCT d.* FROM devices d
   INNER JOIN logs l ON d.id = l.device_id;
   ```

### 查询分析

使用 EXPLAIN 分析查询：

```sql
EXPLAIN SELECT * FROM devices WHERE status = 1;
```

## 常见问题

### Q: 如何编写参数化查询？
A: 使用 `?` 占位符，在查询配置中定义参数。

### Q: 查询超时怎么办？
A: 
1. 优化 SQL 语句
2. 添加索引
3. 增加超时时间

### Q: 如何调试查询？
A: 
1. 使用 EXPLAIN 分析
2. 查看执行计划
3. 检查索引使用情况

### Q: 如何处理大结果集？
A: 
1. 使用分页
2. 限制返回行数
3. 使用流式处理

## 最佳实践

### 1. 查询命名
使用有意义的名称，如"query_device_list"、"query_alarm_summary"。

### 2. 参数化查询
使用参数化查询防止 SQL 注入，提高安全性。

### 3. 查询优化
- 添加合适的索引
- 避免全表扫描
- 使用 EXPLAIN 分析

### 4. 错误处理
- 捕获查询错误
- 提供友好的错误提示
- 记录错误日志

### 5. 查询复用
- 封装常用查询
- 使用视图简化复杂查询
- 建立查询模板库

## 相关资源

- [数据连接管理](./connections.md)
- [后端 API 文档](../backend/README.md)
- [设计中心 - 数据绑定](../designer/data-binding.md)
- [SQL 教程](https://www.w3schools.com/sql/)

---

**版本**: 2.0.0  
**最后更新**: 2025-12-08
