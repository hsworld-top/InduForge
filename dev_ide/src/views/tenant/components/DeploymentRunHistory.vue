<template>
  <section class="deployment-history">
    <details v-if="recent?.completedAt" class="recent-run">
      <summary :class="{ 'run-result-failed': runResultLabel(recent) === '失败' }">
        最近操作 · {{ runOperationLabel(recent) }} · {{ runResultLabel(recent) }} ·
        {{ runDurationLabel(recent) }}
      </summary>
      <p>
        {{ recent.actorDisplayName || '未知用户' }} · {{ formatDateTime(recent.startedAt || '') }}
      </p>
    </details>
    <h3>操作历史</h3>
    <p v-if="error" role="status">记录更新中断 <button @click="loadRuns">重试</button></p>
    <p v-if="loading && !runs.length">正在加载操作记录…</p>
    <p v-else-if="loading" class="history-updating" role="status">正在更新记录…</p>
    <p v-else-if="!runs.length && !error">暂无操作记录</p>
    <table v-if="runs.length" :aria-busy="loading">
      <thead>
        <tr>
          <th>操作</th>
          <th>发起人</th>
          <th>时间</th>
          <th>耗时</th>
          <th>结果</th>
        </tr>
      </thead>
      <tbody>
        <template v-for="run in runs" :key="run.id">
          <tr>
            <td>
              <button :aria-expanded="expandedId === run.id" @click="expand(run.id)">
                {{ expandedId === run.id ? '▾' : '▸' }} {{ runOperationLabel(run) }}
              </button>
            </td>
            <td>{{ run.actorDisplayName || '未知用户' }}</td>
            <td>{{ formatDateTime(run.startedAt || '') }}</td>
            <td>{{ runDurationLabel(run) }}</td>
            <td
              :class="
                runResultLabel(run) === '失败'
                  ? 'run-result-failed'
                  : run.completedAt
                    ? 'run-result-success'
                    : 'run-result-running'
              "
            >
              {{ runResultLabel(run) }}
            </td>
          </tr>
          <tr v-if="expandedId === run.id">
            <td colspan="5" class="event-cell">
              <DeploymentRunEvents
                :key="run.id"
                :run-id="run.id"
                :active="active"
                :revision="revision"
                :live="!run.completedAt"
              />
            </td>
          </tr>
        </template>
      </tbody>
    </table>
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
import { opsAPI, type DeploymentRun } from '@/api/ops.api'
import WorkbenchPagination from '@/components/WorkbenchPagination.vue'
import { formatDateTime } from '@/utils/date'
import DeploymentRunEvents from './DeploymentRunEvents.vue'
import { runOperationLabel, runDurationLabel, runResultLabel } from '../utils/deployment-details'
const props = defineProps<{
  deploymentId: string
  latestRunId?: string
  revision: number
  active: boolean
}>()
const runs = ref<DeploymentRun[]>([])
const recent = ref<DeploymentRun>()
const page = ref(1),
  total = ref(0),
  expandedId = ref('')
const loading = ref(false),
  error = ref(false)
let runsController: AbortController | undefined
let runsDirty = false,
  disposed = false
function refreshRuns() {
  if (loading.value) runsDirty = true
  else void loadRuns()
}
// 切页、切对象和隐藏时取消旧请求；普通实时变更只补当前可见任务，不逐行加载事件。
async function loadRuns() {
  runsController?.abort()
  if (!props.active || disposed) return
  const controller = (runsController = new AbortController())
  loading.value = true
  try {
    const result = await opsAPI.listProjectDeploymentRuns(
      props.deploymentId,
      page.value,
      10,
      controller.signal,
    )
    if (controller.signal.aborted) return
    total.value = result.total
    const last = Math.max(1, Math.ceil(result.total / 10))
    if (page.value > last) {
      page.value = last
      return
    }
    runs.value = result.items
    const current = result.items.find((run) => run.id === props.latestRunId)
    if (current) recent.value = current
    else if (
      props.latestRunId &&
      (recent.value?.id !== props.latestRunId || !recent.value.completedAt)
    ) {
      const current = await opsAPI.getDeploymentRun(props.latestRunId, controller.signal)
      if (controller.signal.aborted) return
      recent.value = current
    }
    error.value = false
  } catch {
    if (!controller.signal.aborted) error.value = true
  } finally {
    if (runsController === controller) {
      loading.value = false
      if (runsDirty && props.active && !disposed) {
        runsDirty = false
        void loadRuns()
      }
    }
  }
}
function expand(id: string) {
  expandedId.value = expandedId.value === id ? '' : id
}
watch(page, () => {
  expandedId.value = ''
  void loadRuns()
})
watch(
  () => props.deploymentId,
  () => {
    runsController?.abort()
    runs.value = []
    recent.value = undefined
    total.value = 0
    expandedId.value = ''
    error.value = false
    if (page.value !== 1) page.value = 1
    else void loadRuns()
  },
)
watch(
  () => props.active,
  (active) => {
    if (active) {
      void loadRuns()
    } else {
      runsDirty = false
      runsController?.abort()
    }
  },
  { immediate: true },
)
watch(
  () => props.revision,
  () => {
    if (!props.active) return
    refreshRuns()
  },
)
onBeforeUnmount(() => {
  disposed = true
  runsDirty = false
  runsController?.abort()
})
</script>

<style scoped>
.deployment-history {
  margin-top: 16px;
  font-size: 12px;
}
h3 {
  font-size: 13px;
  margin: 0 0 12px;
}
.recent-run {
  margin-bottom: 14px;
  color: var(--el-text-color-secondary);
}
.recent-run summary {
  cursor: pointer;
}
table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}
th,
td {
  padding: 10px 6px;
  text-align: left;
  border-bottom: 1px solid var(--el-border-color-lighter);
  overflow-wrap: anywhere;
}
th {
  color: var(--el-text-color-secondary);
  font-weight: 500;
}
th:nth-child(3) {
  width: 30%;
}
button {
  border: 0;
  background: none;
  color: var(--el-color-primary);
  cursor: pointer;
  padding: 0;
  font: inherit;
  text-align: left;
}
.event-cell {
  background: var(--el-fill-color-light);
}
ol {
  list-style: none;
  padding: 0;
  margin: 0;
}
li {
  display: grid;
  grid-template-columns: 130px 1fr;
  gap: 10px;
  padding: 7px 0;
  overflow-wrap: anywhere;
}
time {
  color: var(--el-text-color-secondary);
}
.run-result-failed {
  color: var(--ck-danger, var(--el-color-danger));
}
.run-result-running {
  color: var(--ck-primary, var(--el-color-primary));
}
.run-result-success {
  color: var(--el-color-success);
}
</style>
