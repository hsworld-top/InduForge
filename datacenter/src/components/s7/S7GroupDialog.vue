<template>
  <DcDialog v-model="visible" :title="mode === 'edit' ? '编辑变量组' : '新建变量组'" width="420px">
    <el-form label-position="top">
      <el-form-item label="变量组名称">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="Code">
        <el-input v-model="form.code" />
      </el-form-item>
      <el-form-item label="父级变量组">
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
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">保存</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { S7VariableGroup } from './types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  groups: S7VariableGroup[]
  groupValue?: S7VariableGroup | null
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

const form = reactive({ name: '', code: '', parentId: '', description: '' })

watch(
  () => [props.modelValue, props.groupValue],
  () => {
    form.name = props.groupValue?.name || ''
    form.code = props.groupValue?.code || ''
    form.parentId =
      props.mode === 'create' ? props.defaultParentId || '' : props.groupValue?.parentId || ''
    form.description = props.groupValue?.description || ''
  },
  { immediate: true },
)

const submit = () => {
  emit('submit', {
    name: form.name.trim(),
    code: form.code.trim(),
    parentId: form.parentId || null,
    description: form.description.trim() || null,
    sortOrder: props.groupValue?.sortOrder || 0,
  })
}
</script>
