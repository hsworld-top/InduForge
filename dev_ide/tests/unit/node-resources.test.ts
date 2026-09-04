import { createApp, nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import type { OpsNode } from '@/api/ops.api'
import { nodeResourcePercent } from '@/utils/node-resources'
import OpsNodeResources from '@/views/tenant/components/OpsNodeResources.vue'

const node = (value: unknown, status = 'online') =>
  ({
    id: 'node-1',
    name: '测试节点',
    platform: 'linux',
    capabilities: [],
    observedStatus: status,
    resourceSummary: { disk: { usedPercent: value } },
  }) as OpsNode

describe('节点最近上报资源', () => {
  it('只接受范围内真实数值，缺失及无效指标不制造0或100', () => {
    for (const value of [undefined, null, '', '95', true, -1, 101, NaN, Infinity]) {
      expect(nodeResourcePercent(node(value), 'disk')).toBeNull()
    }
    expect(nodeResourcePercent(node(0), 'disk')).toBe(0)
    expect(nodeResourcePercent(node(84.659), 'disk')).toBe(84.65)
    expect(nodeResourcePercent(node(85), 'disk')).toBe(85)
  })
  it('真实组件显示缺失横杠与磁盘小数，离线只显示最近上报的中性条', async () => {
    const host = document.createElement('div')
    const app = createApp(OpsNodeResources, { node: node(84.65, 'offline') })
    app.mount(host)
    await nextTick()
    expect(host.textContent).toContain('84.65%')
    expect(host.textContent).toContain('CPU—')
    expect(host.textContent).toContain('未连接 · 最近上报')
    expect(host.querySelector('.is-historical')).not.toBeNull()
    expect(host.querySelectorAll('.node-resource')).toHaveLength(3)
    app.unmount()
  })
})
