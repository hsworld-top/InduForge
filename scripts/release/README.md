# 测试打包说明

Linux 测试人员执行：

```bash
./scripts/release/build-offline-package-linux.sh
```

## 前置条件

- Docker 已安装并启动。
- 当前机器能访问镜像源，或者 `scripts/docker/images/` 已经放好所需镜像 tar。

脚本不负责安装 Docker。

## 镜像策略

默认镜像缓存目录：

```text
scripts/docker/images/
```

规则：

- 有 tar 文件时优先加载。
- 本地已有镜像时不重复拉取。
- 基础镜像缺失时自动拉取。
- 业务镜像从当前源码构建。
- 产品体系基础设施镜像从 `scripts/docker/infra/` 构建。
- 安装包内使用 `induforge/*` 产品体系镜像名。
- 镜像 tar 会保存回 `scripts/docker/images/`，但不应提交 Git。

## 输出

```text
dist/induforge-offline-package.tar.gz
```

## 验证安装包

解包后执行：

```bash
cp .env.production.example .env
vi .env
./install.sh
```

卸载保留数据：

```bash
./uninstall.sh
```

卸载并删除数据：

```bash
./uninstall.sh --volumes
```
