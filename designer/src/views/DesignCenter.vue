<template>
  <div class="design-center">
    <!-- 顶部工具栏 -->
    <div class="design-toolbar">
      <div class="toolbar-left">
        <span class="project-name">{{ projectName }}</span>
        <el-divider direction="vertical" />
        <span class="page-name">{{ currentPageName }}</span>
        <el-tag v-if="isDirty" type="warning" size="small" class="dirty-tag">
          未保存
        </el-tag>
      </div>
      <div class="toolbar-center">
        <!-- 撤销/重做按钮 -->
        <el-button-group>
          <el-tooltip content="撤销 (Ctrl+Z)" placement="bottom">
            <el-button :icon="RefreshLeft" :disabled="!canUndo" @click="handleUndo" />
          </el-tooltip>
          <el-tooltip content="重做 (Ctrl+Y)" placement="bottom">
            <el-button :icon="RefreshRight" :disabled="!canRedo" @click="handleRedo" />
          </el-tooltip>
        </el-button-group>
      </div>
      <div class="toolbar-right">
        <!-- 保存按钮 -->
        <el-button 
          type="primary" 
          :icon="Check" 
          :loading="saving"
          :disabled="!isDirty"
          @click="handleSave"
        >
          保存
        </el-button>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="design-main">
      <!-- 左侧面板 -->
      <div class="left-panel">
        <el-tabs v-model="leftActiveTab" class="panel-tabs">
          <el-tab-pane label="页面" name="pages">
            <PageTree />
          </el-tab-pane>
          <el-tab-pane label="组件树" name="tree">
            <ComponentTree />
          </el-tab-pane>
          <el-tab-pane label="组件库" name="library">
            <ComponentLibrary @drag-start="handleDragStart" @drag-end="handleDragEnd" />
          </el-tab-pane>
        </el-tabs>
      </div>

      <!-- 中间画布区域 -->
      <div 
        class="canvas-area"
        @dragover.prevent="handleDragOver"
        @drop="handleDrop"
      >
        <DesignCanvas v-if="currentPage" :show-grid="showGrid" />
        <div v-else class="canvas-empty">
          <el-empty description="请从左侧选择一个页面开始设计" />
        </div>
      </div>

      <!-- 右侧属性面板 -->
      <div class="right-panel">
        <PropertyPanel />
      </div>
    </div>

    <!-- 底部状态栏 -->
    <div class="design-statusbar">
      <span class="status-item">
        缩放: {{ Math.round(canvasScale * 100) }}%
      </span>
      <el-divider direction="vertical" />
      <span class="status-item">
        组件: {{ componentCount }}
      </span>
    </div>
  </div>
</template>

<script setup>
/**
 * DesignCenter - 设计中心主视图
 * 集成三栏布局：PageTree + ComponentTree + ComponentLibrary | Canvas | PropertyPanel
 * Requirements: 1.1, 1.3, 6.2
 */
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { RefreshLeft, RefreshRight, Check } from '@element-plus/icons-vue'
import { useDesignStore } from '@/store/design'
import { useCanvas } from '@/composables/useCanvas'
import { PageTree, ComponentTree, ComponentLibrary, PropertyPanel } from '@/components/panels'
import { DesignCanvas } from '@/components/canvas'
import { registerBasicComponents } from '@/registry/components'

// Route
const route = useRoute()

// Store
const designStore = useDesignStore()

// Composables
const { canvasState } = useCanvas()

// State
const leftActiveTab = ref('pages')
const showGrid = ref(true)
const draggingComponent = ref(null)

// 撤销/重做功能（预留）
const canUndo = ref(false)
const canRedo = ref(false)

// Computed
const projectId = computed(() => route.query.pid || route.query.id)
const projectName = computed(() => route.query.name || '项目')
const currentPage = computed(() => designStore.currentPage)
const currentPageName = computed(() => currentPage.value?.meta?.name || '未选择页面')
const isDirty = computed(() => designStore.isDirty)
const saving = computed(() => designStore.saving)
const canvasScale = computed(() => canvasState.scale)
const componentCount = computed(() => {
  const countComponents = (components) => {
    let count = 0
    for (const comp of components || []) {
      count += 1
      if (comp.children) {
        count += countComponents(comp.children)
      }
    }
    return count
  }
  return countComponents(designStore.components)
})

// Methods

/**
 * 加载项目
 * Requirements: 1.1 - 打开设计中心时显示页面树
 */
async function loadProject() {
  const pid = projectId.value
  if (!pid) {
    ElMessage.warning('未指定项目ID')
    return
  }

  try {
    await designStore.loadProject(pid)
  } catch (error) {
    ElMessage.error('加载项目失败: ' + error.message)
  }
}

/**
 * 保存页面
 * Requirements: 6.2 - 点击保存按钮时持久化 schema
 */
async function handleSave() {
  try {
    await designStore.savePage()
    ElMessage.success('保存成功')
  } catch (error) {
    // Requirements: 6.4 - 显示保存错误信息
    ElMessage.error('保存失败: ' + error.message)
  }
}

/**
 * 撤销操作（预留）
 */
function handleUndo() {
  // TODO: 实现撤销功能
  ElMessage.info('撤销功能开发中')
}

/**
 * 重做操作（预留）
 */
function handleRedo() {
  // TODO: 实现重做功能
  ElMessage.info('重做功能开发中')
}

/**
 * 处理组件拖拽开始
 */
function handleDragStart(component) {
  draggingComponent.value = component
}

/**
 * 处理组件拖拽结束
 */
function handleDragEnd() {
  draggingComponent.value = null
}

/**
 * 处理拖拽经过画布
 */
function handleDragOver(event) {
  event.dataTransfer.dropEffect = 'copy'
}

/**
 * 处理组件放置到画布
 * Requirements: 8.2 - 拖拽组件到画布创建新实例
 */
function handleDrop(event) {
  if (!currentPage.value) {
    ElMessage.warning('请先选择一个页面')
    return
  }

  try {
    const data = event.dataTransfer.getData('application/json')
    if (!data) return

    const component = JSON.parse(data)
    
    // 计算放置位置（相对于画布）
    const canvasArea = event.currentTarget
    const rect = canvasArea.getBoundingClientRect()
    const x = event.clientX - rect.left
    const y = event.clientY - rect.top

    // 设置组件位置
    component.style = {
      ...component.style,
      left: Math.round(x / canvasState.scale),
      top: Math.round(y / canvasState.scale),
    }

    // 添加组件到页面
    designStore.addComponent(component)
    
    // 选中新添加的组件
    designStore.selectComponent(component.id)
  } catch (error) {
    console.error('Drop error:', error)
  }
}

/**
 * 处理键盘快捷键
 */
function handleKeydown(event) {
  // Ctrl+S 保存
  if (event.ctrlKey && event.key === 's') {
    event.preventDefault()
    if (isDirty.value) {
      handleSave()
    }
  }
  // Ctrl+Z 撤销
  if (event.ctrlKey && event.key === 'z') {
    event.preventDefault()
    handleUndo()
  }
  // Ctrl+Y 重做
  if (event.ctrlKey && event.key === 'y') {
    event.preventDefault()
    handleRedo()
  }
  // Delete 删除选中组件
  if (event.key === 'Delete' && designStore.selectedComponentId) {
    const selected = designStore.selectedComponent
    if (selected && !selected.locked) {
      designStore.removeComponent(designStore.selectedComponentId)
    }
  }
  // Escape 取消选择
  if (event.key === 'Escape') {
    designStore.selectComponent(null)
  }
}

// Lifecycle
onMounted(() => {
  // 注册基础组件
  registerBasicComponents()
  
  // 加载项目
  loadProject()
  
  // 添加键盘事件监听
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  // 移除键盘事件监听
  window.removeEventListener('keydown', handleKeydown)
  
  // 重置 store
  designStore.reset()
})

// 监听路由变化，重新加载项目
watch(() => route.query.pid || route.query.id, (newPid) => {
  if (newPid) {
    loadProject()
  }
})
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

/* 顶部工具栏 */
.design-toolbar {
  height: 48px;
  background-color: #fff;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  flex-shrink: 0;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.project-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.page-name {
  font-size: 14px;
  color: #606266;
}

.dirty-tag {
  margin-left: 8px;
}

.toolbar-center {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 主内容区域 */
.design-main {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* 左侧面板 */
.left-panel {
  width: 280px;
  background-color: #fff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
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
  overflow: hidden;
  position: relative;
  background-color: #f0f2f5;
}

.canvas-empty {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 右侧属性面板 */
.right-panel {
  width: 300px;
  background-color: #fff;
  border-left: 1px solid #e4e7ed;
  flex-shrink: 0;
  overflow: hidden;
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

/* 响应式调整 */
@media (max-width: 1200px) {
  .left-panel {
    width: 240px;
  }
  
  .right-panel {
    width: 260px;
  }
}
</style>
