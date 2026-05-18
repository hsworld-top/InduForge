<template>
  <DcDialog
    v-model="visible"
    :title="
      mode === 'create' ? t('subscription.create') : t('subscription.edit')
    "
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
      <el-form-item :label="t('subscription.nameLabel')" prop="name">
        <el-input
          v-model="formData.name"
          :placeholder="t('subscription.namePlaceholder')"
          maxlength="100"
          show-word-limit
        />
      </el-form-item>

      <el-form-item :label="t('subscription.topicLabel')" prop="topic">
        <el-input
          v-model="formData.topic"
          :placeholder="t('subscription.topicPlaceholder')"
          maxlength="500"
        >
          <template #append>
            <el-tooltip
              :content="t('subscription.wildcardTooltip')"
              placement="top"
            >
              <el-icon><IconTablerQuestionMark /></el-icon>
            </el-tooltip>
          </template>
        </el-input>
        <div class="text-xs text-gray-400 mt-1">
          {{ t("subscription.wildcardHint") }}
        </div>
      </el-form-item>

      <el-form-item :label="t('subscription.qosLevel')" prop="qos">
        <el-radio-group v-model="formData.qos">
          <el-radio :label="0">QoS 0 (最多一次)</el-radio>
          <el-radio :label="1">QoS 1 (至少一次)</el-radio>
          <el-radio :label="2">QoS 2 (恰好一次)</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item :label="t('subscription.enabled')" prop="isEnabled">
        <el-switch
          v-model="formData.isEnabled"
          :active-text="t('common.enabled')"
          :inactive-text="t('common.disabled')"
        />
      </el-form-item>

      <el-form-item :label="t('subscription.remark')" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          :placeholder="t('subscription.descriptionPlaceholder')"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="flex justify-end space-x-2">
        <el-button @click="handleClose">{{ t("actions.cancel") }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ mode === "create" ? t("actions.create") : t("actions.save") }}
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from "vue";
import { ElMessage } from "element-plus";
import DcDialog from "@/components/shared/DcDialog.vue";
import IconTablerQuestionMark from "~icons/tabler/question-mark";
import dataAPI from "@/api/data.api";
import { Storage } from "@/utils/storage";
import { t } from "@/i18n/runtime";
import { getApiErrorMessage } from "@/utils/request";

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
    {
      required: true,
      message: t("subscription.namePlaceholder"),
      trigger: "blur",
    },
    {
      min: 2,
      max: 100,
      message: t("subscription.nameLengthError"),
      trigger: "blur",
    },
  ],
  topic: [
    {
      required: true,
      message: t("subscription.topicPlaceholder"),
      trigger: "blur",
    },
    {
      pattern: /^[a-zA-Z0-9_/#+-]+$/,
      message: t("subscription.topicPatternError"),
      trigger: "blur",
    },
  ],
  qos: [
    { required: true, message: t("subscription.qosLevel"), trigger: "change" },
  ],
};

// 计算属性：对话框可见性
const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});

// 获取项目 ID。
// 正式入口下优先使用宿主 bootstrap 已写入的本地上下文，仅在独立调试态下回退到旧 URL 参数。
const getProjectId = () => {
  const projectIdFromStorage = Storage.getProjectId();
  if (projectIdFromStorage) {
    return projectIdFromStorage;
  }

  const urlParams = new window.URLSearchParams(window.location.search);
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
      ElMessage.error(t("subscription.projectMissing"));
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

    ElMessage.success(
      props.mode === "create"
        ? t("subscription.createSuccess")
        : t("subscription.updateSuccess"),
    );
    emit("success", response.data);
    handleClose();
  } catch (error) {
    if (error.errors) {
      // 表单验证错误
      return;
    }
    ElMessage.error(
      props.mode === "create"
        ? t("subscription.createFailed", {
            message: getApiErrorMessage(error, "创建订阅失败"),
          })
        : t("subscription.updateFailed", {
            message: getApiErrorMessage(error, "更新订阅失败"),
          }),
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
