<template>
  <section class="ops-console ck-workbench-page" data-testid="ops-management-console">
    <main class="ops-workspace">
      <div v-if="canAdministerOperations && activeTab === 'environments'" class="ops-view">
        <div v-if="environmentManagementMode" class="ops-page">
          <h2 class="sr-only">{{ $t('opsConsole.sections.environments') }}</h2>
          <div class="ops-toolbar">
            <div class="ops-toolbar__filters">
              <OpsSectionSwitcher
                :model-value="activeTab"
                :can-administer-operations="canAdministerOperations"
                @update:model-value="switchSection"
              />
              <span class="ops-toolbar__divider" />
              <el-input
                v-model="environmentKeyword"
                clearable
                :placeholder="$t('opsConsole.environments.searchPlaceholder')"
              />
              <el-select
                v-model="environmentStatus"
                :placeholder="$t('opsConsole.environments.allStatus')"
              >
                <el-option :label="$t('opsConsole.environments.allStatus')" value="all" />
                <el-option :label="$t('opsConsole.common.available')" value="available" />
                <el-option :label="$t('opsConsole.common.attention')" value="attention" />
                <el-option :label="$t('opsConsole.common.uninitialized')" value="uninitialized" />
              </el-select>
            </div>
            <div class="ops-toolbar__actions">
              <span class="ops-count">{{
                $t('opsConsole.environments.total', { count: environmentPage.total })
              }}</span>
              <el-button :loading="refreshing" @click="loadDashboard">{{
                $t('opsConsole.common.refresh')
              }}</el-button>
              <el-button @click="closeEnvironmentManagement">{{
                $t('opsConsole.environments.backToOverview')
              }}</el-button>
              <el-button type="primary" @click="openEnvironmentDialog">{{
                $t('opsConsole.environments.create')
              }}</el-button>
            </div>
          </div>

          <div class="ops-list-panel ck-content-area">
            <div class="ck-content-scroll">
              <div class="ck-table-shell">
                <el-table
                  :data="pagedEnvironments"
                  row-class-name="ops-clickable-row"
                  :empty-text="$t('opsConsole.environments.empty')"
                  @row-click="openEnvironment"
                >
                  <el-table-column
                    :label="$t('opsConsole.environments.environment')"
                    min-width="200"
                  >
                    <template #default="{ row }">
                      <div class="ops-name-cell">
                        <span class="ops-env-mark">{{ row.name.slice(0, 1) }}</span>
                        <span>
                          <span class="ops-name-title">
                            <strong>{{ row.name }}</strong>
                            <el-tag v-if="row.isDefault" size="small" effect="plain">{{
                              $t('opsConsole.environments.defaultTag')
                            }}</el-tag>
                          </span>
                          <small>{{
                            $t('opsConsole.environments.nodeCount', { count: row.nodeCount })
                          }}</small>
                        </span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('opsConsole.environments.nodes')" min-width="105">
                    <template #default="{ row }">
                      <strong>{{ row.onlineNodes }} / {{ row.nodeCount }}</strong>
                      <small class="ops-inline-note">{{
                        $t('opsConsole.environments.onlineSuffix')
                      }}</small>
                    </template>
                  </el-table-column>
                  <el-table-column
                    :label="$t('opsConsole.environments.foundations')"
                    min-width="105"
                  >
                    <template #default="{ row }">
                      <template v-if="row.foundationTotal">
                        <strong>{{ row.foundationHealthy }} / {{ row.foundationTotal }}</strong>
                        <small class="ops-inline-note">{{
                          $t('opsConsole.environments.healthySuffix')
                        }}</small>
                      </template>
                      <span v-else class="ops-muted">{{ $t('opsConsole.common.undeployed') }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column
                    :label="$t('opsConsole.environments.deployments')"
                    min-width="100"
                  >
                    <template #default="{ row }">
                      <strong>{{ row.projectCount }}</strong>
                      <small class="ops-inline-note">{{ $t('opsConsole.common.running') }}</small>
                    </template>
                  </el-table-column>
                  <el-table-column
                    :label="$t('opsConsole.environments.recentChange')"
                    min-width="220"
                  >
                    <template #default="{ row }">
                      <div class="ops-primary-cell">
                        <strong>{{ row.recentChange }}</strong>
                        <small>{{ row.recentAt }} · {{ row.recentBy }}</small>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('opsConsole.common.status')" width="92">
                    <template #default="{ row }">
                      <span class="ops-status" :class="`is-${row.status}`">
                        {{ environmentStatusLabel(row.status) }}
                      </span>
                    </template>
                  </el-table-column>
                  <el-table-column width="92" fixed="right">
                    <template #default="{ row }">
                      <el-button link type="primary">{{
                        environmentActionLabel(row.status)
                      }}</el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
            <div class="ck-pagination-bar">
              <WorkbenchPagination
                :page-size-label="$t('opsConsole.pagination.perPage')"
                v-model:page="environmentPage.page"
                v-model:limit="environmentPage.limit"
                :total="environmentPage.total"
                :total-pages="environmentTotalPages"
                :summary="environmentPaginationSummary"
                :page-indicator="environmentPaginationIndicator"
              />
            </div>
          </div>
        </div>

        <div v-else-if="selectedEnvironment" class="ops-page ops-environment-detail">
          <div class="ops-detail-header">
            <div class="ops-detail-toolbar">
              <div class="ops-detail-heading">
                <OpsSectionSwitcher
                  :model-value="activeTab"
                  :can-administer-operations="canAdministerOperations"
                  @update:model-value="switchSection"
                />
                <span class="ops-toolbar__divider" />
                <el-select
                  v-if="environmentOptionTotal > 1"
                  v-model="selectedEnvironmentId"
                  class="ops-environment-switcher"
                  :aria-label="$t('opsConsole.environments.switchEnvironment')"
                  @change="selectEnvironment"
                >
                  <el-option
                    v-for="environment in environmentOptions"
                    :key="environment.id"
                    :label="environment.name"
                    :value="environment.id"
                  />
                </el-select>
                <div class="ops-detail-heading__meta">
                  <div class="ops-detail-title">
                    <h2>{{ environmentHeading }}</h2>
                    <span class="ops-status" :class="`is-${selectedEnvironment.status}`">
                      {{ environmentStatusLabel(selectedEnvironment.status) }}
                    </span>
                  </div>
                  <small>{{
                    $t('opsConsole.environments.lastSync', { time: selectedEnvironment.recentAt })
                  }}</small>
                </div>
              </div>
              <div class="ops-detail-actions">
                <el-button :loading="refreshing" @click="runEnvironmentCheck">{{
                  $t('opsConsole.environments.recheck')
                }}</el-button>
                <el-button
                  :disabled="selectedEnvironment.status === 'deleting'"
                  @click="openEnvironmentNodeDialog"
                  >{{ $t('opsConsole.environments.addNode') }}</el-button
                >
                <el-dropdown @command="handleEnvironmentCommand">
                  <el-button :aria-label="$t('opsConsole.environments.more')"
                    >{{ $t('opsConsole.environments.more')
                    }}<el-icon class="el-icon--right"><ArrowDown /></el-icon
                  ></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item
                        command="edit"
                        :disabled="selectedEnvironment.status === 'deleting'"
                        >{{ $t('opsConsole.environments.edit') }}</el-dropdown-item
                      >
                      <el-dropdown-item command="manage" divided>{{
                        $t('opsConsole.environments.manageScopes')
                      }}</el-dropdown-item>
                      <el-dropdown-item
                        command="delete"
                        :disabled="
                          selectedEnvironment.status === 'deleting' || selectedEnvironment.isDefault
                        "
                        >{{
                          selectedEnvironment.isDefault
                            ? $t('opsConsole.environments.defaultProtected')
                            : $t('opsConsole.environments.delete')
                        }}</el-dropdown-item
                      >
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </div>

            <div class="ops-summary-strip">
              <div>
                <span>{{ $t('opsConsole.environments.nodes') }}</span
                ><strong
                  >{{ selectedEnvironment.onlineNodes }} / {{ selectedEnvironment.nodeCount }}
                  {{ $t('opsConsole.common.online') }}</strong
                >
              </div>
              <div>
                <span>{{ $t('opsConsole.environments.foundations') }}</span>
                <strong v-if="selectedEnvironment.foundationTotal"
                  >{{ selectedEnvironment.foundationHealthy }} /
                  {{ selectedEnvironment.foundationTotal }}
                  {{ $t('opsConsole.common.healthy') }}</strong
                >
                <strong v-else>{{ $t('opsConsole.common.undeployed') }}</strong>
              </div>
              <div>
                <span>{{ $t('opsConsole.environments.deployments') }}</span
                ><strong>{{
                  $t('opsConsole.environments.runningCount', {
                    count: selectedEnvironment.projectCount,
                  })
                }}</strong>
              </div>
              <div>
                <span>{{ $t('opsConsole.environments.nodeDisk') }}</span
                ><strong v-if="selectedEnvironment.nodeCount">{{
                  $t('opsConsole.environments.maxDisk', { value: environmentMaxDiskUsage })
                }}</strong
                ><strong v-else>{{ $t('opsConsole.environments.noNodes') }}</strong>
              </div>
            </div>

            <div
              class="ops-detail-switcher"
              :aria-label="$t('opsConsole.environments.environment')"
            >
              <button
                v-for="item in environmentDetailSections"
                :key="item.value"
                type="button"
                :class="{ 'is-active': environmentDetailTab === item.value }"
                @click="environmentDetailTab = item.value"
              >
                {{ item.label }}
              </button>
            </div>
          </div>

          <div class="ops-detail-content">
            <section
              v-if="environmentDetailTab === 'overview'"
              class="ops-detail-pane ops-detail-pane--overview"
            >
              <section
                v-if="selectedEnvironment.status === 'deleting'"
                class="ops-guidance ops-guidance--warning"
              >
                <div>
                  <span class="ops-guidance__eyebrow">{{ $t('opsConsole.common.deleting') }}</span>
                  <h3>{{ $t('opsConsole.environments.deletingTitle') }}</h3>
                  <p>{{ $t('opsConsole.environments.deletingDescription') }}</p>
                </div>
                <el-button :loading="refreshing" @click="runEnvironmentCheck">{{
                  $t('opsConsole.environments.recheck')
                }}</el-button>
              </section>
              <section
                v-else-if="selectedEnvironment.status === 'uninitialized'"
                class="ops-guidance ops-guidance--setup"
              >
                <div class="ops-guidance__heading">
                  <div>
                    <span class="ops-guidance__eyebrow">{{
                      $t('opsConsole.environments.setup')
                    }}</span>
                    <h3>{{ environmentSetupTitle }}</h3>
                  </div>
                  <el-button type="primary" @click="continueEnvironmentSetup">{{
                    environmentSetupAction
                  }}</el-button>
                </div>
                <ol class="ops-setup-steps">
                  <li :class="selectedEnvironment.nodeCount ? 'is-complete' : 'is-current'">
                    <span>1</span><strong>{{ $t('opsConsole.environments.associateNodes') }}</strong
                    ><small>{{
                      selectedEnvironment.nodeCount
                        ? $t('opsConsole.environments.completed')
                        : $t('opsConsole.environments.pending')
                    }}</small>
                  </li>
                  <li
                    :class="{
                      'is-current':
                        selectedEnvironment.nodeCount && !selectedEnvironment.foundationHealthy,
                      'is-complete':
                        selectedEnvironment.foundationHealthy === foundationDefinitions.length,
                    }"
                  >
                    <span>2</span
                    ><strong>{{ $t('opsConsole.environments.deployFoundation') }}</strong
                    ><small>{{
                      selectedEnvironment.foundationTotal
                        ? $t('opsConsole.environments.inProgress')
                        : $t('opsConsole.environments.pending')
                    }}</small>
                  </li>
                  <li>
                    <span>3</span><strong>{{ $t('opsConsole.environments.executeCheck') }}</strong
                    ><small>{{ $t('opsConsole.environments.notStarted') }}</small>
                  </li>
                  <li>
                    <span>4</span
                    ><strong>{{ $t('opsConsole.environments.environmentAvailable') }}</strong
                    ><small>{{ $t('opsConsole.environments.notStarted') }}</small>
                  </li>
                </ol>
              </section>

              <section
                v-else-if="selectedEnvironment.status === 'attention' && foundationDeploying"
                class="ops-guidance"
              >
                <div>
                  <span class="ops-guidance__eyebrow">{{
                    $t('opsConsole.environments.deploymentProgress', {
                      healthy: selectedEnvironment.foundationHealthy,
                      total: selectedEnvironment.foundationTotal,
                    })
                  }}</span>
                  <h3>{{ $t('opsConsole.environments.deploymentInProgressTitle') }}</h3>
                  <p>{{ $t('opsConsole.environments.deploymentInProgressDesc') }}</p>
                </div>
                <el-button type="primary" @click="environmentDetailTab = 'services'">{{
                  $t('opsConsole.environments.viewFoundation')
                }}</el-button>
              </section>

              <section
                v-else-if="selectedEnvironment.status === 'attention'"
                class="ops-guidance ops-guidance--warning"
              >
                <div>
                  <span class="ops-guidance__eyebrow">{{
                    $t('opsConsole.environments.issueCount', { count: environmentProblems.length })
                  }}</span>
                  <h3>{{ environmentProblemTitle }}</h3>
                  <p>{{ environmentProblemDescription }}</p>
                </div>
                <div class="ops-guidance__actions">
                  <el-button @click="environmentDetailTab = 'events'">{{
                    $t('opsConsole.environments.viewEvents')
                  }}</el-button>
                  <el-button type="primary" @click="environmentDetailTab = 'services'">{{
                    $t('opsConsole.environments.handleAbnormal')
                  }}</el-button>
                </div>
              </section>

              <div class="ops-overview-layout">
                <section class="ops-section">
                  <header>
                    <h3>{{ $t('opsConsole.environments.foundationStatus') }}</h3>
                    <el-button link @click="environmentDetailTab = 'services'">{{
                      $t('opsConsole.environments.viewDistribution')
                    }}</el-button>
                  </header>
                  <div
                    v-for="group in foundationServiceGroups"
                    :key="group.name"
                    class="ops-health-row"
                  >
                    <span
                      ><strong>{{ group.name }}</strong
                      ><small>{{ group.description }}</small></span
                    >
                    <span class="ops-muted">{{ group.location }}</span>
                    <span class="ops-status" :class="`is-${group.state}`">{{ group.label }}</span>
                  </div>
                </section>
                <div class="ops-overview-secondary">
                  <section class="ops-section ops-node-section">
                    <header>
                      <h3>{{ $t('opsConsole.environments.nodeLoad') }}</h3>
                      <el-button link @click="environmentDetailTab = 'nodes'">{{
                        $t('opsConsole.environments.viewAllNodes')
                      }}</el-button>
                    </header>
                    <div class="ops-node-grid">
                      <article v-for="node in environmentNodeRows" :key="node.name">
                        <div>
                          <strong>{{ node.name }}</strong
                          ><span
                            class="ops-status"
                            :class="
                              node.observedStatus === 'online' ? 'is-available' : 'is-uninitialized'
                            "
                            >{{
                              node.observedStatus === 'online'
                                ? $t('opsConsole.common.online')
                                : $t('opsConsole.common.offline')
                            }}</span
                          >
                        </div>
                        <dl class="ops-node-metrics">
                          <div>
                            <dt>CPU</dt>
                            <dd>{{ node.cpu }}%</dd>
                          </div>
                          <div>
                            <dt>{{ $t('opsManagement.memory') }}</dt>
                            <dd>{{ node.memory }}%</dd>
                          </div>
                          <div>
                            <dt>{{ $t('opsConsole.nodes.disk') }}</dt>
                            <dd>{{ node.diskUsage }}%</dd>
                          </div>
                        </dl>
                      </article>
                    </div>
                  </section>
                  <section class="ops-section ops-recent-section">
                    <header>
                      <h3>{{ $t('opsConsole.environments.recentEvents') }}</h3>
                      <el-button link @click="environmentDetailTab = 'events'">{{
                        $t('opsConsole.environments.allEvents')
                      }}</el-button>
                    </header>
                    <div
                      v-for="event in selectedEnvironmentEvents.slice(0, 3)"
                      :key="event.time + event.name"
                      class="ops-event-row"
                    >
                      <span class="ops-event-dot" :class="{ 'is-warning': !event.success }" />
                      <span
                        ><strong>{{ event.name }}</strong
                        ><small>{{ event.target }}</small></span
                      >
                      <small>{{ event.time }}</small>
                    </div>
                  </section>
                </div>
              </div>
            </section>
            <section
              v-else-if="environmentDetailTab === 'nodes'"
              class="ops-detail-pane ops-detail-pane--table"
            >
              <div class="ops-detail-list ck-content-area">
                <div class="ck-content-scroll">
                  <div class="ck-table-shell">
                    <el-table
                      :data="pagedEnvironmentNodes"
                      :empty-text="$t('opsConsole.nodes.emptyEnvironment')"
                    >
                      <el-table-column :label="$t('opsConsole.nodes.node')" min-width="220">
                        <template #default="{ row }"
                          ><div class="ops-primary-cell">
                            <strong>{{ row.name }}</strong
                            ><small>{{ row.address }}</small>
                          </div></template
                        >
                      </el-table-column>
                      <el-table-column label="CPU" width="100"
                        ><template #default="{ row }">{{ row.cpu }}%</template></el-table-column
                      >
                      <el-table-column :label="$t('opsManagement.memory')" width="100"
                        ><template #default="{ row }">{{ row.memory }}%</template></el-table-column
                      >
                      <el-table-column :label="$t('opsConsole.nodes.disk')" min-width="150"
                        ><template #default="{ row }">{{
                          $t('opsConsole.nodes.diskUsed', { value: `${row.diskUsage}%` })
                        }}</template></el-table-column
                      >
                      <el-table-column
                        :label="$t('opsConsole.nodes.agentVersion')"
                        width="130"
                        prop="agentVersion"
                      />
                      <el-table-column :label="$t('opsConsole.nodes.type')" width="130">
                        <template #default="{ row }">{{ row.roleLabel }}</template>
                      </el-table-column>
                      <el-table-column :label="$t('opsConsole.nodes.foundation')" width="120">
                        <template #default="{ row }">
                          <span class="ops-status" :class="row.clusterStatusClass">{{
                            row.clusterStatusLabel
                          }}</span>
                        </template>
                      </el-table-column>
                      <el-table-column :label="$t('opsConsole.nodes.timeSync')" width="150">
                        <template #default="{ row }">
                          <el-tooltip :content="row.timeSyncDetail" placement="top">
                            <span class="ops-status" :class="row.timeSyncClass">{{
                              row.timeSyncLabel
                            }}</span>
                          </el-tooltip>
                        </template>
                      </el-table-column>
                      <el-table-column :label="$t('opsConsole.nodes.stateDetail')" min-width="190">
                        <template #default="{ row }">{{ row.clusterMessage || '—' }}</template>
                      </el-table-column>
                      <el-table-column
                        :label="$t('opsConsole.nodes.lastHeartbeat')"
                        width="160"
                        prop="lastHeartbeatAt"
                      />
                      <el-table-column :label="$t('opsConsole.common.status')" width="110"
                        ><template #default="{ row }"
                          ><span
                            class="ops-status"
                            :class="
                              row.observedStatus === 'online' ? 'is-available' : 'is-attention'
                            "
                            >{{
                              row.observedStatus === 'online'
                                ? $t('opsConsole.common.online')
                                : $t('opsConsole.common.offline')
                            }}</span
                          ></template
                        ></el-table-column
                      >
                      <el-table-column
                        :label="$t('opsConsole.common.actions')"
                        width="90"
                        fixed="right"
                      >
                        <template #default="{ row }">
                          <el-tooltip
                            :content="row.unassignReason"
                            :disabled="!row.unassignDisabled"
                            placement="top"
                          >
                            <span class="ops-action-trigger">
                              <el-button
                                link
                                type="danger"
                                :disabled="row.unassignDisabled"
                                @click="removeEnvironmentNode(row.id, row.name)"
                                >{{ $t('opsConsole.nodes.unassign') }}</el-button
                              >
                            </span>
                          </el-tooltip>
                        </template>
                      </el-table-column>
                    </el-table>
                  </div>
                </div>
                <div class="ck-pagination-bar">
                  <WorkbenchPagination
                    :page-size-label="$t('opsConsole.pagination.perPage')"
                    v-model:page="environmentNodePage.page"
                    v-model:limit="environmentNodePage.limit"
                    :total="environmentNodePage.total"
                    :total-pages="environmentNodeTotalPages"
                    :summary="environmentNodePaginationSummary"
                    :page-indicator="environmentNodePaginationIndicator"
                  />
                </div>
              </div>
            </section>
            <section
              v-else-if="environmentDetailTab === 'services'"
              class="ops-detail-pane ops-detail-pane--services"
            >
              <div class="ops-sub-toolbar">
                <span class="ops-muted">{{ $t('opsConsole.foundation.strategy') }}</span>
                <div class="ops-toolbar__actions">
                  <template v-if="foundationAdjusting">
                    <span class="ops-muted">{{
                      $t('opsConsole.foundation.changedCount', { count: foundationChangedCount })
                    }}</span>
                    <el-button @click="cancelFoundationAdjustment">{{
                      $t('common.cancel')
                    }}</el-button>
                    <el-button
                      type="primary"
                      :disabled="!foundationChangedCount"
                      @click="applyFoundationAdjustment"
                      >{{ $t('opsConsole.foundation.checkAndApply') }}</el-button
                    >
                  </template>
                  <template v-else>
                    <el-button
                      v-if="selectedEnvironment.foundationTotal"
                      :disabled="foundationDeployDisabled"
                      @click="startFoundationAdjustment"
                      >{{ $t('opsConsole.foundation.adjustDistribution') }}</el-button
                    >
                    <el-button
                      type="primary"
                      :disabled="foundationDeployDisabled"
                      @click="openFoundationDialog"
                      >{{
                        selectedEnvironment.foundationTotal
                          ? $t('opsConsole.foundation.repair')
                          : $t('opsConsole.environments.deployFoundation')
                      }}</el-button
                    >
                  </template>
                </div>
              </div>
              <div class="ops-detail-table ck-table-shell">
                <el-table :data="foundationRows" :empty-text="$t('opsConsole.foundation.empty')">
                  <el-table-column
                    :label="$t('opsConsole.foundation.service')"
                    min-width="230"
                    prop="name"
                  />
                  <el-table-column
                    :label="$t('opsConsole.foundation.runtimeNode')"
                    min-width="190"
                    prop="nodeName"
                  >
                    <template #default="{ row }">
                      <el-select
                        v-if="foundationAdjusting"
                        v-model="foundationDraft[row.type]"
                        size="small"
                        @change="syncFoundationDraft(row.type)"
                      >
                        <el-option
                          v-for="node in environmentNodes"
                          :key="node.id"
                          :label="node.displayName"
                          :value="node.id"
                        />
                      </el-select>
                      <span v-else>{{ row.nodeName }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('opsConsole.foundation.strategyColumn')" width="120">
                    <template #default="{ row }">{{
                      row.instances ? $t('opsConsole.foundation.single') : '—'
                    }}</template>
                  </el-table-column>
                  <el-table-column :label="$t('opsConsole.common.status')" width="120">
                    <template #default="{ row }">
                      <span class="ops-status" :class="`is-${row.state}`">{{ row.label }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('opsConsole.foundation.stateDetail')" min-width="220">
                    <template #default="{ row }">
                      <span :class="row.message ? 'ops-error-detail' : 'ops-muted'">{{
                        row.message || '—'
                      }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column
                    :label="$t('opsConsole.foundation.lastCheck')"
                    width="150"
                    prop="checkedAt"
                  />
                </el-table>
              </div>
            </section>
            <section v-else class="ops-detail-pane ops-detail-pane--table">
              <div class="ops-detail-list ck-content-area">
                <div class="ck-content-scroll">
                  <div class="ck-table-shell">
                    <el-table
                      :data="pagedEnvironmentEvents"
                      :empty-text="$t('opsConsole.events.empty')"
                    >
                      <el-table-column
                        :label="$t('opsConsole.events.time')"
                        width="160"
                        prop="time"
                      />
                      <el-table-column
                        :label="$t('opsConsole.events.event')"
                        min-width="220"
                        prop="name"
                      />
                      <el-table-column
                        :label="$t('opsConsole.events.target')"
                        min-width="220"
                        prop="target"
                      />
                      <el-table-column :label="$t('opsConsole.events.detail')" min-width="260">
                        <template #default="{ row }">{{ row.message || '—' }}</template>
                      </el-table-column>
                      <el-table-column :label="$t('opsConsole.events.result')" width="120">
                        <template #default="{ row }">
                          <span
                            class="ops-status"
                            :class="row.success ? 'is-available' : 'is-attention'"
                            >{{ row.result }}</span
                          >
                        </template>
                      </el-table-column>
                      <el-table-column
                        :label="$t('opsConsole.events.operator')"
                        width="130"
                        prop="operator"
                      />
                    </el-table>
                  </div>
                </div>
                <div class="ck-pagination-bar">
                  <WorkbenchPagination
                    :page-size-label="$t('opsConsole.pagination.perPage')"
                    v-model:page="environmentEventPage.page"
                    v-model:limit="environmentEventPage.limit"
                    :total="environmentEventPage.total"
                    :total-pages="environmentEventTotalPages"
                    :summary="environmentEventPaginationSummary"
                    :page-indicator="environmentEventPaginationIndicator"
                  />
                </div>
              </div>
            </section>
          </div>
        </div>
        <div v-else class="ops-page">
          <div class="ops-toolbar">
            <OpsSectionSwitcher
              :model-value="activeTab"
              :can-administer-operations="canAdministerOperations"
              @update:model-value="switchSection"
            />
            <el-button :loading="refreshing" @click="loadDashboard">{{
              $t('opsConsole.common.refresh')
            }}</el-button>
          </div>
          <div class="ops-empty-panel ck-content-area">
            <el-empty :description="$t('opsConsole.environments.initializingDefault')" />
          </div>
        </div>
      </div>

      <div v-else-if="canAdministerOperations && activeTab === 'nodes'" class="ops-view">
        <div class="ops-page">
          <h2 class="sr-only">{{ $t('opsConsole.sections.nodes') }}</h2>
          <div class="ops-toolbar">
            <div class="ops-toolbar__filters">
              <OpsSectionSwitcher
                :model-value="activeTab"
                :can-administer-operations="canAdministerOperations"
                @update:model-value="switchSection"
              />
              <span class="ops-toolbar__divider" />
              <el-input
                v-model="nodePage.keyword"
                clearable
                :placeholder="$t('opsConsole.nodes.searchPlaceholder')"
                @keyup.enter="searchNodes"
                @clear="searchNodes"
              />
              <el-select v-model="nodeTypeFilter">
                <el-option :label="$t('opsConsole.nodes.allTypes')" value="all" />
                <el-option :label="$t('opsConsole.nodes.standard')" value="linux" />
                <el-option :label="$t('opsConsole.nodes.windowsCollector')" value="windows" />
              </el-select>
            </div>
            <div class="ops-toolbar__actions">
              <span class="ops-count">{{
                $t('opsConsole.nodes.total', { count: nodePage.total })
              }}</span>
              <el-button @click="packageDialog = true">{{
                $t('opsConsole.nodes.packages')
              }}</el-button>
              <el-button @click="enrollmentDrawer = true">{{
                $t('opsConsole.nodes.requests')
              }}</el-button>
              <el-button type="primary" @click="openEnrollmentDialog">{{
                $t('opsConsole.nodes.enroll')
              }}</el-button>
            </div>
          </div>
          <div class="ops-list-panel ck-content-area">
            <div class="ck-content-scroll">
              <div class="ck-table-shell">
                <el-table
                  :data="filteredNodes"
                  v-loading="loading.nodes"
                  :empty-text="$t('opsConsole.nodes.empty')"
                >
                  <el-table-column :label="$t('opsConsole.nodes.node')" min-width="220"
                    ><template #default="{ row }"
                      ><div class="ops-primary-cell">
                        <strong>{{ row.name }}</strong
                        ><small>{{ row.hostname || $t('opsConsole.nodes.hostPending') }}</small>
                      </div></template
                    ></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.nodes.type')" min-width="150"
                    ><template #default="{ row }">{{
                      nodeTypeLabel(row)
                    }}</template></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.nodes.environment')" min-width="180"
                    ><template #default="{ row, $index }">{{
                      nodeEnvironmentName(row, $index)
                    }}</template></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.nodes.resource')" min-width="210"
                    ><template #default="{ row }"
                      >CPU {{ resourcePercent(row, 'cpu') }} · {{ $t('opsManagement.memory') }}
                      {{ resourcePercent(row, 'memory') }}</template
                    ></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.nodes.disk')" min-width="170"
                    ><template #default="{ row }"
                      ><div class="ops-primary-cell">
                        <strong>{{
                          $t('opsConsole.nodes.diskUsed', { value: resourcePercent(row, 'disk') })
                        }}</strong
                        ><small>{{ $t('opsConsole.nodes.diskHint') }}</small>
                      </div></template
                    ></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.nodes.agentVersion')" width="130"
                    ><template #default="{ row }">{{
                      row.agentVersion || $t('opsConsole.common.notReported')
                    }}</template></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.nodes.connection')" width="120"
                    ><template #default="{ row }"
                      ><span
                        class="ops-status"
                        :class="
                          row.observedStatus === 'online' ? 'is-available' : 'is-uninitialized'
                        "
                        >{{ lifecyclePresentation(row.observedStatus) }}</span
                      ></template
                    ></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.nodes.timeSync')" width="150">
                    <template #default="{ row }">
                      <el-tooltip :content="timeSyncPresentation(row).detail" placement="top">
                        <span class="ops-status" :class="timeSyncPresentation(row).className">{{
                          timeSyncPresentation(row).label
                        }}</span>
                      </el-tooltip>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('opsConsole.nodes.lastHeartbeat')" min-width="165">
                    <template #default="{ row }">{{
                      row.lastHeartbeatAt
                        ? formatTime(row.lastHeartbeatAt)
                        : $t('opsConsole.common.notReported')
                    }}</template>
                  </el-table-column>
                  <el-table-column
                    :label="$t('opsConsole.common.actions')"
                    width="110"
                    fixed="right"
                  >
                    <template #default="{ row }">
                      <span v-if="row.nodeKind === 'center'" class="ops-muted">{{
                        $t('opsConsole.nodes.builtIn')
                      }}</span>
                      <el-button
                        v-else-if="row.platform === 'linux'"
                        link
                        type="danger"
                        :disabled="row.clusterDesiredAction === 'removing'"
                        @click="removePhysicalNode(row)"
                        >{{
                          row.clusterDesiredAction === 'removing'
                            ? $t('opsConsole.nodes.removing')
                            : $t('opsConsole.nodes.remove')
                        }}</el-button
                      >
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
            <div class="ck-pagination-bar">
              <WorkbenchPagination
                :page-size-label="$t('opsConsole.pagination.perPage')"
                v-model:page="nodePage.page"
                v-model:limit="nodePage.limit"
                :total="nodePage.total"
                :total-pages="pageTotal(nodePage.total, nodePage.limit)"
                :summary="paginationSummary(nodePage.page, nodePage.limit, nodePage.total)"
                :page-indicator="paginationIndicator(nodePage.page, nodePage.limit, nodePage.total)"
                @change="loadNodes"
              />
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="activeTab === 'deployments'" class="ops-view">
        <div class="ops-page">
          <h2 class="sr-only">{{ $t('opsConsole.sections.deployments') }}</h2>
          <div class="ops-toolbar">
            <div class="ops-toolbar__filters">
              <OpsSectionSwitcher
                :model-value="activeTab"
                :can-administer-operations="canAdministerOperations"
                @update:model-value="switchSection"
              />
              <span class="ops-toolbar__divider" />
              <el-input
                v-model="deploymentPage.keyword"
                clearable
                :placeholder="$t('opsConsole.deployments.searchPlaceholder')"
                @keyup.enter="searchDeployments"
                @clear="searchDeployments"
              />
              <el-select v-model="deploymentEnvironmentFilter">
                <el-option :label="$t('opsConsole.deployments.allEnvironments')" value="all" />
                <el-option
                  v-for="environment in environments"
                  :key="environment.id"
                  :label="environment.name"
                  :value="environment.id"
                />
              </el-select>
            </div>
            <div class="ops-toolbar__actions">
              <span class="ops-count">{{
                $t('opsConsole.deployments.total', { count: deploymentDisplayTotal })
              }}</span>
              <el-button type="primary" @click="openDeployDialog()">{{
                $t('opsConsole.deployments.create')
              }}</el-button>
            </div>
          </div>
          <div class="ops-list-panel ck-content-area">
            <div class="ck-content-scroll">
              <div class="ck-table-shell">
                <el-table
                  :data="filteredDeploymentRows"
                  v-loading="loading.deployments"
                  :empty-text="$t('opsConsole.deployments.empty')"
                >
                  <el-table-column
                    :label="$t('opsConsole.deployments.projectVersion')"
                    min-width="230"
                    ><template #default="{ row }"
                      ><div class="ops-primary-cell">
                        <strong>{{ row.projectName }}</strong
                        ><small>{{ row.version }}</small>
                      </div></template
                    ></el-table-column
                  >
                  <el-table-column
                    :label="$t('opsConsole.deployments.environment')"
                    min-width="180"
                    prop="environmentName"
                  />
                  <el-table-column
                    :label="$t('opsConsole.deployments.runtimeEngines')"
                    min-width="280"
                    ><template #default="{ row }"
                      ><div class="ops-engine-tags">
                        <el-tag
                          v-for="service in row.services"
                          :key="service"
                          size="small"
                          effect="plain"
                          >{{ service }}</el-tag
                        >
                      </div></template
                    ></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.deployments.state')" width="130"
                    ><template #default="{ row }"
                      ><span class="ops-status" :class="row.statusClass">{{
                        row.status
                      }}</span></template
                    ></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.deployments.latest')" min-width="190"
                    ><template #default="{ row }"
                      ><div class="ops-primary-cell">
                        <strong>{{ row.updatedAt }}</strong
                        ><small>{{ row.status }}</small>
                      </div></template
                    ></el-table-column
                  >
                  <el-table-column
                    :label="$t('opsConsole.common.actions')"
                    width="100"
                    fixed="right"
                    ><template #default="{ row }"
                      ><el-button link type="primary" @click="openDeploymentRow(row)">{{
                        $t('opsConsole.common.details')
                      }}</el-button></template
                    ></el-table-column
                  >
                </el-table>
              </div>
            </div>
            <div class="ck-pagination-bar">
              <WorkbenchPagination
                :page-size-label="$t('opsConsole.pagination.perPage')"
                v-model:page="deploymentPage.page"
                v-model:limit="deploymentPage.limit"
                :total="deploymentDisplayTotal"
                :total-pages="pageTotal(deploymentDisplayTotal, deploymentPage.limit)"
                :summary="
                  paginationSummary(
                    deploymentPage.page,
                    deploymentPage.limit,
                    deploymentDisplayTotal,
                  )
                "
                :page-indicator="
                  paginationIndicator(
                    deploymentPage.page,
                    deploymentPage.limit,
                    deploymentDisplayTotal,
                  )
                "
                @change="loadDeployments"
              />
            </div>
          </div>
        </div>
      </div>

      <div v-else class="ops-view">
        <div class="ops-page">
          <h2 class="sr-only">{{ $t('opsConsole.sections.tasks') }}</h2>
          <div class="ops-toolbar">
            <div class="ops-toolbar__filters">
              <OpsSectionSwitcher
                :model-value="activeTab"
                :can-administer-operations="canAdministerOperations"
                @update:model-value="switchSection"
              />
              <span class="ops-toolbar__divider" />
              <el-input
                v-model="taskKeyword"
                clearable
                :placeholder="$t('opsConsole.tasks.searchPlaceholder')"
              />
              <el-select v-model="taskTypeFilter">
                <el-option :label="$t('opsConsole.tasks.allTypes')" value="all" />
                <el-option
                  :label="$t('opsConsole.tasks.environmentInit')"
                  value="environment-init"
                />
                <el-option
                  :label="$t('opsConsole.tasks.foundationDeploy')"
                  value="foundation-deploy"
                />
                <el-option :label="$t('opsConsole.tasks.projectDeploy')" value="project-deploy" />
              </el-select>
            </div>
            <div class="ops-toolbar__actions">
              <span class="ops-count">{{
                $t('opsConsole.tasks.summary', { active: activeTaskCount, failed: failedTaskCount })
              }}</span
              ><el-button @click="loadDashboard">{{ $t('opsConsole.common.refresh') }}</el-button>
            </div>
          </div>
          <div class="ops-list-panel ck-content-area">
            <div class="ck-content-scroll">
              <div class="ck-table-shell">
                <el-table :data="pagedTasks" :empty-text="$t('opsConsole.tasks.empty')">
                  <el-table-column :label="$t('opsConsole.tasks.task')" min-width="240"
                    ><template #default="{ row }"
                      ><div class="ops-primary-cell">
                        <strong>{{ row.name }}</strong
                        ><small>{{ taskTypeLabel(row.type) }}</small>
                      </div></template
                    ></el-table-column
                  >
                  <el-table-column
                    :label="$t('opsConsole.tasks.target')"
                    min-width="200"
                    prop="target"
                  />
                  <el-table-column :label="$t('opsConsole.tasks.currentStep')" min-width="250"
                    ><template #default="{ row }"
                      ><el-progress
                        v-if="row.status === 'active'"
                        :percentage="row.progress"
                      /><span v-else>{{ row.step }}</span></template
                    ></el-table-column
                  >
                  <el-table-column :label="$t('opsConsole.common.status')" width="120"
                    ><template #default="{ row }"
                      ><span
                        class="ops-status"
                        :class="
                          row.status === 'completed'
                            ? 'is-available'
                            : row.status === 'failed'
                              ? 'is-attention'
                              : 'is-running'
                        "
                        >{{ taskStatusLabel(row.status) }}</span
                      ></template
                    ></el-table-column
                  >
                  <el-table-column
                    :label="$t('opsConsole.tasks.initiator')"
                    width="130"
                    prop="operator"
                  />
                  <el-table-column
                    :label="$t('opsConsole.tasks.startedAt')"
                    width="150"
                    prop="startedAt"
                  />
                  <el-table-column
                    :label="$t('opsConsole.common.actions')"
                    width="120"
                    fixed="right"
                    ><template #default
                      ><el-button link type="primary">{{
                        $t('opsConsole.tasks.viewRecord')
                      }}</el-button></template
                    ></el-table-column
                  >
                </el-table>
              </div>
            </div>
            <div class="ck-pagination-bar">
              <WorkbenchPagination
                :page-size-label="$t('opsConsole.pagination.perPage')"
                v-model:page="taskPage.page"
                v-model:limit="taskPage.limit"
                :total="filteredTasks.length"
                :total-pages="taskTotalPages"
                :summary="taskPaginationSummary"
                :page-indicator="taskPaginationIndicator"
              />
            </div>
          </div>
        </div>
      </div>
    </main>

    <el-dialog
      v-model="environmentDialog"
      :title="
        environmentDialogMode === 'edit'
          ? $t('opsConsole.environments.edit')
          : $t('opsConsole.environments.create')
      "
      width="440px"
    >
      <el-form label-position="top">
        <el-form-item :label="$t('opsConsole.environments.createName')" required>
          <el-input
            v-model="environmentForm.name"
            maxlength="80"
            show-word-limit
            autofocus
            :placeholder="$t('opsConsole.environments.nameExample')"
            @keyup.enter="saveEnvironment"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="environmentDialog = false">{{
          $t('opsConsole.common.cancel')
        }}</el-button>
        <el-button type="primary" :loading="savingEnvironment" @click="saveEnvironment">{{
          environmentDialogMode === 'edit'
            ? $t('opsConsole.environments.saveEnvironment')
            : $t('opsConsole.environments.createEnvironment')
        }}</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="environmentDeleteDialog"
      :title="$t('opsConsole.environments.deleteTitle')"
      width="520px"
    >
      <el-alert
        type="warning"
        :closable="false"
        :title="$t('opsConsole.environments.deleteWarning')"
      />
      <div v-if="selectedEnvironment" class="ops-delete-impact">
        <div>
          <span>{{ $t('opsConsole.environments.nodes') }}</span
          ><strong>{{ selectedEnvironment.nodeCount }}</strong>
        </div>
        <div>
          <span>{{ $t('opsConsole.environments.foundations') }}</span
          ><strong>{{ selectedEnvironment.foundationTotal }}</strong>
        </div>
        <div>
          <span>{{ $t('opsConsole.environments.deployments') }}</span
          ><strong>{{ selectedEnvironment.projectCount }}</strong>
        </div>
      </div>
      <el-form label-position="top">
        <el-form-item
          :label="
            $t('opsConsole.environments.deleteConfirmLabel', {
              name: selectedEnvironment?.name || '',
            })
          "
          required
        >
          <el-input v-model="environmentDeleteConfirmation" autocomplete="off" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="environmentDeleteDialog = false">{{
          $t('opsConsole.common.cancel')
        }}</el-button>
        <el-button
          type="danger"
          :loading="deletingEnvironment"
          :disabled="environmentDeleteConfirmation !== selectedEnvironment?.name"
          @click="deleteEnvironment"
          >{{ $t('opsConsole.environments.confirmDelete') }}</el-button
        >
      </template>
    </el-dialog>

    <el-dialog
      v-model="environmentNodeDialog"
      :title="
        $t('opsConsole.nodes.addTitle', {
          name: selectedEnvironment?.name || $t('opsConsole.sections.environments'),
        })
      "
      width="560px"
    >
      <el-form label-position="top">
        <el-form-item :label="$t('opsConsole.nodes.node')" required>
          <el-select
            v-model="environmentNodeNames"
            multiple
            filterable
            :placeholder="$t('opsConsole.nodes.choose')"
          >
            <el-option
              v-for="node in assignableLinuxNodes"
              :key="node.id"
              :label="`${node.name}${node.hostname ? ` · ${node.hostname}` : ''}`"
              :value="node.id"
            />
          </el-select>
          <small class="ops-form-help">{{ $t('opsConsole.nodes.windowsHint') }}</small>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="environmentNodeDialog = false">{{
          $t('opsConsole.common.cancel')
        }}</el-button>
        <el-button
          type="primary"
          :disabled="!environmentNodeNames.length"
          @click="addEnvironmentNodes"
          >{{ $t('opsConsole.nodes.add') }}</el-button
        >
      </template>
    </el-dialog>

    <el-dialog
      v-model="foundationDialog"
      :title="
        selectedEnvironment?.foundationTotal
          ? $t('opsConsole.foundation.repairTitle')
          : $t('opsConsole.environments.deployFoundation')
      "
      width="min(920px, calc(100vw - 32px))"
    >
      <el-steps :active="1" simple class="ops-deploy-steps"
        ><el-step :title="$t('opsConsole.foundation.selectServices')" /><el-step
          :title="$t('opsConsole.foundation.assignNodes')" /><el-step
          :title="$t('opsConsole.foundation.confirmDeploy')"
      /></el-steps>
      <div class="ops-mode-label">{{ $t('opsConsole.foundation.mode') }}</div>
      <div class="ops-mode-grid">
        <button type="button" class="ops-mode-option is-selected">
          <strong>{{ $t('opsConsole.foundation.singleTitle') }}</strong
          ><small>{{ $t('opsConsole.foundation.singleDesc') }}</small>
        </button>
        <button type="button" class="ops-mode-option" disabled>
          <strong
            >{{ $t('opsConsole.foundation.standby') }}
            <em>{{ $t('opsConsole.foundation.later') }}</em></strong
          ><small>{{ $t('opsConsole.foundation.standbyDesc') }}</small>
        </button>
        <button type="button" class="ops-mode-option" disabled>
          <strong
            >{{ $t('opsConsole.foundation.cluster') }}
            <em>{{ $t('opsConsole.foundation.later') }}</em></strong
          ><small>{{ $t('opsConsole.foundation.clusterDesc') }}</small>
        </button>
      </div>
      <el-alert type="info" :closable="false" :title="$t('opsConsole.foundation.dataPathHint')" />
      <div class="ops-node-selection">
        <span>{{ $t('opsConsole.foundation.participatingNodes') }}</span
        ><el-checkbox
          v-for="name in selectedEnvironment?.nodeNames || []"
          :key="name"
          :model-value="true"
          disabled
          >{{ name }}</el-checkbox
        >
      </div>
      <el-table :data="foundationAssignments" max-height="380">
        <el-table-column :label="$t('opsConsole.foundation.service')" min-width="220" prop="name" />
        <el-table-column :label="$t('opsConsole.foundation.runtimeNode')" min-width="260"
          ><template #default="{ row }"
            ><el-select
              v-model="row.nodeId"
              :disabled="isFoundationRepair || row.type === 'if_timeseries'"
              :placeholder="$t('opsConsole.foundation.chooseNode')"
              @change="syncFoundationAssignment(row)"
              ><el-option
                v-for="node in environmentNodes"
                :key="node.id"
                :label="node.name"
                :value="node.id" /></el-select></template
        ></el-table-column>
        <el-table-column :label="$t('opsConsole.foundation.strategyColumn')" width="130"
          ><template #default>{{
            $t('opsConsole.foundation.instanceCount')
          }}</template></el-table-column
        >
      </el-table>
      <div class="ops-distribution">{{ foundationDistribution }}</div>
      <template #footer
        ><span class="ops-dialog-summary">{{
          $t('opsConsole.foundation.summary', { count: selectedEnvironment?.nodeNames.length || 0 })
        }}</span
        ><el-button @click="foundationDialog = false">{{
          $t('opsConsole.common.cancel')
        }}</el-button
        ><el-button type="primary" @click="createFoundationTask">{{
          selectedEnvironment?.foundationTotal
            ? $t('opsConsole.foundation.createRepairTask')
            : $t('opsConsole.foundation.createTask')
        }}</el-button></template
      >
    </el-dialog>

    <el-dialog v-model="packageDialog" :title="$t('opsConsole.nodes.packages')" width="680px">
      <div v-loading="loading.packages" class="ops-package-list">
        <article v-for="item in packages" :key="item.id">
          <div>
            <strong>{{
              item.platform === 'windows'
                ? $t('opsConsole.nodes.windowsPackage')
                : $t('opsConsole.nodes.linuxPackage')
            }}</strong
            ><small
              >{{ item.architecture || $t('opsConsole.nodes.genericArch') }} ·
              {{ item.version || $t('opsConsole.nodes.versionPending') }}</small
            >
          </div>
          <el-button
            :disabled="!item.available"
            :loading="downloadingPackageId === item.id"
            @click="downloadPackage(item)"
            >{{
              item.available
                ? $t('opsConsole.common.download')
                : $t('opsConsole.common.unavailable')
            }}</el-button
          >
        </article>
        <el-empty
          v-if="!loading.packages && !packages.length"
          :description="$t('opsConsole.nodes.noPackages')"
        />
      </div>
    </el-dialog>

    <el-dialog
      v-model="enrollmentDialog"
      :title="$t('opsConsole.nodes.enrollPhysical')"
      width="560px"
    >
      <el-form label-position="top">
        <el-form-item :label="$t('opsConsole.nodes.enrollName')" required
          ><el-input
            v-model="enrollForm.displayName"
            maxlength="80"
            :placeholder="$t('opsConsole.nodes.enrollNameExample')"
        /></el-form-item>
        <el-form-item :label="$t('opsConsole.nodes.nodeType')" required
          ><el-radio-group v-model="enrollForm.platform"
            ><el-radio value="linux">{{ $t('opsConsole.nodes.standard') }}</el-radio
            ><el-radio value="windows">{{
              $t('opsConsole.nodes.windowsCollector')
            }}</el-radio></el-radio-group
          ></el-form-item
        >
        <el-form-item :label="$t('opsConsole.nodes.ttl')"
          ><el-input-number v-model="enrollForm.ttlMinutes" :min="1" :max="1440"
        /></el-form-item>
      </el-form>
      <template #footer
        ><el-button @click="enrollmentDialog = false">{{
          $t('opsConsole.common.cancel')
        }}</el-button
        ><el-button type="primary" :loading="submittingEnrollment" @click="createEnrollment">{{
          $t('opsConsole.nodes.generateCode')
        }}</el-button></template
      >
    </el-dialog>

    <el-dialog
      v-model="codeDialog"
      :title="$t('opsConsole.nodes.codeTitle')"
      width="min(720px, calc(100vw - 32px))"
      :close-on-click-modal="false"
      @closed="clearEnrollmentSecret"
    >
      <div class="ops-secret">
        <code>{{ enrollmentCode }}</code
        ><el-button @click="copyEnrollmentCode">{{ $t('opsConsole.nodes.copy') }}</el-button>
      </div>
      <el-form label-position="top"
        ><el-form-item
          :label="$t('opsConsole.nodes.centerAddress')"
          :error="enrollmentServerUrlValidation.error || undefined"
          ><el-input
            v-model="enrollmentServerUrl"
            spellcheck="false"
            placeholder="https://center.example.com" /></el-form-item
      ></el-form>
      <div class="ops-command">
        <code>{{ installCommand || $t('opsConsole.nodes.centerAddressRequired') }}</code
        ><el-button type="primary" plain :disabled="!installCommand" @click="copyInstallCommand">{{
          $t('opsConsole.nodes.copyCommand')
        }}</el-button>
      </div>
      <template #footer
        ><el-button type="primary" @click="codeDialog = false">{{
          $t('opsConsole.nodes.saved')
        }}</el-button></template
      >
    </el-dialog>

    <el-drawer
      v-model="enrollmentDrawer"
      :title="$t('opsConsole.nodes.accessRequests')"
      size="min(760px, 92vw)"
    >
      <el-table
        :data="enrollments"
        v-loading="loading.enrollments"
        :empty-text="$t('opsConsole.nodes.noRequests')"
      >
        <el-table-column :label="$t('opsConsole.nodes.node')" min-width="180"
          ><template #default="{ row }"
            ><div class="ops-primary-cell">
              <strong>{{ row.displayName || row.reportedHostName || '-' }}</strong
              ><small>{{
                row.ipAddress || (row.platform === 'windows' ? 'Windows' : 'Linux')
              }}</small>
            </div></template
          ></el-table-column
        >
        <el-table-column :label="$t('opsConsole.common.status')" width="110"
          ><template #default="{ row }"
            ><span
              class="ops-status"
              :class="row.status === 'claimed' ? 'is-attention' : 'is-uninitialized'"
              >{{ lifecyclePresentation(row.status) }}</span
            ></template
          ></el-table-column
        >
        <el-table-column :label="$t('opsConsole.nodes.ttl')" min-width="160"
          ><template #default="{ row }">{{
            formatTime(row.expiresAt) || '—'
          }}</template></el-table-column
        >
        <el-table-column :label="$t('opsConsole.common.actions')" width="150"
          ><template #default="{ row }"
            ><template v-if="row.status === 'claimed'"
              ><el-button link type="primary" @click="approve(row)">{{
                $t('opsConsole.nodes.approve')
              }}</el-button
              ><el-button link type="danger" @click="reject(row)">{{
                $t('opsConsole.nodes.reject')
              }}</el-button></template
            ><span v-else>—</span></template
          ></el-table-column
        >
      </el-table>
      <WorkbenchPagination
        :page-size-label="$t('opsConsole.pagination.perPage')"
        v-model:page="enrollmentPage.page"
        v-model:limit="enrollmentPage.limit"
        :total="enrollmentPage.total"
        @change="loadEnrollments"
      />
    </el-drawer>

    <el-dialog
      v-model="deployDialog"
      :title="$t('opsConsole.deployments.dialogTitle')"
      width="min(680px, 92vw)"
      destroy-on-close
      class="ops-deployment-dialog"
    >
      <div class="ops-deploy-mode-tabs" role="tablist">
        <button
          type="button"
          role="tab"
          :aria-selected="deployForm.mode === 'development'"
          :class="['ops-deploy-mode-tab', { 'is-active': deployForm.mode === 'development' }]"
          @click="deployForm.mode = 'development'"
        >
          {{ $t('opsConsole.deployments.developmentMode') }}
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="deployForm.mode === 'production'"
          :class="['ops-deploy-mode-tab', { 'is-active': deployForm.mode === 'production' }]"
          @click="deployForm.mode = 'production'"
        >
          {{ $t('opsConsole.deployments.productionMode') }}
        </button>
      </div>
      <div
        :class="[
          'ops-deployment-form',
          { 'ops-deployment-form--release': deployForm.mode === 'production' },
        ]"
      >
        <el-alert
          v-if="availableEnvironments.length === 0"
          title="暂无可用运行环境，请先完成运行环境初始化。"
          type="warning"
          :closable="false"
          show-icon
        />
        <div class="ops-deployment-field">
          <label>{{ $t('opsConsole.deployments.project') }}</label>
          <el-select
            v-model="deployForm.projectId"
            filterable
            remote
            :placeholder="$t('opsConsole.deployments.chooseProject')"
            :loading="projectLoading"
            :remote-method="searchProjects"
            @change="onProjectChange"
            ><el-option
              v-for="project in projects"
              :key="project.id"
              :label="project.name"
              :value="project.id"
          /></el-select>
        </div>
        <p v-if="deployForm.mode === 'development'" class="ops-deployment-mode-hint">
          开发模式使用服务端固定的开发快照，不需要选择版本。
        </p>
        <div v-if="deployForm.mode === 'production'" class="ops-deployment-field">
          <label>{{ $t('opsConsole.deployments.releasedVersion') }}</label>
          <el-select
            v-model="deployForm.applicationVersionId"
            filterable
            :placeholder="$t('opsConsole.deployments.chooseRelease')"
            :loading="versionLoading"
            :disabled="!deployForm.projectId"
              ><el-option
              v-for="version in versions.filter((item) => item.status === 'ready')"
              :key="version.id"
              :label="version.name ? `${version.version} · ${version.name}` : version.version"
              :value="version.id"
          /></el-select>
        </div>
        <div v-if="availableEnvironments.length > 1" class="ops-deployment-field">
          <label>{{ $t('opsConsole.deployments.targetEnvironment') }}</label>
          <el-select
            v-model="deployForm.environmentId"
            :placeholder="$t('opsConsole.deployments.chooseEnvironment')"
            @change="loadDeploymentNodes"
            ><el-option
              v-for="environment in availableEnvironments"
              :key="environment.id"
              :label="environment.name"
              :value="environment.id"
          /></el-select>
        </div>
        <div class="ops-deployment-field">
          <label>{{ $t('opsConsole.deployments.accessPort') }}</label>
          <div class="ops-deployment-port">
            <el-input-number v-model="deployForm.accessPort" :min="1024" :max="65532" />
            <small>{{ $t('opsConsole.deployments.accessPortHint') }}</small>
          </div>
        </div>
        <section
          class="ops-engine-placement"
          :aria-label="$t('opsConsole.deployments.runtimeEngines')"
        >
          <div class="ops-engine-placement__header">
            <span>{{ $t('opsConsole.deployments.runtimeEngines') }}</span
            ><span>{{ $t('opsConsole.deployments.deployNode') }}</span>
          </div>
          <div
            v-for="engine in deploymentEngineRows"
            :key="engine.key"
            class="ops-engine-placement__row"
          >
            <div>
              <strong>{{ engine.label }}</strong>
            </div>
            <el-select
              v-model="deployForm.placements[engine.key]"
              filterable
              :loading="deploymentNodeLoading"
              :disabled="deploymentNodeLoading"
              :placeholder="$t('opsConsole.deployments.chooseNode')"
            >
              <el-option
                v-for="node in deploymentNodes"
                :key="node.id"
                :value="node.id"
                :label="deploymentNodeLabel(node)"
                :disabled="!isDeploymentNodeReady(node)"
              />
            </el-select>
          </div>
        </section>
      </div>
      <template #footer>
        <div class="ops-deployment-footer">
          <span class="ops-deployment-footer__spacer" />
          <el-button @click="deployDialog = false">{{ $t('opsConsole.common.cancel') }}</el-button>
          <el-button
            type="primary"
            class="ops-deployment-confirm"
            :loading="submittingDeployment"
            :disabled="!canCreateDeployment"
            @click="createDeployment"
            >{{
              deployForm.mode === 'production'
                ? $t('opsConsole.deployments.deployProduction')
                : $t('opsConsole.deployments.deployDevelopment')
            }}</el-button
          >
        </div>
      </template>
    </el-dialog>

    <el-drawer
      v-model="deploymentDrawer"
      :title="$t('opsConsole.deployments.detailTitle')"
      size="min(640px, 92vw)"
    >
      <template v-if="selectedDeploymentRow">
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="$t('opsConsole.deployments.project')">{{
            selectedDeploymentRow.projectName
          }}</el-descriptions-item>
          <el-descriptions-item :label="$t('opsConsole.deployments.version')">{{
            selectedDeploymentRow.version
          }}</el-descriptions-item>
          <el-descriptions-item :label="$t('opsConsole.deployments.environment')">{{
            selectedDeploymentRow.environmentName
          }}</el-descriptions-item>
          <el-descriptions-item :label="$t('opsConsole.common.status')"
            ><span class="ops-status" :class="selectedDeploymentRow.statusClass">{{
              selectedDeploymentRow.status
            }}</span></el-descriptions-item
          >
          <el-descriptions-item :label="$t('opsConsole.deployments.runtimeEngines')" :span="2">{{
            selectedDeploymentRow.services.join('、')
          }}</el-descriptions-item>
        </el-descriptions>
        <h3 class="ops-drawer-heading">{{ $t('opsConsole.deployments.latestDeployment') }}</h3>
        <el-steps direction="vertical" :active="4" finish-status="success">
          <el-step
            :title="$t('opsConsole.deployments.validation')"
            :description="$t('opsConsole.deployments.validationDesc')"
          />
          <el-step
            :title="$t('opsConsole.deployments.prepare')"
            :description="$t('opsConsole.deployments.prepareDesc')"
          />
          <el-step
            :title="$t('opsConsole.deployments.serviceStart')"
            :description="
              $t('opsConsole.deployments.serviceStartDesc', {
                services: selectedDeploymentRow.services.join(' / '),
              })
            "
          />
          <el-step
            :title="$t('opsConsole.deployments.healthCheck')"
            :description="$t('opsConsole.deployments.healthDesc')"
          />
        </el-steps>
      </template>
    </el-drawer>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowDown } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import WorkbenchPagination from '@/components/WorkbenchPagination.vue'
import { projectAPI } from '@/api/project.api'
import { can } from '@/permissions'
import { useAuthStore } from '@/store'
import OpsSectionSwitcher, { type OpsSection } from './components/OpsSectionSwitcher.vue'
import {
  opsAPI,
  type ApplicationVersion,
  type NodeEnrollment,
  type NodePackage,
  type OpsNode,
  type ProjectDeployment,
  type RuntimeEnvironment as ApiRuntimeEnvironment,
  type RuntimeEnvironmentEvent as ApiRuntimeEnvironmentEvent,
  type RuntimeEnvironmentService,
  type FoundationServiceType,
} from '@/api/ops.api'
import { formatDateTime } from '@/utils/date'
import {
  enrollmentCapabilities,
  enrollmentInstallCommand,
  lifecyclePresentation,
  validateEnrollmentServerUrl,
} from './utils/ops-presentation'

type EnvironmentStatus = 'available' | 'attention' | 'uninitialized' | 'deleting'
type EnvironmentDetailSection = 'overview' | 'nodes' | 'services' | 'events'
interface RuntimeEnvironment {
  id: string
  name: string
  code: string
  isDefault: boolean
  status: EnvironmentStatus
  nodeCount: number
  nodeNames: string[]
  onlineNodes: number
  foundationHealthy: number
  foundationTotal: number
  projectCount: number
  recentChange: string
  recentAt: string
  recentBy: string
}
interface EnvironmentEvent {
  environmentId: string
  time: string
  name: string
  target: string
  operator: string
  result: string
  success: boolean
  message: string
}
interface TaskRow {
  id: string
  name: string
  type: string
  target: string
  step: string
  progress: number
  status: 'active' | 'failed' | 'completed'
  operator: string
  startedAt: string
}
interface DeploymentRow {
  id: string
  projectName: string
  version: string
  environmentId: string
  environmentName: string
  services: string[]
  status: string
  statusClass: string
  updatedAt: string
}
interface ProjectOption {
  id: string
  name: string
}
type DeploymentEngineKey = 'base' | 'compute' | 'alarm' | 'collector'
interface PageState {
  page: number
  limit: number
  total: number
  keyword: string
}

const createPageState = (): PageState => ({ page: 1, limit: 10, total: 0, keyword: '' })
const { t, locale } = useI18n()
const authStore = useAuthStore()
const canAdministerOperations = computed(() => can(authStore.userInfo?.role, 'runtime:operate'))
const activeTab = ref<OpsSection>(canAdministerOperations.value ? 'environments' : 'deployments')
const environmentDetailTab = ref<EnvironmentDetailSection>('overview')
const environmentDetailSections = computed<
  Array<{
    label: string
    value: EnvironmentDetailSection
  }>
>(() => [
  { label: t('opsConsole.environments.overview'), value: 'overview' },
  { label: t('opsConsole.sections.nodes'), value: 'nodes' },
  { label: t('opsConsole.environments.foundations'), value: 'services' },
  { label: t('opsConsole.environments.events'), value: 'events' },
])
const environmentKeyword = ref('')
const environmentStatus = ref<'all' | EnvironmentStatus>('all')
const nodeTypeFilter = ref<'all' | 'linux' | 'windows'>('all')
const deploymentEnvironmentFilter = ref('all')
const taskKeyword = ref('')
const taskTypeFilter = ref('all')
const selectedEnvironmentId = ref('')
const environmentManagementMode = ref(false)
const environmentDialog = ref(false)
const environmentDialogMode = ref<'create' | 'edit'>('create')
const environmentDeleteDialog = ref(false)
const environmentDeleteConfirmation = ref('')
const savingEnvironment = ref(false)
const deletingEnvironment = ref(false)
const environmentNodeDialog = ref(false)
const foundationDialog = ref(false)
const foundationAdjusting = ref(false)
const foundationDraft = reactive<Partial<Record<FoundationServiceType, string>>>({})
const packageDialog = ref(false)
const enrollmentDialog = ref(false)
const codeDialog = ref(false)
const enrollmentDrawer = ref(false)
const deployDialog = ref(false)
const deploymentDrawer = ref(false)
const submittingEnrollment = ref(false)
const submittingDeployment = ref(false)
const downloadingPackageId = ref('')
const enrollmentCode = ref('')
const enrollmentServerUrl = ref('')
const createdEnrollment = ref<NodeEnrollment | null>(null)
const selectedDeploymentRow = ref<DeploymentRow | null>(null)
const nodePage = reactive(createPageState())
const deploymentPage = reactive(createPageState())
const enrollmentPage = reactive(createPageState())
const environmentPage = reactive(createPageState())
const environmentNodePage = reactive(createPageState())
const environmentEventPage = reactive(createPageState())
const taskPage = reactive({ page: 1, limit: 10 })
const loading = reactive({
  environments: false,
  environmentNodes: false,
  environmentEvents: false,
  nodes: false,
  deployments: false,
  enrollments: false,
  packages: false,
})
const nodes = ref<OpsNode[]>([])
const environmentNodes = ref<OpsNode[]>([])
const deployments = ref<ProjectDeployment[]>([])
const enrollments = ref<NodeEnrollment[]>([])
const packages = ref<NodePackage[]>([])
const projects = ref<ProjectOption[]>([])
const versions = ref<ApplicationVersion[]>([])
const deploymentNodes = ref<OpsNode[]>([])
const deploymentNodeLoading = ref(false)
const environmentNodeNames = ref<string[]>([])
const projectLoading = ref(false)
const versionLoading = ref(false)
let disposed = false
let lastExternalDeployRequestId = ''
let environmentReconcileTimer: number | undefined

const environments = ref<RuntimeEnvironment[]>([])
const environmentOptions = ref<RuntimeEnvironment[]>([])
const environmentOptionTotal = ref(0)
// 运维任务只展示后端产生的真实任务。环境初始化当前通过环境事件审计，工程
// 按运行环境部署尚未交付，因此不能用前端定时器或静态行伪造执行进度。
const tasks = ref<TaskRow[]>([])
const foundationDefinitions: Array<{ type: FoundationServiceType; name: string }> = [
  { type: 'if_realtime', name: '' },
  { type: 'if_history', name: '' },
  { type: 'if_timeseries', name: '' },
  { type: 'if_message', name: '' },
  { type: 'if_object', name: '' },
  { type: 'nats_jetstream', name: '' },
  { type: 'nginx', name: '' },
  { type: 'traefik', name: '' },
]
function updateFoundationNames() {
  foundationDefinitions.forEach((item) => {
    item.name = t(`opsConsole.foundation.services.${item.type}`)
  })
}
updateFoundationNames()
const serviceNames = computed(() =>
  foundationDefinitions.map((item) => t(`opsConsole.foundation.services.${item.type}`)),
)
const foundationAssignments = ref(foundationDefinitions.map((item) => ({ ...item, nodeId: '' })))
const foundationServices = ref<RuntimeEnvironmentService[]>([])
const foundationChangedCount = computed(
  () =>
    foundationDefinitions.filter(({ type }) => {
      const current = foundationServices.value.find(
        (service) => service.serviceType === type,
      )?.nodeId
      return current && foundationDraft[type] && current !== foundationDraft[type]
    }).length,
)
const environmentEvents = ref<EnvironmentEvent[]>([])
const environmentForm = reactive({ name: '' })
const enrollForm = reactive({
  displayName: '',
  platform: 'linux' as 'linux' | 'windows',
  ttlMinutes: 60,
})
const deployForm = reactive({
  mode: 'development' as 'development' | 'production',
  projectId: '',
  applicationVersionId: '',
  environmentId: '',
  accessPort: 17800,
  placements: {
    base: '',
    compute: '',
    alarm: '',
    collector: '',
  } as Record<DeploymentEngineKey, string>,
})

const refreshing = computed(() => Object.values(loading).some(Boolean))
const selectedEnvironment = computed(
  () =>
    environmentOptions.value.find((item) => item.id === selectedEnvironmentId.value) ||
    environments.value.find((item) => item.id === selectedEnvironmentId.value) ||
    null,
)
const environmentHeading = computed(() =>
  environmentOptionTotal.value > 1
    ? selectedEnvironment.value?.name || ''
    : t('opsConsole.environments.defaultOverview'),
)
const availableEnvironments = computed(() =>
  environmentOptions.value.filter((item) => item.status === 'available'),
)
const linuxNodes = computed(() => nodes.value.filter(isStandardRuntimeNode))
const assignableLinuxNodes = computed(() => {
  const assigned = new Set(environmentNodes.value.map((node) => node.id))
  return linuxNodes.value.filter((node) => !assigned.has(node.id))
})
const pagedEnvironments = computed(() => environments.value)
const environmentTotalPages = computed(() =>
  pageTotal(environmentPage.total, environmentPage.limit),
)
const environmentPaginationSummary = computed(() =>
  paginationSummary(environmentPage.page, environmentPage.limit, environmentPage.total),
)
const environmentPaginationIndicator = computed(() =>
  paginationIndicator(environmentPage.page, environmentPage.limit, environmentPage.total),
)
const filteredNodes = computed(() =>
  nodes.value.filter(
    (node) =>
      nodeTypeFilter.value === 'all' ||
      (nodeTypeFilter.value === 'linux'
        ? isStandardRuntimeNode(node)
        : node.platform === 'windows'),
  ),
)
const environmentNodeRows = computed(() =>
  environmentNodes.value.map((node) => {
    const hostedServices = foundationServices.value.filter((service) => service.nodeId === node.id)
    const deleting = selectedEnvironment.value?.status === 'deleting'
    const isCenter = node.nodeKind === 'center'
    const hasDeployment = Boolean(node.assignedDeploymentId)
    const unassignReason = deleting
      ? t('opsConsole.nodes.unassignBlockedDeleting')
      : isCenter
        ? t('opsConsole.nodes.unassignBlockedCenter')
        : hostedServices.length
        ? t('opsConsole.nodes.unassignBlockedFoundation', {
            count: hostedServices.length,
            services: hostedServices
              .map((service) => t(`opsConsole.foundation.services.${service.serviceType}`))
              .join('、'),
          })
        : hasDeployment
          ? t('opsConsole.nodes.unassignBlockedDeployment', {
              project: node.assignedProjectName || t('opsConsole.nodes.unknownProject'),
            })
          : ''
    return {
      id: node.id,
      name: node.name,
      address: `${node.ipAddress || t('opsConsole.nodes.addressPending')} · ${node.platform} ${node.architecture || ''}`,
      cpu: resourcePercentNumber(node, 'cpu'),
      memory: resourcePercentNumber(node, 'memory'),
      diskUsage: resourcePercentNumber(node, 'disk'),
      agentVersion: node.agentVersion || '—',
      observedStatus: node.observedStatus || 'offline',
      clusterDesiredAction: node.clusterDesiredAction || 'active',
      roleLabel:
        node.nodeKind === 'center'
          ? t('opsConsole.nodes.centerBuiltIn')
          : t('opsConsole.nodes.runtimeNode'),
      clusterStatusLabel:
        node.observedStatus !== 'online'
          ? t('opsConsole.common.offline')
          : node.clusterDesiredAction === 'removing'
            ? t('opsConsole.nodes.removing')
            : node.clusterStatus === 'ready'
              ? t('opsConsole.common.available')
              : node.clusterStatus === 'failed'
                ? t('opsConsole.common.abnormal')
                : node.clusterStatus === 'starting'
                  ? t('opsConsole.environments.inProgress')
                  : t('opsConsole.common.uninitialized'),
      clusterStatusClass:
        node.observedStatus !== 'online' || node.clusterStatus === 'failed'
          ? 'is-attention'
          : node.clusterDesiredAction === 'removing'
            ? 'is-running'
            : node.clusterStatus === 'ready'
              ? 'is-available'
              : 'is-running',
      clusterMessage: node.clusterMessage || '',
      timeSyncLabel: timeSyncPresentation(node).label,
      timeSyncClass: timeSyncPresentation(node).className,
      timeSyncDetail: timeSyncPresentation(node).detail,
      lastHeartbeatAt: node.lastHeartbeatAt
        ? formatDateTime(node.lastHeartbeatAt)
        : t('opsConsole.common.notReported'),
      unassignDisabled: deleting || isCenter || hostedServices.length > 0 || hasDeployment,
      unassignReason,
    }
  }),
)
const foundationRows = computed(() => {
  return foundationDefinitions.map(({ type }) => {
    const name = t(`opsConsole.foundation.services.${type}`)
    const service = foundationServices.value.find((item) => item.serviceType === type)
    const deployed = Boolean(service)
    const isHealthy = service?.observedStatus === 'running'
    const isUnhealthy = deployed && !isHealthy && service?.observedStatus !== 'pending'
    const isDeploying = deployed && service?.observedStatus === 'pending'
    return {
      type,
      name,
      nodeName: service?.nodeName || t('opsConsole.common.undeployed'),
      instances: deployed ? 1 : 0,
      state: !deployed
        ? 'uninitialized'
        : isDeploying
          ? 'running'
          : isUnhealthy
            ? 'attention'
            : isHealthy
              ? 'available'
              : 'attention',
      label: !deployed
        ? t('opsConsole.common.undeployed')
        : isDeploying
          ? t('opsConsole.common.deploying')
          : isUnhealthy
            ? t('opsConsole.common.abnormal')
            : t('opsConsole.common.healthy'),
      checkedAt: service?.observedAt
        ? formatDateTime(service.observedAt)
        : deployed
          ? t('opsConsole.foundation.firstCheck')
          : '—',
      message: service?.lastMessage || '',
    }
  })
})
const foundationServiceGroups = computed(() => {
  const groups = [
    {
      name: t('opsConsole.foundation.storage'),
      description: serviceNames.value.slice(0, 5).join(' / '),
      names: serviceNames.value.slice(0, 5),
    },
    {
      name: t('opsConsole.foundation.bus'),
      description: 'NATS JetStream',
      names: ['NATS JetStream'],
    },
    {
      name: t('opsConsole.foundation.entry'),
      description: 'Nginx / Traefik',
      names: ['Nginx', 'Traefik'],
    },
  ]
  return groups.map((group) => {
    const rows = foundationRows.value.filter((item) => group.names.includes(item.name))
    const healthy = rows.filter((item) => item.state === 'available').length
    const deployed = rows.filter((item) => item.state !== 'uninitialized').length
    const deploying = rows.some((item) => item.state === 'running')
    const locations = [
      ...new Set(
        rows
          .map((item) => item.nodeName)
          .filter((name) => name !== t('opsConsole.common.undeployed')),
      ),
    ]
    const state = !deployed
      ? 'uninitialized'
      : deploying
        ? 'running'
        : healthy === rows.length
          ? 'available'
          : 'attention'
    return {
      ...group,
      state,
      location: locations.length
        ? t('opsConsole.foundation.runsOn', { nodes: locations.join(' / ') })
        : t('opsConsole.foundation.awaitingNode'),
      label: !deployed
        ? t('opsConsole.common.undeployed')
        : deploying
          ? t('opsConsole.common.deploying')
          : `${healthy} / ${rows.length} ${t('opsConsole.common.healthy')}`,
    }
  })
})
const environmentMaxDiskUsage = computed(() =>
  environmentNodeRows.value.length
    ? Math.max(...environmentNodeRows.value.map((item) => item.diskUsage))
    : 0,
)
const environmentSetupTitle = computed(() =>
  selectedEnvironment.value?.nodeCount
    ? t('opsConsole.environments.setupWithNodes')
    : t('opsConsole.environments.setupWithoutNodes'),
)
const environmentSetupAction = computed(() => {
  const environment = selectedEnvironment.value
  if (!environment?.nodeCount) return t('opsConsole.environments.associateNodes')
  return environment.foundationTotal
    ? t('opsConsole.environments.viewFoundation')
    : t('opsConsole.environments.deployFoundation')
})
const environmentProblems = computed(() => {
  const offlineNodes = environmentNodeRows.value
    .filter(
      (node) => node.observedStatus !== 'online' || node.clusterStatusClass === 'is-attention',
    )
    .map((node) => ({
      name: node.name,
      message: node.clusterMessage || t('opsConsole.nodes.offlineOrFoundation'),
    }))
  const failedServices = foundationRows.value
    .filter((service) => service.state === 'attention')
    .map((service) => ({
      name: service.name,
      message: service.message || t('opsConsole.foundation.healthFailed'),
    }))
  return [...offlineNodes, ...failedServices]
})
const foundationDeploying = computed(
  () =>
    foundationServices.value.length > 0 &&
    foundationServices.value.some((service) => service.observedStatus === 'pending'),
)
const environmentProblemTitle = computed(
  () => environmentProblems.value[0]?.name || t('opsConsole.environments.genericIssue'),
)
const environmentProblemDescription = computed(() => {
  const first = environmentProblems.value[0]
  if (!first) return t('opsConsole.environments.retryAfterRefresh')
  const more =
    environmentProblems.value.length > 1
      ? t('opsConsole.environments.additionalIssues', {
          count: environmentProblems.value.length - 1,
        })
      : ''
  return t('opsConsole.environments.issueBlock', { message: `${first.message}${more}` })
})
const selectedEnvironmentEvents = computed(() =>
  environmentEvents.value.filter((item) => item.environmentId === selectedEnvironment.value?.id),
)
const pagedEnvironmentNodes = computed(() => environmentNodeRows.value)
const environmentNodeTotalPages = computed(() =>
  pageTotal(environmentNodePage.total, environmentNodePage.limit),
)
const environmentNodePaginationSummary = computed(() =>
  paginationSummary(environmentNodePage.page, environmentNodePage.limit, environmentNodePage.total),
)
const environmentNodePaginationIndicator = computed(() =>
  paginationIndicator(
    environmentNodePage.page,
    environmentNodePage.limit,
    environmentNodePage.total,
  ),
)
const pagedEnvironmentEvents = computed(() => selectedEnvironmentEvents.value)
const environmentEventTotalPages = computed(() =>
  pageTotal(environmentEventPage.total, environmentEventPage.limit),
)
const environmentEventPaginationSummary = computed(() =>
  paginationSummary(
    environmentEventPage.page,
    environmentEventPage.limit,
    environmentEventPage.total,
  ),
)
const environmentEventPaginationIndicator = computed(() =>
  paginationIndicator(
    environmentEventPage.page,
    environmentEventPage.limit,
    environmentEventPage.total,
  ),
)
const foundationDistribution = computed(() => {
  const count = new Map<string, number>()
  foundationAssignments.value.forEach((item) => {
    const nodeName = environmentNodes.value.find((node) => node.id === item.nodeId)?.name
    if (nodeName) count.set(nodeName, (count.get(nodeName) || 0) + 1)
  })
  return count.size
    ? t('opsConsole.foundation.distribution', {
        details: [...count.entries()]
          .map(([name, total]) =>
            t('opsConsole.foundation.distributionItem', { name, count: total }),
          )
          .join(locale.value === 'zh' ? '，' : ', '),
      })
    : t('opsConsole.foundation.distributionEmpty')
})
const foundationDeployDisabled = computed(
  () =>
    selectedEnvironment.value?.status === 'deleting' ||
    !environmentNodes.value.length ||
    foundationServices.value.some(
      (service) =>
        service.observedStatus === 'pending' ||
        service.desiredGeneration !== service.observedGeneration,
    ) ||
    environmentNodes.value.some(
      (node) => node.observedStatus !== 'online' || node.clusterStatus !== 'ready',
    ),
)
const isFoundationRepair = computed(() => (selectedEnvironment.value?.foundationTotal || 0) > 0)
const deploymentRows = computed<DeploymentRow[]>(() =>
  deployments.value.map((item) => {
    const failed = item.observedStatus === 'failed' || item.entryStatus === 'failed'
    const running = item.observedStatus === 'running' && !failed
    return {
      id: item.id,
      projectName: item.projectName,
      version: item.version || item.applicationVersionId || '--',
      environmentId: item.environmentId || '',
      environmentName:
        item.environmentName ||
        t('opsConsole.deployments.legacyNode', { name: item.nodeName || item.nodeId || '--' }),
      services:
        item.services?.map((service) => {
          const labels: Record<string, string> = {
            base: t('opsConsole.deployments.baseEngine'),
            compute: t('opsConsole.deployments.computeEngine'),
            alarm: t('opsConsole.deployments.alarmEngine'),
            collector: t('opsConsole.deployments.collectionEngine'),
          }
          return labels[service.serviceType] || service.serviceType
        }) || [t('opsConsole.deployments.baseEngine')],
      status: running
        ? t('opsConsole.common.running')
        : failed
          ? t('opsConsole.common.failed')
          : item.observedStatus === 'stopped'
            ? t('opsConsole.common.stopped')
            : t('opsConsole.common.pending'),
      statusClass: running ? 'is-available' : failed ? 'is-attention' : 'is-uninitialized',
      updatedAt: item.updatedAt
        ? formatDateTime(item.updatedAt)
        : t('opsConsole.common.statePending'),
    }
  }),
)
const filteredDeploymentRows = computed(() =>
  deploymentRows.value.filter(
    (item) =>
      deploymentEnvironmentFilter.value === 'all' ||
      item.environmentId === deploymentEnvironmentFilter.value,
  ),
)
const deploymentDisplayTotal = computed(() =>
  deploymentEnvironmentFilter.value === 'all'
    ? deploymentPage.total
    : filteredDeploymentRows.value.length,
)
const filteredTasks = computed(() => {
  const keyword = taskKeyword.value.trim().toLowerCase()
  const visibleTasks = canAdministerOperations.value
    ? tasks.value
    : tasks.value.filter((item) => item.type === 'project-deploy')
  return visibleTasks.filter(
    (item) =>
      (taskTypeFilter.value === 'all' || item.type === taskTypeFilter.value) &&
      (!keyword || `${item.name} ${item.target}`.toLowerCase().includes(keyword)),
  )
})
const pagedTasks = computed(() => paginateRows(filteredTasks.value, taskPage.page, taskPage.limit))
const taskTotalPages = computed(() => pageTotal(filteredTasks.value.length, taskPage.limit))
const taskPaginationSummary = computed(() =>
  paginationSummary(taskPage.page, taskPage.limit, filteredTasks.value.length),
)
const taskPaginationIndicator = computed(() =>
  paginationIndicator(taskPage.page, taskPage.limit, filteredTasks.value.length),
)
const activeTaskCount = computed(
  () => tasks.value.filter((item) => item.status === 'active').length,
)
const failedTaskCount = computed(
  () => tasks.value.filter((item) => item.status === 'failed').length,
)
const selectedProject = computed(() =>
  projects.value.find((item) => item.id === deployForm.projectId),
)
const selectedVersion = computed(() =>
  versions.value.find((item) => item.id === deployForm.applicationVersionId),
)
const deploymentCapabilities = computed(() => {
  const source =
    deployForm.mode === 'production'
      ? selectedVersion.value
      : versions.value.find((item) => item.version === '__DEV__' || item.mode === 'development')
  const capabilities = (source?.capabilities || source?.manifest?.capabilities || []) as unknown[]
  return new Set(capabilities.map((item) => String(item).toLowerCase()))
})
const deploymentEngineRows = computed(() => {
  const rows: Array<{ key: DeploymentEngineKey; label: string }> = [
    { key: 'base', label: t('opsConsole.deployments.baseEngine') },
  ]
  for (const [key, label] of [
    ['compute', 'computeEngine'],
    ['alarm', 'alarmEngine'],
    ['collector', 'collectionEngine'],
  ]) {
    if (deploymentCapabilities.value.has(key))
      rows.push({ key: key as DeploymentEngineKey, label: t(`opsConsole.deployments.${label}`) })
  }
  return rows
})
const canCreateDeployment = computed(() =>
  Boolean(
    deployForm.projectId &&
    (deployForm.mode === 'development' || deployForm.applicationVersionId) &&
    deployForm.environmentId &&
    deploymentEngineRows.value.every((engine) => deployForm.placements[engine.key]) &&
    deployForm.accessPort >= 1024 &&
    deployForm.accessPort <= 65532 &&
    (deployForm.mode === 'development' || selectedVersion.value),
  ),
)
const defaultEnrollmentServerUrl = window.location.origin.replace(/\/$/, '')
const enrollmentServerUrlValidation = computed(() =>
  validateEnrollmentServerUrl(enrollmentServerUrl.value),
)
const installCommand = computed(() =>
  createdEnrollment.value && enrollmentCode.value && enrollmentServerUrlValidation.value.valid
    ? enrollmentInstallCommand({
        platform: createdEnrollment.value.platform,
        serverUrl: enrollmentServerUrlValidation.value.normalized,
        enrollmentCode: enrollmentCode.value,
      })
    : '',
)

watch([environmentKeyword, environmentStatus], () => {
  environmentPage.page = 1
  void loadEnvironments().catch((error) =>
    ElMessage.error(apiErrorMessage(error, t('opsConsole.environments.queryFailed'))),
  )
})
watch(locale, () => {
  updateFoundationNames()
  foundationAssignments.value = foundationAssignments.value.map((item) => ({
    ...item,
    name: t(`opsConsole.foundation.services.${item.type}`),
  }))
  void Promise.all([loadEnvironments(), loadEnvironmentOptions()])
  if (selectedEnvironmentId.value) void loadEnvironmentEvents(selectedEnvironmentId.value)
})
watch(
  () => [environmentPage.page, environmentPage.limit],
  () => void loadEnvironments(),
)
watch(
  () => [environmentNodePage.page, environmentNodePage.limit],
  () => {
    if (selectedEnvironmentId.value) void loadEnvironmentNodes(selectedEnvironmentId.value)
  },
)
watch(
  () => [environmentEventPage.page, environmentEventPage.limit],
  () => {
    if (selectedEnvironmentId.value) void loadEnvironmentEvents(selectedEnvironmentId.value)
  },
)
watch([taskKeyword, taskTypeFilter], () => {
  taskPage.page = 1
})
watch(deploymentEnvironmentFilter, () => {
  deploymentPage.page = 1
})

function pageTotal(total: number, limit: number) {
  return total > 0 ? Math.ceil(total / limit) : 0
}
function paginateRows<T>(items: T[], page: number, limit: number) {
  const safePage = Math.min(Math.max(page, 1), Math.max(pageTotal(items.length, limit), 1))
  const start = (safePage - 1) * limit
  return items.slice(start, start + limit)
}
function paginationSummary(page: number, limit: number, total: number) {
  if (total <= 0) return t('opsConsole.pagination.summary', { start: 0, end: 0, total: 0 })
  const safePage = Math.min(Math.max(page, 1), pageTotal(total, limit))
  const start = (safePage - 1) * limit + 1
  const end = Math.min(safePage * limit, total)
  return t('opsConsole.pagination.summary', { start, end, total })
}
function paginationIndicator(page: number, limit: number, total: number) {
  const totalPages = pageTotal(total, limit)
  const safePage = totalPages > 0 ? Math.min(Math.max(page, 1), totalPages) : 0
  return t('opsConsole.pagination.page', { page: safePage, pages: totalPages })
}

function environmentStatusLabel(status: EnvironmentStatus) {
  return {
    available: t('opsConsole.common.available'),
    attention: t('opsConsole.common.attention'),
    uninitialized: t('opsConsole.common.uninitialized'),
    deleting: t('opsConsole.common.deleting'),
  }[status]
}
function taskTypeLabel(type: string) {
  return (
    {
      'environment-init': t('opsConsole.tasks.environmentInit'),
      'foundation-deploy': t('opsConsole.tasks.foundationDeploy'),
      'project-deploy': t('opsConsole.tasks.projectDeploy'),
    }[type] || type
  )
}
function taskStatusLabel(status: TaskRow['status']) {
  return status === 'active'
    ? t('opsConsole.tasks.active')
    : status === 'completed'
      ? t('opsConsole.tasks.completed')
      : t('opsConsole.common.failed')
}
function environmentActionLabel(status: EnvironmentStatus) {
  return {
    available: t('opsConsole.common.view'),
    attention: t('opsConsole.environments.handleAbnormal'),
    uninitialized: t('opsConsole.environments.continueSetup'),
    deleting: t('opsConsole.common.view'),
  }[status]
}
function switchSection(section: OpsSection) {
  activeTab.value = section
  if (section !== 'environments') return
  environmentManagementMode.value = false
  const environment = selectedEnvironment.value || defaultEnvironmentOption()
  if (environment) void openEnvironment(environment)
}
async function openEnvironment(item: RuntimeEnvironment) {
  environmentManagementMode.value = false
  selectedEnvironmentId.value = item.id
  environmentDetailTab.value = 'overview'
  environmentNodePage.page = 1
  environmentEventPage.page = 1
  try {
    const [detail] = await Promise.all([
      opsAPI.getRuntimeEnvironment(item.id),
      loadEnvironmentNodes(item.id),
      loadEnvironmentEvents(item.id),
      loadEnvironmentServices(item.id),
    ])
    const index = environments.value.findIndex((environment) => environment.id === item.id)
    if (index >= 0) {
      const view = toEnvironmentView(detail)
      view.nodeNames = environmentNodes.value.map((node) => node.name)
      environments.value[index] = view
    }
    const optionIndex = environmentOptions.value.findIndex(
      (environment) => environment.id === item.id,
    )
    if (optionIndex >= 0) {
      const view = toEnvironmentView(detail)
      view.nodeNames = environmentNodes.value.map((node) => node.name)
      environmentOptions.value[optionIndex] = view
    }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.environments.loadFailed')))
  }
}
function closeEnvironment() {
  selectedEnvironmentId.value = ''
  foundationServices.value = []
  environmentManagementMode.value = true
}
function defaultEnvironmentOption() {
  return (
    environmentOptions.value.find((environment) => environment.isDefault) ||
    environmentOptions.value[0] ||
    null
  )
}
function selectEnvironment(environmentId: string) {
  const environment = environmentOptions.value.find((item) => item.id === environmentId)
  if (environment) void openEnvironment(environment)
}
function closeEnvironmentManagement() {
  environmentManagementMode.value = false
  const environment = selectedEnvironment.value || defaultEnvironmentOption()
  if (environment) void openEnvironment(environment)
}
function openEnvironmentDialog() {
  environmentDialogMode.value = 'create'
  environmentForm.name = ''
  environmentDialog.value = true
}
function handleEnvironmentCommand(command: string) {
  if (command === 'manage') {
    environmentManagementMode.value = true
    return
  }
  if (command === 'edit') {
    const environment = selectedEnvironment.value
    if (!environment) return
    environmentDialogMode.value = 'edit'
    environmentForm.name = environment.name
    environmentDialog.value = true
    return
  }
  if (command === 'delete') {
    environmentDeleteConfirmation.value = ''
    environmentDeleteDialog.value = true
  }
}
async function saveEnvironment() {
  if (!environmentForm.name.trim()) {
    ElMessage.warning(t('opsConsole.environments.nameRequired'))
    return
  }
  savingEnvironment.value = true
  try {
    const current = selectedEnvironment.value
    const item =
      environmentDialogMode.value === 'edit' && current
        ? await opsAPI.updateRuntimeEnvironment(current.id, { name: environmentForm.name.trim() })
        : await opsAPI.createRuntimeEnvironment({ name: environmentForm.name.trim() })
    environmentDialog.value = false
    await Promise.all([loadEnvironments(), loadEnvironmentOptions()])
    await openEnvironment(toEnvironmentView(item))
    ElMessage.success(
      environmentDialogMode.value === 'edit'
        ? t('opsConsole.environments.updateSuccess')
        : t('opsConsole.environments.createSuccess'),
    )
  } catch (error) {
    ElMessage.error(
      apiErrorMessage(
        error,
        environmentDialogMode.value === 'edit'
          ? t('opsConsole.environments.updateFailed')
          : t('opsConsole.environments.createFailed'),
      ),
    )
  } finally {
    savingEnvironment.value = false
  }
}
async function deleteEnvironment() {
  const environment = selectedEnvironment.value
  if (!environment || environmentDeleteConfirmation.value !== environment.name) return
  deletingEnvironment.value = true
  try {
    const result = await opsAPI.deleteRuntimeEnvironment(
      environment.id,
      environmentDeleteConfirmation.value,
    )
    environmentDeleteDialog.value = false
    if (result.status === 'deleted') {
      closeEnvironment()
      await Promise.all([loadEnvironments(), loadEnvironmentOptions()])
      ElMessage.success(t('opsConsole.environments.deleteSuccess'))
      return
    }
    await Promise.all([loadEnvironments(), loadEnvironmentOptions()])
    const updated = environments.value.find((item) => item.id === environment.id)
    if (updated) await openEnvironment(updated)
    ElMessage.success(t('opsConsole.environments.deleteStarted'))
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.environments.deleteFailed')))
  } finally {
    deletingEnvironment.value = false
  }
}
function openEnvironmentNodeDialog() {
  environmentNodeNames.value = []
  environmentNodeDialog.value = true
}
function continueEnvironmentSetup() {
  const environment = selectedEnvironment.value
  if (!environment?.nodeCount) {
    openEnvironmentNodeDialog()
    return
  }
  if (environment.foundationTotal) {
    environmentDetailTab.value = 'services'
    return
  }
  openFoundationDialog()
}
async function addEnvironmentNodes() {
  const environment = selectedEnvironment.value
  if (!environment || !environmentNodeNames.value.length) return
  try {
    await opsAPI.addRuntimeEnvironmentNodes(environment.id, environmentNodeNames.value)
    environmentNodeDialog.value = false
    await Promise.all([loadEnvironments(), loadEnvironmentOptions()])
    await Promise.all([
      loadNodes(),
      loadEnvironmentNodes(environment.id),
      loadEnvironmentEvents(environment.id),
    ])
    ElMessage.success(t('opsConsole.nodes.addSuccess'))
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.nodes.addFailed')))
  }
}
async function removeEnvironmentNode(nodeId: string, nodeName: string) {
  const environment = selectedEnvironment.value
  if (!environment) return
  try {
    await ElMessageBox.confirm(
      t('opsConsole.nodes.removeConfirm', { name: nodeName }),
      t('opsConsole.nodes.removeTitle'),
      {
        type: 'warning',
        confirmButtonText: t('opsConsole.nodes.remove'),
        cancelButtonText: t('opsConsole.common.cancel'),
      },
    )
    await opsAPI.removeRuntimeEnvironmentNode(environment.id, nodeId)
    await Promise.all([loadEnvironments(), loadEnvironmentOptions()])
    await Promise.all([
      loadNodes(),
      loadEnvironmentNodes(environment.id),
      loadEnvironmentEvents(environment.id),
    ])
    ElMessage.success(t('opsConsole.nodes.removeSuccess'))
  } catch (error) {
    if (error !== 'cancel' && error !== 'close')
      ElMessage.error(apiErrorMessage(error, t('opsConsole.nodes.removeFailed')))
  }
}
async function removePhysicalNode(node: OpsNode) {
  try {
    await ElMessageBox.confirm(
      t('opsConsole.nodes.removePhysicalConfirm', { name: node.name }),
      t('opsConsole.nodes.removePhysicalTitle'),
      {
        type: 'warning',
        confirmButtonText: t('opsConsole.nodes.remove'),
        cancelButtonText: t('opsConsole.common.cancel'),
      },
    )
    await opsAPI.removeNode(node.id)
    await loadNodes()
    ElMessage.success(t('opsConsole.nodes.removePhysicalStarted'))
  } catch (error) {
    if (error !== 'cancel' && error !== 'close')
      ElMessage.error(apiErrorMessage(error, t('opsConsole.nodes.removeFailed')))
  }
}
async function runEnvironmentCheck() {
  await loadDashboard()
  if (selectedEnvironmentId.value) {
    await Promise.all([
      loadEnvironmentNodes(selectedEnvironmentId.value),
      loadEnvironmentEvents(selectedEnvironmentId.value),
      loadEnvironmentServices(selectedEnvironmentId.value),
    ])
  }
  ElMessage.success(t('opsConsole.environments.refreshed'))
}
function syncFoundationAssignment(row: { type: FoundationServiceType; nodeId: string }) {
  if (row.type !== 'if_history') return
  const timeseries = foundationAssignments.value.find((item) => item.type === 'if_timeseries')
  if (timeseries) timeseries.nodeId = row.nodeId
}
function startFoundationAdjustment() {
  foundationDefinitions.forEach(({ type }) => {
    foundationDraft[type] =
      foundationServices.value.find((item) => item.serviceType === type)?.nodeId || ''
  })
  foundationAdjusting.value = true
}
function cancelFoundationAdjustment() {
  foundationAdjusting.value = false
}
function syncFoundationDraft(type: FoundationServiceType) {
  if (type !== 'if_history' && type !== 'if_timeseries') return
  const value = foundationDraft[type]
  foundationDraft.if_history = value
  foundationDraft.if_timeseries = value
}
async function applyFoundationAdjustment() {
  const environment = selectedEnvironment.value
  if (!environment || foundationDefinitions.some(({ type }) => !foundationDraft[type])) return
  const changed = foundationChangedCount.value
  try {
    await ElMessageBox.confirm(
      t('opsConsole.foundation.migrationConfirm', { count: changed }),
      t('opsConsole.foundation.adjustDistribution'),
      { type: 'warning', confirmButtonText: t('opsConsole.foundation.checkAndApply') },
    )
    const result = await opsAPI.migrateRuntimeEnvironmentFoundation(
      environment.id,
      foundationDefinitions.map(({ type }) => ({
        serviceType: type,
        nodeId: foundationDraft[type]!,
      })),
    )
    foundationServices.value = result.items
    foundationAdjusting.value = false
    await Promise.all([
      loadEnvironments(),
      loadEnvironmentOptions(),
      loadEnvironmentEvents(environment.id),
    ])
    ElMessage.success(t('opsConsole.foundation.migrationCreated'))
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(apiErrorMessage(error, t('opsConsole.foundation.migrationFailed')))
  }
}
function openFoundationDialog() {
  if (!environmentNodes.value.length) {
    ElMessage.warning(t('opsConsole.foundation.noNodes'))
    return
  }
  foundationAssignments.value = foundationDefinitions.map((item, index) => {
    const deployed = foundationServices.value.find((service) => service.serviceType === item.type)
    return {
      ...item,
      nodeId:
        deployed?.nodeId || environmentNodes.value[index % environmentNodes.value.length]?.id || '',
    }
  })
  const historyNode = foundationAssignments.value.find((item) => item.type === 'if_history')?.nodeId
  const timeseries = foundationAssignments.value.find((item) => item.type === 'if_timeseries')
  if (timeseries && historyNode) timeseries.nodeId = historyNode
  foundationDialog.value = true
}
async function createFoundationTask() {
  if (foundationAssignments.value.some((item) => !item.nodeId)) {
    ElMessage.warning(t('opsConsole.foundation.chooseAll'))
    return
  }
  const environment = selectedEnvironment.value
  if (!environment) return
  try {
    const result = await opsAPI.deployRuntimeEnvironmentFoundation(
      environment.id,
      foundationAssignments.value.map((item) => ({ serviceType: item.type, nodeId: item.nodeId })),
    )
    foundationServices.value = result.items
    foundationDialog.value = false
    await Promise.all([
      loadEnvironments(),
      loadEnvironmentOptions(),
      loadEnvironmentEvents(environment.id),
    ])
    ElMessage.success(
      environment.foundationTotal
        ? t('opsConsole.foundation.repairCreated')
        : t('opsConsole.foundation.deployCreated'),
    )
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.foundation.createFailed')))
  }
}
function nodeEnvironmentName(node: OpsNode, index: number) {
  void index
  if (node.environmentNames?.length) return node.environmentNames.join('、')
  return node.environmentName
    ? t('opsConsole.nodes.joined', { name: node.environmentName })
    : t('opsConsole.nodes.notJoined')
}
function isStandardRuntimeNode(node: OpsNode) {
  return (
    node.platform === 'linux' &&
    node.capabilities.includes('project_entry') &&
    node.capabilities.includes('data_runtime')
  )
}
function nodeTypeLabel(node: OpsNode) {
  if (node.platform === 'windows') return t('opsConsole.nodes.windowsCollector')
  if (node.nodeKind === 'center') return t('opsConsole.nodes.centerBuiltIn')
  return isStandardRuntimeNode(node)
    ? t('opsConsole.nodes.standard')
    : t('opsConsole.nodes.collectorOnly')
}
function resourcePercent(node: OpsNode, key: 'cpu' | 'memory' | 'disk') {
  const value = node.resourceSummary?.[key]
  if (!value || typeof value !== 'object') return '—'
  const percent = Number((value as Record<string, unknown>).usedPercent)
  return Number.isFinite(percent) ? `${Math.round(percent)}%` : '—'
}
function resourcePercentNumber(node: OpsNode, key: 'cpu' | 'memory' | 'disk') {
  const value = resourcePercent(node, key)
  return value === '—' ? 0 : Number(value.replace('%', ''))
}

function timeSyncPresentation(node: OpsNode) {
  const raw = node.resourceSummary?.timeSync
  if (!raw || typeof raw !== 'object') {
    return {
      label: t('opsConsole.nodes.timeUnreported'),
      className: 'is-uninitialized',
      detail: t('opsConsole.nodes.timeUnreportedDetail'),
    }
  }
  const state = raw as Record<string, unknown>
  const status = String(state.status || 'unconfigured')
  const role = String(state.role || '')
  const offset = Number(state.offsetMillis)
  const message = String(state.message || '')
  if (status === 'synchronized') {
    return {
      label:
        role === 'center'
          ? t('opsConsole.nodes.timeCenter')
          : t('opsConsole.nodes.timeSynchronized'),
      className: 'is-available',
      detail:
        role === 'center'
          ? t('opsConsole.nodes.timeCenterDetail')
          : t('opsConsole.nodes.timeOffset', {
              value: Number.isFinite(offset) ? offset.toFixed(1) : '—',
            }),
    }
  }
  if (status === 'adjusting') {
    return {
      label: t('opsConsole.nodes.timeAdjusting'),
      className: 'is-running',
      detail: message || t('opsConsole.nodes.timeOffset', { value: offset.toFixed(1) }),
    }
  }
  return {
    label: t('opsConsole.nodes.timeAbnormal'),
    className: 'is-attention',
    detail: message || t('opsConsole.nodes.timeAbnormalDetail'),
  }
}

function toEnvironmentView(item: ApiRuntimeEnvironment): RuntimeEnvironment {
  return {
    id: item.id,
    name: item.name,
    code: item.code,
    isDefault: item.isDefault,
    status: item.status,
    nodeCount: item.nodeCount,
    nodeNames: [],
    onlineNodes: item.onlineNodeCount,
    foundationHealthy: item.foundationHealthy,
    foundationTotal: item.foundationTotal,
    projectCount: item.projectCount,
    recentChange: localizeRecentChange(item.recentChange),
    recentAt: item.recentAt ? formatDateTime(item.recentAt) : t('opsConsole.common.notReported'),
    recentBy: item.recentBy === '系统' ? t('opsConsole.common.system') : item.recentBy,
  }
}

function localizeRecentChange(value: string) {
  const keys: Record<string, string> = {
    默认运行范围已初始化: 'default_environment_created',
    物理节点已加入默认运行范围: 'node_auto_assigned',
    运行环境已创建: 'environment_created',
    运行环境信息已更新: 'environment_updated',
    运行环境删除任务已创建: 'environment_delete_requested',
    运行环境已删除: 'environment_deleted',
    物理节点已关联: 'node_added',
    物理节点移除任务已创建: 'node_remove_requested',
    物理节点已安全移除: 'node_removed',
    开始清理环境协调节点: 'coordinator_remove_requested',
    节点集群基础设施待初始化: 'cluster_plan_created',
    物理节点已移除: 'node_removed',
    基础服务部署任务已创建: 'foundation_deploy_requested',
    基础服务重新部署任务已创建: 'foundation_redeploy_requested',
    物理节点已恢复在线: 'node_online',
    物理节点心跳超时: 'node_offline',
    节点集群基础设施状态已更新: 'cluster_state_changed',
    基础服务部署完成: 'foundationReady',
    基础服务正在部署: 'foundationStarting',
    基础服务部署异常: 'foundationFailed',
  }
  const key = keys[value]
  return key ? t(`opsConsole.events.${key}`) : value
}

function toEnvironmentEvent(item: ApiRuntimeEnvironmentEvent): EnvironmentEvent {
  const eventKey = `opsConsole.events.${item.eventType}`
  const translatedName = t(eventKey)
  let message = item.message || ''
  if (item.eventType === 'node_offline') message = t('opsConsole.events.offlineMessage')
  if (item.eventType === 'node_online') message = t('opsConsole.events.onlineMessage')
  const knownMessages: Record<string, string> = {
    基础服务正在启动或等待健康检查: 'foundationWaiting',
    '已记录 8 项基础服务的单实例多节点分配，等待节点执行': 'foundationRequested',
    '保持现有节点分配并重新应用固定清单，用于修复异常或升级内置配置': 'foundationRepairRequested',
  }
  if (knownMessages[message]) message = t(`opsConsole.events.${knownMessages[message]}`)
  const target =
    item.target === '全部基础服务'
      ? t('opsConsole.events.allFoundation')
      : item.target === '运行环境'
        ? t('opsConsole.events.runtimeEnvironment')
        : item.target
  return {
    environmentId: item.environmentId,
    time: formatDateTime(item.createdAt),
    name: translatedName === eventKey ? item.name : translatedName,
    target,
    operator: item.operator === '系统' ? t('opsConsole.common.system') : item.operator,
    result:
      item.result === 'success' ? t('opsConsole.common.success') : t('opsConsole.common.failed'),
    success: item.result === 'success',
    message,
  }
}

function apiErrorMessage(error: unknown, fallback: string) {
  return error &&
    typeof error === 'object' &&
    'message' in error &&
    String((error as { message?: unknown }).message || '').trim()
    ? String((error as { message?: unknown }).message)
    : fallback
}
async function loadNodes() {
  loading.nodes = true
  try {
    const result = await opsAPI.listNodes({
      page: nodePage.page,
      pageSize: nodePage.limit,
      keyword: nodePage.keyword.trim() || undefined,
    })
    if (!disposed) {
      nodes.value = result.items
      nodePage.total = result.total
    }
  } finally {
    if (!disposed) loading.nodes = false
  }
}
async function loadEnvironments() {
  loading.environments = true
  try {
    const result = await opsAPI.listRuntimeEnvironments({
      page: environmentPage.page,
      pageSize: environmentPage.limit,
      keyword: environmentKeyword.value.trim() || undefined,
      status: environmentStatus.value === 'all' ? undefined : environmentStatus.value,
    })
    if (!disposed) {
      environments.value = result.items.map(toEnvironmentView)
      environmentPage.total = result.total
    }
  } finally {
    if (!disposed) loading.environments = false
  }
}
async function loadEnvironmentOptions() {
  const result = await opsAPI.listRuntimeEnvironments({ page: 1, pageSize: 200 })
  if (disposed) return
  environmentOptions.value = result.items.map(toEnvironmentView)
  environmentOptionTotal.value = result.total
  if (!environmentOptions.value.some((item) => item.id === selectedEnvironmentId.value)) {
    selectedEnvironmentId.value = defaultEnvironmentOption()?.id || ''
  }
}
async function loadEnvironmentNodes(environmentId: string) {
  loading.environmentNodes = true
  try {
    const result = await opsAPI.listRuntimeEnvironmentNodes(environmentId, {
      page: environmentNodePage.page,
      pageSize: environmentNodePage.limit,
    })
    if (!disposed && selectedEnvironmentId.value === environmentId) {
      environmentNodes.value = result.items
      environmentNodePage.total = result.total
      const environment = environments.value.find((item) => item.id === environmentId)
      if (environment) environment.nodeNames = result.items.map((node) => node.name)
    }
  } finally {
    if (!disposed) loading.environmentNodes = false
  }
}
async function loadEnvironmentEvents(environmentId: string) {
  loading.environmentEvents = true
  try {
    const result = await opsAPI.listRuntimeEnvironmentEvents(environmentId, {
      page: environmentEventPage.page,
      pageSize: environmentEventPage.limit,
    })
    if (!disposed && selectedEnvironmentId.value === environmentId) {
      environmentEvents.value = result.items.map(toEnvironmentEvent)
      environmentEventPage.total = result.total
    }
  } finally {
    if (!disposed) loading.environmentEvents = false
  }
}
async function loadEnvironmentServices(environmentId: string) {
  const result = await opsAPI.listRuntimeEnvironmentServices(environmentId)
  if (!disposed && selectedEnvironmentId.value === environmentId)
    foundationServices.value = result.items
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
    })
    if (!disposed) {
      enrollments.value = result.items
      enrollmentPage.total = result.total
    }
  } finally {
    if (!disposed) loading.enrollments = false
  }
}
async function loadPackages() {
  loading.packages = true
  try {
    const result = await opsAPI.listNodePackages()
    if (!disposed) packages.value = result.items
  } finally {
    if (!disposed) loading.packages = false
  }
}
async function loadDashboard() {
  const results = await Promise.allSettled([
    loadEnvironments(),
    loadEnvironmentOptions(),
    loadNodes(),
    loadDeployments(),
    loadEnrollments(),
    loadPackages(),
  ])
  if (!disposed && results.every((item) => item.status === 'rejected'))
    ElMessage.error(t('opsConsole.environments.loadAllFailed'))
}
async function refreshEnvironmentReconciliation() {
  const environment = selectedEnvironment.value
  if (
    !environment ||
    (environment.status !== 'deleting' &&
      !environmentNodes.value.some((node) => node.clusterDesiredAction === 'removing'))
  )
    return
  await Promise.all([loadEnvironments(), loadEnvironmentOptions()])
  const updated = environments.value.find((item) => item.id === environment.id)
  if (!updated) {
    closeEnvironment()
    ElMessage.success(t('opsConsole.environments.deleteSuccess'))
    return
  }
  await openEnvironment(updated)
}
function searchNodes() {
  nodePage.page = 1
  void loadNodes().catch((error) =>
    ElMessage.error(apiErrorMessage(error, t('opsConsole.nodes.queryFailed'))),
  )
}
function searchDeployments() {
  deploymentPage.page = 1
  void loadDeployments().catch((error) =>
    ElMessage.error(apiErrorMessage(error, t('opsConsole.deployments.queryFailed'))),
  )
}

function collectionFromProjectResponse(response: unknown): ProjectOption[] {
  if (!response || typeof response !== 'object') return []
  const root = response as Record<string, unknown>
  const data =
    root.data && typeof root.data === 'object' ? (root.data as Record<string, unknown>) : root
  const list = data.list
  if (Array.isArray(list)) return list as ProjectOption[]
  if (list && typeof list === 'object' && Array.isArray((list as Record<string, unknown>).items))
    return (list as { items: ProjectOption[] }).items
  return Array.isArray(data.items) ? (data.items as ProjectOption[]) : []
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
    ElMessage.error(apiErrorMessage(error, t('opsConsole.deployments.projectsFailed'))),
  )
}
async function onProjectChange(projectId: string) {
  deployForm.applicationVersionId = ''
  versions.value = []
  if (!projectId) return
  versionLoading.value = true
  try {
    const result = await opsAPI.listProjectVersions(projectId, { page: 1, pageSize: 50 })
    versions.value = result.items
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.deployments.releasesFailed')))
  } finally {
    versionLoading.value = false
  }
}
const isDeploymentNodeReady = (node: OpsNode) =>
  node.platform === 'linux' &&
  node.observedStatus === 'online' &&
  (!node.clusterStatus || node.clusterStatus === 'ready')

function deploymentNodeLabel(node: OpsNode) {
  const address = node.ipAddress ? ` · ${node.ipAddress}` : ''
  const unavailable = isDeploymentNodeReady(node)
    ? ''
    : ` · ${t('opsConsole.deployments.nodeUnavailable')}`
  return `${node.name}${address}${unavailable}`
}

function setDefaultDeploymentPlacements() {
  const readyNodes = deploymentNodes.value.filter(isDeploymentNodeReady)
  const readyNodeIds = new Set(readyNodes.map((node) => node.id))
  const firstNodeId = readyNodes[0]?.id || ''
  deploymentEngineRows.value.forEach(({ key }) => {
    if (!readyNodeIds.has(deployForm.placements[key])) deployForm.placements[key] = firstNodeId
  })
}

async function loadDeploymentNodes(environmentId: string) {
  deploymentNodes.value = []
  if (!environmentId) {
    setDefaultDeploymentPlacements()
    return
  }
  deploymentNodeLoading.value = true
  try {
    const result = await opsAPI.listRuntimeEnvironmentNodes(environmentId, {
      page: 1,
      pageSize: 200,
    })
    deploymentNodes.value = result.items
    setDefaultDeploymentPlacements()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.deployments.nodesFailed')))
  } finally {
    deploymentNodeLoading.value = false
  }
}
async function openDeployDialog(initialProjectId = '') {
  activeTab.value = 'deployments'
  selectedEnvironmentId.value = ''
  Object.assign(deployForm, {
    mode: 'development',
    projectId: '',
    applicationVersionId: '',
    environmentId: availableEnvironments.value[0]?.id || '',
    accessPort: 17800,
  })
  Object.assign(deployForm.placements, { base: '', compute: '', alarm: '', collector: '' })
  deploymentNodes.value = []
  versions.value = []
  deployDialog.value = true
  try {
    await loadProjects()
    await loadDeploymentNodes(deployForm.environmentId)
    if (initialProjectId && projects.value.some((item) => item.id === initialProjectId)) {
      deployForm.projectId = initialProjectId
      await onProjectChange(initialProjectId)
    }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.deployments.prepareFailed')))
  }
}
async function createDeployment() {
  if (
    !canCreateDeployment.value ||
    !selectedProject.value ||
    (deployForm.mode === 'production' && !selectedVersion.value)
  ) {
    ElMessage.warning(t('opsConsole.deployments.selectRequired'))
    return
  }
  submittingDeployment.value = true
  try {
    await opsAPI.createProjectDeployment({
      projectId: deployForm.projectId,
      environmentId: deployForm.environmentId,
      mode: deployForm.mode,
      applicationVersionId:
        deployForm.mode === 'production' ? deployForm.applicationVersionId : undefined,
      accessPort: deployForm.accessPort,
      placements: Object.fromEntries(
        deploymentEngineRows.value.map(({ key }) => [key, deployForm.placements[key]]),
      ),
    })
    deployDialog.value = false
    ElMessage.success(t('opsConsole.deployments.deployCreated'))
    await loadDeployments()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.deployments.createFailed')))
  } finally {
    submittingDeployment.value = false
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
function openDeploymentRow(row: DeploymentRow) {
  selectedDeploymentRow.value = row
  deploymentDrawer.value = true
}

function openEnrollmentDialog() {
  Object.assign(enrollForm, { displayName: '', platform: 'linux', ttlMinutes: 60 })
  enrollmentDialog.value = true
}
async function createEnrollment() {
  if (!enrollForm.displayName.trim()) {
    ElMessage.warning(t('opsConsole.nodes.nameMissing'))
    return
  }
  submittingEnrollment.value = true
  try {
    const item = await opsAPI.createEnrollment({
      platform: enrollForm.platform,
      capabilities: enrollmentCapabilities(enrollForm.platform),
      displayName: enrollForm.displayName.trim(),
      ttlMinutes: enrollForm.ttlMinutes,
    })
    const { enrollmentCode: code, ...enrollment } = item
    createdEnrollment.value = enrollment
    enrollmentCode.value = code || ''
    enrollmentServerUrl.value = defaultEnrollmentServerUrl
    enrollmentDialog.value = false
    codeDialog.value = true
    await loadEnrollments()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.nodes.generateFailed')))
  } finally {
    submittingEnrollment.value = false
  }
}
async function approve(item: NodeEnrollment) {
  const name = item.reportedHostName || item.displayName || item.id
  try {
    await ElMessageBox.confirm(
      t('opsConsole.nodes.approveConfirm', { name }),
      t('opsConsole.nodes.approveTitle'),
      {
        confirmButtonText: t('opsConsole.nodes.approveButton'),
        cancelButtonText: t('opsConsole.common.cancel'),
        type: 'warning',
      },
    )
    await opsAPI.approveEnrollment(item.id)
    await Promise.all([loadEnrollments(), loadNodes()])
    ElMessage.success(t('opsConsole.nodes.approveSuccess'))
  } catch (error) {
    if (error !== 'cancel' && error !== 'close')
      ElMessage.error(apiErrorMessage(error, t('opsConsole.nodes.approveFailed')))
  }
}
async function reject(item: NodeEnrollment) {
  const name = item.reportedHostName || item.displayName || item.id
  try {
    await ElMessageBox.confirm(
      t('opsConsole.nodes.rejectConfirm', { name }),
      t('opsConsole.nodes.rejectTitle'),
      {
        confirmButtonText: t('opsConsole.nodes.reject'),
        cancelButtonText: t('opsConsole.common.cancel'),
        type: 'error',
      },
    )
    await opsAPI.rejectEnrollment(item.id)
    await loadEnrollments()
    ElMessage.success(t('opsConsole.nodes.rejectSuccess'))
  } catch (error) {
    if (error !== 'cancel' && error !== 'close')
      ElMessage.error(apiErrorMessage(error, t('opsConsole.nodes.rejectFailed')))
  }
}
async function copyEnrollmentCode() {
  if (!enrollmentCode.value) return
  try {
    await navigator.clipboard.writeText(enrollmentCode.value)
    ElMessage.success(t('opsConsole.nodes.codeCopied'))
  } catch {
    ElMessage.warning(t('opsConsole.nodes.copyDenied'))
  }
}
async function copyInstallCommand() {
  if (!installCommand.value) return
  try {
    await navigator.clipboard.writeText(installCommand.value)
    ElMessage.success(t('opsConsole.nodes.commandCopied'))
  } catch {
    ElMessage.warning(t('opsConsole.nodes.copyDenied'))
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
    if (!(source instanceof Blob) && !(source instanceof ArrayBuffer))
      throw new Error(t('opsConsole.nodes.packageInvalid'))
    const blob = source instanceof Blob ? source : new Blob([source])
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = item.fileName || item.name || `node-agent-${item.platform}`
    link.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('opsConsole.nodes.packageDownloadFailed')))
  } finally {
    downloadingPackageId.value = ''
  }
}
const formatTime = (value?: string) => (value ? formatDateTime(value) : '')

onMounted(() => {
  disposed = false
  window.addEventListener('ops:open-deploy', handleOpenDeployEvent)
  void loadDashboard().then(() => {
    const environment = selectedEnvironment.value || defaultEnvironmentOption()
    if (canAdministerOperations.value && environment) void openEnvironment(environment)
    else if (canAdministerOperations.value) environmentManagementMode.value = true
  })
  environmentReconcileTimer = window.setInterval(
    () => void refreshEnvironmentReconciliation(),
    5000,
  )
})
onBeforeUnmount(() => {
  disposed = true
  if (environmentReconcileTimer) window.clearInterval(environmentReconcileTimer)
  window.removeEventListener('ops:open-deploy', handleOpenDeployEvent)
})
</script>

<style scoped>
.ops-console {
  height: calc(100vh - 32px);
  padding: 16px;
  background: var(--ck-bg-page, var(--el-bg-color-page));
  color: var(--el-text-color-primary);
}
.ops-workspace,
.ops-view,
.ops-page {
  min-width: 0;
  min-height: 0;
  height: 100%;
}
.ops-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ops-page {
  padding: 0;
}
.ops-toolbar,
.ops-toolbar__filters,
.ops-toolbar__actions,
.ops-detail-toolbar,
.ops-detail-heading,
.ops-detail-actions,
.ops-detail-title,
.ops-sub-toolbar,
.ops-node-selection,
.ops-service-summary {
  display: flex;
  align-items: center;
}
.ops-toolbar {
  justify-content: space-between;
  gap: 12px;
  min-height: 58px;
  padding: 10px 12px;
  border: 1px solid var(--ck-border, var(--el-border-color-lighter));
  border-radius: var(--ck-radius-lg, 14px);
  background: var(--ck-bg-card, var(--el-bg-color));
  box-shadow: var(--ck-shadow-sm, 0 2px 8px rgb(15 23 42 / 5%));
}
.ops-toolbar__filters,
.ops-toolbar__actions {
  gap: 8px;
  flex-wrap: wrap;
}
.ops-toolbar__divider {
  width: 1px;
  height: 24px;
  margin: 0 2px;
  background: var(--ck-border-light, var(--el-border-color-lighter));
}
.ops-toolbar__filters :deep(.el-input) {
  width: 230px;
}
.ops-toolbar__filters :deep(.el-select) {
  width: 156px;
}
.ops-toolbar :deep(.el-input__wrapper),
.ops-toolbar :deep(.el-select__wrapper) {
  min-height: 34px;
  border-radius: 9px;
  box-shadow: 0 0 0 1px var(--ck-border, var(--el-border-color)) inset;
}
.ops-toolbar :deep(.el-button) {
  min-height: 34px;
  border-radius: 9px;
}
.ops-list-panel :deep(.el-table),
.ops-detail-list :deep(.el-table),
.ops-detail-table :deep(.el-table) {
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-secondary);
}
.ops-list-panel :deep(.el-table th.el-table__cell),
.ops-detail-list :deep(.el-table th.el-table__cell),
.ops-detail-table :deep(.el-table th.el-table__cell) {
  height: 48px;
  color: var(--ck-text-secondary) !important;
  background-color: var(--ck-bg-tertiary, var(--el-fill-color-light)) !important;
  border-bottom: 1px solid var(--ck-border-light) !important;
  font-weight: 600;
}
.ops-list-panel :deep(.el-table td.el-table__cell),
.ops-detail-list :deep(.el-table td.el-table__cell),
.ops-detail-table :deep(.el-table td.el-table__cell) {
  border-bottom: 1px solid var(--ck-border-light);
}
.ops-list-panel :deep(.el-table__row:hover > td.el-table__cell),
.ops-detail-list :deep(.el-table__row:hover > td.el-table__cell),
.ops-detail-table :deep(.el-table__row:hover > td.el-table__cell) {
  background-color: var(--ck-bg-hover);
}
.ops-count,
.ops-muted,
.ops-detail-heading small,
.ops-primary-cell small,
.ops-name-cell small,
.ops-node-grid small,
.ops-event-row small,
.ops-form-help,
.ops-dialog-summary,
.ops-package-list small {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.ops-inline-note {
  margin-left: 4px;
  color: var(--el-text-color-secondary);
  font-weight: 400;
}
.ops-error-detail {
  color: var(--el-color-danger);
  font-size: 12px;
}
.ops-primary-cell,
.ops-name-cell span,
.ops-package-list article > div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.ops-name-cell {
  display: flex;
  align-items: center;
  gap: 11px;
}
.ops-name-title {
  display: flex !important;
  flex-direction: row !important;
  align-items: center;
  gap: 7px !important;
}
.ops-env-mark {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  border-radius: 6px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  font-style: normal;
  font-weight: 600;
}
.ops-clickable-row {
  cursor: pointer;
}
.ops-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
}
.ops-status::before {
  content: '';
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
}
.ops-status.is-available {
  color: var(--el-color-success);
}
.ops-status.is-attention {
  color: var(--el-color-warning);
}
.ops-status.is-running {
  color: var(--el-color-primary);
}
.ops-status.is-uninitialized {
  color: var(--el-text-color-secondary);
}
.ops-status.is-deleting {
  color: var(--el-color-warning-dark-2);
  background: var(--el-color-warning-light-9);
  border-color: var(--el-color-warning-light-7);
}
.ops-environment-detail {
  display: flex;
  overflow: hidden;
  padding: 0;
  gap: 0;
  border: 1px solid var(--ck-border, var(--el-border-color-lighter));
  border-radius: var(--ck-radius-lg, 14px);
  background: var(--ck-bg-card, var(--el-bg-color));
  box-shadow: var(--ck-shadow-sm, 0 2px 8px rgb(15 23 42 / 5%));
  font-size: 13px;
}
.ops-detail-header {
  flex: 0 0 auto;
  background: var(--ck-bg-card, var(--el-bg-color));
}
.ops-detail-toolbar {
  justify-content: space-between;
  gap: 16px;
  min-height: 64px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--ck-border-light, var(--el-border-color-lighter));
}
.ops-detail-heading {
  gap: 10px;
}
.ops-detail-actions {
  gap: 8px;
}
.ops-environment-switcher {
  width: 180px;
}
.ops-environment-switcher :deep(.el-select__wrapper) {
  min-height: 34px;
  border-radius: 9px;
}
.ops-detail-heading__meta {
  display: block;
}
.ops-detail-title {
  gap: 8px;
  margin-bottom: 1px;
}
.ops-detail-title h2 {
  margin: 0;
  font-size: 18px;
  line-height: 25px;
  font-weight: 600;
}
.ops-detail-heading__meta > small {
  color: var(--ck-text-tertiary, var(--el-text-color-secondary));
  font-size: 11px;
}
.ops-summary-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(140px, 1fr));
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--ck-bg-subtle, var(--el-fill-color-extra-light));
}
.ops-summary-strip > div {
  padding: 9px 16px;
}
.ops-summary-strip > div + div {
  border-left: 1px solid var(--el-border-color-lighter);
}
.ops-summary-strip span,
.ops-summary-strip strong {
  display: block;
}
.ops-summary-strip span {
  margin-bottom: 2px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
}
.ops-summary-strip strong {
  font-size: 14px;
  line-height: 20px;
  font-weight: 600;
}
.ops-detail-switcher {
  display: flex;
  gap: 24px;
  padding: 0 16px;
  border-bottom: 1px solid var(--ck-border-light, var(--el-border-color-lighter));
}
.ops-detail-switcher button {
  position: relative;
  height: 40px;
  padding: 0 2px;
  border: 0;
  color: var(--ck-text-secondary, var(--el-text-color-regular));
  background: transparent;
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}
.ops-detail-switcher button:hover {
  color: var(--ck-text-primary, var(--el-text-color-primary));
}
.ops-detail-switcher button:focus {
  outline: none;
}
.ops-detail-switcher button:focus-visible {
  color: var(--ck-primary, var(--el-color-primary));
  background: var(--ck-bg-hover, var(--el-fill-color-light));
}
.ops-detail-switcher button.is-active {
  color: var(--ck-primary, var(--el-color-primary));
  font-weight: 600;
}
.ops-detail-switcher button.is-active::after {
  position: absolute;
  right: 0;
  bottom: -1px;
  left: 0;
  height: 2px;
  border-radius: 2px 2px 0 0;
  background: var(--ck-primary, var(--el-color-primary));
  content: '';
}
.ops-guidance {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 12px;
  padding: 13px 16px;
  border: 1px solid;
  border-radius: 10px;
}
.ops-guidance--setup {
  display: block;
  border-color: var(--el-color-primary-light-7);
  background: var(--el-color-primary-light-9);
}
.ops-guidance--warning {
  border-color: var(--el-color-warning-light-7);
  background: var(--el-color-warning-light-9);
}
.ops-guidance__heading,
.ops-guidance__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.ops-guidance h3,
.ops-guidance p {
  margin: 0;
}
.ops-guidance h3 {
  margin-top: 3px;
  font-size: 14px;
  line-height: 20px;
  font-weight: 600;
}
.ops-guidance p {
  margin-top: 5px;
  color: var(--ck-text-secondary, var(--el-text-color-regular));
  font-size: 12px;
}
.ops-guidance__eyebrow {
  color: var(--ck-text-secondary, var(--el-text-color-regular));
  font-size: 11px;
}
.ops-setup-steps {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0;
  margin: 18px 0 0;
  padding: 0;
  list-style: none;
}
.ops-setup-steps li {
  position: relative;
  display: grid;
  grid-template-columns: 26px minmax(0, 1fr);
  grid-template-rows: auto auto;
  column-gap: 8px;
  color: var(--ck-text-secondary, var(--el-text-color-regular));
}
.ops-setup-steps li:not(:last-child)::after {
  position: absolute;
  top: 13px;
  right: 10px;
  left: 34px;
  height: 1px;
  background: var(--el-border-color);
  content: '';
}
.ops-setup-steps li > span {
  z-index: 1;
  display: grid;
  grid-row: 1 / 3;
  place-items: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--el-border-color);
  border-radius: 50%;
  background: var(--el-bg-color);
  font-size: 12px;
}
.ops-setup-steps li strong,
.ops-setup-steps li small {
  z-index: 1;
  width: max-content;
  padding-right: 8px;
  background: var(--el-color-primary-light-9);
}
.ops-setup-steps li small {
  font-size: 11px;
}
.ops-setup-steps li.is-current,
.ops-setup-steps li.is-complete {
  color: var(--el-color-primary);
}
.ops-setup-steps li.is-current > span,
.ops-setup-steps li.is-complete > span {
  border-color: var(--el-color-primary);
}
.ops-setup-steps li.is-complete > span {
  color: white;
  background: var(--el-color-primary);
}
.ops-detail-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background: var(--ck-bg-page, var(--el-bg-color-page));
}
.ops-detail-pane {
  height: 100%;
  min-height: 0;
}
.ops-detail-pane--overview,
.ops-detail-pane--services {
  overflow-y: auto;
  padding: 14px 16px 16px;
}
.ops-detail-pane--table {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.ops-overview-layout {
  display: grid;
  gap: 12px;
}
.ops-overview-secondary {
  display: grid;
  grid-template-columns: minmax(0, 1.7fr) minmax(280px, 0.72fr);
  align-items: start;
  gap: 12px;
}
.ops-section {
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-bg-color);
}
.ops-section + .ops-section {
  margin-top: 0;
}
.ops-section > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 42px;
  padding: 0 14px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.ops-section h3 {
  margin: 0;
  font-size: 14px;
  line-height: 20px;
  font-weight: 600;
}
.ops-health-row {
  display: grid;
  grid-template-columns: minmax(190px, 1.2fr) minmax(180px, 1fr) auto;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  padding: 0 14px;
}
.ops-health-row + .ops-health-row {
  border-top: 1px solid var(--el-border-color-lighter);
}
.ops-health-row > span:first-child {
  display: flex;
  flex-direction: column;
}
.ops-health-row strong,
.ops-node-grid article > div > strong,
.ops-event-row strong {
  font-size: 13px;
  line-height: 18px;
  font-weight: 600;
}
.ops-health-row small {
  color: var(--el-text-color-secondary);
  font-size: 11px;
}
.ops-node-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  padding: 12px 14px;
}
.ops-node-grid article {
  padding: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-fill-color-extra-light);
}
.ops-node-grid article > div {
  display: flex;
  justify-content: space-between;
  gap: 8px;
}
.ops-node-metrics {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin: 11px 0 0;
}
.ops-node-metrics div {
  min-width: 0;
}
.ops-node-metrics dt {
  color: var(--ck-text-tertiary, var(--el-text-color-secondary));
  font-size: 11px;
}
.ops-node-metrics dd {
  margin: 2px 0 0;
  color: var(--ck-text-primary, var(--el-text-color-primary));
  font-size: 12px;
  font-weight: 600;
}
.ops-event-row {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  min-height: 54px;
  padding: 0 12px;
}
.ops-event-row + .ops-event-row {
  border-top: 1px solid var(--el-border-color-lighter);
}
.ops-event-row > span:nth-child(2) {
  display: flex;
  flex-direction: column;
}
.ops-event-row small {
  font-size: 11px;
}
.ops-event-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--el-color-success);
}
.ops-event-dot.is-warning {
  background: var(--el-color-warning);
}
.ops-detail-list {
  height: 100%;
  min-height: 0;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  backdrop-filter: none;
}
.ops-detail-list > .ck-content-scroll {
  padding: 16px;
}
.ops-detail-table {
  overflow: hidden;
  border: 1px solid var(--ck-border, var(--el-border-color-lighter));
  border-radius: var(--ck-radius-md, 10px);
}
.ops-sub-toolbar {
  justify-content: space-between;
  margin-bottom: 10px;
}
.ops-deploy-steps {
  margin-bottom: 18px;
}
.ops-mode-label {
  margin-bottom: 8px;
  font-weight: 600;
}
.ops-mode-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 14px;
}
.ops-mode-option {
  min-height: 66px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  text-align: left;
  cursor: pointer;
}
.ops-mode-option.is-selected {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.ops-mode-option:disabled {
  color: var(--el-text-color-placeholder);
  background: var(--el-fill-color-light);
  cursor: not-allowed;
}
.ops-mode-option strong,
.ops-mode-option small {
  display: block;
}
.ops-mode-option small {
  margin-top: 4px;
  font-size: 12px;
}
.ops-mode-option em {
  float: right;
  font-size: 11px;
  font-style: normal;
  font-weight: 400;
}
.ops-node-selection {
  gap: 12px;
  flex-wrap: wrap;
  margin: 16px 0 10px;
}
.ops-node-selection > span {
  margin-right: 6px;
  font-weight: 600;
}
.ops-distribution {
  margin-top: 12px;
  padding: 10px 12px;
  color: var(--el-text-color-regular);
  background: var(--el-color-primary-light-9);
  font-size: 12px;
}
.ops-delete-impact {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin: 16px 0;
}
.ops-delete-impact > div {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  border: 1px solid var(--ck-border, var(--el-border-color));
  border-radius: 8px;
  background: var(--ck-bg-subtle, var(--el-fill-color-lighter));
}
.ops-delete-impact span {
  color: var(--ck-text-muted, var(--el-text-color-secondary));
  font-size: 12px;
}
.ops-delete-impact strong {
  color: var(--ck-text-primary, var(--el-text-color-primary));
  font-size: 18px;
}
.ops-dialog-summary {
  float: left;
  margin-top: 8px;
}
.ops-package-list article {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 0;
}
.ops-package-list article + article {
  border-top: 1px solid var(--el-border-color-lighter);
}
.ops-secret,
.ops-command {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 10px 0 16px;
  padding: 14px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  background: var(--el-fill-color-light);
}
.ops-secret code,
.ops-command code {
  flex: 1;
  overflow-wrap: anywhere;
}
.ops-form-help {
  display: block;
  margin-top: 5px;
}
.ops-service-summary {
  gap: 8px;
  flex-wrap: wrap;
}
.ops-service-summary :deep(.el-checkbox) {
  margin-left: 8px;
}
.ops-engine-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}
:global(.ops-deployment-dialog.el-dialog) {
  overflow: hidden;
  border: 1px solid var(--ck-border);
  border-radius: var(--ck-radius-lg);
  background: var(--ck-bg-secondary);
  box-shadow: var(--ck-shadow-lg);
}
:global(.ops-deployment-dialog .el-dialog__header) {
  margin: 0;
  padding: 16px 20px 12px;
}
:global(.ops-deployment-dialog .el-dialog__title) {
  color: var(--ck-text-primary);
  font-size: 15px;
  font-weight: 600;
}
:global(.ops-deployment-dialog .el-dialog__headerbtn) {
  top: 8px;
  right: 10px;
}
:global(.ops-deployment-dialog .el-dialog__body) {
  height: 468px;
  padding: 4px 20px 18px;
  overflow: hidden;
}
:global(.ops-deployment-dialog .el-dialog__footer) {
  padding: 12px 20px 14px;
  border-top: 1px solid var(--ck-border-light);
}
.ops-deploy-mode-tabs {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 3px;
  padding: 3px;
  margin-bottom: 16px;
  border-radius: 10px;
  background: rgb(15 23 42 / 4%);
}
.ops-deploy-mode-tab {
  height: 32px;
  border: 0;
  border-radius: 8px;
  color: var(--ck-text-muted);
  font-size: 13px;
  font-weight: 400;
  background: transparent;
  cursor: pointer;
  transition:
    color 0.15s ease,
    background-color 0.15s ease,
    box-shadow 0.15s ease;
}
.ops-deploy-mode-tab:hover {
  color: var(--ck-text-primary);
  background: rgb(255 255 255 / 55%);
}
.ops-deploy-mode-tab:focus-visible {
  outline: none;
  box-shadow: inset 0 0 0 2px var(--ck-primary-light);
}
.ops-deploy-mode-tab.is-active {
  color: var(--ck-primary);
  background: #ffffff;
  box-shadow: 0 2px 8px rgb(15 23 42 / 8%);
}
.ops-deployment-form {
  height: 390px;
  max-height: calc(100vh - 220px);
  overflow-y: auto;
  box-sizing: border-box;
}
.ops-deployment-mode-hint {
  margin: 0 0 12px;
  color: var(--ck-text-muted);
  font-size: 12px;
}
.ops-deployment-form--release {
  height: 428px;
}
.ops-deployment-field {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.ops-deployment-field label {
  color: var(--ck-text-secondary);
  font-size: 12px;
  font-weight: 500;
}
.ops-deployment-field :deep(.el-select),
.ops-deployment-port :deep(.el-input-number) {
  width: 100%;
}
.ops-deployment-field :deep(.el-select__wrapper),
.ops-deployment-port :deep(.el-input__wrapper),
.ops-engine-placement :deep(.el-select__wrapper) {
  width: 100%;
  min-width: 0;
  height: 34px;
  min-height: 34px;
  max-height: 34px;
  box-sizing: border-box;
  border: 1px solid var(--ck-border);
  border-radius: 8px;
  background: var(--ck-bg-card);
  box-shadow: none;
}
.ops-deployment-field :deep(.el-select__wrapper.is-focused),
.ops-deployment-field :deep(.el-select__wrapper.is-hovering),
.ops-deployment-port :deep(.el-input__wrapper.is-focus),
.ops-engine-placement :deep(.el-select__wrapper.is-focused),
.ops-engine-placement :deep(.el-select__wrapper.is-hovering) {
  height: 34px;
  min-height: 34px;
  max-height: 34px;
}
.ops-deployment-field :deep(.el-select__wrapper:hover),
.ops-deployment-port :deep(.el-input__wrapper:hover),
.ops-engine-placement :deep(.el-select__wrapper:hover) {
  border-color: var(--ck-primary);
}
.ops-deployment-port {
  display: grid;
  grid-template-columns: 160px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}
.ops-deployment-port small {
  color: var(--ck-text-muted);
  font-size: 11px;
}
.ops-engine-placement {
  overflow: hidden;
  border: 1px solid rgb(15 23 42 / 6%);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-tertiary);
}
.ops-engine-placement__header,
.ops-engine-placement__row {
  display: grid;
  grid-template-columns: minmax(170px, 0.72fr) minmax(240px, 1.28fr);
  align-items: center;
  column-gap: 12px;
}
.ops-engine-placement__header {
  padding: 8px 12px;
  color: var(--ck-text-muted);
  font-size: 11px;
  background: transparent;
}
.ops-engine-placement__row {
  min-height: 52px;
  padding: 7px 12px;
  border-top: 1px solid var(--ck-border-light);
  background: var(--ck-bg-card);
}
.ops-engine-placement__row > div {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.ops-engine-placement__row strong {
  color: var(--ck-text-secondary);
  font-size: 12px;
  font-weight: 500;
}
.ops-engine-placement :deep(.el-switch) {
  --el-switch-on-color: var(--ck-primary);
}
.ops-engine-placement :deep(.el-select) {
  width: 100%;
  min-width: 0;
}
.ops-deployment-footer {
  display: flex;
  align-items: center;
  width: 100%;
}
.ops-deployment-footer__spacer {
  flex: 1;
}
.ops-deployment-footer :deep(.el-button) {
  min-width: 72px;
  height: 32px;
  border-radius: var(--ck-radius-md);
  font-size: 12px;
}
.ops-deployment-footer :deep(.ops-deployment-confirm) {
  border-color: transparent;
  background: var(--ck-gradient-primary);
  box-shadow: 0 8px 16px rgb(29 78 216 / 18%);
}
:global(html.dark) .ops-deploy-mode-tabs,
:global([data-theme='dark']) .ops-deploy-mode-tabs {
  background: rgb(255 255 255 / 6%);
}
:global(html.dark) .ops-deploy-mode-tab:hover,
:global([data-theme='dark']) .ops-deploy-mode-tab:hover {
  background: rgb(255 255 255 / 8%);
}
:global(html.dark) .ops-deploy-mode-tab.is-active,
:global([data-theme='dark']) .ops-deploy-mode-tab.is-active {
  color: #ffffff;
  background: #334155;
}
.ops-drawer-heading {
  margin: 24px 0 16px;
}
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
}
@media (max-width: 980px) {
  .ops-overview-secondary {
    grid-template-columns: 1fr;
  }
  .ops-summary-strip {
    grid-template-columns: repeat(2, 1fr);
  }
  .ops-summary-strip > div:nth-child(3) {
    border-left: 0;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .ops-summary-strip > div:nth-child(4) {
    border-top: 1px solid var(--el-border-color-lighter);
  }
}
@media (max-width: 720px) {
  .ops-console {
    padding: 12px;
  }
  .ops-toolbar,
  .ops-detail-toolbar,
  .ops-guidance,
  .ops-guidance__heading {
    align-items: stretch;
    flex-direction: column;
  }
  .ops-toolbar__filters :deep(.el-input),
  .ops-toolbar__filters :deep(.el-select) {
    width: 100%;
  }
  .ops-toolbar__actions {
    justify-content: flex-end;
  }
  .ops-detail-actions,
  .ops-guidance__actions {
    justify-content: flex-end;
  }
  .ops-setup-steps {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    row-gap: 16px;
  }
  .ops-mode-grid {
    grid-template-columns: 1fr;
  }
  .ops-detail-switcher {
    max-width: 100%;
    overflow-x: auto;
  }
  .ops-node-grid {
    grid-template-columns: 1fr;
  }
  .ops-engine-placement__header {
    display: none;
  }
  .ops-engine-placement__row {
    grid-template-columns: minmax(100px, 0.78fr) minmax(0, 1.22fr);
    column-gap: 8px;
    padding-inline: 10px;
  }
  .ops-deployment-field {
    grid-template-columns: 1fr;
    gap: 6px;
  }
  .ops-deployment-port {
    grid-template-columns: 1fr;
    gap: 5px;
  }
}
</style>
