<template>
  <div v-loading="loading" class="alarm-levels">
    <header class="alarm-levels__toolbar">
      <div>
        <strong>报警级别</strong>
        <span>顺序由低到高</span>
      </div>
      <div class="alarm-levels__actions">
        <button type="button" class="alarm-levels__secondary" @click="addDefinition">
          <IconTablerPlus />新增级别
        </button>
        <button type="button" class="alarm-levels__primary" :disabled="saving" @click="save">
          <IconTablerDeviceFloppy />{{ saving ? '保存中' : '保存' }}
        </button>
      </div>
    </header>

    <section class="alarm-levels__section">
      <div class="alarm-levels__level-head" aria-hidden="true">
        <span>颜色</span><span>显示名称</span><span>标识</span><span>操作</span>
      </div>
      <div
        v-for="(item, index) in draft.severityDefinitions"
        :key="item.key"
        class="alarm-levels__level-row"
      >
        <input
          v-model="item.color"
          class="alarm-levels__color"
          type="color"
          :aria-label="`${item.displayName}颜色`"
        />
        <el-input v-model="item.displayName" maxlength="20" />
        <el-input v-model="item.key" :disabled="item.isBuiltin" maxlength="30" />
        <div class="alarm-levels__row-actions">
          <button
            type="button"
            title="上移"
            :disabled="index === 0"
            @click="moveDefinition(index, -1)"
          >
            <IconTablerChevronUp />
          </button>
          <button
            type="button"
            title="下移"
            :disabled="index === draft.severityDefinitions.length - 1"
            @click="moveDefinition(index, 1)"
          >
            <IconTablerChevronDown />
          </button>
          <button
            type="button"
            class="is-danger"
            title="删除"
            :disabled="item.isBuiltin"
            @click="removeDefinition(item.key)"
          >
            <IconTablerTrash />
          </button>
        </div>
      </div>
    </section>

    <header class="alarm-levels__subhead">
      <strong>报警升级</strong>
      <button type="button" class="alarm-levels__secondary" @click="addRule">
        <IconTablerPlus />新增规则
      </button>
    </header>
    <section class="alarm-levels__section">
      <div class="alarm-levels__rule-head" aria-hidden="true">
        <span>启用</span><span>当前级别</span><span>未确认持续（秒）</span><span>升级到</span
        ><span></span>
      </div>
      <div v-for="rule in draft.escalationRules" :key="rule.id" class="alarm-levels__rule-row">
        <el-switch v-model="rule.isEnabled" />
        <el-select v-model="rule.sourceSeverity" @change="normalizeRuleTarget(rule)">
          <el-option
            v-for="item in sourceDefinitions"
            :key="item.key"
            :label="item.displayName"
            :value="item.key"
          />
        </el-select>
        <el-input-number
          v-model="rule.unacknowledgedSeconds"
          :min="1"
          :max="604800"
          controls-position="right"
        />
        <el-select v-model="rule.targetSeverity">
          <el-option
            v-for="item in targetDefinitions(rule.sourceSeverity)"
            :key="item.key"
            :label="item.displayName"
            :value="item.key"
          />
        </el-select>
        <button
          type="button"
          class="alarm-levels__remove"
          title="删除升级规则"
          @click="removeRule(rule.id)"
        >
          <IconTablerTrash />
        </button>
      </div>
      <el-empty v-if="!draft.escalationRules.length" :image-size="42" description="暂无升级规则" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerChevronUp from '~icons/tabler/chevron-up'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTrash from '~icons/tabler/trash'
import { getAlarmLevelSettings, saveAlarmLevelSettings } from '@/api/alarm.api'
import type {
  AlarmEscalationRule,
  AlarmLevelSettings,
  AlarmSeverityDefinition,
} from '@/api/schemas/alarm.schema'
import { getApiErrorMessage } from '@/utils/request'

const props = defineProps<{ projectId: string }>()
const emit = defineEmits<{ changed: [settings: AlarmLevelSettings] }>()
const loading = ref(false)
const saving = ref(false)
const draft = reactive<{
  severityDefinitions: AlarmSeverityDefinition[]
  escalationRules: AlarmEscalationRule[]
}>({
  severityDefinitions: [],
  escalationRules: [],
})
const sourceDefinitions = computed(() => draft.severityDefinitions.slice(0, -1))

function applySettings(settings: AlarmLevelSettings) {
  draft.severityDefinitions = settings.severityDefinitions.map((item) => ({ ...item }))
  draft.escalationRules = settings.escalationRules.map((item) => ({ ...item }))
  emit('changed', settings)
}

async function load() {
  if (!props.projectId) return
  loading.value = true
  try {
    applySettings(await getAlarmLevelSettings(props.projectId))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载报警级别失败'))
  } finally {
    loading.value = false
  }
}

function nextCustomKey() {
  let index = 1
  const keys = new Set(draft.severityDefinitions.map((item) => item.key))
  while (keys.has(`custom_${index}`)) index += 1
  return `custom_${index}`
}

function addDefinition() {
  draft.severityDefinitions.push({
    key: nextCustomKey(),
    displayName: '自定义级别',
    color: '#8b5cf6',
    sortOrder: (draft.severityDefinitions.length + 1) * 10,
    isBuiltin: false,
  })
  refreshSortOrder()
}

function moveDefinition(index: number, offset: number) {
  const target = index + offset
  if (target < 0 || target >= draft.severityDefinitions.length) return
  const [item] = draft.severityDefinitions.splice(index, 1)
  if (item) draft.severityDefinitions.splice(target, 0, item)
  refreshSortOrder()
  draft.escalationRules.forEach(normalizeRuleTarget)
}

function removeDefinition(key: string) {
  draft.severityDefinitions = draft.severityDefinitions.filter((item) => item.key !== key)
  draft.escalationRules = draft.escalationRules.filter(
    (item) => item.sourceSeverity !== key && item.targetSeverity !== key,
  )
  refreshSortOrder()
}

function refreshSortOrder() {
  draft.severityDefinitions.forEach((item, index) => {
    item.sortOrder = (index + 1) * 10
  })
}

function targetDefinitions(sourceSeverity: string) {
  const index = draft.severityDefinitions.findIndex((item) => item.key === sourceSeverity)
  return index < 0 ? [] : draft.severityDefinitions.slice(index + 1)
}

function normalizeRuleTarget(rule: AlarmEscalationRule) {
  const targets = targetDefinitions(rule.sourceSeverity)
  if (!targets.some((item) => item.key === rule.targetSeverity))
    rule.targetSeverity = targets[0]?.key || ''
}

function addRule() {
  const source = sourceDefinitions.value[0]
  const target = source ? targetDefinitions(source.key)[0] : undefined
  if (!source || !target) return
  draft.escalationRules.push({
    id: crypto.randomUUID(),
    sourceSeverity: source.key,
    targetSeverity: target.key,
    unacknowledgedSeconds: 300,
    isEnabled: true,
  })
}

function removeRule(id: string) {
  draft.escalationRules = draft.escalationRules.filter((item) => item.id !== id)
}

async function save() {
  saving.value = true
  try {
    refreshSortOrder()
    const result = await saveAlarmLevelSettings(props.projectId, {
      severityDefinitions: draft.severityDefinitions,
      escalationRules: draft.escalationRules,
    })
    applySettings(result)
    ElMessage.success('级别与升级配置已保存')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存级别与升级配置失败'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
watch(() => props.projectId, load)
</script>

<style scoped>
.alarm-levels {
  height: 100%;
  overflow: auto;
  padding: 18px;
  color: var(--dc-text);
}
.alarm-levels__toolbar,
.alarm-levels__subhead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.alarm-levels__toolbar > div:first-child {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.alarm-levels__toolbar span {
  color: var(--dc-text-muted);
  font-size: 12px;
}
.alarm-levels__actions,
.alarm-levels__row-actions {
  display: flex;
  gap: 6px;
}
.alarm-levels__section {
  display: grid;
  gap: 6px;
  margin-top: 12px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.alarm-levels__subhead {
  margin-top: 18px;
}
.alarm-levels__level-head,
.alarm-levels__level-row {
  display: grid;
  grid-template-columns: 54px minmax(140px, 1fr) 160px 112px;
  gap: 8px;
  align-items: center;
}
.alarm-levels__rule-head,
.alarm-levels__rule-row {
  display: grid;
  grid-template-columns: 52px minmax(130px, 1fr) 160px minmax(130px, 1fr) 34px;
  gap: 8px;
  align-items: center;
}
.alarm-levels__level-head,
.alarm-levels__rule-head {
  padding: 0 6px;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.alarm-levels__level-row,
.alarm-levels__rule-row {
  min-height: 42px;
  padding: 6px;
  border: 1px solid var(--dc-border);
  border-radius: 7px;
  background: var(--dc-surface-muted);
}
.alarm-levels__color {
  width: 34px;
  height: 30px;
  padding: 2px;
  border: 1px solid var(--dc-border);
  border-radius: 6px;
  background: var(--dc-surface-raised);
}
.alarm-levels__row-actions button,
.alarm-levels__remove {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-secondary);
}
.alarm-levels__row-actions button:disabled {
  opacity: 0.35;
}
.alarm-levels__row-actions .is-danger,
.alarm-levels__remove {
  color: var(--el-color-danger);
}
.alarm-levels__secondary,
.alarm-levels__primary {
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;
  border-radius: 7px;
  font: inherit;
  font-size: 13px;
}
.alarm-levels__secondary {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-levels__primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: #fff;
}
.alarm-levels :deep(.el-input-number),
.alarm-levels :deep(.el-select) {
  width: 100%;
}
</style>
