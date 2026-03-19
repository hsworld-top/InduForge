# 老版本页面编辑器实现细节分析报告

本文档基于 exe 项目（`D:\SVNCode\exe`）中老版本页面编辑器的代码，对**布局容器（以水平布局为例）**、**页面**、**UI 组件（以按钮为例）**、**页面属性设置**、**UI 组件属性设置**、**布局属性设置**的实现细节进行梳理，供 designer 新版本开发与细节完善参考。

---

## 一、整体架构概览

### 1.1 页面编辑器入口与布局

- **入口**：`kmui_page_editor.js`（`KMUIPageEditor`）
  - 继承 `KMUIEditorBase`，`editorType = kmEditorEnum.KM_PAGE_EDITOR`
  - 创建：主菜单、工具栏、状态栏、图层对象、工具箱（`KMUIPageToolKit`）、画布视图（`KMUIPageView`）
  - 整体为 **EasyUI Layout** 五区布局：北（菜单/工具栏）、南（状态栏）、西（工具箱）、东（属性/连接/变量/自定义属性/图层面板）、中（画布）

### 1.2 右侧 Tab 与属性窗

- 东区为 Tab：属性(IDS_PROPERTY)、连接、变量、自定义属性、图层
- 属性 Tab 内挂载 `propertyObj`（`KMUIProperty`），通过 `addObjPropToPropgrid(jsonArray)` 展示当前选中对象的属性行

### 1.3 数据与选中驱动

- 画布为 `KMUIPageView`，内部持有 `page`（`KMPage`）、`objectMng`、`selectMng` 等
- 选中变化时，调用 `currentEditor.propertyObj.addObjPropToPropgrid(objJson)`，其中 `objJson` 来自当前选中对象的 `propertyToJson(currentEditor)`
- 属性编辑后通过 `KMSignalManager.dispatch(..., kmuiSignalSet.MSG_ONEXECPROPERTY, rowData, ...)` 回写到模型，再由各组件实现 setter 更新 DOM/样式

---

## 二、布局容器（以水平布局为例）

### 2.1 类型与继承关系

- **基类**：`KMLayoutCompBase`（`kmui_layoutcomponentbase.js`）继承 `KMComponentBase`
  - `objectType = objectType.OBJECTTYPE_LAYOUT`
  - 定义布局类型枚举 `kmLayoutTypeEnum`（HORIZONTAL、VERTICAL、TAB、ACCORDION、DOCKINGLAYOUT、TWODIMENSIONAL 等）
- **水平布局**：`KMHorizontalLayout`（`kmui_horizontal_layout.js`）继承 `KMLayoutCompBase`
  - `layoutType = kmLayoutTypeEnum.HORIZONTAL`
  - 最外层 div 类名：`km-layoutH`，`boxSizing: 'border-box'`

### 2.2 数据结构

- **panelMap**：`Map()`，key 为区块索引（0,1,2,...），value 为该区块 div 的 id
- **componentMap**：`Map()`，key 为区块索引，value 为该区块内放置的**组件对象**（布局、UI、页面等）
- **间隙**：`interval`（默认 10），水平布局中为区块之间的固定间距

### 2.3 尺寸策略（与布局强相关）

- 每个组件（含布局）都有 **sizePolicy**（`KMSizePolicy`）：
  - `sizePolicy`：`[水平策略, 垂直策略]`，取值为 `kmSizePolicyTypeEnum.FIXED` 或 `EXPANDING`
  - `minSize`、`maxSize`、`fixedSize`：二维数组 `[宽, 高]`
- 水平布局在 **resetRegionPos** 中：
  - 先按所有子组件汇总 limitValue、fixedValue、beforeExpandingValue、expandingCount
  - 再根据子组件水平/垂直是固定还是可延展，计算每个区块的宽高（setBlockSize）和位置（vertexSize）
  - 可延展方向：水平方向均分剩余宽度（减去间隙），垂直方向可撑满或按策略分配

### 2.4 区块的增删与子对象追加

- **addRegion(regionJson)**：
  - 若 `regionJson.region` 不存在于 panelMap，则新建一个 `KMUIDiv` 作为区块，生成 UUID 作为 id，样式为 `float:left`、高度 99%、宽度按 `100/(beforeSize+1)` 百分比
  - 支持 `nodeInsertPos` 在指定位置插入；否则追加到末尾
  - 将区块 id 存入 panelMap
- **removeRegion(region)**：根据 region 从 DOM 移除对应 div，并删除 panelMap/componentMap 中该项，若仍有区块则调用 `resetRegionPos()` 重新计算
- **appendChild(region, baseobj)**：根据 region 取到区块 div，执行 `regionDiv.appendChild(baseobj.thisElement)`，并 `componentMap.set(region, baseobj)`

### 2.5 序列化 / 反序列化

- **objectToJson**：
  - 输出 baseProperty（name、leftTopPt、width、height、comment、lock、visible、guid、memberAccess 等）、layoutType、objectType
  - insiderObjects：按 **DOM 子节点顺序** 遍历每个区块的第一个子节点，找到 componentMap 中对应对象，调用其 `objectToJson()`，并附带 `internalVertex: { left, top }`
  - 同时保存 version、sizePolicy、fixedSize、isShowBorder、interval、memberAccess
- **jsonToObject**：
  - 解析 baseProperty、layoutType、insiderObjects、version、sizePolicy、fixedSize、isShowBorder、interval、memberAccess
  - 对 insiderObjects 每一项根据 objectType/uiExtendType/uiModelType/layoutType 等用工厂（gModelObjectFactory、gUIObjectFactory、gUIModelObjectFactory、gLayoutObjectFactory）创建实例，再 jsonToObject
  - 通过 getRegionDiv(self, undefined) 得到新区块，appendChild(region, newModelObj)，并设置 internalVertex 或 pageProperty.pagePos 的 left/top
  - 开发态下对子布局/Window/Dialog 绑定事件，并对 Accordion/Tab 绑定 onSelect/onUnselect 以刷新属性窗

### 2.6 布局属性在属性窗中的展示（propertyToJson）

- 水平布局的 **propertyToJson(currentEditor)** 返回一组“行”配置，每行由 `propertyWidget.addTextRow/addNumberRow/addComboxRow` 等生成，包含 name、value、group、editor 等
- 分组与字段包括：
  - **IDS_BASE**：名称、样式配置按钮、类型、描述、显示边框(是/否)、访问(能否被访问)
  - **IDS_CHART_CONFIG**：布局容器内部间隙(interval)
  - **IDS_POSOTION**：X、Y
  - **IDS_SIZE**：宽度、高度
  - **IDS_SIZEPOLICY**：水平策略、垂直策略（固定/可延展）
- 属性变更通过 MSG_ONEXECPROPERTY 到公共处理逻辑，再调用布局的 setInterval、setWidth、setHeight、setHorizontalSizePolicy、setVerticalSizePolicy 等，最后触发 **resetRegionPosByParent** 或 **resetRegionPos** 重算整棵布局树

### 2.7 与编辑器的联动

- **setCurrentEditor(editor)**：布局及其递归子布局都会保存 currentEditor，用于在 resetRegionPos 后更新控制点位置（开发态）、以及区分 Page_Editor / Composite_GraphicMode_Editor 获取 addComponentObj（layoutObjectMap）
- **layoutObjectMap**：页面编辑器通过 `window.addComponentObjPage.layoutObjectMap` 保存所有布局容器 id 到布局实例的映射，便于从任意子组件通过 parentId 找到父布局并调用 resetRegionPos

---

## 三、页面（KMPage）与页面属性（KMPageProperty）

### 3.1 页面模型

- **KMPage**（`km_page.js`）继承 KMPageBase，`objectType = OBJECTTYPE_PAGE`
  - 持有 layerMng（图层管理）、objectMng（图素管理）、pageProperty（KMPageProperty）
  - 提供 getObjects()、图层与序列化相关方法

### 3.2 页面属性 KMPageProperty（km_page_property.js）

- **基础**：name、comment、pagePos、pageSize、windowSize
- **背景**：brushName、brushArray（solid/linear/radial/material）、picPath、picName、picSrc
- **窗口**：title、titleBar、closeBox、sizeable、fontsizeable、autoUserScrollbar、windowType、borderType、openAlways、hideInOptenDialog
- **缩放与尺寸**：scaleControl、appOrientation、minLockWidth、minLockHeight、lockHeightWidthScale、lockHeightWidthNum、lockAspectRatioType
- **安全与权限**：priority、createtime、security、securitySection、resourceGroup、resourceGroupMsg、resDescriptionMap

所有字段均有 get/set，并参与 objectToJson/jsonToObject，与属性窗的“页面”模式一致。

### 3.3 页面在属性窗中的展示

- 在 **KMUIPageView.prototype.propertyToJson** 中，当选中为“页面”时，用 propertyWidget 的 addTextRow/addNumberRow/addComboxRow 等生成多行，分组包括：基本（名称、类型、描述等）、位置、大小、背景、窗口选项等，与 KMPageProperty 的 get 方法一一对应。

---

## 四、UI 组件（以按钮为例）

### 4.1 按钮模型 KMUIButtonModel（kmui_button_model.js）

- 继承 **KMUIModelBase**，`uiModelType = kmUIModeTypeEnum.BUTTON`
- 内部持有一个 **KMUIButton**（this.button），默认宽高 90x25，样式 position:absolute
- 属性字段：
  - caption（显示文本）、style（标准/扁平）、status（正常/多种状态）
  - fontInfo、textColor、bkColor、pressedBkColor、focusedBkColor、disabledBkColor
  - comment、boundRect、prototypeName = 'Button'

### 4.2 属性与 DOM 的同步

- 文本：setCaption/getCaption 直接操作 this.button.thisElement.innerHTML
- 颜色与样式：setBkColor、setTextColor、setStyle、setStatus 等同时写 this.xxx 与 this.button.thisElement.style

### 4.3 属性窗展示（propertyToJson）

- 分组与字段：
  - **IDS_BASE**：名称、类型、描述、能否被访问
  - **IDS_POSOTION**：X、Y
  - **IDS_SIZE**：宽度、高度
  - **IDS_INTERFACE**：标题文本(caption)、显示样式(扁平/标准)、背景颜色、字体、文本颜色、按下/聚焦/不可操作颜色、按钮状态样式
  - **IDS_OTHER**：锁定、可见、可操作、提示信息
- 行类型：TEXT、NUMBER、COMBOBOX、UICOLORPICKER、CHECKBOXVALUE 等，由 propertyNameSet 与 addTextRow/addNumberRow/addComboxRow 决定编辑器形态

### 4.4 序列化

- 按钮的 objectToJson/jsonToObject 由基类与自身字段共同完成，保证保存/加载后 caption、样式、颜色、位置尺寸等一致。

---

## 五、页面属性设置

- **数据源**：页面选中时，属性窗数据来自 `KMUIPageView.prototype.propertyToJson(currentEditor)`，内部读取 `this.page.pageProperty` 的各类 get 方法
- **展示**：通过 `propertyObj.addObjPropToPropgrid(jsonArray)` 将行数据填入 EasyUI propertygrid，分组展示（基本、位置、大小、背景、窗口等）
- **回写**：用户编辑某行后触发 onAfterEdit，通过 `KMSignalManager.dispatch(..., MSG_ONEXECPROPERTY, rowData, ...)`，由统一逻辑根据 rowData.group/name 找到页面并调用对应 set 方法（如 setPageWidth、setPageHeight、setBackgroundColor 等），必要时刷新画布尺寸或背景

---

## 六、UI 组件属性设置

- **数据源**：选中单个 UI 组件（如按钮）时，属性窗数据来自该组件的 **propertyToJson(currentEditor)**（如 KMUIButtonModel.prototype.propertyToJson）
- **展示**：同样用 addObjPropToPropgrid，行内 editor 可为 numberbox、combobox、colorPicker 等（由 propertyNameSet 与 configComboboxItems 等决定）
- **回写**：MSG_ONEXECPROPERTY 的 rowData 包含 group、name、value，公共逻辑根据当前选中对象调用对应 set 方法（如 setCaption、setBkColor、setWidth 等）
- **特殊**：颜色、图片、画刷、字体等会打开“更多”弹窗（如 KMUIPopImage、KMUIBrush），确认后仍通过 MSG_ONEXECPROPERTY 或直接 set 写回模型

---

## 七、布局属性设置

- **数据源**：选中布局容器（如水平布局）时，属性窗数据来自该布局的 **propertyToJson(currentEditor)**（如 KMHorizontalLayout.prototype.propertyToJson）
- **布局特有字段**：
  - 名称、类型、描述、显示边框、**布局容器内部间隙(interval)**、访问
  - 位置 X/Y、宽度、高度
  - **水平策略、垂直策略**（固定/可延展）
- **样式/详细配置**：部分布局提供“样式配置”“详细配置”按钮，对应 IDS_CHART_STYLE_CONFIG、IDS_CHART_DETAILED_CONFIG，点击后打开脚本编辑器（KMUIUIScriptEditor），内容写入 styleDataNew/configureData，再通过 setStyleFormat/eval 等应用
- **回写**：MSG_ONEXECPROPERTY 根据 name 调用 setInterval、setWidth、setHeight、setHorizontalSizePolicy、setVerticalSizePolicy、setIsShowBorder 等，接着调用 **resetRegionPosByParent()**，从父布局起整树 **resetRegionPos()**，保证嵌套布局与子组件位置尺寸一致

---

## 八、属性窗与通用属性编辑（kmui_property.js / kmui_commonproperty_editor.js）

### 8.1 KMUIProperty

- 基于 EasyUI propertygrid，列：name、value、group、editor、id
- **addObjPropToPropgrid(jsonArray)**：先 removeAllRows，再按 jsonArray 逐行 appendRow，每行包含 name、value、group、editor、id
- **行构造**：addTextRow、addNumberRow、addComboxRow 等，返回 { name, value, group, editor } 形态，editor 可为字符串（如 "text"、"numberbox"）或 JSON 字符串（如 combobox 的 data/options）
- **事件**：onBeforeEdit（某些行禁止编辑或弹出专用编辑器，如详细配置/样式配置）、onClickRow（颜色/图片/画刷等行显示“...”按钮并打开弹窗）、onAfterEdit（提交变更并 MSG_ONEXECPROPERTY）

### 8.2 属性变更统一处理

- 监听 MSG_ONEXECPROPERTY，根据 group、name、value 与当前选中对象类型，调用对象上对应的 set 方法（如 setCaption、setWidth、setHorizontalSizePolicy），并视情况调用 resetRegionPos、resetRegionPosByParent、updateControllers 等，最后再次调用 propertyToJson 刷新属性窗（避免多选或只读展示刷新）

### 8.3 kmui_commonproperty_editor.js

- 用于“变量/公共属性”等通用列表的编辑弹窗，与单组件属性窗（propertygrid）是两套：一个是当前选中对象的属性，一个是变量表等列表的增删改。

---

## 九、新版本 designer 对照与建议

### 9.1 已具备的对应关系

| 老版本 | 新版本 designer |
|--------|------------------|
| 页面 + 画布 | DesignerView + DesignCanvas |
| 水平布局 | HorizontalLayout（manifest + 组件） |
| 按钮 | Button（manifest + 组件） |
| 属性 Tab | RightPanel / PropertyPanel |
| 组件/布局 propertyToJson | manifest.props + PropEditor / 分组展示 |
| 选中 → 属性窗 | 选中态 → panelState → PageInspectorPanel / 单选 / MultiInspectorPanel |

### 9.2 建议补齐或强化的细节

1. **布局容器**
   - 老版水平布局的 **尺寸策略**（水平/垂直 固定 vs 可延展）和 **interval（区块间隙）** 在新版 manifest 中已有 gap、justify、align，可考虑显式增加“尺寸策略”或“固定宽度/高度”的配置项，并与 CSS（flex）或运行时布局逻辑统一。
   - 老版 **panelMap + componentMap** 的“区块 → 子组件”映射、以及按 DOM 顺序的序列化，在新版可用“容器子节点有序列表 + 每个子节点 id”的方式在 Document 层统一，避免只依赖 DOM 顺序。

2. **页面属性**
   - 老版 KMPageProperty 的页面尺寸、背景（画刷/图片）、窗口类型、标题栏、可缩放、安全与权限等，在新版 PageInspectorPanel 中建议按需逐项对照，保证保存/加载与运行态一致。

3. **UI 组件属性**
   - 老版按钮的 caption、多种颜色（背景/按下/聚焦/禁用）、样式（标准/扁平）、状态，与新版的 manifest.props（text、type、disabled 等）可做一一映射；若需兼容老工程，可增加“背景色/文字色”等扩展属性或主题映射。

4. **属性回写与联动**
   - 老版通过 MSG_ONEXECPROPERTY 统一回写并触发 resetRegionPos/updateControllers，新版建议保留“单一入口”的变更处理（如 command 或 store action），在修改布局/页面/组件属性后显式触发：
     - 布局：重算子节点位置与尺寸（或标记脏并统一 layout 计算）
     - 控制器/选择框：根据新位置尺寸更新

5. **样式/详细配置**
   - 老版“样式配置”“详细配置”为脚本或 CSS 片段，若新版需要等价能力，可在 AdvancedPanel 或“配置”中增加脚本/CSS 编辑与安全执行机制。

6. **序列化顺序**
   - 老版水平布局按 **DOM 子节点顺序** 序列化 insiderObjects，反序列化时按该顺序 getRegionDiv + appendChild。新版序列化容器时建议显式输出 children 数组顺序，反序列化按该顺序插入，不依赖 DOM 子节点顺序的隐式约定。

---

## 十、关键文件索引（exe）

| 功能 | 路径 |
|------|------|
| 页面编辑器入口 | `DevelopClient/libs/editor/pageeditor/kmui_page_editor.js` |
| 页面视图/画布 | `DevelopClient/libs/editor/pageeditor/kmui_page_view.js` |
| 工具箱 | `DevelopClient/libs/editor/pageeditor/kmui_pagetoolkit.js` |
| 布局基类 | `Common/Graphic/Layout/kmui_layoutcomponentbase.js` |
| 水平布局 | `Common/Graphic/Layout/kmui_horizontal_layout.js` |
| 停靠布局 | `Common/Graphic/Layout/kmui_layout.js` |
| 按钮模型 | `Common/Graphic/UI/kmui_button_model.js` |
| 页面 | `Common/Graphic/Model/km_page.js` |
| 页面属性 | `Common/Graphic/Model/km_page_property.js` |
| 属性窗 | `DevelopClient/libs/ui/kmui_property.js` |
| 通用属性编辑（变量等） | `DevelopClient/libs/ui/kmui_commonproperty_editor.js` |

以上为老版本页面编辑器中布局容器、页面、UI 组件及三类属性设置的核心实现细节分析，可直接用于新版本功能对齐与边界情况完善。
