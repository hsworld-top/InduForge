<template>
  <el-drawer
    v-model="visible"
    class="dc-drawer"
    :class="{ 'is-pinned': pinned }"
    :size="`${drawerWidth}px`"
    :with-header="false"
    destroy-on-close
    append-to-body
  >
    <div class="dc-drawer__shell">
      <header class="dc-drawer__header">
        <h2 class="dc-drawer__title">{{ title }}</h2>
        <div v-if="$slots.actions" class="dc-drawer__actions">
          <slot name="actions" />
        </div>
      </header>

      <div class="dc-drawer__body">
        <slot />
      </div>

      <button
        type="button"
        class="dc-drawer__resize-handle"
        aria-label="拖拽调整抽屉宽度"
        @mousedown.prevent="startResize"
      ></button>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    width?: number;
    max?: number;
    pinned?: boolean;
    title: string;
  }>(),
  {
    width: 480,
    max: 720,
    pinned: false,
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
  (event: "resize", value: number): void;
}>();

const minWidth = 320;
const drawerWidth = ref(clampWidth(props.width));

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

function clampWidth(value: number) {
  const maxWidth = Math.max(minWidth, props.max);
  return Math.min(Math.max(value || props.width, minWidth), maxWidth);
}

watch(
  () => [props.width, props.max],
  () => {
    drawerWidth.value = clampWidth(props.width);
  },
);

function handleResize(event: MouseEvent) {
  const nextWidth = window.innerWidth - event.clientX;
  drawerWidth.value = clampWidth(nextWidth);
  emit("resize", drawerWidth.value);
}

function stopResize() {
  window.removeEventListener("mousemove", handleResize);
  window.removeEventListener("mouseup", stopResize);
  document.body.classList.remove("dc-drawer-resizing");
}

function startResize() {
  window.addEventListener("mousemove", handleResize);
  window.addEventListener("mouseup", stopResize);
  document.body.classList.add("dc-drawer-resizing");
}

onBeforeUnmount(() => {
  stopResize();
});
</script>

<style scoped>
.dc-drawer__shell {
  position: relative;
  height: 100%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
  color: var(--dc-text);
}

.dc-drawer__header {
  min-height: 56px;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 16px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.dc-drawer__title {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dc-drawer__actions {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.dc-drawer__body {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 16px;
}

.dc-drawer__resize-handle {
  position: absolute;
  top: 0;
  left: -3px;
  width: 6px;
  height: 100%;
  border: 0;
  background: transparent;
  cursor: ew-resize;
}

.dc-drawer__resize-handle:hover,
.dc-drawer__resize-handle:focus-visible {
  outline: none;
  background: rgba(29, 78, 216, 0.12);
}

:global(.dc-drawer.el-drawer.rtl) {
  border-left: 1px solid var(--dc-border);
  box-shadow: var(--dc-shadow-popover);
}

:global(.dc-drawer.el-drawer.is-pinned) {
  box-shadow: none;
}

:global(.dc-drawer-resizing) {
  cursor: ew-resize;
  user-select: none;
}
</style>
