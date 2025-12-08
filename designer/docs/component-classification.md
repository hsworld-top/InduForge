# Designer 组件分类说明

## 组件分类体系

### 1. 基础组件 (Basic Components)
**定位**：基础交互控件，用于用户交互和内容展示

**包含组件**：
- ✅ **Image** - 图片（已实现）
- 🔄 **Button** - 按钮（待实现）
- 🔄 **Video** - 视频（待实现）
- 🔄 **Switch** - 开关（待实现）
- 🔄 **CheckboxGroup** - 多选组（待实现）
- 🔄 **Select** - 下拉框（待实现）
- 🔄 **RadioGroup** - 单选组（待实现）

**说明**：
- ❌ **不包括**矩形、圆形、线条等图形组件
- 这些图形应该在**绘图工具**或**图形编辑器**中实现
- 基础组件专注于**功能性交互**，而非纯视觉图形

### 2. 布局组件 (Layout Components)
**定位**：容器组件，用于组织和排列其他组件

**包含组件**：
- ✅ **Container** - 通用容器（支持 Flex/Grid/Block 三种模式）
- ✅ **Flex** - Flexbox 布局容器
- ✅ **Grid** - CSS Grid 布局容器
- ✅ **Row/Col** - 24 栅格系统
- ✅ **CenterLayout** - 居中布局容器

**特点**：
- 支持任意嵌套
- 完整的 CSS 布局能力
- 可视化配置属性

### 3. UI 组件 (UI Components)
**定位**：复杂的用户界面组件

**包含组件**：
- ✅ **Input** - 输入框
- ✅ **Table** - 数据表格
- ✅ **Form** - 表单
- ✅ **Dialog** - 对话框
- ✅ **Tabs** - 标签页
- ✅ **Card** - 卡片
- ✅ **Tag** - 标签
- ✅ **Badge** - 徽章
- ✅ **Alert** - 提示
- ✅ **Progress** - 进度条

**特点**：
- 基于 Element Plus 组件库
- 提供丰富的配置选项
- 完整的事件支持

### 4. 图表组件 (Chart Components)
**定位**：数据可视化组件

**包含组件**：
- ✅ **LineChart** - 折线图
- ✅ **BarChart** - 柱状图
- ✅ **PieChart** - 饼图
- ✅ **ScatterChart** - 散点图
- ✅ **RadarChart** - 雷达图

**特点**：
- 基于 ECharts
- 支持动态数据绑定
- 丰富的配置选项

### 5. 自定义组件 (Custom Components)
**定位**：用户自定义封装的业务组件

**特点**：
- 支持组件封装
- 支持组件复用
- 支持组件发布到组件市场

## 图形 vs 组件的区别

### 图形（Graphics）
- **用途**：纯视觉元素，装饰性
- **示例**：矩形、圆形、线条、多边形
- **特点**：
  - 主要用于绘制和设计
  - 通常无交互功能
  - 更多用于图形设计工具
  - 应该在**绘图工具模块**实现

### 组件（Components）
- **用途**：功能性元素，可交互
- **示例**：按钮、输入框、开关、下拉框
- **特点**：
  - 具有交互功能
  - 可绑定数据和事件
  - 用于构建应用界面
  - 在**组件库模块**实现

## 迁移说明

### 已移除的组件
以下组件已从基础组件中移除：
- ❌ Rectangle - 矩形
- ❌ Circle - 圆形
- ❌ Ellipse - 椭圆
- ❌ Line - 线条
- ❌ Polygon - 多边形

### 建议的实现方式
如果需要绘制图形，建议：

1. **独立的绘图工具模块**
   ```
   designer/
   ├── drawing-tools/       # 新建绘图工具模块
   │   ├── shapes/
   │   │   ├── Rectangle.js
   │   │   ├── Circle.js
   │   │   └── ...
   │   └── DrawingCanvas.vue
   ```

2. **使用 SVG 编辑器**
   - 集成专业的 SVG 编辑器
   - 提供更强大的图形绘制能力

3. **使用背景装饰**
   - 作为容器的背景样式
   - 通过 CSS 实现简单图形

## 待实现的基础组件

### 优先级 P0（高优先级）
- [ ] Button - 按钮
- [ ] Switch - 开关
- [ ] Select - 下拉框

### 优先级 P1（中优先级）
- [ ] RadioGroup - 单选组
- [ ] CheckboxGroup - 多选组
- [ ] Video - 视频

### 实现建议
每个组件应包含：
1. 组件定义文件（`ComponentName.js`）
2. Vue 组件文件（`ComponentName.vue`）
3. 属性编辑器（`ComponentNameEditor.vue`）
4. 测试文件（`ComponentName.test.js`）

## 参考

### Element Plus 组件映射
基础组件可以直接映射 Element Plus：

| Designer 组件 | Element Plus 组件 | 说明 |
|--------------|------------------|------|
| Button | el-button | 按钮 |
| Switch | el-switch | 开关 |
| Select | el-select | 下拉框 |
| RadioGroup | el-radio-group | 单选组 |
| CheckboxGroup | el-checkbox-group | 多选组 |
| Image | el-image | 图片 |
| Video | HTML5 video | 视频播放器 |

### 组件定义模板
```javascript
export default {
  type: 'Button',
  category: 'basic',
  label: '按钮',
  icon: 'button',
  defaultProps: {
    text: '按钮',
    type: 'primary',
    size: 'default',
  },
  defaultStyle: {
    width: 100,
    height: 40,
  },
  propsSchema: {
    text: { type: 'string', label: '文本' },
    type: { type: 'select', label: '类型', options: [...] },
    size: { type: 'select', label: '尺寸', options: [...] },
  },
  // ...
};
```

