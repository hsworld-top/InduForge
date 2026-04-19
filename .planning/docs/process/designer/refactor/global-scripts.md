# 全局脚本（Global Scripts）

全局脚本用于工程级事件与可复用逻辑，集中管理后可被多处调用。

## 1. 模块划分

全局脚本按模块组织，结构固定为：

- 系统脚本（system）
  - 系统启动（startup）
  - 系统关闭（shutdown）
- 定时器（timers）
  - 名称 + 时间（ms）+ 描述 + 脚本
- 变量改变（variableChanges）
  - 监听工程变量变化触发脚本
- 自定义脚本（custom）
  - 可被其他脚本调用的函数

## 2. 分组与操作

- 支持分组管理，最多 5 层嵌套。
- 右键菜单：新建/编辑/移动/复制/粘贴/删除。
- 双击快速打开脚本编辑器。
- `Ctrl/⌘ + 左键` 多选：仅允许移动/复制/删除，编辑/打开置灰。

## 3. 编辑器能力

- 语言：JavaScript。
- 快捷键：
  - 保存：`Ctrl+S`
  - 格式化：`Shift+Alt+F`
- 关闭时提示保存。
- 定时器脚本编辑器支持修改时间（ms）。
- 自定义脚本编辑器显示入参。
- 右侧支持工程变量与自定义脚本树形插入，并支持搜索。

## 4. 数据结构

全局脚本结构固定初始化如下：

```json
{
  "system": {
    "startup": { "code": "" },
    "shutdown": { "code": "" }
  },
  "timers": {
    "groups": [],
    "items": []
  },
  "variableChanges": {
    "groups": [],
    "items": []
  },
  "custom": {
    "groups": [],
    "items": []
  }
}
```

`items` 典型字段（不同模块略有差异）：

```json
{
  "id": "uuid",
  "name": "scriptName",
  "variable": "globalVarName",
  "params": "id, value",
  "interval": 1000,
  "description": "",
  "groupId": null,
  "code": ""
}
```

## 5. 数据持久化

全局脚本保存到 `design_project_settings.globalScripts`，由设计器统一读写。


## 6. 使用示例
### 6.1 组件访问
- `this`：当前事件触发的组件实例，提供 `setText/setProps/setStyle` 等调用
- `components.<组件名>`：当前页面组件引用
- `components.pages["<页面名>"].<组件名>`：指定页面组件

```js
// 当前组件
this.setText("刷新");

// 当前页面其它组件
components.按钮2.setText("我被操作了");

// 跨页面组件
components.pages["首页"].按钮1.setText("跨页操作");
```

### 6.2 系统脚本
```js
// 系统启动
console.log("startup", $global.变量A);

// 系统关闭
console.log("shutdown");
```

### 6.3 定时器
```js
// 每隔 N ms 触发
const now = new Date().toISOString();
console.log("timer tick", now);
```

### 6.4 变量改变
```js
// item.variable = "测试布尔"
console.log("var changed", $event.name, $event.previous, "->", $event.value);
```

### 6.5 自定义脚本
```js
// custom script: name = "sum", params = "a, b"
return a + b;
```

```js
// 在其它脚本内调用
const total = await customScripts.sum(1, 2);
console.log(total);
```

### 6.6 $global 映射数据中心
```js
// 映射 SQL 查询到工程变量后可 await 直接取值
const rows = await $global.users;
console.log(rows);
```

### 6.7 表格方法（仅表格组件生效）
- `setTableHeader(columns)`：设置表头列
- `setTableData(rows[, header])`：设置表格数据

说明：
- 仅会刷新当前表格组件，其他表格不受影响。
- `rows` 支持对象数组，或 SQL 结果 `{ columns, rows }`。
- `columns` 支持 `{ label, prop }` 数组。

```js
const { columns, rows } = $global.users;
const { tableData, tableColumns } = $global.toElementUITable(columns, rows);
components.pages["dada"].表格1.setTableHeader(tableColumns);
components.pages["dada"].表格1.setTableData(tableData);
```

### 6.8 预览态 $global 使用说明
- 工程变量（非映射）直接同步读取：`$global.变量名`
- 映射的数据中心变量：预览态优先取实时值，无需 `await`
- 需要查询接口时，`await $global.xxx` 仍可使用（兼容）

```js
let d = $global.tag1;           // 订阅/Tag 实时值
let e = $global.mqttzzz;        // 订阅实时值
let f = await $global.users;    // 查询结果（兼容 await）
```
