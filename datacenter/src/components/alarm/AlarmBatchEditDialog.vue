<template>
  <DcDrawer v-model="visible" title="批量修改报警项" :width="560" :max="640">
    <div class="alarm-batch">
      <div class="alarm-batch__notice">
        将修改 <strong>{{ affectedCount }}</strong> 条报警项。只有勾选的字段会变化。
      </div>
      <label class="alarm-batch__row"
        ><el-checkbox v-model="fields.isEnabled">启停状态</el-checkbox
        ><el-switch v-model="patch.isEnabled" :disabled="!fields.isEnabled"
      /></label>
      <label class="alarm-batch__row"
        ><el-checkbox v-model="fields.groupId">移动目录</el-checkbox
        ><el-select
          v-model="patch.groupId"
          :disabled="!fields.groupId"
          clearable
          placeholder="根目录"
          ><el-option label="根目录" :value="null" /><el-option
            v-for="group in groups"
            :key="group.id"
            :label="group.fullPath || group.name"
            :value="group.id" /></el-select
      ></label>
      <label class="alarm-batch__row"
        ><el-checkbox v-model="fields.notification">通知方式</el-checkbox
        ><el-select v-model="patch.notification.mode" :disabled="!fields.notification"
          ><el-option label="沿用工程设置" value="inherit" /><el-option
            label="不通知"
            value="off" /><el-option label="单独设置" value="custom" /></el-select
      ></label>
      <section v-if="conditionsCompatible" class="alarm-batch__conditions">
        <el-checkbox v-model="fields.conditions">统一替换报警条件</el-checkbox>
        <p>替换会覆盖所选报警项原有条件。各报警项仍保持独立身份。</p>
        <div v-if="fields.conditions" class="alarm-batch__condition-list">
          <article v-for="condition in patch.conditions" :key="condition.id">
            <el-input v-model="condition.label" placeholder="等级标签" />
            <el-select v-model="condition.severity"
              ><el-option label="提示" value="info" /><el-option
                label="警告"
                value="warning" /><el-option label="重要" value="major" /><el-option
                label="紧急"
                value="critical"
            /></el-select>
            <el-input-number
              v-if="condition.kind === 'threshold'"
              :model-value="Number(condition.params.threshold)"
              @update:model-value="condition.params = { ...condition.params, threshold: $event }"
            />
            <el-input-number
              v-else-if="condition.kind === 'rate_of_change'"
              :model-value="Number(condition.params.limit)"
              @update:model-value="condition.params = { ...condition.params, limit: $event }"
            />
          </article>
        </div>
      </section>
      <div v-else class="alarm-batch__muted">所选报警类型不同，只能修改启停、目录和通知。</div>
      <footer>
        <button type="button" class="dc-button" @click="visible = false">取消</button
        ><button
          type="button"
          class="dc-button dc-button--primary"
          :disabled="saving"
          @click="submit"
        >
          确认修改
        </button>
      </footer>
    </div>
  </DcDrawer>
</template>
<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import DcDrawer from '@/components/shared/DcDrawer.vue'
import type {
  AlarmBatchUpdate,
  AlarmGroup,
  AlarmItem,
  AlarmItemSelection,
} from '@/api/schemas/alarm.schema'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    selection: AlarmItemSelection
    affectedCount: number
    items?: AlarmItem[]
    groups?: AlarmGroup[]
    saving?: boolean
  }>(),
  { items: () => [], groups: () => [], saving: false },
)
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: AlarmBatchUpdate]
}>()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const fields = reactive({
  isEnabled: false,
  groupId: false,
  notification: false,
  conditions: false,
})
const patch = reactive({
  isEnabled: true,
  groupId: null as string | null,
  notification: { mode: 'inherit' as const, channelIds: [] as string[], messageTemplate: '' },
  conditions: [] as AlarmItem['conditions'],
})
const conditionsCompatible = computed(
  () =>
    props.items.length > 0 &&
    props.items.every(
      (item) =>
        item.mode === props.items[0]?.mode &&
        item.alarmType === props.items[0]?.alarmType &&
        item.evaluationMode === props.items[0]?.evaluationMode,
    ),
)
watch(
  () => props.modelValue,
  (opened) => {
    if (!opened) return
    Object.assign(fields, {
      isEnabled: false,
      groupId: false,
      notification: false,
      conditions: false,
    })
    patch.conditions =
      props.items[0]?.conditions.map((condition) => ({
        ...condition,
        params: { ...condition.params },
      })) || []
  },
)
function submit() {
  const selectedFields = (Object.keys(fields) as Array<keyof typeof fields>).filter(
    (key) => fields[key],
  )
  if (!selectedFields.length) return ElMessage.warning('请至少勾选一个要修改的字段')
  emit('save', {
    selection: props.selection,
    fields: selectedFields,
    patch: {
      isEnabled: patch.isEnabled,
      groupId: patch.groupId,
      notification: patch.notification,
      conditions: patch.conditions,
    },
    acknowledgedWarningKeys: [],
  })
}
</script>
<style scoped>
.alarm-batch {
  display: grid;
  gap: 14px;
}
.alarm-batch__notice {
  padding: 10px 12px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-text-secondary);
}
.alarm-batch__row {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}
.alarm-batch__conditions {
  display: grid;
  gap: 8px;
  padding-top: 14px;
  border-top: 1px solid var(--dc-border);
}
.alarm-batch__conditions p,
.alarm-batch__muted {
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-batch__condition-list {
  display: grid;
  gap: 8px;
}
.alarm-batch__condition-list article {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 110px 130px;
  gap: 8px;
}
.alarm-batch footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 14px;
  border-top: 1px solid var(--dc-border);
}
</style>
