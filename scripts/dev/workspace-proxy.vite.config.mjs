import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import {
  createFrontendWorkspaceProxy,
  resolveFrontendWorkspaceProxyPort,
} from './frontend-workspace-proxy.mjs'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')

/**
 * 独立工作区反向代理端口。不承载 IDE 页面、API 或 Wujie，只接收严格的 workspace localhost Host。
 */
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, repoRoot, '')
  if (mode !== 'frontend-linux') {
    throw new Error('工作区本地代理只能以 frontend-linux 模式启动。')
  }
  const port = resolveFrontendWorkspaceProxyPort(env.IF_FRONTEND_WORKSPACE_PROXY_PORT)
  return {
    root: repoRoot,
    server: {
      host: true,
      port,
      strictPort: true,
      proxy: createFrontendWorkspaceProxy(env, { requireConfig: true, localPort: port }),
    },
  }
})
