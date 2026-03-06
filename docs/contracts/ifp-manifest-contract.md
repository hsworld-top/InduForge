# IFP Manifest 契约

## 1. 文档定位
- 本文档定义 `dev_core` 发布出的 `.ifp` 制品结构和 `manifest.json` 契约。
- 本文档是 `dev_core`、`runtime/node_agent`、`runtime_engine` 的共同输入。

## 2. 当前已实现
### 2.1 已有基础
- 平台已能生成版本包并进入部署流程。

### 2.2 当前缺口
- 包结构、资源索引、能力声明和摘要字段尚未冻结。

## 3. 标准制品结构
```text
project.ifp
  manifest.json
  project/project.json
  project/pages/*.json
  project/assets/**
  project/data/points.json
  project/data/queries.json
  project/data/connections.json
```

## 4. `manifest.json` 最小结构
```json
{
  "schemaVersion": "1.1",
  "projectId": "proj_xxx",
  "projectCode": "factory_dashboard",
  "version": "2026.03.06-001",
  "entryPageId": "page_home",
  "pages": [
    {
      "id": "page_home",
      "routePath": "/home",
      "file": "project/pages/page_home.json"
    }
  ],
  "assets": [
    {
      "path": "project/assets/logo.png",
      "checksum": "sha256:xxxx"
    }
  ],
  "capabilities": ["render.basic", "data.datapoint", "data.query"],
  "buildTime": "2026-03-06T10:00:00Z",
  "checksum": "sha256:package_xxx"
}
```

## 5. 字段说明
| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `schemaVersion` | 是 | 制品协议版本 |
| `projectId` | 是 | 工程 ID |
| `projectCode` | 是 | 工程编码 |
| `version` | 是 | 发布版本号 |
| `entryPageId` | 是 | 入口页 |
| `pages` | 是 | 页面清单 |
| `assets` | 否 | 资源清单 |
| `capabilities` | 否 | 运行时能力声明 |
| `buildTime` | 是 | 打包时间 |
| `checksum` | 是 | 包摘要 |

## 6. 子对象约束
### `pages[]`
| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `id` | 是 | 页面 ID |
| `routePath` | 是 | 路由路径 |
| `file` | 是 | 页面文件路径 |

### `assets[]`
| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `path` | 是 | 资源路径 |
| `checksum` | 否 | 单文件摘要 |

## 7. 校验规则
- `entryPageId` 必须存在于 `pages` 中。
- `pages.file` 必须指向制品内真实文件。
- `checksum` 必须覆盖整个包。
- `schemaVersion` 不兼容时，NodeAgent 或 Runtime 必须拒绝启动。

## 8. 当前阶段非目标
- 不做差分包。
- 不做复杂多包依赖。
- 不强制引入数字签名。

## 9. 关联文档
- [发布流水线](../designer/refactor/publish-pipeline.md)
- [运行时闭环专项计划](../运行时闭环专项计划.md)
- [dev_core.task](../../dev_core.task.md)

## 10. 待确认事项
- `version` 是否统一使用时间戳序列风格。
- `capabilities` 是否需要拆成硬依赖与软依赖。
