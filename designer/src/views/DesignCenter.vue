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

                <!-- 画布宽度显示 - 点击打开设置 -->
                <el-tooltip content="点击设置画布" placement="bottom">
                    <div class="canvas-info" @click="showCanvasSettings = true">
                        <span class="canvas-width">{{ canvasWidth }}px</span>
                        <span class="canvas-scale">{{ Math.round(canvasScale * 100) }}%</span>
                    </div>
                </el-tooltip>

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
                        <PageTree />
                    </el-tab-pane>
                    <el-tab-pane label="组件库" name="library">
                        <ComponentLibrary @drag-start="handleDragStart" @drag-end="handleDragEnd" />
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
                    <el-tab-pane name="variables">
                        <template #label>
                            <span class="variable-tab-label">变量</span>
                        </template>
                        <VariablePanel />
                    </el-tab-pane>
                </el-tabs>
            </div>
        </div>

        <!-- 底部状态栏 -->
        <div class="design-statusbar">
            <span class="status-item"> 组件: {{ componentCount }} </span>
        </div>

        <!-- 画布设置对话框 -->
        <el-dialog v-model="showCanvasSettings" title="画布设置" width="500px" :close-on-click-modal="false" :lock-scroll="false">
            <el-form label-width="80px" label-position="left">
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
                    <el-switch v-model="freeLayout" />
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
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
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
import { PageTree, ComponentTree, ComponentLibrary, PropertyPanel, VariablePanel } from '@/components/panels';
import { DesignCanvas, CanvasRuler } from '@/components/canvas';
import ContextMenu from '@/components/canvas/ContextMenu.vue';
import { registerAllComponents } from '@/registry/components';

// Route
const route = useRoute();

// Store
const designStore = useDesignStore();

// Composables
const { canvasState } = useCanvas();

// State
const leftActiveTab = ref('pages');
const rightActiveTab = ref('tree');
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
const freeLayout = ref(true);
const canvasScalePercent = ref(100);

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
 * 确认画布设置
 */
function handleCanvasSettingsConfirm() {
    canvasState.scale = canvasScalePercent.value / 100;
    showCanvasSettings.value = false;
    ElMessage.success('画布设置已更新');
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
