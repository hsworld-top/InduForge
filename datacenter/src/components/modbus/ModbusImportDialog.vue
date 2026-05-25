<template>
  <el-dialog :model-value="modelValue" title="导入 Modbus 变量" width="760px" @close="$emit('update:modelValue', false)">
    <div class="modbus-import-dialog__summary">
      <strong>{{ previewRows.length }}</strong>
      <span>待导入变量</span>
      <em>{{ mode === 'paste' ? '粘贴表格' : '地址段生成' }}</em>
    </div>
    <el-tabs v-model="mode">
      <el-tab-pane label="粘贴表格" name="paste">
        <el-input v-model="pasteText" type="textarea" :rows="9" placeholder="变量名,Code,从站地址,区域,地址,地址基准,类型,倍率,单位" />
      </el-tab-pane>
      <el-tab-pane label="地址段生成" name="range">
        <div class="modbus-import-dialog__range">
          <el-input-number v-model="range.unitId" :min="0" :max="247" />
          <el-select v-model="range.area">
            <el-option label="Holding Register" value="holding_register" />
            <el-option label="Input Register" value="input_register" />
            <el-option label="Coil" value="coil" />
            <el-option label="Discrete Input" value="discrete_input" />
          </el-select>
          <el-input-number v-model="range.startAddress" :min="0" />
          <el-input-number v-model="range.count" :min="1" :max="500" />
          <el-select v-model="range.dataType">
            <el-option label="uint16" value="uint16" />
            <el-option label="bool" value="bool" />
            <el-option label="float32" value="float32" />
          </el-select>
          <el-input v-model="range.prefix" />
        </div>
      </el-tab-pane>
    </el-tabs>
    <el-table :data="previewRows" height="210px">
      <el-table-column prop="name" label="变量名" />
      <el-table-column prop="code" label="Code" />
      <el-table-column prop="unitId" label="从站" width="70" />
      <el-table-column prop="area" label="区域" width="130" />
      <el-table-column prop="address" label="地址" width="90" />
      <el-table-column prop="dataType" label="类型" width="90" />
    </el-table>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="$emit('submit', previewRows)">导入 {{ previewRows.length }} 个变量</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'

defineProps<{ modelValue: boolean; loading?: boolean }>()
defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', rows: Array<Record<string, unknown>>): void
}>()

const mode = ref('paste')
const pasteText = ref('')
const range = reactive({
  unitId: 1,
  area: 'holding_register',
  startAddress: 40001,
  count: 20,
  dataType: 'uint16',
  prefix: 'modbus_reg_',
})

const previewRows = computed(() => {
  if (mode.value === 'range') {
    return Array.from({ length: range.count }, (_, index) => ({
      name: `${range.prefix}${index + 1}`,
      code: `${range.prefix}${index + 1}`,
      unitId: range.unitId,
      area: range.area,
      address: range.startAddress + index,
      addressBase: 'modicon',
      dataType: range.dataType,
      byteOrder: 'ABCD',
      wordOrder: 'high_first',
      scale: 1,
      offset: 0,
    }))
  }
  return pasteText.value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [name, code, unitId, area, address, addressBase, dataType, scale, unit] = line.split(',').map((item) => item.trim())
      return {
        name,
        code,
        unitId: Number(unitId || 1),
        area: area || 'holding_register',
        address: Number(address || 40001),
        addressBase: addressBase || 'modicon',
        dataType: dataType || 'uint16',
        scale: Number(scale || 1),
        unit: unit || null,
        byteOrder: 'ABCD',
        wordOrder: 'high_first',
      }
    })
})
</script>

<style scoped>
.modbus-import-dialog__summary {
  margin-bottom: 10px;
  padding: 10px 12px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 18%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  display: flex;
  align-items: center;
  gap: 8px;
}

.modbus-import-dialog__summary strong {
  color: var(--dc-primary);
  font-size: 18px;
}

.modbus-import-dialog__summary span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.modbus-import-dialog__summary em {
  margin-left: auto;
  font-style: normal;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.modbus-import-dialog__range {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.modbus-import-dialog__range :deep(.el-select),
.modbus-import-dialog__range :deep(.el-input-number) {
  width: 100%;
}
</style>
