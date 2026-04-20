<template>
  <div id="app">
    <router-view />
    <div v-if="loading" class="app-loading">
      <div class="loading-card">
        <div class="loading-mark">
          <div class="loading-ring"></div>
          <div class="loading-dot"></div>
        </div>
        <div class="loading-content">
          <div class="loading-title">数据中心加载中</div>
          <div class="loading-subtitle">正在连接数据服务...</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, onBeforeUnmount } from "vue";
import { useRouter } from "vue-router";

const loading = ref(true);
let loadingStartAt = Date.now();
const minLoadingMs = 500;
const router = useRouter();

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
  font-family: "Inter", "Helvetica Neue", Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  min-height: 100vh;
  height: 100vh;
  overflow: hidden;
}

.app-loading {
  position: fixed;
  inset: 0;
  background: radial-gradient(
    circle at 30% 20%,
    rgba(14, 165, 233, 0.12),
    rgba(15, 23, 42, 0.2)
  );
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
  background: linear-gradient(135deg, #ffffff, #f0f9ff);
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
  border: 3px solid rgba(14, 165, 233, 0.2);
  border-top-color: #0ea5e9;
  animation: loading-spin 0.9s linear infinite;
}

.loading-dot {
  position: absolute;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #0ea5e9;
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
