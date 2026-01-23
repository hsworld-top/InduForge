# 组件脚本方法说明

按组件分类输出，方法格式与示例遵循统一模板。

## 通用属性（适用于多数组件）

### 1 Name
方法名称：组件对象的名称  
功能：获取组件名称  
返回值：string  

```javascript
// 调用格式:
var name = 组件名.Name;
```

### 2 Comment
方法名称：组件类型说明  
功能：获取组件类型描述  
返回值：string  

```javascript
// 调用格式:
var desc = 组件名.Comment;
```

### 3 Location.X / Location.Y
方法名称：组件位置  
功能：读取/设置组件坐标  
返回值：number  

```javascript
// 调用格式:
var x = 组件名.Location.X;
组件名.Location.X = 10;
var y = 组件名.Location.Y;
组件名.Location.Y = 20;
```

### 4 Size.Width / Size.Height
方法名称：组件尺寸  
功能：读取/设置组件宽高  
返回值：number  

```javascript
// 调用格式:
var w = 组件名.Size.Width;
组件名.Size.Width = 300;
var h = 组件名.Size.Height;
组件名.Size.Height = 120;
```

### 5 Visible
方法名称：组件显隐  
功能：读取/设置显示状态  
返回值：boolean  

```javascript
// 调用格式:
var visible = 组件名.Visible;
组件名.Visible = false;
```

### 6 Enable
方法名称：组件使能  
功能：读取/设置禁用状态  
返回值：boolean  

```javascript
// 调用格式:
var enabled = 组件名.Enable;
组件名.Enable = false;
```

### 7 Caption
方法名称：组件显示文本  
功能：读取/设置按钮或标签文本  
返回值：string  

```javascript
// 调用格式:
var text = 组件名.Caption;
组件名.Caption = "请点击";
```

### 8 Image
方法名称：图片源  
功能：读取/设置图片地址  
返回值：string  

```javascript
// 调用格式:
var src = 组件名.Image;
组件名.Image = "https://example.com/a.png";
```

## 文本（Text / Tag / Button）

### 1 SetText(text)
方法名称：设置文本  
功能：设置文本显示内容  
参数说明：text，string  
返回值：无  

```javascript
// 调用格式:
组件名.SetText("Hello");
```

### 2 GetText()
方法名称：获取文本  
功能：获取当前文本  
返回值：string  

```javascript
// 调用格式:
var text = 组件名.GetText();
```

### 3 SetType(type)
方法名称：设置类型  
功能：设置样式类型  
参数说明：type，string（primary/success/warning/danger/info）  
返回值：无  

```javascript
// 调用格式:
组件名.SetType("success");
```

### 4 SetEllipsis(bool)
方法名称：设置省略  
功能：设置文本单行省略  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetEllipsis(true);
```

### 5 SetTooltip(bool)
方法名称：设置溢出提示  
功能：开启/关闭溢出提示  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetTooltip(true);
```

## 按钮（Button）

### 1 SetLoading(bool)
方法名称：设置加载态  
功能：设置按钮 loading 状态  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetLoading(true);
```

### 2 SetDisabled(bool)
方法名称：设置禁用  
功能：设置按钮禁用  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetDisabled(true);
```

### 3 Click()
方法名称：触发点击  
功能：模拟一次点击  
返回值：无  

```javascript
// 调用格式:
组件名.Click();
```

### 4 Focus()
方法名称：获取焦点  
功能：让按钮获得焦点  
返回值：无  

```javascript
// 调用格式:
组件名.Focus();
```

## 图片（Image）

### 1 SetSrc(src)
方法名称：设置图片地址  
功能：设置图片 src  
参数说明：src，string  
返回值：无  

```javascript
// 调用格式:
组件名.SetSrc("https://example.com/a.png");
```

### 2 GetSrc()
方法名称：获取图片地址  
功能：获取图片 src  
返回值：string  

```javascript
// 调用格式:
var src = 组件名.GetSrc();
```

### 3 Preview(urls?, startIndex?)
方法名称：预览图片  
功能：触发图片预览  
参数说明：urls，array；startIndex，number  
返回值：无  

```javascript
// 调用格式:
组件名.Preview(["/a.png", "/b.png"], 0);
```

### 4 Reload()
方法名称：刷新图片  
功能：给 src 添加时间戳刷新  
返回值：无  

```javascript
// 调用格式:
组件名.Reload();
```

## 输入框（Input）

### 1 Focus()
方法名称：获取焦点  
功能：输入框获得焦点  
返回值：无  

```javascript
// 调用格式:
组件名.Focus();
```

### 2 Blur()
方法名称：失去焦点  
功能：输入框失去焦点  
返回值：无  

```javascript
// 调用格式:
组件名.Blur();
```

### 3 Select()
方法名称：选中内容  
功能：选中文本内容  
返回值：无  

```javascript
// 调用格式:
组件名.Select();
```

### 4 SetInputValue(val)
方法名称：设置值  
功能：设置输入框值  
参数说明：val，string/number  
返回值：无  

```javascript
// 调用格式:
组件名.SetInputValue("Hello");
```

### 5 GetInputValue()
方法名称：获取值  
功能：获取输入框值  
返回值：string  

```javascript
// 调用格式:
var val = 组件名.GetInputValue();
```

### 6 Clear()
方法名称：清空  
功能：清空输入内容  
返回值：无  

```javascript
// 调用格式:
组件名.Clear();
```

## 选择器（Select）

### 1 Focus()
方法名称：获取焦点  
功能：选择器获得焦点  
返回值：无  

```javascript
// 调用格式:
组件名.Focus();
```

### 2 Blur()
方法名称：失去焦点  
功能：选择器失去焦点  
返回值：无  

```javascript
// 调用格式:
组件名.Blur();
```

### 3 Open()
方法名称：打开下拉  
功能：显示下拉面板  
返回值：无  

```javascript
// 调用格式:
组件名.Open();
```

### 4 Close()
方法名称：关闭下拉  
功能：隐藏下拉面板  
返回值：无  

```javascript
// 调用格式:
组件名.Close();
```

### 5 SetValue(val)
方法名称：设置值  
功能：设置当前选中值  
参数说明：val，any  
返回值：无  

```javascript
// 调用格式:
组件名.SetValue("option1");
```

### 6 GetValue()
方法名称：获取值  
功能：获取当前选中值  
返回值：any  

```javascript
// 调用格式:
var val = 组件名.GetValue();
```

### 7 SetData(data)
方法名称：设置选项  
功能：设置选择器数据  
参数说明：data，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetData([{ label: "A", value: "a" }]);
```

### 8 GetData()
方法名称：获取选项  
功能：获取选择器数据  
返回值：array  

```javascript
// 调用格式:
var data = 组件名.GetData();
```

### 9 Clear()
方法名称：清空  
功能：清空选中  
返回值：无  

```javascript
// 调用格式:
组件名.Clear();
```

## 开关（Switch）

### 1 SetValue(bool)
方法名称：设置值  
功能：设置开关状态  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetValue(true);
```

### 2 GetValue()
方法名称：获取值  
功能：获取开关状态  
返回值：boolean  

```javascript
// 调用格式:
var val = 组件名.GetValue();
```

### 3 Toggle()
方法名称：切换状态  
功能：切换开关  
返回值：无  

```javascript
// 调用格式:
组件名.Toggle();
```

### 4 Disable(bool)
方法名称：设置禁用  
功能：设置开关禁用  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.Disable(true);
```

## 表格（Table）

### 1 ClearSelection()
方法名称：清空用户选择  
功能：用于多选表格，清空用户的选择  
返回值：无  

```javascript
// 调用格式:
组件名.ClearSelection();
```

### 2 AppendRow(var Row)
方法名称：追加一行  
功能：在表格中追加新行  
参数说明：Row，object  
返回值：无  

```javascript
// 调用格式:
var data = { id: 1, name: "jack" };
组件名.AppendRow(data);
```

### 3 ToggleRowSelection(var Row, var Selected)
方法名称：切换行选中  
功能：切换某一行选中状态  
参数说明：Row，object；Selected，boolean  
返回值：无  

```javascript
// 调用格式:
var rows = 组件名.GetData();
组件名.ToggleRowSelection(rows[0], true);
```

### 4 ToggleAllSelection()
方法名称：切换全选  
功能：切换所有行选中状态  
返回值：无  

```javascript
// 调用格式:
组件名.ToggleAllSelection();
```

### 5 ToggleRowExpansion(var ToggleRow, var Expanded)
方法名称：切换展开  
功能：展开/折叠某行  
参数说明：ToggleRow，object；Expanded，boolean  
返回值：无  

```javascript
// 调用格式:
var row = 组件名.GetData()[0];
组件名.ToggleRowExpansion(row, true);
```

### 6 SetCurrentRow(var Row)
方法名称：设置当前行  
功能：设置单选表格当前选中行  
参数说明：Row，object  
返回值：无  

```javascript
// 调用格式:
组件名.SetCurrentRow(组件名.GetData()[0]);
```

### 7 ClearSort()
方法名称：清空排序  
功能：清空排序条件  
返回值：无  

```javascript
// 调用格式:
组件名.ClearSort();
```

### 8 ClearFilter()
方法名称：清空过滤  
功能：清空过滤条件  
返回值：无  

```javascript
// 调用格式:
组件名.ClearFilter();
```

### 9 Dolayout()
方法名称：重新布局  
功能：触发表格重新布局  
返回值：无  

```javascript
// 调用格式:
组件名.Dolayout();
```

### 10 Sort(var Prop, var Order)
方法名称：排序  
功能：按指定列排序  
参数说明：Prop，string；Order，string  
返回值：无  

```javascript
// 调用格式:
组件名.Sort("date", "descending");
```

### 11 SetData(var DataArr)
方法名称：设置数据  
功能：设置表格数据  
参数说明：DataArr，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetData([{ name: "A" }]);
```

### 12 GetData()
方法名称：获取数据  
功能：获取表格数据  
返回值：array  

```javascript
// 调用格式:
var data = 组件名.GetData();
```

### 13 GetSelection()
方法名称：获取选中行  
功能：返回选中行列表  
返回值：array  

```javascript
// 调用格式:
var rows = 组件名.GetSelection();
```

### 14 SetSelectionByKeys(keys)
方法名称：按 key 选中  
功能：根据 key 选中行  
参数说明：keys，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetSelectionByKeys([1, 2]);
```

### 15 GetSelectionKeys()
方法名称：获取选中 key  
功能：返回选中行 key  
返回值：array  

```javascript
// 调用格式:
var keys = 组件名.GetSelectionKeys();
```

### 16 SetPage(page)
方法名称：设置页码  
功能：设置当前页  
参数说明：page，number  
返回值：无  

```javascript
// 调用格式:
组件名.SetPage(2);
```

### 17 SetPageSize(size)
方法名称：设置页大小  
功能：设置每页条数  
参数说明：size，number  
返回值：无  

```javascript
// 调用格式:
组件名.SetPageSize(20);
```

### 18 GetPageData()
方法名称：获取分页数据  
功能：获取当前页数据  
返回值：array  

```javascript
// 调用格式:
var pageData = 组件名.GetPageData();
```

### 19 UpdateRowByKey(key, patch)
方法名称：更新行  
功能：根据 key 更新行  
参数说明：key，string/number；patch，object  
返回值：无  

```javascript
// 调用格式:
组件名.UpdateRowByKey(1, { name: "New" });
```

### 20 RemoveRowByKey(key)
方法名称：删除行  
功能：根据 key 删除行  
参数说明：key，string/number  
返回值：无  

```javascript
// 调用格式:
组件名.RemoveRowByKey(1);
```

### 21 UpsertRowByKey(key, row)
方法名称：插入/更新行  
功能：不存在则插入，存在则更新  
参数说明：key，string/number；row，object  
返回值：无  

```javascript
// 调用格式:
组件名.UpsertRowByKey(1, { id: 1, name: "X" });
```

### 22 ScrollToTop()
方法名称：滚动到顶部  
功能：表格滚动到顶部  
返回值：无  

```javascript
// 调用格式:
组件名.ScrollToTop();
```

### 23 ScrollToRow(keyOrRow)
方法名称：滚动到行  
功能：滚动到指定行  
参数说明：keyOrRow，string/number/object  
返回值：无  

```javascript
// 调用格式:
组件名.ScrollToRow(1);
```

### 24 DoLayoutSafe()
方法名称：安全布局  
功能：nextTick 后执行 doLayout  
返回值：无  

```javascript
// 调用格式:
组件名.DoLayoutSafe();
```

## 树（Tree）

### 1 UpdateKeyChildren(key, data)
方法名称：设置子节点  
功能：通过 key 设置子节点  
参数说明：key，string/number；data，array  
返回值：无  

```javascript
// 调用格式:
组件名.UpdateKeyChildren(3, [{ id: 7, label: "子节点" }]);
```

### 2 GetCheckedNodes(leafOnly, includeHalfChecked)
方法名称：获取选中节点  
功能：获取选中节点列表  
参数说明：leafOnly，boolean；includeHalfChecked，boolean  
返回值：array  

```javascript
// 调用格式:
var nodes = 组件名.GetCheckedNodes(false, true);
```

### 3 SetCheckedNodes(nodes)
方法名称：设置选中节点  
功能：设置选中的节点  
参数说明：nodes，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetCheckedNodes([{ id: 5 }]);
```

### 4 GetCheckedKeys(leafOnly)
方法名称：获取选中 key  
功能：获取选中节点 key  
参数说明：leafOnly，boolean  
返回值：array  

```javascript
// 调用格式:
var keys = 组件名.GetCheckedKeys(true);
```

### 5 SetCheckedKeys(keys, leafOnly)
方法名称：设置选中 key  
功能：通过 key 设置选中  
参数说明：keys，array；leafOnly，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetCheckedKeys([3], true);
```

### 6 SetChecked(keyOrData, checked, deep)
方法名称：设置选中状态  
功能：设置单节点选中  
参数说明：keyOrData，any；checked，boolean；deep，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetChecked(3, true, true);
```

### 7 GetHalfCheckedNodes()
方法名称：获取半选节点  
功能：获取半选节点列表  
返回值：array  

```javascript
// 调用格式:
var nodes = 组件名.GetHalfCheckedNodes();
```

### 8 GetHalfCheckedKeys()
方法名称：获取半选 key  
功能：获取半选节点 key  
返回值：array  

```javascript
// 调用格式:
var keys = 组件名.GetHalfCheckedKeys();
```

### 9 GetCurrentKey()
方法名称：获取当前 key  
功能：获取当前选中 key  
返回值：string/number  

```javascript
// 调用格式:
var key = 组件名.GetCurrentKey();
```

### 10 GetCurrentNode()
方法名称：获取当前节点  
功能：获取当前选中节点  
返回值：object  

```javascript
// 调用格式:
var node = 组件名.GetCurrentNode();
```

### 11 SetCurrentKey(key)
方法名称：设置当前 key  
功能：设置当前选中 key  
参数说明：key，string/number  
返回值：无  

```javascript
// 调用格式:
组件名.SetCurrentKey(3);
```

### 12 SetCurrentNode(node)
方法名称：设置当前节点  
功能：设置当前选中节点  
参数说明：node，object  
返回值：无  

```javascript
// 调用格式:
组件名.SetCurrentNode({ id: 9, label: "节点" });
```

### 13 GetNode(dataOrKey)
方法名称：获取节点  
功能：通过 data 或 key 获取节点  
参数说明：dataOrKey，object/string/number  
返回值：object  

```javascript
// 调用格式:
var node = 组件名.GetNode(9);
```

### 14 Remove(dataOrNode)
方法名称：删除节点  
功能：删除指定节点  
参数说明：dataOrNode，object  
返回值：无  

```javascript
// 调用格式:
组件名.Remove({ id: 2 });
```

### 15 Append(data, parentNode)
方法名称：追加节点  
功能：为节点追加子节点  
参数说明：data，object；parentNode，any  
返回值：无  

```javascript
// 调用格式:
组件名.Append({ id: 2, label: "新节点" }, 3);
```

### 16 InsertBefore(data, refNode)
方法名称：前插节点  
功能：在指定节点前插入  
参数说明：data，object；refNode，any  
返回值：无  

```javascript
// 调用格式:
组件名.InsertBefore({ id: 9 }, 10);
```

### 17 InsertAfter(data, refNode)
方法名称：后插节点  
功能：在指定节点后插入  
参数说明：data，object；refNode，any  
返回值：无  

```javascript
// 调用格式:
组件名.InsertAfter({ id: 9 }, 10);
```

### 18 ExpandAll()
方法名称：全部展开  
功能：展开所有节点  
返回值：无  

```javascript
// 调用格式:
组件名.ExpandAll();
```

### 19 CollapseAll()
方法名称：全部收起  
功能：收起所有节点  
返回值：无  

```javascript
// 调用格式:
组件名.CollapseAll();
```

### 20 SetExpandedKeys(keys)
方法名称：设置展开 key  
功能：设置展开节点 key  
参数说明：keys，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetExpandedKeys([1, 2]);
```

### 21 GetExpandedKeys()
方法名称：获取展开 key  
功能：获取展开节点 key  
返回值：array  

```javascript
// 调用格式:
var keys = 组件名.GetExpandedKeys();
```

### 22 Filter(keyword)
方法名称：过滤  
功能：过滤树节点  
参数说明：keyword，string  
返回值：无  

```javascript
// 调用格式:
组件名.Filter("关键字");
```

## 下拉菜单（Dropdown）

### 1 InsertItem(item)
方法名称：插入菜单项  
功能：插入菜单项  
参数说明：item，object  
返回值：无  

```javascript
// 调用格式:
组件名.InsertItem({ text: "清空", command: 8 });
```

### 2 GetCommandItem(menuItem)
方法名称：获取菜单指令  
功能：获取菜单项 command  
参数说明：menuItem，string/object  
返回值：any  

```javascript
// 调用格式:
var cmd = 组件名.GetCommandItem("清空");
```

### 3 GetMenuItem(command)
方法名称：获取菜单项  
功能：通过 command 获取菜单项  
参数说明：command，any  
返回值：object  

```javascript
// 调用格式:
var item = 组件名.GetMenuItem(8);
```

### 4 DeleteItem(menuItem)
方法名称：删除菜单项  
功能：删除指定菜单项  
参数说明：menuItem，string  
返回值：无  

```javascript
// 调用格式:
组件名.DeleteItem("清空");
```

### 5 ClearAll()
方法名称：清空菜单  
功能：清空所有菜单项  
返回值：无  

```javascript
// 调用格式:
组件名.ClearAll();
```

### 6 Open()
方法名称：打开菜单  
功能：显示下拉菜单  
返回值：无  

```javascript
// 调用格式:
组件名.Open();
```

### 7 Close()
方法名称：关闭菜单  
功能：隐藏下拉菜单  
返回值：无  

```javascript
// 调用格式:
组件名.Close();
```

### 8 Toggle()
方法名称：切换菜单  
功能：切换显示/隐藏  
返回值：无  

```javascript
// 调用格式:
组件名.Toggle();
```

### 9 GetVisible()
方法名称：获取可见状态  
功能：获取菜单显示状态  
返回值：boolean  

```javascript
// 调用格式:
var visible = 组件名.GetVisible();
```

## 导航菜单（Menu）

### 1 Open(index)
方法名称：展开菜单  
功能：展开子菜单  
参数说明：index，string  
返回值：无  

```javascript
// 调用格式:
组件名.Open("1");
```

### 2 Close(index)
方法名称：收起菜单  
功能：收起子菜单  
参数说明：index，string  
返回值：无  

```javascript
// 调用格式:
组件名.Close("1");
```

### 3 SetActive(index)
方法名称：设置激活项  
功能：设置当前激活菜单  
参数说明：index，string  
返回值：无  

```javascript
// 调用格式:
组件名.SetActive("1");
```

### 4 GetActive()
方法名称：获取激活项  
功能：获取当前激活菜单  
返回值：string  

```javascript
// 调用格式:
var active = 组件名.GetActive();
```

### 5 Collapse(bool)
方法名称：折叠菜单  
功能：折叠/展开菜单  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.Collapse(true);
```

## 单选框（Radio）

### 1 GetRadioChecked()
方法名称：获取选中序号  
功能：获取当前选中项序号  
返回值：number  

```javascript
// 调用格式:
var index = 组件名.GetRadioChecked();
```

### 2 GetRadioValue(labelIndex)
方法名称：获取选项值  
功能：通过索引获取选项值  
参数说明：labelIndex，number  
返回值：string  

```javascript
// 调用格式:
var val = 组件名.GetRadioValue(0);
```

### 3 GetRadioLabel(value)
方法名称：获取选项标签  
功能：通过 value 获取 label  
参数说明：value，string  
返回值：string  

```javascript
// 调用格式:
var label = 组件名.GetRadioLabel("option1");
```

### 4 SetRadioEnable(labelIndex, enable)
方法名称：设置使能  
功能：设置选项使能  
参数说明：labelIndex，number；enable，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetRadioEnable(0, false);
```

### 5 GetRadioEnable(labelIndex)
方法名称：获取使能  
功能：获取选项使能  
参数说明：labelIndex，number  
返回值：boolean  

```javascript
// 调用格式:
var enabled = 组件名.GetRadioEnable(1);
```

### 6 SetRadioVisible(labelIndex, visible)
方法名称：设置可见  
功能：设置选项可见  
参数说明：labelIndex，number；visible，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetRadioVisible(0, false);
```

### 7 GetRadioVisible(labelIndex)
方法名称：获取可见  
功能：获取选项可见  
参数说明：labelIndex，number  
返回值：boolean  

```javascript
// 调用格式:
var visible = 组件名.GetRadioVisible(0);
```

### 8 SetValue(val)
方法名称：设置值  
功能：设置当前选中值  
参数说明：val，any  
返回值：无  

```javascript
// 调用格式:
组件名.SetValue("option1");
```

### 9 GetValue()
方法名称：获取值  
功能：获取当前选中值  
返回值：any  

```javascript
// 调用格式:
var val = 组件名.GetValue();
```

### 10 Clear()
方法名称：清空  
功能：清空选中  
返回值：无  

```javascript
// 调用格式:
组件名.Clear();
```

## 多选框（Checkbox）

### 1 GetCheckState(labelIndex)
方法名称：获取选项状态  
功能：获取选项是否选中  
参数说明：labelIndex，number  
返回值：boolean  

```javascript
// 调用格式:
var checked = 组件名.GetCheckState(1);
```

### 2 SetCheckState(labelIndex, state)
方法名称：设置选项状态  
功能：设置选项选中状态  
参数说明：labelIndex，number；state，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetCheckState(1, true);
```

### 3 SetCheckEnable(labelIndex, enable)
方法名称：设置使能  
功能：设置选项使能  
参数说明：labelIndex，number；enable，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetCheckEnable(0, false);
```

### 4 GetCheckEnable(labelIndex)
方法名称：获取使能  
功能：获取选项使能  
参数说明：labelIndex，number  
返回值：boolean  

```javascript
// 调用格式:
var enabled = 组件名.GetCheckEnable(1);
```

### 5 SetCheckVisible(labelIndex, visible)
方法名称：设置可见  
功能：设置选项可见  
参数说明：labelIndex，number；visible，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetCheckVisible(0, false);
```

### 6 GetCheckVisible(labelIndex)
方法名称：获取可见  
功能：获取选项可见  
参数说明：labelIndex，number  
返回值：boolean  

```javascript
// 调用格式:
var visible = 组件名.GetCheckVisible(1);
```

### 7 CheckAll(bool)
方法名称：全选  
功能：全选/取消全选  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.CheckAll(true);
```

### 8 SetValue(val)
方法名称：设置值  
功能：设置选中值  
参数说明：val，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetValue(["option1"]);
```

### 9 GetValue()
方法名称：获取值  
功能：获取选中值  
返回值：array  

```javascript
// 调用格式:
var val = 组件名.GetValue();
```

### 10 Clear()
方法名称：清空  
功能：清空选中  
返回值：无  

```javascript
// 调用格式:
组件名.Clear();
```

## 级联选择器（Cascader）

### 1 Open()
方法名称：打开  
功能：打开级联面板  
返回值：无  

```javascript
// 调用格式:
组件名.Open();
```

### 2 Close()
方法名称：关闭  
功能：关闭级联面板  
返回值：无  

```javascript
// 调用格式:
组件名.Close();
```

### 3 Toggle()
方法名称：切换  
功能：切换显示/隐藏  
返回值：无  

```javascript
// 调用格式:
组件名.Toggle();
```

### 4 GetCheckedNodes(leafOnly)
方法名称：获取选中节点  
功能：获取选中节点  
参数说明：leafOnly，boolean  
返回值：array  

```javascript
// 调用格式:
var nodes = 组件名.GetCheckedNodes();
```

### 5 SetData(data)
方法名称：设置数据  
功能：设置级联数据  
参数说明：data，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetData([{ value: "a", label: "A" }]);
```

### 6 GetData()
方法名称：获取数据  
功能：获取级联数据  
返回值：array  

```javascript
// 调用格式:
var data = 组件名.GetData();
```

### 7 SetValue(val)
方法名称：设置值  
功能：设置当前值  
参数说明：val，any  
返回值：无  

```javascript
// 调用格式:
组件名.SetValue(["a"]);
```

### 8 GetValue()
方法名称：获取值  
功能：获取当前值  
返回值：any  

```javascript
// 调用格式:
var val = 组件名.GetValue();
```

### 9 Clear()
方法名称：清空  
功能：清空选中  
返回值：无  

```javascript
// 调用格式:
组件名.Clear();
```

## 标签页（Tabs）

### 1 SetActive(name)
方法名称：设置激活  
功能：设置当前标签  
参数说明：name，string  
返回值：无  

```javascript
// 调用格式:
组件名.SetActive("tab1");
```

### 2 GetActive()
方法名称：获取激活  
功能：获取当前标签  
返回值：string  

```javascript
// 调用格式:
var name = 组件名.GetActive();
```

### 3 Next()
方法名称：下一个  
功能：切换到下一个标签  
返回值：无  

```javascript
// 调用格式:
组件名.Next();
```

### 4 Prev()
方法名称：上一个  
功能：切换到上一个标签  
返回值：无  

```javascript
// 调用格式:
组件名.Prev();
```

### 5 AddTab(tab)
方法名称：新增标签  
功能：新增标签页  
参数说明：tab，object  
返回值：无  

```javascript
// 调用格式:
组件名.AddTab({ name: "tab4", label: "标签四" });
```

### 6 RemoveTab(name)
方法名称：删除标签  
功能：删除指定标签页  
参数说明：name，string  
返回值：无  

```javascript
// 调用格式:
组件名.RemoveTab("tab1");
```

## 穿梭框（Transfer）

### 1 SetData(leftData, rightData)
方法名称：设置数据  
功能：设置左侧/右侧数据  
参数说明：leftData，array；rightData，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetData([{ key: 1, label: "A" }], [1]);
```

### 2 GetData()
方法名称：获取数据  
功能：获取穿梭框数据  
返回值：object  

```javascript
// 调用格式:
var data = 组件名.GetData();
```

### 3 ClearQuery(area)
方法名称：清空搜索  
功能：清空某侧搜索  
参数说明：area，string  
返回值：无  

```javascript
// 调用格式:
组件名.ClearQuery("left");
```

### 4 SetValue(targetKeys)
方法名称：设置目标  
功能：设置右侧选中  
参数说明：targetKeys，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetValue([1, 2]);
```

### 5 GetValue()
方法名称：获取目标  
功能：获取右侧选中  
返回值：array  

```javascript
// 调用格式:
var keys = 组件名.GetValue();
```

### 6 Clear()
方法名称：清空  
功能：清空选中  
返回值：无  

```javascript
// 调用格式:
组件名.Clear();
```

### 7 MoveToRight(keys)
方法名称：移动到右侧  
功能：移动指定 key 到右侧  
参数说明：keys，array  
返回值：无  

```javascript
// 调用格式:
组件名.MoveToRight([1, 2]);
```

### 8 MoveToLeft(keys)
方法名称：移动到左侧  
功能：移动指定 key 到左侧  
参数说明：keys，array  
返回值：无  

```javascript
// 调用格式:
组件名.MoveToLeft([1]);
```

## 标签（Tag）

### 1 Close()
方法名称：关闭  
功能：关闭标签  
返回值：无  

```javascript
// 调用格式:
组件名.Close();
```

### 2 SetType(type)
方法名称：设置类型  
功能：设置标签类型  
参数说明：type，string  
返回值：无  

```javascript
// 调用格式:
组件名.SetType("success");
```

### 3 SetText(text)
方法名称：设置文本  
功能：设置标签文本  
参数说明：text，string  
返回值：无  

```javascript
// 调用格式:
组件名.SetText("标签");
```

## 计数器（InputNumber）

### 1 SetValue(num)
方法名称：设置值  
功能：设置数值  
参数说明：num，number  
返回值：无  

```javascript
// 调用格式:
组件名.SetValue(10);
```

### 2 GetValue()
方法名称：获取值  
功能：获取当前值  
返回值：number  

```javascript
// 调用格式:
var val = 组件名.GetValue();
```

### 3 Increase(step)
方法名称：递增  
功能：增加数值  
参数说明：step，number  
返回值：无  

```javascript
// 调用格式:
组件名.Increase(1);
```

### 4 Decrease(step)
方法名称：递减  
功能：减少数值  
参数说明：step，number  
返回值：无  

```javascript
// 调用格式:
组件名.Decrease(1);
```

### 5 Focus()
方法名称：获取焦点  
功能：计数器获取焦点  
返回值：无  

```javascript
// 调用格式:
组件名.Focus();
```

### 6 Blur()
方法名称：失去焦点  
功能：计数器失去焦点  
返回值：无  

```javascript
// 调用格式:
组件名.Blur();
```

## 时间线（Timeline）

### 1 SetItems(items)
方法名称：设置数据  
功能：设置时间线数据  
参数说明：items，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetItems([{ label: "步骤一" }]);
```

### 2 AppendItem(item)
方法名称：追加数据  
功能：追加时间线节点  
参数说明：item，object  
返回值：无  

```javascript
// 调用格式:
组件名.AppendItem({ label: "步骤二" });
```

### 3 Clear()
方法名称：清空  
功能：清空时间线  
返回值：无  

```javascript
// 调用格式:
组件名.Clear();
```

## 轮播（Carousel）

### 1 SetActiveItem(indexOrName)
方法名称：设置激活项  
功能：切换轮播  
参数说明：indexOrName，number/string  
返回值：无  

```javascript
// 调用格式:
组件名.SetActiveItem(1);
```

### 2 Prev()
方法名称：上一张  
功能：切换上一张  
返回值：无  

```javascript
// 调用格式:
组件名.Prev();
```

### 3 Next()
方法名称：下一张  
功能：切换下一张  
返回值：无  

```javascript
// 调用格式:
组件名.Next();
```

### 4 Play()
方法名称：播放  
功能：开启自动播放  
返回值：无  

```javascript
// 调用格式:
组件名.Play();
```

### 5 Pause()
方法名称：暂停  
功能：暂停自动播放  
返回值：无  

```javascript
// 调用格式:
组件名.Pause();
```

### 6 SetItems(items)
方法名称：设置数据  
功能：设置轮播数据  
参数说明：items，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetItems([{ label: "轮播一" }]);
```

## 网页容器（WebContainer）

### 1 Load(url)
方法名称：加载 URL  
功能：设置网页地址  
参数说明：url，string  
返回值：无  

```javascript
// 调用格式:
组件名.Load("https://example.com");
```

### 2 Reload()
方法名称：刷新  
功能：刷新网页容器  
返回值：无  

```javascript
// 调用格式:
组件名.Reload();
```

### 3 PostMessage(data)
方法名称：消息通信  
功能：向网页发送消息  
参数说明：data，any  
返回值：无  

```javascript
// 调用格式:
组件名.PostMessage({ type: "ping" });
```

### 4 GetUrl()
方法名称：获取地址  
功能：获取当前 URL  
返回值：string  

```javascript
// 调用格式:
var url = 组件名.GetUrl();
```

### 5 Back()
方法名称：后退  
功能：浏览器后退  
返回值：无  

```javascript
// 调用格式:
组件名.Back();
```

### 6 Forward()
方法名称：前进  
功能：浏览器前进  
返回值：无  

```javascript
// 调用格式:
组件名.Forward();
```

## 步骤条（Steps）

### 1 SetActive(stepIndex)
方法名称：设置当前步  
功能：设置激活步骤  
参数说明：stepIndex，number  
返回值：无  

```javascript
// 调用格式:
组件名.SetActive(2);
```

### 2 Next()
方法名称：下一步  
功能：步进到下一步  
返回值：无  

```javascript
// 调用格式:
组件名.Next();
```

### 3 Prev()
方法名称：上一步  
功能：回退到上一步  
返回值：无  

```javascript
// 调用格式:
组件名.Prev();
```

### 4 Reset()
方法名称：重置  
功能：重置步骤条  
返回值：无  

```javascript
// 调用格式:
组件名.Reset();
```

## 卡片（Card）

### 1 SetTitle(title)
方法名称：设置标题  
功能：设置卡片标题  
参数说明：title，string  
返回值：无  

```javascript
// 调用格式:
组件名.SetTitle("标题");
```

### 2 SetLoading(bool)
方法名称：设置加载  
功能：设置卡片加载态  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetLoading(true);
```

### 3 Collapse(bool)
方法名称：折叠  
功能：折叠/展开卡片  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.Collapse(true);
```

## 分页（Pagination）

### 1 SetPage(page)
方法名称：设置页码  
功能：设置当前页  
参数说明：page，number  
返回值：无  

```javascript
// 调用格式:
组件名.SetPage(2);
```

### 2 GetPage()
方法名称：获取页码  
功能：获取当前页  
返回值：number  

```javascript
// 调用格式:
var page = 组件名.GetPage();
```

### 3 SetPageSize(size)
方法名称：设置页大小  
功能：设置每页条数  
参数说明：size，number  
返回值：无  

```javascript
// 调用格式:
组件名.SetPageSize(20);
```

### 4 GetPageSize()
方法名称：获取页大小  
功能：获取每页条数  
返回值：number  

```javascript
// 调用格式:
var size = 组件名.GetPageSize();
```

### 5 Reset()
方法名称：重置  
功能：重置到第一页  
返回值：无  

```javascript
// 调用格式:
组件名.Reset();
```

### 6 SetTotal(total)
方法名称：设置总数  
功能：设置总条数  
参数说明：total，number  
返回值：无  

```javascript
// 调用格式:
组件名.SetTotal(1000);
```

### 7 GetTotal()
方法名称：获取总数  
功能：获取总条数  
返回值：number  

```javascript
// 调用格式:
var total = 组件名.GetTotal();
```

## 折叠面板（Collapse）

### 1 Open(nameOrNames)
方法名称：展开面板  
功能：展开指定面板  
参数说明：nameOrNames，string/array  
返回值：无  

```javascript
// 调用格式:
组件名.Open("panel1");
```

### 2 Close(nameOrNames)
方法名称：收起面板  
功能：收起指定面板  
参数说明：nameOrNames，string/array  
返回值：无  

```javascript
// 调用格式:
组件名.Close("panel1");
```

### 3 Toggle(name)
方法名称：切换面板  
功能：切换展开/收起  
参数说明：name，string  
返回值：无  

```javascript
// 调用格式:
组件名.Toggle("panel1");
```

### 4 GetActiveNames()
方法名称：获取激活项  
功能：获取当前展开项  
返回值：array  

```javascript
// 调用格式:
var names = 组件名.GetActiveNames();
```

### 5 SetActiveNames(names)
方法名称：设置激活项  
功能：设置展开项  
参数说明：names，array  
返回值：无  

```javascript
// 调用格式:
组件名.SetActiveNames(["panel1"]);
```

## 业务卡片（BusinessCard）

### 1 SetData(data)
方法名称：设置数据  
功能：设置业务卡片数据  
参数说明：data，object  
返回值：无  

```javascript
// 调用格式:
组件名.SetData({ id: 1 });
```

### 2 GetData()
方法名称：获取数据  
功能：获取业务卡片数据  
返回值：object  

```javascript
// 调用格式:
var data = 组件名.GetData();
```

### 3 SetLoading(bool)
方法名称：设置加载  
功能：设置加载状态  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.SetLoading(true);
```

### 4 Refresh()
方法名称：刷新  
功能：刷新业务卡片  
返回值：无  

```javascript
// 调用格式:
组件名.Refresh();
```

### 5 OpenDetail(id)
方法名称：打开详情  
功能：打开详情页  
参数说明：id，any  
返回值：无  

```javascript
// 调用格式:
组件名.OpenDetail(1);
```

## 条形码（Barcode）

### 1 SetValue(text)
方法名称：设置内容  
功能：设置条形码内容  
参数说明：text，string  
返回值：无  

```javascript
// 调用格式:
组件名.SetValue("123456");
```

### 2 GetValue()
方法名称：获取内容  
功能：获取条形码内容  
返回值：string  

```javascript
// 调用格式:
var text = 组件名.GetValue();
```

### 3 Render()
方法名称：渲染  
功能：重新渲染条形码  
返回值：无  

```javascript
// 调用格式:
组件名.Render();
```

### 4 Download(format)
方法名称：导出  
功能：导出条形码  
参数说明：format，string  
返回值：无  

```javascript
// 调用格式:
组件名.Download("png");
```

## 滑块（Slider）

### 1 SetValue(val)
方法名称：设置值  
功能：设置滑块值  
参数说明：val，number  
返回值：无  

```javascript
// 调用格式:
组件名.SetValue(30);
```

### 2 GetValue()
方法名称：获取值  
功能：获取滑块值  
返回值：number  

```javascript
// 调用格式:
var val = 组件名.GetValue();
```

### 3 Reset()
方法名称：重置  
功能：重置滑块值  
返回值：无  

```javascript
// 调用格式:
组件名.Reset();
```

### 4 Disable(bool)
方法名称：设置禁用  
功能：设置禁用状态  
参数说明：bool，boolean  
返回值：无  

```javascript
// 调用格式:
组件名.Disable(true);
```

## 日历（Calendar）

### 1 SetDate(date)
方法名称：设置日期  
功能：设置日历日期  
参数说明：date，Date/string  
返回值：无  

```javascript
// 调用格式:
组件名.SetDate(new Date());
```

### 2 GetDate()
方法名称：获取日期  
功能：获取当前日期  
返回值：Date/string  

```javascript
// 调用格式:
var date = 组件名.GetDate();
```

### 3 Today()
方法名称：回到今天  
功能：设置为今天  
返回值：无  

```javascript
// 调用格式:
组件名.Today();
```

## 电子签名（Signature）

### 1 Clear()
方法名称：清空  
功能：清空签名  
返回值：无  

```javascript
// 调用格式:
组件名.Clear();
```

### 2 GetImage()
方法名称：获取图片  
功能：获取签名图片  
返回值：string  

```javascript
// 调用格式:
var img = 组件名.GetImage();
```

### 3 SetImage(value)
方法名称：设置图片  
功能：设置签名图片  
参数说明：value，string  
返回值：无  

```javascript
// 调用格式:
组件名.SetImage("data:image/png;base64,xxx");
```

### 4 IsEmpty()
方法名称：是否为空  
功能：判断是否空签名  
返回值：boolean  

```javascript
// 调用格式:
var empty = 组件名.IsEmpty();
```

### 5 SetPen(color, width)
方法名称：设置画笔  
功能：设置画笔样式  
参数说明：color，string；width，number  
返回值：无  

```javascript
// 调用格式:
组件名.SetPen("#000000", 2);
```
