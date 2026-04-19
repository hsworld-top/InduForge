# designer 模块规则

## 模块定位

- `designer` 是低代码页面设计器，负责编辑器内核、组件编排、属性配置、导出与预览宿主。

## 处理该模块时先看哪里

- `designer/package.json`
- `designer/src/`
- `designer/scripts/`
- `docs/designer/README.md`
- `.planning/docs/process/designer/refactor/README.md`
- `docs/contracts/designer-publish-schema.md`
- `.planning/ai-packages/designer-ai-package.md`

## 必须遵守

- 技术栈为 Vue 3 + Pinia + Element Plus + Konva + ECharts + GSAP。
- 复杂状态变更统一走 store action，画布相关逻辑优先拆到 `engine/` 或 composable。
- 表达式与脚本执行必须处在可控上下文内；Bridge 模式要处理未就绪降级。
- 发布态只保留 Runtime 必需字段，不把编辑器内部状态带到运行态。
- 避免频繁触发全量重渲染，保持 DOM 渲染与画布辅助层分层。

## 不要做

- 不要在当前任务中顺手扩复杂动作系统、多视图或完整权限模型。
- 不要把预览宿主和编辑态模型强耦合。
- 不要绕过现有导出契约直接发明新的运行态字段。

## 验证命令

- `pnpm --dir designer typecheck`
- `pnpm --dir designer test`
- `pnpm --dir designer build`

## 相关参考文档

- `docs/designer/README.md`
- `.planning/docs/process/designer/refactor/publish-pipeline.md`
- `.planning/ai-packages/tasks/designer.task.md`
