# InduForge 离线交付包说明

本交付包面向已经安装 Docker 环境的空白服务器或 Windows Docker 环境。

安装脚本只做环境检测、镜像导入和 Compose 启动，不负责安装 Docker。

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

## 卸载说明

默认卸载只停止并删除 InduForge 容器和网络，不删除数据卷。

只有明确传入 `--volumes` 或 `-RemoveVolumes` 时，才会删除平台数据卷。该操作会删除数据库、缓存、消息中心和对象存储数据，正式环境请先备份。
