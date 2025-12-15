# 数据库实现概述

本文档概述 DataCenter 中各数据库功能的实现架构和关键组件。

## 组件层次结构

```
DataCenterNew (主容器)
  ├─ ConnectionList (左侧连接树)
  │   └─ ConnectionItem (连接项)
  │       ├─ 查询列表 (蓝色区域)
  │       └─ 表列表 (绿色区域)
  │
  └─ 统一标签页系统 (右侧内容区)
      ├─ 表列表标签页
      │   ├─ MysqlTableList
      │   ├─ PostgresTableList
      │   └─ SqlServerTableList
      │
      └─ 查询标签页
          ├─ MysqlQueryEditor
          ├─ PostgresQueryEditor
          └─ SqlServerQueryEditor
```

## 核心组件

### DataCenterNew.vue
**职责**: 主容器，管理统一标签页系统

**功能**:
- 管理所有标签页（表列表和查询）
- 处理连接双击（测试连接并展开）
- 处理表双击（创建查询）
- 处理查询双击（打开查询）
- 右键菜单管理
- 标签页滚轮支持

**状态**:
- `tabs`: 所有标签页数组
- `activeTabId`: 当前活动标签页 ID
- `connections`: 连接列表

### QueryEditor 组件（MySQL/PostgreSQL/SQL Server）
**职责**: SQL 查询编辑器

**功能**:
- 连接切换（可在不同数据库间切换）
- 表选择（快速生成 SELECT 语句）
- SQL 编辑（Monaco Editor）
- SQL 美化
- 参数化查询支持（`?` 占位符）
- 查询执行和结果展示
- 结果分页

**Props**:
- `tab`: 标签页对象（包含 SQL、结果、状态等）

**Events**:
- `execute`: 执行查询
- `save`: 保存查询

### TableList 组件（MySQL/PostgreSQL/SQL Server）
**职责**: 表列表展示

**功能**:
- 加载并显示表列表
- 显示表行数
- 双击表触发查询创建
- 表搜索和过滤

### ConnectionList.vue
**职责**: 左侧连接树

**功能**:
- 连接列表展示
- 连接展开/折叠
- 查询列表展示（蓝色区域）
- 表列表展示（绿色区域）
- 右键菜单
- 查询删除

**状态管理**:
使用 `connectionStates` reactive 对象管理每个连接的状态：
```javascript
{
  [connectionId]: {
    expanded: false,        // 是否展开
    loading: false,         // 是否正在加载表
    tables: [],            // 表列表
    tablesExpanded: true,  // 表区域是否展开
    loadingQueries: false, // 是否正在加载查询
    queries: [],           // 查询列表
    queriesExpanded: true  // 查询区域是否展开
  }
}
```

## Composables

### useConnection.js
**职责**: 连接管理逻辑

**提供**:
- `connections`: 连接列表
- `selectedConnection`: 当前选中的连接
- `loadConnections()`: 加载连接列表
- `createConnection()`: 创建连接
- `updateConnection()`: 更新连接
- `deleteConnection()`: 删除连接
- `testConnection()`: 测试连接
- `updateConnectionStatus()`: 更新连接状态

### useConnectionStatus.js
**职责**: 连接状态管理

**提供**:
- 连接状态监控
- 状态更新逻辑

### 数据库专用 Composables

#### useMysql.js
**职责**: MySQL 专用逻辑

**提供**:
- `formatSql()`: SQL 美化（MySQL 方言）
- `extractSqlParameters()`: 提取 SQL 参数

#### usePostgres.js
**职责**: PostgreSQL 专用逻辑

**提供**:
- `formatSql()`: SQL 美化（PostgreSQL 方言）
- `extractSqlParameters()`: 提取 SQL 参数

#### useSqlServer.js
**职责**: SQL Server 专用逻辑

**提供**:
- `formatSql()`: SQL 美化（T-SQL 方言）
- `extractSqlParameters()`: 提取 SQL 参数

## 数据流

### 打开连接
1. 用户双击连接
2. `DataCenterNew.handleConnectionDblClick()` 测试连接
3. 更新连接状态为 "connected"
4. 展开连接树 (`state.expanded = true`)
5. 并行加载表列表和查询列表

### 创建查询
1. 用户双击表
2. `ConnectionList` 触发 `table-dblclick` 事件
3. `DataCenterNew.handleTableDblClick()` 调用 `createQueryFromTable()`
4. 创建新标签页，生成对应数据库的 SELECT 语句：
   - MySQL: `SELECT * FROM \`table\` LIMIT 100`
   - PostgreSQL: `SELECT * FROM "table" LIMIT 100`
   - SQL Server: `SELECT * FROM [table] ORDER BY (SELECT NULL) OFFSET 0 ROWS FETCH NEXT 100 ROWS ONLY`
5. 切换到新标签页

### 执行查询
1. 用户点击"运行"按钮
2. QueryEditor 触发 `execute` 事件
3. `DataCenterNew.handleQueryExecute()` 调用 API
4. 更新 `tab.result` 和 `tab.executing` 状态
5. 结果显示在编辑器下方

### 保存查询
1. 用户点击"保存"按钮
2. 弹出输入框获取查询名称
3. 调用 API 保存查询（包含 SQL 和参数配置）
4. 刷新左侧查询列表
5. 更新标签页名称和 `queryId`

### 查看表结构
1. 用户右键点击表
2. 选择"查看表结构"
3. 打开 TableStructureDialog
4. 加载并显示表结构信息（字段、索引等）

## 样式设计

### 颜色主题
- **查询区域**: 蓝色系 (`blue-50`, `blue-300`, `blue-700`)
- **表区域**: 绿色系 (`green-50`, `green-300`, `green-700`)
- **Hover**: 更深的背景色 + 轻微阴影

### 交互效果
- **Hover 动画**: `hover:translate-x-0.5` (右移 2px)
- **过渡**: `transition-all duration-150` (150ms 平滑过渡)
- **圆角**: `rounded` (4px)
- **阴影**: `hover:shadow-sm` (轻微阴影)

### 响应式
- 左侧树固定宽度 `w-64` (256px)
- 右侧内容区自适应 `flex-1`
- 编辑器固定高度 300px
- 结果区域自适应剩余空间

## 事件冒泡控制

为防止误触发连接折叠，在多个层级添加了 `@dblclick.stop`:
1. 展开区域根容器
2. 查询和表标题栏
3. 查询项和表项
4. 连接项只在头部响应双击

## 数据库特性对比

| 特性 | MySQL | PostgreSQL | SQL Server |
|------|-------|------------|------------|
| 连接管理 | ✅ | ✅ | ✅ |
| 表列表 | ✅ | ✅ | ✅ |
| 表结构查看 | ✅ | ✅ | ✅ |
| SQL 编辑器 | ✅ | ✅ | ✅ |
| 参数化查询 | ✅ | ✅ | ✅ |
| SQL 美化 | ✅ | ✅ | ✅ |
| 查询保存 | ✅ | ✅ | ✅ |
| Schema 支持 | ❌ | ✅ | ✅ |
| 标识符引号 | \` | " | [ ] |
| 分页语法 | LIMIT | LIMIT | OFFSET/FETCH |

## 扩展点

### 添加新数据库类型

1. **创建组件目录**
   ```
   datacenter/src/components/database/oracle/
   ├── OracleTableList.vue
   └── OracleQueryEditor.vue
   ```

2. **创建 Composable**
   ```
   datacenter/src/composables/database/useOracle.js
   ```

3. **注册连接类型**
   在 `config/connectionTypes.js` 中添加配置

4. **更新主容器**
   在 `DataCenterNew.vue` 中添加组件引用和条件渲染

5. **实现 SQL 方言**
   使用 `node-sql-parser` 支持对应的 SQL 方言

### 性能优化

1. **虚拟滚动**: 表列表和查询列表使用虚拟滚动
2. **懒加载**: 表数据按需加载
3. **缓存**: 缓存表列表和查询列表
4. **防抖**: SQL 编辑器输入防抖

### 功能扩展

1. **查询历史**: 记录查询执行历史
2. **SQL 模板**: 提供常用 SQL 模板
3. **导出功能**: 支持导出查询结果（CSV、Excel）
4. **查询计划**: 显示查询执行计划
5. **多语句执行**: 支持执行多条 SQL 语句

---

**版本**: 2.1.0  
**最后更新**: 2025-12-15
