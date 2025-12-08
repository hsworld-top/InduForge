# InduForge - 工业应用低代码开发平台

## 项目简介

InduForge 是一个基于微前端架构的工业应用低代码开发平台，提供可视化设计、数据管理和工程管理等功能，专为工业互联网（IIoT）和 SCADA 应用场景设计。

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
┌──────────────┬──────────────┬──────────────┬──────────────┐
│   IDE 主应用  │  数据中心     │  设计中心     │  后端服务     │
│   (dev_ide)  │ (datacenter) │  (designer)  │  (dev_core)  │
│              │              │              │              │
│  - 工程管理   │  - 数据连接   │  - 页面设计   │  - 用户认证   │
│  - 用户管理   │  - SQL查询   │  - 组件库     │  - 租户管理   │
│  - 租户管理   │  - 数据预览   │  - 数据绑定   │  - 数据API   │
│              │              │              │              │
│  Vue 3       │  Vue 3       │  Vue 3       │  Node.js     │
│  Port: 9091  │  Port: 9092  │  Port: 9093  │  Port: 9099  │
└──────────────┴──────────────┴──────────────┴──────────────┘
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

## 快速开始

### 环境要求

- Node.js >= 18.0.0
- pnpm >= 8.0.0
- MySQL >= 5.7 或 PostgreSQL >= 12

### 安装依赖

```bash
# 安装所有模块依赖
cd InduForge

# 后端
cd dev_core && pnpm install

# IDE
cd ../dev_ide && pnpm install

# 数据中心
cd ../datacenter && pnpm install

# 设计中心
cd ../designer && pnpm install
```

### 初始化数据库

```bash
cd dev_core
cp .env.example .env
# 编辑 .env 配置数据库连接
pnpm run db:init
```

### 启动开发服务器

```bash
# 1. 启动后端服务
cd dev_core
pnpm dev

# 2. 启动 IDE（新终端）
cd dev_ide
pnpm dev

# 3. 启动数据中心（新终端）
cd datacenter
pnpm dev

# 4. 启动设计中心（新终端）
cd designer
pnpm dev
```

访问地址：
- IDE: http://localhost:9091
- 数据中心: http://localhost:9092
- 设计中心: http://localhost:9093

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
├── nginx/              # Nginx 配置
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

```bash
# 构建所有前端应用
cd dev_ide && pnpm build
cd ../datacenter && pnpm build
cd ../designer && pnpm build
```

### Nginx 配置

参考 `nginx/nginx.conf` 配置文件，主要配置：
- 静态资源路径
- API 代理
- 域名和端口

### 后端部署

```bash
cd dev_core
pnpm start
```

建议使用 PM2 进行进程管理：
```bash
pm2 start src/index.js --name induforge-api
```

## 常见问题

### 1. 数据库连接失败
检查 `.env` 文件中的数据库配置是否正确。

### 2. 端口被占用
修改各模块的 `vite.config.js` 或 `.env` 文件中的端口配置。

### 3. 跨域问题
确保 Nginx 配置了正确的 CORS 头，或在开发环境使用代理。

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
**最后更新**: 2025-12-08
