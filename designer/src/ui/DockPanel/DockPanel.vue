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
  width: 26px;
  height: 26px;
  padding: 0;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
  background: #ffffff;
  color: #4b5563;
}

:deep(.dock-panel__action-btn svg) {
  width: 14px;
  height: 14px;
}
</style>
