<!--
  属性面板：尺寸编辑 + 详细/样式配置入口
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import IconEpDocument from "~icons/ep/document";
import IconEpEditPen from "~icons/ep/edit-pen";
import SizeEditor from "./StylePanel/SizeEditor.vue";

interface ConfigStripStyleLike {
  width?: string;
  height?: string;
  [key: string]: unknown;
}

interface ConfigStripContainerMinSize {
  width?: number;
  height?: number;
}

defineProps<{
  showSizeEditor?: boolean;
  currentStyle: ConfigStripStyleLike;
  containerMinSize?: ConfigStripContainerMinSize | null;
  hasDetailConfig?: boolean;
  hasStyleConfig?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:currentStyle", value: ConfigStripStyleLike): void;
  (event: "openConfig", tab: "detail" | "style"): void;
}>();
const { t } = useI18n();

function onStyleUpdate(val: ConfigStripStyleLike) {
  emit("update:currentStyle", val);
}
</script>

<template>
  <div class="property-panel-config-strip">
    <div v-if="showSizeEditor" class="panel-section">
      <SizeEditor
        :model-value="currentStyle"
        :min-width="containerMinSize?.width"
        :min-height="containerMinSize?.height"
        @update:model-value="onStyleUpdate"
      />
    </div>
    <div class="panel-section">
      <div class="config-entry-header">
        <span class="panel-section-title">{{ t("propertyPanel.configStrip.title") }}</span>
      </div>
      <div class="config-entry-bar">
        <button
          class="config-entry"
          :class="{ 'has-config': hasDetailConfig }"
          type="button"
          @click="emit('openConfig', 'detail')"
        >
          <IconEpDocument class="config-entry-icon" />
          <span>{{ t("propertyPanel.configStrip.detail") }}</span>
        </button>
        <button
          class="config-entry"
          :class="{ 'has-config': hasStyleConfig }"
          type="button"
          @click="emit('openConfig', 'style')"
        >
          <IconEpEditPen class="config-entry-icon" />
          <span>{{ t("propertyPanel.configStrip.style") }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.property-panel-config-strip {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-md, 12px);
  min-width: 0;
}

.panel-section {
  box-sizing: border-box;
  min-width: 0;
  padding: 8px 10px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
}

.panel-section-title {
  font-size: var(--designer-font-sm);
  font-weight: 600;
  color: var(--designer-text-secondary);
}

.config-entry-header {
  margin-bottom: var(--designer-gap-xs);
}

.config-entry-bar {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
  min-width: 0;
}

.config-entry {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--designer-gap-xs);
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-group-surface);
  color: var(--designer-text-secondary);
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease,
    color 0.15s ease;
}

.config-entry:hover {
  border-color: var(--designer-primary-border);
  color: var(--designer-primary-text);
  background: var(--designer-primary-soft);
}

.config-entry.has-config {
  border-color: var(--designer-primary-border);
  color: var(--designer-primary-text);
  background: var(--designer-primary-soft);
}

.config-entry-icon {
  width: var(--designer-panel-icon);
  height: var(--designer-panel-icon);
}
</style>
