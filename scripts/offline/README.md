# InduForge 离线交付包说明

本交付包面向已经安装 Docker 环境的空白服务器或 Windows Docker 环境。

安装脚本只做环境检测、镜像导入和 Compose 启动，不负责安装 Docker。

离线包默认只对外暴露 edge/Nginx 入口端口，内部数据库、缓存、消息中心和对象存储通过 Docker 网络内的服务名通信。

## Linux

```bash
cp .env.production.example .env
vi .env
./install.sh
```

卸载但保留数据卷：

```bash
./uninstall.sh
```

卸载并删除数据卷：

```bash
./uninstall.sh --volumes
```

## Windows

```powershell
Copy-Item .env.production.example .env
notepad .env
.\install.ps1
```

卸载但保留数据卷：

```powershell
.\uninstall.ps1
```

卸载并删除数据卷：

```powershell
.\uninstall.ps1 -RemoveVolumes
```

## 必须确认

- Docker Engine 或 Docker Desktop 已安装。
- Docker Compose v2 插件或 `docker-compose` 已安装。
- `.env` 中所有 `change_me_*` 已替换为生产密钥。
- 服务器端口 `IF_EDGE_HOST_PORT` 没有被占用。
- `DB_AUTO_SCHEMA_SYNC=false` 保持生产默认值；控制面数据库结构由安装脚本在安装阶段初始化。

## 启动命令

安装脚本内部执行的核心命令如下：

```bash
docker compose --env-file .env -f scripts/docker/docker-compose.offline.yml up -d
```

## 镜像目录

离线镜像位于：

```text
scripts/docker/images/
```

安装脚本会逐个执行 `docker load -i`。

镜像文件使用 `induforge/*` 产品体系镜像名导出，避免离线 compose 直接依赖底层基础设施镜像名。

## 初始化内容

安装脚本会在容器启动后执行：

- 创建 `if_core`、`if_data`、`if_dev_data`。
- 为 `if_dev_data` 启用时序扩展。
- 在 control 容器内运行 `node scripts/bootstrap/init-core-database.js init`，初始化 `if_core` 表结构和默认管理员数据。

目标机器不需要额外安装 Node、pnpm 或 Go。

## 卸载说明

默认卸载只停止并删除 InduForge 容器和网络，不删除数据卷。

只有明确传入 `--volumes` 或 `-RemoveVolumes` 时，才会删除平台数据卷。该操作会删除数据库、缓存、消息中心和对象存储数据，正式环境请先备份。
