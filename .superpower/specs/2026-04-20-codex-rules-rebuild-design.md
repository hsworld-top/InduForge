# InduForge Codex 规则重建设计

## 目标

- 将仓库规则收敛为仅面向 Codex 的最小高效协作规则。
- 去除 GSD、Claude、Cursor 相关残留，规则只描述当前仓库真实状态。
- 将过程文档统一迁移到 `.superpower/`，默认不提交到 Git。

## 当前仓库事实

- 仓库是多模块单仓，不是统一技术栈项目。
- `dev_core` 是 Node.js + Express + Sequelize 后端。
- `data_service` 是独立 Go 数据域服务。
- `dev_ide`、`datacenter`、`designer`、`runtime/node_agent_front` 都是 Vue 前端，但脚本与依赖成熟度不同。
- `runtime/node_agent` 是独立 Go 节点执行器。
- 各模块验证入口不统一，必须按模块分别声明。

## 规则拓扑

- 根 `AGENTS.md` 作为唯一公共规则正文入口。
- 各模块 `AGENTS.md` 只保留模块差异，统一采用六段骨架：
  - 模块定位
  - 进入前先看
  - 开发约束
  - 禁止事项
  - 验证命令
  - 相关契约
- 新增 `data_service/AGENTS.md`，补齐当前真实模块。
- 不再保留任何 `CLAUDE.md`、`.claude/`、`.cursor/` 与 GSD 相关入口。

## 根规则范围

- 仅保留 Codex 默认工作方式、通用工程约束、目录路由、文档治理与非目标目录。
- 不写项目背景长文、README 式技术栈总览、抽象工作流说明和未来口号。
- 不把 `docs/contracts/` 纳入规则优先级，只在模块规则中作为参考入口保留。

## 文档治理

- 正式业务文档放在 `docs/`。
- Superpower 过程文档放在 `.superpower/specs/`、`.superpower/plans/`、`.superpower/tmp/`。
- `.superpower/` 默认不提交到 Git。

## 预期收益

- Codex 读取规则的成本更低，减少误判模块边界和验证方式的概率。
- 规则内容更贴合当前仓库状态，避免继续引用已经移除的 GSD、Claude、Cursor 体系。
- 后续新增或调整模块时，只需要维护对应模块 `AGENTS.md`。
