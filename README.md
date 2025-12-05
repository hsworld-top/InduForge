# InduForge - 工业应用开发平台

InduForge 是一个基于微前端架构的工业应用开发平台，提供工程管理、数据管理和可视化设计等功能。

## 项目架构

项目采用 **Nginx 同域名代理 + 独立 SPA + 统一 Node 后端** 的架构设计：

- **前端应用**：三个独立的 Vue3 SPA 应用，通过 Nginx 统一代理
- **后端服务**：统一的 Node.js/Express API 服务
- **数据交互**：通过 LocalStorage 共享会话信息，URL 参数传递上下文信息

## 服务说明

### 前端服务

#### 1. IDE (`/dev_ide`)

- **描述**：主入口应用，提供工程管理、用户管理、租户管理等核心功能
- **技术栈**：Vue 3 + Vue Router + Pinia + Element Plus + Tailwind CSS
- **端口**：开发环境 9091（可通过环境变量配置）
- **路径**：`/`（根路径）

#### 2. 数据中心 (`/datacenter`)

- **描述**：数据连接管理、SQL 查询、数据预览等功能
- **技术栈**：Vue 3 + Monaco Editor + Element Plus
- **端口**：开发环境 9092（可通过环境变量配置）
- **路径**：`/datacenter/`
- **特性**：支持 MySQL、PostgreSQL、SQL Server 等多种数据库

#### 3. 设计中心 (`/designer`)

- **描述**：可视化页面设计工具，支持拖拽式组件配置
- **技术栈**：Vue 3 + Vue Router + Element Plus
- **端口**：开发环境 9093（可通过环境变量配置）
- **路径**：`/designer/`
- **特性**：通过 iframe 嵌入第三方设计器

### 后端服务

#### 4. Dev Core (`/dev_core`)

- **描述**：统一的 Node.js 后端 API 服务
- **技术栈**：Express + Sequelize + JWT + i18n
- **端口**：9099（可通过环境变量配置）
- **功能**：
  - 用户认证与授权
  - 租户管理
  - 工程管理
  - 数据连接管理
  - 数据查询执行
  - 系统日志

### 基础设施

#### 5. Nginx (`/nginx`)

- **描述**：反向代理服务器，统一代理前端应用和后端 API
- **配置**：`nginx/nginx.conf`
- **功能**：
  - 静态资源服务
  - API 代理
  - WebSocket 支持
  - Gzip 压缩
  - 安全头设置

## 快速开始

### 环境要求

- Node.js >= 18.0.0
- pnpm >= 8.0.0
- Nginx (生产环境)

### 开发环境启动

```bash
# 1. 安装依赖
cd dev_ide && pnpm install
cd ../datacenter && pnpm install
cd ../designer && pnpm install
cd ../dev_core && pnpm install

# 2. 启动后端服务
cd dev_core
pnpm dev

# 3. 启动前端服务（分别在不同终端）
cd dev_ide && pnpm dev      # http://localhost:9091
cd ../datacenter && pnpm dev   # http://localhost:9092
cd ../designer && pnpm dev     # http://localhost:9093
```

### 生产环境部署

1. 构建前端应用：

```bash
cd dev_ide && pnpm build
cd ../datacenter && pnpm build
cd ../designer && pnpm build
```

2. 配置 Nginx：

   - 将构建产物复制到 `/var/www/ide`、`/var/www/datacenter`、`/var/www/designer`
   - 复制 `nginx/nginx.conf` 到 Nginx 配置目录
   - 修改 `server_name` 为实际域名
   - 重启 Nginx

3. 启动后端服务：

```bash
cd dev_core
pnpm start
```

## 信息交互机制

### 会话级信息（LocalStorage）

- Token、用户信息、租户 ID 等全局共享信息
- 通过 LocalStorage 在同源应用间共享
- 键名统一：`auth_token`、`user_info`、`tenant_id`

### 上下文级信息（URL 参数）

- 工程 ID (`pid`)、页面 ID (`pageid`) 等上下文信息
- 通过 URL Query 参数传递
- 支持多 Tab 并行编辑不同工程

## 项目结构

```
InduForge/
├── dev_ide/          # IDE 主应用
├── datacenter/       # 数据中心应用
├── designer/         # 设计中心应用
├── dev_core/         # 后端 API 服务
└── nginx/            # Nginx 配置文件
```

## 许可证

ISC
