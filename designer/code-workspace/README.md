# Designer 工程开发容器

该目录维护 AI 页面开发工作台使用的固定开发镜像。镜像以
`codercom/code-server:4.131.0` 为基线，补充 Node.js 22.19.0、pnpm 10.19.0、
Pi Web 0.8.8-beta.1、公共 Vue/Vite 模板和 Preview Control。

## 容器服务

| 服务 | 容器端口 | 职责 |
| --- | ---: | --- |
| code-server | `3000` | 文件、搜索、Git、运行、扩展、Monaco 和终端。 |
| Pi Web | `30141` | AI 对话与源码修改。 |
| Vite Preview | `5173` | Designer 独立 iframe 加载的工程页面。 |
| Preview Control | `5174` | 查询、启动、停止和重启 `5173` 上的开发服务。 |

四个服务共享 `/workspace` 和 `coder` 用户。Pi 修改文件后直接触发 Vite HMR，并同时反映到
code-server。Vite 可以独立停止或重启，不参与容器健康和主进程退出判定。

## Preview Control

Preview Control 使用 Node.js 内置 HTTP 服务，提供：

```text
GET  /health
GET  /api/v1/preview/status
POST /api/v1/preview/start
POST /api/v1/preview/stop
POST /api/v1/preview/restart
```

默认自动执行 `/workspace` 中的 `pnpm dev`。模板把 Vite 固定到 `0.0.0.0:5173` 并启用
`strictPort`。如果用户先停止平台进程，再从 code-server 终端执行 `pnpm dev`，控制服务会将其
识别为 `external`；停止和重启仍只处理当前容器内由 `coder` 用户占用 `5173` 的进程。

Preview Control 的跨域来源由 `PREVIEW_CONTROL_ALLOWED_ORIGINS` 配置。开发环境端口只映射到
宿主机 `127.0.0.1`；生产环境由 Traefik、容器网络和平台鉴权共同限制访问。

## code-server 精简

镜像不 Fork code-server，也不注入 DOM 或 CSS。默认设置保留 Activity Bar、Explorer、Search、
SCM、Run、Extensions、Monaco、Terminal、Problems、Output 和状态栏，隐藏 Menu Bar、Command
Center、Layout Controls、默认辅助侧栏、欢迎页、提示、推荐和更新入口。code-server 4.131.0
不再提供 Accounts/Manage 的设置级显隐键，因此保留这两个官方 Activity Bar 入口。

默认设置只在用户设置文件不存在时复制，后续不会覆盖用户修改。启动参数同时禁用遥测、更新检查、
Workspace Trust 提示和 Getting Started 覆盖。

## 模板与缓存

- 公共模板唯一来源：`contracts/project-templates/vue-vite/`。
- 镜像构建时复制到 `/opt/induforge/template`。
- `/workspace/package.json` 不存在时才初始化模板，已有工程不覆盖。
- 依赖按 `package.json` 与锁文件指纹离线安装。
- Eruda 只在 Vite 开发态动态加载，生产构建不会包含调试桥。

## 构建与本地验证

```powershell
docker build --pull `
  --tag induforge/designer-code-server:4.131.0-node22.19-pnpm10.19-piweb0.8.8-beta.1 `
  --file designer/code-workspace/Dockerfile .
```

```powershell
docker run --rm `
  --publish 127.0.0.1:3000:3000 `
  --publish 127.0.0.1:30141:30141 `
  --publish 127.0.0.1:5173:5173 `
  --publish 127.0.0.1:5174:5174 `
  --volume "induforge-workspace-smoke:/workspace" `
  --volume "induforge-workspace-cache:/cache" `
  induforge/designer-code-server:4.131.0-node22.19-pnpm10.19-piweb0.8.8-beta.1
```

验收要求：

- `3000`、`30141`、`5173`、`5174/health` 可访问。
- 停止 Vite 后容器保持健康，code-server 和 Pi Web 不受影响。
- 启动或重启恢复 Vite 与 HMR。
- 终端手动启动的 Vite 被识别为 `external`。
- code-server 首次进入不显示欢迎页、菜单栏、命令中心、默认辅助侧栏或 Workspace Trust 提示。

## 持久化目录

| 共享卷子目录 | 容器目录 | 权限 | 用途 |
| --- | --- | --- | --- |
| `{projectId}/workspace` | `/workspace` | 读写 | Vue/Vite 工程源码。 |
| `{projectId}/context-state/current` | `/workspace/.induforge/context` | 只读 | 平台上下文。 |
| `{projectId}/code-server-data` | `/home/coder/.local/share/code-server` | 读写 | 编辑器设置和状态。 |
| `{projectId}/code-server-config` | `/home/coder/.config/code-server` | 读写 | code-server 配置。 |
| `{projectId}/pi-agent` | `/home/coder/.pi/agent` | 读写 | Pi 配置、凭据和会话。 |
| `{projectId}/cache` | `/cache` | 读写 | pnpm 和开发缓存。 |
