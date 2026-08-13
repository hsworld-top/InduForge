# 公共契约

本目录存放需要由多个模块共同消费的机器可读契约，作为仓库内唯一源码。

- `collector-protocols/`：工业采集驱动 Manifest、连接 Schema、地址 Schema 和 UI Schema。
- `project-templates/vue-vite/`：AI 页面开发工程的 Vue 3 + Vite 公共模板，供 `dev_core` 和开发容器镜像共同使用。

`contracts/` 是这些内容的唯一源码。模块开发时从仓库根目录读取，镜像构建时可将所需目录复制到镜像固定位置，但不得在模块内部维护第二份副本。
