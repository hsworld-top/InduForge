# Designer code-server 基座计划

> 本文已于 2026-08-06 按确认后的产品边界更新，旧的 Node 控制面、一次性代理票据和控制面转发方案不再采用。

## 目标

- 一个工程对应一个共享 `code-server` 容器和一个 `code-server` 进程。
- `dev_core` 后续通过 Go Docker Engine API 创建、启动、停止和查询容器。
- 开发阶段控制面返回容器映射端口，Designer 通过 iframe 直接打开代码中心。
- `dev_core` 不转发 code-server 的 HTTP、静态资源或 WebSocket 流量。
- code-server 内安装哪些编辑器扩展或 AI 工具由客户自行决定，平台只管理挂载目录和开发产物。

## 固定开发镜像

- Dockerfile 位于 `designer/code-workspace/Dockerfile`。
- 基础镜像固定为官方 `codercom/code-server:4.131.0`。
- 开发环境固定提供 Node.js 22.18.0 与 pnpm 10.19.0，满足 Vue 3 JavaScript 工程开发。
- 容器工作目录固定为 `/workspace`。
- code-server 固定监听 `0.0.0.0:3000`，使用 `auth none` 并禁用 telemetry。
- 产品镜像名为 `induforge/designer-code-server:4.131.0-node22-pnpm10.19.0`，由 `CODE_SERVER_IMAGE` 配置。
- `auth none` 不替代平台工程权限；宿主映射端口必须限制在可信网络边界内。

## 工程挂载

```text
workspace/                    # Vue JavaScript 工程源码，读写
.induforge/context/           # 平台生成的只读上下文
.induforge/scenes/            # 2D/3D 场景上下文
code-server-data/             # 工程共享扩展和编辑器状态
code-server-config/           # 工程共享配置
```

发布时仅打包工程源码与需要的场景产物；编辑器缓存、扩展、配置和只读上下文不进入发布工件。

容器内挂载固定为：

```text
{CODE_WORKSPACE_VOLUME}:{projectId}/workspace/              -> /workspace
{CODE_WORKSPACE_VOLUME}:{projectId}/context-state/current/  -> /workspace/.induforge/context（只读）
{CODE_WORKSPACE_VOLUME}:{projectId}/code-server-data/       -> /home/coder/.local/share/code-server
{CODE_WORKSPACE_VOLUME}:{projectId}/code-server-config/     -> /home/coder/.config/code-server
{CODE_WORKSPACE_VOLUME}:{projectId}/cache/                  -> /cache
```

`CODE_WORKSPACE_ROOT` 是控制面进程操作文件时看到的卷内目录；
`CODE_WORKSPACE_VOLUME` 是控制面与工程 code-server 容器共享的 Docker named volume。
工程子目录由控制面先创建，code-server 通过 volume subpath 只挂载当前工程。

## 配置基线

| 变量 | 开发环境默认值 | 生产/离线默认值 | 作用 |
| --- | --- | --- | --- |
| `CODE_SERVER_IMAGE` | `induforge/designer-code-server:4.131.0-node22-pnpm10.19.0` | 同开发环境 | 创建工程容器使用的固定镜像。 |
| `CODE_SERVER_DOCKER_HOST` | `unix:///var/run/docker.sock` | `unix:///var/run/docker.sock` | 控制面连接宿主 Docker Engine 的地址。 |
| `CODE_SERVER_BIND_HOST` | `127.0.0.1` | `127.0.0.1` | Docker 发布动态端口时绑定的宿主地址。 |
| `CODE_WORKSPACE_VOLUME` | `induforge-control-workspaces` | 同开发环境 | 控制面与 code-server 共享的工程持久化卷。 |

`dev_core` 固定运行在 Linux 容器中，并只挂载宿主 Docker Socket；
工程 code-server 子容器不挂载 Docker Socket、平台环境文件或数据库凭据。

## 控制面职责

1. 校验工程访问与操作能力。
2. 准备工程目录和上下文目录。
3. 创建并启动工程级 code-server 容器。
4. 返回开发态访问端口和容器状态。
5. 记录最后修改结果；多人同时开发只提示在线人员，不提供代码合并和冲突解决。

## 明确不做

- 不在 Designer 内自研代码编辑器。
- 不为每个用户创建独立 code-server 会话。
- 不限制客户安装扩展或命令行工具。
- 不由控制面承诺多人编辑不丢代码；源码版本管理由客户负责。
- 不为 code-server 增加控制面 HTTP 或 WebSocket 反向代理；开发阶段直接返回宿主机映射端口。

## 后续验收

- 新建工程可创建唯一的共享 code-server 容器。
- 同一工程的不同用户进入相同工作空间和编辑器数据目录。
- Designer iframe 可使用控制面返回的开发端口直接访问。
- 容器删除不删除工程源码和共享编辑器数据。
- 发布工件不包含 `code-server-data/`、`code-server-config/` 和 `.induforge/context/`。
- 固定镜像中 `node --version`、`pnpm --version` 可用，code-server 在 3000 端口以无二次认证、无遥测方式启动。
