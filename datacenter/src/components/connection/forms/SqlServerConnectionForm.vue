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
        <el-input
          v-model="formData.password"
          type="password"
          :placeholder="t('connection.password')"
          show-password
        />
      </el-form-item>
    </div>

    <el-form-item :label="t('connection.encryption')">
      <el-switch v-model="formData.encrypt" />
      <span class="ml-2 text-sm text-gray-500">{{ t('connection.encryptionHint') }}</span>
    </el-form-item>

    <el-form-item :label="t('connection.trustCertificate')" v-if="formData.encrypt">
      <el-switch v-model="formData.trustServerCertificate" />
      <span class="ml-2 text-sm text-gray-500">{{ t('connection.trustCertificateHint') }}</span>
    </el-form-item>

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

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({}),
  },
})

const emit = defineEmits(['update:modelValue'])

const formRef = ref(null)

const formData = ref({
  name: props.modelValue.name || '',
  host: props.modelValue.host || 'localhost',
  port: props.modelValue.port || 1433,
  database: props.modelValue.database || '',
  username: props.modelValue.username || 'sa',
  password: props.modelValue.password || '',
  encrypt: props.modelValue.encrypt !== undefined ? props.modelValue.encrypt : false,
  trustServerCertificate:
    props.modelValue.trustServerCertificate !== undefined
      ? props.modelValue.trustServerCertificate
      : true,
  timeout: props.modelValue.timeout || 60000,
})

const rules = {
  name: [{ required: true, message: t('connection.name'), trigger: 'blur' }],
  host: [{ required: true, message: t('connection.host'), trigger: 'blur' }],
  port: [{ required: true, message: t('connection.port'), trigger: 'blur' }],
  database: [{ required: true, message: t('connection.database'), trigger: 'blur' }],
  username: [{ required: true, message: t('connection.username'), trigger: 'blur' }],
  password: [{ required: true, message: t('connection.password'), trigger: 'blur' }],
}

watch(
  formData,
  (newVal) => {
    emit('update:modelValue', { ...newVal })
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
