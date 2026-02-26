<template>
  <div class="ops-management h-full flex flex-col">
    <!-- 页面标题和页头操作 -->
    <div class="flex justify-between items-center mb-6 px-6 pt-6">
      <div class="flex items-center space-x-4">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">运维监控中心</h1>
        <el-tag border size="large" type="info" class="rounded-full">
          节点总数: {{ nodePagination.total }}
        </el-tag>
      </div>
      <div class="flex items-center space-x-3">
        <el-radio-group v-model="activeView" size="default">
          <el-radio-button value="dashboard">
            <el-icon class="mr-1"><Grid /></el-icon> 节点大盘
          </el-radio-button>
          <el-radio-button value="list">
            <el-icon class="mr-1"><List /></el-icon> 详细列表
          </el-radio-button>
        </el-radio-group>
        <!-- 待审核申请通知图标 -->
        <el-badge :value="pendingCount" :hidden="pendingCount === 0" class="cursor-pointer" @click="showPendingDialog = true">
          <el-button :type="pendingCount > 0 ? 'warning' : 'default'" :plain="pendingCount === 0" circle>
            <el-icon><Bell /></el-icon>
          </el-button>
        </el-badge>
        <el-button @click="fetchNodes" :loading="nodeLoading">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </div>
    </div>

    <!-- 过滤器面板 -->
    <div class="px-6 mb-6">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-4">
        <el-form :inline="true" :model="nodeSearch" class="flex flex-wrap gap-4 -mb-4">
          <el-form-item label="节点状态">
            <el-select v-model="nodeSearch.status" placeholder="全部状态" clearable style="width: 120px">
              <el-option label="在线" value="online" />
              <el-option label="离线" value="offline" />
              <el-option label="异常" value="error" />
            </el-select>
          </el-form-item>
          <el-form-item label="搜索节点">
            <el-input
              v-model="nodeSearch.keyword"
              placeholder="名称 / IP 地址"
              clearable
              :prefix-icon="Search"
              style="width: 220px"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="fetchNodes">查询</el-button>
            <el-button @click="resetNodeSearch">重置</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 视图：节点大盘 -->
    <div v-if="activeView === 'dashboard'" class="flex-1 overflow-y-auto px-6 pb-6">
      <div v-loading="nodeLoading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        <el-card
          v-for="node in nodeList"
          :key="node.id"
          shadow="hover"
          class="node-card border-none ring-1 ring-gray-200 dark:ring-gray-700"
          :body-style="{ padding: '0px' }"
        >
          <!-- 卡片头部：状态与基本信息 -->
          <div class="p-4 border-b border-gray-100 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/50">
            <div class="flex justify-between items-start mb-2">
              <div class="flex items-center">
                <div 
                  class="w-3 h-3 rounded-full mr-2" 
                  :class="node.status === 'online' ? 'bg-green-500 animate-pulse' : (node.status === 'offline' ? 'bg-gray-400' : 'bg-red-500')"
                ></div>
                <h3 class="font-bold text-gray-800 dark:text-gray-200 truncate">{{ node.name }}</h3>
              </div>
              <el-dropdown trigger="click">
                <el-button link><el-icon><MoreFilled /></el-icon></el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="viewNodeDetail(node)">详细信息</el-dropdown-item>
                    <el-dropdown-item @click="restartNode(node)" :disabled="node.status !== 'online'">重启 Agent</el-dropdown-item>
                    <el-dropdown-item divided @click="deleteNode(node)" type="danger">删除注册</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <div class="text-xs text-gray-500 flex justify-between">
              <span>IP: {{ node.ipAddress || '-' }}</span>
              <span>Port: {{ node.port }}</span>
            </div>
          </div>

          <!-- 卡片中部：资源概览 -->
          <div class="p-4 space-y-4">
            <div class="grid grid-cols-2 gap-4">
              <!-- CPU & Memory Gauges -->
              <div class="text-center">
                <div class="text-[10px] text-gray-400 uppercase mb-1">CPU 使用率</div>
                <el-progress
                  type="dashboard"
                  :percentage="getMetricValue(node, 'cpu')"
                  :width="60"
                  :stroke-width="4"
                  :color="getProgressColor"
                />
              </div>
              <div class="text-center">
                <div class="text-[10px] text-gray-400 uppercase mb-1">内存使用率</div>
                <el-progress
                  type="dashboard"
                  :percentage="getMetricValue(node, 'memory')"
                  :width="60"
                  :stroke-width="4"
                  :color="getProgressColor"
                />
              </div>
            </div>

            <!-- Disk Bars -->
            <div class="text-xs space-y-2">
              <div class="flex justify-between items-center text-gray-600 dark:text-gray-400">
                <span>磁盘 ({{ getDiskLabel(node) }})</span>
                <span class="font-mono">{{ getMetricValue(node, 'disk') }}%</span>
              </div>
              <el-progress :percentage="getMetricValue(node, 'disk')" :show-text="false" :stroke-width="6" class="mb-3" />
            </div>
          </div>

          <!-- 卡片尾部：运行中工程列表 -->
          <div class="bg-gray-50/30 dark:bg-gray-900/20 p-2 border-t border-gray-100 dark:border-gray-700">
            <div class="text-[10px] font-bold text-gray-400 uppercase px-2 py-1 mb-1 flex justify-between">
              <span>运行中工程 ({{ node.deployments?.length || 0 }})</span>
              <el-button link size="small" type="primary" class="text-[10px]" @click="promptDeploy(node)">部署</el-button>
            </div>
            
            <div v-if="node.deployments?.length > 0" class="space-y-1">
              <div 
                v-for="deploy in node.deployments" 
                :key="deploy.id"
                class="bg-white dark:bg-gray-800 rounded p-2 text-xs ring-1 ring-gray-100 dark:ring-gray-700 flex justify-between items-center"
              >
                <div class="flex flex-col">
                  <span class="font-semibold text-gray-700 dark:text-gray-300 truncate w-32">
                    {{ deploy.project?.name }}
                  </span>
                  <div class="flex items-center space-x-2 text-[10px] text-gray-500">
                    <span class="flex items-center"><el-icon class="mr-0.5"><User /></el-icon> {{ deploy.runtimeMetrics?.onlineUsers || 0 }}</span>
                    <span class="flex items-center"><el-icon class="mr-0.5"><Clock /></el-icon> {{ deploy.runtimeMetrics?.concurrentUsers || 0 }}</span>
                  </div>
                </div>
                  <div class="flex items-center space-x-1">
                    <el-tag size="small" :type="getDeployStatusType(deploy.status)">
                      {{ getDeployLabel(deploy.status) }}
                    </el-tag>
                    <el-tag size="small" :type="deploy.mode === 'DEV' ? 'warning' : 'success'">
                      {{ deploy.mode }}
                    </el-tag>
                    <el-dropdown trigger="hover">
                    <el-button link><el-icon size="small"><Tools /></el-icon></el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item @click="handleStartProject(deploy)">
                          <el-icon class="mr-1"><VideoPlay /></el-icon>启动
                        </el-dropdown-item>
                        <el-dropdown-item @click="handleStopProject(deploy)">
                          <el-icon class="mr-1"><VideoPause /></el-icon>停止
                        </el-dropdown-item>
                        <el-dropdown-item @click="handleRestartProject(deploy)">
                          <el-icon class="mr-1"><RefreshRight /></el-icon>重启
                        </el-dropdown-item>
                        <el-dropdown-item @click="handleRollback(deploy)" :disabled="deploy.mode === 'DEV'">
                          <el-icon class="mr-1"><RefreshLeft /></el-icon>回滚版本
                        </el-dropdown-item>
                        <el-dropdown-item @click="handleViewLog(deploy)">
                          <el-icon class="mr-1"><Document /></el-icon>查看日志
                        </el-dropdown-item>
                        <el-dropdown-item divided @click="handleUndeploy(deploy)" type="danger">
                          <el-icon class="mr-1"><Remove /></el-icon>撤销部署
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </div>
            </div>
            <div v-else class="text-center py-4 text-xs text-gray-400">
              暂无运行工程
            </div>
          </div>
        </el-card>
      </div>
      
      <!-- 无数据 -->
      <el-empty v-if="!nodeLoading && nodeList.length === 0" description="暂无在线节点监控" />

      <!-- 分页 -->
      <div class="flex justify-end mt-8">
        <el-pagination
          v-model:current-page="nodePagination.page"
          v-model:page-size="nodePagination.pageSize"
          :total="nodePagination.total"
          :page-sizes="[12, 24, 48]"
          layout="total, sizes, prev, pager, next"
          @size-change="fetchNodes"
          @current-change="fetchNodes"
        />
      </div>
    </div>

    <!-- 视图：详细列表 -->
    <div v-else-if="activeView === 'list'" class="flex-1 overflow-y-auto px-6 pb-6">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 overflow-hidden">
        <el-table :data="nodeList" style="width: 100%">
          <el-table-column type="expand">
            <template #default="props">
              <div class="p-4 bg-gray-50/50 dark:bg-gray-900/50">
                <h4 class="text-sm font-bold mb-3">部署在该节点的工程</h4>
                <el-table :data="props.row.deployments" size="small" border>
                  <el-table-column label="工程名称" prop="project.name" />
                  <el-table-column label="运行版本" prop="version" width="100" />
                  <el-table-column label="运行状态" width="100">
                    <template #default="scope">
                      <el-tag size="small" :type="getDeployStatusType(scope.row.status)">{{ getDeployLabel(scope.row.status) }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="在线用户" width="100">
                    <template #default="scope">
                      {{ scope.row.runtimeMetrics?.onlineUsers || 0 }}
                    </template>
                  </el-table-column>
                  <el-table-column label="并发峰值" width="100">
                    <template #default="scope">
                      {{ scope.row.runtimeMetrics?.concurrentUsers || 0 }}
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="150">
                    <template #default="scope">
                      <el-button link type="primary" size="small" @click="handleViewLog(scope.row)">日志</el-button>
                      <el-button link type="warning" size="small" @click="handleStopProject(scope.row)">停止</el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="节点名称" prop="name" />
          <el-table-column label="IP地址" prop="ipAddress" />
          <el-table-column label="状态" width="120">
            <template #default="scope">
              <el-tag :type="getNodeStatusType(scope.row.status)">{{ getNodeStatusLabel(scope.row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="CPU" width="100">
            <template #default="scope">{{ getMetricValue(scope.row, 'cpu') }}%</template>
          </el-table-column>
          <el-table-column label="内存" width="100">
            <template #default="scope">{{ getMetricValue(scope.row, 'memory') }}%</template>
          </el-table-column>
          <el-table-column label="磁盘" width="100">
            <template #default="scope">{{ getMetricValue(scope.row, 'disk') }}%</template>
          </el-table-column>
          <el-table-column label="工程数" width="100">
            <template #default="scope">{{ scope.row.deployments?.length || 0 }}</template>
          </el-table-column>
          <el-table-column label="最后心跳" prop="lastHeartbeatAt">
            <template #default="scope">{{ formatTime(scope.row.lastHeartbeatAt) }}</template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 弹窗：待审核申请列表 -->
    <el-dialog v-model="showPendingDialog" title="待审核申请" width="900px">
      <div class="bg-white dark:bg-gray-800 rounded-xl overflow-hidden">
        <el-table v-loading="nodeLoading" :data="pendingList" style="width: 100%">
          <el-table-column label="申请节点名称" min-width="180">
            <template #default="scope">
              <div class="flex flex-col">
                <span class="font-bold text-gray-800 dark:text-gray-200">{{ scope.row.name }}</span>
                <span class="text-xs text-gray-500">{{ scope.row.description || '无描述' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="申请人信息" width="180">
            <template #default="scope">
              <div class="flex flex-col text-xs">
                <span class="font-semibold text-gray-700 dark:text-gray-300">
                  {{ scope.row.registrant?.username || '-' }}
                </span>
                <el-tag size="small" :type="getRoleTagType(scope.row.registrant?.role)">
                  {{ getRoleLabel(scope.row.registrant?.role) }}
                </el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="节点模式" width="100">
            <template #default="scope">
              <el-tag size="small" :type="scope.row.mode === 'online' ? 'success' : 'info'">
                {{ scope.row.mode === 'online' ? '在线' : '离线' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="节点网络地址" width="180">
            <template #default="scope">
              <div class="flex flex-col text-xs">
                <span>IP: {{ scope.row.ipAddress }}</span>
                <span>Port: {{ scope.row.port }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="Agent 版本" prop="agentVersion" width="100" />
          <el-table-column label="申请时间" width="160">
            <template #default="scope">{{ formatTime(scope.row.createdAt) }}</template>
          </el-table-column>
          <el-table-column label="审批操作" width="220" fixed="right">
            <template #default="scope">
              <div v-if="canApproveNode" class="flex space-x-2">
                <el-button type="success" size="small" @click="handleApprove(scope.row)">
                  <el-icon class="mr-1"><Check /></el-icon> 通过
                </el-button>
                <el-button type="danger" size="small" plain @click="handleReject(scope.row)">
                  <el-icon class="mr-1"><Close /></el-icon> 拒绝
                </el-button>
              </div>
              <div v-else class="text-xs text-gray-500">
                仅 OPS_ADMIN 和 SYSTEM_ADMIN 可审批
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <el-button @click="showPendingDialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 弹窗：注册节点 -->
    <el-dialog v-model="showAddNodeDialog" title="注册新运维节点" width="500px">
      <el-form ref="nodeFormRef" :model="nodeForm" :rules="nodeFormRules" label-width="100px" class="mt-4">
        <el-form-item label="节点名称" prop="name">
          <el-input v-model="nodeForm.name" placeholder="例如: Production-Edge-01" />
        </el-form-item>
        <el-form-item label="节点描述" prop="description">
          <el-input v-model="nodeForm.description" type="textarea" placeholder="节点物理位置、功能说明等" />
        </el-form-item>
        <el-form-item label="IP 地址">
          <el-input v-model="nodeForm.ipAddress" placeholder="节点内网/外网 IP" />
        </el-form-item>
        <el-form-item label="管理端口">
          <el-input-number v-model="nodeForm.port" :min="1" :max="65535" class="w-full" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddNodeDialog = false">取消</el-button>
        <el-button type="primary" @click="submitNodeForm" :loading="submitting">创建并生成令牌</el-button>
      </template>
    </el-dialog>

    <!-- 弹窗：节点注册完成 -->
    <el-dialog v-model="showTokenDialog" title="节点审批通过" width="500px" :close-on-click-modal="false">
      <el-result icon="success" title="节点令牌已发放" sub-title="请将以下信息配置到 NodeAgent 的安装程序或配置文件中">
        <template #extra>
          <div class="bg-gray-100 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 p-4 rounded-lg font-mono text-xs break-all mb-4 text-left select-all">
            REGISTRATION_TOKEN={{ registrationToken }}<br/>
            NODE_ID={{ newNodeId }}
          </div>
          <el-button type="primary" @click="copyToken">复制并关闭</el-button>
        </template>
      </el-result>
    </el-dialog>

    <!-- 弹窗：详细日志 -->
    <el-dialog v-model="showLogDialog" :title="`运行日志 - ${currentDeployment?.project?.name || ''}`" width="800px">
      <div class="bg-black text-green-500 p-4 rounded-lg h-96 overflow-y-auto font-mono text-xs">
        <div v-for="(log, idx) in logContent" :key="idx" class="mb-1">
          <span class="text-gray-500">[{{ log.time }}]</span>
          <span class="ml-2">{{ log.message }}</span>
        </div>
        <div v-if="logContent.length === 0" class="text-gray-500 text-center mt-20">暂无实时日志上报</div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Grid, List, Refresh, Search, MoreFilled,
  User, Clock, Tools, Check, Close, Bell
} from '@element-plus/icons-vue'
import request from '@/utils/request'
import dayjs from 'dayjs'
import { initSocket, getSocket } from '@/utils/socket'
import { Storage } from '@/utils/storage'
import { canApproveNodes } from '@/permissions'

// 获取当前用户角色
const currentUserInfo = Storage.getUserInfo()
const currentUserRole = currentUserInfo?.role || ''

// 计算是否有审批权限
const canApproveNode = computed(() => {
  return canApproveNodes(currentUserRole)
})

// 状态与视图控制
const activeView = ref('dashboard')
const nodeLoading = ref(false)
const nodeList = ref([])
const approvedCount = ref(0)
const pendingCount = ref(0)

// 节点查询
const nodeSearch = reactive({
  status: '',
  keyword: '',
})
const nodePagination = reactive({
  page: 1,
  pageSize: 12,
  total: 0,
})

// 注册节点状态
const showTokenDialog = ref(false)
const registrationToken = ref('')
const newNodeId = ref('')

// 待审核申请弹窗
const showPendingDialog = ref(false)
const pendingList = ref([])

// 日志弹窗
const showLogDialog = ref(false)
const currentDeployment = ref(null)
const logContent = ref([])

// 获取节点数据
const fetchNodes = async () => {
  nodeLoading.value = true
  try {
    const res = await request.get('/nodes', {
      params: {
        page: nodePagination.page,
        pageSize: nodePagination.pageSize,
        status: nodeSearch.status || undefined,
        search: nodeSearch.keyword || undefined,
        approvalStatus: 'approved',
      },
    })
    if (res.success) {
      nodeList.value = res.data.items
      nodePagination.total = res.data.total
      approvedCount.value = res.data.total
    }
  } catch (error) {
    console.error('获取节点失败:', error)
    ElMessage.error('无法同步节点监控数据')
  } finally {
    nodeLoading.value = false
  }
}

// 获取待审核申请列表
const fetchPendingList = async () => {
  try {
    const res = await request.get('/nodes', {
      params: {
        page: 1,
        pageSize: 100, // 获取最多100条待审核记录
        approvalStatus: 'pending',
      },
    })
    if (res.success) {
      pendingList.value = res.data.items
      pendingCount.value = res.data.total
    }
  } catch (error) {
    console.error('获取待审核申请失败:', error)
  }
}

const handleApprove = async (node) => {
  try {
    const res = await request.put(`/nodes/${node.id}/approve`)
    if (res.success) {
      ElMessage.success('节点审批已通过')
      registrationToken.value = res.data.registrationToken
      newNodeId.value = res.data.id
      showTokenDialog.value = true
      showPendingDialog.value = false
      fetchNodes()
      fetchPendingList() // 刷新待审核列表
      updateCounts() // 异步刷新统计
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '审批操作失败')
  }
}

const handleReject = (node) => {
  ElMessageBox.confirm('确定要拒绝该节点的接入申请吗？', '确认拒绝', {
    confirmButtonText: '确定拒绝',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      const res = await request.put(`/nodes/${node.id}/reject`)
    if (res.success) {
      ElMessage.warning('已拒绝该节点接入申请')
      showPendingDialog.value = false
      fetchNodes()
      fetchPendingList()
        updateCounts()
      }
    } catch {
      ElMessage.error('操作失败')
    }
  })
}

const updateCounts = async () => {
  try {
    const [resApproved, resPending] = await Promise.all([
      request.get('/nodes', { params: { pageSize: 1, approvalStatus: 'approved' } }),
      request.get('/nodes', { params: { pageSize: 1, approvalStatus: 'pending' } })
    ])
    approvedCount.value = resApproved.data.total
    pendingCount.value = resPending.data.total
  } catch (error) {
    console.error('更新节点统计失败:', error)
  }
}

const resetNodeSearch = () => {
  nodeSearch.status = ''
  nodeSearch.keyword = ''
  nodePagination.page = 1
  fetchNodes()
}

const copyToken = () => {
  const text = `REGISTRATION_TOKEN=${registrationToken.value}\nNODE_ID=${newNodeId.value}`
  window.navigator.clipboard.writeText(text)
  ElMessage.success('已复制到剪贴板')
  showTokenDialog.value = false
}

// 指标获取辅助函数
const getMetricValue = (node, type) => {
  if (!node.metrics) return 0
  switch (type) {
    case 'cpu': return Math.round((node.metrics.cpu || 0) * 100)
    case 'memory': return Math.round((node.metrics.memory || 0) * 100)
    case 'disk': return Math.round((node.metrics.disk || 0) * 100)
    default: return 0
  }
}

const getDiskLabel = (node) => {
  if (!node.metrics?.disk_label) return 'System'
  return node.metrics.disk_label
}

const getProgressColor = (percentage) => {
  if (percentage < 60) return '#10b981' // Green
  if (percentage < 85) return '#f59e0b' // Yellow
  return '#ef4444' // Red
}

const formatTime = (time) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'

// 角色标签映射
const getRoleTagType = (role) => {
  const map = {
    'SYSTEM_ADMIN': 'danger',
    'OPS_ADMIN': 'warning',
    'PROJECT_ADMIN': 'primary',
    'USER_ADMIN': 'info',
  }
  return map[role] || 'info'
}

const getRoleLabel = (role) => {
  const map = {
    'SYSTEM_ADMIN': '系统管理员',
    'OPS_ADMIN': '运维管理员',
    'PROJECT_ADMIN': '项目管理员',
    'USER_ADMIN': '用户管理员',
  }
  return map[role] || role
}

// 状态 Label 映射
const getNodeStatusType = (status) => status === 'online' ? 'success' : (status === 'offline' ? 'info' : 'danger')
const getNodeStatusLabel = (status) => {
  const map = { online: '在线', offline: '离线', error: '监控异常' }
  return map[status] || status
}

const getDeployStatusType = (status) => {
  const map = { running: 'success', stopped: 'info', deploying: 'warning', error: 'danger' }
  return map[status] || 'info'
}

const getDeployLabel = (status) => {
  const map = { running: '运行中', stopped: '已停止', deploying: '部署中', error: '故障', pending: '等待中' }
  return map[status] || status
}

// 运维操作
const restartNode = (node) => ElMessage.info(`正在重启 ${node.name} Agent...`)
const deleteNode = async (node) => {
  try {
    await ElMessageBox.confirm(`确定要注销节点 "${node.name}" 吗？此操作不可撤销。`, '警告', { type: 'warning' })
    await request.delete(`/nodes/${node.id}`)
    ElMessage.success('节点注销成功')
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '节点注销失败')
    }
  }
}

const promptDeploy = (node) => ElMessage.info(`请在工程列表中选择版本并部署到节点 ${node.name}`)

// 新增：启动工程
const handleStartProject = async (deploy) => {
  try {
    await request.post(`/deployments/node-deployment/${deploy.id}/start`)
    ElMessage.success('启动指令已下发')
    fetchNodes()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '启动失败')
  }
}

// 新增：停止工程
const handleStopProject = async (deploy) => {
  try {
    await ElMessageBox.confirm(
      `确定要停止 ${deploy.project?.name} 吗？`,
      '确认停止',
      { type: 'warning' }
    )
    await request.post(`/deployments/node-deployment/${deploy.id}/stop`)
    ElMessage.success('停止指令已下发')
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.message || '停止失败')
  }
}

// 新增：查看日志
const handleViewLog = (deploy) => {
  currentDeployment.value = deploy
  logContent.value = deploy.deployLog || []
  showLogDialog.value = true
}

// 新增：重启工程
const handleRestartProject = async (deploy) => {
  try {
    await ElMessageBox.confirm(
      `确定要重启 ${deploy.project?.name} 吗？`,
      '确认重启',
      { type: 'warning' }
    )
    await request.post(`/deployments/node-deployment/${deploy.id}/restart`)
    ElMessage.success('重启指令已下发')
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.message || '重启失败')
  }
}

// 新增：回滚版本（仅RELEASE模式）
const handleRollback = async (deploy) => {
  if (deploy.mode === 'DEV') {
    return ElMessage.warning('DEV模式不支持回滚操作')
  }

  try {
    await ElMessageBox.confirm(
      `确定要回滚到版本 ${deploy.version} 吗？这将重新部署该版本。`,
      '确认回滚',
      { type: 'warning' }
    )

    await request.post(`/deployments/${deploy.deploymentId}/rollback`, {
      nodeId: deploy.nodeId || nodeList.value.find(n => n.id === deploy.nodeId)?.id,
    })

    ElMessage.success('回滚任务已创建')
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.message || '回滚失败')
  }
}

// 新增：撤销部署
const handleUndeploy = async (deploy) => {
  try {
    await ElMessageBox.confirm(
      `确定要从节点撤销部署 ${deploy.project?.name} 吗？这将停止运行并释放资源。`,
      '确认撤销',
      { type: 'warning' }
    )

    await request.delete(`/deployments/node-deployment/${deploy.id}`)
    ElMessage.success('撤销部署成功')
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.message || '撤销失败')
  }
}

// WebSocket 实时处理
const setupRealtimeUpdates = () => {
  const tenantId = Storage.getTenantId() || 'default'
  const socket = initSocket(tenantId)

  // 监听节点指标更新
  socket.on('ops:node:metrics', (data) => {
    const node = nodeList.value.find(n => n.id === data.nodeId)
    if (node) {
      node.metrics = data.metrics
      node.lastHeartbeatAt = data.timestamp
    }
  })

  // 监听节点状态变化
  socket.on('ops:node:status', (data) => {
    const node = nodeList.value.find(n => n.id === data.nodeId)
    if (node) {
      node.status = data.status
      node.lastHeartbeatAt = data.timestamp
    }
  })

  // 监听工程指标更新
  socket.on('ops:project:metrics', (data) => {
    const node = nodeList.value.find(n => n.id === data.nodeId)
    if (node) {
      const deploy = node.deployments?.find(d => d.projectId === data.projectId)
      if (deploy) {
        deploy.runtimeMetrics = data.metrics
      }
    }
  })
}

// 挂载与卸载
onMounted(async () => {
  await fetchNodes()
  fetchPendingList()
  updateCounts()
  setupRealtimeUpdates()
})

onBeforeUnmount(() => {
  // 注意：此处不推荐直接 closeSocket，因为其他组件可能还在使用
  // 但可以取消事件监听
  const socket = getSocket()
  if (socket) {
    socket.off('ops:node:metrics')
    socket.off('ops:node:status')
    socket.off('ops:project:metrics')
  }
})
</script>

<style scoped>
.ops-management {
  background-color: #f8fafc;
}
.dark .ops-management {
  background-color: #0f172a;
}
.node-card {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.node-card:hover {
  transform: translateY(-4px);
}
:deep(.el-progress-circle) {
  margin: 0 auto;
}
:deep(.el-card__header) {
  padding: 12px 16px;
}
</style>
