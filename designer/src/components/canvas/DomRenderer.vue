<template>
    <div class="dom-renderer">
        <ComponentWrapper
            v-for="comp in components"
            :key="comp.id"
            :component="comp"
            :selected="comp.id === selectedId"
            @select="handleSelect"
            @update="handleUpdate"
            @resize="handleResize"
            @resize-end="handleResizeEnd"
            @contextmenu="handleContextMenu"
            @dragstart="handleDragStart"
            @drag="handleDrag"
            @dragend="handleDragEnd"
            @dragover="handleDragOver"
            @dragleave="handleDragLeave"
            @drop="handleDrop">
            <!-- Grid 容器渲染 -->
            <component
                :is="getComponentType(comp.type)"
                v-bind="comp.props"
                :draggable="!comp.locked"
                :style="resolveComponentStyle(comp)"
                :is-empty="!comp.children || comp.children.length === 0">
                <GridLayout
                    v-if="shouldUseGridLayout(comp)"
                    class="grid-layout"
                    :ref="setGridLayoutRef(comp.id)"
                    :auto-size="resolveGridAutoSize(comp)"
                    :layout="getGridLayoutItems(comp)"
                    :col-num="resolveGridColumnCount(comp)"
                    :row-height="resolveGridRowHeight(comp)"
                    :margin="[resolveGridGap(comp), resolveGridGap(comp)]"
                    :is-draggable="resolveGridInteractive(comp)"
                    :is-resizable="resolveGridInteractive(comp)"
                    :vertical-compact="false"
                    :use-css-transforms="true"
                    @layout-mounted="(layout) => handleGridLayoutUpdated(comp, layout, true)"
                    @layout-updated="(layout) => handleGridLayoutUpdated(comp, layout)">
                    <GridItem
                        v-for="item in getGridLayoutChildren(comp)"
                        :key="item.component.id"
                        :x="item.layout.x"
                        :y="item.layout.y"
                        :w="item.layout.w"
                        :h="item.layout.h"
                        :i="item.component.id">
                        <ComponentWrapper
                            :component="item.component"
                            :selected="item.component.id === selectedId"
                            :interaction-disabled="resolveGridInteractive(comp)"
                            :drag-out-enabled="resolveGridDragOut(comp)"
                            layout-mode="grid"
                            @select="handleSelect"
                            @update="handleUpdate"
                            @resize="handleResize"
                            @resize-end="handleResizeEnd"
                            @contextmenu="handleContextMenu"
                            @dragstart="handleDragStart"
                            @drag="handleDrag"
                            @dragend="handleDragEnd"
                            @dragover="handleDragOver"
                            @dragleave="handleDragLeave"
                            @drop="handleDrop">
                            <component
                                :is="getComponentType(item.component.type)"
                                v-bind="item.component.props"
                                :draggable="!item.component.locked && (!resolveGridInteractive(comp) || resolveGridDragOut(comp))"
                                :style="resolveGridChildStyle(item.component, comp)"
                                :is-empty="!item.component.children || item.component.children.length === 0">
                                <DomRenderer
                                    v-if="item.component.children && item.component.children.length > 0"
                                    :components="item.component.children"
                                    :selected-id="selectedId"
                                    @select="handleSelect"
                                    @update="handleUpdate"
                                    @resize="handleResize"
                                    @resize-end="handleResizeEnd"
                                    @contextmenu="handleContextMenu"
                                    @dragstart="handleDragStart"
                                    @drag="handleDrag"
                                    @dragend="handleDragEnd"
                                    @dragover="handleDragOver"
                                    @dragleave="handleDragLeave"
                                    @drop="handleDrop" />
                            </component>
                        </ComponentWrapper>
                    </GridItem>
                </GridLayout>
                <!-- 子组件渲染 -->
                <DomRenderer
                    v-else-if="comp.children && comp.children.length > 0"
                    :components="comp.children"
                    :selected-id="selectedId"
                    @select="handleSelect"
                    @update="handleUpdate"
                    @resize="handleResize"
                    @resize-end="handleResizeEnd"
                    @contextmenu="handleContextMenu"
                    @dragstart="handleDragStart"
                    @drag="handleDrag"
                    @dragend="handleDragEnd"
                    @dragover="handleDragOver"
                    @dragleave="handleDragLeave"
                    @drop="handleDrop" />
            </component>
        </ComponentWrapper>
    </div>
</template>

<script setup>
/**
 * DomRenderer - DOM 渲染器组件
 *
 * 职责：
 * - 递归渲染组件树
 * - 动态加载组件（<component :is="...">）
 * - 管理组件选中状态
 * - 传递组件更新事件
 *
 * Requirements:
 * - Requirement 2: DOM 布局容器渲染
 * - Requirement 3: DOM 业务组件渲染
 * - Requirement 12: 组件注册机制
 *
 * Task: 1.2 - 创建 DomRenderer.vue
 * Sub-tasks:
 * - [x] 实现递归渲染组件树
 * - [x] 实现动态组件加载
 * - [x] 实现组件选中状态管理
 * - [x] 实现组件更新事件传递
 */
import { computed, onBeforeUnmount, reactive } from 'vue';
import { GridLayout, GridItem } from 'vue3-grid-layout';
import ComponentWrapper from './ComponentWrapper.vue';
import { getComponent } from '@/registry';
import { convertDslStyleToCss } from '@/utils/styleConverter';

// Props
const props = defineProps({
    /**
     * 组件列表
     */
    components: {
        type: Array,
        required: true,
        default: () => [],
    },
    /**
     * 选中的组件 ID
     */
    selectedId: {
        type: String,
        default: null,
    },
});

// Emits
const emit = defineEmits(['select', 'update', 'resize', 'resize-end', 'contextmenu', 'dragstart', 'drag', 'dragend', 'dragover', 'dragleave', 'drop']);

const DEFAULT_GRID_COLUMNS = 24;
const DEFAULT_GRID_ROW_HEIGHT = 40;

const gridHeights = reactive({});
const gridObservers = new Map();

/**
 * 绑定 GridLayout 引用，用于计算容器高度
 * @param {string} id - 容器 ID
 * @returns {Function} ref 回调
 */
function setGridLayoutRef(id) {
    return (instance) => {
        if (!id) return;
        const existing = gridObservers.get(id);
        if (existing) {
            existing.disconnect();
            gridObservers.delete(id);
        }
        if (!instance) {
            delete gridHeights[id];
            return;
        }
        const element = instance?.$el || instance;
        if (!element) return;
        if (typeof ResizeObserver === 'undefined') {
            const rect = element.getBoundingClientRect?.();
            if (rect) {
                gridHeights[id] = rect.height;
            }
            return;
        }
        const observer = new ResizeObserver((entries) => {
            entries.forEach((entry) => {
                gridHeights[id] = entry.contentRect.height;
            });
        });
        observer.observe(element);
        gridObservers.set(id, observer);
        const rect = element.getBoundingClientRect?.();
        if (rect) {
            gridHeights[id] = rect.height;
        }
    };
}

onBeforeUnmount(() => {
    gridObservers.forEach((observer) => observer.disconnect());
    gridObservers.clear();
});

/**
 * 获取组件类型
 * 从组件注册表中获取组件定义，返回 Vue 组件
 * 如果组件未注册，返回默认占位符
 *
 * Task 2.4: 实现 getComponentType 方法
 * - 集成 ComponentFactory
 * - 处理未注册组件（显示占位符）
 * - 添加组件加载错误处理
 *
 * @param {string} type - 组件类型
 * @returns {Component|string} Vue 组件或占位符
 */
function getComponentType(type) {
    try {
        const definition = getComponent(type);

        if (!definition) {
            console.warn(`[DomRenderer] Component type "${type}" not found in registry. Using placeholder.`);
            // 返回占位符组件
            return createPlaceholderComponent(type);
        }

        if (!definition.component) {
            console.warn(`[DomRenderer] Component "${type}" has no component implementation. Using placeholder.`);
            return createPlaceholderComponent(type);
        }

        // 返回 Vue 组件
        return definition.component;
    } catch (error) {
        console.error(`[DomRenderer] Error loading component "${type}":`, error);
        return createPlaceholderComponent(type, error.message);
    }
}

/**
 * 创建占位符组件
 * 用于显示未注册或加载失败的组件
 *
 * @param {string} type - 组件类型
 * @param {string} errorMessage - 错误信息（可选）
 * @returns {Object} Vue 组件对象
 */
function createPlaceholderComponent(type, errorMessage = null) {
    return {
        name: 'ComponentPlaceholder',
        props: {},
        template: `
      <div class="component-placeholder">
        <div class="placeholder-icon">⚠️</div>
        <div class="placeholder-type">{{ type }}</div>
        <div v-if="errorMessage" class="placeholder-error">{{ errorMessage }}</div>
        <div class="placeholder-hint">Component not found</div>
      </div>
    `,
        data() {
            return {
                type,
                errorMessage,
            };
        },
    };
}

/**
 * 转换 DSL 样式为 CSS 样式
 *
 * @param {Object} dslStyle - DSL 样式对象
 * @returns {Object} CSS 样式对象
 */
function convertStyle(dslStyle) {
    if (!dslStyle) return {};
    const style = convertDslStyleToCss(dslStyle);

    // Remove layout-related fields because ComponentWrapper already handles positioning
    const layoutKeys = [
        'position',
        'left',
        'top',
        'right',
        'bottom',
        'zIndex',
        'gridColumn',
        'gridColumnStart',
        'gridColumnEnd',
        'gridRow',
        'gridRowStart',
        'gridRowEnd',
        'gridArea',
        'justifySelf',
        'alignSelf',
    ];
    layoutKeys.forEach((key) => {
        if (key in style) {
            delete style[key];
        }
    });

    return style;
}

function resolveComponentStyle(component) {
    const style = convertStyle(component?.style);
    if (shouldUseGridLayout(component)) {
        return {
            ...style,
            '--grid-row-height': `${resolveGridRowHeight(component)}px`,
        };
    }
    return style;
}

/**
 * Grid 子组件样式：去掉固定宽高，填满格子
 * @param {Object} component - 子组件
 * @param {Object} container - 容器组件
 * @returns {Object} 样式对象
 */
function resolveGridChildStyle(component, container) {
    const style = convertStyle(component?.style);
    if (!shouldUseGridLayout(container)) {
        return style;
    }
    const nextStyle = { ...style };
    nextStyle.width = '100%';
    nextStyle.height = '100%';
    nextStyle.minWidth = '100%';
    nextStyle.minHeight = '100%';
    delete nextStyle.maxWidth;
    delete nextStyle.maxHeight;
    return nextStyle;
}

/**
 * 鍒ゆ柇鏄惁浣跨敤 GridLayout 娓叉煋
 * @param {Object} component - 缁勪欢瀵硅薄
 * @returns {boolean} 鏄惁浣跨敤 GridLayout
 */
function shouldUseGridLayout(component) {
    if (!component) return false;
    if (component.type !== 'Container') return false;
    const layoutMode = component.props?.layoutMode || component.props?.layout;
    return layoutMode === 'grid';
}

/**
 * 瑙ｆ瀽 Grid 鍒楁暟
 * @param {Object} container - 瀹瑰櫒缁勪欢
 * @returns {number} 鍒楁暟
 */
function resolveGridColumnCount(container) {
    const template = container?.props?.gridTemplateColumns;
    if (typeof template !== 'string' || !template.trim()) {
        return DEFAULT_GRID_COLUMNS;
    }
    const repeatMatch = template.match(/repeat\(\s*(\d+)\s*,/i);
    if (repeatMatch) {
        const count = Number.parseInt(repeatMatch[1], 10);
        return Number.isFinite(count) && count > 0 ? count : DEFAULT_GRID_COLUMNS;
    }
    if (/repeat\(\s*auto-(fit|fill)/i.test(template)) {
        return DEFAULT_GRID_COLUMNS;
    }
    const tokens = template
        .replace(/\([^)]*\)/g, ' ')
        .trim()
        .split(/\s+/)
        .filter(Boolean);
    return tokens.length > 0 ? tokens.length : DEFAULT_GRID_COLUMNS;
}

/**
 * 瑙ｆ瀽 Grid 琛屽楂?
 * @param {Object} container - 瀹瑰櫒缁勪欢
 * @returns {number} 琛屽楂?
 */
function resolveGridRowCount(container) {
    const template = container?.props?.gridTemplateRows;
    if (typeof template !== 'string' || !template.trim()) {
        return null;
    }
    if (template.trim() === 'auto') {
        return null;
    }
    const repeatMatch = template.match(/repeat\(\s*(\d+)\s*,/i);
    if (repeatMatch) {
        const count = Number.parseInt(repeatMatch[1], 10);
        return Number.isFinite(count) && count > 0 ? count : null;
    }
    if (/repeat\(\s*auto-(fit|fill)/i.test(template)) {
        return null;
    }
    const tokens = template
        .replace(/\([^)]*\)/g, ' ')
        .trim()
        .split(/\s+/)
        .filter(Boolean);
    return tokens.length > 0 ? tokens.length : null;
}

/**
 * 解析 Grid 最大行数
 * @param {Object} container - 容器组件
 * @returns {number|null} 最大行数
 */
function resolveGridMaxRows(container) {
    const maxRows = Number(container?.props?.gridMaxRows);
    return Number.isFinite(maxRows) && maxRows > 0 ? maxRows : null;
}

/**
 * 计算行高分配的行数
 * @param {Object} container - 容器组件
 * @returns {number|null} 行数
 */
function resolveGridDistributionRows(container) {
    const templateRows = resolveGridRowCount(container);
    const maxRows = resolveGridMaxRows(container);
    if (Number.isFinite(templateRows)) {
        return maxRows ? Math.min(templateRows, maxRows) : templateRows;
    }
    return maxRows || null;
}

/**
 * 解析 Grid 容器高度
 * @param {Object} container - 容器组件
 * @returns {number|null} 高度
 */
function resolveGridContainerHeight(container) {
    const height = container?.style?.height;
    if (Number.isFinite(height)) return height;
    if (typeof height === 'string') {
        const match = height.trim().match(/^(\d+(?:\.\d+)?)px$/i);
        if (match) {
            const value = Number.parseFloat(match[1]);
            return Number.isFinite(value) ? value : null;
        }
    }
    const measured = gridHeights?.[container?.id];
    return Number.isFinite(measured) ? measured : null;
}

/**
 * 计算当前 Grid 使用的行数
 * @param {Object} container - 容器组件
 * @returns {number} 行数
 */
function resolveGridUsedRows(container) {
    const items = getGridLayoutItems(container);
    return items.reduce((max, item) => Math.max(max, item.y + item.h), 0);
}

/**
 * 是否需要自动扩展高度
 * @param {Object} container - 容器组件
 * @returns {boolean} 是否自动扩展
 */
function resolveGridAutoSize(container) {
    const maxRows = resolveGridMaxRows(container);
    if (!maxRows) return false;
    return resolveGridUsedRows(container) > maxRows;
}

/**
 * 解析 Grid 行高
 * @param {Object} container - 容器组件
 * @returns {number} 行高
 */
function resolveGridRowHeight(container) {
    const rowHeight = Number(container?.props?.gridRowHeight);
    const fallback = Number.isFinite(rowHeight) && rowHeight > 0 ? rowHeight : DEFAULT_GRID_ROW_HEIGHT;
    const rowCount = resolveGridDistributionRows(container);
    if (!rowCount || rowCount <= 0) return fallback;
    const containerHeight = resolveGridContainerHeight(container);
    if (!Number.isFinite(containerHeight) || containerHeight <= 0) return fallback;
    const gap = resolveGridGap(container);
    const available = containerHeight - gap * (rowCount - 1);
    if (!Number.isFinite(available) || available <= 0) return fallback;
    return available / rowCount;
}

function resolveGridGap(container) {
    const gap = Number(container?.props?.gap);
    return Number.isFinite(gap) ? gap : 0;
}

/**
 * 瑙ｆ瀽 Grid 鎷栨嫿/缂╂斁鏄惁鍚敤
 * @param {Object} container - 瀹瑰櫒缁勪欢
 * @returns {boolean} 鏄惁鍚敤
 */
function resolveGridInteractive(container) {
    const value = container?.props?.gridInteractive;
    return typeof value === 'boolean' ? value : true;
}

/**
 * 瑙ｆ瀽 Grid 鎷栨嫿鍑哄®¹å™¨æ¨¡å¼
 * @param {Object} container - 瀹瑰櫒缁勪欢
 * @returns {boolean} 鏄惁鍏佽®?
 */
function resolveGridDragOut(container) {
    const value = container?.props?.gridDragOut;
    return typeof value === 'boolean' ? value : true;
}

function resolveGridDefaultSpan(container, columns) {
    const mode = container?.props?.gridDefaultSpan || 'full';
    if (mode === 'fixed') {
        const fixed = Number(container?.props?.gridFixedSpan);
        const span = Number.isFinite(fixed) ? fixed : 1;
        return Math.min(Math.max(1, span), columns);
    }
    return columns;
}

function hasGridSpan(style = {}, axis) {
    const gridKey = axis === 'row' ? 'gridRow' : 'gridColumn';
    const startKey = axis === 'row' ? 'gridRowStart' : 'gridColumnStart';
    const endKey = axis === 'row' ? 'gridRowEnd' : 'gridColumnEnd';
    return Boolean(style[gridKey] || style[startKey] || style[endKey]);
}

/**
 * 瑙ｆ瀽 Grid 浜岀淮瀛楁
 * @param {string|number} value - 浼犲叆鍊?
 * @returns {number|null} 瑙ｆ瀽鍊?
 */
function resolveGridLineValue(value) {
    if (Number.isFinite(value)) return value;
    if (typeof value === 'string') {
        const match = value.trim().match(/^(\d+)$/);
        if (match) {
            return Number.parseInt(match[1], 10);
        }
    }
    return null;
}

/**
 * 瑙ｆ瀽 Grid 鍒楀紑濮嬩綅
 * @param {Object} style - 缁勪欢鏍峰紡
 * @param {string} axis - column | row
 * @returns {number|null} 寮€濮嬩綅
 */
function resolveGridStart(style, axis) {
    const startKey = axis === 'row' ? 'gridRowStart' : 'gridColumnStart';
    const gridKey = axis === 'row' ? 'gridRow' : 'gridColumn';
    const startValue = resolveGridLineValue(style?.[startKey]);
    if (Number.isFinite(startValue)) return startValue;
    if (typeof style?.[gridKey] === 'string' && style[gridKey].includes('/')) {
        const [startText] = style[gridKey].split('/').map((item) => item.trim());
        const parsed = Number.parseInt(startText, 10);
        return Number.isFinite(parsed) ? parsed : null;
    }
    return null;
}

/**
 * 瑙ｆ瀽 Grid 鍗犵敤鏁伴噺
 * @param {Object} style - 缁勪欢鏍峰紡
 * @param {string} axis - column | row
 * @param {number} total - 鎬绘暟
 * @returns {number} 鍗犵敤鏁伴噺
 */
function resolveGridSpan(style, axis, total) {
    const startKey = axis === 'row' ? 'gridRowStart' : 'gridColumnStart';
    const endKey = axis === 'row' ? 'gridRowEnd' : 'gridColumnEnd';
    const gridKey = axis === 'row' ? 'gridRow' : 'gridColumn';
    let span = null;

    if (typeof style?.[gridKey] === 'string') {
        const trimmed = style[gridKey].trim();
        const spanMatch = trimmed.match(/span\s+(\d+)/i);
        if (spanMatch) {
            span = Number.parseInt(spanMatch[1], 10);
        } else if (trimmed.includes('/')) {
            const [startText, endText] = trimmed.split('/').map((item) => item.trim());
            const start = Number.parseInt(startText, 10);
            const endSpanMatch = endText?.match(/span\s+(\d+)/i);
            const end = Number.parseInt(endText, 10);
            if (Number.isFinite(start) && endSpanMatch) {
                span = Number.parseInt(endSpanMatch[1], 10);
            } else if (Number.isFinite(start) && Number.isFinite(end)) {
                span = Math.max(1, end - start);
            }
        }
    }

    if (!Number.isFinite(span)) {
        const start = resolveGridLineValue(style?.[startKey]);
        const end = resolveGridLineValue(style?.[endKey]);
        if (Number.isFinite(start) && Number.isFinite(end)) {
            span = Math.max(1, end - start);
        }
    }

    if (!Number.isFinite(span) && typeof style?.[endKey] === 'string') {
        const endSpanMatch = style[endKey].match(/span\s+(\d+)/i);
        if (endSpanMatch) {
            span = Number.parseInt(endSpanMatch[1], 10);
        }
    }

    if (!Number.isFinite(span) || span <= 0) {
        span = axis === 'row' ? 1 : total;
    }

    if (axis !== 'row') {
        return Math.min(Math.max(1, span), total);
    }
    return Math.max(1, span);
}

/**
 * 构建 GridLayout 子项布局
 * @param {Object} component - 子组件
 * @param {number} columns - 总列数
 * @param {number} index - 默认顺序索引
 * @returns {Object} GridLayout 布局数据
 */
function buildGridItemLayout(component, container, columns, index) {
    const style = component?.style || {};
    const columnStart = resolveGridStart(style, 'column');
    const rowStart = resolveGridStart(style, 'row');
    const columnSpan = hasGridSpan(style, 'column')
        ? resolveGridSpan(style, 'column', columns)
        : resolveGridDefaultSpan(container, columns);
    const rowSpan = hasGridSpan(style, 'row') ? resolveGridSpan(style, 'row', 1) : 1;

    const x = Number.isFinite(columnStart) ? Math.max(0, columnStart - 1) : 0;
    const y = Number.isFinite(rowStart) ? Math.max(0, rowStart - 1) : index;
    const maxWidth = Math.max(1, columns - x);

    return {
        i: component.id,
        x,
        y,
        w: Math.min(columnSpan, maxWidth),
        h: rowSpan,
    };
}

/**
 * 组装 GridLayout 子项数据
 * @param {Object} container - 容器组件
 * @returns {Array} 子项数据
 */
function getGridLayoutChildren(container) {
    const children = Array.isArray(container?.children) ? container.children : [];
    const columns = resolveGridColumnCount(container);
    return children.map((child, index) => ({
        component: child,
        layout: buildGridItemLayout(child, container, columns, index),
    }));
}

/**
 * 获取 GridLayout 布局数组
 * @param {Object} container - 容器组件
 * @returns {Array} 布局数组
 */
function getGridLayoutItems(container) {
    return getGridLayoutChildren(container).map((item) => item.layout);
}

/**
 * 比较 Grid 样式是否一致
 * @param {Object} current - 当前样式
 * @param {Object} nextStyle - 新样式
 * @returns {boolean} 是否一致
 */
function isSameGridStyle(current = {}, nextStyle = {}) {
    return (
        current.gridColumnStart === nextStyle.gridColumnStart &&
        current.gridColumnEnd === nextStyle.gridColumnEnd &&
        current.gridRowStart === nextStyle.gridRowStart &&
        current.gridRowEnd === nextStyle.gridRowEnd &&
        current.position === nextStyle.position &&
        current.left === nextStyle.left &&
        current.top === nextStyle.top
    );
}

/**
 * 处理 GridLayout 布局更新
 * @param {Object} container - 容器组件
 * @param {Array} layout - 布局数组
 */
function handleGridLayoutUpdated(container, layout = [], force = false) {
    if (!container || !Array.isArray(layout)) return;
    if (!force && !resolveGridInteractive(container)) return;
    const layoutMap = new Map(layout.map((item) => [String(item.i), item]));
    const children = Array.isArray(container.children) ? container.children : [];

    children.forEach((child) => {
        const item = layoutMap.get(String(child.id));
        if (!item) return;
        const nextStyle = {
            ...child.style,
            position: 'relative',
            left: null,
            top: null,
            gridColumn: null,
            gridRow: null,
            gridColumnStart: item.x + 1,
            gridColumnEnd: item.x + item.w + 1,
            gridRowStart: item.y + 1,
            gridRowEnd: item.y + item.h + 1,
        };
        if (!isSameGridStyle(child.style || {}, nextStyle)) {
            emit('update', child.id, { style: nextStyle });
        }
    });
}

/**
 * 处理组件选择
 *
 * @param {string} id - 组件 ID
 */
/**
 * 处理选择事件
 * Task 5.1: 支持单选和多选
 * @param {string} id - 组件ID
 * @param {boolean} isMultiSelect - 是否多选模式
 */
function handleSelect(id, isMultiSelect = false) {
    emit('select', id, isMultiSelect);
}

/**
 * 处理右键菜单
 * Task 7.4: 实现右键菜单
 * @param {MouseEvent} event - 鼠标事件
 * @param {string} componentId - 组件ID
 */
function handleContextMenu(event, componentId) {
    emit('contextmenu', event, componentId);
}

/**
 * 处理组件更新
 *
 * @param {string} id - 组件 ID
 * @param {Object} updates - 更新内容
 */
function handleUpdate(id, updates) {
    emit('update', id, updates);
}

/**
 * 处理组件缩放
 *
 * @param {string} id - 组件 ID
 * @param {Object} updates - 更新内容
 */
function handleResize(id, updates) {
    emit('resize', id, updates);
}

/**
 * 处理组件缩放结束
 *
 * @param {string} id - 组件 ID
 */
function handleResizeEnd(id) {
    emit('resize-end', id);
}

/**
 * 处理拖拽开始
 * Task 4.4: 转发拖拽事件到父组件
 */
function handleDragStart(payload) {
    emit('dragstart', payload);
}

/**
 * 处理拖拽中
 */
function handleDrag(payload) {
    emit('drag', payload);
}

/**
 * 处理拖拽结束
 */
function handleDragEnd(payload) {
    emit('dragend', payload);
}

/**
 * 处理拖拽经过容器
 */
function handleDragOver(payload) {
    emit('dragover', payload);
}

/**
 * 处理拖拽离开容器
 */
function handleDragLeave(payload) {
    emit('dragleave', payload);
}

/**
 * 处理放置到容器
 */
function handleDrop(payload) {
    emit('drop', payload);
}

// 暴露方法给父组件
defineExpose({
    // 可以暴露一些方法供父组件调用
});
</script>

<style scoped>
/**
 * DOM 渲染器容器
 * - 使用相对定位，作为子组件的定位上下文
 * - 占满整个画布区域
 */
.dom-renderer {
    position: relative;
    width: 100%;
    height: 100%;
}

/**
 * 组件占位符样式
 * Task 2.4: 处理未注册组件的情况（显示占位符）
 */
:deep(.component-placeholder) {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 20px;
    background-color: #fff3cd;
    border: 2px dashed #ffc107;
    border-radius: 4px;
    color: #856404;
    min-height: 100px;
    text-align: center;
}

:deep(.placeholder-icon) {
    font-size: 32px;
    margin-bottom: 8px;
}

:deep(.placeholder-type) {
    font-weight: bold;
    font-size: 14px;
    margin-bottom: 4px;
}

:deep(.placeholder-error) {
    font-size: 12px;
    color: #dc3545;
    margin-bottom: 4px;
}

:deep(.placeholder-hint) {
    font-size: 12px;
    opacity: 0.7;
}

.grid-layout {
    width: 100%;
    height: 100%;
}

</style>
