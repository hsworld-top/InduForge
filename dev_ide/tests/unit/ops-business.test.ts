import { createApp, nextTick } from 'vue'
import { describe, it, expect } from 'vitest'
import OpsMessage from '@/views/tenant/components/OpsMessage.vue'
import { opsMessageSummary } from '@/views/tenant/utils/ops-business'
import { runEventPresentation } from '@/views/tenant/utils/ops-presentation'
describe('运维业务摘要与按需诊断', () => {
  it('未知技术错误不伪装成功，已知错误保留业务原因', () => {
    expect(runEventPresentation('queued', 'start deployment queued').message).toBe(
      '启动任务已创建，等待目标节点执行',
    )
    expect(runEventPresentation('ready', '工程部署操作已结束：stopped').message).toBe(
      '工程部署操作已结束：已停止',
    )
    expect(opsMessageSummary('Kubernetes error /etc/secret/config', 'error')).toContain(
      '运行状态异常',
    )
    expect(opsMessageSummary('Pod ImagePullBackOff', 'error')).toBe('运行组件获取失败')
    expect(opsMessageSummary('Pod CrashLoopBackOff', 'error')).toBe('服务反复启动失败')
    expect(runEventPresentation('failed', 'traefik Pod failed').stage).toBe('执行失败')
    expect(runEventPresentation('observed', 'k3s Pod pending').message).not.toMatch(/k3s|Pod/)
  })
  it('实际组件默认DOM及提示不包含技术原文，主动展开后才创建诊断内容', async () => {
    const host = document.createElement('div')
    document.body.appendChild(host)
    const original = 'nginx Pod failed at /etc/runtime/config'
    const app = createApp(OpsMessage, { text: original, kind: 'error' })
    app.mount(host)
    expect(host.textContent).toContain('运行状态异常')
    expect(host.textContent).not.toContain('nginx')
    expect(host.querySelector('pre')).toBeNull()
    const details = host.querySelector('details')!
    details.open = true
    details.dispatchEvent(new Event('toggle'))
    await nextTick()
    expect(host.querySelector('pre')?.textContent).toBe(original)
    details.open = false
    details.dispatchEvent(new Event('toggle'))
    await nextTick()
    expect(host.querySelector('pre')).toBeNull()
    app.unmount()
    host.remove()
  })
})
