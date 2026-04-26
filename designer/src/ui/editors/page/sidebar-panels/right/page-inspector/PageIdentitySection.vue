<script setup lang="ts">
import type { PageInspectorFormState } from "./page-inspector-types";
import { useI18n } from "vue-i18n";

defineProps<{
  form: PageInspectorFormState;
  isSystemPage: boolean;
  pageId: string;
}>();

defineEmits<{
  updateName: [];
  updateConfig: [];
}>();

const { t } = useI18n();
</script>

<template>
  <div class="page-section-fields">
    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.name") }}</div>
      <div class="page-prop-editor">
        <el-input
          v-model="form.name"
          size="small"
          :disabled="isSystemPage"
          @blur="$emit('updateName')"
        />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">
        <span>{{ t("pageInspector.labels.title") }}</span>
        <el-tooltip :content="t('pageInspector.tooltips.runtimeTitle')" placement="top">
          <span class="label-tip">?</span>
        </el-tooltip>
      </div>
      <div class="page-prop-editor">
        <el-input
          v-model="form.title"
          size="small"
          :placeholder="t('pageInspector.placeholders.title')"
          @blur="$emit('updateConfig')"
        />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.description") }}</div>
      <div class="page-prop-editor">
        <el-input
          v-model="form.description"
          size="small"
          :placeholder="t('pageInspector.placeholders.description')"
          @blur="$emit('updateConfig')"
        />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.pageId") }}</div>
      <div class="page-prop-editor">
        <el-input :model-value="pageId" size="small" readonly />
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

.page-prop-editor :deep(.el-input) {
  width: 100%;
}
</style>
