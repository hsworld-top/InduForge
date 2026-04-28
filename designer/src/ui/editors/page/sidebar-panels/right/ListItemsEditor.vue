<!--
  通用列表项编辑器：只负责数组项的选择、增删复制和顺序调整。
  具体字段表单由父级通过插槽提供，避免把组件类型细节写进通用层。
-->
<script setup lang="ts">
interface ListItemLike {
  name: string;
  title?: string;
  label?: string;
  disabled?: boolean;
}

const props = defineProps<{
  items: ListItemLike[];
  selectedKey: string;
}>();

const emit = defineEmits<{
  (event: "select", key: string): void;
  (event: "add"): void;
  (event: "duplicate", key: string): void;
  (event: "remove", key: string): void;
  (event: "move", key: string, direction: -1 | 1): void;
}>();

function getItemTitle(item: ListItemLike, index: number): string {
  return item.title || item.label || item.name || `项目 ${index + 1}`;
}

function isSelected(item: ListItemLike): boolean {
  return item.name === props.selectedKey;
}
</script>

<template>
  <div class="list-items-editor">
    <div class="list-items-editor__toolbar">
      <span class="list-items-editor__title">面板项</span>
      <el-button size="small" type="primary" @click="emit('add')">新增</el-button>
    </div>

    <div class="list-items-editor__list">
      <button
        v-for="(item, index) in items"
        :key="item.name"
        type="button"
        class="list-items-editor__row"
        :class="{ 'is-active': isSelected(item) }"
        @click="emit('select', item.name)"
      >
        <span class="list-items-editor__name">
          {{ getItemTitle(item, index) }}
          <span v-if="item.disabled" class="list-items-editor__tag">禁用</span>
        </span>
        <span class="list-items-editor__actions" @click.stop>
          <el-button
            size="small"
            text
            :disabled="index === 0"
            @click="emit('move', item.name, -1)"
          >
            上移
          </el-button>
          <el-button
            size="small"
            text
            :disabled="index === items.length - 1"
            @click="emit('move', item.name, 1)"
          >
            下移
          </el-button>
          <el-button size="small" text @click="emit('duplicate', item.name)">复制</el-button>
          <el-button size="small" text type="danger" @click="emit('remove', item.name)">
            删除
          </el-button>
        </span>
      </button>
    </div>

    <div class="list-items-editor__detail">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.list-items-editor {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}

.list-items-editor__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--designer-gap-xs);
}

.list-items-editor__title {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
  font-weight: 600;
}

.list-items-editor__list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.list-items-editor__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 4px;
  width: 100%;
  min-height: 32px;
  padding: 4px 6px;
  border: 1px solid var(--designer-border-soft);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-shell-surface);
  color: var(--designer-text-regular);
  cursor: pointer;
}

.list-items-editor__row:hover,
.list-items-editor__row.is-active {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
}

.list-items-editor__name {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  overflow: hidden;
  font-size: var(--designer-font-label);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.list-items-editor__tag {
  flex-shrink: 0;
  color: var(--designer-text-muted);
  font-size: 11px;
}

.list-items-editor__actions {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
}

.list-items-editor__actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.list-items-editor__detail {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}
</style>
