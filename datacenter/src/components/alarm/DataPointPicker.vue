<template>
  <div class="data-point-picker">
    <div class="data-point-picker__search">
      <Search />
      <input
        v-model="keyword"
        type="text"
        placeholder="搜索名称、路径"
        @keydown.enter.prevent="loadDatapoints"
      />
      <button type="button" :disabled="loading" @click="loadDatapoints">
        <Refresh />
      </button>
    </div>

    <div v-if="selected" class="data-point-picker__selected">
      <span>
        <strong>{{ selected.name || selected.path }}</strong>
        <code>{{ selected.path }}</code>
      </span>
      <em>{{ selected.dataType || '-' }}</em>
    </div>

    <div v-if="error" class="data-point-picker__state is-error">
      {{ error }}
    </div>
    <div v-else-if="loading" class="data-point-picker__state">正在读取数据点</div>
    <div v-else-if="!datapoints.length" class="data-point-picker__state">暂无匹配数据点</div>

    <div v-else class="data-point-picker__list">
      <button
        v-for="item in datapoints"
        :key="item.id"
        type="button"
        class="data-point-picker__row"
        :class="{ 'is-active': item.id === modelValue }"
        @click="selectDatapoint(item)"
      >
        <span>
          <strong>{{ item.name || item.path }}</strong>
          <code>{{ item.path }}</code>
        </span>
        <em>{{ item.dataType || '-' }}</em>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Refresh, Search } from '@element-plus/icons-vue'
import { getDatapoints } from '@/api/datapoint.api'
import type { Datapoint } from '@/api/schemas/datapoint.schema'
import { getApiErrorMessage } from '@/utils/request'

const props = defineProps<{
  projectId: string
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  select: [datapoint: Datapoint]
}>()

const keyword = ref('')
const datapoints = ref<Datapoint[]>([])
const loading = ref(false)
const error = ref('')
const debounceTimer = ref<number | null>(null)

const selected = computed(
  () => datapoints.value.find((item) => item.id === props.modelValue) ?? null,
)

const loadDatapoints = async () => {
  if (!props.projectId) {
    datapoints.value = []
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await getDatapoints(props.projectId, {
      page: 1,
      pageSize: 30,
      q: keyword.value.trim(),
      search: keyword.value.trim(),
    })
    datapoints.value = res.list
  } catch (err) {
    datapoints.value = []
    error.value = getApiErrorMessage(err, '数据点列表不可用')
  } finally {
    loading.value = false
  }
}

const selectDatapoint = (datapoint: Datapoint) => {
  emit('update:modelValue', datapoint.id)
  emit('select', datapoint)
}

watch(keyword, () => {
  if (debounceTimer.value) {
    window.clearTimeout(debounceTimer.value)
  }
  debounceTimer.value = window.setTimeout(() => {
    void loadDatapoints()
  }, 300)
})

watch(
  () => props.projectId,
  () => {
    void loadDatapoints()
  },
)

onMounted(() => {
  void loadDatapoints()
})

onBeforeUnmount(() => {
  if (debounceTimer.value) {
    window.clearTimeout(debounceTimer.value)
  }
})
</script>

<style scoped>
.data-point-picker {
  display: grid;
  gap: 8px;
}

.data-point-picker__search {
  height: 34px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) 28px;
  align-items: center;
  gap: 6px;
  padding: 0 6px 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-muted);
}

.data-point-picker__search svg {
  width: 15px;
  height: 15px;
}

.data-point-picker__search input {
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--dc-text);
  font-family: inherit;
  font-size: 13px;
}

.data-point-picker__search button {
  width: 28px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
}

.data-point-picker__search button:hover:not(:disabled) {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.data-point-picker__selected,
.data-point-picker__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
}

.data-point-picker__selected {
  padding: 8px 10px;
  border: 1px solid rgba(37, 99, 235, 0.22);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
}

.data-point-picker__list {
  max-height: 220px;
  display: grid;
  gap: 6px;
  overflow-y: auto;
}

.data-point-picker__row {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  cursor: pointer;
  font-family: inherit;
  text-align: left;
}

.data-point-picker__row:hover,
.data-point-picker__row.is-active {
  border-color: rgba(37, 99, 235, 0.28);
  background: var(--dc-primary-soft);
}

.data-point-picker strong,
.data-point-picker code {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.data-point-picker strong {
  font-size: 13px;
}

.data-point-picker code {
  margin-top: 3px;
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.data-point-picker em {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
}

.data-point-picker__state {
  padding: 10px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.data-point-picker__state.is-error {
  border-color: rgba(220, 38, 38, 0.32);
  color: var(--dc-danger, #b91c1c);
}
</style>
