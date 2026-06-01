<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    title="移动到分组"
    width="460px"
    body-max-height="260px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="workbench-move-dialog">
      <el-form-item label="目标分组">
        <el-select
          v-model="groupId"
          class="workbench-move-dialog__select"
          clearable
          placeholder="未分组"
        >
          <el-option label="未分组" :value="null" />
          <el-option
            v-for="group in groupOptions"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="workbench-move-dialog__footer">
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
import DcDialog from '@/components/shared/DcDialog.vue'

type WorkbenchGroupOption = {
  id: string
  name: string
  virtual?: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    groups: WorkbenchGroupOption[]
    currentGroupId?: string | null
    loading?: boolean
  }>(),
  {
    currentGroupId: null,
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', groupId: string | null): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const groupId = ref<string | null>(null)
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const normalizedCurrentGroupId = computed(() => props.currentGroupId || null)
const groupOptions = computed(() => props.groups.filter((group) => !group.virtual))
const canSubmit = computed(() => groupId.value !== normalizedCurrentGroupId.value && !props.loading)
const isDirty = computed(() => visible.value && groupId.value !== normalizedCurrentGroupId.value)

function resetForm() {
  groupId.value = normalizedCurrentGroupId.value
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', groupId.value || null)
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm()
  },
)

watch(
  () => props.currentGroupId,
  () => {
    if (props.modelValue) resetForm()
  },
)

defineExpose({
  closeSilently: () => dialogRef.value?.closeSilently(),
})
</script>

<style scoped>
.workbench-move-dialog {
  display: grid;
  gap: 2px;
}

.workbench-move-dialog__select {
  width: 100%;
}

.workbench-move-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
