<template>
  <el-config-provider :locale="elementLocale">
    <div class="app-shell" :class="{ 'locale-en': locale === 'en-US' }">
      <aside class="sidebar">
        <div class="sidebar-top">
          <div class="brand">
            <div class="brand-mark">NA</div>
            <div>
              <div class="brand-name">NodeAgent</div>
              <div class="brand-sub">{{ t('app.subtitle') }}</div>
            </div>
          </div>

          <nav class="nav-list">
            <button
              v-for="item in navItems"
              :key="item.path"
              class="nav-item"
              :class="{ active: isActive(item.path) }"
              @click="go(item.path)"
            >
              <span class="nav-icon">
                <el-icon><component :is="item.icon" /></el-icon>
              </span>
              <span class="nav-label">{{ item.label }}</span>
            </button>
          </nav>
        </div>

        <div class="sidebar-bottom">
          <div class="sidebar-line"></div>
          <div class="quick-actions">
            <el-dropdown
              trigger="click"
              placement="top-start"
              popper-class="quick-locale-dropdown"
              @command="onLocaleSelect"
            >
              <button class="quick-action-btn quick-action-lang" :title="localeTip">
                <svg class="translate-icon" viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    d="M12.87 15.07l-2.54-2.51.03-.03c1.74-1.94 2.97-4.17 3.69-6.53H17V4h-7V2H8v2H1v2h11.17c-.66 1.91-1.73 3.7-3.17 5.26-.93-1.03-1.71-2.16-2.31-3.35H4.67c.73 1.63 1.73 3.17 2.98 4.55l-5.09 5.02L4 19l5-4.93 3.11 3.1.76-2.1zM18.5 10h-2L12 22h2l1.12-3h4.75L21 22h2l-4.5-12zm-2.62 7l1.62-4.33L19.12 17h-3.24z"
                    fill="currentColor"
                  />
                </svg>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="zh-CN">中文</el-dropdown-item>
                  <el-dropdown-item command="en-US">English</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>

            <span class="quick-divider"></span>

            <button class="quick-action-btn" :title="themeTip" @click="toggleTheme">
              <el-icon v-if="theme === 'dark'"><Moon /></el-icon>
              <el-icon v-else><Sunny /></el-icon>
            </button>
          </div>
        </div>
      </aside>

      <main class="main">
        <section class="content">
          <router-view />
        </section>
      </main>
    </div>
  </el-config-provider>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Connection, Grid, Moon, Sunny } from '@element-plus/icons-vue'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'
import { useI18nText } from '@/composables/useI18nText'
import { usePreferencesStore } from '@/store/preferencesStore'

const route = useRoute()
const router = useRouter()
const { locale, t } = useI18nText()
const preferences = usePreferencesStore()

const navItems = computed(() => [
  { path: '/projects/local', label: t('app.navLocal'), icon: Grid },
  { path: '/projects/online', label: t('app.navRemote'), icon: Connection },
])

const elementLocale = computed(() => (locale.value === 'en-US' ? en : zhCn))
const theme = computed(() => preferences.theme)
const themeTip = computed(() => {
  if (locale.value === 'en-US') {
    return theme.value === 'dark' ? 'Switch to Light Theme' : 'Switch to Dark Theme'
  }
  return theme.value === 'dark' ? '切换到浅色主题' : '切换到深色主题'
})
const localeTip = computed(() => {
  if (locale.value === 'en-US') {
    return 'Switch to Chinese'
  }
  return '切换到英文'
})

/**
 * 切换应用主题。
 * @returns {void}
 */
const toggleTheme = () => {
  preferences.setTheme(theme.value === 'dark' ? 'light' : 'dark')
}

/**
 * 切换应用语言。
 * @returns {void}
 */
const onLocaleSelect = (nextLocale) => {
  if (nextLocale === 'zh-CN' || nextLocale === 'en-US') {
    preferences.setLocale(nextLocale)
  }
}

/**
 * 导航到目标路径。
 * @param {string} path 目标路由路径
 * @returns {void}
 */
const go = (path) => {
  if (route.path !== path) {
    router.push(path)
  }
}

/**
 * 判断当前路由是否命中菜单项。
 * @param {string} path 菜单对应路由路径
 * @returns {boolean}
 */
const isActive = (path) => {
  return route.path === path
}
</script>

<style scoped>
.app-shell {
  display: grid;
  grid-template-columns: 198px 1fr;
  height: 100vh;
  overflow: hidden;
  background: var(--bg-canvas);
}

.sidebar {
  background: var(--sidebar-bg);
  border-right: 1px solid var(--sidebar-border);
  padding: 10px 8px 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.sidebar-top {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 72px;
  margin-bottom: 12px;
  padding: 6px 10px 4px;
}

.brand-mark {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  background: linear-gradient(150deg, #4e86ff 0%, #2361d5 100%);
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  display: grid;
  place-items: center;
  box-shadow: 0 10px 18px rgba(26, 88, 190, 0.32);
}

.brand-name {
  font-size: 19px;
  font-weight: 700;
  color: var(--sidebar-brand-name);
  letter-spacing: 0.3px;
  line-height: 1.1;
}

.brand-sub {
  font-size: 11px;
  color: var(--sidebar-brand-sub);
  margin-top: 2px;
}

.nav-list {
  display: grid;
  gap: 7px;
  padding: 4px 2px 0;
  flex: 1;
  align-content: start;
}

.nav-item {
  border: 1px solid transparent;
  background: transparent;
  border-radius: 10px;
  height: 46px;
  padding: 0 12px;
  color: var(--sidebar-nav-text);
  display: flex;
  align-items: center;
  gap: 18px;
  cursor: pointer;
  text-align: left;
  transition: all 0.2s ease;
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 6px;
}

.app-shell.locale-en .nav-item {
  font-size: 14px;
  letter-spacing: 0.6px;
}

.nav-icon {
  width: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--sidebar-nav-icon);
  font-size: 16px;
}

.nav-label {
  line-height: 1;
}

.nav-item:hover {
  background: var(--sidebar-nav-hover-bg);
  border-color: var(--sidebar-nav-hover-border);
  transform: translateX(2px);
}

.nav-item.active {
  color: var(--sidebar-nav-active-text);
  border-color: var(--sidebar-nav-active-border);
  background: var(--sidebar-nav-active-bg);
  box-shadow: var(--sidebar-nav-active-shadow);
}

.nav-item.active .nav-icon {
  color: var(--sidebar-nav-active-text);
}

.sidebar-bottom {
  flex-shrink: 0;
  margin-top: 8px;
  padding: 0 10px 4px;
}

.sidebar-line {
  height: 1px;
  width: 100%;
  background: var(--sidebar-line);
  opacity: 0.55;
}

.quick-actions {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.quick-action-btn {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  border: 1px solid transparent;
  background: rgba(127, 142, 178, 0.12);
  color: var(--sidebar-nav-icon);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
}

.quick-action-btn:hover {
  background: var(--sidebar-nav-hover-bg);
  color: var(--sidebar-brand-name);
  border-color: var(--sidebar-nav-active-border);
}

.quick-action-lang {
  width: 52px;
  min-width: 52px;
  padding: 0;
}

.translate-icon {
  width: 22px;
  height: 22px;
  display: block;
}

.quick-divider {
  width: 1px;
  height: 22px;
  background: var(--sidebar-line);
  opacity: 0.7;
}

:deep(.quick-locale-dropdown) {
  padding: 0 !important;
  border-radius: 12px !important;
  border: 1px solid var(--border-subtle) !important;
  overflow: hidden;
}

:deep(.quick-locale-dropdown .el-dropdown-menu) {
  padding: 0 !important;
}

:deep(.quick-locale-dropdown .el-dropdown-menu__item) {
  min-width: 180px;
  height: 48px;
  font-size: 16px;
}

:deep(.quick-locale-dropdown .el-dropdown-menu__item:not(.is-disabled):focus) {
  background-color: rgba(43, 117, 222, 0.12) !important;
  color: #245ec8 !important;
}

.main {
  padding: var(--space-5);
  overflow-y: auto;
}

.content {
  min-height: 100%;
}

@media (max-width: 1000px) {
  .app-shell {
    grid-template-columns: 1fr;
  }

  .sidebar {
    display: none;
  }
}
</style>
