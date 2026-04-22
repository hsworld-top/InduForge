<template>
  <el-form ref="formRef" :model="formData" :rules="rules" label-width="120px">
    <el-form-item :label="t('connection.name')" prop="name">
      <el-input v-model="formData.name" :placeholder="t('connection.name')" />
    </el-form-item>

    <el-form-item :label="t('connection.host')" prop="host">
      <el-input v-model="formData.host" placeholder="localhost" />
    </el-form-item>

    <el-form-item :label="t('connection.port')" prop="port">
      <el-input-number
        v-model="formData.port"
        :min="1"
        :max="65535"
        class="w-full"
      />
    </el-form-item>

    <el-form-item :label="t('connection.database')" prop="database">
      <el-input
        v-model="formData.database"
        :placeholder="t('connection.database')"
      />
    </el-form-item>

    <el-form-item :label="t('connection.username')" prop="username">
      <el-input v-model="formData.username" placeholder="postgres" />
    </el-form-item>

    <el-form-item :label="t('connection.password')" prop="password">
      <el-input
        v-model="formData.password"
        type="password"
        :placeholder="t('connection.password')"
        show-password
      />
    </el-form-item>

    <el-form-item :label="t('connection.schema')">
      <el-input v-model="formData.schema" placeholder="public" />
      <span class="text-xs text-gray-500 ml-2">{{
        t("connection.schemaHint")
      }}</span>
    </el-form-item>

    <el-form-item :label="t('connection.sslConnection')">
      <el-select
        v-model="formData.sslMode"
        :placeholder="t('connection.sslModePlaceholder')"
        class="w-full"
        @change="handleSslModeChange"
      >
        <el-option :label="t('connection.sslDisabled')" value="disable">
          <span>{{ t("connection.sslDisabled") }}</span>
          <span class="text-xs text-gray-400 ml-2"
            >- {{ t("connection.sslDisabledHint") }}</span
          >
        </el-option>
        <el-option :label="t('connection.sslPrefer')" value="prefer">
          <span>{{ t("connection.sslPrefer") }}</span>
          <span class="text-xs text-gray-400 ml-2"
            >- {{ t("connection.sslPreferHint") }}</span
          >
        </el-option>
        <el-option :label="t('connection.sslRequire')" value="require">
          <span>{{ t("connection.sslRequire") }}</span>
          <span class="text-xs text-gray-400 ml-2"
            >- {{ t("connection.sslRequireHint") }}</span
          >
        </el-option>
        <el-option :label="t('connection.sslVerify')" value="verify-ca">
          <span>{{ t("connection.sslVerify") }}</span>
          <span class="text-xs text-gray-400 ml-2"
            >- {{ t("connection.sslVerifyHint") }}</span
          >
        </el-option>
      </el-select>
    </el-form-item>

    <!-- SSL 证书配置（仅在 verify-ca 模式下显示） -->
    <template v-if="needsCertificate">
      <el-divider content-position="left">{{
        t("connection.sslCertificateConfig")
      }}</el-divider>

      <el-form-item :label="t('connection.caCertificate')">
        <el-input
          v-model="formData.sslCa"
          type="textarea"
          :rows="4"
          placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
        />
        <span class="text-xs text-gray-500 ml-2">{{
          t("connection.caCertificateHint")
        }}</span>
      </el-form-item>

      <el-form-item :label="t('connection.clientCertificate')">
        <el-input
          v-model="formData.sslCert"
          type="textarea"
          :rows="4"
          placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
        />
        <span class="text-xs text-gray-500 ml-2">{{
          t("connection.clientCertificateHint")
        }}</span>
      </el-form-item>

      <el-form-item :label="t('connection.clientKey')">
        <el-input
          v-model="formData.sslKey"
          type="textarea"
          :rows="4"
          placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
        />
        <span class="text-xs text-gray-500 ml-2">{{
          t("connection.clientKeyHint")
        }}</span>
      </el-form-item>
    </template>

    <el-form-item :label="t('connection.connectionTimeout')">
      <el-input-number
        v-model="formData.connectionTimeout"
        :min="1000"
        :max="60000"
        :step="1000"
        class="w-full"
      />
      <span class="text-xs text-gray-500 ml-2">{{
        t("connection.connectionTimeoutHint")
      }}</span>
    </el-form-item>

    <el-form-item :label="t('connection.queryTimeout')">
      <el-input-number
        v-model="formData.queryTimeout"
        :min="1000"
        :max="300000"
        :step="1000"
        class="w-full"
      />
      <span class="text-xs text-gray-500 ml-2">{{
        t("common.milliseconds")
      }}</span>
    </el-form-item>

    <el-form-item :label="t('connection.maxConnections')">
      <el-input-number
        v-model="formData.maxConnections"
        :min="1"
        :max="100"
        class="w-full"
      />
      <span class="text-xs text-gray-500 ml-2">{{
        t("connection.maxConnectionsHint")
      }}</span>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, computed } from "vue";
import { getDefaultConfig } from "@/config/connectionTypes";
import { t } from "@/i18n/runtime";

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({}),
  },
  mode: {
    type: String,
    default: "create", // 'create' | 'edit'
    validator: (value) => ["create", "edit"].includes(value),
  },
});

const emit = defineEmits(["update:modelValue", "validate"]);

const formRef = ref(null);
const formData = ref({
  name: "",
  host: "localhost",
  port: 5432,
  database: "",
  username: "postgres",
  password: "",
  schema: "public",
  sslMode: "disable",
  sslCa: "",
  sslCert: "",
  sslKey: "",
  connectionTimeout: 3000,
  queryTimeout: 30000,
  maxConnections: 10,
});

// 防止循环更新的标志
const isUpdatingFromParent = ref(false);

// 是否需要证书配置
const needsCertificate = computed(() => {
  return formData.value.sslMode === "verify-ca";
});

const rules = {
  name: [
    { required: true, message: t("connection.name"), trigger: "blur" },
    {
      min: 2,
      max: 100,
      message: t("query.nameLengthError"),
      trigger: "blur",
    },
  ],
  host: [{ required: true, message: t("connection.host"), trigger: "blur" }],
  port: [{ required: true, message: t("connection.port"), trigger: "blur" }],
  database: [
    { required: true, message: t("connection.database"), trigger: "blur" },
  ],
  username: [
    { required: true, message: t("connection.username"), trigger: "blur" },
  ],
};

// 初始化表单数据
onMounted(() => {
  isUpdatingFromParent.value = true;
  if (props.mode === "create") {
    // 创建模式：使用默认配置
    const defaultConfig = getDefaultConfig("postgresql");
    formData.value = { ...defaultConfig, ...props.modelValue };
  } else {
    // 编辑模式：使用传入的数据
    formData.value = { ...formData.value, ...props.modelValue };
  }
  // 使用 nextTick 确保更新完成后再重置标志
  setTimeout(() => {
    isUpdatingFromParent.value = false;
  }, 0);
});

// 监听表单数据变化，向上传递
watch(
  formData,
  (newValue) => {
    // 只有在不是从父组件更新时才向上传递
    if (!isUpdatingFromParent.value) {
      emit("update:modelValue", { ...newValue });
    }
  },
  { deep: true },
);

// 监听外部数据变化
watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue && Object.keys(newValue).length > 0) {
      isUpdatingFromParent.value = true;
      formData.value = { ...formData.value, ...newValue };
      setTimeout(() => {
        isUpdatingFromParent.value = false;
      }, 0);
    }
  },
  { deep: true },
);

/**
 * SSL 模式变化处理
 */
const handleSslModeChange = (mode) => {
  // 如果切换到不需要证书的模式，清空证书字段
  if (mode !== "verify-ca") {
    formData.value.sslCa = "";
    formData.value.sslCert = "";
    formData.value.sslKey = "";
  }
};

/**
 * 验证表单
 */
const validate = async () => {
  if (!formRef.value) return false;

  try {
    await formRef.value.validate();
    emit("validate", true, formData.value);
    return true;
  } catch {
    emit("validate", false, null);
    return false;
  }
};

/**
 * 重置表单
 */
const resetFields = () => {
  if (formRef.value) {
    formRef.value.resetFields();
  }
};

/**
 * 清空验证
 */
const clearValidate = () => {
  if (formRef.value) {
    formRef.value.clearValidate();
  }
};

// 暴露方法给父组件
defineExpose({
  validate,
  resetFields,
  clearValidate,
  formData,
});
</script>
