# InduForge NodeAgent（Windows）

请使用“以管理员身份运行”的 PowerShell 解压并安装：

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\install.ps1 `
  -ServerUrl "https://你的中心地址" `
  -EnrollmentCode "中心生成的一次性接入码"
```

默认程序目录为 `%ProgramFiles%\InduForge\NodeAgent`，配置与节点身份位于
`%ProgramData%\InduForge\NodeAgent`。两个接入参数齐全时安装器会创建并启动
`InduForgeNodeAgent` Windows 服务；接入码领取成功后会从配置中清除。

常用命令：

```powershell
Get-Service InduForgeNodeAgent
Restart-Service InduForgeNodeAgent
```

使用 `-NoService` 可以只释放文件、暂不创建服务。`uninstall.ps1` 默认保留配置、节点
身份、日志与诊断数据；确实需要重新注册时再使用 `uninstall.ps1 -Purge` 清除全部状态。

当前仓库只完成了 Windows 二进制交叉构建和安装器静态校验，正式交付前仍需在真实 Windows
主机完成安装、启动、停止、重装与接入测试。

Windows 包只安装并声明 `collector` 能力，默认不会启动 Collector。Gateway、Runtime API 和
RuntimeEngine 不在 Windows 包内；收到这些服务的部署意图必须 fail-closed，不能把 Linux 单节点
模型当作 Windows 已交付能力。

领取节点不要求已有工程 Release。正式 Agent 路径会以自身 token 拉取节点专属 Binding 和同源
Release 流，拒绝重定向、外部下载 URL、中心下发路径/命令/环境变量/Secret；签名公钥只来自本地
trust store。Windows 不具备 `project_entry` 和 `data_runtime` 能力，收到其 Binding 必须拒绝；即使
Collector Release 通过安全安装，正式 launcher 未交付时也不得启动或伪报运行。旧本地 Supervisor
配置不是绕过该门禁的正式部署方式。

模块已提供跨平台的 `ReleaseStore` 状态指针基础：它接收 tar.zst 流、outer SHA-256 和由调用方
从本地 trust store 选定的 Ed25519 key，验证 `release-checksums.v1` 与原始 64-byte
`signature.sig`，再内容寻址保存并原子更新 `current.json`。Release 根目录只可包含
`release-manifest.json`、`client-assets.tar.zst`、`runtime-artifact.tar.zst`、
`health-contract.json`、`resource-recommendation.json`、`schema-plan.json`、`sbom.cdx.json`、可选
`collector-artifact.tar.zst`、`checksums.json` 与 `signature.sig`，不能含目录或其他路径。Manifest
必须声明 `supplyChain.signingKeyId`。`checksums.json` 的 `files` 为按路径严格升序的
`{path,sha256,size}`，不得列出自身或 `signature.sig`；签名精确覆盖原始 checksums 字节。

该原语已接入 Agent 的 Binding/Release 下载与本地 trust 校验，但仍未接入 Secret、真实只读 mount、
Runtime Foundation 或进程启动/回滚；不能据此声称 Windows 已可端到端安装工程。Center 即使同机
仍需独立服务账户、目录、进程、端口和网络连接。
