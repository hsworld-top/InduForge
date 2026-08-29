<template>
  <div
    v-if="selectedCount > 0"
    class="dc-bulk-action-bar"
    :class="{ 'is-inline': placement === 'inline' }"
    role="status"
  >
    <span class="dc-bulk-action-bar__count">
      {{ summary || ui(`已选 ${selectedCount} ${itemLabel}`, `${selectedCount} ${itemLabel} selected`) }}
    </span>
    <div class="dc-bulk-action-bar__actions">
      <slot />
    </div>
    <button type="button" class="dc-bulk-action-bar__clear" @click="$emit('clear')">
      {{ ui('清空选择', 'Clear Selection') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

withDefaults(
  defineProps<{
    selectedCount: number
    /** 行内模式用于表格上方；默认保持列表页已有的悬浮操作条。 */
    placement?: 'floating' | 'inline'
    /** 用于“数据点 / 报警项”等不同资源的数量文案。 */
    itemLabel?: string
    /** 有“全部结果”范围时由调用方提供完整摘要。 */
    summary?: string
  }>(),
  { placement: 'floating', itemLabel: datacenterLocale.value === 'en' ? 'items' : '项', summary: '' },
)

defineEmits<{
  (event: 'clear'): void
}>()
</script>

<style scoped>
.dc-bulk-action-bar {
  position: absolute;
  left: 50%;
  bottom: 16px;
  z-index: 20;
  max-width: calc(100% - 32px);
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 8px 10px 8px 14px;
  border: 1px solid rgba(29, 78, 216, 0.2);
  border-radius: 12px;
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-popover);
  color: var(--dc-text);
  transform: translateX(-50%);
}

.dc-bulk-action-bar__count {
  flex: 0 0 auto;
  color: var(--dc-text-secondary);
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}

.dc-bulk-action-bar__actions {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 auto;
  flex-wrap: wrap;
}

.dc-bulk-action-bar__clear {
  height: 28px;
  flex: 0 0 auto;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 600;
}

.dc-bulk-action-bar__clear:hover {
  border-color: rgba(29, 78, 216, 0.22);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dc-bulk-action-bar.is-inline {
  position: static;
  width: 100%;
  max-width: none;
  min-height: 42px;
  padding: 7px 14px;
  border-width: 0 0 1px;
  border-radius: 0;
  box-shadow: none;
  transform: none;
}
</style>
