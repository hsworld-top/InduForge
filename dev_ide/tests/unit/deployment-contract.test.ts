import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const consoleSource = readFileSync('src/views/tenant/OpsManagementConsole.vue', 'utf8')
const publishSource = readFileSync(
  'src/views/tenant/project-management/ProjectPublishDialog.vue',
  'utf8',
)

describe('工程部署新契约', () => {
  it('零个、一个和多个运行环境分别引导、自动选择和显示切换器', () => {
    expect(consoleSource).toContain('v-if="availableEnvironments.length === 0"')
    expect(consoleSource).toContain("environmentId: availableEnvironments.value[0]?.id || ''")
    expect(consoleSource).toContain('v-if="availableEnvironments.length > 1"')
    expect(consoleSource).toContain('v-if="environmentOptionTotal > 1"')
    expect(consoleSource).toContain('else if (canAdministerOperations.value) environmentManagementMode.value = true')
    expect(publishSource).toContain('v-if="availableEnvironments.length > 1"')
  })

  it('基础引擎恒定，三个可选引擎只由 capabilities 的四种组合推导', () => {
    expect(consoleSource).toContain("{ key: 'base', label: t('opsConsole.deployments.baseEngine') }")
    for (const capability of ['compute', 'alarm', 'collector']) {
      expect(consoleSource).toContain(`deploymentCapabilities.value.has(key)`)
      expect(publishSource).toContain('selectedCapabilities.value.has(key)')
      expect(publishSource).toContain(`'${capability}'`)
    }
    expect(consoleSource).not.toContain('enableCollector')
    expect(publishSource).not.toContain('engine.optional')
  })

  it('生产仅选择 ready 版本，开发不显示版本且使用 development/production 请求值', () => {
    expect(consoleSource).toContain("item.status === 'ready'")
    expect(publishSource).toContain("item.status === 'ready'")
    expect(consoleSource).toContain("v-if=\"deployForm.mode === 'production'\"")
    expect(consoleSource).toContain("v-if=\"deployForm.mode === 'development'\"")
    expect(publishSource).toContain("mode: mode.value === 'RELEASE' ? 'production' : 'development'")
    expect(consoleSource).toContain('mode: deployForm.mode')
  })

  it('已有同工程同环境部署明确呈现更新语义，两个弹窗高度固定', () => {
    expect(consoleSource).toContain('item.projectId === deployForm.projectId && item.environmentId === deployForm.environmentId')
    expect(publishSource).toContain('item.projectId === props.project?.id && item.environmentId === environmentId.value')
    expect(consoleSource).toContain("$t('opsConsole.deployments.updateDeployment')")
    expect(publishSource).toContain("? '更新部署'")
    expect(consoleSource).toMatch(/\.ops-deployment-dialog \.el-dialog__body\) \{[\s\S]*height: 468px;/)
    expect(publishSource).toMatch(/\.project-publish-dialog \.el-dialog__body\) \{[\s\S]*height: 372px;/)
  })
})
