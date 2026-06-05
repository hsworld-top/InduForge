<template>
  <el-dialog
    :model-value="modelValue"
    :title="mode === 'edit' ? '编辑变量组' : '新建变量组'"
    width="420px"
    @close="$emit('update:modelValue', false)"
  >
    <el-form class="opcua-group-form" label-width="86px">
      <el-form-item label="变量组名称" required>
        <el-input v-model="form.name" maxlength="100" />
      </el-form-item>
      <el-form-item label="父级变量组">
        <el-select v-model="form.parentId" clearable placeholder="无">
          <el-option
            v-for="item in groups.filter((group) => group.id !== groupValue?.id)"
            :key="item.id"
            :label="item.name"
            :value="item.id"
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
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { OpcuaNodeGroup } from './types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  groups: OpcuaNodeGroup[]
  groupValue?: OpcuaNodeGroup | null
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (
    event: 'submit',
    value: { name: string; parentId: string | null; description: string | null },
  ): void
}>()

const form = reactive({
  name: '',
  parentId: '',
  description: '',
})

watch(
  () => [props.modelValue, props.groupValue] as const,
  () => {
    if (!props.modelValue) return
    form.name = props.groupValue?.name || ''
    form.parentId = props.groupValue?.parentId || ''
    form.description = props.groupValue?.description || ''
  },
  { immediate: true },
)

const submit = () => {
  if (!form.name.trim()) {
    ElMessage.warning('变量组名称不能为空')
    return
  }
  emit('submit', {
    name: form.name.trim(),
    parentId: form.parentId || null,
    description: form.description.trim() || null,
  })
}
</script>

<style scoped>
.opcua-group-form {
  padding-top: 2px;
}

.opcua-group-form :deep(.el-form-item) {
  margin-bottom: 13px;
}

.opcua-group-form :deep(.el-select) {
  width: 100%;
}
</style>
