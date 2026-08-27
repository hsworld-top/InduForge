<template>
  <DcDialog v-model="visible" width="920px" class="data-contract-dialog" destroy-on-close>
    <template #header>
      <div class="contract-dialog__header">
        <div>
          <div class="contract-dialog__eyebrow">Release Contract Gate</div>
          <h2>{{ dataContractCheckSummary.title }}</h2>
          <p>{{ dataContractCheckSummary.description }}</p>
        </div>
        <div
          class="contract-dialog__result"
          :class="`is-${result?.status || 'idle'}`"
          aria-live="polite"
        >
          <strong>{{ overallStatusText }}</strong>
          <span>{{ checkedAtText }}</span>
        </div>
      </div>
    </template>

    <section class="contract-dialog__overview">
      <div>
        <span>当前项目</span>
        <strong>{{ projectId || '未选择项目' }}</strong>
      </div>
      <div>
        <span>检查范围</span>
        <strong>{{ result?.scope || '项目级' }}</strong>
      </div>
      <div>
        <span>数据来源</span>
        <strong>data_service dry-run</strong>
      </div>
    </section>

    <el-alert
      v-if="errorMessage"
      class="contract-dialog__alert"
      type="error"
      :closable="false"
      show-icon
      :title="errorMessage"
    />

    <div v-loading="loading" class="contract-dialog__content">
      <main class="contract-dialog__main">
        <section v-if="result" class="contract-dialog__summary" aria-label="契约检查摘要">
          <div v-for="item in summaryItems" :key="item.status" :class="`is-${item.status}`">
            <strong>{{ item.count }}</strong>
            <span>{{ item.label }}</span>
          </div>
        </section>

        <section v-if="result?.list.length" class="contract-dialog__items" aria-label="契约检查项">
          <article
            v-for="(item, index) in result.list"
            :key="`${item.module}-${item.objectType}-${item.objectId || index}`"
            class="contract-check-card"
            :class="`contract-check-card--${item.status}`"
          >
            <div class="contract-check-card__status">
              {{ dataContractCheckStatusText[item.status] }}
            </div>
            <div class="contract-check-card__main">
              <div class="contract-check-card__title-row">
                <h3>{{ item.title }}</h3>
                <el-tag effect="plain" size="small">
                  {{ dataContractCheckModuleText[item.module] || item.module }}
                </el-tag>
              </div>
              <p v-if="item.detail">{{ item.detail }}</p>
              <div v-if="item.action" class="contract-check-card__action">
                建议：{{ item.action }}
              </div>
            </div>
            <div class="contract-check-card__owner">
              {{ item.objectType
              }}<template v-if="item.objectId"><br />{{ item.objectId }}</template>
            </div>
          </article>
        </section>

        <section v-else-if="result && !loading" class="contract-dialog__empty">
          <strong>未发现契约问题</strong>
          <span>本次 data_service 检查范围内没有失败、警告或待完成项。</span>
        </section>

        <section v-else-if="!loading" class="contract-dialog__empty">
          <strong>尚未获取检查结果</strong>
          <span>点击“运行检查”后获取真实项目契约状态。</span>
        </section>
      </main>

      <aside class="contract-dialog__history" aria-label="最近检查记录">
        <div class="contract-dialog__history-title">
          <strong>最近检查</strong>
          <span v-if="historyPagination.total">{{ historyPagination.total }} 条</span>
        </div>
        <div v-loading="historyLoading" class="contract-dialog__history-list">
          <div v-for="run in history" :key="run.id" class="contract-dialog__history-item">
            <span :class="`is-${run.status}`">{{ dataContractCheckStatusText[run.status] }}</span>
            <time>{{ formatTime(run.createdAt) }}</time>
          </div>
          <p v-if="!historyLoading && !history.length">暂无历史记录</p>
        </div>
      </aside>
    </div>

    <section class="contract-dialog__note">
      <strong>边界说明</strong>
      <span>{{ dataContractCheckSummary.extensionNote }}</span>
    </section>

    <template #footer>
      <div class="contract-dialog__footer">
        <span>检查结果是发布准入依据，不会修改项目配置。</span>
        <div>
          <el-button @click="visible = false">关闭</el-button>
          <el-button
            type="primary"
            :loading="running"
            :disabled="!normalizedProjectId || loading"
            @click="handleRun"
          >
            {{ result ? '重新检查' : '运行检查' }}
          </el-button>
        </div>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  getContractChecks,
  getLatestContractCheck,
  runContractCheck,
} from '@/api/contract-check.api'
import type {
  ContractCheckResult,
  ContractCheckRun,
  ContractCheckStatus,
} from '@/api/schemas/contract-check.schema'
import { getApiErrorMessage } from '@/utils/request'
import DcDialog from '@/components/shared/DcDialog.vue'
import {
  dataContractCheckModuleText,
  dataContractCheckStatusText,
  dataContractCheckSummary,
} from '@/components/contract/dataContractCheck'

const props = defineProps<{
  modelValue: boolean
  projectId?: string | number
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const normalizedProjectId = computed(() => String(props.projectId || '').trim())
const result = ref<ContractCheckResult | null>(null)
const history = ref<ContractCheckRun[]>([])
const historyPagination = ref({ page: 1, pageSize: 8, total: 0, totalPages: 0 })
const loading = ref(false)
const running = ref(false)
const historyLoading = ref(false)
const errorMessage = ref('')
let loadSequence = 0

const overallStatusText = computed(() => {
  if (running.value) return '检查中'
  if (!result.value) return '未检查'
  return dataContractCheckStatusText[result.value.status]
})
const checkedAtText = computed(() =>
  result.value?.checkedAt ? formatTime(result.value.checkedAt) : '无检查时间',
)
const summaryItems = computed(() => {
  const summary = result.value?.summary
  const statuses: ContractCheckStatus[] = ['failed', 'warning', 'pending', 'passed']
  return statuses.map((status) => ({
    status,
    label: dataContractCheckStatusText[status],
    count: Number(summary?.[status] || 0),
  }))
})

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date)
}

async function loadHistory(projectId: string, sequence: number) {
  historyLoading.value = true
  try {
    const page = await getContractChecks(projectId, { page: 1, pageSize: 8 })
    if (sequence !== loadSequence) return
    history.value = page.list
    historyPagination.value = page.pagination
  } catch (error) {
    if (sequence === loadSequence && !errorMessage.value) {
      errorMessage.value = getApiErrorMessage(error, '加载契约检查历史失败')
    }
  } finally {
    if (sequence === loadSequence) historyLoading.value = false
  }
}

async function loadLatest() {
  const projectId = normalizedProjectId.value
  const sequence = ++loadSequence
  result.value = null
  history.value = []
  errorMessage.value = ''
  if (!projectId) {
    errorMessage.value = '缺少工程上下文，无法执行契约检查'
    return
  }
  loading.value = true
  await Promise.allSettled([
    getLatestContractCheck(projectId).then((value) => {
      if (sequence === loadSequence) result.value = value
    }),
    loadHistory(projectId, sequence),
  ])
  if (sequence === loadSequence && !result.value && !errorMessage.value) {
    errorMessage.value = '加载最近契约检查结果失败'
  }
  if (sequence === loadSequence) loading.value = false
}

async function handleRun() {
  const projectId = normalizedProjectId.value
  if (!projectId || running.value) return
  const sequence = ++loadSequence
  running.value = true
  errorMessage.value = ''
  try {
    const value = await runContractCheck(projectId)
    if (sequence !== loadSequence) return
    result.value = value
    await loadHistory(projectId, sequence)
  } catch (error) {
    if (sequence === loadSequence) {
      errorMessage.value = getApiErrorMessage(error, '运行数据契约检查失败')
    }
  } finally {
    if (sequence === loadSequence) running.value = false
  }
}

watch(
  () => [visible.value, normalizedProjectId.value] as const,
  ([isVisible]) => {
    if (isVisible) void loadLatest()
    else loadSequence++
  },
)
</script>

<style scoped>
.contract-dialog__header,
.contract-dialog__footer,
.contract-dialog__history-title,
.contract-check-card__title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}
.contract-dialog__eyebrow {
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 700;
}
.contract-dialog__header h2 {
  margin: 6px 0;
  color: var(--dc-text);
  font-size: 20px;
}
.contract-dialog__header p {
  max-width: 650px;
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}
.contract-dialog__result {
  min-width: 116px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-subtle);
  text-align: center;
}
.contract-dialog__result strong,
.contract-dialog__result span {
  display: block;
}
.contract-dialog__result strong {
  color: var(--dc-text);
  font-size: 18px;
}
.contract-dialog__result span {
  margin-top: 5px;
  color: var(--dc-text-secondary);
  font-size: 11px;
}
.contract-dialog__result.is-passed strong {
  color: #15803d;
}
.contract-dialog__result.is-warning strong,
.contract-dialog__result.is-pending strong {
  color: #b45309;
}
.contract-dialog__result.is-failed strong {
  color: #b91c1c;
}
.contract-dialog__overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}
.contract-dialog__overview > div {
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-subtle);
}
.contract-dialog__overview span,
.contract-dialog__overview strong {
  display: block;
}
.contract-dialog__overview span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.contract-dialog__overview strong {
  margin-top: 3px;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.contract-dialog__alert {
  margin-bottom: 12px;
}
.contract-dialog__content {
  min-height: 260px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 190px;
  gap: 12px;
}
.contract-dialog__main {
  min-width: 0;
}
.contract-dialog__summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 10px;
}
.contract-dialog__summary > div {
  padding: 9px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.contract-dialog__summary strong {
  margin-right: 6px;
  color: var(--dc-text);
}
.contract-dialog__summary span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.contract-dialog__summary .is-failed strong {
  color: #b91c1c;
}
.contract-dialog__summary .is-warning strong,
.contract-dialog__summary .is-pending strong {
  color: #b45309;
}
.contract-dialog__summary .is-passed strong {
  color: #15803d;
}
.contract-dialog__items {
  display: grid;
  gap: 8px;
  max-height: 360px;
  overflow: auto;
  padding-right: 3px;
}
.contract-check-card {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr) 105px;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
}
.contract-check-card__status {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
}
.contract-check-card--failed .contract-check-card__status {
  background: #fee2e2;
  color: #991b1b;
}
.contract-check-card--warning .contract-check-card__status,
.contract-check-card--pending .contract-check-card__status {
  background: #fef3c7;
  color: #92400e;
}
.contract-check-card--passed .contract-check-card__status {
  background: #dcfce7;
  color: #166534;
}
.contract-check-card__title-row {
  align-items: center;
}
.contract-check-card h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 14px;
}
.contract-check-card p {
  margin: 6px 0 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.contract-check-card__action {
  margin-top: 7px;
  color: var(--dc-primary);
  font-size: 12px;
}
.contract-check-card__owner {
  overflow-wrap: anywhere;
  color: var(--dc-text-secondary);
  font-size: 11px;
  text-align: right;
}
.contract-dialog__empty {
  min-height: 210px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-md);
  color: var(--dc-text-secondary);
  text-align: center;
}
.contract-dialog__empty strong {
  color: var(--dc-text);
}
.contract-dialog__empty span {
  max-width: 420px;
  font-size: 12px;
}
.contract-dialog__history {
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-subtle);
}
.contract-dialog__history-title {
  align-items: center;
}
.contract-dialog__history-title span {
  color: var(--dc-text-secondary);
  font-size: 11px;
}
.contract-dialog__history-list {
  min-height: 150px;
  margin-top: 10px;
}
.contract-dialog__history-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 7px 0;
  border-bottom: 1px solid var(--dc-border);
  font-size: 11px;
}
.contract-dialog__history-item > span {
  font-weight: 700;
}
.contract-dialog__history-item .is-passed {
  color: #15803d;
}
.contract-dialog__history-item .is-failed {
  color: #b91c1c;
}
.contract-dialog__history-item .is-warning,
.contract-dialog__history-item .is-pending {
  color: #b45309;
}
.contract-dialog__history-item time,
.contract-dialog__history-list p {
  color: var(--dc-text-secondary);
}
.contract-dialog__note {
  display: flex;
  gap: 10px;
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.contract-dialog__note strong {
  flex: 0 0 auto;
  color: var(--dc-text);
}
.contract-dialog__footer {
  align-items: center;
}
.contract-dialog__footer > span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.contract-dialog__footer > div {
  display: flex;
  gap: 8px;
}
@media (max-width: 760px) {
  .contract-dialog__header,
  .contract-dialog__footer {
    align-items: stretch;
    flex-direction: column;
  }
  .contract-dialog__overview {
    grid-template-columns: 1fr;
  }
  .contract-dialog__content {
    grid-template-columns: 1fr;
  }
  .contract-dialog__history {
    order: -1;
  }
  .contract-dialog__summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .contract-check-card {
    grid-template-columns: 58px minmax(0, 1fr);
  }
  .contract-check-card__owner {
    grid-column: 2;
    text-align: left;
  }
}
</style>
