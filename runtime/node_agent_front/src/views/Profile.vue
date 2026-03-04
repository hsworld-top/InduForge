<template>
  <div class="profile">
    <el-card class="box-card">
      <template #header>
        <span>运维中心绑定</span>
      </template>

      <div class="status-row">
        <span>当前状态：</span>
        <el-tag :type="bootstrapStatus.centerBound ? 'success' : 'info'">
          {{ bootstrapStatus.centerBound ? '已绑定运维中心' : '未绑定（本地运行）' }}
        </el-tag>
        <el-tag style="margin-left: 8px;" type="warning">
          初始化状态: {{ bootstrapStatus.status || '-' }}
        </el-tag>
      </div>

      <el-form :model="centerForm" label-width="120px" class="center-form">
        <el-form-item label="运维中心地址" required>
          <el-input v-model="centerForm.centerUrl" placeholder="http://127.0.0.1:9099" />
        </el-form-item>
        <el-form-item label="租户代码">
          <el-input v-model="centerForm.tenantCode" placeholder="default（可选）" />
        </el-form-item>
        <el-form-item label="用户名" required>
          <el-input v-model="centerForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="centerForm.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item label="节点名称" required>
          <el-input v-model="centerForm.nodeName" placeholder="node-local-01" />
        </el-form-item>
        <el-form-item label="节点描述">
          <el-input v-model="centerForm.nodeDescription" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>

      <div class="ops-row">
        <el-button type="primary" :loading="binding" @click="bindCenter">
          绑定运维中心
        </el-button>
        <el-button :disabled="!approvalPending" :loading="checkingApproval" @click="checkApproval">
          刷新审批状态
        </el-button>
        <el-button type="danger" plain :disabled="!bootstrapStatus.centerBound" :loading="unbinding" @click="unbindCenter">
          解绑（切回本地）
        </el-button>
      </div>

      <el-alert
        v-if="approvalPending"
        class="mt-12"
        :closable="false"
        type="warning"
        title="已提交绑定申请，等待审批。可点击“刷新审批状态”继续。"
      />
    </el-card>

    <el-card class="box-card">
      <template #header>
        <span>连接配置管理</span>
      </template>

      <el-form label-width="120px">
        <el-form-item label="项目ID" required>
          <el-input v-model="projectId" placeholder="输入项目ID" style="max-width: 300px;" />
          <el-button type="primary" @click="loadProfile" :loading="loading" style="margin-left: 10px;">
            加载配置
          </el-button>
        </el-form-item>
      </el-form>

      <el-descriptions v-if="profileData" :column="2" border>
        <el-descriptions-item label="名称">
          {{ profileData.name || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="端点">
          {{ profileData.endpoint || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="认证类型">
          <el-tag>{{ profileData.authType || '-' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="元数据">
          <pre class="metadata">{{ JSON.stringify(profileData.metadata, null, 2) }}</pre>
        </el-descriptions-item>
      </el-descriptions>

      <div v-if="profileData" style="margin-top: 20px;">
        <el-button type="primary" @click="editMode = !editMode">
          {{ editMode ? '取消编辑' : '编辑配置' }}
        </el-button>
      </div>

      <el-form v-if="editMode" :model="editForm" label-width="120px" style="margin-top: 20px;">
        <el-form-item label="名称">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="端点">
          <el-input v-model="editForm.endpoint" />
        </el-form-item>
        <el-form-item label="认证类型">
          <el-select v-model="editForm.authType" placeholder="选择认证类型">
            <el-option label="Token" value="token" />
            <el-option label="Basic" value="basic" />
            <el-option label="OAuth" value="oauth" />
          </el-select>
        </el-form-item>
        <el-form-item label="认证数据">
          <el-input
            v-model="authDataStr"
            type="textarea"
            :rows="4"
            placeholder="JSON格式的认证数据"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveProfile" :loading="saving">
            保存
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { nodeApi } from '@/api/nodeApi'
import { checkApprovalStatus, loginWithAuth, registerNodeWithToken } from '@/api/registerApi'

const route = useRoute()

const loading = ref(false)
const saving = ref(false)
const editMode = ref(false)
const profileData = ref(null)

const projectId = ref(route.query.projectId || '')
const authDataStr = ref('')
const editForm = reactive({
  name: '',
  endpoint: '',
  authType: '',
  metadata: {},
})

const bootstrapStatus = reactive({
  status: '',
  centerBound: false,
  centerUrl: '',
  nodeId: '',
  nodeName: '',
})
const centerForm = reactive({
  centerUrl: '',
  tenantCode: '',
  username: '',
  password: '',
  nodeName: '',
  nodeDescription: '',
  ipAddress: '',
  port: 8081,
  agentVersion: '1.0.0',
})
const binding = ref(false)
const unbinding = ref(false)
const checkingApproval = ref(false)
const approvalPending = ref(false)
const pendingRegistration = reactive({
  centerUrl: '',
  nodeId: '',
  registrationToken: '',
})

watch(editMode, (newVal) => {
  if (!newVal || !profileData.value) {
    return
  }
  editForm.name = profileData.value.name || ''
  editForm.endpoint = profileData.value.endpoint || ''
  editForm.authType = profileData.value.authType || ''
  editForm.metadata = profileData.value.metadata || {}
})

const loadBootstrapStatus = async () => {
  try {
    const data = await nodeApi.getBootstrapStatus()
    bootstrapStatus.status = data.status || ''
    bootstrapStatus.centerBound = !!data.centerBound
    bootstrapStatus.centerUrl = data.centerUrl || ''
    bootstrapStatus.nodeId = data.nodeId || ''
    bootstrapStatus.nodeName = data.nodeName || ''

    if (data.centerUrl) {
      centerForm.centerUrl = data.centerUrl
    }
    if (data.nodeName) {
      centerForm.nodeName = data.nodeName
    }
  } catch (error) {
    console.error('加载初始化状态失败:', error)
  }
}

const loadServiceConfig = async () => {
  try {
    const config = await nodeApi.getServiceConfig()
    if (config?.listen?.port) {
      centerForm.port = config.listen.port
    }
  } catch (error) {
    console.error('加载服务配置失败:', error)
  }
}

const bindCenter = async () => {
  if (!centerForm.centerUrl || !centerForm.username || !centerForm.password || !centerForm.nodeName) {
    ElMessage.warning('请填写运维中心地址、用户名、密码和节点名称')
    return
  }

  binding.value = true
  try {
    const loginResult = await loginWithAuth({
      centerUrl: centerForm.centerUrl,
      tenantCode: centerForm.tenantCode,
      username: centerForm.username,
      password: centerForm.password,
    })
    const accessToken = loginResult?.data?.accessToken
    if (!accessToken) {
      throw new Error('登录成功但未获取访问令牌')
    }

    const registerResult = await registerNodeWithToken({
      centerUrl: centerForm.centerUrl,
      accessToken,
      nodeName: centerForm.nodeName,
      nodeDescription: centerForm.nodeDescription,
      ipAddress: centerForm.ipAddress,
      port: centerForm.port,
      agentVersion: centerForm.agentVersion,
    })

    if (!registerResult?.success) {
      throw new Error(registerResult?.message || '注册失败')
    }

    pendingRegistration.centerUrl = centerForm.centerUrl
    pendingRegistration.nodeId = registerResult.data.nodeId
    pendingRegistration.registrationToken = registerResult.data.registrationToken || ''

    if (registerResult.data.autoApproved || registerResult.data.approvalStatus === 'approved') {
      await nodeApi.completeBootstrap({
        mode: 'online',
        centerUrl: pendingRegistration.centerUrl,
        nodeId: pendingRegistration.nodeId,
        registrationToken: pendingRegistration.registrationToken,
      })
      approvalPending.value = false
      ElMessage.success('运维中心绑定成功')
      await loadBootstrapStatus()
      return
    }

    approvalPending.value = true
    ElMessage.success('注册申请已提交，请等待审批后刷新状态')
  } catch (error) {
    ElMessage.error(error.message || '绑定失败')
  } finally {
    binding.value = false
  }
}

const checkApproval = async () => {
  if (!approvalPending.value || !pendingRegistration.centerUrl || !pendingRegistration.nodeId) {
    ElMessage.warning('当前没有待审批的绑定申请')
    return
  }

  checkingApproval.value = true
  try {
    const result = await checkApprovalStatus(pendingRegistration.centerUrl, pendingRegistration.nodeId)
    if (!result?.success) {
      throw new Error(result?.message || '查询审批状态失败')
    }

    const status = result.data?.approvalStatus
    if (status === 'approved') {
      await nodeApi.completeBootstrap({
        mode: 'online',
        centerUrl: pendingRegistration.centerUrl,
        nodeId: pendingRegistration.nodeId,
        registrationToken: result.data?.registrationToken || pendingRegistration.registrationToken,
      })
      approvalPending.value = false
      ElMessage.success('审批已通过，绑定完成')
      await loadBootstrapStatus()
      return
    }
    if (status === 'rejected') {
      approvalPending.value = false
      ElMessage.error('审批被拒绝，请重新提交')
      return
    }

    ElMessage.info('审批中，请稍后再试')
  } catch (error) {
    ElMessage.error(error.message || '查询审批状态失败')
  } finally {
    checkingApproval.value = false
  }
}

const unbindCenter = async () => {
  unbinding.value = true
  try {
    const fallbackNodeName = centerForm.nodeName || bootstrapStatus.nodeName || 'node-local'
    await nodeApi.completeBootstrap({
      mode: 'offline',
      nodeName: fallbackNodeName,
    })
    approvalPending.value = false
    pendingRegistration.centerUrl = ''
    pendingRegistration.nodeId = ''
    pendingRegistration.registrationToken = ''
    ElMessage.success('已解绑运维中心，切换为本地运行')
    await loadBootstrapStatus()
  } catch (error) {
    ElMessage.error(error.message || '解绑失败')
  } finally {
    unbinding.value = false
  }
}

const loadProfile = async () => {
  if (!projectId.value) {
    ElMessage.warning('请输入项目ID')
    return
  }

  loading.value = true
  try {
    const data = await nodeApi.getProfile(projectId.value)
    profileData.value = data
    ElMessage.success('配置加载成功')
  } catch (error) {
    console.error('加载配置失败:', error)
    ElMessage.error('加载配置失败')
    profileData.value = null
  } finally {
    loading.value = false
  }
}

const saveProfile = async () => {
  if (!projectId.value) {
    ElMessage.warning('项目ID不能为空')
    return
  }

  saving.value = true
  try {
    let authData = {}
    if (authDataStr.value) {
      try {
        authData = JSON.parse(authDataStr.value)
      } catch (e) {
        ElMessage.error('认证数据格式不正确，请输入有效的JSON')
        saving.value = false
        return
      }
    }

    await nodeApi.saveProfile(projectId.value, {
      ...editForm,
      authData,
    })
    editMode.value = false
    await loadProfile()
    ElMessage.success('保存成功')
  } catch (error) {
    console.error('保存配置失败:', error)
    ElMessage.error('保存配置失败')
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await loadBootstrapStatus()
  await loadServiceConfig()
  if (projectId.value) {
    await loadProfile()
  }
})
</script>

<style scoped>
.profile {
  display: grid;
  gap: 16px;
}

.box-card {
  margin-bottom: 0;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
}

.status-row {
  margin-bottom: 16px;
  display: flex;
  align-items: center;
}

.center-form {
  max-width: 720px;
}

.ops-row {
  margin-top: 12px;
}

.ops-row .el-button + .el-button {
  margin-left: 8px;
}

.mt-12 {
  margin-top: 12px;
}

.metadata {
  background-color: var(--metadata-bg);
  color: var(--metadata-fg);
  padding: 10px;
  border-radius: 4px;
  font-size: 12px;
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid var(--border-subtle);
}
</style>
