<template>
  <el-dialog :model-value="modelValue" :title="mode === 'edit' ? '编辑寄存器组' : '新建寄存器组'" width="420px" @close="$emit('update:modelValue', false)">
    <el-form label-position="top">
      <el-form-item label="寄存器组名称">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="父级寄存器组">
        <el-select v-model="form.parentId" clearable style="width: 100%">
          <el-option v-for="group in groups" :key="group.id" :label="group.name" :value="group.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="说明">
        <el-input v-model="form.description" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import type { ModbusRegisterGroup } from './types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  groups: ModbusRegisterGroup[]
  groupValue?: ModbusRegisterGroup | null
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', payload: Record<string, unknown>): void
}>()

const form = reactive({ name: '', parentId: '', description: '' })

watch(
  () => [props.modelValue, props.groupValue],
  () => {
    form.name = props.groupValue?.name || ''
    form.parentId = props.groupValue?.parentId || ''
    form.description = props.groupValue?.description || ''
  },
  { immediate: true },
)

const submit = () => {
  emit('submit', {
    name: form.name.trim(),
    parentId: form.parentId || null,
    description: form.description.trim() || null,
    sortOrder: props.groupValue?.sortOrder || 0,
  })
}
</script>
