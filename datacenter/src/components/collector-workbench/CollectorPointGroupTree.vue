<template>
  <div class="collector-group-tree">
    <div class="collector-group-tree__head">
      <strong>点位分组</strong><el-button text @click="createGroup">新增</el-button>
    </div>
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
  const result = await ElMessageBox.prompt('请输入分组名称', '新增点位分组')
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
  width: 220px;
  min-width: 220px;
  padding-right: 16px;
  border-right: 1px solid #e4e9ec;
}
.collector-group-tree__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
</style>
