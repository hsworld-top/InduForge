# InduForge - 工业应用低代码开发平台

## 项目简介

InduForge 是一个面向工业互联网场景的多模块单仓低代码平台，覆盖平台治理、数据域、可视化设计、发布部署和节点执行链路。当前仓库同时包含 Node.js 后端、多个 Vue 前端以及 Go 运行时服务，开发时应按模块分别处理，而不是把整仓当成单一应用。

## 核心特性

- 🎨 **可视化设计器** - 拖拽式页面设计，支持实时预览
- 📊 **数据中心** - 多数据源管理，支持 MySQL、PostgreSQL、SQL Server
- 🔧 **工程管理** - 项目、租户、用户统一管理
- 🚀 **微前端架构** - 独立开发、独立部署、灵活扩展
- 🔌 **数据绑定** - 强大的表达式系统，支持实时数据订阅
- 🎭 **组件库** - 丰富的工业组件和图表组件

## 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Nginx 反向代理                            │
│  - 统一域名访问                                              │
│  - 静态资源服务                                              │
│  - API 代理                                                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌──────────────┬──────────────┬──────────────┬──────────────┬──────────────┐
│   IDE 主应用  │  数据中心     │  设计中心     │  控制面后端    │  数据域服务    │
│   (dev_ide)  │ (datacenter) │  (designer)  │  (dev_core)  │ (data_service)│
│              │              │              │              │              │
│  - 工程管理   │  - 数据连接   │  - 页面设计   │  - 用户认证   │  - 数据连接   │
│  - 用户管理   │  - SQL 查询   │  - 组件库     │  - 租户管理   │  - 查询与数据点│
│  - 部署运维   │  - MQTT/数据点│  - 数据绑定   │  - 发布部署   │  - 协议与预览 │
│              │              │              │              │              │
│  Vue 3       │  Vue 3       │  Vue 3       │  Node.js     │  Go          │
│ Port: 18601  │ Port: 18602  │ Port: 18603  │ Port: 19601  │ Port: 19602  │
└──────────────┴──────────────┴──────────────┴──────────────┴──────────────┘
```

## 技术栈

### 前端
- **框架**: Vue 3 + Vite
- **状态管理**: Pinia
- **UI 组件**: Element Plus
- **样式**: Tailwind CSS
- **Canvas 渲染**: Konva.js
- **图表**: ECharts
- **代码编辑器**: Monaco Editor
- **动画**: GSAP

### 后端
- **运行时**: Node.js 18+
- **框架**: Express
- **ORM**: Sequelize
- **数据库**: MySQL / PostgreSQL / SQL Server
- **认证**: JWT
- **国际化**: i18next

### 数据域与运行时
- **数据域服务**: Go (`data_service`)
- **节点执行器**: Go (`runtime/node_agent`)
- **本地运维前端**: Vue 3 + Vite (`runtime/node_agent_front`)
- **对象存储**: SeaweedFS S3 兼容接口

## 当前模块现状

- `dev_core/`：平台控制面后端，负责认证、工程管理、发布部署、节点调度和聚合 API。
- `data_service/`：平台侧与开发态数据域服务，负责连接、查询、数据点、协议接入、计算与预览会话。
- `dev_ide/`：平台管理与运维前端。
- `datacenter/`：数据接入、查询管理与数据语义建模前端。
- `designer/`：低代码页面设计器，当前是前端模块里最重的编辑器工程。
- `runtime/node_agent/`：节点执行器后端，独立部署在节点侧。
- `runtime/node_agent_front/`：节点本地管理前端。
- `scripts/`：本地开发与基础设施辅助脚本，当前包含 `docker/`、`nginx/`、`seaweedfs/`。

## 默认端口总览

| 模块 | 服务类型 | 环境变量 | 默认端口 | 默认访问地址 |
| --- | --- | --- | --- | --- |
| `dev_ide` | 平台管理前端 | `VITE_IDE_PORT` | `18601` | `http://localhost:18601` |
| `datacenter` | 数据中心前端 | `VITE_DATACENTER_PORT` | `18602` | `http://localhost:18602` |
| `designer` | 设计器前端 | `VITE_DESIGNER_PORT` | `18603` | `http://localhost:18603` |
| `runtime/node_agent_front` | 节点本地管理前端 | `VITE_NODE_AGENT_FRONT_PORT` | `18604` | `http://localhost:18604` |
| `dev_core` | 平台控制面后端 | `PORT` | `19601` | `http://localhost:19601` |
| `data_service` | 数据域服务 | `DATA_SERVICE_ADDR` / `VITE_DATA_SERVICE_URL` | `19602` | `http://localhost:19602` |
| `runtime/node_agent` | 节点执行器后端 | `NODE_AGENT_PORT` | `17601` | `http://localhost:17601` |

## 快速开始

### 环境要求

- Node.js >= 18.0.0
- pnpm >= 8.0.0
- Go >= 1.25.0（用于 `data_service` 与 `runtime/node_agent`）
- MySQL >= 5.7 或 PostgreSQL >= 12

### Windows 开发说明

- 当前开发环境默认以 Windows + PowerShell 为主，下面示例优先使用 PowerShell 写法。
- 若需要同时启动多个模块，建议为每个模块单独打开一个 PowerShell 窗口。
- `scripts/seaweedfs/` 下提供了本地 SeaweedFS 启停脚本，可按需单独启动。
- 默认端口以根 `.env` 和 `.env_example` 为准；如果本地仍沿用旧端口，需要同步更新根 `.env`。

### 安装依赖

```powershell
# 在仓库根目录依次安装 Node 模块依赖
Set-Location .\dev_core
pnpm install

Set-Location ..\dev_ide
pnpm install

Set-Location ..\datacenter
pnpm install

Set-Location ..\designer
pnpm install

Set-Location ..\runtime\node_agent_front
pnpm install

Set-Location ..\..
```

```powershell
# 初始化 Go 模块依赖
Set-Location .\data_service
go mod download

Set-Location ..\runtime\node_agent
go mod download

Set-Location ..\..
```

### 初始化数据库

```powershell
Set-Location .\dev_core
# 编辑根目录 .env 后再执行初始化
pnpm run db:init
```

### 启动开发服务器

```powershell
# 1. 启动平台控制面后端
Set-Location .\dev_core
pnpm dev

# 2. 启动数据域服务（新终端，可选）
Set-Location .\data_service
go run .\cmd

# 3. 启动 IDE（新终端）
Set-Location .\dev_ide
pnpm dev

# 4. 启动数据中心（新终端）
Set-Location .\datacenter
pnpm dev

# 5. 启动设计中心（新终端）
Set-Location .\designer
pnpm dev

# 6. 启动节点本地管理前端（新终端，可选）
Set-Location .\runtime\node_agent_front
pnpm dev
```

访问地址：
- IDE: http://localhost:18601
- 数据中心: http://localhost:18602
- 设计中心: http://localhost:18603
- 节点本地管理前端: http://localhost:18604
- 平台控制面后端: http://localhost:19601
- 数据域服务: http://localhost:19602
- 节点执行器后端: http://localhost:17601

### 默认账号

- 超级管理员: `superadmin` / `admin123`
- 系统管理员: `admin` / `admin123`

## 项目结构

```
InduForge/
├── dev_core/           # 后端 API 服务
│   ├── src/            # 源代码
│   ├── database/       # 数据库脚本
│   └── config/         # 配置文件
├── data_service/       # 平台侧与开发态数据域服务（Go）
│   ├── cmd/            # 启动入口
│   ├── internal/       # 内部领域实现
│   └── tests/          # 测试目录
├── dev_ide/            # IDE 主应用
│   ├── src/            # 源代码
│   └── public/         # 静态资源
├── datacenter/         # 数据中心应用
│   ├── src/            # 源代码
│   └── public/         # 静态资源
├── designer/           # 设计中心应用
│   ├── src/            # 源代码
│   ├── engine/         # 核心引擎
│   └── registry/       # 组件注册
├── runtime/            # 运行时相关模块
│   ├── node_agent/     # 节点执行器后端（Go）
│   └── node_agent_front/ # 节点本地管理前端
├── scripts/            # 辅助脚本与基础设施配置
│   ├── docker/         # 本地容器相关脚本
│   ├── nginx/          # Nginx 配置
│   └── seaweedfs/      # SeaweedFS 启停脚本
└── docs/               # 项目文档
```

## 文档导航

### 📚 核心文档
- [DSL 设计规范](./docs/dsl-design.md) - 低代码 DSL 完整规范
- [数据库设计](./docs/database-design.md) - 数据库表结构说明
- [迁移计划](./docs/migration-plan.md) - 从 KingPortal 迁移指南

### 🎨 设计中心文档
- [设计中心概述](./docs/designer/README.md)
- [数据绑定系统](./docs/designer/data-binding.md)
- [Canvas 渲染引擎](./docs/designer/canvas-engine.md)
- [组件开发指南](./docs/designer/component-development.md)

### 📊 数据中心文档
- [数据中心概述](./docs/datacenter/README.md)
- [数据连接管理](./docs/datacenter/connections.md)
- [查询管理](./docs/datacenter/queries.md)

### 🧩 数据域与运行时文档
- [数据域服务概述](./docs/data_service/README.md)
- [节点执行器概述](./docs/node_agent/README.md)
- [节点本地管理前端概述](./docs/node_agent_front/README.md)

### 🔧 后端文档
- [后端 API 文档](./docs/backend/README.md)
- [认证与授权](./docs/backend/auth.md)
- [数据库初始化](./docs/backend/database-init.md)

## 开发指南

### 代码规范

- 使用 ESLint 进行代码检查
- 使用 Prettier 进行代码格式化
- 遵循 Vue 3 Composition API 风格
- 组件命名使用 PascalCase
- 文件命名使用 kebab-case

### Git 提交规范

```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式调整
refactor: 重构
test: 测试相关
chore: 构建/工具链相关
```

### 分支管理

- `main` - 主分支，稳定版本
- `develop` - 开发分支
- `feature/*` - 功能分支
- `hotfix/*` - 紧急修复分支

## 部署指南

### 生产环境构建

```powershell
# 构建所有前端应用
Set-Location .\dev_ide
pnpm build

Set-Location ..\datacenter
pnpm build

Set-Location ..\designer
pnpm build

Set-Location ..\runtime\node_agent_front
pnpm build

Set-Location ..\..
```

### Nginx 配置

参考 `scripts/nginx/nginx.conf` 配置文件，主要配置：
- 静态资源路径
- API 代理
- 域名和端口

### 端口迁移提醒

如果你的本地环境还在使用历史端口，需要同步更新根目录 `.env`。仓库文档、模板和源码中的默认端口已经统一切换到 `176xx / 186xx / 196xx` 新号段。

### 后端部署

```powershell
Set-Location .\dev_core
pnpm start
```

建议使用 PM2 进行进程管理：
```powershell
pm2 start src/index.js --name induforge-api
```

## 常见问题

### 1. 数据库连接失败
检查 `.env` 文件中的数据库配置是否正确。

### 2. 端口被占用
修改各模块的 `vite.config.js` 或 `.env` 文件中的端口配置。

### 3. 跨域问题
确保 Nginx 配置了正确的 CORS 头，或在开发环境使用代理。

### 4. Windows 下脚本无法直接执行
优先使用 PowerShell 执行命令；基础设施脚本统一放在 `scripts/`，例如 `scripts/seaweedfs/start.bat`。

## 贡献指南

欢迎贡献代码和文档！请遵循以下步骤：

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'feat: Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 许可证

ISC

## 联系方式

如有问题或建议，请提交 Issue 或联系开发团队。

---

**版本**: 2.0.0  
**最后更新**: 2026-04-20
