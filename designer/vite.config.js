import { createRequire } from 'node:module'
import path from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig, loadEnv } from 'vite'
import { htProductionPolicy } from './scripts/ht-editor/production-policy.mjs'
import { createFrontendProxy } from '../scripts/dev/frontend-proxy.mjs'
import { resolveFrontendWorkspaceProxyPort } from '../scripts/dev/frontend-workspace-proxy.mjs'

const require = createRequire(import.meta.url)
const Icons = require('unplugin-icons/vite').default

const rootDir = fileURLToPath(new URL('.', import.meta.url))
const envDir = path.resolve(rootDir, '../')

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  // 从项目根目录加载环境变量
  const env = loadEnv(mode, envDir, '')

  return {
    base: '/designer/', // 部署到 /designer/ 路径
    // Vite 客户端环境变量也必须从仓库根目录注入，和代理配置保持同一来源。
    envDir,
    define: {
      __DESIGNER_DEBUG_ROUTE_ENABLED__: JSON.stringify(mode !== 'production'),
      __FRONTEND_WORKSPACE_PROXY_SUFFIX__: JSON.stringify(
        mode === 'frontend-linux' ? env.IF_FRONTEND_WORKSPACE_PROXY_SUFFIX || '' : '',
      ),
      __FRONTEND_WORKSPACE_PROXY_PORT__: JSON.stringify(
        mode === 'frontend-linux'
          ? resolveFrontendWorkspaceProxyPort(env.IF_FRONTEND_WORKSPACE_PROXY_PORT)
          : 0,
      ),
    },
    plugins: [
      vue(),
      Icons({
        autoInstall: true,
        compiler: 'vue3',
        defaultStyle: 'display: inline-block; vertical-align: middle;',
      }),
      htProductionPolicy(rootDir),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: Number(env.VITE_DESIGNER_PORT),
      host: true,
      proxy: createFrontendProxy(env, { requireCenterTarget: mode === 'frontend-linux' }),
      fs: {
        allow: ['..'],
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
    },
    optimizeDeps: {
      include: ['vue'],
    },
  }
})
