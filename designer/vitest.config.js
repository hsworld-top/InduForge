import { fileURLToPath, URL } from "node:url";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: "jsdom",
    environmentOptions: {
      jsdom: {
        url: "http://127.0.0.1:3000",
      },
    },
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
  // 当前 Windows 环境 localhost DNS 可能解析失败，测试用 Vite server 固定走 IPv4 回环。
  server: {
    host: "127.0.0.1",
  },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
      "~icons": fileURLToPath(new URL("./src/test-stubs/icons", import.meta.url)),
    },
  },
});
