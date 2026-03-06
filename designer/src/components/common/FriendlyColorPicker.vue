<template>
  <div class="friendly-color-picker" :class="{ 'is-disabled': disabled }">
    <div class="color-main-row">
      <el-color-picker
        :model-value="modelValue"
        :show-alpha="showAlpha"
        :disabled="disabled"
        :predefine="predefineColors"
        color-format="hex"
        @change="handlePickerChange"
      />
      <el-input
        v-model="inputDraft"
        :disabled="disabled"
        :placeholder="placeholder"
        size="small"
        class="color-input"
        @blur="commitInputDraft"
        @keyup.enter="commitInputDraft"
      />
      <el-button
        v-if="clearable"
        size="small"
        :disabled="disabled"
        class="clear-button"
        @click="clearColor"
      >
        清空
      </el-button>
    </div>

    <div v-if="recentColors.length" class="recent-row">
      <span class="recent-label">最近</span>
      <button
        v-for="color in recentColors"
        :key="color"
        type="button"
        class="recent-color"
        :style="{ backgroundColor: color }"
        :title="color"
        @click="selectRecentColor(color)"
      />
    </div>
  </div>
</template>

<script setup>
/**
 * 友好颜色选择器
 * 提供颜色面板、手动输入、预设色与最近使用能力
 */

import { computed, onMounted, ref, watch } from "vue";
import { ElMessage } from "element-plus";

const RECENT_COLORS_STORAGE_KEY = "designer:recent-colors";
const MAX_RECENT_COLORS = 8;
const DEFAULT_PREDEFINE_COLORS = [
  "#ffffff",
  "#f5f7fa",
  "#e4e7ed",
  "#dcdfe6",
  "#c0c4cc",
  "#909399",
  "#606266",
  "#303133",
  "#000000",
  "#409eff",
  "#67c23a",
  "#e6a23c",
  "#f56c6c",
  "#909399",
  "rgba(64, 158, 255, 0.2)",
  "rgba(0, 0, 0, 0.35)",
];

const props = defineProps({
  modelValue: {
    type: String,
    default: "",
  },
  showAlpha: {
    type: Boolean,
    default: true,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  clearable: {
    type: Boolean,
    default: true,
  },
  placeholder: {
    type: String,
    default: "请输入颜色值",
  },
  predefine: {
    type: Array,
    default: () => [],
  },
});

const emit = defineEmits(["update:modelValue", "change"]);

const inputDraft = ref("");
const recentColors = ref([]);

const predefineColors = computed(() => {
  if (props.predefine.length) return props.predefine;
  return DEFAULT_PREDEFINE_COLORS;
});

/**
 * 读取最近颜色列表
 */
const loadRecentColors = () => {
  try {
    const raw = localStorage.getItem(RECENT_COLORS_STORAGE_KEY);
    if (!raw) {
      recentColors.value = [];
      return;
    }
    const parsed = JSON.parse(raw);
    recentColors.value = Array.isArray(parsed)
      ? parsed.filter((item) => typeof item === "string")
      : [];
  } catch (error) {
    recentColors.value = [];
  }
};

/**
 * 保存最近颜色列表
 */
const persistRecentColors = () => {
  try {
    localStorage.setItem(
      RECENT_COLORS_STORAGE_KEY,
      JSON.stringify(recentColors.value)
    );
  } catch (error) {
    // 本地存储不可用时忽略，不影响颜色选择核心能力
  }
};

/**
 * 规范化输入颜色值
 * @param {string} value - 原始颜色值
 * @returns {string}
 */
const normalizeColorValue = (value) => {
  const text = String(value || "").trim();
  if (!text) return "";
  if (/^#([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/.test(text)) {
    return text;
  }
  if (/^([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/.test(text)) {
    return `#${text}`;
  }
  if (/^(rgb|rgba|hsl|hsla|var)\(/i.test(text)) {
    return text;
  }
  return "";
};

/**
 * 记录最近颜色
 * @param {string} color - 颜色值
 */
const pushRecentColor = (color) => {
  const normalized = normalizeColorValue(color);
  if (!normalized) return;
  recentColors.value = [
    normalized,
    ...recentColors.value.filter((item) => item !== normalized),
  ].slice(0, MAX_RECENT_COLORS);
  persistRecentColors();
};

/**
 * 同步并派发颜色值
 * @param {string} color - 颜色值
 */
const emitColor = (color) => {
  emit("update:modelValue", color);
  emit("change", color);
};

/**
 * 处理面板选择
 * @param {string} color - 颜色值
 */
const handlePickerChange = (color) => {
  const normalized = normalizeColorValue(color);
  emitColor(normalized);
  inputDraft.value = normalized;
  pushRecentColor(normalized);
};

/**
 * 手动输入提交
 */
const commitInputDraft = () => {
  const normalized = normalizeColorValue(inputDraft.value);
  if (!inputDraft.value) {
    emitColor("");
    return;
  }
  if (!normalized) {
    ElMessage.warning("颜色格式无效，请输入 HEX/RGBA/HSL");
    inputDraft.value = props.modelValue || "";
    return;
  }
  emitColor(normalized);
  inputDraft.value = normalized;
  pushRecentColor(normalized);
};

/**
 * 清空颜色
 */
const clearColor = () => {
  inputDraft.value = "";
  emitColor("");
};

/**
 * 选择最近颜色
 * @param {string} color - 颜色值
 */
const selectRecentColor = (color) => {
  if (props.disabled) return;
  const normalized = normalizeColorValue(color);
  if (!normalized) return;
  inputDraft.value = normalized;
  emitColor(normalized);
  pushRecentColor(normalized);
};

watch(
  () => props.modelValue,
  (value) => {
    inputDraft.value = value || "";
  },
  { immediate: true }
);

onMounted(() => {
  loadRecentColors();
});
</script>

<style scoped>
.friendly-color-picker {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.friendly-color-picker.is-disabled {
  opacity: 0.65;
}

.color-main-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.color-input {
  flex: 1;
  min-width: 96px;
}

.clear-button {
  flex: 0 0 auto;
}

.recent-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.recent-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.recent-color {
  width: 14px;
  height: 14px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  cursor: pointer;
  padding: 0;
  background: transparent;
  transition: transform 0.15s ease, border-color 0.15s ease;
}

.recent-color:hover {
  transform: translateY(-1px);
  border-color: var(--el-color-primary);
}
</style>
