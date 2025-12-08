<template>
    <div ref="canvasContainerRef" class="canvas-auxiliary">
        <!-- Konva Stage 将在这里初始化 -->
        <!-- 用于渲染：标尺、对齐线、选择框、拖拽预览 -->
    </div>
</template>

<script setup>
/**
 * CanvasAuxiliary - Canvas 辅助功能组件
 *
 * 职责：
 * - 初始化 Konva Stage 和 Layer
 * - 渲染辅助功能：标尺、对齐线、选择框、拖拽预览
 * - 处理 Canvas Layer 的缩放和滚动
 * - 处理 Canvas Layer 的事件（空白区域点击）
 *
 * Requirements:
 * - Requirement 4: Canvas 辅助功能
 * - Acceptance Criteria 4.1: 渲染水平和垂直标尺
 * - Acceptance Criteria 4.2: 实时显示对齐辅助线
 * - Acceptance Criteria 4.3: 自动吸附并显示对齐线
 * - Acceptance Criteria 4.4: 显示选择框和控制点
 * - Acceptance Criteria 4.5: 显示拖拽预览
 * - Acceptance Criteria 4.6: 显示选择框矩形
 * - Acceptance Criteria 4.7: 同步缩放标尺和辅助线
 */
import { ref, onMounted, onUnmounted, watch } from 'vue';
import Konva from 'konva';
import { DragPreview } from '@/engine/canvas/DragPreview';
import SelectionBox from '@/engine/canvas/SelectionBox';
import AlignmentGuides from '@/engine/canvas/AlignmentGuides';
import SelectionRect from '@/engine/canvas/SelectionRect';
import InsertLine from '@/engine/canvas/InsertLine';
import { useDesignStore } from '@/store/design';

// Props
const props = defineProps({
    width: {
        type: Number,
        required: true,
    },
    height: {
        type: Number,
        required: true,
    },
    zoom: {
        type: Number,
        default: 1,
    },
    scrollX: {
        type: Number,
        default: 0,
    },
    scrollY: {
        type: Number,
        default: 0,
    },
});

// Emits
const emit = defineEmits(['canvas-click', 'canvas-ready']);

// Store
const designStore = useDesignStore();

// Refs
const canvasContainerRef = ref(null);

// Konva 实例
let stage = null;
let mainLayer = null;
let rulerLayer = null;
let guideLayer = null;
let selectionLayer = null;

// 辅助功能实例
let dragPreview = null;
let selectionBox = null;
let alignmentGuides = null;
let selectionRect = null;
let insertLine = null;

/**
 * 初始化 Konva Stage 和 Layer
 * Task 1.3 Sub-task: 初始化 Konva Stage 和 Layer
 */
function initKonvaStage() {
    if (!canvasContainerRef.value) {
        console.error('❌ Canvas container not found');
        return;
    }

    try {
        // 创建 Konva Stage
        stage = new Konva.Stage({
            container: canvasContainerRef.value,
            width: props.width,
            height: props.height,
        });

        // 创建主图层（用于一般辅助元素）
        mainLayer = new Konva.Layer();
        stage.add(mainLayer);

        // 创建标尺图层
        rulerLayer = new Konva.Layer();
        stage.add(rulerLayer);

        // 创建辅助线图层（对齐线、插入线）
        guideLayer = new Konva.Layer();
        stage.add(guideLayer);

        // 创建选择图层（选择框、控制点、拖拽预览）
        selectionLayer = new Konva.Layer();
        stage.add(selectionLayer);

        // 处理空白区域点击事件
        // Task 1.3 Sub-task: 实现 Canvas Layer 的事件处理（空白区域点击）
        stage.on('click', (e) => {
            // 只处理点击 Stage 本身（空白区域）
            if (e.target === stage) {
                emit('canvas-click', {
                    x: e.evt.clientX,
                    y: e.evt.clientY,
                });
            }
        });

        // 初始化辅助功能
        dragPreview = new DragPreview(selectionLayer);
        
        // Phase 5 集成：初始化选择框、对齐辅助线、框选矩形和插入线
        selectionBox = new SelectionBox(selectionLayer, {
            onResizeStart: handleResizeStart,
            onResize: handleResize,
            onResizeEnd: handleResizeEnd,
            onRotateStart: handleRotateStart,
            onRotate: handleRotate,
            onRotateEnd: handleRotateEnd,
        });
        
        alignmentGuides = new AlignmentGuides(guideLayer, {
            threshold: 5,
            strokeColor: '#ff4757',
        });
        
        selectionRect = new SelectionRect(selectionLayer, {
            fillColor: 'rgba(64, 158, 255, 0.1)',
            strokeColor: '#409eff',
        });
        
        insertLine = new InsertLine(guideLayer);

        console.log('✅ Konva Stage initialized successfully');
        console.log('  - Main Layer: General auxiliary elements');
        console.log('  - Ruler Layer: Horizontal and vertical rulers');
        console.log('  - Guide Layer: Alignment guides and insert lines');
        console.log('  - Selection Layer: Selection box, handles, drag preview');
        console.log('  - Phase 5: SelectionBox, AlignmentGuides, SelectionRect, InsertLine');
        console.log('  - selectionBox instance:', selectionBox);
        console.log('  - selectionBox.update exists:', typeof selectionBox?.update);

        // 通知父组件 Canvas 已准备好
        emit('canvas-ready', {
            stage,
            mainLayer,
            rulerLayer,
            guideLayer,
            selectionLayer,
            dragPreview,
        });
    } catch (error) {
        console.error('❌ Failed to initialize Konva Stage:', error);
    }
}

/**
 * 更新 Stage 尺寸
 */
function updateStageSize() {
    if (!stage) return;

    stage.width(props.width);
    stage.height(props.height);
    stage.batchDraw();
}

/**
 * 更新 Stage 缩放
 * Task 1.5: 实现 Canvas Layer 缩放（stage.scale()）
 *
 * 功能说明：
 * - 根据 zoom prop 更新 Konva Stage 的缩放比例
 * - 使用 stage.scale() 方法设置 x 和 y 方向的缩放
 * - 调用 batchDraw() 批量重绘以优化性能
 *
 * Requirements:
 * - Requirement 21: 缩放功能
 * - Acceptance Criteria 21.3: Canvas Layer 缩放时更新 Konva Stage 的 scale 属性
 */
function updateStageZoom() {
    if (!stage) return;

    // 设置 Stage 的缩放比例
    stage.scale({ x: props.zoom, y: props.zoom });

    // 批量重绘以优化性能
    stage.batchDraw();

    console.log(`🔍 Canvas Layer zoom updated: ${props.zoom}`);
}

/**
 * 更新 Stage 滚动位置
 * Task 1.3 Sub-task: 实现 Canvas Layer 的滚动
 */
function updateStageScroll() {
    if (!stage) return;

    stage.position({
        x: -props.scrollX,
        y: -props.scrollY,
    });
    stage.batchDraw();
}

/**
 * Phase 5: 缩放事件处理
 */
function handleResizeStart() {
    console.log('🔧 Resize start');
}

function handleResize(newSize) {
    // 更新选中组件的尺寸
    if (designStore.selectedComponentId) {
        designStore.updateComponent(designStore.selectedComponentId, {
            style: {
                width: Math.round(newSize.width),
                height: Math.round(newSize.height),
            },
        });
    }
}

function handleResizeEnd() {
    console.log('✅ Resize end');
    // 可以在这里保存历史记录
}

/**
 * Phase 5: 旋转事件处理
 */
function handleRotateStart() {
    console.log('🔄 Rotate start');
}

function handleRotate(newRotation) {
    // 更新选中组件的旋转角度
    if (designStore.selectedComponentId) {
        designStore.updateComponent(designStore.selectedComponentId, {
            style: {
                rotate: Math.round(newRotation),
            },
        });
    }
}

function handleRotateEnd() {
    console.log('✅ Rotate end');
    // 可以在这里保存历史记录
}

/**
 * Phase 5: 更新选择框
 * 监听选中组件变化，更新选择框位置
 */
function updateSelectionBox() {
    if (!selectionBox) {
        console.warn('⚠️ updateSelectionBox called but selectionBox is not initialized');
        return;
    }

    if (typeof selectionBox.update !== 'function') {
        console.error('❌ selectionBox.update is not a function', selectionBox);
        return;
    }

    const selectedIds = designStore.selectedComponentIds || [];
    
    if (selectedIds.length === 0) {
        selectionBox.hide();
        return;
    }

    // 获取选中组件的 DOM 元素
    const domElements = selectedIds
        .map((id) => document.getElementById(id))
        .filter(Boolean);

    if (domElements.length > 0) {
        selectionBox.update(domElements, props.zoom, {
            x: props.scrollX,
            y: props.scrollY,
        });
    } else {
        selectionBox.hide();
    }
}

/**
 * 清理 Konva Stage
 */
function destroyKonvaStage() {
    // 清理辅助功能实例
    if (dragPreview) {
        dragPreview.destroy();
        dragPreview = null;
    }
    
    if (selectionBox) {
        selectionBox.destroy();
        selectionBox = null;
    }
    
    if (alignmentGuides) {
        alignmentGuides.destroy();
        alignmentGuides = null;
    }
    
    if (selectionRect) {
        selectionRect.destroy();
        selectionRect = null;
    }
    
    if (insertLine) {
        insertLine.destroy();
        insertLine = null;
    }

    if (stage) {
        stage.destroy();
        stage = null;
        mainLayer = null;
        rulerLayer = null;
        guideLayer = null;
        selectionLayer = null;
        console.log('👋 Konva Stage destroyed');
    }
}

// Watch props changes
watch(() => props.width, updateStageSize);
watch(() => props.height, updateStageSize);
watch(() => props.zoom, updateStageZoom);
watch(() => [props.scrollX, props.scrollY], updateStageScroll);

// Phase 5: 监听选中状态变化，更新选择框
watch(
    () => designStore.selectedComponentIds,
    () => {
        // 确保 selectionBox 已初始化
        if (selectionBox) {
            updateSelectionBox();
        }
    },
    { deep: true }
);

// 监听组件位置变化（通过 MutationObserver 或定时器）
// 这里使用简单的定时器方案，生产环境可以优化为 MutationObserver
let selectionUpdateInterval = null;

// Lifecycle
onMounted(() => {
    console.log('🎨 CanvasAuxiliary mounted - Initializing Konva Stage...');
    initKonvaStage();
    
    // 初始化完成后启动选择框更新定时器
    selectionUpdateInterval = setInterval(() => {
        if (designStore.selectedComponentIds.length > 0 && selectionBox) {
            updateSelectionBox();
        }
    }, 100); // 每100ms更新一次
});

onUnmounted(() => {
    console.log('👋 CanvasAuxiliary unmounted - Cleaning up...');
    
    // 清理定时器
    if (selectionUpdateInterval) {
        clearInterval(selectionUpdateInterval);
    }
    
    // 清理 Konva
    destroyKonvaStage();
});

// Expose methods to parent component
defineExpose({
    stage,
    mainLayer,
    rulerLayer,
    guideLayer,
    selectionLayer,
    dragPreview,
    selectionBox,
    alignmentGuides,
    selectionRect,
    insertLine,
    getStage: () => stage,
    getMainLayer: () => mainLayer,
    getRulerLayer: () => rulerLayer,
    getGuideLayer: () => guideLayer,
    getSelectionLayer: () => selectionLayer,
    getDragPreview: () => dragPreview,
    getSelectionBox: () => selectionBox,
    getAlignmentGuides: () => alignmentGuides,
    getSelectionRect: () => selectionRect,
    getInsertLine: () => insertLine,
    updateSelectionBox, // 暴露更新方法
});
</script>

<style scoped>
/**
 * Canvas 辅助功能容器
 * - 用于承载 Konva Stage
 * - pointer-events: none（不阻挡 DOM 事件）
 * - 位于 DOM Layer 上方
 */
.canvas-auxiliary {
    width: 100%;
    height: 100%;
    pointer-events: none;
}

/**
 * 允许 Konva Stage 内的交互元素接收事件
 * 例如：选择框的控制点、拖拽预览等
 */
.canvas-auxiliary :deep(canvas) {
    pointer-events: auto;
}
</style>
