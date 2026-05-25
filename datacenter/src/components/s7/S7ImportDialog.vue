<template>
  <el-dialog :model-value="modelValue" title="导入 S7 地址表" width="860px" @close="$emit('update:modelValue', false)">
    <el-input v-model="text" type="textarea" :rows="8" placeholder="变量名,Code,分组,地址,类型,单位,倍率,偏移,采集周期,描述" />
    <el-table class="s7-import-dialog__table" :data="rows" height="260" empty-text="粘贴表格文本后预览">
      <el-table-column label="状态" width="70"><template #default="{ row }"><el-tag size="small" :type="row.issue ? 'warning' : 'success'">{{ row.issue ? '检查' : '可导入' }}</el-tag></template></el-table-column>
      <el-table-column prop="name" label="变量名" min-width="120" />
      <el-table-column prop="code" label="Code" min-width="110" />
      <el-table-column prop="addressText" label="原始地址" min-width="120" />
      <el-table-column prop="dataType" label="类型" width="90" />
      <el-table-column prop="issue" label="问题" min-width="160" />
    </el-table>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :disabled="rows.length === 0" :loading="loading" @click="$emit('submit', rows)">导入</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{ modelValue: boolean; loading?: boolean }>()
defineEmits<{ (event: 'update:modelValue', value: boolean): void; (event: 'submit', rows: Array<Record<string, unknown>>): void }>()
const text = ref('')
watch(
  () => props.modelValue,
  (visible) => {
    if (visible) text.value = ''
  },
)
const rows = computed(() =>
  text.value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line, index) => {
      const [name, code, , addressText, dataType, unit, scale, offset, pollIntervalMs, description] = line.split(/,|\t/).map((item) => item.trim())
      return {
        name,
        code,
        addressText,
        dataType: dataType || 'Real',
        unit: unit || null,
        scale: scale ? Number(scale) : 1,
        offset: offset ? Number(offset) : 0,
        pollIntervalMs: pollIntervalMs ? Number(pollIntervalMs) : 1000,
        description: description || null,
        sortOrder: index,
        issue: !addressText ? '地址为空' : '',
      }
    }),
)
</script>

<style scoped>
.s7-import-dialog__table {
  margin-top: 12px;
}
</style>
