import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('应用启动认证恢复', () => {
  it('认证 API 使用静态依赖，避免生产构建在挂载前形成动态导入循环', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/store/index.ts'), 'utf8')
    const bootstrapStores = source.split('// 租户状态管理')[0]

    expect(bootstrapStores).toMatch(
      /import\s*{[\s\S]*authAPI[\s\S]*}\s*from\s*['"]@\/api\/auth\.api['"]/,
    )
    expect(bootstrapStores).not.toMatch(/await\s+import\(['"]@\/api['"]\)/)
  })
})
