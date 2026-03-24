# Designer TS 迁移下一阶段执行计划

## 当前状态

- TS 基础设施已可用：
  - 已接入 `typescript`、`vue-tsc`、TS ESLint 配置
  - `tsconfig.json` 已启用严格模式
  - `pnpm typecheck` 当前可稳定通过
- 已迁移到 TS 的范围主要包括：
  - `src/services/*`
  - `src/types/api.ts`
  - `src/constants/index.ts`
  - `src/utils/request.ts`
  - `src/utils/storage.ts`
  - `src/editor-core/document/types.ts`
  - 部分 `manifests` / `registry` / `descriptors`
- 当前代码规模仍以旧源码为主：
  - `src` 内约 26 个 TS 文件、76 个 JS 文件、45 个 Vue 文件
- 当前最主要的问题不是基础设施，而是：
  - 核心主干仍在 JS：`stores`、`data`、`editor-core` 主实现、画布 composables、预览运行时
  - 兼容层仍大量存在：schema normalize、settings normalize、descriptor fallback、历史类型 normalize、部分预览数据兼容
  - 一些已迁移 `.ts` 文件虽然已去掉 `@ts-nocheck`，但整体类型闭环还没有覆盖到主流程

## 本阶段目标

- 不再横向扩散迁移范围，集中收口已经开始的迁移。
- 以“主流程先闭环、兼容层持续删除、JS 主干逐块收缩”为原则推进。
- 本阶段结束时，要求：
  - 接口层只保留新协议
  - `stores` 层开始摆脱旧 normalize 依赖
  - `editor-core` 主实现开始进入真实 TS 迁移
  - 不再新增新的业务 `.js` 文件

## 执行顺序

### 1. 收紧接口层，完成新协议统一

- 保持 `src/services/*` 全部为 TS，并继续补全返回类型。
- 所有消费资源接口、页面接口、设置接口的地方，统一只读取唯一新结构：
  - 资源：`{ folders: [] }`、`{ assets: [] }`
  - 页面列表：`{ pages: [], entryConfig }`
  - 设置：`{ globalVariables, globalScripts }`
- 删除仍残留的旧兼容读取：
  - `data?.items`
  - `data?.list`
  - 数组直返兜底
  - `result?.data || result` 这类旧读取
- 本阶段优先处理：
  - `src/stores/editor-store.js`
  - `src/ui/editors/page/preview/previewRuntime.js`
  - `src/ui/editors/page/canvas/composables/use-preview.js`
  - `src/ui/editors/page/panels/left/DatapointPanel.vue`

### 2. 清理 schema/settings 兼容层

- 收敛 `src/stores/editor/normalize-schema.js`：
  - 删除页面级 payload 与工程级 payload 双路兼容
  - 删除 `fallbackPageId` 兜底分支
  - 删除历史布局类型纠正
  - 删除旧脏数据修复逻辑
  - 只接受唯一 `ProjectSchema`
- 收敛 `src/stores/editor/normalize-settings.js`：
  - 删除字符串型旧 `source` 转换逻辑
  - 删除旧变量结构兼容兜底
  - 只接受新设置结构
- `editor-store` 中与之对应的调用同步收紧，不再假定后端会返回旧结构。

### 3. 开始拆解并迁移 `editor-store`

- 不直接整文件硬转 TS。
- 先从 `src/stores/editor-store.js` 中拆出以下子模块：
  - 工程加载与保存
  - 页面 CRUD
  - 工程变量与脚本设置
  - 节点更新与布局修正
  - 选中/历史/锁状态桥接
- 拆出的新模块直接使用 TS。
- `editor-store.js` 保留为临时壳层，随着子模块迁移逐步缩薄，最终整体转 TS。

### 4. 迁移 `editor-core` 主实现

- 先迁以下稳定内核模块：
  - `document/DocumentModel.js`
  - `document/Serializer.js`
  - `document/factory.js`
  - `commands/*`
  - `selection/SelectionModel.js`
  - `lock/PageLockManager.js`
- 要求：
  - 全部改为真实 TS，而不是仅改后缀
  - 统一使用 `import type`
  - 所有 JSDoc 类型引用逐步删掉
  - 只保留当前唯一 schema，不保留旧版布局/旧版节点兼容

### 5. 删除 descriptor / manifest fallback

- 当前仍保留的典型兼容点：
  - 未注册 descriptor 时从 manifest 读取
  - 历史组件类型 normalize
  - 某些组件的默认占位/默认列/默认数据兜底
- 本阶段要求：
  - `descriptor`、`manifest`、`componentRegistry` 只保留单一路径
  - 对未注册组件直接报错或拒绝渲染，不再回退
  - 对历史类型拼写不再自动纠正
- 优先清理：
  - `src/components/descriptors/registry.ts`
  - `src/ui/editors/page/canvas/composables/use-node-drop.js`
  - `src/ui/editors/page/canvas/CanvasContainer.vue`
  - `src/ui/editors/page/panels/right/PropertyPanel.vue`

### 6. 最后处理高复杂 UI 和预览运行时

- 在接口层、store 层、editor-core 类型稳定之后，再迁：
  - `PropertyPanel.vue`
  - `CanvasContainer.vue`
  - `NodeRenderer.vue`
  - `use-preview.js`
  - `previewRuntime.js`
- 这些模块允许继续保留：
  - 用户脚本字符串
  - DSL 文本
  - `new Function` 执行
- 但不允许继续保留：
  - 旧结构输入兼容
  - fallback manifest
  - 历史类型自动纠正

## 本阶段验收标准

- `pnpm typecheck` 持续通过。
- 不新增业务 `.js` 文件。
- `services` 和主要消费方只接受新接口结构。
- `normalize-schema` / `normalize-settings` 删除一轮旧兼容。
- `editor-store` 至少拆出 2 到 3 个 TS 子模块。
- `editor-core` 至少完成一组主干模块的真实 TS 迁移。

## 明确不做

- 本阶段不做新功能开发。
- 本阶段不做 UI 视觉改版。
- 本阶段不做“为了兼容老数据而保留旧路径”的处理。
- 本阶段不追求一次性把全部 `.vue` 改成 `lang="ts"`；顺序仍然是先内核、再状态、后复杂界面。
