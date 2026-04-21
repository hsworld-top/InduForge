<template>
  <el-dialog
    v-model="visible"
    :title="dialogTitle"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <!-- 连接类型选择（仅创建模式） -->
    <el-form v-if="mode === 'create'" label-width="120px" class="mb-4">
      <el-form-item :label="t('connection.connectionType')">
        <el-select
          v-model="connectionType"
          :placeholder="t('connection.selectConnectionType')"
          class="w-full"
          @change="handleTypeChange"
        >
          <el-option
            :label="t('connection.relationalDatabase')"
            value="relational"
          />
          <el-option label="MQTT" value="mqtt" />
        </el-select>
      </el-form-item>

      <el-form-item
        v-if="connectionType === 'relational'"
        :label="t('connection.databaseType')"
      >
        <el-select
          v-model="dbType"
          :placeholder="t('connection.selectDatabaseType')"
          class="w-full"
          @change="handleDbTypeChange"
        >
          <el-option
            v-for="type in databaseTypes"
            :key="type.value"
            :label="type.label"
            :value="type.value"
          />
        </el-select>
      </el-form-item>
    </el-form>

    <!-- 动态加载对应的表单组件 -->
    <component
      v-if="formComponent"
      :is="formComponent"
      ref="formRef"
      v-model="formData"
      :mode="mode"
      @validate="handleValidate"
    />

    <template #footer>
      <el-button @click="handleClose">{{ t("actions.cancel") }}</el-button>
      <el-button @click="handleTest" :loading="testing">{{
        t("actions.testConnection")
      }}</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="submitting">
        {{
          mode === "create"
            ? t("actions.createConnection")
            : t("actions.saveChanges")
        }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, markRaw } from "vue";
import { ElMessage } from "element-plus";
import { getDatabaseTypes, getDefaultConfig } from "@/config/connectionTypes";
import { t } from "@/i18n/runtime";
import MysqlConnectionForm from "../connection/forms/MysqlConnectionForm.vue";
import PostgresConnectionForm from "../connection/forms/PostgresConnectionForm.vue";
import SqlServerConnectionForm from "../connection/forms/SqlServerConnectionForm.vue";
import MqttConnectionForm from "../connection/forms/MqttConnectionForm.vue";

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  mode: {
    type: String,
    default: "create", // 'create' | 'edit'
    validator: (value) => ["create", "edit"].includes(value),
  },
  connection: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits(["update:modelValue", "submit", "test"]);

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});

const dialogTitle = computed(() => {
  return props.mode === "create"
    ? t("connection.createTitle")
    : t("connection.editTitle");
});

const connectionType = ref("relational");
const dbType = ref("mysql");
const formData = ref({});
const formRef = ref(null);
const testing = ref(false);
const submitting = ref(false);
const databaseTypes = getDatabaseTypes();

// 动态加载表单组件
const formComponent = computed(() => {
  if (connectionType.value === "relational") {
    const componentMap = {
      mysql: markRaw(MysqlConnectionForm),
      postgresql: markRaw(PostgresConnectionForm),
      sqlserver: markRaw(SqlServerConnectionForm),
    };
    return componentMap[dbType.value] || null;
  } else if (connectionType.value === "mqtt") {
    return markRaw(MqttConnectionForm);
  }
  return null;
});

// 监听连接数据变化（编辑模式）
watch(
  () => props.connection,
  (newConnection) => {
    if (newConnection && props.mode === "edit") {
      connectionType.value = newConnection.type;
      if (
        newConnection.type === "relational" &&
        newConnection.relationalConfig
      ) {
        dbType.value = newConnection.relationalConfig.dbType;
        formData.value = {
          name: newConnection.name,
          ...newConnection.relationalConfig,
        };
      } else if (newConnection.type === "mqtt" && newConnection.mqttConfig) {
        formData.value = {
          name: newConnection.name,
          ...newConnection.mqttConfig,
        };
      }
    }
  },
  { immediate: true },
);

// 监听对话框打开（创建模式）
watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen && props.mode === "create") {
      // 重置为默认值
      connectionType.value = "relational";
      dbType.value = "mysql";
      formData.value = getDefaultConfig("mysql");
    }
  },
);

const handleTypeChange = () => {
  // 连接类型变化时重置表单
  if (connectionType.value === "relational") {
    dbType.value = "mysql";
    formData.value = getDefaultConfig("mysql");
  } else if (connectionType.value === "mqtt") {
    formData.value = getDefaultConfig("mqtt");
  }
};

const handleDbTypeChange = () => {
  // 数据库类型变化时重置表单
  formData.value = getDefaultConfig(dbType.value);
};

const handleValidate = (valid, data) => {
  // 表单验证回调
  console.log("表单验证:", valid, data);
};

const handleTest = async () => {
  // 验证表单
  if (!formRef.value) return;

  const valid = await formRef.value.validate();
  if (!valid) {
    ElMessage.warning(t("connection.incompleteInfoWarning"));
    return;
  }

  testing.value = true;
  try {
    const config = { ...formData.value };
    delete config.name; // 测试连接不需要名称

    // 根据连接类型添加额外字段
    if (connectionType.value === "relational") {
      config.dbType = dbType.value;
    }

    emit("test", {
      type: connectionType.value,
      config,
    });
  } finally {
    testing.value = false;
  }
};

const handleSubmit = async () => {
  // 验证表单
  if (!formRef.value) return;

  const valid = await formRef.value.validate();
  if (!valid) {
    ElMessage.warning(t("connection.incompleteInfoWarning"));
    return;
  }

  submitting.value = true;
  try {
    const config = { ...formData.value };
    const name = config.name;
    delete config.name;

    // 根据连接类型添加额外字段
    if (connectionType.value === "relational") {
      config.dbType = dbType.value;
    }

    emit("submit", {
      name,
      type: connectionType.value,
      config,
    });
  } finally {
    submitting.value = false;
  }
};

const handleClose = () => {
  visible.value = false;
  // 清空表单
  if (formRef.value) {
    formRef.value.clearValidate();
  }
};
</script>
