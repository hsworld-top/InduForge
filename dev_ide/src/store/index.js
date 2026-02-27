import { defineStore } from 'pinia'
import { Storage } from '@/utils'
import defaultLogoUrl from '@/assets/images/default-logo.svg'
import defaultLoginBgUrl from '@/assets/images/default-login-bg.svg'

// 认证状态管理
export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: Storage.getToken(),
    refreshToken: Storage.getRefreshToken(),
    userInfo: Storage.getUserInfo(),
    isAuthenticated: !!Storage.getToken(),
  }),

  getters: {
    getUserRole: (state) => state.userInfo?.role,
    getUserId: (state) => state.userInfo?.id,
    getUsername: (state) => state.userInfo?.username,
    isAdmin: (state) =>
      ['SUPER_ADMIN', 'SYSTEM_ADMIN', 'PROJECT_ADMIN', 'OPS_ADMIN', 'USER_ADMIN'].includes(
        state.userInfo?.role
      ),
  },

  actions: {
    async login(credentials) {
      try {
        const { authAPI } = await import('@/api')
        const response = await authAPI.login(credentials)
        const user = response.data?.user
        const token = response.data?.accessToken || response.data?.token
        const refreshToken = response.data?.refreshToken || null

        if (!token || !user) {
          throw new Error('登录响应缺少必要字段')
        }

        this.setAuthData(token, refreshToken, {
          id: user.id,
          username: user.username,
          email: user.email,
          role: user.role,
          tenantId: user.tenant?.id || user.tenantId || null,
        })
        return { success: true }
      } catch (error) {
        return { success: false, error: error.response?.data?.message || error.message }
      }
    },

    logout() {
      this.clearAuthData()
    },

    refreshToken() {
      // 实现 token 刷新逻辑
    },

    setAuthData(token, refreshToken, userInfo) {
      this.token = token
      this.refreshToken = refreshToken
      this.userInfo = userInfo
      this.isAuthenticated = true

      Storage.setToken(token)
      if (refreshToken) Storage.setRefreshToken(refreshToken)
      Storage.setUserInfo(userInfo)
      if (userInfo.tenantId) Storage.setTenantId(userInfo.tenantId)
    },

    clearAuthData() {
      this.token = null
      this.refreshToken = null
      this.userInfo = null
      this.isAuthenticated = false

      Storage.remove('auth_token')
      Storage.remove('refresh_token')
      Storage.remove('user_info')
      Storage.remove('tenant_id')
    },
  },
})

// 应用配置状态管理
export const useAppStore = defineStore('app', {
  state: () => ({
    theme: Storage.getTheme(),
    language: Storage.getLanguage(),
    loading: false,
    sidebarCollapsed: Storage.getSidebarCollapsed(),
    config: null,
  }),

  getters: {
    isDark: (state) => state.theme === 'dark',
    isLight: (state) => state.theme === 'light',
  },

  actions: {
    setTheme(theme) {
      this.theme = theme
      Storage.setTheme(theme)

      // 应用主题到 body
      document.documentElement.classList.toggle('dark', theme === 'dark')
    },

    setLanguage(language) {
      this.language = language
      Storage.setLanguage(language)
    },

    setLoading(loading) {
      this.loading = loading
    },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed
      Storage.setSidebarCollapsed(this.sidebarCollapsed)
    },

    setSidebarCollapsed(collapsed) {
      this.sidebarCollapsed = !!collapsed
      Storage.setSidebarCollapsed(this.sidebarCollapsed)
    },

    setConfig(config) {
      this.config = config
    },

    async loadConfig() {
      try {
        const { authAPI } = await import('@/api')
        const { data: config } = await authAPI.getConfig()
        // 合并后端配置和前端默认配置
        const mergedConfig = {
          ...config,
          // 添加前端特定的配置
          logoUrl: config.logoUrl || defaultLogoUrl,
          loginBackgroundUrl: config.loginBackgroundUrl || defaultLoginBgUrl,
          // 使用后端返回的multiTenant配置，如果后端没有返回则默认为false（单租户模式）
          multiTenant: config.multiTenant || false,
          allowRegistration: false,
          defaultLanguage: 'zh',
          supportedLanguages: ['zh', 'en'],
          features: {
            captcha: true,
            emailVerification: false,
            twoFactorAuth: false,
          },
        }

        this.setConfig(mergedConfig)
      } catch (error) {
        console.error('Failed to load config:', error)
        // 如果API调用失败，使用默认配置（单租户模式）
        const defaultConfig = {
          name: 'InduForge',
          description: '高效、安全的企业级解决方案',
          logoUrl: defaultLogoUrl,
          loginBackgroundUrl: defaultLoginBgUrl,
          multiTenant: false, // 默认单租户模式
          allowRegistration: false,
          defaultLanguage: 'zh',
          supportedLanguages: ['zh', 'en'],
          features: {
            captcha: true,
            emailVerification: false,
            twoFactorAuth: false,
          },
        }
        this.setConfig(defaultConfig)
      }
    },
  },
})

// 租户状态管理
export const useTenantStore = defineStore('tenant', {
  state: () => ({
    currentTenant: null,
    tenants: [],
    loading: false,
  }),

  getters: {
    getTenantById: (state) => (id) => state.tenants.find((t) => t.id === id),
    activeTenants: (state) => state.tenants.filter((t) => t.status === 'active'),
  },

  actions: {
    async fetchTenants() {
      this.loading = true
      try {
        const { tenantAPI } = await import('@/api')
        const response = await tenantAPI.getTenants({ page: 1, limit: 200 })
        this.tenants = response.data?.tenants || []
      } catch (error) {
        console.error('Failed to fetch tenants:', error)
        this.tenants = []
      } finally {
        this.loading = false
      }
    },

    async createTenant(tenantData) {
      try {
        const { tenantAPI } = await import('@/api')
        const response = await tenantAPI.createTenant(tenantData)
        const newTenant = response.data?.tenant || response.data
        if (newTenant) {
          this.tenants.unshift(newTenant)
        }
        return newTenant
      } catch (error) {
        throw error
      }
    },

    async updateTenant(id, tenantData) {
      try {
        const index = this.tenants.findIndex((t) => t.id === id)
        const targetTenant = index !== -1 ? this.tenants[index] : null
        if (!targetTenant) {
          throw new Error('租户不存在')
        }
        const { tenantAPI } = await import('@/api')
        const response = await tenantAPI.updateTenant(targetTenant.code || id, tenantData)
        const updatedTenant = response.data?.tenant || response.data || {
          ...targetTenant,
          ...tenantData,
        }
        this.tenants[index] = updatedTenant
        return updatedTenant
      } catch (error) {
        throw error
      }
    },

    async deleteTenant(id) {
      try {
        const { tenantAPI } = await import('@/api')
        await tenantAPI.deleteTenant(id)
        this.tenants = this.tenants.filter((t) => t.id !== id)
      } catch (error) {
        throw error
      }
    },

    setCurrentTenant(tenant) {
      this.currentTenant = tenant
      if (tenant?.id) {
        Storage.setTenantId(tenant.id)
      }
    },
  },
})
