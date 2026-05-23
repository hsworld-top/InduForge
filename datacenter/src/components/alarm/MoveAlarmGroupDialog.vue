<template>
  <DcDialog
    v-model="visible"
    title="移动分组"
    width="460px"
    body-max-height="260px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="alarm-group-move-dialog">
      <el-form-item label="目标上级分组">
        <el-select
          v-model="parentId"
          class="alarm-group-move-dialog__select"
          clearable
          placeholder="根目录"
        >
          <el-option label="根目录" :value="null" />
          <el-option
            v-for="group in groupOptions"
            :key="group.id"
            :label="group.label"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="alarm-group-move-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">
          移动
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { AlarmPolicyGroup } from '@/api/schemas/alarm.schema'
import DcDialog from '@/components/shared/DcDialog.vue'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    group: AlarmPolicyGroup | null
    groups: AlarmPolicyGroup[]
    loading?: boolean
  }>(),
  {
    loading: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [parentId: string | null]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const parentId = ref<string | null>(null)
const currentParentId = computed(() => props.group?.parentId || null)

const blockedIds = computed(() => {
  const ids = new Set<string>()
  const collect = (groupId?: string | null) => {
    if (!groupId || ids.has(groupId)) return
    ids.add(groupId)
    props.groups
      .filter((group) => (group.parentId || null) === groupId)
      .forEach((group) => collect(group.id))
  }
  collect(props.group?.id)
  return ids
})

const flattenGroups = (
  groups: AlarmPolicyGroup[],
  parentGroupId: string | null = null,
  depth = 0,
): Array<{ id: string; label: string }> =>
  groups
    .filter((group) => (group.parentId || null) === parentGroupId)
    .flatMap((group) => {
      const children = flattenGroups(groups, group.id, depth + 1)
      if (blockedIds.value.has(group.id)) return children
      return [{ id: group.id, label: `${'　'.repeat(depth)}${group.name}` }, ...children]
    })

const groupOptions = computed(() => flattenGroups(props.groups))

const canSubmit = computed(
  () => Boolean(props.group) && parentId.value !== currentParentId.value && !props.loading,
)
const isDirty = computed(
  () => visible.value && Boolean(props.group) && parentId.value !== currentParentId.value,
)

function resetForm() {
  parentId.value = currentParentId.value
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', parentId.value || null)
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm()
  },
)

watch(
  () => props.group?.id,
  () => {
    if (props.modelValue) resetForm()
  },
)
</script>

<style scoped>
.alarm-group-move-dialog {
  display: grid;
  gap: 2px;
}

.alarm-group-move-dialog__select {
  width: 100%;
}

.alarm-group-move-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
