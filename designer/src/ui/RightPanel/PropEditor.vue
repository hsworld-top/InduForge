<template>
  <div class="prop-editor">
    <MonacoEditor
      v-if="isCodeEditor"
      v-model="codeDraft"
      :language="prop.language || 'javascript'"
      :height="prop.height || '220px'"
      @update:model-value="handleCodeChange"
    />

    <!-- 字符串类型 -->
    <el-input
      v-else-if="prop.type === 'string'"
      :model-value="modelValue"
      :placeholder="prop.placeholder || '请输入'"
      size="small"
      @update:model-value="handleChange"
    />

    <!-- 数字类型 -->
    <el-input-number
      v-else-if="prop.type === 'number'"
      :model-value="modelValue"
      :min="prop.min"
      :max="prop.max"
      :step="prop.step || 1"
      size="small"
      controls-position="right"
      @update:model-value="handleChange"
    />

    <!-- 布尔类型 -->
    <el-switch
      v-else-if="prop.type === 'boolean'"
      :model-value="modelValue"
      size="small"
      @update:model-value="handleChange"
    />

    <!-- 颜色类型 -->
    <el-color-picker
      v-else-if="prop.type === 'color'"
      :model-value="modelValue"
      size="small"
      show-alpha
      @update:model-value="handleChange"
    />

    <!-- 枚举类型 -->
    <el-select
      v-else-if="prop.type === 'enum'"
      :model-value="modelValue"
      size="small"
      @update:model-value="handleChange"
    >
      <el-option
        v-for="opt in prop.options"
        :key="opt.value"
        :label="opt.label"
        :value="opt.value"
      />
    </el-select>

    <!-- JSON -->
    <el-input
      v-else-if="isJsonType"
      :model-value="jsonDraft"
      type="textarea"
      :rows="3"
      :placeholder="prop.placeholder || '请输入 JSON'"
      size="small"
      @update:model-value="handleJsonInput"
      @change="commitJsonDraft"
    />

    <!-- 默认：文本输入 -->
    <el-input
      v-else
      :model-value="String(modelValue ?? '')"
      size="small"
      @update:model-value="handleChange"
    />
  </div>
</template>

<script setup>
/**
 * 通用属性编辑器
 * 根据属性类型渲染对应的 Element Plus 组件
 */

import { computed, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import MonacoEditor from "@/components/common/MonacoEditor.vue";

defineOptions({ name: "PropEditor" });

const props = defineProps({
  /** 属性定义 */
  prop: {
    type: Object,
    required: true,
  },
  /** 当前值 */
  modelValue: {
    type: [String, Number, Boolean, Object, Array],
    default: undefined,
  },
});

const emit = defineEmits(["update:modelValue"]);

const isCodeEditor = computed(() => props.prop?.editor === "code");
const isJsonType = computed(() =>
  ["object", "array"].includes(props.prop?.type)
);
const jsonDraft = ref("");
const codeDraft = ref("");

/**
 * 同步 JSON 草稿
 */
const syncJsonDraft = () => {
  if (!isJsonType.value || isCodeEditor.value) return;
  if (props.modelValue === undefined || props.modelValue === null) {
    jsonDraft.value = "";
    return;
  }
  try {
    jsonDraft.value = JSON.stringify(props.modelValue);
  } catch (error) {
    jsonDraft.value = String(props.modelValue);
  }
};

/**
 * 同步代码草稿
 */
const syncCodeDraft = () => {
  if (!isCodeEditor.value) return;
  if (props.modelValue === undefined || props.modelValue === null) {
    codeDraft.value = "";
    return;
  }
  if (typeof props.modelValue === "string") {
    codeDraft.value = props.modelValue;
    return;
  }
  try {
    codeDraft.value = JSON.stringify(props.modelValue, null, 2);
  } catch (error) {
    codeDraft.value = String(props.modelValue);
  }
};

watch([() => props.modelValue, () => props.prop?.type], syncJsonDraft, {
  immediate: true,
});

watch([() => props.modelValue, () => props.prop?.editor], syncCodeDraft, {
  immediate: true,
});

/**
 * 处理值变更
 * @param {any} value - 新值
 */
const handleChange = (value) => {
  emit("update:modelValue", value);
};

/**
 * 处理代码输入
 * @param {string} value - 新值
 */
const handleCodeChange = (value) => {
  codeDraft.value = value;
  emit("update:modelValue", value);
};

/**
 * 处理 JSON 输入
 * @param {string} value - 新值
 */
const handleJsonInput = (value) => {
  jsonDraft.value = value;
};

/**
 * 提交 JSON 草稿
 */
const commitJsonDraft = () => {
  const trimmed = String(jsonDraft.value ?? "").trim();
  if (!trimmed) {
    emit("update:modelValue", undefined);
    return;
  }
  try {
    emit("update:modelValue", JSON.parse(trimmed));
  } catch (error) {
    ElMessage.warning("请输入合法的 JSON");
  }
};
</script>

<style scoped>
.prop-editor {
  width: 100%;
}

.prop-editor :deep(.el-input-number) {
  width: 100%;
}

.prop-editor :deep(.el-select) {
  width: 100%;
}

.prop-editor :deep(.monaco-editor-container) {
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  overflow: hidden;
}
</style>
