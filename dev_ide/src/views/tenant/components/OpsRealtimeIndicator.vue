<template>
  <span class="ops-live-indicator" role="status" :title="label">
    <i :class="{ online: status === 'online', connecting: status === 'connecting' }" />
    {{ status === 'online' ? '实时' : label }}
  </span>
</template>
<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{ status: 'online' | 'connecting' | 'offline' | 'stale' | 'forbidden' }>()
const label = computed(
  () =>
    ({
      online: '实时同步已连接',
      connecting: '实时连接中',
      offline: '连接断开，低频对账',
      stale: '数据已过期，重试中',
      forbidden: '实时订阅权限已失效',
    })[props.status],
)
</script>
<style scoped>
.ops-live-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
  white-space: nowrap;
}
i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--el-color-warning);
}
i.online {
  background: var(--el-color-success);
}
i.connecting {
  animation: ops-live-pulse 1.5s ease-in-out infinite;
}
@keyframes ops-live-pulse {
  50% {
    opacity: 0.35;
  }
}
@media (prefers-reduced-motion: reduce) {
  i.connecting {
    animation: none;
  }
}
</style>
