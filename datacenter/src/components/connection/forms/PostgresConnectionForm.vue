<template>
  <el-form
    ref="formRef"
    :model="formData"
    :rules="rules"
    label-position="top"
    class="database-connection-form"
  >
    <el-form-item :label="t('connection.name')" prop="name">
      <el-input v-model="formData.name" :placeholder="t('connection.name')" />
    </el-form-item>

    <div class="database-connection-form__grid">
      <el-form-item :label="t('connection.host')" prop="host">
        <el-input v-model="formData.host" placeholder="localhost" />
      </el-form-item>

      <el-form-item :label="t('connection.port')" prop="port">
        <el-input v-model.number="formData.port" inputmode="numeric" placeholder="5432" />
      </el-form-item>
    </div>

    <div class="database-connection-form__grid">
      <el-form-item :label="t('connection.database')" prop="database">
        <el-input v-model="formData.database" :placeholder="t('connection.database')" />
      </el-form-item>

      <el-form-item :label="t('connection.schema')">
        <el-input v-model="formData.schema" placeholder="public" />
      </el-form-item>
    </div>

    <div class="database-connection-form__grid">
      <el-form-item :label="t('connection.username')" prop="username">
        <el-input v-model="formData.username" placeholder="postgres" />
      </el-form-item>

      <el-form-item :label="t('connection.password')" prop="password">
        <SavedPasswordInput
          v-model="formData.password"
          :placeholder="t('connection.password')"
          :saved-password-configured="savedPasswordConfigured"
          :loading="loadingSavedPassword"
          @reveal-saved-password="$emit('reveal-saved-password')"
        />
      </el-form-item>
    </div>

    <RelationalTlsFields
      v-model="formData.sslConfig"
      database-type="postgresql"
      :saved-secrets="savedTlsSecrets"
    />

    <div class="database-connection-form__grid">
      <el-form-item :label="t('connection.connectionTimeout')">
        <el-input
          v-model.number="formData.connectionTimeout"
          inputmode="numeric"
          placeholder="3000"
        >
          <template #append>{{ t('common.milliseconds') }}</template>
        </el-input>
      </el-form-item>

      <el-form-item :label="t('connection.queryTimeout')">
        <el-input v-model.number="formData.queryTimeout" inputmode="numeric" placeholder="30000">
          <template #append>{{ t('common.milliseconds') }}</template>
        </el-input>
      </el-form-item>
    </div>

    <el-form-item :label="t('connection.maxConnections')">
      <el-input v-model.number="formData.maxConnections" inputmode="numeric" placeholder="10" />
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { getDefaultConfig } from '@/config/connectionTypes'
import { t } from '@/i18n/runtime'
import SavedPasswordInput from './SavedPasswordInput.vue'
import RelationalTlsFields from './RelationalTlsFields.vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({}),
  },
  mode: {
    type: String,
    default: 'create', // 'create' | 'edit'
    validator: (value) => ['create', 'edit'].includes(value),
  },
  savedPasswordConfigured: { type: Boolean, default: false },
  loadingSavedPassword: { type: Boolean, default: false },
  savedTlsSecrets: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['update:modelValue', 'validate', 'reveal-saved-password'])

const formRef = ref(null)
const formData = ref({
  name: '',
  host: 'localhost',
  port: 5432,
  database: '',
  username: 'postgres',
  password: '',
  schema: 'public',
  sslConfig: { mode: 'disable', ca: '', cert: '', key: '' },
  connectionTimeout: 3000,
  queryTimeout: 30000,
  maxConnections: 10,
})

// 防止循环更新的标志
const isUpdatingFromParent = ref(false)

const rules = {
  name: [
    { required: true, message: t('connection.name'), trigger: 'blur' },
    {
      min: 2,
      max: 100,
      message: t('query.nameLengthError'),
      trigger: 'blur',
    },
  ],
  host: [{ required: true, message: t('connection.host'), trigger: 'blur' }],
  port: [{ required: true, message: t('connection.port'), trigger: 'blur' }],
  database: [{ required: true, message: t('connection.database'), trigger: 'blur' }],
  username: [{ required: true, message: t('connection.username'), trigger: 'blur' }],
}

// 初始化表单数据
onMounted(() => {
  isUpdatingFromParent.value = true
  if (props.mode === 'create') {
    // 创建模式：使用默认配置
    const defaultConfig = getDefaultConfig('postgresql')
    formData.value = { ...defaultConfig, ...props.modelValue }
  } else {
    // 编辑模式：使用传入的数据
    formData.value = { ...formData.value, ...props.modelValue }
  }
  // 使用 nextTick 确保更新完成后再重置标志
  setTimeout(() => {
    isUpdatingFromParent.value = false
  }, 0)
})

// 监听表单数据变化，向上传递
watch(
  formData,
  (newValue) => {
    // 只有在不是从父组件更新时才向上传递
    if (!isUpdatingFromParent.value) {
      emit('update:modelValue', { ...newValue })
    }
  },
  { deep: true },
)

// 监听外部数据变化
watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue && Object.keys(newValue).length > 0) {
      isUpdatingFromParent.value = true
      formData.value = { ...formData.value, ...newValue }
      setTimeout(() => {
        isUpdatingFromParent.value = false
      }, 0)
    }
  },
  { deep: true },
)

/**
 * 验证表单
 */
const validate = async () => {
  if (!formRef.value) return false

  try {
    await formRef.value.validate()
    emit('validate', true, formData.value)
    return true
  } catch {
    emit('validate', false, null)
    return false
  }
}

/**
 * 重置表单
 */
const resetFields = () => {
  if (formRef.value) {
    formRef.value.resetFields()
  }
}

/**
 * 清空验证
 */
const clearValidate = () => {
  if (formRef.value) {
    formRef.value.clearValidate()
  }
}

// 暴露方法给父组件
defineExpose({
  validate,
  resetFields,
  clearValidate,
  formData,
})
</script>

<style scoped>
.database-connection-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
}

.database-connection-form__grid > .el-form-item {
  min-width: 0;
}

@media (max-width: 760px) {
  .database-connection-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
