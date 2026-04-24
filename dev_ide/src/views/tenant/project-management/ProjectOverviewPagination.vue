<template>
  <div
    class="project-overview-pagination flex items-center justify-between gap-3 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2"
  >
    <p class="text-xs text-gray-500 dark:text-gray-400">{{ summaryText }}</p>

    <div class="flex items-center gap-2">
      <span class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('projectManagement.pageSizeLabel') }}
      </span>
      <el-select
        :model-value="limit"
        size="small"
        class="!w-20"
        @update:model-value="handleLimitChange"
      >
        <el-option
          v-for="sizeOption in limitOptions"
          :key="sizeOption"
          :label="String(sizeOption)"
          :value="sizeOption"
        />
      </el-select>

      <el-button size="small" text :disabled="!canPrev" @click="handlePrev">
        <el-icon><ArrowLeft /></el-icon>
      </el-button>
      <el-button size="small" text :disabled="!canNext" @click="handleNext">
        <el-icon><ArrowRight /></el-icon>
      </el-button>

      <span class="text-xs text-gray-500 dark:text-gray-400 min-w-[74px] text-right">
        {{ pageSummaryText }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowLeft, ArrowRight } from '@element-plus/icons-vue'
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
