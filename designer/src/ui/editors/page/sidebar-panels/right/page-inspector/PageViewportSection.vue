<script setup lang="ts">
import type { PageInspectorFormState, ViewportPreset } from "./page-inspector-types";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

defineProps<{
  form: PageInspectorFormState;
  canEditConstraintOptions: boolean;
}>();

defineEmits<{
  updateConfig: [];
  presetChange: [value: ViewportPreset];
}>();

const { t } = useI18n();

const presetOptions = computed(() => [
  { label: t("pageInspector.viewportPresets.bigscreen"), value: "bigscreen" },
  { label: t("pageInspector.viewportPresets.pc"), value: "pc" },
  { label: t("pageInspector.viewportPresets.tablet"), value: "tablet" },
  { label: t("pageInspector.viewportPresets.phoneLandscape"), value: "phoneLandscape" },
  { label: t("pageInspector.viewportPresets.phonePortrait"), value: "phonePortrait" },
  { label: t("pageInspector.viewportPresets.custom"), value: "custom" },
]);

const overflowOptions = computed(() => [
  { label: t("pageInspector.overflowModes.auto"), value: "auto" },
  { label: t("pageInspector.overflowModes.hidden"), value: "hidden" },
  { label: t("pageInspector.overflowModes.scroll"), value: "scroll" },
]);
</script>

<template>
  <div class="page-section-fields">
    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.viewportPreset") }}</div>
      <div class="page-prop-editor">
        <el-select
          v-model="form.viewportPreset"
          size="small"
          @change="$emit('presetChange', $event as ViewportPreset)"
        >
          <el-option
            v-for="item in presetOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.canvasWidth") }}</div>
      <div class="page-prop-editor">
        <el-input-number v-model="form.width" size="small" :min="1" @change="$emit('updateConfig')" />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.canvasHeight") }}</div>
      <div class="page-prop-editor">
        <el-input-number v-model="form.height" size="small" :min="1" @change="$emit('updateConfig')" />
      </div>
    </div>

    <div class="page-prop-item page-prop-item--switch">
      <div class="page-prop-label">{{ t("pageInspector.labels.autoFit") }}</div>
      <div class="page-prop-editor page-prop-editor-switch">
        <el-switch v-model="form.autoFit" @change="$emit('updateConfig')" />
      </div>
    </div>

    <div class="page-prop-item page-prop-item--switch">
      <div class="page-prop-label">{{ t("pageInspector.labels.lockAspectRatio") }}</div>
      <div class="page-prop-editor page-prop-editor-switch">
        <el-switch
          v-model="form.lockAspectRatio"
          :disabled="!canEditConstraintOptions"
          @change="$emit('updateConfig')"
        />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.minWidth") }}</div>
      <div class="page-prop-editor">
        <el-input-number
          v-model="form.minWidth"
          size="small"
          :min="0"
          :disabled="!canEditConstraintOptions"
          @change="$emit('updateConfig')"
        />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.minHeight") }}</div>
      <div class="page-prop-editor">
        <el-input-number
          v-model="form.minHeight"
          size="small"
          :min="0"
          :disabled="!canEditConstraintOptions"
          @change="$emit('updateConfig')"
        />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.overflowMode") }}</div>
      <div class="page-prop-editor">
        <el-select v-model="form.overflowMode" size="small" @change="$emit('updateConfig')">
          <el-option
            v-for="item in overflowOptions"
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
  width: 92px;
  min-width: 92px;
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

.page-prop-editor {
  flex: 1;
  min-width: 0;
}

.page-prop-editor-switch {
  display: flex;
  justify-content: flex-end;
}

.page-prop-item--switch {
  justify-content: space-between;
}

.page-prop-editor :deep(.el-select),
.page-prop-editor :deep(.el-input-number) {
  width: 100%;
}
</style>
