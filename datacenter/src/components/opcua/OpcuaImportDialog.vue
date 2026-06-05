<template>
  <el-dialog
    :model-value="modelValue"
    title="从 OPC UA 导入变量"
    width="860px"
    @close="$emit('update:modelValue', false)"
  >
    <div class="opcua-import">
      <section>
        <div class="opcua-import__section-head">
          <strong>批量 NodeId</strong>
          <span>每行一个变量，导入前可改变量名</span>
        </div>
        <el-input
          v-model="rawText"
          type="textarea"
          :rows="8"
          placeholder="每行一个 NodeId，可用逗号补充类型与变量名：ns=2;s=Line1.Motor01.Speed,Double,电机转速"
        />
      </section>
      <section>
        <div class="opcua-import__section-head">
          <strong>导入确认</strong>
          <span>{{ rows.length }} 个候选变量</span>
        </div>
        <el-table :data="rows" height="220" size="small">
          <el-table-column label="变量名" min-width="150">
            <template #default="{ row }"><el-input v-model="row.name" size="small" /></template>
          </el-table-column>
          <el-table-column prop="nodeId" label="NodeId" min-width="230" show-overflow-tooltip />
          <el-table-column label="类型" width="120">
            <template #default="{ row }"><el-input v-model="row.dataType" size="small" /></template>
          </el-table-column>
          <el-table-column label="采样" width="110">
            <template #default="{ row }"
              ><el-input-number v-model="row.samplingMs" size="small" :min="1"
            /></template>
          </el-table-column>
        </el-table>
      </section>
    </div>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">导入</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'

defineProps<{
  modelValue: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', rows: Array<Record<string, unknown>>): void
}>()

const rawText = ref('')

const rows = computed(() =>
  rawText.value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [nodeId, dataType = 'Double', name = ''] = line.split(',').map((part) => part.trim())
      return {
        name:
          name ||
          nodeId
            .split(/[.;=:/]/)
            .filter(Boolean)
            .at(-1) ||
          nodeId,
        code: '',
        nodeId,
        dataType,
        samplingMs: 1000,
      }
    }),
)

const submit = () => {
  if (rows.value.length === 0) {
    ElMessage.warning('请至少输入一个 NodeId')
    return
  }
  emit('submit', rows.value)
}
</script>

<style scoped>
.opcua-import {
  display: grid;
  gap: 12px;
}

.opcua-import section {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.opcua-import__section-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.opcua-import__section-head strong {
  color: var(--dc-text);
  font-size: 13px;
}

.opcua-import__section-head span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.opcua-import :deep(.el-table) {
  --el-table-header-bg-color: var(--dc-surface-raised);
  --el-table-border-color: var(--dc-border);
}
</style>
