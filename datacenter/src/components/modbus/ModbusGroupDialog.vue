<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    :title="mode === 'edit' ? '编辑寄存器组' : '新建寄存器组'"
    width="420px"
    body-max-height="320px"
    :dirty="isDirty"
    :close-disabled="loading"
  >
    <el-form label-position="top">
      <el-form-item label="寄存器组名称">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="父级寄存器组">
        <el-select v-model="form.parentId" clearable style="width: 100%">
          <el-option
            v-for="group in groups"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="说明">
        <el-input v-model="form.description" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="requestClose">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">保存</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { ModbusRegisterGroup } from './types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  groups: ModbusRegisterGroup[]
  groupValue?: ModbusRegisterGroup | null
  defaultParentId?: string
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', payload: Record<string, unknown>): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const initialSnapshot = ref('')
const form = reactive({ name: '', parentId: '', description: '' })

watch(
  () => [props.modelValue, props.groupValue],
  () => {
    if (!props.modelValue) return
    form.name = props.groupValue?.name || ''
    form.parentId =
      props.mode === 'create' ? props.defaultParentId || '' : props.groupValue?.parentId || ''
    form.description = props.groupValue?.description || ''
    initialSnapshot.value = snapshotForm()
  },
  { immediate: true },
)

const isDirty = computed(() => props.modelValue && snapshotForm() !== initialSnapshot.value)

const submit = () => {
  emit('submit', {
    name: form.name.trim(),
    parentId: form.parentId || null,
    description: form.description.trim() || null,
    sortOrder: props.groupValue?.sortOrder || 0,
  })
}

function requestClose() {
  void dialogRef.value?.requestClose()
}

function closeSilently() {
  dialogRef.value?.closeSilently()
}

function snapshotForm() {
  return JSON.stringify({ ...form })
}

defineExpose({ closeSilently })
</script>
