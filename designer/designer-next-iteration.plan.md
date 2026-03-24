# Designer 下一步迭代计划

## 摘要
本轮仅做稳定性迭代，不扩产品功能，围绕最近一批 `designer` 重构提交的 review 结论收口高风险问题。目标是修复运行时回归、统一渲染数据链路、收敛 schema 归一化副作用，并补齐最小验证闭环，确保后续实施时不再需要补做方案决策。

## 关键改动
### 1. 修复 `NodeRenderer` 初始化顺序问题 ✅ 已完成
- 调整 `NodeRenderer` 内部声明顺序，保证 `nodeRef`、`contentRef` 等依赖先定义，再传入各 composable。
- 全量检查同文件中“先引用后声明”的 setup 变量，避免再次出现 TDZ 运行时错误。
- 本项完成标准：设计器页面能稳定进入，不因组件初始化直接报错。

### 2. 统一 props 派生链路 ✅ 已完成
- 固定渲染链路为 `resolvedNodeProps -> resolvedProps -> filteredProps`。
- `resolvedNodeProps` 只承载表达式绑定与原始属性合并。
- `resolvedProps` 负责布局派生值与预览态修正值，例如 `ElCol span`、`ElLayoutRow gutter`。
- `filteredProps` 必须改为基于 `resolvedProps` 做 descriptor 过滤，模板绑定与 `renderKey` 计算都使用最终链路，不能再绕回 `resolvedNodeProps`。
- 本项完成标准：布局派生值在模板实际生效，编辑态与预览态行为一致。

### 3. 收敛 schema 归一化职责 ✅ 已完成
- 重构 `normalizeLayoutSchema` / `ensurePageRootNodes` 的职责，只在根节点缺失、非法或历史脏数据不完整时做修复。
- 不再无条件把页面根节点强制改写为 `FreeContainer`。
- 对历史 schema 的兼容逻辑保留，但必须是“补缺修错”，不是“静默重写合法结构”。
- 本项完成标准：已有合法布局根节点的页面重新打开后结构不被改写。

### 4. 补齐核心交互回归验证 ⚠️ 待手动验证
- 回归布局容器链路：`ElLayout / ElLayoutRow / ElCol` 的默认宽高、`span`、`gutter`、拖拽插入。
- 回归 `Tabs` 链路：激活项同步、删除页签、拖入当前激活页签、`activeName/modelValue` 一致性。
- 回归渲染链路：自定义渲染器、表格列刷新、容器空态提示、只读预览态样式。
- 本项完成标准：上述场景均可复现并验证，不再依赖人工猜测重构是否安全。

### 5. 控制后续重构边界 ✅ 已完成
- 给 `NodeRenderer` 周边 composable 固定责任边界：
  - `use-node-props` 只管数据派生（表达式绑定、变量上下文、布局派生值）。
  - `use-node-content` 只管组件特化内容（options、tabs、menu 等）。
  - `use-node-renderer-derivations` 只管模板派生状态（renderKey、nodeClass 等）。
  - 交互、拖拽、缩放、预览不再互相持有隐式状态。
- descriptor 体系统一入口，不再允许同一组件的关键行为同时散落在 descriptor 和 `NodeRenderer` fallback 里。
- 本项完成标准：后续再拆分时不会再次出现”逻辑在 A 计算、模板却从 B 取值”的回归。

## 需要关注的内部接口约束
- 不新增对外 API。
- 内部行为约束如下：
  - `filteredProps` 的输入统一为 `resolvedProps`。
  - `renderKey` 依赖的 props 与模板绑定的 props 必须语义一致。
  - schema 归一化只修复非法数据，不重定义合法页面结构。
  - `Tabs` 的最终激活状态以统一 props 链路为准，不保留多套并行真值源。

## 测试与验收
- 构建验证：`pnpm -C designer run build`
- 关键场景验收：
  - 打开历史页面工程，确认页面根节点结构不被静默替换。
  - 新建并编辑 `ElLayout` 页面，确认 `ElCol` 自动宽度与拖拽行为正常。
  - 预览态下检查 `gutter`、容器空态、自定义渲染器、表格刷新。
  - `Tabs` 新增、切换、删除、拖入组件后，内容区与激活项保持一致。
- 通过标准：
  - 无运行时初始化错误。
  - 无布局派生值失效问题。
  - 无页面结构被打开即改写的问题。
  - 构建持续通过。

## 默认假设
- 本轮优先级是稳定性高于功能扩展。
- 页面根节点允许是合法布局容器，因此不能被统一改写成 `FreeContainer`。
- 本轮只做修复、验证、边界收口，不追加新的设计器能力。
