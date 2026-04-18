import { fileURLToPath, URL } from "node:url";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: "jsdom",
    globals: true,
    /**
     * Windows：fork 池对 node_modules 下 esbuild 子进程 spawn 可能报 EPERM。
     * 使用 threads + 单 worker 降低 spawn；仍失败请排除 designer/node_modules 实时扫描。
     * 如需回退 forks，改回 pool: "forks" 并恢复 poolOptions.forks.singleFork。
     */
    pool: "threads",
    fileParallelism: false,
    maxWorkers: 1,
    include: ["src/**/*.{test,spec}.{js,ts}", "testing/**/*.{test,spec}.{js,ts}"],
    coverage: {
      reporter: ["text", "json", "html"],
    },
  },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
      "~icons": fileURLToPath(new URL("./src/test-stubs/icons", import.meta.url)),
    },
  },
});
