# 多端适配（Multi-View）

Multi-View 模型支持同一路由在不同终端（PC、大屏、平板、手机）下显示不同视图。

## 1. 概述

### 1.1 设计目标

| 场景         | 解决方案                            |
| ------------ | ----------------------------------- |
| 同一页面多端 | 同路由不同 target 有不同 Schema     |
| 响应式布局   | 单页面内使用 Constraints + 媒体查询 |
| 终端特定功能 | 不同视图可有不同组件/绑定           |

### 1.2 概念模型

```
逻辑页面 (logicalId)
├── PC 视图 (target: pc) [default]
├── 大屏视图 (target: bigscreen)
├── 平板视图 (target: tablet)
├── 手机竖向视图 (target: phoneLandscape)
└── 手机竖屏视图 (target: phonePortrait)
```

同一 `logicalId` 的多个视图共享路由路径，运行时根据终端类型选择渲染。

## 2. 数据模型

### 2.1 数据库设计

`design_pages` 表扩展字段：

| 字段            | 类型        | 说明                            |
| --------------- | ----------- | ------------------------------- |
| logicalId       | char(36)    | 逻辑页面 ID（同路由多视图共享） |
| target          | varchar(20) | 目标端：pc/bigscreen/tablet/phoneLandscape/phonePortrait |
| isDefaultTarget | tinyint(1)  | 是否默认视图（fallback）        |

**唯一约束**：`(projectId, path, target)` 三元组唯一。

### 2.2 Schema 结构

```json
{
  "pagesById": {
    "page_home_pc": {
      "id": "page_home_pc",
      "logicalId": "logic_home",
      "path": "/home",
      "target": "pc",
      "isDefaultTarget": true,
      "schemaContent": { ... }
    },
    "page_home_bigscreen": {
      "id": "page_home_bigscreen",
      "logicalId": "logic_home",
      "path": "/home",
      "target": "bigscreen",
      "isDefaultTarget": false,
      "schemaContent": { ... }
    }
  }
}
```

### 2.3 页面元信息

```typescript
interface PageMeta {
  id: string;
  logicalId: string;
  name: string;
  path: string;
  target: TargetType;
  isDefaultTarget: boolean;
}

type TargetType =
  | "pc"
  | "bigscreen"
  | "tablet"
  | "phoneLandscape"
  | "phonePortrait";
```

## 3. 运行时选路

### 3.1 终端识别

```typescript
function detectTarget(): TargetType {
  // 1. 优先从 ConnectionProfile 获取（部署时配置）
  if (connectionProfile.uiTarget) {
    return connectionProfile.uiTarget;
  }

  // 2. 从 URL 参数获取
  const urlTarget = new URLSearchParams(location.search).get("target");
  if (urlTarget && isValidTarget(urlTarget)) {
    return urlTarget as TargetType;
  }

  // 3. 根据屏幕尺寸自动判断
  const width = window.innerWidth;
  if (width >= 1200) return "bigscreen";
  if (width <= 480) return "phonePortrait";
  if (width <= 768) return "phoneLandscape";
  if (width <= 992) return "tablet";
  return "pc";
}
```

### 3.2 页面选择算法

```typescript
function resolvePageView(
  path: string,
  currentTarget: TargetType
): PageMeta | null {
  const pages = getAllPages();

  // 1. 优先匹配当前 target
  let page = pages.find((p) => p.path === path && p.target === currentTarget);
  if (page) return page;

  // 2. fallback 到默认视图
  page = pages.find((p) => p.path === path && p.isDefaultTarget);
  if (page) return page;

  // 3. fallback 到 PC 视图
  page = pages.find((p) => p.path === path && p.target === "pc");
  if (page) return page;

  // 4. 找不到
  return null;
}
```

### 3.3 路由配置

```typescript
// router.ts
const router = createRouter({
  routes: [
    {
      path: "/:pathMatch(.*)*",
      component: PageRenderer,
      beforeEnter: (to, from, next) => {
        const target = detectTarget();
        const page = resolvePageView(to.path, target);

        if (page) {
          to.meta.pageId = page.id;
          to.meta.target = page.target;
          next();
        } else {
          next("/404");
        }
      },
    },
  ],
});
```

## 4. Designer 支持

### 4.1 视图管理面板

```
┌─────────────────────────────────────────────────────────────────┐
│  首页 - 视图管理                                    [+ 新建视图] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  路由: /home                                                    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 视图         │ 目标端    │ 默认   │ 状态     │ 操作      │   │
│  ├──────────────┼───────────┼────────┼──────────┼───────────┤   │
│  │ 首页-PC      │ 💻 PC     │ ✅     │ 已发布   │ [编辑]    │   │
│  │ 首页-大屏    │ 🖥️ 大屏   │        │ 草稿     │ [编辑]    │   │
│  │ 首页-平板    │ 📟 平板   │        │ 草稿     │ [编辑]    │   │
│  │ 首页-手机竖向 │ 📱 手机竖向 │        │ 未创建   │ [创建]    │   │
│  │ 首页-手机竖屏 │ 📱 手机竖屏 │        │ 未创建   │ [创建]    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 视图切换

工具栏提供 target 切换：

```
┌─────────────────────────────────────────────────────────────────┐
│  [💻 PC] [🖥️ 大屏] [📟 平板] [📱 手机竖向] [📱 手机竖屏]    │    [预览] [发布]              │
└─────────────────────────────────────────────────────────────────┘
```

切换后：

- 加载对应 target 的 Schema
- 调整画布尺寸
- 更新分辨率预设

### 4.3 创建新视图

**方式一：空白创建**

```typescript
async function createView(logicalId: string, target: TargetType) {
  const basePage = getPageByLogicalId(logicalId);

  const newPage = {
    id: generateId(),
    logicalId,
    name: `${basePage.name}-${targetLabels[target]}`,
    path: basePage.path,
    target,
    isDefaultTarget: false,
    schemaContent: createEmptySchema(),
  };

  await savePage(newPage);
}
```

**方式二：从现有视图克隆**

```typescript
async function cloneView(sourcePageId: string, targetType: TargetType) {
  const sourcePage = getPage(sourcePageId);

  const newPage = {
    ...sourcePage,
    id: generateId(),
    target: targetType,
    isDefaultTarget: false,
    name: `${sourcePage.name}-${targetLabels[targetType]}`,
    // 深拷贝 Schema
    schemaContent: JSON.parse(JSON.stringify(sourcePage.schemaContent)),
  };

  await savePage(newPage);
}
```

### 4.4 预览切换

```typescript
// PreviewToolbar.vue
<template>
  <div class="preview-toolbar">
    <el-radio-group v-model="previewTarget" @change="handleTargetChange">
      <el-radio-button value="pc">
        <Icon icon="mdi:monitor" /> PC
      </el-radio-button>
      <el-radio-button value="bigscreen">
        <Icon icon="mdi:television" /> 大屏
      </el-radio-button>
      <el-radio-button value="tablet">
        <Icon icon="mdi:tablet" /> 平板
      </el-radio-button>
      <el-radio-button value="phoneLandscape">
        <Icon icon="mdi:cellphone" /> 手机竖向
      </el-radio-button>
      <el-radio-button value="phonePortrait">
        <Icon icon="mdi:cellphone" /> 手机竖屏
      </el-radio-button>
    </el-radio-group>

    <!-- 分辨率选择 -->
    <el-select v-model="resolution" class="ml-4">
      <el-option
        v-for="res in resolutionPresets[previewTarget]"
        :key="res.value"
        :label="res.label"
        :value="res.value"
      />
    </el-select>
  </div>
</template>
```

## 5. 分辨率预设

### 5.1 预设配置

```typescript
const resolutionPresets = {
  bigscreen: [{ label: "1920×1080", value: "1920x1080" }],
  pc: [{ label: "1366×768", value: "1366x768" }],
  tablet: [{ label: "992×744", value: "992x744" }],
  phoneLandscape: [{ label: "768×1024", value: "768x1024" }],
  phonePortrait: [{ label: "480×800", value: "480x800" }],
};
```

### 5.2 画布适配模式

```typescript
interface FitMode {
  mode: "contain" | "cover" | "stretch" | "none";
  alignment?: "center" | "top" | "bottom" | "left" | "right";
}

// pageConfig
{
  "pageConfig": {
    "width": 1920,
    "height": 1080,
    "fitMode": {
      "mode": "contain",
      "alignment": "center"
    }
  }
}
```

## 6. 跨视图共享

### 6.1 共享元素

不同视图可以共享：

| 共享内容     | 方式                    |
| ------------ | ----------------------- |
| 数据绑定路径 | 使用相同的数据点路径    |
| 全局变量     | 全局变量跨页面/视图共享 |
| 自定义组件   | 工程级别共享            |
| 样式主题     | 全局主题配置            |

### 6.2 视图差异

不同视图通常差异：

| 差异内容 | 说明                     |
| -------- | ------------------------ |
| 布局结构 | 大屏横向、平板/手机纵向   |
| 组件选择 | 小屏端使用简化组件        |
| 显示密度 | 大屏信息密度高           |
| 交互方式 | 平板/手机触摸优化         |

## 7. 发布与部署

### 7.1 IFP 包结构

所有视图打包到同一个 IFP：

```
project.ifp
├── manifest.json
├── schema/
│   ├── project.json      # 包含所有视图
│   └── ...
└── assets/
```

### 7.2 部署配置

```json
{
  "deploymentConfig": {
    "nodeId": "node_factory_01",
    "uiTarget": "bigscreen",
    "defaultLocale": "zh-CN",
    "defaultTheme": "dark"
  }
}
```

### 7.3 运行时确定

```typescript
// RuntimeEngine 启动时
const target = connectionProfile.uiTarget || detectTarget();
console.log(`Running in ${target} mode`);
```

## 8. 最佳实践

### 8.1 设计建议

1. **先设计 PC 版**：PC 版作为默认视图，功能最完整
2. **克隆后调整**：大屏/平板/手机视图从 PC 克隆后调整布局
3. **保持数据一致**：不同视图使用相同数据点
4. **适度差异化**：不必完全一致，根据场景裁剪

### 8.2 大屏适配要点

- 使用 FreeLayout + Constraints 实现响应式
- 字体和图表尺寸适配大屏
- 考虑多屏拼接场景
- 轮播/自动刷新

### 8.3 平板/手机适配要点

- 纵向布局为主
- 减少同屏信息量
- 触摸友好的控件尺寸
- 考虑手势操作

---

**相关文档**：

- [布局系统](./layout-system.md)
- [Schema 设计](./schema-design.md)
- [发布流水线](./publish-pipeline.md)
