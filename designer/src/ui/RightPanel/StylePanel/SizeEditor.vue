<template>
  <div class="size-editor">
    <div class="editor-group-title">尺寸</div>
    <div class="form-grid">
      <div class="form-item">
        <label>宽度</label>
        <el-input
          :model-value="width"
          size="small"
          placeholder="auto"
          @update:model-value="handleWidthChange"
        >
          <template #append>
            <el-select
              :model-value="widthUnit"
              size="small"
              style="width: 60px"
              @update:model-value="handleWidthUnitChange"
            >
              <el-option label="px" value="px" />
              <el-option label="%" value="%" />
              <el-option label="auto" value="auto" />
            </el-select>
          </template>
        </el-input>
      </div>
      <div class="form-item">
        <label>高度</label>
        <el-input
          :model-value="height"
          size="small"
          placeholder="auto"
          @update:model-value="handleHeightChange"
        >
          <template #append>
            <el-select
              :model-value="heightUnit"
              size="small"
              style="width: 60px"
              @update:model-value="handleHeightUnitChange"
            >
              <el-option label="px" value="px" />
              <el-option label="%" value="%" />
              <el-option label="auto" value="auto" />
            </el-select>
          </template>
        </el-input>
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

/**
 * 解析尺寸值和单位
 * @param {string | number | undefined} value - 尺寸值
 * @returns {{ value: string, unit: string }}
 */
const parseSize = (value) => {
  if (!value || value === "auto") {
    return { value: "", unit: "auto" };
  }
  const str = String(value);
  const match = str.match(/^([\d.]+)(px|%)?$/);
  if (match) {
    return { value: match[1], unit: match[2] || "px" };
  }
  return { value: "", unit: "auto" };
};

const width = computed(() => parseSize(props.modelValue.width).value);
const widthUnit = computed(() => parseSize(props.modelValue.width).unit);
const height = computed(() => parseSize(props.modelValue.height).value);
const heightUnit = computed(() => parseSize(props.modelValue.height).unit);

const handleWidthChange = (value) => {
  const unit = widthUnit.value === "auto" ? "px" : widthUnit.value;
  const newWidth = value ? `${value}${unit}` : "auto";
  emit("update:modelValue", { ...props.modelValue, width: newWidth });
};

const handleWidthUnitChange = (unit) => {
  if (unit === "auto") {
    emit("update:modelValue", { ...props.modelValue, width: "auto" });
  } else {
    const value = width.value || "100";
    emit("update:modelValue", { ...props.modelValue, width: `${value}${unit}` });
  }
};

const handleHeightChange = (value) => {
  const unit = heightUnit.value === "auto" ? "px" : heightUnit.value;
  const newHeight = value ? `${value}${unit}` : "auto";
  emit("update:modelValue", { ...props.modelValue, height: newHeight });
};

const handleHeightUnitChange = (unit) => {
  if (unit === "auto") {
    emit("update:modelValue", { ...props.modelValue, height: "auto" });
  } else {
    const value = height.value || "100";
    emit("update:modelValue", { ...props.modelValue, height: `${value}${unit}` });
  }
};
</script>

<style scoped>
.size-editor {
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
