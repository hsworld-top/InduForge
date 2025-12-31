<template>
  <el-form
    ref="formRef"
    :model="formData"
    :rules="rules"
    label-width="140px"
    class="mqtt-connection-form"
  >
    <!-- 基础信息 -->
    <div class="form-section">
      <div class="form-section-title">基础信息</div>

      <el-form-item label="连接名称" prop="name">
        <el-input
          v-model="formData.name"
          placeholder="请输入连接名称"
          clearable
        />
      </el-form-item>

      <el-form-item label="Broker 地址" prop="brokerUrl">
        <el-input
          v-model="formData.brokerUrl"
          placeholder="例如: broker.emqx.io"
          clearable
        >
          <template #prepend>
            <el-select v-model="formData.protocol" style="width: 90px">
              <el-option label="mqtt://" value="mqtt" />
              <el-option label="mqtts://" value="mqtts" />
              <el-option label="ws://" value="ws" />
              <el-option label="wss://" value="wss" />
            </el-select>
          </template>
        </el-input>
      </el-form-item>

      <el-form-item label="端口" prop="port">
        <el-input-number
          v-model="formData.port"
          :min="1"
          :max="65535"
          placeholder="端口号"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="客户端 ID" prop="clientId">
        <el-input
          v-model="formData.clientId"
          placeholder="留空自动生成"
          clearable
        >
          <template #append>
            <el-button @click="generateClientId" icon="Refresh">
              生成
            </el-button>
          </template>
        </el-input>
      </el-form-item>
    </div>

    <!-- 认证信息 -->
    <div class="form-section">
      <div class="form-section-title">认证信息</div>

      <el-form-item label="用户名">
        <el-input
          v-model="formData.username"
          placeholder="可选"
          clearable
          autocomplete="off"
        />
      </el-form-item>

      <el-form-item label="密码">
        <el-input
          v-model="formData.password"
          type="password"
          placeholder="可选"
          clearable
          show-password
          autocomplete="new-password"
        />
      </el-form-item>
    </div>

    <!-- 连接参数 -->
    <div class="form-section">
      <div class="form-section-title">连接参数</div>

      <el-form-item label="QoS">
        <el-radio-group v-model="formData.qos">
          <el-radio :label="0">0 - 最多一次</el-radio>
          <el-radio :label="1">1 - 至少一次</el-radio>
          <el-radio :label="2">2 - 恰好一次</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Keep Alive">
        <el-input-number
          v-model="formData.keepalive"
          :min="10"
          :max="300"
          placeholder="秒"
          style="width: 100%"
        />
        <span class="form-item-tip">心跳间隔，单位：秒</span>
      </el-form-item>

      <el-form-item label="Clean Session">
        <el-switch v-model="formData.cleanSession" />
        <span class="form-item-tip ml-2">
          是否清除会话（重连时是否保留订阅）
        </span>
      </el-form-item>

      <el-form-item label="连接超时">
        <el-input-number
          v-model="formData.connectTimeout"
          :min="1000"
          :max="60000"
          :step="1000"
          placeholder="毫秒"
          style="width: 100%"
        />
        <span class="form-item-tip">单位：毫秒</span>
      </el-form-item>

      <el-form-item label="重连间隔">
        <el-input-number
          v-model="formData.reconnectPeriod"
          :min="1000"
          :max="60000"
          :step="1000"
          placeholder="毫秒"
          style="width: 100%"
        />
        <span class="form-item-tip">单位：毫秒</span>
      </el-form-item>
    </div>

    <!-- 高级选项 -->
    <el-collapse v-model="activeCollapse" class="mt-4">
      <el-collapse-item title="遗嘱消息" name="will">
        <el-form-item label="遗嘱主题">
          <el-input
            v-model="formData.will.topic"
            placeholder="例如: device/status"
            clearable
          />
        </el-form-item>

        <el-form-item label="遗嘱消息">
          <el-input
            v-model="formData.will.payload"
            type="textarea"
            :rows="3"
            placeholder="例如: offline"
            clearable
          />
        </el-form-item>

        <el-form-item label="遗嘱 QoS">
          <el-radio-group v-model="formData.will.qos">
            <el-radio :label="0">0</el-radio>
            <el-radio :label="1">1</el-radio>
            <el-radio :label="2">2</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="遗嘱 Retain">
          <el-switch v-model="formData.will.retain" />
        </el-form-item>
      </el-collapse-item>

      <el-collapse-item title="SSL/TLS 配置" name="ssl">
        <el-form-item label="CA 证书">
          <el-input
            v-model="formData.sslConfig.ca"
            type="textarea"
            :rows="4"
            placeholder="PEM 格式的 CA 证书"
          />
        </el-form-item>

        <el-form-item label="客户端证书">
          <el-input
            v-model="formData.sslConfig.cert"
            type="textarea"
            :rows="4"
            placeholder="PEM 格式的客户端证书"
          />
        </el-form-item>

        <el-form-item label="客户端私钥">
          <el-input
            v-model="formData.sslConfig.key"
            type="textarea"
            :rows="4"
            placeholder="PEM 格式的客户端私钥"
          />
        </el-form-item>

        <el-form-item label="验证服务器证书">
          <el-switch v-model="formData.sslConfig.rejectUnauthorized" />
        </el-form-item>
      </el-collapse-item>
    </el-collapse>
  </el-form>
</template>

<script setup>
import { ref, reactive, watch, onMounted } from "vue";
import { getDefaultConfig } from "@/config/connectionTypes";

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
const activeCollapse = ref([]);

// 防止循环更新的标志
const isUpdatingFromParent = ref(false);

// 表单数据
const formData = ref({
  name: "",
  brokerUrl: "localhost",
  protocol: "mqtt",
  port: 1883,
  clientId: "",
  username: "",
  password: "",
  keepalive: 60,
  cleanSession: true,
  qos: 0,
  reconnectPeriod: 5000,
  connectTimeout: 30000,
  will: {
    topic: "",
    payload: "",
    qos: 0,
    retain: false,
  },
  sslConfig: {
    ca: "",
    cert: "",
    key: "",
    rejectUnauthorized: false,
  },
});

// 验证规则
const rules = {
  name: [
    { required: true, message: "请输入连接名称", trigger: "blur" },
    { min: 2, max: 100, message: "长度在 2 到 100 个字符", trigger: "blur" },
  ],
  brokerUrl: [
    { required: true, message: "请输入 Broker 地址", trigger: "blur" },
  ],
  port: [{ required: true, message: "请输入端口号", trigger: "blur" }],
};

// 初始化表单数据
onMounted(() => {
  isUpdatingFromParent.value = true;
  if (props.mode === "create") {
    // 创建模式：使用默认配置
    const defaultConfig = getDefaultConfig("mqtt");
    const newData = { ...defaultConfig, ...props.modelValue };
    // 确保 will 和 sslConfig 始终是对象
    if (!newData.will || typeof newData.will !== "object") {
      newData.will = { topic: "", payload: "", qos: 0, retain: false };
    }
    if (!newData.sslConfig || typeof newData.sslConfig !== "object") {
      newData.sslConfig = {
        ca: "",
        cert: "",
        key: "",
        rejectUnauthorized: false,
      };
    }
    formData.value = newData;
  } else {
    // 编辑模式：使用传入的数据
    const newData = { ...formData.value, ...props.modelValue };
    // 确保 will 和 sslConfig 始终是对象
    if (!newData.will || typeof newData.will !== "object") {
      newData.will = { topic: "", payload: "", qos: 0, retain: false };
    }
    if (!newData.sslConfig || typeof newData.sslConfig !== "object") {
      newData.sslConfig = {
        ca: "",
        cert: "",
        key: "",
        rejectUnauthorized: false,
      };
    }
    formData.value = newData;
  }
  // 使用 setTimeout 确保更新完成后再重置标志
  setTimeout(() => {
    isUpdatingFromParent.value = false;
  }, 0);
});

// 监听协议变化，自动更新默认端口
watch(
  () => formData.value.protocol,
  (newProtocol) => {
    const portMap = {
      mqtt: 1883,
      mqtts: 8883,
      ws: 8083,
      wss: 8084,
    };
    if (!isUpdatingFromParent.value) {
      formData.value.port = portMap[newProtocol] || 1883;
    }
  },
);

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
      const newData = { ...formData.value, ...newValue };
      // 确保 will 和 sslConfig 始终是对象
      if (!newData.will || typeof newData.will !== "object") {
        newData.will = { topic: "", payload: "", qos: 0, retain: false };
      }
      if (!newData.sslConfig || typeof newData.sslConfig !== "object") {
        newData.sslConfig = {
          ca: "",
          cert: "",
          key: "",
          rejectUnauthorized: false,
        };
      }
      formData.value = newData;
      setTimeout(() => {
        isUpdatingFromParent.value = false;
      }, 0);
    }
  },
  { deep: true },
);

// 生成客户端 ID
const generateClientId = () => {
  const randomStr = Math.random().toString(36).substring(2, 10);
  formData.value.clientId = `induforge_${randomStr}`;
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
  } catch (error) {
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

<style scoped>
.mqtt-connection-form {
  padding: 20px;
}

.form-section {
  margin-bottom: 24px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.form-section-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid #dcdfe6;
}

.form-item-tip {
  font-size: 12px;
  color: #909399;
  margin-left: 8px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 24px;
  border-top: 1px solid #dcdfe6;
}

:deep(.el-collapse-item__header) {
  font-weight: 500;
}

:deep(.el-radio) {
  margin-right: 20px;
}
</style>
