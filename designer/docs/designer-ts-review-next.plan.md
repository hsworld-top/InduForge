# Designer TS 迁移 Review 后下一步计划

## Review 结论

- 当前改动已经完成了 TS 基础设施接入，`pnpm typecheck` 与 `pnpm build` 均可通过。
- `services`、`types/api.ts`、部分 `registry` / `descriptors` / `document types` 已进入 TS 轨道。
- 当前最大问题不再是构建或类型检查，而是迁移目标与代码现状还未完全一致：
  - 仍保留多处兼容层
  - 核心主干仍主要是 JS
  - `stores`、`preview runtime`、画布 composables、属性面板仍是主要工作量来源

## 下一步目标

- 不再扩散迁移范围，优先清理与“新开发不保兼容”冲突的路径。
- 让 `editor-store`、`normalize-schema`、`normalize-settings` 与预览运行时真正只接受新协议。
- 开始从“基础设施迁移”转向“主流程迁移”。

## 执行顺序

### 1. 删除剩余接口兼容读取

- 收口以下路径中 `payload?.data ?? payload`、`picked ?? payload?.data ?? payload` 等旧兼容读取：
  - `src/ui/editors/page/preview/previewRuntime.js`
  - `src/ui/editors/page/canvas/composables/use-preview.js`
  - `src/ui/editors/page/panels/left/DatapointPanel.vue`
  - `src/stores/editor-store.js`
- 所有接口消费统一为唯一新结构，不再保留旧 envelope 与裸 payload 双读。

### 2. 收紧 schema/settings 规范化逻辑

- 继续清理 `src/stores/editor/normalize-schema.js`：
  - 删除对非法根节点的自动修复
  - 删除自动补齐行列和历史布局修正
  - 非法结构直接抛错
- 继续清理 `src/stores/editor/normalize-settings.js`：
  - 删除 fallbackDefinitions 驱动的旧变量兜底
  - 只保留新 settings 结构

### 3. 清理属性面板与画布侧的历史类型兼容

- 收口 `src/ui/editors/page/panels/right/PropertyPanel.vue`：
  - 删除 `normalizeElementType` 历史类型纠正
  - 删除 manifest fallback
  - 删除 Menu detailConfig fallback 回写
- 收口画布相关：
  - `src/ui/editors/page/canvas/composables/use-node-drop.js`
  - `src/ui/editors/page/canvas/CanvasContainer.vue`
- 未注册 descriptor 或非法组件类型直接失败，不再自动回退。

### 4. 开始拆分 `editor-store`

- 从 `src/stores/editor-store.js` 拆出 3 个 TS 子模块作为第一批：
  - 工程/页面加载
  - 工程设置与变量
  - 页面 CRUD
- `editor-store.js` 只保留状态组装和对外 action 代理。

### 5. 推进 `editor-core` 主实现转 TS

- 第一批迁移目标：
  - `document/DocumentModel.js`
  - `document/Serializer.js`
  - `commands/Command.js`
  - `commands/History.js`
  - `utils/EventEmitter.js`
- 这些模块迁完后，再迁 `SelectionModel` 与 `PageLockManager`。

## 验收标准

- `pnpm typecheck` 通过
- `pnpm build` 通过
- 上述兼容路径至少清掉一轮
- `editor-store` 至少拆出首批 TS 子模块
- 不新增新的业务 `.js` 文件
