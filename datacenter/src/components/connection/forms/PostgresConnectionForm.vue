<template>
  <el-form
    ref="formRef"
    :model="formData"
    :rules="rules"
    label-width="120px"
  >
    <el-form-item label="连接名称" prop="name">
      <el-input v-model="formData.name" placeholder="请输入连接名称" />
    </el-form-item>

    <el-form-item label="主机地址" prop="host">
      <el-input v-model="formData.host" placeholder="localhost" />
    </el-form-item>

    <el-form-item label="端口" prop="port">
      <el-input-number v-model="formData.port" :min="1" :max="65535" class="w-full" />
    </el-form-item>

    <el-form-item label="数据库名" prop="database">
      <el-input v-model="formData.database" placeholder="请输入数据库名" />
    </el-form-item>

    <el-form-item label="用户名" prop="username">
      <el-input v-model="formData.username" placeholder="postgres" />
    </el-form-item>

    <el-form-item label="密码" prop="password">
      <el-input v-model="formData.password" type="password" placeholder="请输入密码" show-password />
    </el-form-item>

    <el-form-item label="Schema">
      <el-input v-model="formData.schema" placeholder="public" />
      <span class="text-xs text-gray-500 ml-2">默认使用 public schema</span>
    </el-form-item>

    <el-form-item label="SSL 连接">
      <el-select v-model="formData.sslMode" placeholder="选择 SSL 模式" class="w-full" @change="handleSslModeChange">
        <el-option label="禁用" value="disable">
          <span>禁用</span>
          <span class="text-xs text-gray-400 ml-2">- 不使用 SSL（仅本地开发）</span>
        </el-option>
        <el-option label="首选" value="prefer">
          <span>首选</span>
          <span class="text-xs text-gray-400 ml-2">- 优先 SSL，失败则降级（推荐）</span>
        </el-option>
        <el-option label="必需" value="require">
          <span>必需</span>
          <span class="text-xs text-gray-400 ml-2">- 必须使用 SSL</span>
        </el-option>
        <el-option label="验证证书" value="verify-ca">
          <span>验证证书</span>
          <span class="text-xs text-gray-400 ml-2">- 验证服务器证书</span>
        </el-option>
      </el-select>
    </el-form-item>

    <!-- SSL 证书配置（仅在 verify-ca 模式下显示） -->
    <template v-if="needsCertificate">
      <el-divider content-position="left">SSL 证书配置</el-divider>
      
      <el-form-item label="CA 证书">
        <el-input
          v-model="formData.sslCa"
          type="textarea"
          :rows="4"
          placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
        />
        <span class="text-xs text-gray-500 ml-2">粘贴服务器 CA 证书内容（PEM 格式）</span>
      </el-form-item>

      <el-form-item label="客户端证书">
        <el-input
          v-model="formData.sslCert"
          type="textarea"
          :rows="4"
          placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
        />
        <span class="text-xs text-gray-500 ml-2">粘贴客户端证书内容（PEM 格式，可选）</span>
      </el-form-item>

      <el-form-item label="客户端密钥">
        <el-input
          v-model="formData.sslKey"
          type="textarea"
          :rows="4"
          placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
        />
        <span class="text-xs text-gray-500 ml-2">粘贴客户端私钥内容（PEM 格式，可选）</span>
      </el-form-item>
    </template>

    <el-form-item label="连接超时">
      <el-input-number
        v-model="formData.connectionTimeout"
        :min="1000"
        :max="60000"
        :step="1000"
        class="w-full"
      />
      <span class="text-xs text-gray-500 ml-2">毫秒（默认 3000ms）</span>
    </el-form-item>

    <el-form-item label="查询超时">
      <el-input-number
        v-model="formData.queryTimeout"
        :min="1000"
        :max="300000"
        :step="1000"
        class="w-full"
      />
      <span class="text-xs text-gray-500 ml-2">毫秒</span>
    </el-form-item>

    <el-form-item label="最大连接数">
      <el-input-number
        v-model="formData.maxConnections"
        :min="1"
        :max="100"
        class="w-full"
      />
      <span class="text-xs text-gray-500 ml-2">连接池最大连接数</span>
    </el-form-item>
  </el-form>
</template>

<script setup>
import { ref, watch, onMounted, computed } from 'vue'
import { getDefaultConfig } from '@/config/connectionTypes'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  },
  mode: {
    type: String,
    default: 'create', // 'create' | 'edit'
    validator: (value) => ['create', 'edit'].includes(value)
  }
})

const emit = defineEmits(['update:modelValue', 'validate'])

const formRef = ref(null)
const formData = ref({
  name: '',
  host: 'localhost',
  port: 5432,
  database: '',
  username: 'postgres',
  password: '',
  schema: 'public',
  sslMode: 'disable',
  sslCa: '',
  sslCert: '',
  sslKey: '',
  connectionTimeout: 3000,
  queryTimeout: 30000,
  maxConnections: 10,
})

// 防止循环更新的标志
const isUpdatingFromParent = ref(false)

// 是否需要证书配置
const needsCertificate = computed(() => {
  return formData.value.sslMode === 'verify-ca'
})

const rules = {
  name: [
    { required: true, message: '请输入连接名称', trigger: 'blur' },
    { min: 2, max: 100, message: '连接名称长度在 2 到 100 个字符', trigger: 'blur' }
  ],
  host: [
    { required: true, message: '请输入主机地址', trigger: 'blur' }
  ],
  port: [
    { required: true, message: '请输入端口号', trigger: 'blur' }
  ],
  database: [
    { required: true, message: '请输入数据库名', trigger: 'blur' }
  ],
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ]
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
  { deep: true }
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
  { deep: true }
)

/**
 * SSL 模式变化处理
 */
const handleSslModeChange = (mode) => {
  // 如果切换到不需要证书的模式，清空证书字段
  if (mode !== 'verify-ca') {
    formData.value.sslCa = ''
    formData.value.sslCert = ''
    formData.value.sslKey = ''
  }
}

/**
 * 验证表单
 */
const validate = async () => {
  if (!formRef.value) return false
  
  try {
    await formRef.value.validate()
    emit('validate', true, formData.value)
    return true
  } catch (error) {
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
  formData
})
</script>
