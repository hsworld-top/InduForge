import { defineStore } from 'pinia'
import { Storage } from '@/utils'
import type { Role, UserInfo } from '@/types/auth'
import {
  authAPI,
  type AuthConfigPayload,
  type AuthLoginPayload,
  type AuthUserPayload,
} from '@/api/auth.api'
import { getApiErrorMessage } from '@/utils/request'
import { applyTenantBrowserBrand } from '@/utils/tenantBrand'
import defaultLogoUrl from '@/assets/images/default-logo.svg'
import defaultLoginBgUrl from '@/assets/images/default-login-bg.svg'

type TenantId = string | number

type LoginCredentials = {
  username: string
  password: string
  sliderChallengeId?: string
  sliderOffset?: number
  tenantCode?: string
}

type LoginResult = {
  success: boolean
  error?: string
}

type TenantRecord = Record<string, unknown> & {
  id?: TenantId
  code?: string
  name?: string
  logoUrl?: string
  status?: string
}

type AppConfig = Record<string, unknown> & {
  name?: string
  description?: string
  logoUrl?: string
  loginBackgroundUrl?: string
  multiTenant?: boolean
  allowRegistration?: boolean
  defaultLanguage?: string
  supportedLanguages?: string[]
  features?: {
    captcha: boolean
    emailVerification: boolean
    twoFactorAuth: boolean
  }
}

type AuthStoreState = {
  userInfo: UserInfo | null
  isAuthenticated: boolean
}

type AppStoreState = {
  theme: string
  language: string
  loading: boolean
  sidebarCollapsed: boolean
  config: AppConfig | null
}

type TenantStoreState = {
  currentTenant: TenantRecord | null
  tenants: TenantRecord[]
  loading: boolean
}

const normalizeStoredUserInfo = (): UserInfo | null => {
  const rawUserInfo = Storage.getUserInfo()
  if (!rawUserInfo) {
    return null
  }

  return rawUserInfo as unknown as UserInfo
}

const resolveErrorMessage = (error: unknown): string => {
  return getApiErrorMessage(error, '未知错误')
}

const buildDefaultAppConfig = (): AppConfig => ({
  name: 'InduForge',
  description: '高效、安全的企业级解决方案',
  logoUrl: defaultLogoUrl,
  loginBackgroundUrl: defaultLoginBgUrl,
  multiTenant: false,
  allowRegistration: false,
  defaultLanguage: 'zh',
  supportedLanguages: ['zh', 'en'],
  features: {
    captcha: true,
    emailVerification: false,
    twoFactorAuth: false,
  },
})

const unwrapApiData = <T>(response: unknown): T | null => {
  if (!response || typeof response !== 'object') {
    return null
  }

  let currentData: unknown = (response as { data?: unknown }).data

  // 兼容 axios 原生响应：AxiosResponse<ApiResponse<T>>
  if (
    currentData &&
    typeof currentData === 'object' &&
    'data' in (currentData as Record<string, unknown>) &&
    'code' in (currentData as Record<string, unknown>)
  ) {
    currentData = (currentData as { data?: unknown }).data
  }

  // 兼容历史包一层 data 的返回
  if (
    currentData &&
    typeof currentData === 'object' &&
    'data' in (currentData as Record<string, unknown>) &&
    'code' in (currentData as Record<string, unknown>)
  ) {
    currentData = (currentData as { data?: unknown }).data
  }

  return (currentData as T | undefined) ?? null
}

const normalizeAuthUser = (user: AuthUserPayload): UserInfo | null => {
  const nestedTenantId =
    typeof user === 'object' && user !== null && 'tenant' in user
      ? ((user as { tenant?: { id?: TenantId } }).tenant?.id ?? null)
      : null

  if (!user?.id || !user.username || !user.role) {
    return null
  }

  return {
    id: user.id,
    username: user.username,
    email: user.email,
    role: user.role as Role,
    tenantId: nestedTenantId ?? user.tenantId ?? null,
    tenant: user.tenant,
  }
}

// 认证状态管理
export const useAuthStore = defineStore('auth', {
  state: (): AuthStoreState => ({
    userInfo: normalizeStoredUserInfo(),
    isAuthenticated: Boolean(normalizeStoredUserInfo()),
  }),

  getters: {
    getUserRole: (state): Role | null => state.userInfo?.role ?? null,
    getUserId: (state): UserInfo['id'] | null => state.userInfo?.id ?? null,
    getUsername: (state): string => state.userInfo?.username ?? '',
    isAdmin: (state): boolean =>
      ['SUPER_ADMIN', 'SYSTEM_ADMIN', 'PROJECT_ADMIN', 'OPS_ADMIN', 'USER_ADMIN'].includes(
        state.userInfo?.role ?? '',
      ),
  },

  actions: {
    async login(credentials: LoginCredentials): Promise<LoginResult> {
      try {
        const response = await authAPI.login(credentials)
        const payload = unwrapApiData<AuthLoginPayload>(response)
        const user = payload?.user
        const normalizedUser = user ? normalizeAuthUser(user) : null
        if (!normalizedUser) {
          throw new Error('登录响应缺少必要字段')
        }

        this.setAuthData(normalizedUser)

        return { success: true }
      } catch (error) {
        return { success: false, error: resolveErrorMessage(error) }
      }
    },

    async restoreSession(): Promise<boolean> {
      try {
        const response = await authAPI.getCurrentUser({ skipAuthRedirect: true })
        const user = unwrapApiData<AuthUserPayload>(response)
        const normalizedUser = user ? normalizeAuthUser(user) : null
        if (!normalizedUser) {
          this.clearAuthData()
          return false
        }
        this.setAuthData(normalizedUser)
        return true
      } catch {
        this.clearAuthData()
        return false
      }
    },

    async logout(): Promise<void> {
      try {
        await authAPI.logout()
      } finally {
        this.clearAuthData()
      }
    },

    setAuthData(userInfo: UserInfo): void {
      this.userInfo = userInfo
      this.isAuthenticated = true

      Storage.setUserInfo(userInfo as unknown as Record<string, unknown>)
      if (userInfo.tenantId !== null && userInfo.tenantId !== undefined) {
        Storage.setTenantId(String(userInfo.tenantId))
      }
    },

    clearAuthData(): void {
      this.userInfo = null
      this.isAuthenticated = false

      Storage.remove('user_info')
      Storage.remove('tenant_id')
    },
  },
})

// 应用配置状态管理
export const useAppStore = defineStore('app', {
  state: (): AppStoreState => ({
    theme: Storage.getTheme(),
    language: Storage.getLanguage(),
    loading: false,
    sidebarCollapsed: Storage.getSidebarCollapsed(),
    config: null,
  }),

  getters: {
    isDark: (state): boolean => state.theme === 'dark',
    isLight: (state): boolean => state.theme === 'light',
  },

  actions: {
    setTheme(theme: string): void {
      this.theme = theme
      Storage.setTheme(theme)

      // 应用主题到 body
      document.documentElement.classList.toggle('dark', theme === 'dark')
      document.documentElement.setAttribute('data-theme', theme)
    },

    setLanguage(language: string): void {
      this.language = language
      Storage.setLanguage(language)
    },

    setLoading(loading: boolean): void {
      this.loading = loading
    },

    toggleSidebar(): void {
      this.sidebarCollapsed = !this.sidebarCollapsed
      Storage.setSidebarCollapsed(this.sidebarCollapsed)
    },

    setSidebarCollapsed(collapsed: boolean): void {
      this.sidebarCollapsed = !!collapsed
      Storage.setSidebarCollapsed(this.sidebarCollapsed)
    },

    setConfig(config: AppConfig): void {
      this.config = config
    },

    async loadConfig(tenantCode?: string): Promise<void> {
      try {
        const response = await authAPI.getConfig(tenantCode)
        const config = unwrapApiData<AuthConfigPayload>(response)
        const mergedConfig: AppConfig = {
          ...buildDefaultAppConfig(),
          ...(config ?? {}),
          name: config?.tenantName || config?.appName || config?.name || 'InduForge',
          logoUrl: config?.logoUrl || defaultLogoUrl,
          loginBackgroundUrl: config?.loginBackgroundUrl || defaultLoginBgUrl,
          multiTenant: config?.multiTenant || false,
        }

        this.setConfig(mergedConfig)
        applyTenantBrowserBrand({
          name: mergedConfig.name,
          logoUrl: mergedConfig.logoUrl,
        })
      } catch (error) {
        console.error('Failed to load config:', error)
        const fallbackConfig = buildDefaultAppConfig()
        this.setConfig(fallbackConfig)
        applyTenantBrowserBrand({
          name: fallbackConfig.name,
          logoUrl: fallbackConfig.logoUrl,
        })
      }
    },
  },
})

// 租户状态管理
export const useTenantStore = defineStore('tenant', {
  state: (): TenantStoreState => ({
    currentTenant: null,
    tenants: [],
    loading: false,
  }),

  getters: {
    getTenantById:
      (state) =>
      (id: TenantId): TenantRecord | undefined =>
        state.tenants.find((tenant) => tenant.id === id),
    activeTenants: (state): TenantRecord[] =>
      state.tenants.filter((tenant) => tenant.status === 'active'),
  },

  actions: {
    async fetchCurrentTenant(): Promise<TenantRecord | null> {
      const { tenantAPI } = await import('@/api')
      const response = await tenantAPI.getCurrentTenant()
      const responseData = (response as { data?: { tenant?: TenantRecord } }).data
      const tenant = responseData?.tenant ?? null
      if (tenant) {
        this.setCurrentTenant(tenant)
      }
      return tenant
    },

    async fetchTenants(): Promise<void> {
      this.loading = true
      try {
        const { tenantAPI } = await import('@/api')
        const response = await tenantAPI.getTenants({ page: 1, limit: 200 })
        const tenants = (response as { data?: { tenants?: TenantRecord[] } }).data?.tenants
        this.tenants = Array.isArray(tenants) ? tenants : []
      } catch (error) {
        console.error('Failed to fetch tenants:', error)
        this.tenants = []
      } finally {
        this.loading = false
      }
    },

    async createTenant(tenantData: Record<string, unknown>): Promise<TenantRecord | undefined> {
      const { tenantAPI } = await import('@/api')
      const response = await tenantAPI.createTenant(tenantData)
      const responseData = (response as { data?: TenantRecord & { tenant?: TenantRecord } }).data
      const newTenant = responseData?.tenant || responseData

      if (newTenant) {
        this.tenants.unshift(newTenant)
      }

      return newTenant
    },

    async updateTenant(id: TenantId, tenantData: Record<string, unknown>): Promise<TenantRecord> {
      const index = this.tenants.findIndex((tenant) => tenant.id === id)
      const targetTenant = index !== -1 ? this.tenants[index] : null
      if (!targetTenant) {
        throw new Error('租户不存在')
      }

      const { tenantAPI } = await import('@/api')
      const tenantCode = typeof targetTenant.code === 'string' ? targetTenant.code : String(id)
      const response = await tenantAPI.updateTenant(tenantCode, tenantData)
      const responseData = (response as { data?: TenantRecord & { tenant?: TenantRecord } }).data
      const updatedTenant = responseData?.tenant ||
        responseData || { ...targetTenant, ...tenantData }

      this.tenants[index] = updatedTenant
      return updatedTenant
    },

    async deleteTenant(id: TenantId): Promise<void> {
      const { tenantAPI } = await import('@/api')
      await tenantAPI.deleteTenant(id)
      this.tenants = this.tenants.filter((tenant) => tenant.id !== id)
    },

    setCurrentTenant(tenant: TenantRecord | null): void {
      this.currentTenant = tenant
      if (tenant?.id !== null && tenant?.id !== undefined) {
        Storage.setTenantId(String(tenant.id))
      }
      applyTenantBrowserBrand(tenant)
    },
  },
})
