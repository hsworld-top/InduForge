# 查询管理

## 概述

查询管理提供 SQL 查询的编辑、执行、保存和管理功能。

## 查询编辑器

### 功能特性

- **语法高亮**: 支持 SQL 语法高亮（基于 Monaco Editor）
- **智能提示**: 表名、字段名自动补全
- **SQL 美化**: 一键格式化 SQL 代码
- **连接切换**: 在不同数据库连接间快速切换
- **表选择器**: 快速选择表并生成查询
- **参数化查询**: 支持 `?` 占位符和参数面板
- **多标签**: 支持多个查询同时编辑
- **结果分页**: 查询结果支持分页浏览

### 工具栏功能

- **连接选择**: 切换数据库连接
- **表选择**: 选择表快速生成 SELECT 语句
- **运行**: 执行当前 SQL 查询
- **保存**: 保存查询到查询列表
- **美化**: 格式化 SQL 代码

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

**方式一：从表创建**
1. 双击连接展开表列表
2. 双击表自动创建查询（生成 `SELECT * FROM table LIMIT 100`）
3. 修改 SQL 语句
4. 点击"运行"测试
5. 点击"保存"输入查询名称

**方式二：新建空查询**
1. 打开表列表标签页
2. 点击"新建查询"按钮
3. 编写 SQL 语句
4. 点击"运行"测试
5. 点击"保存"输入查询名称

**方式三：打开已保存的查询**
1. 在左侧连接树的"查询"区域（蓝色）
2. 双击查询打开
3. 修改后可重新保存

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

- **表格视图**: 以表格形式展示查询结果
- **分页浏览**: 支持分页查看大量数据（默认每页 20 行）
- **结果统计**: 显示总行数和执行时间

### 结果信息

查询执行后显示：
- 返回行数
- 执行时间（毫秒）
- 列名和数据类型

### 分页控制

- 每页显示行数：20 行（可配置）
- 分页器：快速跳转到指定页
- 总页数和当前页显示

## 查询管理

### 查询列表

已保存的查询显示在左侧连接树的"查询"区域（蓝色背景）：
- 查询名称
- 所属连接
- 快速访问

### 查询操作

**双击查询**: 打开查询编辑器

**右键菜单**:
- 查看详情：显示查询配置信息
- 打开：在新标签页中打开
- 删除：删除查询（会关闭对应标签页）

### 更新查询

1. 打开已保存的查询
2. 修改 SQL 或参数
3. 点击"保存"更新查询

### 删除查询

1. 右键点击查询
2. 选择"删除"
3. 确认删除操作
4. 对应的标签页会自动关闭

## 参数化查询

### 使用参数

在 SQL 中使用 `?` 作为参数占位符：

```sql
SELECT * FROM devices 
WHERE status = ? 
  AND created_at >= ? 
  AND created_at <= ?
```

### 参数面板

查询编辑器下方会自动显示参数面板：
- 自动检测 SQL 中的 `?` 占位符
- 为每个参数生成输入框
- 支持设置参数类型（string/number）
- 支持设置默认值

### 参数类型

| 类型 | 说明 | 示例 |
|------|------|------|
| string | 字符串 | 'hello' |
| number | 数字 | 123 |

### 参数执行

1. 在 SQL 中使用 `?` 占位符
2. 在参数面板中输入参数值
3. 点击"运行"执行查询
4. 参数值会按顺序替换占位符

### 保存参数化查询

保存查询时，参数配置会一起保存：
- 参数数量
- 参数类型
- 参数默认值

下次打开查询时，参数配置会自动恢复

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
A: 在 SQL 中使用 `?` 占位符，参数面板会自动显示，输入参数值后执行

### Q: 查询超时怎么办？
A: 
1. 优化 SQL 语句（添加 WHERE 条件）
2. 添加数据库索引
3. 使用 LIMIT 限制返回行数
4. 检查网络连接

### Q: 如何切换数据库连接？
A: 在查询编辑器顶部使用连接选择器切换

### Q: 如何快速生成查询？
A: 
1. 双击表自动生成 SELECT 查询
2. 使用表选择器快速插入表名

### Q: 查询结果太多怎么办？
A: 
1. 使用分页浏览（每页 20 行）
2. 添加 LIMIT 子句限制返回行数
3. 使用 WHERE 条件过滤数据

### Q: 如何保存查询？
A: 点击"保存"按钮，输入查询名称即可保存

### Q: 参数化查询如何工作？
A: 使用 `?` 占位符，系统会按顺序将参数值替换到 SQL 中

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

**版本**: 2.1.0  
**最后更新**: 2025-12-15
