# dev_ide 全量 TS + Vitest 迁移 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在保持 `dev_ide` 运行行为一致的前提下，完成 `src` 全量 TypeScript 化，并把测试入口从 `node tests/run-tests.js` 迁移到 `Vitest`。

**Architecture:** 采用“基建先行、逻辑层先迁、视图层后迁、测试最终收口”的分层迁移。通过 `tests/tooling` 增加迁移护栏（禁止 `src` 残留 `.js`、禁止 Vue 脚本无 `lang="ts"`），确保迁移结果可自动回归。接口、路由、嵌入通信补齐类型边界，同时允许少量过渡类型并集中收敛。

**Tech Stack:** Vue 3、Vite、Pinia、Element Plus、TypeScript、vue-tsc、Vitest、jsdom

---

## File Structure Map

### 工具链与配置（新增/改造）

- Create: `dev_ide/tsconfig.json`
- Create: `dev_ide/vitest.config.ts`
- Create: `dev_ide/src/env.d.ts`
- Create: `dev_ide/tests/setup.ts`
- Modify: `dev_ide/package.json`
- Modify: `dev_ide/eslint.config.js`
- Modify: `dev_ide/vite.config.js` -> `dev_ide/vite.config.ts`

### 类型定义与核心逻辑迁移

- Create: `dev_ide/src/types/api.ts`
- Create: `dev_ide/src/types/auth.ts`
- Create: `dev_ide/src/types/embedded.ts`
- Create: `dev_ide/src/types/index.ts`
- Modify: `dev_ide/src/constants/index.js` -> `dev_ide/src/constants/index.ts`
- Modify: `dev_ide/src/enums/index.js` -> `dev_ide/src/enums/index.ts`
- Modify: `dev_ide/src/permissions/index.js` -> `dev_ide/src/permissions/index.ts`
- Modify: `dev_ide/src/permissions/rules.js` -> `dev_ide/src/permissions/rules.ts`
- Modify: `dev_ide/src/lang/index.js` -> `dev_ide/src/lang/index.ts`
- Modify: `dev_ide/src/store/index.js` -> `dev_ide/src/store/index.ts`
- Modify: `dev_ide/src/router/index.js` -> `dev_ide/src/router/index.ts`
- Modify: `dev_ide/src/main.js` -> `dev_ide/src/main.ts`

### API 与工具层迁移

- Modify: `dev_ide/src/api/auth.api.js` -> `dev_ide/src/api/auth.api.ts`
- Modify: `dev_ide/src/api/data.api.js` -> `dev_ide/src/api/data.api.ts`
- Modify: `dev_ide/src/api/index.js` -> `dev_ide/src/api/index.ts`
- Modify: `dev_ide/src/api/project.api.js` -> `dev_ide/src/api/project.api.ts`
- Modify: `dev_ide/src/api/system/log.api.js` -> `dev_ide/src/api/system/log.api.ts`
- Modify: `dev_ide/src/api/tenant.api.js` -> `dev_ide/src/api/tenant.api.ts`
- Modify: `dev_ide/src/api/user.api.js` -> `dev_ide/src/api/user.api.ts`
- Modify: `dev_ide/src/utils/appUrl.js` -> `dev_ide/src/utils/appUrl.ts`
- Modify: `dev_ide/src/utils/dashboardEntryHandoff.js` -> `dev_ide/src/utils/dashboardEntryHandoff.ts`
- Modify: `dev_ide/src/utils/dashboardTabState.js` -> `dev_ide/src/utils/dashboardTabState.ts`
- Modify: `dev_ide/src/utils/dashboardTabTitle.js` -> `dev_ide/src/utils/dashboardTabTitle.ts`
- Modify: `dev_ide/src/utils/date.js` -> `dev_ide/src/utils/date.ts`
- Modify: `dev_ide/src/utils/embeddedAppBridge.js` -> `dev_ide/src/utils/embeddedAppBridge.ts`
- Modify: `dev_ide/src/utils/embeddedIframeSync.js` -> `dev_ide/src/utils/embeddedIframeSync.ts`
- Modify: `dev_ide/src/utils/index.js` -> `dev_ide/src/utils/index.ts`
- Modify: `dev_ide/src/utils/request.js` -> `dev_ide/src/utils/request.ts`
- Modify: `dev_ide/src/utils/socket.js` -> `dev_ide/src/utils/socket.ts`
- Modify: `dev_ide/src/utils/storage.js` -> `dev_ide/src/utils/storage.ts`
- Modify: `dev_ide/src/utils/validate.js` -> `dev_ide/src/utils/validate.ts`
- Modify: `dev_ide/src/utils/opentiny/registry.js` -> `dev_ide/src/utils/opentiny/registry.ts`
- Modify: `dev_ide/src/utils/opentiny/composable/index.js` -> `dev_ide/src/utils/opentiny/composable/index.ts`
- Modify: `dev_ide/src/utils/opentiny/composable/http/index.js` -> `dev_ide/src/utils/opentiny/composable/http/index.ts`
- Modify: `dev_ide/src/views/tenant/utils/ops-status.js` -> `dev_ide/src/views/tenant/utils/ops-status.ts`

### Vue 组件脚本 TS 化

- Modify: `dev_ide/src/App.vue`
- Modify: `dev_ide/src/components/EmbeddedApp.vue`
- Modify: `dev_ide/src/components/MonacoEditor.vue`
- Modify: `dev_ide/src/views/admin/AdminDashboard.vue`
- Modify: `dev_ide/src/views/admin/TenantManagement.vue`
- Modify: `dev_ide/src/views/auth/Login.vue`
- Modify: `dev_ide/src/views/Dashboard.vue`
- Modify: `dev_ide/src/views/DashboardContent.vue`
- Modify: `dev_ide/src/views/NotFound.vue`
- Modify: `dev_ide/src/views/profile/Profile.vue`
- Modify: `dev_ide/src/views/tenant/OpsManagement.vue`
- Modify: `dev_ide/src/views/tenant/ProjectManagement.vue`
- Modify: `dev_ide/src/views/tenant/SystemLogs.vue`
- Modify: `dev_ide/src/views/tenant/SystemSettings.vue`
- Modify: `dev_ide/src/views/tenant/UserManagement.vue`

### 测试迁移

- Create: `dev_ide/tests/tooling/no-js-source.test.ts`
- Create: `dev_ide/tests/tooling/sfc-lang-ts.test.ts`
- Create: `dev_ide/tests/unit/permissions.rules.test.ts`
- Create: `dev_ide/tests/unit/ops-status.test.ts`
- Create: `dev_ide/tests/unit/lang.messages.test.ts`
- Create: `dev_ide/tests/unit/app-url.test.ts`
- Create: `dev_ide/tests/unit/dashboard-tab-state.test.ts`
- Create: `dev_ide/tests/unit/embedded-bridge.test.ts`
- Modify: `dev_ide/tests/run-tests.js`（保留兼容过渡后删除）
- Delete: `dev_ide/tests/run-tests.js`（最终任务）

---

### Task 1: 建立 TS + Vitest 基线与迁移护栏

**Files:**
- Create: `dev_ide/tsconfig.json`
- Create: `dev_ide/vitest.config.ts`
- Create: `dev_ide/src/env.d.ts`
- Create: `dev_ide/tests/setup.ts`
- Create: `dev_ide/tests/tooling/no-js-source.test.ts`
- Create: `dev_ide/tests/tooling/sfc-lang-ts.test.ts`
- Modify: `dev_ide/package.json`
- Modify: `dev_ide/eslint.config.js`
- Modify: `dev_ide/vite.config.js` -> `dev_ide/vite.config.ts`
- Test: `dev_ide/tests/tooling/no-js-source.test.ts`

- [ ] **Step 1: 写迁移护栏失败用例（先失败）**

```ts
// dev_ide/tests/tooling/no-js-source.test.ts
import { readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const walk = (dir: string): string[] => {
  return readdirSync(dir).flatMap((name) => {
    const fullPath = join(dir, name)
    if (statSync(fullPath).isDirectory()) return walk(fullPath)
    return [fullPath]
  })
}

describe('迁移护栏: src 不允许残留 .js 文件', () => {
  it('src 目录下的业务源码 .js 数量为 0', () => {
    const files = walk(join(process.cwd(), 'src'))
    const jsFiles = files.filter((file) => file.endsWith('.js'))
    expect(jsFiles, `残留 JS 文件: ${jsFiles.join(', ')}`).toHaveLength(0)
  })
})
```

- [ ] **Step 2: 运行护栏用例确认失败**

Run: `pnpm --dir dev_ide test -- tests/tooling/no-js-source.test.ts`
Expected: FAIL，提示存在多个 `src/**/*.js` 文件。

- [ ] **Step 3: 建立 TS/Vitest 配置与脚本**

```json
// dev_ide/package.json (关键片段)
{
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "typecheck": "vue-tsc --noEmit",
    "test": "vitest run",
    "test:watch": "vitest"
  },
  "devDependencies": {
    "typescript": "^5.7.3",
    "vue-tsc": "^2.2.0",
    "vitest": "^3.2.4",
    "jsdom": "^26.1.0",
    "@types/node": "^22.10.6",
    "@typescript-eslint/parser": "^8.19.0"
  }
}
```

```ts
// dev_ide/vitest.config.ts
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

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
    include: ['tests/**/*.test.ts'],
    coverage: {
      reporter: ['text', 'lcov'],
      include: ['src/**/*.{ts,vue}'],
    },
  },
})
```

```json
// dev_ide/tsconfig.json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "noImplicitOverride": true,
    "noUncheckedIndexedAccess": true,
    "allowJs": false,
    "isolatedModules": true,
    "skipLibCheck": true,
    "useDefineForClassFields": true,
    "types": ["vitest/globals", "node"],
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src/**/*.ts", "src/**/*.vue", "src/**/*.d.ts", "tests/**/*.ts", "vitest.config.ts"]
}
```

- [ ] **Step 4: 运行基础验证（护栏继续失败，工具链可用）**

Run: `pnpm --dir dev_ide typecheck`
Expected: PASS（配置生效）。

Run: `pnpm --dir dev_ide test -- tests/tooling/no-js-source.test.ts`
Expected: FAIL（符合预期，说明护栏生效）。

- [ ] **Step 5: 提交基线**

```bash
git add dev_ide/package.json dev_ide/tsconfig.json dev_ide/vitest.config.ts dev_ide/src/env.d.ts dev_ide/tests/setup.ts dev_ide/tests/tooling/no-js-source.test.ts dev_ide/tests/tooling/sfc-lang-ts.test.ts dev_ide/eslint.config.js dev_ide/vite.config.ts
git commit -m "chore(dev_ide): 建立TS与Vitest基线"
```

### Task 2: 建立共享类型并迁移 constants/enums/permissions

**Files:**
- Create: `dev_ide/src/types/api.ts`
- Create: `dev_ide/src/types/auth.ts`
- Create: `dev_ide/src/types/embedded.ts`
- Create: `dev_ide/src/types/index.ts`
- Modify: `dev_ide/src/constants/index.js` -> `dev_ide/src/constants/index.ts`
- Modify: `dev_ide/src/enums/index.js` -> `dev_ide/src/enums/index.ts`
- Modify: `dev_ide/src/permissions/index.js` -> `dev_ide/src/permissions/index.ts`
- Modify: `dev_ide/src/permissions/rules.js` -> `dev_ide/src/permissions/rules.ts`
- Test: `dev_ide/tests/unit/permissions.rules.test.ts`

- [ ] **Step 1: 写权限规则失败用例**

```ts
// dev_ide/tests/unit/permissions.rules.test.ts
import { describe, expect, it } from 'vitest'
import { ROLES } from '@/constants'
import { hasRole, canAccessTab, canManageUsers } from '@/permissions/rules'

describe('权限规则', () => {
  it('hasRole 正常判定', () => {
    expect(hasRole(ROLES.SYSTEM_ADMIN, [ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN])).toBe(true)
    expect(hasRole(ROLES.USER, [ROLES.SYSTEM_ADMIN])).toBe(false)
  })

  it('canAccessTab 正常判定', () => {
    expect(canAccessTab('tenant-management', ROLES.SUPER_ADMIN)).toBe(true)
    expect(canAccessTab('tenant-management', ROLES.SYSTEM_ADMIN)).toBe(false)
  })

  it('能力函数正常判定', () => {
    expect(canManageUsers(ROLES.USER_ADMIN)).toBe(true)
    expect(canManageUsers(ROLES.USER)).toBe(false)
  })
})
```

- [ ] **Step 2: 运行失败用例确认迁移前失败**

Run: `pnpm --dir dev_ide test -- tests/unit/permissions.rules.test.ts`
Expected: FAIL，提示 TS 模块解析或类型错误。

- [ ] **Step 3: 迁移并补齐核心类型**

```ts
// dev_ide/src/types/auth.ts
export type Role =
  | 'SUPER_ADMIN'
  | 'SYSTEM_ADMIN'
  | 'PROJECT_ADMIN'
  | 'OPS_ADMIN'
  | 'USER_ADMIN'
  | 'USER'

export interface UserInfo {
  id: string | number
  username: string
  email?: string
  role: Role
  tenantId?: string | number | null
}
```

```ts
// dev_ide/src/permissions/rules.ts (关键片段)
import { ROLES } from '@/constants'
import type { Role } from '@/types'

export const hasRole = (userRole: Role | '' | undefined, allowedRoles: Role[] = []): boolean => {
  if (!userRole || allowedRoles.length === 0) return false
  return allowedRoles.includes(userRole)
}
```

- [ ] **Step 4: 运行权限测试并修正导入**

Run: `pnpm --dir dev_ide test -- tests/unit/permissions.rules.test.ts`
Expected: PASS。

Run: `pnpm --dir dev_ide typecheck`
Expected: PASS。

- [ ] **Step 5: 提交核心类型任务**

```bash
git add dev_ide/src/types dev_ide/src/constants/index.ts dev_ide/src/enums/index.ts dev_ide/src/permissions/index.ts dev_ide/src/permissions/rules.ts dev_ide/tests/unit/permissions.rules.test.ts
git commit -m "refactor(dev_ide): 迁移权限与常量到TypeScript"
```

### Task 3: 迁移 storage/date/validate/socket/opentiny 工具层

**Files:**
- Modify: `dev_ide/src/utils/storage.js` -> `dev_ide/src/utils/storage.ts`
- Modify: `dev_ide/src/utils/date.js` -> `dev_ide/src/utils/date.ts`
- Modify: `dev_ide/src/utils/validate.js` -> `dev_ide/src/utils/validate.ts`
- Modify: `dev_ide/src/utils/socket.js` -> `dev_ide/src/utils/socket.ts`
- Modify: `dev_ide/src/utils/index.js` -> `dev_ide/src/utils/index.ts`
- Modify: `dev_ide/src/utils/opentiny/registry.js` -> `dev_ide/src/utils/opentiny/registry.ts`
- Modify: `dev_ide/src/utils/opentiny/composable/index.js` -> `dev_ide/src/utils/opentiny/composable/index.ts`
- Modify: `dev_ide/src/utils/opentiny/composable/http/index.js` -> `dev_ide/src/utils/opentiny/composable/http/index.ts`
- Test: `dev_ide/tests/unit/storage.test.ts`

- [ ] **Step 1: 写 Storage 行为失败用例**

```ts
// dev_ide/tests/unit/storage.test.ts
import { describe, expect, it } from 'vitest'
import { Storage } from '@/utils/storage'

describe('Storage', () => {
  it('set/get 可读写 JSON 值', () => {
    Storage.set('k1', { v: 1 })
    expect(Storage.get('k1')).toEqual({ v: 1 })
  })

  it('getTheme 默认返回 light', () => {
    expect(Storage.getTheme()).toBe('light')
  })
})
```

- [ ] **Step 2: 运行 Storage 用例确认失败**

Run: `pnpm --dir dev_ide test -- tests/unit/storage.test.ts`
Expected: FAIL。

- [ ] **Step 3: 迁移工具层并补类型**

```ts
// dev_ide/src/utils/storage.ts (关键片段)
export class Storage {
  static get<T>(key: string, defaultValue: T | null = null): T | null {
    try {
      const item = localStorage.getItem(key)
      return item ? (JSON.parse(item) as T) : defaultValue
    } catch {
      return defaultValue
    }
  }

  static set<T>(key: string, value: T): void {
    localStorage.setItem(key, JSON.stringify(value))
  }
}
```

- [ ] **Step 4: 运行单测和类型检查**

Run: `pnpm --dir dev_ide test -- tests/unit/storage.test.ts`
Expected: PASS。

Run: `pnpm --dir dev_ide typecheck`
Expected: PASS。

- [ ] **Step 5: 提交工具层迁移**

```bash
git add dev_ide/src/utils dev_ide/tests/unit/storage.test.ts
git commit -m "refactor(dev_ide): 迁移工具层到TypeScript"
```

### Task 4: 迁移 request 与全部 API 文件为 .ts

**Files:**
- Modify: `dev_ide/src/utils/request.js` -> `dev_ide/src/utils/request.ts`
- Modify: `dev_ide/src/api/auth.api.js` -> `dev_ide/src/api/auth.api.ts`
- Modify: `dev_ide/src/api/data.api.js` -> `dev_ide/src/api/data.api.ts`
- Modify: `dev_ide/src/api/index.js` -> `dev_ide/src/api/index.ts`
- Modify: `dev_ide/src/api/project.api.js` -> `dev_ide/src/api/project.api.ts`
- Modify: `dev_ide/src/api/system/log.api.js` -> `dev_ide/src/api/system/log.api.ts`
- Modify: `dev_ide/src/api/tenant.api.js` -> `dev_ide/src/api/tenant.api.ts`
- Modify: `dev_ide/src/api/user.api.js` -> `dev_ide/src/api/user.api.ts`
- Test: `dev_ide/tests/unit/auth-api.test.ts`

- [ ] **Step 1: 写 API 组装失败用例**

```ts
// dev_ide/tests/unit/auth-api.test.ts
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/utils/request', () => ({
  default: {
    post: vi.fn().mockResolvedValue({ data: { token: 't' } }),
    get: vi.fn(),
    put: vi.fn(),
  },
}))

import { authAPI } from '@/api/auth.api'

describe('authAPI', () => {
  it('login 透传认证参数', async () => {
    const res = await authAPI.login({ username: 'u', password: 'p', captchaKey: 'k', captchaCode: 'c' })
    expect(res.data.token).toBe('t')
  })
})
```

- [ ] **Step 2: 运行 API 用例确认失败**

Run: `pnpm --dir dev_ide test -- tests/unit/auth-api.test.ts`
Expected: FAIL。

- [ ] **Step 3: 迁移请求层并定义响应类型**

```ts
// dev_ide/src/types/api.ts
export interface ApiResponse<T> {
  data: T
  message?: string
  success?: boolean
}

export interface PageResponse<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}
```

```ts
// dev_ide/src/utils/request.ts (关键片段)
import axios, { AxiosError, type AxiosRequestConfig } from 'axios'

interface RetryConfig extends AxiosRequestConfig {
  _retry?: boolean
  forcePermissionToast?: boolean
  skipPermissionToast?: boolean
}

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

request.interceptors.response.use(
  (response) => response.data,
  (error: AxiosError) => Promise.reject(error),
)
```

- [ ] **Step 4: 运行 API 用例与类型检查**

Run: `pnpm --dir dev_ide test -- tests/unit/auth-api.test.ts`
Expected: PASS。

Run: `pnpm --dir dev_ide typecheck`
Expected: PASS。

- [ ] **Step 5: 提交 API 迁移**

```bash
git add dev_ide/src/utils/request.ts dev_ide/src/api dev_ide/src/types/api.ts dev_ide/tests/unit/auth-api.test.ts
git commit -m "refactor(dev_ide): 迁移请求与API到TypeScript"
```

### Task 5: 迁移嵌入桥接与 dashboard 辅助工具（含 ops-status）

**Files:**
- Modify: `dev_ide/src/utils/appUrl.js` -> `dev_ide/src/utils/appUrl.ts`
- Modify: `dev_ide/src/utils/dashboardEntryHandoff.js` -> `dev_ide/src/utils/dashboardEntryHandoff.ts`
- Modify: `dev_ide/src/utils/dashboardTabState.js` -> `dev_ide/src/utils/dashboardTabState.ts`
- Modify: `dev_ide/src/utils/dashboardTabTitle.js` -> `dev_ide/src/utils/dashboardTabTitle.ts`
- Modify: `dev_ide/src/utils/embeddedAppBridge.js` -> `dev_ide/src/utils/embeddedAppBridge.ts`
- Modify: `dev_ide/src/utils/embeddedIframeSync.js` -> `dev_ide/src/utils/embeddedIframeSync.ts`
- Modify: `dev_ide/src/views/tenant/utils/ops-status.js` -> `dev_ide/src/views/tenant/utils/ops-status.ts`
- Test: `dev_ide/tests/unit/embedded-bridge.test.ts`

- [ ] **Step 1: 写桥接恢复失败用例**

```ts
// dev_ide/tests/unit/embedded-bridge.test.ts
import { describe, expect, it } from 'vitest'
import { createRestoredEmbeddedTab } from '@/utils/dashboardEntryHandoff'

describe('dashboardEntryHandoff', () => {
  it('可恢复 designer 标签', () => {
    const tab = createRestoredEmbeddedTab({
      handoffId: 'h1',
      appType: 'designer',
      projectId: 'p1',
      tenantId: 't1',
    })
    expect(tab?.key).toBe('design-center-p1')
  })
})
```

- [ ] **Step 2: 运行桥接用例确认失败**

Run: `pnpm --dir dev_ide test -- tests/unit/embedded-bridge.test.ts`
Expected: FAIL。

- [ ] **Step 3: 为 postMessage 与 handoff 增加联合类型**

```ts
// dev_ide/src/types/embedded.ts
export type EmbeddedAppType = 'designer' | 'datacenter'

export interface EmbeddedRestorePayload {
  handoffId: string
  appType: EmbeddedAppType
  projectId: string
  tenantId: string
}

export type EmbeddedHostMessage =
  | { type: 'APP_BOOTSTRAP_REQUEST' }
  | { type: 'AUTH_EXPIRED' }
  | { type: 'AUTH_REFRESHED'; payload: { token: string; refreshToken?: string } }
  | { type: 'THEME_UPDATE'; theme: 'light' | 'dark' }
  | { type: 'LOCALE_UPDATE'; locale: 'zh' | 'en' }
```

- [ ] **Step 4: 运行桥接与运维状态测试**

Run: `pnpm --dir dev_ide test -- tests/unit/embedded-bridge.test.ts`
Expected: PASS。

Run: `pnpm --dir dev_ide test -- tests/unit/ops-status.test.ts`
Expected: PASS。

- [ ] **Step 5: 提交嵌入桥接迁移**

```bash
git add dev_ide/src/utils/appUrl.ts dev_ide/src/utils/dashboardEntryHandoff.ts dev_ide/src/utils/dashboardTabState.ts dev_ide/src/utils/dashboardTabTitle.ts dev_ide/src/utils/embeddedAppBridge.ts dev_ide/src/utils/embeddedIframeSync.ts dev_ide/src/views/tenant/utils/ops-status.ts dev_ide/src/types/embedded.ts dev_ide/tests/unit/embedded-bridge.test.ts dev_ide/tests/unit/ops-status.test.ts
git commit -m "refactor(dev_ide): 迁移嵌入桥接与运维工具到TypeScript"
```

### Task 6: 迁移 lang/router/store/main 并收口应用入口类型

**Files:**
- Modify: `dev_ide/src/lang/index.js` -> `dev_ide/src/lang/index.ts`
- Modify: `dev_ide/src/router/index.js` -> `dev_ide/src/router/index.ts`
- Modify: `dev_ide/src/store/index.js` -> `dev_ide/src/store/index.ts`
- Modify: `dev_ide/src/main.js` -> `dev_ide/src/main.ts`
- Test: `dev_ide/tests/unit/router-guard.test.ts`
- Test: `dev_ide/tests/unit/lang.messages.test.ts`

- [ ] **Step 1: 写路由守卫失败用例**

```ts
// dev_ide/tests/unit/router-guard.test.ts
import { describe, expect, it } from 'vitest'
import router from '@/router'

describe('router', () => {
  it('包含 login 路由', () => {
    const route = router.getRoutes().find((item) => item.path === '/login')
    expect(route).toBeTruthy()
  })
})
```

- [ ] **Step 2: 运行守卫用例确认失败**

Run: `pnpm --dir dev_ide test -- tests/unit/router-guard.test.ts`
Expected: FAIL。

- [ ] **Step 3: 迁移入口与全局类型边界**

```ts
// dev_ide/src/router/index.ts (关键片段)
import type { RouteRecordRaw } from 'vue-router'
import type { Role } from '@/types'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    titleKey?: string
    requiresAuth?: boolean
    roles?: Role[]
  }
}

const routes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: () => import('@/views/auth/Login.vue') },
]
```

```ts
// dev_ide/src/main.ts (关键片段)
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import i18n from './lang'

createApp(App).use(router).use(i18n).mount('#app')
```

- [ ] **Step 4: 运行路由与语言测试、类型检查**

Run: `pnpm --dir dev_ide test -- tests/unit/router-guard.test.ts tests/unit/lang.messages.test.ts`
Expected: PASS。

Run: `pnpm --dir dev_ide typecheck`
Expected: PASS。

- [ ] **Step 5: 提交应用壳层迁移**

```bash
git add dev_ide/src/lang/index.ts dev_ide/src/router/index.ts dev_ide/src/store/index.ts dev_ide/src/main.ts dev_ide/tests/unit/router-guard.test.ts dev_ide/tests/unit/lang.messages.test.ts
git commit -m "refactor(dev_ide): 迁移应用入口与状态路由到TypeScript"
```

### Task 7: 迁移基础 Vue 组件脚本到 `lang="ts"`

**Files:**
- Modify: `dev_ide/src/App.vue`
- Modify: `dev_ide/src/components/EmbeddedApp.vue`
- Modify: `dev_ide/src/components/MonacoEditor.vue`
- Modify: `dev_ide/src/views/admin/AdminDashboard.vue`
- Modify: `dev_ide/src/views/admin/TenantManagement.vue`
- Modify: `dev_ide/src/views/auth/Login.vue`
- Modify: `dev_ide/src/views/NotFound.vue`
- Modify: `dev_ide/src/views/profile/Profile.vue`
- Modify: `dev_ide/src/views/tenant/OpsManagement.vue`
- Test: `dev_ide/tests/tooling/sfc-lang-ts.test.ts`

- [ ] **Step 1: 写 SFC `lang="ts"` 护栏失败用例**

```ts
// dev_ide/tests/tooling/sfc-lang-ts.test.ts
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const walk = (dir: string): string[] =>
  readdirSync(dir).flatMap((name) => {
    const fullPath = join(dir, name)
    if (statSync(fullPath).isDirectory()) return walk(fullPath)
    return [fullPath]
  })

describe('迁移护栏: Vue 脚本必须使用 lang="ts"', () => {
  it('所有 .vue 文件的 script 均为 TS', () => {
    const vueFiles = walk(join(process.cwd(), 'src')).filter((file) => file.endsWith('.vue'))
    const nonTsScriptFiles = vueFiles.filter((file) => {
      const content = readFileSync(file, 'utf-8')
      return /<script(?![^>]*lang=['\"]ts['\"])/.test(content)
    })
    expect(nonTsScriptFiles, `未迁移的 Vue 文件: ${nonTsScriptFiles.join(', ')}`).toHaveLength(0)
  })
})
```

- [ ] **Step 2: 运行 SFC 护栏用例确认失败**

Run: `pnpm --dir dev_ide test -- tests/tooling/sfc-lang-ts.test.ts`
Expected: FAIL，提示多个 `.vue` 仍为 JS 脚本。

- [ ] **Step 3: 迁移基础组件脚本并显式 props/emits 类型**

```vue
<!-- dev_ide/src/components/EmbeddedApp.vue (关键片段) -->
<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { buildAppEntry } from '@/utils/appUrl'
import type { EmbeddedAppType } from '@/types'

interface EmbeddedProjectRef {
  id: string
  tenantId: string
  name?: string
}

const props = defineProps<{
  appType: EmbeddedAppType
  project: EmbeddedProjectRef
  tabKey?: string
}>()
</script>
```

- [ ] **Step 4: 运行 SFC 护栏与类型检查**

Run: `pnpm --dir dev_ide test -- tests/tooling/sfc-lang-ts.test.ts`
Expected: 仍可能 FAIL（剩余复杂页面未迁移，符合预期）。

Run: `pnpm --dir dev_ide typecheck`
Expected: PASS（当前已迁移文件无新增类型错误）。

- [ ] **Step 5: 提交基础组件 TS 化**

```bash
git add dev_ide/src/App.vue dev_ide/src/components/EmbeddedApp.vue dev_ide/src/components/MonacoEditor.vue dev_ide/src/views/admin/AdminDashboard.vue dev_ide/src/views/admin/TenantManagement.vue dev_ide/src/views/auth/Login.vue dev_ide/src/views/NotFound.vue dev_ide/src/views/profile/Profile.vue dev_ide/src/views/tenant/OpsManagement.vue dev_ide/tests/tooling/sfc-lang-ts.test.ts
git commit -m "refactor(dev_ide): 迁移基础视图组件脚本到TypeScript"
```

### Task 8: 迁移复杂 Vue 页面（Dashboard / DashboardContent / ProjectManagement / System 页面）

**Files:**
- Modify: `dev_ide/src/views/Dashboard.vue`
- Modify: `dev_ide/src/views/DashboardContent.vue`
- Modify: `dev_ide/src/views/tenant/ProjectManagement.vue`
- Modify: `dev_ide/src/views/tenant/SystemLogs.vue`
- Modify: `dev_ide/src/views/tenant/SystemSettings.vue`
- Modify: `dev_ide/src/views/tenant/UserManagement.vue`
- Test: `dev_ide/tests/unit/dashboard-tab-state.test.ts`
- Test: `dev_ide/tests/unit/app-url.test.ts`

- [ ] **Step 1: 写复杂页面依赖的核心回归用例**

```ts
// dev_ide/tests/unit/dashboard-tab-state.test.ts
import { describe, expect, it } from 'vitest'
import { serializeDashboardTabState, restoreDashboardTabState } from '@/utils/dashboardTabState'

describe('dashboardTabState', () => {
  it('可序列化并恢复 embedded tab', () => {
    const state = serializeDashboardTabState({
      tabs: [{ key: 'data-center-p2', titleKey: 'projectManagement.dataCenter', component: {}, icon: 'database', props: { appType: 'datacenter', project: { id: 'p2', tenantId: 't2', name: '项目2' } } }],
      activeTab: 'data-center-p2',
      tabConfigMap: {},
      hasTabPermission: () => true,
    })
    const restored = restoreDashboardTabState(state, { tabConfigMap: {}, hasTabPermission: () => true, embeddedComponent: {}, translate: (k: string) => k })
    expect(restored?.activeTab).toBe('data-center-p2')
  })
})
```

- [ ] **Step 2: 运行复杂页面回归用例确认失败**

Run: `pnpm --dir dev_ide test -- tests/unit/dashboard-tab-state.test.ts tests/unit/app-url.test.ts`
Expected: FAIL。

- [ ] **Step 3: 迁移复杂页面脚本并约束事件/状态类型**

```vue
<!-- dev_ide/src/views/Dashboard.vue (关键片段) -->
<script lang="ts">
import { defineComponent, ref, computed } from 'vue'
import type { EmbeddedHostMessage } from '@/types'

export default defineComponent({
  setup() {
    const activeTab = ref<string>('dashboard')

    const handleEmbeddedMessage = (event: MessageEvent<EmbeddedHostMessage>) => {
      if (!event.data?.type) return
    }

    return { activeTab, handleEmbeddedMessage }
  },
})
</script>
```

- [ ] **Step 4: 运行复杂回归测试与护栏**

Run: `pnpm --dir dev_ide test -- tests/unit/dashboard-tab-state.test.ts tests/unit/app-url.test.ts tests/tooling/sfc-lang-ts.test.ts`
Expected: PASS（此时 SFC 护栏应通过）。

Run: `pnpm --dir dev_ide typecheck`
Expected: PASS。

- [ ] **Step 5: 提交复杂页面迁移**

```bash
git add dev_ide/src/views/Dashboard.vue dev_ide/src/views/DashboardContent.vue dev_ide/src/views/tenant/ProjectManagement.vue dev_ide/src/views/tenant/SystemLogs.vue dev_ide/src/views/tenant/SystemSettings.vue dev_ide/src/views/tenant/UserManagement.vue dev_ide/tests/unit/dashboard-tab-state.test.ts dev_ide/tests/unit/app-url.test.ts
git commit -m "refactor(dev_ide): 迁移复杂页面脚本到TypeScript"
```

### Task 9: 拆分并迁移 run-tests.js 用例到 Vitest

**Files:**
- Create: `dev_ide/tests/unit/ops-status.test.ts`
- Create: `dev_ide/tests/unit/lang.messages.test.ts`
- Create: `dev_ide/tests/unit/app-url.test.ts`
- Create: `dev_ide/tests/unit/embedded-bridge.test.ts`
- Delete: `dev_ide/tests/run-tests.js`
- Test: `dev_ide/tests/unit/*.test.ts`

- [ ] **Step 1: 从旧测试提取第一个失败用例**

```ts
// dev_ide/tests/unit/ops-status.test.ts
import { describe, expect, it } from 'vitest'
import { buildDeployStatusSummary } from '@/views/tenant/utils/ops-status'

describe('ops-status', () => {
  it('聚合运行状态统计', () => {
    const summary = buildDeployStatusSummary([
      { deployments: [{ status: 'running' }, { status: 'failed' }] },
      { deployments: [{ status: 'pending' }] },
    ])
    expect(summary).toEqual({ running: 1, deploying: 1, stopped: 0, failed: 1 })
  })
})
```

- [ ] **Step 2: 运行分组测试确认失败**

Run: `pnpm --dir dev_ide test -- tests/unit/ops-status.test.ts`
Expected: FAIL。

- [ ] **Step 3: 完整迁移旧 run-tests 覆盖点并删除旧入口**

```ts
// dev_ide/tests/unit/lang.messages.test.ts (关键片段)
import { describe, expect, it } from 'vitest'
import { messages } from '@/lang'

describe('i18n messages', () => {
  it('中英文核心文案存在', () => {
    expect(messages.zh.dashboard.title).toBe('仪表盘')
    expect(messages.en.dashboard.title).toBe('Dashboard')
  })
})
```

- [ ] **Step 4: 运行全量测试验证替换完成**

Run: `pnpm --dir dev_ide test`
Expected: PASS，包含 `tests/tooling` 与 `tests/unit`。

- [ ] **Step 5: 提交测试迁移**

```bash
git add dev_ide/tests
git rm dev_ide/tests/run-tests.js
git commit -m "test(dev_ide): 迁移测试体系到Vitest"
```

### Task 10: 最终收口（清零 src JS、全量构建验证、迁移报告）

**Files:**
- Modify: `dev_ide/src/**/*`（仅用于清理残留导入或少量过渡类型）
- Modify: `dev_ide/docs`（如需补充 `Vitest` 运行说明）
- Test: `dev_ide/tests/tooling/no-js-source.test.ts`

- [ ] **Step 1: 运行“无 JS 残留”护栏并确认失败点**

Run: `pnpm --dir dev_ide test -- tests/tooling/no-js-source.test.ts`
Expected: 若仍有残留，FAIL 并输出残留文件列表。

- [ ] **Step 2: 清理所有 `src/**/*.js` 与导入后缀问题**

```ts
// 典型清理示例：将 JS 导入改为 TS 无后缀路径
import { authAPI } from '@/api/auth.api'
import { Storage } from '@/utils/storage'
```

- [ ] **Step 3: 全量验证**

Run: `pnpm --dir dev_ide typecheck`
Expected: PASS。

Run: `pnpm --dir dev_ide test`
Expected: PASS。

Run: `pnpm --dir dev_ide build`
Expected: PASS。

- [ ] **Step 4: 产出过渡类型清单（允许少量）**

```md
# dev_ide TS 过渡清单

1. src/views/tenant/ProjectManagement.vue: `deployForm` 临时使用 `Record<string, unknown>`
2. src/utils/request.ts: 刷新队列 token 回调参数暂用 `string | null`
```

- [ ] **Step 5: 最终提交**

```bash
git add dev_ide/src dev_ide/tests dev_ide/package.json dev_ide/vitest.config.ts dev_ide/tsconfig.json
if (Test-Path dev_ide/docs/README.md) { git add dev_ide/docs/README.md }
git commit -m "feat(dev_ide): 完成全量TS化并切换Vitest"
```

---

## Self-Review

1. **Spec coverage 检查**
- `src` 全量 TS 化：Task 2-8 + Task 10 覆盖。
- `Vitest` 替换旧测试：Task 1 + Task 9 覆盖。
- `*.api.ts` 迁移：Task 4 覆盖。
- 允许少量过渡类型并可追踪：Task 10 Step 4 覆盖。

2. **Placeholder scan 检查**
- 计划全文未出现任何待补全占位描述。
- 每个代码步骤都给出明确代码片段。

3. **Type consistency 检查**
- `Role`、`ApiResponse`、`EmbeddedHostMessage` 在任务中统一命名。
- Router/Store/API 任务中的类型名称与前置类型任务一致。

