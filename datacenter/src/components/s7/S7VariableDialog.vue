<template>
  <DcDialog
    v-model="visible"
    :title="mode === 'edit' ? '编辑 S7 变量' : '新建 S7 变量'"
    width="820px"
  >
    <el-form class="s7-variable-dialog" label-position="top">
      <section>
        <h3><span>01</span>基础信息</h3>
        <div class="s7-variable-dialog__grid">
          <el-form-item label="变量名"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="Code"><el-input v-model="form.code" /></el-form-item>
          <el-form-item label="变量组">
            <el-select v-model="form.groupId" clearable>
              <el-option
                v-for="group in groups"
                :key="group.id"
                :label="group.name"
                :value="group.id"
              />
            </el-select>
          </el-form-item>
        </div>
      </section>
      <section>
        <h3><span>02</span>S7 地址</h3>
        <div class="s7-variable-dialog__grid">
          <el-form-item label="地址"
            ><el-input v-model="form.addressText" placeholder="DB1.DBD4 / M0.0"
          /></el-form-item>
          <el-form-item label="数据类型">
            <el-select v-model="form.dataType">
              <el-option v-for="type in dataTypes" :key="type" :label="type" :value="type" />
            </el-select>
          </el-form-item>
          <el-form-item label="String 长度"
            ><el-input-number v-model="form.length" :min="0"
          /></el-form-item>
        </div>
        <p class="s7-variable-dialog__preview">{{ addressPreview }}</p>
      </section>
      <section>
        <h3><span>03</span>数据解释</h3>
        <div class="s7-variable-dialog__grid">
          <el-form-item label="字节序"
            ><el-select v-model="form.byteOrder"
              ><el-option label="Big Endian" value="big_endian" /><el-option
                label="Little Endian"
                value="little_endian" /></el-select
          ></el-form-item>
          <el-form-item label="字序"
            ><el-select v-model="form.wordOrder"
              ><el-option label="Big Endian" value="big_endian" /><el-option
                label="Little Endian"
                value="little_endian" /></el-select
          ></el-form-item>
          <el-form-item label="数组长度"
            ><el-input-number v-model="form.arrayLength" :min="0"
          /></el-form-item>
          <el-form-item label="倍率"
            ><el-input-number v-model="form.scale" :precision="3"
          /></el-form-item>
          <el-form-item label="偏移"
            ><el-input-number v-model="form.offset" :precision="3"
          /></el-form-item>
          <el-form-item label="单位"><el-input v-model="form.unit" /></el-form-item>
        </div>
      </section>
      <section>
        <h3><span>04</span>采样</h3>
        <div class="s7-variable-dialog__grid">
          <el-form-item label="采集周期 ms"
            ><el-input-number v-model="form.pollIntervalMs" :min="100"
          /></el-form-item>
          <el-form-item label="状态"
            ><el-select v-model="form.status"
              ><el-option label="active" value="active" /><el-option
                label="inactive"
                value="inactive" /></el-select
          ></el-form-item>
        </div>
        <el-form-item label="说明"
          ><el-input v-model="form.description" type="textarea" :rows="2"
        /></el-form-item>
      </section>
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
import type { S7Variable, S7VariableGroup } from './types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  groups: S7VariableGroup[]
  variable?: S7Variable | null
  defaultGroupId?: string
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
const dataTypes = ['Bool', 'Byte', 'Word', 'DWord', 'Int', 'DInt', 'Real', 'DateTime', 'String']
const form = reactive({
  groupId: '',
  name: '',
  code: '',
  addressText: 'DB1.DBD0',
  dataType: 'Real',
  length: 0,
  arrayLength: 0,
  byteOrder: 'big_endian',
  wordOrder: 'big_endian',
  scale: 1,
  offset: 0,
  unit: '',
  pollIntervalMs: 1000,
  status: 'active',
  description: '',
})
watch(
  () => [props.modelValue, props.variable, props.defaultGroupId],
  () => {
    const variable = props.variable
    Object.assign(form, {
      groupId: variable?.groupId || props.defaultGroupId || '',
      name: variable?.name || '',
      code: variable?.code || '',
      addressText: variable?.addressText || 'DB1.DBD0',
      dataType: variable?.dataType || 'Real',
      length: variable?.length || 0,
      arrayLength: variable?.arrayLength || 0,
      byteOrder: variable?.byteOrder || 'big_endian',
      wordOrder: variable?.wordOrder || 'big_endian',
      scale: variable?.scale ?? 1,
      offset: variable?.offset ?? 0,
      unit: variable?.unit || '',
      pollIntervalMs: variable?.pollIntervalMs || 1000,
      status: variable?.status || 'active',
      description: variable?.description || '',
    })
  },
  { immediate: true },
)
const addressPreview = computed(
  () => `标准地址将由后端解析保存：${form.addressText || '-'} · 类型 ${form.dataType}`,
)
const submit = () => {
  emit('submit', {
    groupId: form.groupId || null,
    hasGroupId: true,
    name: form.name.trim(),
    code: form.code.trim(),
    addressText: form.addressText.trim(),
    dataType: form.dataType,
    length: form.length || null,
    arrayLength: form.arrayLength || null,
    byteOrder: form.byteOrder,
    wordOrder: form.wordOrder,
    scale: form.scale,
    offset: form.offset,
    unit: form.unit.trim() || null,
    pollIntervalMs: form.pollIntervalMs,
    status: form.status,
    description: form.description.trim() || null,
    sortOrder: props.variable?.sortOrder || 0,
  })
}
</script>

<style scoped>
.s7-variable-dialog {
  display: grid;
  gap: 12px;
}
.s7-variable-dialog section {
  padding: 10px 12px 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: color-mix(in oklch, var(--dc-surface-subtle) 82%, transparent);
}
.s7-variable-dialog h3 {
  margin: 0 0 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--dc-text);
  font-size: 13px;
}
.s7-variable-dialog h3 span {
  width: 24px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: color-mix(in oklch, var(--dc-primary) 14%, var(--dc-surface-raised));
  color: var(--dc-primary);
  font-size: 10px;
  font-weight: 800;
}
.s7-variable-dialog__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px 12px;
}
.s7-variable-dialog__preview {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.s7-variable-dialog :deep(.el-select),
.s7-variable-dialog :deep(.el-input-number) {
  width: 100%;
}
</style>
