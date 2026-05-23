<script setup lang="ts">
import type { PageInspectorFormState, PageRole, RouteMode } from './page-inspector-types'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  form: PageInspectorFormState
  isSystemPage: boolean
}>()

defineEmits<{
  routeModeChange: [value: RouteMode]
  routePathBlur: []
  routeSlugBlur: []
}>()

const { t } = useI18n()

const roleLabel = computed(() => {
  const role = props.form.role as PageRole
  return t(`pageInspector.roles.${role}`)
})

const routeModeOptions = computed(() => [
  { label: t('pageInspector.routeModes.auto'), value: 'auto' },
  { label: t('pageInspector.routeModes.manual'), value: 'manual' },
])

const routePathDisabled = computed(() => props.isSystemPage || props.form.routeMode === 'auto')
</script>

<template>
  <div class="page-section-fields">
    <div class="page-prop-item">
      <div class="page-prop-label">
        <span>{{ t('pageInspector.labels.role') }}</span>
        <el-tooltip :content="t('pageInspector.tooltips.roleReadonly')" placement="top">
          <span class="label-tip">?</span>
        </el-tooltip>
      </div>
      <div class="page-prop-editor">
        <el-input :model-value="roleLabel" size="small" disabled />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">
        <span>{{ t('pageInspector.labels.routeMode') }}</span>
        <el-tooltip
          v-if="isSystemPage"
          :content="t('pageInspector.tooltips.routeModeReadonly')"
          placement="top"
        >
          <span class="label-tip">?</span>
        </el-tooltip>
      </div>
      <div class="page-prop-editor">
        <el-select
          v-model="form.routeMode"
          size="small"
          :disabled="isSystemPage"
          @change="$emit('routeModeChange', $event as RouteMode)"
        >
          <el-option
            v-for="item in routeModeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t('pageInspector.labels.routePath') }}</div>
      <div class="page-prop-editor">
        <el-input
          data-testid="route-path-input"
          v-model="form.routePath"
          size="small"
          :disabled="routePathDisabled"
          :placeholder="t('pageInspector.placeholders.routePath')"
          @blur="$emit('routePathBlur')"
        />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t('pageInspector.labels.routeSlug') }}</div>
      <div class="page-prop-editor">
        <el-input
          v-model="form.routeSlug"
          size="small"
          :disabled="isSystemPage"
          :placeholder="t('pageInspector.placeholders.routeSlug')"
          @blur="$emit('routeSlugBlur')"
        />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t('pageInspector.labels.parentRoutePath') }}</div>
      <div class="page-prop-editor">
        <el-input :model-value="form.parentRoutePath || '/'" size="small" disabled />
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-section-fields {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}

.page-prop-item {
  display: flex;
  align-items: center;
  gap: var(--designer-gap-sm);
  min-height: 32px;
  padding: 4px 6px;
  border-radius: var(--designer-radius-sm);
}

.page-prop-label {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 92px;
  min-width: 92px;
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

.page-prop-editor {
  flex: 1;
  min-width: 0;
}

.label-tip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--designer-group-surface);
  color: var(--designer-text-secondary);
  font-size: 11px;
  cursor: help;
}

.page-prop-editor :deep(.el-input),
.page-prop-editor :deep(.el-select) {
  width: 100%;
}
</style>
