<template>
  <div v-if="supportsStructuredAddress" class="collector-address-helper">
    <div class="collector-address-helper__head">
      <strong>{{ title }}</strong>
      <span>{{ ui('生成后仍会由服务端按驱动规则重新校验', 'The server validates the generated address against the driver rules') }}</span>
    </div>

    <div v-if="helper === 'melsec'" class="collector-address-helper__fields">
      <el-select v-model="draft.area" :placeholder="ui('设备区', 'Device Area')">
        <el-option v-for="area in melsecAreas" :key="area" :label="area" :value="area" />
      </el-select>
      <el-input v-model="draft.offset" :placeholder="ui('地址，如 100 或 A0', 'Address, such as 100 or A0')" />
      <el-input-number v-model="draft.bit" :min="0" :max="15" :placeholder="ui('位（可选）', 'Bit (optional)')" />
    </div>

    <div v-else-if="helper === 'omron'" class="collector-address-helper__fields">
      <el-select v-model="draft.area" :placeholder="ui('内存区', 'Memory Area')">
        <el-option v-for="area in omronAreas" :key="area" :label="area" :value="area" />
      </el-select>
      <el-input-number
        v-if="draft.area === 'E' || draft.area === 'EM'"
        v-model="draft.bank"
        :min="0"
        :max="255"
        placeholder="EM Bank"
      />
      <el-input-number v-model="draft.word" :min="0" :placeholder="ui('字地址', 'Word Address')" />
      <el-input-number v-model="draft.bit" :min="0" :max="15" :placeholder="ui('位（可选）', 'Bit (optional)')" />
    </div>

    <div v-else-if="helper === 'allen_bradley'" class="collector-address-helper__fields is-wide">
      <el-select v-model="draft.scope" :placeholder="ui('作用域', 'Scope')">
        <el-option :label="ui('控制器', 'Controller')" value="controller" />
        <el-option label="Program" value="program" />
      </el-select>
      <el-input
        v-if="draft.scope === 'program'"
        v-model="draft.program"
        :placeholder="ui('Program 名称', 'Program Name')"
      />
      <el-input v-model="draft.tag" :placeholder="ui('标签名，如 Motor.Speed', 'Tag name, such as Motor.Speed')" />
      <el-input-number v-model="draft.index" :min="0" :placeholder="ui('数组索引（可选）', 'Array Index (optional)')" />
    </div>

    <div class="collector-address-helper__actions">
      <code>{{ generatedAddress || ui('请先填写地址字段', 'Complete the address fields') }}</code>
      <el-button size="small" type="primary" :disabled="!generatedAddress" @click="applyAddress">
        {{ ui('应用结构化地址', 'Apply Address') }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps<{
  modelValue: Record<string, unknown>
  helper: string
  schema?: Record<string, unknown>
}>()
const emit = defineEmits<{ 'update:modelValue': [value: Record<string, unknown>] }>()
const melsecAreas = ['X', 'Y', 'M', 'L', 'B', 'D', 'W', 'R', 'ZR', 'T', 'C']
const omronAreas = ['CIO', 'D', 'DM', 'W', 'H', 'A', 'E', 'EM', 'TIM', 'CNT']
const draft = reactive({
  area: '',
  offset: '',
  bit: undefined as number | undefined,
  bank: 0,
  word: 0,
  scope: 'controller',
  program: '',
  tag: '',
  index: undefined as number | undefined,
})
const hasRawAddressField = computed(() => {
  const properties = (props.schema?.properties || {}) as Record<string, unknown>
  return Object.prototype.hasOwnProperty.call(properties, 'address')
})
const supportsStructuredAddress = computed(
  () => hasRawAddressField.value && ['melsec', 'omron', 'allen_bradley'].includes(props.helper),
)
const title = computed(
  () =>
    (
      ({
        melsec: ui('三菱设备地址', 'Mitsubishi Device Address'),
        omron: ui('欧姆龙内存地址', 'Omron Memory Address'),
        allen_bradley: ui('罗克韦尔标签地址', 'Allen-Bradley Tag Address'),
      }) as Record<string, string>
    )[props.helper] || ui('地址助手', 'Address Helper'),
)
const generatedAddress = computed(() => {
  if (props.helper === 'melsec') {
    if (!draft.area || !draft.offset.trim()) return ''
    return `${draft.area}${draft.offset.trim()}${draft.bit === undefined ? '' : `.${draft.bit}`}`
  }
  if (props.helper === 'omron') {
    if (!draft.area) return ''
    const prefix =
      draft.area === 'E' || draft.area === 'EM' ? `${draft.area}${draft.bank}.` : draft.area
    return `${prefix}${draft.word}${draft.bit === undefined ? '' : `.${draft.bit}`}`
  }
  if (props.helper === 'allen_bradley') {
    if (!draft.tag.trim() || (draft.scope === 'program' && !draft.program.trim())) return ''
    const prefix = draft.scope === 'program' ? `Program:${draft.program.trim()}.` : ''
    return `${prefix}${draft.tag.trim()}${draft.index === undefined ? '' : `[${draft.index}]`}`
  }
  return ''
})
function applyAddress() {
  if (!generatedAddress.value) return
  emit('update:modelValue', { ...props.modelValue, address: generatedAddress.value })
}
</script>

<style scoped>
.collector-address-helper {
  display: grid;
  gap: 10px;
  margin-bottom: 12px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.collector-address-helper__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}
.collector-address-helper__head span {
  color: var(--dc-text-muted);
  font-size: 12px;
}
.collector-address-helper__fields {
  display: grid;
  grid-template-columns: 120px minmax(150px, 1fr) 140px;
  gap: 8px;
}
.collector-address-helper__fields.is-wide {
  grid-template-columns: 120px 150px minmax(180px, 1fr) 150px;
}
.collector-address-helper__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.collector-address-helper__actions code {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
@media (max-width: 680px) {
  .collector-address-helper__fields,
  .collector-address-helper__fields.is-wide {
    grid-template-columns: 1fr;
  }
}
</style>
