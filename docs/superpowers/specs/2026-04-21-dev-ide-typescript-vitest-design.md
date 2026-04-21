# dev_ide 全量 TypeScript 化与 Vitest 迁移设计

- 日期：2026-04-21
- 模块：`dev_ide`
- 目标读者：前端开发、测试、评审人员

## 1. 背景与目标

`dev_ide` 当前为 Vue 3 + Vite + Pinia + Element Plus 的 JavaScript 项目（`src` 内 31 个 `.js`，0 个 `.ts`，15 个 `.vue`）。

本次改造目标：

1. 将 `dev_ide/src` 全量迁移为 TypeScript。
2. 将现有 `node tests/run-tests.js` 测试体系迁移为 Vitest。
3. 在保持运行行为一致的前提下补齐类型边界，支持后续渐进收紧类型严格度。

用户已确认约束：

1. `src` 全量 TS 化 + 全量改为 Vitest。
2. 允许 `*.api.js` 改为 `*.api.ts`。
3. 允许少量过渡类型（`any` / `unknown` 缩窄 / 少量 `@ts-expect-error`），不要求本轮零过渡。

## 2. 非目标

本轮不做以下事项：

1. 不改造 `dev_ide` 以外模块（`dev_core`、`datacenter`、`designer`、`runtime`）。
2. 不进行页面交互和视觉重构。
3. 不强制把所有 Options API 组件改写为 Composition API。
4. 不追求“一次性零 any”。

## 3. 迁移范围

范围内：

1. `src/**/*.js` 迁移为 `src/**/*.ts`。
2. `src/**/*.vue` 脚本统一改为 `lang="ts"`。
3. `src/api/*.api.js` 与 `src/api/system/*.api.js` 迁移为 `*.api.ts`。
4. 新增 TypeScript 相关配置（`tsconfig`、`env.d.ts`、类型检查脚本）。
5. 新增 Vitest 配置并迁移 `tests/run-tests.js` 用例到 `tests/**/*.test.ts`。

范围外：

1. 文档中未列出的工程级别工具链重构。
2. 后端接口语义变更与协议升级。

## 4. 方案对比与选型

### 方案 A：一次性重命名迁移

- 描述：单次将所有文件直接改名并修复。
- 优点：总历时短，目录形态快速达成目标。
- 缺点：回归面最大，问题定位成本高。

### 方案 B：分层迁移但单次交付（采纳）

- 描述：按“基建 -> 逻辑层 -> 视图层 -> 测试”分阶段推进，在同一分支完成后整体交付。
- 优点：每阶段可验证，故障定位清晰，仍满足本次一次性交付目标。
- 缺点：过程管理复杂度高于方案 A。

### 方案 C：长期兼容期迁移

- 描述：开启 `allowJs` 长期并行，再逐步清零 JS。
- 优点：短期改动风险最低。
- 缺点：容易停留在半迁移状态，不符合本次目标。

结论：采用方案 B。

## 5. 详细设计

### 5.1 阶段 A：TypeScript 与 Vitest 基建

变更项：

1. 新增 `tsconfig.json`，以 `src` 为核心编译边界。
2. 新增 `src/env.d.ts`（Vite `import.meta.env`、`.vue`、静态资源声明）。
3. `vite.config.js` 迁移为 `vite.config.ts`。
4. ESLint 扩展到 TS 文件（保留现有规则风格，不额外激进收紧）。
5. `package.json` 脚本调整：
   - `typecheck`: `vue-tsc --noEmit`
   - `test`: `vitest run`
   - `test:watch`: `vitest`
6. 新增 `vitest.config.ts`，默认 `jsdom` 环境。

落地原则：

1. 先让工具链可运行，再处理业务文件迁移。
2. 不在基建阶段引入无关 lint 争议。

### 5.2 阶段 B：纯逻辑层迁移（低风险先行）

目标目录：

1. `src/constants`
2. `src/enums`
3. `src/lang`
4. `src/permissions`
5. `src/utils`
6. `src/api`
7. `src/router`
8. `src/store`
9. `src/main`

关键处理：

1. 为请求层定义统一返回形态（`ApiResponse<T>`、分页结构等）。
2. 为认证与租户上下文定义核心类型（`TokenPair`、`UserInfo`、`TenantContext`）。
3. 给路由 `meta` 补充可检查字段（`requiresAuth`、`roles`、`titleKey` 等）。
4. 为 iframe/嵌入桥接消息建立联合类型，替代隐式对象传递。

### 5.3 阶段 C：SFC 全量 TS 化

范围：全部 `src/**/*.vue`。

策略：

1. 对 `<script>` 组件：先加 `lang="ts"`，按最小改动补 `defineComponent` 与关键 state/method 类型。
2. 对 `<script setup>` 组件：补 `ref`、`computed`、`props`、`emits` 类型。
3. 对复杂组件优先做边界类型（输入/输出/副作用），避免在本轮做大规模风格重写。

### 5.4 阶段 D：Vitest 用例迁移

1. 将 `tests/run-tests.js` 中 `run(name, fn)` 模式迁移为 `describe/it`。
2. 维持现有断言语义，优先使用 `expect`。
3. 建立测试辅助：
   - mock `localStorage`
   - mock `window.location`
   - 统一清理全局副作用
4. 回归重点：
   - 权限规则
   - 运维状态聚合
   - i18n 键值
   - appUrl / tabState / embedded bridge

### 5.5 阶段 E：收口与验收

每阶段至少执行：

1. `pnpm --dir dev_ide typecheck`
2. `pnpm --dir dev_ide test`
3. `pnpm --dir dev_ide build`

最终验收标准：

1. `src` 中不再存在 `.js` 业务源码。
2. `tests/run-tests.js` 不再作为默认测试入口。
3. `typecheck`、`test`、`build` 全部通过。
4. 关键行为（登录、路由鉴权、子应用嵌入与恢复）与迁移前一致。

## 6. 架构边界与数据流

### 6.1 类型边界

1. API 边界：接口函数返回显式类型，不把 `any` 扩散到调用层。
2. Store 边界：状态字段可空性和默认值显式化。
3. Router 边界：路由元信息字段收敛为受控集合。
4. Embedded 边界：`postMessage` 协议消息具名且可判别。

### 6.2 数据流（保持不变）

1. 登录与刷新：`View -> Store -> API -> Request Interceptor -> Storage`
2. 鉴权路由：`Router Guard -> Storage/Store -> Permissions`
3. 子应用嵌入：`Dashboard/EmbeddedApp -> Bridge Utils -> postMessage`

说明：本次只做类型显式化，不改变职责归属和调用顺序。

## 7. 异常处理设计

1. `catch (error: unknown)` 统一进入类型缩窄。
2. Axios 错误按 `response/status` 分支处理，保持现有用户提示文案。
3. 刷新 token 队列显式声明 `resolve/reject` 签名，避免隐式 `any` 链路。
4. 若刷新失败，保持“清理本地凭据并跳转登录”的原行为。

## 8. 风险清单与缓解

1. 风险：Options API 组件 `this` 类型复杂。
   - 缓解：优先最小注解，不在本轮重写组件风格。
2. 风险：历史接口返回结构不统一。
   - 缓解：先定义最小可用类型并就地缩窄。
3. 风险：批量改名造成 import 漏改。
   - 缓解：阶段化迁移并在每阶段执行 `typecheck + test + build`。
4. 风险：测试环境与浏览器环境差异。
   - 缓解：Vitest 统一 `jsdom`，将全局 mock 工具化。

## 9. 实施顺序与边界说明

1. 仅改 `dev_ide` 模块，不触达其他模块代码。
2. 先改基建与逻辑层，再改视图层，最后迁测试并收口。
3. 提交时优先保持“单阶段单语义”以便回滚。

## 10. 完成定义（Definition of Done）

1. `dev_ide/src` 全量 TypeScript 化。
2. 默认测试命令已切换为 Vitest。
3. 构建、类型检查、测试通过。
4. 存在少量过渡类型时，位置受控且有后续收敛清单。
