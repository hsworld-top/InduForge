# IFP Manifest 契约

## 1. 文档定位

本文档定义工程不可变 Release 的 Manifest。该 Manifest 是 `dev_core`、NodeAgent、Release
Loader、Project Gateway、`runtime-api` 和 `runtime-engine` 的共同输入。普通用户只看到工程版本、
物理节点和部署状态，不接触底层编排实现。

## 2. 标准 Release 结构

```text
releases/<projectId>/<releaseId>/
├─ release-manifest.json
├─ client-assets.tar.zst
├─ runtime-artifact.tar.zst
├─ collector-artifact.tar.zst
├─ sbom.cdx.json
├─ resource-recommendation.json
├─ health-contract.json
├─ schema-plan.json
├─ checksums.json
└─ signature.sig
```

- `client-assets.tar.zst` 是工程 Vite 的 `dist` 静态产物。
- `runtime-artifact.tar.zst` 包含 HT 2D/3D 场景与资源、运行安全快照及环境无关的运行模型。
- `collector-artifact.tar.zst` 包含当前工程的逻辑采集模型，不包含 Collector 程序、驱动二进制、
  真实设备凭据或目标节点分配。

## 3. `release-manifest.json`

```json
{
  "schemaVersion": "2.0",
  "projectId": "11111111-1111-4111-8111-111111111111",
  "projectCode": "factory_dashboard",
  "releaseId": "22222222-2222-4222-8222-222222222222",
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
  "compatibility": {
    "minNodeAgentVersion": "1.0.0",
    "minRuntimeVersion": "1.0.0",
    "requiredNodeCapabilities": ["project_entry", "data_runtime"]
  },
  "supplyChain": {
    "sbomRef": "sbom.cdx.json",
    "sourceRevision": "git:<commit>",
    "builderId": "induforge-release-builder-v1",
    "signingKeyId": "induforge-release-2026-01",
    "promotable": true
  },
  "preflight": {
    "resourceRecommendationRef": "resource-recommendation.json",
    "healthContractRef": "health-contract.json",
    "schemaPlanRef": "schema-plan.json"
  },
  "buildTime": "2026-08-13T10:00:00Z"
}
```

## 4. 字段约束

| 字段                  | 必填 | 说明                                       |
| --------------------- | ---- | ------------------------------------------ |
| `schemaVersion`       | 是   | Release Manifest 协议版本                  |
| `projectId`           | 是   | 工程 ID                                    |
| `projectCode`         | 是   | 工程编码                                   |
| `releaseId`           | 是   | 不可变 Release ID                          |
| `version`             | 是   | 用户可识别版本号                           |
| `artifacts.client`    | 是   | 工程前端静态资源包                         |
| `artifacts.runtime`   | 是   | 工程运行配置和 HT 场景资源包               |
| `artifacts.collector` | 否   | 工程包含采集任务时的配置包                 |
| `capabilities`        | 否   | Release 需要的运行能力声明                 |
| `compatibility`       | 是   | NodeAgent、Runtime 与物理节点能力兼容要求  |
| `supplyChain`         | 是   | SBOM、源码版本、构建者与是否可晋级         |
| `preflight`           | 是   | 资源、健康契约和数据库结构计划的摘要或引用 |
| `buildTime`           | 是   | 构建完成时间，UTC RFC 3339                 |

每个 Artifact 必须声明包内相对路径和 SHA-256 摘要。`checksums.json` 使用
`release-checksums.v1`，按路径升序列出每个受签文件的 `path`、`sha256` 和 `size`；它覆盖
Manifest、全部 Artifact、SBOM 和三项预检文件，但不得包含自身或 `signature.sig`。
`signature.sig` 是对 `checksums.json` 原始字节的 64 字节 Ed25519 签名。

`compatibility.minNodeAgentVersion` 和 `minRuntimeVersion` 必须是**不带 `v` 前缀**的
SemVer 2.0.0 `MAJOR.MINOR.PATCH`，可带 prerelease 与 build metadata，例如
`1.2.3-rc.1+build.7`。build metadata 合法但不参与版本先后比较；NodeAgent、ReleaseBuilder
和 JSON Schema 对空白、补零、缺失 PATCH、额外段、`v` 前缀及非法 prerelease 一律拒绝。

对象存储可以把上述目录保存为一个外层 `tar.zst` Release bundle；外层摘要由版本记录和
DeploymentBinding 固定。NodeAgent 必须先校验 bundle 摘要，再安全解包并独立复验清单与签名。

## 5. Release 与 DeploymentBinding

同一 Release 必须能够在 development、staging 和 production 之间晋级，环境差异由独立、
版本化的 `DeploymentBinding` 提供。以下内容不得写入 Release：

- 目标物理节点、端口、域名和访问证书。
- 真实设备、数据库、消息服务地址和账号密码。
- Secret 明文或中心长期凭据。
- 副本数、资源请求/限制、存储级别和保留策略。
- 模拟开关、调试授权、维护窗口和生产审批。

Release 可声明逻辑资源别名、所需能力、资源建议和兼容性要求；DeploymentBinding 将这些逻辑
资源绑定到目标物理节点的端口、Secret 引用和本地运行参数。晋级生产时创建新的 Binding revision 和
DeploymentRun，不重新
构建或修改 Release。

`promotable=true` 的 Release 构建时即使用统一可信签名；仅供 development 的临时 Release 可以
使用开发信任根，但必须为 `promotable=false`。预发验收和生产审批生成独立 Promotion
Attestation，不修改 Manifest、Artifact、摘要或 Release 签名。

## 6. 校验规则

- `projectId`、`releaseId`、目标物理节点和 DeploymentBinding 必须一致。
- Artifact 文件必须存在，摘要和签名必须验证通过。
- 不支持的 `schemaVersion` 或能力声明必须阻止部署。
- 目标物理节点的 NodeAgent、Runtime 或能力不满足 compatibility 时必须阻止部署。
- 缺少 SBOM/来源、健康契约、资源建议或数据库结构计划时，按目标 deploymentStage 的策略阻断
  或要求显式风险接受；production 默认阻断。
- `client` 解压后必须包含 `index.html`，且只能包含静态发布文件。
- `runtime` 中引用的场景和资源必须完整，前端声明的场景 ID 必须可解析。
- 下载、校验、展开或探活失败时不得切换当前 Release。

机器契约位于：

- `contracts/runtime/release-manifest-v2.schema.json`
- `contracts/runtime/release-checksums-v1.schema.json`
- `contracts/runtime/deployment-binding-v1.schema.json`

## 7. 非目标

- Manifest 不描述 Vue 路由、页面清单或组件结构。
- Release 不包含开发态上下文、Pi 会话、code-server 数据或依赖缓存。
- 本契约不定义差分包；升级和回滚均以完整不可变 Release 为单位。

## 8. 关联文档

- [工程前端源码与构建产物契约](./工程前端源码与构建产物契约.md)
- [工程发布与资源分发架构](../../02-系统设计/工程发布与资源分发架构.md)
- [工程运行系统架构](../../02-系统设计/工程运行系统架构.md)
- [运维与节点运行态总体架构](../../02-系统设计/运维与节点运行态总体架构.md)
