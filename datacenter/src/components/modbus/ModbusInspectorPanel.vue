<template>
  <aside class="modbus-inspector">
    <template v-if="register">
      <div class="modbus-inspector__summary">
        <strong>{{ register.name }}</strong>
        <span>{{ register.code }}</span>
      </div>
      <section>
        <h3><IconTablerInfoCircle />变量信息</h3>
        <dl>
          <dt>名称</dt>
          <dd>{{ register.name }}</dd>
          <dt>Code</dt>
          <dd>{{ register.code }}</dd>
          <dt>状态</dt>
          <dd>{{ register.status }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerMapPin />Modbus 地址</h3>
        <dl>
          <dt>从站地址</dt>
          <dd>{{ register.unitId }}</dd>
          <dt>区域</dt>
          <dd>{{ formatArea(register.area) }}</dd>
          <dt>用户地址</dt>
          <dd>{{ register.address }}</dd>
          <dt>协议地址</dt>
          <dd>{{ register.protocolAddress }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerBinaryTree />解析规则</h3>
        <dl>
          <dt>类型</dt>
          <dd>{{ register.dataType }}</dd>
          <dt>字节序</dt>
          <dd>{{ register.byteOrder }}</dd>
          <dt>字序</dt>
          <dd>{{ register.wordOrder }}</dd>
          <dt>倍率</dt>
          <dd>{{ register.scale }}</dd>
          <dt>偏移</dt>
          <dd>{{ register.offset }}</dd>
          <dt>单位</dt>
          <dd>{{ register.unit || '-' }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerDatabase />数据点信息</h3>
        <p>{{ register.datapointPath || '未生成' }}</p>
      </section>
      <section>
        <h3><IconTablerArchive />运行契约摘要</h3>
        <dl>
          <dt>当前值</dt>
          <dd>IF 实时库</dd>
          <dt>历史归档</dt>
          <dd>
            <button type="button" class="modbus-inspector__link" @click="openStoragePolicy">
              {{ storageCoverageText }}
            </button>
          </dd>
          <dt>设备冗余</dt>
          <dd>{{ deviceRedundancyText }}</dd>
          <dt>采集冗余</dt>
          <dd>{{ collectionRedundancyText }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerAlertTriangle />校验问题</h3>
        <p :class="{ 'is-warning': registerIssues.length }">
          {{ registerIssues.length ? `${registerIssues.length} 个问题` : '无' }}
        </p>
      </section>
    </template>
    <template v-else>
      <div class="modbus-inspector__summary">
        <strong>{{ group?.name || '全部变量' }}</strong>
        <span>{{ scopedRegisters.length }} 个变量 · {{ unitCount }} 个从站地址</span>
      </div>
      <section>
        <h3><IconTablerStack2 />寄存器组信息</h3>
        <dl>
          <dt>名称</dt>
          <dd>{{ group?.name || '全部变量' }}</dd>
          <dt>变量数</dt>
          <dd>{{ scopedRegisters.length }}</dd>
          <dt>从站地址数</dt>
          <dd>{{ unitCount }}</dd>
          <dt>预计读取次数</dt>
          <dd>{{ estimate.readCount }}</dd>
          <dt>预计 reads/s</dt>
          <dd>{{ estimate.readsPerSecond.toFixed(2) }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerChartBar />地址分布</h3>
        <dl>
          <dt>Holding Register</dt>
          <dd>{{ countByArea.holding_register || 0 }}</dd>
          <dt>Input Register</dt>
          <dd>{{ countByArea.input_register || 0 }}</dd>
          <dt>Coil</dt>
          <dd>{{ countByArea.coil || 0 }}</dd>
          <dt>Discrete Input</dt>
          <dd>{{ countByArea.discrete_input || 0 }}</dd>
        </dl>
      </section>
      <section>
        <h3><IconTablerRoute />运行态读取预估</h3>
        <p>{{ estimate.registerCount }} 个变量，预计合并为 {{ estimate.readCount }} 次读取。</p>
      </section>
    </template>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getStoragePolicyCoverage } from '@/api/storage-policy.api'
import type {
  ModbusReadPlanEstimate,
  ModbusRegister,
  ModbusRegisterGroup,
  ModbusValidationIssue,
} from './types'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerArchive from '~icons/tabler/archive'
import IconTablerBinaryTree from '~icons/tabler/binary-tree'
import IconTablerChartBar from '~icons/tabler/chart-bar'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerInfoCircle from '~icons/tabler/info-circle'
import IconTablerMapPin from '~icons/tabler/map-pin'
import IconTablerRoute from '~icons/tabler/route'
import IconTablerStack2 from '~icons/tabler/stack-2'

const props = defineProps<{
  group: ModbusRegisterGroup | null
  register: ModbusRegister | null
  registers: ModbusRegister[]
  issues: ModbusValidationIssue[]
  estimate: ModbusReadPlanEstimate
  connection: { id?: string; config?: Record<string, any> }
  projectId: string
}>()

const route = useRoute()
const router = useRouter()
const storagePolicyCount = ref<number | null>(null)
const storageCoverageLoading = ref(false)
const storageCoverageFailed = ref(false)

const scopedRegisters = computed(() =>
  props.group
    ? props.registers.filter((item) => item.groupId === props.group?.id)
    : props.registers,
)
const unitCount = computed(() => new Set(scopedRegisters.value.map((item) => item.unitId)).size)
const countByArea = computed(() => {
  const result: Record<string, number> = {}
  for (const item of scopedRegisters.value) result[item.area] = (result[item.area] || 0) + 1
  return result
})
const registerIssues = computed(() =>
  props.register ? props.issues.filter((issue) => issue.registerId === props.register?.id) : [],
)
const deviceRedundancyText = computed(() => {
  const redundancy = props.connection?.config?.redundancy
  if (!redundancy || redundancy.enabled === false) return '未配置'
  const count = Array.isArray(redundancy.endpoints) ? redundancy.endpoints.length : 0
  return count > 1 ? `主备优先级 · ${count} endpoint` : '主备优先级'
})
const collectionRedundancyText = computed(() => '运行部署策略统一配置')
const storageCoverageText = computed(() => {
  if (!props.register?.datapointPath) return '未生成数据点'
  if (storageCoverageLoading.value) return '历史归档检查中'
  if (storageCoverageFailed.value) return '历史归档检查失败'
  if (!storagePolicyCount.value) return '未配置历史归档'
  return `已命中 ${storagePolicyCount.value} 条历史策略`
})

const formatArea = (area: string) => {
  const map: Record<string, string> = {
    coil: 'Coil',
    discrete_input: 'Discrete Input',
    input_register: 'Input Register',
    holding_register: 'Holding Register',
  }
  return map[area] || area
}

const openStoragePolicy = () => {
  const base = route.path.startsWith('/debug/') ? '/debug' : ''
  void router.push({
    path: `${base}/storage-policy`,
    query: {
      ...route.query,
      accessSourceId: props.connection?.id || '',
      datapointPath: props.register?.datapointPath || '',
    },
  })
}

watch(
  () => [props.projectId, props.register?.datapointId, props.register?.datapointPath] as const,
  async ([projectId, datapointId, datapointPath]) => {
    storagePolicyCount.value = null
    storageCoverageFailed.value = false
    if (!projectId || (!datapointId && !datapointPath)) return
    storageCoverageLoading.value = true
    try {
      const coverage = await getStoragePolicyCoverage(projectId, {
        datapointId: datapointId || undefined,
        path: datapointPath || undefined,
      })
      storagePolicyCount.value = coverage.matchedPolicyCount
    } catch {
      storagePolicyCount.value = 0
      storageCoverageFailed.value = true
    } finally {
      storageCoverageLoading.value = false
    }
  },
  { immediate: true },
)
</script>

<style scoped>
.modbus-inspector {
  width: 292px;
  min-width: 0;
  min-height: 0;
  padding: 12px;
  border-left: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  overflow: auto;
}

.modbus-inspector__summary {
  margin-bottom: 10px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  display: grid;
  gap: 3px;
}

.modbus-inspector__summary strong {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.modbus-inspector__summary span {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}

.modbus-inspector section {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.modbus-inspector section + section {
  margin-top: 10px;
}

.modbus-inspector h3 {
  margin: 0 0 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text);
  font-size: 12px;
}

.modbus-inspector h3 svg {
  width: 14px;
  height: 14px;
  color: var(--dc-primary);
}

.modbus-inspector dl {
  margin: 0;
  display: grid;
  grid-template-columns: 86px minmax(0, 1fr);
  gap: 6px 8px;
  font-size: 12px;
}

.modbus-inspector dt {
  color: var(--dc-text-muted);
}

.modbus-inspector dd {
  min-width: 0;
  margin: 0;
  color: var(--dc-text-secondary);
  overflow-wrap: anywhere;
}

.modbus-inspector p {
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
  overflow-wrap: anywhere;
}

.modbus-inspector p.is-warning {
  color: var(--el-color-warning);
  font-weight: 700;
}

.modbus-inspector__link {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 800;
}
</style>
