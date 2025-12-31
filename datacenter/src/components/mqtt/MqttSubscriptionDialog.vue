<template>
  <el-dialog
    v-model="visible"
    :title="mode === 'create' ? '新建订阅' : '编辑订阅'"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item label="订阅名称" prop="name">
        <el-input
          v-model="formData.name"
          placeholder="请输入订阅名称"
          maxlength="100"
          show-word-limit
        />
      </el-form-item>

      <el-form-item label="MQTT主题" prop="topic">
        <el-input
          v-model="formData.topic"
          placeholder="例如: device/+/data 或 sensor/#"
          maxlength="500"
        >
          <template #append>
            <el-tooltip
              content="支持通配符: + 匹配单层, # 匹配多层"
              placement="top"
            >
              <el-icon><IconTablerQuestionMark /></el-icon>
            </el-tooltip>
          </template>
        </el-input>
        <div class="text-xs text-gray-400 mt-1">
          支持通配符：+ (单层) 和 # (多层), 例如: device/+/data 或 sensor/#
        </div>
      </el-form-item>

      <el-form-item label="QoS等级" prop="qos">
        <el-radio-group v-model="formData.qos">
          <el-radio :label="0">QoS 0 (最多一次)</el-radio>
          <el-radio :label="1">QoS 1 (至少一次)</el-radio>
          <el-radio :label="2">QoS 2 (恰好一次)</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="是否启用" prop="isEnabled">
        <el-switch
          v-model="formData.isEnabled"
          active-text="启用"
          inactive-text="禁用"
        />
      </el-form-item>

      <el-form-item label="备注" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          placeholder="请输入备注信息（可选）"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="flex justify-end space-x-2">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ mode === "create" ? "创建" : "保存" }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch, computed } from "vue";
import { ElMessage } from "element-plus";
import IconTablerQuestionMark from "~icons/tabler/question-mark";
import dataAPI from "@/api/data.api";

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  connectionId: {
    type: String,
    required: true,
  },
  subscription: {
    type: Object,
    default: null,
  },
  mode: {
    type: String,
    default: "create",
    validator: (value) => ["create", "edit"].includes(value),
  },
});

const emit = defineEmits(["update:modelValue", "success"]);

const formRef = ref(null);
const submitting = ref(false);

// 表单数据
const formData = ref({
  name: "",
  topic: "",
  qos: 0,
  isEnabled: true,
  description: "",
});

// 验证规则
const rules = {
  name: [
    { required: true, message: "请输入订阅名称", trigger: "blur" },
    { min: 2, max: 100, message: "长度在 2 到 100 个字符", trigger: "blur" },
  ],
  topic: [
    { required: true, message: "请输入MQTT主题", trigger: "blur" },
    {
      pattern: /^[a-zA-Z0-9_\-\/\+\#]+$/,
      message: "主题格式不正确，只能包含字母、数字、_、-、/、+、#",
      trigger: "blur",
    },
  ],
  qos: [{ required: true, message: "请选择QoS等级", trigger: "change" }],
};

// 计算属性：对话框可见性
const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});

// 获取项目ID (从URL或其他地方)
const getProjectId = () => {
  const route = window.location;
  const urlParams = new URLSearchParams(route.search);
  return urlParams.get("pid") || urlParams.get("id");
};

/**
 * 初始化表单数据
 */
const initFormData = () => {
  if (props.mode === "edit" && props.subscription) {
    formData.value = {
      name: props.subscription.name || "",
      topic: props.subscription.topic || "",
      qos: props.subscription.qos ?? 0,
      isEnabled: props.subscription.isEnabled ?? true,
      description: props.subscription.description || "",
    };
  } else {
    formData.value = {
      name: "",
      topic: "",
      qos: 0,
      isEnabled: true,
      description: "",
    };
  }
};

/**
 * 提交表单
 */
const handleSubmit = async () => {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitting.value = true;

    const projectId = getProjectId();
    if (!projectId) {
      ElMessage.error("无法获取项目ID");
      return;
    }

    const data = {
      name: formData.value.name,
      topic: formData.value.topic,
      qos: formData.value.qos,
      isEnabled: formData.value.isEnabled,
      description: formData.value.description || null,
    };

    let response;
    if (props.mode === "create") {
      response = await dataAPI.createMqttSubscription(
        projectId,
        props.connectionId,
        data,
      );
    } else {
      response = await dataAPI.updateMqttSubscription(
        projectId,
        props.subscription.id,
        data,
      );
    }

    if (response.success) {
      ElMessage.success(
        props.mode === "create" ? "订阅创建成功" : "订阅更新成功",
      );
      emit("success", response.data);
      handleClose();
    }
  } catch (error) {
    if (error.errors) {
      // 表单验证错误
      return;
    }
    ElMessage.error(
      (props.mode === "create" ? "创建失败：" : "更新失败：") +
        (error.response?.data?.message || error.message),
    );
  } finally {
    submitting.value = false;
  }
};

/**
 * 关闭对话框
 */
const handleClose = () => {
  if (formRef.value) {
    formRef.value.resetFields();
  }
  visible.value = false;
};

// 监听对话框打开
watch(visible, (newVal) => {
  if (newVal) {
    initFormData();
  }
});

// 监听订阅数据变化
watch(
  () => props.subscription,
  () => {
    if (props.mode === "edit" && props.subscription) {
      initFormData();
    }
  },
  { deep: true },
);
</script>

<style scoped>
:deep(.el-form-item__label) {
  font-weight: 500;
}
</style>
