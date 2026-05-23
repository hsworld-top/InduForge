<template>
  <DcDialog
    v-model="visible"
    :title="mode === 'create' ? '新建分组' : '重命名分组'"
    width="460px"
    body-max-height="320px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="mqtt-subscription-group-dialog">
      <el-form-item label="名称" required>
        <el-input
          v-model="form.name"
          maxlength="60"
          show-word-limit
          placeholder="输入分组名称"
          @keyup.enter="submit"
        />
      </el-form-item>

      <el-form-item v-if="mode === 'create'" label="上级分组">
        <el-select
          v-model="form.parentId"
          class="mqtt-subscription-group-dialog__select"
          clearable
          placeholder="根目录"
        >
          <el-option label="根目录" :value="null" />
          <el-option
            v-for="group in groupOptions"
            :key="group.id"
            :label="group.label"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="mqtt-subscription-group-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">
          {{ mode === 'create' ? '创建' : '保存' }}
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { MqttSubscriptionGroup, MqttSubscriptionGroupNode } from './mqttSubscriptionTreeModel'
import { flattenMqttSubscriptionGroups } from './mqttSubscriptionTreeModel'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    mode: 'create' | 'rename'
    groups: MqttSubscriptionGroup[]
    group?: MqttSubscriptionGroupNode | null
    loading?: boolean
  }>(),
  {
    group: null,
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', data: { name: string; parentId: string | null }): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const form = reactive({
  name: '',
  parentId: null as string | null,
})

const groupOptions = computed(() => flattenMqttSubscriptionGroups(props.groups))
const canSubmit = computed(() => {
  const name = form.name.trim()
  if (!name || props.loading) return false
  if (props.mode === 'rename') return name !== props.group?.name
  return true
})
const isDirty = computed(() => {
  if (!visible.value) return false
  if (props.mode === 'rename') {
    return form.name.trim() !== (props.group?.name || '')
  }
  return form.name.trim().length > 0 || Boolean(form.parentId)
})

function resetForm() {
  form.name = props.mode === 'rename' ? props.group?.name || '' : ''
  form.parentId = null
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', {
    name: form.name.trim(),
    parentId: form.parentId || null,
  })
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm()
  },
)

watch(
  () => props.group?.id,
  () => {
    if (props.modelValue) resetForm()
  },
)
</script>

<style scoped>
.mqtt-subscription-group-dialog {
  display: grid;
  gap: 2px;
}

.mqtt-subscription-group-dialog__select {
  width: 100%;
}

.mqtt-subscription-group-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
