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
const groupProps = computed(() => {
  const { options: _options, buttonStyle: _buttonStyle, ...rest } = props.resolvedProps || {};
  return rest;
});
const buttonStyle = computed(() => props.resolvedProps?.buttonStyle === true);

function handleModelValueUpdate(value: unknown) {
  emit("update:modelValue", value);
}
</script>

<template>
  <el-checkbox-group
    v-bind="groupProps"
    class="core-choice-group"
    @update:model-value="handleModelValueUpdate"
  >
    <component
      :is="buttonStyle ? 'el-checkbox-button' : 'el-checkbox'"
      v-for="option in options"
      :key="String(option.value)"
      :label="option.value"
      :disabled="option.disabled"
    >
      {{ option.label }}
    </component>
  </el-checkbox-group>
</template>

<style scoped>
.core-choice-group {
  display: inline-flex;
  max-width: 100%;
  flex-wrap: wrap;
  gap: 4px 10px;
}
</style>
