<template>
  <div class="data-binding-panel p-3">
    <div v-if="!selectedComponent" class="text-center py-8 text-gray-500 dark:text-gray-400">
      <div class="text-sm">请选择组件</div>
    </div>

    <div v-else>
      <div class="mb-4">
        <div class="text-sm font-medium text-gray-900 dark:text-white mb-2">
          {{ selectedComponent.name || selectedComponent.type }}
        </div>
        <div class="text-xs text-gray-500 dark:text-gray-400">
          ID: {{ selectedComponent.id }}
        </div>
      </div>

      <!-- 绑定列表 -->
      <div class="space-y-3">
        <div
          v-for="(binding, path) in bindings"
          :key="path"
          class="p-3 border border-gray-200 dark:border-gray-700 rounded-lg"
        >
          <div class="flex items-start justify-between mb-2">
            <div class="flex-1">
              <div class="text-sm font-medium text-gray-900 dark:text-white">
                {{ path }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400 mt-1 font-mono">
                {{ binding }}
              </div>
            </div>
            <el-button
              size="small"
              text
              type="danger"
              @click="removeBinding(path)"
            >
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          
          <!-- 实时预览 -->
          <div v-if="previewValues[path] !== undefined" class="mt-2 p-2 bg-gray-50 dark:bg-gray-900 rounded text-xs">
            <div class="text-gray-500 dark:text-gray-400 mb-1">当前值:</div>
            <div class="text-gray-900 dark:text-white font-mono">
              {{ formatValue(previewValues[path]) }}
            </div>
          </div>
        </div>

        <div v-if="Object.keys(bindings).length === 0" class="text-center py-4 text-gray-500 dark:text-gray-400 text-sm">
          暂无数据绑定
        </div>
      </div>

      <!-- 添加绑定按钮 -->
      <el-button
        type="primary"
        size="small"
        class="w-full mt-3"
        @click="showAddDialog = true"
      >
        <el-icon class="mr-1"><Plus /></el-icon>
        添加绑定
      </el-button>
    </div>

    <!-- 添加绑定对话框 -->
    <el-dialog
      v-model="showAddDialog"
      title="添加数据绑定"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="属性路径" prop="path">
          <el-select
            v-model="form.path"
            placeholder="选择或输入属性路径"
            class="w-full"
            filterable
            allow-create
          >
            <el-option-group label="属性 (props)">
              <el-option
                v-for="prop in availableProps"
                :key="'props.' + prop"
                :label="'props.' + prop"
                :value="'props.' + prop"
              />
            </el-option-group>
            <el-option-group label="样式 (style)">
              <el-option
                v-for="style in commonStyles"
                :key="'style.' + style"
                :label="'style.' + style"
                :value="'style.' + style"
              />
            </el-option-group>
          </el-select>
        </el-form-item>

        <el-form-item label="表达式" prop="expression">
          <el-input
            v-model="form.expression"
            type="textarea"
            :rows="4"
            placeholder="{{ data.ds_example.value }}"
          />
          <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            使用 {{ "{{ expression }}" }} 语法
          </div>
        </el-form-item>

        <!-- 快速选择 -->
        <el-form-item label="快速选择">
          <el-select
            v-model="quickSelect"
            placeholder="选择数据源"
            class="w-full"
            @change="onQuickSelect"
          >
            <el-option-group label="数据源">
              <el-option
                v-for="ds in dataSources"
                :key="ds.id"
                :label="ds.id"
                :value="'data.' + ds.id"
              />
            </el-option-group>
            <el-option-group label="变量">
              <el-option
                v-for="(value, key) in variables"
                :key="key"
                :label="key"
                :value="'vars.' + key"
              />
            </el-option-group>
          </el-select>
        </el-form-item>

        <!-- 实时预览 -->
        <el-form-item label="预览">
          <div class="p-2 bg-gray-50 dark:bg-gray-900 rounded text-sm">
            <div v-if="previewError" class="text-red-500">
              错误: {{ previewError }}
            </div>
            <div v-else class="text-gray-900 dark:text-white font-mono">
              {{ formatValue(previewValue) }}
            </div>
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="addBinding">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'
import { useDesignStore } from '@/store/design'
import expressionEngine from '@/engine/binding/ExpressionEngine'

const store = useDesignStore()

const showAddDialog = ref(false)
const formRef = ref(null)
const quickSelect = ref('')
const previewValue = ref(null)
const previewError = ref(null)
const previewValues = ref({})

const form = ref({
  path: '',
  expression: ''
})

const rules = {
  path: [
    { required: true, message: '请输入属性路径', trigger: 'blur' }
  ],
  expression: [
    { required: true, message: '请输入表达式', trigger: 'blur' }
  ]
}

const selectedComponent = computed(() => store.selectedComponent)

const bindings = computed(() => {
  if (!selectedComponent.value?.bindings) {
    return {}
  }
  return selectedComponent.value.bindings
})

const dataSources = computed(() => {
  return Array.from(store.dataSourceManager?.dataSources.values() || [])
})

const variables = computed(() => {
  return store.currentPage?.variables || {}
})

const availableProps = computed(() => {
  if (!selectedComponent.value?.props) {
    return []
  }
  return Object.keys(selectedComponent.value.props)
})

const commonStyles = [
  'color',
  'backgroundColor',
  'opacity',
  'width',
  'height',
  'left',
  'top',
  'transform',
  'display',
  'visibility'
]

const formatValue = (value) => {
  if (value === null || value === undefined) {
    return 'null'
  }
  if (typeof value === 'object') {
    return JSON.stringify(value, null, 2)
  }
  return String(value)
}

const onQuickSelect = (value) => {
  if (value) {
    form.value.expression = `{{ ${value} }}`
    quickSelect.value = ''
  }
}

const addBinding = async () => {
  try {
    await formRef.value.validate()
    
    if (!selectedComponent.value) {
      return
    }
    
    // 更新组件绑定
    const updates = {
      bindings: {
        ...selectedComponent.value.bindings,
        [form.value.path]: form.value.expression
      }
    }
    
    store.updateComponent(selectedComponent.value.id, updates)
    
    // 应用绑定
    store.bindComponent(selectedComponent.value.id, updates.bindings)
    
    ElMessage.success('绑定添加成功')
    showAddDialog.value = false
    resetForm()
  } catch (error) {
    console.error('Add binding error:', error)
  }
}

const removeBinding = (path) => {
  if (!selectedComponent.value) {
    return
  }
  
  const newBindings = { ...selectedComponent.value.bindings }
  delete newBindings[path]
  
  const updates = {
    bindings: newBindings
  }
  
  store.updateComponent(selectedComponent.value.id, updates)
  store.bindComponent(selectedComponent.value.id, newBindings)
  
  ElMessage.success('绑定已删除')
}

const resetForm = () => {
  form.value = {
    path: '',
    expression: ''
  }
  previewValue.value = null
  previewError.value = null
}

// 监听表达式变化，实时预览
watch(() => form.value.expression, (expression) => {
  if (!expression) {
    previewValue.value = null
    previewError.value = null
    return
  }
  
  try {
    const context = buildContext()
    expressionEngine.setContext(context)
    previewValue.value = expressionEngine.evaluate(expression)
    previewError.value = null
  } catch (error) {
    previewValue.value = null
    previewError.value = error.message
  }
}, { immediate: true })

// 监听绑定变化，更新预览值
watch(() => bindings.value, (newBindings) => {
  Object.entries(newBindings).forEach(([path, expression]) => {
    try {
      const context = buildContext()
      expressionEngine.setContext(context)
      previewValues.value[path] = expressionEngine.evaluate(expression)
    } catch (error) {
      previewValues.value[path] = `Error: ${error.message}`
    }
  })
}, { deep: true, immediate: true })

// 监听数据源变化，更新预览值
watch(() => store.dataSources, () => {
  Object.entries(bindings.value).forEach(([path, expression]) => {
    try {
      const context = buildContext()
      expressionEngine.setContext(context)
      previewValues.value[path] = expressionEngine.evaluate(expression)
    } catch (error) {
      previewValues.value[path] = `Error: ${error.message}`
    }
  })
}, { deep: true })

const buildContext = () => {
  return {
    vars: store.currentPage?.variables || {},
    data: store.dataSources || {},
    props: selectedComponent.value?.props || {},
    $user: store.user || {},
    $route: {},
    $env: {
      API_BASE: import.meta.env.VITE_API_BASE || '',
      MODE: import.meta.env.MODE
    },
    $global: {}
  }
}

watch(() => showAddDialog.value, (val) => {
  if (!val) {
    resetForm()
  }
})
</script>

<style scoped>
.data-binding-panel {
  min-height: 200px;
}
</style>
