<script setup lang="ts">
/**
 * 通用属性编辑器
 * 根据属性类型渲染对应的 Element Plus 组件
 */

import { ElMessage } from "element-plus";
import { computed, ref, watch } from "vue";
import FriendlyColorPicker from "@/components/common/FriendlyColorPicker.vue";
import MonacoEditor from "@/components/common/monaco-editor-async";

interface PropEditorOptionLike {
  label: string;
  value: string | number;
}

interface PropEditorPropLike {
  editor?: string;
  type?: string;
  language?: string;
  height?: string;
  placeholder?: string;
  min?: number;
  max?: number;
  step?: number;
  options?: PropEditorOptionLike[];
}

defineOptions({ name: "PropEditor" });

const props = defineProps<{
  /** 属性定义 */
  prop: PropEditorPropLike;
  /** 当前值 */
  modelValue?: string | number | boolean | Record<string, unknown> | unknown[] | null;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", value: unknown): void;
}>();

const modelProxy = computed({
  get: () => props.modelValue,
  set: (value: unknown) => {
    emit("update:modelValue", value);
  },
});

const colorProxy = computed<string>({
  get: () => (typeof props.modelValue === "string" ? props.modelValue : ""),
  set: (value: string) => {
    emit("update:modelValue", value);
  },
});

const isCodeEditor = computed(() => props.prop?.editor === "code");
const isJsonType = computed(() => ["object", "array"].includes(props.prop?.type ?? ""));
const jsonDraft = ref<string>("");
const codeDraft = ref<string>("");

/**
 * 同步 JSON 草稿
 */
function syncJsonDraft() {
  if (!isJsonType.value || isCodeEditor.value) return;
  if (props.modelValue === undefined || props.modelValue === null) {
    jsonDraft.value = "";
    return;
  }
  try {
    jsonDraft.value = JSON.stringify(props.modelValue);
  } catch {
    jsonDraft.value = String(props.modelValue);
  }
}

/**
 * 同步代码草稿
 */
function syncCodeDraft() {
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
  } catch {
    codeDraft.value = String(props.modelValue);
  }
}

watch([() => props.modelValue, () => props.prop?.type], syncJsonDraft, {
  immediate: true,
});

watch([() => props.modelValue, () => props.prop?.editor], syncCodeDraft, {
  immediate: true,
});

/**
 * 处理代码输入
 * @param {string} value - 新值
 */
function handleCodeChange(value: string) {
  codeDraft.value = value;
  emit("update:modelValue", value);
}

/**
 * 处理 JSON 输入
 * @param {string} value - 新值
 */
function handleJsonInput(value: string) {
  jsonDraft.value = value;
}

/**
 * 提交 JSON 草稿
 */
function commitJsonDraft() {
  const trimmed = String(jsonDraft.value ?? "").trim();
  if (!trimmed) {
    emit("update:modelValue", undefined);
    return;
  }
  try {
    emit("update:modelValue", JSON.parse(trimmed));
  } catch {
    ElMessage.warning("请输入合法的 JSON" as never);
  }
}
</script>

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
      v-model="modelProxy"
      :placeholder="prop.placeholder || '请输入'"
      size="small"
    />

    <!-- 数字类型 -->
    <el-input-number
      v-else-if="prop.type === 'number'"
      v-model="modelProxy"
      :min="prop.min"
      :max="prop.max"
      :step="prop.step || 1"
      size="small"
      controls-position="right"
    />

    <!-- 布尔类型 -->
    <el-switch v-else-if="prop.type === 'boolean'" v-model="modelProxy" size="small" />

    <!-- 颜色类型 -->
    <FriendlyColorPicker
      v-else-if="prop.type === 'color'"
      v-model="colorProxy"
      :show-alpha="true"
    />

    <!-- 枚举类型 -->
    <el-select v-else-if="prop.type === 'enum'" v-model="modelProxy" size="small">
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
      @input="handleJsonInput"
      @change="commitJsonDraft"
    />

    <!-- 默认：文本输入 -->
    <el-input v-else v-model="modelProxy" size="small" />
  </div>
</template>

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
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  overflow: hidden;
}
</style>
