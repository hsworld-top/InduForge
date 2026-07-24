<template>
  <div class="collector-schema-form">
    <template v-for="field in visibleFields" :key="field.name">
      <el-form-item
        :label="field.property.title || field.name"
        :required="requiredFields.has(field.name)"
      >
        <el-select
          v-if="field.property.enum"
          :model-value="model[field.name]"
          :disabled="disabled"
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
        <el-input
          v-else
          :model-value="stringValue(model[field.name])"
          :type="field.property['x-induforge-secret'] ? 'password' : 'text'"
          :show-password="Boolean(field.property['x-induforge-secret'])"
          :disabled="disabled"
          :placeholder="field.property.description"
          @update:model-value="setValue(field.name, $event)"
        >
          <template v-if="field.property['x-induforge-unit']" #append>{{
            field.property['x-induforge-unit']
          }}</template>
        </el-input>
        <div v-if="field.property.description" class="collector-schema-form__hint">
          {{ field.property.description }}
        </div>
      </el-form-item>
    </template>

    <el-collapse v-if="advancedFields.length" class="collector-schema-form__advanced">
      <el-collapse-item title="高级参数" name="advanced">
        <el-form-item
          v-for="field in advancedFields"
          :key="field.name"
          :label="field.property.title || field.name"
        >
          <el-select
            v-if="field.property.enum"
            :model-value="model[field.name]"
            :disabled="disabled"
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
          <el-input
            v-else
            :model-value="stringValue(model[field.name])"
            :type="field.property['x-induforge-secret'] ? 'password' : 'text'"
            :show-password="Boolean(field.property['x-induforge-secret'])"
            :disabled="disabled"
            :placeholder="field.property.description"
            @update:model-value="setValue(field.name, $event)"
          />
          <div v-if="field.property.description" class="collector-schema-form__hint">
            {{ field.property.description }}
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
import { resolveCollectorEnumOptionLabel } from './collector-workbench-model'

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
  }>(),
  { uiSchema: () => ({}), section: 'connection', disabled: false },
)

const emit = defineEmits<{ 'update:modelValue': [value: Record<string, unknown>] }>()
const model = computed(() => props.modelValue || {})
const requiredFields = computed(
  () => new Set([...(props.schema.required || []), ...(props.requiredFieldNames || [])]),
)
const fields = computed(() => {
  const properties = props.schema.properties || {}
  const section = props.uiSchema?.[props.section] as { order?: string[] } | undefined
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
function numberValue(value: unknown) {
  return typeof value === 'number' ? value : undefined
}
function enumOptionLabel(property: CollectorJsonSchemaProperty, option: unknown) {
  return resolveCollectorEnumOptionLabel(property, option)
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
.collector-schema-form__unit {
  flex: 0 0 auto;
  color: #667584;
  font-size: 13px;
}
.collector-schema-form__hint {
  margin-top: 6px;
  color: #7b8794;
  font-size: 12px;
  line-height: 1.5;
}
.collector-schema-form__advanced {
  grid-column: 1 / -1;
  margin-top: 8px;
  border-top: 1px solid #e7ebef;
  border-bottom: 0;
}
@media (max-width: 960px) {
  .collector-schema-form {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
