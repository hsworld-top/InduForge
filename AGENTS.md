# InduForge AI 协作总则

本文件是仓库根级公共规则，只保留跨工具、跨模块都稳定成立的约束。处理具体模块前，优先读取离改动目录最近的 `AGENTS.md` 或 `CLAUDE.md`。

## 公共设置

- 使用中文沟通与注释。
- 工作目录为仓库根目录，统一使用根目录 `.env`。
- 包管理器统一使用 `pnpm`。
- 后端 API 前缀统一为 `/api/v1`，健康检查为 `/health`。
- 非业务入口目录 `docker/`、`nginx/`、`miniio/` 默认不作为本轮业务开发目标。

## 目录路由

- `dev_core/`：后端控制面与数据聚合。
- `dev_ide/`：平台管理前端。
- `datacenter/`：数据中心前端。
- `designer/`：低代码设计器。
- `runtime/`：运行时共享层。
- `runtime/node_agent/`：节点执行器后端。
- `runtime/node_agent_front/`：节点本地管理前端。

处理单模块任务时，只加载对应模块规则；跨模块任务按涉及目录分别补读，不要把整仓长文默认塞入上下文。

## 文档与规则文件

- 项目文档、过程文档、任务文档统一放在 `docs/`。
- 模块目录默认不放普通过程 `.md`，仅允许保留 AI 入口文件 `AGENTS.md`、`CLAUDE.md`。
- 根目录业务文档仅保留 `README.md`；AI 入口与忽略文件属于配置例外。
- 修改或新增业务文档前，需先与需求方确认；本次 AI 规则重构相关文件除外。

## 提交规范

- commit message 必须使用中文，推荐格式为 `type(scope): 中文描述`。
- `type` 推荐值：`feat`、`fix`、`refactor`、`docs`、`chore`、`test`。
- `scope` 推荐值：`dev_core`、`dev_ide`、`datacenter`、`designer`、`runtime`、`node_agent`、`product`。
- 单次提交优先聚焦单一目标；不要把无关改动混入同一提交。
- 若本地最近一次提交信息不符合规范，提交前先修正。

## 通用开发约束

- 后端接口统一使用 `ApiResponse` 格式：`success`、`errorCode`、`message`、`requestId`、`data`。
- 业务错误统一使用 `AppError` 与 `ErrorCodes`。
- 数据库初始化统一使用 `dev_core/scripts/init-database.js`，禁止使用 `sequelize.sync()`。
- `dev_core`、`dev_ide`、`datacenter`、`designer` 统一使用 `dayjs`，字符串时间格式为 `YYYY-MM-DD HH:mm:ss`。
- 类、函数尽量补中文文档注释；复杂业务逻辑前补一行目的说明。
- 数据库操作必须使用参数化查询。
- 避免冗余计算，前端注意懒加载与无意义的全量更新。

## 协作要求

- 需求不清时，先交付可运行的基础版本，再明确待确认问题。
- 完成改动后优先运行与改动最接近的验证命令，再汇报结论。
- 若规则与文档冲突，以模块内最新规则和 `docs/contracts/` 契约文档为准。
