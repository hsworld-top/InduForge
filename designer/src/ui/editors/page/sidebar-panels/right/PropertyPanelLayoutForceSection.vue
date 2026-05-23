<!--
  属性面板：布局强制属性区块（静态 prop-section）
-->
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import IconEpLink from '~icons/ep/link'
import PropEditor from './PropEditor.vue'

interface LayoutPropDefLike {
  name: string
  label: string
  type?: string
  editor?: string
  language?: string
  height?: string
  placeholder?: string
  min?: number
  max?: number
  step?: number
  options?: Array<{ label: string; value: string | number }>
}

defineProps<{
  visibleProps: LayoutPropDefLike[]
  shouldShowBindButton: (propDef: LayoutPropDefLike) => boolean
  hasPropBinding: (name: string) => boolean
  getPropValue: (
    name: string,
  ) => string | number | boolean | Record<string, unknown> | unknown[] | null | undefined
}>()

const emit = defineEmits<{
  (event: 'bindClick', propDef: LayoutPropDefLike): void
  (event: 'propChange', name: string, value: unknown): void
}>()
const { t } = useI18n()

function handlePropChange(name: string, value: unknown) {
  emit('propChange', name, value)
}
</script>

<template>
  <div class="prop-section prop-section--static">
    <div class="prop-section-header is-static">
      <span class="prop-section-title">{{ t('propertyPanel.layout.title') }}</span>
    </div>
    <div class="prop-section-body">
      <div
        v-for="(propDef, propIndex) in visibleProps"
        :key="propDef?.name || propIndex"
        class="prop-item"
      >
        <div class="prop-label">
          <span>{{ propDef.label }}</span>
          <el-tooltip
            v-if="shouldShowBindButton(propDef)"
            :content="t('propertyPanel.layout.bindData')"
            placement="top"
          >
            <button
              class="bind-btn"
              :class="{ 'is-active': hasPropBinding(propDef.name) }"
              type="button"
              @click="emit('bindClick', propDef)"
            >
              <IconEpLink class="bind-icon" />
            </button>
          </el-tooltip>
        </div>
        <PropEditor
          class="prop-editor"
          :prop="propDef"
          :model-value="getPropValue(propDef.name) ?? null"
          @update:model-value="(val: unknown) => handlePropChange(propDef.name, val)"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.prop-section {
  overflow: hidden;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
}

.prop-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-height: 30px;
  padding: 0 10px;
  border: none;
  border-bottom: 1px solid var(--designer-border-soft);
  background: var(--designer-group-surface);
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.prop-section-header.is-static {
  cursor: default;
}

.prop-section-header.is-static:hover {
  background: var(--designer-group-surface);
}

.prop-section-title {
  font-size: var(--designer-font-sm);
  font-weight: 600;
  color: var(--designer-text-secondary);
  letter-spacing: 0.02em;
}

.prop-section-body {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
  padding: 4px 6px;
}

.prop-item {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-height: 28px;
  padding: 2px 4px;
  border-radius: var(--designer-radius-sm);
  transition: background-color 0.15s ease;
}

.prop-item:hover {
  background: var(--designer-hover-surface);
}

.prop-item > :last-child:not(.prop-label) {
  flex: 1;
  min-width: 0;
}

.prop-item .prop-editor {
  flex: 1;
  min-width: 0;
}

.prop-item :deep(.el-input),
.prop-item :deep(.el-input-number),
.prop-item :deep(.el-select),
.prop-item :deep(.el-color-picker) {
  width: 100%;
}

.prop-item :deep(.el-switch) {
  margin-left: auto;
}

.prop-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  width: 88px;
  min-width: 72px;
  max-width: 88px;
  gap: 4px;
  font-size: var(--designer-font-sm);
  color: var(--designer-text-regular);
}

.bind-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 1px solid var(--designer-primary-border);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-primary-soft);
  cursor: pointer;
  opacity: 1;
  color: var(--designer-primary-text);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.35);
  transition:
    opacity 0.15s ease,
    border-color 0.15s ease,
    color 0.15s ease,
    background-color 0.15s ease;
  flex-shrink: 0;
}

.bind-btn:hover {
  border-color: var(--designer-primary-border);
  color: var(--designer-primary-text);
  background: var(--designer-active-surface);
}

.bind-btn.is-active {
  opacity: 1;
  border-color: var(--designer-primary-border);
  color: var(--designer-shell-surface);
  background: var(--designer-primary);
  box-shadow: none;
}

.bind-icon {
  width: 13px;
  height: 13px;
}
</style>
