<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  resolvedProps?: Record<string, unknown>
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: number): void
  (event: 'update:currentPage', value: number): void
}>()

const total = computed(() => normalizePositiveNumber(props.resolvedProps?.total, 0))
const pageSize = computed(() => normalizePositiveNumber(props.resolvedProps?.pageSize, 10))
const currentPage = computed(() => normalizePositiveNumber(props.resolvedProps?.currentPage, 1))
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const visiblePages = computed(() =>
  Array.from({ length: Math.min(pageCount.value, 5) }, (_, index) => index + 1),
)

function normalizePositiveNumber(value: unknown, fallback: number): number {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return fallback
  return Math.max(0, Math.floor(parsed))
}

function updateCurrentPage(page: number): void {
  const nextPage = Math.min(Math.max(1, page), pageCount.value)
  if (nextPage === currentPage.value) return
  emit('update:currentPage', nextPage)
  emit('update:modelValue', nextPage)
}
</script>

<template>
  <nav class="core-pagination" aria-label="分页">
    <span class="core-pagination__total">共 {{ total }} 条</span>
    <button
      class="core-pagination__btn"
      type="button"
      :disabled="currentPage <= 1"
      @click="updateCurrentPage(currentPage - 1)"
    >
      &lt;
    </button>
    <button
      v-for="page in visiblePages"
      :key="page"
      class="core-pagination__pager"
      :class="{ 'is-active': page === currentPage }"
      type="button"
      @click="updateCurrentPage(page)"
    >
      {{ page }}
    </button>
    <button
      class="core-pagination__btn"
      type="button"
      :disabled="currentPage >= pageCount"
      @click="updateCurrentPage(currentPage + 1)"
    >
      &gt;
    </button>
  </nav>
</template>

<style scoped>
.core-pagination {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 100%;
  color: #606266;
  font-size: 13px;
  line-height: 1;
  white-space: nowrap;
}

.core-pagination__total {
  margin-right: 2px;
}

.core-pagination__btn,
.core-pagination__pager {
  min-width: 28px;
  height: 28px;
  border: 0;
  border-radius: 4px;
  background: #f4f6f8;
  color: #606266;
  cursor: default;
}

.core-pagination__btn:disabled {
  color: #c0c4cc;
}

.core-pagination__pager.is-active {
  background: #409eff;
  color: #fff;
}
</style>
