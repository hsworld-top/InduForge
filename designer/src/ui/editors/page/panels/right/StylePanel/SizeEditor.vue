<!--
  SizeEditor - 尺寸编辑器
  编辑 width/height，支持 px/%/auto 单位
-->
<template>
  <div class="size-editor">
    <div class="editor-group-title">尺寸</div>
    <div class="size-row">
      <div class="size-label">宽度</div>
      <div class="size-control">
        <el-input
          :model-value="widthInput"
          size="small"
          placeholder="auto"
          class="size-input-with-unit"
          @update:modelValue="handleWidthInput"
          @change="handleWidthChange"
          @blur="handleWidthChange"
        >
          <template #append>
            <el-select
              :model-value="widthUnit"
              size="small"
              class="size-unit-select"
              teleported
              popper-class="size-editor-unit-popper"
              @update:modelValue="handleWidthUnitChange"
            >
              <el-option label="px" value="px" />
              <el-option label="%" value="%" />
              <el-option label="auto" value="auto" />
            </el-select>
          </template>
        </el-input>
      </div>
    </div>
    <div class="size-row">
      <div class="size-label">高度</div>
      <div class="size-control">
        <el-input
          :model-value="heightInput"
          size="small"
          placeholder="auto"
          class="size-input-with-unit"
          @update:modelValue="handleHeightInput"
          @change="handleHeightChange"
          @blur="handleHeightChange"
        >
          <template #append>
            <el-select
              :model-value="heightUnit"
              size="small"
              class="size-unit-select"
              teleported
              popper-class="size-editor-unit-popper"
              @update:modelValue="handleHeightUnitChange"
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
  { immediate: true },
);

watch(
  () => props.modelValue.height,
  (val) => {
    heightInput.value = parseSize(val).value;
  },
  { immediate: true },
);

const handleWidthInput = (value) => {
  widthInput.value = value;
};

const handleHeightInput = (value) => {
  heightInput.value = value;
};

const handleWidthChange = () => {
  const unit = widthUnit.value === "auto" ? "px" : widthUnit.value;
  const newWidth = buildSizeValue(
    widthInput.value,
    unit,
    props.minWidth,
    false,
  );
  emit("update:modelValue", { ...props.modelValue, width: newWidth });
};

const handleWidthUnitChange = (unit) => {
  if (unit === "auto") {
    emit("update:modelValue", { ...props.modelValue, width: "auto" });
  } else {
    const newWidth = buildSizeValue(
      widthInput.value,
      unit,
      props.minWidth,
      true,
    );
    emit("update:modelValue", { ...props.modelValue, width: newWidth });
  }
};

const handleHeightChange = () => {
  const unit = heightUnit.value === "auto" ? "px" : heightUnit.value;
  const newHeight = buildSizeValue(
    heightInput.value,
    unit,
    props.minHeight,
    false,
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
      true,
    );
    emit("update:modelValue", { ...props.modelValue, height: newHeight });
  }
};
</script>

<style scoped>
.size-editor {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}

.editor-group-title {
  display: flex;
  align-items: center;
  min-height: 34px;
  padding: 0 10px;
  margin: calc(var(--designer-gap-md) * -1) calc(var(--designer-gap-md) * -1)
    0;
  background: var(--designer-group-surface);
  border-bottom: 1px solid var(--designer-border-soft);
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-sm);
  font-weight: 600;
  letter-spacing: 0.02em;
}

.size-row {
  display: flex;
  align-items: center;
  gap: var(--designer-gap-sm);
  min-height: 30px;
  padding: 2px 6px;
  border-radius: var(--designer-radius-sm);
  transition: background-color 0.15s ease;
}

.size-row:hover {
  background: var(--designer-hover-surface);
}

.size-label {
  width: 88px;
  min-width: 72px;
  max-width: 88px;
  font-size: var(--designer-font-sm);
  color: var(--designer-text-regular);
}

.size-control {
  flex: 1;
  min-width: 0;
}

.size-control :deep(.el-input) {
  width: 100%;
}

/* 保证 append 区域和单位下拉可见，不被 flex 挤没 */
.size-control :deep(.size-input-with-unit .el-input-group__append) {
  padding: 0;
  min-width: 64px;
  flex-shrink: 0;
}

.size-unit-select {
  width: 64px;
  min-width: 64px;
}
</style>

<!-- 下拉层挂到 body，需单独设 z-index，否则可能被右侧栏遮挡 -->
<style>
.size-editor-unit-popper {
  z-index: 4000 !important;
}
</style>
