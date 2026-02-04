<template>
  <div class="spacing-editor">
    <div class="editor-group-title">{{ title }}</div>
    <div class="spacing-grid">
      <div class="form-item">
        <label>上</label>
        <el-input-number
          :model-value="top"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:modelValue="handleTopChange"
        />
      </div>
      <div class="form-item">
        <label>右</label>
        <el-input-number
          :model-value="right"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:modelValue="handleRightChange"
        />
      </div>
      <div class="form-item">
        <label>下</label>
        <el-input-number
          :model-value="bottom"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:modelValue="handleBottomChange"
        />
      </div>
      <div class="form-item">
        <label>左</label>
        <el-input-number
          :model-value="left"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:modelValue="handleLeftChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  title: {
    type: String,
    default: "间距",
  },
  modelValue: {
    type: Object,
    default: () => ({}),
  },
  prefix: {
    type: String,
    required: true, // 'padding' or 'margin'
  },
});

const emit = defineEmits(["update:modelValue"]);

const parseValue = (value) => {
  if (!value) return 0;
  const num = parseInt(String(value), 10);
  return isNaN(num) ? 0 : num;
};

const top = computed(() => parseValue(props.modelValue[`${props.prefix}Top`]));
const right = computed(() => parseValue(props.modelValue[`${props.prefix}Right`]));
const bottom = computed(() => parseValue(props.modelValue[`${props.prefix}Bottom`]));
const left = computed(() => parseValue(props.modelValue[`${props.prefix}Left`]));

const handleTopChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, [`${props.prefix}Top`]: value });
};

const handleRightChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, [`${props.prefix}Right`]: value });
};

const handleBottomChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, [`${props.prefix}Bottom`]: value });
};

const handleLeftChange = (value) => {
  emit("update:modelValue", { ...props.modelValue, [`${props.prefix}Left`]: value });
};
</script>

<style scoped>
.spacing-editor {
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

.spacing-grid {
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
