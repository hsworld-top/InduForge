<template>
  <el-dialog
    :model-value="modelValue"
    class="collector-wizard-dialog"
    title="新建工业连接"
    width="88vw"
    destroy-on-close
    @close="emit('update:modelValue', false)"
  >
    <div class="collector-wizard">
      <aside class="collector-wizard__catalog">
        <div class="collector-wizard__catalog-head">
          <strong>驱动目录</strong>
          <span>{{ drivers.length }} 个可用驱动</span>
        </div>
        <el-input
          v-model="search"
          clearable
          :prefix-icon="IconTablerSearch"
          placeholder="搜索设备、协议或驱动"
        />

        <div v-loading="loadingDrivers" class="collector-wizard__tree-wrap">
          <el-result
            v-if="loadError"
            icon="warning"
            title="驱动目录暂不可用"
            :sub-title="loadError"
          >
            <template #extra><el-button @click="loadDrivers">重新加载</el-button></template>
          </el-result>
          <el-tree
            v-else
            :data="driverTree"
            node-key="id"
            :current-node-key="driverId"
            :default-expanded-keys="expandedCategoryKeys"
            highlight-current
            empty-text="没有匹配的驱动"
            @node-click="handleNodeClick"
          >
            <template #default="{ data }">
              <div class="collector-wizard__tree-node" :class="`is-${data.type}`">
                <IconTablerCategory v-if="data.type === 'category'" />
                <CollectorDriverIcon
                  v-else
                  :protocol-family="data.protocolFamily || ''"
                  :driver-id="data.driver?.driverId"
                />
                <span>
                  <strong>{{ data.label }}</strong>
                  <small v-if="data.type === 'driver'">{{ data.driver?.driverId }}</small>
                </span>
                <em v-if="data.type === 'family'">{{ data.children?.length || 0 }}</em>
              </div>
            </template>
          </el-tree>
        </div>
      </aside>

      <main class="collector-wizard__config">
        <template v-if="selectedDriver">
          <header class="collector-wizard__driver-head">
            <span>
              <CollectorDriverIcon
                :protocol-family="selectedDriver.protocolFamily"
                :driver-id="selectedDriver.driverId"
              />
            </span>
            <div>
              <small>{{ formatCollectorCategoryLabel(selectedDriver.category) }}</small>
              <h3>{{ formatCollectorDriverDisplayName(selectedDriver) }}</h3>
              <p>{{ selectedDriver.driverId }}@{{ selectedDriver.driverVersion }}</p>
            </div>
          </header>

          <div v-loading="loadingDetail" class="collector-wizard__form-wrap">
            <el-form label-position="top">
              <el-form-item label="连接名称" required>
                <el-input v-model="name" placeholder="请输入便于识别的连接名称" />
              </el-form-item>
              <CollectorSchemaForm
                v-if="driverDetail"
                v-model="values"
                :schema="driverDetail.connectionSchema"
                :ui-schema="driverDetail.uiSchema"
              />
            </el-form>
          </div>
        </template>

        <div v-else class="collector-wizard__empty">
          <span><IconTablerTopologyStar3 /></span>
          <strong>从左侧选择一个工业驱动</strong>
          <p>目录只展示平台已经完成 Manifest、Schema 和适配器登记的驱动。</p>
        </div>
      </main>
    </div>

    <template #footer>
      <div class="collector-wizard__footer">
        <span v-if="selectedDriver">
          已选择 {{ formatCollectorDriverDisplayName(selectedDriver) }}
        </span>
        <span v-else>请选择驱动后填写连接参数</span>
        <div>
          <el-button @click="emit('update:modelValue', false)">取消</el-button>
          <el-button
            data-test="save-connection"
            type="primary"
            :loading="saving"
            :disabled="!name.trim() || !driverDetail"
            @click="save"
          >
            创建连接
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getApiErrorMessage } from '@/utils/request'
import {
  createCollectorConnection,
  getCollectorDriver,
  listCollectorDrivers,
} from '@/api/collector.api'
import {
  splitCollectorFormValues,
  type CollectorDriverDetail,
  type CollectorDriverSummary,
} from '@/api/schemas/collector.schema'
import {
  buildCollectorDriverTree,
  formatCollectorCategoryLabel,
  formatCollectorDriverDisplayName,
  type CollectorDriverTreeNode,
} from './collector-workbench-model'
import IconTablerCategory from '~icons/tabler/category'
import IconTablerSearch from '~icons/tabler/search'
import IconTablerTopologyStar3 from '~icons/tabler/topology-star-3'
import CollectorDriverIcon from './CollectorDriverIcon.vue'
import CollectorSchemaForm from './CollectorSchemaForm.vue'

const props = defineProps<{ modelValue: boolean; projectId: string }>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  created: [connectionId: string]
}>()
const drivers = ref<CollectorDriverSummary[]>([])
const driverId = ref('')
const driverDetail = ref<CollectorDriverDetail | null>(null)
const name = ref('')
const search = ref('')
const values = ref<Record<string, unknown>>({})
const saving = ref(false)
const loadingDrivers = ref(false)
const loadingDetail = ref(false)
const loadError = ref('')

const driverTree = computed(() => buildCollectorDriverTree(drivers.value, search.value))
const expandedCategoryKeys = computed(() => driverTree.value.map((node) => node.id))
const selectedDriver = computed(() =>
  drivers.value.find((driver) => driver.driverId === driverId.value),
)

watch(
  () => props.modelValue,
  async (visible) => {
    if (!visible) return
    driverId.value = ''
    driverDetail.value = null
    name.value = ''
    search.value = ''
    values.value = {}
    await loadDrivers()
  },
)

async function loadDrivers() {
  loadingDrivers.value = true
  loadError.value = ''
  try {
    drivers.value = (await listCollectorDrivers({ page: 1, pageSize: 100 })).list
    if (!drivers.value.length) loadError.value = '当前没有可用的工业采集驱动。'
  } catch (error) {
    drivers.value = []
    loadError.value = getApiErrorMessage(error, '请确认 data_service 已启动并加载驱动目录。')
  } finally {
    loadingDrivers.value = false
  }
}

async function handleNodeClick(node: CollectorDriverTreeNode) {
  if (node.type !== 'driver' || !node.driver || node.driver.driverId === driverId.value) return
  driverId.value = node.driver.driverId
  driverDetail.value = null
  loadingDetail.value = true
  try {
    driverDetail.value = await getCollectorDriver(node.driver.driverId)
    name.value = node.driver.displayName
    values.value = Object.fromEntries(
      Object.entries(driverDetail.value.connectionSchema.properties || {})
        .filter(([, property]) => property.default !== undefined)
        .map(([key, property]) => [key, property.default]),
    )
  } finally {
    loadingDetail.value = false
  }
}

async function save() {
  if (!driverDetail.value) return
  saving.value = true
  try {
    const payload = splitCollectorFormValues(driverDetail.value.connectionSchema, values.value)
    const connection = await createCollectorConnection(props.projectId, {
      name: name.value.trim(),
      driverId: driverDetail.value.driverId,
      ...payload,
      metadata: {},
    })
    emit('created', connection.id)
    emit('update:modelValue', false)
    ElMessage.success('工业采集连接已创建')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.collector-wizard {
  display: grid;
  height: min(680px, 72vh);
  min-height: 500px;
  grid-template-columns: 360px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: 10px;
  background: var(--dc-surface-raised);
}
.collector-wizard__catalog {
  display: flex;
  min-height: 0;
  flex-direction: column;
  gap: 12px;
  padding: 16px 14px;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}
.collector-wizard__catalog-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 2px;
}
.collector-wizard__catalog-head strong {
  color: var(--dc-text);
  font-size: 16px;
}
.collector-wizard__catalog-head span {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.collector-wizard__tree-wrap {
  min-height: 0;
  flex: 1;
  overflow: auto;
}
.collector-wizard__tree-wrap :deep(.el-tree) {
  --el-tree-node-hover-bg-color: var(--dc-primary-soft);
  background: transparent;
  color: var(--dc-text-secondary);
}
.collector-wizard__tree-wrap :deep(.el-tree-node__content) {
  height: 40px;
  margin: 1px 0;
  border-radius: var(--dc-radius-sm);
}
.collector-wizard__tree-wrap :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.collector-wizard__tree-node {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 8px;
  padding-right: 8px;
}
.collector-wizard__tree-node > svg {
  width: 18px;
  height: 18px;
  flex: 0 0 18px;
  color: var(--dc-text-muted);
}
.collector-wizard__tree-node > span {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 1px;
}
.collector-wizard__tree-node strong,
.collector-wizard__tree-node small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collector-wizard__tree-node strong {
  font-size: 12px;
  font-weight: 700;
}
.collector-wizard__tree-node small {
  color: var(--dc-text-muted);
  font-size: 9px;
  font-weight: 400;
}
.collector-wizard__tree-node em {
  min-width: 22px;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 10px;
  font-style: normal;
  text-align: center;
}
.collector-wizard__tree-node.is-category strong {
  font-size: 13px;
}
.collector-wizard__config {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  background: var(--dc-surface-raised);
}
.collector-wizard__driver-head {
  display: flex;
  min-height: 82px;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  border-bottom: 1px solid var(--dc-border);
}
.collector-wizard__driver-head > span {
  display: grid;
  width: 42px;
  height: 42px;
  flex: 0 0 42px;
  place-items: center;
  border-radius: 10px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.collector-wizard__driver-head > span :deep(.collector-driver-icon) {
  width: 24px;
  height: 24px;
}
.collector-wizard__driver-head small {
  color: var(--dc-text-muted);
  font-size: 10px;
  font-weight: 700;
}
.collector-wizard__driver-head h3 {
  margin: 3px 0 2px;
  color: var(--dc-text);
  font-size: 18px;
}
.collector-wizard__driver-head p {
  margin: 0;
  color: var(--dc-text-muted);
  font-family: var(--dc-font-mono);
  font-size: 10px;
}
.collector-wizard__form-wrap {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 20px 24px 28px;
}
.collector-wizard__form-wrap :deep(.el-form) {
  max-width: 760px;
}
.collector-wizard__empty {
  display: flex;
  max-width: 460px;
  align-self: center;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  text-align: center;
}
.collector-wizard__empty > span {
  display: grid;
  width: 64px;
  height: 64px;
  place-items: center;
  margin-bottom: 16px;
  border-radius: 18px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.collector-wizard__empty svg {
  width: 30px;
  height: 30px;
}
.collector-wizard__empty strong {
  font-size: 16px;
}
.collector-wizard__empty p {
  margin: 9px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.7;
}
.collector-wizard__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.collector-wizard__footer > span {
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
@media (max-width: 900px) {
  .collector-wizard {
    grid-template-columns: 300px minmax(0, 1fr);
  }
}
</style>

<style>
.collector-wizard-dialog {
  max-width: 1180px;
}
.collector-wizard-dialog .el-dialog__body {
  padding: 8px 20px 12px;
}
</style>
