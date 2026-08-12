import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'

describe('control socket path', () => {
  it('控制面客户端使用独立 Socket.IO 路径', () => {
    const source = readFileSync('src/utils/socket.ts', 'utf8')
    expect(source).toContain("path: '/control-socket.io'")
    expect(source).not.toContain("path: '/socket.io'")
  })
})
