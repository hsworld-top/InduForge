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

## 3. 共创后建议目标
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
| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `schemaVersion` | 是 | 发布态 Schema 版本 |
| `projectId` | 是 | 工程主键 |
| `projectCode` | 是 | 工程编码 |
| `projectName` | 是 | 工程名称 |
| `entryPageId` | 是 | 入口页 ID |
| `theme` | 否 | 默认主题 |
| `locale` | 否 | 默认语言 |
| `globalVars` | 否 | 全局变量声明 |
| `pages` | 是 | 页面索引 |

## 6. `page.json` 契约
### 最小结构
```json
{
  "id": "page_home",
  "name": "首页",
  "routePath": "/home",
  "layout": {
    "type": "absolute",
    "width": 1920,
    "height": 1080
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
| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `id` | 是 | 页面 ID |
| `name` | 是 | 页面名称 |
| `routePath` | 是 | 运行时路由路径 |
| `layout` | 是 | 页面布局信息 |
| `componentTree` | 是 | 根组件列表 |

## 7. 组件节点契约
### 最小字段
| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `id` | 是 | 节点 ID |
| `type` | 是 | 组件类型 |
| `props` | 否 | 组件属性 |
| `style` | 否 | 样式与定位 |
| `bindings` | 否 | 绑定对象 |
| `events` | 否 | 事件列表 |
| `children` | 否 | 子节点 |

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
| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `sourceType` | 是 | `static`/`datapoint`/`query`/`globalVar` |
| `sourceKey` | 否 | 引用的数据主键 |
| `expression` | 否 | 表达式 |
| `fallbackValue` | 否 | 兜底值 |

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
- [运行时闭环专项计划](../运行时闭环专项计划.md)
- [详细设计](../详细设计.md)
- [designer.task](../ai-packages/tasks/designer.task.md)

## 12. 待确认事项
- `page.json` 是否需要保留 `meta` 扩展字段。
- 首版是否允许白名单外组件导出但在运行时降级。
