<template>
  <div class="mysql-query-tabs flex-1 flex flex-col overflow-hidden">
    <el-tabs
      v-model="localActiveTab"
      type="card"
      closable
      class="query-tabs flex-1 flex flex-col overflow-hidden"
      @tab-remove="handleRemove"
      @tab-change="handleChange"
    >
      <el-tab-pane
        v-for="tab in localTabs"
        :key="tab.id"
        :name="tab.id"
        :lazy="false"
        class="flex-1 flex flex-col overflow-hidden"
      >
        <template #label>
          <span class="flex items-center">
            <span>{{ tab.name }}</span>
            <IconTablerAlertCircle v-if="tab.modified" class="ml-1 text-orange-500 w-3 h-3" />
          </span>
        </template>

        <div class="query-editor-tab-content flex-1 flex flex-col overflow-y-auto p-4">
          <MysqlQueryEditor
            v-if="localActiveTab === tab.id"
            :tab="tab"
            @execute="handleExecute"
            @save="handleSave"
            @update:tab="updateTab(tab.id, $event)"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import IconTablerAlertCircle from '~icons/tabler/alert-circle'
import MysqlQueryEditor from './MysqlQueryEditor.vue'

const props = defineProps({
  activeTab: {
    type: String,
    default: ''
  },
  tabs: {
    type: Array,
    default: () => []
  },
  connectionId: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['update:activeTab', 'update:tabs', 'execute', 'save', 'close'])

const localActiveTab = ref(props.activeTab)
const localTabs = ref([...props.tabs])
const isUpdatingFromParent = ref(false)

watch(() => props.activeTab, (val) => {
  isUpdatingFromParent.value = true
  localActiveTab.value = val
  setTimeout(() => {
    isUpdatingFromParent.value = false
  }, 0)
})

watch(() => props.tabs, (val) => {
  isUpdatingFromParent.value = true
  localTabs.value = [...val]
  setTimeout(() => {
    isUpdatingFromParent.value = false
  }, 0)
}, { deep: true })

watch(localActiveTab, (val) => {
  if (!isUpdatingFromParent.value) {
    emit('update:activeTab', val)
  }
})

watch(localTabs, (val) => {
  if (!isUpdatingFromParent.value) {
    emit('update:tabs', val)
  }
}, { deep: true })

const handleRemove = (tabId) => {
  emit('close', tabId)
}

const handleChange = (tabId) => {
  localActiveTab.value = tabId
}

const handleExecute = (tab) => {
  emit('execute', tab)
}

const handleSave = (tab) => {
  emit('save', tab)
}

const updateTab = (tabId, updatedTab) => {
  const index = localTabs.value.findIndex(t => t.id === tabId)
  if (index > -1) {
    localTabs.value[index] = { ...updatedTab }
  }
}
</script>

<style scoped>
.query-tabs :deep(.el-tabs__header) {
  margin: 0;
  flex-shrink: 0;
}

.query-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.query-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
</style>
