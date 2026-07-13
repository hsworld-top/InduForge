# designer 协作规则

## 模块定位

- `designer` 是低代码页面设计器，负责编辑器内核、组件编排、属性配置、导出与预览宿主。

## 进入前先看

- `designer/package.json`
- `designer/src/`
- `designer/scripts/`
- `docs/03-模块设计/designer/README.md`
- `docs/04-契约与规范/跨模块契约/designer-publish-schema.md`

## 开发约束

- 运行时为 Vue 3 + TypeScript + Vite + Pinia + Element Plus + Konva + ECharts + GSAP。
- 复杂状态变更统一走 store action，画布相关逻辑优先拆到编辑器核心或 composable。
- 表达式与脚本执行必须处在可控上下文内；Bridge 模式要处理未就绪降级。
- 发布态只保留 Runtime 必需字段，不把编辑器内部状态带到运行态。
- 避免频繁触发全量重渲染，保持 DOM 渲染层与画布辅助层分层。

## 禁止事项

- 不要借当前任务顺手扩展复杂动作系统、多视图或完整权限模型。
- 不要把预览宿主与编辑态模型强耦合。
- 不要绕过现有导出契约直接发明新的运行态字段。

## 验证命令

- `pnpm --dir designer typecheck`
- `pnpm --dir designer test`
- `pnpm --dir designer build`

## 相关契约

- `docs/03-模块设计/designer/README.md`
- `docs/04-契约与规范/跨模块契约/designer-publish-schema.md`
