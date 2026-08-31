# Project Gateway

Project Gateway 是单个已发布工程在物理节点上的 HTTP 入口。它只做三件事：

- 从 NodeAgent 已验签并原子激活的 Release `client/` 目录托管静态文件；
- 将 `/api/v1/*` 与 `/ws/*` 转发给同一节点回环地址上的 Runtime API；
- 暴露入口自身的 `/health` 和聚合 `/api/v1/status`。

它不会下载 Release、解释工程源码、连接中心对象存储或直接访问工业协议。Runtime API 被强制配置为本机回环 URL，避免浏览器绕过工程入口。

```sh
project-gateway \
  --listen 0.0.0.0:17800 \
  --client-root /var/lib/induforge-node/releases/release-1/client \
  --runtime-api http://127.0.0.1:17801 \
  --deployment-id deployment-1 \
  --project-id 11111111-1111-4111-8111-111111111111 \
  --site-id site-1 \
  --node-id node-1 \
  --version release-1 \
  --execution-form native-linux
```

`client-root` 必须包含普通文件 `index.html`，并且整棵目录不能包含符号链接。NodeAgent 负责保证该目录来自不可变 Release 且运行时只读。缺失的带扩展名资源返回 404；只有前端路由回退到 `index.html`。

验证：`make build test vet`。
