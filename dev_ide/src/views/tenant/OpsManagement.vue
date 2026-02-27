<template>
  <div class="ops-management h-full flex flex-col">
    <!-- 页面标题和页头操作 -->
    <div class="flex justify-between items-center mb-6 px-6 pt-6">
      <div class="flex items-center space-x-4">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('opsManagement.title') }}</h1>
        <el-tag border size="large" type="info" class="rounded-full">
          {{ t('opsManagement.totalNodes') }}: {{ nodePagination.total }}
        </el-tag>
      </div>
      <div class="flex items-center space-x-3">
        <el-radio-group v-model="activeView" size="default">
          <el-radio-button value="dashboard">
            <el-icon class="mr-1"><Grid /></el-icon> {{ t('opsManagement.dashboardView') }}
          </el-radio-button>
          <el-radio-button value="list">
            <el-icon class="mr-1"><List /></el-icon> {{ t('opsManagement.listView') }}
          </el-radio-button>
        </el-radio-group>
        <el-button v-if="canApproveNode" type="primary" @click="showAddNodeDialog = true">
          {{ t('opsManagement.registerNode') }}
        </el-button>
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
          <el-form-item :label="t('opsManagement.nodeStatus')">
            <el-select v-model="nodeSearch.status" :placeholder="t('opsManagement.allStatus')" clearable style="width: 120px">
              <el-option :label="t('opsManagement.online')" value="online" />
              <el-option :label="t('opsManagement.offline')" value="offline" />
              <el-option :label="t('opsManagement.abnormal')" value="error" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('opsManagement.searchNode')">
            <el-input
              v-model="nodeSearch.keyword"
              :placeholder="t('opsManagement.searchPlaceholder')"
              clearable
              :prefix-icon="Search"
              style="width: 220px"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="fetchNodes">{{ t('opsManagement.query') }}</el-button>
            <el-button @click="resetNodeSearch">{{ t('opsManagement.reset') }}</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <div v-if="nodeLoadError" class="px-6 mb-4">
      <el-alert
        :title="nodeLoadError"
        type="error"
        show-icon
        :closable="false"
      >
        <template #default>
          <el-button text type="primary" @click="fetchNodes">{{ t('opsManagement.reload') }}</el-button>
        </template>
      </el-alert>
    </div>

    <div class="px-6 mb-6">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="status-stat-card">
          <div class="status-stat-label">{{ t('opsManagement.running') }}</div>
          <div class="status-stat-value text-status-success">
            {{ deployStatusSummary.running }}
          </div>
        </div>
        <div class="status-stat-card">
          <div class="status-stat-label">{{ t('opsManagement.deploying') }}</div>
          <div class="status-stat-value text-status-warning">
            {{ deployStatusSummary.deploying }}
          </div>
        </div>
        <div class="status-stat-card">
          <div class="status-stat-label">{{ t('opsManagement.stopped') }}</div>
          <div class="status-stat-value text-gray-700 dark:text-gray-300">
            {{ deployStatusSummary.stopped }}
          </div>
        </div>
        <div class="status-stat-card">
          <div class="status-stat-label">{{ t('opsManagement.abnormal') }}</div>
          <div class="status-stat-value text-status-danger">
            {{ deployStatusSummary.failed }}
          </div>
        </div>
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
                    <el-dropdown-item @click="viewNodeDetail(node)">{{ t('opsManagement.detailInfo') }}</el-dropdown-item>
                    <el-dropdown-item @click="restartNode(node)" :disabled="node.status !== 'online'">{{ t('opsManagement.restartAgent') }}</el-dropdown-item>
                    <el-dropdown-item divided @click="deleteNode(node)" type="danger">{{ t('opsManagement.deleteRegistration') }}</el-dropdown-item>
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
                <div class="text-[10px] text-gray-400 uppercase mb-1">{{ t('opsManagement.cpuUsage') }}</div>
                <el-progress
                  type="dashboard"
                  :percentage="getMetricValue(node, 'cpu')"
                  :width="60"
                  :stroke-width="4"
                  :color="getProgressColor"
                />
              </div>
              <div class="text-center">
                <div class="text-[10px] text-gray-400 uppercase mb-1">{{ t('opsManagement.memoryUsage') }}</div>
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
                <span>{{ t('opsManagement.diskUsage') }} ({{ getDiskLabel(node) }})</span>
                <span class="font-mono">{{ getMetricValue(node, 'disk') }}%</span>
              </div>
              <el-progress :percentage="getMetricValue(node, 'disk')" :show-text="false" :stroke-width="6" class="mb-3" />
            </div>
          </div>

          <!-- 卡片尾部：运行中工程列表 -->
          <div class="bg-gray-50/30 dark:bg-gray-900/20 p-2 border-t border-gray-100 dark:border-gray-700">
            <div class="text-[10px] font-bold text-gray-400 uppercase px-2 py-1 mb-1 flex justify-between">
              <span>{{ t('opsManagement.runningProjects') }} ({{ node.deployments?.length || 0 }})</span>
              <el-button link size="small" type="primary" class="text-[10px]" @click="promptDeploy(node)">{{ t('opsManagement.deploy') }}</el-button>
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
                          <el-icon class="mr-1"><VideoPlay /></el-icon>{{ t('opsManagement.start') }}
                        </el-dropdown-item>
                        <el-dropdown-item @click="handleStopProject(deploy)">
                          <el-icon class="mr-1"><VideoPause /></el-icon>{{ t('opsManagement.stop') }}
                        </el-dropdown-item>
                        <el-dropdown-item @click="handleRestartProject(deploy)">
                          <el-icon class="mr-1"><RefreshRight /></el-icon>{{ t('opsManagement.restart') }}
                        </el-dropdown-item>
                        <el-dropdown-item @click="handleRollback(deploy)" :disabled="deploy.mode === 'DEV'">
                          <el-icon class="mr-1"><RefreshLeft /></el-icon>{{ t('opsManagement.rollback') }}
                        </el-dropdown-item>
	                        <el-dropdown-item @click="handleViewLog(deploy)">
	                          <el-icon class="mr-1"><Document /></el-icon>{{ t('opsManagement.viewLog') }}
	                        </el-dropdown-item>
	                        <el-dropdown-item v-if="isFailedDeploy(deploy)" @click="openFailureDetail(deploy)">
	                          <el-icon class="mr-1"><Warning /></el-icon>{{ t('opsManagement.failureDetail') }}
	                        </el-dropdown-item>
	                        <el-dropdown-item divided @click="handleUndeploy(deploy)" type="danger">
	                          <el-icon class="mr-1"><Remove /></el-icon>{{ t('opsManagement.undeploy') }}
	                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </div>
            </div>
            <div v-else class="text-center py-4 text-xs text-gray-400">
              {{ t('opsManagement.noProjectsRunning') }}
            </div>
          </div>
        </el-card>
      </div>
      
      <!-- 无数据 -->
      <el-empty v-if="!nodeLoading && nodeList.length === 0" :description="t('opsManagement.noNodesOnline')" />

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
                <h4 class="text-sm font-bold mb-3">{{ t('opsManagement.runningProjects') }}</h4>
                <el-table :data="props.row.deployments" size="small" border>
                  <el-table-column :label="t('opsManagement.projectName')" prop="project.name" />
                  <el-table-column :label="t('opsManagement.runtimeVersion')" prop="version" width="100" />
                  <el-table-column :label="t('opsManagement.runtimeStatus')" width="100">
                    <template #default="scope">
                      <el-tag size="small" :type="getDeployStatusType(scope.row.status)">{{ getDeployLabel(scope.row.status) }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column :label="t('opsManagement.onlineUsers')" width="100">
                    <template #default="scope">
                      {{ scope.row.runtimeMetrics?.onlineUsers || 0 }}
                    </template>
                  </el-table-column>
                  <el-table-column :label="t('opsManagement.concurrentPeak')" width="100">
                    <template #default="scope">
                      {{ scope.row.runtimeMetrics?.concurrentUsers || 0 }}
                    </template>
                  </el-table-column>
                  <el-table-column :label="t('opsManagement.actions')" width="150">
                    <template #default="scope">
                      <el-button link type="primary" size="small" @click="handleViewLog(scope.row)">{{ t('opsManagement.log') }}</el-button>
                      <el-button
                        v-if="isFailedDeploy(scope.row)"
                        link
                        type="danger"
                        size="small"
                        @click="openFailureDetail(scope.row)"
                      >
                        {{ t('opsManagement.failureDetail') }}
                      </el-button>
                      <el-button link type="warning" size="small" @click="handleStopProject(scope.row)">{{ t('opsManagement.stop') }}</el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.nodeName')" prop="name" />
          <el-table-column :label="t('opsManagement.ipAddress')" prop="ipAddress" />
          <el-table-column :label="t('opsManagement.status')" width="120">
            <template #default="scope">
              <el-tag :type="getNodeStatusType(scope.row.status)">{{ getNodeStatusLabel(scope.row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="CPU" width="100">
            <template #default="scope">{{ getMetricValue(scope.row, 'cpu') }}%</template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.memory')" width="100">
            <template #default="scope">{{ getMetricValue(scope.row, 'memory') }}%</template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.disk')" width="100">
            <template #default="scope">{{ getMetricValue(scope.row, 'disk') }}%</template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.projectCount')" width="100">
            <template #default="scope">{{ scope.row.deployments?.length || 0 }}</template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.lastHeartbeat')" prop="lastHeartbeatAt">
            <template #default="scope">{{ formatTime(scope.row.lastHeartbeatAt) }}</template>
          </el-table-column>
        </el-table>
      </div>
      <el-empty v-if="!nodeLoading && nodeList.length === 0" :description="t('opsManagement.noNodes')" class="mt-6" />
    </div>

    <!-- 弹窗：待审核申请列表 -->
    <el-dialog v-model="showPendingDialog" :title="t('opsManagement.pendingRequests')" width="900px">
      <div class="bg-white dark:bg-gray-800 rounded-xl overflow-hidden">
        <el-table v-loading="nodeLoading" :data="pendingList" style="width: 100%">
          <el-table-column :label="t('opsManagement.requestedNodeName')" min-width="180">
            <template #default="scope">
              <div class="flex flex-col">
                <span class="font-bold text-gray-800 dark:text-gray-200">{{ scope.row.name }}</span>
                <span class="text-xs text-gray-500">{{ scope.row.description || t('opsManagement.noDescription') }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.applicantInfo')" width="180">
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
          <el-table-column :label="t('opsManagement.nodeMode')" width="100">
            <template #default="scope">
              <el-tag size="small" :type="scope.row.mode === 'online' ? 'success' : 'info'">
                {{ scope.row.mode === 'online' ? t('opsManagement.online') : t('opsManagement.offline') }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.nodeNetworkAddress')" width="180">
            <template #default="scope">
              <div class="flex flex-col text-xs">
                <span>IP: {{ scope.row.ipAddress }}</span>
                <span>Port: {{ scope.row.port }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.agentVersion')" prop="agentVersion" width="100" />
          <el-table-column :label="t('opsManagement.requestTime')" width="160">
            <template #default="scope">{{ formatTime(scope.row.createdAt) }}</template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.approveAction')" width="220" fixed="right">
            <template #default="scope">
              <div v-if="canApproveNode" class="flex space-x-2">
                <el-button type="success" size="small" @click="handleApprove(scope.row)">
                  <el-icon class="mr-1"><Check /></el-icon> {{ t('opsManagement.approve') }}
                </el-button>
                <el-button type="danger" size="small" plain @click="handleReject(scope.row)">
                  <el-icon class="mr-1"><Close /></el-icon> {{ t('opsManagement.reject') }}
                </el-button>
              </div>
              <div v-else class="text-xs text-gray-500">
                {{ t('opsManagement.approvePermissionHint') }}
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <el-button @click="showPendingDialog = false">{{ t('opsManagement.close') }}</el-button>
      </template>
    </el-dialog>

    <!-- 弹窗：注册节点 -->
    <el-dialog v-model="showAddNodeDialog" :title="t('opsManagement.registerNewNode')" width="500px">
      <el-form ref="nodeFormRef" :model="nodeForm" :rules="nodeFormRules" label-width="100px" class="mt-4">
        <el-form-item :label="t('opsManagement.nodeName')" prop="name">
          <el-input v-model="nodeForm.name" :placeholder="t('opsManagement.nodeNamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('opsManagement.nodeDesc')" prop="description">
          <el-input v-model="nodeForm.description" type="textarea" :placeholder="t('opsManagement.nodeDescPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('opsManagement.ipAddressLabel')">
          <el-input v-model="nodeForm.ipAddress" :placeholder="t('opsManagement.ipPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('opsManagement.managePort')">
          <el-input-number v-model="nodeForm.port" :min="1" :max="65535" class="w-full" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddNodeDialog = false">{{ t('opsManagement.cancel') }}</el-button>
        <el-button type="primary" @click="submitNodeForm" :loading="submitting">{{ t('opsManagement.createAndToken') }}</el-button>
      </template>
    </el-dialog>

    <!-- 弹窗：节点注册完成 -->
    <el-dialog v-model="showTokenDialog" :title="t('opsManagement.approvePassed')" width="500px" :close-on-click-modal="false">
      <el-result icon="success" :title="t('opsManagement.approveSuccessTitle')" :sub-title="t('opsManagement.approveSuccessSubtitle')">
        <template #extra>
          <div class="bg-gray-100 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 p-4 rounded-lg font-mono text-xs break-all mb-4 text-left select-all">
            REGISTRATION_TOKEN={{ registrationToken }}<br/>
            NODE_ID={{ newNodeId }}
          </div>
          <el-button type="primary" @click="copyToken">{{ t('opsManagement.copyAndClose') }}</el-button>
        </template>
      </el-result>
    </el-dialog>

    <!-- 弹窗：详细日志 -->
    <el-dialog v-model="showLogDialog" :title="t('opsManagement.runtimeLogTitle', { name: currentDeployment?.project?.name || '' })" width="800px">
      <div class="bg-black text-green-500 p-4 rounded-lg h-96 overflow-y-auto font-mono text-xs">
        <div v-for="(log, idx) in logContent" :key="idx" class="mb-1">
          <span class="text-gray-500">[{{ log.time }}]</span>
          <span class="ml-2">{{ log.message }}</span>
        </div>
        <div v-if="logContent.length === 0" class="text-gray-500 text-center mt-20">{{ t('opsManagement.noRealtimeLog') }}</div>
      </div>
    </el-dialog>

    <el-drawer v-model="showFailureDrawer" :title="t('opsManagement.failureDrawerTitle', { name: failedDeployment?.project?.name || '-' })" size="520px">
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="t('opsManagement.nodeName')">
          {{ failedDeployment?.node?.name || failedDeployment?.nodeName || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('opsManagement.runtimeVersion')">
          {{ failedDeployment?.version || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('opsManagement.status')">
          <el-tag :type="getDeployStatusType(failedDeployment?.status)">
            {{ getDeployLabel(failedDeployment?.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('opsManagement.failureReason')">
          {{ getDeployFailureReason(failedDeployment) }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('opsManagement.lastUpdated')">
          {{ formatTime(failedDeployment?.updatedAt || failedDeployment?.lastHeartbeatAt) }}
        </el-descriptions-item>
      </el-descriptions>

      <div class="mt-4">
        <div class="text-sm font-semibold text-gray-700 dark:text-gray-200 mb-2">{{ t('opsManagement.recentLogs') }}</div>
        <div class="bg-black text-green-400 rounded p-3 h-56 overflow-y-auto text-xs font-mono">
          <template v-if="failedDeployLogs.length > 0">
            <div v-for="(line, idx) in failedDeployLogs" :key="idx" class="mb-1">
              [{{ line.time || '-' }}] {{ line.message || line }}
            </div>
          </template>
          <div v-else class="text-gray-500 text-center mt-20">{{ t('opsManagement.noFailureLogs') }}</div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Grid, List, Refresh, Search, MoreFilled,
  User, Clock, Tools, Check, Close, Bell, Warning
} from '@element-plus/icons-vue'
import request from '@/utils/request'
import dayjs from 'dayjs'
import { initSocket, getSocket } from '@/utils/socket'
import { Storage } from '@/utils/storage'
import { canApproveNodes } from '@/permissions'
import { useAuthStore } from '@/store'
import { RoleEnum, ENUM_LABELS } from '@/enums'
import {
  getNodeStatusType,
  getNodeStatusLabel,
  getDeployStatusType,
  getDeployLabel,
  isFailedDeploy,
  getDeployFailureReason,
  buildDeployStatusSummary,
  getProgressColor,
} from './utils/ops-status'

const authStore = useAuthStore()
const { t } = useI18n()
const currentUserRole = computed(() => authStore.userInfo?.role || Storage.getUserInfo()?.role || '')

// 计算是否有审批权限
const canApproveNode = computed(() => {
  return canApproveNodes(currentUserRole.value)
})

// 状态与视图控制
const activeView = ref('dashboard')
const nodeLoading = ref(false)
const nodeLoadError = ref('')
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
const showAddNodeDialog = ref(false)
const submitting = ref(false)
const showTokenDialog = ref(false)
const registrationToken = ref('')
const newNodeId = ref('')
const nodeFormRef = ref(null)
const nodeForm = reactive({
  name: '',
  description: '',
  ipAddress: '',
  port: 8080,
})
const nodeFormRules = {
  name: [{ required: true, message: t('opsManagement.inputNodeName'), trigger: 'blur' }],
}

// 待审核申请弹窗
const showPendingDialog = ref(false)
const pendingList = ref([])

// 日志弹窗
const showLogDialog = ref(false)
const currentDeployment = ref(null)
const logContent = ref([])
const showFailureDrawer = ref(false)
const failedDeployment = ref(null)
const failedDeployLogs = ref([])

const deployStatusSummary = computed(() => {
  return buildDeployStatusSummary(nodeList.value)
})

// 获取节点数据
const fetchNodes = async () => {
  nodeLoading.value = true
  nodeLoadError.value = ''
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
    nodeLoadError.value = error.response?.data?.message || t('opsManagement.fetchNodesFailed')
    ElMessage.error(t('opsManagement.fetchNodesFailed'))
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
      ElMessage.success(t('opsManagement.approvePassed'))
      registrationToken.value = res.data.registrationToken
      newNodeId.value = res.data.id
      showTokenDialog.value = true
      showPendingDialog.value = false
      fetchNodes()
      fetchPendingList() // 刷新待审核列表
      updateCounts() // 异步刷新统计
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('opsManagement.approveFailed'))
  }
}

const handleReject = (node) => {
  ElMessageBox.confirm(t('opsManagement.rejectConfirm'), t('opsManagement.rejectConfirmTitle'), {
    confirmButtonText: t('opsManagement.rejectConfirmBtn'),
    cancelButtonText: t('opsManagement.cancel'),
    type: 'warning'
  }).then(async () => {
    try {
      const res = await request.put(`/nodes/${node.id}/reject`)
    if (res.success) {
      ElMessage.warning(t('opsManagement.rejectedTip'))
      showPendingDialog.value = false
      fetchNodes()
      fetchPendingList()
        updateCounts()
      }
    } catch {
      ElMessage.error(t('opsManagement.operationFailed'))
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

const resetNodeForm = () => {
  nodeForm.name = ''
  nodeForm.description = ''
  nodeForm.ipAddress = ''
  nodeForm.port = 8080
  nodeFormRef.value?.clearValidate?.()
}

const submitNodeForm = async () => {
  if (!nodeFormRef.value) return
  try {
    await nodeFormRef.value.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    let res
    try {
      res = await request.post('/nodes/register', { ...nodeForm })
    } catch {
      res = await request.post('/nodes', { ...nodeForm })
    }

    if (res?.success) {
      ElMessage.success(t('opsManagement.submittedTip'))
      showAddNodeDialog.value = false
      resetNodeForm()
      await Promise.all([fetchPendingList(), updateCounts()])
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || t('opsManagement.createNodeFailed'))
  } finally {
    submitting.value = false
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
  ElMessage.success(t('opsManagement.copied'))
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
  if (!node.metrics?.disk_label) return t('opsManagement.systemDisk')
  return node.metrics.disk_label
}

const formatTime = (time) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'

// 角色标签映射
const getRoleTagType = (role) => {
  const map = {
    [RoleEnum.SUPER_ADMIN]: 'danger',
    [RoleEnum.SYSTEM_ADMIN]: 'danger',
    [RoleEnum.OPS_ADMIN]: 'warning',
    [RoleEnum.PROJECT_ADMIN]: 'primary',
    [RoleEnum.USER_ADMIN]: 'info',
  }
  return map[role] || 'info'
}

const getRoleLabel = (role) => {
  return ENUM_LABELS[role] || role
}

// 运维操作
const restartNode = (node) => ElMessage.info(t('opsManagement.restartingAgent', { name: node.name }))
const viewNodeDetail = (node) => {
  ElMessageBox.alert(
    t('opsManagement.nodeDetailBody', {
      name: node?.name || '-',
      ip: node?.ipAddress || '-',
      port: node?.port || '-',
      status: getNodeStatusLabel(node?.status),
    }),
    t('opsManagement.nodeDetailTitle'),
    { confirmButtonText: t('opsManagement.close') }
  )
}
const deleteNode = async (node) => {
  try {
    await ElMessageBox.confirm(
      t('opsManagement.nodeDeleteConfirm', { name: node.name }),
      t('opsManagement.warning'),
      { type: 'warning' }
    )
    await request.delete(`/nodes/${node.id}`)
    ElMessage.success(t('opsManagement.nodeDeleted'))
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || t('opsManagement.nodeDeleteFailed'))
    }
  }
}

const promptDeploy = (node) => ElMessage.info(t('opsManagement.deployHint', { name: node.name }))

// 新增：启动工程
const handleStartProject = async (deploy) => {
  try {
    await request.post(`/deployments/node-deployment/${deploy.id}/start`)
    ElMessage.success(t('opsManagement.startIssued'))
    fetchNodes()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || t('opsManagement.startFailed'))
  }
}

// 新增：停止工程
const handleStopProject = async (deploy) => {
  try {
    await ElMessageBox.confirm(
      t('opsManagement.stopConfirm', { name: deploy.project?.name }),
      t('opsManagement.stopConfirmTitle'),
      { type: 'warning' }
    )
    await request.post(`/deployments/node-deployment/${deploy.id}/stop`)
    ElMessage.success(t('opsManagement.stopIssued'))
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.message || t('opsManagement.stopFailed'))
  }
}

// 新增：查看日志
const handleViewLog = (deploy) => {
  currentDeployment.value = deploy
  logContent.value = deploy.deployLog || []
  showLogDialog.value = true
}

const openFailureDetail = (deploy) => {
  failedDeployment.value = deploy
  failedDeployLogs.value = Array.isArray(deploy?.deployLog) ? deploy.deployLog.slice(-50) : []
  showFailureDrawer.value = true
}

// 新增：重启工程
const handleRestartProject = async (deploy) => {
  try {
    await ElMessageBox.confirm(
      t('opsManagement.restartConfirm', { name: deploy.project?.name }),
      t('opsManagement.restartConfirmTitle'),
      { type: 'warning' }
    )
    await request.post(`/deployments/node-deployment/${deploy.id}/restart`)
    ElMessage.success(t('opsManagement.restartIssued'))
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.message || t('opsManagement.restartFailed'))
  }
}

// 新增：回滚版本（仅RELEASE模式）
const handleRollback = async (deploy) => {
  if (deploy.mode === 'DEV') {
    return ElMessage.warning(t('opsManagement.rollbackUnsupported'))
  }

  try {
    await ElMessageBox.confirm(
      t('opsManagement.rollbackConfirm', { version: deploy.version }),
      t('opsManagement.rollbackConfirmTitle'),
      { type: 'warning' }
    )

    await request.post(`/deployments/${deploy.deploymentId}/rollback`, {
      nodeId: deploy.nodeId || nodeList.value.find(n => n.id === deploy.nodeId)?.id,
    })

    ElMessage.success(t('opsManagement.rollbackCreated'))
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.message || t('opsManagement.rollbackFailed'))
  }
}

// 新增：撤销部署
const handleUndeploy = async (deploy) => {
  try {
    await ElMessageBox.confirm(
      t('opsManagement.undeployConfirm', { name: deploy.project?.name }),
      t('opsManagement.undeployConfirmTitle'),
      { type: 'warning' }
    )

    await request.delete(`/deployments/node-deployment/${deploy.id}`)
    ElMessage.success(t('opsManagement.undeploySuccess'))
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.message || t('opsManagement.undeployFailed'))
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

.status-stat-card {
  @apply bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4;
}

.status-stat-label {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.status-stat-value {
  @apply text-2xl font-semibold mt-2;
}
:deep(.el-progress-circle) {
  margin: 0 auto;
}
:deep(.el-card__header) {
  padding: 12px 16px;
}
</style>
