import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const source = readFileSync('src/views/tenant/OpsManagementConsole.vue', 'utf8')
const switcherSource = readFileSync('src/views/tenant/components/OpsSectionSwitcher.vue', 'utf8')
const dashboardSource = readFileSync('src/views/Dashboard.vue', 'utf8')
const langSource = readFileSync('src/lang/index.ts', 'utf8')

describe('ops management console', () => {
  it('使用工具栏下拉切换运行环境、物理节点、工程部署和运维任务', () => {
    for (const section of ['environments', 'nodes', 'deployments', 'tasks']) {
      expect(switcherSource).toContain(`t('opsConsole.sections.${section}')`)
    }
    expect(source).toContain('<OpsSectionSwitcher')
    expect(source).not.toContain('ops-navigation')
    expect(switcherSource).toContain("t('opsConsole.sections.switchAria'")
    expect(source).not.toContain('ops-primary-tabs')
    expect(source).not.toContain('<el-tabs')
    expect(source).not.toContain('存储与备份')
    expect(dashboardSource).toContain("import('@/views/tenant/OpsManagementConsole.vue')")
  })

  it('新建运行环境只填写名称，标识和物理节点由后续流程处理', () => {
    expect(source).toContain("const environmentForm = reactive({ name: '' })")
    expect(source).toContain(':label="$t(\'opsConsole.environments.createName\')"')
    expect(source).not.toContain('environmentForm.code')
    expect(source).not.toContain('environmentForm.nodeNames')
    expect(source).toContain(
      'opsAPI.createRuntimeEnvironment({ name: environmentForm.name.trim() })',
    )
    expect(source).toContain('code: item.code')
  })

  it('四个可增长主列表使用用户管理相同的表格壳与固定分页条', () => {
    expect(source.match(/class="ops-list-panel ck-content-area"/g)).toHaveLength(4)
    expect(source.match(/class="ck-table-shell"/g)).toHaveLength(6)
    expect(source.match(/class="ck-pagination-bar"/g)).toHaveLength(6)
    expect(source).toContain("t('opsConsole.pagination.summary'")
    expect(source).toContain("t('opsConsole.pagination.page'")
  })

  it('运行环境详情根据正常、异常和未初始化状态提供不同内容', () => {
    expect(source).toContain("selectedEnvironment.status === 'uninitialized'")
    expect(source).toContain("selectedEnvironment.status === 'attention'")
    expect(source).toContain("selectedEnvironment.status === 'attention' && foundationDeploying")
    expect(source).toContain("$t('opsConsole.environments.deploymentInProgressTitle')")
    expect(source).toContain("$t('opsConsole.environments.setup')")
    expect(source).toContain('environmentProblems')
    expect(source).toContain("t('opsConsole.environments.issueBlock'")
    expect(source).toContain('environmentEvents.value.filter')
    expect(source).toContain('opsAPI.listRuntimeEnvironmentEvents')
  })

  it('运维入口直达默认运行概览，只有多个运行范围时显示切换器', () => {
    expect(source).toContain('const environmentManagementMode = ref(false)')
    expect(source).toContain('v-if="environmentOptionTotal > 1"')
    expect(source).toContain('environment.isDefault')
    expect(source).toContain('defaultEnvironmentOption()')
    expect(source).toContain('command="manage"')
    expect(source).not.toContain('ops-back-button')
    expect(source).not.toContain('ops-breadcrumb')
    expect(source).toContain("$t('opsConsole.environments.addNode')")
    expect(source).toContain("t('opsConsole.environments.setupWithNodes')")
    expect(source).toContain("$t('opsConsole.foundation.strategy')")
    expect(source).toContain("$t('opsConsole.environments.maxDisk'")
  })

  it('环境编辑、受控删除和节点安全移除都有真实交互', () => {
    expect(source).toContain('@command="handleEnvironmentCommand"')
    expect(source).toContain('opsAPI.updateRuntimeEnvironment')
    expect(source).toContain('opsAPI.deleteRuntimeEnvironment')
    expect(source).toContain('environmentDeleteConfirmation !== selectedEnvironment?.name')
    expect(source).toContain("selectedEnvironment.status === 'deleting'")
    expect(source).toContain("node.clusterDesiredAction === 'removing'")
    expect(source).toContain('opsAPI.removeRuntimeEnvironmentNode')
    expect(source).toContain(':disabled="row.unassignDisabled"')
    expect(source).toContain("t('opsConsole.nodes.unassignBlockedFoundation'")
    expect(source).toContain("t('opsConsole.nodes.unassignBlockedDeployment'")
  })

  it('详情表格占满主体高度并将节点和事件分页固定在底部', () => {
    expect(source).toContain('class="ops-detail-header"')
    expect(source.match(/class="ops-detail-pane ops-detail-pane--table"/g)).toHaveLength(2)
    expect(source).toMatch(/\.ops-detail-content \{[\s\S]*flex: 1;[\s\S]*min-height: 0;/)
    expect(source).toMatch(/\.ops-detail-list \{[\s\S]*height: 100%;[\s\S]*min-height: 0;/)
  })

  it('详情页使用紧凑且一致的字体层级', () => {
    expect(source).toMatch(/\.ops-detail-title h2 \{[\s\S]*font-size: 18px;/)
    expect(source).toMatch(/\.ops-section h3 \{[\s\S]*font-size: 14px;/)
    expect(source).toMatch(/\.ops-environment-detail \{[\s\S]*font-size: 13px;/)
    expect(source).toMatch(/\.ops-summary-strip span \{[\s\S]*font-size: 11px;/)
  })

  it('基础服务当前按单实例跨节点分配，并预留不可用的主备和集群模式', () => {
    expect(source).toContain("$t('opsConsole.foundation.singleDesc')")
    expect(source).toMatch(
      /class="ops-mode-option" disabled>[\s\S]*opsConsole\.foundation\.standby/,
    )
    expect(source).toMatch(
      /class="ops-mode-option" disabled>[\s\S]*opsConsole\.foundation\.cluster/,
    )
    expect(source).toContain("$t('opsConsole.foundation.dataPathHint')")
    expect(source).toContain(':disabled="isFoundationRepair || row.type === \'if_timeseries\'"')
  })

  it('工程部署与工程快捷入口使用一致的模式和运行引擎分配语义', () => {
    expect(source).toContain("deployForm.mode === 'DEV'")
    expect(source).toContain("deployForm.mode === 'RELEASE'")
    expect(source).toContain("<label>{{ $t('opsConsole.deployments.targetEnvironment') }}</label>")
    expect(source).toMatch(
      /<div class="ops-deployment-field">\s*<label>\{\{ \$t\('opsConsole\.deployments\.project'\) \}\}<\/label>/,
    )
    expect(source).toMatch(
      /<div v-if="availableEnvironments\.length > 1" class="ops-deployment-field">\s*<label>\{\{ \$t\('opsConsole\.deployments\.targetEnvironment'\) \}\}<\/label>/,
    )
    expect(source).toContain('opsAPI.listRuntimeEnvironmentNodes(environmentId')
    expect(source).toContain("node.platform === 'linux'")
    expect(source).toContain("node.observedStatus === 'online'")
    expect(source).toContain("node.clusterStatus === 'ready'")
    expect(source).toContain("key: 'alarm' as const")
    expect(source).toContain("key: 'collection' as const")
  })

  it('创建工程部署弹窗沿用发布工程弹窗的家族化视觉规则', () => {
    expect(source).toContain('class="ops-deployment-dialog"')
    expect(source).toContain('class="ops-deployment-field"')
    expect(source).toContain('class="ops-deployment-footer"')
    expect(source).toContain('class="ops-deployment-confirm"')
    expect(source).toContain("$t('opsConsole.deployments.deployDevelopment')")
    expect(source).toContain("$t('opsConsole.deployments.deployProduction')")
    expect(source).toContain('background: var(--ck-gradient-primary)')
    expect(source).toContain("'ops-deployment-form--release': deployForm.mode === 'RELEASE'")
    expect(source).toMatch(
      /\.ops-deployment-form \{[\s\S]*height: 390px;[\s\S]*max-height: calc\(100vh - 220px\);[\s\S]*overflow-y: auto;/,
    )
    expect(source).toMatch(/\.ops-deployment-form--release \{[\s\S]*height: 428px;/)
    expect(source).toMatch(
      /\.ops-engine-placement :deep\(\.el-select\) \{[\s\S]*width: 100%;[\s\S]*min-width: 0;/,
    )
    expect(source).toMatch(
      /\.ops-engine-placement :deep\(\.el-select__wrapper\) \{[\s\S]*width: 100%;[\s\S]*height: 34px;[\s\S]*min-height: 34px;[\s\S]*max-height: 34px;[\s\S]*box-sizing: border-box;/,
    )
    expect(source).toContain('.ops-deployment-field :deep(.el-select__wrapper.is-focused)')
    expect(source).toContain('.ops-deployment-port :deep(.el-input__wrapper.is-focus)')
    expect(source).toContain('grid-template-columns: minmax(100px, 0.78fr) minmax(0, 1.22fr)')
    for (const engine of ['baseEngine', 'computeEngine', 'alarmEngine', 'collectionEngine']) {
      expect(source).toContain(`t('opsConsole.deployments.${engine}')`)
    }
  })

  it('工程访问端口使用适合双列布局的简短提示', () => {
    expect(langSource).toContain("accessPortHint: '平台将自动检测端口冲突。'")
    expect(langSource).toContain(
      "accessPortHint: 'The platform automatically checks for port conflicts.'",
    )
  })

  it('工程部署列表以紧凑标签展示四类运行引擎并保持固定分页', () => {
    expect(source).toContain("$t('opsConsole.deployments.runtimeEngines')")
    expect(source).toContain('class="ops-engine-tags"')
    expect(source).toContain("t('opsConsole.deployments.alarm')")
    expect(source).toContain("t('opsConsole.deployments.collection')")
  })

  it('区分接入物理节点与将已有节点加入运行环境', () => {
    expect(source).toContain('@click="openEnvironmentNodeDialog"')
    expect(source).toContain('@click="openEnrollmentDialog"')
    expect(source).toContain("$t('opsConsole.nodes.windowsHint')")
  })

  it('按角色隐藏运行环境和物理节点管理入口', () => {
    expect(source).toContain("canAdministerOperations && activeTab === 'environments'")
    expect(source).toContain("canAdministerOperations && activeTab === 'nodes'")
    expect(source).toContain("can(authStore.userInfo?.role, 'runtime:operate')")
    expect(switcherSource).toContain('props.canAdministerOperations || !section.administratorOnly')
  })

  it('只展示后端真实状态，不制造前端任务或工程部署结果', () => {
    expect(source).toContain('const tasks = ref<TaskRow[]>([])')
    expect(source).not.toContain('localDeployments')
    expect(source).toContain("ElMessage.info(t('opsConsole.deployments.nextPhase'))")
    expect(source).toContain('foundation_redeploy_requested')
  })
})
