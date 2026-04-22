import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import path from 'node:path'
import { defineConfig, loadEnv } from 'vite'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  // 从项目根目录加载环境变量
  const env = loadEnv(mode, path.resolve(fileURLToPath(new URL('.', import.meta.url)), '../'), '')

  return {
    base: './', // 资源用相对路径，/edit 页面会落到 /edit/assets/...
    plugins: [vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: Number(env.VITE_IDE_PORT),
      host: true,
      proxy: {
        '/api': {
          target: env.VITE_API_URL,
          changeOrigin: true,
          secure: false,
        },
        '/socket.io': {
          target: env.VITE_API_URL,
          changeOrigin: true,
          secure: false,
          ws: true,
        },
        // 前端子应用代理到各自 dev server，保持同源访问以共享 LocalStorage
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
