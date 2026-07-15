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
          @update:model-value="setValue(field.name, $event)"
        >
          <el-option
            v-for="option in field.property.enum"
            :key="String(option)"
            :label="String(option)"
            :value="option"
          />
        </el-select>
        <el-switch
          v-else-if="field.property.type === 'boolean'"
          :model-value="Boolean(model[field.name])"
          @update:model-value="setValue(field.name, $event)"
        />
        <el-input-number
          v-else-if="field.property.type === 'number' || field.property.type === 'integer'"
          :model-value="numberValue(model[field.name])"
          :min="field.property.minimum"
          :max="field.property.maximum"
          :step="field.property.type === 'integer' ? 1 : 0.1"
          @update:model-value="setValue(field.name, $event)"
        />
        <CollectorSchemaForm
          v-else-if="field.property.type === 'object'"
          :model-value="objectValue(model[field.name])"
          :schema="field.property"
          :ui-schema="{}"
          @update:model-value="setValue(field.name, $event)"
        />
        <el-input
          v-else
          :model-value="stringValue(model[field.name])"
          :type="field.property['x-induforge-secret'] ? 'password' : 'text'"
          :show-password="Boolean(field.property['x-induforge-secret'])"
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
          <el-input-number
            v-if="field.property.type === 'number' || field.property.type === 'integer'"
            :model-value="numberValue(model[field.name])"
            :min="field.property.minimum"
            :max="field.property.maximum"
            @update:model-value="setValue(field.name, $event)"
          />
          <el-switch
            v-else-if="field.property.type === 'boolean'"
            :model-value="Boolean(model[field.name])"
            @update:model-value="setValue(field.name, $event)"
          />
          <el-input
            v-else
            :model-value="stringValue(model[field.name])"
            @update:model-value="setValue(field.name, $event)"
          />
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

defineOptions({ name: 'CollectorSchemaForm' })

const props = withDefaults(
  defineProps<{
    modelValue: Record<string, unknown>
    schema: CollectorJsonSchema
    uiSchema?: Record<string, unknown>
    section?: string
  }>(),
  { uiSchema: () => ({}), section: 'connection' },
)

const emit = defineEmits<{ 'update:modelValue': [value: Record<string, unknown>] }>()
const model = computed(() => props.modelValue || {})
const requiredFields = computed(() => new Set(props.schema.required || []))
const fields = computed(() => {
  const properties = props.schema.properties || {}
  const section = props.uiSchema?.[props.section] as { order?: string[] } | undefined
  const order = section?.order || Object.keys(properties)
  return order
    .filter((name) => properties[name])
    .map((name) => ({ name, property: properties[name] as CollectorJsonSchemaProperty }))
})
const visibleFields = computed(() =>
  fields.value.filter((field) => !field.property['x-induforge-advanced']),
)
const advancedFields = computed(() =>
  fields.value.filter((field) => field.property['x-induforge-advanced']),
)

function setValue(name: string, value: unknown) {
  emit('update:modelValue', { ...model.value, [name]: value })
}
function stringValue(value: unknown) {
  return value === undefined || value === null ? '' : String(value)
}
function numberValue(value: unknown) {
  return typeof value === 'number' ? value : undefined
}
function objectValue(value: unknown) {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {}
}
</script>

<style scoped>
.collector-schema-form {
  width: 100%;
}
.collector-schema-form :deep(.el-select),
.collector-schema-form :deep(.el-input-number) {
  width: 100%;
}
.collector-schema-form__hint {
  margin-top: 6px;
  color: #7b8794;
  font-size: 12px;
  line-height: 1.5;
}
.collector-schema-form__advanced {
  margin-top: 8px;
  border-top: 1px solid #e7ebef;
  border-bottom: 0;
}
</style>
