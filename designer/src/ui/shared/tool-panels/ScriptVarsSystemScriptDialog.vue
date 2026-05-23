<!--
  脚本面板：系统启动/关闭脚本编辑器兼容包装。
-->
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ScriptEditorDialog from './ScriptEditorDialog.vue'

interface SidebarNodeLike {
  id: string
  label: string
  type: string
}

interface PageVariableRowLike {
  name: string
  type?: string
}

const props = withDefaults(
  defineProps<{
    dialogTitle: string
    metaTitle: string
    customScriptSidebarTree?: SidebarNodeLike[]
    pageSidebarTree?: SidebarNodeLike[]
    pageVariableRows?: PageVariableRowLike[]
    jsCompletions?: unknown[]
    filterSidebarNode: (value: string, data: SidebarNodeLike) => boolean
    beforeClose?: (...args: unknown[]) => void
    tabs?: Array<{ key: string; label: string }>
  }>(),
  {
    customScriptSidebarTree: () => [],
    pageSidebarTree: () => [],
    pageVariableRows: () => [],
    jsCompletions: () => [],
    tabs: () => [],
  },
)

const emit = defineEmits([
  'openVariableEnum',
  'save',
  'customInsert',
  'pageInsert',
  'pageVariableInsert',
])
const visible = defineModel<boolean>({ default: false })
const systemCode = defineModel<string>('systemCode', { default: '' })
const scriptSearch = defineModel<string>('scriptSearch', { default: '' })
const pageSearch = defineModel<string>('pageSearch', { default: '' })
const pageVariableSearch = defineModel<string>('pageVariableSearch', { default: '' })
const activeTab = defineModel<string>('activeTab', { default: '' })

const editorRef = ref<InstanceType<typeof ScriptEditorDialog> | null>(null)
const { t } = useI18n()
const metaDescription = computed(() => t('scriptPanel.editor.systemScript'))
const dialogAttrs = computed(() =>
  props.beforeClose
    ? {
        beforeClose: props.beforeClose,
      }
    : {},
)

defineExpose({
  insertText: (text: string) => editorRef.value?.insertText?.(text),
  format: () => editorRef.value?.format?.(),
})
</script>

<template>
  <ScriptEditorDialog
    ref="editorRef"
    v-model="visible"
    v-model:code="systemCode"
    v-model:script-search="scriptSearch"
    v-model:page-search="pageSearch"
    v-model:page-variable-search="pageVariableSearch"
    v-model:active-tab="activeTab"
    scope="global"
    :dialog-title="dialogTitle"
    :meta-title="metaTitle"
    :meta-description="metaDescription"
    :custom-script-sidebar-tree="props.customScriptSidebarTree"
    :page-sidebar-tree="props.pageSidebarTree"
    :page-variable-rows="props.pageVariableRows"
    :js-completions="props.jsCompletions"
    :tabs="props.tabs"
    :filter-sidebar-node="filterSidebarNode"
    v-bind="dialogAttrs"
    @open-variable-enum="emit('openVariableEnum')"
    @save="emit('save')"
    @custom-insert="(data) => emit('customInsert', data)"
    @page-insert="(data) => emit('pageInsert', data)"
    @page-variable-insert="(row) => emit('pageVariableInsert', row)"
  />
</template>
