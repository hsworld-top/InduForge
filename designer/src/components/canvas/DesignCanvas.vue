<template>
    <div ref="viewportRef" class="design-canvas-viewport" :class="{ 'show-grid': showGrid }" @dragenter="handleDragEnter" @dragover="handleDragOver" @dragleave="handleDragLeave" @drop="handleDrop" @contextmenu="handleContextMenu">
        <!-- Canvas Layer (Konva) - 辅助功能层 -->
        <!-- z-index: 100, pointer-events: none -->
        <div
            class="canvas-layer"
            :style="{
                width: `${canvasWidth}px`,
                height: `${canvasHeight}px`,
            }">
            <CanvasAuxiliary
                ref="canvasAuxiliaryRef"
                :width="canvasWidth"
                :height="canvasHeight"
                :zoom="zoom"
                :scroll-x="scrollX"
                :scroll-y="scrollY"
                @canvas-click="handleCanvasClick"
                @canvas-ready="handleCanvasReady" />
        </div>

        <!-- DOM Layer (Vue Components) - 组件渲染层 -->
        <!-- z-index: 1 -->
        <!-- Task 1.5: DOM Layer 缩放 - 使用 CSS transform: scale() -->
        <div
            ref="domLayerRef"
            class="dom-layer"
            :style="{
                width: `${canvasWidth}px`,
                height: `${canvasHeight}px`,
                backgroundColor: backgroundColor,
                transform: `scale(${zoom})`,
                transformOrigin: 'top left',
            }">
            <DomRenderer
                v-if="currentPage"
                :components="components"
                :selected-id="selectedComponentId"
                :canvas-width="canvasWidth"
                :canvas-height="canvasHeight"
                @select="handleSelect"
                @update="handleUpdate"
                @contextmenu="handleContextMenu"
                @drop="handleContainerDrop" />
        </div>

        <!-- Context Menu - 右键菜单 -->
        <ContextMenu ref="contextMenuRef" :component-id="contextMenuComponentId" />
    </div>
</template>

<script setup>
/**
 * DesignCanvas - 设计画布组件（混合渲染架构）
 *
 * 架构说明：
 * - DOM Layer: 渲染所有组件和布局容器（Vue 组件 + CSS 原生布局）
 * - Canvas Layer: 渲染辅助功能（标尺、对齐线、选择框、拖拽预览）
 *
 * 层级关系：
 * - Canvas Layer (z-index: 100, pointer-events: none) - 上层
 * - DOM Layer (z-index: 1) - 下层
 *
 * Requirements:
 * - Requirement 1: 混合渲染架构
 * - Acceptance Criteria 1.1: 创建两个独立的渲染层
 * - Acceptance Criteria 1.4: 确保两层坐标系统同步且事件不冲突
 */
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue';
import { useDesignStore } from '@/store/design';
import { useCanvas } from '@/composables/useCanvas';
import { useCoordinateSync } from '@/composables/useCoordinateSync';
import DomRenderer from './DomRenderer.vue';
import CanvasAuxiliary from './CanvasAuxiliary.vue';
import ContextMenu from './ContextMenu.vue';

// Props
const props = defineProps({
    showGrid: {
        type: Boolean,
        default: true,
    },
});

// Emits
const emit = defineEmits(['contextmenu']);

// Store
const designStore = useDesignStore();

// Composables
const { canvasState } = useCanvas();

// Refs
const viewportRef = ref(null);
const canvasAuxiliaryRef = ref(null);
const domLayerRef = ref(null);
const contextMenuRef = ref(null);

// Drag state
const isDragging = ref(false);
const draggedComponent = ref(null);

// Context menu state
const contextMenuComponentId = ref(null);

// Canvas state
const zoom = ref(1);
const scrollX = ref(0);
const scrollY = ref(0);

// Zoom center point (for preserving center during zoom)
const zoomCenterX = ref(0);
const zoomCenterY = ref(0);

// 坐标系统同步
const coordinateSync = useCoordinateSync({
    canvasContainerRef: viewportRef,
    zoom,
    scrollX,
    scrollY,
});

// Computed
const currentPage = computed(() => designStore.currentPage);
const pageConfig = computed(() => designStore.pageConfig);
const components = computed(() => designStore.components);
const selectedComponentId = computed(() => designStore.selectedComponentId);

const canvasWidth = computed(() => pageConfig.value?.width || 1920);
const canvasHeight = computed(() => pageConfig.value?.height || 1080);
const backgroundColor = computed(() => pageConfig.value?.backgroundColor || '#ffffff');

/**
 * 处理组件选择
 */
/**
 * 处理组件选择
 * Task 5.1: 支持单选和多选
 * @param {string} id - 组件ID
 * @param {boolean} isMultiSelect - 是否多选模式（Ctrl+点击）
 */
function handleSelect(id, isMultiSelect = false) {
    if (isMultiSelect) {
        // 多选模式：切换选中状态
        designStore.toggleComponentSelection(id);
    } else {
        // 单选模式
        designStore.selectComponent(id);
    }
}

/**
 * 处理右键菜单
 * Task 7.4: 实现右键菜单
 * @param {MouseEvent} event - 鼠标事件
 * @param {string} componentId - 组件ID（可选）
 */
function handleContextMenu(event, componentId = null) {
    event.preventDefault();

    // 如果右键点击的是组件，且该组件未被选中，则先选中它
    if (componentId && !designStore.selectedComponentIds.includes(componentId)) {
        designStore.selectComponent(componentId);
    }

    // 设置右键菜单的组件ID
    contextMenuComponentId.value = componentId;

    // 显示右键菜单
    if (contextMenuRef.value) {
        contextMenuRef.value.show(event);
    }
}

/**
 * 处理键盘事件
 * Task 5.1: 实现全选（Ctrl+A）
 * Task 7.2: 实现撤销/重做快捷键
 * @param {KeyboardEvent} event - 键盘事件
 */
function handleKeyDown(event) {
    // Ctrl+Z 或 Cmd+Z：撤销
    if ((event.ctrlKey || event.metaKey) && event.key === 'z' && !event.shiftKey) {
        event.preventDefault();
        designStore.undo();
        console.log('↩️ Undo');
        return;
    }

    // Ctrl+Y 或 Cmd+Shift+Z：重做
    if (
        ((event.ctrlKey || event.metaKey) && event.key === 'y') ||
        ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === 'z')
    ) {
        event.preventDefault();
        designStore.redo();
        console.log('↪️ Redo');
        return;
    }

    // Ctrl+A 或 Cmd+A：全选
    if ((event.ctrlKey || event.metaKey) && event.key === 'a') {
        event.preventDefault();
        designStore.selectAllComponents();
        console.log('📋 Select all components');
    }

    // Delete 或 Backspace：删除选中的组件
    if (event.key === 'Delete' || event.key === 'Backspace') {
        if (designStore.selectedComponentIds.length > 1) {
            // 多选：批量删除
            event.preventDefault();
            designStore.batchDeleteComponents(designStore.selectedComponentIds);
            designStore.saveHistory('批量删除组件');
            console.log('🗑️ Batch delete selected components');
        } else if (designStore.selectedComponentId) {
            // 单选：删除单个
            event.preventDefault();
            designStore.deleteSelectedComponent();
            designStore.saveHistory('删除组件');
        }
    }

    // Ctrl+C 或 Cmd+C：复制
    if ((event.ctrlKey || event.metaKey) && event.key === 'c') {
        if (designStore.selectedComponentIds.length > 0) {
            event.preventDefault();
            designStore.copySelectedComponent();
            console.log('📋 Copy selected components');
        }
    }

    // Ctrl+V 或 Cmd+V：粘贴
    if ((event.ctrlKey || event.metaKey) && event.key === 'v') {
        if (designStore.selectedComponentIds.length > 1) {
            event.preventDefault();
            const newIds = designStore.batchCopyComponents(designStore.selectedComponentIds);
            designStore.saveHistory('批量复制组件');
            console.log('📋 Paste selected components');
        } else if (designStore.clipboard) {
            event.preventDefault();
            designStore.pasteComponent();
            designStore.saveHistory('粘贴组件');
        }
    }

    // Ctrl+D 或 Cmd+D：复制并粘贴
    if ((event.ctrlKey || event.metaKey) && event.key === 'd') {
        event.preventDefault();
        if (designStore.selectedComponentIds.length > 1) {
            designStore.batchCopyComponents(designStore.selectedComponentIds);
            designStore.saveHistory('批量复制组件');
        } else if (designStore.selectedComponentId) {
            designStore.duplicateSelectedComponent();
            designStore.saveHistory('复制组件');
        }
        console.log('📋 Duplicate selected components');
    }

    // Ctrl+Plus/Equal：放大
    if ((event.ctrlKey || event.metaKey) && (event.key === '+' || event.key === '=')) {
        event.preventDefault();
        const newZoom = Math.min(zoom.value + 0.1, 5); // 最大500%
        setZoom(newZoom);
        console.log(`🔍 Zoom in: ${Math.round(newZoom * 100)}%`);
    }

    // Ctrl+Minus：缩小
    if ((event.ctrlKey || event.metaKey) && event.key === '-') {
        event.preventDefault();
        const newZoom = Math.max(zoom.value - 0.1, 0.1); // 最小10%
        setZoom(newZoom);
        console.log(`🔍 Zoom out: ${Math.round(newZoom * 100)}%`);
    }

    // Ctrl+0：重置缩放（100%）
    if ((event.ctrlKey || event.metaKey) && event.key === '0') {
        event.preventDefault();
        setZoom(1);
        console.log('🔍 Reset zoom: 100%');
    }

    // Escape：取消选择
    if (event.key === 'Escape') {
        designStore.clearSelection();
        console.log('❌ Clear selection');
    }
}

/**
 * 处理组件更新
 */
function handleUpdate(id, updates) {
    designStore.updateComponent(id, updates);
    // 保存历史记录
    designStore.saveHistory(`更新组件 ${id}`);
}

/**
 * 处理 Canvas 空白区域点击
 */
function handleCanvasClick(event) {
    console.log('Canvas blank area clicked:', event);
    // 取消选择所有组件
    designStore.selectComponent(null);
}

/**
 * 处理 Canvas 准备就绪
 */
function handleCanvasReady(canvasLayers) {
    console.log('✅ Canvas layers ready:', canvasLayers);
    // 可以在这里保存 Canvas 图层的引用，用于后续的辅助功能渲染
    // 例如：绘制标尺、对齐线、选择框等

    // Task 1.4: 初始化坐标系统同步
    initCoordinateSync();
}

/**
 * 处理拖拽进入画布
 * Task 4.1: 在 Canvas Layer 显示拖拽预览
 */
function handleDragEnter(event) {
    event.preventDefault();
    console.log('Drag enter canvas');
}

/**
 * 处理拖拽在画布上移动
 * Task 4.1: 实现拖拽预览跟随鼠标
 */
function handleDragOver(event) {
    event.preventDefault();

    if (!isDragging.value || !draggedComponent.value) {
        // 尝试从 dataTransfer 获取拖拽数据
        try {
            const data = event.dataTransfer.getData('application/json');
            if (data) {
                draggedComponent.value = JSON.parse(data);
                isDragging.value = true;
            }
        } catch (error) {
            // 无法获取数据，可能是浏览器安全限制
        }
    }

    // 更新拖拽预览位置
    if (isDragging.value && canvasAuxiliaryRef.value) {
        const dragPreview = canvasAuxiliaryRef.value.getDragPreview();
        if (dragPreview) {
            // 计算鼠标在画布中的位置（考虑缩放和滚动）
            const rect = viewportRef.value.getBoundingClientRect();
            const x = (event.clientX - rect.left - 40 + scrollX.value) / zoom.value;
            const y = (event.clientY - rect.top - 40 + scrollY.value) / zoom.value;

            // 如果预览未显示，先显示
            if (!dragPreview.isVisible && draggedComponent.value) {
                dragPreview.show(draggedComponent.value, x, y);
            } else {
                dragPreview.updatePosition(x, y);
            }
        }
    }
}

/**
 * 处理拖拽离开画布
 */
function handleDragLeave(event) {
    // 只在真正离开画布时隐藏预览
    if (!event.currentTarget.contains(event.relatedTarget)) {
        hideDragPreview();
    }
}

/**
 * 处理放置到画布
 * Task 4.2: 实现画布接收拖放
 */
function handleDrop(event) {
    event.preventDefault();
    event.stopPropagation(); // 阻止事件冒泡，防止 DesignCenter 也处理此事件

    try {
        // 获取拖拽数据
        const data = event.dataTransfer.getData('application/json');
        if (!data) {
            console.warn('No drag data found');
            return;
        }

        const component = JSON.parse(data);

        // 计算放置位置（画布坐标系）
        const rect = viewportRef.value.getBoundingClientRect();
        const x = (event.clientX - rect.left - 40 + scrollX.value) / zoom.value;
        const y = (event.clientY - rect.top - 40 + scrollY.value) / zoom.value;

        // 更新组件位置
        component.style = {
            ...component.style,
            left: x,
            top: y,
            position: 'absolute',
        };
        
        // 如果是容器组件且宽度是百分比字符串，计算实际像素值
        if (component.type === 'Container' && typeof component.style.width === 'string' && component.style.width.includes('%')) {
            const percentage = parseFloat(component.style.width) / 100;
            const canvasWidth = pageConfig.value?.width || 1920;
            component.style.width = Math.round(canvasWidth * percentage - 80); // 减去左右 padding
        }

        // 添加组件到 Store
        designStore.addComponent(component);
        designStore.saveHistory(`添加组件 ${component.name}`);

        console.log('✅ Component dropped:', component);
    } catch (error) {
        console.error('❌ Failed to drop component:', error);
    } finally {
        // 清理拖拽状态
        hideDragPreview();
        isDragging.value = false;
        draggedComponent.value = null;
    }
}

/**
 * 隐藏拖拽预览
 */
function hideDragPreview() {
    if (canvasAuxiliaryRef.value) {
        const dragPreview = canvasAuxiliaryRef.value.getDragPreview();
        if (dragPreview) {
            dragPreview.hide();
        }
    }
}

/**
 * 处理放置到容器
 * Task 4.3: 实现拖拽到容器内
 */
function handleContainerDrop(payload) {
    try {
        const { container, dragData, source, event } = payload;
        
        console.log('✨ Drop to container:', container.type, container.id);
        console.log('  Drag data:', dragData);
        console.log('  Source:', source);

        let component;
        
        if (source === 'library') {
            // 从组件库拖拽，dragData 已经是完整的组件实例
            component = dragData;
            
            // 对于布局容器，子组件使用相对定位
            if (container.type === 'Container' || container.type === 'FlexLayout' || container.type === 'Grid') {
                component.style = {
                    ...component.style,
                    position: 'relative',
                    left: 'auto',
                    top: 'auto',
                };
            }
        } else {
            // 从画布内移动，dragData 只包含 id 和 type
            // TODO: 实现画布内组件移动到容器的逻辑
            console.log('  Moving component from canvas to container (not implemented yet)');
            return;
        }

        // 添加组件到容器
        designStore.addComponent(component, container.id);
        designStore.saveHistory(`添加组件 ${component.type} 到容器`);

        console.log('✅ Component added to container');
    } catch (error) {
        console.error('❌ Failed to drop component to container:', error);
    } finally {
        // 清理拖拽状态
        hideDragPreview();
        isDragging.value = false;
        draggedComponent.value = null;
    }
}

/**
 * 初始化坐标系统同步
 * Task 1.4: 实现坐标系统同步
 */
function initCoordinateSync() {
    if (!domLayerRef.value) {
        console.warn('[DesignCanvas] DOM Layer not ready for coordinate sync');
        return;
    }

    // 获取所有组件 ID
    const componentIds = components.value.map((comp) => comp.id);

    // 初始化所有组件的边界
    coordinateSync.updateAllComponentBounds(componentIds);

    // 监听组件尺寸变化
    coordinateSync.observeComponentResize(componentIds, (componentId, bounds) => {
        console.log(`[CoordinateSync] Component ${componentId} resized:`, bounds);
        // TODO: 更新 Canvas Layer 的选择框、对齐线等
    });

    // 监听组件位置变化
    coordinateSync.observeComponentPosition(domLayerRef.value, (componentId, bounds) => {
        console.log(`[CoordinateSync] Component ${componentId} moved:`, bounds);
        // TODO: 更新 Canvas Layer 的选择框、对齐线等
    });

    console.log('✅ Coordinate sync initialized');
}

/**
 * 更新坐标同步（当组件列表变化时）
 */
function updateCoordinateSync() {
    if (!domLayerRef.value) return;

    const componentIds = components.value.map((comp) => comp.id);

    // 重新监听组件
    coordinateSync.observeComponentResize(componentIds, (componentId, bounds) => {
        console.log(`[CoordinateSync] Component ${componentId} resized:`, bounds);
    });

    // 更新所有组件边界
    coordinateSync.updateAllComponentBounds(componentIds);
}

// Watch components changes to update coordinate sync
watch(
    components,
    () => {
        nextTick(() => {
            updateCoordinateSync();
        });
    },
    { deep: true },
);

// Lifecycle
onMounted(async () => {
    console.log('✅ DesignCanvas mounted - Hybrid Rendering Architecture');
    console.log('  - DOM Layer: Rendering components and layout containers');
    console.log('  - Canvas Layer: Auxiliary features (rulers, guides, selection box)');

    // ✅ Task 1.3 - 初始化 Konva Stage 和 Layer (完成)
    // ✅ Task 1.4 - 实现坐标系统同步 (完成)
    // TODO: Task 1.5 - 实现缩放和滚动同步

    // 监听滚动事件
    if (viewportRef.value) {
        viewportRef.value.addEventListener('scroll', handleScroll);
    }

    // Task 5.1: 监听键盘事件（全选）
    window.addEventListener('keydown', handleKeyDown);
});

onUnmounted(() => {
    console.log('👋 DesignCanvas unmounted');

    // 清理滚动事件监听器
    if (viewportRef.value) {
        viewportRef.value.removeEventListener('scroll', handleScroll);
    }

    // 清理键盘事件监听器
    window.removeEventListener('keydown', handleKeyDown);
});

/**
 * 处理滚动事件
 */
function handleScroll(event) {
    if (!viewportRef.value) return;

    scrollX.value = viewportRef.value.scrollLeft;
    scrollY.value = viewportRef.value.scrollTop;
}

/**
 * 设置缩放比例（保持中心点不变）
 * Task 1.5: 实现缩放中心点保持
 *
 * 功能说明：
 * - 在缩放时保持画布的视觉中心点不变
 * - 计算缩放前后的中心点偏移，并调整滚动位置
 * - 确保用户看到的画布区域在缩放前后保持一致
 *
 * Requirements:
 * - Requirement 21: 缩放功能
 * - Acceptance Criteria 21.4: 缩放后保持画布中心点不变
 *
 * @param {number} newZoom - 新的缩放比例 (0.1 ~ 5.0)
 * @param {number} centerX - 缩放中心点 X 坐标（相对于视口，可选）
 * @param {number} centerY - 缩放中心点 Y 坐标（相对于视口，可选）
 */
function setZoom(newZoom, centerX = null, centerY = null) {
    if (!viewportRef.value) {
        console.warn('[DesignCanvas] Viewport not ready for zoom');
        return;
    }

    // 限制缩放范围 (10% ~ 500%)
    const clampedZoom = Math.max(0.1, Math.min(5.0, newZoom));

    if (clampedZoom === zoom.value) {
        return; // 缩放比例未变化，无需处理
    }

    const oldZoom = zoom.value;
    const viewport = viewportRef.value;

    // 如果未指定缩放中心点，使用视口中心
    const viewportWidth = viewport.clientWidth;
    const viewportHeight = viewport.clientHeight;

    const zoomCenterXPos = centerX !== null ? centerX : viewportWidth / 2;
    const zoomCenterYPos = centerY !== null ? centerY : viewportHeight / 2;

    // 计算缩放中心点在画布坐标系中的位置（缩放前）
    const canvasX = (viewport.scrollLeft + zoomCenterXPos) / oldZoom;
    const canvasY = (viewport.scrollTop + zoomCenterYPos) / oldZoom;

    // 更新缩放比例
    zoom.value = clampedZoom;

    // 等待 DOM 更新后调整滚动位置
    nextTick(() => {
        // 计算新的滚动位置，使得缩放中心点在画布坐标系中的位置保持不变
        const newScrollLeft = canvasX * clampedZoom - zoomCenterXPos;
        const newScrollTop = canvasY * clampedZoom - zoomCenterYPos;

        // 更新滚动位置
        viewport.scrollLeft = newScrollLeft;
        viewport.scrollTop = newScrollTop;

        // 更新滚动状态（触发 Canvas Layer 同步）
        scrollX.value = newScrollLeft;
        scrollY.value = newScrollTop;

        console.log(`🔍 Zoom updated: ${oldZoom.toFixed(2)} → ${clampedZoom.toFixed(2)}`);
        console.log(`  Center point preserved at canvas (${canvasX.toFixed(1)}, ${canvasY.toFixed(1)})`);
        console.log(`  Scroll adjusted: (${newScrollLeft.toFixed(1)}, ${newScrollTop.toFixed(1)})`);
    });
}

// 暴露方法给父组件
defineExpose({
    canvasAuxiliaryRef,
    zoom,
    scrollX,
    scrollY,
    setZoom,
    coordinateSync,
    updateCoordinateSync,
    hideDragPreview,
});
</script>

<style scoped>
/**
 * 设计画布视口
 * - 提供滚动容器
 * - 显示背景网格（可选）
 * - 包含 Canvas Layer 和 DOM Layer
 */
.design-canvas-viewport {
    width: 100%;
    height: 100%;
    overflow: auto;
    background-color: #f5f5f5;
    display: flex;
    align-items: flex-start;
    justify-content: flex-start;
    padding: 40px;
    position: relative;
}

/**
 * 背景网格（可选）
 * - 20px x 20px 网格
 * - 浅灰色线条
 */
.design-canvas-viewport.show-grid {
    background-image: linear-gradient(to right, #e0e0e0 1px, transparent 1px), linear-gradient(to bottom, #e0e0e0 1px, transparent 1px);
    background-size: 20px 20px;
}

/**
 * Canvas Layer (Konva)
 * - 位于 DOM Layer 上方
 * - z-index: 100
 * - pointer-events: none（不阻挡 DOM 事件）
 * - 用于渲染辅助功能：标尺、对齐线、选择框、拖拽预览
 */
.canvas-layer {
    position: absolute;
    top: 40px;
    left: 40px;
    pointer-events: none;
    /* Konva Stage 将在这里初始化 */
}

/**
 * DOM Layer (Vue Components)
 * - 位于 Canvas Layer 下方
 * - z-index: 1
 * - 渲染所有组件和布局容器
 * - 使用 Vue 组件 + CSS 原生布局
 */
.dom-layer {
    position: relative;
    z-index: 1;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}
</style>
