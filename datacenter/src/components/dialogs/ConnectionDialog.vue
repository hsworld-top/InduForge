<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    class="connection-dialog"
    width="920px"
    :dirty="isDialogDirty"
    :close-on-click-modal="canCloseByModalClick"
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
        <div class="connection-dialog__type-section">
          <div class="connection-dialog__section-title">内置运行库</div>
          <div class="connection-dialog__step1-grid">
            <button
              v-for="source in builtinSourceOptions"
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
        <div class="connection-dialog__type-section">
          <div class="connection-dialog__section-title">外部数据源</div>
          <div class="connection-dialog__step1-grid">
            <button
              v-for="source in externalSourceOptions"
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
            <el-button v-if="mode === 'create'" size="small" text @click="resetCurrentForm">
              重置默认值
            </el-button>
          </div>

          <component
            v-if="formComponent"
            :is="formComponent"
            ref="formRef"
            v-model="formData"
            :mode="mode"
            :saved-password-configured="showSavedPassword"
            :saved-tls-secrets="savedTlsSecrets"
            :loading-saved-password="revealingPassword"
            @validate="handleValidate"
            @reveal-saved-password="revealSavedPassword"
          />

          <el-form
            v-else
            ref="protocolFormRef"
            :model="formData"
            :rules="protocolRules"
            label-position="top"
            class="connection-dialog__protocol-form"
          >
            <el-form-item :label="isBuiltinStoreSelected ? '名称' : '连接名称'" prop="name">
              <el-input v-model="formData.name" :placeholder="connectionNamePlaceholder" />
            </el-form-item>

            <template v-if="connectionType === 'kafka'">
              <div class="connection-dialog__form-section">
                <div class="connection-dialog__form-section-title">基础信息</div>
                <el-form-item prop="brokers">
                  <template #label>
                    <span class="connection-dialog__field-label">
                      服务器地址
                      <el-tooltip
                        content="保持 IP:端口 格式；支持配置多个地址，用英文逗号分隔。"
                        placement="top"
                      >
                        <IconTablerHelpCircle class="connection-dialog__field-help" />
                      </el-tooltip>
                    </span>
                  </template>
                  <el-input
                    v-model="formData.brokers"
                    placeholder="127.0.0.1:9092,127.0.0.2:9092"
                  />
                  <p class="connection-dialog__field-tip">支持配置多个地址，用英文逗号分隔。</p>
                </el-form-item>
              </div>

              <div class="connection-dialog__form-section">
                <div class="connection-dialog__form-section-title">认证信息</div>
                <div class="connection-dialog__form-grid">
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        安全协议
                        <el-tooltip
                          content="本地测试通常选择 PLAINTEXT；启用 TLS 或 SASL 时按 Broker 要求选择。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-select v-model="formData.securityProtocol" class="w-full">
                      <el-option
                        v-for="option in kafkaSecurityProtocolOptions"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      />
                    </el-select>
                  </el-form-item>
                  <el-form-item v-if="isKafkaSaslEnabled">
                    <template #label>
                      <span class="connection-dialog__field-label">
                        SASL 机制
                        <el-tooltip content="需与 Broker 开启的认证机制一致。" placement="top">
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-select v-model="formData.saslMechanism" class="w-full">
                      <el-option
                        v-for="option in kafkaSaslMechanismOptions"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      />
                    </el-select>
                  </el-form-item>
                </div>
                <div v-if="isKafkaSaslEnabled" class="connection-dialog__form-grid">
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        用户名
                        <el-tooltip content="Kafka SASL 认证用户名。" placement="top">
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input
                      v-model="formData.username"
                      placeholder="请输入 SASL 用户名"
                      clearable
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        密码
                        <el-tooltip
                          content="Kafka SASL 认证密码，保存后按接入源配置加密/脱敏策略处理。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input
                      v-model="formData.password"
                      type="password"
                      show-password
                      clearable
                      :placeholder="
                        mode === 'edit' && connection?.secretStatus?.['option.password']
                          ? '已保存，留空保持不变'
                          : '请输入 SASL 密码'
                      "
                    />
                  </el-form-item>
                </div>
              </div>

              <div class="connection-dialog__form-section">
                <div class="connection-dialog__form-section-title">连接参数</div>
                <el-form-item>
                  <template #label>
                    <span class="connection-dialog__field-label">
                      客户端 ID
                      <el-tooltip
                        content="用于 Broker 日志和监控识别当前接入源；留空时系统会自动生成。"
                        placement="top"
                      >
                        <IconTablerHelpCircle class="connection-dialog__field-help" />
                      </el-tooltip>
                    </span>
                  </template>
                  <el-input v-model="formData.clientId" placeholder="自动生成" clearable>
                    <template #append>
                      <el-button @click="generateKafkaClientId" icon="Refresh">生成</el-button>
                    </template>
                  </el-input>
                </el-form-item>
                <div class="connection-dialog__form-grid">
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        连接超时
                        <el-tooltip
                          content="建立 TCP/TLS/SASL 连接的最大等待时间，单位毫秒。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number
                      v-model="formData.dialTimeoutMs"
                      :min="1000"
                      :step="1000"
                      class="w-full"
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        请求超时
                        <el-tooltip
                          content="测试连接和读取元信息时的最大等待时间，单位毫秒。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number
                      v-model="formData.requestTimeoutMs"
                      :min="1000"
                      :step="1000"
                      class="w-full"
                    />
                  </el-form-item>
                </div>
              </div>

              <el-collapse v-model="kafkaActiveCollapse" class="connection-dialog__collapse">
                <el-collapse-item title="SSL/TLS 配置" name="ssl">
                  <el-form-item label="CA 证书">
                    <el-input
                      v-model="formData.sslConfig.ca"
                      type="textarea"
                      :rows="4"
                      placeholder="PEM，可选"
                    />
                  </el-form-item>
                  <el-form-item label="客户端证书">
                    <el-input
                      v-model="formData.sslConfig.cert"
                      type="textarea"
                      :rows="4"
                      placeholder="PEM，可选"
                    />
                  </el-form-item>
                  <el-form-item label="客户端私钥">
                    <el-input
                      v-model="formData.sslConfig.key"
                      type="textarea"
                      :rows="4"
                      placeholder="PEM，可选"
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        验证服务器证书
                        <el-tooltip
                          content="生产环境建议开启；使用自签名证书调试时可按需关闭。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-switch v-model="formData.sslConfig.rejectUnauthorized" />
                    <span class="connection-dialog__field-inline-tip">
                      {{
                        formData.sslConfig.rejectUnauthorized
                          ? '校验证书链与主机名'
                          : '跳过证书校验'
                      }}
                    </span>
                  </el-form-item>
                </el-collapse-item>
              </el-collapse>
              <el-form-item
                v-if="mode === 'edit' && Object.keys(connection?.secretStatus || {}).length"
                label="密钥操作"
              >
                <el-checkbox v-model="formData.clearSecrets"
                  >清除已保存的 Kafka 密码和证书密钥</el-checkbox
                >
              </el-form-item>
            </template>

            <template v-else-if="connectionType === 'redis'">
              <el-form-item label="连接方式">
                <el-segmented v-model="formData.mode" :options="redisModeOptions" />
              </el-form-item>
              <el-form-item label="节点地址" prop="address">
                <el-input
                  v-model="formData.address"
                  placeholder="127.0.0.1:6379；多个节点用英文逗号分隔"
                />
              </el-form-item>
              <div class="connection-dialog__form-grid">
                <el-form-item label="数据库编号">
                  <el-input v-model.number="formData.db" inputmode="numeric" placeholder="0" />
                </el-form-item>
                <el-form-item v-if="formData.mode === 'sentinel'" label="主节点名称">
                  <el-input v-model="formData.masterName" placeholder="mymaster" />
                </el-form-item>
                <el-form-item label="Key 筛选规则">
                  <el-input v-model="formData.keyPattern" placeholder="* 表示全部 Key" />
                </el-form-item>
              </div>
              <div class="connection-dialog__form-grid">
                <el-form-item label="用户名（可选）">
                  <el-input v-model="formData.username" placeholder="未配置时使用默认用户" />
                </el-form-item>
                <el-form-item label="密码（可选）">
                  <SavedPasswordInput
                    v-model="formData.password"
                    :placeholder="
                      mode === 'edit' && connection?.secretStatus?.password
                        ? '已保存，留空保持不变'
                        : '可选'
                    "
                    :saved-password-configured="showSavedPassword"
                    :loading="revealingPassword"
                    @reveal-saved-password="revealSavedPassword"
                  />
                </el-form-item>
              </div>
            </template>

            <template v-else-if="connectionType === 'tdengine'">
              <div class="connection-dialog__form-grid">
                <el-form-item label="协议">
                  <el-segmented v-model="formData.protocol" :options="tdengineProtocolOptions" />
                </el-form-item>
                <el-form-item label="主机" prop="host">
                  <el-input v-model="formData.host" placeholder="127.0.0.1" />
                </el-form-item>
                <el-form-item label="端口">
                  <el-input v-model.number="formData.port" inputmode="numeric" placeholder="6041" />
                </el-form-item>
              </div>
              <p class="connection-dialog__field-tip connection-dialog__tdengine-tip">
                使用 taosAdapter WebSocket 接口，默认端口 6041；支持 TDengine Server 3.3.6.0
                及以上版本。
              </p>
              <div class="connection-dialog__form-grid">
                <el-form-item label="Database" prop="database">
                  <el-input v-model="formData.database" placeholder="iot_data" />
                </el-form-item>
                <el-form-item label="时区">
                  <el-select
                    v-model="formData.timezone"
                    filterable
                    class="w-full"
                    placeholder="请选择时区"
                  >
                    <el-option
                      v-for="timezone in timezoneOptions"
                      :key="timezone"
                      :label="timezone"
                      :value="timezone"
                    />
                  </el-select>
                </el-form-item>
              </div>
              <div class="connection-dialog__form-grid">
                <el-form-item label="用户名">
                  <el-input v-model="formData.username" placeholder="root" />
                </el-form-item>
                <el-form-item label="密码">
                  <SavedPasswordInput
                    v-model="formData.password"
                    :placeholder="
                      mode === 'edit' && connection?.secretStatus?.password
                        ? '已保存，留空保持不变'
                        : '请输入密码'
                    "
                    :saved-password-configured="showSavedPassword"
                    :loading="revealingPassword"
                    @reveal-saved-password="revealSavedPassword"
                  />
                </el-form-item>
              </div>
              <el-form-item v-if="formData.protocol === 'wss'" label="证书校验">
                <el-radio-group v-model="formData.tlsSkipVerify">
                  <el-radio-button :value="false">验证证书</el-radio-button>
                  <el-radio-button :value="true">信任自签名证书</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </template>

            <template v-else-if="isSimpleMetadataSource">
              <el-form-item label="说明">
                <el-input
                  v-model="formData.description"
                  type="textarea"
                  :rows="3"
                  placeholder="可选，描述该接入源在工程中的用途"
                />
              </el-form-item>
              <template v-if="connectionType === 'builtin.realtime'">
                <el-form-item label="默认 TTL">
                  <el-input-number v-model="formData.defaultTtlSeconds" :min="0" :max="86400" />
                </el-form-item>
              </template>
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

          <section v-if="showTestButton" class="connection-dialog__test">
            <div class="connection-dialog__section-title">连接测试</div>
            <div class="connection-dialog__test-state" :class="`is-${testState.status}`">
              <component :is="testState.icon" class="connection-dialog__test-icon" />
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
            <el-button @click="requestClose">{{ tc('actions.cancel') }}</el-button>
          </template>
          <template v-else>
            <!-- 返回上一步：仅 create 模式可见 -->
            <el-button v-if="mode === 'create'" @click="goBackToStep1"> ← 返回上一步 </el-button>
            <el-button @click="requestClose">{{ tc('actions.cancel') }}</el-button>
            <el-button v-if="showTestButton" @click="handleTest" :loading="testing">
              {{ tc('actions.testConnection') }}
            </el-button>
            <el-button type="primary" @click="handleSubmit" :loading="submitting">
              {{ mode === 'create' ? tc('actions.createConnection') : tc('actions.saveChanges') }}
            </el-button>
          </template>
        </div>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, markRaw, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { getDefaultConfig } from '@/config/connectionTypes'
import { t } from '@/i18n/runtime'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerCircleCheck from '~icons/tabler/circle-check'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerHelpCircle from '~icons/tabler/help-circle'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerMessageCircle from '~icons/tabler/message-circle'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import IconTablerServer from '~icons/tabler/server'
import IconTablerWorldWww from '~icons/tabler/world-www'
import IconTablerWebhook from '~icons/tabler/webhook'
import IconTablerBolt from '~icons/tabler/bolt'
import IconTablerRadio from '~icons/tabler/radio'
import IconTablerTimeline from '~icons/tabler/timeline'
import MysqlConnectionForm from '../connection/forms/MysqlConnectionForm.vue'
import PostgresConnectionForm from '../connection/forms/PostgresConnectionForm.vue'
import SqlServerConnectionForm from '../connection/forms/SqlServerConnectionForm.vue'
import SavedPasswordInput from '../connection/forms/SavedPasswordInput.vue'
import MqttConnectionForm from '../connection/forms/MqttConnectionForm.vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import {
  BUILTIN_STORE_TYPES,
  isBuiltinStoreType,
} from '@/components/access-source/workbench/builtin-store'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  mode: {
    type: String,
    default: 'create', // 'create' | 'edit'
    validator: (value: string) => ['create', 'edit'].includes(value),
  },
  connection: {
    type: Object,
    default: null,
  },
  projectId: {
    type: [String, Number],
    default: '',
  },
})

const emit = defineEmits(['update:modelValue', 'submit'])
const tc = (key: string, params?: Record<string, any>) => t(key, params)

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

// step 状态机：create 从 1 开始，edit 直接 2
const step = ref<1 | 2>(props.mode === 'edit' ? 2 : 1)

// 点击协议卡进入 Step 2
const selectSourceAndAdvance = (value: string) => {
  selectSource(value)
  step.value = 2
  emptyFormSignature.value = formInputSignature.value
}

// 返回上一步：清空表单与测试状态，保留协议高亮
const goBackToStep1 = () => {
  formData.value = {}
  emptyFormSignature.value = ''
  resetTestState()
  step.value = 1
}

const connectionType = ref('mysql')
const dbType = ref('mysql')
const formData = ref<Record<string, any>>({})
const formRef = ref(null)
const protocolFormRef = ref(null)
const testing = ref(false)
const submitting = ref(false)
const lastTestSignature = ref('')
const emptyFormSignature = ref('')
const kafkaActiveCollapse = ref([])
const revealingPassword = ref(false)
const revealedPassword = ref('')
const testResult = ref({
  status: 'idle',
  title: '尚未测试',
  message: '填写连接参数后，可以先验证网络、认证与基础协议是否可用。',
  detail: '',
  durationMs: 0,
})

const builtinSourceOptions = BUILTIN_STORE_TYPES.map((store) => ({
  value: store.type,
  label: store.name,
  description: store.description,
  icon: markRaw(
    store.type === 'builtin.timeseries'
      ? IconTablerTimeline
      : store.type === 'builtin.realtime'
        ? IconTablerBolt
        : store.type === 'builtin.message'
          ? IconTablerRadio
          : IconTablerDatabase,
  ),
}))

const externalSourceOptions = [
  {
    value: 'mysql',
    label: 'MySQL',
    description: 'MySQL 数据库连接',
    icon: markRaw(IconTablerDatabase),
  },
  {
    value: 'postgresql',
    label: 'PG',
    description: 'PostgreSQL 数据库连接',
    icon: markRaw(IconTablerDatabase),
  },
  {
    value: 'sqlserver',
    label: 'SqlServer',
    description: 'SQL Server 数据库连接',
    icon: markRaw(IconTablerDatabase),
  },
  {
    value: 'mqtt',
    label: 'MQTT',
    description: 'Broker 连接与订阅配置',
    icon: markRaw(IconTablerMessageCircle),
  },
  {
    value: 'kafka',
    label: 'Kafka',
    description: 'Broker 地址与客户端参数',
    icon: markRaw(IconTablerServer),
  },
  {
    value: 'http',
    label: 'HTTP',
    description: '接口请求与响应样本',
    icon: markRaw(IconTablerWorldWww),
  },
  {
    value: 'websocket',
    label: 'WebSocket',
    description: '短连接消息预览',
    icon: markRaw(IconTablerWebhook),
  },
  {
    value: 'redis',
    label: 'Redis',
    description: '浏览、维护 Key 并生成数据点',
    icon: markRaw(IconTablerDatabase),
  },
  {
    value: 'tdengine',
    label: 'TDengine',
    description: '时序库查询与历史存储',
    icon: markRaw(IconTablerDatabase),
  },
]
const sourceOptions = [...builtinSourceOptions, ...externalSourceOptions]

const previewProtocolTypes = ['kafka', 'http', 'websocket', 'redis']
const industrialProtocolTypes = ['tdengine']
const relationalSourceTypes = ['mysql', 'postgresql', 'sqlserver']
const specializedProtocolTypes = [...previewProtocolTypes, ...industrialProtocolTypes]
const kafkaSecurityProtocolOptions = [
  { label: 'PLAINTEXT', value: 'PLAINTEXT' },
  { label: 'SSL', value: 'SSL' },
  { label: 'SASL_PLAINTEXT', value: 'SASL_PLAINTEXT' },
  { label: 'SASL_SSL', value: 'SASL_SSL' },
]
const kafkaSaslMechanismOptions = [
  { label: 'PLAIN', value: 'PLAIN' },
  { label: 'SCRAM-SHA-256', value: 'SCRAM-SHA-256' },
  { label: 'SCRAM-SHA-512', value: 'SCRAM-SHA-512' },
]
const redisModeOptions = [
  { label: '单机', value: 'standalone' },
  { label: '哨兵', value: 'sentinel' },
  { label: '集群', value: 'cluster' },
]
const tdengineProtocolOptions = [
  { label: 'WebSocket', value: 'ws' },
  { label: 'WebSocket TLS', value: 'wss' },
]
const fallbackTimezoneOptions = [
  'Asia/Shanghai',
  'UTC',
  'Asia/Tokyo',
  'Asia/Singapore',
  'Europe/London',
  'Europe/Berlin',
  'America/New_York',
  'America/Los_Angeles',
]
const timezoneOptions = (() => {
  const supportedValuesOf = (
    Intl as typeof Intl & {
      supportedValuesOf?: (key: 'timeZone') => string[]
    }
  ).supportedValuesOf
  const values = supportedValuesOf ? supportedValuesOf('timeZone') : fallbackTimezoneOptions
  return [...new Set(['Asia/Shanghai', 'UTC', ...values])]
})()
const protocolRules = {
  name: [{ required: true, message: '连接名称不能为空', trigger: 'blur' }],
  brokers: [{ required: true, message: '服务器地址不能为空', trigger: 'blur' }],
  url: [{ required: true, message: '连接地址不能为空', trigger: 'blur' }],
  address: [{ required: true, message: 'Redis 节点地址不能为空', trigger: 'blur' }],
  ip: [{ required: true, message: 'IP 地址不能为空', trigger: 'blur' }],
  username: [{ required: true, message: '用户名不能为空', trigger: 'blur' }],
  serialPort: [{ required: true, message: '串口不能为空', trigger: 'blur' }],
  database: [{ required: true, message: 'Database 不能为空', trigger: 'blur' }],
}

const activeSource = computed(() => {
  return sourceOptions.find((item) => item.value === connectionType.value) || sourceOptions[0]
})

const activeDatabase = computed(() => sourceOptions.find((item) => item.value === dbType.value))

const activeFormTitle = computed(() => {
  if (isBuiltinStoreSelected.value) return activeSource.value.label
  if (connectionType.value === 'mqtt') {
    return 'MQTT Broker'
  }
  if (connectionType.value === 'kafka') return 'Kafka 接入源'
  if (connectionType.value === 'http') return 'HTTP Source'
  if (connectionType.value === 'websocket') return 'WebSocket Source'
  if (connectionType.value === 'redis') return 'Redis 接入源'
  if (connectionType.value === 'tdengine') return 'TDengine 连接'
  return activeDatabase.value?.label || '数据库连接'
})

// 动态加载表单组件
const formComponent = computed(() => {
  if (relationalSourceTypes.includes(connectionType.value)) {
    const componentMap = {
      mysql: markRaw(MysqlConnectionForm),
      postgresql: markRaw(PostgresConnectionForm),
      sqlserver: markRaw(SqlServerConnectionForm),
    }
    return componentMap[dbType.value] || null
  } else if (connectionType.value === 'mqtt') {
    return markRaw(MqttConnectionForm)
  }
  return null
})

const isBuiltinStoreSelected = computed(() => isBuiltinStoreType(connectionType.value))
const connectionNamePlaceholder = computed(() => {
  if (isBuiltinStoreSelected.value) return activeSource.value.label
  const examples: Record<string, string> = {
    kafka: '例如：生产 Kafka',
    http: '例如：设备接口',
    websocket: '例如：实时消息',
    redis: '例如：生产 Redis',
    tdengine: '例如：生产时序库',
  }
  return examples[connectionType.value] || `例如：${activeSource.value.label} 接入源`
})
const isSimpleMetadataSource = computed(
  () => isBuiltinStoreSelected.value || ['http', 'websocket'].includes(connectionType.value),
)
const showTestButton = computed(() => !isSimpleMetadataSource.value)
const isKafkaSaslEnabled = computed(() =>
  String(formData.value.securityProtocol || '').includes('SASL'),
)
const savedPasswordKey = computed(() =>
  connectionType.value === 'kafka' ? 'option.password' : 'password',
)
const showSavedPassword = computed(
  () =>
    props.mode === 'edit' &&
    ['mysql', 'postgresql', 'sqlserver', 'redis', 'tdengine'].includes(connectionType.value) &&
    Boolean(props.connection?.secretStatus?.[savedPasswordKey.value]),
)
const savedTlsSecrets = computed(() => ({
  ca: Boolean(props.connection?.secretStatus?.['tls.ca']),
  cert: Boolean(props.connection?.secretStatus?.['tls.cert']),
  key: Boolean(props.connection?.secretStatus?.['tls.key']),
}))

const configSignature = computed(() => {
  return JSON.stringify({
    step: step.value,
    type: connectionType.value,
    dbType: dbType.value,
    data: formData.value,
  })
})

const formInputSignature = computed(() => JSON.stringify(formData.value || {}))

const canCloseByModalClick = computed(() => true)
const dialogRef = ref(null)
const silentClosing = ref(false)
const isDialogDirty = computed(
  () =>
    !silentClosing.value &&
    visible.value &&
    step.value === 2 &&
    Boolean(emptyFormSignature.value) &&
    formInputSignature.value !== emptyFormSignature.value,
)

const isTestStale = computed(() => {
  return (
    Boolean(lastTestSignature.value) &&
    lastTestSignature.value !== configSignature.value &&
    testResult.value.status === 'success'
  )
})

const testState = computed(() => {
  if (testing.value) {
    return {
      status: 'testing',
      icon: markRaw(IconTablerLoader2),
      title: '正在测试',
      message: '正在向 data_service 发起一次短时连接测试。',
      detail: '',
    }
  }

  const iconMap = {
    idle: markRaw(IconTablerPlugConnected),
    success: markRaw(IconTablerCircleCheck),
    error: markRaw(IconTablerAlertTriangle),
  }

  return {
    ...testResult.value,
    icon: iconMap[testResult.value.status] || iconMap.idle,
  }
})

const redisModeLabel = (mode: unknown) => {
  const option = redisModeOptions.find((item) => item.value === String(mode || 'standalone'))
  return option?.label || '单机'
}

const summaryRows = computed(() => {
  const data = formData.value || {}
  const rows = [
    { label: '类型', value: activeSource.value.label },
    { label: '名称', value: data.name || '未填写' },
  ]

  if (isBuiltinStoreSelected.value) {
    rows.push({ label: '能力', value: '平台内置' }, { label: '运行态', value: '随工程环境映射' })
  } else if (relationalSourceTypes.includes(connectionType.value)) {
    rows.push(
      { label: '数据库', value: activeDatabase.value?.label || dbType.value },
      { label: '地址', value: formatEndpoint(data.host, data.port) },
      { label: '库名', value: data.database || '未填写' },
      { label: '账号', value: data.username || '未填写' },
      {
        label: '超时',
        value: formatTimeout(data.queryTimeout || data.timeout),
      },
    )
  } else if (connectionType.value === 'mqtt') {
    rows.push(
      { label: '协议', value: data.protocol || 'mqtt' },
      { label: 'Broker', value: formatEndpoint(data.brokerUrl, data.port) },
      { label: 'Client ID', value: data.clientId || '自动生成' },
      { label: 'QoS', value: String(data.qos ?? 0) },
      { label: '认证', value: data.username ? '用户名/密码' : '匿名' },
    )
  } else if (connectionType.value === 'kafka') {
    rows.push(
      { label: '服务器地址', value: data.brokers || '未填写' },
      { label: '安全协议', value: data.securityProtocol || 'PLAINTEXT' },
      {
        label: 'SASL',
        value: String(data.securityProtocol || '').includes('SASL')
          ? data.saslMechanism || 'PLAIN'
          : '未启用',
      },
      { label: '超时', value: formatTimeout(data.requestTimeoutMs || data.dialTimeoutMs) },
    )
  } else if (connectionType.value === 'http') {
    rows.push({ label: '配置方式', value: '请求在工作台维护' })
  } else if (connectionType.value === 'websocket') {
    rows.push({ label: '配置方式', value: '会话在工作台维护' })
  } else if (connectionType.value === 'redis') {
    rows.push(
      { label: '连接方式', value: redisModeLabel(data.mode) },
      { label: '节点地址', value: data.address || '未填写' },
      { label: '数据库编号', value: String(data.db ?? 0) },
      { label: 'Key 筛选规则', value: data.keyPattern || '*' },
    )
  } else if (connectionType.value === 'tdengine') {
    rows.push(
      { label: '地址', value: formatEndpoint(data.host, data.port) },
      { label: '库名', value: data.database || '未填写' },
      { label: '账号', value: data.username || 'root' },
      { label: '时区', value: data.timezone || '未填写' },
    )
  }

  return rows
})

const checklist = computed(() => {
  const data = formData.value || {}
  const hasName = Boolean(data.name)
  if (isBuiltinStoreSelected.value) {
    return [
      { label: '名称已填写', ready: hasName },
      { label: '使用工程内置运行库', ready: true },
      { label: '无需配置外部地址和账号', ready: true },
      { label: '保存后在工作台验证功能', ready: true },
    ]
  }
  const hasEndpoint =
    connectionType.value === 'mqtt'
      ? Boolean(data.brokerUrl && data.port)
      : connectionType.value === 'kafka'
        ? Boolean(data.brokers)
        : ['http', 'websocket'].includes(connectionType.value)
          ? true
          : connectionType.value === 'redis'
            ? Boolean(data.address)
            : connectionType.value === 'tdengine'
              ? Boolean(data.host && data.port)
              : Boolean(data.host && data.port)
  const hasTarget = ['mqtt', 'kafka', 'http', 'websocket', 'redis'].includes(connectionType.value)
    ? true
    : Boolean(data.database)

  return [
    { label: '基础名称已填写', ready: hasName },
    {
      label: ['http', 'websocket'].includes(connectionType.value)
        ? connectionType.value === 'http'
          ? '工作台内配置请求'
          : '工作台内配置会话'
        : '网络地址已填写',
      ready: hasEndpoint,
    },
    {
      label: ['http', 'websocket'].includes(connectionType.value)
        ? connectionType.value === 'http'
          ? '保存后创建请求项'
          : '保存后创建会话'
        : '目标资源已明确',
      ready: hasTarget,
    },
    {
      label: ['http', 'websocket'].includes(connectionType.value)
        ? '无需测试连接'
        : previewProtocolTypes.includes(connectionType.value)
          ? connectionType.value === 'kafka'
            ? '可先测试 Broker 连通'
            : '保存后可短时预览'
          : industrialProtocolTypes.includes(connectionType.value)
            ? '保存后进入只读 SQL 工作台'
            : '连接测试可选完成',
      ready:
        ['http', 'websocket'].includes(connectionType.value) ||
        (previewProtocolTypes.includes(connectionType.value) && connectionType.value !== 'kafka') ||
        industrialProtocolTypes.includes(connectionType.value) ||
        (testResult.value.status === 'success' && !isTestStale.value),
    },
  ]
})

// 监听对话框打开
watch(
  () => props.modelValue,
  (isOpen) => {
    document.body.classList.toggle('connection-dialog-open', isOpen)
    document.documentElement.classList.toggle('connection-dialog-open', isOpen)
    if (isOpen && props.mode === 'create') {
      // create 打开时回到 step 1 并重置
      step.value = 1
      connectionType.value = 'mysql'
      dbType.value = 'mysql'
      formData.value = getDefaultConfig('mysql')
      emptyFormSignature.value = ''
      resetTestState()
    } else if (isOpen && props.mode === 'edit') {
      // edit 模式直接进入 step 2
      step.value = 2
      hydrateEditForm(props.connection)
    } else if (!isOpen) {
      document.body.classList.remove('connection-dialog-open')
      document.documentElement.classList.remove('connection-dialog-open')
    }
  },
)

onBeforeUnmount(() => {
  document.body.classList.remove('connection-dialog-open')
  document.documentElement.classList.remove('connection-dialog-open')
})

const selectSource = (value) => {
  if (props.mode === 'edit' || connectionType.value === value) return
  connectionType.value = value
  // 连接类型变化时重置表单
  if (isBuiltinStoreSelected.value) {
    formData.value = normalizeBuiltinFormData(connectionType.value, {})
  } else if (relationalSourceTypes.includes(connectionType.value)) {
    dbType.value = connectionType.value
    formData.value = getDefaultConfig(dbType.value)
  } else if (connectionType.value === 'mqtt') {
    formData.value = getDefaultConfig('mqtt')
  } else {
    formData.value = getProtocolDefaultConfig(connectionType.value)
  }
  resetTestState()
  emptyFormSignature.value = formInputSignature.value
}

const resetCurrentForm = () => {
  if (isBuiltinStoreSelected.value) {
    formData.value = normalizeBuiltinFormData(connectionType.value, {})
  } else if (relationalSourceTypes.includes(connectionType.value)) {
    dbType.value = connectionType.value
    formData.value = getDefaultConfig(dbType.value)
  } else {
    formData.value =
      connectionType.value === 'mqtt'
        ? getDefaultConfig('mqtt')
        : getProtocolDefaultConfig(connectionType.value)
  }
  resetTestState()
  emptyFormSignature.value = formInputSignature.value
}

const resetTestState = () => {
  lastTestSignature.value = ''
  testResult.value = {
    status: 'idle',
    title: '尚未测试',
    message: '填写连接参数后，可以先验证网络、认证与基础协议是否可用。',
    detail: '',
    durationMs: 0,
  }
}

const hydrateEditForm = (connection) => {
  if (!connection || props.mode !== 'edit') return
  revealedPassword.value = ''
  connectionType.value = connection.type
  const protocolConfig = resolveConnectionConfigForForm(connection)
  if (isBuiltinStoreType(connection.type)) {
    formData.value = normalizeBuiltinFormData(connection.type, {
      name: connection.name,
      ...protocolConfig,
    })
  } else if (connection.type === 'relational' && connection.relationalConfig) {
    dbType.value = connection.relationalConfig.dbType
    connectionType.value = dbType.value
    formData.value = {
      name: connection.name,
      ...normalizeRelationalFormConfig(connection.relationalConfig),
    }
  } else if (connection.type === 'mqtt' && connection.mqttConfig) {
    formData.value = {
      name: connection.name,
      ...connection.mqttConfig,
    }
  } else {
    formData.value = normalizeProtocolFormData(connection.type, {
      name: connection.name,
      ...protocolConfig,
    })
  }
  resetTestState()
  emptyFormSignature.value = formInputSignature.value
}

const resolveConnectionConfigForForm = (connection) => {
  const config =
    connection?.config && typeof connection.config === 'object' ? { ...connection.config } : {}
  if (config.config && typeof config.config === 'object') {
    Object.assign(config, config.config)
    delete config.config
  }

  return config
}

// 将已有关系库连接转换为统一 TLS 表单模型；仅用于开发期已有连接的编辑展示。
const normalizeRelationalFormConfig = (config) => {
  const normalized = { ...(config || {}) }
  const source =
    normalized.sslConfig && typeof normalized.sslConfig === 'object' ? normalized.sslConfig : {}
  normalized.sslConfig = {
    mode: source.mode || 'disable',
    ca: source.ca || '',
    cert: source.cert || '',
    key: source.key || '',
  }
  return normalized
}

const handleValidate = () => undefined

const validateCurrentForm = async () => {
  const validator = formComponent.value ? formRef.value : protocolFormRef.value
  if (!validator) return false

  try {
    const valid = await validator.validate()
    if (!valid) return false
    buildConnectionPayload()
    return true
  } catch (error) {
    if (error instanceof Error) {
      ElMessage.warning(error.message)
    }
    return false
  }
}

const buildConnectionPayload = () => {
  const config = { ...formData.value }
  const name = config.name
  delete config.name

  if (isBuiltinStoreSelected.value) {
    normalizeBuiltinSubmitConfig(connectionType.value, config)
  } else if (relationalSourceTypes.includes(connectionType.value)) {
    config.dbType = dbType.value
    normalizeRelationalTLSConfig(config)
    // 编辑态未查看、未修改密码时不提交空值，避免误删已保存密钥。
    if (props.mode === 'edit' && !String(config.password || '')) delete config.password
  } else if (specializedProtocolTypes.includes(connectionType.value)) {
    normalizeProtocolSubmitConfig(connectionType.value, config)
  }

  return {
    name,
    type: relationalSourceTypes.includes(connectionType.value)
      ? 'relational'
      : connectionType.value,
    config,
  }
}

// 统一关系库 TLS 提交结构；空证书字段不提交，编辑时即可保留已加密保存的证书。
const normalizeRelationalTLSConfig = (config) => {
  const source = config.sslConfig && typeof config.sslConfig === 'object' ? config.sslConfig : {}
  const sslConfig = {
    mode: source.mode || 'disable',
  }
  for (const key of ['ca', 'cert', 'key']) {
    const value = String(source[key] || '').trim()
    if (value) sslConfig[key] = value
  }
  config.sslConfig = sslConfig
  delete config.sslMode
  delete config.sslCa
  delete config.sslCert
  delete config.sslKey
  delete config.encrypt
  delete config.trustServerCertificate
}

const handleMqttConnectionTest = async () => {
  testing.value = true
  const startAt = performance.now()
  try {
    const payload = buildConnectionPayload()
    await dataAPI.testMqttConnection(props.projectId, {
      name: payload.name,
      ...payload.config,
    })

    const durationMs = Math.round(performance.now() - startAt)
    lastTestSignature.value = configSignature.value
    testResult.value = {
      status: 'success',
      title: '测试通过',
      message: `MQTT Broker 连接验证通过，耗时 ${durationMs}ms。`,
      detail: '',
      durationMs,
    }
    ElMessage.success(tc('query.connectionTestSuccess'))
  } catch (error) {
    const durationMs = Math.round(performance.now() - startAt)
    testResult.value = {
      status: 'error',
      title: '测试失败',
      message: 'MQTT Broker 连接未通过验证，请检查地址、端口或认证配置。',
      detail: getApiErrorMessage(error, 'MQTT 连接测试失败'),
      durationMs,
    }
    ElMessage.error(getApiErrorMessage(error, 'MQTT 连接测试失败'))
  } finally {
    testing.value = false
  }
}

const generateKafkaClientId = () => {
  const randomStr = Math.random().toString(36).substring(2, 10)
  formData.value.clientId = `induforge_kafka_${randomStr}`
}

const handleTest = async () => {
  const valid = await validateCurrentForm()
  if (!valid) {
    return
  }

  if (!props.projectId) {
    ElMessage.warning('缺少工程上下文，无法测试连接')
    return
  }

  if (connectionType.value === 'mqtt') {
    await handleMqttConnectionTest()
    return
  }

  if (isBuiltinStoreSelected.value) {
    testResult.value = {
      status: 'idle',
      title: '保存后测试',
      message: '内置运行库不需要连接测试，保存后进入对应工作台执行真实开发态测试。',
      detail: '',
      durationMs: 0,
    }
    ElMessage.info('保存后进入内置运行库工作台测试')
    return
  }

  testing.value = true
  const startAt = performance.now()
  try {
    const payload = buildConnectionPayload()
    const testConfig = { ...payload.config }
    let testPassword = String(formData.value.password || '')
    if (
      props.mode === 'edit' &&
      !testPassword &&
      props.connection?.secretStatus?.[savedPasswordKey.value]
    ) {
      testPassword = await fetchSavedPassword()
    }
    if (connectionType.value === 'kafka' && testPassword) {
      testConfig.options = { ...(testConfig.options || {}), password: testPassword }
    } else if (
      ['mysql', 'postgresql', 'sqlserver', 'redis', 'tdengine'].includes(connectionType.value)
    ) {
      testConfig.password = testPassword
    }
    if (['redis', 'tdengine'].includes(connectionType.value)) {
      delete testConfig.secrets
      delete testConfig.clearSecretKeys
    }
    const response = await dataAPI.testConnection(props.projectId, {
      type: payload.type,
      config: testConfig,
    })
    const result = response?.data || response || {}

    const durationMs = Math.round(performance.now() - startAt)
    lastTestSignature.value = configSignature.value
    testResult.value = {
      status: 'success',
      title: '测试通过',
      message: result.message || `data_service 已完成短时连接验证，耗时 ${durationMs}ms。`,
      detail: result.detail || '',
      durationMs,
    }
    ElMessage.success(tc('query.connectionTestSuccess'))
  } catch (error) {
    const durationMs = Math.round(performance.now() - startAt)
    testResult.value = {
      status: 'error',
      title: '测试失败',
      message: '连接参数未通过验证，请检查网络、认证或协议配置。',
      detail: getApiErrorMessage(error, '连接测试失败'),
      durationMs,
    }
    ElMessage.error(getApiErrorMessage(error, '连接测试失败'))
  } finally {
    testing.value = false
  }
}

const handleSubmit = async () => {
  const valid = await validateCurrentForm()
  if (!valid) {
    return
  }

  submitting.value = true
  try {
    const payload = buildConnectionPayload()

    emit('submit', {
      name: payload.name,
      type: payload.type,
      config: payload.config,
      tested:
        testResult.value.status === 'success' &&
        Boolean(lastTestSignature.value) &&
        lastTestSignature.value === configSignature.value,
    })
  } finally {
    submitting.value = false
  }
}

const requestClose = () => {
  visible.value = false
}

const closeSilently = () => {
  silentClosing.value = true
  emptyFormSignature.value = formInputSignature.value
  dialogRef.value?.closeSilently?.()
}

const handleClosed = () => {
  emptyFormSignature.value = ''
  silentClosing.value = false
  revealedPassword.value = ''
  // 清空表单
  if (formRef.value) {
    formRef.value.clearValidate()
  }
  if (protocolFormRef.value) {
    protocolFormRef.value.clearValidate()
  }
}

const revealSavedPassword = async () => {
  if (!props.projectId || !props.connection?.id || revealingPassword.value) return
  const wasClean = formInputSignature.value === emptyFormSignature.value
  revealingPassword.value = true
  try {
    revealedPassword.value = await fetchSavedPassword()
    formData.value = { ...formData.value, password: revealedPassword.value }
    if (wasClean) emptyFormSignature.value = formInputSignature.value
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '查看已保存密码失败'))
  } finally {
    revealingPassword.value = false
  }
}

const fetchSavedPassword = async () => {
  if (!props.projectId || !props.connection?.id) throw new Error('缺少接入源上下文')
  const response = await dataAPI.revealConnectionSecret(
    props.projectId,
    props.connection.id,
    savedPasswordKey.value,
  )
  const password = String(response?.data?.value ?? response?.value ?? '')
  if (!password) throw new Error('已保存密码为空')
  return password
}

defineExpose({ closeSilently })

const getProtocolDefaultConfig = (type) => {
  const defaults = {
    kafka: {
      name: '',
      brokers: '',
      securityProtocol: 'PLAINTEXT',
      saslMechanism: 'PLAIN',
      username: '',
      password: '',
      clientId: '',
      dialTimeoutMs: 5000,
      requestTimeoutMs: 5000,
      clearSecrets: false,
      sslConfig: {
        ca: '',
        cert: '',
        key: '',
        rejectUnauthorized: true,
      },
    },
    http: {
      name: '',
      description: '',
    },
    websocket: {
      name: '',
      description: '',
    },
    redis: {
      name: '',
      address: '127.0.0.1:6379',
      db: 0,
      username: '',
      password: '',
      keyPattern: '*',
      mode: 'standalone',
      masterName: '',
      clearSecrets: false,
    },
    tdengine: {
      name: '',
      protocol: 'ws',
      host: '127.0.0.1',
      port: 6041,
      database: '',
      username: 'root',
      password: '',
      timezone: 'Asia/Shanghai',
      tlsSkipVerify: false,
      clearPassword: false,
    },
  }
  return { ...(defaults[type] || {}) }
}

const normalizeBuiltinFormData = (type, config) => {
  const found = BUILTIN_STORE_TYPES.find((item) => item.type === type)
  return {
    name: found?.name || '',
    ...(found?.defaultConfig || {}),
    ...config,
  }
}

const normalizeBuiltinSubmitConfig = (type, config) => {
  if (type === 'builtin.realtime') {
    config.defaultTtlSeconds = Number(config.defaultTtlSeconds) || 300
  }
}

const normalizeKafkaOptionsForForm = (options) => {
  const sslConfig =
    options.sslConfig && typeof options.sslConfig === 'object' ? options.sslConfig : {}
  return {
    securityProtocol: String(options.securityProtocol || 'PLAINTEXT').toUpperCase(),
    saslMechanism: String(options.saslMechanism || 'PLAIN').toUpperCase(),
    username: options.username || '',
    password: options.password || '',
    clientId: options.clientId || '',
    dialTimeoutMs: Number(options.dialTimeoutMs) || 5000,
    requestTimeoutMs: Number(options.requestTimeoutMs) || 5000,
    sslConfig: {
      ca: sslConfig.ca || '',
      cert: sslConfig.cert || '',
      key: sslConfig.key || '',
      rejectUnauthorized:
        typeof sslConfig.rejectUnauthorized === 'boolean'
          ? sslConfig.rejectUnauthorized
          : !options.tlsInsecureSkipVerify,
    },
  }
}

const buildKafkaOptions = (config) => {
  const options: Record<string, any> = {
    securityProtocol: String(config.securityProtocol || 'PLAINTEXT').toUpperCase(),
  }
  if (String(options.securityProtocol).includes('SASL')) {
    options.saslMechanism = String(config.saslMechanism || 'PLAIN').toUpperCase()
    if (String(config.username || '').trim()) {
      options.username = String(config.username).trim()
    }
    if (String(config.password || '').trim()) {
      options.password = String(config.password)
    }
  }
  if (String(config.clientId || '').trim()) {
    options.clientId = String(config.clientId).trim()
  }
  const dialTimeoutMs = Number(config.dialTimeoutMs)
  if (Number.isFinite(dialTimeoutMs) && dialTimeoutMs > 0) {
    options.dialTimeoutMs = dialTimeoutMs
  }
  const requestTimeoutMs = Number(config.requestTimeoutMs)
  if (Number.isFinite(requestTimeoutMs) && requestTimeoutMs > 0) {
    options.requestTimeoutMs = requestTimeoutMs
  }
  if (String(options.securityProtocol).includes('SSL')) {
    const sslConfig = config.sslConfig || {}
    const normalizedSSLConfig: Record<string, any> = {
      rejectUnauthorized: sslConfig.rejectUnauthorized !== false,
    }
    if (String(sslConfig.ca || '').trim()) {
      normalizedSSLConfig.ca = String(sslConfig.ca)
    }
    if (String(sslConfig.cert || '').trim()) {
      normalizedSSLConfig.cert = String(sslConfig.cert)
    }
    if (String(sslConfig.key || '').trim()) {
      normalizedSSLConfig.key = String(sslConfig.key)
    }
    options.sslConfig = normalizedSSLConfig
  }
  return options
}

const normalizeProtocolFormData = (type, config) => {
  const data = { ...getProtocolDefaultConfig(type), ...config }
  if (type === 'tdengine' && !data.database && config.databaseName) {
    data.database = config.databaseName
  }
  if (type === 'kafka' && config.options && typeof config.options === 'object') {
    Object.assign(data, normalizeKafkaOptionsForForm(config.options))
  }
  if (data.headers && typeof data.headers === 'object') {
    data.headersText = JSON.stringify(data.headers, null, 2)
  }
  if (data.options && typeof data.options === 'object') {
    if (type !== 'kafka' && type !== 'tdengine') {
      data.optionsText = JSON.stringify(data.options, null, 2)
    }
    data.masterName = data.options.masterName || data.masterName
  }
  if (data.serialConfig && typeof data.serialConfig === 'object') {
    data.serialConfigText = JSON.stringify(data.serialConfig, null, 2)
  }
  if (data.bodyTemplate && typeof data.bodyTemplate === 'object') {
    data.bodyTemplateText = JSON.stringify(data.bodyTemplate, null, 2)
  }
  return data
}

const normalizeProtocolSubmitConfig = (type, config) => {
  if (type === 'kafka') {
    config.options = buildKafkaOptions(config)
    const secrets: Record<string, string> = {}
    if (String(config.options.password || ''))
      secrets['option.password'] = String(config.options.password)
    delete config.options.password
    const tls =
      config.options.sslConfig && typeof config.options.sslConfig === 'object'
        ? { ...config.options.sslConfig }
        : {}
    for (const key of ['ca', 'cert', 'key']) {
      if (String(tls[key] || '')) secrets[`tls.${key}`] = String(tls[key])
      delete tls[key]
    }
    config.options.sslConfig = tls
    config.secrets = secrets
    config.clearSecretKeys = config.clearSecrets
      ? ['option.password', 'tls.ca', 'tls.cert', 'tls.key']
      : []
    delete config.securityProtocol
    delete config.saslMechanism
    delete config.username
    delete config.password
    delete config.clientId
    delete config.dialTimeoutMs
    delete config.requestTimeoutMs
    delete config.sslConfig
    delete config.clearSecrets
    delete config.topic
    delete config.consumerGroup
    delete config.startPosition
    return
  }
  if (type === 'http') {
    delete config.baseUrl
    delete config.method
    delete config.headers
    delete config.headersText
    delete config.timeoutMs
    delete config.bodyTemplate
    delete config.bodyTemplateText
    return
  }
  if (type === 'websocket') {
    delete config.url
    delete config.topic
    delete config.headers
    delete config.headersText
    delete config.heartbeatIntervalMs
    delete config.subscribeMessage
    return
  }
  if (type === 'redis') {
    const options: Record<string, any> = {}
    if (config.masterName) options.masterName = config.masterName
    config.options = options
    const password = String(config.password || '')
    config.secrets = password ? { password } : {}
    config.clearSecretKeys = config.clearSecrets ? ['password'] : []
    delete config.password
    delete config.clearSecrets
    delete config.masterName
    return
  }
  if (type === 'tdengine') {
    config.options = {}
    config.databaseName = config.database
    const password = String(config.password || '')
    config.secrets = password ? { password } : {}
    config.clearSecretKeys = config.clearPassword ? ['password'] : []
    delete config.database
    delete config.password
    delete config.clearPassword
    if (!config.timezone) delete config.timezone
    delete config.optionsText
  }
}

const formatEndpoint = (host, port) => {
  const safeHost = host || '未填写'
  return port ? `${safeHost}:${port}` : safeHost
}

const formatTimeout = (timeout) => {
  if (!timeout) return '未设置'
  return `${timeout}ms`
}

// 监听连接数据变化（编辑模式）
watch(
  () => props.connection,
  (newConnection) => {
    hydrateEditForm(newConnection)
  },
  { immediate: true },
)
</script>

<style scoped>
:deep(.connection-dialog.el-dialog) {
  height: min(760px, calc(100vh - 56px));
  max-height: calc(100vh - 56px) !important;
  margin: 28px auto !important;
}

:deep(.connection-dialog.el-dialog .el-dialog__body) {
  flex: 1 1 auto;
  min-height: 0;
  max-height: none;
  overflow: hidden;
  padding-bottom: 12px;
}

:deep(.connection-dialog.el-dialog .dc-dialog__body) {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

:deep(.connection-dialog.el-dialog .el-dialog__footer) {
  flex: 0 0 auto;
}

/* ── Step 1：选择接入类型 ── */
.connection-dialog__step1 {
  height: 100%;
  min-height: 0;
  padding: 24px 20px 8px;
  overflow-y: auto;
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
  transition:
    background 0.16s ease,
    border-color 0.16s ease,
    color 0.16s ease;
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
  height: 100%;
  max-height: 100%;
  min-height: 0;
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
  height: 100%;
  max-height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-left: 1px solid var(--dc-border);
  overflow-y: hidden;
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
  height: 100%;
  max-height: 100%;
  min-height: 0;
  padding: 18px 20px;
  overflow-y: scroll;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
}

.connection-dialog__tdengine-tip {
  margin-top: -8px;
  margin-bottom: 14px;
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
  background: color-mix(in oklch, var(--dc-success) 8%, var(--dc-surface-raised));
}

.connection-dialog__test-state.is-success .connection-dialog__test-icon {
  color: var(--dc-success);
}

.connection-dialog__test-state.is-error {
  border-color: color-mix(in oklch, var(--dc-danger) 34%, var(--dc-border));
  background: color-mix(in oklch, var(--dc-danger) 7%, var(--dc-surface-raised));
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
  flex: 1 1 auto !important;
  min-height: 0 !important;
  max-height: none !important;
  overflow: hidden !important;
  padding: 24px 20px 12px;
}

:global(.connection-dialog .dc-dialog__body) {
  height: 100%;
  min-height: 0;
  overflow: hidden;
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

.connection-dialog__field-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.connection-dialog__field-help {
  width: 14px;
  height: 14px;
  color: var(--dc-text-muted);
  cursor: help;
}

.connection-dialog__field-help:hover {
  color: var(--dc-primary);
}

.connection-dialog__readonly-value {
  min-height: 32px;
  display: flex;
  align-items: center;
  padding: 0 11px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  font-size: 13px;
}

.connection-dialog__form-section {
  margin-bottom: 18px;
}

.connection-dialog__form-section-title {
  margin-bottom: 12px;
  padding-left: 8px;
  border-left: 3px solid var(--dc-primary);
  color: var(--dc-text);
  font-size: 14px;
  font-weight: 650;
}

.connection-dialog__empty-config {
  display: flex;
  gap: 12px;
  padding: 14px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.connection-dialog__empty-config-icon {
  flex: 0 0 auto;
  width: 22px;
  height: 22px;
  color: var(--dc-primary);
}

.connection-dialog__empty-config strong {
  display: block;
  margin-bottom: 4px;
  font-size: 13px;
}

.connection-dialog__empty-config p {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.connection-dialog__field-tip,
.connection-dialog__field-inline-tip {
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.connection-dialog__field-tip {
  margin: 6px 0 0;
}

.connection-dialog__field-inline-tip {
  margin-left: 8px;
}

.connection-dialog__segmented-full {
  width: 100%;
}

.connection-dialog__segmented-full :deep(.el-segmented__item) {
  flex: 1;
}

.connection-dialog__switch-line {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
}

.connection-dialog__switch-text {
  color: var(--dc-text);
  font-size: 13px;
}

.connection-dialog__collapse {
  margin-top: 4px;
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

<style>
html.connection-dialog-open,
body.connection-dialog-open {
  overflow: hidden !important;
}

html.connection-dialog-open .el-overlay-dialog {
  overflow: hidden !important;
}

.connection-dialog.el-dialog {
  height: min(760px, calc(100vh - 56px)) !important;
  max-height: calc(100vh - 56px) !important;
  overflow: hidden !important;
}

.connection-dialog.el-dialog .el-dialog__body {
  flex: 1 1 0 !important;
  min-height: 0 !important;
  max-height: none !important;
  overflow: hidden !important;
  padding: 24px 20px 12px !important;
}

.connection-dialog.el-dialog .dc-dialog__body {
  height: 100%;
  min-height: 0;
  overflow: hidden !important;
}

.connection-dialog.el-dialog .connection-dialog__body {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.connection-dialog.el-dialog .connection-dialog__form-panel {
  height: 100%;
  min-height: 0;
  overflow-y: scroll;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
}

.connection-dialog.el-dialog .connection-dialog__inspector {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
</style>
