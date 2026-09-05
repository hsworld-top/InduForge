# InduForge

InduForge 是私有化部署优先的工业应用开发与交付平台，面向为各行业工厂提供应用的集成商，也支持工厂购买整个平台自行开发。

产品提供两种交付方式：交付带独立用户体系的工程应用，或交付完整开发平台。它们是产品目标；当前实现与交付缺口见 [产品定义](docs/01-产品与架构/产品定义.md) 和 [平台系统架构](docs/01-产品与架构/平台系统架构.md)。

## 开发入口

在仓库根目录安装 Node 依赖，工具链版本以 [package.json](package.json) 为准；Go 版本见各模块 go.mod，Collector 使用 .NET 工程。

```sh
pnpm install
```

当前默认前端开发模式连接 Linux 中心与工程工作区，配置和启动步骤见 [开发脚本说明](scripts/dev/README.md)。根 pnpm dev:frontend 会启动前端组合和工作区代理；显式本地后端模式使用 dev:frontend:local 等对应入口。不要混用两套环境与会话配置。

检查按改动模块选择，例如：

```sh
pnpm typecheck:ide
pnpm test:designer
go -C dev_core test ./internal/project
```

默认 build、test、lint、typecheck 是部分模块的聚合入口，不代表覆盖整仓；.NET、安装脚本和目标系统验收需单独选择。端口与暴露边界见 [环境端口规划](docs/06-运维与安全/环境端口规划.md)，不在此复制一份端口表。

## 当前源码导航

| 路径                                                                   | 职责                                |
| ---------------------------------------------------------------------- | ----------------------------------- |
| dev_core / dev_ide                                                     | 控制面与管理工作台                  |
| data_service / datacenter                                              | 数据建模、开发调试与数据工作台      |
| designer / designer/code-workspace                                     | 应用创作工作台与受控代码工作区      |
| collector                                                              | .NET 驱动、开发调试代理与生产采集器 |
| runtime/node_agent / runtime/node_agent_front                          | 节点管理与本机控制台                |
| runtime/project_gateway / runtime/runtime_api / runtime/runtime_engine | 工程入口、运行 API 与数据执行       |
| compute_sandbox                                                        | 隔离计算执行                        |
| runtime/web-sdk / contracts                                            | 工程 SDK、机器契约和模板            |
| scripts                                                                | 开发、交付与验证脚本                |
| docs                                                                   | 有效产品、架构、契约与规范          |

生产采集统一使用 collector 中的 .NET Runtime，由 NodeAgent 托管原生进程；驱动支持和目标平台交付分别验收。designer/pi-web 是上游项目，保持原位。目标目录映射和模块开发要求见 [仓库结构与模块规范](docs/05-研发与交付/仓库结构与模块规范.md)，源码尚未按目标目录迁移。

## 交付与维护

中心已有 [K3s 部署入口](scripts/k3s/center/README.md)，工作区与 Release 构建仍依赖 Docker。scripts/offline 中仍有整个平台的 Compose 安装流程；这些入口尚未收敛为一致的客户交付清单，不能将其视为独立工程应用离线包。

工程用户交付、生产采集安全配置、离线冷安装、升级与备份恢复必须分别验收。静态检查或制品生成成功不代表这些链路已交付。

- [文档中心](docs/README.md)
- [开发规范](docs/05-研发与交付/开发规范.md)
- [仓库协作规则](AGENTS.md)
- [视觉与体验](PRODUCT.md)
- [脚本入口](scripts/README.md)

历史版本和过程记录通过 Git 与任务对话查询，不维护文档归档。项目为专有闭源软件，授权边界见 [LICENSE](LICENSE)；第三方组件保留各自许可与来源。
