<template>
  <div
    class="project-overview-pagination"
  >
    <p class="pagination-summary">{{ summaryText }}</p>

    <div class="pagination-controls">
      <span class="pagination-label">
        {{ t('projectManagement.pageSizeLabel') }}
      </span>
      <el-popover placement="top" :width="100" trigger="click">
        <template #reference>
          <button type="button" class="pagination-pill-btn">
            {{ limit }}
            <el-icon class="text-[10px] ml-0.5"><ArrowDown /></el-icon>
          </button>
        </template>
        <div class="pagination-size-menu">
          <button
            v-for="sizeOption in limitOptions"
            :key="sizeOption"
            type="button"
            class="pagination-size-item"
            :class="{ 'is-active': limit === sizeOption }"
            @click="handleLimitChange(sizeOption)"
          >
            {{ sizeOption }}
          </button>
        </div>
      </el-popover>

      <div class="pagination-nav-group">
        <button
          type="button"
          class="pagination-nav-btn"
          :disabled="!canPrev"
          @click="handlePrev"
        >
          <el-icon><ArrowLeft /></el-icon>
        </button>
        <button
          type="button"
          class="pagination-nav-btn"
          :disabled="!canNext"
          @click="handleNext"
        >
          <el-icon><ArrowRight /></el-icon>
        </button>
      </div>

      <span class="pagination-page-info">
        {{ pageSummaryText }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowDown, ArrowLeft, ArrowRight } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import { buildProjectPaginationSummary } from './use-project-overview'

const props = withDefaults(
  defineProps<{
    page: number
    limit: number
    total: number
    totalPages?: number
    summary?: string
    limitOptions?: number[]
  }>(),
  {
    totalPages: 0,
    summary: '',
    limitOptions: () => [10, 20, 50, 100],
  },
)

const emit = defineEmits<{
  (event: 'update:page', value: number): void
  (event: 'update:limit', value: number): void
  (event: 'change', value: { page: number; limit: number }): void
}>()

const { t } = useI18n()

const safeLimit = computed(() => {
  const parsed = Number.parseInt(String(props.limit), 10)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 20
})

const safeTotal = computed(() => {
  const parsed = Number.parseInt(String(props.total), 10)
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : 0
})

const safeTotalPages = computed(() => {
  if (props.totalPages && props.totalPages > 0) {
    return props.totalPages
  }
  if (safeTotal.value <= 0) {
    return 0
  }
  return Math.ceil(safeTotal.value / safeLimit.value)
})

const safePage = computed(() => {
  const parsed = Number.parseInt(String(props.page), 10)
  const normalized = Number.isInteger(parsed) && parsed > 0 ? parsed : 1
  if (safeTotalPages.value <= 0) {
    return 1
  }
  return Math.min(normalized, safeTotalPages.value)
})

const summaryText = computed(() => {
  if (props.summary) {
    return props.summary
  }
  return buildProjectPaginationSummary({
    page: safePage.value,
    limit: safeLimit.value,
    total: safeTotal.value,
  })
})

const pageSummaryText = computed(() => {
  if (safeTotalPages.value <= 0) {
    return t('projectManagement.pageIndicator', { page: 0, totalPages: 0 })
  }
  return t('projectManagement.pageIndicator', {
    page: safePage.value,
    totalPages: safeTotalPages.value,
  })
})

const canPrev = computed(() => safeTotalPages.value > 0 && safePage.value > 1)
const canNext = computed(() => safeTotalPages.value > 0 && safePage.value < safeTotalPages.value)

const emitChange = (nextPage: number, nextLimit: number) => {
  emit('change', { page: nextPage, limit: nextLimit })
}

const handlePrev = () => {
  if (!canPrev.value) {
    return
  }
  const nextPage = safePage.value - 1
  emit('update:page', nextPage)
  emitChange(nextPage, safeLimit.value)
}

const handleNext = () => {
  if (!canNext.value) {
    return
  }
  const nextPage = safePage.value + 1
  emit('update:page', nextPage)
  emitChange(nextPage, safeLimit.value)
}

const handleLimitChange = (value: number | string) => {
  const parsed = Number.parseInt(String(value), 10)
  const nextLimit = Number.isInteger(parsed) && parsed > 0 ? parsed : safeLimit.value
  emit('update:limit', nextLimit)
  emit('update:page', 1)
  emitChange(1, nextLimit)
}
</script>

<style scoped>
.project-overview-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 4px;
}

.pagination-summary {
  font-size: 12px;
  color: var(--ck-text-muted, #94a3b8);
  margin: 0;
  white-space: nowrap;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pagination-label {
  font-size: 12px;
  color: var(--ck-text-muted, #94a3b8);
  white-space: nowrap;
}

/* pill 按钮：每页条数选择器 */
.pagination-pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 0 10px;
  height: 28px;
  border-radius: 10px;
  border: none;
  background: rgba(0, 0, 0, 0.04);
  color: var(--ck-text-secondary, #475569);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.pagination-pill-btn:hover {
  background: rgba(0, 0, 0, 0.08);
  color: var(--ck-text-primary, #0f172a);
}

html.dark .pagination-pill-btn,
[data-theme='dark'] .pagination-pill-btn {
  background: rgba(255, 255, 255, 0.06);
  color: var(--ck-text-secondary);
}

html.dark .pagination-pill-btn:hover,
[data-theme='dark'] .pagination-pill-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}

/* 导航按钮组：前后羻页 */
.pagination-nav-group {
  display: flex;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 10px;
  padding: 2px;
}

html.dark .pagination-nav-group,
[data-theme='dark'] .pagination-nav-group {
  background: rgba(255, 255, 255, 0.06);
}

.pagination-nav-btn {
  width: 28px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--ck-text-secondary, #475569);
  cursor: pointer;
  transition: all 0.2s;
  font-size: 14px;
}

.pagination-nav-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.6);
  color: var(--ck-text-primary, #0f172a);
}

.pagination-nav-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

html.dark .pagination-nav-btn:hover:not(:disabled),
[data-theme='dark'] .pagination-nav-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.08);
}

/* 页码信息 */
.pagination-page-info {
  font-size: 12px;
  color: var(--ck-text-muted, #94a3b8);
  min-width: 64px;
  text-align: right;
  white-space: nowrap;
}
</style>
