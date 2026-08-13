# 公共契约

本目录存放需要由多个模块共同消费的机器可读契约，作为仓库内唯一源码。

- `collector-protocols/`：工业采集驱动 Manifest、连接 Schema、地址 Schema 和 UI Schema。

开发环境中的 `data_service` 从仓库根目录定位这些契约。构建和发布阶段如何分发契约后续单独设计，本次不引入复制或打包流程。
