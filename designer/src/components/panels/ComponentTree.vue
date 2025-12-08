<template>
    <div class="component-tree">
        <!-- 标题 -->
        <div class="component-tree-header">
            <span class="header-title">组件树</span>
        </div>

        <!-- 空状态 -->
        <div v-if="!components.length" class="empty-state">
            <el-empty description="暂无组件" :image-size="60" />
        </div>

        <!-- 组件树 -->
        <el-tree
            v-else
            ref="treeRef"
            :data="components"
            :props="treeProps"
            node-key="id"
            :highlight-current="true"
            :expand-on-click-node="false"
            :default-expand-all="true"
            :current-node-key="selectedComponentId"
            draggable
            :allow-drop="allowDrop"
            :allow-drag="allowDrag"
            @node-click="handleNodeClick"
            @node-drop="handleNodeDrop">
            <template #default="{ node, data }">
                <div class="tree-node" :class="{ 'tree-node--locked': data.locked }">
                    <el-icon class="node-icon">
                        <Lock v-if="data.locked" />
                        <component :is="getComponentIcon(data.type)" v-else />
                    </el-icon>
                    <span class="node-label">{{ data.label || data.type }}</span>
                    <el-icon v-if="data.locked" class="lock-icon" title="已锁定">
                        <Lock />
                    </el-icon>
                </div>
            </template>
        </el-tree>
    </div>
</template>

<script setup>
/**
 * ComponentTree - 组件树组件
 * 显示当前页面组件层级，支持选择、拖拽重排序
 * Requirements: 5.1, 5.2, 5.3, 5.4, 5.5
 */
import { ref, computed, watch } from 'vue';
import { useDesignStore } from '@/store/design';
import { getComponent } from '@/registry';
import { Lock, Document, Folder, Picture, EditPen, Pointer } from '@element-plus/icons-vue';

// Store
const designStore = useDesignStore();

// Refs
const treeRef = ref(null);

// Tree props
const treeProps = {
    children: 'children',
    label: 'label',
};

// Computed
/**
 * 组件列表
 * Requirements: 5.1 - 显示组件层级结构
 */
const components = computed(() => designStore.components);

/**
 * 当前选中的组件 ID
 */
const selectedComponentId = computed(() => designStore.selectedComponentId);

// Methods

/**
 * 获取组件图标
 */
function getComponentIcon(type) {
    const iconMap = {
        Container: Folder,
        Text: Document,
        Button: Pointer,
        Image: Picture,
        Input: EditPen,
    };
    return iconMap[type] || Document;
}

/**
 * 处理节点点击
 * Requirements: 5.2 - 点击组件时在画布中选中
 */
function handleNodeClick(data) {
    // Requirements: 5.5 - 锁定的组件不能选择
    if (data.locked) {
        return;
    }
    designStore.selectComponent(data.id);
}

/**
 * 判断是否允许拖拽
 * Requirements: 5.5 - 锁定的组件不能拖拽
 */
function allowDrag(node) {
    return !node.data.locked;
}

/**
 * 判断是否允许放置
 * Requirements: 5.3 - 支持拖拽重排序
 */
function allowDrop(draggingNode, dropNode, type) {
    // 不能放到锁定的组件内部
    if (dropNode.data.locked && type === 'inner') {
        return false;
    }
    // 只有容器类型可以接收子组件
    if (type === 'inner' && dropNode.data.type !== 'Container') {
        return false;
    }
    return true;
}

/**
 * 处理节点拖拽放置
 * Requirements: 5.3 - 拖拽重排序或重新父级
 */
function handleNodeDrop(draggingNode, dropNode, dropType, event) {
    const componentId = draggingNode.data.id;
    let targetParentId = null;
    let index = 0;

    if (dropType === 'inner') {
        // 放到节点内部
        targetParentId = dropNode.data.id;
        index = dropNode.data.children?.length || 0;
    } else if (dropType === 'before') {
        // 放到节点前面
        targetParentId = dropNode.parent?.data?.id || null;
        const siblings = dropNode.parent?.data?.children || designStore.components;
        index = siblings.findIndex((c) => c.id === dropNode.data.id);
    } else if (dropType === 'after') {
        // 放到节点后面
        targetParentId = dropNode.parent?.data?.id || null;
        const siblings = dropNode.parent?.data?.children || designStore.components;
        index = siblings.findIndex((c) => c.id === dropNode.data.id) + 1;
    }

    designStore.moveComponent(componentId, targetParentId, index);
}

// 监听选中变化，同步树的高亮
watch(selectedComponentId, (newId) => {
    if (newId && treeRef.value) {
        treeRef.value.setCurrentKey(newId);
    }
});
</script>

<style scoped>
.component-tree {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: #fff;
}

.component-tree-header {
    padding: 12px;
    border-bottom: 1px solid #e4e7ed;
    font-weight: 500;
}

.header-title {
    font-size: 14px;
    color: #303133;
}

.empty-state {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
}

.tree-node {
    display: flex;
    align-items: center;
    gap: 4px;
    flex: 1;
    min-width: 0;
}

.tree-node--locked {
    opacity: 0.6;
    cursor: not-allowed;
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

.lock-icon {
    flex-shrink: 0;
    color: #e6a23c;
    font-size: 12px;
}

:deep(.el-tree) {
    flex: 1;
    overflow: auto;
    padding: 8px;
}

:deep(.el-tree-node__content) {
    height: 32px;
}

:deep(.el-tree-node.is-current > .el-tree-node__content) {
    background-color: #ecf5ff;
}
</style>
