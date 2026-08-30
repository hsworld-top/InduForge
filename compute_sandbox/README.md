# Compute Sandbox

`compute_sandbox` 是 JavaScript/Python 计算脚本的隔离执行服务。开发调试与正式运行使用同一镜像，
但必须通过 `COMPUTE_SANDBOX_RUNTIME_PROFILE` 显式选择运行边界。

## 配置

| 环境变量 | 说明 |
| --- | --- |
| `COMPUTE_SANDBOX_RUNTIME_PROFILE` | `development`（默认）或 `runtime`。`runtime` 启用严格运行态限制。 |
| `COMPUTE_SANDBOX_SITE_ID` | `runtime` 必填的站点稳定标识。 |
| `COMPUTE_SANDBOX_NODE_ID` | 可选节点标识，用于 `/api/v1/status`。 |
| `COMPUTE_SANDBOX_DEPLOYMENT_ID` | `runtime` 必填的发布实例稳定标识。 |
| `COMPUTE_SANDBOX_PROJECT_ID` | `runtime` 必填的工程 UUID，必须与发布身份一致。 |
| `COMPUTE_SANDBOX_ARTIFACT_DIGEST` | `runtime` 必填的小写 `sha256:<64 位十六进制摘要>`。 |
| `COMPUTE_SANDBOX_ARTIFACT_ROOT` / `COMPUTE_SANDBOX_ARTIFACT_FILE` | runtime 只读 Project Artifact 根目录及相对文件名。 |
| `COMPUTE_SANDBOX_EXECUTION_FORM` | 可解析 `k3s-workload`、`native-linux` 或 `native-windows`；`runtime` 仅接受前两项，未设置时默认 `k3s-workload`。 |
| `COMPUTE_SANDBOX_MAX_CONCURRENT_EXECUTIONS` | 有界并发数，范围 `1..128`，默认 `4`；超过上限的执行立即返回 `429`。 |
| `COMPUTE_SANDBOX_MAX_PIDS` | runtime 接受的当前 cgroup `pids.max` 上限，范围 `1..4096`，默认 `64`。 |

正式运行必须设置 `COMPUTE_SANDBOX_RUNTIME_PROFILE=runtime` 与
`COMPUTE_SANDBOX_SITE_ID`、`COMPUTE_SANDBOX_DEPLOYMENT_ID`、`COMPUTE_SANDBOX_PROJECT_ID` 和
`COMPUTE_SANDBOX_ARTIFACT_DIGEST`。实际执行仍只支持 Linux 的 `bubblewrap` + `prlimit` 隔离环境，不能以
`native-windows` 作为 runtime 的 executionForm；
K3s/容器部署还必须在当前 cgroup 配置有限且不超过 `COMPUTE_SANDBOX_MAX_PIDS` 的 `pids.max`。
无法读取、无限或过大的 PID 限额都会使 `/health`、`/api/v1/status` 和执行端点进入不可服务状态。
Artifact 原始字节摘要必须等于配置 digest，且 Artifact 根和依赖根都必须由最具体的 Linux 只读挂载覆盖；启动时会对 Artifact 实际启用语言运行一次固定、无用户输入的隔离 probe 并缓存结果，probe 失败即不可服务。

## 执行契约

`runtime` profile 的 `POST /v1/execute` 必须携带以下字段：

- `executionId`：UUID
- `deploymentId`：必须等于当前 `COMPUTE_SANDBOX_DEPLOYMENT_ID`
- `projectId`：UUID
- `computeUnitId`：UUID
- `artifactDigest`：`sha256:<64 位十六进制摘要>`

运行态不得携带 `language`、`script`、`dependencies` 或 `timeoutMs`。服务只从已验证 Artifact 的启用 computeUnit 获取脚本、语言、依赖、超时和 revision，并在成功响应回显 `computeUnitId`、`revision`。

运行态只允许纯计算结果：任何 `sideEffects` 都会拒绝整次执行。它不开放依赖安装、导入、卸载或
语法检查，也不会宣称 `set`、`refresh`、`run`、`execute`、`publish` 等副作用 SDK 能力。
开发态保留 `/v1` 调试与依赖管理契约，供当前 `data_service` 调用。

开发态依赖仍按 `DependenciesDir/<projectId>` 选择。runtime 将 `DependenciesDir` 视为当前 Release/Artifact
唯一的精确依赖根，并直接只读挂载到 `/dependencies`；部署层必须提供该只读挂载，不能指向可写开发缓存。

`GET /health` 与 `GET /api/v1/status` 使用 `code`、`msg`、`data`、`reqId` 包络；状态接口只返回
运行身份和健康摘要，不返回令牌、脚本、路径或业务值。

## 安全边界与部署验收

JavaScript 的 Node `vm` 与 Python 的语言级限制只用于缩小脚本可见 API，**不是安全边界**；尤其是
带有原生实现的声明依赖可能获得 Node/Python 进程能力。runtime 的安全边界是目标 Linux 上实际生效的
`bubblewrap` 挂载、PID/网络 namespace、cgroup `pids.max` 与 `prlimit`。部署验收必须确认执行子进程仅有
`PATH`、`LANG` 环境变量、未挂载宿主根目录或凭据目录、网络 namespace 无外网能力，并且由有限 cgroup PID
上限约束子进程。还必须确认 `/usr/bin/setpriv`、bubblewrap、prlimit 与两个语言运行时均可执行；仅文件存在
不等同于目标 K3s 策略允许 user namespace 或 mount 操作。上述 Linux 集成条件未验证前，不应将语言级隔离
作为安全承诺。
