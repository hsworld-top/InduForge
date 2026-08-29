<template>
  <section class="ops-page" data-testid="ops-management-page">
    <header class="ops-page__header">
      <div>
        <p class="eyebrow">平台运维</p>
        <h1>运行管理</h1>
        <p class="subtitle">接入节点、查看运行资源，并管理工程的计算、报警、采集演示服务。</p>
      </div>
      <div class="header-actions">
        <el-button :loading="loading" @click="reloadAll"
          ><el-icon><Refresh /></el-icon>刷新状态</el-button
        >
        <el-button type="primary" @click="openEnrollment()"
          ><el-icon><Plus /></el-icon>接入节点</el-button
        >
      </div>
    </header>

    <nav class="ops-nav" aria-label="运行管理模块">
      <button
        v-for="item in navItems"
        :key="item.id"
        type="button"
        :class="{ active: activeTab === item.id }"
        @click="activeTab = item.id"
      >
        <el-icon><component :is="item.icon" /></el-icon><span>{{ item.label }}</span
        ><small>{{ item.count }}</small>
      </button>
    </nav>
    <el-alert v-if="loadError" type="warning" :title="loadError" :closable="false" show-icon
      ><template #default
        ><el-button link type="primary" @click="reloadAll">重新尝试</el-button></template
      ></el-alert
    >

    <section
      v-if="!loading && clusters.length === 0"
      class="onboarding"
      data-testid="ops-first-enrollment"
    >
      <el-icon class="onboarding-icon"><Connection /></el-icon>
      <div>
        <p class="eyebrow">首次接入</p>
        <h2>先创建运行集群并接入第一台节点</h2>
        <p>
          创建一次性接入任务，下载节点包后在目标主机安装。接入后，平台会持续显示健康、资源和服务状态。
        </p>
        <el-button type="primary" size="large" @click="openEnrollment()">开始首次接入</el-button>
      </div>
      <ol>
        <li><b>1</b>选择节点用途</li>
        <li><b>2</b>下载安装包</li>
        <li><b>3</b>输入接入码</li>
        <li><b>4</b>等待注册完成</li>
      </ol>
    </section>

    <section v-show="activeTab === 'overview'" class="content">
      <div class="stats">
        <article>
          <span>运行集群</span><b>{{ clusters.length }}</b
          ><small>{{ healthyClusterCount }} 个健康</small>
        </article>
        <article>
          <span>已接入节点</span><b>{{ nodes.length }}</b
          ><small>{{ healthyNodeCount }} 个正常上报</small>
        </article>
        <article>
          <span>工程部署</span><b>{{ deployments.length }}</b
          ><small>{{ runningDeploymentCount }} 个运行中</small>
        </article>
        <article class="attention">
          <span>需要关注</span><b>{{ attentionCount }}</b
          ><small>节点或工程服务异常</small>
        </article>
      </div>
      <div class="overview-grid">
        <article class="panel">
          <header>
            <div>
              <h2>运行集群</h2>
              <p>工程服务的运行目标</p>
            </div>
            <el-button link type="primary" @click="activeTab = 'clusters'">查看全部</el-button>
          </header>
          <div v-if="clusters.length" class="mini-list">
            <button
              v-for="cluster in clusters.slice(0, 4)"
              :key="cluster.id"
              type="button"
              @click="activeTab = 'clusters'"
            >
              <i :class="healthClass(cluster.health)" /><b>{{ cluster.name }}</b
              ><small
                >{{ healthPresentation(cluster.health).label }} ·
                {{ cluster.nodeCount || 0 }} 个节点</small
              >
            </button>
          </div>
          <el-empty v-else description="尚未创建运行集群" :image-size="72" />
        </article>
        <article class="panel">
          <header>
            <div>
              <h2>最近接入</h2>
              <p>等待安装、注册或确认的节点</p>
            </div>
            <el-button link type="primary" @click="openEnrollment()">新建任务</el-button>
          </header>
          <div v-if="enrollments.length" class="enrollment-list">
            <div v-for="task in enrollments.slice(0, 4)" :key="task.id">
              <span>{{ nodeRoleLabel[task.role] }}</span
              ><b>{{ enrollmentStatusPresentation(task.status) }}</b
              ><small>
                {{ task.reportedHostName || task.ipAddress || '等待主机注册' }} ·
                {{ formatTime(task.updatedAt || task.createdAt) }}
              </small>
              <template v-if="task.status === 'claimed'">
                <code>指纹：{{ task.machineFingerprint || '等待上报' }}</code>
                <div class="enrollment-actions">
                  <el-button size="small" type="primary" @click="approveEnrollment(task)"
                    >确认接入</el-button
                  >
                  <el-button size="small" @click="rejectEnrollment(task)">拒绝</el-button>
                </div>
              </template>
            </div>
          </div>
          <el-empty v-else description="没有进行中的接入任务" :image-size="72" />
        </article>
      </div>
    </section>

    <section v-show="activeTab === 'clusters'" class="content">
      <header class="section-heading">
        <div>
          <h2>运行集群</h2>
          <p>以工程运行目标的方式管理承载服务的资源。</p>
        </div>
        <el-button type="primary" @click="clusterDialog = true"
          ><el-icon><Plus /></el-icon>新建运行集群</el-button
        >
      </header>
      <div v-loading="loading" class="resource-grid">
        <article v-for="cluster in clusters" :key="cluster.id" class="resource-card">
          <header>
            <div>
              <i :class="healthClass(cluster.health)" />
              <h3>{{ cluster.name }}</h3>
            </div>
            <el-tag :type="healthPresentation(cluster.health).type" effect="plain">{{
              healthPresentation(cluster.health).label
            }}</el-tag>
          </header>
          <p>{{ cluster.description || '用于承载工程的计算、报警等服务。' }}</p>
          <dl>
            <div>
              <dt>控制器状态</dt>
              <dd>{{ lifecyclePresentation(cluster.controllerStatus) }}</dd>
            </div>
            <div>
              <dt>已接入节点</dt>
              <dd>{{ cluster.onlineNodeCount || 0 }} / {{ cluster.nodeCount || 0 }}</dd>
            </div>
            <div>
              <dt>运行版本</dt>
              <dd>{{ cluster.version || '初始化中' }}</dd>
            </div>
          </dl>
          <footer>
            <small>最近更新 {{ formatTime(cluster.updatedAt || cluster.createdAt) }}</small
            ><el-button link type="primary" @click="openEnrollment(cluster.id)">添加节点</el-button>
          </footer>
        </article>
        <button type="button" class="create-card" @click="clusterDialog = true">
          <el-icon><Plus /></el-icon><b>新建运行集群</b><span>为工程服务准备运行目标</span>
        </button>
      </div>
      <el-pagination
        v-if="clusterPager.total > clusterPager.pageSize"
        v-model:current-page="clusterPager.page"
        :page-size="clusterPager.pageSize"
        :total="clusterPager.total"
        layout="prev, pager, next"
        @current-change="reloadAll"
      />
    </section>

    <section v-show="activeTab === 'nodes'" class="content">
      <header class="section-heading">
        <div>
          <h2>节点</h2>
          <p>节点主动上报资源、Agent 与服务健康状态。</p>
        </div>
        <el-button type="primary" @click="openEnrollment()"
          ><el-icon><Plus /></el-icon>接入节点</el-button
        >
      </header>
      <div class="table-wrap" v-loading="loading">
        <el-table :data="nodes" :empty-text="'暂无节点，可通过接入任务添加'"
          ><el-table-column label="节点" min-width="210"
            ><template #default="{ row }"
              ><div class="node-name">
                <i :class="healthClass(row.health)" />
                <div>
                  <b>{{ row.name }}</b
                  ><small
                    >{{ nodeRoleLabel[row.role] }} · {{ row.ipAddress || row.os || '-' }}</small
                  >
                </div>
              </div></template
            ></el-table-column
          ><el-table-column label="健康" width="100"
            ><template #default="{ row }"
              ><el-tag size="small" :type="healthPresentation(row.health).type">{{
                healthPresentation(row.health).label
              }}</el-tag></template
            ></el-table-column
          ><el-table-column label="最后心跳" min-width="145"
            ><template #default="{ row }">{{
              relativeTime(row.lastHeartbeatAt)
            }}</template></el-table-column
          ><el-table-column label="资源使用" min-width="205"
            ><template #default="{ row }"
              ><span class="metrics"
                >CPU {{ metric(row.metrics?.cpuPercent) }} / 内存
                {{ metric(row.metrics?.memoryPercent) }} / 磁盘
                {{ metric(row.metrics?.diskPercent) }}</span
              ></template
            ></el-table-column
          ><el-table-column label="Agent 状态" min-width="190"
            ><template #default="{ row }"
              ><div class="service-tags">
                <el-tag size="small" effect="plain"
                  >Agent {{ row.agentVersion || '等待注册' }}</el-tag
                ><el-tag size="small" effect="plain" :type="healthPresentation(row.health).type">{{
                  row.health === 'healthy' ? '正常上报' : '等待状态'
                }}</el-tag>
              </div></template
            ></el-table-column
          ></el-table
        >
      </div>
      <el-pagination
        v-if="nodePager.total > nodePager.pageSize"
        v-model:current-page="nodePager.page"
        :page-size="nodePager.pageSize"
        :total="nodePager.total"
        layout="prev, pager, next"
        @current-change="reloadAll"
      />
    </section>

    <section v-show="activeTab === 'deployments'" class="content">
      <header class="section-heading">
        <div>
          <h2>工程部署</h2>
          <p>工程版本在目标运行集群中启动，并由平台跟踪发布进度。</p>
        </div>
        <el-button
          type="primary"
          :disabled="deployableRuntimeClusters.length === 0"
          @click="deploymentDialog = true"
          ><el-icon><UploadFilled /></el-icon>发布工程</el-button
        >
      </header>
      <div class="table-wrap" v-loading="loading">
        <el-table :data="deployments" :empty-text="'暂无工程部署'"
          ><el-table-column label="工程" prop="projectName" min-width="175" /><el-table-column
            label="部署模式"
            min-width="105"
            ><template #default="{ row }"
              ><el-tag size="small" effect="plain">{{
                row.mode === 'production' ? '生产运行' : '开发验证'
              }}</el-tag></template
            ></el-table-column
          ><el-table-column label="运行集群" min-width="145"
            ><template #default="{ row }">{{
              row.runtimeClusterName || '未分配'
            }}</template></el-table-column
          ><el-table-column label="当前版本" min-width="120"
            ><template #default="{ row }">{{ row.version || '-' }}</template></el-table-column
          ><el-table-column label="状态" min-width="105"
            ><template #default="{ row }"
              ><el-tag size="small" :type="healthPresentation(row.health).type">{{
                lifecyclePresentation(row.observedStatus)
              }}</el-tag></template
            ></el-table-column
          ><el-table-column label="发布进度" min-width="150"
            ><template #default="{ row }"
              ><el-progress
                :percentage="progress(row.progress)"
                :stroke-width="8" /></template></el-table-column
          ><el-table-column label="更新时间" min-width="145"
            ><template #default="{ row }">{{
              formatTime(row.updatedAt)
            }}</template></el-table-column
          ><el-table-column label="操作" width="80" fixed="right"
            ><template #default="{ row }"
              ><el-button link type="primary" @click="openDeploymentDetail(row)"
                >查看</el-button
              ></template
            ></el-table-column
          ></el-table
        >
      </div>
      <el-pagination
        v-if="deploymentPager.total > deploymentPager.pageSize"
        v-model:current-page="deploymentPager.page"
        :page-size="deploymentPager.pageSize"
        :total="deploymentPager.total"
        layout="prev, pager, next"
        @current-change="reloadAll"
      />
    </section>

    <el-dialog v-model="clusterDialog" title="新建运行集群" width="min(520px, calc(100vw - 32px))"
      ><el-form label-position="top"
        ><el-form-item label="集群名称"
          ><el-input v-model="clusterForm.name" placeholder="例如：生产运行集群" /></el-form-item
        ><el-form-item label="集群编码"
          ><el-input
            v-model="clusterForm.code"
            placeholder="例如：production-runtime" /></el-form-item
        ><el-form-item label="运行形态"
          ><el-radio-group v-model="clusterForm.topology"
            ><el-radio-button value="single_node">单节点</el-radio-button
            ><el-radio-button value="high_availability" disabled
              >高可用（规划中）</el-radio-button
            ></el-radio-group
          ></el-form-item
        ><el-form-item label="说明"
          ><el-input
            v-model="clusterForm.description"
            type="textarea"
            placeholder="描述该集群承载的工程范围" /></el-form-item></el-form
      ><template #footer
        ><el-button @click="clusterDialog = false">取消</el-button
        ><el-button type="primary" :loading="submitting" @click="createCluster"
          >创建并继续接入节点</el-button
        ></template
      ></el-dialog
    >

    <el-dialog
      v-model="wizardDialog"
      title="接入节点"
      width="min(760px, calc(100vw - 32px))"
      :close-on-click-modal="false"
      ><el-steps :active="wizardStep" finish-status="success" simple
        ><el-step title="选择用途" /><el-step title="创建接入任务" /><el-step title="安装并确认"
      /></el-steps>
      <div v-if="wizardStep === 0" class="wizard">
        <h3>这台主机将承担什么工作？</h3>
        <p>根据用途选择节点包；接入后可在节点页面查看状态。</p>
        <div class="role-grid">
          <button
            v-for="role in roles"
            :key="role.value"
            type="button"
            :class="{ selected: enrollmentForm.role === role.value }"
            @click="enrollmentForm.role = role.value"
          >
            <el-icon><component :is="role.icon" /></el-icon><b>{{ role.label }}</b
            ><span>{{ role.description }}</span>
          </button>
        </div>
        <el-form label-position="top" class="wizard-form"
          ><el-form-item label="节点显示名称"
            ><el-input
              v-model="enrollmentForm.displayName"
              placeholder="例如：产线边缘节点 01" /></el-form-item
          ><el-form-item v-if="enrollmentForm.role === 'runtime_linux'" label="加入运行集群"
            ><el-select
              v-model="enrollmentForm.runtimeClusterId"
              placeholder="选择运行集群"
              class="w-full"
              ><el-option
                v-for="cluster in clusters"
                :key="cluster.id"
                :label="cluster.name"
                :value="cluster.id" /></el-select
            ><small>没有运行集群？先创建一个运行集群。</small></el-form-item
          ><el-form-item label="节点安装包"
            ><el-select
              v-model="enrollmentForm.packageId"
              placeholder="选择对应安装包"
              class="w-full"
              ><el-option
                v-for="item in availablePackages"
                :key="item.id"
                :label="`${item.name}${item.version ? ` · ${item.version}` : ''}${item.size ? ` · ${formatPackageSize(item.size)}` : ''}`"
                :value="item.id" /></el-select></el-form-item
          ><el-alert
            v-if="availablePackages.length === 0"
            type="warning"
            :closable="false"
            title="当前节点类型尚未构建可用安装包，请先联系平台管理员发布节点包。" />
          ><el-form-item label="接入码有效期"
            ><el-select v-model="enrollmentForm.ttlMinutes" class="w-full"
              ><el-option label="30 分钟" :value="30" /><el-option
                label="1 小时"
                :value="60" /><el-option label="4 小时" :value="240" /></el-select></el-form-item
        ></el-form>
        <footer>
          <el-button @click="wizardDialog = false">取消</el-button
          ><el-button
            v-if="enrollmentForm.role === 'runtime_linux' && !clusters.length"
            type="primary"
            @click="clusterDialog = true"
            >新建运行集群</el-button
          ><el-button v-else type="primary" :loading="submitting" @click="createEnrollment"
            >创建接入任务</el-button
          >
        </footer>
      </div>
      <div v-else class="wizard success">
        <el-result
          icon="success"
          title="接入任务已创建"
          sub-title="在目标主机下载安装包并输入接入码。页面会自动刷新注册状态。"
        />
        <div class="enrollment-result">
          <div>
            <span>节点包</span
            ><b>{{ activeEnrollment?.packageName || chosenPackage?.name || '节点安装包' }}</b>
          </div>
          <div>
            <span>一次性接入码</span
            ><code>{{ activeEnrollment?.enrollmentCode || '创建结果未返回接入码' }}</code
            ><small>仅在创建后本次页面展示，请立即复制保存。</small>
          </div>
          <div>
            <span>到期时间</span><b>{{ formatTime(activeEnrollment?.expiresAt) }}</b>
          </div>
          <div>
            <span>当前状态</span
            ><el-tag :type="enrollmentTagType(activeEnrollment?.status)">{{
              enrollmentStatusPresentation(activeEnrollment?.status)
            }}</el-tag>
          </div>
        </div>
        <div class="download-actions">
          <el-button
            type="primary"
            :disabled="!activeEnrollment?.packageId"
            @click="downloadPackage(activeEnrollment?.packageId)"
            ><el-icon><Download /></el-icon>下载安装包</el-button
          ><el-button :disabled="!activeEnrollment?.enrollmentCode" @click="copyCode"
            >复制接入码</el-button
          >
        </div>
        <p class="install-hint">
          下载并解压后，请按包内 README 使用中心地址和上方一次性接入码执行安装。
        </p>
        <footer>
          <el-button @click="wizardDialog = false">完成</el-button
          ><el-button type="primary" plain @click="refreshEnrollment">刷新注册状态</el-button>
        </footer>
      </div></el-dialog
    >

    <el-dialog v-model="deploymentDialog" title="发布工程" width="min(560px, calc(100vw - 32px))"
      ><el-form label-position="top"
        ><el-form-item label="工程"
          ><el-select
            v-model="deploymentForm.projectId"
            class="w-full"
            filterable
            remote
            :remote-method="loadProjectOptions"
            :loading="projectOptionsLoading"
            placeholder="搜索并选择可访问工程"
            @visible-change="loadProjectOptionsOnVisible"
            ><el-option
              v-for="project in projectOptions"
              :key="project.id"
              :label="project.name"
              :value="project.id" /></el-select></el-form-item
        ><el-form-item label="部署模式"
          ><el-radio-group v-model="deploymentForm.deploymentMode"
            ><el-radio-button value="development">开发验证</el-radio-button
            ><el-radio-button value="production">生产运行</el-radio-button></el-radio-group
          ></el-form-item
        ><el-form-item label="运行集群"
          ><el-select
            v-model="deploymentForm.runtimeClusterId"
            class="w-full"
            :disabled="deployableRuntimeClusters.length === 0"
            ><el-option
              v-for="cluster in deployableRuntimeClusters"
              :key="cluster.id"
              :label="cluster.name"
              :value="cluster.id"
          /></el-select>
          <p v-if="deployableRuntimeClusters.length === 0" class="form-field-hint is-warning">
            暂无可发布的运行集群。请使用已就绪的单节点集群。
          </p></el-form-item
        ><el-form-item label="采集节点（采集服务需要）"
          ><el-select
            v-model="deploymentForm.hostNodeId"
            clearable
            class="w-full"
            :disabled="collectorNodes.length === 0"
            ><el-option
              v-for="node in collectorNodes"
              :key="node.id"
              :label="node.name"
              :value="node.id"
          /></el-select>
          <p v-if="collectorNodes.length === 0" class="form-field-hint is-warning">
            暂无可用采集节点。请完成节点审批，并确认 Agent 已启用且正在上报心跳。
          </p></el-form-item
        ></el-form
      ><template #footer
        ><el-button @click="deploymentDialog = false">取消</el-button
        ><el-button
          type="primary"
          :loading="submitting"
          :disabled="deployableRuntimeClusters.length === 0"
          @click="createDeployment"
          >开始发布</el-button
        ></template
      ></el-dialog
    >

    <el-drawer
      v-model="deploymentDetailDialog"
      :title="selectedDeployment?.projectName || '工程部署详情'"
      size="min(620px, 100vw)"
      ><template v-if="selectedDeployment"
        ><div class="deployment-summary">
          <i :class="healthClass(selectedDeployment.health)" />
          <div>
            <b>{{ lifecyclePresentation(selectedDeployment.observedStatus) }}</b
            ><small
              >{{ selectedDeployment.runtimeClusterName || '未分配运行集群' }} ·
              {{ selectedDeployment.version || '等待版本' }}</small
            >
          </div>
        </div>
        <h3>运行服务</h3>
        <p class="drawer-hint">首版以计算、报警、采集演示服务表示工程运行态。</p>
        <div class="workload-list">
          <article v-for="workload in workloadRows" :key="workload.role">
            <div>
              <b>{{ workloadRoleLabel[workload.role] }}</b
              ><span>{{ workload.lastMessage || 'Demo 工作负载' }}</span>
            </div>
            <el-tag size="small" :type="lifecycleTagType(workload.observedStatus)">{{
              lifecyclePresentation(workload.observedStatus)
            }}</el-tag>
            <footer>
              <el-button
                size="small"
                :loading="workloadActionKey === `${workload.role}:start`"
                :disabled="workloadActionDisabled(workload, 'start')"
                @click="operateWorkload(workload.role, 'start')"
                ><el-icon><VideoPlay /></el-icon>启动</el-button
              ><el-button
                size="small"
                :loading="workloadActionKey === `${workload.role}:stop`"
                :disabled="workloadActionDisabled(workload, 'stop')"
                @click="operateWorkload(workload.role, 'stop')"
                ><el-icon><VideoPause /></el-icon>停止</el-button
              ><el-button
                size="small"
                :loading="workloadActionKey === `${workload.role}:restart`"
                :disabled="workloadActionDisabled(workload, 'restart')"
                @click="operateWorkload(workload.role, 'restart')"
                ><el-icon><RefreshRight /></el-icon>重启</el-button
              >
            </footer>
          </article>
        </div>
        <section class="timeline">
          <header class="timeline__header">
            <div>
              <h3>异步任务时间线</h3>
              <p v-if="deploymentRunPollingState === 'timed_out'" class="timeline__pending">
                任务仍在执行或状态暂不可确认，已暂停自动刷新。
              </p>
            </div>
            <div class="timeline__actions">
              <el-button link :loading="refreshingDeployment" @click="refreshDeploymentDetail"
                >刷新详情</el-button
              >
              <el-button
                v-if="activeDeploymentRunId"
                link
                type="primary"
                :loading="refreshingRun"
                @click="refreshDeploymentRun"
                >刷新任务</el-button
              >
            </div>
          </header>
          <el-timeline v-if="runEvents.length"
            ><el-timeline-item
              v-for="event in runEvents"
              :key="event.id || `${event.createdAt}-${event.stage}`"
              :timestamp="formatTime(event.updatedAt || event.createdAt)"
              :type="timelineType(event.status || event.stage)"
              ><b>{{ runEventStagePresentation(event.stage) }}</b>
              <p>{{ runEventMessagePresentation(event.message) }}</p></el-timeline-item
            ></el-timeline
          ><el-empty v-else description="尚未产生异步任务" :image-size="72" /></section></template
    ></el-drawer>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Connection,
  Cpu,
  DataAnalysis,
  Download,
  FolderOpened,
  Monitor,
  Plus,
  Refresh,
  RefreshRight,
  UploadFilled,
  VideoPause,
  VideoPlay,
} from '@element-plus/icons-vue'
import {
  opsAPI,
  type DeploymentRunEvent,
  type DeploymentWorkload,
  type HostNode,
  type NodeEnrollment,
  type OpsNodeRole,
  type OpsWorkloadAction,
  type OpsWorkloadRole,
  type ProjectDeployment,
  type RuntimeCluster,
} from '@/api/ops.api'
import { getApiErrorMessage } from '@/utils/request'
import { projectAPI } from '@/api/project.api'
import {
  enrollmentStatusPresentation,
  healthPresentation,
  isDeployableRuntimeCluster,
  isDeploymentPending,
  isDeploymentRunActive,
  isSchedulableCollectorNode,
  lifecyclePresentation,
  lifecycleTagType,
  nodeRoleLabel,
  runEventMessagePresentation,
  runEventStagePresentation,
  sortRunEvents,
  workloadRoleLabel,
} from './utils/ops-presentation'

type Tab = 'overview' | 'clusters' | 'nodes' | 'deployments'
const activeTab = ref<Tab>('overview')
const loading = ref(false)
const submitting = ref(false)
const workloadActionKey = ref('')
const loadError = ref('')
const clusters = ref<RuntimeCluster[]>([])
const nodes = ref<HostNode[]>([])
const enrollments = ref<NodeEnrollment[]>([])
const packages = ref<Awaited<ReturnType<typeof opsAPI.listNodePackages>>['items']>([])
const deployments = ref<ProjectDeployment[]>([])
const clusterDialog = ref(false)
const wizardDialog = ref(false)
const wizardStep = ref(0)
const activeEnrollment = ref<NodeEnrollment | null>(null)
const deploymentDialog = ref(false)
const deploymentDetailDialog = ref(false)
const selectedDeployment = ref<ProjectDeployment | null>(null)
const runEvents = ref<DeploymentRunEvent[]>([])
const activeDeploymentRunId = ref('')
const deploymentRunPollingState = ref<'idle' | 'polling' | 'timed_out'>('idle')
const refreshingDeployment = ref(false)
const refreshingRun = ref(false)
const projectOptions = ref<Array<{ id: string; name: string }>>([])
const projectOptionsLoading = ref(false)
const clusterPager = reactive({ page: 1, pageSize: 20, total: 0 })
const nodePager = reactive({ page: 1, pageSize: 20, total: 0 })
const deploymentPager = reactive({ page: 1, pageSize: 20, total: 0 })
let enrollmentTimer: ReturnType<typeof setInterval> | undefined
let deploymentRunTimer: ReturnType<typeof setInterval> | undefined
const clusterForm = reactive({
  name: '',
  code: '',
  topology: 'single_node' as 'single_node' | 'high_availability',
  description: '',
})
const enrollmentForm = reactive<{
  role: OpsNodeRole
  displayName: string
  ttlMinutes: number
  runtimeClusterId: string
  packageId: string
}>({ role: 'runtime_linux', displayName: '', ttlMinutes: 60, runtimeClusterId: '', packageId: '' })
const deploymentForm = reactive<{
  projectId: string
  deploymentMode: 'development' | 'production'
  runtimeClusterId: string
  hostNodeId: string
}>({ projectId: '', deploymentMode: 'development', runtimeClusterId: '', hostNodeId: '' })
const navItems = computed(() => [
  { id: 'overview' as const, label: '运维总览', count: clusters.value.length, icon: DataAnalysis },
  { id: 'clusters' as const, label: '运行集群', count: clusters.value.length, icon: Connection },
  { id: 'nodes' as const, label: '节点', count: nodes.value.length, icon: Monitor },
  {
    id: 'deployments' as const,
    label: '工程部署',
    count: deployments.value.length,
    icon: FolderOpened,
  },
])
const roles = [
  {
    value: 'runtime_linux' as const,
    label: 'Linux 运行节点',
    description: '承载工程的计算和报警服务',
    icon: Cpu,
  },
  {
    value: 'collector_linux' as const,
    label: 'Linux 采集节点',
    description: '运行 Linux 原生采集服务',
    icon: Connection,
  },
  {
    value: 'collector_windows' as const,
    label: 'Windows 采集节点',
    description: '运行 Windows 原生采集服务',
    icon: Monitor,
  },
]
const availablePackages = computed(() =>
  packages.value.filter((item) => item.role === enrollmentForm.role && item.available === true),
)
const chosenPackage = computed(() =>
  packages.value.find((item) => item.id === enrollmentForm.packageId),
)
const collectorNodes = computed(() =>
  nodes.value.filter((node) => isSchedulableCollectorNode(node)),
)
const deployableRuntimeClusters = computed(() =>
  clusters.value.filter((cluster) => isDeployableRuntimeCluster(cluster)),
)
const healthyClusterCount = computed(
  () => clusters.value.filter((item) => item.health === 'healthy').length,
)
const healthyNodeCount = computed(
  () => nodes.value.filter((item) => item.health === 'healthy').length,
)
const runningDeploymentCount = computed(
  () => deployments.value.filter((item) => item.observedStatus === 'running').length,
)
const attentionCount = computed(
  () =>
    [...nodes.value, ...deployments.value].filter(
      (item) => item.health === 'degraded' || item.health === 'unavailable',
    ).length,
)
const workloadRows = computed<DeploymentWorkload[]>(() => {
  const existing = selectedDeployment.value?.workloads || []
  return (['compute', 'alert', 'collector'] as OpsWorkloadRole[]).map(
    (role) =>
      existing.find((item) => item.role === role) || {
        role,
        observedStatus: 'stopped',
        health: 'unknown',
        version: 'demo',
      },
  )
})
const metric = (value?: number) =>
  typeof value === 'number' && Number.isFinite(value) ? `${Math.round(value)}%` : '--'
const formatPackageSize = (bytes?: number) => {
  if (!bytes || bytes < 1024) return bytes ? `${bytes} B` : ''
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
const progress = (value?: number) => Math.min(100, Math.max(0, value || 0))
const healthClass = (value?: string) => `health-dot ${value || 'unknown'}`
const formatTime = (value?: string) =>
  value
    ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'short' }).format(
        new Date(value),
      )
    : '-'
function relativeTime(value?: string) {
  if (!value) return '尚未上报'
  const minutes = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 60000))
  return minutes < 1 ? '刚刚' : minutes < 60 ? `${minutes} 分钟前` : formatTime(value)
}
const enrollmentTagType = (status?: string) =>
  status === 'approved'
    ? 'success'
    : ['failed', 'rejected', 'expired'].includes(status || '')
      ? 'danger'
      : 'warning'
const timelineType = (status?: string) =>
  status === 'failed'
    ? 'danger'
    : ['running', 'completed', 'observed'].includes(status || '')
      ? 'success'
      : 'primary'
async function reloadAll() {
  loading.value = true
  loadError.value = ''
  const results = await Promise.allSettled([
    opsAPI.listRuntimeClusters({ page: clusterPager.page, pageSize: clusterPager.pageSize }),
    opsAPI.listHostNodes({ page: nodePager.page, pageSize: nodePager.pageSize }),
    opsAPI.listEnrollments({ page: 1, pageSize: 20 }),
    opsAPI.listNodePackages({ pageSize: 100 }),
    opsAPI.listProjectDeployments({
      page: deploymentPager.page,
      pageSize: deploymentPager.pageSize,
    }),
  ])
  const [a, b, c, d, e] = results
  if (a.status === 'fulfilled') {
    clusters.value = a.value.items
    clusterPager.total = a.value.total
  }
  if (b.status === 'fulfilled') {
    nodes.value = b.value.items
    nodePager.total = b.value.total
  }
  if (c.status === 'fulfilled') enrollments.value = c.value.items
  if (d.status === 'fulfilled') packages.value = d.value.items
  if (e.status === 'fulfilled') {
    deployments.value = e.value.items
    deploymentPager.total = e.value.total
  }
  const failed = results.find((result) => result.status === 'rejected')
  if (failed?.status === 'rejected')
    loadError.value = getApiErrorMessage(failed.reason, '部分运行状态暂时无法加载')
  loading.value = false
}
function openEnrollment(clusterId = '') {
  wizardStep.value = 0
  activeEnrollment.value = null
  enrollmentForm.role = 'runtime_linux'
  enrollmentForm.displayName = ''
  enrollmentForm.runtimeClusterId = clusterId || clusters.value[0]?.id || ''
  enrollmentForm.packageId = packages.value.find((item) => item.role === 'runtime_linux')?.id || ''
  wizardDialog.value = true
}
async function createCluster() {
  if (!clusterForm.name.trim() || !clusterForm.code.trim())
    return ElMessage.warning('请填写运行集群名称和编码')
  submitting.value = true
  try {
    const cluster = await opsAPI.createRuntimeCluster({
      ...clusterForm,
      name: clusterForm.name.trim(),
      code: clusterForm.code.trim(),
    })
    clusters.value = [cluster, ...clusters.value]
    enrollmentForm.runtimeClusterId = cluster.id
    clusterDialog.value = false
    clusterForm.name = ''
    clusterForm.code = ''
    clusterForm.description = ''
    ElMessage.success('运行集群已创建，请继续创建接入任务')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '创建运行集群失败'))
  } finally {
    submitting.value = false
  }
}
async function createEnrollment() {
  if (!enrollmentForm.displayName.trim()) return ElMessage.warning('请填写节点显示名称')
  if (enrollmentForm.role === 'runtime_linux' && !enrollmentForm.runtimeClusterId)
    return ElMessage.warning('请选择运行集群')
  if (!enrollmentForm.packageId) return ElMessage.warning('请选择节点安装包')
  submitting.value = true
  try {
    const task = await opsAPI.createEnrollment({
      role: enrollmentForm.role,
      displayName: enrollmentForm.displayName.trim(),
      ttlMinutes: enrollmentForm.ttlMinutes,
      runtimeClusterId:
        enrollmentForm.role === 'runtime_linux' ? enrollmentForm.runtimeClusterId : null,
      packageId: enrollmentForm.packageId,
    })
    activeEnrollment.value = task
    enrollments.value = [task, ...enrollments.value.filter((item) => item.id !== task.id)]
    wizardStep.value = 1
    startEnrollmentPolling()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '创建接入任务失败'))
  } finally {
    submitting.value = false
  }
}
async function refreshEnrollment() {
  if (!activeEnrollment.value) return
  try {
    const task = await opsAPI.getEnrollment(activeEnrollment.value.id)
    const mergedTask = {
      ...activeEnrollment.value,
      ...task,
      enrollmentCode: activeEnrollment.value.enrollmentCode,
      packageId: activeEnrollment.value.packageId,
      packageName: activeEnrollment.value.packageName,
    }
    activeEnrollment.value = mergedTask
    enrollments.value = [mergedTask, ...enrollments.value.filter((item) => item.id !== task.id)]
    if (task.status === 'approved') await reloadAll()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '刷新接入状态失败'))
  }
}
async function approveEnrollment(task: NodeEnrollment) {
  const fingerprint = task.machineFingerprint || '未提供'
  try {
    await ElMessageBox.confirm(
      `请确认主机“${task.reportedHostName || task.displayName || task.id}”及机器指纹“${fingerprint}”可信。确认后节点将加入平台。`,
      '确认节点接入',
      { type: 'warning', confirmButtonText: '确认接入', cancelButtonText: '取消' },
    )
    const approved = await opsAPI.approveEnrollment(task.id)
    enrollments.value = enrollments.value.map((item) => (item.id === approved.id ? approved : item))
    ElMessage.success('节点已确认接入')
    await reloadAll()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(getApiErrorMessage(error, '确认节点接入失败'))
  }
}
async function rejectEnrollment(task: NodeEnrollment) {
  try {
    await ElMessageBox.confirm(
      '拒绝后该节点不能使用当前接入码继续注册。确定拒绝吗？',
      '拒绝节点接入',
      { type: 'warning', confirmButtonText: '拒绝', cancelButtonText: '取消' },
    )
    const rejected = await opsAPI.rejectEnrollment(task.id)
    enrollments.value = enrollments.value.map((item) => (item.id === rejected.id ? rejected : item))
    ElMessage.success('已拒绝节点接入')
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(getApiErrorMessage(error, '拒绝节点接入失败'))
  }
}
async function loadProjectOptions(keyword = '') {
  projectOptionsLoading.value = true
  try {
    const response = await projectAPI.getProjects({ page: 1, limit: 20, name: keyword })
    const payload = (response as { data?: unknown }).data ?? response
    const source = payload as
      | {
          projects?: Array<{ id?: string; name?: string }>
          items?: Array<{ id?: string; name?: string }>
          list?: Array<{ id?: string; name?: string }>
        }
      | Array<{ id?: string; name?: string }>
    const items = Array.isArray(source)
      ? source
      : source.projects || source.items || source.list || []
    projectOptions.value = items
      .filter((item) => item.id && item.name)
      .map((item) => ({ id: String(item.id), name: String(item.name) }))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载可访问工程失败'))
  } finally {
    projectOptionsLoading.value = false
  }
}
function loadProjectOptionsOnVisible(visible: boolean) {
  if (visible && projectOptions.value.length === 0) void loadProjectOptions()
}
function startEnrollmentPolling() {
  if (enrollmentTimer) clearInterval(enrollmentTimer)
  enrollmentTimer = setInterval(() => {
    if (wizardDialog.value && activeEnrollment.value) void refreshEnrollment()
  }, 10000)
}
function contentDispositionFileName(value?: string) {
  const encoded = value?.match(/filename\*=UTF-8''([^;]+)/i)?.[1]
  if (encoded) return decodeURIComponent(encoded)
  const plain = value?.match(/filename="?([^";]+)"?/i)?.[1]
  return plain || ''
}
async function downloadPackage(id?: string) {
  if (!id) return
  try {
    const response = (await opsAPI.downloadNodePackage(id)) as unknown as {
      data: Blob
      headers?: Record<string, string>
    }
    const url = URL.createObjectURL(response.data)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download =
      contentDispositionFileName(response.headers?.['content-disposition']) ||
      chosenPackage.value?.fileName ||
      chosenPackage.value?.name ||
      'induforge-node-package'
    anchor.click()
    URL.revokeObjectURL(url)
    ElMessage.success('安装包下载已开始')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '下载安装包失败'))
  }
}
async function copyCode() {
  if (!activeEnrollment.value?.enrollmentCode) return
  try {
    await navigator.clipboard.writeText(activeEnrollment.value.enrollmentCode)
    ElMessage.success('接入码已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制接入码')
  }
}
async function createDeployment() {
  if (!deploymentForm.projectId || !deploymentForm.runtimeClusterId)
    return ElMessage.warning('请选择工程并选择运行集群')
  if (
    !deployableRuntimeClusters.value.some(
      (cluster) => cluster.id === deploymentForm.runtimeClusterId,
    )
  )
    return ElMessage.warning('请选择已就绪的单节点运行集群')
  if (!deploymentForm.hostNodeId) return ElMessage.warning('请选择采集节点，以启动采集演示服务')
  submitting.value = true
  try {
    const deployment = await opsAPI.createProjectDeployment({
      projectId: deploymentForm.projectId.trim(),
      runtimeClusterId: deploymentForm.runtimeClusterId,
      deploymentMode: deploymentForm.deploymentMode,
      workloads: [
        { role: 'compute' },
        { role: 'alert' },
        { role: 'collector', hostNodeId: deploymentForm.hostNodeId },
      ],
    })
    deployments.value = [deployment, ...deployments.value]
    deploymentDialog.value = false
    deploymentForm.projectId = ''
    deploymentForm.hostNodeId = ''
    ElMessage.success('发布任务已创建')
    await openDeploymentDetail(deployment)
    if (deployment.runId) startDeploymentRunPolling(deployment.runId)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '创建工程部署失败'))
  } finally {
    submitting.value = false
  }
}
async function openDeploymentDetail(deployment: ProjectDeployment) {
  stopDeploymentRunPolling()
  activeDeploymentRunId.value = ''
  deploymentRunPollingState.value = 'idle'
  selectedDeployment.value = deployment
  deploymentDetailDialog.value = true
  runEvents.value = []
  try {
    selectedDeployment.value = await opsAPI.getProjectDeployment(deployment.id)
    if (selectedDeployment.value.runId) startDeploymentRunPolling(selectedDeployment.value.runId)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载部署详情失败'))
  }
}
async function operateWorkload(role: OpsWorkloadRole, action: OpsWorkloadAction) {
  if (
    !selectedDeployment.value ||
    workloadActionKey.value ||
    isDeploymentPending(selectedDeployment.value.observedStatus)
  )
    return
  workloadActionKey.value = `${role}:${action}`
  try {
    const run = await opsAPI.runWorkloadAction(selectedDeployment.value.id, role, action)
    runEvents.value = sortRunEvents([...(run?.events || []), ...runEvents.value])
    ElMessage.success(
      `${workloadRoleLabel[role]}${action === 'start' ? '启动' : action === 'stop' ? '停止' : '重启'}任务已提交`,
    )
    selectedDeployment.value = await opsAPI.getProjectDeployment(selectedDeployment.value.id)
    deployments.value = deployments.value.map((item) =>
      item.id === selectedDeployment.value?.id ? selectedDeployment.value! : item,
    )
    if (run?.id) startDeploymentRunPolling(run.id)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '服务操作提交失败'))
  } finally {
    workloadActionKey.value = ''
  }
}
function workloadActionDisabled(workload: DeploymentWorkload, action: OpsWorkloadAction) {
  if (
    workloadActionKey.value ||
    deploymentRunPollingState.value !== 'idle' ||
    isDeploymentPending(selectedDeployment.value?.observedStatus)
  )
    return true
  const status = String(workload.observedStatus || '').toLowerCase()
  if (['pending', 'starting', 'stopping'].includes(status)) return true
  if (action === 'start') return status === 'running'
  return status !== 'running'
}
async function refreshDeploymentAfterRun() {
  if (!selectedDeployment.value) return
  selectedDeployment.value = await opsAPI.getProjectDeployment(selectedDeployment.value.id)
  deployments.value = deployments.value.map((item) =>
    item.id === selectedDeployment.value?.id ? selectedDeployment.value! : item,
  )
}
async function refreshDeploymentDetail() {
  if (!selectedDeployment.value) return
  refreshingDeployment.value = true
  try {
    await refreshDeploymentAfterRun()
    ElMessage.success('部署详情已刷新')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '刷新部署详情失败'))
  } finally {
    refreshingDeployment.value = false
  }
}
async function syncDeploymentRun(runId: string) {
  const [run, eventResult] = await Promise.all([
    opsAPI.getDeploymentRun(runId),
    opsAPI.listDeploymentRunEvents(runId),
  ])
  runEvents.value = sortRunEvents(eventResult.items)
  await refreshDeploymentAfterRun()
  return isDeploymentRunActive(run.status, run.completedAt)
}
function stopDeploymentRunPolling() {
  if (deploymentRunTimer) clearInterval(deploymentRunTimer)
  deploymentRunTimer = undefined
}
function completeDeploymentRunPolling() {
  stopDeploymentRunPolling()
  activeDeploymentRunId.value = ''
  deploymentRunPollingState.value = 'idle'
}
function pauseDeploymentRunPolling() {
  stopDeploymentRunPolling()
  deploymentRunPollingState.value = 'timed_out'
}
async function refreshDeploymentRun() {
  const runId = activeDeploymentRunId.value
  if (!runId) return
  refreshingRun.value = true
  try {
    if (await syncDeploymentRun(runId)) {
      deploymentRunPollingState.value = 'timed_out'
      ElMessage.info('任务仍在执行，请稍后继续刷新')
    } else {
      completeDeploymentRunPolling()
      ElMessage.success('任务已完成，详情已同步')
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '刷新任务状态失败'))
  } finally {
    refreshingRun.value = false
  }
}
function startDeploymentRunPolling(runId: string) {
  stopDeploymentRunPolling()
  activeDeploymentRunId.value = runId
  deploymentRunPollingState.value = 'polling'
  let attempt = 0
  const poll = async () => {
    attempt += 1
    try {
      if (!(await syncDeploymentRun(runId))) return completeDeploymentRunPolling()
      if (attempt >= 30) return pauseDeploymentRunPolling()
    } catch {
      if (attempt >= 3) pauseDeploymentRunPolling()
    }
  }
  void poll()
  deploymentRunTimer = setInterval(() => void poll(), 2000)
}
watch(
  () => enrollmentForm.role,
  () => {
    enrollmentForm.packageId = availablePackages.value[0]?.id || ''
    if (enrollmentForm.role !== 'runtime_linux') enrollmentForm.runtimeClusterId = ''
  },
)
watch(wizardDialog, (visible) => {
  if (!visible && enrollmentTimer) {
    clearInterval(enrollmentTimer)
    enrollmentTimer = undefined
  }
})
watch(deploymentDetailDialog, (visible) => {
  if (!visible) completeDeploymentRunPolling()
})
onMounted(() => {
  void reloadAll()
})
onBeforeUnmount(() => {
  if (enrollmentTimer) clearInterval(enrollmentTimer)
  stopDeploymentRunPolling()
})
</script>

<style scoped>
.ops-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  gap: 18px;
  overflow: auto;
  padding: 24px;
  background: #f5f7fb;
  color: #263852;
}
.ops-page h1,
.ops-page h2,
.ops-page h3,
.ops-page p {
  margin: 0;
}
.ops-page h1 {
  font-size: 24px;
}
.ops-page h2 {
  font-size: 17px;
}
.ops-page h3 {
  font-size: 15px;
}
.eyebrow {
  color: #4772c8;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
}
.subtitle,
.section-heading p,
.panel header p,
.drawer-hint {
  margin-top: 5px !important;
  color: #77869b;
  font-size: 13px;
}
.form-field-hint {
  margin: 7px 0 0;
  color: #77869b;
  font-size: 12px;
  line-height: 1.5;
}
.form-field-hint.is-warning {
  color: #a26a15;
}
.ops-page__header,
.header-actions,
.section-heading,
.panel header,
.resource-card header,
.resource-card footer,
.wizard footer,
.download-actions,
.deployment-summary,
.workload-list article {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.header-actions {
  flex-wrap: wrap;
}
.ops-nav {
  display: flex;
  gap: 8px;
  overflow: auto;
  padding: 6px;
  border: 1px solid #e0e7f0;
  border-radius: 14px;
  background: #fff;
}
.ops-nav button {
  display: flex;
  min-width: 130px;
  flex: 1;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: #637288;
  cursor: pointer;
  font: inherit;
  white-space: nowrap;
}
.ops-nav button small {
  padding: 1px 7px;
  border-radius: 99px;
  background: #edf1f7;
  font-size: 11px;
}
.ops-nav button.active {
  background: #eaf1ff;
  color: #1d4ed8;
  font-weight: 700;
}
.onboarding {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 28px;
  border: 1px solid #cfe0ff;
  border-radius: 18px;
  background: linear-gradient(120deg, #eff6ff, #fff);
}
.onboarding-icon {
  display: grid;
  width: 64px;
  height: 64px;
  flex: none;
  place-items: center;
  border-radius: 18px;
  background: #dbeafe;
  color: #2563eb;
  font-size: 30px;
}
.onboarding > div {
  flex: 1;
}
.onboarding h2 {
  margin: 5px 0 8px;
}
.onboarding p:not(.eyebrow) {
  margin-bottom: 16px;
  color: #58677d;
  line-height: 1.7;
}
.onboarding ol {
  display: grid;
  min-width: 210px;
  gap: 12px;
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: 13px;
  color: #53637a;
}
.onboarding li {
  display: flex;
  align-items: center;
  gap: 9px;
}
.onboarding b {
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  border-radius: 50%;
  background: #dbeafe;
  color: #2563eb;
  font-size: 12px;
}
.content {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: 16px;
}
.stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}
.stats article,
.panel,
.resource-card,
.table-wrap {
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  background: #fff;
  box-shadow: 0 2px 8px rgba(31, 53, 84, 0.035);
}
.stats article {
  padding: 17px;
}
.stats span,
.stats small {
  display: block;
  color: #78869a;
  font-size: 12px;
}
.stats b {
  display: block;
  margin: 7px 0 4px;
  color: #263c63;
  font-size: 28px;
}
.stats .attention b {
  color: #c56e0b;
}
.overview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}
.panel {
  min-height: 205px;
  padding: 18px;
}
.mini-list,
.enrollment-list {
  display: grid;
  gap: 8px;
}
.mini-list button {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 10px;
  border: 0;
  border-radius: 10px;
  background: #f8fafc;
  color: inherit;
  cursor: pointer;
  text-align: left;
}
.mini-list small,
.enrollment-list small {
  color: #7d899a;
  font-size: 12px;
}
.enrollment-list > div {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 3px 12px;
  padding: 9px 2px;
  border-bottom: 1px solid #edf1f5;
  color: #758297;
  font-size: 12px;
}
.enrollment-list b {
  color: #2f60b5;
  font-size: 13px;
}
.enrollment-list small {
  grid-column: 1/-1;
}
.resource-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(270px, 1fr));
  gap: 14px;
}
.resource-card {
  display: flex;
  min-height: 220px;
  flex-direction: column;
  padding: 18px;
}
.resource-card header > div,
.node-name {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}
.resource-card p {
  flex: 1;
  margin: 14px 0;
  color: #718096;
  font-size: 13px;
  line-height: 1.6;
}
.resource-card dl {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin: 0 0 12px;
}
.resource-card dt {
  color: #93a0b1;
  font-size: 11px;
}
.resource-card dd {
  margin: 4px 0 0;
  color: #34445e;
  font-size: 12px;
  font-weight: 600;
}
.resource-card footer {
  padding-top: 12px;
  border-top: 1px solid #edf1f5;
}
.resource-card footer small {
  color: #91a0b4;
  font-size: 11px;
}
.create-card {
  display: flex;
  min-height: 220px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px dashed #b6c9e9;
  border-radius: 14px;
  background: #f9fbff;
  color: #3f6fbe;
  cursor: pointer;
  font: inherit;
}
.create-card .el-icon {
  font-size: 28px;
}
.create-card span {
  color: #8190a6;
  font-size: 12px;
}
.table-wrap {
  overflow: hidden;
}
.node-name b,
.node-name small {
  display: block;
}
.node-name small {
  margin-top: 3px;
  color: #8793a5;
  font-size: 12px;
}
.metrics,
.service-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  color: #5d6c80;
  font-size: 12px;
}
.health-dot {
  display: inline-block;
  width: 9px;
  height: 9px;
  flex: none;
  border-radius: 50%;
  background: #9aa6b8;
}
.health-dot.healthy {
  background: #27a56b;
  box-shadow: 0 0 0 3px #e1f7eb;
}
.health-dot.degraded {
  background: #e2a02d;
  box-shadow: 0 0 0 3px #fff3d7;
}
.health-dot.unavailable {
  background: #d65757;
  box-shadow: 0 0 0 3px #fde8e8;
}
.wizard {
  min-height: 380px;
  padding: 22px 4px 0;
}
.wizard > h3 {
  margin-bottom: 6px;
  font-size: 18px;
}
.wizard > p {
  margin-bottom: 20px;
  color: #738095;
  font-size: 13px;
}
.role-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.role-grid button {
  display: flex;
  min-height: 145px;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  padding: 18px;
  border: 1px solid #dfe6f0;
  border-radius: 13px;
  background: #fff;
  color: #4d5d73;
  cursor: pointer;
  font: inherit;
  text-align: left;
}
.role-grid .el-icon {
  color: #5275b9;
  font-size: 23px;
}
.role-grid b {
  color: #2f405b;
}
.role-grid span {
  font-size: 12px;
  line-height: 1.55;
}
.role-grid button.selected {
  border-color: #5a82d4;
  background: #f2f6ff;
  box-shadow: 0 0 0 3px #e5efff;
}
.wizard-form {
  margin-top: 20px;
}
.wizard-form small {
  display: block;
  margin-top: 7px;
  color: #8a97a9;
  font-size: 12px;
}
.wizard footer {
  margin-top: 24px;
}
.success {
  padding-top: 4px;
}
.enrollment-result {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid #e4e9f0;
  border-radius: 12px;
  background: #e4e9f0;
}
.enrollment-result > div {
  min-height: 72px;
  padding: 13px;
  background: #fff;
}
.enrollment-result span,
.enrollment-result b,
.enrollment-result code,
.enrollment-result small {
  display: block;
}
.enrollment-result span,
.enrollment-result small {
  color: #7d899a;
  font-size: 12px;
}
.enrollment-result b,
.enrollment-result code {
  margin: 6px 0;
  color: #2e405e;
  font-size: 13px;
}
.enrollment-result code {
  color: #1d4ed8;
  font-weight: 700;
}
.download-actions {
  justify-content: center;
  margin-top: 18px;
}
.install-hint {
  margin-top: 10px !important;
  color: #748196;
  font-size: 12px;
  text-align: center;
}
.deployment-summary {
  margin-bottom: 24px;
  padding: 16px;
  border-radius: 12px;
  background: #f5f8fc;
}
.deployment-summary b,
.deployment-summary small,
.workload-list b,
.workload-list span {
  display: block;
}
.deployment-summary small,
.workload-list span {
  margin-top: 4px;
  color: #748196;
  font-size: 12px;
}
.workload-list {
  display: grid;
  gap: 9px;
}
.workload-list article {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  padding: 13px;
  border: 1px solid #e5eaf1;
  border-radius: 10px;
}
.workload-list footer {
  grid-column: 1/-1;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 11px;
}
.timeline {
  margin-top: 28px;
}
.timeline__header,
.timeline__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.timeline__header {
  margin-bottom: 14px;
}
.timeline__header h3 {
  margin-bottom: 0;
}
.timeline__pending {
  color: #a26a15;
}
.timeline h3 {
  margin-bottom: 14px;
}
.timeline p {
  margin: 4px 0 0;
  color: #738094;
  font-size: 12px;
}
@media (max-width: 900px) {
  .ops-page {
    padding: 16px;
  }
  .onboarding {
    flex-wrap: wrap;
  }
  .onboarding ol {
    width: 100%;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .overview-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 640px) {
  .ops-page__header,
  .section-heading {
    align-items: flex-start;
    flex-direction: column;
  }
  .header-actions {
    width: 100%;
  }
  .header-actions .el-button {
    flex: 1;
  }
  .ops-nav button {
    min-width: 110px;
  }
  .onboarding {
    padding: 20px;
  }
  .onboarding ol,
  .stats,
  .role-grid,
  .enrollment-result {
    grid-template-columns: 1fr;
  }
  .resource-card dl {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .table-wrap {
    overflow: auto;
  }
  .table-wrap :deep(.el-table) {
    min-width: 830px;
  }
}
</style>
