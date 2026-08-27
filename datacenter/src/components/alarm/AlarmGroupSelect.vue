<template>
  <div class="alarm-group-select">
    <el-tree-select
      v-model="value"
      :data="treeOptions"
      :props="treeProps"
      node-key="id"
      check-strictly
      clearable
      filterable
      default-expand-all
      :render-after-expand="false"
      :disabled="disabled"
      :loading="loading"
      placeholder="根目录（未分组）"
      empty-text="暂无目录"
      @visible-change="handleVisible"
    />
    <small v-if="selectedPath" :title="selectedPath">{{ selectedPath }}</small>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { listAlarmGroups } from '@/api/alarm.api'
import type { AlarmGroup } from '@/api/schemas/alarm.schema'
import { getApiErrorMessage } from '@/utils/request'

type AlarmGroupTreeNode = AlarmGroup & { children: AlarmGroupTreeNode[] }

const props = withDefaults(
  defineProps<{
    modelValue?: string | null
    projectId: string
    disabled?: boolean
    initialLabel?: string
  }>(),
  { modelValue: null, disabled: false, initialLabel: '' },
)
const emit = defineEmits<{ 'update:modelValue': [value: string | null] }>()
const value = computed({
  get: () => props.modelValue || null,
  set: (next) => emit('update:modelValue', next || null),
})
const groups = ref<AlarmGroup[]>([])
const loading = ref(false)
let requestVersion = 0

const treeProps = { value: 'id', label: 'name', children: 'children' }
const treeOptions = computed(() => {
  const nodes = new Map<string, AlarmGroupTreeNode>()
  for (const group of groups.value) nodes.set(group.id, { ...group, children: [] })
  const roots: AlarmGroupTreeNode[] = []
  for (const group of groups.value) {
    const node = nodes.get(group.id)
    if (!node) continue
    const parent = group.parentId ? nodes.get(group.parentId) : undefined
    if (parent) parent.children.push(node)
    else roots.push(node)
  }
  const sortNodes = (items: AlarmGroupTreeNode[]) => {
    items.sort(
      (left, right) => left.sortOrder - right.sortOrder || left.name.localeCompare(right.name),
    )
    items.forEach((item) => sortNodes(item.children))
  }
  sortNodes(roots)
  return roots
})
const selectedPath = computed(
  () => groups.value.find((group) => group.id === props.modelValue)?.fullPath || props.initialLabel,
)

async function loadTree() {
  if (!props.projectId || loading.value) return
  const version = ++requestVersion
  loading.value = true
  try {
    const result = await listAlarmGroups(props.projectId, { view: 'tree' })
    if (version === requestVersion) groups.value = result.list
  } catch (error) {
    if (version === requestVersion) ElMessage.error(getApiErrorMessage(error, '加载报警目录失败'))
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

function handleVisible(open: boolean) {
  if (open && groups.value.length === 0) void loadTree()
}

watch(
  () => props.projectId,
  (projectId) => {
    groups.value = []
    if (projectId) void loadTree()
  },
  { immediate: true },
)
</script>

<style scoped>
.alarm-group-select {
  min-width: 0;
  display: grid;
  gap: 4px;
}
.alarm-group-select small {
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 10px;
  line-height: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
