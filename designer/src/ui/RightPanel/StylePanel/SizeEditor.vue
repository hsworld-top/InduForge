<template>
  <div class="size-editor">
    <div class="editor-group-title">尺寸</div>
    <div class="form-grid">
      <div class="form-item">
        <label>宽度</label>
        <el-input
          :model-value="widthInput"
          size="small"
          placeholder="auto"
          @update:model-value="handleWidthInput"
          @change="handleWidthChange"
          @blur="handleWidthChange"
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
          :model-value="heightInput"
          size="small"
          placeholder="auto"
          @update:model-value="handleHeightInput"
          @change="handleHeightChange"
          @blur="handleHeightChange"
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
import { computed, ref, watch } from "vue";

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({}),
  },
  minWidth: {
    type: Number,
    default: undefined,
  },
  minHeight: {
    type: Number,
    default: undefined,
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

/**
 * 限制尺寸最小值
 * @param {string} value - 当前值
 * @param {number | undefined} minValue - 最小值
 * @returns {string}
 */
const clampSizeValue = (value, minValue) => {
  if (minValue === undefined || minValue === null) return value;
  if (value === "" || value === undefined || value === null) return value;
  const num = Number.parseFloat(value);
  if (!Number.isFinite(num)) return value;
  return String(Math.max(num, minValue));
};

/**
 * 获取默认尺寸值
 * @param {number | undefined} minValue - 最小值
 * @returns {string}
 */
const resolveDefaultValue = (minValue) => {
  if (Number.isFinite(minValue)) return String(minValue);
  return "100";
};

/**
 * 生成尺寸字符串
 * @param {string} rawValue - 当前输入值
 * @param {string} unit - 单位
 * @param {number | undefined} minValue - 最小值
 * @param {boolean} useDefault - 是否使用默认值
 * @returns {string}
 */
const buildSizeValue = (rawValue, unit, minValue, useDefault) => {
  const text = String(rawValue ?? "").trim();
  if (!text) {
    if (!useDefault) return "auto";
    const defaultValue = resolveDefaultValue(minValue);
    const nextValue =
      unit === "px" ? clampSizeValue(defaultValue, minValue) : defaultValue;
    return `${nextValue}${unit}`;
  }
  let nextValue = text;
  if (unit === "px") {
    nextValue = clampSizeValue(text, minValue);
  }
  return `${nextValue}${unit}`;
};

const widthUnit = computed(() => parseSize(props.modelValue.width).unit);
const heightUnit = computed(() => parseSize(props.modelValue.height).unit);

const widthInput = ref("");
const heightInput = ref("");

watch(
  () => props.modelValue.width,
  (val) => {
    widthInput.value = parseSize(val).value;
  },
  { immediate: true }
);

watch(
  () => props.modelValue.height,
  (val) => {
    heightInput.value = parseSize(val).value;
  },
  { immediate: true }
);

const handleWidthInput = (value) => {
  widthInput.value = value;
};

const handleHeightInput = (value) => {
  heightInput.value = value;
};

const handleWidthChange = () => {
  const unit = widthUnit.value === "auto" ? "px" : widthUnit.value;
  const newWidth = buildSizeValue(widthInput.value, unit, props.minWidth, false);
  emit("update:modelValue", { ...props.modelValue, width: newWidth });
};

const handleWidthUnitChange = (unit) => {
  if (unit === "auto") {
    emit("update:modelValue", { ...props.modelValue, width: "auto" });
  } else {
    const newWidth = buildSizeValue(widthInput.value, unit, props.minWidth, true);
    emit("update:modelValue", { ...props.modelValue, width: newWidth });
  }
};

const handleHeightChange = () => {
  const unit = heightUnit.value === "auto" ? "px" : heightUnit.value;
  const newHeight = buildSizeValue(
    heightInput.value,
    unit,
    props.minHeight,
    false
  );
  emit("update:modelValue", { ...props.modelValue, height: newHeight });
};

const handleHeightUnitChange = (unit) => {
  if (unit === "auto") {
    emit("update:modelValue", { ...props.modelValue, height: "auto" });
  } else {
    const newHeight = buildSizeValue(
      heightInput.value,
      unit,
      props.minHeight,
      true
    );
    emit("update:modelValue", { ...props.modelValue, height: newHeight });
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




