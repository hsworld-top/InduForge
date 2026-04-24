import { defineConfig } from "vitest/config";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  server: {
    // Windows 本机环境可能无法解析 localhost，测试启动时固定走 IPv4 回环地址。
    host: "127.0.0.1",
  },
  define: {
    "import.meta.env.DEV": "false",
  },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  test: {
    include: ["tests/**/*.test.ts"],
    environment: "node",
    clearMocks: true,
  },
});
