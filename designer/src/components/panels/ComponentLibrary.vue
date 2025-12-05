<template>
  <div class="component-library">
    <!-- 标题 -->
    <div class="component-library-header">
      <span class="header-title">组件库</span>
    </div>
    
    <!-- 分类组件列表 -->
    <div class="component-categories">
      <el-collapse v-model="activeCategories">
        <el-collapse-item 
          v-for="(components, category) in componentsByCategory" 
          :key="category"
          :name="category"
          :title="category"
        >
          <div class="component-grid">
            <div
              v-for="component in components"
              :key="component.type"
              class="component-item"
              draggable="true"
              @dragstart="handleDragStart($event, component)"
              @dragend="handleDragEnd"
            >
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
 * Requirements: 8.1, 8.2
 */
import { ref, computed, onMounted } from 'vue'
import { getComponentsByCategory, createComponentInstance } from '@/registry'
import { registerBasicComponents } from '@/registry/components'
import { 
  Folder, 
  Document, 
  Pointer, 
  Picture, 
  EditPen,
  Grid
} from '@element-plus/icons-vue'

// Emits
const emit = defineEmits(['drag-start', 'drag-end'])

// State
const activeCategories = ref(['Layout', 'Basic', 'Form'])

// Computed
/**
 * 按分类分组的组件
 * Requirements: 8.1 - 按分类显示可用组件
 */
const componentsByCategory = computed(() => getComponentsByCategory())

// Methods

/**
 * 获取图标组件
 */
function getIcon(iconName) {
  const iconMap = {
    folder: Folder,
    document: Document,
    pointer: Pointer,
    picture: Picture,
    'edit-pen': EditPen,
    grid: Grid,
  }
  return iconMap[iconName] || Document
}

/**
 * 处理拖拽开始
 * Requirements: 8.2 - 支持拖拽到画布
 */
function handleDragStart(event, component) {
  // 创建组件实例
  const instance = createComponentInstance(component.type)
  
  // 设置拖拽数据
  event.dataTransfer.setData('application/json', JSON.stringify(instance))
  event.dataTransfer.effectAllowed = 'copy'
  
  // 设置拖拽图像
  const dragImage = event.target.cloneNode(true)
  dragImage.style.position = 'absolute'
  dragImage.style.top = '-1000px'
  document.body.appendChild(dragImage)
  event.dataTransfer.setDragImage(dragImage, 0, 0)
  setTimeout(() => document.body.removeChild(dragImage), 0)
  
  emit('drag-start', instance)
}

/**
 * 处理拖拽结束
 */
function handleDragEnd() {
  emit('drag-end')
}

// 初始化时注册基础组件
onMounted(() => {
  registerBasicComponents()
})
</script>

<style scoped>
.component-library {
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: #fff;
}

.component-library-header {
  padding: 12px;
  border-bottom: 1px solid #e4e7ed;
  font-weight: 500;
}

.header-title {
  font-size: 14px;
  color: #303133;
}

.component-categories {
  flex: 1;
  overflow: auto;
}

.component-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  padding: 8px;
}

.component-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 12px 8px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  cursor: grab;
  transition: all 0.2s;
  background-color: #fafafa;
}

.component-item:hover {
  border-color: #409eff;
  background-color: #ecf5ff;
}

.component-item:active {
  cursor: grabbing;
}

.component-icon {
  font-size: 24px;
  color: #606266;
  margin-bottom: 4px;
}

.component-name {
  font-size: 12px;
  color: #606266;
  text-align: center;
}

:deep(.el-collapse) {
  border: none;
}

:deep(.el-collapse-item__header) {
  padding: 0 12px;
  font-size: 13px;
  font-weight: 500;
  background-color: #f5f7fa;
}

:deep(.el-collapse-item__content) {
  padding-bottom: 0;
}
</style>
