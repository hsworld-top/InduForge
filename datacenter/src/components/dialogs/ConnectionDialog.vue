<template>
  <DcDialog
    v-model="visible"
    class="connection-dialog"
    width="920px"
    :dirty="isDirty"
    :body-max-height="'none'"
    @close="handleClosed"
  >
    <!-- Step 1：仅 create 模式，选择接入类型 -->
    <template v-if="step === 1">
      <div class="connection-dialog__step1">
        <div class="connection-dialog__step1-header">
          <h3 class="connection-dialog__step1-title">选择接入类型</h3>
          <p class="connection-dialog__step1-sub">先选择协议，再填写配置参数</p>
        </div>
        <div class="connection-dialog__step1-grid">
          <button
            v-for="source in sourceOptions"
            :key="source.value"
            type="button"
            class="connection-dialog__step1-card"
            :class="{ 'is-active': connectionType === source.value }"
            @click="selectSourceAndAdvance(source.value)"
          >
            <component :is="source.icon" class="connection-dialog__step1-icon" />
            <strong>{{ source.label }}</strong>
            <span class="connection-dialog__step1-desc">{{ source.description }}</span>
          </button>
        </div>
      </div>
    </template>

    <!-- Step 2：两栏布局（表单 + inspector），顶部展示当前协议 -->
    <template v-else>
    <div class="connection-dialog__body">
      <main class="connection-dialog__form-panel">
        <div class="connection-dialog__form-heading">
          <div class="connection-dialog__heading-main">
            <span class="connection-dialog__heading-chip">
              <component
                v-if="activeSource.icon"
                :is="activeSource.icon"
                class="connection-dialog__heading-chip-icon"
              />
              <span>{{ activeSource.label }}</span>
            </span>
            <h3>{{ activeFormTitle }}</h3>
          </div>
          <el-button
            v-if="mode === 'create'"
            size="small"
            text
            @click="resetCurrentForm"
          >
            重置默认值
          </el-button>
        </div>

        <component
          v-if="formComponent"
          :is="formComponent"
          ref="formRef"
          v-model="formData"
          :mode="mode"
          @validate="handleValidate"
        />

        <el-form
          v-else
          ref="protocolFormRef"
          :model="formData"
          :rules="protocolRules"
          label-position="top"
          class="connection-dialog__protocol-form"
        >
          <el-form-item label="连接名称" prop="name">
            <el-input v-model="formData.name" placeholder="例如：产线 Kafka" />
          </el-form-item>

          <template v-if="connectionType === 'kafka'">
            <el-form-item label="Broker" prop="brokers">
              <el-input
                v-model="formData.brokers"
                placeholder="127.0.0.1:9092,127.0.0.2:9092"
              />
            </el-form-item>
            <el-form-item label="Topic" prop="topic">
              <el-input
                v-model="formData.topic"
                placeholder="device.telemetry"
              />
            </el-form-item>
            <el-form-item label="Consumer Group" prop="consumerGroup">
              <el-input
                v-model="formData.consumerGroup"
                placeholder="datacenter-preview"
              />
            </el-form-item>
            <el-form-item label="起始位置">
              <el-segmented
                v-model="formData.startPosition"
                :options="kafkaStartOptions"
              />
            </el-form-item>
            <el-form-item label="扩展参数">
              <el-input
                v-model="formData.optionsText"
                type="textarea"
                :rows="4"
                placeholder='{"securityProtocol":"PLAINTEXT"}'
              />
            </el-form-item>
          </template>

          <template v-else-if="connectionType === 'http'">
            <el-form-item label="请求地址" prop="baseUrl">
              <el-input
                v-model="formData.baseUrl"
                placeholder="https://api.example.com/data"
              />
            </el-form-item>
            <div class="connection-dialog__form-grid">
              <el-form-item label="方法">
                <el-select v-model="formData.method" class="w-full">
                  <el-option
                    v-for="method in httpMethods"
                    :key="method"
                    :label="method"
                    :value="method"
                  />
                </el-select>
              </el-form-item>
              <el-form-item label="超时">
                <el-input
                  v-model.number="formData.timeoutMs"
                  inputmode="numeric"
                  placeholder="5000"
                >
                  <template #append>ms</template>
                </el-input>
              </el-form-item>
            </div>
            <el-form-item label="请求头">
              <el-input
                v-model="formData.headersText"
                type="textarea"
                :rows="4"
                placeholder='{"Authorization":"Bearer token"}'
              />
            </el-form-item>
            <el-form-item label="请求体模板">
              <el-input
                v-model="formData.bodyTemplateText"
                type="textarea"
                :rows="5"
                placeholder='{"deviceId":"demo"}'
              />
            </el-form-item>
          </template>

          <template v-else-if="connectionType === 'websocket'">
            <el-form-item label="连接地址" prop="url">
              <el-input
                v-model="formData.url"
                placeholder="wss://example.com/realtime"
              />
            </el-form-item>
            <div class="connection-dialog__form-grid">
              <el-form-item label="主题/通道">
                <el-input v-model="formData.topic" placeholder="可选" />
              </el-form-item>
              <el-form-item label="心跳间隔">
                <el-input
                  v-model.number="formData.heartbeatIntervalMs"
                  inputmode="numeric"
                  placeholder="30000"
                >
                  <template #append>ms</template>
                </el-input>
              </el-form-item>
            </div>
            <el-form-item label="握手 Header">
              <el-input
                v-model="formData.headersText"
                type="textarea"
                :rows="4"
                placeholder='{"Authorization":"Bearer token"}'
              />
            </el-form-item>
            <el-form-item label="订阅消息">
              <el-input
                v-model="formData.subscribeMessage"
                type="textarea"
                :rows="4"
                placeholder='{"type":"subscribe","topic":"device.telemetry"}'
              />
            </el-form-item>
          </template>

          <template v-else-if="connectionType === 'redis'">
            <el-form-item label="部署模式">
              <el-segmented
                v-model="formData.mode"
                :options="redisModeOptions"
              />
            </el-form-item>
            <el-form-item label="地址" prop="address">
              <el-input
                v-model="formData.address"
                placeholder="127.0.0.1:6379，多节点用英文逗号分隔"
              />
            </el-form-item>
            <div class="connection-dialog__form-grid">
              <el-form-item label="DB">
                <el-input
                  v-model.number="formData.db"
                  inputmode="numeric"
                  placeholder="0"
                />
              </el-form-item>
              <el-form-item v-if="formData.mode === 'sentinel'" label="Master">
                <el-input
                  v-model="formData.masterName"
                  placeholder="mymaster"
                />
              </el-form-item>
              <el-form-item label="Key Pattern">
                <el-input
                  v-model="formData.keyPattern"
                  placeholder="device:*"
                />
              </el-form-item>
            </div>
            <div class="connection-dialog__form-grid">
              <el-form-item label="用户名">
                <el-input v-model="formData.username" placeholder="可选" />
              </el-form-item>
              <el-form-item label="密码">
                <el-input
                  v-model="formData.password"
                  type="password"
                  show-password
                  placeholder="可选"
                />
              </el-form-item>
            </div>
          </template>

          <template v-else-if="connectionType === 'opcua'">
            <div class="connection-dialog__form-grid">
              <el-form-item label="IP 地址" prop="ip">
                <el-input v-model="formData.ip" placeholder="127.0.0.1" />
              </el-form-item>
              <el-form-item label="端口">
                <el-input
                  v-model.number="formData.port"
                  inputmode="numeric"
                  placeholder="4840"
                />
              </el-form-item>
            </div>
            <div class="connection-dialog__form-grid">
              <el-form-item label="安全策略">
                <el-select v-model="formData.securityPolicy" class="w-full">
                  <el-option label="None" value="None" />
                  <el-option label="Basic256Sha256" value="Basic256Sha256" />
                  <el-option label="Basic256" value="Basic256" />
                </el-select>
              </el-form-item>
              <el-form-item label="安全模式">
                <el-segmented
                  v-model="formData.securityMode"
                  :options="opcuaSecurityModeOptions"
                />
              </el-form-item>
            </div>
            <el-form-item label="认证方式">
              <el-segmented
                v-model="formData.authType"
                :options="opcuaAuthOptions"
              />
            </el-form-item>
            <div
              v-if="formData.authType === 'username_password'"
              class="connection-dialog__form-grid"
            >
              <el-form-item label="用户名" prop="username">
                <el-input
                  v-model="formData.username"
                  placeholder="OPC UA 用户名"
                />
              </el-form-item>
              <el-form-item label="密码">
                <el-input
                  v-model="formData.password"
                  type="password"
                  show-password
                  placeholder="OPC UA 密码"
                />
              </el-form-item>
            </div>
            <el-form-item label="采样周期">
              <el-input
                v-model.number="formData.samplingMs"
                inputmode="numeric"
                placeholder="1000"
              >
                <template #append>ms</template>
              </el-input>
            </el-form-item>
            <el-form-item label="扩展参数">
              <el-input
                v-model="formData.optionsText"
                type="textarea"
                :rows="4"
                placeholder='{"namespace":"urn:demo"}'
              />
            </el-form-item>
          </template>

          <template v-else-if="connectionType === 's7'">
            <div class="connection-dialog__form-grid">
              <el-form-item label="PLC IP" prop="ip">
                <el-input v-model="formData.ip" placeholder="192.168.1.10" />
              </el-form-item>
              <el-form-item label="端口">
                <el-input
                  v-model.number="formData.port"
                  inputmode="numeric"
                  placeholder="102"
                />
              </el-form-item>
            </div>
            <div class="connection-dialog__form-grid">
              <el-form-item label="Rack">
                <el-input
                  v-model.number="formData.rack"
                  inputmode="numeric"
                  placeholder="0"
                />
              </el-form-item>
              <el-form-item label="Slot">
                <el-input
                  v-model.number="formData.slot"
                  inputmode="numeric"
                  placeholder="1"
                />
              </el-form-item>
            </div>
            <el-form-item label="轮询周期">
              <el-input
                v-model.number="formData.pollIntervalMs"
                inputmode="numeric"
                placeholder="1000"
              >
                <template #append>ms</template>
              </el-input>
            </el-form-item>
            <el-form-item label="扩展参数">
              <el-input
                v-model="formData.optionsText"
                type="textarea"
                :rows="4"
                placeholder='{"pduSize":480}'
              />
            </el-form-item>
          </template>

          <template v-else-if="connectionType === 'modbus'">
            <el-form-item label="模式">
              <el-segmented
                v-model="formData.mode"
                :options="modbusModeOptions"
              />
            </el-form-item>
            <div
              v-if="formData.mode === 'tcp'"
              class="connection-dialog__form-grid"
            >
              <el-form-item label="IP 地址" prop="ip">
                <el-input v-model="formData.ip" placeholder="192.168.1.20" />
              </el-form-item>
              <el-form-item label="端口">
                <el-input
                  v-model.number="formData.port"
                  inputmode="numeric"
                  placeholder="502"
                />
              </el-form-item>
            </div>
            <el-form-item v-else label="串口配置" prop="serialConfigText">
              <el-input
                v-model="formData.serialConfigText"
                type="textarea"
                :rows="5"
                placeholder='{"port":"COM3","baudRate":9600,"dataBits":8,"parity":"N","stopBits":1}'
              />
            </el-form-item>
            <div class="connection-dialog__form-grid">
              <el-form-item label="站号">
                <el-input
                  v-model.number="formData.slaveId"
                  inputmode="numeric"
                  placeholder="1"
                />
              </el-form-item>
              <el-form-item label="起始地址">
                <el-input
                  v-model.number="formData.startAddress"
                  inputmode="numeric"
                  placeholder="0"
                />
              </el-form-item>
              <el-form-item label="数量">
                <el-input
                  v-model.number="formData.quantity"
                  inputmode="numeric"
                  placeholder="1"
                />
              </el-form-item>
            </div>
            <el-form-item label="轮询周期">
              <el-input
                v-model.number="formData.pollIntervalMs"
                inputmode="numeric"
                placeholder="1000"
              >
                <template #append>ms</template>
              </el-input>
            </el-form-item>
            <el-form-item label="扩展参数">
              <el-input
                v-model="formData.optionsText"
                type="textarea"
                :rows="4"
                placeholder='{"functionCode":3}'
              />
            </el-form-item>
          </template>

          <template v-else-if="connectionType === 'tdengine'">
            <div class="connection-dialog__form-grid">
              <el-form-item label="IP 地址" prop="ip">
                <el-input v-model="formData.ip" placeholder="127.0.0.1" />
              </el-form-item>
              <el-form-item label="端口">
                <el-input
                  v-model.number="formData.port"
                  inputmode="numeric"
                  placeholder="6030"
                />
              </el-form-item>
            </div>
            <div class="connection-dialog__form-grid">
              <el-form-item label="Database" prop="database">
                <el-input v-model="formData.database" placeholder="iot_data" />
              </el-form-item>
              <el-form-item label="时区">
                <el-input
                  v-model="formData.timezone"
                  placeholder="Asia/Shanghai"
                />
              </el-form-item>
            </div>
            <div class="connection-dialog__form-grid">
              <el-form-item label="用户名">
                <el-input v-model="formData.username" placeholder="root" />
              </el-form-item>
              <el-form-item label="密码">
                <el-input
                  v-model="formData.password"
                  type="password"
                  show-password
                  placeholder="taosdata"
                />
              </el-form-item>
            </div>
            <el-form-item label="扩展参数">
              <el-input
                v-model="formData.optionsText"
                type="textarea"
                :rows="4"
                placeholder='{"stable":"device_metrics"}'
              />
            </el-form-item>
          </template>
        </el-form>
      </main>

      <aside class="connection-dialog__inspector">
        <section class="connection-dialog__summary">
          <div class="connection-dialog__section-title">配置摘要</div>
          <dl>
            <template v-for="row in summaryRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </section>

        <section class="connection-dialog__test">
          <div class="connection-dialog__section-title">连接测试</div>
          <div
            class="connection-dialog__test-state"
            :class="`is-${testState.status}`"
          >
            <component
              :is="testState.icon"
              class="connection-dialog__test-icon"
            />
            <div>
              <strong>{{ testState.title }}</strong>
              <p>{{ testState.message }}</p>
            </div>
          </div>
          <div
            v-if="testState.detail"
            class="connection-dialog__test-detail"
            :title="testState.detail"
          >
            {{ testState.detail }}
          </div>
          <div v-if="isTestStale" class="connection-dialog__test-stale">
            配置已修改，建议重新测试后保存。
          </div>
        </section>

        <section class="connection-dialog__checklist">
          <div class="connection-dialog__section-title">保存前检查</div>
          <div
            v-for="item in checklist"
            :key="item.label"
            class="connection-dialog__check-item"
            :class="{ 'is-ready': item.ready }"
          >
            <IconTablerCircleCheck class="connection-dialog__check-icon" />
            <span>{{ item.label }}</span>
          </div>
        </section>
      </aside>
    </div>

    </template>

    <!-- footer：step 1 只有取消，step 2 完整操作 -->
    <template #footer>
      <div class="connection-dialog__footer">
        <div class="connection-dialog__footer-actions">
          <template v-if="step === 1">
            <el-button @click="requestClose">{{ t("actions.cancel") }}</el-button>
          </template>
          <template v-else>
            <!-- 返回上一步：仅 create 模式可见 -->
            <el-button v-if="mode === 'create'" @click="goBackToStep1">
              ← 返回上一步
            </el-button>
            <el-button @click="requestClose">{{ t("actions.cancel") }}</el-button>
            <el-button @click="handleTest" :loading="testing">
              {{ t("actions.testConnection") }}
            </el-button>
            <el-button type="primary" @click="handleSubmit" :loading="submitting">
              {{
                mode === "create"
                  ? t("actions.createConnection")
                  : t("actions.saveChanges")
              }}
            </el-button>
          </template>
        </div>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, markRaw } from "vue";
import { ElMessage } from "element-plus";
import { getDefaultConfig } from "@/config/connectionTypes";
import { t } from "@/i18n/runtime";
import dataAPI from "@/api/data.api";
import { getApiErrorMessage } from "@/utils/request";
import IconTablerAlertTriangle from "~icons/tabler/alert-triangle";
import IconTablerCircleCheck from "~icons/tabler/circle-check";
import IconTablerDatabase from "~icons/tabler/database";
import IconTablerLoader2 from "~icons/tabler/loader-2";
import IconTablerMessageCircle from "~icons/tabler/message-circle";
import IconTablerPlugConnected from "~icons/tabler/plug-connected";
import IconTablerServer from "~icons/tabler/server";
import IconTablerWorldWww from "~icons/tabler/world-www";
import IconTablerWebhook from "~icons/tabler/webhook";
import MysqlConnectionForm from "../connection/forms/MysqlConnectionForm.vue";
import PostgresConnectionForm from "../connection/forms/PostgresConnectionForm.vue";
import SqlServerConnectionForm from "../connection/forms/SqlServerConnectionForm.vue";
import MqttConnectionForm from "../connection/forms/MqttConnectionForm.vue";
import DcDialog from "@/components/shared/DcDialog.vue";

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
  projectId: {
    type: [String, Number],
    default: "",
  },
});

const emit = defineEmits(["update:modelValue", "submit"]);

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});

// step 状态机：create 从 1 开始，edit 直接 2
const step = ref<1 | 2>(props.mode === "edit" ? 2 : 1);

// 点击协议卡进入 Step 2
const selectSourceAndAdvance = (value: string) => {
  selectSource(value);
  step.value = 2;
};

// 返回上一步：清空表单与测试状态，保留协议高亮
const goBackToStep1 = () => {
  formData.value = {};
  resetTestState();
  step.value = 1;
};

const connectionType = ref("mysql");
const dbType = ref("mysql");
const formData = ref({});
const formRef = ref(null);
const protocolFormRef = ref(null);
const testing = ref(false);
const submitting = ref(false);
const lastTestSignature = ref("");
const initialDialogSignature = ref("");
const testResult = ref({
  status: "idle",
  title: "尚未测试",
  message: "填写连接参数后，可以先验证网络、认证与基础协议是否可用。",
  detail: "",
  durationMs: 0,
});

const sourceOptions = [
  {
    value: "mysql",
    label: "MySQL",
    description: "MySQL 数据库连接",
    icon: markRaw(IconTablerDatabase),
  },
  {
    value: "postgresql",
    label: "PG",
    description: "PostgreSQL 数据库连接",
    icon: markRaw(IconTablerDatabase),
  },
  {
    value: "sqlserver",
    label: "SqlServer",
    description: "SQL Server 数据库连接",
    icon: markRaw(IconTablerDatabase),
  },
  {
    value: "mqtt",
    label: "MQTT",
    description: "Broker 连接与订阅配置",
    icon: markRaw(IconTablerMessageCircle),
  },
  {
    value: "kafka",
    label: "Kafka",
    description: "Topic 消息短时抓样",
    icon: markRaw(IconTablerServer),
  },
  {
    value: "http",
    label: "HTTP",
    description: "接口请求与响应样本",
    icon: markRaw(IconTablerWorldWww),
  },
  {
    value: "websocket",
    label: "WebSocket",
    description: "短连接消息预览",
    icon: markRaw(IconTablerWebhook),
  },
  {
    value: "redis",
    label: "Redis",
    description: "Key 扫描与值样本",
    icon: markRaw(IconTablerDatabase),
  },
  {
    value: "opcua",
    label: "OPC UA",
    description: "IP、端口与安全策略",
    icon: markRaw(IconTablerServer),
  },
  {
    value: "s7",
    label: "Siemens S7",
    description: "PLC Rack/Slot 配置",
    icon: markRaw(IconTablerServer),
  },
  {
    value: "modbus",
    label: "Modbus",
    description: "TCP/RTU 站号与寄存器",
    icon: markRaw(IconTablerWebhook),
  },
  {
    value: "tdengine",
    label: "TDengine",
    description: "时序库连接契约",
    icon: markRaw(IconTablerDatabase),
  },
];

const previewProtocolTypes = ["kafka", "http", "websocket", "redis"];
const industrialProtocolTypes = ["opcua", "s7", "modbus", "tdengine"];
const relationalSourceTypes = ["mysql", "postgresql", "sqlserver"];
const specializedProtocolTypes = [
  ...previewProtocolTypes,
  ...industrialProtocolTypes,
];
const httpMethods = ["GET", "POST", "PUT", "PATCH", "DELETE"];
const kafkaStartOptions = [
  { label: "Latest", value: "latest" },
  { label: "Earliest", value: "earliest" },
];
const redisModeOptions = [
  { label: "Standalone", value: "standalone" },
  { label: "Sentinel", value: "sentinel" },
  { label: "Cluster", value: "cluster" },
];
const opcuaSecurityModeOptions = [
  { label: "None", value: "none" },
  { label: "Sign", value: "sign" },
  { label: "Sign & Encrypt", value: "signandencrypt" },
];
const opcuaAuthOptions = [
  { label: "匿名", value: "anonymous" },
  { label: "用户名密码", value: "username_password" },
];
const modbusModeOptions = [
  { label: "TCP", value: "tcp" },
  { label: "RTU", value: "rtu" },
];

const protocolRules = {
  name: [{ required: true, message: "连接名称不能为空", trigger: "blur" }],
  brokers: [{ required: true, message: "Broker 不能为空", trigger: "blur" }],
  topic: [{ required: true, message: "Topic 不能为空", trigger: "blur" }],
  consumerGroup: [
    { required: true, message: "Consumer Group 不能为空", trigger: "blur" },
  ],
  baseUrl: [{ required: true, message: "请求地址不能为空", trigger: "blur" }],
  url: [{ required: true, message: "连接地址不能为空", trigger: "blur" }],
  address: [{ required: true, message: "Redis 地址不能为空", trigger: "blur" }],
  ip: [{ required: true, message: "IP 地址不能为空", trigger: "blur" }],
  username: [{ required: true, message: "用户名不能为空", trigger: "blur" }],
  serialConfigText: [
    { required: true, message: "串口配置不能为空", trigger: "blur" },
  ],
  database: [{ required: true, message: "Database 不能为空", trigger: "blur" }],
};

const activeSource = computed(() => {
  return (
    sourceOptions.find((item) => item.value === connectionType.value) ||
    sourceOptions[0]
  );
});

const activeDatabase = computed(() =>
  sourceOptions.find((item) => item.value === dbType.value),
);

const activeFormTitle = computed(() => {
  if (connectionType.value === "mqtt") {
    return "MQTT Broker";
  }
  if (connectionType.value === "kafka") return "Kafka Topic";
  if (connectionType.value === "http") return "HTTP Source";
  if (connectionType.value === "websocket") return "WebSocket Source";
  if (connectionType.value === "redis") return "Redis Source";
  if (connectionType.value === "opcua") return "OPC UA Server";
  if (connectionType.value === "s7") return "Siemens S7 PLC";
  if (connectionType.value === "modbus") return "Modbus Device";
  if (connectionType.value === "tdengine") return "TDengine Source";
  return activeDatabase.value?.label || "数据库连接";
});

// 动态加载表单组件
const formComponent = computed(() => {
  if (relationalSourceTypes.includes(connectionType.value)) {
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

const configSignature = computed(() => {
  return JSON.stringify({
    step: step.value,
    type: connectionType.value,
    dbType: dbType.value,
    data: formData.value,
  });
});

const isDirty = computed(
  () =>
    visible.value &&
    Boolean(initialDialogSignature.value) &&
    configSignature.value !== initialDialogSignature.value,
);

const isTestStale = computed(() => {
  return (
    Boolean(lastTestSignature.value) &&
    lastTestSignature.value !== configSignature.value &&
    testResult.value.status === "success"
  );
});

const testState = computed(() => {
  if (testing.value) {
    return {
      status: "testing",
      icon: markRaw(IconTablerLoader2),
      title: "正在测试",
      message: "正在向 data_service 发起一次短时连接测试。",
      detail: "",
    };
  }

  const iconMap = {
    idle: markRaw(IconTablerPlugConnected),
    success: markRaw(IconTablerCircleCheck),
    error: markRaw(IconTablerAlertTriangle),
  };

  return {
    ...testResult.value,
    icon: iconMap[testResult.value.status] || iconMap.idle,
  };
});

const summaryRows = computed(() => {
  const data = formData.value || {};
  const rows = [
    { label: "类型", value: activeSource.value.label },
    { label: "名称", value: data.name || "未填写" },
  ];

  if (relationalSourceTypes.includes(connectionType.value)) {
    rows.push(
      { label: "数据库", value: activeDatabase.value?.label || dbType.value },
      { label: "地址", value: formatEndpoint(data.host, data.port) },
      { label: "库名", value: data.database || "未填写" },
      { label: "账号", value: data.username || "未填写" },
      {
        label: "超时",
        value: formatTimeout(data.queryTimeout || data.timeout),
      },
    );
  } else if (connectionType.value === "mqtt") {
    rows.push(
      { label: "协议", value: data.protocol || "mqtt" },
      { label: "Broker", value: formatEndpoint(data.brokerUrl, data.port) },
      { label: "Client ID", value: data.clientId || "自动生成" },
      { label: "QoS", value: String(data.qos ?? 0) },
      { label: "认证", value: data.username ? "用户名/密码" : "匿名" },
    );
  } else if (connectionType.value === "kafka") {
    rows.push(
      { label: "Broker", value: data.brokers || "未填写" },
      { label: "Topic", value: data.topic || "未填写" },
      { label: "Group", value: data.consumerGroup || "未填写" },
      { label: "Offset", value: data.startPosition || "latest" },
    );
  } else if (connectionType.value === "http") {
    rows.push(
      { label: "方法", value: data.method || "GET" },
      { label: "URL", value: data.baseUrl || "未填写" },
      { label: "超时", value: formatTimeout(data.timeoutMs) },
    );
  } else if (connectionType.value === "websocket") {
    rows.push(
      { label: "URL", value: data.url || "未填写" },
      { label: "Topic", value: data.topic || "可选" },
      { label: "心跳", value: formatTimeout(data.heartbeatIntervalMs) },
    );
  } else if (connectionType.value === "redis") {
    rows.push(
      { label: "模式", value: data.mode || "standalone" },
      { label: "地址", value: data.address || "未填写" },
      { label: "DB", value: String(data.db ?? 0) },
      { label: "Key", value: data.keyPattern || "*" },
    );
  } else if (connectionType.value === "opcua") {
    rows.push(
      { label: "地址", value: formatEndpoint(data.ip, data.port) },
      {
        label: "安全",
        value: `${data.securityPolicy || "None"} / ${data.securityMode || "none"}`,
      },
      {
        label: "认证",
        value: data.authType === "username_password" ? "用户名/密码" : "匿名",
      },
      { label: "采样", value: formatTimeout(data.samplingMs) },
    );
  } else if (connectionType.value === "s7") {
    rows.push(
      { label: "地址", value: formatEndpoint(data.ip, data.port) },
      { label: "Rack", value: String(data.rack ?? 0) },
      { label: "Slot", value: String(data.slot ?? 1) },
      { label: "周期", value: formatTimeout(data.pollIntervalMs) },
    );
  } else if (connectionType.value === "modbus") {
    rows.push(
      { label: "模式", value: (data.mode || "tcp").toUpperCase() },
      {
        label: "地址",
        value:
          data.mode === "rtu" ? "串口 RTU" : formatEndpoint(data.ip, data.port),
      },
      { label: "站号", value: String(data.slaveId ?? 1) },
      {
        label: "范围",
        value: `${data.startAddress ?? 0} / ${data.quantity ?? 1}`,
      },
    );
  } else if (connectionType.value === "tdengine") {
    rows.push(
      { label: "地址", value: formatEndpoint(data.ip, data.port) },
      { label: "库名", value: data.database || "未填写" },
      { label: "账号", value: data.username || "root" },
      { label: "时区", value: data.timezone || "未填写" },
    );
  }

  return rows;
});

const checklist = computed(() => {
  const data = formData.value || {};
  const hasName = Boolean(data.name);
  const hasEndpoint =
    connectionType.value === "mqtt"
      ? Boolean(data.brokerUrl && data.port)
      : connectionType.value === "kafka"
        ? Boolean(data.brokers)
        : connectionType.value === "http"
          ? Boolean(data.baseUrl)
          : connectionType.value === "websocket"
            ? Boolean(data.url)
            : connectionType.value === "redis"
              ? Boolean(data.address)
              : connectionType.value === "opcua"
                ? Boolean(data.ip && data.port)
                : connectionType.value === "modbus" && data.mode === "rtu"
                  ? Boolean(data.serialConfigText)
                  : connectionType.value === "tdengine"
                    ? Boolean(data.ip && data.port)
                    : relationalSourceTypes.includes(connectionType.value)
                      ? Boolean(data.host && data.port)
                      : Boolean(data.ip && data.port);
  const hasTarget = ["mqtt", "http", "websocket", "redis", "opcua"].includes(
    connectionType.value,
  )
    ? true
    : connectionType.value === "kafka"
      ? Boolean(data.topic && data.consumerGroup)
      : connectionType.value === "s7"
        ? data.rack !== undefined && data.slot !== undefined
        : connectionType.value === "modbus"
          ? Boolean(data.quantity)
          : relationalSourceTypes.includes(connectionType.value) ||
              connectionType.value === "tdengine"
            ? Boolean(data.database)
            : Boolean(data.database);

  return [
    { label: "基础名称已填写", ready: hasName },
    { label: "网络地址已填写", ready: hasEndpoint },
    { label: "目标资源已明确", ready: hasTarget },
    {
      label: previewProtocolTypes.includes(connectionType.value)
        ? "保存后可短时预览"
        : industrialProtocolTypes.includes(connectionType.value)
          ? "保存为节点侧运行配置"
          : "连接测试可选完成",
      ready:
        previewProtocolTypes.includes(connectionType.value) ||
        industrialProtocolTypes.includes(connectionType.value) ||
        (testResult.value.status === "success" && !isTestStale.value),
    },
  ];
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
        connectionType.value = dbType.value;
        formData.value = {
          name: newConnection.name,
          ...newConnection.relationalConfig,
        };
      } else if (newConnection.type === "mqtt" && newConnection.mqttConfig) {
        formData.value = {
          name: newConnection.name,
          ...newConnection.mqttConfig,
        };
      } else {
        formData.value = normalizeProtocolFormData(newConnection.type, {
          name: newConnection.name,
          ...(newConnection.config || {}),
        });
      }
      resetTestState();
    }
  },
  { immediate: true },
);

// 监听对话框打开
watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen && props.mode === "create") {
      // create 打开时回到 step 1 并重置
      step.value = 1;
      connectionType.value = "mysql";
      dbType.value = "mysql";
      formData.value = getDefaultConfig("mysql");
      resetTestState();
      initialDialogSignature.value = configSignature.value;
    } else if (isOpen && props.mode === "edit") {
      // edit 模式直接进入 step 2
      step.value = 2;
      initialDialogSignature.value = configSignature.value;
    }
  },
);

const selectSource = (value) => {
  if (props.mode === "edit" || connectionType.value === value) return;
  connectionType.value = value;
  // 连接类型变化时重置表单
  if (relationalSourceTypes.includes(connectionType.value)) {
    dbType.value = connectionType.value;
    formData.value = getDefaultConfig(dbType.value);
  } else if (connectionType.value === "mqtt") {
    formData.value = getDefaultConfig("mqtt");
  } else {
    formData.value = getProtocolDefaultConfig(connectionType.value);
  }
  resetTestState();
};

const resetCurrentForm = () => {
  if (relationalSourceTypes.includes(connectionType.value)) {
    dbType.value = connectionType.value;
    formData.value = getDefaultConfig(dbType.value);
  } else {
    formData.value =
      connectionType.value === "mqtt"
        ? getDefaultConfig("mqtt")
        : getProtocolDefaultConfig(connectionType.value);
  }
  resetTestState();
};

const resetTestState = () => {
  lastTestSignature.value = "";
  testResult.value = {
    status: "idle",
    title: "尚未测试",
    message: "填写连接参数后，可以先验证网络、认证与基础协议是否可用。",
    detail: "",
    durationMs: 0,
  };
};

const handleValidate = (valid, data) => {
  // 表单验证回调
  console.log("表单验证:", valid, data);
};

const validateCurrentForm = async () => {
  const validator = formComponent.value ? formRef.value : protocolFormRef.value;
  if (!validator) return false;

  try {
    const valid = await validator.validate();
    if (!valid) return false;
    buildConnectionPayload();
    return true;
  } catch (error) {
    if (error instanceof Error) {
      ElMessage.warning(error.message);
    }
    return false;
  }
};

const buildConnectionPayload = () => {
  const config = { ...formData.value };
  const name = config.name;
  delete config.name;

  if (relationalSourceTypes.includes(connectionType.value)) {
    config.dbType = dbType.value;
  } else if (specializedProtocolTypes.includes(connectionType.value)) {
    normalizeProtocolSubmitConfig(connectionType.value, config);
  }

  return {
    name,
    type: relationalSourceTypes.includes(connectionType.value)
      ? "relational"
      : connectionType.value,
    config,
  };
};

const handleProtocolPreviewTest = async () => {
  testing.value = true;
  const startAt = performance.now();
  try {
    const response = await dataAPI.previewProtocol(
      props.projectId,
      props.connection.id,
      {
        limit: 5,
        timeoutMs: 5000,
        options: buildProtocolPreviewOptions(),
      },
    );
    const payload = response?.data || {};
    const durationMs =
      payload.durationMs ?? Math.round(performance.now() - startAt);
    lastTestSignature.value = configSignature.value;
    testResult.value = {
      status: payload.status === "failed" ? "error" : "success",
      title: payload.status === "failed" ? "预览失败" : "预览完成",
      message: `读取到 ${payload.samples?.length || 0} 条样本，耗时 ${durationMs}ms。`,
      detail: payload.diagnostics?.error || "",
      durationMs,
    };
    ElMessage.success("短时预览完成");
  } catch (error) {
    const durationMs = Math.round(performance.now() - startAt);
    testResult.value = {
      status: "error",
      title: "预览失败",
      message: "短时预览未完成，请检查协议配置或网络连通性。",
      detail: getApiErrorMessage(error, "短时预览失败"),
      durationMs,
    };
    ElMessage.error(getApiErrorMessage(error, "短时预览失败"));
  } finally {
    testing.value = false;
  }
};

const handleMqttConnectionTest = async () => {
  testing.value = true;
  const startAt = performance.now();
  try {
    const payload = buildConnectionPayload();
    await dataAPI.testMqttConnection(props.projectId, {
      name: payload.name,
      ...payload.config,
    });

    const durationMs = Math.round(performance.now() - startAt);
    lastTestSignature.value = configSignature.value;
    testResult.value = {
      status: "success",
      title: "测试通过",
      message: `MQTT Broker 连接验证通过，耗时 ${durationMs}ms。`,
      detail: "",
      durationMs,
    };
    ElMessage.success(t("query.connectionTestSuccess"));
  } catch (error) {
    const durationMs = Math.round(performance.now() - startAt);
    testResult.value = {
      status: "error",
      title: "测试失败",
      message: "MQTT Broker 连接未通过验证，请检查地址、端口或认证配置。",
      detail: getApiErrorMessage(error, "MQTT 连接测试失败"),
      durationMs,
    };
    ElMessage.error(getApiErrorMessage(error, "MQTT 连接测试失败"));
  } finally {
    testing.value = false;
  }
};

const handleTest = async () => {
  const valid = await validateCurrentForm();
  if (!valid) {
    ElMessage.warning(t("connection.incompleteInfoWarning"));
    return;
  }

  if (!props.projectId) {
    ElMessage.warning("缺少工程上下文，无法测试连接");
    return;
  }

  if (connectionType.value === "mqtt") {
    await handleMqttConnectionTest();
    return;
  }

  if (!relationalSourceTypes.includes(connectionType.value)) {
    if (["s7", "tdengine"].includes(connectionType.value)) {
      testResult.value = {
        status: "idle",
        title: "节点侧运行配置",
        message:
          "工业协议已在开发态保存配置契约，真实连通与采集由节点侧工业协议运行器执行。",
        detail: "",
        durationMs: 0,
      };
      ElMessage.info("工业协议保存后进入节点侧运行联调");
      return;
    }

    if (connectionType.value === "kafka") {
      if (props.mode === "edit" && props.connection?.id) {
        await handleProtocolPreviewTest();
        return;
      }
      testResult.value = {
        status: "idle",
        title: "保存后预览",
        message: "Kafka 需要先保存配置，再通过统一协议预览读取真实样本。",
        detail: "",
        durationMs: 0,
      };
      ElMessage.info("保存 Kafka 接入源后可执行短时真实预览");
      return;
    }
  }

  testing.value = true;
  const startAt = performance.now();
  try {
    const payload = buildConnectionPayload();
    const response = await dataAPI.testConnection(props.projectId, {
      type: payload.type,
      config: payload.config,
    });
    const result = response?.data || response || {};

    const durationMs = Math.round(performance.now() - startAt);
    lastTestSignature.value = configSignature.value;
    testResult.value = {
      status: "success",
      title: "测试通过",
      message: result.message || `data_service 已完成短时连接验证，耗时 ${durationMs}ms。`,
      detail: result.detail || "",
      durationMs,
    };
    ElMessage.success(t("query.connectionTestSuccess"));
  } catch (error) {
    const durationMs = Math.round(performance.now() - startAt);
    testResult.value = {
      status: "error",
      title: "测试失败",
      message: "连接参数未通过验证，请检查网络、认证或协议配置。",
      detail: getApiErrorMessage(error, "连接测试失败"),
      durationMs,
    };
    ElMessage.error(getApiErrorMessage(error, "连接测试失败"));
  } finally {
    testing.value = false;
  }
};

const handleSubmit = async () => {
  const valid = await validateCurrentForm();
  if (!valid) {
    ElMessage.warning(t("connection.incompleteInfoWarning"));
    return;
  }

  submitting.value = true;
  try {
    const payload = buildConnectionPayload();

    emit("submit", {
      name: payload.name,
      type: payload.type,
      config: payload.config,
    });
    initialDialogSignature.value = configSignature.value;
  } finally {
    submitting.value = false;
  }
};

const requestClose = () => {
  visible.value = false;
};

const handleClosed = () => {
  initialDialogSignature.value = "";
  // 清空表单
  if (formRef.value) {
    formRef.value.clearValidate();
  }
  if (protocolFormRef.value) {
    protocolFormRef.value.clearValidate();
  }
};

const getProtocolDefaultConfig = (type) => {
  const defaults = {
    kafka: {
      name: "",
      brokers: "",
      topic: "",
      consumerGroup: "datacenter-preview",
      startPosition: "latest",
      optionsText: "{}",
    },
    http: {
      name: "",
      baseUrl: "",
      method: "GET",
      headersText: "{}",
      timeoutMs: 5000,
      bodyTemplateText: "",
    },
    websocket: {
      name: "",
      url: "",
      topic: "",
      headersText: "{}",
      heartbeatIntervalMs: 30000,
      subscribeMessage: "",
    },
    redis: {
      name: "",
      address: "127.0.0.1:6379",
      db: 0,
      username: "",
      password: "",
      keyPattern: "*",
      mode: "standalone",
      masterName: "",
    },
    opcua: {
      name: "",
      ip: "127.0.0.1",
      port: 4840,
      securityPolicy: "None",
      securityMode: "none",
      authType: "anonymous",
      username: "",
      password: "",
      samplingMs: 1000,
      optionsText: "{}",
    },
    s7: {
      name: "",
      ip: "",
      port: 102,
      rack: 0,
      slot: 1,
      pollIntervalMs: 1000,
      optionsText: "{}",
    },
    modbus: {
      name: "",
      mode: "tcp",
      ip: "",
      port: 502,
      serialConfigText:
        '{"port":"COM3","baudRate":9600,"dataBits":8,"parity":"N","stopBits":1}',
      slaveId: 1,
      startAddress: 0,
      quantity: 1,
      pollIntervalMs: 1000,
      optionsText: "{}",
    },
    tdengine: {
      name: "",
      ip: "127.0.0.1",
      port: 6030,
      database: "",
      username: "root",
      password: "taosdata",
      timezone: "Asia/Shanghai",
      optionsText: "{}",
    },
  };
  return { ...(defaults[type] || {}) };
};

const normalizeProtocolFormData = (type, config) => {
  const data = { ...getProtocolDefaultConfig(type), ...config };
  if (type === "opcua" && config.endpoint) {
    Object.assign(data, parseOpcuaEndpoint(config.endpoint));
  }
  if (["s7", "modbus"].includes(type) && config.host) {
    data.ip = config.host;
  }
  if (type === "tdengine" && config.dsn) {
    Object.assign(data, parseTdengineDsn(config.dsn));
  }
  if (data.headers && typeof data.headers === "object") {
    data.headersText = JSON.stringify(data.headers, null, 2);
  }
  if (data.options && typeof data.options === "object") {
    data.optionsText = JSON.stringify(data.options, null, 2);
    data.masterName = data.options.masterName || data.masterName;
  }
  if (data.serialConfig && typeof data.serialConfig === "object") {
    data.serialConfigText = JSON.stringify(data.serialConfig, null, 2);
  }
  if (data.bodyTemplate && typeof data.bodyTemplate === "object") {
    data.bodyTemplateText = JSON.stringify(data.bodyTemplate, null, 2);
  }
  return data;
};

const normalizeProtocolSubmitConfig = (type, config) => {
  if (type === "kafka") {
    config.options = parseOptionalJsonObject(config.optionsText, "扩展参数");
    delete config.optionsText;
    return;
  }
  if (type === "http") {
    config.method = (config.method || "GET").toUpperCase();
    config.headers = parseOptionalJsonObject(config.headersText, "请求头");
    if (String(config.bodyTemplateText || "").trim()) {
      config.bodyTemplate = parseOptionalJsonObject(
        config.bodyTemplateText,
        "请求体模板",
      );
    }
    delete config.headersText;
    delete config.bodyTemplateText;
    return;
  }
  if (type === "websocket") {
    config.headers = parseOptionalJsonObject(config.headersText, "握手 Header");
    delete config.headersText;
    delete config.subscribeMessage;
    if (!config.topic) delete config.topic;
    return;
  }
  if (type === "redis") {
    const options = {};
    if (config.masterName) {
      options.masterName = config.masterName;
    }
    config.options = options;
    delete config.masterName;
    return;
  }
  if (type === "opcua") {
    config.options = parseOptionalJsonObject(config.optionsText, "扩展参数");
    config.endpoint = buildOpcuaEndpoint(config.ip, config.port);
    if (config.authType !== "username_password") {
      delete config.username;
      delete config.password;
    }
    delete config.ip;
    delete config.port;
    delete config.optionsText;
    return;
  }
  if (type === "s7") {
    config.options = parseOptionalJsonObject(config.optionsText, "扩展参数");
    config.host = config.ip;
    delete config.ip;
    delete config.optionsText;
    return;
  }
  if (type === "modbus") {
    config.options = parseOptionalJsonObject(config.optionsText, "扩展参数");
    if (config.mode === "rtu") {
      config.serialConfig = parseOptionalJsonObject(
        config.serialConfigText,
        "串口配置",
      );
      delete config.ip;
      delete config.port;
    } else {
      config.host = config.ip;
      delete config.ip;
      delete config.serialConfig;
    }
    delete config.serialConfigText;
    delete config.optionsText;
    return;
  }
  if (type === "tdengine") {
    config.options = parseOptionalJsonObject(config.optionsText, "扩展参数");
    config.dsn = buildTdengineDsn(config);
    delete config.ip;
    delete config.port;
    delete config.username;
    delete config.password;
    if (!config.timezone) delete config.timezone;
    delete config.optionsText;
  }
};

const buildProtocolPreviewOptions = () => {
  if (connectionType.value === "websocket" && formData.value.subscribeMessage) {
    return {
      subscribeMessage: parseJsonOrString(formData.value.subscribeMessage),
    };
  }
  if (connectionType.value === "redis" && formData.value.masterName) {
    return { masterName: formData.value.masterName };
  }
  return {};
};

const buildOpcuaEndpoint = (ip, port) => {
  const safeIp = String(ip || "").trim();
  const safePort = Number(port) || 4840;
  return `opc.tcp://${safeIp}:${safePort}`;
};

const parseOpcuaEndpoint = (endpoint) => {
  const text = String(endpoint || "").trim();
  const match = text.match(/^opc\.tcp:\/\/([^:/]+)(?::(\d+))?/i);
  if (!match) return {};
  return {
    ip: match[1],
    port: match[2] ? Number(match[2]) : 4840,
  };
};

const buildTdengineDsn = (config) => {
  const username = String(config.username || "root").trim();
  const password = String(config.password || "taosdata").trim();
  const ip = String(config.ip || "").trim();
  const port = Number(config.port) || 6030;
  return `${username}:${password}@tcp(${ip}:${port})/`;
};

const parseTdengineDsn = (dsn) => {
  const text = String(dsn || "").trim();
  const match = text.match(/^(.*?):(.*?)@tcp\(([^:)]+)(?::(\d+))?\)\//i);
  if (!match) return {};
  return {
    username: match[1] || "root",
    password: match[2] || "taosdata",
    ip: match[3],
    port: match[4] ? Number(match[4]) : 6030,
  };
};

const parseOptionalJsonObject = (value, label) => {
  const text = String(value || "").trim();
  if (!text) return {};
  try {
    const parsed = JSON.parse(text);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed;
    }
  } catch {
    // 统一在下方返回带字段名的可读错误。
  }
  throw new Error(`${label} 必须是 JSON 对象`);
};

const parseJsonOrString = (value) => {
  const text = String(value || "").trim();
  if (!text) return "";
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
};

const formatEndpoint = (host, port) => {
  const safeHost = host || "未填写";
  return port ? `${safeHost}:${port}` : safeHost;
};

const formatTimeout = (timeout) => {
  if (!timeout) return "未设置";
  return `${timeout}ms`;
};
</script>

<style scoped>
/* ── Step 1：选择接入类型 ── */
.connection-dialog__step1 {
  padding: 24px 20px 8px;
}

.connection-dialog__step1-header {
  margin-bottom: 20px;
}

.connection-dialog__step1-title {
  margin: 0 0 4px;
  color: var(--dc-text);
  font-size: 18px;
  font-weight: 700;
}

.connection-dialog__step1-sub {
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.connection-dialog__step1-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}

.connection-dialog__step1-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  cursor: pointer;
  text-align: left;
  transition: background 0.16s ease, border-color 0.16s ease, color 0.16s ease;
}

.connection-dialog__step1-card:hover,
.connection-dialog__step1-card.is-active {
  border-color: color-mix(in oklch, var(--dc-primary) 36%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.connection-dialog__step1-icon {
  width: 28px;
  height: 28px;
  color: var(--dc-text-muted);
  flex-shrink: 0;
}

.connection-dialog__step1-card:hover .connection-dialog__step1-icon,
.connection-dialog__step1-card.is-active .connection-dialog__step1-icon {
  color: var(--dc-primary);
}

.connection-dialog__step1-card strong {
  display: block;
  color: inherit;
  font-size: 14px;
  font-weight: 700;
}

.connection-dialog__step1-desc {
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.connection-dialog__step1-card:hover .connection-dialog__step1-desc,
.connection-dialog__step1-card.is-active .connection-dialog__step1-desc {
  color: color-mix(in oklch, var(--dc-primary) 72%, transparent);
}

/* ── Step 2：两栏布局（表单 + inspector） ── */
.connection-dialog__body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 232px;
  min-height: 520px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  overflow: hidden;
}

/* 顶部协议徽标 */
.connection-dialog__heading-main {
  min-width: 0;
}

.connection-dialog__heading-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 10px;
  margin-bottom: 6px;
  border-radius: 999px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 700;
}

.connection-dialog__heading-chip-icon {
  width: 14px;
  height: 14px;
}

.connection-dialog__rail,
.connection-dialog__inspector {
  background: var(--dc-surface-muted);
}

.connection-dialog__rail {
  padding: 14px;
  border-right: 1px solid var(--dc-border);
  overflow-y: auto;
}

.connection-dialog__inspector {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-left: 1px solid var(--dc-border);
}

.connection-dialog__section + .connection-dialog__section {
  margin-top: 18px;
}

.connection-dialog__section-title {
  margin-bottom: 8px;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 650;
  letter-spacing: 0;
}

.connection-dialog__source,
.connection-dialog__db {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-md);
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  text-align: left;
  transition:
    background 0.16s ease,
    border-color 0.16s ease,
    color 0.16s ease;
}

.connection-dialog__source-wrap {
  display: block;
}

.connection-dialog__source {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  min-height: 38px;
  padding: 10px;
}

.connection-dialog__source-wrap + .connection-dialog__source-wrap,
.connection-dialog__db + .connection-dialog__db {
  margin-top: 6px;
}

.connection-dialog__source strong {
  display: block;
  color: inherit;
  font-size: 13px;
  font-weight: 650;
}

.connection-dialog__source-icon,
.connection-dialog__db-icon {
  width: 16px;
  height: 16px;
  color: var(--dc-text-muted);
}

.connection-dialog__source-icon {
  margin-top: 1px;
}

.connection-dialog__db {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 8px 10px;
  font-size: 12px;
}

.connection-dialog__source:hover,
.connection-dialog__db:hover,
.connection-dialog__source.is-active,
.connection-dialog__db.is-active {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, transparent);
  background: var(--dc-primary-soft);
  color: var(--dc-text);
}

.connection-dialog__source:disabled,
.connection-dialog__db:disabled {
  cursor: default;
  opacity: 0.72;
}

.connection-dialog__form-panel {
  min-width: 0;
  max-height: 620px;
  padding: 18px 20px;
  overflow: auto;
}

.connection-dialog__form-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.connection-dialog__eyebrow {
  margin-bottom: 4px;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 650;
}

.connection-dialog__form-heading h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 16px;
  font-weight: 650;
}

.connection-dialog__summary,
.connection-dialog__test,
.connection-dialog__checklist {
  padding-bottom: 12px;
  border-bottom: 1px solid var(--dc-border);
}

.connection-dialog__checklist {
  border-bottom: 0;
}

.connection-dialog__summary dl {
  display: grid;
  grid-template-columns: 58px minmax(0, 1fr);
  gap: 8px 10px;
  margin: 0;
}

.connection-dialog__summary dt {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.connection-dialog__summary dd {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connection-dialog__test-state {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr);
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
}

.connection-dialog__test-state strong {
  display: block;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 650;
}

.connection-dialog__test-state p {
  margin: 3px 0 0;
  color: var(--dc-text-secondary);
  font-size: 11px;
  line-height: 1.45;
}

.connection-dialog__test-icon {
  width: 16px;
  height: 16px;
  margin-top: 2px;
  color: var(--dc-text-muted);
}

.connection-dialog__test-state.is-success {
  border-color: color-mix(in oklch, var(--dc-success) 36%, var(--dc-border));
  background: color-mix(
    in oklch,
    var(--dc-success) 8%,
    var(--dc-surface-raised)
  );
}

.connection-dialog__test-state.is-success .connection-dialog__test-icon {
  color: var(--dc-success);
}

.connection-dialog__test-state.is-error {
  border-color: color-mix(in oklch, var(--dc-danger) 34%, var(--dc-border));
  background: color-mix(
    in oklch,
    var(--dc-danger) 7%,
    var(--dc-surface-raised)
  );
}

.connection-dialog__test-state.is-error .connection-dialog__test-icon {
  color: var(--dc-danger);
}

.connection-dialog__test-state.is-testing .connection-dialog__test-icon {
  color: var(--dc-primary);
  animation: connection-dialog-spin 0.9s linear infinite;
}

.connection-dialog__test-detail,
.connection-dialog__test-stale {
  margin-top: 8px;
  color: var(--dc-text-secondary);
  font-size: 11px;
  line-height: 1.45;
}

.connection-dialog__test-detail {
  max-height: 64px;
  overflow: auto;
  padding: 8px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.connection-dialog__test-stale {
  color: var(--dc-warning);
}

.connection-dialog__check-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 26px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.connection-dialog__check-item.is-ready {
  color: var(--dc-text-secondary);
}

.connection-dialog__check-icon {
  width: 14px;
  height: 14px;
  color: currentColor;
}

.connection-dialog__check-item.is-ready .connection-dialog__check-icon {
  color: var(--dc-success);
}

.connection-dialog__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
  width: 100%;
}

.connection-dialog__footer-actions {
  display: flex;
  flex-shrink: 0;
  gap: 8px;
}

:global(.connection-dialog .el-dialog__header) {
  height: 0;
  padding: 0;
  margin-right: 0;
  border-bottom: 0;
}

:global(.connection-dialog .el-dialog__headerbtn) {
  top: 9px;
  right: 10px;
  z-index: 2;
}

:global(.connection-dialog .el-dialog__body) {
  padding: 24px 20px 16px;
}

:global(.connection-dialog .el-dialog__footer) {
  padding: 12px 20px 16px;
  border-top: 0;
}

:deep(.el-form) {
  --el-text-color-regular: var(--dc-text-secondary);
}

:deep(.el-form-item__label) {
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.25;
}

:deep(.el-form--label-top .el-form-item__label) {
  margin-bottom: 6px;
}

:deep(.el-input__wrapper),
:deep(.el-input-group__append),
:deep(.el-input-group__prepend),
:deep(.el-select__wrapper),
:deep(.el-textarea__inner) {
  border-radius: var(--dc-radius-sm);
}

:deep(.el-input-group__append),
:deep(.el-input-group__prepend) {
  padding: 0 10px;
  color: var(--dc-text-muted);
  font-size: 12px;
  background: var(--dc-surface-muted);
  box-shadow: 0 0 0 1px var(--dc-border) inset;
}

:deep(.mqtt-connection-form) {
  padding: 0;
}

:deep(.mqtt-connection-form .form-section) {
  padding: 14px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
}

:deep(.mqtt-connection-form .form-section-title) {
  color: var(--dc-text);
  border-bottom-color: var(--dc-border);
}

.connection-dialog__protocol-form {
  max-width: 620px;
}

.connection-dialog__form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
  width: 100%;
}

.connection-dialog__form-grid > .el-form-item {
  min-width: 0;
}

@keyframes connection-dialog-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 760px) {
  .connection-dialog__body {
    grid-template-columns: 1fr;
  }

  .connection-dialog__inspector {
    grid-column: 1 / -1;
    border-top: 1px solid var(--dc-border);
    border-left: 0;
  }

  .connection-dialog__footer {
    align-items: stretch;
    flex-direction: column;
  }

  .connection-dialog__footer-actions {
    justify-content: flex-end;
  }

  .connection-dialog__form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
