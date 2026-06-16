<template>
  <WorkbenchGroupDialog
    ref="dialogRef"
    v-model="visible"
    :mode="mode"
    :title="mode === 'create' ? '新建变量组' : '编辑变量组'"
    :group="groupValue"
    :group-options="groupOptions"
    :initial-parent-id="defaultParentId || null"
    :loading="loading"
    @submit="$emit('submit', $event)"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import WorkbenchGroupDialog from '@/components/workbench/WorkbenchGroupDialog.vue'
import type { OpcuaNodeGroup } from './types'

type OpcuaGroupNode = OpcuaNodeGroup & {
  children: OpcuaGroupNode[]
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    mode: 'create' | 'edit'
    groups: OpcuaNodeGroup[]
    groupValue?: OpcuaNodeGroup | null
    defaultParentId?: string
    loading?: boolean
  }>(),
  {
    groupValue: null,
    defaultParentId: '',
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: { name: string; parentId: string | null }): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const dialogRef = ref<InstanceType<typeof WorkbenchGroupDialog> | null>(null)

const blockedIds = computed(() => {
  if (props.mode !== 'edit' || !props.groupValue) return new Set<string>()
  return collectGroupIds(props.groupValue.id)
})

const groupOptions = computed(() => flattenOpcuaGroups(props.groups, blockedIds.value))

function flattenOpcuaGroups(groups: OpcuaNodeGroup[], blocked: Set<string>) {
  const nodes = buildGroupTree(groups)
  const visit = (group: OpcuaGroupNode, depth: number): Array<{ id: string; label: string }> => {
    const children = group.children.flatMap((child) => visit(child, depth + 1))
    if (blocked.has(group.id)) return children
    return [{ id: group.id, label: `${'　'.repeat(depth)}${group.name}` }, ...children]
  }
  return nodes.flatMap((group) => visit(group, 0))
}

function collectGroupIds(rootId: string) {
  const result = new Set<string>()
  const childrenByParent = new Map<string, OpcuaNodeGroup[]>()
  props.groups.forEach((group) => {
    const parentId = group.parentId || ''
    childrenByParent.set(parentId, [...(childrenByParent.get(parentId) || []), group])
  })
  const visit = (id: string) => {
    result.add(id)
    const children = childrenByParent.get(id) || []
    children.forEach((child) => visit(child.id))
  }
  visit(rootId)
  return result
}

function buildGroupTree(groups: OpcuaNodeGroup[]) {
  const nodeMap = new Map<string, OpcuaGroupNode>()
  groups.forEach((group) => {
    nodeMap.set(group.id, { ...group, children: [] })
  })

  const roots: OpcuaGroupNode[] = []
  nodeMap.forEach((node) => {
    const parent = node.parentId ? nodeMap.get(node.parentId) : null
    if (parent) {
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  })

  const sortNodes = (items: OpcuaGroupNode[]) => {
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )
    items.forEach((item) => sortNodes(item.children))
  }
  sortNodes(roots)
  return roots
}

function closeSilently() {
  dialogRef.value?.closeSilently()
}

defineExpose({ closeSilently })
</script>
