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

Windows 包安装并声明 collector 能力，但默认不会启动 collector。领取节点不要求已有工程
release；要允许中心的部署命令启动 collector，仍须由本机运维人员在 `config.yaml` 为该服务
补齐 `releaseRoot`、`releaseDigest`、`executable`、`healthUrl` 等受控 release 配置。中心不会
下发命令、路径、环境变量或密钥；缺少物料时 NodeAgent 会拒绝启动服务而不伪报运行。
