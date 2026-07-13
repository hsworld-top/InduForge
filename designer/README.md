# Designer（设计中心前端）

## 本地门禁（请在仓库根目录执行）

```bash
pnpm install
pnpm typecheck:designer
pnpm lint:designer
pnpm test:designer
pnpm build:designer
```

Node 依赖由根目录 pnpm workspace 统一管理，不要在 `designer` 目录单独执行 `pnpm install`。若依赖异常，请删除安装产物后回到仓库根目录重新安装；仍失败时再检查杀毒软件或 Defender 是否阻止 pnpm 创建链接或启动二进制依赖。

## 测试（Windows `esbuild spawn EPERM`）

默认已在 `vitest.config.js` 使用 `pool: "threads"` 与 `maxWorkers: 1`。若仍失败，请将 `designer/node_modules` 加入杀毒/Defender 排除；必要时可改回 `pool: "forks"` 并配置 `poolOptions.forks.singleFork`（见该文件注释）。
