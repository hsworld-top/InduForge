<template>
  <div class="property-panel">
    <!-- 空状态 -->
    <div v-if="!selectedComponent" class="empty-state">
      <el-empty description="请选择组件" :image-size="60" />
    </div>
    
    <!-- 属性编辑区域 -->
    <template v-else>
      <!-- 组件信息 -->
      <div class="panel-header">
        <span class="component-type">{{ componentDef?.name || selectedComponent.type }}</span>
        <span class="component-id">{{ selectedComponent.id.slice(0, 8) }}</span>
      </div>
      
      <!-- 标签页 -->
      <el-tabs v-model="activeTab" class="panel-tabs">
        <!-- 属性标签页 -->
        <el-tab-pane label="属性" name="props">
          <div class="props-section">
            <!-- 基础属性 -->
            <div class="section-group">
              <div class="section-title">基础</div>
              <el-form label-position="left" label-width="80px" size="small">
                <el-form-item label="标签">
                  <el-input 
                    v-model="componentLabel" 
                    @change="handleLabelChange"
                  />
                </el-form-item>
                <el-form-item label="可见">
                  <el-switch 
                    v-model="componentVisible" 
                    @change="handleVisibleChange"
                  />
                </el-form-item>
                <el-form-item label="锁定">
                  <el-switch 
                    v-model="componentLocked" 
                    @change="handleLockedChange"
                  />
                </el-form-item>
              </el-form>
            </div>
            
            <!-- 组件特有属性 - 使用 PropsEditor 组件 -->
            <div v-if="propsSchema && Object.keys(propsSchema).length" class="section-group">
              <div class="section-title">组件属性</div>
              <PropsEditor
                :props="selectedComponent.props"
                :props-schema="propsSchema"
                @change="handlePropChange"
              />
            </div>
          </div>
        </el-tab-pane>
        
        <!-- 样式标签页 - 使用 StyleEditor 组件 -->
        <el-tab-pane label="样式" name="style">
          <StyleEditor
            :style="selectedComponent.style"
            @change="handleStyleChange"
          />
        </el-tab-pane>
      </el-tabs>
    </template>
  </div>
</template>

<script setup>
/**
 * PropertyPanel - 属性面板组件
 * 显示选中组件的属性编辑器
 * 包含 StyleEditor 和 PropsEditor
 * Requirements: 3.3, 4.1
 */
import { ref, computed } from 'vue'
import { useDesignStore } from '@/store/design'
import { getComponent } from '@/registry'
import { StyleEditor, PropsEditor } from '@/components/editors'

// Store
const designStore = useDesignStore()

// State
const activeTab = ref('props')

// Computed
/**
 * 选中的组件
 * Requirements: 3.3 - 选中组件时显示属性
 */
const selectedComponent = computed(() => designStore.selectedComponent)

/**
 * 组件定义
 */
const componentDef = computed(() => {
  if (!selectedComponent.value) return null
  return getComponent(selectedComponent.value.type)
})

/**
 * 属性 Schema
 * Requirements: 4.1 - 显示组件属性表单
 */
const propsSchema = computed(() => componentDef.value?.propsSchema || {})

/**
 * 组件标签
 */
const componentLabel = computed({
  get: () => selectedComponent.value?.label || '',
  set: () => {},
})

/**
 * 组件可见性
 */
const componentVisible = computed({
  get: () => selectedComponent.value?.visible !== false,
  set: () => {},
})

/**
 * 组件锁定状态
 */
const componentLocked = computed({
  get: () => selectedComponent.value?.locked === true,
  set: () => {},
})

// Methods

/**
 * 处理标签变更
 */
function handleLabelChange(value) {
  if (!selectedComponent.value) return
  designStore.updateComponent(selectedComponent.value.id, { label: value })
}

/**
 * 处理可见性变更
 */
function handleVisibleChange(value) {
  if (!selectedComponent.value) return
  designStore.updateComponent(selectedComponent.value.id, { visible: value })
}

/**
 * 处理锁定状态变更
 */
function handleLockedChange(value) {
  if (!selectedComponent.value) return
  designStore.updateComponent(selectedComponent.value.id, { locked: value })
}

/**
 * 处理属性变更
 * Requirements: 4.2 - 修改属性时更新 schema
 */
function handlePropChange(key, value) {
  if (!selectedComponent.value) return
  designStore.updateComponent(selectedComponent.value.id, {
    props: { [key]: value },
  })
}

/**
 * 处理样式变更
 * Requirements: 4.3 - 修改样式时更新 schema 并重新渲染
 */
function handleStyleChange(key, value) {
  if (!selectedComponent.value) return
  designStore.updateComponent(selectedComponent.value.id, {
    style: { [key]: value },
  })
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
</style>
