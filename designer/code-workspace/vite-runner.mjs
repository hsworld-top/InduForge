import path from 'node:path'
import process from 'node:process'
import { pathToFileURL } from 'node:url'

const workspaceRoot = path.resolve(process.env.WORKSPACE_ROOT || '/workspace')
const previewPort = Number.parseInt(process.env.VITE_PORT || '5173', 10)
const viteEntry = path.join(workspaceRoot, 'node_modules', 'vite', 'dist', 'node', 'index.js')
const { createServer } = await import(pathToFileURL(viteEntry).href)

const server = await createServer({
  root: workspaceRoot,
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
