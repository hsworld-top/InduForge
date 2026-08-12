# Designer code-server 开发镜像

该目录维护 Designer 页面工程使用的固定开发镜像。镜像以官方
`codercom/code-server:4.131.0` 为基线，并补充 Vue 3 JavaScript 工程需要的
Node.js 22.18.0 与 pnpm 10.19.0。

## 固定运行配置

- 容器工作目录：`/workspace`
- code-server 监听地址：`0.0.0.0:3000`
- 登录方式：`auth none`
- 遥测：禁用
- 无访问连接 30 分钟后进程退出，容器随之停止
- 默认用户：`coder`
- 工作空间、扩展和配置均由控制面创建容器时挂载，镜像本身不保存工程数据。
- 镜像内预取默认 Vue 工程锁文件对应的 pnpm 包；首次启动时复制到工程共享缓存，断网环境可直接安装默认模板依赖。

`auth none` 只表示 code-server 不再维护第二套账号。工程访问权限仍由平台控制，
宿主机端口必须按部署网络边界限制访问，不能直接暴露到不可信网络。

## 构建

在仓库根目录执行：

```powershell
docker build --pull --tag induforge/designer-code-server:4.131.0-node22-pnpm10.19.0 --file designer/code-workspace/Dockerfile .
```

构建结果应与环境变量保持一致：

```dotenv
CODE_SERVER_IMAGE=induforge/designer-code-server:4.131.0-node22-pnpm10.19.0
```

## 本地验证

```powershell
docker run --rm `
  --publish 127.0.0.1:3000:3000 `
  --volume "${PWD}:/workspace" `
  induforge/designer-code-server:4.131.0-node22-pnpm10.19.0
```

打开 `http://127.0.0.1:3000` 后，应直接进入 `/workspace`。终端中执行以下命令应成功：

```sh
node --version
pnpm --version
pnpm install --offline --frozen-lockfile
pnpm build
```

## 控制面挂载约定

每个工程只创建一个共享容器。控制面后续创建容器时使用以下挂载：

| 共享卷子目录 | 容器目录 | 权限 | 用途 |
| --- | --- | --- | --- |
| `{CODE_WORKSPACE_VOLUME}:{projectId}/workspace` | `/workspace` | 读写 | Vue 3 JavaScript 工程源码 |
| `{CODE_WORKSPACE_VOLUME}:{projectId}/context-state/current` | `/workspace/.induforge/context` | 只读 | 平台生成上下文 |
| `{CODE_WORKSPACE_VOLUME}:{projectId}/code-server-data` | `/home/coder/.local/share/code-server` | 读写 | 共享扩展和编辑器状态 |
| `{CODE_WORKSPACE_VOLUME}:{projectId}/code-server-config` | `/home/coder/.config/code-server` | 读写 | 共享编辑器配置 |
| `{CODE_WORKSPACE_VOLUME}:{projectId}/cache` | `/cache` | 读写 | pnpm 与开发缓存 |

平台生成的 `.induforge/context/` 与 `.induforge/scenes/` 位于工程源码目录内，
由控制面负责生成和更新；code-server 只把它们作为工程上下文读取。

## 环境变量

| 变量 | 含义 |
| --- | --- |
| `CODE_SERVER_IMAGE` | 控制面创建工程容器时使用的完整镜像名，必须指向本目录构建出的固定镜像。 |
| `CODE_SERVER_DOCKER_HOST` | 控制面连接 Docker Engine 的地址；容器部署通常使用 `unix:///var/run/docker.sock`。 |
| `CODE_SERVER_BIND_HOST` | Docker 发布 code-server 动态端口时绑定的宿主机地址；默认仅绑定 `127.0.0.1`。 |
| `CODE_WORKSPACE_VOLUME` | 控制面和工程 code-server 容器共享的 Docker named volume。 |
| `CODE_WORKSPACE_ROOT` | `dev_core` 进程自身看到的工作空间根目录，用于创建和管理工程文件。 |

`dev_core` 通过 `CODE_WORKSPACE_ROOT` 操作共享卷内容，Docker Engine
按 `CODE_WORKSPACE_VOLUME` 和工程子目录把同一份数据挂载到 code-server。
