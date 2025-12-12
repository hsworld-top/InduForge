# MySQL 实现概述

本文档概述 DataCenter 中 MySQL 功能的实现架构和关键组件。

## 组件层次结构

```
DataCenterNew (主容器)
  ├─ ConnectionList (左侧连接树)
  │   └─ ConnectionItem (连接项)
  │       ├─ 查询列表 (蓝色区域)
  │       └─ 表列表 (绿色区域)
  │
  └─ MysqlContent (右侧内容区)
      ├─ 表列表标签页 (固定)
      │   └─ MysqlTableList
      │
      └─ 查询标签页 (可关闭)
          └─ MysqlQueryEditor
              ├─ 连接选择器
              ├─ 表选择器
              ├─ Monaco Editor
              ├─ 参数面板
              └─ 结果表格
```

## 核心组件

### MysqlContent.vue
**职责**: MySQL 内容的主容器，管理标签页系统

**功能**:
- 管理表列表标签页（固定，不可关闭）
- 管理查询标签页（可创建、关闭）
- 处理标签页切换
- 提供 `createQueryFromTable`、`openQuery`、`showTableList` 等方法

**状态**:
- `queryTabs`: 查询标签页数组
- `activeTab`: 当前活动标签页 ID

### MysqlQueryEditor.vue
**职责**: SQL 查询编辑器

**功能**:
- 连接切换（可在不同数据库间切换）
- 表选择（快速生成 SELECT 语句）
- SQL 编辑（Monaco Editor）
- SQL 美化
- 参数化查询支持
- 查询执行和结果展示
- 结果分页

**Props**:
- `tab`: 标签页对象（包含 SQL、结果、状态等）

**Events**:
- `execute`: 执行查询
- `save`: 保存查询
- `connection-change`: 连接切换

### MysqlTableList.vue
**职责**: 表列表展示

**功能**:
- 加载并显示表列表
- 显示表行数
- 双击表触发查询创建

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

### useMysql.js
**职责**: MySQL 专用逻辑

**提供**:
- `tables`: 表列表
- `loadTables()`: 加载表列表
- `formatSql()`: SQL 美化
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
3. `DataCenterNew.handleTableDblClick()` 调用 `MysqlContent.createQueryFromTable()`
4. 创建新标签页，生成 `SELECT * FROM table LIMIT 100`
5. 切换到新标签页

### 执行查询
1. 用户点击"运行"按钮
2. `MysqlQueryEditor` 触发 `execute` 事件
3. `DataCenterNew.handleQueryExecute()` 调用 API
4. 更新 `tab.result` 和 `tab.executing` 状态
5. 结果显示在编辑器下方

### 保存查询
1. 用户点击"保存"按钮
2. 弹出输入框获取查询名称
3. 调用 API 保存查询
4. 刷新左侧查询列表
5. 更新标签页名称和 `queryId`

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

## 扩展点

### 添加新功能
1. **表结构查看**: 在 `MysqlTableList` 中添加查看按钮
2. **查询历史**: 在 `MysqlContent` 中添加历史标签页
3. **SQL 模板**: 在 `MysqlQueryEditor` 中添加模板选择器
4. **导出功能**: 在结果表格中添加导出按钮

### 性能优化
1. **虚拟滚动**: 表列表和查询列表使用虚拟滚动
2. **懒加载**: 表数据按需加载
3. **缓存**: 缓存表列表和查询列表
4. **防抖**: SQL 编辑器输入防抖
