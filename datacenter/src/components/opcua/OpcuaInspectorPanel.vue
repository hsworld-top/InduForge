<template>
  <aside class="opcua-inspector">
    <template v-if="node">
      <div class="opcua-inspector__hero">
        <div>
          <span>OPC UA 变量</span>
          <strong>{{ node.name }}</strong>
        </div>
        <el-tag size="small" type="success">{{ node.dataType }}</el-tag>
      </div>
      <section class="opcua-inspector__section">
        <div class="opcua-inspector__section-title">变量信息</div>
        <dl>
          <dt>NodeId</dt>
          <dd class="is-code">{{ node.nodeId }}</dd>
          <dt>Code</dt>
          <dd>{{ node.code }}</dd>
          <dt>访问级别</dt>
          <dd>{{ node.accessLevel }}</dd>
        </dl>
      </section>
      <section class="opcua-inspector__section">
        <div class="opcua-inspector__section-title">采样策略</div>
        <dl>
          <dt>采样周期</dt>
          <dd>{{ node.samplingMs }}ms</dd>
          <dt>死区</dt>
          <dd>{{ node.deadband ?? '-' }}</dd>
          <dt>状态</dt>
          <dd>{{ node.status || 'active' }}</dd>
        </dl>
      </section>
      <section class="opcua-inspector__section">
        <div class="opcua-inspector__section-title">数据点</div>
        <dl>
          <dt>路径</dt>
          <dd class="is-code">{{ node.datapointPath || '未生成' }}</dd>
          <dt>状态</dt>
          <dd>{{ node.datapointStatus || '-' }}</dd>
        </dl>
      </section>
      <section class="opcua-inspector__section">
        <div class="opcua-inspector__section-title">运行契约摘要</div>
        <dl>
          <dt>当前值</dt>
          <dd>IF 实时库</dd>
          <dt>历史归档</dt>
          <dd>
            <button type="button" class="opcua-inspector__link" @click="openStoragePolicy">
              查看存储策略
            </button>
          </dd>
          <dt>设备冗余</dt>
          <dd>{{ deviceRedundancyText }}</dd>
          <dt>采集冗余</dt>
          <dd>{{ collectionRedundancyText }}</dd>
        </dl>
      </section>
    </template>

    <template v-else-if="group">
      <div class="opcua-inspector__hero">
        <div>
          <span>变量组</span>
          <strong>{{ group.name }}</strong>
        </div>
        <el-tag size="small" type="success">变量组</el-tag>
      </div>
      <section class="opcua-inspector__section">
        <div class="opcua-inspector__section-title">变量组信息</div>
        <dl>
          <dt>变量数</dt>
          <dd>{{ groupNodes.length }}</dd>
          <dt>数据点数</dt>
          <dd>{{ groupNodes.filter((item) => item.datapointPath).length }}</dd>
          <dt>说明</dt>
          <dd>{{ group.description || '-' }}</dd>
        </dl>
      </section>
    </template>

    <div v-else class="opcua-inspector__empty">
      <IconTablerLayoutSidebarRight />
      <strong>选择变量组或变量</strong>
      <span>这里展示建模配置、数据点和校验问题</span>
    </div>

    <section class="opcua-inspector__section opcua-inspector__issues">
      <div class="opcua-inspector__section-title">校验问题</div>
      <div v-if="visibleIssues.length === 0" class="opcua-inspector__ok">当前无问题</div>
      <div
        v-for="issue in visibleIssues"
        :key="`${issue.code}-${issue.nodeId || issue.groupId}`"
        class="opcua-inspector__issue"
      >
        <el-tag size="small" :type="issue.severity === 'error' ? 'danger' : 'warning'">
          {{ issue.severity }}
        </el-tag>
        <span>{{ issue.message }}</span>
      </div>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import IconTablerLayoutSidebarRight from '~icons/tabler/layout-sidebar-right'
import type { OpcuaNode, OpcuaNodeGroup, OpcuaValidationIssue } from './types'

const props = defineProps<{
  group?: OpcuaNodeGroup | null
  node?: OpcuaNode | null
  nodes: OpcuaNode[]
  issues: OpcuaValidationIssue[]
  connection?: { id?: string; config?: Record<string, any> } | null
  projectId: string
}>()

const route = useRoute()
const router = useRouter()

const groupNodes = computed(() =>
  props.group ? props.nodes.filter((node) => node.groupId === props.group?.id) : [],
)

const visibleIssues = computed(() => {
  if (props.node) return props.issues.filter((issue) => issue.nodeId === props.node?.id)
  if (props.group) return props.issues.filter((issue) => issue.groupId === props.group?.id)
  return props.issues.slice(0, 8)
})

const deviceRedundancyText = computed(() => {
  const redundancy = props.connection?.config?.redundancy
  if (!redundancy || redundancy.enabled === false) return '未配置'
  const count = Array.isArray(redundancy.endpoints) ? redundancy.endpoints.length : 0
  return count > 1 ? `主备优先级 · ${count} endpoint` : '主备优先级'
})

const collectionRedundancyText = computed(() => '运行部署策略统一配置')

const openStoragePolicy = () => {
  const base = route.path.startsWith('/debug/') ? '/debug' : ''
  void router.push({
    path: `${base}/storage-policy`,
    query: {
      ...route.query,
      accessSourceId: props.connection?.id || '',
      datapointPath: props.node?.datapointPath || '',
    },
  })
}
</script>

<style scoped>
.opcua-inspector {
  width: 286px;
  border-left: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  padding: 12px;
  overflow: auto;
}

.opcua-inspector__hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  margin-bottom: 10px;
}

.opcua-inspector__hero div {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.opcua-inspector__hero span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.opcua-inspector__hero strong {
  min-width: 0;
  color: var(--dc-text);
  font-size: 14px;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.opcua-inspector__section {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.opcua-inspector__section + .opcua-inspector__section {
  margin-top: 10px;
}

.opcua-inspector dl {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 8px 10px;
  margin: 0;
  font-size: 12px;
}

.opcua-inspector dt {
  color: var(--dc-text-muted);
}

.opcua-inspector dd {
  margin: 0;
  color: var(--dc-text);
  overflow-wrap: anywhere;
}

.opcua-inspector dd.is-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  line-height: 16px;
}

.opcua-inspector__empty {
  min-height: 170px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--dc-text-muted);
  text-align: center;
}

.opcua-inspector__empty svg {
  width: 26px;
  height: 26px;
}

.opcua-inspector__section-title {
  margin: 0 0 8px;
  color: var(--dc-text);
  font-weight: 600;
  font-size: 13px;
}

.opcua-inspector__ok {
  color: var(--dc-success);
  font-size: 13px;
}

.opcua-inspector__issue {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 0;
  border-top: 1px solid var(--dc-border);
  font-size: 13px;
}

.opcua-inspector__link {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 800;
}
</style>
