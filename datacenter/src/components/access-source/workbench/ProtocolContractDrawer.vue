<template>
  <DcDrawer v-model="visible" :title="title" :width="620" :max="820">
    <div class="protocol-contract">
      <p v-if="subtitle" class="protocol-contract__subtitle">{{ subtitle }}</p>

      <section v-for="section in sections" :key="section.title" class="protocol-contract__section">
        <h3>{{ section.title }}</h3>
        <dl>
          <template v-for="row in section.rows" :key="`${section.title}-${row.label}`">
            <dt>{{ row.label }}</dt>
            <dd :class="row.tone ? `is-${row.tone}` : ''">{{ formatValue(row.value) }}</dd>
          </template>
        </dl>
        <p v-for="note in section.notes || []" :key="note" class="protocol-contract__note">
          {{ note }}
        </p>
      </section>

      <section class="protocol-contract__section">
        <h3>诊断问题</h3>
        <div v-if="issues.length === 0" class="protocol-contract__empty">当前范围暂无诊断问题</div>
        <div
          v-for="issue in issues"
          v-else
          :key="`${issue.severity}-${issue.message}`"
          class="protocol-contract__issue"
        >
          <el-tag size="small" :type="issue.severity === 'error' ? 'danger' : 'warning'">
            {{ issue.severity }}
          </el-tag>
          <span>{{ issue.message }}</span>
        </div>
      </section>
    </div>
  </DcDrawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DcDrawer from '@/components/shared/DcDrawer.vue'

type ContractRow = {
  label: string
  value: string | number | null | undefined
  tone?: 'ok' | 'warning' | 'danger' | 'muted'
}

type ContractSection = {
  title: string
  rows: ContractRow[]
  notes?: string[]
}

const props = defineProps<{
  modelValue: boolean
  title: string
  subtitle?: string
  sections: ContractSection[]
  issues?: Array<{ severity: string; message: string }>
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const issues = computed(() => props.issues || [])

function formatValue(value: string | number | null | undefined) {
  if (value === null || value === undefined || value === '') return '-'
  return String(value)
}
</script>

<style scoped>
.protocol-contract {
  display: grid;
  gap: 12px;
}

.protocol-contract__subtitle {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 18px;
}

.protocol-contract__section {
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.protocol-contract__section h3 {
  margin: 0 0 10px;
  color: var(--dc-text);
  font-size: 13px;
  line-height: 18px;
}

.protocol-contract__section dl {
  margin: 0;
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr);
  gap: 8px 10px;
  font-size: 12px;
}

.protocol-contract__section dt {
  color: var(--dc-text-muted);
}

.protocol-contract__section dd {
  min-width: 0;
  margin: 0;
  color: var(--dc-text-secondary);
  overflow-wrap: anywhere;
}

.protocol-contract__section dd.is-ok {
  color: var(--dc-success);
  font-weight: 700;
}

.protocol-contract__section dd.is-warning {
  color: var(--el-color-warning);
  font-weight: 700;
}

.protocol-contract__section dd.is-danger {
  color: var(--dc-danger);
  font-weight: 700;
}

.protocol-contract__section dd.is-muted {
  color: var(--dc-text-muted);
}

.protocol-contract__note {
  margin: 10px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 18px;
}

.protocol-contract__empty {
  color: var(--dc-success);
  font-size: 12px;
}

.protocol-contract__issue {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 0;
  border-top: 1px solid var(--dc-border);
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 18px;
}

.protocol-contract__issue:first-of-type {
  border-top: 0;
  padding-top: 0;
}
</style>
