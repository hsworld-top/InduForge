# 布局系统

本文档描述 Designer 的布局系统，包括 Flex、Free、Grid 三种布局模式，以及 FreeLayout Constraints（约束布局）。

## 1. 布局模式概览

| 布局模式      | 适用场景           | 特点               |
| ------------- | ------------------ | ------------------ |
| FlexContainer | 流式布局、响应式   | 自动排列、弹性伸缩 |
| FreeContainer | 自由定位、大屏设计 | 绝对坐标或约束定位 |
| GridContainer | 栅格布局、仪表盘   | 行列定位、跨格     |

## 2. FlexContainer

### 2.1 Props

```typescript
interface FlexContainerProps {
  direction: "row" | "column";
  wrap: "nowrap" | "wrap" | "wrap-reverse";
  justify:
    | "flex-start"
    | "flex-end"
    | "center"
    | "space-between"
    | "space-around"
    | "space-evenly";
  align: "flex-start" | "flex-end" | "center" | "stretch" | "baseline";
  gap: number | [number, number]; // [rowGap, columnGap]
}
```

### 2.2 子元素 LayoutItem

```json
{
  "layoutItem": {
    "flex": {
      "grow": 1,
      "shrink": 0,
      "basis": "auto",
      "alignSelf": "stretch"
    }
  }
}
```

## 3. FreeContainer

### 3.1 Props

```typescript
interface FreeContainerProps {
  width: number | "auto";
  height: number | "auto";
  overflow: "visible" | "hidden" | "scroll" | "auto";
}
```

### 3.2 子元素 LayoutItem

#### 模式 A：绝对定位（abs）

```json
{
  "layoutItem": {
    "free": {
      "mode": "abs",
      "abs": {
        "x": 100,
        "y": 50,
        "w": 200,
        "h": 80,
        "z": 1
      }
    }
  }
}
```

#### 模式 B：约束定位（constraints）

```json
{
  "layoutItem": {
    "free": {
      "mode": "constraints",
      "constraints": {
        "top": 12,
        "right": 12,
        "width": 160,
        "height": 48,
        "keepAspect": true
      },
      "z": 10
    }
  }
}
```

### 3.3 Constraints 计算规则

父容器尺寸：`Pw × Ph`

| 约束组合        | 计算公式                             |
| --------------- | ------------------------------------ |
| right + width   | `x = Pw - right - width`             |
| left + width    | `x = left`                           |
| left + right    | `width = Pw - left - right`（拉伸）  |
| top + height    | `y = top`                            |
| bottom + height | `y = Ph - bottom - height`           |
| top + bottom    | `height = Ph - top - bottom`（拉伸） |

**实现代码**：

```typescript
function calculateConstraints(
  parentWidth: number,
  parentHeight: number,
  constraints: Constraints
): { x: number; y: number; width: number; height: number } {
  let x = 0,
    y = 0,
    width = 0,
    height = 0;

  // 水平方向
  if (constraints.left !== undefined && constraints.right !== undefined) {
    x = constraints.left;
    width = parentWidth - constraints.left - constraints.right;
  } else if (
    constraints.left !== undefined &&
    constraints.width !== undefined
  ) {
    x = constraints.left;
    width = constraints.width;
  } else if (
    constraints.right !== undefined &&
    constraints.width !== undefined
  ) {
    width = constraints.width;
    x = parentWidth - constraints.right - width;
  }

  // 垂直方向
  if (constraints.top !== undefined && constraints.bottom !== undefined) {
    y = constraints.top;
    height = parentHeight - constraints.top - constraints.bottom;
  } else if (
    constraints.top !== undefined &&
    constraints.height !== undefined
  ) {
    y = constraints.top;
    height = constraints.height;
  } else if (
    constraints.bottom !== undefined &&
    constraints.height !== undefined
  ) {
    height = constraints.height;
    y = parentHeight - constraints.bottom - height;
  }

  // 保持宽高比
  if (constraints.keepAspect && constraints.width && constraints.height) {
    const ratio = constraints.width / constraints.height;
    if (width / height > ratio) {
      width = height * ratio;
    } else {
      height = width / ratio;
    }
  }

  return { x, y, width, height };
}
```

## 4. GridContainer

### 4.1 Props

```typescript
interface GridContainerProps {
  columns: number | string; // 列数或模板 "1fr 2fr 1fr"
  rows: number | string; // 行数或模板
  gap: number | [number, number];
  autoRows: string; // 自动行高
  autoCols: string; // 自动列宽
}
```

### 4.2 子元素 LayoutItem

```json
{
  "layoutItem": {
    "grid": {
      "row": 1,
      "col": 2,
      "rowSpan": 2,
      "colSpan": 1
    }
  }
}
```

## 5. 大屏适配（fitMode）

### 5.1 PageConfig

```json
{
  "config": {
    "width": 1920,
    "height": 1080,
    "fitMode": "contain",
    "safeArea": {
      "top": 0,
      "right": 0,
      "bottom": 0,
      "left": 0
    }
  }
}
```

### 5.2 fitMode 选项

| 模式    | 行为                                 |
| ------- | ------------------------------------ |
| contain | 等比缩放，完整显示，可能有 letterbox |
| cover   | 等比缩放填满，可能裁切               |
| stretch | 拉伸填满，可能变形                   |
| none    | 不缩放，按设计尺寸显示               |

### 5.3 实现

```typescript
function calculateFit(
  designWidth: number,
  designHeight: number,
  viewportWidth: number,
  viewportHeight: number,
  fitMode: FitMode
): { scale: number; offsetX: number; offsetY: number } {
  const designRatio = designWidth / designHeight;
  const viewportRatio = viewportWidth / viewportHeight;

  let scale = 1,
    offsetX = 0,
    offsetY = 0;

  switch (fitMode) {
    case "contain":
      if (viewportRatio > designRatio) {
        // 视口更宽，以高度为准
        scale = viewportHeight / designHeight;
        offsetX = (viewportWidth - designWidth * scale) / 2;
      } else {
        // 视口更高，以宽度为准
        scale = viewportWidth / designWidth;
        offsetY = (viewportHeight - designHeight * scale) / 2;
      }
      break;

    case "cover":
      if (viewportRatio > designRatio) {
        scale = viewportWidth / designWidth;
        offsetY = (viewportHeight - designHeight * scale) / 2;
      } else {
        scale = viewportHeight / designHeight;
        offsetX = (viewportWidth - designWidth * scale) / 2;
      }
      break;

    case "stretch":
      // 不保持比例
      return {
        scaleX: viewportWidth / designWidth,
        scaleY: viewportHeight / designHeight,
        offsetX: 0,
        offsetY: 0,
      };

    case "none":
      offsetX = (viewportWidth - designWidth) / 2;
      offsetY = (viewportHeight - designHeight) / 2;
      break;
  }

  return { scale, offsetX, offsetY };
}
```

### 5.4 Constraints + fitMode 的关系

**关键问题**：当使用 `fitMode: contain` 时，可能出现 letterbox（黑边）。此时"右上角"是设计画布的右上角，还是屏幕的右上角？

**解决方案**：

1. **相对设计尺寸**（默认）：constraints 相对于设计画布尺寸计算，整体缩放
2. **相对视口**（可选）：添加 `relativeTo: 'viewport'` 选项

```json
{
  "layoutItem": {
    "free": {
      "mode": "constraints",
      "constraints": {
        "top": 12,
        "right": 12,
        "width": 160,
        "height": 48
      },
      "relativeTo": "viewport",
      "z": 10
    }
  }
}
```

## 6. Designer UI

### 6.1 约束面板

```
┌─ 约束配置 ──────────────────────────────────┐
│                                              │
│     ┌───────────────────────────┐           │
│     │    [ ] Top: [  12  ] px   │           │
│     │                           │           │
│ [ ] │                           │ [ ]       │
│ Left│                           │ Right     │
│ [16]│                           │ [12]      │
│     │                           │           │
│     │    [ ] Bottom: [    ] px  │           │
│     └───────────────────────────┘           │
│                                              │
│  宽度: [●] Fixed [  160  ] px               │
│        [ ] Stretch                           │
│                                              │
│  高度: [●] Fixed [   48  ] px               │
│        [ ] Stretch                           │
│                                              │
│  [ ] 保持宽高比                              │
│                                              │
└──────────────────────────────────────────────┘
```

### 6.2 预览分辨率切换

```
┌─────────────────────────────────────────────┐
│  预览分辨率:                                 │
│  [1920×1080 ▼]  [横向/纵向 ⟳]  [适应窗口]   │
│                                              │
│  常用分辨率:                                 │
│  • 1920×1080 (PC)                           │
│  • 3840×2160 (4K)                           │
│  • 1280×720 (HD)                            │
│  • 自定义...                                 │
└─────────────────────────────────────────────┘
```

## 7. 实现步骤

### Step 1：实现 LayoutItem 类型

```typescript
// renderer/layout/types.ts
export type LayoutItem = ...
```

### Step 2：实现 Constraints 计算

```typescript
// renderer/layout/constraintsCalculator.ts
export function calculateConstraints(...): Rect
```

### Step 3：实现 fitMode 计算

```typescript
// renderer/layout/fitModeCalculator.ts
export function calculateFit(...): Transform
```

### Step 4：实现容器组件

```typescript
// registry/layout/FlexContainer.vue
// registry/layout/FreeContainer.vue
// registry/layout/GridContainer.vue
```

### Step 5：实现 Designer 约束面板

```typescript
// ui/RightPanel/ConstraintsEditor.vue
```

### Step 6：实现分辨率预览

```typescript
// ui/TopToolbar/ResolutionSwitcher.vue
```

## 8. 测试要点

- [ ] FlexContainer 各属性组合
- [ ] FreeContainer abs 模式定位
- [ ] FreeContainer constraints 各约束组合
- [ ] constraints 边界情况（缺少约束、冲突约束）
- [ ] fitMode 各模式计算正确性
- [ ] constraints + fitMode 组合
- [ ] 分辨率切换预览

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [设计态交互](./design-interaction.md)
