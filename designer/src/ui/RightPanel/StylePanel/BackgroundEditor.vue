<template>
  <div class="background-editor">
    <div class="editor-group-title">背景</div>
    <div class="form-item">
      <label>背景颜色</label>
      <el-color-picker
        :model-value="backgroundColor"
        size="small"
        show-alpha
        @update:modelValue="handleColorChange"
      />
    </div>
  </div>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({}),
  },
});

const emit = defineEmits(["update:modelValue"]);

const backgroundColor = computed(() => props.modelValue.background || "");

const handleColorChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, background: value });
};
</script>

<style scoped>
.background-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.editor-group-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  padding-bottom: 4px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-item label {
  font-size: 12px;
  color: var(--el-text-color-regular);
}
</style>
