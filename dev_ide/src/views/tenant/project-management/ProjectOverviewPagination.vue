<template>
  <WorkbenchPagination
    :page="page"
    :limit="limit"
    :total="total"
    :total-pages="totalPages"
    :summary="summaryText"
    :page-size-label="t('projectManagement.pageSizeLabel')"
    :page-indicator="pageSummaryText"
    :limit-options="limitOptions"
    @update:page="emit('update:page', $event)"
    @update:limit="emit('update:limit', $event)"
    @change="emit('change', $event)"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import WorkbenchPagination from '@/components/WorkbenchPagination.vue'
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
</script>
