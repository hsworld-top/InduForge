<template>
  <section class="ops-page" data-testid="ops-management-page">
    <header class="ops-page__header">
      <div class="page-heading">
        <div class="page-heading__title">
          <h1>运行管理</h1>
          <span
            class="system-summary"
            :class="{ attention: Boolean(loadError) }"
            aria-live="polite"
          >
            <i :class="healthClass(loadError ? 'degraded' : 'healthy')" />
            {{ loadError ? '部分数据加载失败' : '数据已更新' }}
          </span>
        </div>
        <p class="subtitle">查看节点状态、接入新节点并管理工程服务。</p>
      </div>
      <div class="header-actions">
        <span class="last-refresh">
          {{ lastRefreshedAt ? `更新于 ${relativeTime(lastRefreshedAt)}` : '正在加载…' }}
        </span>
        <el-button :loading="loading" @click="reloadAll"
          ><el-icon><Refresh /></el-icon>刷新</el-button
        >
        <el-button type="primary" @click="openEnrollment()"
          ><el-icon><Plus /></el-icon>接入节点</el-button
        >
      </div>
    </header>

    <nav class="ops-nav" role="tablist" aria-label="运行管理模块">
      <button
        v-for="item in navItems"
        :key="item.id"
        type="button"
        role="tab"
        :class="{ active: activeTab === item.id }"
        :aria-selected="activeTab === item.id"
        :aria-controls="`ops-panel-${item.id}`"
        @click="activeTab = item.id"
      >
        <el-icon><component :is="item.icon" /></el-icon><span>{{ item.label }}</span
        ><small v-if="item.count !== null">{{ item.count }}</small>
      </button>
    </nav>
    <el-alert v-if="loadError" type="warning" :title="loadError" :closable="false" show-icon
      ><template #default
        ><el-button link type="primary" @click="reloadAll">重新尝试</el-button></template
      ></el-alert
    >

    <section v-if="showOnboarding" class="onboarding" data-testid="ops-first-enrollment">
      <el-icon class="onboarding-icon"><Connection /></el-icon>
      <div>
        <h2>接入第一台服务器节点</h2>
        <p>在服务器上安装节点程序并完成接入后，即可查看资源和运行服务。</p>
        <el-button type="primary" size="large" @click="openEnrollment()">接入服务器</el-button>
      </div>
      <ol>
        <li><b>1</b>选择节点用途</li>
        <li><b>2</b>下载安装包</li>
        <li><b>3</b>输入接入码</li>
        <li><b>4</b>确认服务器身份</li>
      </ol>
    </section>

    <section
      v-show="activeTab === 'overview' && !showOnboarding"
      id="ops-panel-overview"
      v-loading="loading"
      class="content content--overview"
      role="tabpanel"
    >
      <div class="stats">
        <button
          v-for="item in overviewStats"
          :key="item.id"
          type="button"
          class="stat-card"
          :class="`is-${item.tone}`"
          :aria-label="`查看${item.label}`"
          @click="activeTab = item.target"
        >
          <span class="stat-card__icon"
            ><el-icon><component :is="item.icon" /></el-icon
          ></span>
          <span class="stat-card__content">
            <span>{{ item.label }}</span>
            <b>{{ item.value }}</b>
            <small>{{ item.detail }}</small>
          </span>
          <el-icon class="stat-card__arrow"><ArrowRight /></el-icon>
        </button>
      </div>
      <div class="overview-grid">
        <article class="panel">
          <header>
            <div>
              <h2>服务器节点</h2>
            </div>
            <el-button link type="primary" @click="activeTab = 'nodes'">查看全部</el-button>
          </header>
          <div v-if="nodes.length" class="mini-list">
            <button
              v-for="node in nodes.slice(0, 4)"
              :key="node.id"
              type="button"
              @click="openNodeDetail(node)"
            >
              <i :class="healthClass(node.health)" />
              <span class="mini-list__identity">
                <b>{{ node.name }}</b>
                <small>{{ nodeRoleLabel[node.role] }}</small>
              </span>
              <span class="mini-list__status">
                <el-tag
                  size="small"
                  :type="nodeHealthPresentation(node.health, node.observedStatus).type"
                  effect="light"
                >
                  {{ nodeHealthPresentation(node.health, node.observedStatus).label }}
                </el-tag>
                <small>{{ relativeTime(node.lastHeartbeatAt, '尚未上报') }}</small>
              </span>
            </button>
          </div>
          <el-empty v-else description="尚未接入服务器节点" :image-size="72" />
        </article>
        <article class="panel">
          <header>
            <div>
              <h2>节点接入</h2>
            </div>
            <el-button link type="primary" @click="openEnrollmentTasks">查看全部</el-button>
          </header>
          <div v-if="enrollments.length" class="enrollment-list">
            <div v-for="task in sortedEnrollments.slice(0, 4)" :key="task.id">
              <span class="enrollment-list__identity">
                <b>{{ task.displayName || nodeRoleLabel[task.role] }}</b>
                <small>{{ nodeRoleLabel[task.role] }}</small>
              </span>
              <el-tag size="small" :type="enrollmentTagType(task.status)" effect="light">
                {{ enrollmentStatusPresentation(task.status) }}
              </el-tag>
              <small class="enrollment-list__meta">
                {{ task.reportedHostName || task.ipAddress || '等待服务器接入' }} ·
                {{ relativeTime(task.updatedAt || task.createdAt) }}
              </small>
              <template v-if="task.status === 'claimed'">
                <code>{{
                  task.machineFingerprint
                    ? `设备标识：${task.machineFingerprint}`
                    : '未获取到设备标识，暂不能确认接入'
                }}</code>
                <div class="enrollment-actions">
                  <el-button
                    size="small"
                    type="primary"
                    :disabled="!task.machineFingerprint"
                    @click="approveEnrollment(task)"
                    >确认接入</el-button
                  >
                  <el-button size="small" @click="rejectEnrollment(task)">拒绝</el-button>
                </div>
              </template>
            </div>
          </div>
          <el-empty v-else description="暂无接入记录" :image-size="72" />
        </article>
      </div>
    </section>

    <section
      v-show="activeTab === 'clusters'"
      id="ops-panel-clusters"
      class="content"
      role="tabpanel"
    >
      <header class="section-heading">
        <div>
          <div class="section-heading__title">
            <h2>运行资源池</h2>
            <span>{{ clusterPager.total }}</span>
          </div>
          <p>决定工程服务运行在哪些服务器上；单机使用默认资源池即可。</p>
        </div>
        <el-button type="primary" @click="clusterDialog = true"
          ><el-icon><Plus /></el-icon>新建运行资源池</el-button
        >
      </header>
      <div v-loading="loading" class="resource-grid">
        <article v-for="cluster in clusters" :key="cluster.id" class="resource-card">
          <header>
            <div class="resource-card__title">
              <span class="resource-card__icon"
                ><el-icon><Connection /></el-icon
              ></span>
              <div>
                <h3>{{ cluster.name }}</h3>
              </div>
            </div>
            <div class="resource-card__tags">
              <el-tag size="small" :type="healthPresentation(cluster.health).type" effect="light">
                {{ healthPresentation(cluster.health).label }}
              </el-tag>
            </div>
          </header>
          <p v-if="cluster.description">{{ cluster.description }}</p>
          <div class="cluster-capacity">
            <span>在线节点</span>
            <b>{{ cluster.onlineNodeCount || 0 }} / {{ cluster.nodeCount || 0 }}</b>
            <el-progress
              :percentage="
                cluster.nodeCount
                  ? Math.round(((cluster.onlineNodeCount || 0) / cluster.nodeCount) * 100)
                  : 0
              "
              :show-text="false"
              :stroke-width="5"
            />
          </div>
          <footer>
            <small>更新于 {{ relativeTime(cluster.updatedAt || cluster.createdAt) }}</small
            ><el-button link type="primary" @click="openEnrollment(cluster.id)">添加节点</el-button>
          </footer>
        </article>
        <div v-if="!loading && clusters.length === 0" class="resource-empty">
          <el-empty description="尚未创建运行资源池" :image-size="72">
            <el-button type="primary" @click="clusterDialog = true">新建运行资源池</el-button>
          </el-empty>
        </div>
      </div>
      <WorkbenchPagination
        v-if="clusterPager.total > 0"
        v-model:page="clusterPager.page"
        v-model:limit="clusterPager.pageSize"
        :total="clusterPager.total"
        @change="reloadClusters"
      />
    </section>

    <section v-show="activeTab === 'nodes'" id="ops-panel-nodes" class="content" role="tabpanel">
      <header class="section-heading">
        <div>
          <div class="section-heading__title">
            <h2>服务器节点</h2>
            <span>{{ nodePager.total }}</span>
          </div>
          <p>查看服务器在线状态、资源使用和可运行的服务。</p>
        </div>
        <el-button type="primary" @click="openEnrollment()"
          ><el-icon><Plus /></el-icon>接入节点</el-button
        >
      </header>
      <div class="table-wrap" v-loading="loading">
        <el-table :data="nodes" height="100%" @row-dblclick="openNodeDetail"
          ><el-table-column label="节点" min-width="230"
            ><template #default="{ row }"
              ><div class="node-name">
                <span class="node-name__icon"
                  ><el-icon><Monitor /></el-icon
                ></span>
                <div>
                  <b>{{ row.name }}</b
                  ><small>{{ nodeRoleLabel[row.role] }} · {{ row.ipAddress || '暂无地址' }}</small>
                </div>
              </div></template
            ></el-table-column
          ><el-table-column label="状态" width="100"
            ><template #default="{ row }"
              ><el-tag
                size="small"
                :type="nodeHealthPresentation(row.health, row.observedStatus).type"
                >{{ nodeHealthPresentation(row.health, row.observedStatus).label }}</el-tag
              ></template
            ></el-table-column
          ><el-table-column label="运行位置" min-width="190"
            ><template #default="{ row }"
              ><div class="table-primary-cell">
                <b>{{ nodeLocationPresentation(row) }}</b>
                <small>{{ nodeServicePresentation(row) }}</small>
              </div></template
            ></el-table-column
          ><el-table-column label="最后上报" min-width="150"
            ><template #default="{ row }"
              ><div class="time-cell">
                <b>{{ relativeTime(row.lastHeartbeatAt, '尚未上报') }}</b>
                <small>{{ formatTime(row.lastHeartbeatAt) }}</small>
              </div></template
            ></el-table-column
          ><el-table-column label="资源使用" min-width="270"
            ><template #default="{ row }"
              ><div class="resource-metrics">
                <div :class="`is-${metricTone(row.metrics?.cpuPercent, row.health)}`">
                  <span>CPU</span><b>{{ metric(row.metrics?.cpuPercent, row.health) }}</b
                  ><i
                    ><em :style="{ width: `${metricValue(row.metrics?.cpuPercent, row.health)}%` }"
                  /></i>
                </div>
                <div :class="`is-${metricTone(row.metrics?.memoryPercent, row.health)}`">
                  <span>内存</span><b>{{ metric(row.metrics?.memoryPercent, row.health) }}</b
                  ><i
                    ><em
                      :style="{
                        width: `${metricValue(row.metrics?.memoryPercent, row.health)}%`,
                      }"
                  /></i>
                </div>
                <div :class="`is-${metricTone(row.metrics?.diskPercent, row.health)}`">
                  <span>磁盘</span><b>{{ metric(row.metrics?.diskPercent, row.health) }}</b
                  ><i
                    ><em
                      :style="{ width: `${metricValue(row.metrics?.diskPercent, row.health)}%` }"
                  /></i>
                </div></div></template></el-table-column
          ><el-table-column label="操作" width="110" fixed="right" align="center"
            ><template #default="{ row }"
              ><el-button link type="primary" @click="openNodeDetail(row)"
                >查看详情<el-icon><ArrowRight /></el-icon></el-button></template
          ></el-table-column>
          <template #empty>
            <el-empty description="尚未接入服务器节点" :image-size="68">
              <el-button type="primary" @click="openEnrollment()">接入节点</el-button>
            </el-empty>
          </template></el-table
        >
      </div>
      <WorkbenchPagination
        v-if="nodePager.total > 0"
        v-model:page="nodePager.page"
        v-model:limit="nodePager.pageSize"
        :total="nodePager.total"
        @change="reloadNodes"
      />
    </section>

    <section
      v-show="activeTab === 'deployments'"
      id="ops-panel-deployments"
      class="content"
      role="tabpanel"
    >
      <header class="section-heading">
        <div>
          <div class="section-heading__title">
            <h2>工程服务</h2>
            <span>{{ deploymentPager.total }}</span>
          </div>
          <p>查看各工程的计算、报警和采集服务，管理发布和启停。</p>
        </div>
        <div class="section-heading__actions">
          <small v-if="!canPublish">{{ publishBlockedReason }}</small>
          <el-tooltip :disabled="canPublish" :content="publishBlockedReason" placement="bottom">
            <span>
              <el-button type="primary" :disabled="!canPublish" @click="deploymentDialog = true">
                <el-icon><UploadFilled /></el-icon>发布工程
              </el-button>
            </span>
          </el-tooltip>
        </div>
      </header>
      <div class="table-wrap" v-loading="loading">
        <el-table :data="deployments" height="100%"
          ><el-table-column label="工程" min-width="220"
            ><template #default="{ row }"
              ><div class="deployment-name">
                <span class="deployment-name__icon"
                  ><el-icon><FolderOpened /></el-icon
                ></span>
                <div>
                  <b>{{ row.projectName }}</b>
                  <small>{{ row.mode === 'production' ? '生产模式' : '开发模式' }}</small>
                </div>
              </div></template
            ></el-table-column
          ><el-table-column label="运行位置" min-width="175"
            ><template #default="{ row }"
              ><div class="table-primary-cell">
                <b>{{ row.runtimeClusterName || '未分配' }}</b>
                <small>{{
                  deploymentHasCollectorNode(row) ? '采集节点已配置' : '采集节点未配置'
                }}</small>
              </div></template
            ></el-table-column
          ><el-table-column label="当前版本" min-width="120"
            ><template #default="{ row }">{{ row.version || '-' }}</template></el-table-column
          ><el-table-column label="状态" min-width="190"
            ><template #default="{ row }"
              ><div class="deployment-status">
                <el-tag size="small" :type="deploymentPresentation(row).type" effect="light">{{
                  deploymentPresentation(row).label
                }}</el-tag>
                <small>{{ deploymentPresentation(row).detail }}</small>
              </div></template
            ></el-table-column
          ><el-table-column label="发布进度" min-width="165"
            ><template #default="{ row }"
              ><div class="deployment-progress">
                <el-progress
                  :percentage="progress(row.progress)"
                  :show-text="false"
                  :stroke-width="6"
                />
                <span>{{ progress(row.progress) }}%</span>
              </div></template
            ></el-table-column
          ><el-table-column label="更新时间" min-width="155"
            ><template #default="{ row }"
              ><div class="time-cell">
                <b>{{ relativeTime(row.updatedAt) }}</b>
                <small>{{ formatTime(row.updatedAt) }}</small>
              </div></template
            ></el-table-column
          ><el-table-column label="操作" width="96" fixed="right" align="center"
            ><template #default="{ row }"
              ><el-button link type="primary" @click="openDeploymentDetail(row)"
                >详情<el-icon><ArrowRight /></el-icon></el-button></template
          ></el-table-column>
          <template #empty>
            <el-empty description="尚未发布工程" :image-size="68">
              <el-tooltip :disabled="canPublish" :content="publishBlockedReason" placement="top">
                <span>
                  <el-button
                    type="primary"
                    :disabled="!canPublish"
                    @click="deploymentDialog = true"
                  >
                    发布工程
                  </el-button>
                </span>
              </el-tooltip>
            </el-empty>
          </template></el-table
        >
      </div>
      <WorkbenchPagination
        v-if="deploymentPager.total > 0"
        v-model:page="deploymentPager.page"
        v-model:limit="deploymentPager.pageSize"
        :total="deploymentPager.total"
        @change="reloadDeployments"
      />
    </section>

    <section
      v-show="activeTab === 'enrollments'"
      id="ops-panel-enrollments"
      class="content"
      role="tabpanel"
    >
      <header class="section-heading">
        <div>
          <div class="section-heading__title">
            <h2>节点接入</h2>
            <span>{{ enrollmentPager.total }}</span>
          </div>
          <p>查看接入进度，并确认待接入的服务器身份。</p>
        </div>
        <el-button type="primary" @click="openEnrollment()">
          <el-icon><Plus /></el-icon>接入节点
        </el-button>
      </header>
      <div class="table-wrap" v-loading="loading">
        <el-table :data="sortedEnrollments" height="100%">
          <el-table-column label="节点" min-width="230">
            <template #default="{ row }">
              <div class="node-name">
                <span class="node-name__icon"
                  ><el-icon><Monitor /></el-icon
                ></span>
                <div>
                  <b>{{ row.displayName || '未命名节点' }}</b>
                  <small>{{ nodeRoleLabel[row.role] }}</small>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="140">
            <template #default="{ row }">
              <el-tag size="small" :type="enrollmentTagType(row.status)" effect="light">
                {{ enrollmentStatusPresentation(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="服务器信息" min-width="190">
            <template #default="{ row }">
              <div class="table-primary-cell">
                <b>{{ row.reportedHostName || '等待服务器接入' }}</b>
                <small>{{ row.ipAddress || '暂无地址信息' }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="有效期" min-width="155">
            <template #default="{ row }">
              <div class="time-cell">
                <b>{{ enrollmentCodeState(row).label }}</b>
                <small>{{ enrollmentCodeState(row).detail }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" min-width="155">
            <template #default="{ row }">
              <div class="time-cell">
                <b>{{ relativeTime(row.updatedAt || row.createdAt) }}</b>
                <small>{{ formatTime(row.updatedAt || row.createdAt) }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="168" fixed="right" align="center">
            <template #default="{ row }">
              <div v-if="row.status === 'claimed'" class="table-actions">
                <el-button
                  link
                  type="primary"
                  :disabled="!row.machineFingerprint"
                  @click="approveEnrollment(row)"
                  >确认接入</el-button
                >
                <el-button link type="danger" @click="rejectEnrollment(row)">拒绝</el-button>
              </div>
              <span v-else class="table-placeholder">—</span>
            </template>
          </el-table-column>
          <template #empty>
            <el-empty description="暂无接入记录" :image-size="68">
              <el-button type="primary" @click="openEnrollment()">接入节点</el-button>
            </el-empty>
          </template>
        </el-table>
      </div>
      <WorkbenchPagination
        v-if="enrollmentPager.total > 0"
        v-model:page="enrollmentPager.page"
        v-model:limit="enrollmentPager.pageSize"
        :total="enrollmentPager.total"
        @change="reloadEnrollments"
      />
    </section>

    <el-dialog
      v-model="clusterDialog"
      class="ops-dialog"
      title="新建运行资源池"
      width="min(560px, calc(100vw - 32px))"
      ><el-form :model="clusterForm" label-position="top" class="dialog-form dialog-form--grid"
        ><el-form-item label="资源池名称" required
          ><el-input v-model="clusterForm.name" placeholder="例如：默认运行资源池" /></el-form-item
        ><el-form-item label="用途说明" class="is-full"
          ><el-input
            v-model="clusterForm.description"
            type="textarea"
            :rows="3"
            maxlength="120"
            show-word-limit
            placeholder="例如：用于工程验证" /></el-form-item></el-form
      ><template #footer
        ><el-button @click="clusterDialog = false">取消</el-button
        ><el-button type="primary" :loading="submitting" @click="createCluster"
          >创建资源池</el-button
        ></template
      ></el-dialog
    >

    <el-dialog
      v-model="wizardDialog"
      class="ops-dialog enrollment-dialog"
      title="接入节点"
      width="min(760px, calc(100vw - 32px))"
      :close-on-click-modal="false"
      ><el-steps :active="wizardStep" finish-status="success" simple
        ><el-step title="配置节点" /><el-step title="安装与接入"
      /></el-steps>
      <div v-if="wizardStep === 0" class="wizard">
        <div class="wizard-heading">
          <h3>选择节点用途</h3>
          <p>选择这台服务器需要运行的服务。</p>
        </div>
        <div class="role-grid" role="radiogroup" aria-label="节点用途">
          <button
            v-for="role in roles"
            :key="role.value"
            type="button"
            role="radio"
            :class="{ selected: enrollmentForm.role === role.value }"
            :aria-checked="enrollmentForm.role === role.value"
            @click="enrollmentForm.role = role.value"
          >
            <el-icon><component :is="role.icon" /></el-icon>
            <span
              ><b>{{ role.label }}</b
              ><small>{{ role.description }}</small></span
            >
          </button>
        </div>
        <el-form :model="enrollmentForm" label-position="top" class="wizard-form"
          ><el-form-item label="节点名称" required
            ><el-input
              v-model="enrollmentForm.displayName"
              placeholder="例如：产线边缘节点 01" /></el-form-item
          ><el-form-item
            v-if="enrollmentForm.role === 'runtime_linux' && clusters.length"
            label="运行位置"
            :required="clusters.length > 0"
          >
            <el-select
              v-model="enrollmentForm.runtimeClusterId"
              placeholder="选择运行资源池"
              :disabled="clusters.length === 1"
              class="w-full"
              ><el-option
                v-for="cluster in clusters"
                :key="cluster.id"
                :label="cluster.name"
                :value="cluster.id" /></el-select></el-form-item
          ><el-form-item label="安装包" required
            ><el-select
              v-model="enrollmentForm.packageId"
              placeholder="选择对应安装包"
              class="w-full"
              ><el-option
                v-for="item in availablePackages"
                :key="item.id"
                :label="packagePresentation(item)"
                :value="item.id" /></el-select></el-form-item
          ><el-alert
            v-if="availablePackages.length === 0"
            type="warning"
            class="wizard-form__alert"
            :closable="false"
            title="暂无适用于该节点的安装包，请联系平台管理员。" />
          <el-form-item label="接入码有效期"
            ><el-select v-model="enrollmentForm.ttlMinutes" class="w-full"
              ><el-option label="30 分钟" :value="30" /><el-option
                label="1 小时"
                :value="60" /><el-option label="4 小时" :value="240" /></el-select></el-form-item
        ></el-form>
        <footer>
          <el-button @click="wizardDialog = false">取消</el-button
          ><el-button type="primary" :loading="submitting" @click="createEnrollment"
            >生成接入码</el-button
          >
        </footer>
      </div>
      <div v-else class="wizard success">
        <div class="wizard-success-head">
          <span
            ><el-icon><CircleCheck /></el-icon
          ></span>
          <div>
            <h3>接入码已生成</h3>
            <p>在目标服务器安装节点程序并输入接入码，页面会自动刷新接入进度。</p>
          </div>
        </div>
        <div class="enrollment-result">
          <div>
            <span>安装包</span
            ><b>{{
              chosenPackage
                ? packagePresentation(chosenPackage)
                : `${nodeRoleLabel[activeEnrollment?.role || enrollmentForm.role]}安装包`
            }}</b>
          </div>
          <div>
            <span>一次性接入码</span
            ><code>{{ activeEnrollment?.enrollmentCode || '未能生成接入码，请重新创建' }}</code
            ><small>此接入码仅显示一次，请立即复制并妥善保存。</small>
          </div>
          <div>
            <span>到期时间</span><b>{{ formatTime(activeEnrollment?.expiresAt) }}</b>
          </div>
          <div>
            <span>接入进度</span
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
        <p class="install-hint">安装完成后，请在“节点接入”中核对并确认服务器身份。</p>
        <footer>
          <el-button @click="openEnrollmentTasks">查看接入记录</el-button
          ><el-button type="primary" plain @click="refreshEnrollment">刷新状态</el-button>
        </footer>
      </div></el-dialog
    >

    <el-dialog
      v-model="deploymentDialog"
      class="ops-dialog"
      title="发布工程"
      width="min(560px, calc(100vw - 32px))"
      ><el-form :model="deploymentForm" label-position="top" class="dialog-form"
        ><el-form-item label="工程" required
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
        ><el-form-item label="运行模式" required
          ><el-radio-group v-model="deploymentForm.deploymentMode"
            ><el-radio-button value="development">开发模式</el-radio-button
            ><el-radio-button value="production" disabled
              >生产模式（暂不可用）</el-radio-button
            ></el-radio-group
          ></el-form-item
        ><el-form-item label="运行位置" required
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
            暂无可用运行位置，请先接入 Linux 运行节点并确保其在线。
          </p></el-form-item
        ><el-form-item label="采集节点" required
          ><el-select
            v-model="deploymentForm.hostNodeId"
            class="w-full"
            :disabled="collectorNodes.length === 0"
            ><el-option
              v-for="node in collectorNodes"
              :key="node.id"
              :label="node.name"
              :value="node.id"
          /></el-select>
          <p v-if="collectorNodes.length === 0" class="form-field-hint is-warning">
            暂无可用采集节点，请先接入采集节点并确保其在线。
          </p></el-form-item
        ></el-form
      ><template #footer
        ><el-button @click="deploymentDialog = false">取消</el-button
        ><el-button
          type="primary"
          :loading="submitting"
          :disabled="!canPublish"
          @click="createDeployment"
          >开始发布</el-button
        ></template
      ></el-dialog
    >

    <el-drawer
      v-model="nodeDetailDialog"
      class="ops-drawer"
      :title="selectedNode?.name || '服务器节点详情'"
      size="min(560px, 100vw)"
    >
      <template v-if="selectedNode">
        <div class="node-detail-head">
          <span class="node-detail-head__icon"
            ><el-icon><Monitor /></el-icon
          ></span>
          <div>
            <b>{{ nodeRoleLabel[selectedNode.role] }}</b>
            <small>{{ selectedNode.ipAddress || '暂无地址' }}</small>
          </div>
          <el-tag
            :type="nodeHealthPresentation(selectedNode.health, selectedNode.observedStatus).type"
            effect="light"
          >
            {{ nodeHealthPresentation(selectedNode.health, selectedNode.observedStatus).label }}
          </el-tag>
        </div>
        <div class="node-detail-metrics">
          <div>
            <span>CPU</span
            ><b>{{ metric(selectedNode.metrics?.cpuPercent, selectedNode.health) }}</b>
          </div>
          <div>
            <span>内存</span
            ><b>{{ metric(selectedNode.metrics?.memoryPercent, selectedNode.health) }}</b>
          </div>
          <div>
            <span>磁盘</span
            ><b>{{ metric(selectedNode.metrics?.diskPercent, selectedNode.health) }}</b>
          </div>
        </div>
        <section class="drawer-section">
          <header class="drawer-section__header">
            <h3>可运行服务</h3>
          </header>
          <div class="node-service-guide">
            <span class="node-service-guide__icon">
              <el-icon
                ><component :is="selectedNode.role === 'runtime_linux' ? Cpu : Connection"
              /></el-icon>
            </span>
            <div>
              <b>{{ nodeServicePresentation(selectedNode) }}</b>
              <p>
                {{
                  selectedNode.role === 'runtime_linux'
                    ? '工程发布到此运行位置后，可运行计算和报警服务。'
                    : '该节点用于运行工程的采集服务。'
                }}
              </p>
            </div>
            <el-button type="primary" plain @click="openNodeServices">前往工程服务</el-button>
          </div>
        </section>
        <dl class="deployment-meta node-detail-meta">
          <div>
            <dt>运行位置</dt>
            <dd>{{ nodeLocationPresentation(selectedNode) }}</dd>
          </div>
          <div>
            <dt>节点程序版本</dt>
            <dd>{{ selectedNode.agentVersion || '暂无版本信息' }}</dd>
          </div>
          <div>
            <dt>操作系统</dt>
            <dd>{{ selectedNode.os || '暂无系统信息' }}</dd>
          </div>
          <div>
            <dt>系统架构</dt>
            <dd>{{ selectedNode.architecture || '暂无架构信息' }}</dd>
          </div>
          <div>
            <dt>节点程序状态</dt>
            <dd>{{ lifecyclePresentation(selectedNode.observedStatus) }}</dd>
          </div>
          <div>
            <dt>最后上报</dt>
            <dd>{{ relativeTime(selectedNode.lastHeartbeatAt, '尚未上报') }}</dd>
          </div>
        </dl>
      </template>
    </el-drawer>

    <el-drawer
      v-model="deploymentDetailDialog"
      class="ops-drawer"
      :title="selectedDeployment?.projectName || '工程服务详情'"
      size="min(620px, 100vw)"
      ><template v-if="selectedDeployment"
        ><div class="deployment-summary">
          <div class="deployment-summary__state">
            <span class="deployment-summary__icon"
              ><el-icon><FolderOpened /></el-icon
            ></span>
            <div>
              <b>{{ deploymentPresentation(selectedDeployment).label }}</b>
              <small>{{ deploymentPresentation(selectedDeployment).detail }}</small>
            </div>
          </div>
          <div class="deployment-summary__progress">
            <span>发布进度</span><b>{{ progress(selectedDeployment.progress) }}%</b>
            <el-progress
              :percentage="progress(selectedDeployment.progress)"
              :show-text="false"
              :stroke-width="6"
            />
          </div>
        </div>
        <dl class="deployment-meta">
          <div>
            <dt>运行模式</dt>
            <dd>{{ selectedDeployment.mode === 'production' ? '生产模式' : '开发模式' }}</dd>
          </div>
          <div>
            <dt>运行位置</dt>
            <dd>{{ selectedDeployment.runtimeClusterName || '未分配' }}</dd>
          </div>
          <div>
            <dt>当前版本</dt>
            <dd>{{ selectedDeployment.version || '暂无版本信息' }}</dd>
          </div>
          <div>
            <dt>更新时间</dt>
            <dd>{{ formatTime(selectedDeployment.updatedAt) }}</dd>
          </div>
        </dl>
        <section class="drawer-section">
          <header class="drawer-section__header">
            <h3>运行服务</h3>
            <span>{{ workloadRows.length }}</span>
          </header>
          <div v-if="workloadRows.length" class="workload-list">
            <article v-for="workload in workloadRows" :key="workload.role">
              <span class="workload-icon">
                <el-icon><component :is="workloadRoleIcon[workload.role]" /></el-icon>
              </span>
              <div class="workload-main">
                <b>{{ workloadRoleLabel[workload.role] }}</b>
                <span>{{ workloadPresentation(workload).detail }}</span>
              </div>
              <el-tag size="small" :type="workloadPresentation(workload).type" effect="light">
                {{ workloadPresentation(workload).label }}
              </el-tag>
              <footer class="workload-actions">
                <el-button
                  v-if="workload.observedStatus !== 'running'"
                  size="small"
                  type="primary"
                  plain
                  :loading="workloadActionKey === `${workload.role}:start`"
                  :disabled="workloadActionDisabled(workload, 'start')"
                  @click="operateWorkload(workload.role, 'start')"
                  ><el-icon><VideoPlay /></el-icon>启动</el-button
                ><el-button
                  v-else
                  size="small"
                  :loading="workloadActionKey === `${workload.role}:stop`"
                  :disabled="workloadActionDisabled(workload, 'stop')"
                  @click="operateWorkload(workload.role, 'stop')"
                  ><el-icon><VideoPause /></el-icon>停止</el-button
                ><el-button
                  v-if="workload.observedStatus === 'running'"
                  size="small"
                  :loading="workloadActionKey === `${workload.role}:restart`"
                  :disabled="workloadActionDisabled(workload, 'restart')"
                  @click="operateWorkload(workload.role, 'restart')"
                  ><el-icon><RefreshRight /></el-icon>重启</el-button
                >
              </footer>
            </article>
          </div>
          <el-empty v-else description="当前工程未配置运行服务" :image-size="64" />
        </section>
        <section class="timeline">
          <header class="timeline__header">
            <div>
              <h3>操作记录</h3>
              <p v-if="deploymentRunPollingState === 'timed_out'" class="timeline__pending">
                状态更新较慢，请手动刷新。
              </p>
            </div>
            <div class="timeline__actions">
              <el-button link :loading="refreshingDeployment" @click="refreshDeploymentDetail"
                >刷新状态</el-button
              >
              <el-button
                v-if="activeDeploymentRunId"
                link
                type="primary"
                :loading="refreshingRun"
                @click="refreshDeploymentRun"
                >刷新进度</el-button
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
          ><el-empty v-else description="暂无操作记录" :image-size="72" /></section></template
    ></el-drawer>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowRight,
  CircleCheck,
  Connection,
  Cpu,
  DataAnalysis,
  Download,
  FolderOpened,
  Monitor,
  Plus,
  Refresh,
  RefreshRight,
  Tickets,
  UploadFilled,
  VideoPause,
  VideoPlay,
  WarningFilled,
} from '@element-plus/icons-vue'
import {
  opsAPI,
  type DeploymentRunEvent,
  type DeploymentWorkload,
  type HostNode,
  type NodeEnrollment,
  type NodePackage,
  type OpsNodeRole,
  type OpsWorkloadAction,
  type OpsWorkloadRole,
  type ProjectDeployment,
  type RuntimeCluster,
} from '@/api/ops.api'
import { getApiErrorMessage as getRawApiErrorMessage } from '@/utils/request'
import { projectAPI } from '@/api/project.api'
import WorkbenchPagination from '@/components/WorkbenchPagination.vue'
import {
  deploymentCollectorNodeId,
  deploymentHasCollectorNode,
  deploymentStatePresentation,
  enrollmentStatusPresentation,
  healthPresentation,
  isDeployableRuntimeCluster,
  isDeploymentPending,
  isDeploymentRunActive,
  isSchedulableCollectorNode,
  lifecyclePresentation,
  nodeHealthPresentation,
  nodeLocationPresentation,
  nodeRoleLabel,
  nodeServicePresentation,
  runEventMessagePresentation,
  runEventStagePresentation,
  sortRunEvents,
  workloadStatePresentation,
  workloadRoleLabel,
} from './utils/ops-presentation'

type Tab = 'overview' | 'clusters' | 'nodes' | 'deployments' | 'enrollments'
const activeTab = ref<Tab>('overview')
const loading = ref(false)
const initialLoadComplete = ref(false)
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
const nodeDetailDialog = ref(false)
const selectedNode = ref<HostNode | null>(null)
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
const enrollmentPager = reactive({ page: 1, pageSize: 20, total: 0 })
const deploymentPager = reactive({ page: 1, pageSize: 20, total: 0 })
const lastRefreshedAt = ref('')
const timeTick = ref(0)
let enrollmentTimer: ReturnType<typeof setInterval> | undefined
let deploymentRunTimer: ReturnType<typeof setInterval> | undefined
let timeTickTimer: ReturnType<typeof setInterval> | undefined
const clusterForm = reactive({
  name: '',
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
  { id: 'overview' as const, label: '运维总览', count: null, icon: DataAnalysis },
  { id: 'nodes' as const, label: '服务器节点', count: nodePager.total, icon: Monitor },
  {
    id: 'deployments' as const,
    label: '工程服务',
    count: deploymentPager.total,
    icon: FolderOpened,
  },
  {
    id: 'enrollments' as const,
    label: '节点接入',
    count: enrollmentPager.total,
    icon: Tickets,
  },
  { id: 'clusters' as const, label: '运行资源池', count: clusterPager.total, icon: Connection },
])
const roles = [
  {
    value: 'runtime_linux' as const,
    label: 'Linux 运行节点',
    description: '运行工程中的计算和报警服务',
    icon: Cpu,
  },
  {
    value: 'collector_linux' as const,
    label: 'Linux 采集节点',
    description: '运行 Linux 采集服务',
    icon: Connection,
  },
  {
    value: 'collector_windows' as const,
    label: 'Windows 采集节点',
    description: '运行 Windows 采集服务',
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
const canPublish = computed(
  () => deployableRuntimeClusters.value.length > 0 && collectorNodes.value.length > 0,
)
const publishBlockedReason = computed(() => {
  if (deployableRuntimeClusters.value.length === 0)
    return '当前未找到可用运行资源池，请检查资源池状态'
  if (collectorNodes.value.length === 0) return '当前未找到可用采集节点，请检查节点是否已接入并在线'
  return ''
})
const deploymentCollectorNode = (deployment: ProjectDeployment) => {
  const nodeId = deploymentCollectorNodeId(deployment)
  return nodes.value.find((node) => node.id === nodeId)
}
const deploymentRuntimeCluster = (deployment: ProjectDeployment) =>
  clusters.value.find((cluster) => cluster.id === deployment.runtimeClusterId)
const deploymentPresentation = (deployment: ProjectDeployment) =>
  deploymentStatePresentation(
    deployment,
    deploymentCollectorNode(deployment),
    deploymentRuntimeCluster(deployment),
  )
const sortedEnrollments = computed(() => {
  const priority: Record<string, number> = {
    claimed: 0,
    registered: 1,
    failed: 2,
    created: 3,
    downloaded: 4,
    approved: 5,
    expired: 6,
    rejected: 7,
  }
  return [...enrollments.value].sort((left, right) => {
    const statusOrder = (priority[left.status] ?? 99) - (priority[right.status] ?? 99)
    if (statusOrder !== 0) return statusOrder
    return (
      new Date(right.updatedAt || right.createdAt || 0).getTime() -
      new Date(left.updatedAt || left.createdAt || 0).getTime()
    )
  })
})
const overviewStats = computed(() => [
  {
    id: 'nodes',
    label: '服务器节点',
    value: nodePager.total,
    detail: '查看节点状态',
    icon: Monitor,
    target: 'nodes' as Tab,
    tone: 'success',
  },
  {
    id: 'deployments',
    label: '工程服务',
    value: deploymentPager.total,
    detail: '查看发布状态',
    icon: FolderOpened,
    target: 'deployments' as Tab,
    tone: 'info',
  },
  {
    id: 'enrollments',
    label: '节点接入',
    value: enrollmentPager.total,
    detail: '查看接入记录',
    icon: Tickets,
    target: 'enrollments' as Tab,
    tone: 'info',
  },
])
const showOnboarding = computed(
  () =>
    activeTab.value === 'overview' &&
    initialLoadComplete.value &&
    !loadError.value &&
    nodePager.total === 0 &&
    enrollmentPager.total === 0,
)
const workloadRows = computed<DeploymentWorkload[]>(() => {
  const order: Record<DeploymentWorkload['role'], number> = {
    compute: 0,
    alert: 1,
    collector: 2,
  }
  return [...(selectedDeployment.value?.workloads || [])].sort(
    (left, right) => order[left.role] - order[right.role],
  )
})
const workloadPresentation = (workload: DeploymentWorkload) => {
  const deployment = selectedDeployment.value
  const hostHealth =
    workload.role === 'collector'
      ? deployment && deploymentCollectorNode(deployment)?.health
      : deployment && deploymentRuntimeCluster(deployment)?.health
  return workloadStatePresentation(workload, hostHealth || undefined)
}
const workloadRoleIcon = { compute: Cpu, alert: WarningFilled, collector: Connection }
// 节点断开后不再把最后一次采样值展示成当前资源使用率。
const metric = (value?: number, health?: string) =>
  health !== 'unavailable' && typeof value === 'number' && Number.isFinite(value)
    ? `${Math.round(value)}%`
    : '--'
const metricValue = (value?: number, health?: string) =>
  health !== 'unavailable' && typeof value === 'number' && Number.isFinite(value)
    ? Math.min(100, Math.max(0, Math.round(value)))
    : 0
const metricTone = (value?: number, health?: string) =>
  health === 'unavailable' || typeof value !== 'number'
    ? 'unknown'
    : value >= 85
      ? 'danger'
      : value >= 70
        ? 'warning'
        : 'normal'
const formatPackageSize = (bytes?: number) => {
  if (!bytes || bytes < 1024) return bytes ? `${bytes} B` : ''
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
const packagePresentation = (item: NodePackage) =>
  [
    `${nodeRoleLabel[item.role]}安装包`,
    item.architecture === 'multi' ? '通用架构' : item.architecture?.toUpperCase(),
    item.version,
    item.size ? formatPackageSize(item.size) : '',
  ]
    .filter(Boolean)
    .join(' · ')
const progress = (value?: number) => Math.min(100, Math.max(0, value || 0))
const healthClass = (value?: string) => `health-dot ${value || 'unknown'}`
const getApiErrorMessage = (error: unknown, fallback: string) => {
  const message = getRawApiErrorMessage(error, fallback)
  if (/network error|failed to fetch|err_network|econnrefused/i.test(message)) {
    return '网络连接失败，请检查网络后重试'
  }
  if (/timeout|timed out|econnaborted/i.test(message)) return '请求超时，请重试'
  if (/request failed with status code 5\d\d/i.test(message)) return fallback
  return message
}
const formatTime = (value?: string) =>
  value
    ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'short' }).format(
        new Date(value),
      )
    : '-'
function relativeTime(value?: string, emptyLabel = '暂无更新') {
  void timeTick.value
  if (!value) return emptyLabel
  const minutes = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 60000))
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  if (minutes < 24 * 60) return `${Math.floor(minutes / 60)} 小时前`
  return formatTime(value)
}
function timeUntil(value?: string) {
  void timeTick.value
  if (!value) return '未设置'
  const minutes = Math.round((new Date(value).getTime() - Date.now()) / 60_000)
  if (minutes <= 0) return '已到期'
  if (minutes < 60) return `${minutes} 分钟后`
  if (minutes < 24 * 60) return `${Math.ceil(minutes / 60)} 小时后`
  return formatTime(value)
}
function enrollmentCodeState(task: NodeEnrollment) {
  if (task.status === 'approved') return { label: '已使用', detail: '接入已完成' }
  if (task.status === 'claimed') return { label: '已使用', detail: '等待确认' }
  if (['rejected', 'expired', 'failed'].includes(task.status)) {
    return { label: '已失效', detail: formatTime(task.expiresAt) }
  }
  return { label: timeUntil(task.expiresAt), detail: formatTime(task.expiresAt) }
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
  const resourceNames = ['运行资源池', '服务器节点', '节点接入', '安装包', '工程服务']
  const results = await Promise.allSettled([
    opsAPI.listRuntimeClusters({ page: clusterPager.page, pageSize: clusterPager.pageSize }),
    opsAPI.listHostNodes({ page: nodePager.page, pageSize: nodePager.pageSize }),
    opsAPI.listEnrollments({
      page: enrollmentPager.page,
      pageSize: enrollmentPager.pageSize,
    }),
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
  if (c.status === 'fulfilled') {
    enrollments.value = c.value.items
    enrollmentPager.total = c.value.total
  }
  if (d.status === 'fulfilled') packages.value = d.value.items
  if (e.status === 'fulfilled') {
    deployments.value = e.value.items
    deploymentPager.total = e.value.total
  }
  const failedNames = results
    .map((result, index) => (result.status === 'rejected' ? resourceNames[index] : ''))
    .filter(Boolean)
  if (failedNames.length) loadError.value = `${failedNames.join('、')}加载失败，请重试`
  if (results.some((result) => result.status === 'fulfilled')) {
    lastRefreshedAt.value = new Date().toISOString()
  }
  initialLoadComplete.value = true
  loading.value = false
}
async function reloadSection(action: () => Promise<void>, fallback: string) {
  loading.value = true
  loadError.value = ''
  try {
    await action()
    lastRefreshedAt.value = new Date().toISOString()
  } catch (error) {
    loadError.value = getApiErrorMessage(error, fallback)
  } finally {
    loading.value = false
  }
}
const reloadClusters = () =>
  reloadSection(async () => {
    const result = await opsAPI.listRuntimeClusters({
      page: clusterPager.page,
      pageSize: clusterPager.pageSize,
    })
    clusters.value = result.items
    clusterPager.total = result.total
  }, '运行资源池加载失败')
const reloadNodes = () =>
  reloadSection(async () => {
    const result = await opsAPI.listHostNodes({
      page: nodePager.page,
      pageSize: nodePager.pageSize,
    })
    nodes.value = result.items
    nodePager.total = result.total
  }, '服务器节点加载失败')
const reloadEnrollments = () =>
  reloadSection(async () => {
    const result = await opsAPI.listEnrollments({
      page: enrollmentPager.page,
      pageSize: enrollmentPager.pageSize,
    })
    enrollments.value = result.items
    enrollmentPager.total = result.total
  }, '接入记录加载失败')
const reloadDeployments = () =>
  reloadSection(async () => {
    const result = await opsAPI.listProjectDeployments({
      page: deploymentPager.page,
      pageSize: deploymentPager.pageSize,
    })
    deployments.value = result.items
    deploymentPager.total = result.total
  }, '工程服务加载失败')
function openEnrollment(clusterId = '') {
  wizardStep.value = 0
  activeEnrollment.value = null
  enrollmentForm.role = 'runtime_linux'
  enrollmentForm.displayName = ''
  enrollmentForm.runtimeClusterId =
    clusterId || (clusters.value.length === 1 ? clusters.value[0]?.id || '' : '')
  enrollmentForm.packageId =
    packages.value.find((item) => item.role === 'runtime_linux' && item.available === true)?.id ||
    ''
  wizardDialog.value = true
}
function openEnrollmentTasks() {
  wizardDialog.value = false
  activeTab.value = 'enrollments'
}
function openNodeDetail(node: HostNode) {
  selectedNode.value = node
  nodeDetailDialog.value = true
}
function openNodeServices() {
  nodeDetailDialog.value = false
  activeTab.value = 'deployments'
}
async function createCluster() {
  if (!clusterForm.name.trim()) return ElMessage.warning('请填写资源池名称')
  submitting.value = true
  try {
    const cluster = await opsAPI.createRuntimeCluster({
      ...clusterForm,
      name: clusterForm.name.trim(),
      // 资源池编码仅用于系统识别，不要求普通用户手动维护。
      code: `runtime-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
    })
    clusters.value = [cluster, ...clusters.value]
    clusterPager.total += 1
    enrollmentForm.runtimeClusterId = cluster.id
    clusterDialog.value = false
    clusterForm.name = ''
    clusterForm.description = ''
    ElMessage.success(wizardDialog.value ? '运行资源池已创建，可继续配置节点' : '运行资源池已创建')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '创建运行资源池失败'))
  } finally {
    submitting.value = false
  }
}
async function createEnrollment() {
  if (!enrollmentForm.displayName.trim()) return ElMessage.warning('请填写节点名称')
  if (
    enrollmentForm.role === 'runtime_linux' &&
    clusters.value.length > 0 &&
    !enrollmentForm.runtimeClusterId
  )
    return ElMessage.warning('请选择运行位置')
  if (!enrollmentForm.packageId) return ElMessage.warning('请选择安装包')
  submitting.value = true
  try {
    const runtimeClusterId = enrollmentForm.runtimeClusterId || undefined
    const task = await opsAPI.createEnrollment({
      role: enrollmentForm.role,
      displayName: enrollmentForm.displayName.trim(),
      ttlMinutes: enrollmentForm.ttlMinutes,
      ...(enrollmentForm.role === 'runtime_linux' && runtimeClusterId ? { runtimeClusterId } : {}),
      packageId: enrollmentForm.packageId,
    })
    activeEnrollment.value = task
    enrollments.value = [task, ...enrollments.value.filter((item) => item.id !== task.id)]
    enrollmentPager.total += 1
    wizardStep.value = 1
    startEnrollmentPolling()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '生成接入码失败'))
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
    ElMessage.error(getApiErrorMessage(error, '刷新接入进度失败'))
  }
}
async function approveEnrollment(task: NodeEnrollment) {
  if (!task.machineFingerprint) {
    ElMessage.warning('未获取到设备标识，暂不能确认接入')
    return
  }
  const serverName = task.reportedHostName || task.displayName || '未命名服务器'
  try {
    await ElMessageBox.confirm(
      `请核对服务器“${serverName}”和设备标识“${task.machineFingerprint}”。确认无误后，服务器将接入平台。`,
      '确认服务器接入',
      { type: 'warning', confirmButtonText: '确认接入', cancelButtonText: '取消' },
    )
    const approved = await opsAPI.approveEnrollment(task.id)
    enrollments.value = enrollments.value.map((item) => (item.id === approved.id ? approved : item))
    ElMessage.success('服务器已接入')
    await reloadAll()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(getApiErrorMessage(error, '确认服务器接入失败'))
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
    ElMessage.success('已拒绝服务器接入')
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(getApiErrorMessage(error, '拒绝服务器接入失败'))
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
    ElMessage.error(getApiErrorMessage(error, '工程列表加载失败'))
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
  if (!canPublish.value) return ElMessage.warning(publishBlockedReason.value)
  if (!deploymentForm.projectId || !deploymentForm.runtimeClusterId)
    return ElMessage.warning('请选择工程和运行位置')
  if (
    !deployableRuntimeClusters.value.some(
      (cluster) => cluster.id === deploymentForm.runtimeClusterId,
    )
  )
    return ElMessage.warning('请选择已就绪的运行位置')
  if (!deploymentForm.hostNodeId) return ElMessage.warning('请选择采集节点')
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
    deploymentPager.total += 1
    deploymentDialog.value = false
    deploymentForm.projectId = ''
    deploymentForm.hostNodeId = ''
    ElMessage.success('工程已开始发布')
    await openDeploymentDetail(deployment)
    if (deployment.runId) startDeploymentRunPolling(deployment.runId)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '工程发布失败'))
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
    ElMessage.error(getApiErrorMessage(error, '加载工程服务详情失败'))
  }
}
async function operateWorkload(role: OpsWorkloadRole, action: OpsWorkloadAction) {
  if (
    !selectedDeployment.value ||
    workloadActionKey.value ||
    isDeploymentPending(selectedDeployment.value.observedStatus)
  )
    return
  if (action !== 'start') {
    try {
      await ElMessageBox.confirm(
        action === 'stop'
          ? '停止后服务将不可用，需重新启动才能恢复。是否继续？'
          : '重启期间服务会短暂不可用，是否继续？',
        `${action === 'stop' ? '停止' : '重启'}${workloadRoleLabel[role]}`,
        {
          type: 'warning',
          confirmButtonText: action === 'stop' ? '确认停止' : '确认重启',
          cancelButtonText: '取消',
        },
      )
    } catch (error) {
      if (error === 'cancel' || error === 'close') return
      ElMessage.error('操作失败，请重试')
      return
    }
  }
  workloadActionKey.value = `${role}:${action}`
  try {
    const run = await opsAPI.runWorkloadAction(selectedDeployment.value.id, role, action)
    runEvents.value = sortRunEvents([...(run?.events || []), ...runEvents.value])
    ElMessage.success(
      `${workloadRoleLabel[role]}${action === 'start' ? '正在启动' : action === 'stop' ? '正在停止' : '正在重启'}`,
    )
    selectedDeployment.value = await opsAPI.getProjectDeployment(selectedDeployment.value.id)
    deployments.value = deployments.value.map((item) =>
      item.id === selectedDeployment.value?.id ? selectedDeployment.value! : item,
    )
    if (run?.id) startDeploymentRunPolling(run.id)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '无法执行该操作，请重试'))
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
    ElMessage.success('状态已刷新')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '刷新状态失败'))
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
      ElMessage.info('操作仍在进行，请稍后刷新')
    } else {
      completeDeploymentRunPolling()
      ElMessage.success('操作已完成，状态已更新')
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '刷新操作进度失败'))
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
    if (enrollmentForm.role !== 'runtime_linux') {
      enrollmentForm.runtimeClusterId = ''
    } else if (clusters.value.length === 1) {
      enrollmentForm.runtimeClusterId = clusters.value[0]?.id || ''
    }
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
  timeTickTimer = setInterval(() => {
    timeTick.value += 1
  }, 60_000)
})
onBeforeUnmount(() => {
  if (enrollmentTimer) clearInterval(enrollmentTimer)
  if (timeTickTimer) clearInterval(timeTickTimer)
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
  grid-template-columns: repeat(3, minmax(0, 1fr));
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

/* 运维工作台统一使用 dev_ide 的 cockpit token，下面样式覆盖早期页面的硬编码浅色实现。 */
.ops-page {
  gap: 12px;
  overflow: hidden;
  padding: 16px;
  background: var(--ck-bg-primary);
  color: var(--ck-text-primary);
}
.ops-page h1 {
  color: var(--ck-text-primary);
  font-size: 20px;
  line-height: 1.25;
}
.ops-page h2 {
  color: var(--ck-text-primary);
  font-size: 16px;
}
.ops-page h3 {
  color: var(--ck-text-primary);
  font-size: 14px;
}
.subtitle,
.section-heading p,
.panel header p,
.form-field-hint {
  color: var(--ck-text-secondary);
}
.ops-page__header {
  min-height: 64px;
  flex: none;
  padding: 12px 16px;
  border: 1px solid var(--ck-border);
  border-radius: var(--ck-radius-lg);
  background: var(--ck-bg-card);
  box-shadow: var(--ck-shadow-sm);
  backdrop-filter: blur(10px);
}
.page-heading {
  min-width: 0;
}
.page-heading__title {
  display: flex;
  align-items: center;
  gap: 10px;
}
.system-summary {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 24px;
  padding: 2px 9px;
  border: 1px solid rgba(34, 197, 94, 0.24);
  border-radius: var(--ck-radius-full);
  background: rgba(34, 197, 94, 0.08);
  color: #15803d;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.system-summary.attention {
  border-color: rgba(245, 158, 11, 0.3);
  background: rgba(245, 158, 11, 0.1);
  color: #b45309;
}
.system-summary .health-dot {
  width: 6px;
  height: 6px;
  box-shadow: none;
}
.header-actions {
  flex-wrap: nowrap;
  gap: 8px;
}
.last-refresh {
  margin-right: 2px;
  color: var(--ck-text-secondary);
  font-size: 12px;
  white-space: nowrap;
}
.ops-nav {
  flex: none;
  gap: 4px;
  padding: 4px;
  border-color: var(--ck-border);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-card);
  box-shadow: var(--ck-shadow-sm);
}
.ops-nav button {
  min-width: 112px;
  min-height: 36px;
  gap: 7px;
  padding: 7px 12px;
  border-radius: 9px;
  color: var(--ck-text-secondary);
  font-size: 13px;
  transition:
    background-color 0.18s ease,
    color 0.18s ease,
    box-shadow 0.18s ease;
}
.ops-nav button:hover {
  background: var(--ck-bg-hover);
  color: var(--ck-text-primary);
}
.ops-nav button:focus-visible,
.role-grid button:focus-visible,
.stat-card:focus-visible,
.mini-list button:focus-visible {
  outline: 2px solid var(--ck-primary);
  outline-offset: 2px;
}
.ops-nav button small {
  min-width: 22px;
  padding: 1px 6px;
  background: var(--ck-bg-tertiary);
  color: var(--ck-text-secondary);
  text-align: center;
}
.ops-nav button.active {
  background: var(--ck-primary-light);
  color: var(--ck-primary);
  box-shadow: inset 0 0 0 1px rgba(29, 78, 216, 0.08);
}
.ops-nav button.active small {
  background: var(--ck-bg-secondary);
  color: var(--ck-primary);
}
.onboarding {
  min-height: 0;
  flex: 1;
  align-items: center;
  border-color: var(--ck-border);
  border-radius: var(--ck-radius-lg);
  background: var(--ck-bg-card);
  box-shadow: var(--ck-shadow-sm);
}
.onboarding-icon,
.onboarding b {
  background: var(--ck-primary-light);
  color: var(--ck-primary);
}
.onboarding p:not(.eyebrow),
.onboarding ol {
  color: var(--ck-text-secondary);
}
.content {
  gap: 12px;
  overflow: hidden;
}
.content--overview {
  overflow-y: auto;
  padding: 1px;
}
.stats {
  gap: 12px;
}
.stat-card {
  display: grid;
  min-width: 0;
  min-height: 82px;
  grid-template-columns: 38px minmax(0, 1fr) auto;
  align-items: center;
  gap: 11px;
  padding: 13px;
  border: 1px solid var(--ck-border-light);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-secondary);
  box-shadow: var(--ck-shadow-sm);
  color: inherit;
  cursor: pointer;
  font: inherit;
  text-align: left;
  transition:
    transform 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}
.stat-card:hover {
  transform: translateY(-1px);
  border-color: var(--ck-border);
  box-shadow: var(--ck-shadow-md);
}
.stat-card__icon,
.node-name__icon,
.deployment-name__icon,
.resource-card__icon,
.workload-icon,
.deployment-summary__icon {
  display: grid;
  width: 34px;
  height: 34px;
  flex: none;
  place-items: center;
  border-radius: 10px;
  background: var(--ck-primary-light);
  color: var(--ck-primary);
  font-size: 17px;
}
.stats .stat-card__icon {
  display: grid;
  color: var(--ck-primary);
  font-size: 17px;
}
.stat-card__content,
.stat-card__content > span,
.stat-card__content b,
.stat-card__content small {
  display: block;
  min-width: 0;
}
.stat-card__content > span,
.stat-card__content small {
  color: var(--ck-text-secondary);
  font-size: 12px;
}
.stat-card__content b {
  margin: 2px 0;
  color: var(--ck-text-primary);
  font-size: 22px;
  line-height: 1.15;
}
.stat-card__arrow {
  color: var(--ck-text-secondary);
}
.stat-card.is-success .stat-card__icon {
  background: rgba(34, 197, 94, 0.1);
  color: #15803d;
}
.stat-card.is-info .stat-card__icon {
  background: rgba(14, 165, 165, 0.1);
  color: #0f766e;
}
.stat-card.is-warning .stat-card__icon {
  background: rgba(245, 158, 11, 0.12);
  color: #b45309;
}
.stat-card.is-muted .stat-card__icon {
  background: var(--ck-bg-tertiary);
  color: var(--ck-text-secondary);
}
.overview-grid {
  min-height: 0;
  flex: 1;
  gap: 12px;
}
.panel,
.resource-card,
.table-wrap {
  border-color: var(--ck-border);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-secondary);
  box-shadow: var(--ck-shadow-sm);
}
.panel {
  min-height: 224px;
  padding: 14px;
}
.panel > header {
  margin-bottom: 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--ck-border-light);
}
.mini-list,
.enrollment-list {
  gap: 5px;
}
.mini-list button {
  min-height: 50px;
  grid-template-columns: auto minmax(0, 1fr) auto;
  padding: 7px 9px;
  background: transparent;
}
.mini-list button:hover {
  background: var(--ck-bg-hover);
}
.mini-list__identity,
.mini-list__identity b,
.mini-list__identity small,
.mini-list__status,
.mini-list__status small {
  display: block;
  min-width: 0;
}
.mini-list__identity b,
.enrollment-list__identity b {
  overflow: hidden;
  color: var(--ck-text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mini-list__identity small,
.mini-list__status small,
.enrollment-list small {
  color: var(--ck-text-secondary);
  font-size: 12px;
}
.mini-list__status {
  text-align: right;
}
.mini-list__status small {
  margin-top: 3px;
}
.enrollment-list > div {
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 2px 10px;
  min-height: 54px;
  padding: 7px 6px;
  border-color: var(--ck-border-light);
  color: var(--ck-text-secondary);
}
.enrollment-list__identity,
.enrollment-list__identity small {
  display: block;
  min-width: 0;
}
.enrollment-list__meta {
  grid-column: 1/-1;
}
.enrollment-list code {
  overflow: hidden;
  color: var(--ck-text-secondary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.enrollment-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
}
.section-heading {
  min-height: 44px;
  flex: none;
  padding: 0 2px;
}
.section-heading__title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.section-heading__title > span {
  min-width: 24px;
  padding: 1px 7px;
  border-radius: var(--ck-radius-full);
  background: var(--ck-bg-tertiary);
  color: var(--ck-text-secondary);
  font-size: 11px;
  font-weight: 600;
  text-align: center;
}
.section-heading__actions {
  display: flex;
  max-width: 520px;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}
.section-heading__actions > small {
  color: var(--ck-warning);
  font-size: 12px;
  line-height: 1.35;
  text-align: right;
}
.form-field-hint.is-warning {
  color: var(--ck-warning);
}
.resource-grid {
  grid-template-columns: repeat(auto-fill, minmax(320px, 420px));
  align-content: start;
  gap: 12px;
  min-height: 0;
  overflow-y: auto;
  padding: 1px;
}
.resource-card {
  min-height: 198px;
  padding: 14px;
}
.resource-card__title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 9px;
}
.resource-card__title > div {
  min-width: 0;
}
.resource-card__title h3,
.resource-card__title small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.resource-card__title small {
  margin-top: 3px;
  color: var(--ck-text-secondary);
  font-size: 11px;
}
.resource-card__tags {
  display: flex;
  flex: none;
  align-items: center;
  gap: 5px;
}
.resource-card header > .resource-card__title {
  gap: 9px;
}
.resource-card header > .resource-card__tags {
  gap: 5px;
}
.resource-card > p {
  display: -webkit-box;
  flex: none;
  min-height: 36px;
  margin: 10px 0;
  overflow: hidden;
  color: var(--ck-text-secondary);
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.resource-card dl {
  gap: 6px;
  margin-bottom: 9px;
  padding: 9px;
  border-radius: 9px;
  background: var(--ck-bg-tertiary);
}
.resource-card dt {
  color: var(--ck-text-secondary);
  font-size: 11px;
}
.resource-card dd {
  color: var(--ck-text-primary);
  font-size: 12px;
}
.cluster-capacity {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 3px 10px;
  color: var(--ck-text-secondary);
  font-size: 11px;
}
.cluster-capacity b {
  color: var(--ck-text-primary);
  font-size: 12px;
}
.cluster-capacity .el-progress {
  grid-column: 1/-1;
}
.resource-card footer {
  margin-top: 9px;
  padding-top: 8px;
  border-color: var(--ck-border-light);
}
.resource-card footer small {
  color: var(--ck-text-secondary);
}
.resource-empty {
  grid-column: 1/-1;
  display: grid;
  min-height: 260px;
  place-items: center;
  border: 1px dashed var(--ck-border);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-card);
}
.table-wrap {
  min-height: 240px;
  flex: 1;
}
.table-wrap :deep(.el-table) {
  --el-table-bg-color: var(--ck-bg-secondary);
  --el-table-tr-bg-color: var(--ck-bg-secondary);
  --el-table-header-bg-color: var(--ck-bg-tertiary);
  --el-table-row-hover-bg-color: var(--ck-bg-hover);
  --el-table-border-color: var(--ck-border-light);
  color: var(--ck-text-primary);
}
.table-wrap :deep(.el-table th.el-table__cell) {
  height: 44px;
  background: var(--ck-bg-tertiary);
  color: var(--ck-text-secondary);
  font-size: 12px;
  font-weight: 600;
}
.table-wrap :deep(.el-table td.el-table__cell) {
  height: 68px;
  border-bottom-color: var(--ck-border-light);
}
.table-wrap :deep(.el-table__empty-block) {
  min-height: 220px;
}
.node-name,
.deployment-name {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 9px;
}
.node-name > div,
.deployment-name > div,
.table-primary-cell,
.time-cell,
.agent-state > div {
  min-width: 0;
}
.node-name b,
.deployment-name b,
.table-primary-cell b,
.time-cell b,
.agent-state b,
.node-name small,
.deployment-name small,
.table-primary-cell small,
.time-cell small,
.agent-state small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-name b,
.deployment-name b,
.table-primary-cell b,
.time-cell b,
.agent-state b {
  color: var(--ck-text-primary);
  font-size: 13px;
  font-weight: 600;
}
.node-name small,
.deployment-name small,
.table-primary-cell small,
.time-cell small,
.agent-state small,
.deployment-status small {
  margin-top: 3px;
  color: var(--ck-text-secondary);
  font-size: 11px;
}
.resource-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.resource-metrics > div {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 3px 5px;
  min-width: 0;
}
.resource-metrics span,
.resource-metrics b {
  font-size: 11px;
}
.resource-metrics span {
  color: var(--ck-text-secondary);
}
.resource-metrics b {
  color: var(--ck-text-primary);
}
.resource-metrics i {
  grid-column: 1/-1;
  display: block;
  height: 4px;
  overflow: hidden;
  border-radius: var(--ck-radius-full);
  background: var(--ck-bg-tertiary);
}
.resource-metrics em {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--ck-primary);
}
.resource-metrics .is-warning em {
  background: var(--ck-warning);
}
.resource-metrics .is-danger em {
  background: var(--ck-danger);
}
.resource-metrics .is-unknown em {
  width: 0 !important;
}
.agent-state {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.deployment-status {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}
.deployment-progress {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 34px;
  align-items: center;
  gap: 8px;
}
.deployment-progress > span {
  color: var(--ck-text-secondary);
  font-size: 11px;
  text-align: right;
}
.table-actions {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
}
.table-placeholder {
  color: var(--ck-text-secondary);
}
:deep(.ops-dialog) {
  display: flex;
  max-height: calc(100vh - 32px);
  flex-direction: column;
  margin-top: 16px !important;
  margin-bottom: 16px;
  overflow: hidden;
  border: 1px solid var(--ck-border);
  border-radius: var(--ck-radius-lg);
  background: var(--ck-bg-secondary);
}
:deep(.ops-dialog .el-dialog__header) {
  flex: none;
  margin: 0;
  padding: 16px 20px 13px;
  border-bottom: 1px solid var(--ck-border-light);
}
:deep(.ops-dialog .el-dialog__title) {
  color: var(--ck-text-primary);
  font-size: 16px;
  font-weight: 700;
}
:deep(.ops-dialog .el-dialog__body) {
  min-height: 0;
  overflow-y: auto;
  padding: 16px 20px;
  color: var(--ck-text-primary);
}
:deep(.ops-dialog .el-dialog__footer) {
  flex: none;
  padding: 12px 20px 16px;
  border-top: 1px solid var(--ck-border-light);
  background: var(--ck-bg-secondary);
}
.dialog-form--grid,
.wizard-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
}
.dialog-form .is-full,
.wizard-form__alert {
  grid-column: 1/-1;
}
.dialog-form :deep(.el-form-item),
.wizard-form :deep(.el-form-item) {
  margin-bottom: 15px;
}
.dialog-form :deep(.el-form-item__label),
.wizard-form :deep(.el-form-item__label) {
  padding-bottom: 6px;
  color: var(--ck-text-secondary);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.3;
}
.wizard {
  min-height: 0;
  padding: 14px 0 0;
}
.wizard-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}
.wizard-heading p {
  color: var(--ck-text-secondary);
  font-size: 12px;
}
.role-grid {
  gap: 8px;
}
.role-grid button {
  display: grid;
  min-height: 78px;
  grid-template-columns: 28px minmax(0, 1fr);
  align-items: center;
  gap: 9px;
  padding: 11px;
  border-color: var(--ck-border);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-secondary);
  color: var(--ck-text-secondary);
}
.role-grid button:hover {
  border-color: rgba(29, 78, 216, 0.35);
  background: var(--ck-bg-hover);
}
.role-grid button.selected {
  border-color: var(--ck-primary);
  background: var(--ck-primary-light);
  box-shadow: inset 0 0 0 1px var(--ck-primary);
}
.role-grid .el-icon {
  color: var(--ck-primary);
  font-size: 20px;
}
.role-grid button > span,
.role-grid button b,
.role-grid button small {
  display: block;
  min-width: 0;
}
.role-grid button b {
  color: var(--ck-text-primary);
  font-size: 13px;
}
.role-grid button small {
  margin-top: 4px;
  color: var(--ck-text-secondary);
  font-size: 11px;
  line-height: 1.35;
}
.wizard-form {
  margin-top: 14px;
}
.wizard-form__alert {
  margin-bottom: 12px;
}
.wizard > footer {
  position: sticky;
  z-index: 2;
  bottom: -16px;
  justify-content: flex-end;
  margin-top: 2px;
  padding: 12px 0 0;
  border-top: 1px solid var(--ck-border-light);
  background: var(--ck-bg-secondary);
}
.success {
  padding-top: 2px;
}
.wizard-success-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}
.wizard-success-head > span {
  display: grid;
  width: 42px;
  height: 42px;
  flex: none;
  place-items: center;
  border-radius: 12px;
  background: rgba(34, 197, 94, 0.1);
  color: #15803d;
  font-size: 23px;
}
.wizard-success-head p {
  margin-top: 4px;
  color: var(--ck-text-secondary);
  font-size: 12px;
}
.enrollment-result {
  border-color: var(--ck-border-light);
  background: var(--ck-border-light);
}
.enrollment-result > div {
  min-height: 66px;
  padding: 11px 12px;
  background: var(--ck-bg-secondary);
}
.enrollment-result span,
.enrollment-result small,
.install-hint {
  color: var(--ck-text-secondary);
}
.enrollment-result b,
.enrollment-result code {
  color: var(--ck-text-primary);
}
.enrollment-result code {
  overflow-wrap: anywhere;
  color: var(--ck-primary);
}
.download-actions {
  margin-top: 14px;
}
.install-hint {
  margin-top: 8px !important;
}
:deep(.ops-drawer) {
  border-left: 1px solid var(--ck-border);
  background: var(--ck-bg-secondary);
}
:deep(.ops-drawer .el-drawer__header) {
  min-height: 56px;
  margin: 0;
  padding: 14px 18px;
  border-bottom: 1px solid var(--ck-border-light);
  color: var(--ck-text-primary);
  font-size: 16px;
  font-weight: 700;
}
:deep(.ops-drawer .el-drawer__body) {
  padding: 16px 18px 24px;
  background: var(--ck-bg-secondary);
  color: var(--ck-text-primary);
}
.node-detail-head {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 13px;
  border: 1px solid var(--ck-border-light);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-tertiary);
}
.node-detail-head__icon,
.node-service-guide__icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: none;
  place-items: center;
  border-radius: 10px;
  background: var(--ck-primary-light);
  color: var(--ck-primary);
  font-size: 18px;
}
.node-detail-head > div {
  min-width: 0;
  flex: 1;
}
.node-detail-head b,
.node-detail-head small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-detail-head b {
  color: var(--ck-text-primary);
  font-size: 13px;
}
.node-detail-head small {
  margin-top: 3px;
  color: var(--ck-text-secondary);
  font-size: 11px;
}
.node-detail-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-top: 10px;
}
.node-detail-metrics > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  padding: 9px 10px;
  border: 1px solid var(--ck-border-light);
  border-radius: 9px;
}
.node-detail-metrics span {
  color: var(--ck-text-secondary);
  font-size: 11px;
}
.node-detail-metrics b {
  color: var(--ck-text-primary);
  font-size: 12px;
}
.node-service-guide {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr) auto;
  align-items: center;
  gap: 11px;
  padding: 12px;
  border: 1px solid var(--ck-border-light);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-tertiary);
}
.node-service-guide > div {
  min-width: 0;
}
.node-service-guide b {
  color: var(--ck-text-primary);
  font-size: 13px;
}
.node-service-guide p {
  margin-top: 4px;
  color: var(--ck-text-secondary);
  font-size: 11px;
  line-height: 1.45;
}
.deployment-summary {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 220px;
  gap: 16px;
  margin-bottom: 12px;
  padding: 13px;
  border: 1px solid var(--ck-border-light);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-tertiary);
}
.deployment-summary__state {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}
.deployment-summary__state b,
.deployment-summary__state small,
.deployment-summary__progress span,
.deployment-summary__progress b {
  display: block;
}
.deployment-summary__state b,
.deployment-summary__progress b {
  color: var(--ck-text-primary);
  font-size: 13px;
}
.deployment-summary__state small,
.deployment-summary__progress span {
  margin-top: 3px;
  color: var(--ck-text-secondary);
  font-size: 11px;
}
.deployment-summary__progress {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 3px 8px;
}
.deployment-summary__progress .el-progress {
  grid-column: 1/-1;
}
.deployment-meta {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  margin: 0 0 18px;
  overflow: hidden;
  border: 1px solid var(--ck-border-light);
  border-radius: var(--ck-radius-md);
  background: var(--ck-border-light);
}
.node-detail-meta {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-top: 18px;
}
.deployment-meta > div {
  min-width: 0;
  padding: 10px;
  background: var(--ck-bg-secondary);
}
.deployment-meta dt {
  color: var(--ck-text-secondary);
  font-size: 11px;
}
.deployment-meta dd {
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--ck-text-primary);
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.drawer-section,
.timeline {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--ck-border-light);
}
.drawer-section__header,
.timeline__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}
.drawer-section__header > span {
  min-width: 24px;
  padding: 1px 7px;
  border-radius: var(--ck-radius-full);
  background: var(--ck-bg-tertiary);
  color: var(--ck-text-secondary);
  font-size: 11px;
  text-align: center;
}
.workload-list {
  gap: 7px;
}
.workload-list article {
  display: grid;
  min-height: 64px;
  grid-template-columns: 34px minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 10px;
  padding: 10px 11px;
  border-color: var(--ck-border-light);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-secondary);
}
.workload-icon {
  width: 32px;
  height: 32px;
  border-radius: 9px;
  font-size: 16px;
}
.workload-list .workload-icon {
  display: grid;
  margin-top: 0;
  color: var(--ck-primary);
  font-size: 16px;
}
.workload-main,
.workload-main b,
.workload-main span {
  display: block;
  min-width: 0;
}
.workload-main b {
  color: var(--ck-text-primary);
  font-size: 13px;
}
.workload-main span {
  margin-top: 3px;
  overflow: hidden;
  color: var(--ck-text-secondary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.workload-list .workload-actions {
  grid-column: auto;
  display: flex;
  flex-wrap: nowrap;
  gap: 5px;
  margin: 0;
}
.timeline {
  margin-top: 18px;
}
.timeline p {
  color: var(--ck-text-secondary);
}
.timeline .timeline__header h3 {
  margin: 0;
}
.timeline__pending {
  color: #b45309 !important;
}

html.dark .system-summary,
[data-theme='dark'] .system-summary {
  color: #86efac;
}
html.dark .system-summary.attention,
[data-theme='dark'] .system-summary.attention {
  color: #fcd34d;
}

@media (max-width: 1100px) {
  .stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .overview-grid {
    grid-template-columns: 1fr;
  }
  .panel {
    min-height: 210px;
  }
}
@media (max-width: 900px) {
  .ops-page {
    padding: 12px;
  }
  .ops-page__header {
    align-items: flex-start;
  }
  .last-refresh {
    display: none;
  }
  .resource-grid {
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  }
}
@media (max-width: 640px) {
  .ops-page {
    gap: 8px;
    padding: 8px;
  }
  .ops-page__header {
    padding: 10px 12px;
  }
  .page-heading__title {
    align-items: flex-start;
    flex-direction: column;
    gap: 5px;
  }
  .header-actions {
    width: 100%;
  }
  .header-actions .el-button {
    flex: 1;
  }
  .ops-nav {
    overflow-x: auto;
  }
  .ops-nav button {
    min-width: 108px;
    flex: none;
  }
  .stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
  .stat-card {
    min-height: 76px;
    grid-template-columns: 30px minmax(0, 1fr);
    gap: 8px;
    padding: 10px;
  }
  .stat-card__icon {
    width: 30px;
    height: 30px;
  }
  .stat-card__arrow {
    display: none;
  }
  .resource-grid {
    grid-template-columns: 1fr;
  }
  .section-heading {
    gap: 8px;
  }
  .section-heading > .el-button,
  .section-heading > .el-tooltip,
  .section-heading > .el-tooltip > span,
  .section-heading > .el-tooltip .el-button {
    width: 100%;
  }
  .section-heading__actions {
    width: 100%;
    max-width: none;
    align-items: flex-start;
    flex-direction: column;
  }
  .section-heading__actions > small {
    text-align: left;
  }
  .section-heading__actions > span,
  .section-heading__actions .el-button {
    width: 100%;
  }
  .table-wrap {
    overflow: auto;
  }
  .table-wrap :deep(.el-table) {
    min-width: 880px;
  }
  :deep(.ops-dialog) {
    width: calc(100vw - 16px) !important;
    max-height: calc(100vh - 16px);
    margin-top: 8px !important;
    margin-bottom: 8px;
  }
  :deep(.ops-dialog .el-dialog__header),
  :deep(.ops-dialog .el-dialog__body),
  :deep(.ops-dialog .el-dialog__footer) {
    padding-right: 14px;
    padding-left: 14px;
  }
  .role-grid,
  .dialog-form--grid,
  .wizard-form,
  .enrollment-result {
    grid-template-columns: 1fr;
  }
  .role-grid button {
    min-height: 66px;
  }
  .wizard-heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 4px;
  }
  .deployment-summary {
    grid-template-columns: 1fr;
  }
  .node-service-guide {
    grid-template-columns: 38px minmax(0, 1fr);
  }
  .node-service-guide .el-button {
    grid-column: 2;
    justify-self: start;
  }
  .deployment-meta {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .workload-list article {
    grid-template-columns: 32px minmax(0, 1fr) auto;
  }
  .workload-actions {
    grid-column: 2/-1;
    justify-content: flex-end;
  }
  .timeline__header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
