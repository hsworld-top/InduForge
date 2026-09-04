import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import Icons from 'unplugin-icons/vite'
import { fileURLToPath, URL } from 'node:url'
import path from 'path'
import { createFrontendProxy } from '../scripts/dev/frontend-proxy.mjs'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  // 从项目根目录加载环境变量
  const env = loadEnv(mode, path.resolve(__dirname, '../'), '')
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
    define: {
      __DATACENTER_DEBUG_ROUTE_ENABLED__: JSON.stringify(mode !== 'production'),
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: Number(env.VITE_DATACENTER_PORT),
      host: true,
      proxy: createFrontendProxy(env, { requireCenterTarget: mode === 'frontend-linux' }),
      fs: {
        allow: ['..'],
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (!id.includes('node_modules')) return undefined
            if (id.includes('/monaco-editor/')) return 'monaco'
            if (id.includes('/element-plus/') || id.includes('/@element-plus/')) return 'ui'
            if (id.includes('/vue/') || id.includes('/vue-router/') || id.includes('/pinia/')) {
              return 'vendor'
            }
            return undefined
          },
        },
      },
    },
    optimizeDeps: {
      include: ['vue', 'monaco-editor'],
    },
  }
})
