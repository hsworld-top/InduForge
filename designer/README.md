# Designer（设计中心前端）

## 本地门禁（请在 `designer` 目录执行）

```bash
cd designer
pnpm install
pnpm run typecheck
pnpm run lint
pnpm run test
pnpm run build
```

若在仓库根目录或其它路径执行 `pnpm run lint`，可能出现 **`Cannot find module 'typescript'`**：请在 **`designer` 目录** 安装依赖，并确认存在 `node_modules/typescript`。

若仍报错：删除本目录下 `node_modules` 后重新 `pnpm install`；排除杀毒软件对 `designer/node_modules` 的实时扫描。仍失败时可在团队允许下于 `.npmrc` 增加 `shamefully-hoist=true` 后重装。

## 测试（Windows `esbuild spawn EPERM`）

默认已在 `vitest.config.js` 使用 `pool: "threads"` 与 `maxWorkers: 1`。若仍失败，请将 `designer/node_modules` 加入杀毒/Defender 排除；必要时可改回 `pool: "forks"` 并配置 `poolOptions.forks.singleFork`（见该文件注释）。
