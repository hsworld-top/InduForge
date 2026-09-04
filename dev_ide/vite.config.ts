import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import path from 'node:path'
import { defineConfig, loadEnv, type PluginOption } from 'vite'
import { createFrontendProxy } from '../scripts/dev/frontend-proxy.mjs'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  // 从项目根目录加载环境变量
  const env = loadEnv(mode, path.resolve(fileURLToPath(new URL('.', import.meta.url)), '../'), '')

  return {
    base: './', // 资源用相对路径，/edit 页面会落到 /edit/assets/...
    // workspace 同时包含 Vite 7/8 模板，pnpm 提升后插件会携带另一份 Vite 类型；运行时兼容由插件 peer 范围保证。
    plugins: [vue() as unknown as PluginOption],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: Number(env.VITE_IDE_PORT),
      host: true,
      proxy: {
        ...createFrontendProxy(env, { requireCenterTarget: mode === 'frontend-linux' }),
        // Wujie 子应用通过 IDE 同源路径加载，浏览器只携带 HttpOnly Cookie。
        '/datacenter': {
          target: `http://localhost:${env.VITE_DATACENTER_PORT || 18602}`,
          changeOrigin: true,
          secure: false,
        },
        '/designer': {
          target: `http://localhost:${env.VITE_DESIGNER_PORT || 18603}`,
          changeOrigin: true,
          secure: false,
        },
      },
      fs: {
        // 允许访问 node_modules
        allow: ['..'],
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
      rollupOptions: {
        output: {
          manualChunks: {
            vendor: ['vue', 'vue-router', 'pinia'],
            ui: ['element-plus', '@element-plus/icons-vue'],
          },
        },
      },
    },
    optimizeDeps: {
      // tiny-engine 依赖的外包放这里，防止慢扫描
      include: ['vue', '@vueuse/core', 'mitt'],
    },
  }
})
