# 技术栈

## 前端应用 (dev_ide, datacenter, designer)

- **框架**: Vue 3 with Composition API
- **构建工具**: Vite 7.x
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **UI 组件库**: Element Plus
- **样式**: Tailwind CSS + PostCSS
- **代码编辑器**: Monaco Editor
- **Canvas 渲染**: Konva.js (仅 designer)
- **图表**: ECharts (仅 designer)
- **动画**: GSAP (仅 designer)
- **HTTP 客户端**: Axios
- **包管理器**: pnpm

## 后端 (dev_core)

- **运行时**: Node.js >= 18.0.0
- **框架**: Express 5.x
- **ORM**: Sequelize 6.x
- **数据库驱动**: mysql2, pg, mssql
- **认证**: JWT (jsonwebtoken)
- **验证**: express-validator, Joi
- **日志**: Winston with daily-rotate-file
- **国际化**: i18next with fs-backend
- **安全**: helmet, cors, bcryptjs
- **限流**: express-rate-limit
- **缓存**: ioredis (Redis 客户端)
- **API 文档**: Swagger (swagger-jsdoc, swagger-ui-express)

## 测试

- **后端**: Jest with supertest
- **前端**: Vitest with @vue/test-utils, jsdom
- **属性测试**: fast-check

## 开发环境要求

- Node.js >= 18.0.0
- pnpm >= 8.0.0
- MySQL >= 5.7 或 PostgreSQL >= 12

## 常用命令

### 后端 (dev_core)
```bash
pnpm dev              # 启动开发服务器（nodemon）
pnpm start            # 启动生产服务器
pnpm db:init          # 初始化数据库
pnpm db:reset         # 重置并重新初始化数据库
pnpm test             # 运行测试
```

### 前端 (dev_ide, datacenter, designer)
```bash
pnpm dev              # 启动 Vite 开发服务器
pnpm build            # 构建生产版本
pnpm preview          # 预览生产构建
pnpm lint             # 运行 ESLint
pnpm format           # 使用 Prettier 格式化代码
```

### 仅 Designer
```bash
pnpm test             # 运行 Vitest 测试
pnpm test:watch       # 监听模式运行测试
```

## 默认端口

- dev_core: 9099
- dev_ide: 9091
- datacenter: 9092
- designer: 9093

## 代码风格

- 使用 ESLint 和 Vue 插件
- 使用 Prettier 格式化
- 优先使用 Vue 3 Composition API 风格
- 组件名使用 PascalCase
- 文件名使用 kebab-case
