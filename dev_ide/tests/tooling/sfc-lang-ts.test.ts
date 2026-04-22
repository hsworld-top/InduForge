import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, test } from 'vitest'

const SRC_ROOT = join(process.cwd(), 'src')

const collectVueFiles = (rootDir: string): string[] => {
  const vueFiles: string[] = []
  const pendingDirs: string[] = [rootDir]

  while (pendingDirs.length > 0) {
    const currentDir = pendingDirs.pop()
    if (!currentDir) {
      continue
    }
    const entries = readdirSync(currentDir)

    for (const entryName of entries) {
      const entryPath = join(currentDir, entryName)
      const entryStat = statSync(entryPath)

      if (entryStat.isDirectory()) {
        pendingDirs.push(entryPath)
        continue
      }

      if (entryPath.endsWith('.vue')) {
        vueFiles.push(entryPath)
      }
    }
  }

  return vueFiles.sort()
}

describe('tooling: SFC 脚本语言护栏', () => {
  test('Vue SFC 应使用 <script lang="ts">', () => {
    const invalidVueFiles: string[] = []

    for (const vueFile of collectVueFiles(SRC_ROOT)) {
      const content = readFileSync(vueFile, 'utf8')
      const hasScriptBlock = /<script\b[^>]*>/i.test(content)

      if (!hasScriptBlock) {
        continue
      }

      const hasTsScript = /<script\b[^>]*lang\s*=\s*["']ts["'][^>]*>/i.test(content)
      if (!hasTsScript) {
        invalidVueFiles.push(vueFile.replace(SRC_ROOT + '\\', '').replaceAll('\\', '/'))
      }
    }

    expect(
      invalidVueFiles,
      `以下 Vue 文件仍未声明 <script lang=\"ts\">：\n${invalidVueFiles.join('\n')}`
    ).toEqual([])
  })
})
