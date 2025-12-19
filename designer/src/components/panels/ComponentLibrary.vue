<template>
    <div class="component-library">
        <!-- 搜索框 -->
        <div class="component-library-search">
            <el-input v-model="searchKeyword" placeholder="搜索组件" :prefix-icon="Search" size="small" clearable />
        </div>

        <!-- 组件分类 -->
        <div class="component-categories">
            <el-collapse v-model="activeCategories" accordion>
                <!-- 布局组件 -->
                <el-collapse-item :name="CATEGORY_KEYS.layout" title="布局组件">
                    <div class="component-list">
                        <div
                            v-for="component in filteredComponents(CATEGORY_KEYS.layout)"
                            :key="component.type"
                            class="component-item"
                            draggable="true"
                            @dragstart="handleDragStart($event, component)"
                            @dragend="handleDragEnd">
                            <el-icon class="component-icon">
                                <component :is="getIcon(component)" />
                            </el-icon>
                            <span class="component-name">{{ component.name }}</span>
                        </div>
                    </div>
                </el-collapse-item>

                <!-- 基础组件 -->
                <el-collapse-item :name="CATEGORY_KEYS.basic" title="基础组件">
                    <div class="component-list">
                        <div
                            v-for="component in filteredComponents(CATEGORY_KEYS.basic)"
                            :key="component.type"
                            class="component-item"
                            draggable="true"
                            @dragstart="handleDragStart($event, component)"
                            @dragend="handleDragEnd">
                            <el-icon class="component-icon">
                                <component :is="getIcon(component)" />
                            </el-icon>
                            <span class="component-name">{{ component.name }}</span>
                        </div>
                    </div>
                </el-collapse-item>

                <!-- Element 组件 -->
                <el-collapse-item :name="CATEGORY_KEYS.element" title="Element 组件">
                    <div class="element-subcategories">
                        <div
                            v-for="subcategory in elementDisplayCategories"
                            :key="subcategory.name"
                            class="element-subcategory">
                            <div
                                class="element-subcategory__header"
                                :class="{ active: activeElementSubCategory === subcategory.name }"
                                @click="toggleElementSubCategory(subcategory.name)">
                                <span class="element-subcategory__title">{{ subcategory.name }}</span>
                                <el-icon class="element-subcategory__arrow">
                                    <component :is="activeElementSubCategory === subcategory.name ? ArrowDown : ArrowRight" />
                                </el-icon>
                            </div>
                            <div v-if="activeElementSubCategory === subcategory.name" class="element-subcategory__body">
                                <div v-if="subcategory.components.length" class="component-list">
                                    <div
                                        v-for="component in subcategory.components"
                                        :key="component.type"
                                        class="component-item"
                                        draggable="true"
                                        @dragstart="handleDragStart($event, component)"
                                        @dragend="handleDragEnd">
                                        <el-icon class="component-icon">
                                            <component :is="getIcon(component)" />
                                        </el-icon>
                                        <span class="component-name">{{ component.name }}</span>
                                    </div>
                                </div>
                                <el-empty v-else description="暂无组件" />
                            </div>
                        </div>
                    </div>
                </el-collapse-item>

                <!-- 图表组件 -->
                <el-collapse-item :name="CATEGORY_KEYS.chart" title="图表组件">
                    <div class="component-list">
                        <div
                            v-for="component in filteredComponents(CATEGORY_KEYS.chart)"
                            :key="component.type"
                            class="component-item"
                            draggable="true"
                            @dragstart="handleDragStart($event, component)"
                            @dragend="handleDragEnd">
                            <el-icon class="component-icon">
                                <component :is="getIcon(component)" />
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
 * ComponentLibrary - 组件库列表
 * 按分类展示可用组件，支持拖拽到画布
 */
import { ref, computed, onMounted } from 'vue';
import * as ElementPlusIconsVue from '@element-plus/icons-vue';
import { getComponentsByCategory, createComponentInstance } from '@/registry';
import { registerAllComponents } from '@/registry/components';

// Emits
const emit = defineEmits(['drag-start', 'drag-end']);

const CATEGORY_KEYS = {
    layout: '布局组件',
    basic: '基础组件',
    element: 'Element 组件',
    chart: '图表组件',
};

// State
const activeCategories = ref([CATEGORY_KEYS.layout]);
const activeElementSubCategory = ref('布局');
const searchKeyword = ref('');
const componentsByCategory = ref({});

const ELEMENT_CATEGORIES = ['布局', '基础', '表单', '数据', '导航', '反馈', '其他'];

const ELEMENT_TYPE_CATEGORY_MAP = {
    // 布局
    ElContainer: '布局',
    ElHeader: '布局',
    ElAside: '布局',
    ElMain: '布局',
    ElFooter: '布局',
    ElRow: '布局',
    ElCol: '布局',
    ElSpace: '布局',

    // 基础
    Button: '基础',
    Icon: '基础',
    ElDivider: '基础',
    ElCard: '基础',
    ElLink: '基础',
    ElText: '基础',

    // 表单
    Input: '表单',
    ElInput: '表单',
    ElInputNumber: '表单',
    Select: '表单',
    ElSelect: '表单',
    Switch: '表单',
    ElSwitch: '表单',
    Slider: '表单',
    ElSlider: '表单',
    DatePicker: '表单',
    ElDatePicker: '表单',
    ElTimePicker: '表单',
    ElTimeSelect: '表单',
    ElCascader: '表单',
    ElColorPicker: '表单',
    ElRate: '表单',
    ElTransfer: '表单',
    ElUpload: '表单',
    Form: '表单',
    ElRadioGroup: '表单',
    ElCheckboxGroup: '表单',

    // 数据
    Progress: '数据',
    Table: '数据',
    ElTree: '数据',
    ElCalendar: '数据',
    ElDescriptions: '数据',
    ElSkeleton: '数据',
    ElImage: '数据',
    ElEmpty: '数据',
    ElStatistic: '数据',
    ElTag: '数据',
    ElBadge: '数据',
    ElAvatar: '数据',
    ElTimeline: '数据',
    ElCollapse: '数据',
    ElCarousel: '数据',
    ElPagination: '数据',
    ElCard: '数据',

    // 导航
    ElBreadcrumb: '导航',
    ElTabs: '导航',
    ElSteps: '导航',
    ElMenu: '导航',
    ElDropdown: '导航',
    ElBacktop: '导航',
    ElAffix: '导航',

    // 反馈
    ElTooltip: '反馈',
    ElPopover: '反馈',
    ElPopconfirm: '反馈',
    ElDialog: '反馈',
    ElDrawer: '反馈',
    ElAlert: '反馈',
    ElResult: '反馈',
};

const ELEMENT_ICON_MAP = {
    // 布局
    ElContainer: 'Collection',
    ElHeader: 'Top',
    ElAside: 'List',
    ElMain: 'Document',
    ElFooter: 'Bottom',
    ElRow: 'Grid',
    ElCol: 'Grid',
    ElSpace: 'Rank',

    // 基础
    Button: 'Pointer',
    Icon: 'Brush',
    ElDivider: 'Minus',
    ElCard: 'Collection',
    ElLink: 'Link',
    ElText: 'Document',

    // 表单
    Input: 'EditPen',
    ElInput: 'EditPen',
    ElInputNumber: 'List',
    Select: 'ArrowDown',
    ElSelect: 'ArrowDown',
    Switch: 'SwitchButton',
    ElSwitch: 'SwitchButton',
    Slider: 'Operation',
    ElSlider: 'Operation',
    DatePicker: 'Calendar',
    ElDatePicker: 'Calendar',
    ElTimePicker: 'Timer',
    ElTimeSelect: 'Timer',
    ElCascader: 'Connection',
    ElColorPicker: 'Brush',
    ElRate: 'StarFilled',
    ElTransfer: 'SwitchFilled',
    ElUpload: 'UploadFilled',
    Form: 'DocumentChecked',
    ElRadioGroup: 'Select',
    ElCheckboxGroup: 'Finished',

    // 数据
    Progress: 'DataLine',
    Table: 'Grid',
    ElTree: 'Connection',
    ElCalendar: 'Calendar',
    ElDescriptions: 'Document',
    ElSkeleton: 'Memo',
    ElImage: 'PictureFilled',
    ElEmpty: 'Box',
    ElStatistic: 'Histogram',
    ElTag: 'PriceTag',
    ElBadge: 'Bell',
    ElAvatar: 'UserFilled',
    ElTimeline: 'Timer',
    ElCollapse: 'Fold',
    ElCarousel: 'Picture',
    ElPagination: 'MoreFilled',
    ElCard: 'Collection',

    // 导航
    ElBreadcrumb: 'More',
    ElTabs: 'Tickets',
    ElSteps: 'List',
    ElMenu: 'Menu',
    ElDropdown: 'ArrowDown',
    ElBacktop: 'Top',
    ElAffix: 'Position',

    // 反馈
    ElTooltip: 'ChatDotRound',
    ElPopover: 'ChatLineSquare',
    ElPopconfirm: 'QuestionFilled',
    ElDialog: 'ChatDotRound',
    ElDrawer: 'Expand',
    ElAlert: 'WarningFilled',
    ElResult: 'CircleCheckFilled',
};

const Search = ElementPlusIconsVue.Search;
const ArrowRight = ElementPlusIconsVue.ArrowRight;
const ArrowDown = ElementPlusIconsVue.ArrowDown;
const DEFAULT_ICON = ElementPlusIconsVue.Document;

// 更新组件分类
function updateComponentsByCategory() {
    componentsByCategory.value = getComponentsByCategory();
}

function loadComponents(attempt = 0) {
    updateComponentsByCategory();
    const total =
        Object.values(componentsByCategory.value || {}).reduce((sum, list) => sum + (Array.isArray(list) ? list.length : 0), 0) || 0;
    if (total === 0 && attempt < 5) {
        setTimeout(() => loadComponents(attempt + 1), 200);
    }
}

const matchesKeyword = (name = '', type = '') => {
    if (!searchKeyword.value) return true;
    const keyword = searchKeyword.value.toLowerCase();
    return name.toLowerCase().includes(keyword) || type.toLowerCase().includes(keyword);
};

/**
 * 过滤后的组件列表（非 Element 分组）
 */
const filteredComponents = (categoryKey) => {
    const components = componentsByCategory.value[categoryKey] || [];
    return components.filter((comp) => matchesKeyword(comp.name || '', comp.type || ''));
};

const filteredElementCategories = computed(() => {
    const elementList = componentsByCategory.value[CATEGORY_KEYS.element] || [];
    const filtered = elementList.filter((comp) => matchesKeyword(comp.name || '', comp.type || ''));

    const grouped = ELEMENT_CATEGORIES.reduce((acc, name) => {
        acc[name] = [];
        return acc;
    }, {});

    filtered.forEach((comp) => {
        const category = ELEMENT_TYPE_CATEGORY_MAP[comp.type] || '其他';
        if (!grouped[category]) grouped[category] = [];
        grouped[category].push(comp);
    });

    return ELEMENT_CATEGORIES.map((name) => ({
        name,
        components: grouped[name] || [],
    }));
});

const elementDisplayCategories = computed(() => filteredElementCategories.value);

// Methods

function toggleElementSubCategory(name) {
    activeElementSubCategory.value = activeElementSubCategory.value === name ? '' : name;
}

/**
 * 获取图标组件
 */
function getIcon(component) {
    const iconName = component?.type ? ELEMENT_ICON_MAP[component.type] : component;

    if (iconName && ElementPlusIconsVue[iconName]) {
        return ElementPlusIconsVue[iconName];
    }

    const fallbackName = component?.icon
        ? component.icon.replace(/(^\w|-\w)/g, (match) => match.replace('-', '').toUpperCase())
        : '';
    if (fallbackName && ElementPlusIconsVue[fallbackName]) {
        return ElementPlusIconsVue[fallbackName];
    }

    return DEFAULT_ICON;
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
    loadComponents();
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

/* 搜索框 */
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

.element-subcategories {
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.element-subcategory {
    border-bottom: 1px solid #f0f2f5;
}

.element-subcategory__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    cursor: pointer;
    color: #303133;
}

.element-subcategory__header.active {
    color: #5e7ce0;
    background-color: #f7f8fc;
}

.element-subcategory__title {
    font-size: 13px;
}

.element-subcategory__arrow {
    font-size: 13px;
    color: #909399;
}

.element-subcategory__body {
    padding: 4px 0 8px;
}

/* 组件列表 - 参考 OpenTiny 的网格分布 */
.component-list {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
    padding: 8px 12px;
}

/* 组件卡片 - 参考 OpenTiny 的卡片样式 */
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
