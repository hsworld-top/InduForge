<template>
  <div class="page-tree">
    <!-- 工具栏 -->
    <div class="page-tree-toolbar">
      <el-button size="small" @click="handleCreatePage" :icon="Plus">
        页面
      </el-button>
      <el-button size="small" @click="handleCreateFolder" :icon="Folder">
        文件夹
      </el-button>
    </div>
    
    <!-- 页面树 -->
    <el-tree
      ref="treeRef"
      :data="pageTree"
      :props="treeProps"
      node-key="id"
      :highlight-current="true"
      :expand-on-click-node="false"
      :default-expand-all="true"
      @node-click="handleNodeClick"
      @node-contextmenu="handleContextMenu"
    >
      <template #default="{ node, data }">
        <div class="tree-node">
          <el-icon class="node-icon">
            <Folder v-if="data.type === 'folder'" />
            <Document v-else />
          </el-icon>
          <span 
            v-if="editingId !== data.id" 
            class="node-label"
          >
            {{ data.name }}
          </span>
          <el-input
            v-else
            v-model="editingName"
            size="small"
            @blur="handleRenameConfirm(data)"
            @keyup.enter="handleRenameConfirm(data)"
            @keyup.escape="handleRenameCancel"
            ref="editInputRef"
            class="node-input"
          />
        </div>
      </template>
    </el-tree>
    
    <!-- 右键菜单 -->
    <div
      v-if="contextMenuVisible"
      class="context-menu"
      :style="contextMenuStyle"
    >
      <div 
        class="context-menu-item"
        @click="handleRename"
      >
        <el-icon><Edit /></el-icon>
        <span>重命名</span>
      </div>
      <div 
        class="context-menu-item context-menu-item--danger"
        @click="handleDelete"
      >
        <el-icon><Delete /></el-icon>
        <span>删除</span>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * PageTree - 页面树组件
 * 显示项目页面层级结构，支持创建、重命名、删除页面和文件夹
 * Requirements: 1.1, 1.2, 1.4, 1.5, 1.6
 */
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { useDesignStore } from '@/store/design'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Folder, Document, Edit, Delete } from '@element-plus/icons-vue'

// Store
const designStore = useDesignStore()

// Refs
const treeRef = ref(null)
const editInputRef = ref(null)

// State
const editingId = ref(null)
const editingName = ref('')
const contextMenuVisible = ref(false)
const contextMenuPosition = ref({ x: 0, y: 0 })
const contextMenuNode = ref(null)

// Tree props
const treeProps = {
  children: 'children',
  label: 'name',
}

// Computed
/**
 * 页面树数据
 * Requirements: 1.1 - 显示项目页面层级结构
 */
const pageTree = computed(() => designStore.pageTree)

const contextMenuStyle = computed(() => ({
  left: `${contextMenuPosition.value.x}px`,
  top: `${contextMenuPosition.value.y}px`,
}))

// Methods

/**
 * 处理节点点击
 * Requirements: 1.3 - 选择页面时加载 schema 并显示在画布
 */
async function handleNodeClick(data) {
  if (data.type === 'folder') {
    return
  }
  
  try {
    await designStore.loadPage(data.id)
  } catch (error) {
    ElMessage.error('加载页面失败: ' + error.message)
  }
}

/**
 * 创建新页面
 * Requirements: 1.2 - 创建新页面
 */
async function handleCreatePage() {
  try {
    const page = await designStore.createPage('新页面', null, 'page')
    ElMessage.success('页面创建成功')
    // 开始编辑名称
    editingId.value = page.id
    editingName.value = page.name
    await nextTick()
    editInputRef.value?.focus()
  } catch (error) {
    ElMessage.error('创建页面失败: ' + error.message)
  }
}

/**
 * 创建文件夹
 * Requirements: 1.6 - 创建文件夹
 */
async function handleCreateFolder() {
  try {
    const folder = await designStore.createPage('新文件夹', null, 'folder')
    ElMessage.success('文件夹创建成功')
    // 开始编辑名称
    editingId.value = folder.id
    editingName.value = folder.name
    await nextTick()
    editInputRef.value?.focus()
  } catch (error) {
    ElMessage.error('创建文件夹失败: ' + error.message)
  }
}

/**
 * 处理右键菜单
 */
function handleContextMenu(event, data, node) {
  event.preventDefault()
  contextMenuNode.value = data
  contextMenuPosition.value = { x: event.clientX, y: event.clientY }
  contextMenuVisible.value = true
}

/**
 * 关闭右键菜单
 */
function closeContextMenu() {
  contextMenuVisible.value = false
  contextMenuNode.value = null
}

/**
 * 开始重命名
 * Requirements: 1.4 - 重命名页面
 */
function handleRename() {
  if (!contextMenuNode.value) return
  
  editingId.value = contextMenuNode.value.id
  editingName.value = contextMenuNode.value.name
  closeContextMenu()
  
  nextTick(() => {
    editInputRef.value?.focus()
  })
}

/**
 * 确认重命名
 */
async function handleRenameConfirm(data) {
  if (!editingId.value || !editingName.value.trim()) {
    handleRenameCancel()
    return
  }
  
  const newName = editingName.value.trim()
  if (newName === data.name) {
    handleRenameCancel()
    return
  }
  
  try {
    await designStore.renamePage(editingId.value, newName)
    ElMessage.success('重命名成功')
  } catch (error) {
    ElMessage.error('重命名失败: ' + error.message)
  } finally {
    handleRenameCancel()
  }
}

/**
 * 取消重命名
 */
function handleRenameCancel() {
  editingId.value = null
  editingName.value = ''
}

/**
 * 删除页面/文件夹
 * Requirements: 1.5 - 删除页面
 */
async function handleDelete() {
  if (!contextMenuNode.value) return
  
  const node = contextMenuNode.value
  closeContextMenu()
  
  try {
    await ElMessageBox.confirm(
      `确定要删除 "${node.name}" 吗？${node.type === 'folder' ? '文件夹内的页面也会被删除。' : ''}`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    await designStore.deletePage(node.id)
    ElMessage.success('删除成功')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + error.message)
    }
  }
}

// 点击外部关闭右键菜单
function handleClickOutside(event) {
  if (contextMenuVisible.value) {
    closeContextMenu()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.page-tree {
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: #fff;
}

.page-tree-toolbar {
  padding: 8px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  gap: 8px;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.node-icon {
  flex-shrink: 0;
  color: #909399;
}

.node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-input {
  flex: 1;
}

.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  z-index: 3000;
  min-width: 120px;
  padding: 4px 0;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  cursor: pointer;
  font-size: 14px;
  color: #606266;
}

.context-menu-item:hover {
  background-color: #f5f7fa;
}

.context-menu-item--danger {
  color: #f56c6c;
}

.context-menu-item--danger:hover {
  background-color: #fef0f0;
}

:deep(.el-tree) {
  flex: 1;
  overflow: auto;
  padding: 8px;
}

:deep(.el-tree-node__content) {
  height: 32px;
}
</style>
