<template>
  <aside class="collector-connection-list">
    <div class="collector-connection-list__head">
      <div><small>INDUSTRIAL LINKS</small><strong>工业连接</strong></div>
      <el-button type="primary" circle @click="emit('create')">+</el-button>
    </div>
    <el-input
      v-model="search"
      clearable
      placeholder="搜索连接"
      @keyup.enter="reload(1)"
      @clear="reload(1)"
    />
    <div v-loading="loading" class="collector-connection-list__items">
      <button
        v-for="item in items"
        :key="item.id"
        type="button"
        :class="['collector-connection-list__item', { 'is-active': item.id === selectedId }]"
        @click="emit('select', item.id)"
      >
        <span class="collector-connection-list__pulse" :class="{ 'is-enabled': item.enabled }" />
        <span
          ><strong>{{ item.name }}</strong
          ><small>{{ item.protocolFamily }} · {{ item.driverId }}</small></span
        >
      </button>
      <el-empty v-if="!loading && !items.length" description="暂无工业连接" :image-size="70" />
    </div>
    <el-pagination
      small
      layout="prev, pager, next"
      :current-page="page"
      :page-size="pageSize"
      :total="total"
      @current-change="reload"
    />
  </aside>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { listCollectorConnections } from '@/api/collector.api'
import type { CollectorConnection } from '@/api/schemas/collector.schema'

const props = defineProps<{ projectId: string; selectedId?: string }>()
const emit = defineEmits<{
  select: [id: string]
  create: []
  loaded: [items: CollectorConnection[]]
}>()
const items = ref<CollectorConnection[]>([])
const loading = ref(false)
const search = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)

async function reload(nextPage = page.value) {
  page.value = nextPage
  loading.value = true
  try {
    const result = await listCollectorConnections(props.projectId, {
      page: page.value,
      pageSize,
      search: search.value,
    })
    items.value = result.list
    total.value = result.pagination.total
    emit('loaded', result.list)
  } finally {
    loading.value = false
  }
}

defineExpose({ reload })
watch(
  () => props.projectId,
  () => reload(1),
)
onMounted(() => reload())
</script>

<style scoped>
.collector-connection-list {
  display: flex;
  width: 280px;
  min-width: 280px;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  border-right: 1px solid #dfe6ea;
  background: #f5f7f8;
}
.collector-connection-list__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.collector-connection-list__head div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.collector-connection-list__head small {
  color: #84919a;
  font-size: 10px;
  letter-spacing: 0.12em;
}
.collector-connection-list__head strong {
  color: #17232c;
  font-size: 18px;
}
.collector-connection-list__items {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.collector-connection-list__item {
  display: grid;
  width: 100%;
  grid-template-columns: 10px 1fr;
  gap: 10px;
  margin-bottom: 7px;
  padding: 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: #23313a;
  cursor: pointer;
  text-align: left;
}
.collector-connection-list__item:hover {
  background: #fff;
}
.collector-connection-list__item.is-active {
  border-color: #9fc2d2;
  background: #fff;
  box-shadow: 0 6px 18px rgb(28 67 85 / 8%);
}
.collector-connection-list__item span:last-child {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 5px;
}
.collector-connection-list__item small {
  overflow: hidden;
  color: #7c8992;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collector-connection-list__pulse {
  width: 8px;
  height: 8px;
  margin-top: 5px;
  border-radius: 50%;
  background: #aab3b9;
}
.collector-connection-list__pulse.is-enabled {
  background: #2c9a73;
  box-shadow: 0 0 0 3px rgb(44 154 115 / 12%);
}
</style>
