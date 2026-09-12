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
  IF_CENTER_DATA_IMAGE=induforge/data:<version> \
  IF_CENTER_EDGE_IMAGE=induforge/edge:<version> \
  IF_CENTER_DATA_ROOT=/var/lib/induforge/center \
  /opt/induforge/center-k3s/centerctl apply /etc/induforge/center.env
```

`apply` 只在全部中心工作负载和服务端点健康后，将本次解析后的镜像、节点、
数据目录与入口参数原子写入 `/etc/induforge/center-k3s.conf`，后续不带环境变量的
普通 `apply` 会继续使用已验证的版本。

`apply` 在中心数据库和数据服务可用后，会重建一个幂等的一次性
`center-control-bootstrap` Job。Job 严格校验控制面数据库名，仅在数据库不存在时创建，随后使用
当前 control 镜像的 `init` 命令初始化空库或校验现有结构；成功后才继续等待正式控制服务健康。
初始化期间正式控制服务保持为零副本，Job 成功后安装器才放行为一个副本。
已有旧结构不会被迁移或修补，而会以明确的不兼容错误终止安装。数据域服务同样只为空库创建最终
基线，已有结构缺少必需表或字段时拒绝启动。

旧版 Docker/宿主进程中心可在确认镜像已经导入 K3s 后执行一次性迁移脚本。脚本先保存 PostgreSQL
逻辑备份和中心资产归档，再停止旧服务、复制固定数据目录并部署 K3s 工作负载；失败会缩容新工作负载
并尝试恢复旧服务。成功后不会删除旧 Docker Volume，需在人工验收和备份确认后另行清理。
迁移会备份并停用旧版中心 NodeAgent。新版中心直接把安装时已有的 K3s Server 映射为中心内置节点，
基础服务和工程由中心控制面调和；额外服务器仍通过各自的 NodeAgent 接入。

```bash
sudo IF_CENTER_NODE_NAME=<k3s-node-name> \
  IF_CENTER_CONTROL_IMAGE=induforge/control:<version> \
  IF_CENTER_EDGE_IMAGE=induforge/edge:<version> \
  ./migrate-docker-center.sh
```

常用诊断命令：

```bash
sudo induforge status
sudo induforge doctor
sudo induforge logs control
sudo induforge restart control
```

`induforge status` 只展示中心开发系统的 IDE 入口、控制服务、中心数据库、缓存、消息库和
对象存储；物理节点、运行环境、基础服务分布和时间同步统一在 Web 运维管理中查看。
交互终端会显示 IF Logo 和状态颜色；管道、重定向、`NO_COLOR=1` 或哑终端自动使用纯文本。
自动化可使用 `sudo induforge status --json` 和 `sudo induforge doctor --json`。

安装流程会将节点名、镜像版本、数据根目录和入口端口写入 root-only 的
`/etc/induforge/center-k3s.conf`，因此上述诊断命令不依赖当前 shell 的环境变量。

`center-data` 作为独立 Deployment 通过 `center-data:18102` ClusterIP Service 向控制面提供数据域与
正式 Release 工件，不依赖宿主端口。`center-control` 暂时只为现有 Docker 代码工作区挂载宿主 Docker Socket。中心数据库、缓存、对象
存储、控制面、数据域和 IDE 入口均由 K3s 管理；代码工作区改为 K3s 调度后必须删除该兼容挂载和 Docker
依赖。

正式 Release Builder 默认使用独立的受控离线构建镜像
`induforge/release-builder:1.0.0-node24-pnpm11.21.0`，它只包含 Node、Corepack 与
审核后的 pnpm 离线 store，不复用包含 IDE 和 code-server 的开发镜像。它使用 Docker local bind volume
`induforge-center-workspaces`。`centerctl apply` 会核验该卷必须精确绑定
`$IF_CENTER_DATA_ROOT/workspaces`，已有不同配置的卷会失败，不会覆盖。它还会在首次安装时于
`$IF_CENTER_DATA_ROOT/secrets/release-signing-key.pem` 创建 PKCS#8 Ed25519 私钥（目录 `0700`、文件
`0600`），并同步为命名空间 Secret。请把该私钥纳入离线安全备份：丢失后历史 Release 的签名信任
链无法延续，必须经过显式密钥轮换和公钥分发流程；公钥分发将在后续节点发布阶段实现。私钥绝不能写入
`center.env`、配置文件或命令输出。
