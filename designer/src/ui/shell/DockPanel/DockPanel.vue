<!--
  DockPanel - 停靠面板
  左右侧可停靠/浮动的面板，含标题、操作槽、固定/关闭按钮
-->
<template>
  <aside
    class="dock-panel"
    :class="[`dock-panel--${side}`, { 'is-floating': floating }]"
  >
    <div class="dock-panel__header">
      <span class="dock-panel__title">{{ title }}</span>
      <div class="dock-panel__actions">
        <slot name="actions" />
        <el-tooltip :content="floating ? '固定' : '悬浮'">
          <el-button class="dock-panel__action-btn" @click="handleToggle">
            <IconLucidePinOff v-if="floating" />
            <IconLucidePin v-else />
          </el-button>
        </el-tooltip>
        <el-tooltip content="关闭">
          <el-button class="dock-panel__action-btn" @click="handleClose">
            <IconLucideX />
          </el-button>
        </el-tooltip>
      </div>
    </div>
    <div class="dock-panel__body">
      <slot />
    </div>
  </aside>
</template>

<script setup>
import IconLucidePin from "~icons/lucide/pin";
import IconLucidePinOff from "~icons/lucide/pin-off";
import IconLucideX from "~icons/lucide/x";

const props = defineProps({
  side: {
    type: String,
    default: "left",
  },
  title: {
    type: String,
    default: "",
  },
  floating: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["close", "toggleFloating"]);

/**
 * 关闭面板
 */
const handleClose = () => {
  emit("close");
};

/**
 * 切换固定/悬浮
 */
const handleToggle = () => {
  emit("toggleFloating");
};
</script>

<style scoped>
.dock-panel__action-btn {
  width: var(--designer-panel-control-height);
  height: var(--designer-panel-control-height);
  padding: 0;
  border-radius: var(--designer-radius-sm);
  border: none;
  background: transparent;
  color: var(--designer-text-primary);
  box-shadow: none;
}

.dock-panel__action-btn:hover {
  background: var(--designer-hover-surface);
  color: var(--designer-text-primary);
}

:deep(.dock-panel__action-btn svg) {
  width: var(--designer-panel-icon);
  height: var(--designer-panel-icon);
}
</style>
