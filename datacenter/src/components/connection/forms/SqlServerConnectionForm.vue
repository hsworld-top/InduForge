<template>
  <el-form :model="formData" :rules="rules" ref="formRef" label-width="120px">
    <el-form-item label="连接名称" prop="name">
      <el-input v-model="formData.name" placeholder="请输入连接名称" />
    </el-form-item>

    <el-form-item label="主机地址" prop="host">
      <el-input v-model="formData.host" placeholder="例如: localhost" />
    </el-form-item>

    <el-form-item label="端口" prop="port">
      <el-input-number v-model="formData.port" :min="1" :max="65535" class="w-full" />
    </el-form-item>

    <el-form-item label="数据库名" prop="database">
      <el-input v-model="formData.database" placeholder="请输入数据库名" />
    </el-form-item>

    <el-form-item label="用户名" prop="username">
      <el-input v-model="formData.username" placeholder="请输入用户名" />
    </el-form-item>

    <el-form-item label="密码" prop="password">
      <el-input
        v-model="formData.password"
        type="password"
        placeholder="请输入密码"
        show-password
      />
    </el-form-item>

    <el-form-item label="加密连接">
      <el-switch v-model="formData.encrypt" />
      <span class="ml-2 text-sm text-gray-500">启用 TLS/SSL 加密</span>
    </el-form-item>

    <el-form-item label="信任证书" v-if="formData.encrypt">
      <el-switch v-model="formData.trustServerCertificate" />
      <span class="ml-2 text-sm text-gray-500">信任服务器自签名证书</span>
    </el-form-item>

    <el-form-item label="连接超时">
      <el-input-number
        v-model="formData.timeout"
        :min="1000"
        :max="300000"
        :step="1000"
        class="w-full"
      />
      <span class="ml-2 text-sm text-gray-500">毫秒</span>
    </el-form-item>
  </el-form>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
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
  trustServerCertificate: props.modelValue.trustServerCertificate !== undefined ? props.modelValue.trustServerCertificate : true,
  timeout: props.modelValue.timeout || 60000
})

const rules = {
  name: [{ required: true, message: '请输入连接名称', trigger: 'blur' }],
  host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
  database: [{ required: true, message: '请输入数据库名', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

watch(
  formData,
  (newVal) => {
    emit('update:modelValue', { ...newVal })
  },
  { deep: true }
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
  clearValidate
})
</script>
