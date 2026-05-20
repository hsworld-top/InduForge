<script setup lang="ts">
const props = defineProps<{
  resolvedProps?: Record<string, unknown>;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
}>();

function handleModelValueUpdate(value: unknown): void {
  emit("update:modelValue", Boolean(value));
}

function handleToggle(): void {
  if (props.resolvedProps?.disabled === true) return;
  emit("update:modelValue", !Boolean(props.resolvedProps?.modelValue));
}
</script>

<template>
  <el-switch
    v-bind="resolvedProps"
    class="core-switch"
    @update:model-value="handleModelValueUpdate"
  />
  <button
    class="core-switch__hit"
    type="button"
    aria-label="切换开关"
    @click.stop="handleToggle"
  />
</template>

<style scoped>
.core-switch__hit {
  position: absolute;
  inset: 0;
  z-index: 2;
  border: 0;
  background: transparent;
  cursor: pointer;
}
</style>
