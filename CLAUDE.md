# InduForge 开发规则

## 概述

- 本文件是仓库唯一公共规则正文入口。
- 处理任务时先读本文件，再读离改动目录最近的 `AGENTS.md`。
- 若改动位于 `runtime/node_agent/` 或 `runtime/node_agent_front/`，先读 `runtime/AGENTS.md`，再读对应子模块规则。
- 默认按模块分别处理，不把整仓当成单一应用统一修改。

## 工作方式

- 使用中文沟通，新增或修改注释优先使用中文。
- 当前开发环境以 Windows 为主，命令优先使用 PowerShell、`cmd`、模块自带脚本或 `Makefile`。
- 遇到 `rg`、`grep`、`sed` 等 Unix 风格命令不可用或不稳定时，直接改用 PowerShell 原生命令。
- 默认只改与当前任务直接相关的文件，不顺手扩散重构无关模块。
- 涉及多个模块时，按真实影响范围联动修改，并在汇报中说明改动边界。
- 需求不清或方案存在分歧时，优先讨论澄清，确认后再开始实现。
- 完成改动后，先执行最接近的验证命令，再汇报结果、未验证项和剩余风险。
- 开发时不创建 worktree，可以创建 branch，完成任务后合并 master 分支，清理本地 branch。
- 所有的开发不需要考虑兼容历史，选择最优的方式即可。
- 代码带有必要注释，但不过度。

## 通用工程约束

- 默认工作目录为仓库根目录，统一使用根目录 `.env`。
- 端口配置以根目录 `.env` 为本机实际值来源；`.env_example` 与 `README.md` 只维护统一默认端口。
- Node 相关模块统一使用 `pnpm`；Go 模块遵循各自 `Makefile` 或 `go` 命令。
- 后端 API 前缀统一为 `/api/v1`，健康检查为 `/health`。
- 对外 JSON 接口响应统一使用 `code`、`msg`、`data`、`reqId` 四字段；`code = 0` 是唯一成功码。
- 业务成功失败统一看业务码：`HTTP 2xx + code = 0` 表示成功，`HTTP 2xx + code != 0` 表示业务失败。
- 分页接口统一将分页信息放入 `data.pagination`，列表主体放入 `data.list`。
- 新增错误码必须先复用现有统一错误码规则；若确需新增，必须同步更新 `docs/统一错误码枚举表.md`。
- `dev_core`、`dev_ide`、`datacenter`、`designer` 统一使用 `dayjs`，字符串时间格式为 `YYYY-MM-DD HH:mm:ss`。
- `dev_core` 数据库初始化统一使用 `dev_core/scripts/init-database.js`。
- 数据库操作必须使用参数化查询。
- commit message 使用中文，推荐格式为 `type(scope): 中文描述`。

## 目录路由

- `dev_core/`：平台控制面后端与发布部署聚合能力。
- `data_service/`：平台侧与开发态数据域服务。
- `dev_ide/`：平台管理与运维前端。
- `datacenter/`：数据接入与数据语义建模前端。
- `designer/`：低代码页面设计器。
- `runtime/`：运行时相关共享规则入口。
- `runtime/node_agent/`：节点执行器后端。
- `runtime/node_agent_front/`：节点本地管理前端。
- `scripts/`：本地开发与基础设施脚本目录。

## 文档治理

- 正式业务文档统一放在 `docs/`。
- Superpower 过程文档统一放在根目录 `.superpower/`。
- `.superpower/` 默认不提交到 Git。
- 新增正式文档优先使用中文文件名。

## 非目标目录

- `scripts/docker/`、`scripts/nginx/`、`scripts/seaweedfs/` 默认不作为开发目标，除非需求明确点名。