import { readFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath, pathToFileURL } from 'node:url'

const workspaceRoot = path.resolve(process.env.WORKSPACE_ROOT || '/workspace')
const previewPort = Number.parseInt(process.env.VITE_PORT || '5173', 10)
const platformRoot = path.dirname(fileURLToPath(import.meta.url))
const viteEntry = path.join(workspaceRoot, 'node_modules', 'vite', 'dist', 'node', 'index.js')
const erudaSource = await readFile('/usr/local/lib/node_modules/eruda/eruda.js')
const bridgeSource = await readFile(path.join(platformRoot, 'preview-devtools-client.js'))
const { createServer } = await import(pathToFileURL(viteEntry).href)

const platformPlugin = {
  name: 'induforge-preview-platform',
  configureServer(server) {
    server.middlewares.use((request, response, next) => {
      if (request.url === '/__induforge/eruda.js') {
        response.writeHead(200, { 'Content-Type': 'text/javascript; charset=utf-8' })
        response.end(erudaSource)
        return
      }
      if (request.url === '/__induforge/preview-devtools.js') {
        response.writeHead(200, { 'Content-Type': 'text/javascript; charset=utf-8' })
        response.end(bridgeSource)
        return
      }
      next()
    })
  },
  transformIndexHtml() {
    return [
      { tag: 'script', attrs: { src: '/__induforge/eruda.js' }, injectTo: 'head' },
      { tag: 'script', attrs: { src: '/__induforge/preview-devtools.js' }, injectTo: 'head' },
    ]
  },
}

const server = await createServer({
  root: workspaceRoot,
  plugins: [platformPlugin],
  server: { host: '0.0.0.0', port: previewPort, strictPort: true },
})

await server.listen()
server.printUrls()

async function shutdown() {
  await server.close()
  process.exit(0)
}

process.on('SIGTERM', shutdown)
process.on('SIGINT', shutdown)
