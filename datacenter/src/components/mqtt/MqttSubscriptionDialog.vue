<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    :title="mode === 'create' ? t('subscription.create') : t('subscription.edit')"
    width="600px"
    :dirty="isDirty"
    @close="handleClosed"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item :label="t('subscription.nameLabel')" prop="name">
        <el-input
          v-model="formData.name"
          :placeholder="t('subscription.namePlaceholder')"
          maxlength="100"
          show-word-limit
        />
      </el-form-item>

      <el-form-item :label="t('subscription.topicLabel')" prop="topic">
        <el-input
          v-model="formData.topic"
          :placeholder="t('subscription.topicPlaceholder')"
          maxlength="500"
        >
          <template #append>
            <el-tooltip :content="t('subscription.wildcardTooltip')" placement="top">
              <el-icon><IconTablerQuestionMark /></el-icon>
            </el-tooltip>
          </template>
        </el-input>
        <div class="text-xs text-gray-400 mt-1">
          {{ t('subscription.wildcardHint') }}
        </div>
      </el-form-item>

      <el-form-item :label="t('subscription.qosLevel')" prop="qos">
        <el-radio-group v-model="formData.qos">
          <el-radio :label="0">QoS 0 (最多一次)</el-radio>
          <el-radio :label="1">QoS 1 (至少一次)</el-radio>
          <el-radio :label="2">QoS 2 (恰好一次)</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item :label="t('subscription.usageMode')" prop="usageMode">
        <el-select
          v-model="formData.usageMode"
          class="mqtt-subscription-dialog__usage-select"
          :disabled="mode === 'edit'"
          placeholder="请选择使用方式"
        >
          <el-option
            v-for="option in usageModeOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </el-select>
        <div v-if="mode === 'edit'" class="mqtt-subscription-dialog__hint">
          {{ t('subscription.usageModeLockedHint') }}
        </div>
      </el-form-item>

      <el-form-item :label="t('subscription.remark')" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          :placeholder="t('subscription.descriptionPlaceholder')"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="flex justify-end space-x-2">
        <el-button @click="requestClose">{{ t('actions.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ mode === 'create' ? t('actions.create') : t('actions.save') }}
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import DcDialog from '@/components/shared/DcDialog.vue'
import IconTablerQuestionMark from '~icons/tabler/question-mark'
import dataAPI from '@/api/data.api'
import { Storage } from '@/utils/storage'
import { getCurrentProjectId, isWujieMicroApp } from '@/runtime/wujie-context'
import { t } from '@/i18n/runtime'
import { getApiErrorMessage } from '@/utils/request'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  connectionId: {
    type: String,
    required: true,
  },
  projectId: {
    type: String,
    default: '',
  },
  groupId: {
    type: [String, null],
    default: null,
  },
  subscription: {
    type: Object,
    default: null,
  },
  mode: {
    type: String,
    default: 'create',
    validator: (value) => ['create', 'edit'].includes(value),
  },
})

const emit = defineEmits(['update:modelValue', 'success'])

const formRef = ref(null)
const submitting = ref(false)
const initialFormSnapshot = ref('')
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const defaultSubscriptionUsageMode = 'batch_variable'

// 表单数据
const formData = ref({
  name: '',
  topic: '',
  qos: 0,
  usageMode: defaultSubscriptionUsageMode,
  description: '',
  groupId: null,
})

const usageModeOptions = computed(() => [
  { label: t('subscription.usageRawDatapoint'), value: 'raw_datapoint' },
  { label: t('subscription.usageSingleVariable'), value: 'single_variable' },
  { label: t('subscription.usageBatchVariable'), value: 'batch_variable' },
])

// 验证规则
const rules = {
  name: [
    {
      required: true,
      message: t('subscription.namePlaceholder'),
      trigger: 'blur',
    },
    {
      min: 2,
      max: 100,
      message: t('subscription.nameLengthError'),
      trigger: 'blur',
    },
  ],
  topic: [
    {
      required: true,
      message: t('subscription.topicPlaceholder'),
      trigger: 'blur',
    },
    {
      pattern: /^[a-zA-Z0-9_/#+-]+$/,
      message: t('subscription.topicPatternError'),
      trigger: 'blur',
    },
  ],
  qos: [{ required: true, message: t('subscription.qosLevel'), trigger: 'change' }],
  usageMode: [{ required: true, message: t('subscription.usageMode'), trigger: 'change' }],
}

// 计算属性：对话框可见性
const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const formSnapshot = computed(() => JSON.stringify(formData.value))
const isDirty = computed(() => visible.value && formSnapshot.value !== initialFormSnapshot.value)

// 正式入口由 Wujie 上下文提供工程 ID；debug 才允许从 URL 或本地调试缓存回退。
const getProjectId = () => {
  const projectId = getCurrentProjectId()
  if (projectId) return projectId

  if (isWujieMicroApp()) return null

  const projectIdFromStorage = Storage.getProjectId()
  if (projectIdFromStorage) {
    return projectIdFromStorage
  }

  const urlParams = new window.URLSearchParams(window.location.search)
  return urlParams.get('pid') || urlParams.get('id')
}

/**
 * 初始化表单数据
 */
const initFormData = () => {
  if (props.mode === 'edit' && props.subscription) {
    formData.value = {
      name: props.subscription.name || '',
      topic: props.subscription.topic || '',
      qos: props.subscription.qos ?? 0,
      usageMode: props.subscription.usageMode || 'single_variable',
      description: props.subscription.description || '',
      groupId: props.groupId ?? props.subscription.groupId ?? null,
    }
  } else {
    formData.value = {
      name: '',
      topic: '',
      qos: 0,
      usageMode: defaultSubscriptionUsageMode,
      description: '',
      groupId: props.groupId ?? null,
    }
  }
  initialFormSnapshot.value = formSnapshot.value
}

/**
 * 提交表单
 */
const handleSubmit = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    submitting.value = true

    const projectId = props.projectId || getProjectId()
    if (!projectId) {
      ElMessage.error(t('subscription.projectMissing'))
      return
    }

    const data = {
      name: formData.value.name,
      topic: formData.value.topic,
      qos: formData.value.qos,
      usageMode: formData.value.usageMode,
      description: formData.value.description || null,
      groupId: formData.value.groupId || null,
    }

    let response
    if (props.mode === 'create') {
      response = await dataAPI.createMqttSubscription(projectId, props.connectionId, data)
    } else {
      response = await dataAPI.updateMqttSubscription(projectId, props.subscription.id, {
        ...data,
        hasGroupId: true,
      })
    }

    ElMessage.success(
      props.mode === 'create' ? t('subscription.createSuccess') : t('subscription.updateSuccess'),
    )
    initialFormSnapshot.value = formSnapshot.value
    emit('success', response.data)
    dialogRef.value?.closeSilently()
  } catch (error) {
    if (error.errors) {
      // 表单验证错误
      return
    }
    ElMessage.error(
      props.mode === 'create'
        ? t('subscription.createFailed', {
            message: getApiErrorMessage(error, '创建订阅失败'),
          })
        : t('subscription.updateFailed', {
            message: getApiErrorMessage(error, '更新订阅失败'),
          }),
    )
  } finally {
    submitting.value = false
  }
}

/**
 * 关闭对话框
 */
const requestClose = () => {
  void dialogRef.value?.requestClose()
}

const handleClosed = () => {
  if (formRef.value) {
    formRef.value.resetFields()
  }
}

// 监听对话框打开
watch(visible, (newVal) => {
  if (newVal) {
    initFormData()
  }
})

// 监听订阅数据变化
watch(
  () => props.subscription,
  () => {
    if (props.mode === 'edit' && props.subscription) {
      initFormData()
    }
  },
  { deep: true },
)
</script>

<style scoped>
:deep(.el-form-item__label) {
  font-weight: 500;
}

.mqtt-subscription-dialog__usage-select {
  width: 100%;
}

.mqtt-subscription-dialog__hint {
  margin-top: 6px;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.5;
}
</style>
