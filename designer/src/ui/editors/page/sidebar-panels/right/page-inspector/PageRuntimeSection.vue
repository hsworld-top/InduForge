<script setup lang="ts">
import type { OpenMode, PageInspectorFormState } from "./page-inspector-types";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  form: PageInspectorFormState;
  isSystemPage: boolean;
}>();

defineEmits<{
  updateConfig: [];
}>();

const { t } = useI18n();

const openModeOptions = computed(() => [
  { label: t("pageInspector.openModes.popup"), value: "popup" },
  { label: t("pageInspector.openModes.cover"), value: "cover" },
  { label: t("pageInspector.openModes.replace"), value: "replace" },
]);

const cacheOptions = computed(() => [
  { label: t("pageInspector.cacheModes.default"), value: "default" },
  { label: t("pageInspector.cacheModes.cache"), value: "cache" },
  { label: t("pageInspector.cacheModes.noCache"), value: "no-cache" },
]);

const preloadOptions = computed(() => [
  { label: t("pageInspector.preloadModes.lazy"), value: "lazy" },
  { label: t("pageInspector.preloadModes.eager"), value: "eager" },
]);

const showPopupOptions = computed(() => props.form.openMode === "popup");
</script>

<template>
  <div class="page-section-fields">
    <div class="page-prop-item">
      <div class="page-prop-label">
        <span>{{ t("pageInspector.labels.openMode") }}</span>
        <el-tooltip
          v-if="isSystemPage"
          :content="t('pageInspector.tooltips.openModeReadonly')"
          placement="top"
        >
          <span class="label-tip">?</span>
        </el-tooltip>
      </div>
      <div class="page-prop-editor">
        <el-select
          v-model="form.openMode"
          size="small"
          :disabled="isSystemPage"
          @change="$emit('updateConfig')"
        >
          <el-option
            v-for="item in openModeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>

    <template v-if="showPopupOptions">
      <div class="page-prop-item">
        <div class="page-prop-label">{{ t("pageInspector.labels.popupWidth") }}</div>
        <div class="page-prop-editor">
          <el-input-number
            v-model="form.popupWidth"
            size="small"
            :min="1"
            @change="$emit('updateConfig')"
          />
        </div>
      </div>

      <div class="page-prop-item">
        <div class="page-prop-label">{{ t("pageInspector.labels.popupHeight") }}</div>
        <div class="page-prop-editor">
          <el-input-number
            v-model="form.popupHeight"
            size="small"
            :min="1"
            @change="$emit('updateConfig')"
          />
        </div>
      </div>

      <div class="page-prop-item page-prop-item--switch">
        <div class="page-prop-label">{{ t("pageInspector.labels.popupCenter") }}</div>
        <div class="page-prop-editor page-prop-editor-switch">
          <el-switch v-model="form.popupCenter" @change="$emit('updateConfig')" />
        </div>
      </div>

      <div class="page-prop-item page-prop-item--switch">
        <div class="page-prop-label">{{ t("pageInspector.labels.popupMaskClosable") }}</div>
        <div class="page-prop-editor page-prop-editor-switch">
          <el-switch v-model="form.popupMaskClosable" @change="$emit('updateConfig')" />
        </div>
      </div>
    </template>

    <div class="page-prop-item">
      <div class="page-prop-label">
        <span>{{ t("pageInspector.labels.cacheMode") }}</span>
        <el-tooltip :content="t('pageInspector.tooltips.cacheMode')" placement="top">
          <span class="label-tip">?</span>
        </el-tooltip>
      </div>
      <div class="page-prop-editor">
        <el-select v-model="form.cacheMode" size="small" @change="$emit('updateConfig')">
          <el-option
            v-for="item in cacheOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">
        <span>{{ t("pageInspector.labels.preloadMode") }}</span>
        <el-tooltip :content="t('pageInspector.tooltips.preloadMode')" placement="top">
          <span class="label-tip">?</span>
        </el-tooltip>
      </div>
      <div class="page-prop-editor">
        <el-select v-model="form.preloadMode" size="small" @change="$emit('updateConfig')">
          <el-option
            v-for="item in preloadOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
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

.page-prop-editor-inline {
  display: flex;
  align-items: center;
}

.page-prop-editor-stacked {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: var(--designer-gap-xs);
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

.page-prop-editor-switch {
  display: flex;
  justify-content: flex-end;
}

.page-prop-item--switch {
  justify-content: space-between;
}

.page-prop-editor :deep(.el-select),
.page-prop-editor :deep(.el-input-number),
.page-prop-editor :deep(.el-input) {
  width: 100%;
}

</style>
