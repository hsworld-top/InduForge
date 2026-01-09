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
          <el-button size="small" text @click="handleToggle">
            <svg
              class="dock-panel__pin-icon"
              :class="{ 'is-floating': floating }"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <rect x="6" y="3" width="12" height="4" rx="1" />
              <rect x="11" y="7" width="2" height="8" />
              <polygon points="12,15 8,21 16,21" />
            </svg>
          </el-button>
        </el-tooltip>
        <el-tooltip content="关闭">
          <el-button size="small" text @click="handleClose">
            <IconEpClose />
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
import IconEpClose from "~icons/ep/close";

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
.dock-panel__pin-icon {
  width: 16px;
  height: 16px;
  fill: currentColor;
  transform: rotate(-28deg);
}

.dock-panel__pin-icon.is-floating {
  opacity: 0.6;
}
</style>
