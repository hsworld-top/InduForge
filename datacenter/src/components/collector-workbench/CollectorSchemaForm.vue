<template>
  <div class="collector-schema-form">
    <template v-for="field in visibleFields" :key="field.name">
      <el-form-item
        :label="fieldLabel(field.name, field.property)"
        :required="requiredFields.has(field.name)"
      >
        <el-select
          v-if="field.property.enum"
          :model-value="model[field.name]"
          :disabled="disabled"
          :placeholder="selectPlaceholder(field.name, field.property)"
          @update:model-value="setValue(field.name, $event)"
        >
          <el-option
            v-for="option in field.property.enum"
            :key="String(option)"
            :label="enumOptionLabel(field.property, option)"
            :value="option"
          />
        </el-select>
        <el-switch
          v-else-if="field.property.type === 'boolean'"
          :model-value="Boolean(model[field.name])"
          :disabled="disabled"
          @update:model-value="setValue(field.name, $event)"
        />
        <div
          v-else-if="field.property.type === 'number' || field.property.type === 'integer'"
          class="collector-schema-form__number"
        >
          <el-input-number
            :model-value="numberValue(model[field.name])"
            :min="field.property.minimum"
            :max="field.property.maximum"
            :step="field.property.type === 'integer' ? 1 : 0.1"
            :disabled="disabled"
            @update:model-value="setValue(field.name, $event)"
          />
          <span v-if="field.property['x-induforge-unit']" class="collector-schema-form__unit">
            {{ field.property['x-induforge-unit'] }}
          </span>
        </div>
        <CollectorSchemaForm
          v-else-if="field.property.type === 'object'"
          :model-value="objectValue(model[field.name])"
          :schema="field.property"
          :ui-schema="{}"
          :disabled="disabled"
          @update:model-value="setValue(field.name, $event)"
        />
        <div v-else-if="hasStringOptions(field.name)" class="collector-schema-form__resource">
          <el-select
            :model-value="stringValue(model[field.name])"
            :disabled="disabled"
            :placeholder="resourcePlaceholder(field.name, field.property)"
            allow-create
            clearable
            default-first-option
            filterable
            @update:model-value="setValue(field.name, $event)"
          >
            <el-option
              v-for="option in stringOptions(field.name)"
              :key="option"
              :label="option"
              :value="option"
            />
          </el-select>
          <el-tooltip :content="ui('刷新调试代理资源', 'Refresh Debug Agent Resources')" placement="top">
            <el-button
              :disabled="disabled"
              :icon="IconTablerRefresh"
              :aria-label="ui('刷新可用选项', 'Refresh Available Options')"
              @click="emit('refresh-options', field.name)"
            />
          </el-tooltip>
        </div>
        <el-input
          v-else
          :model-value="stringValue(model[field.name])"
          :type="field.property['x-induforge-secret'] ? 'password' : 'text'"
          :show-password="Boolean(field.property['x-induforge-secret'])"
          :disabled="disabled"
          :placeholder="inputPlaceholder(field.name, field.property)"
          @update:model-value="setValue(field.name, $event)"
        >
          <template v-if="field.property['x-induforge-unit']" #append>{{
            field.property['x-induforge-unit']
          }}</template>
        </el-input>
        <div v-if="fieldHint(field.name, field.property)" class="collector-schema-form__hint">
          {{ fieldHint(field.name, field.property) }}
        </div>
      </el-form-item>
    </template>

    <el-collapse v-if="advancedFields.length" class="collector-schema-form__advanced">
      <el-collapse-item :title="ui('高级参数', 'Advanced Settings')" name="advanced">
        <el-form-item
          v-for="field in advancedFields"
          :key="field.name"
          :label="fieldLabel(field.name, field.property)"
        >
          <el-select
            v-if="field.property.enum"
            :model-value="model[field.name]"
            :disabled="disabled"
            :placeholder="selectPlaceholder(field.name, field.property)"
            @update:model-value="setValue(field.name, $event)"
          >
            <el-option
              v-for="option in field.property.enum"
              :key="String(option)"
              :label="enumOptionLabel(field.property, option)"
              :value="option"
            />
          </el-select>
          <div
            v-else-if="field.property.type === 'number' || field.property.type === 'integer'"
            class="collector-schema-form__number"
          >
            <el-input-number
              :model-value="numberValue(model[field.name])"
              :min="field.property.minimum"
              :max="field.property.maximum"
              :step="field.property.type === 'integer' ? 1 : 0.1"
              :disabled="disabled"
              @update:model-value="setValue(field.name, $event)"
            />
            <span v-if="field.property['x-induforge-unit']" class="collector-schema-form__unit">
              {{ field.property['x-induforge-unit'] }}
            </span>
          </div>
          <el-switch
            v-else-if="field.property.type === 'boolean'"
            :model-value="Boolean(model[field.name])"
            :disabled="disabled"
            @update:model-value="setValue(field.name, $event)"
          />
          <div v-else-if="hasStringOptions(field.name)" class="collector-schema-form__resource">
            <el-select
              :model-value="stringValue(model[field.name])"
              :disabled="disabled"
              :placeholder="resourcePlaceholder(field.name, field.property)"
              allow-create
              clearable
              default-first-option
              filterable
              @update:model-value="setValue(field.name, $event)"
            >
              <el-option
                v-for="option in stringOptions(field.name)"
                :key="option"
                :label="option"
                :value="option"
              />
            </el-select>
            <el-tooltip :content="ui('刷新调试代理资源', 'Refresh Debug Agent Resources')" placement="top">
              <el-button
                :disabled="disabled"
                :icon="IconTablerRefresh"
                :aria-label="ui('刷新可用选项', 'Refresh Available Options')"
                @click="emit('refresh-options', field.name)"
              />
            </el-tooltip>
          </div>
          <el-input
            v-else
            :model-value="stringValue(model[field.name])"
            :type="field.property['x-induforge-secret'] ? 'password' : 'text'"
            :show-password="Boolean(field.property['x-induforge-secret'])"
            :disabled="disabled"
            :placeholder="inputPlaceholder(field.name, field.property)"
            @update:model-value="setValue(field.name, $event)"
          />
          <div v-if="fieldHint(field.name, field.property)" class="collector-schema-form__hint">
            {{ fieldHint(field.name, field.property) }}
          </div>
        </el-form-item>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type {
  CollectorJsonSchema,
  CollectorJsonSchemaProperty,
} from '@/api/schemas/collector.schema'
import {
  formatCollectorSchemaFieldHint,
  formatCollectorSchemaFieldLabel,
  resolveCollectorEnumOptionLabel,
} from './collector-workbench-model'
import IconTablerRefresh from '~icons/tabler/refresh'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

defineOptions({ name: 'CollectorSchemaForm' })

const props = withDefaults(
  defineProps<{
    modelValue: Record<string, unknown>
    schema: CollectorJsonSchema
    uiSchema?: Record<string, unknown>
    section?: string
    disabled?: boolean
    visibleFieldNames?: string[]
    requiredFieldNames?: string[]
    stringFieldOptions?: Record<string, string[]>
  }>(),
  { uiSchema: () => ({}), section: 'connection', disabled: false },
)

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
  'refresh-options': [fieldName: string]
}>()
const model = computed(() => props.modelValue || {})
const requiredFields = computed(
  () => new Set([...(props.schema.required || []), ...(props.requiredFieldNames || [])]),
)
const fields = computed(() => {
  const properties = props.schema.properties || {}
  const nestedSection = props.uiSchema?.[props.section] as { order?: string[] } | undefined
  const section =
    nestedSection ||
    (props.section === 'connection' && Array.isArray(props.uiSchema?.order)
      ? (props.uiSchema as { order?: string[] })
      : undefined)
  const order = section?.order || Object.keys(properties)
  const visibleFieldNames = props.visibleFieldNames ? new Set(props.visibleFieldNames) : null
  return order
    .filter((name) => properties[name] && (!visibleFieldNames || visibleFieldNames.has(name)))
    .map((name) => ({ name, property: properties[name] as CollectorJsonSchemaProperty }))
})
const visibleFields = computed(() =>
  fields.value.filter((field) => !field.property['x-induforge-advanced']),
)
const advancedFields = computed(() =>
  fields.value.filter((field) => field.property['x-induforge-advanced']),
)

function setValue(name: string, value: unknown) {
  if (props.disabled) return
  emit('update:modelValue', { ...model.value, [name]: value })
}
function stringValue(value: unknown) {
  return value === undefined || value === null ? '' : String(value)
}
function hasStringOptions(name: string) {
  return (props.stringFieldOptions?.[name]?.length || 0) > 0
}
function stringOptions(name: string) {
  return props.stringFieldOptions?.[name] || []
}
function numberValue(value: unknown) {
  return typeof value === 'number' ? value : undefined
}
function enumOptionLabel(property: CollectorJsonSchemaProperty, option: unknown) {
  return resolveCollectorEnumOptionLabel(property, option)
}
function fieldLabel(name: string, property: CollectorJsonSchemaProperty) {
  return formatCollectorSchemaFieldLabel(name, property)
}
function fieldHint(name: string, property: CollectorJsonSchemaProperty) {
  return formatCollectorSchemaFieldHint(name, property)
}
function selectPlaceholder(name: string, property: CollectorJsonSchemaProperty) {
  return ui(`请选择${fieldLabel(name, property)}`, `Select ${fieldLabel(name, property)}`)
}
function resourcePlaceholder(name: string, property: CollectorJsonSchemaProperty) {
  return ui(`选择或输入${fieldLabel(name, property)}`, `Select or enter ${fieldLabel(name, property)}`)
}
function inputPlaceholder(name: string, property: CollectorJsonSchemaProperty) {
  const examples: Record<string, string> = {
    host: '例如 192.168.1.10 或 plc.local',
    portName: '例如 COM3 或 /dev/ttyUSB0',
    endpointPath: '例如 /induforge/sim',
    targetAmsNetId: '例如 192.168.1.10.1.1',
    senderAmsNetId: '例如 192.168.1.20.1.1',
    clientId: '例如 line-01-gateway',
    deviceTopic: '例如 devices/device-01',
  }
  const englishExamples: Record<string, string> = {
    host: 'For example, 192.168.1.10 or plc.local',
    portName: 'For example, COM3 or /dev/ttyUSB0',
    endpointPath: 'For example, /induforge/sim',
    targetAmsNetId: 'For example, 192.168.1.10.1.1',
    senderAmsNetId: 'For example, 192.168.1.20.1.1',
    clientId: 'For example, line-01-gateway',
    deviceTopic: 'For example, devices/device-01',
  }
  return datacenterLocale.value === 'en'
    ? englishExamples[name] || `Enter ${fieldLabel(name, property)}`
    : examples[name] || `请输入${fieldLabel(name, property)}`
}
function objectValue(value: unknown) {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {}
}
</script>

<style scoped>
.collector-schema-form {
  display: grid;
  width: 100%;
  grid-template-columns: repeat(2, minmax(280px, 560px));
  justify-content: space-between;
  gap: 2px 24px;
}
.collector-schema-form > :deep(.el-form-item:has(.collector-schema-form)) {
  grid-column: 1 / -1;
}
.collector-schema-form :deep(.el-select),
.collector-schema-form :deep(.el-input-number) {
  width: 100%;
}
.collector-schema-form__number {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 10px;
}
.collector-schema-form__resource {
  display: grid;
  width: 100%;
  grid-template-columns: minmax(0, 1fr) 34px;
  gap: 8px;
}
.collector-schema-form__resource :deep(.el-button) {
  width: 34px;
  padding: 0;
}
.collector-schema-form__unit {
  flex: 0 0 auto;
  color: var(--dc-text-secondary);
  font-size: 13px;
}
.collector-schema-form__hint {
  margin-top: 6px;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.5;
}
.collector-schema-form__advanced {
  grid-column: 1 / -1;
  margin-top: 8px;
  border-top: 1px solid var(--dc-border);
  border-bottom: 0;
}
@media (max-width: 960px) {
  .collector-schema-form {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
