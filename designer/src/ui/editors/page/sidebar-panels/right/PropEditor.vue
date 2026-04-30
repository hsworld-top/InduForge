<script setup lang="ts">
/**
 * 通用属性编辑器
 * 根据属性类型渲染对应的 Element Plus 组件
 */

import { ElMessage } from "element-plus";
import { computed, ref, watch } from "vue";
import FriendlyColorPicker from "@/ui/shared/widgets/base/FriendlyColorPicker.vue";
import MonacoEditor from "@/ui/shared/widgets/base/monaco-editor-async";

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
const isIconEditor = computed(() => props.prop?.editor === "icon");
const isJsonType = computed(() => ["object", "array"].includes(props.prop?.type ?? ""));
const jsonDraft = ref<string>("");
const codeDraft = ref<string>("");
const iconDraft = ref<string>("");

const iconOptions = computed(() =>
  (props.prop?.options || []).map((option) => ({
    label: String(option.label || option.value || ""),
    value: String(option.value || ""),
  })),
);

const iconProxy = computed<string>({
  get: () => iconDraft.value,
  set: (value: string) => {
    iconDraft.value = value;
    const matchedOption = findIconOptionByLabel(value);
    emit("update:modelValue", matchedOption?.value ?? value);
  },
});

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

watch([() => props.modelValue, () => props.prop?.options, isIconEditor], syncIconDraft, {
  immediate: true,
  deep: true,
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

function findIconOptionByValue(value: string) {
  return iconOptions.value.find((option) => option.value === value);
}

function findIconOptionByLabel(label: string) {
  return iconOptions.value.find((option) => option.label === label);
}

function syncIconDraft() {
  if (!isIconEditor.value) return;
  const value = typeof props.modelValue === "string" ? props.modelValue : "";
  iconDraft.value = findIconOptionByValue(value)?.label ?? value;
}

/**
 * 按当前语言下的图标名称过滤候选项，同时保留手动输入能力。
 */
function queryIconSuggestions(query: string, callback: (items: Array<{ label: string; value: string }>) => void) {
  const keyword = String(query || "").trim().toLowerCase();
  const matched = iconOptions.value.filter((option) => {
    if (!keyword) return true;
    return option.label.toLowerCase().includes(keyword);
  });
  callback(matched);
}

function handleIconSelect(item: { label?: string; value?: string }) {
  iconDraft.value = String(item?.label || item?.value || "");
  emit("update:modelValue", String(item?.value || ""));
}

function handleIconClear() {
  iconDraft.value = "";
  emit("update:modelValue", "");
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

    <!-- 图标类型：常用图标可搜索选择，同时允许手动输入自定义图标名 -->
    <el-autocomplete
      v-else-if="isIconEditor"
      v-model="iconProxy"
      :fetch-suggestions="queryIconSuggestions"
      :placeholder="prop.placeholder || '选择或输入图标名'"
      value-key="label"
      size="small"
      clearable
      :trigger-on-focus="true"
      @select="handleIconSelect"
      @clear="handleIconClear"
    >
      <template #default="{ item }">
        <div class="icon-option">
          <span class="icon-option__label">{{ item.label }}</span>
        </div>
      </template>
    </el-autocomplete>

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

.prop-editor :deep(.el-autocomplete) {
  width: 100%;
}

.prop-editor :deep(.monaco-editor-container) {
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  overflow: hidden;
}

.icon-option__label {
  color: var(--designer-text-primary);
}
</style>
