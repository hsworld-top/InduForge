<!--
  BorderEditor - 边框样式编辑器
  编辑边框宽度、样式、颜色、圆角
-->
<script setup lang="ts">
import { computed } from "vue";
import FriendlyColorPicker from "@/components/common/FriendlyColorPicker.vue";

interface BorderStyleModel {
  borderWidth?: number | string;
  borderStyle?: string;
  borderColor?: string;
  borderRadius?: number | string;
}

const props = withDefaults(defineProps<{ modelValue?: BorderStyleModel }>(), {
  modelValue: () => ({}),
});

const emit = defineEmits<{
  (event: "update:modelValue", value: BorderStyleModel): void;
}>();

function parseValue(value: unknown, defaultValue = 0): number {
  if (!value) return defaultValue;
  const num = Number.parseInt(String(value), 10);
  return Number.isNaN(num) ? defaultValue : num;
}

const borderWidth = computed(() => parseValue(props.modelValue.borderWidth, 0));
const borderStyle = computed(() => props.modelValue.borderStyle || "none");
const borderColor = computed(() => props.modelValue.borderColor || "#000000");
const borderRadius = computed(() => parseValue(props.modelValue.borderRadius, 0));

function handleWidthChange(value: number) {
  emit("update:modelValue", { ...props.modelValue, borderWidth: value });
}

function handleStyleChange(value: string) {
  emit("update:modelValue", { ...props.modelValue, borderStyle: value });
}

function handleColorChange(value: string) {
  emit("update:modelValue", { ...props.modelValue, borderColor: value });
}

function handleRadiusChange(value: number) {
  emit("update:modelValue", { ...props.modelValue, borderRadius: value });
}
</script>

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
        <el-select :model-value="borderStyle" size="small" @update:model-value="handleStyleChange">
          <el-option label="无" value="none" />
          <el-option label="实线" value="solid" />
          <el-option label="虚线" value="dashed" />
          <el-option label="点线" value="dotted" />
        </el-select>
      </div>
      <div class="form-item">
        <label>颜色</label>
        <FriendlyColorPicker :model-value="borderColor" @update:model-value="handleColorChange" />
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
