# AI 分发包索引

## 1. 说明
- 本目录用于给独立 AI 或模块负责人提供最小上下文包。
- 每个分发包只保留继续推进该模块所必须的信息，不重复整套平台文档。
- 若分发包与平台正式文档冲突，以平台正式文档和契约文档为准。

## 2. 当前分发包
- [dev_core AI 分发包](./dev_core-ai-package.md)
- [designer AI 分发包](./designer-ai-package.md)
- [runtime_node_agent AI 分发包](./runtime_node_agent-ai-package.md)
- [runtime_engine AI 分发包](./runtime_engine-ai-package.md)
- [dev_ide AI 分发包](./dev_ide-ai-package.md)

## 3. 推荐使用顺序
1. 先看本模块 AI 分发包。
2. 再看本模块 `*.task.md`。
3. 如涉及跨模块契约，再看 `docs/contracts/`。
4. 如涉及平台主线，先看 [开发态预览专项计划](../开发态预览专项计划.md)，再看 [运行时闭环专项计划](../运行时闭环专项计划.md)。

## 4. 当前缺失
- `datacenter` 还未单独生成 AI 分发包；若后续要并行推进处理层一期，再单独补充。
