# 组件开发指南

## 1. 文档定位

- 本文档定义 `designer` 后续新增和优化组件的正式约定，用于约束物料声明、基础属性区、描述符、事件暴露、国际化和发布态字段边界。
- 本文档聚焦组件开发规则，不替代 [设计器概览](./README.md)、[尺寸约定](./size-convention.md) 和 [Designer 发布态 Schema 契约](../contracts/designer-publish-schema.md)。
- 本文档默认以当前按钮组件优化后的结构作为参考实现：高频属性放基础属性区，事件继续留在高级面板，权限继续留在权限控制区域。

## 2. 新增组件文件规范

新增一个组件时，优先按以下位置组织代码：

- `src/materials/<Component>/manifest.ts`：声明组件名称、分类、默认样式、属性、事件和物料面板展示信息。
- `src/editor-core/descriptors/<component>.ts`：声明渲染标签、默认尺寸、容器能力、属性过滤、展示文案和自定义渲染规则。
- `src/editor-core/descriptors/<component>-props.ts`：当存在设计态辅助字段、属性映射、图标解析或运行态过滤时，单独承载 props 归一化逻辑。
- `src/materials/index.ts`：注册组件 descriptor。
- `src/materials/manifests/manifest-registry.ts`、`src/i18n/messages/zh.ts`、`src/i18n/messages/en.ts`：补充 manifest 本地化映射和文案。
- `*.test.ts`：为 manifest 本地化、descriptor 映射、特殊属性编辑器补充就近测试。

组件目录可以继续保持轻量；只有当组件有特殊渲染、特殊 props 映射或容器规则时，才需要增加 descriptor 辅助文件。

## 3. Manifest 声明原则

- `type` 必须稳定，发布态、画布节点、descriptor 注册都依赖它，不得随中文名称或 UI 调整变化。
- `name`、`label`、`group`、`placeholder` 和 `options.label` 优先写中文源文案，并同步补充中英文 i18n 映射。
- `defaultValue` 必须是用户拖入组件后能直接理解和使用的默认值。例如按钮默认文字使用“按钮”，不使用 `button` 这类工程占位。
- 属性必须声明清楚类型、默认值、可选项和是否可绑定，不依赖运行时猜测。
- 设计态辅助字段可以存在于 manifest 中，但必须在 descriptor 或 props 归一化层转换为渲染库可消费的字段。
- 不把低频 HTML 原生能力默认塞进基础属性区。确实需要开放时，应先确认它是目标用户高频配置项，否则交给高级脚本能力处理。

推荐的 manifest 结构如下：

```typescript
export const manifest: ComponentManifest = {
  type: "Button",
  name: "按钮",
  category: "PC端组件",
  defaultStyle: { width: "auto", height: "auto" },
  props: [
    {
      name: "text",
      type: "string",
      label: "按钮文字",
      group: "内容",
      defaultValue: "按钮",
      bindable: true,
    },
  ],
  events: [],
}
```

## 4. 基础属性区分组规范

基础属性区只放用户拖入组件后最可能立刻修改的配置。分组应稳定、少而清晰，优先使用以下顺序：

- `内容`：展示文本、占位文本、图标、图片源、选项内容等直接影响组件内容的字段。
- `外观`：类型、视觉规格、颜色语义、样式变体、圆角样式等组件自身视觉字段。
- `状态`：禁用、加载中、只读、展开状态、默认选中等组件状态字段。

除非组件确实有独立于全局布局系统的组件内布局能力，否则不要新增 `布局` 分组。画布尺寸、宽度、高度、位置、对齐、间距等统一使用现有全局属性面板和尺寸能力，避免每个组件重复一套宽度模式或布局模式。

`更多属性` 不作为默认分组。低频配置应优先放到高级能力、脚本或专门的高级面板中；只有当某个组件确实存在用户高频但不适合前三类分组的配置时，才允许新增，并在评审时说明原因。

## 5. 属性命名与交互规范

- 组件外框尺寸使用全局尺寸能力；如果组件库自身还有 `size` 一类视觉档位，标签应命名为 `视觉规格`，避免用户误以为它会改变画布节点宽高。
- 互斥布尔属性不要直接暴露多个开关。例如 `round` 和 `circle` 应合并为 `shape: 默认 / 圆角 / 圆形`，再映射为底层渲染 props。
- 图标、颜色、类型等有明确枚举的字段应使用选择器，不让用户必须记忆英文值。
- 图标字段存储稳定值，展示本地化文案。中文界面只显示中文，英文界面只显示英文；切换语言后输入框和下拉项都跟随语言切换。
- 对于暂时无法做完整选择器的字段，可以先保留字符串输入，但必须提供清晰的中文标签和示例 placeholder。
- 文本字段如果与底层组件库 prop 重名但语义不同，必须在渲染前过滤或转换，避免把展示文案误传为底层布尔或结构化 prop。

## 6. 事件与权限边界

- 点击、变更、提交、二次确认、防抖、节流等事件相关配置继续放在右侧高级面板，不在组件基础属性区重复入口。
- 权限控制继续放在当前统一的权限控制区域，不迁移到组件自身 manifest props。
- 组件 manifest 可以声明组件支持哪些事件，但基础属性区不显示事件编辑项。
- 不为了某个组件单独扩展动作系统字段。事件结构必须继续遵守发布态 Schema 和现有动作链路。

## 7. Descriptor 与运行态字段规范

当组件属性不能直接传给底层渲染组件时，必须通过 descriptor 或 props 归一化函数处理：

- 过滤设计态辅助字段，避免 `shape`、`widthMode` 等编辑器内部字段直接污染 Element Plus 或发布态运行组件。
- 将友好枚举转换为底层 props。例如 `shape = "round"` 转换为 `round = true, circle = false`。
- 将稳定资源值转换为可渲染对象。例如按钮图标值 `Search` 转换为实际图标组件。
- 保留高级用法需要的自定义值，但不要让自定义值破坏默认渲染。
- 明确 `getDisplayContent` 或等价能力，保证画布、图层树和预览中的展示文案一致。

如果组件是容器或布局类组件，还必须在 descriptor 中声明容器能力、子节点接受规则、默认布局方式和子节点尺寸行为。

## 8. 布局与画布规范

- `defaultStyle` 和 descriptor 默认尺寸只负责拖入画布后的初始可见状态，不替代全局尺寸编辑器。
- 普通组件不要新增 `block`、`widthMode`、`fixedWidth` 等组件私有宽度字段；宽高、撑满、固定尺寸等由全局尺寸和布局能力处理。
- 容器组件必须明确 `isContainer`、允许的子节点关系和内部布局方式。
- 组件样式不应依赖频繁全量重渲染；影响画布 DOM 的逻辑优先放到 descriptor、composable 或稳定的渲染适配层。

## 9. 国际化规范

- 每新增一个 manifest 展示文案，都要补充中文和英文消息。
- `label`、`group`、`placeholder`、`options.label` 都需要覆盖本地化。
- 选择类字段应存储稳定值，不存储当前语言文案。
- 新增或调整属性后，需要覆盖 `builtin-manifests.i18n.test.ts` 或就近测试，确保中英文 manifest 都能正确展示。

## 10. 测试与验证规范

根据改动范围选择最近的验证：

- Manifest 文案、分组、选项变化：补充或更新 manifest i18n 测试。
- Descriptor props 映射、过滤、默认展示变化：补充 descriptor 单元测试。
- 新增属性编辑器类型：补充属性编辑器交互测试。
- 涉及渲染、画布或发布结构：至少运行 `pnpm --dir designer typecheck` 和相关单元测试。
- 完整验证命令仍以模块规则为准：`pnpm --dir designer typecheck`、`pnpm --dir designer test`、`pnpm --dir designer build`。

如果全量测试存在与当前任务无关的历史失败，汇报时必须明确区分“本次改动相关验证”和“已有失败项”。

## 11. 新增组件检查清单

提交新增组件前逐项确认：

- 是否有稳定 `type`，并完成 descriptor 注册。
- 拖入画布后的默认文案、默认尺寸和默认状态是否可直接理解。
- 基础属性是否只包含高频配置，并按 `内容`、`外观`、`状态` 组织。
- 是否避免重复全局尺寸、布局、权限和事件入口。
- 是否将设计态辅助字段转换或过滤，不直接泄漏到运行态。
- 是否补齐中文和英文文案。
- 是否为 manifest、descriptor 或特殊编辑器补充了对应测试。
- 是否确认发布态字段仍符合 [Designer 发布态 Schema 契约](../contracts/designer-publish-schema.md)。

## 12. 禁止事项

- 禁止在组件基础属性区新增事件编辑入口。
- 禁止把权限控制迁移到组件私有属性中。
- 禁止让设计态辅助字段直接进入底层组件 props 或发布态结构。
- 禁止为普通组件重复实现宽度、高度、撑满容器等全局尺寸能力。
- 禁止为了低频原生能力堆叠 `更多属性`，导致基础属性区变成底层 API 面板。
- 禁止只改中文 manifest 文案而不补英文映射和本地化测试。

## 13. 关联文档

- [设计器概览](./README.md)
- [Designer 发布态 Schema 契约](../contracts/designer-publish-schema.md)
- [层级约定](./layer-order-convention.md)
- [放置与堆叠](./placement-and-stacking.md)
- [尺寸约定](./size-convention.md)
- [详细设计](../详细设计.md)
