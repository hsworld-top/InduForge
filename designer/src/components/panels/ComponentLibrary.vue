<template>
    <div class="component-library">
        <!-- 搜索栏 -->
        <div class="component-library-search">
            <el-input v-model="searchKeyword" placeholder="搜索组件" :prefix-icon="Search" size="small" clearable />
        </div>

        <!-- 组件分类 -->
        <div class="component-categories">
            <el-collapse v-model="activeCategories" accordion>
                <!-- 布局组件 -->
                <el-collapse-item name="布局组件" title="布局组件">
                    <div class="component-list">
                        <div
                            v-for="component in filteredComponents('布局组件')"
                            :key="component.type"
                            class="component-item"
                            draggable="true"
                            @dragstart="handleDragStart($event, component)"
                            @dragend="handleDragEnd">
                            <el-icon class="component-icon">
                                <component :is="getIcon(component.icon)" />
                            </el-icon>
                            <span class="component-name">{{ component.name }}</span>
                        </div>
                    </div>
                </el-collapse-item>

                <!-- 基础组件 -->
                <el-collapse-item name="基础组件" title="基础组件">
                    <div class="component-list">
                        <div
                            v-for="component in filteredComponents('基础组件')"
                            :key="component.type"
                            class="component-item"
                            draggable="true"
                            @dragstart="handleDragStart($event, component)"
                            @dragend="handleDragEnd">
                            <el-icon class="component-icon">
                                <component :is="getIcon(component.icon)" />
                            </el-icon>
                            <span class="component-name">{{ component.name }}</span>
                        </div>
                    </div>
                </el-collapse-item>

                <!-- Element 组件 -->
                <el-collapse-item name="Element 组件" title="Element 组件">
                    <div class="component-list">
                        <div
                            v-for="component in filteredComponents('Element 组件')"
                            :key="component.type"
                            class="component-item"
                            draggable="true"
                            @dragstart="handleDragStart($event, component)"
                            @dragend="handleDragEnd">
                            <el-icon class="component-icon">
                                <component :is="getIcon(component.icon)" />
                            </el-icon>
                            <span class="component-name">{{ component.name }}</span>
                        </div>
                    </div>
                </el-collapse-item>

                <!-- 图表组件 -->
                <el-collapse-item name="图表组件" title="图表组件">
                    <div class="component-list">
                        <div
                            v-for="component in filteredComponents('图表组件')"
                            :key="component.type"
                            class="component-item"
                            draggable="true"
                            @dragstart="handleDragStart($event, component)"
                            @dragend="handleDragEnd">
                            <el-icon class="component-icon">
                                <component :is="getIcon(component.icon)" />
                            </el-icon>
                            <span class="component-name">{{ component.name }}</span>
                        </div>
                    </div>
                </el-collapse-item>
            </el-collapse>
        </div>
    </div>
</template>

<script setup>
/**
 * ComponentLibrary - 组件库组件
 * 按分类显示可用组件，支持拖拽到画布
 */
import { ref, computed, onMounted } from 'vue';
import { getComponentsByCategory, createComponentInstance } from '@/registry';
import { registerAllComponents } from '@/registry/components';
import { Search, Folder, Document, Pointer, Picture, EditPen, Grid, Menu, List, Operation } from '@element-plus/icons-vue';

// Emits
const emit = defineEmits(['drag-start', 'drag-end']);

// State
const activeCategories = ref(['布局组件']);
const searchKeyword = ref('');

// State
const componentsByCategory = ref({});

// 更新组件分类
function updateComponentsByCategory() {
    componentsByCategory.value = getComponentsByCategory();
    console.log('更新后的 componentsByCategory:', componentsByCategory.value);
}

/**
 * 过滤后的组件列表
 */
const filteredComponents = (category) => {
    const components = componentsByCategory.value[category] || [];
    if (!searchKeyword.value) {
        return components;
    }
    return components.filter((comp) => comp.name.toLowerCase().includes(searchKeyword.value.toLowerCase()));
};

// Methods

/**
 * 获取图标组件
 */
function getIcon(iconName) {
    const iconMap = {
        // 布局组件
        folder: Folder,
        menu: Menu,
        'reorder-four': Grid,
        remove: Menu,

        // 基础组件
        'crop-square': Grid,
        'radio-button-off': Operation,
        'ellipse-outline': Operation,
        'triangle-outline': Operation,
        'document-text': Document,
        image: Picture,

        // Element 组件
        pointer: Pointer,
        create: EditPen,
        'chevron-down': List,
        toggle: Operation,
        options: Operation,
        'stats-chart': Grid,
        grid: Grid,

        // 图表组件
        'trending-up': Grid,
        'bar-chart': Grid,
        'pie-chart': Grid,
        speedometer: Grid,
        ellipse: Operation,
    };
    return iconMap[iconName] || Document;
}

/**
 * 处理拖拽开始
 */
function handleDragStart(event, component) {
    // 创建组件实例
    const instance = createComponentInstance(component.type);

    // 设置拖拽数据
    event.dataTransfer.setData('application/json', JSON.stringify(instance));
    event.dataTransfer.effectAllowed = 'copy';

    // 设置拖拽图像
    const dragImage = event.target.cloneNode(true);
    dragImage.style.position = 'absolute';
    dragImage.style.top = '-1000px';
    document.body.appendChild(dragImage);
    event.dataTransfer.setDragImage(dragImage, 0, 0);
    setTimeout(() => document.body.removeChild(dragImage), 0);

    emit('drag-start', instance);
}

/**
 * 处理拖拽结束
 */
function handleDragEnd() {
    emit('drag-end');
}

// 初始化时注册所有组件
onMounted(() => {
    registerAllComponents();
    updateComponentsByCategory();
});
</script>

<style scoped>
/* 参考 OpenTiny 风格 */
.component-library {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: #fff;
}

/* 搜索栏 */
.component-library-search {
    padding: 12px;
    border-bottom: 1px solid #e4e7ed;
}

/* 分类列表 */
.component-categories {
    flex: 1;
    overflow: auto;
}

:deep(.el-collapse) {
    border: none;
}

:deep(.el-collapse-item__header) {
    height: 40px;
    line-height: 40px;
    padding: 0 16px;
    font-size: 13px;
    font-weight: 500;
    color: #303133;
    background-color: #f5f7fa;
    border-bottom: 1px solid #e4e7ed;
}

:deep(.el-collapse-item__wrap) {
    border-bottom: none;
}

:deep(.el-collapse-item__content) {
    padding: 8px 0;
    background-color: #fff;
}

/* 组件列表 - 参考 OpenTiny 的网格布局 */
.component-list {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
    padding: 8px 12px;
}

/* 组件项 - 参考 OpenTiny 的卡片样式 */
.component-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 16px 8px;
    border: 1px solid #dcdfe6;
    border-radius: 4px;
    cursor: grab;
    transition: all 0.2s;
    background-color: #fff;
    min-height: 80px;
}

.component-item:hover {
    border-color: #5e7ce0;
    background-color: #f2f5fc;
    box-shadow: 0 2px 8px rgba(94, 124, 224, 0.15);
}

.component-item:active {
    cursor: grabbing;
    transform: scale(0.98);
}

.component-icon {
    font-size: 28px;
    color: #5e7ce0;
    margin-bottom: 8px;
}

.component-name {
    font-size: 12px;
    color: #575d6c;
    text-align: center;
    line-height: 1.4;
}

/* 空状态 */
:deep(.el-empty) {
    padding: 40px 0;
}
</style>
