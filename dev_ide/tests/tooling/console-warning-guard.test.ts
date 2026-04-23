import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, test } from 'vitest'

const readSource = (relativePath: string) =>
  readFileSync(join(process.cwd(), 'src', ...relativePath.split('/')), 'utf8')

describe('tooling: 控制台告警回归护栏', () => {
  test('登录页切换语言时必须暴露 locale，并为账号密码声明 autocomplete', () => {
    const loginView = readSource('views/auth/Login.vue')

    expect(loginView).toMatch(/return\s*\{[\s\S]*\blocale\b[\s\S]*\}/)
    expect(loginView).toMatch(/id="username"[\s\S]*autocomplete="username"/)
    expect(loginView).toMatch(/id="password"[\s\S]*autocomplete="current-password"/)
  })

  test('Dashboard 不应给 ElTabs 传入函数型 closable，且动态组件需标记为非响应式', () => {
    const dashboardView = readSource('views/Dashboard.vue')

    expect(dashboardView).not.toMatch(/<el-tabs[\s\S]*:closable="\s*\(tab\)\s*=>/)
    expect(dashboardView).toMatch(/<el-tab-pane[\s\S]*:closable="tab\.key !== 'dashboard'"/)
    expect(dashboardView).toMatch(/import\s+\{[^}]*\bmarkRaw\b[^}]*\}\s+from 'vue'/)
    expect(dashboardView).toMatch(/const\s+DashboardContent\s*=\s*markRaw\(defineAsyncComponent\(/)
  })

  test('Dashboard 侧边栏在全屏切换时不应对多根组件使用 v-show', () => {
    const dashboardView = readSource('views/Dashboard.vue')

    expect(dashboardView).toMatch(/<DashboardSidebar[\s\S]*v-if="!isTabMaximized"/)
    expect(dashboardView).not.toMatch(/<DashboardSidebar[\s\S]*v-show=/)
  })

  test('系统日志时间范围筛选控件应声明 name，避免浏览器表单可访问性告警', () => {
    const systemLogsView = readSource('views/tenant/SystemLogs.vue')

    expect(systemLogsView).toMatch(
      /<el-date-picker[\s\S]*:name="\['system-log-time-range-start', 'system-log-time-range-end'\]"/,
    )
  })
})
