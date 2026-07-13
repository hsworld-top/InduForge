# Docker 脚本目录

## Compose 文件

- `docker-compose.dev.yml`：开发基础设施，只启动数据库、缓存、MQTT 接入、JetStream 内部总线、对象存储等依赖。
- `docker-compose.prod.yml`：生产源码构建拓扑，适合构建机或演示环境。
- `docker-compose.offline.yml`：离线安装包拓扑，只引用 `induforge/*` 产品体系镜像。

开发 compose 只面向开发机本地调试，会把基础设施端口映射到宿主机 `18xxx` 段。生产和离线 compose 默认只暴露 edge/Nginx 入口，其他服务通过 Docker 内部服务名通信。

## 镜像缓存

默认镜像缓存目录：

```text
scripts/docker/images/
```

该目录用于测试打包和离线交付，不建议提交 Git。

测试构建脚本会优先加载该目录已有 tar；缺失时拉取基础镜像、构建业务镜像和产品体系基础设施镜像，再把交付需要的 `induforge/*` 镜像保存回该目录。

## 产品体系基础设施镜像

`infra/` 下的 Dockerfile 会把底层基础设施镜像包装成 InduForge 产品镜像：

```text
induforge/meta-store:latest
induforge/cache-store:latest
induforge/message-hub:latest
induforge/object-store:latest
```

离线安装包中的 compose 文件只引用这些产品体系镜像名。

产品体系镜像用于离线交付命名隔离，不表示把所有服务合并进一个镜像。当前保持每个服务一个容器，便于数据卷隔离、独立重启、日志排查和后续迁移到 Compose/Swarm/Kubernetes。
