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
sudo /opt/induforge/center-k3s/centerctl import-images /path/to/center-images
```

```bash
sudo IF_CENTER_NODE_NAME=<k3s-node-name> \
  IF_CENTER_CONTROL_IMAGE=induforge/control:<version> \
  IF_CENTER_EDGE_IMAGE=induforge/edge:<version> \
  IF_CENTER_DATA_ROOT=/var/lib/induforge/center \
  /opt/induforge/center-k3s/centerctl apply /etc/induforge/center.env
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
sudo induforge status
sudo induforge doctor
sudo induforge logs control
sudo induforge restart control
```

`induforge status` 只展示中心开发系统的 IDE 入口、控制服务、中心数据库、缓存和
对象存储；物理节点、运行环境、基础服务分布和时间同步统一在 Web 运维管理中查看。
交互终端会显示 IF Logo 和状态颜色；管道、重定向、`NO_COLOR=1` 或哑终端自动使用纯文本。
自动化可使用 `sudo induforge status --json` 和 `sudo induforge doctor --json`。

安装流程会将节点名、镜像版本、数据根目录和入口端口写入 root-only 的
`/etc/induforge/center-k3s.conf`，因此上述诊断命令不依赖当前 shell 的环境变量。

`center-control` 暂时只为现有 Docker 代码工作区挂载宿主 Docker Socket。中心数据库、缓存、对象
存储、控制面和 IDE 入口均由 K3s 管理；代码工作区改为 K3s 调度后必须删除该兼容挂载和 Docker
依赖。

正式 Release Builder 默认使用受控的离线前端构建镜像和 Docker local bind volume
`induforge-center-workspaces`。`centerctl apply` 会核验该卷必须精确绑定
`$IF_CENTER_DATA_ROOT/workspaces`，已有不同配置的卷会失败，不会覆盖。它还会在首次安装时于
`$IF_CENTER_DATA_ROOT/secrets/release-signing-key.pem` 创建 PKCS#8 Ed25519 私钥（目录 `0700`、文件
`0600`），并同步为命名空间 Secret。请把该私钥纳入离线安全备份：丢失后历史 Release 的签名信任
链无法延续，必须经过显式密钥轮换和公钥分发流程；公钥分发将在后续节点发布阶段实现。私钥绝不能写入
`center.env`、配置文件或命令输出。
