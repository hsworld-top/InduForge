<template>
  <div class="flex flex-col gap-3">
    <el-input v-model="keyword" size="small" placeholder="搜索组件" clearable />
    <div v-if="filteredCategories.length" class="flex flex-col gap-4">
      <div v-for="category in filteredCategories" :key="category.key">
        <div class="text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">
          {{ category.label }}
        </div>
        <!-- 卡片式网格布局 -->
        <div class="grid grid-cols-2 gap-2">
          <div
            v-for="item in category.items"
            :key="item.type"
            class="component-card"
            draggable="true"
            @dragstart="handleDragStart(item, $event)"
          >
            <!-- 卡片预览区 -->
            <div class="card-preview">
              <component :is="getPreviewComponent(item.type)" />
            </div>
            <!-- 卡片信息 -->
            <div class="card-info">
              <div class="text-xs font-medium text-gray-800 dark:text-gray-200">
                {{ item.name }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="text-sm text-gray-400 text-center py-6">
      暂无可用组件
    </div>
  </div>
</template>

<script setup>
import { computed, ref, h } from "vue";
import { componentRegistry } from "@/editor-core";

/**
 * 搜索关键字
 */
const keyword = ref("");

/**
 * 组件分类映射（仅保留布局、基础、表单）
 */
const categoryLabels = {
  layout: "布局",
  basic: "基础",
  form: "表单",
};

/**
 * 允许的分类（过滤其他分类）
 */
const allowedCategories = ["layout", "basic", "form"];

/**
 * 过滤后的组件分类列表（仅显示允许的分类）
 */
const filteredCategories = computed(() => {
  const results = [];
  const keywordValue = keyword.value.trim().toLowerCase();
  const categories = componentRegistry.getCategories();

  for (const category of categories) {
    // ✅ 只显示允许的分类
    if (!allowedCategories.includes(category)) continue;

    const items = componentRegistry
      .getByCategory(category)
      .filter((item) => {
        if (!keywordValue) return true;
        return (
          item.type.toLowerCase().includes(keywordValue) ||
          item.name.toLowerCase().includes(keywordValue)
        );
      });

    if (items.length) {
      results.push({
        key: category,
        label: categoryLabels[category] || category,
        items,
      });
    }
  }

  return results;
});

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
  };
  return previewMap[type] || { render: () => h("div", { class: "preview-default" }, type) };
};

/**
 * 处理拖拽开始
 * @param {{ type: string, name: string }} item - 组件项
 * @param {DragEvent} event - 拖拽事件
 */
const handleDragStart = (item, event) => {
  if (!event.dataTransfer) return;

  const payload = JSON.stringify({ type: item.type });
  event.dataTransfer.effectAllowed = "copy";
  event.dataTransfer.setData("application/x-designer-component", payload);
  event.dataTransfer.setData("text/plain", item.type);

  // ✅ 创建拖拽预览（卡片样式）
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

  // 拖拽结束后移除预览元素
  setTimeout(() => {
    document.body.removeChild(dragPreview);
  }, 0);
};

/**
 * 获取预览图标（简化版）
 * @param {string} type - 组件类型
 * @returns {string} HTML 字符串
 */
const getPreviewIcon = (type) => {
  const iconMap = {
    Button: '<div style="width:60px;height:28px;background:#409eff;border-radius:4px;"></div>',
    Text: '<div style="width:60px;height:20px;background:#606266;border-radius:2px;"></div>',
    FlexContainer:
      '<div style="width:60px;height:40px;background:#e4e7ed;border-radius:4px;display:flex;gap:4px;padding:4px;"><div style="flex:1;background:#909399;"></div><div style="flex:1;background:#909399;"></div></div>',
    GridContainer:
      '<div style="width:60px;height:40px;background:#e4e7ed;border-radius:4px;display:grid;grid-template-columns:1fr 1fr;gap:4px;padding:4px;"><div style="background:#909399;"></div><div style="background:#909399;"></div><div style="background:#909399;"></div><div style="background:#909399;"></div></div>',
    FreeContainer:
      '<div style="width:60px;height:40px;background:#f5f7fa;border:2px dashed #dcdfe6;border-radius:4px;"></div>',
  };
  return iconMap[type] || '<div style="width:60px;height:40px;background:#f0f0f0;border-radius:4px;"></div>';
};
</script>

<style scoped>
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
  transform: translateY(-2px);
}

.component-card:active {
  cursor: grabbing;
  transform: scale(0.98);
}

/* 卡片预览区 */
.card-preview {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
  padding: 8px;
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
  background: #409eff;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 12px;
}

.preview-text {
  width: 60px;
  height: 20px;
  background: #606266;
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
  background: #e4e7ed;
  border-radius: 4px;
  display: flex;
  gap: 4px;
  padding: 4px;
}

.preview-flex-item {
  flex: 1;
  background: #909399;
  border-radius: 2px;
}

.preview-grid-container {
  width: 60px;
  height: 40px;
  background: #e4e7ed;
  border-radius: 4px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  padding: 4px;
}

.preview-grid-item {
  background: #909399;
  border-radius: 2px;
}

.preview-free-container {
  width: 60px;
  height: 40px;
  background: #f5f7fa;
  border: 2px dashed #dcdfe6;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  color: #909399;
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
</style>
