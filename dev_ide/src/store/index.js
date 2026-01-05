import { defineStore } from 'pinia'
import dayjs from 'dayjs'
import { Storage } from '@/utils'
import { TIME_FORMAT } from '@/constants'

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
    isAdmin: (state) => ['SYSTEM_ADMIN', 'TENANT_ADMIN'].includes(state.userInfo?.role),
  },

  actions: {
    async login(credentials) {
      try {
        // 这里会调用 API
        // const response = await authAPI.login(credentials)

        // 模拟登录成功
        const mockUser = {
          id: 1,
          username: credentials.username,
          email: 'admin@example.com',
          role: 'SYSTEM_ADMIN',
          tenantId: null,
        }

        const mockToken = 'mock-jwt-token'

        this.setAuthData(mockToken, null, mockUser)
        return { success: true }
      } catch (error) {
        return { success: false, error: error.message }
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
    sidebarCollapsed: false,
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
          logoUrl: '/images/logo.png',
          loginBackgroundUrl: '/images/login-bg.jpg',
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
          name: '多租户管理系统',
          description: '高效、安全的企业级解决方案',
          logoUrl: '/images/logo.png',
          loginBackgroundUrl: '/images/login-bg.jpg',
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
        // const tenants = await tenantAPI.getTenants()
        // this.tenants = tenants

        // 模拟租户数据
        this.tenants = [
          {
            id: 1,
            name: '默认租户',
            code: 'default',
            status: 'active',
            createdAt: '2024-01-01',
            userCount: 10,
            logoUrl: '/images/demo.png',
          },
          {
            id: 2,
            name: '示例公司',
            code: 'example',
            status: 'active',
            createdAt: '2024-02-01',
            userCount: 25,
            logoUrl: '/images/demo.png',
          },
        ]
      } catch (error) {
        console.error('Failed to fetch tenants:', error)
      } finally {
        this.loading = false
      }
    },

    async createTenant(tenantData) {
      try {
        // const newTenant = await tenantAPI.createTenant(tenantData)
        // this.tenants.push(newTenant)
        // return newTenant

        // 模拟创建租户
        const newTenant = {
          id: Date.now(),
          ...tenantData,
          status: 'active',
          createdAt: dayjs().format(TIME_FORMAT),
          userCount: 0,
        }
        this.tenants.push(newTenant)
        return newTenant
      } catch (error) {
        throw error
      }
    },

    async updateTenant(id, tenantData) {
      try {
        // const updatedTenant = await tenantAPI.updateTenant(id, tenantData)
        // const index = this.tenants.findIndex(t => t.id === id)
        // if (index !== -1) {
        //   this.tenants[index] = updatedTenant
        // }
        // return updatedTenant

        // 模拟更新租户
        const index = this.tenants.findIndex((t) => t.id === id)
        if (index !== -1) {
          this.tenants[index] = { ...this.tenants[index], ...tenantData }
          return this.tenants[index]
        }
      } catch (error) {
        throw error
      }
    },

    async deleteTenant(id) {
      try {
        // await tenantAPI.deleteTenant(id)
        // this.tenants = this.tenants.filter(t => t.id !== id)

        // 模拟删除租户
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
