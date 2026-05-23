import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import Icons from 'unplugin-icons/vite'
import { fileURLToPath, URL } from 'node:url'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  // 从项目根目录加载环境变量
  const env = loadEnv(mode, path.resolve(__dirname, '../'), '')
  // 数据服务本地调试地址：
  // 1. 支持从环境变量覆盖（VITE_DATA_SERVICE_URL）
  // 2. 未配置时默认走新的本地 Go 数据服务端口 19602
  const dataServiceUrl = env.VITE_DATA_SERVICE_URL || 'http://localhost:19602'

  return {
    base: '/datacenter/', // 部署到 /datacenter/ 路径
    plugins: [
      vue(),
      Icons({
        autoInstall: true,
        compiler: 'vue3',
        defaultStyle: 'display: inline-block; vertical-align: middle;',
      }),
    ],
    // 确保环境变量被注入到前端代码中
    define: {
      __DATACENTER_DEBUG_ROUTE_ENABLED__: JSON.stringify(mode !== 'production'),
      __VITE_API_URL__: JSON.stringify(env.VITE_API_URL),
      __VITE_DATA_SERVICE_URL__: JSON.stringify(dataServiceUrl),
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: Number(env.VITE_DATACENTER_PORT),
      host: true,
      proxy: {
        // 数据中心的数据域接口统一打到独立 Go 数据服务
        // 例如 /api/v1/data/projects/:projectId/...
        '/api/v1/data': {
          target: dataServiceUrl,
          changeOrigin: true,
          secure: false,
        },
        '/socket.io': {
          target: dataServiceUrl,
          changeOrigin: true,
          secure: false,
          ws: true,
          rewriteWsOrigin: true,
        },
        // 其余 /api 仍走原有后端（dev_core）
        '/api': {
          target: env.VITE_API_URL,
          changeOrigin: true,
          secure: false,
        },
      },
      fs: {
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
            monaco: ['monaco-editor'],
          },
        },
      },
    },
    optimizeDeps: {
      include: ['vue', 'monaco-editor'],
    },
  }
})
