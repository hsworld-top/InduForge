# 中心系统 K3s 部署

正式用户环境把中心系统服务部署到全局 K3s 集群的 `induforge-system` 命名空间。中心 Pod 使用
`induforge.io/center-node=true` 标签固定在中心服务器；运行环境继续使用独立的 `if-env-*` 命名空间，
两者不共享数据库、缓存、对象数据或凭据。

当前单中心版本使用安装时确定的 `IF_CENTER_DATA_ROOT` 宿主目录保存中心 PostgreSQL、Redis、
SeaweedFS、代码工作区、IDE 静态资源和 TLS 文件。目录不在界面中逐服务配置。多中心 HA 需要重新
设计数据库和存储拓扑，不能把本清单直接扩为多个副本。

节点安装包需要预载清单引用的基础镜像和不可变 `induforge/control:<version>` 镜像。安装器准备好
中心数据目录、`/etc/induforge/center.env`、IDE 与 TLS 资产后执行：

```bash
sudo centerctl import-images /path/to/center-images
```

```bash
sudo IF_CENTER_NODE_NAME=<k3s-node-name> \
  IF_CENTER_CONTROL_IMAGE=induforge/control:<version> \
  IF_CENTER_EDGE_IMAGE=induforge/edge:<version> \
  IF_CENTER_DATA_ROOT=/var/lib/induforge/center \
  ./centerctl apply /etc/induforge/center.env
```

旧版 Docker/宿主进程中心可在确认镜像已经导入 K3s 后执行一次性迁移脚本。脚本先保存 PostgreSQL
逻辑备份和中心资产归档，再停止旧服务、复制固定数据目录并部署 K3s 工作负载；失败会缩容新工作负载
并尝试恢复旧服务。成功后不会删除旧 Docker Volume，需在人工验收和备份确认后另行清理。
迁移同时把中心证书加入中心服务器系统信任，并将内置 NodeAgent 切换到中心 HTTPS 入口。

```bash
sudo IF_CENTER_NODE_NAME=<k3s-node-name> \
  IF_CENTER_CONTROL_IMAGE=induforge/control:<version> \
  IF_CENTER_EDGE_IMAGE=induforge/edge:<version> \
  IF_CENTER_AGENT_SERVER_URL=https://<center-ip>:18443 \
  ./migrate-docker-center.sh
```

常用诊断命令：

```bash
sudo centerctl status
sudo centerctl doctor
sudo centerctl logs center-control
sudo centerctl restart center-control
```

安装流程会将节点名、镜像版本、数据根目录和入口端口写入 root-only 的
`/etc/induforge/center-k3s.conf`，因此上述诊断命令不依赖当前 shell 的环境变量。

`center-control` 暂时只为现有 Docker 代码工作区挂载宿主 Docker Socket。中心数据库、缓存、对象
存储、控制面和 IDE 入口均由 K3s 管理；代码工作区改为 K3s 调度后必须删除该兼容挂载和 Docker
依赖。
