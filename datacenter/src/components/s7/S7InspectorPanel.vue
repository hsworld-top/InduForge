<template>
  <aside class="s7-inspector">
    <template v-if="variable">
      <div class="s7-inspector__summary">
        <strong>{{ variable.name }}</strong>
        <span>{{ variable.normalizedAddress || variable.addressText }}</span>
      </div>
      <section>
        <h3><IconTablerInfoCircle />变量信息</h3>
        <dl>
          <dt>名称</dt>
          <dd>{{ variable.name }}</dd>
          <dt>Code</dt>
          <dd>{{ variable.code }}</dd>
          <dt>状态</dt>
          <dd>{{ variable.status }}</dd>
          <dt>读写权限</dt>
          <dd>{{ variable.accessLevel || 'Read' }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerMapPin />S7 地址</h3>
        <dl>
          <dt>地址区</dt>
          <dd>{{ variable.area }}</dd>
          <dt>DB 号</dt>
          <dd>{{ variable.dbNumber ?? '-' }}</dd>
          <dt>字节</dt>
          <dd>{{ variable.byteOffset }}-{{ variable.byteOffset + variable.readLength - 1 }}</dd>
          <dt>Bit</dt>
          <dd>{{ variable.bitOffset ?? '-' }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerBinaryTree />数据解释</h3>
        <dl>
          <dt>类型</dt>
          <dd>{{ variable.dataType }}</dd>
          <dt>字节序</dt>
          <dd>{{ variable.byteOrder }}</dd>
          <dt>字序</dt>
          <dd>{{ variable.wordOrder }}</dd>
          <dt>倍率</dt>
          <dd>{{ variable.scale }}</dd>
          <dt>偏移</dt>
          <dd>{{ variable.offset }}</dd>
          <dt>单位</dt>
          <dd>{{ variable.unit || '-' }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerDatabase />数据点信息</h3>
        <p :title="variable.datapointPath || ''">{{ variable.datapointPath || '未生成' }}</p>
      </section>
      <section>
        <h3><IconTablerActivityHeartbeat />最近读取</h3>
        <dl>
          <dt>最近值</dt>
          <dd>{{ formatValue(variable.lastValue) }}</dd>
          <dt>质量</dt>
          <dd>{{ variable.quality || 'unknown' }}</dd>
          <dt>时间</dt>
          <dd>{{ variable.lastUpdatedAt || '-' }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerAlertTriangle />校验问题</h3>
        <p :class="{ 'is-warning': variableIssues.length }">
          {{ variableIssues.length ? `${variableIssues.length} 个问题` : '无' }}
        </p>
      </section>
    </template>
    <template v-else>
      <div class="s7-inspector__summary">
        <strong>{{ group?.name || '全部变量' }}</strong>
        <span>{{ scopedVariables.length }} 个变量 · {{ profile?.plcFamily || 'PLC 未确认' }}</span>
      </div>
      <section>
        <h3><IconTablerCpu />PLC 档案</h3>
        <dl>
          <dt>系列</dt>
          <dd>{{ profile?.plcFamily || '-' }}</dd>
          <dt>连接地址</dt>
          <dd>{{ profile ? `${profile.host}:${profile.port}` : '-' }}</dd>
          <dt>Rack/Slot</dt>
          <dd>{{ profile ? `${profile.rack}/${profile.slot}` : '-' }}</dd>
          <dt>合并间隙</dt>
          <dd>{{ profile?.maxGapBytes ?? '-' }} bytes</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerChartBar />地址分布</h3>
        <dl>
          <dt>DB</dt>
          <dd>{{ countByArea.DB || 0 }}</dd>
          <dt>M</dt>
          <dd>{{ countByArea.M || 0 }}</dd>
          <dt>I</dt>
          <dd>{{ countByArea.I || 0 }}</dd>
          <dt>Q</dt>
          <dd>{{ countByArea.Q || 0 }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerRoute />读取计划摘要</h3>
        <p>
          {{ estimate.variableCount }} 个变量，预计合并为 {{ estimate.blockCount }} 个读取块，{{
            estimate.readsPerSecond.toFixed(2)
          }}
          reads/s。
        </p>
      </section>
      <section>
        <h3><IconTablerAlertTriangle />风险提示</h3>
        <p :class="{ 'is-warning': issues.length }">
          {{ issues.length ? `${issues.length} 个校验问题` : '暂无明显风险' }}
        </p>
      </section>
    </template>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type {
  S7Profile,
  S7ReadPlanEstimate,
  S7ValidationIssue,
  S7Variable,
  S7VariableGroup,
} from './types'
import IconTablerActivityHeartbeat from '~icons/tabler/activity-heartbeat'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerBinaryTree from '~icons/tabler/binary-tree'
import IconTablerChartBar from '~icons/tabler/chart-bar'
import IconTablerCpu from '~icons/tabler/cpu'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerInfoCircle from '~icons/tabler/info-circle'
import IconTablerMapPin from '~icons/tabler/map-pin'
import IconTablerRoute from '~icons/tabler/route'

const props = defineProps<{
  profile: S7Profile | null
  group: S7VariableGroup | null
  variable: S7Variable | null
  variables: S7Variable[]
  issues: S7ValidationIssue[]
  estimate: S7ReadPlanEstimate
  connection: { id?: string; config?: Record<string, any> }
  projectId: string
}>()

const scopedVariables = computed(() =>
  props.group
    ? props.variables.filter((item) => item.groupId === props.group?.id)
    : props.variables,
)
const countByArea = computed(() => {
  const result: Record<string, number> = {}
  for (const item of scopedVariables.value) result[item.area] = (result[item.area] || 0) + 1
  return result
})
const variableIssues = computed(() =>
  props.variable ? props.issues.filter((issue) => issue.variableId === props.variable?.id) : [],
)
const formatValue = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
</script>

<style scoped>
.s7-inspector {
  width: 292px;
  min-width: 0;
  min-height: 0;
  padding: 12px;
  border-left: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  overflow: auto;
}
.s7-inspector__summary,
.s7-inspector section {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.s7-inspector__summary {
  margin-bottom: 10px;
  display: grid;
  gap: 3px;
}
.s7-inspector__summary strong,
.s7-inspector__summary span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.s7-inspector__summary strong {
  color: var(--dc-text);
  font-size: 13px;
}
.s7-inspector__summary span {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.s7-inspector section + section {
  margin-top: 10px;
}
.s7-inspector h3 {
  margin: 0 0 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text);
  font-size: 12px;
}
.s7-inspector h3 svg {
  width: 14px;
  height: 14px;
  color: var(--dc-primary);
}
.s7-inspector dl {
  margin: 0;
  display: grid;
  grid-template-columns: 78px minmax(0, 1fr);
  gap: 6px 8px;
  font-size: 12px;
}
.s7-inspector dt {
  color: var(--dc-text-muted);
}
.s7-inspector dd,
.s7-inspector p {
  min-width: 0;
  margin: 0;
  color: var(--dc-text-secondary);
  overflow-wrap: anywhere;
  font-size: 12px;
}
.s7-inspector p.is-warning {
  color: var(--el-color-warning);
  font-weight: 700;
}
.s7-inspector__link {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 800;
}
</style>
