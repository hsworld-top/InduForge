import { readdirSync, statSync } from 'node:fs'
import { join, extname } from 'node:path'
import { describe, expect, test } from 'vitest'

const SRC_ROOT = join(process.cwd(), 'src')
const ALLOWED_JS_SOURCE_FILES = [
  'api/auth.api.js',
  'api/data.api.js',
  'api/index.js',
  'api/project.api.js',
  'api/system/log.api.js',
  'api/tenant.api.js',
  'api/user.api.js',
  'constants/index.js',
  'enums/index.js',
  'lang/index.js',
  'main.js',
  'permissions/index.js',
  'permissions/rules.js',
  'router/index.js',
  'store/index.js',
  'utils/date.js',
  'utils/index.js',
  'utils/opentiny/composable/http/index.js',
  'utils/opentiny/composable/index.js',
  'utils/opentiny/registry.js',
  'utils/request.js',
  'utils/socket.js',
  'utils/storage.js',
  'utils/validate.js',
] as const
const ALLOWED_JS_SOURCE_SET = new Set<string>(ALLOWED_JS_SOURCE_FILES)

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
  test('src 下不应新增 .js 源文件（允许存量白名单）', () => {
    const jsFiles = collectJsFiles(SRC_ROOT)
    const unexpectedJsFiles = jsFiles.filter((filePath) => !ALLOWED_JS_SOURCE_SET.has(filePath))

    expect(
      unexpectedJsFiles,
      `发现超出白名单的新增 JS 源文件：\n${unexpectedJsFiles.join('\n')}`
    ).toEqual([])
  })
})
