# 公共契约

本目录存放需要由多个模块共同消费的机器可读契约，作为仓库内唯一源码。

- `collector-protocols/`：工业采集驱动 Manifest、连接 Schema、地址 Schema 和 UI Schema。
- `project-templates/catalog.json`：工程模板机器可读目录。
- `project-templates/vite-vue-js/`、`vite-vue-ts/`、`vite-react-js/`、`vite-react-ts/`：由固定版本
  `create-vite` 生成的四套官方模板，模板页面不包含平台示例业务代码；模板统一声明 Node.js
  `24.19.0` LTS、npm `11.17.0` 和 pnpm `11.21.0` 工具链约束。

`contracts/` 是这些内容的唯一源码。模块开发时从仓库根目录读取，镜像构建时可将所需目录复制到镜像固定位置，但不得在模块内部维护第二份副本。
