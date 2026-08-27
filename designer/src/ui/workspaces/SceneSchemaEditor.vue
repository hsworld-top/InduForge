<script setup lang="ts">
import { computed } from 'vue'

type SchemaType = 'string' | 'number' | 'integer' | 'boolean' | 'object' | 'array'
const props = defineProps<{ modelValue: Record<string, unknown> }>()
const emit = defineEmits<{ 'update:modelValue': [value: Record<string, unknown>] }>()
const schemaType = computed(() => String(props.modelValue?.type || 'object') as SchemaType)
const properties = computed(() => (props.modelValue?.properties || {}) as Record<string, Record<string, unknown>>)
const required = computed(() => new Set(Array.isArray(props.modelValue?.required) ? props.modelValue.required as string[] : []))

function schemaFor(type: SchemaType): Record<string, unknown> {
  if (type === 'object') return { type, properties: {}, required: [], additionalProperties: false }
  if (type === 'array') return { type, items: { type: 'string' } }
  return { type }
}
function changeType(event: Event): void { emit('update:modelValue', schemaFor((event.target as HTMLSelectElement).value as SchemaType)) }
function addProperty(): void {
  const used = new Set(Object.keys(properties.value)); let index = used.size + 1
  while (used.has(`field${index}`)) index += 1
  const name = `field${index}`
  emit('update:modelValue', { ...props.modelValue, properties: { ...properties.value, [name]: { type: 'string' } } })
}
function renameProperty(oldName: string, nextName: string): void {
  nextName = nextName.trim()
  if (!nextName || nextName === oldName || properties.value[nextName]) return
  const nextProperties: Record<string, Record<string, unknown>> = {}
  for (const [name, schema] of Object.entries(properties.value)) nextProperties[name === oldName ? nextName : name] = schema
  emit('update:modelValue', { ...props.modelValue, properties: nextProperties, required: [...required.value].map((name) => name === oldName ? nextName : name) })
}
function updateProperty(name: string, value: Record<string, unknown>): void { emit('update:modelValue', { ...props.modelValue, properties: { ...properties.value, [name]: value } }) }
function toggleRequired(name: string, checked: boolean): void {
  const next = new Set(required.value); checked ? next.add(name) : next.delete(name)
  emit('update:modelValue', { ...props.modelValue, required: [...next] })
}
function removeProperty(name: string): void {
  const nextProperties = { ...properties.value }; delete nextProperties[name]
  emit('update:modelValue', { ...props.modelValue, properties: nextProperties, required: [...required.value].filter((item) => item !== name) })
}
</script>

<template>
  <div class="schema-editor">
    <select :value="schemaType" aria-label="数据类型" @change="changeType">
      <option value="string">文本</option><option value="number">数字</option><option value="integer">整数</option>
      <option value="boolean">布尔</option><option value="object">对象</option><option value="array">数组</option>
    </select>
    <div v-if="schemaType === 'object'" class="schema-properties">
      <div v-for="(child, name) in properties" :key="name" class="schema-property">
        <div class="property-head">
          <input :value="name" aria-label="字段名称" @change="renameProperty(name, ($event.target as HTMLInputElement).value)" />
          <label><input type="checkbox" :checked="required.has(name)" @change="toggleRequired(name, ($event.target as HTMLInputElement).checked)" />必填</label>
          <button type="button" @click="removeProperty(name)">删除</button>
        </div>
        <SceneSchemaEditor :model-value="child" @update:model-value="updateProperty(name, $event)" />
      </div>
      <button type="button" class="add-field" @click="addProperty">添加字段</button>
    </div>
    <div v-else-if="schemaType === 'array'" class="array-item">
      <span>数组元素</span>
      <SceneSchemaEditor :model-value="(modelValue.items as Record<string, unknown>) || { type: 'string' }" @update:model-value="emit('update:modelValue', { ...modelValue, items: $event })" />
    </div>
  </div>
</template>

<style scoped>
.schema-editor{display:grid;gap:7px;min-width:0}select,input{box-sizing:border-box;min-width:0;height:30px;border:1px solid #aeb6b1;background:#fff}.schema-properties,.array-item{display:grid;gap:8px;padding:8px;border-left:2px solid #d7ddd8}.schema-property{display:grid;gap:7px}.property-head{display:grid;grid-template-columns:minmax(80px,1fr) auto auto;gap:6px;align-items:center}.property-head label{display:flex;gap:4px;align-items:center;font-size:11px}.property-head label input{width:14px;height:14px}button{height:28px;border:1px solid #9fa9a3;background:#fff;cursor:pointer}.add-field{justify-self:start}.array-item>span{color:#64706b;font-size:11px}
</style>
