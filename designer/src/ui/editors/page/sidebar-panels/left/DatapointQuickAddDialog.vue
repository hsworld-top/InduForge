<!--
  数据点面板：从数据中心批量勾选并快速添加变量
-->
<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

interface QuickAddFieldLike {
  id?: string
  name: string
  path?: string
  sourceLabel?: string
  type?: string
  typeLabel?: string
  updatedAtLabel?: string
  mappedName?: string
  statusLabel?: string
  statusType?: string
  mappingLabel?: string
  [key: string]: unknown
}

interface QuickGroupLike {
  id: string
  name: string
}

interface QuickSourceOptionLike {
  id: string
  label: string
}

const props = defineProps<{
  quickLoading?: boolean
  filteredFields?: QuickAddFieldLike[]
  quickPageSize?: number
  quickTotal?: number
  quickPage?: number
  selectedCount?: number
  selectedMappableCount?: number
  selectedKeys?: string[]
  activeField?: QuickAddFieldLike | null
  types?: string[]
  sourceOptions?: QuickSourceOptionLike[]
  groupOptions?: QuickGroupLike[]
  rootGroupId?: string
  buildVarName: (name: string) => string
  isFieldSelectable?: (row: QuickAddFieldLike) => boolean
}>()

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'refresh'): void
  (event: 'pageChange', page: number): void
  (event: 'sizeChange', size: number): void
  (event: 'selectionChange', selection: unknown[]): void
  (event: 'statusClick', row: QuickAddFieldLike): void
  (event: 'singleConfirm'): void
  (event: 'singleClose'): void
  (event: 'singleTypeChange', type: string): void
}>()

const visible = defineModel<boolean>({ default: false })
const searchKey = defineModel<string>('searchKey', { default: '' })
const prefix = defineModel<string>('prefix', { default: '' })
const suffix = defineModel<string>('suffix', { default: '' })
const replaceFrom = defineModel<string>('replaceFrom', { default: '' })
const replaceTo = defineModel<string>('replaceTo', { default: '' })
const statusFilter = defineModel<string>('statusFilter', { default: '' })
const typeFilter = defineModel<string>('typeFilter', { default: '' })
const sourceIdFilter = defineModel<string>('sourceIdFilter', { default: '' })
const singleName = defineModel<string>('singleName', { default: '' })
const singleType = defineModel<string>('singleType', { default: 'string' })
const singleGroupId = defineModel<string>('singleGroupId', { default: '' })
const singleDefaultValue = defineModel<string>('singleDefaultValue', { default: '' })
const singleDescription = defineModel<string>('singleDescription', { default: '' })
const tableRef = ref<any>(null)
const syncingSelection = ref(false)

function resolveFieldSelectable(row: QuickAddFieldLike): boolean {
  return props.isFieldSelectable?.(row) ?? true
}

function resolveFieldKey(row: QuickAddFieldLike): string {
  return String(row.id || row.path || row.name || '')
}

const namingRuleVisible = ref(false)
const { t } = useI18n()

const namingSummary = computed(() => {
  const parts: string[] = []
  if (prefix.value) {
    parts.push(t('datapointPanel.quickAddDialog.prefixSummary', { value: prefix.value }))
  }
  if (suffix.value) {
    parts.push(t('datapointPanel.quickAddDialog.suffixSummary', { value: suffix.value }))
  }
  if (replaceFrom.value) {
    parts.push(
      t('datapointPanel.quickAddDialog.replaceSummary', {
        from: replaceFrom.value,
        to: replaceTo.value || '',
      }),
    )
  }
  return parts.length ? parts.join('，') : t('datapointPanel.quickAddDialog.defaultNaming')
})

const confirmDisabled = computed(() => !props.selectedMappableCount)

function handleSelectionChange(selection: unknown[]) {
  if (syncingSelection.value) return
  emit('selectionChange', selection)
}

function handleCurrentChange(page: number) {
  emit('pageChange', page)
}

function handleSizeChange(size: number) {
  emit('sizeChange', size)
}

function isActiveRow(row: QuickAddFieldLike): boolean {
  const active = props.activeField
  if (!active) return false
  const rowKey = resolveFieldKey(row)
  const activeKey = resolveFieldKey(active)
  return Boolean(rowKey && activeKey && rowKey === activeKey)
}

function resolveRowClassName({ row }: { row: QuickAddFieldLike }): string {
  const classes: string[] = []
  if (isActiveRow(row)) classes.push('is-active-datapoint')
  if (!resolveFieldSelectable(row)) classes.push('is-disabled-row')
  return classes.join(' ')
}

function resolveActionLabel(row: QuickAddFieldLike): string {
  if (row.mappedName) return t('datapointPanel.quickAddDialog.mappedAction')
  if (!resolveFieldSelectable(row)) return t('datapointPanel.quickAddDialog.unavailableAction')
  return t('datapointPanel.quickAddDialog.mapAction')
}

function syncPageSelection(): void {
  const table = tableRef.value
  if (!table) return
  const selectedKeys = new Set(props.selectedKeys || [])
  const rows = props.filteredFields || []
  syncingSelection.value = true
  table.clearSelection?.()
  rows.forEach((row) => {
    const key = resolveFieldKey(row)
    if (key && selectedKeys.has(key) && resolveFieldSelectable(row)) {
      table.toggleRowSelection?.(row, true)
    }
  })
  nextTick(() => {
    syncingSelection.value = false
  })
}

watch(
  [() => visible.value, () => props.filteredFields, () => (props.selectedKeys || []).join('|')],
  () => {
    if (!visible.value) return
    void nextTick(syncPageSelection)
  },
  { immediate: true },
)
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="t('datapointPanel.quickAddDialog.title')"
    width="min(1280px, 92vw)"
    top="3vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
    class="quick-add-dialog"
  >
    <div class="quick-searchbar">
      <el-input
        v-model="searchKey"
        class="quick-search"
        clearable
        size="small"
        :placeholder="t('datapointPanel.quickAddDialog.searchPlaceholder')"
        @keyup.enter="emit('refresh')"
      />
      <el-select
        v-model="statusFilter"
        class="quick-filter"
        size="small"
        :placeholder="t('datapointPanel.quickAddDialog.statusAll')"
        clearable
      >
        <el-option :label="t('datapointPanel.quickAddDialog.statusAll')" value="" />
        <el-option :label="t('datapointPanel.quickAddDialog.statusActive')" value="active" />
        <el-option :label="t('datapointPanel.quickAddDialog.statusInvalid')" value="invalid" />
        <el-option :label="t('datapointPanel.quickAddDialog.statusDisabled')" value="disabled" />
      </el-select>
      <el-select
        v-model="typeFilter"
        class="quick-filter"
        size="small"
        :placeholder="t('datapointPanel.quickAddDialog.typePlaceholder')"
        clearable
        filterable
      >
        <el-option v-for="item in props.types || []" :key="item" :label="item" :value="item" />
      </el-select>
      <el-select
        v-model="sourceIdFilter"
        class="quick-source-filter"
        size="small"
        :placeholder="t('datapointPanel.quickAddDialog.sourcePlaceholder')"
        clearable
        filterable
        allow-create
        default-first-option
      >
        <el-option
          v-for="source in props.sourceOptions || []"
          :key="source.id"
          :label="source.label"
          :value="source.id"
        />
      </el-select>
      <div class="quick-stats">
        <span>{{
          t('datapointPanel.quickAddDialog.totalCount', { count: props.quickTotal || 0 })
        }}</span>
        <span>{{
          t('datapointPanel.quickAddDialog.crossPageSelectedCount', {
            count: props.selectedCount || 0,
          })
        }}</span>
      </div>
      <el-button size="small" @click="emit('refresh')">
        {{ t('datapointPanel.quickAddDialog.refresh') }}
      </el-button>
    </div>

    <section class="quick-naming" :class="{ 'is-open': namingRuleVisible }">
      <button
        class="quick-naming__header"
        type="button"
        @click="namingRuleVisible = !namingRuleVisible"
      >
        <span>{{ t('datapointPanel.quickAddDialog.namingRule') }}</span>
        <span class="quick-naming__summary">{{ namingSummary }}</span>
      </button>
      <el-form
        v-if="namingRuleVisible"
        :inline="true"
        class="quick-form"
        label-width="70px"
        size="small"
      >
        <el-row :gutter="12" class="quick-form-row">
          <el-col :span="5">
            <el-form-item :label="t('datapointPanel.quickAddDialog.prefix')">
              <el-input
                v-model="prefix"
                :placeholder="t('datapointPanel.quickAddDialog.prefixPlaceholder')"
              />
            </el-form-item>
          </el-col>
          <el-col :span="5">
            <el-form-item :label="t('datapointPanel.quickAddDialog.suffix')">
              <el-input
                v-model="suffix"
                :placeholder="t('datapointPanel.quickAddDialog.suffixPlaceholder')"
              />
            </el-form-item>
          </el-col>
          <el-col :span="14">
            <el-form-item :label="t('datapointPanel.quickAddDialog.replace')" class="quick-replace">
              <el-input
                v-model="replaceFrom"
                :placeholder="t('datapointPanel.quickAddDialog.replaceFromPlaceholder')"
              />
              <span class="quick-arrow">→</span>
              <el-input
                v-model="replaceTo"
                :placeholder="t('datapointPanel.quickAddDialog.replaceToPlaceholder')"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </section>

    <div class="quick-body" :class="{ 'has-detail': props.activeField }">
      <div class="quick-table-wrap">
        <el-table
          ref="tableRef"
          v-loading="props.quickLoading"
          :data="props.filteredFields"
          border
          stripe
          size="small"
          height="520"
          :row-class-name="resolveRowClassName"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="46" :selectable="resolveFieldSelectable" />
          <el-table-column :label="t('datapointPanel.quickAddDialog.datapoint')" min-width="260">
            <template #default="{ row }">
              <div class="datapoint-cell">
                <span class="datapoint-cell__name">{{ row.name }}</span>
                <span class="datapoint-cell__path">{{ row.path }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('datapointPanel.quickAddDialog.typeSource')" width="150">
            <template #default="{ row }">
              <div class="meta-cell">
                <span>{{ row.typeLabel || row.type || '-' }}</span>
                <span>{{ row.sourceLabel || '-' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('datapointPanel.quickAddDialog.variableName')" min-width="160">
            <template #default="{ row }">
              <span class="variable-preview">{{ props.buildVarName(row.name) }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('datapointPanel.quickAddDialog.mappingStatus')" width="150">
            <template #default="{ row }">
              <el-tag size="small" :type="row.statusType || 'info'">{{ row.mappingLabel }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column
            prop="updatedAtLabel"
            :label="t('datapointPanel.quickAddDialog.updatedAt')"
            width="150"
          />
          <el-table-column
            :label="t('datapointPanel.quickAddDialog.action')"
            fixed="right"
            width="88"
          >
            <template #default="{ row }">
              <el-button
                size="small"
                text
                :disabled="!resolveFieldSelectable(row)"
                @click.stop="emit('statusClick', row)"
              >
                {{ resolveActionLabel(row) }}
              </el-button>
            </template>
          </el-table-column>
          <template #empty>
            <el-empty :description="t('datapointPanel.quickAddDialog.noData')" :image-size="80" />
          </template>
        </el-table>
      </div>

      <aside v-if="props.activeField" class="quick-detail">
        <div class="quick-detail__header">
          <div>
            <strong>{{ t('datapointPanel.quickAddDialog.singleTitle') }}</strong>
            <span>{{ props.activeField.name }}</span>
          </div>
          <el-button size="small" text @click="emit('singleClose')">
            {{ t('datapointPanel.quickAddDialog.closeDetail') }}
          </el-button>
        </div>
        <div class="quick-detail__path">{{ props.activeField.path }}</div>
        <el-form label-width="72px" size="small" class="quick-detail__form">
          <el-form-item :label="t('datapointPanel.quickAddDialog.variableName')">
            <el-input v-model="singleName" />
          </el-form-item>
          <el-form-item :label="t('datapointPanel.quickAddDialog.type')">
            <el-select
              v-model="singleType"
              @change="(value: string) => emit('singleTypeChange', value)"
            >
              <el-option
                v-for="item in props.types || []"
                :key="item"
                :label="item"
                :value="item"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('datapointPanel.quickAddDialog.group')">
            <el-select
              v-model="singleGroupId"
              :placeholder="t('datapointPanel.quickAddDialog.rootGroup')"
            >
              <el-option
                :label="t('datapointPanel.quickAddDialog.rootGroup')"
                :value="props.rootGroupId || '__root__'"
              />
              <el-option
                v-for="group in props.groupOptions || []"
                :key="group.id"
                :label="group.name"
                :value="group.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('datapointPanel.quickAddDialog.defaultValue')">
            <el-input v-model="singleDefaultValue" />
          </el-form-item>
          <el-form-item :label="t('datapointPanel.quickAddDialog.description')">
            <el-input v-model="singleDescription" type="textarea" :rows="3" />
          </el-form-item>
        </el-form>
        <div class="quick-detail__actions">
          <el-button size="small" @click="emit('singleClose')">
            {{ t('datapointPanel.quickAddDialog.cancel') }}
          </el-button>
          <el-button size="small" type="primary" @click="emit('singleConfirm')">
            {{ t('datapointPanel.quickAddDialog.saveSingle') }}
          </el-button>
        </div>
      </aside>
    </div>
    <div class="quick-pagination">
      <el-pagination
        background
        layout="prev, pager, next, sizes, total"
        :page-size="props.quickPageSize"
        :page-sizes="[50, 100, 200, 500]"
        :total="props.quickTotal"
        :current-page="props.quickPage"
        @current-change="handleCurrentChange"
        @size-change="handleSizeChange"
      />
    </div>

    <template #footer>
      <div class="quick-footer">
        <span>
          {{
            t('datapointPanel.quickAddDialog.selectionSummary', {
              selected: props.selectedCount || 0,
              mappable: props.selectedMappableCount || 0,
            })
          }}
        </span>
        <div class="quick-footer__actions">
          <el-button @click="visible = false">{{
            t('datapointPanel.quickAddDialog.cancel')
          }}</el-button>
          <el-button type="primary" :disabled="confirmDisabled" @click="emit('confirm')">
            {{ t('datapointPanel.quickAddDialog.addSelected') }}
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.quick-form {
  --quick-control-height: var(--el-component-size-small, 28px);
  margin-top: 10px;
}

.quick-form-row {
  width: 100%;
  margin-bottom: 10px;
}

.quick-form :deep(.el-form-item) {
  width: 100%;
  margin-bottom: 8px;
  align-items: center;
}

.quick-form :deep(.el-form-item__label) {
  line-height: var(--quick-control-height);
}

.quick-form :deep(.el-form-item__content) {
  flex: 1;
  min-width: 0;
}

.quick-form :deep(.el-input),
.quick-form :deep(.el-select) {
  width: 100%;
}

.quick-form :deep(.el-input__wrapper),
.quick-form :deep(.el-select .el-input__wrapper) {
  height: var(--quick-control-height);
  min-height: var(--quick-control-height);
}

.quick-form :deep(.el-input__inner),
.quick-form :deep(.el-select .el-input__inner) {
  height: var(--quick-control-height);
  line-height: var(--quick-control-height);
}

.quick-replace :deep(.el-form-item__content) {
  display: flex;
  align-items: center;
  gap: 6px;
}

.quick-replace :deep(.el-input) {
  flex: 1;
}

.quick-arrow {
  flex-shrink: 0;
  color: var(--designer-text-muted);
  flex: 0 0 auto;
}

.quick-searchbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.quick-search {
  flex: 1 1 320px;
  min-width: 260px;
}

.quick-filter {
  width: 130px;
  flex: 0 0 130px;
}

.quick-source-filter {
  width: 170px;
  flex: 0 0 170px;
}

.quick-stats {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 88px;
  font-size: 12px;
  color: var(--designer-text-muted);
  line-height: 1.35;
}

.quick-naming {
  border: 1px solid var(--designer-border-color);
  border-radius: 8px;
  margin-bottom: 10px;
  background: var(--designer-shell-surface);
}

.quick-naming__header {
  width: 100%;
  border: 0;
  background: transparent;
  padding: 8px 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  cursor: pointer;
  color: var(--designer-text-primary);
  font-size: 13px;
  text-align: left;
}

.quick-naming__summary {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--designer-text-muted);
  font-size: 12px;
  font-weight: 400;
}

.quick-naming.is-open {
  padding-bottom: 2px;
}

.quick-body {
  display: flex;
  gap: 12px;
  min-height: 0;
}

.quick-table-wrap {
  flex: 1;
  min-width: 0;
}

.quick-body.has-detail .quick-table-wrap {
  flex-basis: calc(100% - 340px);
}

.quick-detail {
  width: 320px;
  flex: 0 0 320px;
  border: 1px solid var(--designer-border-color);
  border-radius: 8px;
  padding: 12px;
  background: var(--designer-shell-surface);
}

.quick-detail__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.quick-detail__header div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.quick-detail__header strong {
  font-size: 13px;
  color: var(--designer-text-primary);
}

.quick-detail__header span,
.quick-detail__path {
  font-size: 12px;
  color: var(--designer-text-muted);
  word-break: break-all;
}

.quick-detail__path {
  margin: 8px 0 12px;
}

.quick-detail__form :deep(.el-select),
.quick-detail__form :deep(.el-input),
.quick-detail__form :deep(.el-textarea) {
  width: 100%;
}

.quick-detail__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.datapoint-cell,
.meta-cell {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.datapoint-cell__name {
  color: var(--designer-text-primary);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-cell__path,
.meta-cell span:last-child {
  color: var(--designer-text-muted);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.variable-preview {
  color: var(--designer-text-primary);
  font-family: var(--designer-mono-font, Consolas, monospace);
}

.status-button {
  border: 0;
  padding: 0;
  background: transparent;
  cursor: pointer;
}

:deep(.is-active-datapoint > td) {
  background: var(--designer-primary-soft) !important;
}

:deep(.is-disabled-row > td) {
  opacity: 0.5;
  cursor: not-allowed;
}

.quick-pagination {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}

.quick-footer {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--designer-text-muted);
  font-size: 12px;
}

.quick-footer__actions {
  display: flex;
  gap: 8px;
}
</style>
