import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, test } from 'vitest'

const SRC_ROOT = join(process.cwd(), 'src')
const ALLOWED_NON_TS_SFC_FILES = [
  'App.vue',
  'components/EmbeddedApp.vue',
  'components/MonacoEditor.vue',
  'views/Dashboard.vue',
  'views/DashboardContent.vue',
  'views/NotFound.vue',
  'views/admin/AdminDashboard.vue',
  'views/admin/TenantManagement.vue',
  'views/auth/Login.vue',
  'views/profile/Profile.vue',
  'views/tenant/OpsManagement.vue',
  'views/tenant/ProjectManagement.vue',
  'views/tenant/SystemLogs.vue',
  'views/tenant/SystemSettings.vue',
  'views/tenant/UserManagement.vue',
] as const
const ALLOWED_NON_TS_SFC_SET = new Set<string>(ALLOWED_NON_TS_SFC_FILES)

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
  test('Vue SFC 不应新增未声明 <script lang="ts"> 的文件（允许存量白名单）', () => {
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

    const unexpectedVueFiles = invalidVueFiles.filter(
      (filePath) => !ALLOWED_NON_TS_SFC_SET.has(filePath)
    )

    expect(
      unexpectedVueFiles,
      `以下 Vue 文件超出未 TS 化白名单：\n${unexpectedVueFiles.join('\n')}`
    ).toEqual([])
  })
})
