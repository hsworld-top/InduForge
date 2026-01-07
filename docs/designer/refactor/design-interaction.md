# 设计态交互（Design Interaction）

本文档定义 Designer 的设计态交互，包括工具栏、属性面板、组件面板、绘图工具等 UI 功能边界。

## 1. 整体布局

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              顶部工具栏 (TopToolbar)                          │
├─────────────┬────────────────────────────────────────────────┬───────────────┤
│             │                                                │               │
│  左侧面板    │                   画布区域                      │   右侧面板    │
│ (LeftPanel) │                  (Canvas)                      │ (RightPanel)  │
│             │                                                │               │
│ ┌─────────┐ │  ┌──────────────────────────────────────────┐  │ ┌───────────┐ │
│ │绘图工具  │ │  │  ┌────────────────────────────────────┐ │  │ │  属性面板  │ │
│ │         │ │  │  │      Canvas 层（绘图图形）         │ │  │ │           │ │
│ │ ─────── │ │  │  │  线/矩形/圆/管道/工艺流程图背景    │ │  │ │ ───────── │ │
│ │组件面板  │ │  │  └────────────────────────────────────┘ │  │ │  样式面板  │ │
│ │         │ │  │  ┌────────────────────────────────────┐ │  │ │           │ │
│ │ ─────── │ │  │  │      DOM 层（交互组件）            │ │  │ │ ───────── │ │
│ │符号库   │ │  │  │  Text/Button/Table/Chart/Input    │ │  │ │  事件面板  │ │
│ │         │ │  │  └────────────────────────────────────┘ │  │ │           │ │
│ │ ─────── │ │  └──────────────────────────────────────────┘  │ │ ───────── │ │
│ │页面/大纲│ │                                                │ │  绑定面板  │ │
│ └─────────┘ │  ┌──────────────────────────────────────────┐  │ └───────────┘ │
│             │  │           底部面板（可选展开）             │  │               │
│             │  │  控制台 / 诊断 / 变量监视 / 快捷操作       │  │               │
│             │  └──────────────────────────────────────────┘  │               │
└─────────────┴────────────────────────────────────────────────┴───────────────┘
```

## 2. 画布分层架构

画布采用 **Canvas + DOM 双层架构**，分离绘图图形和交互组件：

```
┌─────────────────────────────────────────────────────────────────┐
│                        画布区域                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Canvas 层（绘图）z-index: 1                 │   │
│  │                                                         │   │
│  │   - 线段、矩形、圆、多边形、路径                         │   │
│  │   - 管道（带流动动画）                                  │   │
│  │   - 工艺流程图背景                                      │   │
│  │   - 连接线                                              │   │
│  │   - 用「绘图工具」绘制                                  │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              DOM 层（组件）z-index: 10                   │   │
│  │                                                         │   │
│  │   - Text、Button、Input、Select                        │   │
│  │   - Table、List、Card                                  │   │
│  │   - Chart、Gauge                                       │   │
│  │   - 用「组件面板」拖入                                  │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.1 两层对比

| 方面     | Canvas 绘图层             | DOM 组件层               |
| -------- | ------------------------- | ------------------------ |
| 添加方式 | 选择工具后**在画布绘制**  | 从面板**拖入**           |
| 适合内容 | 静态图形、管道、流程背景  | 交互控件、数据展示、表单 |
| 渲染方式 | Canvas 2D / WebGL         | Vue 组件 + CSS           |
| 性能     | 大量图形时性能好          | 复杂交互能力强           |
| 数据绑定 | ✅ 支持（颜色/文本/动画） | ✅ 支持                  |
| 事件     | 基础事件（click/hover）   | 完整事件（表单/键盘等）  |

### 2.2 典型应用场景

```
工艺流程图示例：

    Canvas 层绘制管道和设备轮廓
         ↓
    ┌─────●━━━━━━━━━━━━━━━━━●─────┐
    │                             │
    │    DOM 层放置数值显示组件    │
    │         ┌────────┐          │
    │    ●━━━━│ 25.5℃ │━━━━●     │  ← 温度值绑定数据点
    │         └────────┘          │
    │         ┌────────┐          │
    │    ●━━━━│ [启动] │━━━━●     │  ← 按钮触发动作
    │         └────────┘          │
    │                             │
    └─────●━━━━━━━━━━━━━━━━━●─────┘
          ↑
    管道流动动画绑定流速数据点
```

## 3. 顶部工具栏（TopToolbar）

### 3.1 功能分区

```
┌───────────────────────────────────────────────────────────────────────────────────────────┐
│ [← →] │ [📄▼] │ [🖱️ 🔲 ✋] │ [─ ▭ ○ ◇ 🔧 ✏️] │ [⬜ 📱 📺] │ [🔍 100%] │ [▶️ 预览] [💾 保存] │
│ 导航   │ 页面  │  选择工具   │    绘图工具       │  分辨率    │   缩放    │     操作按钮       │
└───────────────────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 功能定义

| 分区         | 功能                          | 说明                                 |
| ------------ | ----------------------------- | ------------------------------------ |
| **导航**     | 撤销 / 重做                   | Ctrl+Z / Ctrl+Y                      |
| **页面**     | 页面切换 / 新建 / 管理        | 下拉选择当前页面，支持新建/复制/删除 |
| **选择工具** | 选择 / 框选 / 拖拽画布        | 切换选择交互模式                     |
| **绘图工具** | 直线/矩形/圆/多边形/管道/文字 | 切换绘图模式，在画布绘制图形         |
| **分辨率**   | PC / 平板 / 大屏              | 切换画布尺寸，预览不同端适配效果     |
| **缩放**     | 缩小 / 缩放比 / 放大          | 画布缩放，支持滚轮/快捷键            |
| **操作**     | 预览 / 保存 / 更多            | 主要操作入口                         |

### 3.3 选择工具

| 工具 | 图标 | 快捷键 | 说明                             |
| ---- | ---- | ------ | -------------------------------- |
| 选择 | 🖱️   | V      | 点击选中图形/组件，支持移动/缩放 |
| 框选 | 🔲   | M      | 拖拽框选多个元素                 |
| 拖拽 | ✋   | H      | 拖拽画布平移（大画布时有用）     |

### 3.4 绘图工具

| 工具   | 图标 | 快捷键 | 说明                       |
| ------ | ---- | ------ | -------------------------- |
| 直线   | ─    | L      | 绘制直线/折线              |
| 矩形   | ▭    | R      | 绘制矩形/圆角矩形          |
| 圆形   | ○    | O      | 绘制圆形/椭圆              |
| 多边形 | ◇    | P      | 绘制多边形（点击添加顶点） |
| 管道   | 🔧   | T      | 绘制管道（带流动动画）     |
| 文字   | ✏️   | X      | 添加文字标注               |

**绘图操作流程**：

1. 选择绘图工具（如矩形）
2. 在画布上按住拖拽绘制
3. 释放鼠标完成绘制
4. 自动切回选择工具，选中刚绘制的图形
5. 在右侧面板配置属性/绑定/动画

### 3.5 分辨率预设

| 预设   | 尺寸      | 说明         |
| ------ | --------- | ------------ |
| PC     | 1920×1080 | 桌面端       |
| 平板   | 1024×768  | 平板横屏     |
| 大屏   | 3840×2160 | 4K 大屏      |
| 自定义 | W × H     | 手动输入尺寸 |

### 3.6 更多菜单

| 功能       | 快捷键 | 说明                    |
| ---------- | ------ | ----------------------- |
| 导出 JSON  | Ctrl+E | 导出当前页面 Schema     |
| 导入 JSON  | Ctrl+I | 导入页面 Schema         |
| 页面设置   | -      | 打开页面配置对话框      |
| 全局变量   | -      | 打开全局变量管理        |
| 发布管理   | -      | 跳转到 dev_ide 发布界面 |
| 快捷键帮助 | Ctrl+/ | 显示快捷键列表          |

## 4. 左侧面板（LeftPanel）

左侧面板分为多个折叠区域：

```
左侧面板
├─ 🎨 绘图工具          ← 绘图快捷入口
├─ 📦 组件面板          ← 拖入 DOM 组件
├─ 🏭 符号库            ← 预设工业符号
├─ 📄 页面树            ← 工程所有页面
├─ 🌲 大纲树            ← 当前页面结构
└─ 📊 数据点            ← 数据中心数据点
```

### 4.1 绘图工具（快捷入口）

与工具栏绘图工具相同，提供快捷入口：

| 工具     | 说明                         |
| -------- | ---------------------------- |
| 直线     | 绘制直线/折线                |
| 矩形     | 绘制矩形/圆角矩形            |
| 圆形     | 绘制圆形/椭圆                |
| 多边形   | 绘制多边形                   |
| 管道     | 绘制管道（自带流动动画属性） |
| 文字标注 | 在 Canvas 层添加文字         |

### 4.2 组件面板

**功能边界**：

- 展示可用 DOM 组件列表，按分类分组
- 支持搜索过滤
- **拖拽**组件到画布
- 显示组件图标和名称

**分类**：

| 分类     | 组件示例                             |
| -------- | ------------------------------------ |
| 布局     | FlexContainer, FreeContainer, Grid   |
| 基础     | Text, Image, Button, Icon            |
| 表单     | Input, Select, Switch, DatePicker    |
| 数据展示 | Table, List, Card, Progress          |
| 图表     | LineChart, BarChart, PieChart, Gauge |
| 系统     | LocaleSwitcher, ThemeSwitcher        |
| 自定义   | 用户创建的自定义组件                 |

### 4.3 符号库

**功能边界**：

- 展示预设的工业符号（Canvas 图形组合）
- 按分类分组：管道阀门、设备仪表、流程符号
- **拖拽**符号到画布（添加到 Canvas 层）
- 支持用户自定义符号

**分类**：

| 分类     | 符号示例                             |
| -------- | ------------------------------------ |
| 管道阀门 | 直管、弯头、三通、闸阀、球阀、止回阀 |
| 设备     | 泵、电机、储罐、换热器、压缩机       |
| 仪表     | 压力表、温度计、流量计、液位计       |
| 流程     | 箭头、标注、区域框                   |
| 自定义   | 用户创建的符号                       |

**符号与组件的区别**：

| 方面     | 符号（Canvas）             | 组件（DOM）            |
| -------- | -------------------------- | ---------------------- |
| 渲染层   | Canvas 层                  | DOM 层                 |
| 适合场景 | 工艺流程图、管道连线       | 数据展示、表单交互     |
| 添加方式 | 绘图工具绘制 或 符号库拖入 | 组件面板拖入           |
| 交互能力 | 基础（click/hover）        | 完整（表单/键盘/焦点） |
| 性能     | 大量图形时性能好           | 复杂交互能力强         |

### 4.4 页面树

**功能边界**：

- 显示工程所有页面
- 支持新建/复制/删除页面
- 双击切换页面
- 支持拖拽排序
- 显示页面图标（登录页/首页标记）

### 4.5 大纲树

**功能边界**：

- 展示当前页面的**所有元素**层级结构（包括 Canvas 图形和 DOM 组件）
- 点击选中对应元素
- 支持拖拽调整层级
- 显示元素名称和类型图标
  - 🎨 Canvas 图形（线/矩形/圆/管道等）
  - 📦 DOM 组件
- 支持重命名元素
- 显示可见性状态（眼睛图标）
- 显示锁定状态（锁图标）

### 4.6 数据点面板

**功能边界**：

- 显示数据中心的数据点树
- 按连接/分组展示
- 显示数据点状态（🟢 active / 🔴 invalid / 🟡 unknown）
- 支持搜索过滤
- 拖拽数据点到组件属性
- 显示数据点当前值（只读）

## 5. 右侧面板（RightPanel）

右侧面板为**选中元素**（Canvas 图形或 DOM 组件）提供统一的配置界面。

### 5.1 属性面板（PropertyPanel）

**功能边界**：

| 功能         | 说明                                         |
| ------------ | -------------------------------------------- |
| 基础属性编辑 | 根据组件 Manifest 自动生成表单               |
| 属性类型支持 | string, number, boolean, color, enum, object |
| 属性分组     | 按逻辑分组显示（基础、高级、布局等）         |
| 数据绑定入口 | 属性旁显示绑定按钮 🔗，点击切换到绑定面板    |
| 多语言配置   | 文本属性旁显示 🌐 按钮，配置 i18n            |
| 默认值显示   | 未修改的属性显示为浅色                       |
| 重置功能     | 支持重置单个属性到默认值                     |
| 多选编辑     | 多选时显示共同属性，批量修改                 |

**属性类型编辑器**：

| 类型    | 编辑器                       |
| ------- | ---------------------------- |
| string  | Input / Textarea             |
| number  | InputNumber（支持步进/范围） |
| boolean | Switch                       |
| color   | ColorPicker                  |
| enum    | Select / Radio               |
| object  | JSON 编辑器 / 结构化表单     |
| array   | 列表编辑器                   |
| asset   | 资源选择器（图片/视频/图标） |

### 5.2 样式面板（StylePanel）

**功能边界**：

| 分类 | 属性                                   |
| ---- | -------------------------------------- |
| 尺寸 | width, height, minWidth, maxWidth 等   |
| 边距 | margin, padding（支持四边独立设置）    |
| 定位 | position, top, left, right, bottom, z  |
| 背景 | backgroundColor, backgroundImage       |
| 边框 | border, borderRadius                   |
| 文字 | fontSize, fontWeight, color, textAlign |
| 阴影 | boxShadow                              |
| 变换 | transform, opacity                     |
| 其他 | cursor, overflow                       |

**Canvas 图形特有样式**：

| 属性        | 说明                              |
| ----------- | --------------------------------- |
| fill        | 填充颜色/渐变                     |
| stroke      | 描边颜色                          |
| strokeWidth | 描边宽度                          |
| dash        | 虚线样式                          |
| lineJoin    | 线段连接方式（miter/round/bevel） |
| lineCap     | 线段端点样式（butt/round/square） |

**特殊功能**：

- 快捷设置（常用值预设）
- CSS 单位切换（px/em/%/vw/vh）
- 可视化边距调整器
- 颜色历史记录

### 5.3 事件面板（EventPanel）

**功能边界**：

| 功能     | 说明                                   |
| -------- | -------------------------------------- |
| 事件列表 | 显示元素支持的事件（click, change 等） |
| 添加动作 | 点击事件后添加动作                     |
| 动作配置 | 配置动作类型和参数                     |
| 动作排序 | 拖拽调整动作执行顺序                   |
| 动作删除 | 删除已配置的动作                       |
| 动作复制 | 复制动作配置                           |
| 动作权限 | 配置动作执行所需权限                   |

**Canvas 图形支持的事件**：

| 事件       | 说明     |
| ---------- | -------- |
| click      | 点击图形 |
| dblclick   | 双击图形 |
| mouseenter | 鼠标进入 |
| mouseleave | 鼠标离开 |

**动作配置 UI**：

```
┌─ click 事件 ─────────────────────────────────┐
│                                              │
│  [+] 添加动作                                │
│                                              │
│  1. [setVar] 设置变量                        │
│     └─ scope: page, name: isLoading, value: true │
│     [编辑] [删除] [↑] [↓]                    │
│                                              │
│  2. [callApi] 调用接口                       │
│     └─ endpoint: /api/data, method: GET      │
│     [编辑] [删除] [↑] [↓]                    │
│                                              │
└──────────────────────────────────────────────┘
```

### 5.4 绑定面板（BindingPanel）

**功能边界**：

| 功能            | 说明                             |
| --------------- | -------------------------------- |
| 绑定列表        | 显示当前组件已配置的绑定         |
| 添加绑定        | 选择属性，配置绑定               |
| 绑定类型切换    | datapoint / var / expr           |
| 数据点选择器    | 弹出数据点树，选择数据点         |
| 变量选择器      | 选择页面/全局变量                |
| 表达式编辑器    | 输入 {{ }} 表达式，语法高亮/提示 |
| Transform 配置  | 添加/编辑转换操作链              |
| Fallback 配置   | 配置降级值                       |
| DesignMock 配置 | 配置设计态 Mock 值               |
| 状态显示        | 显示数据点状态（失效提示）       |

**绑定配置 UI**：

```
┌─ 绑定配置: text ─────────────────────────────┐
│                                              │
│  绑定类型: [数据点 ▼]                        │
│                                              │
│  数据点:  mqtt.EMQX.温度组.temperature      │
│           🟢 正常 - 最后更新: 10:30:15       │
│           [选择数据点]                       │
│                                              │
│  转换:    [+] 添加转换                       │
│           1. toFixed(1)                      │
│           2. suffix("℃")                    │
│                                              │
│  降级值:  "--"                               │
│                                              │
│  设计值:  "25.5℃"                           │
│                                              │
│  [确定] [取消]                               │
└──────────────────────────────────────────────┘
```

## 6. 画布区域（Canvas）

### 6.1 功能边界

| 功能          | 说明                          |
| ------------- | ----------------------------- |
| **Canvas 层** | 绘制图形、管道、符号          |
| **DOM 层**    | 渲染交互组件                  |
| 拖放接收      | 接收组件面板/符号库拖入的元素 |
| 选中交互      | 点击/框选元素，显示选中框     |
| 移动/缩放     | 拖拽移动，控制点缩放          |
| 对齐辅助线    | 智能对齐线，吸附功能          |
| 右键菜单      | 复制/粘贴/删除/层级调整等     |
| 空容器占位    | 空容器显示虚线占位框          |
| 拖拽高亮      | 拖拽经过容器时高亮显示        |

### 6.2 Canvas 层交互

**绘图操作**：

1. 选择绘图工具（如矩形）
2. 在画布上按住左键开始绘制
3. 拖拽确定大小
4. 释放完成绘制
5. 自动切回选择工具，选中新图形

**图形选中与编辑**：

- 单击选中图形
- 选中后显示控制点（缩放/旋转）
- 双击进入锚点编辑（多边形/路径）
- Shift + 点击多选

**管道绘制**：

```
1. 选择管道工具
2. 点击确定起点
3. 点击确定中间拐点
4. 双击确定终点
5. 属性面板配置流动方向/速度/颜色
```

### 6.3 DOM 层交互

**组件放置**：

1. 从组件面板拖拽组件
2. 拖到画布/容器上方
3. 显示放置预览位置
4. 释放完成放置

**组件选中与编辑**：

- 单击选中组件
- 选中后显示边框和控制点
- 双击可进入文本编辑（Text 组件）
- Shift + 点击多选

### 6.4 右键菜单

| 菜单项   | 快捷键       | 说明                   |
| -------- | ------------ | ---------------------- |
| 复制     | Ctrl+C       | 复制选中元素           |
| 粘贴     | Ctrl+V       | 粘贴元素               |
| 剪切     | Ctrl+X       | 剪切选中元素           |
| 删除     | Delete       | 删除选中元素           |
| 锁定     | Ctrl+L       | 锁定/解锁元素          |
| 隐藏     | Ctrl+H       | 隐藏/显示元素          |
| 置顶     | Ctrl+]       | 移到最上层             |
| 置底     | Ctrl+[       | 移到最下层             |
| 上移一层 | Ctrl+↑       | 上移一层               |
| 下移一层 | Ctrl+↓       | 下移一层               |
| 编组     | Ctrl+G       | 将选中元素编组         |
| 取消编组 | Ctrl+Shift+G | 取消编组               |
| 创建符号 | -            | 将 Canvas 图形存为符号 |
| 创建组件 | -            | 另存为自定义组件       |

### 6.5 空容器占位符

```
┌─ FlexContainer (空) ─────────────────────────┐
│ ┌ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┐ │
│   将组件拖放到此处                            │
│ └ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┘ │
└──────────────────────────────────────────────┘
```

**实现要点**：

- 占位符仅在设计态显示，运行态不渲染
- 占位符使用虚线边框 + 浅灰背景
- 最小高度 48px
- 显示提示文字

### 6.6 拖拽高亮

```
拖拽组件经过容器时：
┌─ FlexContainer ──────────────────────────────┐
│ ╔════════════════════════════════════════════╗│
│ ║  Drop Here                                 ║│
│ ╚════════════════════════════════════════════╝│
│  [已有组件]  [已有组件]                       │
└──────────────────────────────────────────────┘

高亮边框颜色：主题色（如 #1890ff）
背景透明度：10%
```

## 7. 预览功能

### 7.1 实现思路

预览模式**直接调用开发系统的数据中心 API**，无需 Bridge/iframe 通信：

```
┌─────────────────────────────────────────────────────────────────┐
│                     Designer (开发环境)                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  设计态                          预览态                          │
│  ┌───────────────┐              ┌───────────────┐               │
│  │ DesignRenderer │              │RuntimeRenderer│               │
│  │               │              │               │               │
│  │  显示 Mock 值  │    切换模式   │  显示真实数据  │               │
│  │               │  ─────────▶  │               │               │
│  └───────────────┘              └───────┬───────┘               │
│                                         │                       │
│                                         ▼                       │
│                            ┌───────────────────────┐            │
│                            │   DataService (API)   │            │
│                            │                       │            │
│                            │  fetch / WebSocket    │            │
│                            └───────────┬───────────┘            │
│                                        │                        │
└────────────────────────────────────────┼────────────────────────┘
                                         │
                                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     dev_core (后端)                              │
├─────────────────────────────────────────────────────────────────┤
│  /api/v1/datapoints/subscribe      (WebSocket 订阅)             │
│  /api/v1/datapoints/:path/value    (获取单值)                   │
│  /api/v1/queries/:id/execute       (执行查询)                   │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 预览模式切换

```typescript
// stores/designerStore.ts
interface DesignerState {
  mode: "edit" | "preview";
  // ...
}

function togglePreview() {
  if (state.mode === "edit") {
    // 切换到预览
    state.mode = "preview";
    dataService.connect(); // 连接数据中心
    dataService.subscribeAll(); // 订阅所有数据点
  } else {
    // 切换回编辑
    dataService.disconnect();
    state.mode = "edit";
  }
}
```

### 7.3 DataService 实现

```typescript
class DataService {
  private socket: Socket | null = null;
  private baseUrl: string;

  constructor(baseUrl: string = "/api/v1") {
    this.baseUrl = baseUrl;
  }

  // 连接数据中心
  async connect(): Promise<void> {
    this.socket = io("/datapoint", {
      transports: ["websocket"],
    });

    this.socket.on("datapoint:value", (data) => {
      this.cache.set(data.path, data.value);
      this.notifySubscribers(data.path, data.value);
    });
  }

  // 订阅数据点
  subscribe(path: string, callback: (value: any) => void): () => void {
    // 发送订阅请求
    this.socket?.emit("datapoint:subscribe", { path });

    // 注册回调
    const callbacks = this.subscribers.get(path) || new Set();
    callbacks.add(callback);
    this.subscribers.set(path, callbacks);

    // 返回取消订阅函数
    return () => {
      callbacks.delete(callback);
      if (callbacks.size === 0) {
        this.socket?.emit("datapoint:unsubscribe", { path });
        this.subscribers.delete(path);
      }
    };
  }

  // 执行查询
  async executeQuery(queryId: string, params?: object): Promise<any> {
    const response = await fetch(`${this.baseUrl}/queries/${queryId}/execute`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(params),
    });
    return response.json();
  }

  disconnect(): void {
    this.socket?.disconnect();
    this.socket = null;
    this.subscribers.clear();
    this.cache.clear();
  }
}
```

### 7.4 预览 UI

```
┌─────────────────────────────────────────────────────────────────┐
│  [← 返回编辑]    预览模式    [📱 PC ▼]  [🔄 刷新]  [🔗 新窗口]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│                                                                 │
│                    （渲染页面内容）                               │
│                                                                 │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  数据连接: 🟢 已连接    最后更新: 10:30:15    数据点: 15/15 正常 │
└─────────────────────────────────────────────────────────────────┘
```

## 8. 快捷键一览

### 8.1 通用

| 快捷键 | 功能       |
| ------ | ---------- |
| Ctrl+Z | 撤销       |
| Ctrl+Y | 重做       |
| Ctrl+S | 保存       |
| Ctrl+P | 预览       |
| Ctrl+/ | 快捷键帮助 |
| Delete | 删除       |
| Esc    | 取消选中   |

### 8.2 选择

| 快捷键      | 功能          |
| ----------- | ------------- |
| Click       | 选中元素      |
| Ctrl+Click  | 多选/取消选中 |
| Ctrl+A      | 全选          |
| Shift+Click | 范围选中      |

### 8.3 编辑

| 快捷键       | 功能       |
| ------------ | ---------- |
| Ctrl+C       | 复制       |
| Ctrl+V       | 粘贴       |
| Ctrl+X       | 剪切       |
| Ctrl+D       | 复制并粘贴 |
| Ctrl+G       | 编组       |
| Ctrl+Shift+G | 取消编组   |

### 8.4 视图

| 快捷键    | 功能     |
| --------- | -------- |
| Ctrl++    | 放大     |
| Ctrl+-    | 缩小     |
| Ctrl+0    | 重置缩放 |
| Ctrl+滚轮 | 缩放     |

### 8.5 层级

| 快捷键 | 功能     |
| ------ | -------- |
| Ctrl+] | 置顶     |
| Ctrl+[ | 置底     |
| Ctrl+↑ | 上移一层 |
| Ctrl+↓ | 下移一层 |

### 8.6 绘图工具

| 快捷键 | 功能   |
| ------ | ------ |
| V      | 选择   |
| M      | 框选   |
| H      | 拖拽   |
| L      | 直线   |
| R      | 矩形   |
| O      | 圆形   |
| P      | 多边形 |
| T      | 管道   |
| X      | 文字   |

### 8.7 位置微调

| 快捷键         | 功能             |
| -------------- | ---------------- |
| ↑ ↓ ← →        | 移动 1px         |
| Shift + 方向键 | 移动 10px        |
| Ctrl + 方向键  | 移动到下一网格线 |

### 8.8 锁定与隐藏

| 快捷键       | 功能          |
| ------------ | ------------- |
| Ctrl+L       | 切换锁定状态  |
| Ctrl+Shift+L | 解锁所有      |
| Ctrl+H       | 切换显示/隐藏 |

## 9. Canvas 渲染引擎

### 9.1 技术选型

推荐使用 **Konva.js** 作为 Canvas 渲染引擎：

| 方案        | 优点                           | 缺点               | 推荐度     |
| ----------- | ------------------------------ | ------------------ | ---------- |
| **Konva**   | 熟悉度高、事件系统完善、性能好 | 包体积中等         | ⭐⭐⭐⭐⭐ |
| Fabric.js   | 功能丰富、序列化强             | 包体积大、学习成本 | ⭐⭐⭐⭐   |
| 原生 Canvas | 轻量、无依赖                   | 事件处理需自己实现 | ⭐⭐⭐     |
| PixiJS      | WebGL 高性能                   | 偏游戏、API 差异大 | ⭐⭐       |

### 9.2 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                     CanvasLayer.vue                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   Konva.Stage   │  │   Konva.Layer   │  │ Konva.Transformer│ │
│  │   (容器)        │  │  (图形渲染层)    │  │  (变换控制器)    │ │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘ │
│           │                    │                    │           │
│           └────────────────────┴────────────────────┘           │
│                                │                                │
│  ┌─────────────────────────────┴─────────────────────────────┐ │
│  │                    GraphicsRenderer                        │ │
│  │                                                           │ │
│  │  graphicsById → Konva Shapes 映射                         │ │
│  │  - Canvas.Line   → Konva.Line                             │ │
│  │  - Canvas.Rect   → Konva.Rect                             │ │
│  │  - Canvas.Circle → Konva.Circle                           │ │
│  │  - Canvas.Pipe   → PipeShape (自定义)                     │ │
│  │  - Canvas.Symbol → SymbolShape (自定义)                   │ │
│  │                                                           │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 9.3 管道流动动画

```typescript
// PipeShape.ts - 自定义管道形状
class PipeShape extends Konva.Group {
  private flowAnimation: Konva.Animation | null = null;
  private flowOffset = 0;

  constructor(config: PipeConfig) {
    super(config);
    this.createPipeGraphics();
    if (config.flowSpeed > 0) {
      this.startFlowAnimation();
    }
  }

  private createPipeGraphics(): void {
    // 1. 绘制管道主体（矩形/圆角路径）
    // 2. 绘制流动指示（虚线动画）
  }

  startFlowAnimation(): void {
    this.flowAnimation = new Konva.Animation((frame) => {
      // 根据 flowSpeed 和 flowDirection 更新 flowOffset
      this.flowOffset += this.flowSpeed * (frame.timeDiff / 1000);
      this.updateFlowIndicator();
    }, this.getLayer());
    this.flowAnimation.start();
  }

  // 数据绑定时调用
  setFlowSpeed(speed: number): void {
    this.flowSpeed = speed;
    if (speed > 0 && !this.flowAnimation) {
      this.startFlowAnimation();
    } else if (speed === 0 && this.flowAnimation) {
      this.flowAnimation.stop();
      this.flowAnimation = null;
    }
  }
}
```

## 10. 管道锚点连接系统

### 10.1 锚点定义

```typescript
interface Anchor {
  name: string; // 锚点名称，如 'inlet', 'outlet', 'top', 'bottom'
  x: number; // 相对于符号中心的 X 偏移
  y: number; // 相对于符号中心的 Y 偏移
  direction: "up" | "down" | "left" | "right"; // 连接方向
}

// 符号定义中的锚点
interface SymbolDef {
  id: string;
  anchors: Anchor[];
  // ...
}
```

### 10.2 管道连接

```typescript
// 管道可以连接到符号的锚点
interface PipeConnection {
  startAnchor?: {
    symbolId: string; // 连接的符号 ID
    anchorName: string; // 锚点名称
  };
  endAnchor?: {
    symbolId: string;
    anchorName: string;
  };
}

// 管道 props 扩展
interface PipeProps {
  points: [number, number][];
  connections?: PipeConnection; // 锚点连接信息
  // ...其他属性
}
```

### 10.3 连接交互

```
连接管道到设备锚点：

1. 选择管道工具
2. 鼠标靠近设备符号时，显示可用锚点（高亮圆点）
3. 点击锚点开始绘制管道
4. 绘制中间点
5. 靠近目标设备锚点时自动吸附
6. 点击目标锚点完成连接

┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│     ┌───┐                                      ┌───┐            │
│     │泵 │●━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━●│阀门│            │
│     └───┘ (outlet)              管道        (inlet) └───┘       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

移动设备时，连接的管道端点跟随移动
```

### 10.4 连接跟随逻辑

```typescript
// 当符号移动时，更新连接的管道端点
function onSymbolMove(symbolId: string, newX: number, newY: number): void {
  // 找到所有连接到此符号的管道
  const connectedPipes = findPipesConnectedTo(symbolId);

  for (const pipe of connectedPipes) {
    const connections = pipe.props.connections;
    if (!connections) continue;

    const newPoints = [...pipe.props.points];

    // 更新起点（如果连接到此符号）
    if (connections.startAnchor?.symbolId === symbolId) {
      const anchor = getAnchorPosition(
        symbolId,
        connections.startAnchor.anchorName
      );
      newPoints[0] = [anchor.x, anchor.y];
    }

    // 更新终点（如果连接到此符号）
    if (connections.endAnchor?.symbolId === symbolId) {
      const anchor = getAnchorPosition(
        symbolId,
        connections.endAnchor.anchorName
      );
      newPoints[newPoints.length - 1] = [anchor.x, anchor.y];
    }

    // 执行更新命令
    executeCommand(
      new UpdateGraphicCommand(pipe.id, {
        props: { ...pipe.props, points: newPoints },
      })
    );
  }
}
```

## 11. 对齐、分布与吸附

### 11.1 对齐操作

| 操作     | 快捷键 | 说明             |
| -------- | ------ | ---------------- |
| 左对齐   | Alt+L  | 以最左元素为基准 |
| 右对齐   | Alt+R  | 以最右元素为基准 |
| 上对齐   | Alt+T  | 以最上元素为基准 |
| 下对齐   | Alt+B  | 以最下元素为基准 |
| 水平居中 | Alt+H  | 水平中心对齐     |
| 垂直居中 | Alt+V  | 垂直中心对齐     |

### 11.2 分布操作

| 操作     | 说明                   |
| -------- | ---------------------- |
| 水平分布 | 等间距水平排列选中元素 |
| 垂直分布 | 等间距垂直排列选中元素 |

### 11.3 智能吸附

```typescript
interface SnapConfig {
  enabled: boolean;
  gridSize: number; // 网格尺寸，如 8px
  snapToGrid: boolean; // 吸附到网格
  snapToGuides: boolean; // 吸附到参考线
  snapToElements: boolean; // 吸附到其他元素边缘
  snapThreshold: number; // 吸附阈值，如 5px
}

// 吸附类型
type SnapLine = {
  type: "vertical" | "horizontal";
  position: number; // x 或 y 坐标
  sourceId: string; // 吸附来源元素 ID
};
```

### 11.4 吸附时机

- **移动元素时**：显示对齐辅助线，自动吸附到最近的吸附点
- **缩放元素时**：边缘吸附到其他元素或网格
- **绘制图形时**：起点/终点吸附到网格或其他图形

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│     ┌─────────┐                                                │
│     │  元素A  │                                                │
│     └────┬────┘                                                │
│          │                                                      │
│          │ 吸附辅助线                                           │
│          │                                                      │
│     ┌────┴────┐                                                │
│     │  元素B  │  ← 正在移动，左边缘与 A 对齐时显示辅助线        │
│     └─────────┘                                                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 11.5 网格模式

```typescript
interface GridConfig {
  visible: boolean; // 是否显示网格
  size: number; // 网格尺寸
  color: string; // 网格颜色
  opacity: number; // 透明度
  subdivisions: number; // 细分数量（每大格分几小格）
}
```

## 12. 复制粘贴策略

### 12.1 基本规则

| 场景           | 行为                                |
| -------------- | ----------------------------------- |
| 同页面复制粘贴 | 粘贴位置偏移 (10, 10)，避免完全重叠 |
| 跨页面复制粘贴 | 粘贴到目标页面相同位置，无偏移      |
| 多选复制       | 保持相对位置关系                    |
| 混合复制       | 分别处理节点和图形，保持各自层级    |

### 12.2 ID 重新生成

```typescript
// 复制时生成新 ID 的策略
function generateCopyId(originalId: string): string {
  // 格式: 原ID_copy_时间戳后4位
  const timestamp = Date.now().toString().slice(-4);
  return `${originalId}_copy_${timestamp}`;
}

// 批量复制时保持引用关系
function copyElements(elements: SelectableElement[]): CopiedData {
  const idMapping = new Map<string, string>(); // 旧ID -> 新ID

  // 1. 先生成所有新 ID
  for (const el of elements) {
    idMapping.set(el.id, generateCopyId(el.id));
  }

  // 2. 复制数据并更新内部引用
  const copiedNodes = [];
  const copiedGraphics = [];

  for (const el of elements) {
    if (el.kind === "node") {
      const node = deepClone(doc.getNode(el.id));
      node.id = idMapping.get(el.id);
      // 更新 children 中的引用
      node.children = node.children
        .map((childId) => idMapping.get(childId) ?? childId)
        .filter(
          (id) => idMapping.has(id) || !elements.some((e) => e.id === id)
        );
      copiedNodes.push(node);
    } else {
      const graphic = deepClone(doc.getGraphic(el.id));
      graphic.id = idMapping.get(el.id);
      // 更新管道连接引用
      if (graphic.props.connections) {
        updateConnectionRefs(graphic.props.connections, idMapping);
      }
      copiedGraphics.push(graphic);
    }
  }

  return { nodes: copiedNodes, graphics: copiedGraphics, idMapping };
}
```

### 12.3 粘贴位置计算

```typescript
interface PasteConfig {
  offsetX: number; // 同页面偏移 X
  offsetY: number; // 同页面偏移 Y
  maxOffset: number; // 最大累计偏移（连续粘贴时）
}

const defaultPasteConfig: PasteConfig = {
  offsetX: 10,
  offsetY: 10,
  maxOffset: 100, // 超过后重置偏移
};

// 连续粘贴时递增偏移
let pasteCount = 0;
function getPasteOffset(isSamePage: boolean): { x: number; y: number } {
  if (!isSamePage) {
    pasteCount = 0;
    return { x: 0, y: 0 };
  }

  pasteCount++;
  const offset = pasteCount * defaultPasteConfig.offsetX;

  // 超过最大偏移后重置
  if (offset > defaultPasteConfig.maxOffset) {
    pasteCount = 1;
  }

  return {
    x: pasteCount * defaultPasteConfig.offsetX,
    y: pasteCount * defaultPasteConfig.offsetY,
  };
}
```

### 12.4 剪贴板格式

```typescript
interface ClipboardData {
  type: "designer-elements";
  version: 1;
  sourcePageId: string;
  nodes: ComponentNode[];
  graphics: GraphicNode[];
  timestamp: number;
}

// 写入系统剪贴板
async function copyToClipboard(data: ClipboardData): Promise<void> {
  const json = JSON.stringify(data);
  await navigator.clipboard.writeText(json);
}

// 从系统剪贴板读取
async function readFromClipboard(): Promise<ClipboardData | null> {
  const text = await navigator.clipboard.readText();
  try {
    const data = JSON.parse(text);
    if (data.type === "designer-elements") {
      return data;
    }
  } catch {}
  return null;
}
```

## 13. 键盘微调

### 13.1 移动微调

| 快捷键         | 移动距离   | 说明               |
| -------------- | ---------- | ------------------ |
| ↑ ↓ ← →        | 1px        | 精细调整           |
| Shift + 方向键 | 10px       | 快速调整           |
| Ctrl + 方向键  | 吸附到网格 | 移动到下一个网格线 |

### 13.2 实现

```typescript
function handleKeyboardMove(e: KeyboardEvent): void {
  const selected = selectionModel.getSelectedElements();
  if (selected.length === 0) return;

  const directions: Record<string, { x: number; y: number }> = {
    ArrowUp: { x: 0, y: -1 },
    ArrowDown: { x: 0, y: 1 },
    ArrowLeft: { x: -1, y: 0 },
    ArrowRight: { x: 1, y: 0 },
  };

  const dir = directions[e.key];
  if (!dir) return;

  e.preventDefault();

  // 计算移动距离
  let distance = 1;
  if (e.shiftKey) {
    distance = 10;
  } else if (e.ctrlKey) {
    distance = snapConfig.gridSize;
  }

  const deltaX = dir.x * distance;
  const deltaY = dir.y * distance;

  // 批量移动
  history.startBatch();
  for (const el of selected) {
    if (el.kind === "node") {
      // 更新节点 layoutItem
      executeCommand(new MoveNodeCommand(el.id, deltaX, deltaY));
    } else {
      // 更新图形位置
      executeCommand(new MoveGraphicCommand(el.id, deltaX, deltaY));
    }
  }
  history.endBatch();
}
```

### 13.3 缩放微调

| 快捷键               | 效果         |
| -------------------- | ------------ |
| Ctrl + +             | 放大选中元素 |
| Ctrl + -             | 缩小选中元素 |
| Alt + Shift + 方向键 | 单边缩放     |

## 14. Canvas.Text 编辑

### 14.1 交互流程

```
1. 选中 Canvas.Text 图形
2. 双击进入编辑模式
3. 显示文本光标，支持选中/输入
4. 点击外部或按 Esc 退出编辑
5. 按 Enter 确认（单行）或 Shift+Enter 换行（多行）
```

### 14.2 编辑态 UI

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│     双击进入编辑模式：                                           │
│     ┌──────────────────────────────────────────┐                │
│     │ 温度: 25.5℃|                            │ ← 文本光标     │
│     │             ↑                            │                │
│     │         编辑中文本                        │                │
│     └──────────────────────────────────────────┘                │
│     ↓                                                           │
│     [B] [I] [U]  [左] [中] [右]  [字体▼] [大小▼]  [颜色■]        │
│     ↑                                                           │
│     迷你工具栏（选中文本时显示）                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 14.3 属性面板配置

| 属性          | 类型   | 说明                          |
| ------------- | ------ | ----------------------------- |
| text          | string | 文本内容（支持 `{{ }}` 绑定） |
| fontSize      | number | 字体大小                      |
| fontFamily    | enum   | 字体族                        |
| fontWeight    | enum   | 字重（normal/bold）           |
| fontStyle     | enum   | 样式（normal/italic）         |
| fill          | color  | 文字颜色                      |
| align         | enum   | 水平对齐（left/center/right） |
| verticalAlign | enum   | 垂直对齐（top/middle/bottom） |
| lineHeight    | number | 行高                          |
| wrap          | enum   | 换行模式（none/word/char）    |

### 14.4 字体选择

```typescript
// 预设字体列表
const fontFamilies = [
  { label: "系统默认", value: "system-ui" },
  { label: "思源黑体", value: '"Source Han Sans SC", sans-serif' },
  { label: "思源宋体", value: '"Source Han Serif SC", serif' },
  { label: "等宽字体", value: '"JetBrains Mono", monospace' },
  { label: "Arial", value: "Arial, sans-serif" },
  // 工业场景常用
  { label: "数码字体", value: '"DSEG7 Classic", monospace' },
  { label: "LED 字体", value: '"LED Board", monospace' },
];
```

## 15. 图形锁定与隐藏

### 15.1 锁定行为

| 状态 | 可选中 | 可移动 | 可缩放 | 显示大纲树 | 视觉样式     |
| ---- | ------ | ------ | ------ | ---------- | ------------ |
| 正常 | ✅     | ✅     | ✅     | ✅         | 正常         |
| 锁定 | ✅     | ❌     | ❌     | ✅ 🔒 图标 | 选中框虚线   |
| 隐藏 | ❌     | ❌     | ❌     | ✅ 👁 图标  | 设计态不显示 |

### 15.2 锁定样式

```typescript
// 锁定图形的选中框样式
const lockedSelectionStyle = {
  stroke: "#999",
  strokeDashArray: [4, 4], // 虚线
  fill: "transparent",
};

// 锁定图形的鼠标样式
const lockedCursor = "not-allowed";
```

### 15.3 大纲树显示

```
页面大纲
├─ 📦 FlexContainer
│  ├─ 📝 Text "标题"
│  └─ 🖼️ Image
├─ 🎨 Canvas.Rect "背景框"        👁️
├─ 🎨 Canvas.Pipe "主管道"        🔒
└─ 🎨 Canvas.Symbol "泵-001"
```

### 15.4 批量锁定/解锁

```typescript
// 快捷键
const lockShortcuts = {
  "Ctrl+L": "toggleLock", // 切换锁定状态
  "Ctrl+Shift+L": "unlockAll", // 解锁所有
  "Ctrl+H": "toggleVisibility", // 切换显示/隐藏
};

// 批量操作
function toggleLockSelected(): void {
  const selected = selectionModel.getSelectedElements();
  const allLocked = selected.every((el) => getElement(el.id)?.locked);

  history.startBatch();
  for (const el of selected) {
    if (el.kind === "graphic") {
      executeCommand(new UpdateGraphicCommand(el.id, { locked: !allLocked }));
    } else {
      executeCommand(new UpdateNodeCommand(el.id, { locked: !allLocked }));
    }
  }
  history.endBatch();
}
```

## 16. 批量样式编辑

### 16.1 多选时属性面板

```
┌─ 属性面板（已选中 3 个元素）──────────────────────────────────┐
│                                                              │
│  ⚠️ 多个元素选中，仅显示共同属性                              │
│                                                              │
│  ┌─ 共同属性 ─────────────────────────────────────────────┐ │
│  │                                                        │ │
│  │  透明度    [████████░░] 80%                           │ │
│  │                                                        │ │
│  │  可见性    [✅ 显示]                                   │ │
│  │                                                        │ │
│  │  锁定      [❌ 未锁定]                                 │ │
│  │                                                        │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌─ 样式（混合值显示 "--"）────────────────────────────────┐ │
│  │                                                        │ │
│  │  填充颜色  [■ --]  [设置统一值]                        │ │
│  │                                                        │ │
│  │  描边颜色  [■ #333333]  ← 相同值直接显示               │ │
│  │                                                        │ │
│  │  描边宽度  [-- ]px  [设置统一值]                       │ │
│  │                                                        │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### 16.2 混合值处理

```typescript
interface MultiSelectValue<T> {
  type: "same" | "mixed";
  value?: T; // type === 'same' 时有值
  values?: T[]; // type === 'mixed' 时的所有值
}

function getMultiSelectValue<T>(
  elements: (ComponentNode | GraphicNode)[],
  path: string
): MultiSelectValue<T> {
  const values = elements.map((el) => get(el, path));
  const uniqueValues = [...new Set(values.map((v) => JSON.stringify(v)))];

  if (uniqueValues.length === 1) {
    return { type: "same", value: values[0] };
  }
  return { type: "mixed", values };
}

// 设置统一值
function setUnifiedValue(path: string, value: any): void {
  history.startBatch();
  for (const el of selectedElements) {
    if (el.kind === "node") {
      executeCommand(new UpdateNodeCommand(el.id, set({}, path, value)));
    } else {
      executeCommand(new UpdateGraphicCommand(el.id, set({}, path, value)));
    }
  }
  history.endBatch();
}
```

## 17. 自动保存

### 17.1 配置

```typescript
interface AutoSaveConfig {
  enabled: boolean;
  intervalSeconds: number; // 保存间隔，默认 30 秒
  maxVersions: number; // 保留的本地版本数，默认 10
  saveOnBlur: boolean; // 窗口失焦时保存
}

const defaultAutoSaveConfig: AutoSaveConfig = {
  enabled: true,
  intervalSeconds: 30,
  maxVersions: 10,
  saveOnBlur: true,
};
```

### 17.2 保存策略

```typescript
class AutoSaveManager {
  private timer: number | null = null;
  private lastSaveTime = 0;
  private isDirty = false;

  start(): void {
    if (this.timer) return;

    this.timer = setInterval(() => {
      if (this.isDirty) {
        this.save();
      }
    }, this.config.intervalSeconds * 1000);

    // 监听变更
    documentModel.on("change", () => {
      this.isDirty = true;
    });

    // 窗口失焦保存
    if (this.config.saveOnBlur) {
      window.addEventListener("blur", () => {
        if (this.isDirty) {
          this.save();
        }
      });
    }

    // 页面关闭前保存
    window.addEventListener("beforeunload", (e) => {
      if (this.isDirty) {
        this.saveSync();
        e.returnValue = "有未保存的更改，确定离开吗？";
      }
    });
  }

  private async save(): Promise<void> {
    try {
      // 1. 保存到本地 IndexedDB（即时）
      await this.saveToLocal();

      // 2. 保存到服务器（异步）
      await this.saveToServer();

      this.isDirty = false;
      this.lastSaveTime = Date.now();

      // 显示保存成功提示
      toast.success("自动保存成功", { duration: 1000 });
    } catch (error) {
      console.error("自动保存失败:", error);
      toast.error("自动保存失败，请手动保存");
    }
  }

  private async saveToLocal(): Promise<void> {
    const data = documentModel.serialize();
    const version = {
      id: generateId("local_"),
      timestamp: Date.now(),
      data,
    };

    // 保存到 IndexedDB
    await localDB.put("autoSaveVersions", version);

    // 清理旧版本
    await this.cleanupOldVersions();
  }
}
```

### 17.3 本地版本恢复

```
┌─ 恢复本地版本 ───────────────────────────────────────────────┐
│                                                              │
│  检测到本地有未保存的更改，是否恢复？                          │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  📄 自动保存 - 2026-01-07 14:30:25                     │ │
│  │     比服务器版本新 5 分钟                               │ │
│  │     [预览] [恢复此版本]                                 │ │
│  ├────────────────────────────────────────────────────────┤ │
│  │  📄 自动保存 - 2026-01-07 14:25:10                     │ │
│  │     [预览] [恢复此版本]                                 │ │
│  ├────────────────────────────────────────────────────────┤ │
│  │  📄 自动保存 - 2026-01-07 14:20:05                     │ │
│  │     [预览] [恢复此版本]                                 │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  [使用服务器版本]                    [恢复最新本地版本]       │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### 17.4 保存状态指示

```
工具栏右侧显示保存状态：

[✓ 已保存]              - 无未保存更改
[● 未保存]              - 有更改未保存（圆点闪烁）
[↻ 保存中...]           - 正在保存
[⚠ 保存失败 - 重试]     - 保存失败，点击重试
```

## 18. 测试要点

- [ ] 工具栏各功能正常工作
- [ ] 绘图工具绘制图形正常
- [ ] 符号库拖入符号正常
- [ ] 组件面板拖入组件正常
- [ ] Canvas 图形和 DOM 组件分层渲染正确
- [ ] 属性面板根据选中元素类型正确渲染
- [ ] 样式面板修改实时生效（Canvas 和 DOM 均支持）
- [ ] 事件面板动作配置保存正确
- [ ] 绑定面板数据点选择正确
- [ ] 画布拖放功能正常
- [ ] 空容器占位符显示
- [ ] 预览模式数据连接正常
- [ ] 快捷键响应正确
- [ ] **管道锚点连接正常**
- [ ] **移动设备时管道跟随**
- [ ] **管道流动动画正常**
- [ ] **智能吸附功能正常**
- [ ] **对齐分布操作正确**
- [ ] **混合选择（节点 + 图形）正常**
- [ ] **复制粘贴（同页面/跨页面）正确**
- [ ] **键盘微调移动正常**
- [ ] **Canvas.Text 双击编辑正常**
- [ ] **图形锁定/隐藏行为正确**
- [ ] **多选批量样式编辑正常**
- [ ] **自动保存触发和恢复正常**

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [编辑器内核](./editor-core.md)
- [组件清单](./component-manifest.md)
- [动作系统](./action-system.md)
- [数据绑定 v2](./data-binding-v2.md)
