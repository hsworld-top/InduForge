import { readdirSync, statSync } from 'node:fs'
import { join, extname } from 'node:path'
import { describe, expect, test } from 'vitest'

const SRC_ROOT = join(process.cwd(), 'src')

const collectJsFiles = (rootDir: string): string[] => {
  const jsFiles: string[] = []
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

      if (extname(entryPath) === '.js') {
        jsFiles.push(entryPath.replace(rootDir + '\\', '').replaceAll('\\', '/'))
      }
    }
  }

  return jsFiles.sort()
}

describe('tooling: src 目录 JS 源文件护栏', () => {
  test('src 下不应新增 .js 源文件', () => {
    const jsFiles = collectJsFiles(SRC_ROOT)

    expect(jsFiles, `发现待迁移的 JS 源文件：\n${jsFiles.join('\n')}`).toEqual([])
  })
})
