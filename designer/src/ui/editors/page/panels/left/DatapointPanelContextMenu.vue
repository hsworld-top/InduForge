<!--
  数据点面板：变量树右键菜单 +「移动到」子菜单
-->
<script setup>
defineProps({
  contextMenuVisible: { type: Boolean, default: false },
  contextMenuStyle: { type: Object, required: true },
  contextMenuNode: { type: Object, default: null },
  showMoveToMenu: { type: Boolean, default: false },
  submenuStyle: { type: Object, required: true },
  varClipboard: { default: null },
  canEditSelection: { type: Boolean, default: true },
  canDeleteSelection: { type: Boolean, default: true },
  availableGroups: { type: Array, default: () => [] },
});

const emit = defineEmits([
  "openGroupCreate",
  "openCreateVar",
  "openQuickAdd",
  "paste",
  "openEdit",
  "copy",
  "toggleMoveTo",
  "removeVar",
  "openGroupEdit",
  "removeGroup",
  "moveTo",
]);
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
      <div class="context-menu-item" @click="emit('openGroupCreate')">新建分组</div>
      <div class="context-menu-item" @click="emit('openCreateVar')">新增变量</div>
      <div class="context-menu-item" @click="emit('openQuickAdd')">快速添加</div>
      <div
        class="context-menu-item"
        :class="{ 'is-disabled': !varClipboard }"
        @click="emit('paste')"
      >
        粘贴
      </div>
    </template>
    <template v-if="contextMenuNode?.type === 'variable'">
      <div
        class="context-menu-item"
        :class="{ 'is-disabled': !canEditSelection }"
        @click="emit('openEdit')"
      >
        编辑变量
      </div>
      <div class="context-menu-item" @click="emit('copy')">复制</div>
      <div class="context-menu-item" @click="emit('toggleMoveTo')">移动到</div>
      <div
        class="context-menu-item context-menu-item--danger"
        :class="{ 'is-disabled': !canDeleteSelection }"
        @click="emit('removeVar')"
      >
        删除
      </div>
    </template>
    <template v-else-if="contextMenuNode?.type === 'group'">
      <div
        class="context-menu-item"
        :class="{ 'is-disabled': !canEditSelection }"
        @click="emit('openGroupEdit')"
      >
        编辑分组
      </div>
      <div class="context-menu-item" @click="emit('openGroupCreate')">新建子分组</div>
      <div class="context-menu-item" @click="emit('toggleMoveTo')">移动到</div>
      <div
        class="context-menu-item context-menu-item--danger"
        :class="{ 'is-disabled': !canDeleteSelection }"
        @click="emit('removeGroup')"
      >
        删除分组
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
    <div class="context-menu-item" @click="emit('moveTo', null)">根目录</div>
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
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.12);
  z-index: 4000;
  min-width: 160px;
  padding: 6px 0;
}

.context-menu-item {
  padding: 10px 18px;
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  white-space: nowrap;
}

.context-menu-item:hover {
  background-color: #f5f7fa;
}

.context-menu-item.is-disabled {
  color: #c0c4cc;
  pointer-events: none;
}

.context-menu-item--danger {
  color: #f56c6c;
}

.context-menu-item--danger:hover {
  background-color: #fef0f0;
}

.context-submenu {
  max-height: 300px;
  overflow-y: auto;
}
</style>
