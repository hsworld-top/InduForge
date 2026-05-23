<!--
  MaterialPanel - 物料面板
  根据编辑模式切换：页面模式（组件/绘图区/资源 Tab）、Canvas 模式（绘图工具）
-->
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import IconEpBack from '~icons/ep/back'
import CanvasToolsPanel from './CanvasToolsPanel.vue'
import ComponentPanel from './ComponentPanel.vue'
import DiagramAreaPanel from './DiagramAreaPanel.vue'
import ResourcePanel from './ResourcePanel.vue'

type MaterialEditMode = 'page' | 'canvas'
type MaterialToolType = 'select' | 'line' | 'rect' | 'circle' | 'text' | 'image' | 'pipe' | 'path'

const props = withDefaults(
  defineProps<{
    /**
     * 编辑模式：page - 页面编辑，canvas - Canvas 绘图
     */
    editMode?: MaterialEditMode
    /**
     * 当前激活的工具
     */
    activeTool?: MaterialToolType | ''
  }>(),
  {
    editMode: 'page',
    activeTool: '',
  },
)

const emit = defineEmits<{
  (event: 'update:editMode', value: MaterialEditMode): void
  (event: 'update:activeTool', value: MaterialToolType | ''): void
}>()
const { t } = useI18n()

const activeTab = ref('components')

/**
 * Canvas 工具栏 v-model 代理，避免直接修改 props
 */
const activeToolModel = computed({
  get: () => (props.activeTool || 'select') as MaterialToolType,
  set: (value: MaterialToolType | '') => emit('update:activeTool', value),
})

/**
 * 退出 Canvas 模式
 */
function exitCanvasMode() {
  emit('update:editMode', 'page')
}
</script>

<template>
  <div class="material-panel">
    <!-- 页面编辑模式：组件 + 绘图区 + 资源 -->
    <template v-if="editMode === 'page'">
      <el-tabs v-model="activeTab">
        <el-tab-pane :label="t('materialPanel.components')" name="components">
          <ComponentPanel />
        </el-tab-pane>
        <el-tab-pane :label="t('materialPanel.diagram')" name="diagram" disabled>
          <DiagramAreaPanel />
        </el-tab-pane>
        <el-tab-pane :label="t('materialPanel.resources')" name="resources">
          <ResourcePanel />
        </el-tab-pane>
      </el-tabs>
    </template>

    <!-- Canvas 绘图模式：绘图工具 -->
    <template v-else-if="editMode === 'canvas'">
      <div class="canvas-tools-panel">
        <div class="panel-header">
          <h3 class="text-sm font-medium">{{ t('materialPanel.canvasTools') }}</h3>
          <el-button size="small" text @click="exitCanvasMode">
            <IconEpBack />
            {{ t('materialPanel.backToPageEditor') }}
          </el-button>
        </div>
        <CanvasToolsPanel v-model="activeToolModel" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.material-panel {
  height: 100%;
  overflow: hidden;
  background: transparent;
}

.material-panel :deep(.el-tabs) {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.material-panel :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
}

.material-panel :deep(.el-tabs__header) {
  margin: 0;
  padding: 0 12px;
  border-bottom: 1px solid var(--designer-border-color);
  background: var(--designer-shell-surface);
}

.material-panel :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.material-panel :deep(.el-tabs__item) {
  height: 42px;
  padding: 0 12px;
  color: var(--designer-text-secondary);
  font-size: 13px;
  font-weight: 500;
}

.material-panel :deep(.el-tabs__item.is-active) {
  color: var(--designer-primary-text);
}

.material-panel :deep(.el-tabs__active-bar) {
  height: 2px;
  background-color: var(--designer-primary);
}

.material-panel :deep(.el-tab-pane) {
  height: 100%;
}

.canvas-tools-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--designer-border-color);
  background: transparent;
  color: var(--designer-text-primary);
}
</style>
