<template>
  <div class="init-wizard">
    <div class="wizard-container">
      <el-card class="wizard-card" shadow="always">
        <template #header>
          <div class="card-header">
            <h1>NodeAgent 初始化向导</h1>
            <p class="subtitle">首次运行配置</p>
          </div>
        </template>

        <!-- 步骤指示器 -->
        <el-steps :active="currentStep" align-center finish-status="success">
          <el-step title="选择模式" />
          <el-step title="配置信息" />
          <el-step :title="wizardData.mode === 'online' ? '等待审批' : '完成配置'" />
          <el-step title="初始化完成" />
        </el-steps>

        <!-- 步骤内容 -->
        <div class="step-content">
          <!-- 步骤 0: 选择模式 -->
          <div v-show="currentStep === 0" class="step-container">
            <h3 class="step-title">选择节点运行模式</h3>
            <div class="mode-selection">
              <el-card
                :class="['mode-card', { selected: wizardData.mode === 'online' }]"
                @click="selectMode('online')"
                shadow="hover"
              >
                <div class="mode-icon">🌐</div>
                <h4>在线模式</h4>
                <ul class="mode-features">
                  <li>✓ 受运维中心统一管控</li>
                  <li>✓ 支持远程部署和操作</li>
                  <li>✓ 实时监控和告警</li>
                  <li>✓ 自动接收部署指令</li>
                </ul>
                <div class="mode-note">推荐用于生产环境</div>
              </el-card>

              <el-card
                :class="['mode-card', { selected: wizardData.mode === 'offline' }]"
                @click="selectMode('offline')"
                shadow="hover"
              >
                <div class="mode-icon">💻</div>
                <h4>离线模式</h4>
                <ul class="mode-features">
                  <li>✓ 独立运行，不连接运维中心</li>
                  <li>✓ 手动导入工程包</li>
                  <li>✓ 本地管理启停</li>
                  <li>✓ 适合隔离网络环境</li>
                </ul>
                <div class="mode-note">适合开发测试</div>
              </el-card>
            </div>
            <el-alert
              v-if="wizardData.mode"
              type="warning"
              :closable="false"
              show-icon
              class="mt-4"
            >
              <template #title>
                模式选择后将无法修改，请谨慎选择
              </template>
            </el-alert>
          </div>

          <!-- 步骤 1: 在线模式配置 -->
          <div v-show="currentStep === 1 && wizardData.mode === 'online'" class="step-container">
            <h3 class="step-title">在线模式配置</h3>
            <el-form
              ref="onlineFormRef"
              :model="wizardData.online"
              :rules="onlineRules"
              label-width="120px"
              class="config-form"
            >
              <el-divider content-position="left">运维中心连接</el-divider>
              <el-form-item label="运维中心地址" prop="centerUrl">
                <el-input
                  v-model="wizardData.online.centerUrl"
                  placeholder="http://192.168.1.100:9099"
                  clearable
                >
                  <template #append>
                    <el-button @click="testConnection" :loading="testingConnection">
                      测试连接
                    </el-button>
                  </template>
                </el-input>
                <div class="form-hint">请输入运维中心的完整地址（包含 http:// 或 https://）</div>
              </el-form-item>

              <el-divider content-position="left">用户认证</el-divider>
              <el-form-item label="用户名" prop="username">
                <el-input
                  v-model="wizardData.online.username"
                  placeholder="请输入用户名"
                  clearable
                />
              </el-form-item>
              <el-form-item label="密码" prop="password">
                <el-input
                  v-model="wizardData.online.password"
                  type="password"
                  placeholder="请输入密码"
                  show-password
                  clearable
                />
                <div class="form-hint">用于验证身份，具有运维权限的用户将自动审批通过</div>
              </el-form-item>

              <el-divider content-position="left">节点信息</el-divider>
              <el-form-item label="节点名称" prop="nodeName">
                <el-input
                  v-model="wizardData.online.nodeName"
                  placeholder="production-node-01"
                  clearable
                  @input="validateName"
                />
                <div v-if="nameValidation.message" :class="['form-hint', nameValidation.valid ? 'text-success' : 'text-error']">
                  {{ nameValidation.message }}
                </div>
                <div v-else class="form-hint">{{ getNameRuleHint() }}</div>
              </el-form-item>
              <el-form-item label="节点描述">
                <el-input
                  v-model="wizardData.online.nodeDescription"
                  type="textarea"
                  :rows="3"
                  placeholder="节点物理位置、功能说明等"
                />
              </el-form-item>
              <el-form-item label="IP 地址">
                <el-input
                  v-model="wizardData.online.ipAddress"
                  placeholder="自动获取或手动输入"
                  clearable
                />
                <div class="form-hint">留空将自动获取客户端IP</div>
              </el-form-item>
              <el-form-item label="管理端口">
                <el-input-number
                  v-model="wizardData.online.port"
                  :min="1"
                  :max="65535"
                  class="w-full"
                />
              </el-form-item>
            </el-form>
          </div>

          <!-- 步骤 1: 离线模式配置 -->
          <div v-show="currentStep === 1 && wizardData.mode === 'offline'" class="step-container">
            <h3 class="step-title">离线模式配置</h3>
            <el-form
              ref="offlineFormRef"
              :model="wizardData.offline"
              :rules="offlineRules"
              label-width="120px"
              class="config-form"
            >
              <el-form-item label="节点名称" prop="nodeName">
                <el-input
                  v-model="wizardData.offline.nodeName"
                  placeholder="local-dev-node"
                  clearable
                  @input="validateName"
                />
                <div v-if="nameValidation.message" :class="['form-hint', nameValidation.valid ? 'text-success' : 'text-error']">
                  {{ nameValidation.message }}
                </div>
                <div v-else class="form-hint">{{ getNameRuleHint() }}</div>
              </el-form-item>
              <el-form-item label="节点描述">
                <el-input
                  v-model="wizardData.offline.nodeDescription"
                  type="textarea"
                  :rows="3"
                  placeholder="节点用途说明"
                />
              </el-form-item>
            </el-form>

            <el-alert type="info" :closable="false" class="mt-4">
              <template #title>
                离线模式说明
              </template>
              <ul class="offline-notes">
                <li>不连接运维中心，无法接收远程指令</li>
                <li>需要手动导入工程包（.ifp 文件）</li>
                <li>手动控制工程的启动和停止</li>
                <li>适合开发测试或隔离网络环境</li>
              </ul>
            </el-alert>
          </div>

          <!-- 步骤 2: 等待审批（仅在线模式） -->
          <div v-show="currentStep === 2 && wizardData.mode === 'online'" class="step-container">
            <div v-if="wizardData.approvalStatus === 'pending'" class="approval-waiting">
              <el-result icon="info" title="等待运维中心审批">
                <template #sub-title>
                  <div class="waiting-info">
                    <p>已向运维中心提交注册申请</p>
                    <p class="text-sm text-gray-500 mt-2">
                      节点ID: <span class="font-mono">{{ wizardData.nodeId }}</span>
                    </p>
                    <p class="text-sm text-gray-500">
                      申请时间: {{ formatTime(new Date()) }}
                    </p>
                    <p class="text-sm text-gray-500">
                      最后刷新: {{ lastRefreshTime }}
                    </p>
                  </div>
                </template>
                <template #extra>
                  <div class="flex space-x-3">
                    <el-button @click="checkApproval" :loading="checkingApproval">
                      <el-icon><Refresh /></el-icon>
                      手动刷新
                    </el-button>
                    <el-button type="info" @click="stopPolling">
                      停止自动刷新
                    </el-button>
                  </div>
                </template>
              </el-result>
              <div class="polling-status">
                <el-icon class="is-loading"><Loading /></el-icon>
                <span>自动刷新中 ({{ pollingCount }}/360)</span>
              </div>
            </div>

            <div v-else-if="wizardData.approvalStatus === 'approved'" class="approval-success">
              <el-result icon="success" title="审批通过">
                <template #sub-title>
                  节点已成功注册到运维中心，即将进入控制台
                </template>
              </el-result>
            </div>

            <div v-else-if="wizardData.approvalStatus === 'rejected'" class="approval-rejected">
              <el-result icon="error" title="申请被拒绝">
                <template #sub-title>
                  管理员拒绝了您的注册申请，请联系管理员了解详情
                </template>
                <template #extra>
                  <el-button type="primary" @click="resetWizard">
                    重新申请
                  </el-button>
                </template>
              </el-result>
            </div>
          </div>

          <!-- 步骤 2/3: 完成（离线模式直接完成） -->
          <div v-show="(currentStep === 2 && wizardData.mode === 'offline') || currentStep === 3" class="step-container">
            <el-result icon="success" title="初始化完成">
              <template #sub-title>
                <div v-if="wizardData.mode === 'online'">
                  节点已成功注册到运维中心，可以开始使用
                </div>
                <div v-else>
                  离线模式配置完成，您可以开始手动管理节点
                </div>
              </template>
              <template #extra>
                <el-button type="primary" size="large" @click="enterDashboard">
                  <el-icon><Right /></el-icon>
                  进入控制台
                </el-button>
              </template>
            </el-result>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="wizard-actions">
          <el-button
            v-if="currentStep > 0 && currentStep < finalStep"
            @click="prevStep"
            :disabled="submitting"
          >
            上一步
          </el-button>
          <el-button
            v-if="showNextButton"
            type="primary"
            @click="nextStep"
            :disabled="!canGoNext"
          >
            下一步
          </el-button>
          <el-button
            v-if="showSubmitButton"
            type="primary"
            @click="submitRegistration"
            :loading="submitting"
            :disabled="!canSubmit"
          >
            提交注册申请
          </el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Refresh, Right, Loading } from '@element-plus/icons-vue'
import { registerWithAuth, checkApprovalStatus, testCenterConnection } from '@/api/registerApi'
import { validateNodeName, getNameRuleHint } from '@/utils/nameValidator'
import { saveNodeConfig, getCenterUrl } from '@/utils/initCheck'
import { nodeApi } from '@/api/nodeApi'
import dayjs from 'dayjs'

const router = useRouter()

// 向导数据
const wizardData = reactive({
  mode: '',
  nodeId: '',
  approvalStatus: '',
  online: {
    centerUrl: '',
    username: '',
    password: '',
    nodeName: '',
    nodeDescription: '',
    ipAddress: '',
    port: 8080,
    agentVersion: '1.0.0',
  },
  offline: {
    nodeName: '',
    nodeDescription: '',
  },
})

// 当前步骤
const currentStep = ref(0)
const submitting = ref(false)
const testingConnection = ref(false)
const checkingApproval = ref(false)

// 名称验证状态
const nameValidation = reactive({
  valid: false,
  message: '',
})

// 轮询相关
let pollingTimer = null
const pollingCount = ref(0)
const lastRefreshTime = ref('')

// 表单验证规则
const onlineRules = {
  centerUrl: [
    { required: true, message: '请输入运维中心地址', trigger: 'blur' },
    { 
      pattern: /^https?:\/\/.+/, 
      message: '请输入有效的URL地址（以http://或https://开头）', 
      trigger: 'blur' 
    },
  ],
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
  ],
  nodeName: [
    { required: true, message: '请输入节点名称', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        const result = validateNodeName(value)
        if (result.valid) {
          callback()
        } else {
          callback(new Error(result.message))
        }
      },
      trigger: 'blur',
    },
  ],
}

const offlineRules = {
  nodeName: [
    { required: true, message: '请输入节点名称', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        const result = validateNodeName(value)
        if (result.valid) {
          callback()
        } else {
          callback(new Error(result.message))
        }
      },
      trigger: 'blur',
    },
  ],
}

const onlineFormRef = ref(null)
const offlineFormRef = ref(null)

// 计算属性
const finalStep = computed(() => {
  return wizardData.mode === 'online' ? 3 : 2
})

const showNextButton = computed(() => {
  if (currentStep.value === 0) return true
  return false
})

const showSubmitButton = computed(() => {
  return currentStep.value === 1
})

const canGoNext = computed(() => {
  if (currentStep.value === 0) {
    return !!wizardData.mode
  }
  return false
})

const canSubmit = computed(() => {
  if (wizardData.mode === 'online') {
    return nameValidation.valid
  } else {
    return nameValidation.valid
  }
})

// 方法
const selectMode = (mode) => {
  wizardData.mode = mode
}

const validateName = () => {
  const name = wizardData.mode === 'online' 
    ? wizardData.online.nodeName 
    : wizardData.offline.nodeName
  
  const result = validateNodeName(name)
  nameValidation.valid = result.valid
  nameValidation.message = result.message
}

const testConnection = async () => {
  if (!wizardData.online.centerUrl) {
    ElMessage.warning('请先输入运维中心地址')
    return
  }

  testingConnection.value = true
  try {
    const isConnected = await testCenterConnection(wizardData.online.centerUrl)
    if (isConnected) {
      ElMessage.success('连接测试成功')
    } else {
      ElMessage.error('无法连接到运维中心，请检查地址是否正确')
    }
  } catch (error) {
    ElMessage.error('连接测试失败: ' + error.message)
  } finally {
    testingConnection.value = false
  }
}

const nextStep = () => {
  if (currentStep.value < finalStep.value) {
    currentStep.value++
  }
}

const prevStep = () => {
  if (currentStep.value > 0) {
    currentStep.value--
  }
}

const submitRegistration = async () => {
  // 表单验证
  const formRef = wizardData.mode === 'online' ? onlineFormRef.value : offlineFormRef.value
  
  try {
    await formRef.validate()
  } catch (error) {
    ElMessage.warning('请检查表单填写是否正确')
    return
  }

  submitting.value = true

  try {
    if (wizardData.mode === 'online') {
      // 在线模式：提交注册申请
      const result = await registerWithAuth({
        centerUrl: wizardData.online.centerUrl,
        username: wizardData.online.username,
        password: wizardData.online.password,
        nodeName: wizardData.online.nodeName,
        nodeDescription: wizardData.online.nodeDescription,
        ipAddress: wizardData.online.ipAddress,
        port: wizardData.online.port,
        agentVersion: wizardData.online.agentVersion,
        mode: 'online',
      })

      if (result.success) {
        wizardData.nodeId = result.data.nodeId
        wizardData.approvalStatus = result.data.approvalStatus

        // 保存基础配置
        saveNodeConfig({
          mode: 'online',
          nodeId: result.data.nodeId,
          nodeName: wizardData.online.nodeName,
          centerUrl: wizardData.online.centerUrl,
          approvalStatus: result.data.approvalStatus,
          ipAddress: wizardData.online.ipAddress,
          port: wizardData.online.port,
        })

        if (result.data.autoApproved) {
          // 自动审批通过
          saveNodeConfig({
            registrationToken: result.data.registrationToken,
          })
          
          // 通知 NodeAgent 后端保存配置
          await saveBackendConfig('online', {
            centerUrl: wizardData.online.centerUrl,
            nodeId: result.data.nodeId,
            registrationToken: result.data.registrationToken,
          })
          
          ElMessage.success('注册成功，已自动审批通过')
          currentStep.value = 3 // 直接跳到完成步骤
        } else {
          // 需要等待审批
          ElMessage.success('注册申请已提交，请等待管理员审批')
          currentStep.value = 2
          startPolling()
        }
      }
    } else {
      // 离线模式：直接保存配置
      saveNodeConfig({
        mode: 'offline',
        nodeName: wizardData.offline.nodeName,
      })
      ElMessage.success('离线模式配置完成')
      currentStep.value = 2
    }
  } catch (error) {
    ElMessage.error(error.message || '提交失败')
  } finally {
    submitting.value = false
  }
}

const startPolling = () => {
  pollingCount.value = 0
  lastRefreshTime.value = formatTime(new Date())
  
  pollingTimer = setInterval(async () => {
    pollingCount.value++
    
    // 超时停止（30分钟 = 360次 * 5秒）
    if (pollingCount.value > 360) {
      stopPolling()
      ElMessage.warning('审批查询超时，请稍后手动刷新')
      return
    }

    await checkApproval()
  }, 5000)
}

const stopPolling = () => {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
    ElMessage.info('已停止自动刷新')
  }
}

const checkApproval = async () => {
  if (checkingApproval.value) return

  checkingApproval.value = true
  try {
    const result = await checkApprovalStatus(
      wizardData.online.centerUrl,
      wizardData.nodeId
    )

    lastRefreshTime.value = formatTime(new Date())

    if (result.success) {
      wizardData.approvalStatus = result.data.approvalStatus

      if (result.data.approvalStatus === 'approved') {
        stopPolling()
        
        // 保存 token
        saveNodeConfig({
          approvalStatus: 'approved',
          registrationToken: result.data.registrationToken,
        })

        // 通知 NodeAgent 后端保存配置
        await saveBackendConfig('online', {
          centerUrl: wizardData.online.centerUrl,
          nodeId: wizardData.nodeId,
          registrationToken: result.data.registrationToken,
        })

        ElMessage.success('审批通过！')
        
        // 等待1秒后跳转到完成页
        setTimeout(() => {
          currentStep.value = 3
        }, 1000)
      } else if (result.data.approvalStatus === 'rejected') {
        stopPolling()
        ElMessage.error('注册申请被拒绝')
      }
    }
  } catch (error) {
    console.error('查询审批状态失败:', error)
  } finally {
    checkingApproval.value = false
  }
}

const enterDashboard = () => {
  router.push('/')
}

const resetWizard = () => {
  currentStep.value = 0
  wizardData.mode = ''
  wizardData.nodeId = ''
  wizardData.approvalStatus = ''
  wizardData.online = {
    centerUrl: '',
    username: '',
    password: '',
    nodeName: '',
    nodeDescription: '',
    ipAddress: '',
    port: 8080,
    agentVersion: '1.0.0',
  }
  wizardData.offline = {
    nodeName: '',
    nodeDescription: '',
  }
}

const formatTime = (time) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

// 保存配置到 NodeAgent 后端
const saveBackendConfig = async (mode, config) => {
  try {
    await nodeApi.saveConfig({
      mode,
      ...config,
    })
  } catch (error) {
    console.error('保存后端配置失败:', error)
    // 不阻塞流程，仅记录错误
  }
}

onBeforeUnmount(() => {
  stopPolling()
})
</script>

<style scoped>
.init-wizard {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.wizard-container {
  width: 100%;
  max-width: 900px;
}

.wizard-card {
  background: white;
  border-radius: 16px;
}

.card-header {
  text-align: center;
}

.card-header h1 {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
  margin: 0 0 8px 0;
}

.card-header .subtitle {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

:deep(.el-steps) {
  margin: 30px 0;
}

.step-content {
  min-height: 400px;
  padding: 20px;
}

.step-container {
  max-width: 600px;
  margin: 0 auto;
}

.step-title {
  font-size: 20px;
  font-weight: bold;
  color: #303133;
  margin-bottom: 24px;
  text-align: center;
}

.mode-selection {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
  margin-top: 30px;
}

.mode-card {
  cursor: pointer;
  transition: all 0.3s;
  text-align: center;
  padding: 20px;
}

.mode-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.1);
}

.mode-card.selected {
  border: 2px solid #409eff;
  background-color: #ecf5ff;
}

.mode-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.mode-card h4 {
  font-size: 18px;
  font-weight: bold;
  margin-bottom: 16px;
  color: #303133;
}

.mode-features {
  list-style: none;
  padding: 0;
  margin: 0 0 16px 0;
  text-align: left;
}

.mode-features li {
  font-size: 14px;
  color: #606266;
  margin-bottom: 8px;
  padding-left: 20px;
  position: relative;
}

.mode-note {
  font-size: 12px;
  color: #909399;
  font-style: italic;
}

.config-form {
  margin-top: 20px;
}

.form-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.form-hint.text-success {
  color: #67c23a;
}

.form-hint.text-error {
  color: #f56c6c;
}

.offline-notes {
  list-style: none;
  padding-left: 0;
  margin: 8px 0 0 0;
}

.offline-notes li {
  margin-bottom: 6px;
  font-size: 14px;
}

.approval-waiting {
  text-align: center;
}

.waiting-info p {
  margin: 8px 0;
}

.polling-status {
  margin-top: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 14px;
  color: #909399;
}

.wizard-actions {
  display: flex;
  justify-content: center;
  gap: 16px;
  padding: 20px;
  border-top: 1px solid #ebeef5;
  margin-top: 20px;
}

:deep(.el-card__header) {
  padding: 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

:deep(.el-card__header) .card-header h1 {
  color: white;
}

:deep(.el-card__header) .card-header .subtitle {
  color: rgba(255, 255, 255, 0.8);
}
</style>
