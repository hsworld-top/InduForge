<template>
  <footer class="dc-pagination">
    <p class="dc-pagination__summary">{{ summary }}</p>

    <div class="dc-pagination__controls">
      <span class="dc-pagination__label">每页</span>
      <el-popover placement="top" :width="100" trigger="click">
        <template #reference>
          <button type="button" class="dc-pagination__size-button">
            {{ pageSize }}
            <IconTablerChevronDown />
          </button>
        </template>
        <div class="dc-pagination__size-menu">
          <button
            v-for="size in pageSizeOptions"
            :key="size"
            type="button"
            class="dc-pagination__size-item"
            :class="{ 'is-active': pageSize === size }"
            @click="changePageSize(size)"
          >
            {{ size }}
          </button>
        </div>
      </el-popover>

      <div class="dc-pagination__nav">
        <button
          type="button"
          :disabled="!canPrev"
          aria-label="上一页"
          @click="changePage(page - 1)"
        >
          <IconTablerChevronLeft />
        </button>
        <input
          v-model="jumpPage"
          type="number"
          min="1"
          :max="Math.max(totalPages, 1)"
          :disabled="totalPages === 0"
          aria-label="跳转页码"
          @blur="applyJumpPage"
          @keydown.enter.prevent="applyJumpPage"
        />
        <button
          type="button"
          :disabled="!canNext"
          aria-label="下一页"
          @click="changePage(page + 1)"
        >
          <IconTablerChevronRight />
        </button>
      </div>

      <span class="dc-pagination__indicator">{{ pageIndicator }}</span>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerChevronLeft from '~icons/tabler/chevron-left'
import IconTablerChevronRight from '~icons/tabler/chevron-right'

const props = withDefaults(
  defineProps<{
    page: number
    pageSize: number
    total: number
    totalPages?: number
    pageSizeOptions?: number[]
  }>(),
  {
    totalPages: 0,
    pageSizeOptions: () => [10, 20, 50],
  },
)

const emit = defineEmits<{
  (event: 'change', value: { page: number; pageSize: number }): void
}>()

const resolvedTotalPages = computed(() => {
  if (props.totalPages > 0) return props.totalPages
  return props.total > 0 ? Math.ceil(props.total / props.pageSize) : 0
})
const currentPage = computed(() => {
  if (resolvedTotalPages.value === 0) return 1
  return Math.min(Math.max(props.page, 1), resolvedTotalPages.value)
})
const summary = computed(() => {
  if (props.total === 0) return '共 0 条'
  const start = (currentPage.value - 1) * props.pageSize + 1
  const end = Math.min(currentPage.value * props.pageSize, props.total)
  return `${start}-${end} / 共 ${props.total} 条`
})
const pageIndicator = computed(
  () => `${resolvedTotalPages.value === 0 ? 0 : currentPage.value} / ${resolvedTotalPages.value}`,
)
const canPrev = computed(() => resolvedTotalPages.value > 0 && currentPage.value > 1)
const canNext = computed(
  () => resolvedTotalPages.value > 0 && currentPage.value < resolvedTotalPages.value,
)
const jumpPage = ref(String(currentPage.value))

watch(currentPage, (value) => {
  jumpPage.value = String(value)
})

const changePage = (page: number) => {
  if (resolvedTotalPages.value === 0) return
  const nextPage = Math.min(Math.max(page, 1), resolvedTotalPages.value)
  if (nextPage === currentPage.value) return
  emit('change', { page: nextPage, pageSize: props.pageSize })
}

const changePageSize = (pageSize: number) => {
  if (pageSize === props.pageSize) return
  emit('change', { page: 1, pageSize })
}

const applyJumpPage = () => {
  const parsed = Number.parseInt(jumpPage.value, 10)
  if (!Number.isInteger(parsed)) {
    jumpPage.value = String(currentPage.value)
    return
  }
  changePage(parsed)
  jumpPage.value = String(Math.min(Math.max(parsed, 1), Math.max(resolvedTotalPages.value, 1)))
}
</script>

<style scoped>
.dc-pagination {
  width: 100%;
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 14px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.dc-pagination__summary,
.dc-pagination__label,
.dc-pagination__indicator {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  white-space: nowrap;
}

.dc-pagination__controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dc-pagination__size-button {
  height: 28px;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0 10px;
  border: 0;
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.05);
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.dc-pagination__size-button:hover {
  background: rgba(15, 23, 42, 0.09);
  color: var(--dc-text);
}

.dc-pagination__size-button svg {
  width: 12px;
  height: 12px;
}

.dc-pagination__size-menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: -6px -8px;
}

.dc-pagination__size-item {
  width: 100%;
  padding: 6px 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.dc-pagination__size-item:hover {
  background: rgba(15, 23, 42, 0.05);
  color: var(--dc-text);
}

.dc-pagination__size-item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-weight: 700;
}

.dc-pagination__nav {
  display: flex;
  align-items: center;
  gap: 3px;
  padding: 2px;
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.05);
}

.dc-pagination__nav button {
  width: 28px;
  height: 24px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--dc-text-secondary);
}

.dc-pagination__nav button:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.7);
  color: var(--dc-text);
}

.dc-pagination__nav button:disabled {
  cursor: not-allowed;
  opacity: 0.32;
}

.dc-pagination__nav svg {
  width: 14px;
  height: 14px;
}

.dc-pagination__nav input {
  width: 44px;
  height: 24px;
  padding: 0 5px;
  border: 0;
  border-radius: 8px;
  outline: none;
  background: rgba(255, 255, 255, 0.78);
  color: var(--dc-text);
  font-size: 12px;
  text-align: center;
}

.dc-pagination__nav input:focus {
  box-shadow: inset 0 0 0 1px color-mix(in oklch, var(--dc-primary) 45%, transparent);
}

.dc-pagination__nav input::-webkit-outer-spin-button,
.dc-pagination__nav input::-webkit-inner-spin-button {
  margin: 0;
  appearance: none;
}

.dc-pagination__indicator {
  min-width: 64px;
  text-align: right;
}

@media (max-width: 720px) {
  .dc-pagination {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
