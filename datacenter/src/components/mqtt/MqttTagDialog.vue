<template>
  <DcDialog
    ref="dialogRef"
    :model-value="visible"
    :title="dialogTitle"
    width="800px"
    :dirty="isDirty"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="120px"
      :disabled="mode === 'view'"
    >
      <el-form-item :label="ui('变量名称', 'Variable Name')" prop="name">
        <el-input v-model="formData.name" :placeholder="ui('例如：温度传感器', 'For example: Temperature Sensor')" clearable />
      </el-form-item>

      <el-form-item :label="ui('描述', 'Description')">
        <el-input v-model="formData.description" type="textarea" :rows="2" :placeholder="ui('变量描述', 'Variable description')" />
      </el-form-item>

      <el-form-item :label="ui('数据类型', 'Data Type')" prop="dataType">
        <el-select v-model="formData.dataType" :placeholder="ui('选择数据类型', 'Select a data type')">
          <el-option :label="ui('字符串', 'String')" value="string" />
          <el-option :label="ui('数值（float64）', 'Number (float64)')" value="float64" />
          <el-option :label="ui('布尔', 'Boolean')" value="bool" />
          <el-option :label="ui('对象', 'Object')" value="object" />
          <el-option :label="ui('数组', 'Array')" value="array" />
        </el-select>
      </el-form-item>

      <el-form-item :label="ui('解析类型', 'Parse Type')" prop="parseType">
        <el-select
          v-model="formData.parseType"
          :placeholder="ui('选择解析类型', 'Select a parse type')"
          @change="handleParseTypeChange"
        >
          <el-option label="JSONPath" value="jsonpath" />
          <el-option :label="ui('正则表达式', 'Regular Expression')" value="regex" />
          <el-option :label="ui('JavaScript 脚本', 'JavaScript Script')" value="script" />
          <el-option :label="ui('固定值', 'Fixed Value')" value="fixed" />
        </el-select>
      </el-form-item>

      <el-form-item :label="ui('解析规则', 'Parse Rule')" prop="parseRule">
        <el-input
          v-if="formData.parseType !== 'script'"
          v-model="formData.parseRule"
          :placeholder="getParseRulePlaceholder()"
          clearable
        />
        <el-input
          v-else
          v-model="formData.parseRule"
          type="textarea"
          :rows="4"
          :placeholder="ui('例如：function(message) { return message.data.value; }', 'For example: function(message) { return message.data.value; }')"
        />
        <span class="text-xs text-gray-500 mt-1 block">
          {{ getParseRuleHint() }}
        </span>
      </el-form-item>

      <el-form-item :label="ui('默认值', 'Default Value')">
        <el-input v-model="formData.defaultValue" :placeholder="ui('解析失败时使用的默认值', 'Value used when parsing fails')" clearable />
      </el-form-item>

      <el-form-item :label="ui('单位', 'Unit')">
        <el-input
          v-model="formData.unit"
          :placeholder="ui('例如：°C、%、m/s', 'For example: °C, %, m/s')"
          clearable
          style="width: 200px"
        />
      </el-form-item>

      <el-form-item :label="ui('值转换函数', 'Value Transform')">
        <el-input
          v-model="formData.transform"
          type="textarea"
          :rows="3"
          :placeholder="ui('例如：(v) => v * 10', 'For example: (v) => v * 10')"
        />
        <span class="text-xs text-gray-500"> {{ ui('可选，用于进一步处理解析后的值', 'Optional; further transforms the parsed value') }} </span>
      </el-form-item>

      <el-form-item :label="ui('验证规则', 'Validation Rules')">
        <el-input
          v-model="validationStr"
          type="textarea"
          :rows="3"
          :placeholder="validationPlaceholder"
        />
        <span class="text-xs text-gray-500">
          {{ ui('JSON 格式，支持 min/max（数值）、minLength/maxLength/pattern（字符串）和 enum（枚举值）', 'JSON format; supports min/max (number), minLength/maxLength/pattern (string), and enum') }}
        </span>
      </el-form-item>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="requestClose">
          {{ mode === 'view' ? ui('关闭', 'Close') : ui('取消', 'Cancel') }}
        </el-button>
        <el-button
          v-if="mode !== 'view'"
          type="primary"
          @click="handleSubmit"
          :loading="submitting"
        >
          {{ ui('确定', 'Confirm') }}
        </el-button>
      </span>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import DcDialog from '@/components/shared/DcDialog.vue'
import { createMqttTag, updateMqttTag } from '@/api/data.api'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)
const validationPlaceholder = computed(() =>
  ui('例如：{"min": 0, "max": 100}', 'For example: {"min": 0, "max": 100}'),
)

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  tag: {
    type: Object,
    default: null,
  },
  projectId: {
    type: String,
    required: true,
  },
  subscriptionId: {
    type: String,
    required: true,
  },
  mode: {
    type: String,
    default: 'create', // create | edit | view
  },
})

const emit = defineEmits(['close', 'success'])

const formRef = ref(null)
const submitting = ref(false)
const initialFormSnapshot = ref('')
const formInitialized = ref(false)
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)

const formData = ref({
  name: '',
  code: '',
  description: '',
  dataType: 'string',
  parseType: 'jsonpath',
  parseRule: '',
  defaultValue: '',
  unit: '',
  transform: '',
  validation: null,
  order: 0,
})

const validationStr = ref('')

const rules = computed(() => ({
  name: [
    { required: true, message: ui('请输入变量名称', 'Enter a variable name'), trigger: 'blur' },
    { min: 1, max: 100, message: ui('长度为 1 到 100 个字符', 'Length must be between 1 and 100 characters'), trigger: 'blur' },
  ],
  dataType: [{ required: true, message: ui('请选择数据类型', 'Select a data type'), trigger: 'change' }],
  parseType: [{ required: true, message: ui('请选择解析类型', 'Select a parse type'), trigger: 'change' }],
  parseRule: [{ required: true, message: ui('请输入解析规则', 'Enter a parse rule'), trigger: 'blur' }],
}))

const dialogTitle = computed(() => {
  const titles = {
    create: ui('新建变量', 'New Variable'),
    edit: ui('编辑变量', 'Edit Variable'),
    view: ui('查看变量', 'View Variable'),
  }
  return titles[props.mode] || ui('变量', 'Variable')
})

const formSnapshot = computed(() =>
  JSON.stringify({
    formData: formData.value,
    validationStr: validationStr.value,
  }),
)
const isDirty = computed(
  () =>
    props.visible &&
    formInitialized.value &&
    props.mode !== 'view' &&
    formSnapshot.value !== initialFormSnapshot.value,
)

/**
 * 初始化表单数据
 * @returns {void}
 */
const initForm = () => {
  // 弹窗打开首帧先禁止 dirty 判断，避免默认表单和空快照短暂不一致触发关闭确认。
  formInitialized.value = false
  if (props.tag && props.mode !== 'create') {
    formData.value = {
      ...props.tag,
      validation: props.tag.validation || null,
    }

    if (props.tag.validation) {
      validationStr.value = JSON.stringify(props.tag.validation, null, 2)
    }
  } else {
    formData.value = {
      name: '',
      code: '',
      description: '',
      dataType: 'string',
      parseType: 'jsonpath',
      parseRule: '',
      defaultValue: '',
      unit: '',
      transform: '',
      validation: null,
      order: 0,
    }
    validationStr.value = ''
  }
  initialFormSnapshot.value = formSnapshot.value
  formInitialized.value = true
}

/**
 * 将名称转换为标识符
 * @param {string} name - 变量名称
 * @returns {string} 标识符
 */
const normalizeCode = (name) => {
  const normalized = String(name || '')
    .toLowerCase()
    .replace(/\s+/g, '_')
    .replace(/[^a-z0-9_]/g, '')
    .replace(/^_+|_+$/g, '')
    .replace(/_+/g, '_')
  return normalized || `tag_${Date.now()}`
}

/**
 * 处理解析类型变化
 * @returns {void}
 */
const handleParseTypeChange = () => {
  formData.value.parseRule = ''
}

/**
 * 获取解析规则占位符
 * @returns {string} 占位符文本
 */
const getParseRulePlaceholder = () => {
  const placeholders = {
    jsonpath: ui('例如：$.data.temperature', 'For example: $.data.temperature'),
    regex: ui('例如：temperature:\\s*(\\d+\\.?\\d*)', 'For example: temperature:\\s*(\\d+\\.?\\d*)'),
    fixed: ui('例如：25.5', 'For example: 25.5'),
  }
  return placeholders[formData.value.parseType] || ui('请输入解析规则', 'Enter a parse rule')
}

/**
 * 获取解析规则提示
 * @returns {string} 提示文本
 */
const getParseRuleHint = () => {
  const hints = {
    jsonpath: ui('使用 JSONPath 表达式从 JSON 消息中提取值', 'Extract a value from a JSON message with JSONPath'),
    regex: ui('使用正则表达式的第一个捕获组从文本消息中提取值', 'Extract a value from text using the first regular-expression capture group'),
    script: ui('编写接收 message 参数并返回解析值的 JavaScript 函数', 'Write a JavaScript function that receives message and returns the parsed value'),
    fixed: ui('直接将此内容作为变量的固定值', 'Use this content directly as the fixed variable value'),
  }
  return hints[formData.value.parseType] || ''
}

// 提交
/**
 * 提交表单
 * @returns {Promise<void>}
 * @throws 表单校验失败或接口调用异常
 */
const handleSubmit = async () => {
  try {
    if (!formData.value.code) {
      formData.value.code = normalizeCode(formData.value.name)
    }
    await formRef.value.validate()

    // 解析验证规则JSON
    if (validationStr.value) {
      try {
        formData.value.validation = JSON.parse(validationStr.value)
      } catch {
        ElMessage.error(ui('验证规则不是合法 JSON', 'Validation rules are not valid JSON'))
        return
      }
    } else {
      formData.value.validation = null
    }

    submitting.value = true

    if (props.mode === 'create') {
      await createMqttTag(props.projectId, props.subscriptionId, formData.value)
      ElMessage.success(ui('创建成功', 'Created successfully'))
    } else {
      await updateMqttTag(props.projectId, props.tag.id, formData.value)
      ElMessage.success(ui('更新成功', 'Updated successfully'))
    }

    initialFormSnapshot.value = formSnapshot.value
    emit('success')
  } catch (error) {
    if (error !== false) {
      ElMessage.error(ui('操作失败：', 'Operation failed: ') + (error instanceof Error ? error.message : String(error)))
    }
  } finally {
    submitting.value = false
  }
}

/**
 * 关闭对话框
 * @returns {void}
 */
const handleClose = () => {
  emit('close')
}

const requestClose = () => {
  void dialogRef.value?.requestClose()
}

// 监听visible变化，重新初始化表单
watch(
  () => props.visible,
  (val) => {
    if (val) {
      initForm()
    }
  },
)

onMounted(() => {
  initForm()
})
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.text-gray-500 {
  color: var(--dc-text-muted);
}
</style>
