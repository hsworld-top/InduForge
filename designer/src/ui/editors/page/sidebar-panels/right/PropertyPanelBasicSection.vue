<!--
  属性面板「基本」区块：名称、描述、类型、ID、只读位置
-->
<script setup lang="ts">
import { unref } from "vue";
import { formatStyleValue } from "./property-panel-utils";

interface BasicSectionStyleLike {
  left?: string | number | null;
  top?: string | number | null;
}

defineProps<{
  elementType: string;
  elementId: string;
  currentStyle: BasicSectionStyleLike;
}>();

const emit = defineEmits<{
  (event: "labelCommit"): void;
  (event: "descriptionCommit"): void;
}>();

const label = defineModel<string>("label", { default: "" });
const description = defineModel<string>("description", { default: "" });

function onLabelUpdate(v: string | null | undefined) {
  label.value = v ?? "";
}

function onDescriptionUpdate(v: string | null | undefined) {
  description.value = v ?? "";
}
</script>

<template>
  <div class="prop-section">
    <div class="prop-section-header is-static">
      <span class="prop-section-title">基本</span>
    </div>
    <div class="prop-section-body">
      <div class="prop-item">
        <div class="prop-label">名称</div>
        <el-input
          :model-value="unref(label)"
          size="small"
          placeholder="未命名"
          @update:model-value="onLabelUpdate"
          @change="emit('labelCommit')"
        />
      </div>
      <div class="prop-item">
        <div class="prop-label">描述</div>
        <el-input
          :model-value="unref(description)"
          size="small"
          placeholder="请输入描述"
          @update:model-value="onDescriptionUpdate"
          @change="emit('descriptionCommit')"
        />
      </div>
      <div class="prop-item">
        <div class="prop-label">类型</div>
        <el-input :model-value="elementType" size="small" disabled />
      </div>
      <div class="prop-item">
        <div class="prop-label">ID</div>
        <el-input :model-value="elementId" size="small" disabled />
      </div>
      <div class="prop-item">
        <div class="prop-label">位置</div>
        <div class="axis-inline-group">
          <div class="axis-inline-item">
            <span class="axis-inline-tag">X</span>
            <el-input :model-value="formatStyleValue(currentStyle.left)" size="small" disabled />
          </div>
          <div class="axis-inline-item">
            <span class="axis-inline-tag">Y</span>
            <el-input :model-value="formatStyleValue(currentStyle.top)" size="small" disabled />
          </div>
        </div>
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

.prop-section-header:hover {
  background: var(--designer-hover-surface);
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

.prop-item :deep(.el-input) {
  width: 100%;
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

.axis-inline-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--designer-gap-xs);
  flex: 1;
  min-width: 0;
}

.axis-inline-item {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.axis-inline-item :deep(.el-input) {
  width: 100%;
}

.axis-inline-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  min-width: 18px;
  height: 18px;
  border-radius: 999px;
  background: var(--designer-group-surface);
  color: var(--designer-text-secondary);
  font-size: 11px;
  font-weight: 600;
}
</style>
