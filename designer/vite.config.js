import path from "node:path";
import { fileURLToPath, URL } from "node:url";
import vue from "@vitejs/plugin-vue";
import Icons from "unplugin-icons/vite";
import { defineConfig, loadEnv } from "vite";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  // 从项目根目录加载环境变量
  const env = loadEnv(mode, path.resolve(__dirname, "../"), "");

  return {
    base: "/designer/", // 部署到 /designer/ 路径
    plugins: [
      vue(),
      Icons({
        autoInstall: true,
        compiler: "vue3",
        defaultStyle: "display: inline-block; vertical-align: middle;",
      }),
    ],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
    server: {
      port: Number(env.VITE_DESIGNER_PORT),
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
            ui: ["element-plus"],
          },
        },
      },
    },
    optimizeDeps: {
      include: ["vue", "@vueuse/core"],
    },
  };
});
