# 开发环境初始化

Linux 开发人员执行：

```bash
./scripts/dev/init-linux.sh
```

## 前置条件

- Docker 已安装并启动。
- Docker Compose 可用。
- Node 已安装。
- pnpm 已安装。

脚本不负责安装这些工具。

## 执行内容

- 生成根目录 `.env`，如果它不存在。
- 启动 `scripts/docker/docker-compose.dev.yml` 中的基础设施容器。
- 初始化平台数据库。
- 启用开发态时序扩展。
- 执行 `dev_core` 数据库初始化。

## 内部脚本

- `init-linux.sh`：开发人员入口。
- `init-infra.mjs`：开发初始化内部实现，负责创建数据库和检查基础设施端口。
- `init-infra.ps1`：Windows 开发初始化备用入口。

## 不执行内容

脚本不会启动任何业务项目。

开发人员需要自行启动后端和前端模块。
