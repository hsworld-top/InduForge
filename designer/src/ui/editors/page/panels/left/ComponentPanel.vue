<!--
  ComponentPanel - 组件物料面板
  当前版本仅保留：6 个布局容器 + Button
-->
<script setup lang="ts">
import { computed, h, ref } from "vue";
import { componentRegistry } from "@/editor-core";
import { endDrag, startDrag } from "@/ui/editors/page/canvas/composables/use-drag-state";

interface ComponentItemLike {
  type: string;
  name: string;
}

type PreviewComponentMapLike = Record<string, () => ReturnType<typeof h>>;

const keyword = ref("");
const activeSections = ref(["layout", "ui"]);

const layoutTypeOrder = [
  "HorizontalLayout",
  "VerticalLayout",
  "Collapse",
  "Tabs",
  "FormLayout",
  "ElContainer",
];

const allowedTypesByCategory: Record<string, string[]> = {
  layout: layoutTypeOrder,
  uiPc: ["Button"],
};

function filterItemsByCategory(category: string): ComponentItemLike[] {
  const keywordValue = keyword.value.trim().toLowerCase();
  const allowedTypes = allowedTypesByCategory[category] ?? [];
  const items = componentRegistry.getByCategory(category) as ComponentItemLike[];
  const filtered = items.filter((item) => {
    if (!allowedTypes.includes(item.type)) return false;
    if (!keywordValue) return true;
    return (
      item.type.toLowerCase().includes(keywordValue) || item.name.toLowerCase().includes(keywordValue)
    );
  });

  if (category !== "layout") return filtered;
  return filtered.sort((a, b) => layoutTypeOrder.indexOf(a.type) - layoutTypeOrder.indexOf(b.type));
}

const layoutItems = computed<ComponentItemLike[]>(() => filterItemsByCategory("layout"));
const uiItems = computed<ComponentItemLike[]>(() => filterItemsByCategory("uiPc"));

function getPreviewComponent(type: string): { render: () => ReturnType<typeof h> } {
  const previewMap: PreviewComponentMapLike = {
    Button: () => h("div", { class: "preview-button" }, "按钮"),
    HorizontalLayout: () =>
      h("div", { class: "preview-flex" }, [
        h("div", { class: "preview-block" }),
        h("div", { class: "preview-block" }),
        h("div", { class: "preview-block" }),
      ]),
    VerticalLayout: () =>
      h("div", { class: "preview-flex preview-flex-column" }, [
        h("div", { class: "preview-block" }),
        h("div", { class: "preview-block" }),
        h("div", { class: "preview-block" }),
      ]),
    FormLayout: () =>
      h("div", { class: "preview-form" }, [
        h("div", { class: "preview-form-item" }),
        h("div", { class: "preview-form-item" }),
        h("div", { class: "preview-form-item" }),
      ]),
    ElContainer: () =>
      h("div", { class: "preview-el-container" }, [
        h("div", { class: "preview-el-header" }),
        h("div", { class: "preview-el-body" }, [
          h("div", { class: "preview-el-aside" }),
          h("div", { class: "preview-el-main" }),
        ]),
        h("div", { class: "preview-el-footer" }),
      ]),
    Tabs: () =>
      h("div", { class: "preview-tabs" }, [
        h("div", { class: "preview-tabs-header" }),
        h("div", { class: "preview-tabs-body" }),
      ]),
    Collapse: () =>
      h("div", { class: "preview-collapse" }, [
        h("div", { class: "preview-collapse-header" }),
        h("div", { class: "preview-collapse-body" }),
      ]),
  };

  const render = previewMap[type] ?? (() => h("div", { class: "preview-unknown" }, type));
  return { render };
}

function getPreviewIcon(type: string): string {
  const iconMap: Record<string, string> = {
    Button:
      '<div style="width:58px;height:26px;border:1px solid #3b6cff;border-radius:4px;background:#fff;"></div>',
    HorizontalLayout:
      '<div style="width:58px;height:36px;border:1px solid #3b6cff;border-radius:4px;display:flex;gap:4px;padding:4px;"><div style="flex:1;background:#3b6cff;border-radius:2px;"></div><div style="flex:1;background:#3b6cff;border-radius:2px;"></div><div style="flex:1;background:#3b6cff;border-radius:2px;"></div></div>',
    VerticalLayout:
      '<div style="width:58px;height:36px;border:1px solid #3b6cff;border-radius:4px;display:flex;flex-direction:column;gap:4px;padding:4px;"><div style="height:6px;background:#3b6cff;border-radius:2px;"></div><div style="height:6px;background:#3b6cff;border-radius:2px;"></div><div style="height:6px;background:#3b6cff;border-radius:2px;"></div></div>',
    FormLayout:
      '<div style="width:58px;height:36px;border:1px solid #3b6cff;border-radius:4px;display:flex;flex-direction:column;gap:4px;padding:4px;"><div style="height:6px;border:1px solid #3b6cff;border-radius:2px;"></div><div style="height:6px;border:1px solid #3b6cff;border-radius:2px;"></div><div style="height:6px;border:1px solid #3b6cff;border-radius:2px;"></div></div>',
    ElContainer:
      '<div style="width:58px;height:36px;border:1px solid #3b6cff;border-radius:4px;display:flex;flex-direction:column;gap:3px;padding:4px;"><div style="height:5px;background:#3b6cff;border-radius:2px;"></div><div style="flex:1;display:flex;gap:3px;"><div style="width:10px;background:#3b6cff;border-radius:2px;"></div><div style="flex:1;background:#e4e7ed;border-radius:2px;"></div></div><div style="height:5px;background:#3b6cff;border-radius:2px;"></div></div>',
    Tabs:
      '<div style="width:58px;height:36px;border:1px solid #3b6cff;border-radius:4px;"><div style="height:9px;background:#3b6cff;"></div><div style="height:16px;margin:5px;border:1px solid #3b6cff;border-radius:2px;"></div></div>',
    Collapse:
      '<div style="width:58px;height:36px;border:1px solid #3b6cff;border-radius:4px;"><div style="height:9px;background:#3b6cff;"></div><div style="height:14px;margin:5px;border:1px solid #3b6cff;border-radius:2px;"></div></div>',
  };
  return (
    iconMap[type] ||
    '<div style="width:58px;height:36px;border:1px dashed #c0c4cc;border-radius:4px;"></div>'
  );
}

function handleDragStart(item: ComponentItemLike, event: DragEvent): void {
  startDrag(item.type);
  if (!event.dataTransfer) return;

  const payload = JSON.stringify({ type: item.type });
  event.dataTransfer.effectAllowed = "copy";
  event.dataTransfer.setData("application/x-designer-component", payload);
  event.dataTransfer.setData("text/plain", item.type);

  const dragPreview = document.createElement("div");
  dragPreview.innerHTML = `
    <div style="display:flex;flex-direction:column;align-items:center;gap:8px;padding:10px;background:#fff;border:1px solid #409eff;border-radius:8px;box-shadow:0 4px 12px rgba(0,0,0,.15);">
      <div>${getPreviewIcon(item.type)}</div>
      <div style="font-size:12px;color:#303133;">${item.name}</div>
    </div>
  `;
  dragPreview.style.position = "absolute";
  dragPreview.style.top = "-1000px";
  dragPreview.style.left = "-1000px";
  dragPreview.style.pointerEvents = "none";
  document.body.appendChild(dragPreview);
  event.dataTransfer.setDragImage(dragPreview, 60, 30);
  setTimeout(() => {
    if (dragPreview.parentNode) dragPreview.parentNode.removeChild(dragPreview);
  }, 0);
}

function handlePointerStart(item: ComponentItemLike, event: MouseEvent): void {
  if (event.button !== 0) return;
  startDrag(item.type);
}

function handleDragEnd(): void {
  endDrag();
}
</script>

<template>
  <div class="component-panel">
    <el-input v-model="keyword" size="small" placeholder="搜索组件" clearable />
    <div class="component-list">
      <el-collapse v-model="activeSections" class="component-collapse">
        <el-collapse-item name="layout">
          <template #title>
            <span class="component-section-title">布局</span>
          </template>
          <div v-if="layoutItems.length" class="component-grid">
            <div
              v-for="item in layoutItems"
              :key="item.type"
              class="component-card"
              draggable="true"
              @mousedown="handlePointerStart(item, $event)"
              @dragstart="handleDragStart(item, $event)"
              @dragend="handleDragEnd"
            >
              <div class="card-preview">
                <component :is="getPreviewComponent(item.type)" />
              </div>
              <div class="card-name">{{ item.name }}</div>
            </div>
          </div>
          <div v-else class="empty-tip">暂无可用布局</div>
        </el-collapse-item>

        <el-collapse-item name="ui">
          <template #title>
            <span class="component-section-title">UI组件</span>
          </template>
          <div v-if="uiItems.length" class="component-grid">
            <div
              v-for="item in uiItems"
              :key="item.type"
              class="component-card"
              draggable="true"
              @mousedown="handlePointerStart(item, $event)"
              @dragstart="handleDragStart(item, $event)"
              @dragend="handleDragEnd"
            >
              <div class="card-preview">
                <component :is="getPreviewComponent(item.type)" />
              </div>
              <div class="card-name">{{ item.name }}</div>
            </div>
          </div>
          <div v-else class="empty-tip">暂无可用 UI 组件</div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>

<style>
.component-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
}

.component-list {
  flex: 1;
  overflow: auto;
  padding-right: 4px;
}

.component-collapse {
  border: none;
}

.component-collapse .el-collapse-item__header {
  background: #eef3ff;
  border: none;
  border-radius: 6px;
  padding: 0 10px;
  margin-bottom: 6px;
  height: 34px;
  font-size: 13px;
  color: #303133;
}

.component-collapse .el-collapse-item__content {
  padding-bottom: 12px;
}

.component-section-title {
  font-weight: 500;
}

.component-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
  gap: 8px;
}

.component-card {
  display: flex;
  flex-direction: column;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
  cursor: grab;
  transition: all 0.2s;
  background: #fff;
}

.component-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.2);
}

.component-card:active {
  cursor: grabbing;
}

.card-preview {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f7f8fa;
  padding: 10px 8px 6px;
}

.card-name {
  padding: 8px;
  text-align: center;
  font-size: 12px;
  font-weight: 500;
  border-top: 1px solid #e4e7ed;
}

.empty-tip {
  color: #909399;
  font-size: 12px;
  text-align: center;
  padding: 16px 0;
}

.preview-button {
  width: 58px;
  height: 26px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: #3b6cff;
  background: #fff;
}

.preview-flex {
  width: 58px;
  height: 36px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  gap: 4px;
  padding: 4px;
  background: #fff;
}

.preview-flex-column {
  flex-direction: column;
}

.preview-block {
  flex: 1;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-form {
  width: 58px;
  height: 36px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px;
  background: #fff;
}

.preview-form-item {
  height: 6px;
  border: 1px solid #3b6cff;
  border-radius: 2px;
}

.preview-el-container {
  width: 58px;
  height: 36px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 4px;
  background: #fff;
}

.preview-el-header,
.preview-el-footer {
  height: 5px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-el-body {
  flex: 1;
  display: flex;
  gap: 3px;
}

.preview-el-aside {
  width: 10px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-el-main {
  flex: 1;
  background: #e4e7ed;
  border-radius: 2px;
}

.preview-tabs {
  width: 58px;
  height: 36px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  padding: 4px;
  background: #fff;
}

.preview-tabs-header {
  height: 8px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-tabs-body {
  margin-top: 4px;
  height: 16px;
  border: 1px solid #3b6cff;
  border-radius: 2px;
}

.preview-collapse {
  width: 58px;
  height: 36px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  background: #fff;
}

.preview-collapse-header {
  height: 8px;
  background: #3b6cff;
  border-radius: 4px 4px 0 0;
}

.preview-collapse-body {
  height: 14px;
  margin: 5px;
  border: 1px solid #3b6cff;
  border-radius: 2px;
}

.preview-unknown {
  font-size: 10px;
  color: #909399;
}
</style>

