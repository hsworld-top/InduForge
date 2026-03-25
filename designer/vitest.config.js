import { fileURLToPath, URL } from "node:url";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: "jsdom",
    globals: true,
    /**
     * Windows：杀毒/策略对 node_modules 下 esbuild 等子进程 spawn 可能报 EPERM。
     * 优先排除 designer/node_modules 实时扫描；仍失败可尝试改 pool 为 "threads" 且 maxWorkers: 1。
     */
    pool: "forks",
    fileParallelism: false,
    maxWorkers: 1,
    poolOptions: {
      forks: {
        singleFork: true,
      },
    },
    include: ["src/**/*.{test,spec}.{js,ts}"],
    coverage: {
      reporter: ["text", "json", "html"],
    },
  },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
});
