# InduForge Claude 入口

@./AGENTS.md

整仓启动 Claude 时，处理具体模块前先补读对应目录下的规则文件：

- `dev_core/AGENTS.md`
- `dev_ide/AGENTS.md`
- `datacenter/AGENTS.md`
- `designer/AGENTS.md`
- `runtime/AGENTS.md`
- `runtime/node_agent/AGENTS.md`
- `runtime/node_agent_front/AGENTS.md`

如果当前工作目录已经位于某个模块目录内，优先遵守该目录下的 `CLAUDE.md` 与 `AGENTS.md`，不要把其他模块规则默认带入上下文。
