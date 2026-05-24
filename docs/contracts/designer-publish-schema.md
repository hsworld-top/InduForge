# Designer 发布态 Schema 契约

## 1. 文档定位

- 本文档定义 `designer` 输出给 `dev_core` 与 `runtime_engine` 的发布态最小 Schema。
- 本文档只约束发布态，不约束编辑器内部存储结构。
- 本文档是运行时闭环的前置契约，未冻结前不应并行扩 Runtime 解析细节。

## 2. 当前已实现

### 2.1 已有基础

- 设计器已经具备页面、组件树、属性、事件、绑定与表达式基础能力。
- 页面与工程数据已经可以被保存和聚合。

### 2.2 当前缺口

- 编辑态模型和发布态模型尚未正式拆开。
- 组件节点、绑定对象、入口页字段缺少统一发布协议。

## 3. 契约目标

- 发布态只保留 Runtime 必需字段。
- 字段命名稳定，避免发布后仍需要二次转换。
- 明确“设计器可存更多字段，但导出必须裁剪到统一结构”。

## 4. 顶层文件结构

```text
project/
  project.json
  pages/
    <pageId>.json
```

## 5. `project.json` 契约

### 最小结构

```json
{
  "schemaVersion": "1.1",
  "projectId": "proj_xxx",
  "projectCode": "factory_dashboard",
  "projectName": "产线看板",
  "entryPageId": "page_home",
  "theme": "default",
  "locale": "zh-CN",
  "globalVars": [
    {
      "key": "siteName",
      "valueType": "string",
      "defaultValue": "A厂区"
    }
  ],
  "pages": [
    {
      "id": "page_home",
      "name": "首页",
      "routePath": "/home"
    }
  ]
}
```

### 字段说明

| 字段            | 必填 | 说明               |
| --------------- | ---- | ------------------ |
| `schemaVersion` | 是   | 发布态 Schema 版本 |
| `projectId`     | 是   | 工程主键           |
| `projectCode`   | 是   | 工程编码           |
| `projectName`   | 是   | 工程名称           |
| `entryPageId`   | 是   | 入口页 ID          |
| `theme`         | 否   | 默认主题           |
| `locale`        | 否   | 默认语言           |
| `globalVars`    | 否   | 全局变量声明       |
| `pages`         | 是   | 页面索引           |

## 6. `page.json` 契约

### 最小结构

```json
{
  "id": "page_home",
  "name": "首页",
  "routePath": "/home",
  "config": {
    "viewport": {
      "preset": "pc",
      "width": 1920,
      "height": 1080,
      "autoFit": true,
      "lockAspectRatio": false,
      "minWidth": 0,
      "minHeight": 0,
      "overflowMode": "auto"
    },
    "background": {
      "kind": "color",
      "value": "#ffffff",
      "size": "cover",
      "position": "center",
      "repeat": "no-repeat"
    },
    "runtime": {
      "openMode": "replace",
      "popup": {
        "width": 1280,
        "height": 720,
        "center": true,
        "maskClosable": true
      },
      "permission": {
        "summary": ""
      },
      "cacheMode": "default",
      "preloadMode": "lazy"
    }
  },
  "componentTree": [
    {
      "id": "node_title",
      "type": "text",
      "props": {
        "text": "产线总览"
      },
      "style": {
        "x": 80,
        "y": 40
      },
      "bindings": {},
      "events": [],
      "children": []
    }
  ]
}
```

### 字段说明

| 字段            | 必填 | 说明                                                      |
| --------------- | ---- | --------------------------------------------------------- |
| `id`            | 是   | 页面 ID                                                   |
| `name`          | 是   | 页面名称                                                  |
| `routePath`     | 是   | 运行时路由路径                                            |
| `config`        | 是   | 页面发布配置，主结构为 `viewport/background/runtime` 分组 |
| `componentTree` | 是   | 根组件列表                                                |

### `config` 分组说明

`config` 以分组字段作为发布态主结构，预览和运行时应优先读取分组字段，再兼容旧平铺字段。

| 分组         | 说明                                                                                                 |
| ------------ | ---------------------------------------------------------------------------------------------------- |
| `meta`       | 页面运行标题与描述。`meta.title` 作为运行态显示标题，不等同于编辑器内部页面名称。                    |
| `viewport`   | 页面尺寸、缩放与自适应策略。预览宿主以此计算画布外框与可视尺寸。                                     |
| `background` | 页面背景色、背景图与重复/位置/尺寸。预览宿主优先使用分组背景。                                       |
| `runtime`    | 页面运行时策略，如打开方式、弹窗参数、权限摘要、缓存与预加载模式。预览宿主保留该组，运行时按需消费。 |

### `meta.title` 语义

- `meta.title` 表示页面的运行标题，优先级高于页面名称。
- 当页面以主页面激活时，运行时可将其同步到浏览器标题。
- 当页面以覆盖式、弹窗式或容器内部标签形式展示时，运行时优先将其用于容器标题，而不强制要求同步浏览器标题。

### 旧字段兼容

发布态在迁移期间仍可携带下列旧平铺字段，读取顺序为“分组优先，旧字段回退”：

- `width`
- `height`
- `autoFit`
- `backgroundColor`
- `backgroundImage`
- `backgroundSize`
- `backgroundPosition`
- `backgroundRepeat`

导出实现应优先写入 `config.viewport`、`config.background`、`config.runtime`，旧字段仅用于兼容旧消费链与渐进迁移。

## 7. 组件节点契约

### 最小字段

| 字段       | 必填 | 说明       |
| ---------- | ---- | ---------- |
| `id`       | 是   | 节点 ID    |
| `type`     | 是   | 组件类型   |
| `props`    | 否   | 组件属性   |
| `style`    | 否   | 样式与定位 |
| `bindings` | 否   | 绑定对象   |
| `events`   | 否   | 事件列表   |
| `children` | 否   | 子节点     |

### 当前阶段组件白名单

- `text`
- `image`
- `button`
- `table`
- `container`

## 8. 绑定对象契约

### 最小结构

```json
{
  "value": {
    "sourceType": "datapoint",
    "sourceKey": "mqtt.workshop.temperature",
    "expression": null,
    "fallbackValue": "--"
  }
}
```

### 字段说明

| 字段            | 必填 | 说明                                     |
| --------------- | ---- | ---------------------------------------- |
| `sourceType`    | 是   | `static`/`datapoint`/`query`/`globalVar` |
| `sourceKey`     | 否   | 引用的数据主键                           |
| `expression`    | 否   | 表达式                                   |
| `fallbackValue` | 否   | 兜底值                                   |

## 9. 校验规则

- `entryPageId` 必须存在于页面索引中。
- `routePath` 在同一工程内不得重复。
- 组件 `type` 必须在 Runtime 白名单中，或有明确降级策略。
- `bindings.sourceType=datapoint/query/globalVar` 时，`sourceKey` 不可为空。

## 10. 非目标

- 不定义编辑器内部状态字段。
- 不定义复杂动作系统结构。
- 不定义多视图和复杂权限结构。

## 11. 关联文档

- [产品定义](../产品定义.md)
- [高层设计](../高层设计.md)
- [详细设计](../详细设计.md)
- [设计器概览](../designer/README.md)
- [测试与质量策略](../测试与质量策略.md)
