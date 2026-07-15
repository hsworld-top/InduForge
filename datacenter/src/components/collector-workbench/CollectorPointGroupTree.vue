<template>
  <div class="collector-group-tree">
    <div class="collector-group-tree__head">
      <div>
        <small>变量组织</small>
        <strong>变量分组</strong>
      </div>
      <el-button text @click="createGroup">新增</el-button>
    </div>
    <button type="button" class="collector-group-tree__all" @click="emit('select', null)">
      全部变量
    </button>
    <el-tree
      lazy
      :load="loadNode"
      node-key="id"
      :props="{ label: 'name', children: 'children', isLeaf: 'leaf' }"
      highlight-current
      @current-change="emit('select', $event?.id || null)"
    />
  </div>
</template>

<script setup lang="ts">
import { ElMessageBox } from 'element-plus'
import { createCollectorPointGroup, listCollectorPointGroups } from '@/api/collector.api'

const props = defineProps<{ projectId: string; connectionId: string }>()
const emit = defineEmits<{ select: [groupId: string | null] }>()
async function loadNode(
  node: { level: number; data?: { id: string } },
  resolve: (data: unknown[]) => void,
) {
  const groups = await listCollectorPointGroups(
    props.projectId,
    props.connectionId,
    node.level === 0 ? null : node.data?.id,
  )
  resolve(groups.map((group) => ({ ...group, leaf: false })))
}
async function createGroup() {
  const result = await ElMessageBox.prompt('请输入分组名称', '新增变量分组')
  await createCollectorPointGroup(props.projectId, props.connectionId, {
    parentId: null,
    name: result.value,
    sortOrder: 0,
    metadata: {},
  })
  location.reload()
}
</script>

<style scoped>
.collector-group-tree {
  width: 210px;
  min-width: 210px;
  padding-right: 14px;
  border-right: 1px solid var(--dc-border);
}
.collector-group-tree__head {
  display: flex;
  min-height: 48px;
  align-items: center;
  justify-content: space-between;
}
.collector-group-tree__head > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.collector-group-tree__head small {
  color: var(--dc-text-muted);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.08em;
}
.collector-group-tree__head strong {
  color: var(--dc-text-secondary);
  font-size: 13px;
}
.collector-group-tree__all {
  width: 100%;
  height: 32px;
  margin-bottom: 5px;
  padding: 0 10px;
  border: none;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
  text-align: left;
}
.collector-group-tree :deep(.el-tree) {
  --el-tree-node-hover-bg-color: var(--dc-surface-subtle);
  background: transparent;
  color: var(--dc-text-secondary);
}
.collector-group-tree :deep(.el-tree-node__content) {
  height: 32px;
  border-radius: var(--dc-radius-sm);
}
</style>
