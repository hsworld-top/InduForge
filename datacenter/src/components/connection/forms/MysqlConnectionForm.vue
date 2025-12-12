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
      <el-input v-model="formData.username" placeholder="root" />
    </el-form-item>

    <el-form-item label="密码" prop="password">
      <el-input v-model="formData.password" type="password" placeholder="请输入密码" show-password />
    </el-form-item>

    <el-form-item label="字符集">
      <el-select v-model="formData.charset" placeholder="选择字符集" class="w-full">
        <el-option label="utf8mb4" value="utf8mb4" />
        <el-option label="utf8" value="utf8" />
        <el-option label="latin1" value="latin1" />
        <el-option label="gbk" value="gbk" />
      </el-select>
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
  </el-form>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
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
  port: 3306,
  database: '',
  username: 'root',
  password: '',
  charset: 'utf8mb4',
  queryTimeout: 30000,
})

// 防止循环更新的标志
const isUpdatingFromParent = ref(false)

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
    const defaultConfig = getDefaultConfig('mysql')
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
