<!--
  DockPanel - 停靠面板
  左右侧可停靠/浮动的面板，含标题、操作槽、固定/关闭按钮
-->
<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue";
import IconLucidePin from "~icons/lucide/pin";
import IconLucidePinOff from "~icons/lucide/pin-off";
import IconLucideX from "~icons/lucide/x";

const props = withDefaults(
  defineProps<{
    side?: "left" | "right";
    title?: string;
    floating?: boolean;
    width?: number;
    minWidth?: number;
    maxWidth?: number;
    resizable?: boolean;
  }>(),
  {
    side: "left",
    title: "",
    floating: false,
    width: 272,
    minWidth: 220,
    maxWidth: 520,
    resizable: true,
  },
);

const emit = defineEmits<{
  close: [];
  toggleFloating: [];
  resize: [width: number];
}>();

const resizing = ref(false);
const panelStyle = computed<Record<string, string>>(() => ({
  width: `${props.width}px`,
}));
const showResizeHandle = computed(() => props.resizable);

let pointerMoveHandler: ((event: PointerEvent) => void) | null = null;
let pointerUpHandler: ((event: PointerEvent) => void) | null = null;
let startX = 0;
let startWidth = 0;

function handleClose() {
  emit("close");
}

function handleToggle() {
  emit("toggleFloating");
}

/**
 * 约束面板宽度，避免拖拽过窄或过宽。
 * @param {number} width - 目标宽度
 * @returns {number}
 */
function clampWidth(width: number): number {
  return Math.min(props.maxWidth, Math.max(props.minWidth, Math.round(width)));
}

/**
 * 结束拖拽并清理全局事件。
 */
function stopResize() {
  if (!resizing.value) return;
  resizing.value = false;
  document.body.classList.remove("dock-panel-resizing");
  if (pointerMoveHandler) {
    window.removeEventListener("pointermove", pointerMoveHandler);
    pointerMoveHandler = null;
  }
  if (pointerUpHandler) {
    window.removeEventListener("pointerup", pointerUpHandler);
    window.removeEventListener("pointercancel", pointerUpHandler);
    pointerUpHandler = null;
  }
}

/**
 * 开始拖拽调整面板宽度。
 * @param {PointerEvent} event - 指针事件
 */
function handleResizePointerDown(event: PointerEvent) {
  if (!showResizeHandle.value) return;
  if (event.button !== 0) return;
  event.preventDefault();
  startX = event.clientX;
  startWidth = props.width;
  resizing.value = true;
  document.body.classList.add("dock-panel-resizing");

  pointerMoveHandler = (moveEvent: PointerEvent) => {
    const delta = moveEvent.clientX - startX;
    const nextWidth = props.side === "left" ? startWidth + delta : startWidth - delta;
    emit("resize", clampWidth(nextWidth));
  };

  pointerUpHandler = () => {
    stopResize();
  };

  window.addEventListener("pointermove", pointerMoveHandler);
  window.addEventListener("pointerup", pointerUpHandler);
  window.addEventListener("pointercancel", pointerUpHandler);
}

onBeforeUnmount(() => {
  stopResize();
});
</script>

<template>
  <aside
    class="dock-panel"
    :class="[`dock-panel--${side}`, { 'is-floating': floating, 'is-resizing': resizing }]"
    :style="panelStyle"
  >
    <div
      v-if="showResizeHandle"
      class="dock-panel__resize-handle"
      :class="`dock-panel__resize-handle--${side}`"
      @pointerdown="handleResizePointerDown"
    />
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

.dock-panel:not(.is-floating) {
  position: relative;
}

.dock-panel.is-floating {
  position: absolute;
  top: 0;
  bottom: 0;
  box-shadow: var(--designer-shadow-panel);
  z-index: 20;
}

.dock-panel--left.is-floating {
  left: 0;
}

.dock-panel--right.is-floating {
  right: 0;
}

.dock-panel__resize-handle {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 6px;
  z-index: 8;
  cursor: col-resize;
  touch-action: none;
  user-select: none;
  transition: background-color 0.12s ease;
}

.dock-panel__resize-handle--left {
  right: 0;
}

.dock-panel__resize-handle--right {
  left: 0;
}

.dock-panel__resize-handle:hover {
  background: rgba(var(--designer-primary-rgb, 29, 78, 216), 0.12);
}

.dock-panel.is-resizing .dock-panel__resize-handle {
  background: rgba(var(--designer-primary-rgb, 29, 78, 216), 0.18);
}
</style>
