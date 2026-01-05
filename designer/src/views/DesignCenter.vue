<template>
    <div class="design-center">
        <!-- 顶部工具栏 - 参考 OpenTiny 风格 -->
        <div class="design-toolbar">
            <div class="toolbar-left">
                <span class="project-name">{{ projectName }}</span>
                <el-divider direction="vertical" />
                <span class="page-name">{{ currentPageName }}</span>
                <el-tag v-if="isDirty" type="warning" size="small" class="dirty-tag"> 未保存 </el-tag>
            </div>
            <div class="toolbar-center">
                <!-- 撤销/重做 -->
                <el-button-group>
                    <el-tooltip content="撤销 (Ctrl+Z)" placement="bottom">
                        <el-button :icon="IconTablerArrowBackUp" :disabled="!canUndo" size="small" @click="handleUndo" />
                    </el-tooltip>
                    <el-tooltip content="重做 (Ctrl+Y)" placement="bottom">
                        <el-button :icon="IconTablerArrowForwardUp" :disabled="!canRedo" size="small" @click="handleRedo" />
                    </el-tooltip>
                </el-button-group>

                <el-divider direction="vertical" />

                <!-- 锁定/解锁 -->
                <el-tooltip :content="isLocked ? '解锁画布' : '锁定画布'" placement="bottom">
                    <el-button :icon="isLocked ? IconTablerLock : IconTablerLockOpen" :type="isLocked ? 'warning' : ''" size="small" @click="isLocked = !isLocked" />
                </el-tooltip>

                <!-- 设备类型 -->
                <el-button-group>
                    <el-tooltip content="桌面设备" placement="bottom">
                        <el-button :icon="IconTablerDeviceDesktop" :type="deviceType === 'desktop' ? 'primary' : ''" size="small" @click="handleDeviceChange('desktop')" />
                    </el-tooltip>
                    <el-tooltip content="移动设备" placement="bottom">
                        <el-button :icon="IconTablerDeviceMobile" :type="deviceType === 'mobile' ? 'primary' : ''" size="small" @click="handleDeviceChange('mobile')" />
                    </el-tooltip>
                </el-button-group>

                <!-- 方向切换 -->
                <el-tooltip :content="orientation === 'portrait' ? '竖屏' : '横屏'" placement="bottom">
                    <el-button :icon="IconTablerStack" size="small" @click="toggleOrientation" />
                </el-tooltip>

                <el-divider direction="vertical" />

                <!-- 页面宽度显示 - 点击打开设置 -->
                <div class="canvas-info">
                    <span class="canvas-width">{{ canvasWidth }}px</span>
                    <span class="canvas-scale">{{ Math.round(canvasScale * 100) }}%</span>
                </div>

                <el-divider direction="vertical" />

                <!-- 视图选项 -->
                <el-tooltip content="显示/隐藏标尺" placement="bottom">
                    <el-button :icon="IconTablerLayoutGrid" :type="showRuler ? 'primary' : ''" size="small" @click="showRuler = !showRuler"> 标尺 </el-button>
                </el-tooltip>
            </div>
            <div class="toolbar-right">
                <!-- 保存按钮 -->
                <el-button type="primary" :icon="IconTablerCheck" :loading="saving" :disabled="!isDirty" size="small" @click="handleSave"> 保存 </el-button>
            </div>
        </div>

        <!-- 主内容区域 -->
        <div class="design-main">
            <!-- 左侧面板 -->
            <div class="left-panel" :style="{ width: leftPanelWidth + 'px' }">
                <el-tabs v-model="leftActiveTab" class="panel-tabs">
                    <el-tab-pane label="页面" name="pages">
                        <PageTree :current-page-id="designStore.currentPageId" @page-create-requested="handlePageCreateRequested" />
                    </el-tab-pane>
                    <el-tab-pane label="组件库" name="library">
                        <ComponentLibrary @drag-start="handleDragStart" @drag-end="handleDragEnd" />
                    </el-tab-pane>
                    <el-tab-pane label="工程变量" name="variables">
                        <GlobalVariables />
                    </el-tab-pane>
                    <el-tab-pane label="全局脚本" name="global-scripts">
                        <GlobalScripts />
                    </el-tab-pane>
                </el-tabs>
                <!-- 调整宽度的拖拽条 -->
                <div class="resize-handle resize-handle-right" @mousedown="startResizeLeft"></div>
            </div>

            <!-- 中间画布区域 -->
            <div ref="canvasAreaRef" class="canvas-area" :class="{ 'with-ruler': showRuler }" @dragover.prevent="handleDragOver" @drop="handleDrop" @scroll="handleCanvasScroll">
                <!-- 画布内容 -->
                <div class="canvas-content" @contextmenu.prevent="handleCanvasContextMenu">
                    <DesignCanvas v-if="currentPage" :show-grid="showGrid" @contextmenu="handleComponentContextMenu" />
                    <div v-else class="canvas-empty">
                        <el-empty description="请从左侧选择一个页面开始设计" />
                    </div>
                </div>

                <!-- 右键菜单 -->
                <ContextMenu ref="contextMenuRef" />

                <!-- 标尺（覆盖在画布上方） -->
                <CanvasRuler v-if="showRuler && currentPage" :scale="canvasScale" :scroll-left="canvasScrollLeft" :scroll-top="canvasScrollTop" />

                <!-- 左下角缩放控制 -->
                <div class="canvas-zoom-control" :style="{ left: zoomControlLeft + 'px' }">
                    <el-button-group size="small">
                        <el-tooltip content="缩小" placement="top">
                            <el-button :icon="IconTablerZoomOut" @click="handleZoomOut" />
                        </el-tooltip>
                        <el-button disabled>{{ Math.round(canvasScale * 100) }}%</el-button>
                        <el-tooltip content="放大" placement="top">
                            <el-button :icon="IconTablerZoomIn" @click="handleZoomIn" />
                        </el-tooltip>
                        <el-tooltip content="适应画布" placement="top">
                            <el-button :icon="FullScreen" @click="handleFitCanvas" />
                        </el-tooltip>
                    </el-button-group>
                </div>
            </div>

            <!-- 右侧面板 -->
            <div class="right-panel" :style="{ width: rightPanelWidth + 'px' }">
                <!-- 调整宽度的拖拽条 -->
                <div class="resize-handle resize-handle-left" @mousedown="startResizeRight"></div>
                <el-tabs v-model="rightActiveTab" class="panel-tabs right-panel-tabs">
                    <el-tab-pane label="组件树" name="tree">
                        <ComponentTree />
                    </el-tab-pane>
                    <el-tab-pane label="属性" name="properties">
                        <PropertyPanel :show-tabs="false" initial-tab="props" />
                    </el-tab-pane>
                    <el-tab-pane label="样式" name="style">
                        <PropertyPanel :show-tabs="false" initial-tab="style" />
                    </el-tab-pane>
                    <el-tab-pane label="连接" name="events">
                        <PropertyPanel :show-tabs="false" initial-tab="events" />
                    </el-tab-pane>
                    <el-tab-pane label="绑定" name="binding">
                        <DataBindingPanel />
                    </el-tab-pane>
                    <el-tab-pane name="data">
                        <template #label>
                            <span class="variable-tab-label">数据</span>
                        </template>
                        <div class="data-panel">
                            <el-collapse v-model="dataCollapse">
                                <el-collapse-item title="数据源" name="sources">
                                    <DataSourcePanel />
                                </el-collapse-item>
                                <el-collapse-item title="页面变量" name="variables">
                                    <VariablePanel />
                                </el-collapse-item>
                            </el-collapse>
                        </div>
                    </el-tab-pane>
                </el-tabs>
            </div>
        </div>

        <!-- 底部状态栏 -->
        <div class="design-statusbar">
            <span class="status-item"> 组件: {{ componentCount }} </span>
        </div>

        <!-- 页面设置对话框 -->
        <el-dialog v-model="showCanvasSettings" title="页面设置" width="500px" :close-on-click-modal="false" :lock-scroll="false">
            <el-form label-width="80px" label-position="left">
                <el-form-item label="页面名称">
                    <el-input v-model.trim="pageName" placeholder="请输入页面名称" maxlength="50" show-word-limit />
                </el-form-item>
                <el-form-item label="宽度">
                    <el-input v-model.number="canvasWidth" type="number" suffix-icon="px">
                        <template #append>px</template>
                    </el-input>
                </el-form-item>
                <el-form-item label="缩放">
                    <el-input v-model.number="canvasScalePercent" type="number">
                        <template #append>%</template>
                    </el-input>
                </el-form-item>
                <el-form-item label="自由布局">
                    <el-switch v-model="settingsFreeLayout" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="showCanvasSettings = false">取消</el-button>
                <el-button type="primary" @click="handleCanvasSettingsConfirm">确定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
/**
 * DesignCenter - 设计中心主视图
 * 集成三栏布局：PageTree + ComponentTree + ComponentLibrary | Canvas | PropertyPanel
 * Requirements: 1.1, 1.3, 6.2
 */
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue';
import { useRoute } from 'vue-router';
import { ElMessage } from 'element-plus';
import IconTablerArrowBackUp from '~icons/tabler/arrow-back-up';
import IconTablerArrowForwardUp from '~icons/tabler/arrow-forward-up';
import IconTablerCheck from '~icons/tabler/check';
import IconTablerZoomIn from '~icons/tabler/zoom-in';
import IconTablerZoomOut from '~icons/tabler/zoom-out';
import IconTablerMaximize from '~icons/tabler/maximize';
import IconTablerLayoutGrid from '~icons/tabler/layout-grid';
import IconTablerLock from '~icons/tabler/lock';
import IconTablerLockOpen from '~icons/tabler/lock-open';
import IconTablerDeviceDesktop from '~icons/tabler/device-desktop';
import IconTablerDeviceMobile from '~icons/tabler/device-mobile';
import IconTablerStack from '~icons/tabler/stack';
import { useDesignStore } from '@/store/design';
import { useCanvas } from '@/composables/useCanvas';
import { useKeyboard } from '@/composables/useKeyboard';
import {
    PageTree,
    ComponentTree,
    ComponentLibrary,
    PropertyPanel,
    GlobalVariables,
    GlobalScripts,
    VariablePanel,
    DataSourcePanel,
    DataBindingPanel,
} from '@/components/panels';
import { DesignCanvas, CanvasRuler } from '@/components/canvas';
import ContextMenu from '@/components/canvas/ContextMenu.vue';
import { registerAllComponents } from '@/registry/components';
import { createComponentInstance, getComponent } from '@/registry';

// Route
const route = useRoute();

// Store
const designStore = useDesignStore();

// Composables
const { canvasState } = useCanvas();

// State
const leftActiveTab = ref('pages');
const rightActiveTab = ref('tree');
const dataCollapse = ref(['sources', 'variables']);
const showGrid = ref(true);
const showRuler = ref(true);
const draggingComponent = ref(null);
const canvasScrollLeft = ref(0);
const canvasScrollTop = ref(0);
const canvasAreaRef = ref(null);
const contextMenuRef = ref(null);

// 面板宽度
const leftPanelWidth = ref(280);
const rightPanelWidth = ref(300);
const isResizingLeft = ref(false);
const isResizingRight = ref(false);

// 画布控制
const isLocked = ref(false);
const deviceType = ref('desktop');
const orientation = ref('portrait');
const canvasWidth = ref(1200);
const showCanvasSettings = ref(false);
const freeLayout = ref(false);
const settingsFreeLayout = ref(false);
const canvasScalePercent = ref(100);
const layoutInitPending = ref(false);
const pendingNewPage = ref(false);
const pageName = ref('');
const pageSettingsApplied = ref(false);
const syncingPageConfig = ref(false);

// 撤销/重做功能
const canUndo = computed(() => designStore.canUndo);
const canRedo = computed(() => designStore.canRedo);

// Computed
const projectId = computed(() => route.query.pid || route.query.id);
const projectName = computed(() => route.query.name || '项目');
const currentPage = computed(() => designStore.currentPage);
const currentPageName = computed(() => currentPage.value?.meta?.name || '未选择页面');
const isDirty = computed(() => designStore.isDirty);
const saving = computed(() => designStore.saving);
const canvasScale = computed(() => canvasState.scale);
const componentCount = computed(() => {
    const countComponents = (components) => {
        let count = 0;
        for (const comp of components || []) {
            count += 1;
            if (comp.children) {
                count += countComponents(comp.children);
            }
        }
        return count;
    };
    return countComponents(designStore.components);
});

const zoomControlLeft = computed(() => {
    return leftPanelWidth.value + 20;
});

// Methods

/**
 * 创建默认布局容器（填充画布）。
 * @returns {Object|null} 布局容器组件实例
 */
function createDefaultLayoutContainer() {
    const overrides = {
        props: {
            layout: 'grid',
            gridTemplateColumns: 'repeat(24, minmax(0, 1fr))',
            gridTemplateRows: 'auto',
            gridAutoFlow: 'row',
            gap: 16,
            justifyItems: 'stretch',
            alignItems: 'stretch',
            justifyContent: 'start',
            alignContent: 'start',
            __autoLayout: true,
        },
        style: {
            position: 'absolute',
            left: 0,
            top: 0,
            width: '100%',
            height: '100%',
            zIndex: 1,
        },
    };

    const instance = createComponentInstance('Container', overrides);
    if (!instance) {
        return null;
    }
    instance.meta = { ...(instance.meta || {}), autoLayout: true };
    instance.autoLayout = true;

    return instance;
}

/**
 * 在关闭自由布局时确保画布存在默认布局容器。
 * @returns {void}
 */
function ensureDefaultLayoutContainer() {
    if (freeLayout.value) return;
    if (!currentPage.value || !Array.isArray(currentPage.value.components)) return;
    if (currentPage.value.components.length > 0) return;

    if (!getComponent('Container')) {
        if (!layoutInitPending.value) {
            layoutInitPending.value = true;
            setTimeout(() => {
                layoutInitPending.value = false;
                ensureDefaultLayoutContainer();
            }, 0);
        }
        return;
    }

    const container = createDefaultLayoutContainer();
    if (!container) return;

    designStore.addComponent(container);
}

/**
 * 判断是否为默认布局容器
 * @param {Object} component - 组件数据
 * @returns {boolean} 是否为默认布局容器
 */
function isDefaultLayoutContainer(component) {
    if (!component || component.type !== 'Container') return false;
    if (component.meta?.autoLayout === true || component.autoLayout === true) return true;
    const props = component.props || {};
    if (props.__autoLayout === true) return true;
    const style = component.style || {};

    const normalizeText = (value) => String(value || '').replace(/\s+/g, '').toLowerCase();
    const gridColumns = normalizeText(props.gridTemplateColumns);
    const isGridLayout = props.layout === 'grid' || props.layoutMode === 'grid';
    const isGapMatch = Number(props.gap) === 16;
    const isLeftZero = style.left === 0 || style.left === '0' || style.left === '0px';
    const isTopZero = style.top === 0 || style.top === '0' || style.top === '0px';
    const widthText = normalizeText(style.width);
    const heightText = normalizeText(style.height);
    const isFullWidth = widthText === '100%';
    const isFullHeight = heightText === '100%';
    const hasGridTemplate = Boolean(props.gridTemplateColumns || props.gridTemplateRows || props.gridAutoFlow);
    const gridColumnsLooksAuto =
        gridColumns === 'repeat(24,minmax(0,1fr))' ||
        gridColumns.includes('repeat(24') ||
        gridColumns.includes('minmax(0,1fr)');

    const gridRowsMatch = !props.gridTemplateRows || normalizeText(props.gridTemplateRows) === 'auto';
    const gridAutoFlowMatch = !props.gridAutoFlow || normalizeText(props.gridAutoFlow) === 'row';
    const justifyItemsMatch = !props.justifyItems || normalizeText(props.justifyItems) === 'stretch';
    const alignItemsMatch = !props.alignItems || normalizeText(props.alignItems) === 'stretch';
    const justifyContentMatch = !props.justifyContent || normalizeText(props.justifyContent) === 'start';
    const alignContentMatch = !props.alignContent || normalizeText(props.alignContent) === 'start';

    return (
        (isGridLayout || hasGridTemplate) &&
        gridColumnsLooksAuto &&
        gridRowsMatch &&
        gridAutoFlowMatch &&
        isGapMatch &&
        justifyItemsMatch &&
        alignItemsMatch &&
        justifyContentMatch &&
        alignContentMatch &&
        style.position === 'absolute' &&
        isLeftZero &&
        isTopZero &&
        isFullWidth &&
        isFullHeight
    );
}

/**
 * 自由布局下移除自动布局容器
 * @returns {void}
 */
function removeAutoLayoutContainerIfNeeded() {
    if (!freeLayout.value) return;
    if (!currentPage.value || !Array.isArray(currentPage.value.components)) return;
    const components = currentPage.value.components;
    if (components.length === 0) return;

    const nextComponents = [];
    let removed = false;

    components.forEach((component) => {
        if (!isDefaultLayoutContainer(component)) {
            nextComponents.push(component);
            return;
        }

        const children = Array.isArray(component?.children) ? component.children : [];
        if (children.length > 0) {
            nextComponents.push(...children);
        }
        if (designStore.selectedComponentId === component.id) {
            designStore.selectedComponentId = null;
        }
        removed = true;
    });

    if (!removed) return;

    currentPage.value.components = nextComponents;
    designStore.isDirty = true;
}

/**
 * 获取页面自由布局状态
 * @param {Object} page - 页面数据
 * @returns {boolean} 是否启用自由布局
 */
function resolvePageFreeLayout(page) {
    if (!page) return false;
    const freeLayoutValue = page.config?.freeLayout;
    if (typeof freeLayoutValue === 'boolean') return freeLayoutValue;
    if (typeof freeLayoutValue === 'string') {
        const normalizedValue = freeLayoutValue.trim().toLowerCase();
        return normalizedValue === 'true' || normalizedValue === '1';
    }
    return freeLayoutValue === 1;
}

/**
 * 同步当前页面状态（自由布局/表单/布局容器）
 * @returns {Promise<void>} 同步完成
 */
async function applyPageSwitchState() {
    if (!currentPage.value) return;
    syncingPageConfig.value = true;
    const pageFreeLayout = resolvePageFreeLayout(currentPage.value);
    if (freeLayout.value !== pageFreeLayout) {
        freeLayout.value = pageFreeLayout;
    }
    if (!showCanvasSettings.value) {
        syncPageSettingsForm();
    }
    await nextTick();
    syncingPageConfig.value = false;
    if (!freeLayout.value) {
        ensureDefaultLayoutContainer();
    } else {
        removeAutoLayoutContainerIfNeeded();
    }
}

/**
 * 同步页面设置表单数据
 * @returns {void}
 */
function syncPageSettingsForm() {
    if (!currentPage.value) {
        pageName.value = '';
        canvasWidth.value = 1200;
        return;
    }

    const configWidth = currentPage.value?.config?.width;
    canvasWidth.value = Number.isFinite(configWidth) ? configWidth : 1200;
    settingsFreeLayout.value = resolvePageFreeLayout(currentPage.value);
    pageName.value = currentPage.value?.meta?.name || '新页面';
}

/**
 * 打开页面设置
 * @returns {void}
 */
function handleCanvasSettingsOpen() {
    if (!currentPage.value) {
        ElMessage.warning('请先选择一个页面');
        return;
    }
    showCanvasSettings.value = true;
    syncPageSettingsForm();
    settingsFreeLayout.value = freeLayout.value;
}

/**
 * 处理新页面创建入口
 * @returns {void}
 */
function handlePageCreateRequested() {
    pendingNewPage.value = true;
    pageSettingsApplied.value = false;
    pageName.value = '新页面';
    canvasWidth.value = 1200;
    settingsFreeLayout.value = false;
    showCanvasSettings.value = true;
}
/**
 * 加载项目
 * Requirements: 1.1 - 打开设计中心时显示页面树
 */
async function loadProject() {
    const pid = projectId.value;
    if (!pid) {
        ElMessage.warning('未指定项目ID');
        return;
    }

    try {
        await designStore.loadProject(pid);

        // 如果 URL 中有 pageid 参数，自动加载该页面
        const pageId = route.query.pageid;
        if (pageId) {
            try {
                await designStore.loadPage(pageId);
                console.log('✅ 自动加载页面:', pageId);
            } catch (error) {
                console.error('❌ 自动加载页面失败:', error);
                ElMessage.warning(`无法加载页面 ${pageId}，请从左侧页面树中选择一个页面`);
            }
        } else {
            // 如果没有指定页面，尝试加载第一个页面
            if (designStore.pages.length > 0) {
                const firstPage = designStore.pages.find((p) => p.type !== 'folder');
                if (firstPage) {
                    try {
                        await designStore.loadPage(firstPage.id);
                        console.log('✅ 自动加载第一个页面:', firstPage.id);
                    } catch (error) {
                        console.error('❌ 自动加载第一个页面失败:', error);
                    }
                }
            }
        }
    } catch (error) {
        ElMessage.error('加载项目失败: ' + error.message);
    }
}

/**
 * 保存页面
 * Requirements: 6.2 - 点击保存按钮时持久化 schema
 */
async function handleSave() {
    try {
        await designStore.savePage();
        ElMessage.success('保存成功');
    } catch (error) {
        // Requirements: 6.4 - 显示保存错误信息
        ElMessage.error('保存失败: ' + error.message);
    }
}

/**
 * 撤销操作
 */
function handleUndo() {
    designStore.undo();
    ElMessage.success('已撤销');
}

/**
 * 重做操作
 */
function handleRedo() {
    designStore.redo();
    ElMessage.success('已重做');
}

/**
 * 缩小画布
 */
function handleZoomOut() {
    const newScale = Math.max(0.1, canvasScale.value - 0.1);
    canvasState.scale = newScale;
}

/**
 * 放大画布
 */
function handleZoomIn() {
    const newScale = Math.min(2, canvasScale.value + 0.1);
    canvasState.scale = newScale;
}

/**
 * 适应画布
 */
function handleFitCanvas() {
    canvasState.scale = 1;
    ElMessage.success('已重置缩放');
}

/**
 * 处理组件拖拽开始
 */
function handleDragStart(component) {
    draggingComponent.value = component;
}

/**
 * 处理组件拖拽结束
 */
function handleDragEnd() {
    draggingComponent.value = null;
}

/**
 * 处理拖拽经过画布
 */
function handleDragOver(event) {
    event.dataTransfer.dropEffect = 'copy';
}

/**
 * 处理组件放置到画布
 * Requirements: 8.2 - 拖拽组件到画布创建新实例
 */
function handleDrop(event) {
    if (!currentPage.value) {
        ElMessage.warning('请先选择一个页面');
        return;
    }

    try {
        console.info('[DesignCenter] drop');
        const data =
            event.dataTransfer.getData('application/json') ||
            event.dataTransfer.getData('text/plain') ||
            event.dataTransfer.getData('application/x-designer-component');
        if (!data) return;

        const component = JSON.parse(data);

        // 画布内拖动：直接移动位置而不是新增
        if (component?.source === 'canvas' && component.id) {
            const canvasArea = event.currentTarget;
            const rect = canvasArea.getBoundingClientRect();
            const x = event.clientX - rect.left;
            const y = event.clientY - rect.top;

            const offsetX = component.offsetX || 0;
            const offsetY = component.offsetY || 0;
            const newLeft = Math.round(x / canvasState.scale - offsetX);
            const newTop = Math.round(y / canvasState.scale - offsetY);

            designStore.updateComponent(component.id, {
                style: {
                    left: newLeft,
                    top: newTop,
                },
            });
            designStore.saveHistory(`移动组件 ${component.name || component.type}`);
            designStore.selectComponent(component.id);
            return;
        }

        // 计算放置位置（相对于画布）
        const canvasArea = event.currentTarget;
        const rect = canvasArea.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;

        // 设置组件位置
        component.style = {
            ...component.style,
            left: Math.round(x / canvasState.scale),
            top: Math.round(y / canvasState.scale),
        };

        // 添加组件到页面
        designStore.addComponent(component);

        // 选中新添加的组件
        designStore.selectComponent(component.id);
    } catch (error) {
        console.error('Drop error:', error);
    }
}

/**
 * 处理画布滚动
 */
function handleCanvasScroll(event) {
    canvasScrollLeft.value = event.target.scrollLeft;
    canvasScrollTop.value = event.target.scrollTop;
}

/**
 * 开始调整左侧面板宽度
 */
function startResizeLeft(event) {
    isResizingLeft.value = true;
    event.preventDefault();
}

/**
 * 开始调整右侧面板宽度
 */
function startResizeRight(event) {
    isResizingRight.value = true;
    event.preventDefault();
}

/**
 * 处理鼠标移动（调整面板宽度）
 */
function handleMouseMove(event) {
    if (isResizingLeft.value) {
        const newWidth = event.clientX;
        if (newWidth >= 200 && newWidth <= 500) {
            leftPanelWidth.value = newWidth;
        }
    } else if (isResizingRight.value) {
        const newWidth = window.innerWidth - event.clientX;
        if (newWidth >= 200 && newWidth <= 500) {
            rightPanelWidth.value = newWidth;
        }
    }
}

/**
 * 停止调整面板宽度
 */
function stopResize() {
    isResizingLeft.value = false;
    isResizingRight.value = false;
}

/**
 * 切换设备类型
 */
function handleDeviceChange(type) {
    deviceType.value = type;
    if (type === 'desktop') {
        canvasWidth.value = 1200;
        orientation.value = 'landscape';
    } else {
        canvasWidth.value = 375;
        orientation.value = 'portrait';
    }
}

/**
 * 切换方向
 */
function toggleOrientation() {
    if (orientation.value === 'portrait') {
        orientation.value = 'landscape';
    } else {
        orientation.value = 'portrait';
    }
}

/**
 * 规范化页面宽度输入
 * @param {number|string} value - 输入的宽度
 * @param {number} fallback - 回退宽度
 * @returns {number} 规范化后的宽度
 */
function normalizeCanvasWidth(value, fallback = 1200) {
    const width = Number(value);
    if (!Number.isFinite(width) || width <= 0) {
        return fallback;
    }
    return Math.round(width);
}

/**
 * 确认页面设置
 * @returns {Promise<void>} 页面设置更新流程
 */
async function handleCanvasSettingsConfirm() {
    const rawName = pageName.value.trim();
    const normalizedName = rawName || (pendingNewPage.value ? '新页面' : currentPage.value?.meta?.name || '页面');
    const normalizedWidth = normalizeCanvasWidth(canvasWidth.value, 1200);
    const desiredFreeLayout = settingsFreeLayout.value;

    if (pendingNewPage.value) {
        try {
            const page = await designStore.createPage(normalizedName, null, 'page', {
                activate: true,
                schemaOverrides: {
                    config: {
                        width: normalizedWidth,
                        freeLayout: desiredFreeLayout,
                    },
                },
            });
            designStore.cachePageConfig(page.id, {
                width: normalizedWidth,
                freeLayout: desiredFreeLayout,
            });
            try {
                await designStore.loadPage(page.id);
            } catch (error) {
                ElMessage.warning('页面已创建，但加载失败: ' + error.message);
            }
        } catch (error) {
            ElMessage.error('创建页面失败: ' + error.message);
            return;
        }

        pendingNewPage.value = false;
        freeLayout.value = desiredFreeLayout;
        await applyPageSwitchState();
        canvasWidth.value = normalizedWidth;
        pageName.value = normalizedName;
        pageSettingsApplied.value = true;
        canvasState.scale = canvasScalePercent.value / 100;
        showCanvasSettings.value = false;
        ElMessage.success('页面创建成功');
        return;
    }

    if (normalizedName && designStore.currentPageId && normalizedName !== currentPage.value?.meta?.name) {
        try {
            await designStore.renamePage(designStore.currentPageId, normalizedName);
        } catch (error) {
            ElMessage.error('页面名称更新失败: ' + error.message);
            return;
        }
    }

    if (currentPage.value && Number.isFinite(normalizedWidth)) {
        if (!currentPage.value.config || typeof currentPage.value.config !== 'object') {
            currentPage.value.config = {};
        }
        if (currentPage.value.config.width !== normalizedWidth) {
            currentPage.value.config.width = normalizedWidth;
            designStore.isDirty = true;
        }
        if (currentPage.value.config.freeLayout !== desiredFreeLayout) {
            currentPage.value.config.freeLayout = desiredFreeLayout;
            designStore.isDirty = true;
        }
    }

    if (desiredFreeLayout !== freeLayout.value) {
        freeLayout.value = desiredFreeLayout;
    }

    canvasWidth.value = normalizedWidth;
    pageName.value = normalizedName;
    if (designStore.currentPageId) {
        designStore.cachePageConfig(designStore.currentPageId, {
            width: normalizedWidth,
            freeLayout: desiredFreeLayout,
        });
    }
    pageSettingsApplied.value = true;
    canvasState.scale = canvasScalePercent.value / 100;
    showCanvasSettings.value = false;
    ElMessage.success('页面设置已更新');
}

/**
 * 处理键盘快捷键
 */
function handleKeydown(event) {
    // Ctrl+S 保存
    if (event.ctrlKey && event.key === 's') {
        event.preventDefault();
        if (isDirty.value) {
            handleSave();
        }
    }
    // Ctrl+Z 撤销
    if (event.ctrlKey && event.key === 'z') {
        event.preventDefault();
        handleUndo();
    }
    // Ctrl+Y 重做
    if (event.ctrlKey && event.key === 'y') {
        event.preventDefault();
        handleRedo();
    }
    // Delete 删除选中组件
    if (event.key === 'Delete' && designStore.selectedComponentId) {
        const selected = designStore.selectedComponent;
        if (selected && !selected.locked) {
            designStore.removeComponent(designStore.selectedComponentId);
        }
    }
    // Escape 取消选择
    if (event.key === 'Escape') {
        designStore.selectComponent(null);
    }
}

// 启用键盘快捷键
useKeyboard();

// 右键菜单处理
function handleCanvasContextMenu(e) {
    // 画布空白区域右键，取消选择
    if (!designStore.selectedComponentId) {
        return;
    }
}

// 组件右键菜单处理
function handleComponentContextMenu({ id, x, y }) {
    if (contextMenuRef.value) {
        contextMenuRef.value.show(x, y);
    }
}

// 点击其他地方关闭右键菜单
function handleClickOutside() {
    if (contextMenuRef.value) {
        contextMenuRef.value.hide();
    }
}

// Lifecycle
onMounted(() => {
    // 注册所有组件
    registerAllComponents();

    // 监听点击事件关闭右键菜单
    document.addEventListener('click', handleClickOutside);

    // 加载项目
    loadProject();

    // 添加键盘事件监听
    window.addEventListener('keydown', handleKeydown);

    // 添加鼠标事件监听（用于调整面板宽度）
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', stopResize);
});

onUnmounted(() => {
    // 移除键盘事件监听
    window.removeEventListener('keydown', handleKeydown);

    // 移除右键菜单点击监听
    document.removeEventListener('click', handleClickOutside);

    // 移除鼠标事件监听
    window.removeEventListener('mousemove', handleMouseMove);
    window.removeEventListener('mouseup', stopResize);

    // 重置 store
    designStore.reset();
});

// 监听路由变化，重新加载项目
watch(
    () => route.query.pid || route.query.id,
    (newPid) => {
        if (newPid) {
            loadProject();
        }
    },
);

watch(
    () => currentPage.value?.meta?.id,
    async () => {
        await applyPageSwitchState();
    },
    { immediate: true },
);

// 关闭自由布局时初始化默认布局容器
watch(
    () => freeLayout.value,
    () => {
        if (pendingNewPage.value || syncingPageConfig.value) return;
        if (freeLayout.value) {
            removeAutoLayoutContainerIfNeeded();
            return;
        }
        ensureDefaultLayoutContainer();
    },
);

watch(
    () => showCanvasSettings.value,
    (visible) => {
        if (visible) {
            pageSettingsApplied.value = false;
            if (pendingNewPage.value) return;
            syncPageSettingsForm();
            settingsFreeLayout.value = freeLayout.value;
            return;
        }
        if (!pageSettingsApplied.value) {
            syncPageSettingsForm();
            settingsFreeLayout.value = freeLayout.value;
        }
        if (pendingNewPage.value) {
            pendingNewPage.value = false;
        }
        pageSettingsApplied.value = false;
    },
);
// 同步缩放百分比
watch(
    () => canvasState.scale,
    (newScale) => {
        canvasScalePercent.value = Math.round(newScale * 100);
    },
);
</script>

<style scoped>
.design-center {
    width: 100%;
    height: 100vh;
    display: flex;
    flex-direction: column;
    background-color: #f5f7fa;
    overflow: hidden;
}

/* 顶部工具栏 - 参考 OpenTiny 风格 */
.design-toolbar {
    height: 56px;
    background-color: #fff;
    border-bottom: 1px solid #dcdfe6;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px;
    flex-shrink: 0;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
}

.toolbar-left {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
}

.project-name {
    font-size: 14px;
    font-weight: 600;
    color: #252b3a;
}

.page-name {
    font-size: 13px;
    color: #575d6c;
}

.dirty-tag {
    margin-left: 8px;
}

.toolbar-center {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 2;
    justify-content: center;
}

.canvas-info {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 12px;
    cursor: pointer;
    border-radius: 4px;
    transition: background-color 0.2s;
}

.canvas-info:hover {
    background-color: #f5f7fa;
}

.canvas-width {
    font-size: 14px;
    color: #575d6c;
    font-weight: 500;
}

.canvas-scale {
    font-size: 13px;
    color: #909399;
}

.toolbar-right {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    justify-content: flex-end;
}

/* 主内容区域 */
.design-main {
    flex: 1;
    display: flex;
    overflow: hidden;
}

/* 左侧面板 */
.left-panel {
    min-width: 200px;
    max-width: 500px;
    background-color: #fff;
    border-right: 1px solid #e4e7ed;
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    position: relative;
}

.panel-tabs {
    height: 100%;
    display: flex;
    flex-direction: column;
}

:deep(.left-panel .el-tabs__header) {
    margin: 0;
    padding: 0 8px;
    background-color: #fafafa;
}

:deep(.left-panel .el-tabs__content) {
    flex: 1;
    overflow: hidden;
    padding: 0;
}

:deep(.left-panel .el-tab-pane) {
    height: 100%;
    overflow: auto;
}

/* 中间画布区域 */
.canvas-area {
    flex: 1;
    overflow: auto;
    position: relative;
    background-color: #f5f5f5;
    background-image: linear-gradient(to right, #e0e0e0 1px, transparent 1px), linear-gradient(to bottom, #e0e0e0 1px, transparent 1px);
    background-size: 10px 10px;
}

.canvas-area.with-ruler {
    /* 标尺会覆盖在画布上 */
}

.canvas-content {
    width: 100%;
    height: 100%;
    position: relative;
    z-index: 1;
}

.canvas-empty {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
}

/* 左下角缩放控制 */
.canvas-zoom-control {
    position: fixed;
    bottom: 40px;
    z-index: 100;
    background-color: #fff;
    border-radius: 4px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    padding: 4px;
    transition: left 0.1s;
}

.canvas-zoom-control :deep(.el-button-group) {
    display: flex;
}

.canvas-zoom-control :deep(.el-button[disabled]) {
    color: #575d6c;
    background-color: #fff;
    border-color: #dcdfe6;
    cursor: default;
    font-weight: 500;
    min-width: 60px;
}

/* 右侧面板 */
.right-panel {
    min-width: 200px;
    max-width: 500px;
    background-color: #fff;
    border-left: 1px solid #e4e7ed;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    position: relative;
}

:deep(.right-panel .el-tabs__header) {
    margin: 0;
    padding: 0 8px;
    background-color: #fafafa;
}

:deep(.right-panel-tabs .el-tabs__item) {
    padding: 0 17px;
}

:deep(.right-panel-tabs .variable-tab-label) {
    display: inline-block;
    margin-right: 10px;
}

.data-panel {
    padding: 8px 10px;
}

.data-panel :deep(.el-collapse-item__content) {
    padding-bottom: 12px;
}

:deep(.right-panel .el-tabs__content) {
    flex: 1;
    overflow: hidden;
    padding: 0;
}

:deep(.right-panel .el-tab-pane) {
    height: 100%;
    overflow: auto;
}

/* 底部状态栏 */
.design-statusbar {
    height: 24px;
    background-color: #fff;
    border-top: 1px solid #e4e7ed;
    display: flex;
    align-items: center;
    padding: 0 16px;
    font-size: 12px;
    color: #909399;
    flex-shrink: 0;
}

.status-item {
    display: flex;
    align-items: center;
}

/* 调整宽度的拖拽条 */
.resize-handle {
    position: absolute;
    top: 0;
    bottom: 0;
    width: 4px;
    cursor: col-resize;
    z-index: 10;
    transition: background-color 0.2s;
}

.resize-handle:hover {
    background-color: #5e7ce0;
}

.resize-handle-right {
    right: -2px;
}

.resize-handle-left {
    left: -2px;
}

/* 响应式调整 */
@media (max-width: 1200px) {
    .left-panel {
        min-width: 200px;
    }

    .right-panel {
        min-width: 200px;
    }
}
</style>
