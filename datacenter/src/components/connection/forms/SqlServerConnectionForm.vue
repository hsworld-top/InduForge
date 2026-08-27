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
        <el-input v-model.number="formData.port" inputmode="numeric" placeholder="1433" />
      </el-form-item>
    </div>

    <el-form-item :label="t('connection.database')" prop="database">
      <el-input v-model="formData.database" :placeholder="t('connection.database')" />
    </el-form-item>

    <div class="database-connection-form__grid">
      <el-form-item :label="t('connection.username')" prop="username">
        <el-input v-model="formData.username" :placeholder="t('connection.username')" />
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
      database-type="sqlserver"
      :saved-secrets="savedTlsSecrets"
    />

    <el-form-item :label="t('connection.connectionTimeout')">
      <el-input v-model.number="formData.timeout" inputmode="numeric" placeholder="60000">
        <template #append>{{ t('common.milliseconds') }}</template>
      </el-input>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { t } from '@/i18n/runtime'
import SavedPasswordInput from './SavedPasswordInput.vue'
import RelationalTlsFields from './RelationalTlsFields.vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({}),
  },
  mode: { type: String, default: 'create' },
  savedPasswordConfigured: { type: Boolean, default: false },
  loadingSavedPassword: { type: Boolean, default: false },
  savedTlsSecrets: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['update:modelValue', 'reveal-saved-password'])

const formRef = ref(null)

const formData = ref({
  name: props.modelValue.name || '',
  host: props.modelValue.host || 'localhost',
  port: props.modelValue.port || 1433,
  database: props.modelValue.database || '',
  username: props.modelValue.username || 'sa',
  password: props.modelValue.password || '',
  sslConfig: props.modelValue.sslConfig || { mode: 'disable', ca: '', cert: '', key: '' },
  timeout: props.modelValue.timeout || 60000,
})

const rules = {
  name: [{ required: true, message: t('connection.name'), trigger: 'blur' }],
  host: [{ required: true, message: t('connection.host'), trigger: 'blur' }],
  port: [{ required: true, message: t('connection.port'), trigger: 'blur' }],
  database: [{ required: true, message: t('connection.database'), trigger: 'blur' }],
  username: [{ required: true, message: t('connection.username'), trigger: 'blur' }],
  password:
    props.mode === 'create' || !props.savedPasswordConfigured
      ? [{ required: true, message: t('connection.password'), trigger: 'blur' }]
      : [],
}

watch(
  formData,
  (newVal) => {
    emit('update:modelValue', { ...newVal })
  },
  { deep: true },
)

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue.password !== formData.value.password) {
      formData.value.password = newValue.password || ''
    }
  },
  { deep: true },
)

const validate = () => {
  return formRef.value.validate()
}

const clearValidate = () => {
  if (formRef.value) {
    formRef.value.clearValidate()
  }
}

defineExpose({
  validate,
  clearValidate,
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
