# NodeAgent 与运行系统协议

## 1. 文档定位

本文档定义中心控制面、Site NodeAgent、K3s 工程运行栈和原生采集进程之间的启动、发布、回滚与状态协议。

## 2. 正式运行基线

- Linux 服务站点以一套 K3s 承载多个工程。
- 每个工程对应一个 K3s Namespace 和一个 NATS Account。
- Windows/Linux 工业采集均使用原生进程，不进入 K3s。
- Site NodeAgent 运行在 K3s 外部。
- Docker Compose 不作为正式生产运行基线；如保留，仅用于本地开发或最小验证。

## 3. 核心对象

### 3.1 SiteDesiredState

```json
{
  "siteId": "site-a",
  "revision": 12,
  "infrastructureProfile": "single-node",
  "k3sVersion": "<managed-version>",
  "natsProfile": "single-node",
  "projects": ["project-a", "project-b"]
}
```

### 3.2 ProjectReleaseDesiredState

```json
{
  "siteId": "site-a",
  "projectId": "project-a",
  "releaseId": "2026.07.13-001",
  "releaseManifestUrl": "<short-lived-url>",
  "namespace": "if-project-a",
  "strategy": "rolling",
  "rollbackReleaseId": "2026.07.01-001"
}
```

### 3.3 NativeCollectorDesiredState

```json
{
  "nodeId": "collector-win-01",
  "projectId": "project-a",
  "releaseId": "2026.07.13-001",
  "collectorVersion": "1.0.0",
  "platform": "windows-x64",
  "configManifestUrl": "<short-lived-url>",
  "processPlanVersion": 1
}
```

## 4. Site NodeAgent 职责

- 与中心建立设备身份和状态通道。
- 维护 K3s 与站点基础设施健康。
- 将工程期望状态转换为 Kubernetes 资源。
- 应用 Namespace、ServiceAccount、Secret、ConfigMap、PVC、Job、Deployment、Service 和 Ingress。
- 托管 Windows/Linux 原生 Collector 子进程。
- 聚合工程和采集健康状态。
- 执行升级、回滚和故障恢复。

NodeAgent 不解释页面 Schema、数据点、报警规则、计算脚本和工业地址。

## 5. K3s 工程发布流程

1. NodeAgent 接收 `ProjectReleaseDesiredState`。
2. 校验站点、工程、Release 和协议版本。
3. 创建或更新工程 Namespace 和基础资源。
4. 创建工程 Release Loader Job。
5. Loader 下载、校验并展开资源到工程 PVC。
6. Loader 成功后应用新版本运行工作负载。
7. 等待 Deployment、Service、Ingress 和业务健康检查通过。
8. 切换工程当前 Release。
9. 回传 `ReleaseApplyResult` 和 `ObservedState`。
10. 失败时保留旧版本运行，并记录失败阶段。

## 6. 原生采集发布流程

1. NodeAgent 接收 `NativeCollectorDesiredState`。
2. 检查目标平台、Collector 版本和驱动能力。
3. 必要时安装或升级 Collector 能力包。
4. 下载工程采集配置并校验签名和 SHA-256。
5. 生成 `native-processes.json`。
6. 预创建工程日志、WAL、密钥和版本目录。
7. 启动新版本 Collector 并执行本地健康检查。
8. 健康后切换当前版本；失败时恢复旧进程。
9. 回传进程状态、配置版本和驱动能力。

## 7. native-processes.json

```json
{
  "nodeId": "collector-win-01",
  "processes": [
    {
      "id": "collector-project-a",
      "projectId": "project-a",
      "executable": "industrial-collector",
      "arguments": ["--mode=runtime", "--project=project-a", "--release=2026.07.13-001"],
      "workingDirectory": "<project-release-dir>",
      "restartPolicy": "always",
      "healthCheck": {
        "type": "http",
        "address": "http://127.0.0.1:19101/health"
      }
    }
  ]
}
```

OPC DA Worker 作为工程级可选进程加入同一计划。

## 8. 运行目录

### 8.1 Linux Site NodeAgent

```text
/opt/induforge/node-agent/
/var/lib/induforge/node-agent/
/var/log/induforge/node-agent/
```

### 8.2 原生采集节点

```text
projects/<projectId>/
├─ releases/<releaseId>/collector/
├─ current-state.json
├─ logs/
├─ wal/
└─ secrets/
```

## 9. NATS 和凭据

- 站点共享一套 NATS JetStream。
- 每个工程使用独立 Account。
- Collector 只获取目标工程最小权限凭据。
- NodeAgent 可以部署凭据文件，但不使用该凭据处理工程业务消息。

## 10. 健康状态

```json
{
  "siteId": "site-a",
  "observedRevision": 12,
  "k3s": { "status": "healthy" },
  "jetstream": { "status": "healthy" },
  "projects": [
    {
      "projectId": "project-a",
      "releaseId": "2026.07.13-001",
      "status": "running"
    }
  ],
  "nativeProcesses": [
    {
      "id": "collector-project-a",
      "status": "running"
    }
  ]
}
```

详细字段遵循 [Runtime 健康检查与状态协议](./runtime-health-status-contract.md)。

## 11. 回滚

- K3s 工程回滚到指定 Release ID。
- 原生采集回滚到指定配置 Release 和兼容的 Collector 版本。
- 不允许在当前 Release 目录中原地修改配置实现回滚。
- 回滚完成后必须重新执行健康检查并回传结果。

## 12. 安全约束

- Site NodeAgent 的 K3s 凭据不下发给中心其他服务。
- Release 下载使用短期授权。
- 工程 Secret 只进入目标 Namespace。
- 原生采集密钥只写入节点安全目录。
- 本地诊断端口默认绑定 `127.0.0.1`。

## 13. 关联文档

- [节点管理与交付架构](../../02-系统设计/节点管理与交付架构.md)
- [工程运行系统架构](../../02-系统设计/工程运行系统架构.md)
- [工业采集架构](../../02-系统设计/工业采集架构.md)
