<template>
  <div id="app">
    <el-config-provider :locale="elementLocale">
      <router-view />
    </el-config-provider>
    <div v-if="loading" class="app-loading">
      <div class="loading-card">
        <div class="loading-mark">
          <div class="loading-ring"></div>
          <div class="loading-dot"></div>
        </div>
        <div class="loading-content">
          <div class="loading-title">{{ t("app.loadingTitle") }}</div>
          <div class="loading-subtitle">{{ t("app.loadingSubtitle") }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onBeforeUnmount } from "vue";
import { useRouter } from "vue-router";
import { elementPlusLocale, t } from "./i18n/runtime";

const loading = ref(true);
let loadingStartAt = Date.now();
const minLoadingMs = 500;
const router = useRouter();
const elementLocale = elementPlusLocale;

const startLoading = () => {
  loadingStartAt = Date.now();
  loading.value = true;
};

const stopLoading = async () => {
  const elapsed = Date.now() - loadingStartAt;
  const waitMs = Math.max(0, minLoadingMs - elapsed);
  if (waitMs > 0) {
    await new Promise((resolve) => setTimeout(resolve, waitMs));
  }
  await nextTick();
  loading.value = false;
};

const removeBefore = router.beforeEach((to, from, next) => {
  startLoading();
  next();
});
const removeAfter = router.afterEach(() => {
  stopLoading();
});
const removeError = router.onError(() => {
  loading.value = false;
});

router.isReady().then(() => stopLoading());

onBeforeUnmount(() => {
  removeBefore();
  removeAfter();
  removeError?.();
});
</script>

<style>
#app {
  font-family: var(--dc-font-sans);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  min-height: 100vh;
  height: 100vh;
  overflow: hidden;
}

.app-loading {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.18);
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
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-popover);
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
  border: 3px solid rgba(29, 78, 216, 0.16);
  border-top-color: var(--dc-primary);
  animation: loading-spin 0.9s linear infinite;
}

.loading-dot {
  position: absolute;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--dc-primary);
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
  color: var(--dc-text);
}

.loading-subtitle {
  font-size: 12px;
  color: var(--dc-text-secondary);
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
