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
            <el-form-item :label="isBuiltinStoreSelected ? '名称' : '连接名称'" prop="name">
              <el-input
                v-model="formData.name"
                :placeholder="isBuiltinStoreSelected ? activeSource.label : '例如：产线 Kafka'"
              />
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
                      placeholder="请输入 SASL 密码"
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
            </template>

            <template v-else-if="connectionType === 'redis'">
              <el-form-item label="部署模式">
                <el-segmented v-model="formData.mode" :options="redisModeOptions" />
              </el-form-item>
              <el-form-item label="地址" prop="address">
                <el-input
                  v-model="formData.address"
                  placeholder="127.0.0.1:6379，多节点用英文逗号分隔"
                />
              </el-form-item>
              <div class="connection-dialog__form-grid">
                <el-form-item label="DB">
                  <el-input v-model.number="formData.db" inputmode="numeric" placeholder="0" />
                </el-form-item>
                <el-form-item v-if="formData.mode === 'sentinel'" label="Master">
                  <el-input v-model="formData.masterName" placeholder="mymaster" />
                </el-form-item>
                <el-form-item label="Key Pattern">
                  <el-input v-model="formData.keyPattern" placeholder="device:*" />
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
              <!-- 基础连接配置 -->
              <div class="connection-dialog__form-section">
                <div class="connection-dialog__form-section-title">基础连接配置</div>
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
              </div>

              <!-- 安全与认证配置 -->
              <div class="connection-dialog__form-section">
                <div class="connection-dialog__form-section-title">安全与认证配置</div>
                <el-form-item label="保护模式">
                  <el-segmented
                    v-model="formData.securityMode"
                    :options="opcuaSecurityModeOptions"
                    class="connection-dialog__segmented-full"
                  />
                  <p class="connection-dialog__field-tip">
                    None 不签名也不加密；Sign 仅签名；Sign & Encrypt 同时签名和加密。
                  </p>
                </el-form-item>
                <div v-if="formData.securityMode !== 'none'" class="connection-dialog__form-grid">
                  <el-form-item label="安全算法">
                    <el-select v-model="formData.securityPolicy" class="w-full">
                      <el-option label="None" value="None" />
                      <el-option label="Basic256Sha256" value="Basic256Sha256" />
                      <el-option label="Basic256" value="Basic256" />
                    </el-select>
                  </el-form-item>
                </div>
                <el-form-item label="认证方式">
                  <el-segmented
                    v-model="formData.authType"
                    :options="opcuaAuthOptions"
                    class="connection-dialog__segmented-full"
                  />
                </el-form-item>
                <div
                  v-if="formData.authType === 'username_password'"
                  class="connection-dialog__form-grid"
                >
                  <el-form-item label="用户名" prop="username">
                    <el-input v-model="formData.username" placeholder="OPC UA 用户名" />
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
              </div>

              <div class="connection-dialog__form-section">
                <div class="connection-dialog__form-section-title">设备冗余</div>
                <div class="connection-dialog__switch-line">
                  <el-switch v-model="formData.redundancy.enabled" />
                  <span class="connection-dialog__switch-text">
                    {{ formData.redundancy.enabled ? '启用主备服务器' : '不启用' }}
                  </span>
                </div>
                <div v-if="formData.redundancy.enabled" class="connection-dialog__redundancy-grid">
                  <el-form-item
                    v-for="endpoint in formData.redundancy.endpoints"
                    :key="endpoint.role"
                    :label="endpoint.role === 'primary' ? '主服务器' : '备用服务器'"
                  >
                    <div
                      v-if="endpoint.role === 'primary'"
                      class="connection-dialog__readonly-value"
                    >
                      {{ formatEndpoint(formData.ip, formData.port) }}
                    </div>
                    <div v-else class="connection-dialog__endpoint-row">
                      <el-input v-model="endpoint.host" placeholder="备用服务器 IP" />
                      <el-input-number v-model="endpoint.port" :min="1" class="w-full" />
                    </div>
                  </el-form-item>
                </div>
                <div v-if="formData.redundancy.enabled" class="connection-dialog__redundancy-grid">
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        故障超时
                        <el-tooltip
                          content="主服务器连续无响应超过该时间后，切换到备用服务器。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number
                      v-model="formData.redundancy.failoverPolicy.timeoutMs"
                      :min="1000"
                      :step="1000"
                      class="w-full"
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        冷却时间
                        <el-tooltip
                          content="完成一次切换后，至少等待该时间再允许下一次切换。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number
                      v-model="formData.redundancy.failoverPolicy.cooldownMs"
                      :min="1000"
                      :step="1000"
                      class="w-full"
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        自动回切
                        <el-tooltip
                          content="主服务器恢复稳定后，自动从备用服务器切回主服务器。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-switch v-model="formData.redundancy.failoverPolicy.autoFailback" />
                  </el-form-item>
                </div>
              </div>

              <!-- SSL/TLS 及高级参数配置 -->
              <el-collapse v-model="opcuaActiveCollapse" class="connection-dialog__collapse mt-4">
                <el-collapse-item title="高级参数配置 (含超时与证书)" name="ssl">
                  <div class="connection-dialog__form-grid mb-4">
                    <el-form-item label="连接超时">
                      <el-input
                        v-model.number="formData.connectionTimeoutMs"
                        inputmode="numeric"
                        placeholder="5000"
                      >
                        <template #append>ms</template>
                      </el-input>
                    </el-form-item>
                    <el-form-item label="请求超时">
                      <el-input
                        v-model.number="formData.requestTimeoutMs"
                        inputmode="numeric"
                        placeholder="5000"
                      >
                        <template #append>ms</template>
                      </el-input>
                    </el-form-item>
                  </div>
                  <el-form-item label="CA 证书 (服务器)">
                    <el-input
                      v-model="formData.sslConfig.ca"
                      type="textarea"
                      :rows="4"
                      placeholder="PEM格式公钥，用于校验 OPC UA 服务器 of 身份"
                    />
                  </el-form-item>
                  <el-form-item label="客户端证书">
                    <el-input
                      v-model="formData.sslConfig.cert"
                      type="textarea"
                      :rows="4"
                      placeholder="PEM格式公钥证书，用于双向 SSL 认证时的客户端声明"
                    />
                  </el-form-item>
                  <el-form-item label="客户端私钥">
                    <el-input
                      v-model="formData.sslConfig.key"
                      type="textarea"
                      :rows="4"
                      placeholder="PEM格式私钥，与客户端证书成对使用"
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        验证服务器证书
                        <el-tooltip
                          content="启用后会校验证书合法性及主机名；自签名证书调试时建议关闭。"
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
            </template>

            <template v-else-if="connectionType === 's7'">
              <div class="connection-dialog__form-grid">
                <el-form-item label="PLC IP" prop="ip">
                  <el-input v-model="formData.ip" placeholder="192.168.1.10" />
                </el-form-item>
                <el-form-item label="端口">
                  <el-input v-model.number="formData.port" inputmode="numeric" placeholder="102" />
                </el-form-item>
              </div>
              <div class="connection-dialog__form-section">
                <div class="connection-dialog__form-section-title">PLC 类型</div>
                <div class="connection-dialog__form-grid">
                  <el-form-item label="PLC 系列">
                    <el-select
                      v-model="formData.plcFamily"
                      class="w-full"
                      @change="applyS7FamilyDefaults"
                    >
                      <el-option
                        v-for="family in s7FamilyOptions"
                        :key="family"
                        :label="family"
                        :value="family"
                      />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="通信方式">
                    <el-select v-model="formData.communicationMode" class="w-full">
                      <el-option label="机架/槽位" value="rack_slot" />
                      <el-option label="TSAP" value="tsap" />
                    </el-select>
                  </el-form-item>
                </div>
                <div class="connection-dialog__form-grid">
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        机架号（Rack）
                        <el-tooltip
                          content="S7-300/400 常见为 0；不同 PLC 或网关请按设备手册填写。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number v-model="formData.rack" :min="0" :step="1" class="w-full" />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        槽位号（Slot）
                        <el-tooltip
                          content="S7-1200/1500 常见为 1，S7-300/400 常见为 2。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number v-model="formData.slot" :min="0" :step="1" class="w-full" />
                  </el-form-item>
                </div>
                <div class="connection-dialog__switch-line">
                  <el-switch v-model="formData.optimizedBlockAccess" />
                  <span class="connection-dialog__switch-text">启用优化块访问提示</span>
                  <el-tooltip
                    content="S7-1200/1500 使用优化 DB 时，绝对地址读取可能需要在 PLC 工程里额外确认。"
                    placement="top"
                  >
                    <IconTablerHelpCircle class="connection-dialog__field-help" />
                  </el-tooltip>
                </div>
              </div>
              <div class="connection-dialog__form-section">
                <div class="connection-dialog__form-section-title">设备冗余</div>
                <div class="connection-dialog__switch-line">
                  <el-switch v-model="formData.redundancy.enabled" />
                  <span class="connection-dialog__switch-text">
                    {{ formData.redundancy.enabled ? '启用主备 PLC' : '不启用' }}
                  </span>
                </div>
                <div v-if="formData.redundancy.enabled" class="connection-dialog__redundancy-grid">
                  <el-form-item
                    v-for="endpoint in formData.redundancy.endpoints"
                    :key="endpoint.role"
                    :label="endpoint.role === 'primary' ? '主 PLC' : '备 PLC'"
                  >
                    <div
                      v-if="endpoint.role === 'primary'"
                      class="connection-dialog__readonly-value"
                    >
                      {{ formatEndpoint(formData.ip, formData.port) }}
                    </div>
                    <div v-else class="connection-dialog__endpoint-row">
                      <el-input v-model="endpoint.host" placeholder="备用 PLC IP" />
                      <el-input-number v-model="endpoint.port" :min="1" class="w-full" />
                    </div>
                  </el-form-item>
                </div>
                <div v-if="formData.redundancy.enabled" class="connection-dialog__redundancy-grid">
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        故障超时
                        <el-tooltip
                          content="主 PLC 连续无响应超过该时间后，切换到备用 PLC。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number
                      v-model="formData.redundancy.failoverPolicy.timeoutMs"
                      :min="1000"
                      :step="1000"
                      class="w-full"
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        冷却时间
                        <el-tooltip
                          content="完成一次切换后，至少等待该时间再允许下一次切换。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number
                      v-model="formData.redundancy.failoverPolicy.cooldownMs"
                      :min="1000"
                      :step="1000"
                      class="w-full"
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        自动回切
                        <el-tooltip
                          content="主 PLC 恢复稳定后，自动从备用 PLC 切回主 PLC。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-switch v-model="formData.redundancy.failoverPolicy.autoFailback" />
                  </el-form-item>
                </div>
              </div>
              <el-collapse v-model="s7ActiveCollapse" class="connection-dialog__collapse mt-4">
                <el-collapse-item title="高级参数配置" name="advanced">
                  <div class="connection-dialog__form-grid">
                    <el-form-item>
                      <template #label>
                        <span class="connection-dialog__field-label">
                          PDU 长度
                          <el-tooltip
                            content="PLC 单次通信的数据包上限，常见值为 240、480、960。"
                            placement="top"
                          >
                            <IconTablerHelpCircle class="connection-dialog__field-help" />
                          </el-tooltip>
                        </span>
                      </template>
                      <el-input-number
                        v-model="formData.pduSize"
                        :min="0"
                        :step="120"
                        class="w-full"
                        placeholder="480"
                      />
                    </el-form-item>
                    <el-form-item label="本地 TSAP">
                      <el-input v-model="formData.localTsap" placeholder="可选，例如 0100" />
                    </el-form-item>
                  </div>
                  <div class="connection-dialog__form-grid">
                    <el-form-item label="远端 TSAP">
                      <el-input v-model="formData.remoteTsap" placeholder="可选，例如 0102" />
                    </el-form-item>
                    <el-form-item label="连接超时">
                      <el-input-number
                        v-model="formData.connectTimeoutMs"
                        :min="1000"
                        :step="1000"
                        class="w-full"
                      />
                    </el-form-item>
                    <el-form-item label="读取超时">
                      <el-input-number
                        v-model="formData.readTimeoutMs"
                        :min="1000"
                        :step="1000"
                        class="w-full"
                      />
                    </el-form-item>
                  </div>
                  <div class="connection-dialog__form-grid">
                    <el-form-item label="最大读取字节">
                      <el-input-number
                        v-model="formData.maxReadBytes"
                        :min="0"
                        :step="16"
                        class="w-full"
                      />
                    </el-form-item>
                    <el-form-item label="合并间隙字节">
                      <el-input-number
                        v-model="formData.maxGapBytes"
                        :min="0"
                        :step="1"
                        class="w-full"
                      />
                    </el-form-item>
                    <el-form-item label="并发读取数">
                      <el-input-number
                        v-model="formData.maxConcurrentReads"
                        :min="1"
                        :step="1"
                        class="w-full"
                      />
                    </el-form-item>
                  </div>
                  <div class="connection-dialog__checkbox-line">
                    <el-checkbox v-model="formData.allowAbsoluteAddress">允许绝对地址</el-checkbox>
                    <el-checkbox v-model="formData.allowSymbolAddress">允许符号地址</el-checkbox>
                  </div>
                  <el-form-item label="支持地址区">
                    <el-checkbox-group v-model="formData.supportedAreas">
                      <el-checkbox-button v-for="area in s7AreaOptions" :key="area" :label="area" />
                    </el-checkbox-group>
                  </el-form-item>
                </el-collapse-item>
              </el-collapse>
            </template>

            <template v-else-if="connectionType === 'modbus'">
              <el-form-item label="模式">
                <el-segmented v-model="formData.mode" :options="modbusModeOptions" />
              </el-form-item>
              <div v-if="formData.mode === 'tcp'" class="connection-dialog__form-grid">
                <el-form-item label="IP 地址" prop="ip">
                  <el-input v-model="formData.ip" placeholder="192.168.1.20" />
                </el-form-item>
                <el-form-item label="端口">
                  <el-input v-model.number="formData.port" inputmode="numeric" placeholder="502" />
                </el-form-item>
              </div>
              <div v-else>
                <div class="connection-dialog__form-grid">
                  <el-form-item label="串口" prop="serialPort">
                    <el-input v-model="formData.serialPort" placeholder="/dev/ttyUSB0" />
                  </el-form-item>
                  <el-form-item label="波特率">
                    <el-input-number
                      v-model="formData.baudRate"
                      :min="1200"
                      :step="1200"
                      class="w-full"
                    />
                  </el-form-item>
                </div>
                <div class="connection-dialog__form-grid">
                  <el-form-item label="数据位">
                    <el-select v-model="formData.dataBits" class="w-full">
                      <el-option :value="7" label="7" />
                      <el-option :value="8" label="8" />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="校验位">
                    <el-select v-model="formData.parity" class="w-full">
                      <el-option label="无校验" value="N" />
                      <el-option label="偶校验" value="E" />
                      <el-option label="奇校验" value="O" />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="停止位">
                    <el-select v-model="formData.stopBits" class="w-full">
                      <el-option :value="1" label="1" />
                      <el-option :value="2" label="2" />
                    </el-select>
                  </el-form-item>
                </div>
              </div>
              <div class="connection-dialog__form-section">
                <div class="connection-dialog__form-section-title">设备冗余</div>
                <div class="connection-dialog__switch-line">
                  <el-switch
                    v-model="formData.redundancy.enabled"
                    :disabled="formData.mode === 'rtu'"
                  />
                  <span class="connection-dialog__switch-text">
                    {{
                      formData.mode === 'rtu'
                        ? 'RTU 暂不启用设备冗余'
                        : formData.redundancy.enabled
                          ? '启用主备 TCP 网关'
                          : '不启用'
                    }}
                  </span>
                </div>
                <p v-if="formData.mode === 'rtu'" class="connection-dialog__field-tip">
                  当前 Linux 目标环境下 RTU 优先级较低，串口冗余后续在采集节点侧配置。
                </p>
                <div
                  v-if="formData.redundancy.enabled && formData.mode !== 'rtu'"
                  class="connection-dialog__redundancy-grid"
                >
                  <el-form-item
                    v-for="endpoint in formData.redundancy.endpoints"
                    :key="endpoint.role"
                    :label="endpoint.role === 'primary' ? '主网关' : '备网关'"
                  >
                    <div
                      v-if="endpoint.role === 'primary'"
                      class="connection-dialog__readonly-value"
                    >
                      {{ formatEndpoint(formData.ip, formData.port) }}
                    </div>
                    <div v-else class="connection-dialog__endpoint-row">
                      <el-input v-model="endpoint.host" placeholder="备用网关 IP" />
                      <el-input-number v-model="endpoint.port" :min="1" class="w-full" />
                    </div>
                  </el-form-item>
                </div>
                <div
                  v-if="formData.redundancy.enabled && formData.mode !== 'rtu'"
                  class="connection-dialog__redundancy-grid"
                >
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        故障超时
                        <el-tooltip
                          content="主网关连续无响应超过该时间后，切换到备用网关。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number
                      v-model="formData.redundancy.failoverPolicy.timeoutMs"
                      :min="1000"
                      :step="1000"
                      class="w-full"
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        冷却时间
                        <el-tooltip
                          content="完成一次切换后，至少等待该时间再允许下一次切换。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-input-number
                      v-model="formData.redundancy.failoverPolicy.cooldownMs"
                      :min="1000"
                      :step="1000"
                      class="w-full"
                    />
                  </el-form-item>
                  <el-form-item>
                    <template #label>
                      <span class="connection-dialog__field-label">
                        自动回切
                        <el-tooltip
                          content="主网关恢复稳定后，自动从备用网关切回主网关。"
                          placement="top"
                        >
                          <IconTablerHelpCircle class="connection-dialog__field-help" />
                        </el-tooltip>
                      </span>
                    </template>
                    <el-switch v-model="formData.redundancy.failoverPolicy.autoFailback" />
                  </el-form-item>
                </div>
              </div>
              <el-collapse v-model="modbusActiveCollapse" class="connection-dialog__collapse mt-4">
                <el-collapse-item title="高级参数配置" name="advanced">
                  <div class="connection-dialog__form-grid">
                    <el-form-item label="连接超时">
                      <el-input-number
                        v-model="formData.connectTimeoutMs"
                        :min="1000"
                        :step="1000"
                        class="w-full"
                      />
                    </el-form-item>
                    <el-form-item label="请求超时">
                      <el-input-number
                        v-model="formData.requestTimeoutMs"
                        :min="1000"
                        :step="1000"
                        class="w-full"
                      />
                    </el-form-item>
                  </div>
                  <div class="connection-dialog__form-grid">
                    <el-form-item label="重试次数">
                      <el-input-number
                        v-model="formData.retries"
                        :min="0"
                        :max="10"
                        class="w-full"
                      />
                    </el-form-item>
                    <el-form-item label="请求间隔">
                      <el-input-number
                        v-model="formData.requestIntervalMs"
                        :min="0"
                        :step="50"
                        class="w-full"
                      />
                    </el-form-item>
                  </div>
                  <div class="connection-dialog__form-grid">
                    <el-form-item label="最大连续读取">
                      <el-input-number
                        v-model="formData.maxReadQuantity"
                        :min="1"
                        :max="2000"
                        class="w-full"
                      />
                    </el-form-item>
                    <el-form-item label="最大批量请求">
                      <el-input-number
                        v-model="formData.maxBatchRequests"
                        :min="1"
                        :max="1000"
                        class="w-full"
                      />
                    </el-form-item>
                  </div>
                  <div class="connection-dialog__form-grid">
                    <el-form-item label="地址基准">
                      <el-select v-model="formData.addressBase" class="w-full">
                        <el-option label="Modicon 地址 40001/30001" value="modicon" />
                        <el-option label="1 基地址" value="one_based" />
                        <el-option label="0 基地址" value="zero_based" />
                      </el-select>
                    </el-form-item>
                    <el-form-item label="默认从站号">
                      <el-input-number
                        v-model="formData.slaveId"
                        :min="0"
                        :max="247"
                        class="w-full"
                      />
                    </el-form-item>
                  </div>
                  <div class="connection-dialog__form-grid">
                    <el-form-item label="默认字节序">
                      <el-select v-model="formData.byteOrder" class="w-full">
                        <el-option label="ABCD" value="ABCD" />
                        <el-option label="BADC" value="BADC" />
                        <el-option label="CDAB" value="CDAB" />
                        <el-option label="DCBA" value="DCBA" />
                      </el-select>
                    </el-form-item>
                    <el-form-item label="默认字序">
                      <el-select v-model="formData.wordOrder" class="w-full">
                        <el-option label="高字在前" value="high_first" />
                        <el-option label="低字在前" value="low_first" />
                      </el-select>
                    </el-form-item>
                    <el-form-item label="默认轮询周期">
                      <el-input-number
                        v-model="formData.pollIntervalMs"
                        :min="100"
                        :step="100"
                        class="w-full"
                      />
                    </el-form-item>
                  </div>
                  <p class="connection-dialog__field-tip">
                    默认从站和默认采集参数只用于新建变量、批量导入时带入，不限制该接入源只能访问一个从站。
                  </p>
                </el-collapse-item>
              </el-collapse>
            </template>

            <template v-else-if="connectionType === 'tdengine'">
              <div class="connection-dialog__form-grid">
                <el-form-item label="IP 地址" prop="ip">
                  <el-input v-model="formData.ip" placeholder="127.0.0.1" />
                </el-form-item>
                <el-form-item label="端口">
                  <el-input v-model.number="formData.port" inputmode="numeric" placeholder="6030" />
                </el-form-item>
              </div>
              <div class="connection-dialog__form-grid">
                <el-form-item label="Database" prop="database">
                  <el-input v-model="formData.database" placeholder="iot_data" />
                </el-form-item>
                <el-form-item label="时区">
                  <el-input v-model="formData.timezone" placeholder="Asia/Shanghai" />
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
const opcuaActiveCollapse = ref([])
const s7ActiveCollapse = ref([])
const modbusActiveCollapse = ref([])
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
    description: 'Key 扫描与值样本',
    icon: markRaw(IconTablerDatabase),
  },
  {
    value: 'opcua',
    label: 'OPC UA',
    description: 'IP、端口与安全策略',
    icon: markRaw(IconTablerServer),
  },
  {
    value: 's7',
    label: 'Siemens S7',
    description: 'PLC 类型与通信参数',
    icon: markRaw(IconTablerServer),
  },
  {
    value: 'modbus',
    label: 'Modbus',
    description: 'TCP/RTU 网关连接',
    icon: markRaw(IconTablerWebhook),
  },
  {
    value: 'tdengine',
    label: 'TDengine',
    description: '时序库连接契约',
    icon: markRaw(IconTablerDatabase),
  },
]
const sourceOptions = [...builtinSourceOptions, ...externalSourceOptions]

const previewProtocolTypes = ['kafka', 'http', 'websocket', 'redis']
const industrialProtocolTypes = ['opcua', 's7', 'modbus', 'tdengine']
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
  { label: 'Standalone', value: 'standalone' },
  { label: 'Sentinel', value: 'sentinel' },
  { label: 'Cluster', value: 'cluster' },
]
const opcuaSecurityModeOptions = [
  { label: 'None', value: 'none' },
  { label: 'Sign', value: 'sign' },
  { label: 'Sign & Encrypt', value: 'signandencrypt' },
]
const opcuaAuthOptions = [
  { label: '匿名', value: 'anonymous' },
  { label: '用户名密码', value: 'username_password' },
]
const modbusModeOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'RTU', value: 'rtu' },
]
const s7FamilyOptions = [
  'S7-200',
  'S7-200 SMART',
  'S7-300',
  'S7-400',
  'S7-1200',
  'S7-1500',
  'S7 Compatible',
]
const s7AreaOptions = ['DB', 'M', 'I', 'Q']

const protocolRules = {
  name: [{ required: true, message: '连接名称不能为空', trigger: 'blur' }],
  brokers: [{ required: true, message: '服务器地址不能为空', trigger: 'blur' }],
  url: [{ required: true, message: '连接地址不能为空', trigger: 'blur' }],
  address: [{ required: true, message: 'Redis 地址不能为空', trigger: 'blur' }],
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
  if (connectionType.value === 'redis') return 'Redis Source'
  if (connectionType.value === 'opcua') return 'OPC UA Server'
  if (connectionType.value === 's7') return 'Siemens S7 PLC'
  if (connectionType.value === 'modbus') return 'Modbus Device'
  if (connectionType.value === 'tdengine') return 'TDengine Source'
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
const isSimpleMetadataSource = computed(
  () => isBuiltinStoreSelected.value || ['http', 'websocket'].includes(connectionType.value),
)
const showTestButton = computed(() => !isSimpleMetadataSource.value)
const isKafkaSaslEnabled = computed(() =>
  String(formData.value.securityProtocol || '').includes('SASL'),
)

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
      { label: '模式', value: data.mode || 'standalone' },
      { label: '地址', value: data.address || '未填写' },
      { label: 'DB', value: String(data.db ?? 0) },
      { label: 'Key', value: data.keyPattern || '*' },
    )
  } else if (connectionType.value === 'opcua') {
    rows.push(
      { label: '地址', value: formatEndpoint(data.ip, data.port) },
      { label: '设备冗余', value: formatRedundancySummary(data.redundancy) },
      {
        label: '安全',
        value: `${data.securityPolicy || 'None'} / ${data.securityMode || 'none'}`,
      },
      {
        label: '认证',
        value: data.authType === 'username_password' ? '用户名/密码' : '匿名',
      },
    )
  } else if (connectionType.value === 's7') {
    rows.push(
      { label: '地址', value: formatEndpoint(data.ip, data.port) },
      { label: '设备冗余', value: formatRedundancySummary(data.redundancy) },
      { label: 'PLC 系列', value: data.plcFamily || 'S7 Compatible' },
      { label: '机架/槽位', value: `${data.rack ?? 0}/${data.slot ?? 1}` },
    )
  } else if (connectionType.value === 'modbus') {
    rows.push(
      { label: '模式', value: (data.mode || 'tcp').toUpperCase() },
      {
        label: '地址',
        value: data.mode === 'rtu' ? '串口 RTU' : formatEndpoint(data.ip, data.port),
      },
      { label: '设备冗余', value: formatRedundancySummary(data.redundancy) },
      { label: '从站与寄存器', value: '进入工作台维护' },
    )
  } else if (connectionType.value === 'tdengine') {
    rows.push(
      { label: '地址', value: formatEndpoint(data.ip, data.port) },
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
            : connectionType.value === 'opcua'
              ? Boolean(data.ip && data.port)
              : connectionType.value === 'modbus' && data.mode === 'rtu'
                ? Boolean(data.serialPort)
                : connectionType.value === 'tdengine'
                  ? Boolean(data.ip && data.port)
                  : relationalSourceTypes.includes(connectionType.value)
                    ? Boolean(data.host && data.port)
                    : Boolean(data.ip && data.port)
  const hasTarget = ['mqtt', 'http', 'websocket', 'redis', 'opcua'].includes(connectionType.value)
    ? true
    : connectionType.value === 'kafka'
      ? true
      : connectionType.value === 's7'
        ? data.rack !== undefined && data.slot !== undefined && Boolean(data.plcFamily)
        : connectionType.value === 'modbus'
          ? true
          : relationalSourceTypes.includes(connectionType.value) ||
              connectionType.value === 'tdengine'
            ? Boolean(data.database)
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
            ? '保存为节点侧运行配置'
            : '连接测试可选完成',
      ready:
        ['http', 'websocket'].includes(connectionType.value) ||
        (previewProtocolTypes.includes(connectionType.value) && connectionType.value !== 'kafka') ||
        industrialProtocolTypes.includes(connectionType.value) ||
        (testResult.value.status === 'success' && !isTestStale.value),
    },
  ]
})

// 监听 OPC UA 的安全模式/安全策略/证书联动
watch(
  () =>
    [connectionType.value, formData.value?.securityMode, formData.value?.securityPolicy] as const,
  ([type, secMode, secPolicy], oldVal) => {
    if (type !== 'opcua' || !formData.value) return
    const [_, oldSecMode, oldSecPolicy] = oldVal || []

    // 1. 安全模式与安全策略的互斥与推导
    if (secMode !== oldSecMode) {
      if (secMode === 'none') {
        if (formData.value.securityPolicy !== 'None') {
          formData.value.securityPolicy = 'None'
        }
      } else if (secMode === 'sign' || secMode === 'signandencrypt') {
        if (formData.value.securityPolicy === 'None') {
          formData.value.securityPolicy = 'Basic256Sha256'
        }
      }
    } else if (secPolicy !== oldSecPolicy) {
      if (secPolicy === 'None') {
        if (formData.value.securityMode !== 'none') {
          formData.value.securityMode = 'none'
        }
      } else if (secPolicy === 'Basic256Sha256' || secPolicy === 'Basic256') {
        if (formData.value.securityMode === 'none') {
          formData.value.securityMode = 'signandencrypt'
        }
      }
    }

    // 2. 安全模式与证书折叠面板的展开联动
    if (secMode === 'sign' || secMode === 'signandencrypt') {
      if (!opcuaActiveCollapse.value.includes('ssl')) {
        opcuaActiveCollapse.value = [...opcuaActiveCollapse.value, 'ssl']
      }
    } else if (secMode === 'none' && opcuaActiveCollapse.value.includes('ssl')) {
      opcuaActiveCollapse.value = opcuaActiveCollapse.value.filter((name) => name !== 'ssl')
    }
  },
  { deep: true },
)

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
      ...connection.relationalConfig,
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
  if (!config.redundancy && connection?.redundancy && typeof connection.redundancy === 'object') {
    config.redundancy = connection.redundancy
  }
  return config
}

const handleValidate = (valid, data) => {
  // 表单验证回调
  console.log('表单验证:', valid, data)
}

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

  if (!relationalSourceTypes.includes(connectionType.value)) {
    if (['s7', 'tdengine'].includes(connectionType.value)) {
      testResult.value = {
        status: 'idle',
        title: '节点侧运行配置',
        message: '工业协议已在开发态保存配置契约，真实连通与采集由节点侧工业协议运行器执行。',
        detail: '',
        durationMs: 0,
      }
      ElMessage.info('工业协议保存后进入节点侧运行联调')
      return
    }
  }

  testing.value = true
  const startAt = performance.now()
  try {
    const payload = buildConnectionPayload()
    const response = await dataAPI.testConnection(props.projectId, {
      type: payload.type,
      config: payload.config,
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
  // 清空表单
  if (formRef.value) {
    formRef.value.clearValidate()
  }
  if (protocolFormRef.value) {
    protocolFormRef.value.clearValidate()
  }
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
    },
    opcua: {
      name: '',
      ip: '127.0.0.1',
      port: 4840,
      securityPolicy: 'None',
      securityMode: 'none',
      authType: 'anonymous',
      username: '',
      password: '',
      samplingMs: 1000,
      sessionName: 'InduForge_Session',
      namespaceUrl: '',
      connectionTimeoutMs: 5000,
      requestTimeoutMs: 5000,
      sslConfig: {
        ca: '',
        cert: '',
        key: '',
        rejectUnauthorized: false,
      },
      redundancy: defaultRedundancyConfig(4840),
    },
    s7: {
      name: '',
      ip: '',
      port: 102,
      plcFamily: 'S7 Compatible',
      communicationMode: 'rack_slot',
      rack: 0,
      slot: 1,
      pollIntervalMs: 1000,
      pduSize: 480,
      maxReadBytes: 0,
      maxGapBytes: 8,
      maxConcurrentReads: 1,
      localTsap: '',
      remoteTsap: '',
      connectTimeoutMs: 3000,
      readTimeoutMs: 3000,
      byteOrder: 'big_endian',
      wordOrder: 'big_endian',
      optimizedBlockAccess: false,
      allowAbsoluteAddress: true,
      allowSymbolAddress: false,
      supportedAreas: ['DB', 'M', 'I', 'Q'],
      redundancy: defaultRedundancyConfig(102),
    },
    modbus: {
      name: '',
      mode: 'tcp',
      ip: '',
      port: 502,
      serialPort: '/dev/ttyUSB0',
      baudRate: 9600,
      dataBits: 8,
      parity: 'N',
      stopBits: 1,
      connectTimeoutMs: 5000,
      requestTimeoutMs: 5000,
      retries: 1,
      requestIntervalMs: 0,
      maxReadQuantity: 125,
      maxBatchRequests: 64,
      addressBase: 'modicon',
      byteOrder: 'ABCD',
      wordOrder: 'high_first',
      pollIntervalMs: 1000,
      slaveId: 1,
      redundancy: defaultRedundancyConfig(502),
    },
    tdengine: {
      name: '',
      ip: '127.0.0.1',
      port: 6030,
      database: '',
      username: 'root',
      password: 'taosdata',
      timezone: 'Asia/Shanghai',
      optionsText: '{}',
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

const normalizeS7OptionsForForm = (options = {}) => ({
  pduSize: Number(options.pduSize) || 480,
  localTsap: options.localTsap || '',
  remoteTsap: options.remoteTsap || '',
  connectTimeoutMs: Number(options.connectTimeoutMs || options.connectionTimeoutMs) || 3000,
  readTimeoutMs: Number(options.readTimeoutMs || options.requestTimeoutMs) || 3000,
  maxReadBytes: Number(options.maxReadBytes) || 0,
  maxGapBytes: Number(options.maxGapBytes) || 8,
  maxConcurrentReads: Number(options.maxConcurrentReads) || 1,
  byteOrder: options.byteOrder || 'big_endian',
  wordOrder: options.wordOrder || 'big_endian',
  optimizedBlockAccess: options.optimizedBlockAccess === true,
  allowAbsoluteAddress: options.allowAbsoluteAddress !== false,
  allowSymbolAddress: options.allowSymbolAddress === true,
  supportedAreas:
    Array.isArray(options.supportedAreas) && options.supportedAreas.length
      ? [...options.supportedAreas]
      : ['DB', 'M', 'I', 'Q'],
})

const buildS7Options = (config) => {
  const options: Record<string, any> = {
    pduSize: Number(config.pduSize) || 480,
    connectTimeoutMs: Number(config.connectTimeoutMs) || 3000,
    readTimeoutMs: Number(config.readTimeoutMs) || 3000,
    maxReadBytes: Number(config.maxReadBytes) || null,
    maxGapBytes: Number(config.maxGapBytes) || 8,
    maxConcurrentReads: Number(config.maxConcurrentReads) || 1,
    byteOrder: config.byteOrder || 'big_endian',
    wordOrder: config.wordOrder || 'big_endian',
    optimizedBlockAccess: config.optimizedBlockAccess === true,
    allowAbsoluteAddress: config.allowAbsoluteAddress !== false,
    allowSymbolAddress: config.allowSymbolAddress === true,
    supportedAreas:
      Array.isArray(config.supportedAreas) && config.supportedAreas.length
        ? [...config.supportedAreas]
        : ['DB', 'M', 'I', 'Q'],
  }
  if (String(config.localTsap || '').trim()) {
    options.localTsap = String(config.localTsap).trim()
  }
  if (String(config.remoteTsap || '').trim()) {
    options.remoteTsap = String(config.remoteTsap).trim()
  }
  return options
}

const applyS7FamilyDefaults = () => {
  if (!formData.value) return
  const family = formData.value.plcFamily
  if (family === 'S7-300' || family === 'S7-400') {
    Object.assign(formData.value, {
      communicationMode: 'rack_slot',
      rack: 0,
      slot: 2,
      maxGapBytes: 8,
      optimizedBlockAccess: false,
    })
  } else if (family === 'S7-1200' || family === 'S7-1500') {
    Object.assign(formData.value, {
      communicationMode: 'rack_slot',
      rack: 0,
      slot: 1,
      maxGapBytes: 16,
      optimizedBlockAccess: true,
    })
  } else if (family === 'S7-200' || family === 'S7-200 SMART') {
    Object.assign(formData.value, {
      communicationMode: 'tsap',
      maxGapBytes: 8,
      optimizedBlockAccess: false,
    })
  } else {
    Object.assign(formData.value, {
      communicationMode: 'rack_slot',
      rack: 0,
      slot: 1,
      maxGapBytes: 8,
      optimizedBlockAccess: false,
    })
  }
}

const normalizeModbusOptionsForForm = (options = {}) => ({
  connectTimeoutMs: Number(options.connectTimeoutMs) || 5000,
  requestTimeoutMs: Number(options.requestTimeoutMs) || 5000,
  retries: Number(options.retries ?? 1),
  requestIntervalMs: Number(options.requestIntervalMs) || 0,
  maxReadQuantity: Number(options.maxReadQuantity) || 125,
  maxBatchRequests: Number(options.maxBatchRequests) || 64,
  addressBase: options.addressBase || 'modicon',
  byteOrder: options.byteOrder || 'ABCD',
  wordOrder: options.wordOrder || 'high_first',
})

const buildModbusOptions = (config) => ({
  connectTimeoutMs: Number(config.connectTimeoutMs) || 5000,
  requestTimeoutMs: Number(config.requestTimeoutMs) || 5000,
  retries: Number(config.retries ?? 1),
  requestIntervalMs: Number(config.requestIntervalMs) || 0,
  maxReadQuantity: Number(config.maxReadQuantity) || 125,
  maxBatchRequests: Number(config.maxBatchRequests) || 64,
  addressBase: config.addressBase || 'modicon',
  byteOrder: config.byteOrder || 'ABCD',
  wordOrder: config.wordOrder || 'high_first',
})

const normalizeModbusSerialConfigForForm = (serialConfig = {}) => ({
  serialPort: serialConfig.port || serialConfig.serialPort || '/dev/ttyUSB0',
  baudRate: Number(serialConfig.baudRate) || 9600,
  dataBits: Number(serialConfig.dataBits) || 8,
  parity: serialConfig.parity || 'N',
  stopBits: Number(serialConfig.stopBits) || 1,
})

const buildModbusSerialConfig = (config) => ({
  port: String(config.serialPort || '').trim(),
  baudRate: Number(config.baudRate) || 9600,
  dataBits: Number(config.dataBits) || 8,
  parity: String(config.parity || 'N').toUpperCase(),
  stopBits: Number(config.stopBits) || 1,
})

const normalizeProtocolFormData = (type, config) => {
  const data = { ...getProtocolDefaultConfig(type), ...config }
  if (type === 'kafka' && config.options && typeof config.options === 'object') {
    Object.assign(data, normalizeKafkaOptionsForForm(config.options))
  }
  if (type === 'opcua') {
    if (config.endpoint) {
      Object.assign(data, parseOpcuaEndpoint(config.endpoint))
    }
    data.securityMode = normalizeOpcuaSecurityModeForForm(data.securityMode)
    if (data.securityMode === 'none') {
      data.securityPolicy = 'None'
    } else if (!data.securityPolicy || data.securityPolicy === 'None') {
      data.securityPolicy = 'Basic256Sha256'
    }
    const options = config.options || {}
    data.sessionName = options.sessionName || 'InduForge_Session'
    data.namespaceUrl = options.namespaceUrl || ''
    data.connectionTimeoutMs = options.connectionTimeoutMs || 5000
    data.requestTimeoutMs = options.requestTimeoutMs || 5000

    const ssl = options.sslConfig || {}
    data.sslConfig = {
      ca: ssl.ca || '',
      cert: ssl.cert || '',
      key: ssl.key || '',
      rejectUnauthorized: ssl.rejectUnauthorized === true,
    }
  }
  if (industrialProtocolTypes.includes(type)) {
    data.redundancy = normalizeRedundancyForForm(type, data, config.redundancy)
  }
  if (['s7', 'modbus'].includes(type) && config.host) {
    data.ip = config.host
  }
  if (type === 's7' && config.options && typeof config.options === 'object') {
    Object.assign(data, normalizeS7OptionsForForm(config.options))
  }
  if (type === 'modbus') {
    if (config.options && typeof config.options === 'object') {
      Object.assign(data, normalizeModbusOptionsForForm(config.options))
    }
    if (config.serialConfig && typeof config.serialConfig === 'object') {
      Object.assign(data, normalizeModbusSerialConfigForForm(config.serialConfig))
    }
  }
  if (type === 'tdengine' && config.dsn) {
    Object.assign(data, parseTdengineDsn(config.dsn))
  }
  if (data.headers && typeof data.headers === 'object') {
    data.headersText = JSON.stringify(data.headers, null, 2)
  }
  if (data.options && typeof data.options === 'object') {
    if (!['kafka', 's7', 'modbus'].includes(type)) {
      data.optionsText = JSON.stringify(data.options, null, 2)
    }
    data.masterName = data.options.masterName || data.masterName
  }
  if (data.serialConfig && typeof data.serialConfig === 'object' && type !== 'modbus') {
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
    delete config.securityProtocol
    delete config.saslMechanism
    delete config.username
    delete config.password
    delete config.clientId
    delete config.dialTimeoutMs
    delete config.requestTimeoutMs
    delete config.sslConfig
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
    if (config.masterName) {
      options.masterName = config.masterName
    }
    config.options = options
    delete config.masterName
    return
  }
  if (type === 'opcua') {
    const defaultSessionName = `InduForge-Client-${config.name || 'OPCUA'}`
    config.options = {
      sessionName:
        config.sessionName === 'InduForge_Session' || !config.sessionName
          ? defaultSessionName
          : config.sessionName,
      namespaceUrl: config.namespaceUrl || '',
      connectionTimeoutMs: Number(config.connectionTimeoutMs) || 5000,
      requestTimeoutMs: Number(config.requestTimeoutMs) || 5000,
      sslConfig: {
        ca: config.sslConfig?.ca || '',
        cert: config.sslConfig?.cert || '',
        key: config.sslConfig?.key || '',
        rejectUnauthorized: config.sslConfig?.rejectUnauthorized === true,
      },
    }
    config.endpoint = buildOpcuaEndpoint(config.ip, config.port)
    config.redundancy = normalizeRedundancyForSubmit('opcua', config)
    if (config.authType !== 'username_password') {
      delete config.username
      delete config.password
    }
    delete config.ip
    delete config.port
    delete config.sessionName
    delete config.namespaceUrl
    delete config.connectionTimeoutMs
    delete config.requestTimeoutMs
    delete config.sslConfig
    return
  }
  if (type === 's7') {
    config.options = buildS7Options(config)
    config.redundancy = normalizeRedundancyForSubmit('s7', config)
    config.host = config.ip
    delete config.ip
    delete config.pduSize
    delete config.maxReadBytes
    delete config.maxGapBytes
    delete config.maxConcurrentReads
    delete config.localTsap
    delete config.remoteTsap
    delete config.connectTimeoutMs
    delete config.readTimeoutMs
    delete config.byteOrder
    delete config.wordOrder
    delete config.optimizedBlockAccess
    delete config.allowAbsoluteAddress
    delete config.allowSymbolAddress
    delete config.supportedAreas
    return
  }
  if (type === 'modbus') {
    config.options = buildModbusOptions(config)
    config.redundancy = normalizeRedundancyForSubmit('modbus', config)
    if (config.mode === 'rtu') {
      config.serialConfig = buildModbusSerialConfig(config)
      config.redundancy = {
        enabled: false,
        mode: 'none',
        endpoints: [],
        failoverPolicy: defaultFailoverPolicy(),
      }
      delete config.ip
      delete config.port
    } else {
      config.host = config.ip
      delete config.ip
      delete config.serialConfig
    }
    delete config.serialPort
    delete config.baudRate
    delete config.dataBits
    delete config.parity
    delete config.stopBits
    delete config.connectTimeoutMs
    delete config.requestTimeoutMs
    delete config.retries
    delete config.requestIntervalMs
    delete config.maxReadQuantity
    delete config.maxBatchRequests
    delete config.addressBase
    delete config.byteOrder
    delete config.wordOrder
    return
  }
  if (type === 'tdengine') {
    config.options = parseOptionalJsonObject(config.optionsText, '扩展参数')
    config.dsn = buildTdengineDsn(config)
    delete config.ip
    delete config.port
    delete config.username
    delete config.password
    if (!config.timezone) delete config.timezone
    delete config.optionsText
  }
}

const buildOpcuaEndpoint = (ip, port) => {
  const safeIp = String(ip || '').trim()
  const safePort = Number(port) || 4840
  return `opc.tcp://${safeIp}:${safePort}`
}

const parseOpcuaEndpoint = (endpoint) => {
  const text = String(endpoint || '').trim()
  const match = text.match(/^opc\.tcp:\/\/([^:/]+)(?::(\d+))?/i)
  if (!match) return {}
  return {
    ip: match[1],
    port: match[2] ? Number(match[2]) : 4840,
  }
}

const normalizeOpcuaSecurityModeForForm = (value) => {
  const normalized = String(value || '')
    .trim()
    .toLowerCase()
    .replaceAll('_', '')
  if (normalized === 'sign') return 'sign'
  if (normalized === 'signandencrypt') return 'signandencrypt'
  return 'none'
}

const buildTdengineDsn = (config) => {
  const username = String(config.username || 'root').trim()
  const password = String(config.password || 'taosdata').trim()
  const ip = String(config.ip || '').trim()
  const port = Number(config.port) || 6030
  return `${username}:${password}@tcp(${ip}:${port})/`
}

const parseTdengineDsn = (dsn) => {
  const text = String(dsn || '').trim()
  const match = text.match(/^(.*?):(.*?)@tcp\(([^:)]+)(?::(\d+))?\)\//i)
  if (!match) return {}
  return {
    username: match[1] || 'root',
    password: match[2] || 'taosdata',
    ip: match[3],
    port: match[4] ? Number(match[4]) : 6030,
  }
}

const parseOptionalJsonObject = (value, label) => {
  const text = String(value || '').trim()
  if (!text) return {}
  try {
    const parsed = JSON.parse(text)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed
    }
  } catch {
    // 统一在下方返回带字段名的可读错误。
  }
  throw new Error(`${label} 必须是 JSON 对象`)
}

function defaultFailoverPolicy() {
  return {
    timeoutMs: 3000,
    cooldownMs: 10000,
    autoFailback: false,
    stableDurationMs: 30000,
  }
}

function defaultRedundancyConfig(defaultPort = 0) {
  return {
    enabled: false,
    mode: 'priority_failover',
    endpoints: [
      {
        id: 'primary',
        name: '主路径',
        role: 'primary',
        host: '',
        port: defaultPort,
        priority: 1,
        enabled: true,
        healthCheck: true,
      },
      {
        id: 'standby',
        name: '备用路径',
        role: 'standby',
        host: '',
        port: defaultPort,
        priority: 2,
        enabled: true,
        healthCheck: true,
      },
    ],
    failoverPolicy: defaultFailoverPolicy(),
  }
}

function normalizeRedundancyForForm(type: string, data: Record<string, any>, source: any) {
  const fallbackPort = type === 'opcua' ? 4840 : type === 's7' ? 102 : 502
  const base = defaultRedundancyConfig(Number(data.port) || fallbackPort)
  const sourceEndpoints = Array.isArray(source?.endpoints) ? source.endpoints : []
  const primary = sourceEndpoints.find((endpoint) => endpoint?.role === 'primary')
  const standby = sourceEndpoints.find((endpoint) => endpoint?.role === 'standby')
  const endpoints = base.endpoints.map((endpoint) => {
    const matched = endpoint.role === 'primary' ? primary : standby
    const fallbackHost = endpoint.role === 'primary' ? data.host || data.ip || '' : ''
    return {
      ...endpoint,
      ...(matched || {}),
      host: matched?.host || matched?.endpoint || fallbackHost,
      port: Number(matched?.port || data.port || endpoint.port || fallbackPort),
    }
  })
  return {
    ...base,
    ...(source || {}),
    enabled: source?.enabled === true,
    mode: source?.mode || 'priority_failover',
    endpoints,
    failoverPolicy: {
      ...base.failoverPolicy,
      ...(source?.failoverPolicy || {}),
    },
  }
}

function normalizeRedundancyForSubmit(type: string, config: Record<string, any>) {
  const redundancy = normalizeRedundancyForForm(type, config, config.redundancy)
  if (!redundancy.enabled) {
    return {
      enabled: false,
      mode: 'none',
      endpoints: [],
      failoverPolicy: defaultFailoverPolicy(),
    }
  }
  const labels = redundancyLabels(type)
  const endpoints = redundancy.endpoints.map((endpoint, index) => ({
    id: endpoint.id || endpoint.role || `endpoint-${index + 1}`,
    name: endpoint.name || (endpoint.role === 'primary' ? '主路径' : '备用路径'),
    role: endpoint.role || (index === 0 ? 'primary' : 'standby'),
    host:
      endpoint.role === 'primary'
        ? String(config.ip || config.host || '').trim()
        : String(endpoint.host || '').trim(),
    port:
      endpoint.role === 'primary'
        ? Number(config.port || 0)
        : Number(endpoint.port || config.port || 0),
    priority: Number(endpoint.priority || index + 1),
    enabled: endpoint.enabled !== false,
    healthCheck: endpoint.healthCheck !== false,
  }))
  const primary = endpoints.find((endpoint) => endpoint.role === 'primary')
  const standby = endpoints.find((endpoint) => endpoint.role === 'standby')
  if (!primary?.host) {
    throw new Error(`请填写${labels.primaryBase}地址`)
  }
  if (!standby?.host) {
    throw new Error(`请填写${labels.standby} IP`)
  }
  if (!primary.port || primary.port < 1 || primary.port > 65535) {
    throw new Error(`请填写有效的${labels.primaryBase}端口`)
  }
  if (!standby.port || standby.port < 1 || standby.port > 65535) {
    throw new Error(`请填写有效的${labels.standby}端口`)
  }
  return {
    enabled: true,
    mode: 'priority_failover',
    endpoints,
    failoverPolicy: {
      ...defaultFailoverPolicy(),
      ...(redundancy.failoverPolicy || {}),
    },
  }
}

function redundancyLabels(type: string) {
  if (type === 's7') {
    return { primaryBase: 'PLC', standby: '备用 PLC' }
  }
  if (type === 'modbus') {
    return { primaryBase: '网关', standby: '备用网关' }
  }
  return { primaryBase: '服务器', standby: '备用服务器' }
}

const formatEndpoint = (host, port) => {
  const safeHost = host || '未填写'
  return port ? `${safeHost}:${port}` : safeHost
}

const formatRedundancySummary = (redundancy: any) => {
  if (!redundancy?.enabled) return '无'
  const endpoints = Array.isArray(redundancy.endpoints) ? redundancy.endpoints : []
  const enabledCount = endpoints.filter((endpoint) => endpoint?.enabled !== false).length
  return enabledCount > 1 ? `主备已配置(${enabledCount})` : '待补备用路径'
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
