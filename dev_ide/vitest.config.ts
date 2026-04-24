import dns from 'node:dns'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'

// 当前桌面环境里 localhost 可能无法通过系统 DNS 解析。
// Vitest 启动阶段会提前做主机解析，这里仅对测试进程补一个 localhost -> 127.0.0.1 的兼容映射，
// 避免单测在真正执行前就因 getaddrinfo EAI_FAIL 失败。
const originalLookup = dns.lookup.bind(dns)
const originalPromiseLookup = dns.promises.lookup.bind(dns.promises)
const normalizeHost = (hostname: string) => (hostname === 'localhost' ? '127.0.0.1' : hostname)

dns.lookup = ((hostname: string, ...rest: unknown[]) =>
  originalLookup(normalizeHost(hostname), ...(rest as []))) as typeof dns.lookup

dns.promises.lookup = ((hostname: string, ...rest: unknown[]) =>
  originalPromiseLookup(normalizeHost(hostname), ...(rest as []))) as typeof dns.promises.lookup

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./tests/setup.ts'],
  },
})
