<!--
  App.vue - 设计器根组件
  职责：路由视图、全局加载态（设计页切换时显示，预览页不显示）
-->
<script setup lang="ts">
/**
 * 根组件脚本
 * - loading: 全局加载态
 * - 路由切换时显示至少 minLoadingMs 的加载动画，避免闪烁
 * - 从预览页返回设计页时跳过加载（skipNextLoading）
 */
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import type { Language } from 'element-plus/es/locale'
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { resolveAppLocaleForPath, resolveScopedLocaleForPath } from '@/router/runtime-settings'
import { getEditorUiStore } from '@/stores/editor-ui-store'

/** 是否处于加载中 */
const loading = ref(true)
/** 本次加载开始时间戳 */
let loadingStartAt = Date.now()
/** 最小加载展示时长（ms），避免加载过快导致闪烁 */
const minLoadingMs = 500
const router = useRouter()
const route = useRoute()
const editorUi = getEditorUiStore()
const { locale } = useI18n()
/** 当前是否为预览路由 */
const isPreviewRoute = computed(
  () => route?.name === 'Preview' || String(route?.path || '').includes('/preview'),
)
/** 是否显示加载遮罩（设计页加载时显示，预览页不显示） */
const showLoading = computed(() => loading.value && !isPreviewRoute.value)
const elementLocale = computed<Language>(() =>
  resolveScopedLocaleForPath(
    route.path,
    editorUi.elementLocale.value as Language,
    zhCn as Language,
  ),
)

/** 是否跳过下一次加载（从预览返回设计时使用） */
const skipNextLoading = ref(false)

/** 开始加载，重置计时 */
function startLoading() {
  if (skipNextLoading.value) {
    skipNextLoading.value = false
    loading.value = false
    return
  }
  loadingStartAt = Date.now()
  loading.value = true
}

/** 结束加载，若未达到最小时长则等待补足 */
async function stopLoading() {
  const elapsed = Date.now() - loadingStartAt
  const waitMs = Math.max(0, minLoadingMs - elapsed)
  if (waitMs > 0) {
    await new Promise((resolve) => setTimeout(resolve, waitMs))
  }
  await nextTick()
  loading.value = false
}

/** 路由进入前：启动加载 */
const removeBefore = router.beforeEach((to, from, next) => {
  if (from?.name === 'Preview' || String(from?.path || '').includes('/preview')) {
    skipNextLoading.value = true
  }
  startLoading()
  next()
})
/** 路由完成后：结束加载 */
const removeAfter = router.afterEach(() => {
  stopLoading()
})
/** 路由错误时：立即结束加载 */
const removeError = router.onError(() => {
  loading.value = false
})

/** 应用就绪时结束初始加载 */
router.isReady().then(() => stopLoading())

watch(
  [() => editorUi.locale.value, () => route.path],
  ([value, path]) => {
    locale.value = resolveAppLocaleForPath(path, value)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  removeBefore()
  removeAfter()
  removeError?.()
})
</script>

<template>
  <el-config-provider :locale="elementLocale">
    <div id="app">
      <router-view />
      <!-- 设计态路由切换时显示加载动画，预览态不显示 -->
      <div v-if="showLoading" class="app-loading">
        <div class="loading-card">
          <div class="loading-mark">
            <div class="loading-ring"></div>
            <div class="loading-dot"></div>
          </div>
          <div class="loading-content">
            <div class="loading-title">设计器加载中</div>
            <div class="loading-subtitle">正在准备画布与资源...</div>
          </div>
        </div>
      </div>
    </div>
  </el-config-provider>
</template>

<style>
#app {
  font-family: 'Inter', 'Helvetica Neue', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  min-height: 100vh;
  height: 100vh;
  overflow: hidden;
}

.app-loading {
  position: fixed;
  inset: 0;
  background: radial-gradient(circle at 30% 20%, rgba(37, 99, 235, 0.12), rgba(15, 23, 42, 0.2));
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  backdrop-filter: blur(3px);
}

.loading-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 22px;
  border-radius: 16px;
  background: linear-gradient(135deg, #ffffff, #f5f7ff);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.2);
}

.loading-mark {
  position: relative;
  width: 36px;
  height: 36px;
}

.loading-ring {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 3px solid rgba(37, 99, 235, 0.2);
  border-top-color: #2563eb;
  animation: loading-spin 0.9s linear infinite;
}

.loading-dot {
  position: absolute;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #2563eb;
  top: 2px;
  left: 50%;
  transform: translateX(-50%);
  animation: loading-pulse 1.2s ease-in-out infinite;
}

.loading-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.loading-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.loading-subtitle {
  font-size: 12px;
  color: #64748b;
}

@keyframes loading-spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes loading-pulse {
  0%,
  100% {
    transform: translateX(-50%) scale(0.9);
    opacity: 0.6;
  }
  50% {
    transform: translateX(-50%) scale(1.15);
    opacity: 1;
  }
}
</style>
