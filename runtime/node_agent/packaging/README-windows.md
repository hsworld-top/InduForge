# InduForge NodeAgent（Windows 采集节点）

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
