<script setup lang="ts">
/**
 * 友好颜色选择器
 * 提供颜色面板、手动输入、预设色与最近使用能力
 */

import { computed, onMounted, ref, watch } from 'vue'
import IconCircleClose from '~icons/ep/circle-close'
import { ElMessage } from '@/ui/shell/el-message-compat'

interface FriendlyColorPickerProps {
  modelValue?: string
  showAlpha?: boolean
  disabled?: boolean
  clearable?: boolean
  placeholder?: string
  predefine?: string[]
  showRecent?: boolean
  layout?: 'inline' | 'block'
}

const props = withDefaults(defineProps<FriendlyColorPickerProps>(), {
  modelValue: '',
  showAlpha: true,
  disabled: false,
  clearable: true,
  placeholder: '请输入颜色值',
  predefine: () => [],
  showRecent: true,
  layout: 'inline',
})
const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'change', value: string): void
}>()
const RECENT_COLORS_STORAGE_KEY = 'designer:recent-colors'
const MAX_RECENT_COLORS = 5
const HEX_COLOR_RE = /^#(?:[0-9a-f]{3,4}|[0-9a-f]{6}|[0-9a-f]{8})$/i
const HEX_COLOR_BODY_RE = /^(?:[0-9a-f]{3,4}|[0-9a-f]{6}|[0-9a-f]{8})$/i
const FUNCTION_COLOR_RE = /^(?:rgb|rgba|hsl|hsla|var)\(/i
const DEFAULT_PREDEFINE_COLORS = [
  '#ffffff',
  '#f5f7fa',
  '#e4e7ed',
  '#dcdfe6',
  '#c0c4cc',
  '#909399',
  '#606266',
  '#303133',
  '#000000',
  '#409eff',
  '#67c23a',
  '#e6a23c',
  '#f56c6c',
  '#909399',
  'rgba(64, 158, 255, 0.2)',
  'rgba(0, 0, 0, 0.35)',
]

const inputDraft = ref('')
const recentColors = ref<string[]>([])

const predefineColors = computed(() => {
  if (props.predefine.length) return props.predefine
  return DEFAULT_PREDEFINE_COLORS
})

/** 展示用最近颜色，最多 5 个 */
const displayRecentColors = computed(() => recentColors.value.slice(0, MAX_RECENT_COLORS))

/**
 * 读取最近颜色列表
 */
function loadRecentColors() {
  try {
    const raw = localStorage.getItem(RECENT_COLORS_STORAGE_KEY)
    if (!raw) {
      recentColors.value = []
      return
    }
    const parsed = JSON.parse(raw)
    recentColors.value = Array.isArray(parsed)
      ? parsed.filter((item): item is string => typeof item === 'string')
      : []
  } catch {
    recentColors.value = []
  }
}

/**
 * 保存最近颜色列表
 */
function persistRecentColors() {
  try {
    localStorage.setItem(RECENT_COLORS_STORAGE_KEY, JSON.stringify(recentColors.value))
  } catch {
    // 本地存储不可用时忽略，不影响颜色选择核心能力
  }
}

/**
 * 规范化输入颜色值
 * @param {string} value - 原始颜色值
 * @returns {string}
 */
function normalizeColorValue(value: string): string {
  const text = String(value || '').trim()
  if (!text) return ''
  if (HEX_COLOR_RE.test(text)) {
    return text
  }
  if (HEX_COLOR_BODY_RE.test(text)) {
    return `#${text}`
  }
  if (FUNCTION_COLOR_RE.test(text)) {
    return text
  }
  return ''
}

/**
 * 记录最近颜色
 * @param {string} color - 颜色值
 */
function pushRecentColor(color: string) {
  const normalized = normalizeColorValue(color)
  if (!normalized) return
  recentColors.value = [
    normalized,
    ...recentColors.value.filter((item) => item !== normalized),
  ].slice(0, MAX_RECENT_COLORS)
  persistRecentColors()
}

/**
 * 同步并派发颜色值
 * @param {string} color - 颜色值
 */
function emitColor(color: string) {
  emit('update:modelValue', color)
  emit('change', color)
}

/**
 * 处理面板选择
 * @param {string} color - 颜色值
 */
function handlePickerChange(color: string) {
  const normalized = normalizeColorValue(color)
  emitColor(normalized)
  inputDraft.value = normalized
  pushRecentColor(normalized)
}

/**
 * 手动输入提交
 */
function commitInputDraft() {
  const normalized = normalizeColorValue(inputDraft.value)
  if (!inputDraft.value) {
    emitColor('')
    return
  }
  if (!normalized) {
    ElMessage.warning('颜色格式无效，请输入 HEX/RGBA/HSL')
    inputDraft.value = props.modelValue || ''
    return
  }
  emitColor(normalized)
  inputDraft.value = normalized
  pushRecentColor(normalized)
}

/**
 * 清空颜色
 */
function clearColor() {
  inputDraft.value = ''
  emitColor('')
}

/**
 * 选择最近颜色
 * @param {string} color - 颜色值
 */
function selectRecentColor(color: string) {
  if (props.disabled) return
  const normalized = normalizeColorValue(color)
  if (!normalized) return
  inputDraft.value = normalized
  emitColor(normalized)
  pushRecentColor(normalized)
}

watch(
  () => props.modelValue,
  (value: string) => {
    inputDraft.value = value || ''
  },
  { immediate: true },
)

onMounted(() => {
  loadRecentColors()
})
</script>

<template>
  <div
    class="friendly-color-picker"
    :class="[`friendly-color-picker--${layout}`, { 'is-disabled': disabled }]"
  >
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
      >
        <template v-if="clearable" #suffix>
          <el-icon class="clear-icon" :class="{ 'is-disabled': disabled }" @click.stop="clearColor">
            <IconCircleClose />
          </el-icon>
        </template>
      </el-input>
    </div>

    <div v-if="showRecent && recentColors.length" class="recent-row">
      <span class="recent-label">最近</span>
      <button
        v-for="color in displayRecentColors"
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

<style scoped>
.friendly-color-picker {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.friendly-color-picker.is-disabled {
  opacity: 0.65;
}

.color-main-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.color-input {
  flex: 1;
  min-width: 0;
}

.friendly-color-picker--block {
  gap: 6px;
}

.friendly-color-picker--block .color-main-row {
  align-items: stretch;
  flex-direction: column;
  gap: 6px;
}

.friendly-color-picker--block :deep(.el-color-picker) {
  display: block;
  width: 100%;
  height: 28px;
}

.friendly-color-picker--block :deep(.el-color-picker__trigger) {
  width: 100%;
  height: 28px;
  padding: 0;
  overflow: hidden;
  border: 1px solid var(--designer-border-color, var(--el-border-color));
  border-radius: var(--designer-radius-sm, 4px);
  box-shadow: none;
}

.friendly-color-picker--block :deep(.el-color-picker__color) {
  width: 100%;
  height: 100%;
  border: 0;
}

.friendly-color-picker--block :deep(.el-color-picker__color-inner) {
  border-radius: var(--designer-radius-sm, 4px);
}

.friendly-color-picker--block :deep(.el-color-picker__icon) {
  display: none;
}

.friendly-color-picker--block .color-input {
  width: 100%;
  flex: none;
}

.clear-icon {
  cursor: pointer;
  font-size: 14px;
  color: var(--el-text-color-placeholder);
  transition: color 0.2s;
}

.clear-icon:hover {
  color: var(--el-color-danger);
}

.clear-icon.is-disabled {
  cursor: not-allowed;
  pointer-events: none;
}

.recent-row {
  display: flex;
  align-items: center;
  gap: 5px;
  padding-left: 2px;
}

.recent-label {
  flex: 0 0 auto;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  user-select: none;
}

.recent-color {
  width: 16px;
  height: 16px;
  border: 1px solid var(--el-border-color);
  border-radius: 3px;
  cursor: pointer;
  padding: 0;
  background: transparent;
  transition:
    transform 0.15s ease,
    border-color 0.15s ease;
}

.recent-color:hover {
  transform: scale(1.15);
  border-color: var(--el-color-primary);
}
</style>
