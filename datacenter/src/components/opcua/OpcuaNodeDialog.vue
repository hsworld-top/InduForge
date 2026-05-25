<template>
  <el-dialog :model-value="modelValue" :title="mode === 'edit' ? '编辑变量' : '新建变量'" width="720px" @close="$emit('update:modelValue', false)">
    <el-form class="opcua-node-form" label-width="98px">
      <div class="opcua-node-form__section">基础信息</div>
      <el-form-item label="变量名" required>
        <el-input v-model="form.name" maxlength="100" />
      </el-form-item>
      <el-form-item label="标识符" required>
        <el-input v-model="form.code" maxlength="100" />
      </el-form-item>
      <el-form-item label="变量组">
        <el-select v-model="form.groupId" clearable placeholder="未分组">
          <el-option v-for="group in groups" :key="group.id" :label="group.name" :value="group.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="NodeId" required>
        <el-input v-model="form.nodeId" maxlength="1000" />
      </el-form-item>
      <div class="opcua-node-form__section">OPC UA 属性</div>
      <el-form-item label="BrowseName">
        <el-input v-model="form.browseName" />
      </el-form-item>
      <el-form-item label="数据类型" required>
        <el-select v-model="form.dataType" allow-create filterable>
          <el-option v-for="type in dataTypes" :key="type" :label="type" :value="type" />
        </el-select>
      </el-form-item>
      <el-form-item label="单位">
        <el-input v-model="form.unit" maxlength="20" />
      </el-form-item>
      <div class="opcua-node-form__section">采样策略</div>
      <el-form-item label="采样周期">
        <el-input-number v-model="form.samplingMs" :min="1" :step="100" />
      </el-form-item>
      <el-form-item label="死区">
        <el-input-number v-model="form.deadband" :min="0" :precision="3" :step="0.1" />
      </el-form-item>
      <el-form-item label="访问级别">
        <el-segmented v-model="form.accessLevel" :options="['Read', 'Write', 'ReadWrite']" />
      </el-form-item>
      <div class="opcua-node-form__section">说明</div>
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
import type { OpcuaNode, OpcuaNodeGroup } from './types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  groups: OpcuaNodeGroup[]
  node?: OpcuaNode | null
  defaultGroupId?: string
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: Record<string, unknown>): void
}>()

const dataTypes = ['Boolean', 'Int16', 'Int32', 'Int64', 'Float', 'Double', 'String']

const form = reactive({
  name: '',
  code: '',
  groupId: '',
  nodeId: '',
  browseName: '',
  dataType: 'Double',
  unit: '',
  samplingMs: 1000,
  deadband: null as number | null,
  accessLevel: 'Read',
  description: '',
})

watch(
  () => [props.modelValue, props.node, props.defaultGroupId] as const,
  () => {
    if (!props.modelValue) return
    form.name = props.node?.name || ''
    form.code = props.node?.code || ''
    form.groupId = props.node?.groupId || props.defaultGroupId || ''
    form.nodeId = props.node?.nodeId || ''
    form.browseName = props.node?.browseName || ''
    form.dataType = props.node?.dataType || 'Double'
    form.unit = props.node?.unit || ''
    form.samplingMs = props.node?.samplingMs || 1000
    form.deadband = props.node?.deadband ?? null
    form.accessLevel = props.node?.accessLevel || 'Read'
    form.description = props.node?.description || ''
  },
  { immediate: true },
)

const submit = () => {
  if (!form.name.trim() || !form.nodeId.trim() || !form.dataType.trim()) {
    ElMessage.warning('变量名、NodeId 和数据类型不能为空')
    return
  }
  emit('submit', {
    name: form.name.trim(),
    code: form.code.trim(),
    groupId: form.groupId || null,
    hasGroupId: true,
    nodeId: form.nodeId.trim(),
    browseName: form.browseName.trim() || null,
    dataType: form.dataType.trim(),
    unit: form.unit.trim() || null,
    samplingMs: form.samplingMs,
    deadband: form.deadband,
    accessLevel: form.accessLevel,
    description: form.description.trim() || null,
  })
}
</script>

<style scoped>
.opcua-node-form {
  display: grid;
  grid-template-columns: 1fr 1fr;
  column-gap: 14px;
  padding: 2px 0;
}

.opcua-node-form__section {
  grid-column: 1 / -1;
  margin: 4px 0 2px;
  padding: 7px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.opcua-node-form :deep(.el-form-item) {
  margin-bottom: 12px;
}

.opcua-node-form :deep(.el-form-item:nth-child(5)),
.opcua-node-form :deep(.el-form-item:last-child) {
  grid-column: 1 / -1;
}

.opcua-node-form :deep(.el-select),
.opcua-node-form :deep(.el-input-number),
.opcua-node-form :deep(.el-segmented) {
  width: 100%;
}
</style>
