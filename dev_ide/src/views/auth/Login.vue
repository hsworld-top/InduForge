<template>
  <!-- 背景图片容器 -->
  <div
    class="min-h-screen bg-cover bg-center bg-no-repeat relative"
    :style="{ backgroundImage: `url(${backgroundImageUrl})` }"
  >
    <!-- 背景遮罩层：使用轻量蓝色蒙层，避免默认背景被压灰 -->
    <div class="absolute inset-0 bg-blue-900 bg-opacity-20"></div>

    <!-- 右上角控制按钮 -->
    <div class="absolute top-4 right-4 flex items-center space-x-2 z-10">
      <!-- 语言选择下拉框 -->
      <el-dropdown @command="selectLanguage" trigger="click">
        <span
          class="flex items-center space-x-1 p-2 rounded-lg bg-white bg-opacity-20 hover:bg-opacity-30 text-white transition-colors cursor-pointer"
          :title="$t('system.language')"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129"
            />
          </svg>
          <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 9l-7 7-7-7"
            />
          </svg>
        </span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item
              v-for="lang in languages"
              :key="lang.code"
              :command="lang.code"
              :class="{ 'is-active': locale === lang.code }"
            >
              {{ lang.name }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>

      <!-- 主题切换按钮 -->
      <button
        @click="toggleTheme"
        class="p-2 rounded-lg bg-white bg-opacity-20 hover:bg-opacity-30 text-white transition-colors"
        :title="isDark ? $t('system.light') : $t('system.dark')"
      >
        <svg v-if="isDark" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
          />
        </svg>
        <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
          />
        </svg>
      </button>
    </div>

    <!-- 主要内容区域 -->
    <div class="relative min-h-screen flex">
      <!-- 左侧区域 - Logo 和品牌信息 -->
      <div class="hidden lg:flex lg:w-1/2 xl:w-2/5 items-center justify-center p-8">
        <div class="text-center">
          <!-- Logo 图片位置 -->
          <div class="mb-8">
            <img
              v-if="logoUrl"
              :src="logoUrl"
              :alt="appConfig?.name || 'Logo'"
              class="mx-auto h-20 w-auto"
            />
            <div
              v-else
              class="mx-auto h-20 w-20 bg-white bg-opacity-20 rounded-xl flex items-center justify-center"
            >
              <svg
                class="h-12 w-12 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                />
              </svg>
            </div>
          </div>

          <!-- 品牌标题 -->
          <h1 class="text-4xl font-bold text-white mb-4">
            {{ appConfig?.name || 'InduFrame' }}
          </h1>
          <p class="text-xl text-white text-opacity-90">
            {{ appConfig?.description || $t('auth.defaultDescription') }}
          </p>
        </div>
      </div>

      <!-- 右侧区域 - 登录表单 -->
      <div
        class="w-full lg:w-1/2 xl:w-3/5 flex items-center justify-end pr-12 lg:pr-16 xl:pr-20 p-8"
      >
        <div
          class="w-full max-w-md space-y-6 bg-white dark:bg-gray-800 rounded-xl shadow-2xl p-8 login-form"
        >
          <!-- Logo 和标题 (移动端显示) -->
          <div class="text-center lg:hidden">
            <img
              v-if="logoUrl"
              :src="logoUrl"
              :alt="appConfig?.name || 'Logo'"
              class="mx-auto h-12 w-auto mb-4"
            />
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ $t('auth.welcome') }}
            </h2>
          </div>

          <!-- 桌面端标题 -->
          <div class="hidden lg:block text-center">
            <h2 class="text-3xl font-bold text-gray-900 dark:text-white">
              {{ $t('auth.welcome') }}
            </h2>
          </div>

          <!-- 登录表单 -->
          <form class="space-y-6" @submit.prevent="handleLogin">
            <!-- 租户代码（多租户模式下显示） -->
            <div v-if="appConfig?.multiTenant" class="space-y-2">
              <label
                for="tenantCode"
                class="block text-sm font-semibold text-gray-700 dark:text-gray-300"
              >
                {{ $t('auth.tenantCode') }}
              </label>
              <div class="relative">
                <input
                  id="tenantCode"
                  v-model="form.tenantCode"
                  type="text"
                  class="input-field"
                  :placeholder="$t('auth.tenantCodePlaceholder')"
                />
                <svg
                  class="absolute right-3 top-3.5 h-5 w-5 input-icon"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
                  />
                </svg>
              </div>
            </div>

            <!-- 用户名 -->
            <div class="space-y-2">
              <label
                for="username"
                class="block text-sm font-semibold text-gray-700 dark:text-gray-300"
              >
                {{ $t('auth.username') }}
              </label>
              <div class="relative">
                <input
                  id="username"
                  v-model="form.username"
                  type="text"
                  autocomplete="username"
                  required
                  class="input-field"
                  :placeholder="$t('auth.username')"
                />
                <svg
                  class="absolute right-3 top-3.5 h-5 w-5 input-icon"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
                  />
                </svg>
              </div>
            </div>

            <!-- 密码 -->
            <div class="space-y-2">
              <label
                for="password"
                class="block text-sm font-semibold text-gray-700 dark:text-gray-300"
              >
                {{ $t('auth.password') }}
              </label>
              <div class="relative">
                <input
                  id="password"
                  v-model="form.password"
                  :type="showPassword ? 'text' : 'password'"
                  autocomplete="current-password"
                  required
                  class="input-field"
                  :placeholder="$t('auth.password')"
                  style="padding-right: 3rem"
                />
                <button
                  type="button"
                  @click="togglePasswordVisibility"
                  class="absolute right-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 transition-colors cursor-pointer password-toggle-btn"
                  :title="showPassword ? $t('auth.hidePassword') : $t('auth.showPassword')"
                >
                  <svg v-if="showPassword" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L3 3m6.878 6.878L21 21"
                    />
                  </svg>
                  <svg v-else fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                    />
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                    />
                  </svg>
                </button>
              </div>
            </div>

            <!-- 滑块验证 -->
            <div v-if="showCaptcha" class="space-y-2">
              <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300">
                {{ $t('auth.captcha') }}
              </label>
              <div class="flex items-center gap-2">
                <div
                  ref="sliderTrack"
                  class="slider-captcha-track"
                  :class="{ 'is-dragging': sliderDragging, 'is-ready': sliderCompleted }"
                  :style="sliderTrackStyle"
                  @pointermove="moveSlider"
                  @pointerup="endSlider"
                  @pointercancel="endSlider"
                >
                  <div class="slider-captcha-progress" :style="sliderProgressStyle"></div>
                  <button
                    type="button"
                    class="slider-captcha-thumb"
                    :style="sliderThumbStyle"
                    role="slider"
                    :aria-label="$t('auth.captcha')"
                    aria-valuemin="0"
                    :aria-valuemax="sliderMaxOffset"
                    :aria-valuenow="sliderOffset"
                    @pointerdown="startSlider"
                    @keydown="moveSliderByKeyboard"
                  >
                    <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m9 5 7 7-7 7" />
                    </svg>
                  </button>
                </div>
                <button
                  type="button"
                  class="slider-captcha-refresh"
                  :title="$t('auth.refreshCaptcha')"
                  @click="refreshCaptcha"
                >
                  <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 0 0 4.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 0 1-15.357-2M15 15h-4" />
                  </svg>
                </button>
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ sliderCompleted ? $t('auth.sliderCaptchaReady') : $t('auth.sliderCaptchaHint') }}
              </p>
            </div>

            <!-- 记住我选项 -->
            <div class="flex items-center justify-between">
              <div class="flex items-center">
                <input
                  id="remember-me"
                  v-model="form.rememberMe"
                  type="checkbox"
                  class="checkbox-custom"
                />
                <label
                  for="remember-me"
                  class="ml-2 text-sm text-gray-600 dark:text-gray-400 cursor-pointer"
                >
                  {{ $t('auth.rememberMe') }}
                </label>
              </div>
            </div>

            <!-- 登录按钮 -->
            <button
              type="submit"
              :disabled="loading || (showCaptcha && !sliderCompleted)"
              class="login-button group relative w-full flex justify-center py-4 px-4 border border-transparent text-sm font-semibold rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg"
            >
              <span v-if="loading" class="flex items-center">
                <svg
                  class="animate-spin -ml-1 mr-2 h-4 w-4 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    class="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    stroke-width="4"
                  ></circle>
                  <path
                    class="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  ></path>
                </svg>
                {{ $t('auth.loading') }}
              </span>
              <span v-else class="flex items-center">
                <svg
                  class="-ml-1 mr-2 h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1"
                  />
                </svg>
                {{ $t('auth.login') }}
              </span>
            </button>
          </form>
        </div>
      </div>
    </div>

    <div
      v-if="loginDisplayItems.length"
      class="absolute bottom-4 left-1/2 -translate-x-1/2 z-10 px-4 py-2 rounded-lg bg-black/25 text-white text-xs md:text-sm max-w-[90vw]"
    >
      <div class="flex flex-wrap items-center justify-center gap-x-4 gap-y-1">
        <template v-for="item in loginDisplayItems" :key="item.key">
          <a
            v-if="item.isLink"
            :href="item.href || item.value"
            target="_blank"
            rel="noopener noreferrer"
            class="hover:underline"
          >
            {{ item.label }}{{ item.value }}
          </a>
          <span v-else>{{ item.label }}{{ item.value }}</span>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
// @ts-nocheck
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, useAppStore, useTenantStore } from '@/store'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { ROLES, ROUTE_NAMES } from '@/constants'
import { Storage } from '@/utils/storage'
import { getApiErrorCode, getApiErrorMessage } from '@/utils/request'
import defaultLogoUrl from '@/assets/images/default-logo.svg'
import defaultLoginBgUrl from '@/assets/images/default-login-bg.svg'

const AUTH_INVALID_CAPTCHA_CODE = 10010

export default {
  name: 'Login',
  setup() {
    const router = useRouter()
    const authStore = useAuthStore()
    const appStore = useAppStore()
    const tenantStore = useTenantStore()
    const { locale, t } = useI18n()

    const form = reactive({
      tenantCode: '',
      username: '',
      password: '',
      rememberMe: false,
    })

    const loading = ref(false)
    const showCaptcha = ref(false)
    const captchaData = ref(null) // { challengeId, trackWidth, thumbWidth, expireSeconds }
    const sliderTrack = ref(null)
    const sliderOffset = ref(0)
    const sliderCompleted = ref(false)
    const sliderDragging = ref(false)
    const showPassword = ref(false) // 密码可见性
    let tenantConfigTimer = null

    // 背景图和Logo（可以从应用配置中获取）
    const backgroundImageUrl = computed(() => {
      // 优先使用租户配置的背景图，然后是平台默认背景图
      return appStore.config?.loginBackgroundUrl || defaultLoginBgUrl
    })

    const logoUrl = computed(() => {
      // 优先使用租户配置的Logo，然后是平台默认Logo
      return appStore.config?.logoUrl || defaultLogoUrl
    })

    const appConfig = computed(() => appStore.config)
    const isDark = computed(() => appStore.isDark)
    const loginDisplay = computed(() => appConfig.value?.loginDisplay || {})
    const loginDisplayItems = computed(() => {
      const list = []
      if (loginDisplay.value.showCompanyName && loginDisplay.value.companyName) {
        list.push({
          key: 'companyName',
          label: `${t('auth.companyLabel')}: `,
          value: loginDisplay.value.companyName,
        })
      }
      if (loginDisplay.value.showCompanyPhone && loginDisplay.value.companyPhone) {
        list.push({
          key: 'companyPhone',
          label: `${t('auth.phoneLabel')}: `,
          value: loginDisplay.value.companyPhone,
        })
      }
      if (loginDisplay.value.showCompanyAddress && loginDisplay.value.companyAddress) {
        list.push({
          key: 'companyAddress',
          label: `${t('auth.addressLabel')}: `,
          value: loginDisplay.value.companyAddress,
        })
      }
      if (loginDisplay.value.showCompanyWebsite && loginDisplay.value.companyWebsite) {
        const website = String(loginDisplay.value.companyWebsite)
        const websiteHref = /^https?:\/\//i.test(website) ? website : `https://${website}`
        list.push({
          key: 'companyWebsite',
          label: `${t('auth.websiteLabel')}: `,
          value: website,
          href: websiteHref,
          isLink: true,
        })
      }
      if (loginDisplay.value.showIcp && loginDisplay.value.icpNumber) {
        list.push({
          key: 'icp',
          label: `${t('auth.icpLabel')}: `,
          value: loginDisplay.value.icpNumber,
        })
      }
      return list
    })

    // 语言相关
    const languages = computed(() => [
      { code: 'zh', name: t('system.languageZh') },
      { code: 'en', name: t('system.languageEn') },
    ])

    const toggleTheme = () => {
      appStore.setTheme(isDark.value ? 'light' : 'dark')
    }

    const selectLanguage = (langCode) => {
      locale.value = langCode
      appStore.setLanguage(langCode)
    }

    const togglePasswordVisibility = () => {
      showPassword.value = !showPassword.value
    }

    const sliderTrackWidth = computed(() => captchaData.value?.trackWidth || 280)
    const sliderThumbWidth = computed(() => captchaData.value?.thumbWidth || 44)
    const sliderMaxOffset = computed(() => Math.max(0, sliderTrackWidth.value - sliderThumbWidth.value))
    const sliderTrackStyle = computed(() => ({
      '--slider-thumb-width': `${(sliderThumbWidth.value / sliderTrackWidth.value) * 100}%`,
    }))
    const sliderThumbStyle = computed(() => ({
      left: `${(sliderOffset.value / Math.max(1, sliderTrackWidth.value)) * 100}%`,
    }))
    const sliderProgressStyle = computed(() => ({
      width: `${((sliderOffset.value + sliderThumbWidth.value / 2) / sliderTrackWidth.value) * 100}%`,
    }))

    const resetSlider = () => {
      sliderOffset.value = 0
      sliderCompleted.value = false
      sliderDragging.value = false
    }

    const setSliderOffset = (clientX) => {
      const track = sliderTrack.value
      if (!track) {
        return
      }
      const rect = track.getBoundingClientRect()
      if (!rect.width) {
        return
      }
      const position = ((clientX - rect.left) / rect.width) * sliderTrackWidth.value
      sliderOffset.value = Math.round(
        Math.min(sliderMaxOffset.value, Math.max(0, position - sliderThumbWidth.value / 2)),
      )
      sliderCompleted.value = sliderOffset.value >= sliderMaxOffset.value
    }

    const startSlider = (event) => {
      if (!captchaData.value?.challengeId) {
        return
      }
      sliderDragging.value = true
      event.currentTarget.setPointerCapture?.(event.pointerId)
    }

    const moveSlider = (event) => {
      if (sliderDragging.value) {
        setSliderOffset(event.clientX)
      }
    }

    const endSlider = () => {
      sliderDragging.value = false
    }

    const moveSliderByKeyboard = (event) => {
      const moves = { ArrowLeft: -8, ArrowRight: 8, Home: -sliderMaxOffset.value, End: sliderMaxOffset.value }
      if (!(event.key in moves)) {
        return
      }
      event.preventDefault()
      sliderOffset.value = Math.min(
        sliderMaxOffset.value,
        Math.max(0, sliderOffset.value + moves[event.key]),
      )
      sliderCompleted.value = sliderOffset.value >= sliderMaxOffset.value
    }

    const refreshCaptcha = async () => {
      resetSlider()
      try {
        const { authAPI } = await import('@/api')
        const result = await authAPI.getCaptcha(form.username, form.tenantCode || undefined)
        captchaData.value = result.data // 提取响应中的data部分
      } catch (error) {
        console.error('获取验证码失败:', error)
        ElMessage.error(t('auth.getCaptchaFailed'))
      }
    }

    const handleLogin = async () => {
      if (!form.username || !form.password) {
        ElMessage.warning(t('auth.pleaseInputUsernamePassword'))
        return
      }

      if (showCaptcha.value && (!sliderCompleted.value || !captchaData.value?.challengeId)) {
        ElMessage.warning(t('auth.pleaseInputCaptcha'))
        return
      }

      loading.value = true

      try {
        const { authAPI } = await import('@/api')

        // 准备登录参数
        const loginParams = {
          username: form.username,
          password: form.password,
          sliderChallengeId: showCaptcha.value ? captchaData.value?.challengeId : undefined,
          sliderOffset: showCaptcha.value ? sliderOffset.value : undefined,
          tenantCode: form.tenantCode || undefined, // 多租户时可选
        }

        const result = await authAPI.login(loginParams)

        // 登录成功后由服务端写入 HttpOnly Cookie，前端只保存用户资料。
        const user = result.data?.user

        if (!user) {
          throw new Error('登录返回数据格式不正确')
        }

        // 更新store状态
        authStore.setAuthData({
          id: user.id,
          username: user.username,
          role: user.role,
          tenantId: user.tenant?.id,
          tenant: user.tenant,
        })
        if (user.tenant) {
          tenantStore.setCurrentTenant(user.tenant)
        }

        // 保存记住的凭据（如果勾选了记住我）
        Storage.setRememberMeCredentials({
          username: form.username,
          password: form.rememberMe ? form.password : '',
          tenantCode: form.tenantCode,
          rememberMe: form.rememberMe,
        })

        // 登录成功后清除验证码状态
        showCaptcha.value = false
        captchaData.value = null
        resetSlider()

        ElMessage.success(t('auth.loginSuccess'))

        // 根据用户角色跳转到合适的页面
        if (user.role === ROLES.SUPER_ADMIN) {
          router.push('/admin')
        } else {
          router.push({ name: ROUTE_NAMES.DASHBOARD })
        }
      } catch (error) {
        console.error('登录失败:', error)
        const errorStatus = error?.response?.status
        const errorCode = getApiErrorCode(error)
        const isCaptchaError = errorCode === AUTH_INVALID_CAPTCHA_CODE
        const isCredentialError = errorCode === 10007

        if (errorStatus === 429) {
          ElMessage.error(t('auth.tooManyRequests'))
        } else {
          const fallbackMessage =
            errorStatus === 401 || isCredentialError || isCaptchaError
              ? t('auth.invalidCredentials')
              : t('auth.retry')
          ElMessage.error(getApiErrorMessage(error, fallbackMessage))

          // 首次凭据错误后才显示滑块；滑块验证失败后必须获取一次新挑战。
          if (isCredentialError || isCaptchaError) {
            if (!showCaptcha.value) {
              showCaptcha.value = true
              await refreshCaptcha()
            } else if (isCaptchaError || errorCode === 10007) {
              await refreshCaptcha()
            }
          }
        }
      } finally {
        loading.value = false
      }
    }

    onMounted(async () => {
      // 检查是否已登录
      if (authStore.isAuthenticated) {
        router.push({ name: ROUTE_NAMES.DASHBOARD })
        return
      }

      // 加载记住的凭据
      const rememberedCredentials = Storage.getRememberMeCredentials()
      if (rememberedCredentials) {
        form.username = rememberedCredentials.username || ''
        form.password = rememberedCredentials.password || ''
        form.tenantCode = rememberedCredentials.tenantCode || ''
        form.rememberMe = rememberedCredentials.rememberMe || false
      }

      // 异步加载应用配置，不阻塞页面渲染
      appStore.loadConfig(form.tenantCode || undefined).catch((error) => {
        console.error('Failed to load config:', error)
        // 错误已在 loadConfig 中处理，这里只记录日志
      })
    })

    watch(
      () => form.tenantCode,
      (value) => {
        window.clearTimeout(tenantConfigTimer)
        tenantConfigTimer = window.setTimeout(() => {
          appStore.loadConfig(value || undefined).catch((error) => {
            console.error('Failed to load tenant config:', error)
          })
        }, 300)
      },
    )

    watch(
      () => [form.username, form.tenantCode],
      () => {
        // 挑战与用户名、租户编码及客户端 IP 绑定，修改登录上下文后必须重新触发验证。
        if (showCaptcha.value) {
          showCaptcha.value = false
          captchaData.value = null
          resetSlider()
        }
      },
    )

    return {
      form,
      loading,
      showCaptcha,
      captchaData,
      sliderTrack,
      sliderOffset,
      sliderCompleted,
      sliderDragging,
      sliderMaxOffset,
      sliderTrackStyle,
      sliderThumbStyle,
      sliderProgressStyle,
      showPassword,
      locale,
      backgroundImageUrl,
      logoUrl,
      appConfig,
      loginDisplayItems,
      isDark,
      languages,
      handleLogin,
      refreshCaptcha,
      startSlider,
      moveSlider,
      endSlider,
      moveSliderByKeyboard,
      toggleTheme,
      selectLanguage,
      togglePasswordVisibility,
    }
  },
}
</script>

<style scoped>
/* 自定义输入框样式 */
.input-field {
  @apply w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200;
}

/* 登录按钮悬停效果 */
.login-button {
  background: linear-gradient(135deg, #14b8a6 0%, #0f766e 100%);
  transition: all 0.3s ease;
}

.login-button:hover {
  background: linear-gradient(135deg, #0d9488 0%, #115e59 100%);
  transform: translateY(-1px);
  box-shadow: 0 10px 25px rgba(13, 148, 136, 0.3);
}

/* 表单容器动画 */
.login-form {
  animation: slideIn 0.5s ease-out;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 输入框图标样式 */
.input-icon {
  pointer-events: none;
  color: #9ca3af;
  transition: color 0.2s ease;
}

input:focus + .input-icon {
  color: #14b8a6;
}

/* 密码切换按钮样式 */
.password-toggle-btn {
  pointer-events: auto !important;
}

.slider-captcha-track {
  --slider-thumb-width: 16%;
  position: relative;
  flex: 1;
  height: 48px;
  overflow: hidden;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: #eef2f5;
  touch-action: none;
}

.slider-captcha-progress {
  position: absolute;
  inset: 0 auto 0 0;
  background: rgba(20, 184, 166, 0.18);
  border-right: 1px solid rgba(13, 148, 136, 0.5);
  pointer-events: none;
}

.slider-captcha-thumb {
  position: absolute;
  top: 2px;
  bottom: 2px;
  width: var(--slider-thumb-width);
  min-width: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #0f766e;
  border-radius: 5px;
  color: #0f766e;
  background: #ffffff;
  cursor: grab;
  touch-action: none;
  transform: translateX(0);
}

.slider-captcha-thumb:active,
.slider-captcha-track.is-dragging .slider-captcha-thumb {
  cursor: grabbing;
}

.slider-captcha-thumb svg {
  width: 18px;
  height: 18px;
}

.slider-captcha-track.is-ready .slider-captcha-thumb {
  color: #047857;
  border-color: #047857;
}

.slider-captcha-refresh {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  color: #4b5563;
  background: transparent;
}

.slider-captcha-refresh:hover {
  color: #0f766e;
  border-color: #0f766e;
}

.slider-captcha-refresh svg {
  width: 18px;
  height: 18px;
}

/* 记住我复选框样式 */
.checkbox-custom {
  @apply rounded border-gray-300 text-blue-600 focus:ring-blue-500 focus:ring-2;
}

/* Element Plus Dropdown 自定义样式 */
:deep(.el-dropdown) {
  --el-dropdown-menu-box-shadow:
    0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
}

:deep(.el-dropdown-menu) {
  background-color: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 8px;
  padding: 4px 0;
  min-width: 120px;
}

:deep(.dark .el-dropdown-menu) {
  background-color: rgba(31, 41, 55, 0.95);
  border: 1px solid rgba(75, 85, 99, 0.3);
}

:deep(.el-dropdown-menu__item) {
  color: rgb(55, 65, 81);
  padding: 8px 16px;
  font-size: 14px;
  transition: all 0.2s ease;
}

:deep(.dark .el-dropdown-menu__item) {
  color: rgb(209, 213, 219);
}

:deep(.el-dropdown-menu__item:hover) {
  background-color: rgba(20, 184, 166, 0.1);
  color: rgb(20, 184, 166);
}

:deep(.el-dropdown-menu__item.is-active) {
  background-color: rgba(20, 184, 166, 0.1);
  color: rgb(20, 184, 166);
  font-weight: 600;
}

:deep(.dark .el-dropdown-menu__item:hover) {
  background-color: rgba(20, 184, 166, 0.2);
}

:deep(.dark .el-dropdown-menu__item.is-active) {
  background-color: rgba(20, 184, 166, 0.2);
}
</style>
