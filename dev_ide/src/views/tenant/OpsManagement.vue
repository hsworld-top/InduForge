<template>
  <div class="ops-management ck-workbench-page">
    <!-- 一体化操作栏 (Cockpit-style) -->
    <div class="ck-workbench-toolbar">
      <!-- 左侧：标题、总数与搜索筛选 -->
      <div class="ck-toolbar-left">
        <span class="ck-toolbar-count">
          {{ t('opsManagement.totalNodes') }}: {{ nodePagination.total }}
        </span>

        <!-- 圆角搜索框 -->
        <el-input
          v-model="nodeSearch.keyword"
          size="small"
          :placeholder="t('opsManagement.searchPlaceholder')"
          clearable
          class="ck-toolbar-search rounded-full-input"
          prefix-icon="Search"
        />

        <!-- 筛选栏 -->
        <div class="ck-toolbar-filters">
          <el-popover placement="bottom-start" :width="180" trigger="click">
            <template #reference>
              <button type="button" class="ck-toolbar-pill-btn">
                <el-icon><FolderOpened /></el-icon>
                {{ selectedProjectFilterLabel }}
              </button>
            </template>
            <div class="sort-popover-menu">
              <button
                type="button"
                class="sort-popover-item"
                :class="{ 'is-active': !nodeSearch.projectName }"
                @click="setNodeProjectFilter('')"
              >
                {{ t('projectManagement.allProjects') }}
              </button>
              <button
                v-for="item in projectOptions"
                :key="item.id"
                type="button"
                class="sort-popover-item"
                :class="{ 'is-active': nodeSearch.projectName === item.name }"
                @click="setNodeProjectFilter(item.name)"
              >
                {{ item.name }}
              </button>
            </div>
          </el-popover>

          <el-popover placement="bottom-start" :width="160" trigger="click">
            <template #reference>
              <button type="button" class="ck-toolbar-pill-btn">
                <el-icon><Connection /></el-icon>
                {{ selectedNodeStatusLabel }}
              </button>
            </template>
            <div class="sort-popover-menu">
              <button
                v-for="option in nodeStatusFilterOptions"
                :key="option.value || 'all'"
                type="button"
                class="sort-popover-item"
                :class="{ 'is-active': nodeSearch.status === option.value }"
                @click="setNodeStatusFilter(option.value)"
              >
                {{ option.label }}
              </button>
            </div>
          </el-popover>
        </div>
      </div>

      <!-- 右侧：视图切换与全局操作 -->
      <div class="ck-toolbar-right">
        <!-- 视图切换 -->
        <div class="ck-view-toggle">
          <button
            @click="activeView = 'dashboard'"
            :class="['ck-view-toggle__button', activeView === 'dashboard' ? 'is-active' : '']"
          >
            <el-icon><Grid /></el-icon>
          </button>
          <button
            @click="activeView = 'list'"
            :class="['ck-view-toggle__button', activeView === 'list' ? 'is-active' : '']"
          >
            <svg
              class="w-4 h-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="8" y1="6" x2="21" y2="6"></line>
              <line x1="8" y1="12" x2="21" y2="12"></line>
              <line x1="8" y1="18" x2="21" y2="18"></line>
              <line x1="3" y1="6" x2="3.01" y2="6"></line>
              <line x1="3" y1="12" x2="3.01" y2="12"></line>
              <line x1="3" y1="18" x2="3.01" y2="18"></line>
            </svg>
          </button>
        </div>

        <el-badge
          v-if="canApproveNode"
          :value="pendingCount"
          :hidden="pendingCount === 0"
          class="cursor-pointer"
          @click="showPendingDialog = true"
        >
          <button :class="['ck-icon-button', pendingCount > 0 ? 'ck-icon-button--warning' : '']">
            <el-icon><Bell /></el-icon>
          </button>
        </el-badge>

        <el-tooltip :content="t('common.refresh')" placement="top">
          <button class="ck-icon-button" @click="fetchNodes">
            <el-icon><RefreshRight /></el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>

    <div v-if="nodeLoadError">
      <el-alert :title="nodeLoadError" type="error" show-icon :closable="false">
        <template #default>
          <el-button text type="primary" @click="fetchNodes">{{
            t('opsManagement.reload')
          }}</el-button>
        </template>
      </el-alert>
    </div>

    <div class="ck-stat-grid">
      <div class="ck-stat-card">
        <div class="ck-stat-card__label">{{ t('opsManagement.running') }}</div>
        <div class="ck-stat-card__value text-status-success">
          {{ deployStatusSummary.running }}
        </div>
      </div>
      <div class="ck-stat-card">
        <div class="ck-stat-card__label">
          {{ t('opsManagement.deploying') }}
        </div>
        <div class="ck-stat-card__value text-status-warning">
          {{ deployStatusSummary.deploying }}
        </div>
      </div>
      <div class="ck-stat-card">
        <div class="ck-stat-card__label">{{ t('opsManagement.stopped') }}</div>
        <div class="ck-stat-card__value text-gray-700 dark:text-gray-300">
          {{ deployStatusSummary.stopped }}
        </div>
      </div>
      <div class="ck-stat-card">
        <div class="ck-stat-card__label">{{ t('opsManagement.abnormal') }}</div>
        <div class="ck-stat-card__value text-status-danger">
          {{ deployStatusSummary.failed }}
        </div>
      </div>
    </div>

    <div class="ck-content-area">
      <!-- 视图：节点大盘 -->
      <div v-if="activeView === 'dashboard'" class="ck-content-scroll">
        <div
          v-loading="nodeLoading"
          :class="['grid grid-cols-1 md:grid-cols-2 gap-6', cardGridClass]"
        >
          <el-card
            v-for="node in filteredNodeList"
            :key="node.id"
            shadow="hover"
            class="node-card border-none rounded-2xl ring-1 ring-gray-200 dark:ring-gray-700 shadow-[0_2px_12px_rgba(0,0,0,0.04)] hover:shadow-[0_8px_24px_rgba(0,0,0,0.08)] hover:-translate-y-0.5 transition-all duration-300"
            :body-style="{ padding: '0px' }"
          >
            <!-- 卡片头部：状态与基本信息 -->
            <div
              class="p-4 border-b border-gray-100 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/50"
            >
              <div class="flex justify-between items-start mb-2">
                <div class="flex items-center">
                  <div
                    class="w-3 h-3 rounded-full mr-2"
                    :class="
                      node.status === 'online'
                        ? 'bg-green-500 animate-pulse'
                        : node.status === 'offline'
                          ? 'bg-gray-400'
                          : 'bg-red-500'
                    "
                  ></div>
                  <h3 class="font-bold text-gray-800 dark:text-gray-200 truncate">
                    {{ node.name }}
                  </h3>
                </div>
                <el-dropdown trigger="click">
                  <el-button link
                    ><el-icon><MoreFilled /></el-icon
                  ></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item @click="deleteNode(node)" type="danger">{{
                        t('opsManagement.deleteRegistration')
                      }}</el-dropdown-item>
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
                  <div class="text-[10px] text-gray-400 uppercase mb-1">
                    {{ t('opsManagement.cpuUsage') }}
                  </div>
                  <el-progress
                    type="dashboard"
                    :percentage="getMetricValue(node, 'cpu')"
                    :width="60"
                    :stroke-width="4"
                    :color="getProgressColor"
                  />
                </div>
                <div class="text-center">
                  <div class="text-[10px] text-gray-400 uppercase mb-1">
                    {{ t('opsManagement.memoryUsage') }}
                  </div>
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
                <el-progress
                  :percentage="getMetricValue(node, 'disk')"
                  :show-text="false"
                  :stroke-width="6"
                  class="mb-3"
                />
              </div>
            </div>

            <!-- 卡片尾部：运行中工程列表 -->
            <div
              class="bg-gray-50/30 dark:bg-gray-900/20 p-2 border-t border-gray-100 dark:border-gray-700"
            >
              <div
                class="text-[10px] font-bold text-gray-400 uppercase px-2 py-1 mb-1 flex justify-between"
              >
                <span
                  >{{ t('opsManagement.runningProjects') }} ({{
                    getVisibleDeployments(node).length
                  }})</span
                >
                <el-button
                  link
                  size="small"
                  type="primary"
                  class="text-[10px]"
                  @click="openProjectManagement"
                >
                  {{ t('projectManagement.publishAndDeploy') }}
                </el-button>
              </div>

              <div v-if="getVisibleDeployments(node).length > 0" class="space-y-1">
                <div
                  v-for="deploy in getVisibleDeployments(node)"
                  :key="deploy.id"
                  class="bg-white dark:bg-gray-800 rounded p-2 text-xs ring-1 ring-gray-100 dark:ring-gray-700 flex justify-between items-center"
                >
                  <div class="flex flex-col">
                    <span class="font-semibold text-gray-700 dark:text-gray-300 truncate w-32">
                      {{ deploy.project?.name }}
                    </span>
                    <div class="flex items-center space-x-2 text-[10px] text-gray-500">
                      <span class="flex items-center"
                        ><el-icon class="mr-0.5"><User /></el-icon>
                        {{ deploy.runtimeMetrics?.onlineUsers || 0 }}</span
                      >
                      <span class="flex items-center"
                        ><el-icon class="mr-0.5"><Clock /></el-icon>
                        {{ deploy.runtimeMetrics?.concurrentUsers || 0 }}</span
                      >
                    </div>
                  </div>
                  <div class="flex items-center space-x-1">
                    <el-tag size="small" :type="getDeployStatusType(deploy.status)">
                      {{ getDeployDisplayLabel(deploy) }}
                    </el-tag>
                    <el-tag size="small" :type="deploy.mode === 'DEV' ? 'warning' : 'success'">
                      {{ getDeployModeLabel(deploy.mode) }}
                    </el-tag>
                    <el-dropdown trigger="hover">
                      <el-button link
                        ><el-icon size="small"><Tools /></el-icon
                      ></el-button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item @click="handleStartProject(deploy)">
                            <el-icon class="mr-1"><VideoPlay /></el-icon
                            >{{ t('opsManagement.start') }}
                          </el-dropdown-item>
                          <el-dropdown-item @click="handleStopProject(deploy)">
                            <el-icon class="mr-1"><VideoPause /></el-icon
                            >{{ t('opsManagement.stop') }}
                          </el-dropdown-item>
                          <el-dropdown-item @click="handleRestartProject(deploy)">
                            <el-icon class="mr-1"><RefreshRight /></el-icon
                            >{{ t('opsManagement.restart') }}
                          </el-dropdown-item>
                          <el-dropdown-item
                            @click="handleRollback(deploy)"
                            :disabled="deploy.mode === 'DEV'"
                          >
                            <el-icon class="mr-1"><RefreshLeft /></el-icon
                            >{{ t('opsManagement.rollback') }}
                          </el-dropdown-item>
                          <el-dropdown-item @click="handleViewLog(deploy)">
                            <el-icon class="mr-1"><Document /></el-icon
                            >{{ t('opsManagement.viewLog') }}
                          </el-dropdown-item>
                          <el-dropdown-item
                            v-if="isFailedDeploy(deploy)"
                            @click="openFailureDetail(deploy)"
                          >
                            <el-icon class="mr-1"><Warning /></el-icon
                            >{{ t('opsManagement.failureDetail') }}
                          </el-dropdown-item>
                          <el-dropdown-item divided @click="handleUndeploy(deploy)" type="danger">
                            <el-icon class="mr-1"><Remove /></el-icon
                            >{{ t('opsManagement.undeploy') }}
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
        <el-empty
          v-if="!nodeLoading && filteredNodeList.length === 0"
          :description="t('opsManagement.noNodesOnline')"
        />
      </div>

      <!-- 视图：详细列表 -->
      <div v-else-if="activeView === 'list'" class="ck-content-scroll">
        <div class="ck-table-shell">
          <el-table
            :data="filteredNodeList"
            style="width: 100%"
            row-key="id"
            :expand-row-keys="expandedNodeRowKeys"
            @expand-change="handleExpandChange"
          >
            <el-table-column type="expand">
              <template #default="props">
                <div class="p-4 bg-gray-50/50 dark:bg-gray-900/50">
                  <h4 class="text-sm font-bold mb-3">
                    {{ t('opsManagement.runningProjects') }}
                  </h4>
                  <el-table :data="getVisibleDeployments(props.row)" size="small" border>
                    <el-table-column :label="t('opsManagement.projectName')" prop="project.name" />
                    <el-table-column
                      :label="t('opsManagement.runtimeVersion')"
                      prop="version"
                      width="100"
                    />
                    <el-table-column :label="t('opsManagement.runtimeStatus')" width="100">
                      <template #default="scope">
                        <el-tag size="small" :type="getDeployStatusType(scope.row.status)">{{
                          getDeployDisplayLabel(scope.row)
                        }}</el-tag>
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
                    <el-table-column :label="t('opsManagement.actions')" width="420">
                      <template #default="scope">
                        <div class="flex flex-wrap gap-1">
                          <el-button
                            link
                            type="success"
                            size="small"
                            @click="handleStartProject(scope.row)"
                          >
                            {{ t('opsManagement.start') }}
                          </el-button>
                          <el-button
                            link
                            type="warning"
                            size="small"
                            @click="handleStopProject(scope.row)"
                          >
                            {{ t('opsManagement.stop') }}
                          </el-button>
                          <el-button
                            link
                            type="primary"
                            size="small"
                            @click="handleRestartProject(scope.row)"
                          >
                            {{ t('opsManagement.restart') }}
                          </el-button>
                          <el-button
                            link
                            type="primary"
                            size="small"
                            :disabled="scope.row.mode === 'DEV'"
                            @click="handleRollback(scope.row)"
                          >
                            {{ t('opsManagement.rollback') }}
                          </el-button>
                          <el-button
                            link
                            type="primary"
                            size="small"
                            @click="handleViewLog(scope.row)"
                          >
                            {{ t('opsManagement.viewLog') }}
                          </el-button>
                          <el-button
                            v-if="isFailedDeploy(scope.row)"
                            link
                            type="danger"
                            size="small"
                            @click="openFailureDetail(scope.row)"
                          >
                            {{ t('opsManagement.failureDetail') }}
                          </el-button>
                          <el-button
                            link
                            type="danger"
                            size="small"
                            @click="handleUndeploy(scope.row)"
                          >
                            {{ t('opsManagement.undeploy') }}
                          </el-button>
                        </div>
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
                <el-tag :type="getNodeStatusType(scope.row.status)">{{
                  getNodeStatusLabel(scope.row.status)
                }}</el-tag>
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
              <template #default="scope">{{ getVisibleDeployments(scope.row).length }}</template>
            </el-table-column>
            <el-table-column :label="t('opsManagement.lastHeartbeat')" prop="lastHeartbeatAt">
              <template #default="scope">{{ formatTime(scope.row.lastHeartbeatAt) }}</template>
            </el-table-column>
          </el-table>
        </div>
        <el-empty
          v-if="!nodeLoading && filteredNodeList.length === 0"
          :description="t('opsManagement.noNodes')"
          class="mt-6"
        />
      </div>

      <!-- 固定分页区 -->
      <div class="ck-pagination-bar ops-pagination-bar">
        <WorkbenchPagination
          :page="nodePagination.page"
          :limit="nodePagination.pageSize"
          :total="nodePagination.total"
          :total-pages="nodePaginationTotalPages"
          :summary="nodePaginationSummary"
          :page-size-label="t('projectManagement.pageSizeLabel')"
          :page-indicator="nodePaginationPageIndicator"
          :limit-options="[12, 24, 48]"
          @change="handleNodePaginationChange"
        />
      </div>
    </div>

    <!-- 弹窗：待审核申请列表 -->
    <el-dialog
      v-model="showPendingDialog"
      :title="t('opsManagement.pendingRequests')"
      width="900px"
    >
      <div class="bg-white dark:bg-gray-800 rounded-xl overflow-hidden">
        <el-table v-loading="nodeLoading" :data="pendingList" style="width: 100%">
          <el-table-column :label="t('opsManagement.requestedNodeName')" min-width="180">
            <template #default="scope">
              <div class="flex flex-col">
                <span class="font-bold text-gray-800 dark:text-gray-200">{{ scope.row.name }}</span>
                <span class="text-xs text-gray-500">{{
                  scope.row.description || t('opsManagement.noDescription')
                }}</span>
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
                {{
                  scope.row.mode === 'online'
                    ? t('opsManagement.online')
                    : t('opsManagement.offline')
                }}
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
          <el-table-column
            :label="t('opsManagement.agentVersion')"
            prop="agentVersion"
            width="100"
          />
          <el-table-column :label="t('opsManagement.requestTime')" width="160">
            <template #default="scope">{{ formatTime(scope.row.createdAt) }}</template>
          </el-table-column>
          <el-table-column :label="t('opsManagement.approveAction')" width="220" fixed="right">
            <template #default="scope">
              <div v-if="canApproveNode" class="flex space-x-2">
                <el-button type="success" size="small" @click="handleApprove(scope.row)">
                  <el-icon class="mr-1"><Check /></el-icon>
                  {{ t('opsManagement.approve') }}
                </el-button>
                <el-button type="danger" size="small" plain @click="handleReject(scope.row)">
                  <el-icon class="mr-1"><Close /></el-icon>
                  {{ t('opsManagement.reject') }}
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

    <!-- 弹窗：详细日志 -->
    <el-dialog
      v-model="showLogDialog"
      :title="
        t('opsManagement.runtimeLogTitle', {
          name: currentDeployment?.project?.name || '',
        })
      "
      width="700px"
    >
      <div class="bg-black text-green-500 p-4 rounded-lg h-80 overflow-y-auto font-mono text-xs">
        <div v-for="(log, idx) in logContent" :key="idx" class="mb-1">
          <span class="text-gray-500">[{{ log.time }}]</span>
          <span class="ml-2">{{ log.message }}</span>
        </div>
        <div v-if="logContent.length === 0" class="text-gray-500 text-center mt-20">
          {{ t('opsManagement.noRealtimeLog') }}
        </div>
      </div>
      <div class="mt-4">
        <div class="text-sm font-semibold text-gray-700 dark:text-gray-200 mb-2">
          {{ t('opsManagement.commandTimeline') }}
        </div>
        <el-table :data="commandTimeline" size="small" border max-height="220">
          <el-table-column prop="type" :label="t('opsManagement.commandType')" width="120" />
          <el-table-column :label="t('opsManagement.status')" width="140">
            <template #default="scope">
              <el-tag size="small" :type="getCommandStatusType(scope.row.status)">
                {{ getCommandStatusLabel(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="attempts" :label="t('opsManagement.retryCount')" width="100" />
          <el-table-column :label="t('opsManagement.lastUpdated')" min-width="180">
            <template #default="scope">
              {{
                formatTime(
                  scope.row.updatedAt ||
                    scope.row.completedAt ||
                    scope.row.issuedAt ||
                    scope.row.requestedAt,
                )
              }}
            </template>
          </el-table-column>
          <el-table-column
            prop="lastError"
            :label="t('opsManagement.failureReason')"
            min-width="220"
          />
        </el-table>
      </div>
    </el-dialog>

    <el-dialog
      v-model="showRollbackDialog"
      :title="
        t('opsManagement.rollbackSelectTitle', {
          name: rollbackSourceDeploy?.project?.name || '-',
        })
      "
      width="640px"
      append-to-body
    >
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        class="mb-3"
        :title="
          t('opsManagement.rollbackCurrentVersion', {
            version: rollbackSourceDeploy?.version || '-',
          })
        "
      />
      <el-select
        v-model="rollbackTargetDeploymentId"
        filterable
        class="w-full"
        :loading="rollbackDialogLoading"
        :placeholder="t('opsManagement.rollbackSelectPlaceholder')"
      >
        <el-option
          v-for="item in rollbackCandidateVersions"
          :key="item.id"
          :label="`v${item.version} · ${formatTime(item.createdAt)}`"
          :value="item.id"
        />
      </el-select>
      <div class="text-xs text-gray-500 mt-2">
        {{ t('opsManagement.rollbackHint') }}
      </div>

      <template #footer>
        <el-button @click="showRollbackDialog = false">{{ t('opsManagement.cancel') }}</el-button>
        <el-button
          type="primary"
          :loading="rollbackSubmitLoading"
          :disabled="!rollbackTargetDeploymentId"
          @click="confirmRollback"
        >
          {{ t('opsManagement.rollbackExecute') }}
        </el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="showFailureDrawer"
      :title="
        t('opsManagement.failureDrawerTitle', {
          name: failedDeployment?.project?.name || '-',
        })
      "
      size="520px"
    >
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="t('opsManagement.nodeName')">
          {{ failedDeployment?.node?.name || failedDeployment?.nodeName || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('opsManagement.runtimeVersion')">
          {{ failedDeployment?.version || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('opsManagement.status')">
          <el-tag :type="getDeployStatusType(failedDeployment?.status)">
            {{ getDeployDisplayLabel(failedDeployment) }}
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
        <div class="text-sm font-semibold text-gray-700 dark:text-gray-200 mb-2">
          {{ t('opsManagement.recentLogs') }}
        </div>
        <div class="bg-black text-green-400 rounded p-3 h-56 overflow-y-auto text-xs font-mono">
          <template v-if="failedDeployLogs.length > 0">
            <div v-for="(line, idx) in failedDeployLogs" :key="idx" class="mb-1">
              [{{ line.time || '-' }}] {{ line.message || line }}
            </div>
          </template>
          <div v-else class="text-gray-500 text-center mt-20">
            {{ t('opsManagement.noFailureLogs') }}
          </div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
// @ts-nocheck
import { ref, reactive, onMounted, computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Grid,
  List,
  Refresh,
  Search,
  MoreFilled,
  User,
  Clock,
  Tools,
  Check,
  Close,
  Bell,
  Warning,
  FolderOpened,
  Connection,
} from '@element-plus/icons-vue'
import request, { getApiErrorMessage } from '@/utils/request'
import dayjs from 'dayjs'
import { initSocket, getSocket } from '@/utils/socket'
import { Storage } from '@/utils/storage'
import { canApproveNodes } from '@/permissions'
import { useAuthStore } from '@/store'
import { RoleEnum, ENUM_LABELS } from '@/enums'
import WorkbenchPagination from '@/components/WorkbenchPagination.vue'
import {
  getNodeStatusType,
  getNodeStatusLabel,
  getDeployStatusType,
  getDeployLabel as getDeployLabelByStatus,
  isFailedDeploy,
  getDeployFailureReason,
  buildDeployStatusSummary,
  getProgressColor,
} from './utils/ops-status'

const emit = defineEmits(['open-tab'])
const authStore = useAuthStore()
const { t } = useI18n()
const currentUserRole = computed(
  () => authStore.userInfo?.role || Storage.getUserInfo()?.role || '',
)
const OPS_VIEW_MODE_STORAGE_KEY = 'ops_management_view_mode'

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
  projectName: '',
  status: '',
  keyword: '',
})
const nodePagination = reactive({
  page: 1,
  pageSize: 12,
  total: 0,
})
const searchDebounceTimer = ref(null)
const expandedNodeRowKeys = ref([])

// 待审核申请弹窗
const showPendingDialog = ref(false)
const pendingList = ref([])

// 日志弹窗
const showLogDialog = ref(false)
const currentDeployment = ref(null)
const logContent = ref([])
const commandTimeline = ref([])
const showFailureDrawer = ref(false)
const failedDeployment = ref(null)
const failedDeployLogs = ref([])
const showRollbackDialog = ref(false)
const rollbackDialogLoading = ref(false)
const rollbackSubmitLoading = ref(false)
const rollbackSourceDeploy = ref(null)
const rollbackTargetDeploymentId = ref('')
const rollbackCandidateVersions = ref([])

const deployStatusSummary = computed(() => {
  return buildDeployStatusSummary(filteredNodeList.value)
})

const projectOptions = computed(() => {
  const projectMap = new Map()
  nodeList.value.forEach((node) => {
    ;(node.deployments || []).forEach((deploy) => {
      if (!deploy?.projectId) return
      if (!projectMap.has(deploy.projectId)) {
        projectMap.set(deploy.projectId, {
          id: deploy.projectId,
          name: deploy.project?.name || deploy.projectId,
        })
      }
    })
  })
  return Array.from(projectMap.values())
})

const nodeStatusFilterOptions = computed(() => [
  { label: t('opsManagement.allStatus'), value: '' },
  { label: t('opsManagement.online'), value: 'online' },
  { label: t('opsManagement.offline'), value: 'offline' },
  { label: t('opsManagement.abnormal'), value: 'error' },
])

const selectedProjectFilterLabel = computed(() => {
  return nodeSearch.projectName || t('projectManagement.allProjects')
})

const selectedNodeStatusLabel = computed(() => {
  return (
    nodeStatusFilterOptions.value.find((option) => option.value === nodeSearch.status)?.label ||
    t('opsManagement.allStatus')
  )
})

const setNodeProjectFilter = (value) => {
  nodeSearch.projectName = value
  nodePagination.page = 1
}

const setNodeStatusFilter = (value) => {
  nodeSearch.status = value
}

const getVisibleDeployments = (node) => {
  const deployments = Array.isArray(node?.deployments) ? node.deployments : []
  if (!nodeSearch.projectName) {
    return deployments
  }
  return deployments.filter(
    (deploy) => (deploy.project?.name || deploy.projectId) === nodeSearch.projectName,
  )
}

const filteredNodeList = computed(() => {
  if (!nodeSearch.projectName) {
    return nodeList.value
  }
  return nodeList.value.filter((node) => getVisibleDeployments(node).length > 0)
})

const cardGridClass = computed(() => {
  if (filteredNodeList.value.length > 0 && filteredNodeList.value.length < 4) {
    return 'lg:grid-cols-2 xl:grid-cols-2'
  }
  return 'lg:grid-cols-3 xl:grid-cols-4'
})

const nodePaginationTotalPages = computed(() => {
  if (nodePagination.total <= 0) {
    return 0
  }
  return Math.ceil(nodePagination.total / nodePagination.pageSize)
})

const nodePaginationSummary = computed(() => {
  if (nodePagination.total <= 0) {
    return `${t('opsManagement.totalNodes')}: 0`
  }
  const start = (nodePagination.page - 1) * nodePagination.pageSize + 1
  const end = Math.min(nodePagination.page * nodePagination.pageSize, nodePagination.total)
  return `${start}-${end} / ${t('opsManagement.totalNodes')}: ${nodePagination.total}`
})

const nodePaginationPageIndicator = computed(() =>
  t('projectManagement.pageIndicator', {
    page: nodePaginationTotalPages.value > 0 ? nodePagination.page : 0,
    totalPages: nodePaginationTotalPages.value,
  }),
)

const handleNodePaginationChange = ({ page, limit }) => {
  nodePagination.page = page
  nodePagination.pageSize = limit
  fetchNodes()
}

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
    const nodeItems = Array.isArray(res?.data?.items) ? res.data.items : []
    nodeList.value = nodeItems
    nodePagination.total = res?.data?.total || 0
    approvedCount.value = res?.data?.total || 0
    // 详细列表视图默认全展开；其他视图保持原展开态过滤。
    const currentNodeIds = nodeItems.map((item) => item.id)
    if (activeView.value === 'list') {
      expandedNodeRowKeys.value = currentNodeIds
    } else {
      const currentNodeIdSet = new Set(currentNodeIds)
      expandedNodeRowKeys.value = expandedNodeRowKeys.value.filter((id) => currentNodeIdSet.has(id))
    }
  } catch (error) {
    console.error('获取节点失败:', error)
    nodeLoadError.value = getApiErrorMessage(error, t('opsManagement.fetchNodesFailed'))
    ElMessage.error(t('opsManagement.fetchNodesFailed'))
  } finally {
    nodeLoading.value = false
  }
}

const handleExpandChange = (row, expandedRows) => {
  expandedNodeRowKeys.value = expandedRows.map((item) => item.id)
}

// 获取待审核申请列表
const fetchPendingList = async () => {
  if (!canApproveNode.value) {
    pendingList.value = []
    pendingCount.value = 0
    return
  }
  try {
    const res = await request.get('/nodes', {
      params: {
        page: 1,
        pageSize: 100, // 获取最多100条待审核记录
        approvalStatus: 'pending',
      },
    })
    pendingList.value = res?.data?.items || []
    pendingCount.value = res?.data?.total || 0
  } catch (error) {
    console.error('获取待审核申请失败:', error)
  }
}

const handleApprove = async (node) => {
  try {
    await request.put(`/nodes/${node.id}/approve`)
    ElMessage.success(t('opsManagement.approvePassed'))
    showPendingDialog.value = false
    fetchNodes()
    fetchPendingList() // 刷新待审核列表
    updateCounts() // 异步刷新统计
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('opsManagement.approveFailed')))
  }
}

const handleReject = (node) => {
  ElMessageBox.confirm(t('opsManagement.rejectConfirm'), t('opsManagement.rejectConfirmTitle'), {
    confirmButtonText: t('opsManagement.rejectConfirmBtn'),
    cancelButtonText: t('opsManagement.cancel'),
    type: 'warning',
  }).then(async () => {
    try {
      await request.put(`/nodes/${node.id}/reject`)
      ElMessage.warning(t('opsManagement.rejectedTip'))
      showPendingDialog.value = false
      fetchNodes()
      fetchPendingList()
      updateCounts()
    } catch (error) {
      ElMessage.error(getApiErrorMessage(error, t('opsManagement.operationFailed')))
    }
  })
}

const updateCounts = async () => {
  if (!canApproveNode.value) {
    pendingCount.value = 0
    return
  }
  try {
    const [resApproved, resPending] = await Promise.all([
      request.get('/nodes', {
        params: { pageSize: 1, approvalStatus: 'approved' },
      }),
      request.get('/nodes', {
        params: { pageSize: 1, approvalStatus: 'pending' },
      }),
    ])
    approvedCount.value = resApproved.data.total
    pendingCount.value = resPending.data.total
  } catch (error) {
    console.error('更新节点统计失败:', error)
  }
}

// 指标获取辅助函数
const getMetricValue = (node, type) => {
  if (!node.metrics) return 0
  switch (type) {
    case 'cpu':
      return Math.round((node.metrics.cpu || 0) * 100)
    case 'memory':
      return Math.round((node.metrics.memory || 0) * 100)
    case 'disk':
      return Math.round((node.metrics.disk || 0) * 100)
    default:
      return 0
  }
}

const getDiskLabel = (node) => {
  if (!node.metrics?.disk_label) return t('opsManagement.systemDisk')
  return node.metrics.disk_label
}

const formatTime = (time) => (time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-')

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
const restartNode = (node) =>
  ElMessage.info(t('opsManagement.restartingAgent', { name: node.name }))
const viewNodeDetail = (node) => {
  ElMessageBox.alert(
    t('opsManagement.nodeDetailBody', {
      name: node?.name || '-',
      ip: node?.ipAddress || '-',
      port: node?.port || '-',
      status: getNodeStatusLabel(node?.status),
    }),
    t('opsManagement.nodeDetailTitle'),
    { confirmButtonText: t('opsManagement.close') },
  )
}
const deleteNode = async (node) => {
  try {
    await ElMessageBox.confirm(
      t('opsManagement.nodeDeleteConfirm', { name: node.name }),
      t('opsManagement.warning'),
      { type: 'warning' },
    )
    await request.delete(`/nodes/${node.id}`, { skipPermissionToast: true })
    ElMessage.success(t('opsManagement.nodeDeleted'))
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(getApiErrorMessage(error, t('opsManagement.nodeDeleteFailed')))
    }
  }
}

const openProjectManagement = () => {
  emit('open-tab', 'project-management')
}

/**
 * 打开待审核申请弹窗并刷新数据。
 */
const openPendingRequestsDialog = async () => {
  if (!canApproveNode.value) return
  await fetchPendingList()
  showPendingDialog.value = true
}

const toTimestamp = (value) => {
  if (!value) return 0
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? 0 : ts
}

const sortCommandTimeline = (commands = []) => {
  return [...commands].sort((a, b) => {
    const aTs = toTimestamp(a.requestedAt || a.issuedAt || a.createdAt || a.updatedAt)
    const bTs = toTimestamp(b.requestedAt || b.issuedAt || b.createdAt || b.updatedAt)
    return bTs - aTs
  })
}

const getInFlightCommandType = (deploy) => {
  const commands = Array.isArray(deploy?.commands) ? deploy.commands : []
  const activeCommands = commands.filter((item) =>
    ['pending', 'issued', 'acknowledged'].includes(item?.status),
  )
  if (activeCommands.length === 0) return ''
  const latest = sortCommandTimeline(activeCommands)[0]
  return latest?.type || ''
}

const getDeployDisplayLabel = (deploy) => {
  const status = deploy?.status
  const activeType = getInFlightCommandType(deploy)
  if (activeType === 'stop') return t('opsManagement.stopping')
  if (activeType === 'restart') return t('opsManagement.restarting')
  if (activeType === 'start') return t('opsManagement.starting')
  if (status === 'deploying') {
    return t('opsManagement.deploying')
  }
  return getDeployLabelByStatus(status)
}

// 新增：启动工程
const handleStartProject = async (deploy) => {
  try {
    await request.post(`/deployments/node-deployment/${deploy.id}/start`)
    ElMessage.success(t('opsManagement.startIssued'))
    fetchNodes()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('opsManagement.startFailed')))
  }
}

// 新增：停止工程
const handleStopProject = async (deploy) => {
  try {
    await ElMessageBox.confirm(
      t('opsManagement.stopConfirm', { name: deploy.project?.name }),
      t('opsManagement.stopConfirmTitle'),
      { type: 'warning' },
    )
    await request.post(`/deployments/node-deployment/${deploy.id}/stop`)
    ElMessage.success(t('opsManagement.stopIssued'))
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel')
      ElMessage.error(getApiErrorMessage(error, t('opsManagement.stopFailed')))
  }
}

// 新增：查看日志
const handleViewLog = (deploy) => {
  currentDeployment.value = deploy
  logContent.value = deploy.deployLog || []
  commandTimeline.value = sortCommandTimeline(Array.isArray(deploy.commands) ? deploy.commands : [])
  showLogDialog.value = true
}

const getCommandStatusType = (status) => {
  if (status === 'completed') return 'success'
  if (status === 'failed' || status === 'dead_letter') return 'danger'
  if (status === 'pending' || status === 'issued' || status === 'acknowledged') return 'warning'
  return 'info'
}

const getCommandStatusLabel = (status) => {
  const map = {
    pending: t('opsManagement.commandPending'),
    issued: t('opsManagement.commandIssued'),
    acknowledged: t('opsManagement.commandAck'),
    completed: t('opsManagement.commandCompleted'),
    failed: t('opsManagement.commandFailed'),
    dead_letter: t('opsManagement.commandDeadLetter'),
  }
  return map[status] || status
}

const openFailureDetail = (deploy) => {
  failedDeployment.value = deploy
  failedDeployLogs.value = Array.isArray(deploy?.deployLog) ? deploy.deployLog.slice(-50) : []
  showFailureDrawer.value = true
}

const isRuntimeActiveDeploy = (status) => {
  return ['pending', 'deploying', 'running'].includes(status)
}

const getDeployModeLabel = (mode) => {
  if (mode === 'DEV') return t('projectManagement.modeDisplayDev')
  if (mode === 'RELEASE') return t('projectManagement.modeDisplayRelease')
  return mode || '-'
}

// 新增：重启工程
const handleRestartProject = async (deploy) => {
  try {
    await ElMessageBox.confirm(
      t('opsManagement.restartConfirm', { name: deploy.project?.name }),
      t('opsManagement.restartConfirmTitle'),
      { type: 'warning' },
    )
    await request.post(`/deployments/node-deployment/${deploy.id}/restart`)
    ElMessage.success(t('opsManagement.restartIssued'))
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel')
      ElMessage.error(getApiErrorMessage(error, t('opsManagement.restartFailed')))
  }
}

/**
 * 打开回滚弹窗并加载可回滚版本。
 * @param {object} deploy - 当前部署对象
 * @returns {Promise<void>}
 */
const handleRollback = async (deploy) => {
  if (deploy.mode === 'DEV') {
    return ElMessage.warning(t('opsManagement.rollbackUnsupported'))
  }

  try {
    rollbackSourceDeploy.value = deploy
    rollbackTargetDeploymentId.value = ''
    rollbackCandidateVersions.value = []
    showRollbackDialog.value = true
    rollbackDialogLoading.value = true

    const projectId = deploy?.projectId || deploy?.project?.id
    if (!projectId) {
      throw new Error(t('opsManagement.rollbackProjectMissing'))
    }

    const res = await request.get(`/publish/${projectId}/versions`, {
      params: { page: 1, pageSize: 200 },
    })
    const payload = res?.data ?? res
    const data = payload?.data || payload || {}
    const allItems = Array.isArray(data.items) ? data.items : []
    const candidates = allItems
      .filter((item) => item?.mode === 'RELEASE')
      .filter((item) => item?.status === 'success')
      .filter((item) => item?.id !== deploy?.deploymentId)
      .sort((a, b) => new Date(b?.createdAt || 0).getTime() - new Date(a?.createdAt || 0).getTime())
    rollbackCandidateVersions.value = candidates

    if (candidates.length === 0) {
      ElMessage.warning(t('opsManagement.rollbackNoCandidates'))
    } else {
      rollbackTargetDeploymentId.value = candidates[0].id
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('opsManagement.rollbackLoadFailed')))
    showRollbackDialog.value = false
  } finally {
    rollbackDialogLoading.value = false
  }
}

/**
 * 执行回滚。
 * @returns {Promise<void>}
 */
const confirmRollback = async () => {
  const deploy = rollbackSourceDeploy.value
  if (!deploy || !rollbackTargetDeploymentId.value) {
    return ElMessage.warning(t('opsManagement.rollbackSelectRequired'))
  }
  if (!deploy.nodeId) {
    return ElMessage.warning(t('opsManagement.rollbackNodeMissing'))
  }

  try {
    await ElMessageBox.confirm(
      t('opsManagement.rollbackConfirmSelected'),
      t('opsManagement.rollbackConfirmTitle'),
      { type: 'warning' },
    )

    rollbackSubmitLoading.value = true
    await request.post(`/deployments/${rollbackTargetDeploymentId.value}/rollback`, {
      nodeId: deploy.nodeId,
    })
    ElMessage.success(t('opsManagement.rollbackCreated'))
    showRollbackDialog.value = false
    await fetchNodes()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(getApiErrorMessage(error, t('opsManagement.rollbackFailed')))
    }
  } finally {
    rollbackSubmitLoading.value = false
  }
}

// 新增：撤销部署
const handleUndeploy = async (deploy) => {
  try {
    const confirmMessageKey = isRuntimeActiveDeploy(deploy?.status)
      ? 'opsManagement.undeployConfirmRunning'
      : 'opsManagement.undeployConfirmStopped'

    await ElMessageBox.confirm(
      t(confirmMessageKey, { name: deploy.project?.name }),
      t('opsManagement.undeployConfirmTitle'),
      { type: 'warning' },
    )

    await request.delete(`/deployments/node-deployment/${deploy.id}`)
    ElMessage.success(t('opsManagement.undeploySuccess'))
    fetchNodes()
  } catch (error) {
    if (error !== 'cancel')
      ElMessage.error(getApiErrorMessage(error, t('opsManagement.undeployFailed')))
  }
}

// WebSocket 实时处理
const setupRealtimeUpdates = () => {
  const tenantId = Storage.getTenantId() || 'default'
  const socket = initSocket(tenantId)

  // 监听节点指标更新
  socket.on('ops:node:metrics', (data) => {
    const node = nodeList.value.find((n) => n.id === data.nodeId)
    if (node) {
      node.metrics = data.metrics
      node.lastHeartbeatAt = data.timestamp
    }
  })

  // 监听节点状态变化
  socket.on('ops:node:status', (data) => {
    const node = nodeList.value.find((n) => n.id === data.nodeId)
    if (node) {
      node.status = data.status
      node.lastHeartbeatAt = data.timestamp
    }
  })

  // 监听工程指标更新
  socket.on('ops:project:metrics', (data) => {
    const node = nodeList.value.find((n) => n.id === data.nodeId)
    if (node) {
      const deploy = node.deployments?.find((d) => d.projectId === data.projectId)
      if (deploy) {
        deploy.runtimeMetrics = data.metrics
      }
    }
  })

  socket.on('ops:deploy:status', (data) => {
    const node = nodeList.value.find((n) => n.id === data.nodeId)
    if (!node || !Array.isArray(node.deployments)) {
      return
    }
    if (data.removed) {
      node.deployments = node.deployments.filter((d) => d.id !== data.deploymentId)
      return
    }
    const deploy = node.deployments.find((d) => d.id === data.deploymentId)
    if (!deploy) {
      return
    }
    deploy.status = data.status || deploy.status
    if (data.startedAt) {
      deploy.startedAt = data.startedAt
    }
    if (data.stoppedAt) {
      deploy.stoppedAt = data.stoppedAt
    }
    if (data.errorMessage) {
      deploy.errorMessage = data.errorMessage
    }

    const ack = data.commandAck
    if (!ack || !Array.isArray(deploy.commands)) {
      return
    }
    const command = deploy.commands.find((item) => item.id === ack.id)
    if (!command) {
      return
    }
    command.status = ack.status || command.status
    command.acknowledgedAt = ack.acknowledgedAt || command.acknowledgedAt
    command.completedAt = ack.completedAt || command.completedAt
    command.lastError = ack.lastError || null
    command.updatedAt = ack.completedAt || ack.acknowledgedAt || command.updatedAt

    if (currentDeployment.value?.id === deploy.id) {
      commandTimeline.value = sortCommandTimeline(deploy.commands)
    }
  })

  // 监听新的待审核节点注册申请
  if (canApproveNode.value) {
    socket.on('ops:node:pending', async (data = {}) => {
      console.log('[OpsManagement][WS] 收到待审核事件:', data)
      await fetchPendingList()
    })
  }
}

/**
 * 初始化运维页面视图模式（持久化）。
 * @returns {void}
 */
const initActiveViewMode = () => {
  const storedMode = window.localStorage.getItem(OPS_VIEW_MODE_STORAGE_KEY)
  if (storedMode === 'dashboard' || storedMode === 'list') {
    activeView.value = storedMode
  }
}

const handleSetProjectFilter = async (event) => {
  const projectName = event?.detail?.projectName || ''
  const projectId = event?.detail?.projectId || ''
  if (projectName) {
    nodeSearch.projectName = projectName
  } else if (projectId) {
    // 兼容旧事件：若仅传 projectId，尝试映射为工程名称。
    let matchedName = ''
    nodeList.value.forEach((node) => {
      ;(node.deployments || []).forEach((deploy) => {
        if (!matchedName && deploy.projectId === projectId) {
          matchedName = deploy.project?.name || deploy.projectId
        }
      })
    })
    nodeSearch.projectName = matchedName
  } else {
    nodeSearch.projectName = ''
  }
  nodePagination.page = 1
  await fetchNodes()
}

// 挂载与卸载
onMounted(async () => {
  initActiveViewMode()
  await fetchNodes()
  if (canApproveNode.value) {
    fetchPendingList()
    updateCounts()
  }
  setupRealtimeUpdates()
  window.addEventListener('ops:open-pending-requests', openPendingRequestsDialog)
  window.addEventListener('ops:set-project-filter', handleSetProjectFilter)
})

onBeforeUnmount(() => {
  // 注意：此处不推荐直接 closeSocket，因为其他组件可能还在使用
  // 但可以取消事件监听
  const socket = getSocket()
  if (socket) {
    socket.off('ops:node:metrics')
    socket.off('ops:node:status')
    socket.off('ops:project:metrics')
    socket.off('ops:deploy:status')
    socket.off('ops:node:pending')
  }
  if (searchDebounceTimer.value) {
    window.clearTimeout(searchDebounceTimer.value)
    searchDebounceTimer.value = null
  }
  window.removeEventListener('ops:open-pending-requests', openPendingRequestsDialog)
  window.removeEventListener('ops:set-project-filter', handleSetProjectFilter)
})

watch(
  () => nodeSearch.status,
  () => {
    nodePagination.page = 1
    fetchNodes()
  },
)

watch(
  () => nodeSearch.keyword,
  () => {
    if (searchDebounceTimer.value) {
      window.clearTimeout(searchDebounceTimer.value)
    }
    searchDebounceTimer.value = window.setTimeout(() => {
      nodePagination.page = 1
      fetchNodes()
    }, 800)
  },
)

watch(
  () => activeView.value,
  (val) => {
    if (val === 'dashboard' || val === 'list') {
      window.localStorage.setItem(OPS_VIEW_MODE_STORAGE_KEY, val)
      if (val === 'list') {
        expandedNodeRowKeys.value = filteredNodeList.value.map((node) => node.id)
      }
    }
  },
)
</script>

<style scoped>
.ops-management {
  background: transparent;
}

.node-card {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.node-card:hover {
  transform: translateY(-4px);
}

.ops-pagination-bar {
  justify-content: flex-end;
}

:deep(.el-table) {
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-secondary);
}

:deep(.el-table th) {
  height: 48px;
  background-color: #f4f6f9 !important;
  color: var(--ck-text-secondary) !important;
  font-weight: 600;
  border-bottom: 1px solid var(--ck-border-light) !important;
}

:deep(.el-table td) {
  border-bottom: 1px solid var(--ck-border-light);
}

:deep(.el-table__row:hover > td.el-table__cell) {
  background-color: var(--ck-bg-hover);
}

:deep(.el-progress-circle) {
  margin: 0 auto;
}

:deep(.el-card__header) {
  padding: 12px 16px;
}
</style>
