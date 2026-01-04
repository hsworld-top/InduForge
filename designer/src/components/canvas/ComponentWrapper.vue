<template>
    <div
        ref="wrapperRef"
        :id="component.id"
        class="component-wrapper"
        :class="{
            selected: selected,
            'is-container': isContainer,
            'is-dragging': isDragging,
            'is-drop-target': isDropTarget,
        }"
        :style="wrapperStyle"
        :draggable="!component.locked && !isResizing"
        @click.capture.stop="handleClick($event)"
        @contextmenu.stop="handleContextMenu($event)"
        @dragstart.capture="handleDragStart"
        @drag.capture="handleDrag"
        @dragend.capture="handleDragEnd"
        @dragover.prevent="handleDragOver"
        @dragleave="handleDragLeave"
        @drop.prevent="handleDrop">
        <slot />
        <template v-if="selected && !component.locked">
            <span
                v-for="handle in resizeHandles"
                :key="handle"
                class="resize-handle"
                :class="`resize-${handle}`"
                draggable="false"
                @mousedown.stop.prevent="startResize(handle, $event)"
                @touchstart.stop.prevent="startResize(handle, $event)"
                @dragstart.stop.prevent="noop"
                @click.stop="noop" />
        </template>
    </div>
</template>

<script setup>
/**
 * ComponentWrapper - 组件包装器
 *
 * 职责：
 * - 包装每个组件，提供统一的交互接口
 * - 处理组件的选中、hover 状态
 * - 提供拖拽、缩放、旋转的控制点（通过 Canvas Layer）
 * - Task 4.4: 实现画布内拖拽
 *
 * Requirements:
 * - Requirement 6: 事件系统
 * - Requirement 7: 组件选择和变换
 *
 * Task: 1.2 - 创建 DomRenderer.vue (依赖组件)
 */
import { computed, ref, onUnmounted } from 'vue';
import { getComponent } from '@/registry';
import { useCanvas } from '@/composables/useCanvas';

// Props
const props = defineProps({
    /**
     * 组件数据
     */
    component: {
        type: Object,
        required: true,
    },
    /**
     * 是否选中
     */
    selected: {
        type: Boolean,
        default: false,
    },
});

// Emits
const emit = defineEmits(['select', 'update', 'resize', 'resize-end', 'contextmenu', 'dragstart', 'drag', 'dragend', 'dragover', 'dragleave', 'drop']);

const { canvasState } = useCanvas();

const wrapperRef = ref(null);
const resizeHandles = ['nw', 'n', 'ne', 'e', 'se', 's', 'sw', 'w'];
const MIN_RESIZE_SIZE = 20;
const noop = () => {};

// 拖拽状态
// drag state
const isDragging = ref(false);
const isResizing = ref(false);
const resizeState = ref(null);
const isDropTarget = ref(false);

function isInnermostWrapperEvent(event) {
    const current = event?.currentTarget;
    const target = event?.target;
    if (!current || !target) return true;
    if (typeof target.closest !== 'function') return current === target;
    return target.closest('.component-wrapper') === current;
}

/**
 * 是否为容器组件
 */
const FALLBACK_CONTAINER_TYPES = new Set([
    'Container',
    'Row',
    'Col',
    'FlexLayout',
    'Grid',
    'CenterLayout',
    'ElContainer',
    'ElRow',
    'ElCol',
    'ElMain',
    'ElAside',
    'ElHeader',
    'ElFooter',
    'ElTabs',
    'ElCard',
    'ElDialog',
    'ElDrawer',
    'Form',
]);

const isContainer = computed(() => {
    const definition = getComponent(props.component.type);
    if (definition?.container) return true;
    return FALLBACK_CONTAINER_TYPES.has(props.component.type);
});

/**
 * ComponentWrapper 的样式（定位相关）
 * 从 component.style 中提取定位属性，应用到 wrapper 上
 */
const wrapperStyle = computed(() => {
    const style = props.component.style || {};
    // 提取定位相关的属性，避免 0 被判定为 falsy
    return {
        position: style.position || 'absolute',
        left: style.left !== undefined && style.left !== null ? `${style.left}px` : undefined,
        top: style.top !== undefined && style.top !== null ? `${style.top}px` : undefined,
        width: style.width !== undefined && style.width !== null && typeof style.width === 'number' ? `${style.width}px` : style.width,
        height: style.height !== undefined && style.height !== null && typeof style.height === 'number' ? `${style.height}px` : style.height,
        zIndex: style.zIndex,
        gridColumn: typeof style.gridColumn === 'number' ? String(style.gridColumn) : style.gridColumn,
        gridColumnStart:
            typeof style.gridColumnStart === 'number' ? String(style.gridColumnStart) : style.gridColumnStart,
        gridColumnEnd: typeof style.gridColumnEnd === 'number' ? String(style.gridColumnEnd) : style.gridColumnEnd,
        gridRow: typeof style.gridRow === 'number' ? String(style.gridRow) : style.gridRow,
        gridRowStart: typeof style.gridRowStart === 'number' ? String(style.gridRowStart) : style.gridRowStart,
        gridRowEnd: typeof style.gridRowEnd === 'number' ? String(style.gridRowEnd) : style.gridRowEnd,
        gridArea: style.gridArea,
        justifySelf: style.justifySelf,
        alignSelf: style.alignSelf,
    };
});

function getEventPoint(event) {
    if (!event) return null;
    const touch = event.touches?.[0] || event.changedTouches?.[0];
    if (touch) {
        return { x: touch.clientX, y: touch.clientY };
    }
    if (typeof event.clientX === 'number' && typeof event.clientY === 'number') {
        return { x: event.clientX, y: event.clientY };
    }
    return null;
}

function cleanupResize() {
    window.removeEventListener('mousemove', handleResizeMove);
    window.removeEventListener('mouseup', handleResizeEnd);
    window.removeEventListener('touchmove', handleResizeMove);
    window.removeEventListener('touchend', handleResizeEnd);
    isResizing.value = false;
    resizeState.value = null;
}

function startResize(handle, event) {
    if (props.component.locked) return;
    const point = getEventPoint(event);
    if (!point) return;

    const zoom = canvasState.scale || 1;
    const style = props.component.style || {};
    const rect = wrapperRef.value?.getBoundingClientRect?.();

    const startWidth =
        Number.isFinite(style.width) ? style.width : rect ? rect.width / zoom : 0;
    const startHeight =
        Number.isFinite(style.height) ? style.height : rect ? rect.height / zoom : 0;
    const startLeft = Number.isFinite(style.left) ? style.left : 0;
    const startTop = Number.isFinite(style.top) ? style.top : 0;
    const position = style.position || 'absolute';
    const canUpdateLeft = position === 'absolute' || Number.isFinite(style.left);
    const canUpdateTop = position === 'absolute' || Number.isFinite(style.top);

    resizeState.value = {
        componentId: props.component.id,
        handle,
        startX: point.x,
        startY: point.y,
        startWidth,
        startHeight,
        startLeft,
        startTop,
        canUpdateLeft,
        canUpdateTop,
        zoom,
    };
    isResizing.value = true;

    window.addEventListener('mousemove', handleResizeMove);
    window.addEventListener('mouseup', handleResizeEnd);
    window.addEventListener('touchmove', handleResizeMove, { passive: false });
    window.addEventListener('touchend', handleResizeEnd);
}

function handleResizeMove(event) {
    if (!resizeState.value) return;
    const point = getEventPoint(event);
    if (!point) return;
    if (event.cancelable) event.preventDefault();

    const {
        handle,
        startX,
        startY,
        startWidth,
        startHeight,
        startLeft,
        startTop,
        canUpdateLeft,
        canUpdateTop,
        zoom,
    } = resizeState.value;

    const dx = (point.x - startX) / zoom;
    const dy = (point.y - startY) / zoom;

    let newWidth = startWidth;
    let newHeight = startHeight;
    let newLeft = startLeft;
    let newTop = startTop;

    if (handle.includes('e')) {
        newWidth = startWidth + dx;
    }
    if (handle.includes('s')) {
        newHeight = startHeight + dy;
    }
    if (handle.includes('w')) {
        newWidth = startWidth - dx;
        if (canUpdateLeft) {
            newLeft = startLeft + dx;
        }
    }
    if (handle.includes('n')) {
        newHeight = startHeight - dy;
        if (canUpdateTop) {
            newTop = startTop + dy;
        }
    }

    if (newWidth < MIN_RESIZE_SIZE) {
        if (handle.includes('w') && canUpdateLeft) {
            newLeft = startLeft + (startWidth - MIN_RESIZE_SIZE);
        }
        newWidth = MIN_RESIZE_SIZE;
    }
    if (newHeight < MIN_RESIZE_SIZE) {
        if (handle.includes('n') && canUpdateTop) {
            newTop = startTop + (startHeight - MIN_RESIZE_SIZE);
        }
        newHeight = MIN_RESIZE_SIZE;
    }

    const updates = {
        width: Math.round(newWidth),
        height: Math.round(newHeight),
    };
    if (canUpdateLeft) {
        updates.left = Math.round(newLeft);
    }
    if (canUpdateTop) {
        updates.top = Math.round(newTop);
    }

    emit('resize', props.component.id, { style: updates });
}

function handleResizeEnd() {
    if (!resizeState.value) return;
    const componentId = resizeState.value.componentId;
    cleanupResize();
    emit('resize-end', componentId);
}

onUnmounted(() => {
    cleanupResize();
});
/**
 * 处理点击事件
 * Task 5.1: 实现单选和多选
 * - 普通点击：单选
 * - Ctrl+点击：切换选中状态（多选）
 * @param {MouseEvent} event - 鼠标事件
 */
function handleClick(event) {
    // 检查是否按下 Ctrl 或 Cmd 键
    const isMultiSelect = event.ctrlKey || event.metaKey;
    
    emit('select', props.component.id, isMultiSelect);
}

/**
 * 处理右键菜单
 * Task 7.4: 实现右键菜单
 * @param {MouseEvent} event - 鼠标事件
 */
function handleContextMenu(event) {
    emit('contextmenu', event, props.component.id);
}

/**
 * 处理拖拽开始
 * Task 4.4: 实现 @dragstart 事件处理
 */
function handleDragStart(event) {
    if (!isInnermostWrapperEvent(event)) return;
    if (isResizing.value || event?.target?.closest?.('.resize-handle')) {
        event.preventDefault();
        return;
    }
    if (props.component.locked) {
        event.preventDefault();
        return;
    }

    event.stopPropagation();
    event.dataTransfer.dropEffect = 'move';
    isDragging.value = true;
    if (event.currentTarget) {
        event.currentTarget.style.outline = '2px dashed #5e7ce0';
        event.currentTarget.style.outlineOffset = '2px';
    }

    // 设置拖拽数据
    event.dataTransfer.effectAllowed = 'move';
    const rect = event.currentTarget.getBoundingClientRect();
    const offsetX = event.clientX - rect.left;
    const offsetY = event.clientY - rect.top;
    const payload = {
        id: props.component.id,
        type: props.component.type,
        source: 'canvas', // 标记来源是画布内
        offsetX,
        offsetY,
    };
    const payloadStr = JSON.stringify(payload);
    event.dataTransfer.setData('application/x-designer-component', payloadStr);
    // 兼容只有 application/json 的读取场景（画布内拖动也能被 DesignCanvas 识别）
    event.dataTransfer.setData('application/json', payloadStr);
    // 再写入 text/plain，避免部分浏览器丢失自定义 MIME 导致拖动失败
    event.dataTransfer.setData('text/plain', payloadStr);

    // 设置拖拽图像（使用当前元素的克隆）
    const dragImage = event.currentTarget.cloneNode(true);
    dragImage.style.opacity = '0.5';
    dragImage.style.position = 'absolute';
    dragImage.style.left = '-1000px';
    dragImage.style.top = '-1000px';
    dragImage.style.outline = '2px dashed #5e7ce0';
    dragImage.style.outlineOffset = '2px';
    document.body.appendChild(dragImage);
    event.dataTransfer.setDragImage(dragImage, offsetX, offsetY);
    setTimeout(() => document.body.removeChild(dragImage), 0);

    // 触发拖拽开始事件
    emit('dragstart', {
        component: props.component,
        event,
    });
}

/**
 * 处理拖拽中
 * Task 4.4: 实现 @drag 事件处理（更新位置）
 */
function handleDrag(event) {
    if (!isInnermostWrapperEvent(event)) return;
    if (isResizing.value) return;
    if (!isDragging.value) return;

    emit('drag', {
        component: props.component,
        event,
        clientX: event.clientX,
        clientY: event.clientY,
    });
}

/**
 * 处理拖拽结束
 * Task 4.4: 实现 @dragend 事件处理（保存历史）
 */
function handleDragEnd(event) {
    if (!isInnermostWrapperEvent(event)) return;
    if (isResizing.value) return;
    isDragging.value = false;
    if (event.currentTarget) {
        event.currentTarget.style.outline = '';
        event.currentTarget.style.outlineOffset = '';
    }

    emit('dragend', {
        component: props.component,
        event,
    });
}

/**
 * 处理拖拽经过容器
 * 当拖拽经过容器组件时触发
 */
function handleDragOver(event) {
    event.dataTransfer.dropEffect = event.dataTransfer.effectAllowed === 'copy' ? 'copy' : 'move';
    if (!isContainer.value) return;
    if (props.component.locked) {
        isDropTarget.value = false;
        event.dataTransfer.dropEffect = 'none';
        return;
    }

    event.preventDefault();
    // 显式允许放置，兼容库拖拽（copy）与画布内拖拽（move）
    event.dataTransfer.dropEffect = event.dataTransfer.effectAllowed === 'copy' ? 'copy' : 'move';
    
    isDropTarget.value = true;

    // 触发拖拽经过事件，用于计算插入位置
    emit('dragover', {
        container: props.component,
        event,
        clientX: event.clientX,
        clientY: event.clientY,
    });
}

/**
 * 处理拖拽离开容器
 */
function handleDragLeave(event) {
    if (!isContainer.value) return;

    // 检查是否真的离开了容器（而不是进入子元素）
    const rect = event.currentTarget.getBoundingClientRect();
    const x = event.clientX;
    const y = event.clientY;

    if (x < rect.left || x > rect.right || y < rect.top || y > rect.bottom) {
        isDropTarget.value = false;
        emit('dragleave', {
            container: props.component,
            event,
            clientX: event.clientX,
            clientY: event.clientY,
        });
    }
}

/**
 * 处理放置到容器
 * Task 4.3: 释放时添加为子组件
 */
function handleDrop(event) {
    if (!isContainer.value) return;
    if (props.component.locked) {
        isDropTarget.value = false;
        return;
    }

    isDropTarget.value = false;

    try {
        let data = event.dataTransfer.getData('application/x-designer-component');
        let source = 'canvas';

        if (!data) {
            data = event.dataTransfer.getData('application/json');
            source = 'library';
        }
        if (!data) {
            data = event.dataTransfer.getData('text/plain');
            source = 'library';
        }

        if (!data) {
            console.warn('[ComponentWrapper] No drag data found');
            return;
        }

        const dragData = JSON.parse(data);
        const resolvedSource = dragData?.source || source;

        // 只有真正向容器投递子节点时才阻止冒泡
        event.stopPropagation();

        console.log('🎯 [ComponentWrapper] Drop to container:', props.component.type, props.component.id);

        emit('drop', {
            container: props.component,
            dragData,
            source: resolvedSource, // 标记数据来源
            event,
            clientX: event.clientX,
            clientY: event.clientY,
        });
    } catch (error) {
        console.error('[ComponentWrapper] Failed to parse drop data:', error);
    }
}

// 暴露方法给父组件
defineExpose({
    // 可以暴露一些方法供父组件调用
});
</script>

<style scoped>
/**
 * 组件包装器样式
 * - 提供基础的交互样式
 * - 选中状态的视觉反馈
 */
.component-wrapper {
    /* 基础样式 */
    box-sizing: border-box;
    cursor: pointer;
    transition: outline 0.2s ease;
    user-select: none;
}

/**
 * 选中状态
 * - 显示蓝色虚线边框（临时，最终由 Canvas Layer 绘制选择框）
 */
.component-wrapper.selected {
    outline: 2px dashed #409eff;
    outline-offset: 2px;
}

/**
 * Hover 状态
 */
.component-wrapper:hover {
    outline: 1px dashed #909399;
    outline-offset: 1px;
}

/**
 * 选中时不显示 hover 效果
 */
.component-wrapper.selected:hover {
    outline: 2px dashed #409eff;
    outline-offset: 2px;
}

/* Resize handles */
.resize-handle {
    position: absolute;
    width: 8px;
    height: 8px;
    background-color: #ffffff;
    border: 1px solid #409eff;
    box-sizing: border-box;
    z-index: 2;
}

.resize-nw {
    top: -4px;
    left: -4px;
    cursor: nw-resize;
}

.resize-n {
    top: -4px;
    left: 50%;
    transform: translateX(-50%);
    cursor: n-resize;
}

.resize-ne {
    top: -4px;
    right: -4px;
    cursor: ne-resize;
}

.resize-e {
    top: 50%;
    right: -4px;
    transform: translateY(-50%);
    cursor: e-resize;
}

.resize-se {
    right: -4px;
    bottom: -4px;
    cursor: se-resize;
}

.resize-s {
    left: 50%;
    bottom: -4px;
    transform: translateX(-50%);
    cursor: s-resize;
}

.resize-sw {
    left: -4px;
    bottom: -4px;
    cursor: sw-resize;
}

.resize-w {
    top: 50%;
    left: -4px;
    transform: translateY(-50%);
    cursor: w-resize;
}


/**
 * 容器组件样式
 * - 提供视觉提示，表明这是一个容器
 */
.component-wrapper.is-container {
    min-height: 50px;
    min-width: 50px;
}

/**
 * 容器为空时的占位提示
 */
.component-wrapper.is-container:empty::before {
    content: '拖拽组件到此处';
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    min-height: 50px;
    color: #909399;
    font-size: 14px;
    border: 2px dashed #dcdfe6;
    border-radius: 4px;
    background-color: #f5f7fa;
}

/**
 * 拖拽状态样式
 * Task 4.4: 画布内拖拽
 */
.component-wrapper.is-dragging {
    opacity: 0.5;
    cursor: move;
    outline: 2px dashed #5e7ce0;
    outline-offset: 2px;
}

/**
 * 拖放目标状态（容器高亮）
 * Task 4.3: 高亮显示容器边框（绿色）
 */
.component-wrapper.is-drop-target {
    outline: 2px solid #4caf50 !important;
    outline-offset: 2px;
    background-color: rgba(76, 175, 80, 0.05);
}

</style>
