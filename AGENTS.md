# InduForge Codex 协作规则

## 文件定位

- 本文件是仓库唯一公共规则正文入口。
- 处理任务时先读本文件，再读离改动目录最近的 `AGENTS.md`。
- 若改动位于 `runtime/node_agent/` 或 `runtime/node_agent_front/`，先读 `runtime/AGENTS.md`，再读对应子模块规则。
- 默认按模块分别处理，不把整仓当成单一应用统一修改。

## Codex 默认工作方式

- 使用中文沟通，新增或修改注释优先使用中文。
- 当前开发环境以 Windows 为主，命令优先使用 PowerShell、`cmd`、模块自带脚本或 `Makefile`。
- 遇到 `rg`、`grep`、`sed` 等 Unix 风格命令不可用或不稳定时，直接改用 PowerShell 原生命令，不要反复尝试。
- 默认只改与当前任务直接相关的文件，不顺手扩散重构无关模块。
- 涉及多个模块时，按真实影响范围联动修改，并在汇报中说明改动边界。
- 需求不清或方案存在分歧时，优先使用`$brainstorming` 技能讨论澄清，确认后再开始实现, 实现时候是否使用superpower的流程需要用户明确指示后才能使用superpower的流程。
- 完成改动后，先执行最接近的验证命令，再汇报结果、未验证项和剩余风险。
- 开发时不创建worktree,可以创建branch,完成任务后，合并master分支，清理本地branch。
- 所有的开发不需要考虑兼容历史，当前属于开发阶段，选择最优的方式即可，不要考虑兼容旧的写法。
- 代码尽量带有注释，但是不需要过度的注释。
- Codex只关注开发，如果需要写测试，请单开个Claude来写测试。
- 把Claude当做一个测试人员

## 通用工程约束

- 默认工作目录为仓库根目录，统一使用根目录 `.env`。
- 端口配置以根目录 `.env` 为本机实际值来源；`.env_example` 与 `README.md` 只维护统一默认端口，不记录临时本机端口。
- Node 相关模块统一使用 `pnpm`；Go 模块遵循各自 `Makefile` 或 `go` 命令。
- 后端 API 前缀统一为 `/api/v1`，健康检查为 `/health`。
- 对外 JSON 接口响应统一使用 `code`、`msg`、`data`、`reqId` 四字段；`code = 0` 是唯一成功码，禁止再返回 `success`、`errorCode`、`message`、`requestId` 作为正式 HTTP 契约。
- 业务成功失败统一看业务码：`HTTP 2xx + code = 0` 表示成功，`HTTP 2xx + code != 0` 表示业务失败；技术异常统一使用 `HTTP 4xx/5xx + code != 0`。
- 分页接口统一将分页信息放入 `data.pagination`，列表主体放入 `data.list` 或 `data.list.<业务字段>`，前后端都不得继续依赖顶层 `pagination`。
- 文件流、下载流、图片流接口允许成功时直接返回二进制内容，但一旦返回错误，仍必须回到统一 JSON 包络。
- 新增错误码必须先复用现有统一错误码规则；若确需新增，必须同步更新 `docs/统一错误码枚举表.md` 与相关接口文档，不得在模块内私自扩展另一套对外错误码。
- `dev_core`、`dev_ide`、`datacenter`、`designer` 统一使用 `dayjs`，字符串时间格式为 `YYYY-MM-DD HH:mm:ss`。
- `dev_core` 数据库初始化统一使用 `dev_core/scripts/init-database.js`，禁止使用 `sequelize.sync()`。
- 新增或重构代码时补充详细代码注释，至少说明模块目的、关键流程、边界条件和异常分支；避免只重复字面含义的空洞注释。
- 数据库操作必须使用参数化查询。
- commit message 使用中文，推荐格式为 `type(scope): 中文描述`。

## 目录路由

- `dev_core/`：平台控制面后端与发布部署聚合能力。
- `data_service/`：平台侧与开发态数据域服务，负责连接、查询、数据点、协议和预览链路。
- `dev_ide/`：平台管理与运维前端。
- `datacenter/`：数据接入与数据语义建模前端。
- `designer/`：低代码页面设计器。
- `runtime/`：运行时相关共享规则入口。
- `runtime/node_agent/`：节点执行器后端。
- `runtime/node_agent_front/`：节点本地管理前端。
- `scripts/`：本地开发与基础设施脚本目录，当前包含 `docker/`、`nginx/`、`seaweedfs/`。

## 文档治理

- 正式业务文档统一放在 `docs/`。
- Superpower 过程文档统一放在根目录 `.superpower/`。
- `.superpower/specs/` 放设计稿，`.superpower/plans/` 放实施计划，`.superpower/tmp/` 放临时产物。
- `.superpower/` 默认不提交到 Git。
- 新增正式文档、过程文档、分析稿优先使用中文文件名；标准协议名、第三方约定文件和工具固定入口文件可保留英文。
- 模块目录默认不放普通过程文档；例外仅限 `AGENTS.md`。

## 非目标目录

- `scripts/docker/`、`scripts/nginx/`、`scripts/seaweedfs/` 默认不作为本轮业务开发目标，除非需求明确点名。
