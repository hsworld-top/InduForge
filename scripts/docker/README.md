# Docker 脚本目录

## Compose 文件

- `docker-compose.dev.yml`：开发基础设施，只启动数据库、缓存、消息、对象存储等依赖。
- `docker-compose.prod.yml`：生产源码构建拓扑，适合构建机或演示环境。
- `docker-compose.offline.yml`：离线安装包拓扑，只引用 `induforge/*` 产品体系镜像。

## 镜像缓存

默认镜像缓存目录：

```text
scripts/docker/images/
```

该目录用于测试打包和离线交付，不建议提交 Git。

## 产品体系基础设施镜像

`infra/` 下的 Dockerfile 会把底层基础设施镜像包装成 InduForge 产品镜像：

```text
induforge/meta-store:latest
induforge/cache-store:latest
induforge/message-hub:latest
induforge/object-store:latest
```

离线安装包中的 compose 文件只引用这些产品体系镜像名。
