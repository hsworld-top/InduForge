# IFP Manifest 契约

## 1. 文档定位

本文档定义工程不可变 Release 的 Manifest。该 Manifest 是 `dev_core`、NodeAgent、
Release Loader、`project-nginx`、`runtime-api` 和运行数据引擎的共同输入。

## 2. 标准 Release 结构

```text
releases/<projectId>/<releaseId>/
├─ release-manifest.json
├─ client-assets.tar.zst
├─ runtime-artifact.tar.zst
├─ collector-artifact.tar.zst
├─ checksums.json
└─ signature.sig
```

- `client-assets.tar.zst` 是工程 Vite 的 `dist` 静态产物。
- `runtime-artifact.tar.zst` 包含 HT 2D/3D 场景与资源、运行安全快照及运行配置。
- `collector-artifact.tar.zst` 包含当前工程的采集配置，不包含 Collector 程序和驱动二进制。

## 3. `release-manifest.json`

```json
{
  "schemaVersion": "2.0",
  "projectId": "proj_xxx",
  "projectCode": "factory_dashboard",
  "releaseId": "release_xxx",
  "version": "2026.08.13-001",
  "artifacts": {
    "client": {
      "file": "client-assets.tar.zst",
      "checksum": "sha256:client"
    },
    "runtime": {
      "file": "runtime-artifact.tar.zst",
      "checksum": "sha256:runtime"
    },
    "collector": {
      "file": "collector-artifact.tar.zst",
      "checksum": "sha256:collector"
    }
  },
  "capabilities": ["runtime.auth", "runtime.datapoint", "runtime.scene"],
  "buildTime": "2026-08-13T10:00:00Z"
}
```

## 4. 字段约束

| 字段                  | 必填 | 说明                         |
| --------------------- | ---- | ---------------------------- |
| `schemaVersion`       | 是   | Release Manifest 协议版本    |
| `projectId`           | 是   | 工程 ID                      |
| `projectCode`         | 是   | 工程编码                     |
| `releaseId`           | 是   | 不可变 Release ID            |
| `version`             | 是   | 用户可识别版本号             |
| `artifacts.client`    | 是   | 工程前端静态资源包           |
| `artifacts.runtime`   | 是   | 工程运行配置和 HT 场景资源包 |
| `artifacts.collector` | 否   | 工程包含采集任务时的配置包   |
| `capabilities`        | 否   | Release 需要的运行能力声明   |
| `buildTime`           | 是   | 构建完成时间，UTC RFC 3339   |

每个 Artifact 必须声明包内相对路径和 SHA-256 摘要。`checksums.json` 覆盖 Manifest 与全部
Artifact，`signature.sig` 对摘要清单签名。

## 5. 校验规则

- `projectId`、`releaseId` 和目标部署计划必须一致。
- Artifact 文件必须存在，摘要和签名必须验证通过。
- 不支持的 `schemaVersion` 或能力声明必须阻止部署。
- `client` 解压后必须包含 `index.html`，且只能包含静态发布文件。
- `runtime` 中引用的场景和资源必须完整，前端声明的场景 ID 必须可解析。
- 下载、校验、展开或探活失败时不得切换当前 Release。

## 6. 非目标

- Manifest 不描述 Vue 路由、页面清单或组件结构。
- Release 不包含开发态上下文、Pi 会话、code-server 数据或依赖缓存。
- 本契约不定义差分包；升级和回滚均以完整不可变 Release 为单位。

## 7. 关联文档

- [工程前端源码与构建产物契约](./工程前端源码与构建产物契约.md)
- [工程发布与资源分发架构](../../02-系统设计/工程发布与资源分发架构.md)
- [工程运行系统架构](../../02-系统设计/工程运行系统架构.md)
