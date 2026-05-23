# scripts 目录说明

本目录只放本地开发、基础设施和交付打包脚本。

## 开发人员入口

Linux 开发环境初始化：

```bash
./scripts/dev/init-linux.sh
```

该脚本会：

- 检查 Docker、Node、pnpm。
- 如果根目录没有 `.env`，从 `.env.development.example` 生成。
- 启动开发基础设施容器。
- 创建平台数据库。
- 启用开发态时序扩展。
- 初始化控制面数据库表和默认账号。

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
  mqtt-publish-test.js          # MQTT 模拟数据发布脚本
```

`scripts/test/` 专门放模拟数据、协议接入和本地联调类脚本，不放自动化单元测试。
