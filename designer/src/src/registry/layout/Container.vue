<template>
    <div class="layout-container" :class="[`layout-mode-${layoutMode}`, { 'is-drop-zone': isDropZone }]" :style="mergedStyle">
        <slot></slot>
        <div v-if="isEmpty" class="empty-hint">拖拽组件到此处</div>
    </div>
</template>

<script setup>
/**
 * Container - 布局容器组件
 * Task 3.1: 实现 Container 组件
 *
 * 支持三种布局模式：
 * - Flex: Flexbox 布局
 * - Grid: CSS Grid 布局
 * - Block: 块级布局
 */
import { computed, useAttrs } from 'vue';

// 禁用自动继承 attrs（我们手动处理 style）
defineOptions({
    inheritAttrs: false,
});

// 获取父组件传入的 attrs（包含 style）
const attrs = useAttrs();

const props = defineProps({
    layoutMode: {
        type: String,
        default: 'flex',
        validator: (value) => ['flex', 'grid', 'block'].includes(value),
    },
    // Flex 属性
    flexDirection: {
        type: String,
        default: 'row',
    },
    justifyContent: {
        type: String,
        default: 'flex-start',
    },
    alignItems: {
        type: String,
        default: 'flex-start',
    },
    flexWrap: {
        type: String,
        default: 'nowrap',
    },
    gap: {
        type: Number,
        default: 10,
    },
    // Grid 属性
    gridTemplateColumns: {
        type: String,
        default: 'repeat(3, 1fr)',
    },
    gridTemplateRows: {
        type: String,
        default: 'auto',
    },
    gridAutoFlow: {
        type: String,
        default: 'row',
    },
    // 样式属性（从 defaultProps 中接收）
    backgroundColor: {
        type: String,
        default: 'transparent',
    },
    borderWidth: {
        type: Number,
        default: 1,
    },
    borderStyle: {
        type: String,
        default: 'dashed',
    },
    borderColor: {
        type: String,
        default: '#409EFF',
    },
    borderRadius: {
        type: Number,
        default: 4,
    },
    padding: {
        type: Number,
        default: 0,
    },
    // 拖放标记
    isDropZone: {
        type: Boolean,
        default: false,
    },
    isEmpty: {
        type: Boolean,
        default: false,
    },
});

const containerStyle = computed(() => {
    const style = {};

    // 布局模式
    if (props.layoutMode === 'flex') {
        style.display = 'flex';
        style.flexDirection = props.flexDirection;
        style.justifyContent = props.justifyContent;
        style.alignItems = props.alignItems;
        style.flexWrap = props.flexWrap;
        style.gap = `${props.gap}px`;
    } else if (props.layoutMode === 'grid') {
        style.display = 'grid';
        style.gridTemplateColumns = props.gridTemplateColumns;
        style.gridTemplateRows = props.gridTemplateRows;
        style.gridAutoFlow = props.gridAutoFlow;
        style.gap = `${props.gap}px`;
    } else {
        style.display = 'block';
    }

    // 应用样式属性（边框、背景等）
    style.backgroundColor = props.backgroundColor;
    style.borderWidth = `${props.borderWidth}px`;
    style.borderStyle = props.borderStyle;
    style.borderColor = props.borderColor;
    style.borderRadius = `${props.borderRadius}px`;
    if (props.padding) {
        style.padding = `${props.padding}px`;
    }

    return style;
});

// 合并父组件传入的 style 和组件内部的 containerStyle
const mergedStyle = computed(() => {
    // attrs.style 包含父组件传入的样式
    const parentStyle = attrs.style || {};
    
    // 过滤掉定位相关的属性（这些由 ComponentWrapper 处理）
    const { position, left, top, right, bottom, width, height, zIndex, ...otherStyles } = parentStyle;
    
    // 合并样式：组件内部样式 + 父组件的非定位样式
    // Container 填充 ComponentWrapper（width: 100%, height: 100%）
    return {
        ...containerStyle.value,  // 组件内部样式（布局模式、边框）
        ...otherStyles,           // 父组件的其他样式（如果有）
        // Container 固定样式：填充父容器，relative 定位
        width: '100%',
        height: '100%',
        position: 'relative',
    };
});
</script>

<style scoped>
.layout-container {
    box-sizing: border-box;
    /* 尺寸由父组件传入的 style 决定，不再强制撑满 */
    min-width: 50px;
    min-height: 50px;
    position: relative; /* 确保 empty-hint 绝对定位基准 */
}

.layout-container.is-drop-zone {
    border: 2px dashed #4caf50 !important;
    background-color: rgba(76, 175, 80, 0.05) !important;
}

.empty-hint {
    /* 绝对定位居中 */
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #909399;
    font-size: 14px;
    pointer-events: none;
    user-select: none;
    white-space: nowrap;
    /* 确保在 flex/grid 子元素之上但不阻挡交互 */
    z-index: 1;
}
</style>
