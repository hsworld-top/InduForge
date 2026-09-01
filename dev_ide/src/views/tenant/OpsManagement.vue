<template>
  <section class="ops-management">
    <header class="ops-hero">
      <div class="hero-copy">
        <span class="eyebrow">工程运行与节点接入</span>
        <h2>节点与工程部署</h2>
        <p>
          用户只需要接入物理节点并选择工程版本。当前仅开放单节点部署方式；只有具备正式 Release
          的工程版本可部署，工程入口、数据运行和可选数据采集均由同一节点承载。
        </p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" @click="openDeployDialog">部署工程</el-button>
        <el-button :loading="refreshing" @click="loadDashboard">刷新</el-button>
      </div>
    </header>

    <div class="summary-grid">
      <article class="summary-card">
        <span>已接入物理节点</span>
        <strong>{{ nodePage.total }}</strong>
        <small>独立节点接入身份</small>
      </article>
      <article class="summary-card">
        <span>当前页可部署</span>
        <strong>{{ deployableNodes.length }}</strong>
        <small>在线且具备入口、运行能力</small>
      </article>
      <article class="summary-card">
        <span>正式工程部署</span>
        <strong>{{ deploymentPage.total }}</strong>
        <small>一个工程仅保留一个单节点部署</small>
      </article>
      <article class="summary-card">
        <span>当前页待确认</span>
        <strong>{{ pendingEnrollmentCount }}</strong>
        <small>确认主机指纹后才允许运行</small>
      </article>
    </div>

    <el-alert class="scope-alert" type="info" :closable="false" show-icon>
      <template #title>
        多节点分布式部署尚未开放。当前不会把“多节点”包装成高可用，也不会向用户暴露底层编排概念。
      </template>
    </el-alert>

    <el-tabs v-model="tab" class="ops-tabs">
      <el-tab-pane label="物理节点" name="nodes">
        <div class="table-toolbar">
          <el-input
            v-model="nodePage.keyword"
            clearable
            placeholder="搜索节点名称或主机名"
            @keyup.enter="searchNodes"
            @clear="searchNodes"
          />
          <el-button @click="searchNodes">查询</el-button>
        </div>

        <el-table :data="nodes" v-loading="loading.nodes" empty-text="暂无已接入节点">
          <el-table-column label="物理节点" min-width="210">
            <template #default="{ row }">
              <div class="primary-cell">
                <strong>{{ row.name }}</strong>
                <small>{{ row.hostname || '等待节点上报主机名' }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="系统" min-width="150">
            <template #default="{ row }">
              <span>{{ platformLabel(row.platform) }}</span>
              <small class="inline-secondary">{{ row.architecture || '架构待上报' }}</small>
            </template>
          </el-table-column>
          <el-table-column label="运行能力" min-width="250">
            <template #default="{ row }">
              <div class="tag-list">
                <el-tag
                  v-for="capability in row.capabilities"
                  :key="capability"
                  size="small"
                  effect="plain"
                >
                  {{ capabilityLabel[capability] }}
                </el-tag>
                <span v-if="!row.capabilities?.length" class="muted">未上报能力</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="连接状态" min-width="150">
            <template #default="{ row }">
              <div class="status-cell">
                <el-tag :type="lifecycleTagType(row.observedStatus)">
                  {{ lifecyclePresentation(row.observedStatus) }}
                </el-tag>
                <small>{{ relativeTime(row.lastHeartbeatAt) }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="部署条件" width="120">
            <template #default="{ row }">
              <el-tag :type="isDeployableNode(row) ? 'success' : 'info'" effect="plain">
                {{
                  isDeployableNode(row) ? '满足' : row.assignedDeploymentId ? '已部署' : '不满足'
                }}
              </el-tag>
              <small v-if="row.assignedProjectName" class="inline-secondary">
                {{ row.assignedProjectName }}
              </small>
            </template>
          </el-table-column>
          <el-table-column label="接入程序版本" min-width="140">
            <template #default="{ row }">
              <span>{{ row.agentVersion || '版本待上报' }}</span>
            </template>
          </el-table-column>
        </el-table>

        <WorkbenchPagination
          v-model:page="nodePage.page"
          v-model:limit="nodePage.limit"
          :total="nodePage.total"
          @change="loadNodes"
        />
      </el-tab-pane>

      <el-tab-pane label="工程部署" name="deployments">
        <div class="table-toolbar">
          <el-input
            v-model="deploymentPage.keyword"
            clearable
            placeholder="搜索工程名称"
            @keyup.enter="searchDeployments"
            @clear="searchDeployments"
          />
          <el-button @click="searchDeployments">查询</el-button>
          <span class="toolbar-spacer" />
          <el-button type="primary" @click="openDeployDialog">创建单节点部署</el-button>
        </div>

        <el-table :data="deployments" v-loading="loading.deployments" empty-text="暂无正式工程部署">
          <el-table-column label="工程 / 版本" min-width="200">
            <template #default="{ row }">
              <div class="primary-cell">
                <strong>{{ row.projectName }}</strong>
                <small>{{ row.version || row.applicationVersionId }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="部署节点" min-width="160">
            <template #default="{ row }">
              {{ row.nodeName || row.nodeId }}
            </template>
          </el-table-column>
          <el-table-column label="总体状态" width="130">
            <template #default="{ row }">
              <el-tag :type="deploymentStatePresentation(row).type">
                {{ deploymentStatePresentation(row).label }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="运行组成" min-width="310">
            <template #default="{ row }">
              <div class="service-tags">
                <el-tag
                  v-for="service in row.services"
                  :key="service.serviceType"
                  :type="lifecycleTagType(service.observedStatus)"
                  effect="plain"
                >
                  {{ serviceLabel[service.serviceType] }} ·
                  {{ lifecyclePresentation(service.observedStatus) }}
                </el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="工程入口" min-width="210">
            <template #default="{ row }">
              <el-button
                v-if="row.accessUrl"
                link
                type="primary"
                @click="openAccessUrl(row.accessUrl)"
              >
                打开工程
              </el-button>
              <span v-else class="muted">等待工程入口上报</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openDeploymentDetail(row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>

        <WorkbenchPagination
          v-model:page="deploymentPage.page"
          v-model:limit="deploymentPage.limit"
          :total="deploymentPage.total"
          @change="loadDeployments"
        />
      </el-tab-pane>

      <el-tab-pane label="节点接入" name="enrollments">
        <section class="package-panel">
          <div class="panel-heading">
            <div>
              <h3>安装独立节点接入程序</h3>
              <p>即使节点与中心安装在同一台机器，也必须使用独立服务、配置目录和运行身份。</p>
            </div>
          </div>
          <div v-loading="loading.packages" class="package-grid">
            <article v-for="item in packages" :key="item.id" class="package-card">
              <div>
                <strong>{{ platformLabel(item.platform) }} 节点接入程序</strong>
                <small
                  >{{ item.architecture || '通用架构' }} · {{ item.version || '版本待发布' }}</small
                >
              </div>
              <el-button
                size="small"
                :disabled="!item.available"
                :loading="downloadingPackageId === item.id"
                @click="downloadPackage(item)"
              >
                {{ item.available ? '下载安装包' : '安装包未生成' }}
              </el-button>
            </article>
            <el-empty v-if="!loading.packages && !packages.length" description="暂无节点安装包" />
          </div>
        </section>

        <div class="table-toolbar">
          <el-input
            v-model="enrollmentPage.keyword"
            clearable
            placeholder="搜索接入节点"
            @keyup.enter="searchEnrollments"
            @clear="searchEnrollments"
          />
          <el-button @click="searchEnrollments">查询</el-button>
          <span class="toolbar-spacer" />
          <el-button type="primary" @click="openEnrollmentDialog">生成一次性接入码</el-button>
        </div>

        <el-table :data="enrollments" v-loading="loading.enrollments" empty-text="暂无节点接入记录">
          <el-table-column label="计划接入节点" min-width="180">
            <template #default="{ row }">
              <div class="primary-cell">
                <strong>{{ row.displayName || '-' }}</strong>
                <small>{{ platformLabel(row.platform) }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="申请能力" min-width="230">
            <template #default="{ row }">
              <div class="tag-list">
                <el-tag
                  v-for="capability in row.capabilities"
                  :key="capability"
                  size="small"
                  effect="plain"
                >
                  {{ capabilityLabel[capability] }}
                </el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="节点上报" min-width="220">
            <template #default="{ row }">
              <div class="primary-cell">
                <span>{{ row.reportedHostName || '尚未领取接入码' }}</span>
                <small v-if="row.machineFingerprint"
                  >指纹 {{ shortFingerprint(row.machineFingerprint) }}</small
                >
                <small v-if="row.ipAddress">{{ row.ipAddress }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态 / 有效期" min-width="170">
            <template #default="{ row }">
              <div class="status-cell">
                <el-tag :type="lifecycleTagType(row.status)">
                  {{ lifecyclePresentation(row.status) }}
                </el-tag>
                <small v-if="row.expiresAt">{{ formatTime(row.expiresAt) }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <template v-if="row.status === 'claimed'">
                <el-button link type="primary" @click="approve(row)">确认</el-button>
                <el-button link type="danger" @click="reject(row)">拒绝</el-button>
              </template>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
        </el-table>

        <WorkbenchPagination
          v-model:page="enrollmentPage.page"
          v-model:limit="enrollmentPage.limit"
          :total="enrollmentPage.total"
          @change="loadEnrollments"
        />
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="deployDialog" title="创建正式单节点部署" width="620px">
      <el-alert
        type="info"
        :closable="false"
        title="工程入口与数据运行必须位于同一物理节点；数据采集可按工程需要启用。"
      />
      <el-form label-position="top" class="dialog-form">
        <el-form-item label="工程" required>
          <el-select
            v-model="deployForm.projectId"
            filterable
            remote
            placeholder="选择工程"
            :loading="projectLoading"
            :remote-method="searchProjects"
            @change="onProjectChange"
          >
            <el-option
              v-for="project in projects"
              :key="project.id"
              :label="project.name"
              :value="project.id"
              :disabled="deployedProjectIds.has(project.id)"
            >
              <span>{{ project.name }}</span>
              <small v-if="deployedProjectIds.has(project.id)" class="option-note">已有部署</small>
            </el-option>
          </el-select>
          <small v-if="selectedProjectAlreadyDeployed" class="form-help error">
            该工程已经有单节点部署，当前不允许重复部署。
          </small>
        </el-form-item>
        <el-form-item label="已发布版本" required>
          <el-select
            v-model="deployForm.applicationVersionId"
            filterable
            placeholder="选择构建成功的正式版本"
            :loading="versionLoading"
            :disabled="
              !deployForm.projectId || projectValidationPending || selectedProjectAlreadyDeployed
            "
          >
            <el-option
              v-for="version in versions"
              :key="version.id"
              :label="version.name ? `${version.version} · ${version.name}` : version.version"
              :value="version.id"
            />
          </el-select>
          <el-button
            v-if="versionHasMore"
            link
            type="primary"
            :loading="versionLoading"
            :disabled="projectValidationPending || selectedProjectAlreadyDeployed"
            @click="loadMoreVersions"
          >
            加载更多版本（已读取 {{ versionLoadedCount }} / {{ versionTotal }}）
          </el-button>
          <small
            v-if="
              deployForm.projectId &&
              !projectValidationPending &&
              !selectedProjectAlreadyDeployed &&
              !versionLoading &&
              !versionHasMore &&
              !versions.length
            "
            class="form-help error"
          >
            没有可部署的正式 Release（仅发布成功的源码版本不满足）。
          </small>
        </el-form-item>
        <el-form-item label="目标物理节点" required>
          <el-select
            v-model="deployForm.nodeId"
            filterable
            remote
            placeholder="选择满足部署条件的节点"
            :loading="deploymentNodeLoading"
            :remote-method="searchDeploymentNodes"
          >
            <el-option
              v-for="node in deploymentNodeOptions"
              :key="node.id"
              :value="node.id"
              :label="node.hostname ? `${node.name} · ${node.hostname}` : node.name"
            />
          </el-select>
          <small
            v-if="!deploymentNodeLoading && !deploymentNodeOptions.length"
            class="form-help error"
          >
            没有已批准、在线且同时具备工程入口和数据运行能力的节点。
          </small>
        </el-form-item>
        <el-form-item label="工程访问端口" required>
          <el-input-number v-model="deployForm.accessPort" :min="1024" :max="65532" />
          <small class="form-help">平台会同时检测该端口及后续三个本机运行端口是否冲突。</small>
        </el-form-item>
        <el-form-item label="数据采集">
          <el-checkbox
            v-model="deployForm.enableCollector"
            :disabled="!selectedDeployNode?.capabilities.includes('collector')"
          >
            在该节点启用数据采集服务
          </el-checkbox>
          <small
            v-if="selectedDeployNode && !selectedDeployNode.capabilities.includes('collector')"
            class="form-help"
          >
            当前节点未声明数据采集能力。
          </small>
          <small
            v-else-if="
              deployForm.enableCollector &&
              selectedDeployVersion &&
              !isDeployableReleaseVersion(selectedDeployVersion, true)
            "
            class="form-help error"
          >
            当前 Release 不包含数据采集工件，请选择包含采集工件的正式版本。
          </small>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="deployDialog = false">取消</el-button>
        <el-button
          type="primary"
          :loading="submittingDeployment"
          :disabled="!canCreateDeployment"
          @click="createDeployment"
        >
          创建部署
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="enrollmentDialog" title="接入物理节点" width="560px">
      <el-alert
        type="warning"
        :closable="false"
        title="接入码只能查看一次。节点领取后，仍需核对主机名和机器指纹并人工确认。"
      />
      <el-form label-position="top" class="dialog-form">
        <el-form-item label="节点显示名称" required>
          <el-input
            v-model="enrollForm.displayName"
            maxlength="80"
            placeholder="例如：一号产线边缘服务器"
          />
        </el-form-item>
        <el-form-item label="操作系统" required>
          <el-radio-group v-model="enrollForm.platform">
            <el-radio value="linux">Linux</el-radio>
            <el-radio value="windows">Windows</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="节点用途">
          <div class="node-purpose">
            <strong>{{ enrollForm.platform === 'linux' ? '完整运行节点' : '数据采集节点' }}</strong>
            <span v-if="enrollForm.platform === 'linux'">
              固定启用工程入口和数据运行能力，可直接承载正式单节点部署。
            </span>
            <span v-else>Windows 节点固定用于连接现场设备并运行数据采集。</span>
          </div>
        </el-form-item>
        <el-form-item v-if="enrollForm.platform === 'linux'" label="数据采集">
          <el-checkbox v-model="enrollForm.enableCollector">
            同时在该节点启用数据采集能力
          </el-checkbox>
          <small class="form-help">不启用时，该节点仍固定承担工程入口和数据运行。</small>
        </el-form-item>
        <el-form-item label="接入码有效期（分钟）">
          <el-input-number v-model="enrollForm.ttlMinutes" :min="1" :max="1440" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="enrollmentDialog = false">取消</el-button>
        <el-button type="primary" :loading="submittingEnrollment" @click="createEnrollment">
          生成接入码
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="codeDialog"
      title="一次性节点接入码"
      width="min(720px, calc(100vw - 32px))"
      :close-on-click-modal="false"
      @closed="clearEnrollmentSecret"
    >
      <p class="dialog-description">
        仅在目标机器安装节点接入程序时使用。系统不会再次返回明文接入码，请在关闭前完成复制。
      </p>
      <div class="secret-code">
        <code>{{ enrollmentCode }}</code>
        <el-button size="small" @click="copyEnrollmentCode">复制</el-button>
      </div>
      <p v-if="createdEnrollment?.expiresAt" class="expiry-text">
        失效时间：{{ formatTime(createdEnrollment.expiresAt) }}
      </p>
      <ol class="install-steps">
        <li>
          下载并解压 {{ platformLabel(createdEnrollment?.platform) }} 节点安装包。
          <el-button
            v-if="selectedEnrollmentPackage?.available"
            link
            type="primary"
            :loading="downloadingPackageId === selectedEnrollmentPackage.id"
            @click="downloadPackage(selectedEnrollmentPackage)"
          >
            立即下载
          </el-button>
          <span v-else class="muted">安装包尚未生成</span>
        </li>
        <li>
          在解压目录中打开{{ createdEnrollment?.platform === 'windows' ? ' PowerShell' : '终端' }}。
        </li>
        <li>复制并执行下方命令。命令只在当前弹窗中生成，不会自动执行。</li>
      </ol>
      <el-form label-position="top" class="install-server-form">
        <el-form-item
          label="节点可访问的中心地址"
          :error="enrollmentServerUrlValidation.error || undefined"
        >
          <el-input
            v-model="enrollmentServerUrl"
            clearable
            spellcheck="false"
            placeholder="https://center.example.com"
          />
          <small v-if="enrollmentServerUrlValidation.valid" class="form-help">
            安装命令将使用：<code>{{ enrollmentServerUrlValidation.normalized }}</code>
          </small>
        </el-form-item>
      </el-form>
      <el-alert
        v-if="enrollmentServerUrlValidation.valid && enrollmentServerUrlValidation.loopback"
        type="warning"
        :closable="false"
        show-icon
        title="仅同机节点可用，远程物理节点必须改为其可访问的 HTTPS 地址"
        class="loopback-warning"
      />
      <div class="install-command">
        <code>{{ installCommand || '请先填写有效的中心地址' }}</code>
        <el-button
          size="small"
          type="primary"
          plain
          :disabled="!installCommand"
          @click="copyInstallCommand"
        >
          复制安装命令
        </el-button>
      </div>
      <template #footer>
        <el-button type="primary" @click="codeDialog = false">我已保存</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailDrawer" title="工程部署详情" size="620px">
      <div v-loading="detailLoading" class="deployment-detail">
        <template v-if="selectedDeployment">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="工程">{{
              selectedDeployment.projectName
            }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{
              selectedDeployment.version || '-'
            }}</el-descriptions-item>
            <el-descriptions-item label="物理节点">
              {{ selectedDeployment.nodeName || selectedDeployment.nodeId }}
            </el-descriptions-item>
            <el-descriptions-item label="部署状态">
              <el-tag :type="deploymentStatePresentation(selectedDeployment).type">
                {{ deploymentStatePresentation(selectedDeployment).label }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="工程入口" :span="2">
              <el-button
                v-if="selectedDeployment.accessUrl"
                link
                type="primary"
                @click="openAccessUrl(selectedDeployment.accessUrl)"
              >
                {{ selectedDeployment.accessUrl }}
              </el-button>
              <span v-else class="muted">工程入口尚未上报可访问地址</span>
            </el-descriptions-item>
          </el-descriptions>

          <div class="deployment-actions">
            <span>工程部署操作</span>
            <el-button
              size="small"
              :loading="deploymentOperation === 'start'"
              :disabled="Boolean(deploymentOperation) || !canStartSelectedDeployment"
              @click="operateDeployment('start')"
            >
              启动工程
            </el-button>
            <el-button
              size="small"
              :loading="deploymentOperation === 'restart'"
              :disabled="Boolean(deploymentOperation) || !selectedDeploymentIsRunning"
              @click="operateDeployment('restart')"
            >
              重启工程
            </el-button>
            <el-button
              size="small"
              type="danger"
              plain
              :loading="deploymentOperation === 'stop'"
              :disabled="Boolean(deploymentOperation) || !selectedDeploymentIsRunning"
              @click="operateDeployment('stop')"
            >
              停止工程
            </el-button>
          </div>

          <h3 class="detail-heading">运行组成</h3>
          <article
            v-for="service in selectedDeployment.services"
            :key="service.serviceType"
            class="service-card"
          >
            <div class="service-card-main">
              <div>
                <strong>{{ serviceLabel[service.serviceType] }}</strong>
                <small>{{ serviceStatePresentation(service).detail }}</small>
              </div>
              <el-tag :type="serviceStatePresentation(service).type">
                {{ serviceStatePresentation(service).label }}
              </el-tag>
            </div>
            <div class="service-meta">
              <span>最后上报：{{ formatTime(service.observedAt) || '尚未上报' }}</span>
            </div>
          </article>

          <template v-if="selectedRun">
            <h3 class="detail-heading">最近一次操作</h3>
            <el-progress
              :percentage="selectedRun.progress || 0"
              :status="selectedRun.status === 'failed' ? 'exception' : undefined"
            />
            <div class="event-list">
              <p v-for="event in runEvents" :key="event.id || `${event.stage}-${event.createdAt}`">
                <span>{{ formatTime(event.createdAt) || '-' }}</span>
                <strong>{{ runEventPresentation(event.stage, event.message).stage }}</strong>
                {{ runEventPresentation(event.stage, event.message).message }}
              </p>
              <p v-if="!runEvents.length" class="muted">暂无操作事件</p>
            </div>
          </template>
        </template>
      </div>
    </el-drawer>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import WorkbenchPagination from '@/components/WorkbenchPagination.vue'
import { projectAPI } from '@/api/project.api'
import {
  opsAPI,
  type ApplicationVersion,
  type DeploymentRun,
  type DeploymentRunEvent,
  type NodeEnrollment,
  type NodePackage,
  type OpsDeploymentAction,
  type OpsNode,
  type ProjectDeployment,
} from '@/api/ops.api'
import { formatDateTime, getRelativeTime } from '@/utils/date'
import {
  capabilityLabel,
  deploymentStatePresentation,
  enrollmentCapabilities,
  enrollmentInstallCommand,
  isDeployableNode,
  isDeployableReleaseVersion,
  lifecyclePresentation,
  lifecycleTagType,
  platformLabel,
  runEventPresentation,
  serviceLabel,
  serviceStatePresentation,
  validateEnrollmentServerUrl,
} from './utils/ops-presentation'

interface ProjectOption {
  id: string
  name: string
  status?: string
}

interface PageState {
  page: number
  limit: number
  total: number
  keyword: string
}

const createPageState = (): PageState => ({ page: 1, limit: 20, total: 0, keyword: '' })
const tab = ref('nodes')
const loading = reactive({ nodes: false, deployments: false, enrollments: false, packages: false })
const nodePage = reactive(createPageState())
const deploymentPage = reactive(createPageState())
const enrollmentPage = reactive(createPageState())
const nodes = ref<OpsNode[]>([])
const deployments = ref<ProjectDeployment[]>([])
const enrollments = ref<NodeEnrollment[]>([])
const packages = ref<NodePackage[]>([])
const projects = ref<ProjectOption[]>([])
const versions = ref<ApplicationVersion[]>([])
const versionPage = ref(1)
const versionLoadedCount = ref(0)
const versionTotal = ref(0)
const deploymentNodeOptions = ref<OpsNode[]>([])

const deployDialog = ref(false)
const enrollmentDialog = ref(false)
const codeDialog = ref(false)
const detailDrawer = ref(false)
const projectLoading = ref(false)
const versionLoading = ref(false)
const projectValidationPending = ref(false)
const deploymentNodeLoading = ref(false)
const submittingDeployment = ref(false)
const submittingEnrollment = ref(false)
const downloadingPackageId = ref('')
const enrollmentCode = ref('')
const enrollmentServerUrl = ref('')
const createdEnrollment = ref<NodeEnrollment | null>(null)
const selectedDeployment = ref<ProjectDeployment | null>(null)
const selectedRun = ref<DeploymentRun | null>(null)
const runEvents = ref<DeploymentRunEvent[]>([])
const detailLoading = ref(false)
const deploymentOperation = ref<OpsDeploymentAction | ''>('')
const existingDeploymentProjectId = ref('')
let disposed = false
let projectValidationRequest = 0
let versionLoadRequest = 0
let lastExternalDeployRequestId = ''

const deployForm = reactive({
  projectId: '',
  nodeId: '',
  applicationVersionId: '',
  accessPort: 17800,
  enableCollector: false,
})
const enrollForm = reactive({
  displayName: '',
  platform: 'linux' as 'linux' | 'windows',
  enableCollector: false,
  ttlMinutes: 60,
})

const refreshing = computed(() => Object.values(loading).some(Boolean))
const deployableNodes = computed(() => nodes.value.filter((node) => isDeployableNode(node)))
const deployedProjectIds = computed(() => new Set(deployments.value.map((item) => item.projectId)))
const selectedProjectAlreadyDeployed = computed(
  () =>
    Boolean(deployForm.projectId) &&
    (deployedProjectIds.value.has(deployForm.projectId) ||
      existingDeploymentProjectId.value === deployForm.projectId),
)
const selectedDeployNode = computed(() =>
  deploymentNodeOptions.value.find((node) => node.id === deployForm.nodeId),
)
const selectedDeployVersion = computed(() =>
  versions.value.find((version) => version.id === deployForm.applicationVersionId),
)
const selectedEnrollmentPackage = computed(() =>
  packages.value.find((item) => item.platform === createdEnrollment.value?.platform),
)
const versionHasMore = computed(() => versionLoadedCount.value < versionTotal.value)
const pendingEnrollmentCount = computed(
  () => enrollments.value.filter((item) => item.status === 'claimed').length,
)
const defaultEnrollmentServerUrl = window.location.origin.replace(/\/$/, '')
const enrollmentServerUrlValidation = computed(() =>
  validateEnrollmentServerUrl(enrollmentServerUrl.value),
)
const installCommand = computed(() => {
  if (
    !createdEnrollment.value ||
    !enrollmentCode.value ||
    !enrollmentServerUrlValidation.value.valid
  ) {
    return ''
  }
  return enrollmentInstallCommand({
    platform: createdEnrollment.value.platform,
    serverUrl: enrollmentServerUrlValidation.value.normalized,
    enrollmentCode: enrollmentCode.value,
    enableCollector:
      createdEnrollment.value.platform === 'linux' &&
      createdEnrollment.value.capabilities.includes('collector'),
  })
})
const canCreateDeployment = computed(
  () =>
    Boolean(
      deployForm.projectId &&
      deployForm.applicationVersionId &&
      deployForm.nodeId &&
      deployForm.accessPort >= 1024 &&
      deployForm.accessPort <= 65532 &&
      selectedDeployNode.value &&
      isDeployableNode(selectedDeployNode.value) &&
      selectedDeployVersion.value &&
      isDeployableReleaseVersion(selectedDeployVersion.value, deployForm.enableCollector),
    ) &&
    !selectedProjectAlreadyDeployed.value &&
    (!deployForm.enableCollector || selectedDeployNode.value?.capabilities.includes('collector')),
)
const selectedDeploymentStatus = computed(() =>
  String(selectedDeployment.value?.observedStatus || '').toLowerCase(),
)
const selectedDeploymentIsRunning = computed(() => selectedDeploymentStatus.value === 'running')
const canStartSelectedDeployment = computed(() =>
  ['stopped', 'failed'].includes(selectedDeploymentStatus.value),
)

watch(
  () => enrollForm.platform,
  (platform) => {
    if (platform === 'windows') enrollForm.enableCollector = false
  },
)
watch(
  () => deployForm.nodeId,
  () => {
    if (!selectedDeployNode.value?.capabilities.includes('collector')) {
      deployForm.enableCollector = false
    }
  },
)

function apiErrorMessage(error: unknown, fallback: string) {
  if (error && typeof error === 'object' && 'message' in error) {
    const message = String((error as { message?: unknown }).message || '').trim()
    if (message) return message
  }
  return fallback
}

function collectionFromProjectResponse(response: unknown): ProjectOption[] {
  if (!response || typeof response !== 'object') return []
  const root = response as Record<string, unknown>
  const business =
    root.data && typeof root.data === 'object' ? (root.data as Record<string, unknown>) : root
  const list = business.list
  if (Array.isArray(list)) return list as ProjectOption[]
  if (list && typeof list === 'object') {
    const nested = list as Record<string, unknown>
    if (Array.isArray(nested.items)) return nested.items as ProjectOption[]
    if (Array.isArray(nested.projects)) return nested.projects as ProjectOption[]
  }
  return Array.isArray(business.items) ? (business.items as ProjectOption[]) : []
}

async function loadNodes() {
  loading.nodes = true
  try {
    const result = await opsAPI.listNodes({
      page: nodePage.page,
      pageSize: nodePage.limit,
      keyword: nodePage.keyword.trim() || undefined,
    })
    nodes.value = result.items
    nodePage.total = result.total
  } finally {
    loading.nodes = false
  }
}

async function loadDeployments() {
  loading.deployments = true
  try {
    const result = await opsAPI.listProjectDeployments({
      page: deploymentPage.page,
      pageSize: deploymentPage.limit,
      keyword: deploymentPage.keyword.trim() || undefined,
    })
    if (!disposed) {
      deployments.value = result.items
      deploymentPage.total = result.total
    }
  } finally {
    if (!disposed) loading.deployments = false
  }
}

async function loadEnrollments() {
  loading.enrollments = true
  try {
    const result = await opsAPI.listEnrollments({
      page: enrollmentPage.page,
      pageSize: enrollmentPage.limit,
      keyword: enrollmentPage.keyword.trim() || undefined,
    })
    enrollments.value = result.items
    enrollmentPage.total = result.total
  } finally {
    loading.enrollments = false
  }
}

async function loadPackages() {
  loading.packages = true
  try {
    const result = await opsAPI.listNodePackages()
    packages.value = result.items
  } finally {
    loading.packages = false
  }
}

async function loadDashboard() {
  const results = await Promise.allSettled([
    loadNodes(),
    loadDeployments(),
    loadEnrollments(),
    loadPackages(),
  ])
  if (results.some((item) => item.status === 'rejected')) {
    ElMessage.error('部分运维信息加载失败，请稍后重试')
  }
}

function searchNodes() {
  nodePage.page = 1
  void loadNodes().catch((error) => ElMessage.error(apiErrorMessage(error, '节点查询失败')))
}

function searchDeployments() {
  deploymentPage.page = 1
  void loadDeployments().catch((error) =>
    ElMessage.error(apiErrorMessage(error, '工程部署查询失败')),
  )
}

function searchEnrollments() {
  enrollmentPage.page = 1
  void loadEnrollments().catch((error) =>
    ElMessage.error(apiErrorMessage(error, '节点接入记录查询失败')),
  )
}

async function loadProjects(keyword = '') {
  projectLoading.value = true
  try {
    const response = await projectAPI.getProjects({
      page: 1,
      limit: 50,
      name: keyword.trim() || undefined,
    })
    projects.value = collectionFromProjectResponse(response).filter((item) => item.id && item.name)
  } finally {
    projectLoading.value = false
  }
}

function searchProjects(keyword: string) {
  void loadProjects(keyword).catch((error) =>
    ElMessage.error(apiErrorMessage(error, '工程列表加载失败')),
  )
}

async function loadDeploymentNodes(keyword = '') {
  deploymentNodeLoading.value = true
  try {
    const result = await opsAPI.listNodes({
      page: 1,
      pageSize: 50,
      keyword: keyword.trim() || undefined,
    })
    deploymentNodeOptions.value = result.items.filter((node) => isDeployableNode(node))
  } finally {
    deploymentNodeLoading.value = false
  }
}

function searchDeploymentNodes(keyword: string) {
  void loadDeploymentNodes(keyword).catch((error) =>
    ElMessage.error(apiErrorMessage(error, '可部署节点加载失败')),
  )
}

async function onProjectChange(projectId: string) {
  const requestId = ++projectValidationRequest
  versionLoadRequest += 1
  deployForm.applicationVersionId = ''
  deployForm.accessPort = 17800
  existingDeploymentProjectId.value = ''
  versions.value = []
  versionPage.value = 1
  versionLoadedCount.value = 0
  versionTotal.value = 0
  if (!projectId) {
    projectValidationPending.value = false
    versionLoading.value = false
    return
  }
  projectValidationPending.value = true
  versionLoading.value = true
  try {
    const [versionResult, deploymentResult] = await Promise.all([
      opsAPI.listProjectVersions(projectId, { page: 1, pageSize: 50 }),
      opsAPI.listProjectDeployments({ page: 1, pageSize: 1, projectId }),
    ])
    if (requestId !== projectValidationRequest || deployForm.projectId !== projectId) return
    if (deploymentResult.total > 0) {
      existingDeploymentProjectId.value = projectId
      return
    }
    versionLoadedCount.value = versionResult.items.length
    versionTotal.value = versionResult.total
    versions.value = versionResult.items.filter((version) => isDeployableReleaseVersion(version))
  } catch (error) {
    if (requestId === projectValidationRequest && deployForm.projectId === projectId) {
      ElMessage.error(apiErrorMessage(error, '部署条件加载失败'))
    }
  } finally {
    if (requestId === projectValidationRequest) {
      projectValidationPending.value = false
      versionLoading.value = false
    }
  }
}

async function loadMoreVersions() {
  const projectId = deployForm.projectId
  if (
    !projectId ||
    projectValidationPending.value ||
    selectedProjectAlreadyDeployed.value ||
    versionLoading.value ||
    !versionHasMore.value
  ) {
    return
  }
  const nextPage = versionPage.value + 1
  const requestId = ++versionLoadRequest
  versionLoading.value = true
  try {
    const result = await opsAPI.listProjectVersions(projectId, {
      page: nextPage,
      pageSize: 50,
    })
    if (requestId === versionLoadRequest && deployForm.projectId === projectId) {
      const matches = result.items.filter((version) => isDeployableReleaseVersion(version))
      const known = new Set(versions.value.map((version) => version.id))
      versions.value.push(...matches.filter((version) => !known.has(version.id)))
      versionPage.value = nextPage
      versionLoadedCount.value += result.items.length
      versionTotal.value = result.total
    }
  } catch (error) {
    if (requestId === versionLoadRequest && deployForm.projectId === projectId) {
      ElMessage.error(apiErrorMessage(error, '更多正式版本加载失败'))
    }
  } finally {
    if (requestId === versionLoadRequest) versionLoading.value = false
  }
}

async function openDeployDialog(initialProjectId = '') {
  projectValidationRequest += 1
  versionLoadRequest += 1
  deployForm.projectId = ''
  deployForm.nodeId = ''
  deployForm.applicationVersionId = ''
  deployForm.enableCollector = false
  existingDeploymentProjectId.value = ''
  versions.value = []
  versionPage.value = 1
  versionLoadedCount.value = 0
  versionTotal.value = 0
  projectValidationPending.value = false
  versionLoading.value = false
  deploymentNodeOptions.value = []
  deployDialog.value = true
  try {
    await Promise.all([loadProjects(), loadDeploymentNodes()])
    if (initialProjectId && projects.value.some((project) => project.id === initialProjectId)) {
      deployForm.projectId = initialProjectId
      await onProjectChange(initialProjectId)
    }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, '部署准备信息加载失败'))
  }
}

function handleOpenDeployEvent(event: Event) {
  const detail = (event as CustomEvent<{ projectId?: string; requestId?: string }>).detail || {}
  const projectId = String(detail.projectId || '').trim()
  const requestId = String(detail.requestId || '').trim()
  if (!projectId || !requestId || requestId === lastExternalDeployRequestId) return
  lastExternalDeployRequestId = requestId
  void openDeployDialog(projectId)
}

async function createDeployment() {
  if (!canCreateDeployment.value) {
    ElMessage.warning('请选择未部署的工程、正式版本和满足条件的物理节点')
    return
  }
  submittingDeployment.value = true
  try {
    const item = await opsAPI.createProjectDeployment({
      projectId: deployForm.projectId,
      nodeId: deployForm.nodeId,
      applicationVersionId: deployForm.applicationVersionId,
      accessPort: deployForm.accessPort,
      enableCollector: deployForm.enableCollector,
    })
    deployDialog.value = false
    tab.value = 'deployments'
    deploymentPage.page = 1
    await Promise.all([loadDeployments(), loadNodes()])
    ElMessage.success('单节点部署已创建，等待目标节点执行')
    await openDeploymentDetail(item)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, '创建工程部署失败'))
  } finally {
    submittingDeployment.value = false
  }
}

function openEnrollmentDialog() {
  enrollForm.displayName = ''
  enrollForm.platform = 'linux'
  enrollForm.enableCollector = false
  enrollForm.ttlMinutes = 60
  enrollmentDialog.value = true
}

async function createEnrollment() {
  if (!enrollForm.displayName.trim()) {
    ElMessage.warning('请填写节点名称')
    return
  }
  submittingEnrollment.value = true
  try {
    const item = await opsAPI.createEnrollment({
      platform: enrollForm.platform,
      capabilities: enrollmentCapabilities(enrollForm.platform, enrollForm.enableCollector),
      displayName: enrollForm.displayName.trim(),
      ttlMinutes: enrollForm.ttlMinutes,
    })
    const { enrollmentCode: oneTimeCode, ...enrollment } = item
    createdEnrollment.value = enrollment
    enrollmentCode.value = oneTimeCode || ''
    enrollmentServerUrl.value = defaultEnrollmentServerUrl
    enrollmentDialog.value = false
    codeDialog.value = true
    enrollmentPage.page = 1
    try {
      await loadEnrollments()
    } catch {
      ElMessage.warning('接入码已生成，但接入记录列表刷新失败；请先保存当前接入码')
    }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, '生成节点接入码失败'))
  } finally {
    submittingEnrollment.value = false
  }
}

async function approve(item: NodeEnrollment) {
  try {
    await ElMessageBox.confirm(
      `确认接入主机“${item.reportedHostName || item.displayName || item.id}”？机器指纹：${item.machineFingerprint || '未上报'}`,
      '核对物理节点身份',
      { confirmButtonText: '确认接入', cancelButtonText: '取消', type: 'warning' },
    )
    await opsAPI.approveEnrollment(item.id)
    await Promise.all([loadEnrollments(), loadNodes()])
    ElMessage.success('物理节点已确认接入')
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(apiErrorMessage(error, '确认节点接入失败'))
  }
}

async function reject(item: NodeEnrollment) {
  try {
    await ElMessageBox.confirm(
      `拒绝主机“${item.reportedHostName || item.displayName || item.id}”的接入请求？`,
      '拒绝节点接入',
      { confirmButtonText: '拒绝', cancelButtonText: '取消', type: 'error' },
    )
    await opsAPI.rejectEnrollment(item.id)
    await loadEnrollments()
    ElMessage.success('已拒绝节点接入')
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(apiErrorMessage(error, '拒绝节点接入失败'))
  }
}

async function copyEnrollmentCode() {
  if (!enrollmentCode.value) return
  try {
    await navigator.clipboard.writeText(enrollmentCode.value)
    ElMessage.success('接入码已复制')
  } catch {
    ElMessage.warning('浏览器未允许自动复制，请手动选择接入码')
  }
}

async function copyInstallCommand() {
  if (!installCommand.value) {
    ElMessage.warning(enrollmentServerUrlValidation.value.error || '请先填写有效的中心地址')
    return
  }
  try {
    await navigator.clipboard.writeText(installCommand.value)
    ElMessage.success('安装命令已复制')
  } catch {
    ElMessage.warning('浏览器未允许自动复制，请手动选择安装命令')
  }
}

function clearEnrollmentSecret() {
  enrollmentCode.value = ''
  enrollmentServerUrl.value = ''
  createdEnrollment.value = null
}

async function downloadPackage(item: NodePackage) {
  downloadingPackageId.value = item.id
  try {
    const response = (await opsAPI.downloadNodePackage(item.id)) as unknown as {
      data?: Blob | ArrayBuffer
    }
    const source = response?.data ?? response
    if (!(source instanceof Blob) && !(source instanceof ArrayBuffer)) {
      throw new Error('节点安装包响应格式无效')
    }
    const blob = source instanceof Blob ? source : new Blob([source])
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = item.fileName || item.name || `node-agent-${item.platform}`
    link.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, '节点安装包下载失败'))
  } finally {
    downloadingPackageId.value = ''
  }
}

async function openDeploymentDetail(item: ProjectDeployment) {
  detailDrawer.value = true
  detailLoading.value = true
  selectedRun.value = null
  runEvents.value = []
  try {
    const detail = await opsAPI.getProjectDeployment(item.id)
    if (disposed) return
    selectedDeployment.value = detail
    if (detail.latestRunId) await loadRun(detail.latestRunId)
  } catch (error) {
    if (!disposed) ElMessage.error(apiErrorMessage(error, '工程部署详情加载失败'))
  } finally {
    if (!disposed) detailLoading.value = false
  }
}

async function loadRun(runId: string) {
  const [run, events] = await Promise.all([
    opsAPI.getDeploymentRun(runId),
    opsAPI.listDeploymentRunEvents(runId),
  ])
  if (!disposed) {
    selectedRun.value = run
    runEvents.value = events.items
  }
  return run
}

const delay = (milliseconds: number) =>
  new Promise<void>((resolve) => window.setTimeout(resolve, milliseconds))

async function pollRun(runId: string, deploymentId: string) {
  for (let attempt = 0; attempt < 20 && !disposed; attempt += 1) {
    const run = await loadRun(runId)
    if (disposed || run.status !== 'pending') break
    await delay(1500)
  }
  if (disposed) return
  const detail = await opsAPI.getProjectDeployment(deploymentId)
  if (disposed) return
  selectedDeployment.value = detail
  await loadDeployments()
}

async function operateDeployment(action: OpsDeploymentAction) {
  const deployment = selectedDeployment.value
  if (!deployment || deploymentOperation.value) return
  try {
    if (action === 'stop' || action === 'restart') {
      await ElMessageBox.confirm(
        `${action === 'stop' ? '停止' : '重启'}工程“${deployment.projectName}”会影响其在物理节点上的运行，是否继续？`,
        '确认工程部署操作',
        { confirmButtonText: '继续', cancelButtonText: '取消', type: 'warning' },
      )
    }
    deploymentOperation.value = action
    const result = await opsAPI.operateProjectDeployment(deployment.id, action)
    if (disposed) return
    selectedDeployment.value = result.deployment
    selectedRun.value = result.run
    ElMessage.success('工程部署操作已下发到目标物理节点')
    if (result.run.id) await pollRun(result.run.id, deployment.id)
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    if (!disposed) ElMessage.error(apiErrorMessage(error, '工程部署操作失败'))
  } finally {
    if (!disposed) deploymentOperation.value = ''
  }
}

function openAccessUrl(value: string) {
  try {
    const url = new URL(value, window.location.origin)
    if (!['http:', 'https:'].includes(url.protocol)) throw new Error('invalid protocol')
    window.open(url.toString(), '_blank', 'noopener,noreferrer')
  } catch {
    ElMessage.error('工程入口地址无效，请检查节点上的入口配置')
  }
}

const formatTime = (value?: string) => (value ? formatDateTime(value) : '')
const relativeTime = (value?: string) =>
  value ? getRelativeTime(value) || formatTime(value) : '尚未心跳'
const shortFingerprint = (value: string) =>
  value.length > 20 ? `${value.slice(0, 10)}…${value.slice(-8)}` : value

onMounted(() => {
  disposed = false
  window.addEventListener('ops:open-deploy', handleOpenDeployEvent)
  void loadDashboard()
})
onBeforeUnmount(() => {
  disposed = true
  window.removeEventListener('ops:open-deploy', handleOpenDeployEvent)
})
</script>

<style scoped>
.ops-management {
  min-height: 100%;
  padding: 24px;
  background: var(--el-bg-color-page);
}

.ops-hero,
.table-toolbar,
.panel-heading,
.service-card-main,
.deployment-actions {
  display: flex;
  align-items: center;
}

.ops-hero {
  justify-content: space-between;
  gap: 24px;
  padding: 24px 28px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 16px;
  background: linear-gradient(135deg, var(--el-bg-color) 0%, var(--el-fill-color-light) 100%);
}

.hero-copy {
  max-width: 760px;
}

.eyebrow {
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.12em;
}

.ops-hero h2 {
  margin: 6px 0 8px;
  color: var(--el-text-color-primary);
  font-size: 26px;
}

.ops-hero p,
.panel-heading p,
.package-card small,
.primary-cell small,
.status-cell small,
.service-card small,
.dialog-description,
.expiry-text,
.form-help,
.muted {
  color: var(--el-text-color-secondary);
}

.ops-hero p,
.panel-heading p,
.dialog-description,
.expiry-text {
  margin: 0;
  line-height: 1.65;
}

.hero-actions {
  display: flex;
  flex-shrink: 0;
  gap: 8px;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin: 16px 0;
}

.summary-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 18px 20px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  background: var(--el-bg-color);
}

.summary-card > span,
.summary-card small {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.summary-card strong {
  color: var(--el-text-color-primary);
  font-size: 28px;
  line-height: 1;
}

.scope-alert {
  margin-bottom: 16px;
}

.ops-tabs {
  padding: 4px 20px 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  background: var(--el-bg-color);
}

.table-toolbar {
  gap: 8px;
  margin: 12px 0 16px;
}

.table-toolbar :deep(.el-input) {
  width: 280px;
}

.toolbar-spacer {
  flex: 1;
}

.primary-cell,
.status-cell,
.package-card > div,
.service-card-main > div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.inline-secondary {
  margin-left: 6px;
  color: var(--el-text-color-secondary);
}

.tag-list,
.service-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.package-panel {
  margin: 12px 0 22px;
  padding: 18px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-fill-color-extra-light);
}

.panel-heading {
  justify-content: space-between;
  margin-bottom: 12px;
}

.panel-heading h3,
.detail-heading {
  margin: 0 0 6px;
  color: var(--el-text-color-primary);
  font-size: 16px;
}

.package-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  min-height: 72px;
}

.package-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  background: var(--el-bg-color);
}

.dialog-form {
  margin-top: 18px;
}

.dialog-form :deep(.el-select),
.dialog-form :deep(.el-input-number) {
  width: 100%;
}

.form-help {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.5;
}

.form-help.error {
  color: var(--el-color-danger);
}

.node-purpose {
  display: flex;
  flex-direction: column;
  gap: 5px;
  width: 100%;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  background: var(--el-fill-color-extra-light);
}

.node-purpose span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.option-note {
  float: right;
  margin-left: 16px;
  color: var(--el-text-color-secondary);
}

.secret-code {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 16px 0 10px;
  padding: 14px;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  background: var(--el-fill-color-light);
}

.secret-code code {
  flex: 1;
  overflow-wrap: anywhere;
  color: var(--el-text-color-primary);
  font-size: 14px;
}

.install-steps {
  margin: 18px 0 12px;
  padding-left: 22px;
  color: var(--el-text-color-regular);
  line-height: 1.8;
}

.install-server-form {
  margin-top: 14px;
}

.install-server-form :deep(.el-form-item) {
  margin-bottom: 10px;
}

.install-server-form :deep(.el-form-item__content) {
  display: block;
}

.install-server-form code {
  color: var(--el-text-color-primary);
}

.loopback-warning {
  margin-bottom: 10px;
}

.install-command {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  background: var(--el-fill-color-light);
}

.install-command code {
  flex: 1;
  overflow-wrap: anywhere;
  color: var(--el-text-color-primary);
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
}

.deployment-detail {
  min-height: 240px;
}

.detail-heading {
  margin-top: 24px;
}

.deployment-actions {
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 18px;
}

.deployment-actions > span {
  margin-right: auto;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.service-card {
  margin-top: 10px;
  padding: 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
}

.service-card-main {
  justify-content: space-between;
  gap: 16px;
}

.service-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 18px;
  margin-top: 12px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.event-list {
  margin-top: 12px;
}

.event-list p {
  display: grid;
  grid-template-columns: 140px 80px 1fr;
  gap: 10px;
  margin: 0;
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-regular);
  font-size: 12px;
}

.event-list p span {
  color: var(--el-text-color-secondary);
}

@media (max-width: 1100px) {
  .summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .ops-management {
    padding: 12px;
  }

  .ops-hero,
  .table-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .hero-actions,
  .table-toolbar :deep(.el-input) {
    width: 100%;
  }

  .summary-grid,
  .package-grid {
    grid-template-columns: 1fr;
  }
}
</style>
