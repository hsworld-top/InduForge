# Collector

`collector/` 是工业采集共享代码入口，开发态调试代理和运行态工程采集器复用同一套驱动契约与协议适配器。

当前阶段实现：

- 115 类驱动的共享 Manifest、连接与地址契约。
- Windows DevAgent 全驱动注册、连接测试、会话打开/关闭和点读取。
- OPC UA 地址空间浏览与批量读取。
- HSL Adapter、专用协议 Adapter 与 Windows 采集调试代理所需的共享契约。
- `industrial_collector` Runtime V1：仅白名单驱动的长期采集、本站 WAL、NATS JetStream 回放及本地健康状态端点。

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

## 生产 Runtime V1

`industrial_collector` 是生产运行采集器；DevAgent 仅用于开发态联通性验证，不能替代长期采集运行时。

发布自包含产物（输出均位于已忽略的 `collector/artifacts/runtime/`）：

```powershell
pnpm collector:publish:runtime:win-x64
pnpm collector:publish:runtime:linux-x64
```

启动时只接受 Artifact、Binding、受限 index、WAL 和监听地址等文件/引用参数。生产环境必须提供稳定站点身份 `--site-id`（或 `INDUFORGE_COLLECTOR_SITE_ID`）；非生产环境未提供时才使用 `local`：

```text
industrial_collector \
  --artifact <绝对路径/artifact.json> \
  --binding <绝对路径/binding.json> \
  --index <绝对路径/index.json> \
  --wal <绝对路径/wal目录> \
  --listen http://127.0.0.1:18081/ \
  --site-id site-a \
  --production true
```

也可使用同名 `INDUFORGE_COLLECTOR_ARTIFACT`、`BINDING`、`INDEX`、`WAL`、`LISTEN`、`SITE_ID` 和 `PRODUCTION` 环境变量。Artifact/Binding/index 必须是绝对路径。生产 index 和其 secret 必须在不可写、不可替换的受控挂载中：Linux 要求同目录相对 secret、无链接、私有读权限及只读挂载；Windows 以最终打开的 handle 验证普通对象、拒绝 reparse/junction，以及当前进程具有写入、删除或目录替换能力的挂载。无法证明只读时启动失败。

NATS resource 的 JSON 形状严格为（不得增加字段）：

```json
{"url":"nats://nats.internal:4222","accountId":"account-a"}
```

其中 `accountId` 必须为 stableId，且必须与 Binding 的 `accountId` 完全一致；比对发生在读取凭据和建立连接之前。凭据 schema 始终为 `nats-credential.v1`，生产环境不允许 `authType: "none"`。URL 与凭据不会写入错误输出、状态接口或 `ToString`；Binding 的稳定 `accountId` 仅作为 `/api/v1/status` 的租户审计身份返回。

健康接口为 `GET /health`（只返回 `{status,observedAt}`）和 `GET /api/v1/status`（完整 `runtime-health-status.v1`）。未知路径返回 JSON 404；已知路径的非 GET 或带请求体请求返回 JSON 405，并带 `Allow: GET`。`/health` 只在 `RUNNING` 且健康为 `HEALTHY` 或 `DEGRADED` 时返回 200/UP，否则返回 503/DOWN。

收到 Ctrl+C 或 Unix `SIGTERM` 时，Runtime 先停止新采集，再在最多 20 秒内继续回放现有 WAL，直到每条记录取得 PubAck 并写入 ACK marker。上游不可用导致超时会保留 WAL，退出码为 `4`，不会声称已排空；配置错误为 `64`，启动失败为 `2`，运行任务故障为 `3`。
