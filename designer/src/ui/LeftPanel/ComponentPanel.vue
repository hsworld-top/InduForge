<!--
  ComponentPanel - 组件物料面板
  展示可拖拽组件列表（布局、UI、图表等），支持搜索、常用/全部切换
-->
<template>
  <div class="flex flex-col gap-3 component-panel">
    <el-input v-model="keyword" size="small" placeholder="搜索组件" clearable />
    <el-switch
      v-model="showAllComponents"
      size="small"
      inline-prompt
      active-text="全部"
      inactive-text="常用"
    />
    <div class="component-list">
      <el-collapse v-model="activeSections" class="component-collapse">
        <el-collapse-item name="layout">
          <template #title>
            <div class="component-section-title">布局</div>
          </template>
          <div v-if="layoutItems.length" class="grid grid-cols-2 gap-2">
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
              <div class="card-info">
                <div
                  class="text-xs font-medium text-gray-800 dark:text-gray-200"
                >
                  {{ item.name }}
                </div>
              </div>
            </div>
          </div>
          <div v-else class="text-sm text-gray-400 text-center py-6">
            暂无可用组件
          </div>
        </el-collapse-item>

        <el-collapse-item name="ui">
          <template #title>
            <div class="component-section-title">UI组件</div>
          </template>
          <el-collapse v-model="activeUiSections" class="component-subcollapse">
            <el-collapse-item name="pc">
              <template #title>
                <div class="component-subsection-title">PC端组件</div>
              </template>
              <div v-if="pcItems.length" class="grid grid-cols-2 gap-2">
                <div
                  v-for="item in pcItems"
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
                  <div class="card-info">
                    <div
                      class="text-xs font-medium text-gray-800 dark:text-gray-200"
                    >
                      {{ item.name }}
                    </div>
                  </div>
                </div>
              </div>
              <div v-else class="text-sm text-gray-400 text-center py-6">
                暂无可用组件
              </div>
            </el-collapse-item>
          </el-collapse>
        </el-collapse-item>

        <el-collapse-item name="chart">
          <template #title>
            <div class="component-section-title">图表</div>
          </template>
          <div v-if="chartItems.length" class="grid grid-cols-2 gap-2">
            <div
              v-for="item in chartItems"
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
              <div class="card-info">
                <div
                  class="text-xs font-medium text-gray-800 dark:text-gray-200"
                >
                  {{ item.name }}
                </div>
              </div>
            </div>
          </div>
          <div v-else class="text-sm text-gray-400 text-center py-6">
            暂无可用组件
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, h } from "vue";
import { componentRegistry } from "@/editor-core";
import { startDrag, endDrag } from "@/ui/Canvas/use-drag-state";

/**
 * 搜索关键字
 */
const keyword = ref("");
const showAllComponents = ref(false);

/**
 * 折叠面板状态
 */
const activeSections = ref(["layout", "ui", "chart"]);
const activeUiSections = ref(["pc"]);

/**
 * 组件筛选范围
 */
const allowedTypesByCategory = {
  layout: ["HorizontalLayout", "VerticalLayout"],
  uiPc: ["Button"],
  chart: ["EChart"],
};

/**
 * 过滤组件列表
 * @param {string} category - 组件分类
 * @returns {Array}
 */
const filterItemsByCategory = (category) => {
  const keywordValue = keyword.value.trim().toLowerCase();
  const allowedTypes = showAllComponents.value
    ? null
    : allowedTypesByCategory[category];
  return componentRegistry.getByCategory(category).filter((item) => {
    if (Array.isArray(allowedTypes) && !allowedTypes.includes(item.type)) {
      return false;
    }
    if (!keywordValue) return true;
    return (
      item.type.toLowerCase().includes(keywordValue) ||
      item.name.toLowerCase().includes(keywordValue)
    );
  });
};

const layoutItems = computed(() => filterItemsByCategory("layout"));
const pcItems = computed(() => filterItemsByCategory("uiPc"));
const chartItems = computed(() => filterItemsByCategory("chart"));

/**
 * 获取组件预览组件
 * @param {string} type - 组件类型
 * @returns {import('vue').Component}
 */
const getPreviewComponent = (type) => {
  const previewMap = {
    Button: {
      render: () =>
        h("div", { class: "preview-button" }, [
          h("span", { class: "text-xs" }, "按钮"),
        ]),
    },
    Text: {
      render: () => h("div", { class: "preview-text" }, "文本"),
    },
    FlexContainer: {
      render: () =>
        h("div", { class: "preview-flex-container" }, [
          h("div", { class: "preview-flex-item" }),
          h("div", { class: "preview-flex-item" }),
        ]),
    },
    HorizontalLayout: {
      render: () =>
        h("div", { class: "preview-flex-container" }, [
          h("div", { class: "preview-flex-item" }),
          h("div", { class: "preview-flex-item" }),
          h("div", { class: "preview-flex-item" }),
        ]),
    },
    VerticalLayout: {
      render: () =>
        h(
          "div",
          {
            class: "preview-flex-container",
            style: {
              flexDirection: "column",
              alignItems: "stretch",
              justifyContent: "space-between",
            },
          },
          [
            h("div", { class: "preview-flex-item" }),
            h("div", { class: "preview-flex-item" }),
            h("div", { class: "preview-flex-item" }),
          ],
        ),
    },
    GridContainer: {
      render: () =>
        h("div", { class: "preview-grid-container" }, [
          h("div", { class: "preview-grid-item" }),
          h("div", { class: "preview-grid-item" }),
          h("div", { class: "preview-grid-item" }),
          h("div", { class: "preview-grid-item" }),
        ]),
    },
    FreeContainer: {
      render: () => h("div", { class: "preview-free-container" }, "自由"),
    },
    ElContainer: {
      render: () =>
        h("div", { class: "preview-el-container" }, [
          h("div", { class: "preview-el-header" }),
          h("div", { class: "preview-el-body" }, [
            h("div", { class: "preview-el-aside" }),
            h("div", { class: "preview-el-main" }),
          ]),
          h("div", { class: "preview-el-footer" }),
        ]),
    },
    ResponsiveLayout: {
      render: () =>
        h("div", { class: "preview-responsive-layout" }, [
          h("div", { class: "preview-responsive-block" }),
          h("div", { class: "preview-responsive-block" }),
          h("div", { class: "preview-responsive-block" }),
        ]),
    },
    ElLayout: {
      render: () =>
        h("div", { class: "preview-el-layout" }, [
          h("div", { class: "preview-el-layout-col" }),
          h("div", { class: "preview-el-layout-col" }),
          h("div", { class: "preview-el-layout-col" }),
        ]),
    },
    ColumnLayout1: {
      render: () =>
        h("div", { class: "preview-columns preview-columns-1" }, [
          h("div", { class: "preview-columns-item" }),
        ]),
    },
    ColumnLayout2: {
      render: () =>
        h("div", { class: "preview-columns preview-columns-2" }, [
          h("div", { class: "preview-columns-item" }),
          h("div", { class: "preview-columns-item" }),
        ]),
    },
    ColumnLayout4: {
      render: () =>
        h("div", { class: "preview-columns preview-columns-4" }, [
          h("div", { class: "preview-columns-item" }),
          h("div", { class: "preview-columns-item" }),
          h("div", { class: "preview-columns-item" }),
          h("div", { class: "preview-columns-item" }),
        ]),
    },
    Table: {
      render: () =>
        h("div", { class: "preview-table" }, [
          h("div", { class: "preview-table-header" }),
          h("div", { class: "preview-table-row" }),
          h("div", { class: "preview-table-row" }),
        ]),
    },
    Tree: {
      render: () =>
        h("div", { class: "preview-tree" }, [
          h("div", { class: "preview-tree-node" }),
          h("div", { class: "preview-tree-node" }),
          h("div", { class: "preview-tree-node" }),
        ]),
    },
    Dropdown: {
      render: () =>
        h("div", { class: "preview-dropdown" }, [
          h("div", { class: "preview-dropdown-bar" }),
          h("span", { class: "preview-dropdown-arrow" }),
        ]),
    },
    Menu: {
      render: () =>
        h("div", { class: "preview-menu" }, [
          h("div", { class: "preview-menu-row" }, [
            h("span", { class: "preview-menu-dot" }),
            h("span", { class: "preview-menu-line" }),
          ]),
          h("div", { class: "preview-menu-row" }, [
            h("span", { class: "preview-menu-dot" }),
            h("span", { class: "preview-menu-line" }),
          ]),
          h("div", { class: "preview-menu-row" }, [
            h("span", { class: "preview-menu-dot" }),
            h("span", { class: "preview-menu-line" }),
          ]),
        ]),
    },
    Radio: {
      render: () =>
        h("div", { class: "preview-radio" }, [
          h("div", { class: "preview-radio-dot" }),
        ]),
    },
    Checkbox: {
      render: () =>
        h("div", { class: "preview-checkbox" }, [
          h("div", { class: "preview-checkbox-mark" }),
        ]),
    },
    Input: {
      render: () =>
        h("div", { class: "preview-input" }, [
          h("div", { class: "preview-input-line" }),
        ]),
    },
    Cascader: {
      render: () =>
        h("div", { class: "preview-cascader" }, [
          h("div", { class: "preview-cascader-node" }),
          h("span", { class: "preview-cascader-arrow" }),
        ]),
    },
    Tabs: {
      render: () =>
        h("div", { class: "preview-tabs" }, [
          h("div", { class: "preview-tabs-header" }),
          h("div", { class: "preview-tabs-body" }),
        ]),
    },
    Transfer: {
      render: () =>
        h("div", { class: "preview-transfer" }, [
          h("div", { class: "preview-transfer-panel" }),
          h("div", { class: "preview-transfer-arrow" }),
          h("div", { class: "preview-transfer-panel" }),
        ]),
    },
    Tag: {
      render: () => h("div", { class: "preview-tag" }, "TAG"),
    },
    InputNumber: {
      render: () =>
        h("div", { class: "preview-input-number" }, [
          h("div", { class: "preview-input-number-btn" }),
          h("div", { class: "preview-input-number-line" }),
          h("div", { class: "preview-input-number-btn" }),
        ]),
    },
    Timeline: {
      render: () =>
        h("div", { class: "preview-timeline" }, [
          h("div", { class: "preview-timeline-item" }),
          h("div", { class: "preview-timeline-item" }),
          h("div", { class: "preview-timeline-item" }),
        ]),
    },
    ImageCarousel: {
      render: () => h("div", { class: "preview-carousel-image" }, ""),
    },
    CarouselComponent: {
      render: () => h("div", { class: "preview-carousel" }, ""),
    },
    WebContainer: {
      render: () => h("div", { class: "preview-web-container" }, ""),
    },
    Steps: {
      render: () =>
        h("div", { class: "preview-steps" }, [
          h("div", { class: "preview-step-dot" }),
          h("div", { class: "preview-step-line" }),
          h("div", { class: "preview-step-dot" }),
        ]),
    },
    Card: {
      render: () => h("div", { class: "preview-card" }, ""),
    },
    Pagination: {
      render: () => h("div", { class: "preview-pagination" }, ""),
    },
    Collapse: {
      render: () =>
        h("div", { class: "preview-collapse" }, [
          h("div", { class: "preview-collapse-header" }),
          h("div", { class: "preview-collapse-body" }),
        ]),
    },
    BigDataTable: {
      render: () =>
        h("div", { class: "preview-table" }, [
          h("div", { class: "preview-table-header" }),
          h("div", { class: "preview-table-row" }),
          h("div", { class: "preview-table-row" }),
        ]),
    },
    BusinessCard: {
      render: () => h("div", { class: "preview-business-card" }, ""),
    },
    Barcode: {
      render: () => h("div", { class: "preview-barcode" }, ""),
    },
    Slider: {
      render: () =>
        h("div", { class: "preview-slider" }, [
          h("div", { class: "preview-slider-track" }),
          h("div", { class: "preview-slider-thumb" }),
        ]),
    },
    Calendar: {
      render: () =>
        h("div", { class: "preview-calendar" }, [
          h("div", { class: "preview-calendar-header" }),
          h("div", { class: "preview-calendar-grid" }),
        ]),
    },
    Signature: {
      render: () => h("div", { class: "preview-signature" }, ""),
    },
    Select: {
      render: () =>
        h("div", { class: "preview-select" }, [
          h("div", { class: "preview-select-line" }),
          h("span", { class: "preview-select-arrow" }),
        ]),
    },
    Image: {
      render: () =>
        h("div", { class: "preview-image" }, [
          h("div", { class: "preview-image-mountain" }),
          h("div", { class: "preview-image-sun" }),
        ]),
    },
    EChart: {
      render: () =>
        h("div", { class: "preview-chart" }, [
          h("span", { class: "preview-chart-bar" }),
          h("span", { class: "preview-chart-bar" }),
          h("span", { class: "preview-chart-bar" }),
          h("span", { class: "preview-chart-bar" }),
        ]),
    },
  };
  return (
    previewMap[type] || {
      render: () => h("div", { class: "preview-default" }, type),
    }
  );
};

/**
 * 处理拖拽开始
 * @param {{ type: string, name: string }} item - 组件项
 * @param {DragEvent} event - 拖拽事件
 */
const handleDragStart = (item, event) => {
  startDrag(item.type);
  if (!event.dataTransfer) return;

  const payload = JSON.stringify({ type: item.type });
  event.dataTransfer.effectAllowed = "copy";
  event.dataTransfer.setData("application/x-designer-component", payload);
  event.dataTransfer.setData("text/plain", item.type);

  const dragPreview = document.createElement("div");
  dragPreview.className = "drag-preview-card";
  dragPreview.innerHTML = `
    <div class="drag-preview-icon">${getPreviewIcon(item.type)}</div>
    <div class="drag-preview-name">${item.name}</div>
  `;
  dragPreview.style.cssText = `
    position: absolute;
    top: -1000px;
    left: -1000px;
    width: 120px;
    padding: 12px;
    background: white;
    border: 2px solid #409eff;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    pointer-events: none;
    z-index: 9999;
  `;
  document.body.appendChild(dragPreview);
  event.dataTransfer.setDragImage(dragPreview, 60, 30);

  setTimeout(() => {
    document.body.removeChild(dragPreview);
  }, 0);
};

/**
 * 处理鼠标拖拽开始（HTML5 drag 失效时兜底）
 * @param {{ type: string }} item - 组件项
 * @param {MouseEvent} event - 鼠标事件
 */
const handlePointerStart = (item, event) => {
  if (event.button !== 0) return;
  startDrag(item.type);
};

/**
 * 处理拖拽结束
 */
const handleDragEnd = () => {
  endDrag();
};

/**
 * 获取预览图标（简化版）
 * @param {string} type - 组件类型
 * @returns {string} HTML 字符串
 */
const getPreviewIcon = (type) => {
  const iconMap = {
    Button:
      '<div style="width:60px;height:28px;background:#409eff;border-radius:4px;"></div>',
    Text: '<div style="width:60px;height:20px;background:#606266;border-radius:2px;"></div>',
    FlexContainer:
      '<div style="width:60px;height:40px;background:#e4e7ed;border-radius:4px;display:flex;gap:4px;padding:4px;"><div style="flex:1;background:#909399;"></div><div style="flex:1;background:#909399;"></div></div>',
    GridContainer:
      '<div style="width:60px;height:40px;background:#e4e7ed;border-radius:4px;display:grid;grid-template-columns:1fr 1fr;gap:4px;padding:4px;"><div style="background:#909399;"></div><div style="background:#909399;"></div><div style="background:#909399;"></div><div style="background:#909399;"></div></div>',
    FreeContainer:
      '<div style="width:60px;height:40px;background:#f5f7fa;border:2px dashed #dcdfe6;border-radius:4px;"></div>',
    ElContainer:
      '<div style="width:60px;height:40px;border:1px solid #dcdfe6;border-radius:4px;display:flex;flex-direction:column;gap:3px;padding:4px;background:#f5f7fa;"><div style="height:6px;background:#409eff;border-radius:2px;"></div><div style="flex:1;display:flex;gap:3px;"><div style="width:12px;background:#409eff;border-radius:2px;"></div><div style="flex:1;background:#e4e7ed;border-radius:2px;"></div></div><div style="height:6px;background:#409eff;border-radius:2px;"></div></div>',
    ResponsiveLayout:
      '<div style="width:60px;height:40px;background:#f5f7fa;border:1px solid #dcdfe6;border-radius:4px;display:flex;gap:4px;padding:4px;"><div style="flex:1;background:#c0c4cc;"></div><div style="flex:1;background:#c0c4cc;"></div><div style="flex:1;background:#c0c4cc;"></div></div>',
    ElLayout:
      '<div style="width:60px;height:40px;background:#f5f7fa;border:1px solid #dcdfe6;border-radius:4px;display:flex;gap:3px;padding:4px;"><div style="flex:1;background:#409eff;border-radius:2px;"></div><div style="flex:1;background:#409eff;border-radius:2px;"></div><div style="flex:1;background:#409eff;border-radius:2px;"></div></div>',
    ColumnLayout1:
      '<div style="width:60px;height:40px;background:#f5f7fa;border:1px solid #dcdfe6;border-radius:4px;padding:4px;"><div style="width:100%;height:100%;background:#c0c4cc;"></div></div>',
    ColumnLayout2:
      '<div style="width:60px;height:40px;background:#f5f7fa;border:1px solid #dcdfe6;border-radius:4px;display:grid;grid-template-columns:1fr 1fr;gap:4px;padding:4px;"><div style="background:#c0c4cc;"></div><div style="background:#c0c4cc;"></div></div>',
    ColumnLayout4:
      '<div style="width:60px;height:40px;background:#f5f7fa;border:1px solid #dcdfe6;border-radius:4px;display:grid;grid-template-columns:1fr 1fr 1fr 1fr;gap:4px;padding:4px;"><div style="background:#c0c4cc;"></div><div style="background:#c0c4cc;"></div><div style="background:#c0c4cc;"></div><div style="background:#c0c4cc;"></div></div>',
    Table:
      '<div style="width:60px;height:40px;background:#f5f7fa;border:1px solid #dcdfe6;border-radius:4px;"><div style="height:10px;background:#e4e7ed;"></div><div style="height:8px;margin:6px 6px 0;background:#c0c4cc;"></div><div style="height:8px;margin:4px 6px 0;background:#c0c4cc;"></div></div>',
    Tree: '<div style="width:60px;height:40px;background:#f5f7fa;border:1px solid #dcdfe6;border-radius:4px;padding:6px;display:flex;flex-direction:column;gap:4px;"><div style="height:6px;background:#c0c4cc;"></div><div style="height:6px;background:#c0c4cc;"></div><div style="height:6px;background:#c0c4cc;"></div></div>',
    Dropdown:
      '<div style="width:60px;height:28px;background:#409eff;border-radius:4px;"></div>',
    Menu: '<div style="width:60px;height:40px;background:#f5f7fa;border:1px solid #dcdfe6;border-radius:4px;padding:6px;display:flex;flex-direction:column;gap:4px;"><div style="height:6px;background:#c0c4cc;"></div><div style="height:6px;background:#c0c4cc;"></div></div>',
    Radio:
      '<div style="width:60px;height:20px;border:1px solid #dcdfe6;border-radius:10px;"></div>',
    Checkbox:
      '<div style="width:18px;height:18px;border:2px solid #409eff;border-radius:4px;"></div>',
    Input:
      '<div style="width:60px;height:24px;border:1px solid #dcdfe6;border-radius:4px;"></div>',
    Cascader:
      '<div style="width:60px;height:24px;border:1px solid #dcdfe6;border-radius:4px;background:#f5f7fa;"></div>',
    Tabs: '<div style="width:60px;height:40px;border:1px solid #dcdfe6;border-radius:4px;"><div style="height:12px;background:#e4e7ed;"></div><div style="height:20px;margin:4px;background:#f5f7fa;"></div></div>',
    Transfer:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;display:flex;align-items:center;gap:4px;padding:4px;"><div style="flex:1;height:26px;border:1px solid #3b6cff;"></div><div style="width:0;height:0;border-left:4px solid transparent;border-right:4px solid transparent;border-top:6px solid #3b6cff;"></div><div style="flex:1;height:26px;border:1px solid #3b6cff;"></div></div>',
    Tag: '<div style="width:42px;height:18px;border:1px solid #3b6cff;border-radius:2px;background:#ffffff;"></div>',
    InputNumber:
      '<div style="width:60px;height:24px;border:1px solid #3b6cff;border-radius:4px;display:flex;align-items:center;gap:4px;padding:2px 4px;"><div style="width:8px;height:8px;background:#3b6cff;"></div><div style="flex:1;height:4px;background:#3b6cff;"></div><div style="width:8px;height:8px;background:#3b6cff;"></div></div>',
    Timeline:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;padding:6px;display:flex;flex-direction:column;gap:6px;"><div style="height:4px;background:#3b6cff;"></div><div style="height:4px;background:#3b6cff;"></div><div style="height:4px;background:#3b6cff;"></div></div>',
    ImageCarousel:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#ffffff;position:relative;"><div style="position:absolute;left:8px;bottom:8px;width:24px;height:12px;background:#3b6cff;"></div><div style="position:absolute;right:10px;top:8px;width:6px;height:6px;background:#3b6cff;border-radius:50%;"></div></div>',
    CarouselComponent:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#ffffff;"><div style="height:6px;background:#3b6cff;margin:6px;"></div><div style="height:6px;background:#3b6cff;margin:6px;"></div></div>',
    WebContainer:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;"><div style="height:6px;background:#3b6cff;"></div><div style="height:20px;margin:6px;border:1px solid #3b6cff;"></div></div>',
    Steps:
      '<div style="width:60px;height:24px;border:1px solid #3b6cff;border-radius:4px;display:flex;align-items:center;gap:4px;padding:0 6px;"><div style="width:6px;height:6px;background:#3b6cff;border-radius:50%;"></div><div style="flex:1;height:2px;background:#3b6cff;"></div><div style="width:6px;height:6px;background:#3b6cff;border-radius:50%;"></div></div>',
    Card: '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;"><div style="height:8px;background:#3b6cff;"></div><div style="height:16px;margin:6px;border:1px solid #3b6cff;"></div></div>',
    Pagination:
      '<div style="width:60px;height:18px;border:1px solid #3b6cff;border-radius:4px;display:flex;align-items:center;justify-content:space-around;"><div style="width:8px;height:8px;background:#3b6cff;"></div><div style="width:8px;height:8px;background:#3b6cff;"></div><div style="width:8px;height:8px;background:#3b6cff;"></div></div>',
    Collapse:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;"><div style="height:10px;background:#3b6cff;"></div><div style="height:14px;margin:6px;border:1px solid #3b6cff;"></div></div>',
    BigDataTable:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;"><div style="height:10px;background:#3b6cff;"></div><div style="height:6px;margin:6px;background:#3b6cff;"></div><div style="height:6px;margin:6px;background:#3b6cff;"></div></div>',
    BusinessCard:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;"><div style="height:8px;background:#3b6cff;"></div><div style="height:16px;margin:6px;border:1px solid #3b6cff;"></div></div>',
    Barcode:
      '<div style="width:60px;height:24px;border:1px solid #3b6cff;border-radius:4px;background:repeating-linear-gradient(90deg,#3b6cff 0 2px,transparent 2px 4px);"></div>',
    Slider:
      '<div style="width:60px;height:12px;border:1px solid #3b6cff;border-radius:6px;position:relative;"><div style="position:absolute;left:12px;top:-3px;width:8px;height:8px;background:#3b6cff;border-radius:50%;"></div></div>',
    Calendar:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;"><div style="height:8px;background:#3b6cff;"></div><div style="height:16px;margin:6px;border:1px solid #3b6cff;"></div></div>',
    Signature:
      '<div style="width:60px;height:24px;border:1px solid #3b6cff;border-radius:4px;"><div style="height:6px;margin:8px;background:#3b6cff;"></div></div>',
    Select:
      '<div style="width:60px;height:24px;border:1px solid #dcdfe6;border-radius:4px;"></div>',
    Image:
      '<div style="width:60px;height:40px;background:#f5f7fa;border:1px solid #dcdfe6;border-radius:4px;"></div>',
  };
  return (
    iconMap[type] ||
    '<div style="width:60px;height:40px;background:#f0f0f0;border-radius:4px;"></div>'
  );
};
</script>

<style>
/* 组件卡片 */
.component-card {
  display: flex;
  flex-direction: column;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
  cursor: grab;
  transition: all 0.2s;
  background: white;
}

.dark .component-card {
  background: #1a1a1a;
  border-color: #3a3a3a;
}

.component-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.2);
  transform: none;
}

.component-card:active {
  cursor: grabbing;
  transform: scale(0.98);
}

.component-panel {
  height: 100%;
  overflow: hidden;
}

.component-list {
  flex: 1;
  overflow: auto;
  padding-right: 4px;
}

.component-collapse,
.component-subcollapse {
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

.component-subcollapse .el-collapse-item__header {
  background: #f3f6ff;
  border: none;
  border-radius: 6px;
  padding: 0 10px 0 18px;
  margin: 6px 0 10px;
  height: 30px;
  font-size: 12px;
  color: #606266;
}

.component-subcollapse .el-collapse-item__content {
  padding-bottom: 12px;
}

.component-section-title,
.component-subsection-title {
  display: flex;
  align-items: center;
  gap: 6px;
}

/* 卡片预览区 */
.card-preview {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f7f8fa;
  padding: 12px 8px 6px;
}

.dark .card-preview {
  background: #2a2a2a;
}

/* 卡片信息区 */
.card-info {
  padding: 8px;
  text-align: center;
  background: white;
  border-top: 1px solid #e4e7ed;
}

.dark .card-info {
  background: #1a1a1a;
  border-top-color: #3a3a3a;
}

/* 预览组件样式 */
.preview-button {
  width: 60px;
  height: 28px;
  background: #ffffff;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #3b6cff;
  font-size: 12px;
}

.preview-text {
  width: 60px;
  height: 20px;
  background: #3b6cff;
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 12px;
}

.preview-flex-container {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border-radius: 4px;
  display: flex;
  gap: 4px;
  padding: 4px;
  border: 1px solid #3b6cff;
}

.preview-flex-item {
  flex: 1;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-grid-container {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border-radius: 4px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  padding: 4px;
  border: 1px solid #3b6cff;
}

.preview-grid-item {
  background: #3b6cff;
  border-radius: 2px;
}

.preview-free-container {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border: 2px dashed #3b6cff;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  color: #3b6cff;
}

.preview-el-container {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 4px;
}

.preview-el-header,
.preview-el-footer {
  height: 6px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-el-body {
  flex: 1;
  display: flex;
  gap: 3px;
}

.preview-el-aside {
  width: 12px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-el-main {
  flex: 1;
  background: #e4e7ed;
  border-radius: 2px;
}

.preview-responsive-layout {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  gap: 4px;
  padding: 4px;
}

.preview-el-layout {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  gap: 3px;
  padding: 4px;
}

.preview-el-layout-col {
  flex: 1;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-responsive-block {
  flex: 1;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-columns {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: grid;
  gap: 4px;
  padding: 4px;
}

.preview-columns-1 {
  grid-template-columns: 1fr;
}

.preview-columns-2 {
  grid-template-columns: 1fr 1fr;
}

.preview-columns-4 {
  grid-template-columns: 1fr 1fr 1fr 1fr;
}

.preview-columns-item {
  background: #3b6cff;
  border-radius: 2px;
}

.preview-default {
  width: 60px;
  height: 40px;
  background: #f0f0f0;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  color: #666;
}

.preview-table {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px;
}

.preview-table-header {
  height: 8px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-table-row {
  height: 6px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-tree {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px;
}

.preview-tree-node {
  height: 6px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-dropdown,
.preview-menu,
.preview-radio,
.preview-checkbox,
.preview-input,
.preview-cascader,
.preview-select {
  width: 60px;
  height: 24px;
  border-radius: 4px;
  background: #ffffff;
  border: 1px solid #3b6cff;
}

.preview-dropdown,
.preview-menu {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  color: #606266;
}

.preview-dropdown {
  gap: 4px;
}

.preview-dropdown-arrow {
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid #3b6cff;
}

.preview-dropdown-bar {
  width: 32px;
  height: 3px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-menu {
  gap: 3px;
  flex-direction: column;
  padding: 4px 6px;
  align-items: flex-start;
  justify-content: center;
}

.preview-menu-row {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
}

.preview-menu-dot {
  width: 5px;
  height: 5px;
  background: #3b6cff;
  border-radius: 50%;
}

.preview-menu-line {
  width: 100%;
  height: 3px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-radio,
.preview-checkbox {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.preview-radio {
  border-radius: 10px;
}

.preview-radio-dot {
  width: 8px;
  height: 8px;
  background: #3b6cff;
  border-radius: 50%;
}

.preview-checkbox-mark {
  width: 10px;
  height: 6px;
  border-left: 2px solid #3b6cff;
  border-bottom: 2px solid #3b6cff;
  transform: rotate(-45deg);
}

.preview-input,
.preview-select,
.preview-cascader {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 6px;
}

.preview-input-line {
  width: 36px;
  height: 3px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-select-line,
.preview-cascader-node {
  width: 30px;
  height: 3px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-select-arrow,
.preview-cascader-arrow {
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid #3b6cff;
}

.preview-tabs {
  width: 60px;
  height: 40px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  background: #ffffff;
  display: flex;
  flex-direction: column;
  padding: 4px;
  gap: 4px;
}

.preview-tabs-header {
  height: 8px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-tabs-body {
  flex: 1;
  background: white;
  border-radius: 2px;
}

.preview-image {
  width: 60px;
  height: 40px;
  background: #ffffff;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  position: relative;
}

.preview-image-mountain {
  position: absolute;
  left: 10px;
  bottom: 8px;
  width: 28px;
  height: 14px;
  background: #3b6cff;
  border-radius: 2px;
  transform: skewX(-12deg);
}

.preview-image-sun {
  position: absolute;
  right: 12px;
  top: 8px;
  width: 6px;
  height: 6px;
  background: #3b6cff;
  border-radius: 50%;
}

.preview-transfer {
  width: 60px;
  height: 40px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px;
  background: #ffffff;
}

.preview-transfer-panel {
  flex: 1;
  height: 26px;
  border: 1px solid #3b6cff;
  border-radius: 2px;
}

.preview-transfer-arrow {
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 6px solid #3b6cff;
}

.preview-tag {
  width: 42px;
  height: 18px;
  border: 1px solid #3b6cff;
  border-radius: 2px;
  color: #3b6cff;
  font-size: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.preview-input-number {
  width: 60px;
  height: 24px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 4px;
}

.preview-input-number-btn {
  width: 8px;
  height: 8px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-input-number-line {
  flex: 1;
  height: 4px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-timeline {
  width: 60px;
  height: 40px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 6px;
}

.preview-timeline-item {
  height: 4px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-carousel-image,
.preview-carousel,
.preview-web-container,
.preview-card,
.preview-business-card,
.preview-calendar {
  width: 60px;
  height: 40px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  background: #ffffff;
  position: relative;
}

.preview-carousel-image::before {
  content: "";
  position: absolute;
  left: 8px;
  bottom: 8px;
  width: 24px;
  height: 12px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-carousel-image::after {
  content: "";
  position: absolute;
  right: 10px;
  top: 8px;
  width: 6px;
  height: 6px;
  background: #3b6cff;
  border-radius: 50%;
}

.preview-carousel::before,
.preview-web-container::before,
.preview-card::before,
.preview-business-card::before,
.preview-calendar::before {
  content: "";
  position: absolute;
  left: 0;
  top: 0;
  width: 100%;
  height: 8px;
  background: #3b6cff;
  border-radius: 4px 4px 0 0;
}

.preview-web-container::after,
.preview-card::after,
.preview-business-card::after,
.preview-calendar::after {
  content: "";
  position: absolute;
  left: 6px;
  top: 14px;
  width: calc(100% - 12px);
  height: 16px;
  border: 1px solid #3b6cff;
  border-radius: 2px;
}

.preview-steps {
  width: 60px;
  height: 24px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 6px;
}

.preview-step-dot {
  width: 6px;
  height: 6px;
  background: #3b6cff;
  border-radius: 50%;
}

.preview-step-line {
  flex: 1;
  height: 2px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-pagination {
  width: 60px;
  height: 18px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: space-around;
}

.preview-pagination::before,
.preview-pagination::after {
  content: "";
  width: 8px;
  height: 8px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-collapse {
  width: 60px;
  height: 40px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  background: #ffffff;
  position: relative;
}

.preview-collapse-header {
  height: 10px;
  background: #3b6cff;
  border-radius: 4px 4px 0 0;
}

.preview-collapse-body {
  margin: 6px;
  height: 14px;
  border: 1px solid #3b6cff;
  border-radius: 2px;
}

.preview-barcode {
  width: 60px;
  height: 24px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  background: repeating-linear-gradient(
    90deg,
    #3b6cff 0 2px,
    transparent 2px 4px
  );
}

.preview-slider {
  width: 60px;
  height: 12px;
  border: 1px solid #3b6cff;
  border-radius: 6px;
  position: relative;
}

.preview-slider-track {
  position: absolute;
  left: 6px;
  right: 6px;
  top: 4px;
  height: 2px;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-slider-thumb {
  position: absolute;
  left: 12px;
  top: 1px;
  width: 8px;
  height: 8px;
  background: #3b6cff;
  border-radius: 50%;
}

.preview-calendar-header,
.preview-calendar-grid {
  position: absolute;
  left: 6px;
  right: 6px;
  border: 1px solid #3b6cff;
  border-radius: 2px;
}

.preview-calendar-header {
  top: 14px;
  height: 6px;
}

.preview-calendar-grid {
  top: 22px;
  height: 10px;
}

.preview-signature {
  width: 60px;
  height: 24px;
  border: 1px solid #3b6cff;
  border-radius: 4px;
  position: relative;
}

.preview-signature::after {
  content: "";
  position: absolute;
  left: 8px;
  right: 8px;
  top: 9px;
  height: 3px;
  background: #3b6cff;
  border-radius: 2px;
}

/* 拖拽预览卡片样式（已在 JS 中内联） */
.drag-preview-card .drag-preview-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.drag-preview-card .drag-preview-name {
  font-size: 12px;
  font-weight: 500;
  color: #303133;
  text-align: center;
}

.preview-chart {
  width: 60px;
  height: 40px;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 4px;
}

.preview-chart-bar {
  flex: 1;
  height: 60%;
  background: #3b6cff;
  border-radius: 2px;
}

.preview-chart-bar:nth-child(2) {
  height: 85%;
}

.preview-chart-bar:nth-child(3) {
  height: 45%;
}
</style>
