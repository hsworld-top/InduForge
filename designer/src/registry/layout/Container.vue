<template>
    <div class="layout-container" :class="[`layout-mode-${resolvedLayoutMode}`, { 'is-drop-zone': isDropZone }]" :style="mergedStyle">
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

// 禁用自动继承 attrs（手动处理 style）
defineOptions({
    inheritAttrs: false,
});

// 获取父组件传入的 attrs（包含 style）
const attrs = useAttrs();

const props = defineProps({
    layoutMode: {
        type: String,
        default: '',
        validator: (value) => ['flex', 'grid', 'block', ''].includes(value),
    },
    layout: {
        type: String,
        default: '',
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
    gridDefaultSpan: {
        type: String,
        default: 'full',
    },
    gridFixedSpan: {
        type: Number,
        default: 6,
    },
    gridRowHeight: {
        type: Number,
        default: 40,
    },
    gridMaxRows: {
        type: Number,
        default: 0,
    },
    gridAutoFlow: {
        type: String,
        default: 'row',
    },
    justifyItems: {
        type: String,
        default: 'stretch',
    },
    alignContent: {
        type: String,
        default: 'start',
    },
    gridInteractive: {
        type: Boolean,
        default: true,
    },
    gridDragOut: {
        type: Boolean,
        default: true,
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

/**
 * 归一化 Grid 对齐值，兼容 flex-start/flex-end
 * @param {string} value - 对齐值
 * @param {string} fallback - 默认值
 * @returns {string} 归一化后的对齐值
 */
function normalizeGridAlign(value, fallback) {
    if (!value) return fallback;
    if (value === 'flex-start') return 'start';
    if (value === 'flex-end') return 'end';
    return value;
}

/**
 * 解析 Grid 列数
 * @param {string} template - gridTemplateColumns
 * @returns {number} 列数
 */
function resolveGridColumnCount(template) {
    if (typeof template !== 'string' || !template.trim()) {
        return 24;
    }
    const repeatMatch = template.match(/repeat\(\s*(\d+)\s*,/i);
    if (repeatMatch) {
        const count = Number.parseInt(repeatMatch[1], 10);
        return Number.isFinite(count) && count > 0 ? count : 24;
    }
    if (/repeat\(\s*auto-(fit|fill)/i.test(template)) {
        return 24;
    }
    const tokens = template
        .replace(/\([^)]*\)/g, ' ')
        .trim()
        .split(/\s+/)
        .filter(Boolean);
    return tokens.length > 0 ? tokens.length : 24;
}

const resolvedLayoutMode = computed(() => {
    return props.layoutMode || props.layout || 'flex';
});

const containerStyle = computed(() => {
    const style = {};

    // 布局模式
    if (resolvedLayoutMode.value === 'flex') {
        style.display = 'flex';
        style.flexDirection = props.flexDirection;
        style.justifyContent = props.justifyContent;
        style.alignItems = props.alignItems;
        style.flexWrap = props.flexWrap;
        style.gap = `${props.gap}px`;
    } else if (resolvedLayoutMode.value === 'grid') {
        style.display = 'block';
        style['--grid-columns'] = resolveGridColumnCount(props.gridTemplateColumns);
        style['--grid-row-height'] = `${props.gridRowHeight}px`;
        style['--grid-gap'] = `${props.gap}px`;
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
        ...containerStyle.value,
        ...otherStyles,
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

.layout-container.layout-mode-grid {
    background-image:
        linear-gradient(to right, rgba(64, 158, 255, 0.15) 1px, transparent 1px),
        linear-gradient(to bottom, rgba(64, 158, 255, 0.15) 1px, transparent 1px);
    background-size: calc(100% / var(--grid-columns)) var(--grid-row-height);
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
