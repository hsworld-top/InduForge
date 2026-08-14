# Designer 工程开发容器

该目录维护 AI 页面开发工作台使用的固定开发镜像。镜像以
`codercom/code-server:4.131.0` 为基线，补充 Node.js 24.19.0 LTS、npm 11.17.0、pnpm 11.21.0、
仓库内 `designer/pi-web` 源码、四套官方 Vite 模板和 Preview Control。

## 容器服务

| 服务 | 容器端口 | 职责 |
| --- | ---: | --- |
| code-server | `3000` | 文件、搜索、Git、运行、扩展、Monaco 和终端。 |
| Pi Web | `30141` | AI 对话与源码修改。 |
| Vite Preview | `5173` | Designer 独立 iframe 加载的工程页面。 |
| Preview Control | `5174` | 查询、启动、停止和重启 `5173` 上的开发服务。 |

四个服务共享 `/workspace` 和 `coder` 用户。Pi 修改文件后直接触发 Vite HMR，并同时反映到
code-server。Vite 可以独立停止或重启，不参与容器健康和主进程退出判定。

## InduForge AI

Pi Web Fork 固定使用 `/workspace`，不提供品牌顶栏、项目选择、语言/主题选择、Plugins、系统提示词
查看和 Worktree。保留模型、Provider、API Key、会话、会话分叉、文件、Diff、工具调用和 Project
Skills。Project Skills 只允许 `/workspace/.pi/skills`；平台只读 Extension 只从
`/opt/induforge/pi/extensions` 加载。

容器通过 `PI_WEB_ALLOWED_PARENT_ORIGINS` 限制 Designer Origin。Designer 与 Pi Web 使用
`INDUFORGE_PI_READY` / `INDUFORGE_PI_CONTEXT` 同步工程 ID、`/workspace`、语言和主题，双方均校验
Origin、消息版本和工程 ID。

## Preview Control

Preview Control 使用 Node.js 内置 HTTP 服务，提供：

```text
GET  /health
GET  /api/v1/workspace/status
GET  /api/v1/workspace/templates
POST /api/v1/workspace/initialize
GET  /api/v1/preview/status
POST /api/v1/preview/start
POST /api/v1/preview/stop
POST /api/v1/preview/restart
```

空工作区启动时只运行 Pi Web、code-server 和 Preview Control，不启动 Vite。用户通过初始化接口
选择模板后，控制服务离线安装依赖、创建本地 Git 初始提交，再使用平台 Vite Runner 加载工程自身
Vite 配置并固定到 `0.0.0.0:5173` 与 `strictPort`。Runner 只在开发态注入 Eruda 桥，不修改模板源码。

如果用户先停止平台进程，再从 code-server 终端执行 `pnpm dev -- --host 0.0.0.0 --port 5173 --strictPort`，
控制服务会将其识别为 `external`；停止和重启仍只处理当前容器内由 `coder` 用户占用 `5173` 的进程。
外部 Vite 不包含平台 Eruda 注入，执行“重启”可恢复受控 Runner。

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

- 公共模板唯一来源：`contracts/project-templates/`，目录由 `catalog.json` 描述。
- 模板固定为 Vite 官方 Vue JavaScript、Vue TypeScript、React JavaScript 和 React TypeScript 四种。
- 镜像构建时复制到只读 `/opt/induforge/templates`，同时预热 pnpm store。
- `/workspace` 必须为空且没有初始化标记；非空未知工作区不会被覆盖。
- 初始化在临时目录完成复制、离线安装、标记写入和 Git 提交，成功后再移动到 `/workspace`。
- `/workspace/.induforge/project.json` 是初始化状态标记；已初始化工程再次初始化返回 HTTP `409`。
- 初始仓库分支为 `main`，只配置本地提交身份，不配置远端。
- Eruda 由平台 Vite Runner 在开发态注入，四套官方模板和生产 `dist` 均不包含调试桥代码。

## 构建与本地验证

```powershell
docker build --pull `
  --tag induforge/designer-code-server:workspace-templates-source `
  --file designer/code-workspace/Dockerfile .
```

开发工作区固定使用以下宿主端口，和根目录 `.env.development.example` 保持一致：

| 服务 | 宿主端口 | 容器端口 |
| --- | ---: | ---: |
| code-server | `36200` | `3000` |
| Pi Web | `36241` | `30141` |
| Vite Preview | `38173` | `5173` |
| Preview Control | `38174` | `5174` |

构建新镜像后，在 WSL Ubuntu 中运行固定重建脚本：

```bash
cd /mnt/e/personal_dev/InduForge
INDUFORGE_WORKSPACE_IMAGE=induforge/designer-code-server:workspace-templates-source \
  sh ./designer/code-workspace/recreate-development-container.sh
```

脚本固定重建 `induforge-designer-workspace-dev`。工程、缓存、Pi 配置与会话、code-server 设置使用
独立命名卷持久化，因此日常更新只删除旧容器并使用新镜像重建，不删除这些卷。只有明确需要重置开发
工程或凭据时才手动删除对应卷。

```powershell
docker run --rm `
  --publish 127.0.0.1:3000:3000 `
  --publish 127.0.0.1:30141:30141 `
  --publish 127.0.0.1:5173:5173 `
  --publish 127.0.0.1:5174:5174 `
  --volume "induforge-workspace-smoke:/workspace" `
  --volume "induforge-workspace-cache:/cache" `
  induforge/designer-code-server:workspace-templates-source
```

镜像使用多阶段构建直接编译 `designer/pi-web`，随后安装本次构建生成的 tarball，不依赖已发布的
`@induforge/pi-web` 包。

验收要求：

- 空卷启动后 `3000`、`30141` 和 `5174/health` 可访问，`5173` 保持停止。
- 四套模板均可完成初始化、离线安装、Git 初始提交、Vite 启动和生产构建。
- 停止 Vite 后容器保持健康，code-server 和 Pi Web 不受影响。
- 启动或重启恢复 Vite 与 HMR。
- 终端手动启动的 Vite 被识别为 `external`。
- code-server 首次进入不显示欢迎页、菜单栏、命令中心、默认辅助侧栏或 Workspace Trust 提示。

## 持久化目录

| 共享卷子目录 | 容器目录 | 权限 | 用途 |
| --- | --- | --- | --- |
| `{projectId}/workspace` | `/workspace` | 读写 | Vue 或 React Vite 工程源码。 |
| `{projectId}/context-state/current` | `/workspace/.induforge/context` | 只读 | 平台上下文。 |
| `{projectId}/code-server-data` | `/home/coder/.local/share/code-server` | 读写 | 编辑器设置和状态。 |
| `{projectId}/code-server-config` | `/home/coder/.config/code-server` | 读写 | code-server 配置。 |
| `{projectId}/pi-agent` | `/home/coder/.pi/agent` | 读写 | Pi 配置、凭据和会话。 |
| `{projectId}/cache` | `/cache` | 读写 | pnpm 和开发缓存。 |
