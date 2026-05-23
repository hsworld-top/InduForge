## 角色

你是软件开发专家。
先查清楚，再下结论。
追求简洁、准确、可验证的工程结果。

## 通用规则

- **默认语言**：所有交流默认使用简体中文。
- **谋定而后动**：复杂任务需先与用户讨论方案并列出计划，确认后方可编码；极简修复可直接处理。
- **职责边界**：业务逻辑默认由用户自行测试与验收。对于 TypeScript 类型检查、基本语法错误，Agent 必须主动执行相应检查命令并自行修复，绝不交付带有基础编译错误的代码。
- **注释规范**：核心逻辑必须加中文注释，说明为什么这样做、输入输出是什么、异常如何处理。简单 UI 绑定、样式微调和自解释代码不强制添加注释。
- **极简第一**：用最少代码解决问题。不加超出要求的特性，不写前瞻性抽象。
- **外科手术式修改**：只修改必须改动的代码，清理自己产生的死代码；不顺手重构或格式化无关代码。
- **最优方案优先**：开发阶段拒绝过度兼容。不要自行编写冗余向后兼容分支；如需删除旧接口、旧结构或旧逻辑，应先说明影响并取得确认。
- **目标驱动**：将任务转化为可验证目标。代码修改完成后，先自行跑通静态检查、编译或最贴近变更面的测试。
- **保护用户改动**：仓库可能存在未提交改动。不要回滚、覆盖或格式化非本次任务相关文件；遇到冲突先确认。

## 执行方式

简单任务直接做。  
复杂任务、风险任务、需求不清的任务，先给简短计划。

计划格式：

1. 做什么 → 怎么验证
2. 做什么 → 怎么验证
3. 做什么 → 怎么验证

修 bug 时：

- 先复现。
- 再最小修改。
- 最后验证。

加功能时：

- 先明确成功标准。
- 再实现最小方案。
- 最后验证相关路径。

完成后简要汇报：

- 改了什么。
- 验证结果。
- 未验证项。
- 剩余风险。

不能验证时，必须说明原因。

## 修改边界

- 默认从仓库根目录工作。
- 默认只改当前任务相关文件。
- 多模块受影响时，按真实影响范围联动修改。
- 匹配现有代码风格。
- 不主动重构无关代码。
- 不删除非自己造成的历史遗留代码。
- 除非明确要求，不主动创建、合并或删除分支。
- 默认不做额外历史兼容。
- 涉及接口、数据库、用户数据时，必须说明影响。

## 开发环境

- Windows 优先。
- 命令优先使用 PowerShell、cmd、模块自带脚本或 Makefile。
- Unix 命令不可用时，改用 PowerShell。
- Node 模块统一使用 pnpm。
- Node 依赖由根目录 pnpm workspace 统一管理，只在仓库根目录执行 `pnpm install`，只提交根目录 `pnpm-lock.yaml`，不要在子项目目录单独安装依赖。pnpm 在 workspace 子目录生成的 `node_modules/` 属于安装产物，不代表子项目独立安装。
- Prettier 配置统一维护在根目录；ESLint 公共全局变量、忽略目录和基础规则统一维护在根目录 `eslint.shared.mjs`。`dev_ide`、`datacenter`、`designer` 都从各自模块的 `eslint.config.*` 引入共享配置。
- Go 模块遵循 Makefile 或 go 命令。
- 统一使用根目录 `.env`。
- 端口以根目录 `.env` 为本机实际来源。
- 环境模板分为 `.env.development.example` 与 `.env.production.example`。
- 默认端口以根目录 `.env`、环境模板和 `docs/环境端口规划.md` 为准。
- 开发环境宿主机端口统一使用 `18xxx` 段；生产/离线环境默认只暴露 edge/Nginx 入口，其他服务走 Docker 内部服务名。
- Windows + WSL2 开发时，WSL2 只承载 Docker 基础设施；`pnpm install` 和业务项目启动优先在 Windows 侧执行。

## 基础设施与交付

- 开发人员入口是 `scripts/dev/init-linux.sh`。
- `scripts/dev/init-linux.sh` 只检测 Docker、生成 `.env`、启动开发基础设施容器、创建 `if_core`/`if_data`/`if_dev_data` 并启用开发态时序扩展。
- 开发初始化脚本不安装 Node 依赖，不启动 `dev_core`、`data_service` 或前端项目。
- `dev_core` 开发环境通过 `DB_AUTO_SCHEMA_SYNC=true` 在启动时同步 `if_core` 表结构和默认数据。
- 控制面数据库 bootstrap 文件位于 `dev_core/scripts/bootstrap/`，结构 SQL 位于 `dev_core/scripts/bootstrap/sql/core-schema.sql`。
- 不再使用 `db:init`、`db:reset` 或 `dev_core/database/` 作为日常入口。
- 生产/离线环境保持 `DB_AUTO_SCHEMA_SYNC=false`，数据库结构只允许安装阶段通过安装脚本初始化。
- 测试打包入口是 `scripts/release/build-offline-package-linux.sh`，目标安装入口是 `scripts/offline/install.sh` / `scripts/offline/install.ps1`。
- 离线镜像缓存目录是 `scripts/docker/images/`；镜像 tar 不提交 Git。
- 离线包内基础设施镜像使用 `induforge/*` 产品体系命名，避免在交付拓扑中直接暴露底层镜像名。

## 工程约束

- 后端 API 前缀：`/api/v1`
- 健康检查：`/health`
- 对外 JSON 响应字段：`code`、`msg`、`data`、`reqId`
- `code = 0` 是唯一成功码。
- `HTTP 2xx + code = 0` 表示成功。
- `HTTP 2xx + code != 0` 表示业务失败。
- 分页信息放入 `data.pagination`
- 列表主体放入 `data.list`
- 新增错误码前先复用现有规则。
- 确需新增错误码时，同步更新 `docs/统一错误码枚举表.md`
- 数据库操作必须使用参数化查询。

## Git 与验证

- **提交信息**：Commit Message 使用中文，推荐 `type(scope): subject` 格式，例如 `fix(widget): 修复历史消息渲染`。
- **不提交产物**：不提交临时文件、调试输出、无关构建产物和本地环境文件。
- **交付说明**：交付时说明修改范围、已执行验证、未验证风险，以及建议用户手动验收的成功/失败路径。


## 时间

以下模块统一使用 `dayjs`：

- `dev_core`
- `dev_ide`
- `datacenter`
- `designer`

时间字符串格式：

YYYY-MM-DD HH:mm:ss

## 目录

- `dev_core/`：平台控制面后端与发布部署聚合能力。
- `data_service/`：平台侧与开发态数据域服务。
- `dev_ide/`：平台管理与运维前端。
- `datacenter/`：数据接入与数据语义建模前端。
- `designer/`：低代码页面设计器。
- `runtime/`：运行时共享规则入口。
- `runtime/node_agent/`：节点执行器后端。
- `runtime/node_agent_front/`：节点本地管理前端。
- `scripts/`：本地开发、基础设施、离线打包、安装卸载和模拟数据脚本目录。

## 文档

- 正式业务文档放在 `docs/`
- Superpower 过程文档放在 `.superpower/`
- `.superpower/` 默认不提交
- 新增正式文档优先使用中文文件名

## 默认不碰的目录

除非需求明确点名，否则不改：

- `scripts/docker/`
- `scripts/nginx/`

## 注释

- 注释使用中文。
- 注释解释意图，不复述代码。
- 普通小函数不要强行加大块注释。
- 复杂流程可以用步骤注释分隔。

## 禁止行为

- 不确定时装作确定。
- 需求不清时直接实现。
- 为了显得完整而写额外功能。
- 未经要求重构无关代码。
- 修改无关格式。
- 删除非自己造成的历史遗留代码。
- 用复杂抽象包装简单逻辑。
- 跳过验证后声称完成。
- 隐瞒未验证项。
