# Collector

`collector/` 是工业采集共享代码入口，开发态调试代理和运行态工程采集器复用同一套驱动契约与协议适配器。

当前阶段实现：

- 115 类驱动的共享 Manifest、连接与地址契约。
- Windows DevAgent 全驱动注册、连接测试、会话打开/关闭和点读取。
- OPC UA 地址空间浏览与批量读取。
- HSL Adapter、专用协议 Adapter 与 Windows 采集调试代理所需的共享契约。

数据中心可以在没有 Agent 时完成稳定的数据建模。DevAgent 只承担开发态短链联通性验证，不承担项目运行态长期采集。

不在公开类型、页面、工程配置和普通日志中暴露商业 SDK 供应商或授权信息。

## 验证

```powershell
dotnet test collector/InduForge.Collector.slnx
```

macOS 可用于还原、静态编译和不依赖 Windows Desktop Runtime 的测试：

```bash
DOTNET_CLI_HOME=/private/tmp/induforge-dotnet \
NUGET_PACKAGES=/private/tmp/induforge-nuget \
DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1 \
dotnet restore collector/InduForge.Collector.slnx -p:EnableWindowsTargeting=true
```

WinForms 调试代理、DPAPI、串口枚举、HSL 设备通信和完整测试宿主必须在 Windows 上验收。macOS 编译成功不能替代真实连接、读取失败提示、取消和资源释放测试。

## Windows 调试代理

当前程序是 Windows 托盘 EXE：

- 启动后显示状态窗口。
- 关闭窗口后继续驻留托盘。
- 托盘“退出”才结束进程。
- 支持使用一次性注册码接入中心，并使用 DPAPI 保存 Agent Token；注册码提交后立即从界面清空。
- 支持 15 秒心跳、2 秒任务轮询和 Connect/Browse/Read 串行执行。
- 可使用 `opc.tcp://127.0.0.1:18540/induforge/sim` 执行 OPC UA 本地自检。
- 凭据保存到 EXE 同级目录的 `data/credentials.dat`。
- 日志保存到 EXE 同级目录的 `logs/collector-dev-agent-yyyyMMdd.log`，按天滚动并保留最近 14 个文件。

EXE 所在目录必须可写。旧版 `LocalAppData` 凭据不再读取，升级后需要重新注册一次。

生成自包含单文件：

```powershell
pnpm collector:publish:dev
```

输出位于本地忽略目录 `collector/artifacts/win-x64/`。
