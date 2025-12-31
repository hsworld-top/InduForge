import { defineConfig, loadEnv } from "vite";
import vue from "@vitejs/plugin-vue";
import Icons from "unplugin-icons/vite";
import { fileURLToPath, URL } from "node:url";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  // 从项目根目录加载环境变量
  const env = loadEnv(mode, path.resolve(__dirname, "../"), "");

  return {
    base: "/datacenter/", // 部署到 /datacenter/ 路径
    plugins: [
      vue(),
      Icons({
        autoInstall: true,
        compiler: "vue3",
        defaultStyle: "display: inline-block; vertical-align: middle;",
      }),
    ],
    // 确保环境变量被注入到前端代码中
    define: {
      __VITE_API_URL__: JSON.stringify(env.VITE_API_URL),
    },
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
    server: {
      port: Number(env.VITE_DATACENTER_PORT),
      host: true,
      proxy: {
        "/api": {
          target: env.VITE_API_URL,
          changeOrigin: true,
          secure: false,
        },
      },
      fs: {
        allow: [".."],
      },
    },
    build: {
      outDir: "dist",
      sourcemap: false,
      rollupOptions: {
        output: {
          manualChunks: {
            vendor: ["vue", "vue-router", "pinia"],
            ui: ["element-plus", "@element-plus/icons-vue"],
            monaco: ["monaco-editor"],
          },
        },
      },
    },
    optimizeDeps: {
      include: ["vue", "monaco-editor"],
    },
  };
});
