# Designer code-server 重构路线图

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 Designer 从普通页面拖拉拽低代码产品重构为工程级共享 `code-server`、HT 2D 编辑器、HT 3D 场景编辑器、HT 3D 模型编辑器组成的工业应用设计中心。

**架构：** `dev_core` 管理工程工作空间、工程级 `code-server` 容器、HT 工程文件、开发上下文、源码快照和构建；开发阶段返回容器映射端口，Designer iframe 直连。Designer 提供四个工作区、状态展示与预览入口；3D 场景与 3D 模型共用 HT 3D 内核和资源目录。工程源码与场景源数据是发布输入，IDE 状态、缓存、容器系统层和只读上下文不进入 Release。

**技术栈：** Go、Chi、pgx/sqlc、Redis、Vue 3、JavaScript、Vite、pnpm、code-server、Docker Engine API、PostgreSQL、HT for Web 源码版。

---

## 1. 已确认边界

- Designer 只有代码中心、2D 编辑器、3D 场景编辑器、3D 模型编辑器四个一级入口。
- 一个工程对应一个共享 `code-server` 容器和一个 `code-server` 进程。
- 平台不开发、不集成、不管理任何 AI 能力，也不识别用户安装的扩展或 CLI。
- 平台只承诺挂载目录、工程访问控制、容器生命周期和工程开发产物。
- 客户普通页面使用 Vue 3 + JavaScript，路由源码是唯一事实。
- 多人共享同一工作空间，提示在线人员但不锁定，最后一次成功保存生效。
- 2D、3D 使用 HT 源码版编辑器保存源数据，并向代码工程公开稳定调用契约。
- 3D 场景和 3D 模型复用 `index3d.html` 与同一资源目录，通过 `workspace=scene|model` 决定默认工作区。
- 正式构建读取源码快照，不读取正在运行的共享开发容器。

## 2. 持久化与发布边界

工程宿主机目录统一为：

```text
{CODE_WORKSPACE_ROOT}/{projectId}/
├─ workspace/                    # 工程源码和场景源数据
├─ code-server-data/             # 扩展、编辑器状态、用户数据
├─ code-server-config/           # 工程共享 code-server 配置
├─ cache/                        # pnpm store、Vite 等开发缓存
└─ context-state/
   ├─ current/                   # 当前完整上下文，只读挂载
   └─ staging/                   # 原子生成临时目录
```

容器内固定挂载：

```text
/workspace                              <- workspace/，读写
/workspace/.induforge/context           <- context-state/current/，只读
/home/coder/.local/share/code-server    <- code-server-data/，读写
/home/coder/.config/code-server         <- code-server-config/，读写
/cache                                  <- cache/，读写
```

Release 只允许包含：

- `/workspace` 中的 Vue、JavaScript、路由、菜单、组件和工程配置。
- 2D、3D 场景源数据及公开契约源数据。
- 图片、字体、模型和其他工程素材。
- `package.json`、`pnpm-lock.yaml` 与受控构建配置。

Release 必须排除：

- `.induforge/context/`。
- `node_modules/`、缓存、日志和临时文件。
- `code-server-data/` 与 `code-server-config/`。
- 容器系统可写层中安装的工具。

## 3. 阶段总览

| 阶段                   | 可独立验收结果                                         | 主要模块                                  |
| ---------------------- | ------------------------------------------------------ | ----------------------------------------- |
| 1. code-server 基座    | 工程可首次启动共享代码中心，目录持久化并返回映射端口   | `dev_core`、`designer`、Docker       |
| 2. JavaScript 工程基线 | 新工程得到可运行 Vue 3 + JavaScript 模板与固定版本 SDK | `dev_core`、`runtime`、受控 npm 交付 |
| 3. 工程上下文包        | 角色、数据点、场景公开契约生成只读索引和分片文档       | `dev_core`、`data_service`、HT 契约  |
| 4. HT 四工作区接入     | 2D、3D 场景、3D 模型可创建、保存、重开和预览           | `designer`、`dev_core`               |
| 5. 统一预览与构建      | Vue 与 HT 产物可统一预览；独立快照构建生成不可变版本   | `designer`、`dev_core`、`dev_ide`    |
| 6. 旧实现清理          | 删除普通页面拖拉拽、Monaco 和旧 Schema/物料死代码      | `designer`、正式文档                      |

## 4. 阶段 1：code-server 基座

**详细计划：** `docs/superpowers/plans/2026-08-03-designer-code-server-foundation-plan.md`

### 文件职责

- `dev_core/internal/project/`：创建、校验、隔离和清理工程持久化目录。
- `dev_core/internal/codeworkspace/`：按工程确定性创建、启动、检查、重建和删除容器。
- `dev_core` 提供工程容器生命周期接口并返回开发态映射端口，控制面不转发编辑器流量。
- `designer/code-workspace/Dockerfile`：固定版本工程开发镜像。
- `designer/src/ui/workspaces/DesignerWorkspaceView.vue`：代码、2D、3D 场景、3D 模型四工作区壳层。
- `designer/src/ui/workspaces/code/CodeWorkspacePanel.vue`：启动状态、共享提示、在线人员和 iframe。

### 验收

- [ ] 新建工程只创建目录，不创建容器。
- [ ] 首次打开代码中心创建并启动唯一工程容器。
- [ ] 两名平台工程开发人员可进入同一工作空间。
- [ ] `dev_core` 校验工程权限后返回开发态映射端口，Designer iframe 直接访问。
- [ ] 停止或重建容器后源码、扩展、设置和缓存仍保留。
- [ ] 子容器没有 Docker Socket、平台 `.env`、数据库凭据或其他工程挂载。
- [ ] 默认空闲 30 分钟后 `code-server` 进程退出，容器停止。
- [ ] Designer 只展示在线人员，不解析 code-server 当前文件和工具状态。

## 5. 阶段 2：Vue JavaScript 模板与固定 SDK

### 文件职责

- 在 `dev_core/internal/project/` 维护标准 Vue 3 + JavaScript 工程模板。
- 创建 `runtime/web-sdk/`：固定版本 JavaScript SDK 源码、类型提示和构建脚本。
- 修改 `projectWorkspaceService`：仅在空工作空间中复制模板，不覆盖客户文件。
- 修改离线构建与 npm 缓存：确保模板依赖和 SDK 在无互联网环境可安装。
- 创建模板验证脚本：从空目录初始化、安装、检查并构建生产包。

### 固定 SDK 边界

```js
import { points, access, scenes } from '@induforge/runtime-sdk'

const rows = await points.orders.get({ startTime, endTime, lineId })
await points.speed.set(1200)
const unsubscribe = points.temperature.sub((event) => console.log(event.value))

if (access.hasRole('operator')) {
  await scenes.open2D('main-process')
}
```

SDK 是固定版本包，不按工程动态生成。工程动态信息只进入只读上下文包。

### 验收

- [ ] 新工程可直接执行 `pnpm install` 和 `pnpm build`。
- [ ] 客户源码不需要 TypeScript。
- [ ] 模板路由同时支持 Vue、2D、3D 入口。
- [ ] SDK 在在线与离线依赖源中使用同一版本。
- [ ] 工作空间已有文件时初始化流程不覆盖任何客户产物。

## 6. 阶段 3：机器契约与工程上下文包

### 文件职责

- `data_service/internal/http/router/router.go`：增加工程数据点开发契约分页/流式导出接口。
- `data_service/internal/service/datapoint_contract_service.go`：从数据点定义生成稳定机器契约。
- `dev_core/internal/contextpack/`：聚合角色、数据点和 HT 场景契约，生成索引、分片详情与 manifest。
- `dev_core` 提供上下文状态查询和手动刷新接口。
- 2D、3D 场景服务：输出公开机器契约，不输出 Markdown。

### 上下文结构

```text
/workspace/.induforge/context/
├─ README.md
├─ manifest.json
├─ project/
│  ├─ overview.md
│  └─ roles.md
├─ points/
│  ├─ index.md
│  ├─ index.json
│  └─ shards/*.md
├─ scenes/
│  ├─ 2d.md
│  └─ 3d.md
└─ sdk/
   └─ usage.md
```

### 验收

- [ ] 10 万级数据点使用索引加分片，不生成单个超大 Markdown。
- [ ] 文档只包含定义、参数、返回、权限和示例，不包含实时值、SQL、密码或 Token。
- [ ] 生成失败保留上一份完整上下文并标记过期。
- [ ] `data_service` 与 HT 场景只输出机器契约，Markdown 由 `dev_core` 统一生成。
- [ ] 上下文目录在 code-server 中只读，用户无法反向修改数据库和场景。

## 7. 阶段 4：HT 四工作区接入

### 文件职责

- 将源码版 HT 的 `client/` 与 `instance/custom/` 合并到 `designer/public/ht-editor/`。
- 新增 `InduForgeService.js`，以 REST 适配 `explore/upload/source/remove/rename/mkdir/locate/paste` 协议。
- 在 `dev_core/internal/designworkspace/` 增加工程设计文件浏览、读写、重命名、删除和复制接口。
- 重建 `designer/src/ui/workspaces/`：代码、2D、3D 场景、3D 模型四工作区。
- 2D 使用 `index.html`；3D 场景与模型共用 `index3d.html`，分别传入 `workspace=scene|model`。
- 场景成功保存后触发项目上下文原子刷新。

### 场景源码边界

- 编辑器内部源数据可读写，但普通 Vue 代码不直接修改内部结构。
- Vue 代码只通过固定 SDK 和场景公开标识进行导航、变量传递和事件交互。
- 场景保存采用完整写入，最后一次成功保存生效，不做 CRDT、锁或自动合并。

### 验收

- [ ] 2D、3D 场景和 3D 模型均可创建、保存、重新打开和预览。
- [ ] 多人同时保存时不锁定，失败保存不会显示成功。
- [ ] 场景公开变量、事件和方法保存后进入上下文包。
- [ ] Vue 代码无需读取编辑器私有 Schema 即可调用场景。

## 8. 阶段 5：统一预览、快照构建和 Release

### 文件职责

- `dev_core/internal/deployment/`：生成只读源码快照、启动一次性干净构建容器并记录发布信息。
- Designer 统一预览入口：承载 Vue 页面、2D、3D 和数据点错误面板。
- `dev_ide` 版本界面：选择 Designer 生成的不可变版本并部署。

### 构建输入过滤

```js
const excludedPaths = ['.induforge/context', 'node_modules', '.pnpm-store', 'dist', '*.log']
```

### 验收

- [ ] 构建前创建稳定快照；文件变化时丢弃并重建快照。
- [ ] 构建容器不复用开发容器的依赖、环境变量或系统层。
- [ ] Release 可追溯到源码哈希、SDK 版本、锁文件哈希和构建日志。
- [ ] 预览明确区分源码编译、数据点、权限和运行时错误。
- [ ] `dev_ide` 只部署已成功构建的不可变应用版本。

## 9. 阶段 6：删除旧拖拉拽与 Monaco

### 删除范围

- 普通页面画布、物料、属性面板、页面 Schema 和页面锁。
- Monaco 依赖、源码工作台残留和相关 Vite 手工分包。
- 只为旧普通页面运行态服务的导出、预览和发布接口。
- 与新源码事实冲突的正式文档和跨模块契约。

### 保留范围

- 可被新 2D 编辑器实际复用且通过测试的 Konva、几何、素材能力。
- 可被统一预览实际复用的数据点错误展示和运行时桥接能力。
- 已被新 Release 契约引用的通用发布、部署和对象存储服务。

### 验收

- [ ] `designer/package.json` 不再包含 `monaco-editor` 和普通页面拖拉拽专用依赖。
- [ ] 根路由不再加载旧 `DesignerView.vue`。
- [ ] 代码库不存在旧普通页面 Schema 的生产调用链。
- [ ] `pnpm lint`、`pnpm typecheck`、Designer 测试和构建全部通过。
- [ ] 远端备份分支 `拖拉拽版本Designer` 保持可追溯，不在主分支保留兼容代码。

## 10. 依赖顺序与执行规则

1. 阶段 1 先固定目录、容器和访问路径，后续阶段不得改变已发布挂载契约。
2. 阶段 2 固定 JavaScript 模板和 SDK 包名，阶段 3 的文档示例必须使用同一 API。
3. 阶段 3 先完成机器契约，再让阶段 4 的场景编辑器触发上下文刷新。
4. 阶段 5 只读取阶段 1 定义的工程产物和阶段 2 定义的锁文件，不读取开发容器状态。
5. 阶段 6 必须在新预览和新构建链路通过后执行，删除旧代码但不增加兼容分支。

每个阶段单独创建详细实施计划、单独完成静态检查和最贴近变更面的测试。任一阶段没有通过验收时，不开始依赖它的下一阶段。
