<template>
  <section class="run-events" :aria-busy="loading">
    <p v-if="error" role="status">事件更新中断 <button @click="load">重试</button></p>
    <p v-if="loading" role="status">{{ events.length ? '正在更新事件…' : '正在加载事件…' }}</p>
    <p v-else-if="!events.length && !error">暂无事件</p>
    <ol>
      <li v-for="event in events" :key="event.id">
        <time>{{ formatDateTime(event.createdAt) }}</time
        ><span
          >{{ runEventPresentation(event.stage, event.message).stage }} ·
          <OpsMessage
            :text="event.message"
            :summary="runEventPresentation(event.stage, event.message).message"
        /></span>
      </li>
    </ol>
    <WorkbenchPagination
      v-if="total > 10"
      v-model:page="page"
      :limit="10"
      :total="total"
      :limit-options="[10]"
    />
  </section>
</template>
<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { opsAPI, type DeploymentRunEvent } from '@/api/ops.api'
import { formatDateTime } from '@/utils/date'
import WorkbenchPagination from '@/components/WorkbenchPagination.vue'
import { runEventPresentation } from '../utils/ops-presentation'
import OpsMessage from './OpsMessage.vue'
const props = defineProps<{ runId: string; active: boolean; revision: number; live: boolean }>()
const events = ref<DeploymentRunEvent[]>([]),
  page = ref(1),
  total = ref(0),
  loading = ref(false),
  error = ref(false)
let controller: AbortController | undefined,
  dirty = false,
  disposed = false
// 只为展开任务按页加载；实时通知合并为一次补刷，切页/对象才废弃旧响应。
async function load() {
  controller?.abort()
  if (!props.active || disposed) return
  const current = (controller = new AbortController())
  loading.value = true
  try {
    const result = await opsAPI.listDeploymentRunEventsPage(
      props.runId,
      page.value,
      10,
      current.signal,
    )
    if (current.signal.aborted) return
    total.value = result.total
    const last = Math.max(1, Math.ceil(total.value / 10))
    if (page.value > last) {
      page.value = last
      return
    }
    events.value = result.items
    error.value = false
  } catch {
    if (!current.signal.aborted) error.value = true
  } finally {
    if (controller === current) {
      loading.value = false
      if (dirty && props.active && !disposed) {
        dirty = false
        void load()
      }
    }
  }
}
watch(page, () => void load())
watch(
  () => props.runId,
  () => {
    controller?.abort()
    events.value = []
    total.value = 0
    error.value = false
    dirty = false
    if (page.value !== 1) page.value = 1
    else void load()
  },
)
watch(
  () => props.active,
  (active) => {
    if (active) void load()
    else {
      dirty = false
      controller?.abort()
    }
  },
  { immediate: true },
)
watch(
  () => [props.revision, props.live] as const,
  (_, previous) => {
    if (!props.active || (!props.live && !previous?.[1])) return
    if (loading.value) dirty = true
    else void load()
  },
)
onBeforeUnmount(() => {
  disposed = true
  dirty = false
  controller?.abort()
})
</script>
<style scoped>
.run-events {
  font-size: 12px;
  color: var(--ck-text-secondary, var(--el-text-color-regular));
}
ol {
  list-style: none;
  padding: 0;
  margin: 0;
}
li {
  display: grid;
  grid-template-columns: 130px minmax(0, 1fr);
  gap: 10px;
  padding: 7px 0;
  overflow-wrap: anywhere;
}
time {
  color: var(--ck-text-muted, var(--el-text-color-secondary));
}
button {
  border: 0;
  background: none;
  color: var(--el-color-primary);
  font: inherit;
  cursor: pointer;
}
</style>
