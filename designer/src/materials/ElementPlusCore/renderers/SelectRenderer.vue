<script setup lang="ts">
import { computed } from "vue";
import { normalizeOptions } from "./shared";

const props = defineProps<{
  resolvedProps?: Record<string, unknown>;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", value: unknown): void;
}>();

const options = computed(() => normalizeOptions(props.resolvedProps?.options));

function handleModelValueUpdate(value: unknown) {
  emit("update:modelValue", value);
}
</script>

<template>
  <el-select
    v-bind="resolvedProps"
    class="core-select"
    @update:model-value="handleModelValueUpdate"
  >
    <el-option
      v-for="option in options"
      :key="String(option.value)"
      :label="option.label"
      :value="option.value"
      :disabled="option.disabled"
    />
  </el-select>
</template>

<style scoped>
.core-select {
  width: 100%;
}
</style>
