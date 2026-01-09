<template>
  <div class="border-editor">
    <div class="editor-group-title">边框</div>
    <div class="form-grid">
      <div class="form-item">
        <label>宽度</label>
        <el-input-number
          :model-value="borderWidth"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:model-value="handleWidthChange"
        />
      </div>
      <div class="form-item">
        <label>样式</label>
        <el-select
          :model-value="borderStyle"
          size="small"
          @update:model-value="handleStyleChange"
        >
          <el-option label="无" value="none" />
          <el-option label="实线" value="solid" />
          <el-option label="虚线" value="dashed" />
          <el-option label="点线" value="dotted" />
        </el-select>
      </div>
      <div class="form-item">
        <label>颜色</label>
        <el-color-picker
          :model-value="borderColor"
          size="small"
          @update:model-value="handleColorChange"
        />
      </div>
      <div class="form-item">
        <label>圆角</label>
        <el-input-number
          :model-value="borderRadius"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:model-value="handleRadiusChange"
        />
      </div>
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

const parseValue = (value, defaultValue = 0) => {
  if (!value) return defaultValue;
  const num = parseInt(String(value), 10);
  return isNaN(num) ? defaultValue : num;
};

const borderWidth = computed(() => parseValue(props.modelValue.borderWidth, 0));
const borderStyle = computed(() => props.modelValue.borderStyle || "none");
const borderColor = computed(() => props.modelValue.borderColor || "#000000");
const borderRadius = computed(() => parseValue(props.modelValue.borderRadius, 0));

const handleWidthChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, borderWidth: value });
};

const handleStyleChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, borderStyle: value });
};

const handleColorChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, borderColor: value });
};

const handleRadiusChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, borderRadius: value });
};
</script>

<style scoped>
.border-editor {
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

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
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
