# Collector

`collector/` 是工业采集共享代码入口，开发态调试代理和运行态工程采集器复用同一套驱动契约与协议适配器。

当前阶段只实现：

- OPC UA 连接测试。
- OPC UA 地址空间浏览。
- OPC UA 批量读取。
- Windows 采集调试代理所需的共享契约。

不在公开类型、页面、工程配置和普通日志中暴露商业 SDK 供应商或授权信息。

## 验证

```powershell
dotnet test collector/InduForge.Collector.slnx
```

## Windows 调试代理

当前程序是 Windows 托盘 EXE：

- 启动后显示状态窗口。
- 关闭窗口后继续驻留托盘。
- 托盘“退出”才结束进程。
- 支持使用一次性注册码接入中心，并使用 DPAPI 保存 Agent Token。
- 支持 15 秒心跳、2 秒任务轮询和 Connect/Browse/Read 串行执行。
- 可使用 `opc.tcp://127.0.0.1:18540/induforge/sim` 执行 OPC UA 本地自检。

生成自包含单文件：

```powershell
pnpm dotnet:publish:collector-dev
```

输出位于本地忽略目录 `collector/artifacts/win-x64/`。
