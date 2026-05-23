<!--
  数据点面板：变量树右键菜单 +「移动到」子菜单
-->
<script setup lang="ts">
import { useI18n } from 'vue-i18n'

interface ContextMenuGroupLike {
  id: string
  name: string
}

interface ContextMenuNodeLike {
  type?: 'blank' | 'variable' | 'group'
}

defineProps<{
  contextMenuVisible?: boolean
  contextMenuStyle: Record<string, string>
  contextMenuNode?: ContextMenuNodeLike | null
  showMoveToMenu?: boolean
  submenuStyle: Record<string, string>
  varClipboard?: unknown
  canEditSelection?: boolean
  canDeleteSelection?: boolean
  availableGroups?: ContextMenuGroupLike[]
}>()

const emit = defineEmits<{
  (event: 'openGroupCreate'): void
  (event: 'openCreateVar'): void
  (event: 'openQuickAdd'): void
  (event: 'paste'): void
  (event: 'openEdit'): void
  (event: 'copy'): void
  (event: 'toggleMoveTo'): void
  (event: 'removeVar'): void
  (event: 'openGroupEdit'): void
  (event: 'removeGroup'): void
  (event: 'moveTo', groupId: string | null): void
}>()

const { t } = useI18n()
</script>

<template>
  <div
    v-if="contextMenuVisible"
    class="context-menu"
    :style="contextMenuStyle"
    @click.stop
    @mousedown.stop
  >
    <template v-if="contextMenuNode?.type === 'blank'">
      <div class="context-menu-item" @click="emit('openGroupCreate')">
        {{ t('datapointPanel.contextMenu.newGroup') }}
      </div>
      <div class="context-menu-item" @click="emit('openCreateVar')">
        {{ t('datapointPanel.contextMenu.newVar') }}
      </div>
      <div class="context-menu-item" @click="emit('openQuickAdd')">
        {{ t('datapointPanel.contextMenu.quickAdd') }}
      </div>
      <div
        class="context-menu-item"
        :class="{ 'is-disabled': !varClipboard }"
        @click="emit('paste')"
      >
        {{ t('datapointPanel.contextMenu.paste') }}
      </div>
    </template>
    <template v-if="contextMenuNode?.type === 'variable'">
      <div
        class="context-menu-item"
        :class="{ 'is-disabled': !canEditSelection }"
        @click="emit('openEdit')"
      >
        {{ t('datapointPanel.contextMenu.editVar') }}
      </div>
      <div class="context-menu-item" @click="emit('copy')">
        {{ t('datapointPanel.contextMenu.copy') }}
      </div>
      <div class="context-menu-item" @click="emit('toggleMoveTo')">
        {{ t('datapointPanel.contextMenu.moveTo') }}
      </div>
      <div
        class="context-menu-item context-menu-item--danger"
        :class="{ 'is-disabled': !canDeleteSelection }"
        @click="emit('removeVar')"
      >
        {{ t('datapointPanel.contextMenu.delete') }}
      </div>
    </template>
    <template v-else-if="contextMenuNode?.type === 'group'">
      <div
        class="context-menu-item"
        :class="{ 'is-disabled': !canEditSelection }"
        @click="emit('openGroupEdit')"
      >
        {{ t('datapointPanel.contextMenu.editGroup') }}
      </div>
      <div class="context-menu-item" @click="emit('openGroupCreate')">
        {{ t('datapointPanel.contextMenu.newChildGroup') }}
      </div>
      <div class="context-menu-item" @click="emit('toggleMoveTo')">
        {{ t('datapointPanel.contextMenu.moveTo') }}
      </div>
      <div
        class="context-menu-item context-menu-item--danger"
        :class="{ 'is-disabled': !canDeleteSelection }"
        @click="emit('removeGroup')"
      >
        {{ t('datapointPanel.contextMenu.deleteGroup') }}
      </div>
    </template>
  </div>

  <div
    v-if="contextMenuVisible && showMoveToMenu"
    class="context-menu context-submenu"
    :style="submenuStyle"
    @click.stop
    @mousedown.stop
  >
    <div class="context-menu-item" @click="emit('moveTo', null)">
      {{ t('datapointPanel.root') }}
    </div>
    <div
      v-for="group in availableGroups"
      :key="group.id"
      class="context-menu-item"
      @click="emit('moveTo', group.id)"
    >
      {{ group.name }}
    </div>
  </div>
</template>

<style scoped>
.context-menu {
  position: fixed;
  background: var(--designer-shell-surface);
  border: 1px solid var(--designer-border-color);
  border-radius: 8px;
  box-shadow: var(--designer-shadow-popover);
  z-index: 4000;
  min-width: 160px;
  padding: 6px 0;
}

.context-menu-item {
  padding: 10px 18px;
  cursor: pointer;
  font-size: 14px;
  color: var(--designer-text-secondary);
  white-space: nowrap;
}

.context-menu-item:hover {
  background-color: var(--designer-hover-surface);
}

.context-menu-item.is-disabled {
  color: var(--designer-text-muted);
  pointer-events: none;
}

.context-menu-item--danger {
  color: var(--designer-danger-text);
}

.context-menu-item--danger:hover {
  background-color: var(--designer-danger-surface);
}

.context-submenu {
  max-height: 300px;
  overflow-y: auto;
}
</style>
