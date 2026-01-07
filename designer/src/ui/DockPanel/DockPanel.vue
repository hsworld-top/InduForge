<template>
  <aside
    class="dock-panel"
    :class="[`dock-panel--${side}`, { 'is-floating': floating }]"
  >
    <div class="dock-panel__header">
      <span class="dock-panel__title">{{ title }}</span>
      <div class="dock-panel__actions">
        <el-button size="small" text @click="handleToggle">
          {{ floating ? "固定" : "悬浮" }}
        </el-button>
        <el-button size="small" text @click="handleClose">关闭</el-button>
      </div>
    </div>
    <div class="dock-panel__body">
      <slot />
    </div>
  </aside>
</template>

<script setup>
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
