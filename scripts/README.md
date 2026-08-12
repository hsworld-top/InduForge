# scripts 目录说明

本目录只放本地开发、基础设施和交付打包脚本。

## 开发人员入口

Linux 开发环境初始化：

```bash
./scripts/dev/init-linux.sh
```

该脚本会：

- 检查 Docker。
- 如果根目录没有 `.env`，从 `.env.development.example` 生成。
- 启动开发基础设施容器。
- 创建 `if_core`、`if_data`、`if_dev_data`。
- 为 `if_dev_data` 启用开发态时序扩展。

默认不会安装 Node 依赖，也不会启动业务项目。Node 依赖由根目录 pnpm workspace 统一管理，只在仓库根目录执行 `pnpm install`，不要在子项目目录单独安装依赖。pnpm 可能会在 workspace 子目录生成 `node_modules/` 链接或提升目录，这属于安装产物；Windows + WSL2 共用工作区时，仍建议在 Windows 侧安装 Node 依赖，避免 Linux 侧生成的二进制依赖影响 Windows 开发。`dev_core` 在 `DB_AUTO_SCHEMA_SYNC=true` 时会在自身启动阶段同步 `if_core` 表结构和默认租户/管理员数据。

该脚本不会启动业务项目。开发人员需要自行启动：

- `dev_core`
- `data_service`
- `dev_ide`
- `datacenter`
- `designer`

## 测试人员入口

Linux 离线安装包构建：

```bash
./scripts/release/build-offline-package-linux.sh
```

该脚本会：

- 检查 Docker。
- 优先加载 `scripts/docker/images/*.tar` 中已有镜像。
- 本地缺镜像时自动拉取基础镜像。
- 构建 InduForge 业务镜像。
- 构建 InduForge 产品体系基础设施镜像。
- 保存镜像到 `scripts/docker/images/`。
- 生成 `dist/induforge-offline-package.tar.gz`。

镜像 tar 不建议提交 Git。它们体积较大，正式交付应通过安装包、制品库、内网文件服务器或 Release 附件分发。

## Docker 目录

```text
scripts/docker/
  docker-compose.dev.yml       # 开发基础设施
  docker-compose.prod.yml      # 生产源码构建拓扑
  docker-compose.offline.yml   # 离线安装包拓扑
  edge/                        # edge/nginx 镜像构建
  infra/                       # 产品体系基础设施镜像 wrapper
  images/                      # 本地镜像缓存，不建议提交 Git
```

`scripts/docker/images/` 只保留目录和 `.gitkeep`，镜像 tar 体积较大，不提交 Git。测试构建脚本会优先加载该目录下已有 tar，缺失时再拉取或构建。

## 控制面数据库 bootstrap

```text
dev_core/db/schema/core-schema.sql
  init-core-database.js        # 安装阶段和开发启动时复用的控制面数据库 bootstrap
  sql/core-schema.sql          # if_core 表结构
```

开发人员不需要手动执行 `db:init` 或 `db:reset`。生产和离线环境保持 `DB_AUTO_SCHEMA_SYNC=false`，由安装脚本在容器内执行 bootstrap。

## 离线包目录

```text
scripts/offline/
  build-package.sh             # 从本机镜像生成离线包
  install.sh                   # Linux 目标机安装入口
  uninstall.sh                 # Linux 目标机卸载入口
  *.ps1                        # Windows 目标机入口
```

## 模拟数据测试脚本

```text
scripts/test/
  industrial-sim/                # Modbus/OPC UA/S7 工业协议模拟设备（Python）
  mqtt/                          # MQTT 模拟数据发布脚本（Node）
    mqtt-publish-test.js
    README.md
```

`scripts/test/` 专门放模拟数据、协议接入和本地联调类脚本，不放自动化单元测试。
