<!--
  SizeEditor - 尺寸编辑器
  编辑 width/height，支持 px/%/auto 单位
-->
<script setup lang="ts">
import { computed, ref, watch } from "vue";

interface SizeStyleModel {
  [key: string]: unknown;
  width?: string;
  height?: string;
}

const props = withDefaults(
  defineProps<{
    modelValue?: SizeStyleModel;
    minWidth?: number | undefined;
    minHeight?: number | undefined;
  }>(),
  {
    modelValue: () => ({}),
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: SizeStyleModel): void;
}>();

const SIZE_VALUE_RE = /^([\d.]+)(px|%)?$/;

/**
 * 解析尺寸值和单位
 * @param {string | number | undefined} value - 尺寸值
 * @returns {{ value: string, unit: string }}
 */
function parseSize(value: unknown): { value: string; unit: "auto" | "px" | "%" } {
  if (!value || value === "auto") {
    return { value: "", unit: "auto" };
  }
  const str = String(value);
  const match = str.match(SIZE_VALUE_RE);
  if (match) {
    return { value: match[1] ?? "", unit: (match[2] || "px") as "px" | "%" };
  }
  return { value: "", unit: "auto" };
}

/**
 * 限制尺寸最小值
 * @param {string} value - 当前值
 * @param {number | undefined} minValue - 最小值
 * @returns {string}
 */
function clampSizeValue(value: string, minValue: number | undefined): string {
  if (minValue === undefined || minValue === null) return value;
  if (value === "" || value === undefined || value === null) return value;
  const num = Number.parseFloat(value);
  if (!Number.isFinite(num)) return value;
  return String(Math.max(num, minValue));
}

/**
 * 获取默认尺寸值
 * @param {number | undefined} minValue - 最小值
 * @returns {string}
 */
function resolveDefaultValue(minValue: number | undefined): string {
  if (Number.isFinite(minValue)) return String(minValue);
  return "100";
}

/**
 * 生成尺寸字符串
 * @param {string} rawValue - 当前输入值
 * @param {string} unit - 单位
 * @param {number | undefined} minValue - 最小值
 * @param {boolean} useDefault - 是否使用默认值
 * @returns {string}
 */
function buildSizeValue(
  rawValue: string,
  unit: "auto" | "px" | "%",
  minValue: number | undefined,
  useDefault: boolean,
): string {
  const text = String(rawValue ?? "").trim();
  if (!text) {
    if (!useDefault) return "auto";
    const defaultValue = resolveDefaultValue(minValue);
    const nextValue = unit === "px" ? clampSizeValue(defaultValue, minValue) : defaultValue;
    return `${nextValue}${unit}`;
  }
  let nextValue = text;
  if (unit === "px") {
    nextValue = clampSizeValue(text, minValue);
  }
  return `${nextValue}${unit}`;
}

const widthUnit = computed<"auto" | "px" | "%">(() => parseSize(props.modelValue.width).unit);
const heightUnit = computed<"auto" | "px" | "%">(() => parseSize(props.modelValue.height).unit);

const widthInput = ref("");
const heightInput = ref("");

watch(
  () => props.modelValue.width,
  (val: unknown) => {
    widthInput.value = parseSize(val).value;
  },
  { immediate: true },
);

watch(
  () => props.modelValue.height,
  (val: unknown) => {
    heightInput.value = parseSize(val).value;
  },
  { immediate: true },
);

function handleWidthInput(value: string) {
  widthInput.value = value;
}

function handleHeightInput(value: string) {
  heightInput.value = value;
}

function handleWidthChange() {
  const unit = widthUnit.value === "auto" ? "px" : widthUnit.value;
  const newWidth = buildSizeValue(widthInput.value, unit, props.minWidth, false);
  emit("update:modelValue", { ...props.modelValue, width: newWidth });
}

function handleWidthUnitChange(unit: "auto" | "px" | "%") {
  if (unit === "auto") {
    emit("update:modelValue", { ...props.modelValue, width: "auto" });
  } else {
    const newWidth = buildSizeValue(widthInput.value, unit, props.minWidth, true);
    emit("update:modelValue", { ...props.modelValue, width: newWidth });
  }
}

function handleHeightChange() {
  const unit = heightUnit.value === "auto" ? "px" : heightUnit.value;
  const newHeight = buildSizeValue(heightInput.value, unit, props.minHeight, false);
  emit("update:modelValue", { ...props.modelValue, height: newHeight });
}

function handleHeightUnitChange(unit: "auto" | "px" | "%") {
  if (unit === "auto") {
    emit("update:modelValue", { ...props.modelValue, height: "auto" });
  } else {
    const newHeight = buildSizeValue(heightInput.value, unit, props.minHeight, true);
    emit("update:modelValue", { ...props.modelValue, height: newHeight });
  }
}
</script>

<template>
  <div class="size-editor">
    <div class="editor-group-title">尺寸</div>
    <div class="size-row">
      <div class="size-label">宽度</div>
      <div class="size-control">
        <el-input
          :model-value="widthInput"
          size="small"
          placeholder="数值"
          class="size-value-input"
          @update:model-value="handleWidthInput"
          @change="handleWidthChange"
          @blur="handleWidthChange"
        />
        <el-select
          :model-value="widthUnit"
          size="small"
          class="size-unit-select"
          teleported
          popper-class="size-editor-unit-popper"
          @update:model-value="handleWidthUnitChange"
        >
          <el-option label="px" value="px" />
          <el-option label="%" value="%" />
          <el-option label="auto" value="auto" />
        </el-select>
      </div>
    </div>
    <div class="size-row">
      <div class="size-label">高度</div>
      <div class="size-control">
        <el-input
          :model-value="heightInput"
          size="small"
          placeholder="数值"
          class="size-value-input"
          @update:model-value="handleHeightInput"
          @change="handleHeightChange"
          @blur="handleHeightChange"
        />
        <el-select
          :model-value="heightUnit"
          size="small"
          class="size-unit-select"
          teleported
          popper-class="size-editor-unit-popper"
          @update:model-value="handleHeightUnitChange"
        >
          <el-option label="px" value="px" />
          <el-option label="%" value="%" />
          <el-option label="auto" value="auto" />
        </el-select>
      </div>
    </div>
  </div>
</template>

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
  margin: calc(var(--designer-gap-md) * -1) calc(var(--designer-gap-md) * -1) 0;
  background: var(--designer-group-surface);
  border-bottom: 1px solid var(--designer-border-soft);
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-sm);
  font-weight: 600;
  letter-spacing: 0.02em;
}

.size-row {
  display: grid;
  grid-template-columns: minmax(44px, 56px) minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 2px 6px;
  border-radius: var(--designer-radius-sm);
  transition: background-color 0.15s ease;
}

.size-row:hover {
  background: var(--designer-hover-surface);
}

.size-label {
  font-size: var(--designer-font-sm);
  color: var(--designer-text-regular);
  white-space: nowrap;
}

.size-control {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 62px;
  gap: 6px;
  min-width: 0;
}

.size-control :deep(.el-input),
.size-control :deep(.el-select) {
  width: 100%;
}

.size-control :deep(.el-input__wrapper),
.size-control :deep(.el-select__wrapper) {
  min-height: 28px;
  border-radius: var(--designer-radius-md);
  box-shadow: inset 0 0 0 1px var(--designer-border-strong);
  background: var(--designer-group-surface);
}

.size-control :deep(.el-input__inner),
.size-control :deep(.el-select__selected-item) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--designer-font-label);
}

.size-unit-select {
  width: 62px;
  min-width: 62px;
}

.size-unit-select :deep(.el-select__wrapper) {
  padding: 0 6px;
}
</style>

<!-- 下拉层挂到 body，需单独设 z-index，否则可能被右侧栏遮挡 -->
<style>
.size-editor-unit-popper {
  z-index: 4000 !important;
  min-width: 86px !important;
}
</style>
