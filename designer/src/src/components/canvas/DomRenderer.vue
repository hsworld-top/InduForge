<template>
    <div class="dom-renderer">
        <ComponentWrapper
            v-for="comp in components"
            :key="comp.id"
            :component="comp"
            :selected="comp.id === selectedId"
            @select="handleSelect"
            @update="handleUpdate"
            @contextmenu="handleContextMenu"
            @dragstart="handleDragStart"
            @drag="handleDrag"
            @dragend="handleDragEnd"
            @dragover="handleDragOver"
            @dragleave="handleDragLeave"
            @drop="handleDrop">
            <!-- 动态渲染组件 -->
            <component
                :is="getComponentType(comp.type)"
                v-bind="comp.props"
                :draggable="!comp.locked"
                :style="convertStyle(comp.style)"
                :is-empty="!comp.children || comp.children.length === 0">
                <!-- 递归渲染子组件 -->
                <DomRenderer
                    v-if="comp.children && comp.children.length > 0"
                    :components="comp.children"
                    :selected-id="selectedId"
                    @select="handleSelect"
                    @update="handleUpdate"
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
import { computed } from 'vue';
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
const emit = defineEmits(['select', 'update', 'contextmenu', 'dragstart', 'drag', 'dragend', 'dragover', 'dragleave', 'drop']);

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
    const layoutKeys = ['position', 'left', 'top', 'right', 'bottom', 'zIndex'];
    layoutKeys.forEach((key) => {
        if (key in style) {
            delete style[key];
        }
    });

    return style;
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
</style>
