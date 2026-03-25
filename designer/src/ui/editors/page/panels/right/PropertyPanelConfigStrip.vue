<!--
  属性面板：尺寸编辑 + 详细/样式配置入口
-->
<script setup>
import IconEpDocument from "~icons/ep/document";
import IconEpEditPen from "~icons/ep/edit-pen";
import SizeEditor from "./StylePanel/SizeEditor.vue";

defineProps({
  showSizeEditor: { type: Boolean, default: false },
  currentStyle: { type: Object, required: true },
  containerMinSize: { type: Object, default: null },
  hasDetailConfig: { type: Boolean, default: false },
  hasStyleConfig: { type: Boolean, default: false },
});

const emit = defineEmits(["update:currentStyle", "openConfig"]);

function onStyleUpdate(val) {
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
        <span class="panel-section-title">配置</span>
      </div>
      <div class="config-entry-bar">
        <button
          class="config-entry"
          :class="{ 'has-config': hasDetailConfig }"
          type="button"
          @click="emit('openConfig', 'detail')"
        >
          <IconEpDocument class="config-entry-icon" />
          <span>详细</span>
        </button>
        <button
          class="config-entry"
          :class="{ 'has-config': hasStyleConfig }"
          type="button"
          @click="emit('openConfig', 'style')"
        >
          <IconEpEditPen class="config-entry-icon" />
          <span>样式</span>
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
}

.panel-section {
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
}

.config-entry {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--designer-gap-xs);
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
