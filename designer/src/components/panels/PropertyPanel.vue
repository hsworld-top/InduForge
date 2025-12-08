<template>
    <div class="property-panel">
        <!-- 空状态 -->
        <div v-if="!selectedComponent && selectedComponents.length === 0" class="empty-state">
            <el-empty description="请选择组件" :image-size="60" />
        </div>

        <!-- 多选状态 -->
        <div v-else-if="selectedComponents.length > 1" class="multi-select-state">
            <div class="panel-header">
                <span class="component-type">已选择 {{ selectedComponents.length }} 个组件</span>
            </div>

            <el-tabs v-model="activeTab" class="panel-tabs">
                <!-- 通用样式编辑 -->
                <el-tab-pane label="样式" name="style">
                    <el-scrollbar>
                        <PositionEditor :style="commonStyle" @change="handleBatchStyleChange" />
                        <SpacingEditor :style="commonStyle" @change="handleBatchStyleChange" />
                        <TransformEditor :style="commonStyle" @change="handleBatchStyleChange" />
                    </el-scrollbar>
                </el-tab-pane>
            </el-tabs>
        </div>

        <!-- 单选状态 -->
        <template v-else-if="selectedComponent">
            <!-- 组件信息 -->
            <div class="panel-header">
                <span class="component-type">{{ componentDef?.name || selectedComponent.type }}</span>
                <span class="component-id">{{ selectedComponent.id.slice(0, 8) }}</span>
            </div>

            <!-- 标签页 -->
            <el-tabs v-model="activeTab" class="panel-tabs">
                <!-- 属性标签页 -->
                <el-tab-pane label="属性" name="props">
                    <el-scrollbar>
                        <!-- 基础属性 -->
                        <div class="section-group">
                            <div class="section-title">基础</div>
                            <el-form label-position="left" label-width="80px" size="small">
                                <el-form-item label="标签">
                                    <el-input v-model="componentLabel" @change="handleLabelChange" />
                                </el-form-item>
                                <el-form-item label="可见">
                                    <el-switch v-model="componentVisible" @change="handleVisibleChange" />
                                </el-form-item>
                                <el-form-item label="锁定">
                                    <el-switch v-model="componentLocked" @change="handleLockedChange" />
                                </el-form-item>
                            </el-form>
                        </div>

                        <!-- 组件专有属性编辑器 -->
                        <component
                            v-if="componentEditorName"
                            :is="componentEditorName"
                            :props="selectedComponent.props"
                            :style="selectedComponent.style"
                            @change-props="handlePropsChange"
                            @change-style="handleStyleObjectChange" />

                        <!-- 布局属性编辑器 -->
                        <FlexEditor v-if="isFlexContainer" :props="selectedComponent.props" @change="handlePropsChange" />
                        <GridEditor v-if="isGridContainer" :props="selectedComponent.props" @change="handlePropsChange" />

                        <!-- 通用属性 - 使用 PropsEditor 组件 -->
                        <div v-if="propsSchema && Object.keys(propsSchema).length && !componentEditorName" class="section-group">
                            <div class="section-title">组件属性</div>
                            <PropsEditor :props="selectedComponent.props" :props-schema="propsSchema" @change="handlePropChange" />
                        </div>
                    </el-scrollbar>
                </el-tab-pane>

                <!-- 样式标签页 -->
                <el-tab-pane label="样式" name="style">
                    <el-scrollbar>
                        <PositionEditor :style="selectedComponent.style" @change="handleStyleObjectChange" />
                        <SpacingEditor :style="selectedComponent.style" @change="handleStyleObjectChange" />
                        <TransformEditor :style="selectedComponent.style" @change="handleStyleObjectChange" />
                        <!-- 旧的通用样式编辑器（兜底） -->
                        <StyleEditor v-if="selectedComponent" :style="selectedComponent.style" @change="handleStyleChange" />
                    </el-scrollbar>
                </el-tab-pane>
            </el-tabs>
        </template>
    </div>
</template>

<script setup>
/**
 * PropertyPanel - 属性面板组件（重构版）
 * Task 6.1: 重构 PropertyPanel.vue
 *
 * 新增功能：
 * - 支持多选时显示通用属性编辑器
 * - 集成专用编辑器（PositionEditor, SpacingEditor, TransformEditor等）
 * - 根据组件类型显示对应的专有属性编辑器
 * - 支持布局组件的特殊编辑器（FlexEditor, GridEditor）
 *
 * Requirements: 3.3, 4.1
 */
import { ref, computed } from 'vue';
import { useDesignStore } from '@/store/design';
import { getComponent } from '@/registry';
import { StyleEditor, PropsEditor } from '@/components/editors';

// 导入新的编辑器
import PositionEditor from '@/components/editors/PositionEditor.vue';
import SpacingEditor from '@/components/editors/SpacingEditor.vue';
import TransformEditor from '@/components/editors/TransformEditor.vue';
import FlexEditor from '@/components/editors/FlexEditor.vue';
import GridEditor from '@/components/editors/GridEditor.vue';
import TextComponentEditor from '@/components/editors/TextComponentEditor.vue';

// Store
const designStore = useDesignStore();

// State
const activeTab = ref('props');

// Computed
/**
 * 选中的组件（单选）
 */
const selectedComponent = computed(() => designStore.selectedComponent);

/**
 * 选中的组件列表（多选）
 * Task 6.1: 支持多选
 */
const selectedComponents = computed(() => {
    const ids = designStore.selectedComponentIds || [];
    if (ids.length === 0) return [];

    return ids
        .map((id) => {
            const findById = (components) => {
                for (const comp of components) {
                    if (comp.id === id) return comp;
                    if (comp.children) {
                        const found = findById(comp.children);
                        if (found) return found;
                    }
                }
                return null;
            };
            return findById(designStore.components);
        })
        .filter(Boolean);
});

/**
 * 多选时的通用样式（取第一个组件的样式）
 */
const commonStyle = computed(() => {
    if (selectedComponents.value.length === 0) return {};
    return selectedComponents.value[0].style || {};
});

/**
 * 组件定义
 */
const componentDef = computed(() => {
    if (!selectedComponent.value) return null;
    return getComponent(selectedComponent.value.type);
});

/**
 * 属性 Schema
 */
const propsSchema = computed(() => componentDef.value?.propsSchema || {});

/**
 * 判断是否为 Flex 容器
 */
const isFlexContainer = computed(() => {
    if (!selectedComponent.value) return false;
    return (
        selectedComponent.value.type === 'Flex' ||
        (selectedComponent.value.type === 'Container' && selectedComponent.value.props?.layoutMode === 'flex')
    );
});

/**
 * 判断是否为 Grid 容器
 */
const isGridContainer = computed(() => {
    if (!selectedComponent.value) return false;
    return (
        selectedComponent.value.type === 'Grid' ||
        (selectedComponent.value.type === 'Container' && selectedComponent.value.props?.layoutMode === 'grid')
    );
});

/**
 * 获取组件专有编辑器名称
 * Task 6.3: 根据组件类型返回对应的编辑器
 */
const componentEditorName = computed(() => {
    if (!selectedComponent.value) return null;

    const editorMap = {
        Text: 'TextComponentEditor',
        // 后续添加其他组件的编辑器
        // Image: 'ImageComponentEditor',
        // Button: 'ButtonComponentEditor',
        // Input: 'InputComponentEditor',
        // Table: 'TableComponentEditor',
        // LineChart: 'ChartComponentEditor',
        // BarChart: 'ChartComponentEditor',
    };

    return editorMap[selectedComponent.value.type] || null;
});

/**
 * 组件标签
 */
const componentLabel = computed({
    get: () => selectedComponent.value?.label || '',
    set: () => {},
});

/**
 * 组件可见性
 */
const componentVisible = computed({
    get: () => selectedComponent.value?.visible !== false,
    set: () => {},
});

/**
 * 组件锁定状态
 */
const componentLocked = computed({
    get: () => selectedComponent.value?.locked === true,
    set: () => {},
});

// Methods

/**
 * 处理标签变更
 */
function handleLabelChange(value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { label: value });
}

/**
 * 处理可见性变更
 */
function handleVisibleChange(value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { visible: value });
}

/**
 * 处理锁定状态变更
 */
function handleLockedChange(value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { locked: value });
}

/**
 * 处理属性变更（单个属性）
 */
function handlePropChange(key, value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, {
        props: { [key]: value },
    });
}

/**
 * 处理属性变更（整个 props 对象）
 * Task 6.2 & 6.4: 支持编辑器返回完整的 props 对象
 */
function handlePropsChange(props) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { props });
}

/**
 * 处理样式变更（单个样式）
 */
function handleStyleChange(key, value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, {
        style: { [key]: value },
    });
}

/**
 * 处理样式变更（整个 style 对象）
 * Task 6.2: 支持编辑器返回完整的 style 对象
 */
function handleStyleObjectChange(style) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { style });
}

/**
 * 处理批量样式变更（多选）
 * Task 6.1: 多选时批量更新样式
 */
function handleBatchStyleChange(style) {
    if (selectedComponents.value.length === 0) return;

    const ids = selectedComponents.value.map((comp) => comp.id);
    designStore.batchUpdateComponents(ids, { style });
}
</script>

<style scoped>
.property-panel {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: #fff;
}

.empty-state {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
}

.panel-header {
    padding: 12px;
    border-bottom: 1px solid #e4e7ed;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.component-type {
    font-size: 14px;
    font-weight: 500;
    color: #303133;
}

.component-id {
    font-size: 12px;
    color: #909399;
    font-family: monospace;
}

.panel-tabs {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

:deep(.el-tabs__content) {
    flex: 1;
    overflow: auto;
    padding: 0;
}

:deep(.el-tab-pane) {
    height: 100%;
}

.props-section,
.style-section {
    padding: 12px;
}

.section-group {
    margin-bottom: 16px;
}

.section-group:last-child {
    margin-bottom: 0;
}

.section-title {
    font-size: 12px;
    font-weight: 500;
    color: #909399;
    margin-bottom: 8px;
    text-transform: uppercase;
}

:deep(.el-form-item) {
    margin-bottom: 12px;
}

:deep(.el-form-item:last-child) {
    margin-bottom: 0;
}

:deep(.el-form-item__label) {
    font-size: 12px;
    color: #606266;
}

:deep(.el-input-number) {
    width: 100%;
}

:deep(.el-select) {
    width: 100%;
}

:deep(.el-tabs__header) {
    margin: 0;
    padding: 0 12px;
}

:deep(.el-tabs__nav-wrap::after) {
    height: 1px;
}

:deep(.el-scrollbar) {
    height: 100%;
}

:deep(.el-scrollbar__wrap) {
    overflow-x: hidden;
}

.multi-select-state {
    height: 100%;
    display: flex;
    flex-direction: column;
}

.multi-select-state .panel-tabs {
    flex: 1;
    display: flex;
    flex-direction: column;
}

.section-group {
    padding: 12px;
    margin-bottom: 0;
}
</style>
