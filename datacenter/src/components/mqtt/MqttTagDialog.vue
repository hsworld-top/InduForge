<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="800px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="120px"
      :disabled="mode === 'view'"
    >
      <el-form-item label="变量名称" prop="name">
        <el-input
          v-model="formData.name"
          placeholder="例如: 温度传感器"
          clearable
        />
      </el-form-item>

      <el-form-item label="变量标识符" prop="code">
        <el-input
          v-model="formData.code"
          placeholder="例如: temperature_sensor_1"
          clearable
        >
          <template #append>
            <el-button @click="generateCode">自动生成</el-button>
          </template>
        </el-input>
        <span class="text-xs text-gray-500">
          用于在系统中引用该变量，建议使用英文、数字和下划线
        </span>
      </el-form-item>

      <el-form-item label="描述">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="2"
          placeholder="变量描述"
        />
      </el-form-item>

      <el-form-item label="所属分组">
        <el-select
          v-model="formData.groupId"
          placeholder="选择所属分组（可选）"
          clearable
        >
          <el-option
            v-for="group in groups"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          >
            <div class="flex items-center gap-2">
              <el-icon :color="group.color">
                <Folder />
              </el-icon>
              <span>{{ group.name }}</span>
            </div>
          </el-option>
        </el-select>
        <span class="text-xs text-gray-500"> 不选择则归入"未分组" </span>
      </el-form-item>

      <el-form-item label="数据类型" prop="dataType">
        <el-select v-model="formData.dataType" placeholder="选择数据类型">
          <el-option label="字符串" value="string" />
          <el-option label="数值" value="number" />
          <el-option label="布尔" value="boolean" />
          <el-option label="对象" value="object" />
          <el-option label="数组" value="array" />
        </el-select>
      </el-form-item>

      <el-form-item label="解析类型" prop="parseType">
        <el-select
          v-model="formData.parseType"
          placeholder="选择解析类型"
          @change="handleParseTypeChange"
        >
          <el-option label="JSONPath" value="jsonpath" />
          <el-option label="正则表达式" value="regex" />
          <el-option label="JavaScript脚本" value="script" />
          <el-option label="固定值" value="fixed" />
        </el-select>
      </el-form-item>

      <el-form-item label="解析规则" prop="parseRule">
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
          placeholder="例如: function(message) { return message.data.value; }"
        />
        <span class="text-xs text-gray-500 mt-1 block">
          {{ getParseRuleHint() }}
        </span>
      </el-form-item>

      <el-form-item label="默认值">
        <el-input
          v-model="formData.defaultValue"
          placeholder="解析失败时使用的默认值"
          clearable
        />
      </el-form-item>

      <el-form-item label="单位">
        <el-input
          v-model="formData.unit"
          placeholder="例如: °C, %, m/s"
          clearable
          style="width: 200px"
        />
      </el-form-item>

      <el-form-item label="值转换函数">
        <el-input
          v-model="formData.transform"
          type="textarea"
          :rows="3"
          placeholder="例如: (v) => v * 10"
        />
        <span class="text-xs text-gray-500">
          可选，用于对解析后的值进行进一步处理
        </span>
      </el-form-item>

      <el-form-item label="验证规则">
        <el-input
          v-model="validationStr"
          type="textarea"
          :rows="3"
          placeholder='例如: {"min": 0, "max": 100}'
        />
        <span class="text-xs text-gray-500">
          JSON格式，支持 min/max (数值), minLength/maxLength/pattern (字符串),
          enum (枚举值)
        </span>
      </el-form-item>

      <el-form-item label="显示顺序">
        <el-input-number v-model="formData.order" :min="0" :max="9999" />
      </el-form-item>

      <el-form-item label="启用状态">
        <el-switch
          v-model="formData.isEnabled"
          active-text="启用"
          inactive-text="禁用"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">
          {{ mode === "view" ? "关闭" : "取消" }}
        </el-button>
        <el-button
          v-if="mode !== 'view'"
          type="primary"
          @click="handleSubmit"
          :loading="submitting"
        >
          确定
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, onMounted } from "vue";
import { ElMessage } from "element-plus";
import { createMqttTag, updateMqttTag } from "@/api/data.api";
import { Folder } from "@element-plus/icons-vue";

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
  groups: {
    type: Array,
    default: () => [],
  },
  mode: {
    type: String,
    default: "create", // create | edit | view
  },
});

const emit = defineEmits(["close", "success"]);

const formRef = ref(null);
const submitting = ref(false);

const formData = ref({
  name: "",
  code: "",
  description: "",
  groupId: null,
  dataType: "string",
  parseType: "jsonpath",
  parseRule: "",
  defaultValue: "",
  unit: "",
  transform: "",
  validation: null,
  order: 0,
  isEnabled: true,
});

const validationStr = ref("");

const rules = {
  name: [
    { required: true, message: "请输入变量名称", trigger: "blur" },
    { min: 1, max: 100, message: "长度在 1 到 100 个字符", trigger: "blur" },
  ],
  code: [
    { required: true, message: "请输入变量标识符", trigger: "blur" },
    {
      pattern: /^[a-zA-Z][a-zA-Z0-9_]*$/,
      message: "只能包含字母、数字和下划线，且以字母开头",
      trigger: "blur",
    },
  ],
  dataType: [{ required: true, message: "请选择数据类型", trigger: "change" }],
  parseType: [{ required: true, message: "请选择解析类型", trigger: "change" }],
  parseRule: [{ required: true, message: "请输入解析规则", trigger: "blur" }],
};

const dialogTitle = computed(() => {
  const titles = {
    create: "新建变量",
    edit: "编辑变量",
    view: "查看变量",
  };
  return titles[props.mode] || "变量";
});

// 初始化表单
const initForm = () => {
  if (props.tag && props.mode !== "create") {
    formData.value = {
      ...props.tag,
      validation: props.tag.validation || null,
    };

    if (props.tag.validation) {
      validationStr.value = JSON.stringify(props.tag.validation, null, 2);
    }
  } else {
    formData.value = {
      name: "",
      code: "",
      description: "",
      groupId: null,
      dataType: "string",
      parseType: "jsonpath",
      parseRule: "",
      defaultValue: "",
      unit: "",
      transform: "",
      validation: null,
      order: 0,
      isEnabled: true,
    };
    validationStr.value = "";
  }
};

// 自动生成code
const generateCode = () => {
  if (formData.value.name) {
    // 简单的拼音转换（实际项目中可以使用更完善的库）
    const code = formData.value.name
      .toLowerCase()
      .replace(/\s+/g, "_")
      .replace(/[^a-z0-9_]/g, "");
    formData.value.code = code || `tag_${Date.now()}`;
  } else {
    formData.value.code = `tag_${Date.now()}`;
  }
};

// 解析类型变化
const handleParseTypeChange = () => {
  formData.value.parseRule = "";
};

// 获取解析规则占位符
const getParseRulePlaceholder = () => {
  const placeholders = {
    jsonpath: "例如: $.data.temperature",
    regex: "例如: temperature:\\s*(\\d+\\.?\\d*)",
    fixed: "例如: 25.5",
  };
  return placeholders[formData.value.parseType] || "请输入解析规则";
};

// 获取解析规则提示
const getParseRuleHint = () => {
  const hints = {
    jsonpath: "使用JSONPath表达式从JSON消息中提取值",
    regex: "使用正则表达式从文本消息中提取值（第一个捕获组）",
    script: "编写JavaScript函数，接收message参数，返回解析后的值",
    fixed: "直接使用此值作为变量的固定值",
  };
  return hints[formData.value.parseType] || "";
};

// 提交
const handleSubmit = async () => {
  try {
    await formRef.value.validate();

    // 解析验证规则JSON
    if (validationStr.value) {
      try {
        formData.value.validation = JSON.parse(validationStr.value);
      } catch {
        ElMessage.error("验证规则JSON格式不正确");
        return;
      }
    } else {
      formData.value.validation = null;
    }

    submitting.value = true;

    if (props.mode === "create") {
      await createMqttTag(
        props.projectId,
        props.subscriptionId,
        formData.value,
      );
      ElMessage.success("创建成功");
    } else {
      await updateMqttTag(props.tag.id, formData.value);
      ElMessage.success("更新成功");
    }

    emit("success");
  } catch (error) {
    if (error !== false) {
      ElMessage.error("操作失败: " + error.message);
    }
  } finally {
    submitting.value = false;
  }
};

// 关闭
const handleClose = () => {
  emit("close");
};

// 监听visible变化，重新初始化表单
watch(
  () => props.visible,
  (val) => {
    if (val) {
      initForm();
    }
  },
);

onMounted(() => {
  initForm();
});
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
