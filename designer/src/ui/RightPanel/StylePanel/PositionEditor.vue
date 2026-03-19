<!--
  PositionEditor - 定位编辑器
  编辑 position、left/top/right/bottom
-->
<template>
  <div class="position-editor">
    <div class="editor-group-title">定位</div>
    <div class="form-grid">
      <div class="form-item full-width">
        <label>定位方式</label>
        <el-select
          :model-value="position"
          size="small"
          @update:modelValue="handlePositionChange"
        >
          <el-option label="relative" value="relative" />
          <el-option label="absolute" value="absolute" />
          <el-option label="fixed" value="fixed" />
          <el-option label="sticky" value="sticky" />
        </el-select>
      </div>

      <template v-if="showCoordinates">
        <div class="form-item">
          <label>Left</label>
          <el-input-number
            :model-value="left"
            size="small"
            :step="1"
            controls-position="right"
            @update:modelValue="handleLeftChange"
          />
        </div>
        <div class="form-item">
          <label>Top</label>
          <el-input-number
            :model-value="top"
            size="small"
            :step="1"
            controls-position="right"
            @update:modelValue="handleTopChange"
          />
        </div>
        <div class="form-item">
          <label>Right</label>
          <el-input-number
            :model-value="right"
            size="small"
            :step="1"
            controls-position="right"
            @update:modelValue="handleRightChange"
          />
        </div>
        <div class="form-item">
          <label>Bottom</label>
          <el-input-number
            :model-value="bottom"
            size="small"
            :step="1"
            controls-position="right"
            @update:modelValue="handleBottomChange"
          />
        </div>
      </template>

      <div class="form-item">
        <label>z-index</label>
        <el-input-number
          :model-value="zIndex"
          size="small"
          :step="1"
          controls-position="right"
          @update:modelValue="handleZIndexChange"
        />
      </div>

      <div class="form-item">
        <label>overflow</label>
        <el-select
          :model-value="overflow"
          size="small"
          @update:modelValue="handleOverflowChange"
        >
          <el-option label="visible" value="visible" />
          <el-option label="hidden" value="hidden" />
          <el-option label="scroll" value="scroll" />
          <el-option label="auto" value="auto" />
        </el-select>
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

const parseValue = (value) => {
  if (value === undefined || value === null || value === "") return undefined;
  const num = parseInt(String(value), 10);
  return isNaN(num) ? undefined : num;
};

const position = computed(() => props.modelValue.position || "relative");
const showCoordinates = computed(() =>
  ["absolute", "fixed"].includes(position.value),
);
const left = computed(() => parseValue(props.modelValue.left));
const top = computed(() => parseValue(props.modelValue.top));
const right = computed(() => parseValue(props.modelValue.right));
const bottom = computed(() => parseValue(props.modelValue.bottom));
const zIndex = computed(() => parseValue(props.modelValue.zIndex) || 0);
const overflow = computed(() => props.modelValue.overflow || "visible");

const handlePositionChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, position: value });
};

const handleLeftChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, left: value });
};

const handleTopChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, top: value });
};

const handleRightChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, right: value });
};

const handleBottomChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, bottom: value });
};

const handleZIndexChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, zIndex: value });
};

const handleOverflowChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, overflow: value });
};
</script>

<style scoped>
.position-editor {
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

.form-item.full-width {
  grid-column: 1 / -1;
}

.form-item label {
  font-size: 12px;
  color: var(--el-text-color-regular);
}
</style>
