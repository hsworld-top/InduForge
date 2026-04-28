<template>
  <div class="datapoint-list" :class="`datapoint-list--${mode}`">
    <div v-if="showToolbar" class="datapoint-list__toolbar">
      <div class="datapoint-list__toolbar-left">
        <el-input
          v-model="searchText"
          size="small"
          class="datapoint-list__search"
          clearable
          :prefix-icon="Search"
          placeholder="搜索名称、路径或来源"
        />
        <el-popover placement="bottom-start" :width="160" trigger="click">
          <template #reference>
            <button type="button" class="datapoint-list__pill-button">
              <Filter />
              <span>{{ currentStatusLabel }}</span>
            </button>
          </template>
          <div class="datapoint-list__popover-menu">
            <button
              v-for="option in statusOptions"
              :key="option.value"
              type="button"
              class="datapoint-list__popover-item"
              :class="{ 'is-active': statusFilter === option.value }"
              @click="statusFilter = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </el-popover>
        <div ref="tagFilterRootRef" class="datapoint-tag-filter">
          <button
            type="button"
            class="datapoint-tag-filter__trigger"
            :class="{ 'is-active': tagFilterVisible || tagFilterValues.length > 0 }"
            @click.stop="toggleTagFilterPanel"
          >
            <CollectionTag />
            <span>{{ tagFilterLabel }}</span>
          </button>

          <Transition name="datapoint-tag-filter">
            <div
              v-if="tagFilterVisible"
              class="datapoint-tag-filter__panel"
              @click.stop
            >
              <div class="datapoint-tag-filter__search">
                <Search />
                <input
                  v-model="tagFilterKeyword"
                  type="text"
                  placeholder="搜索标签"
                />
              </div>

              <div class="datapoint-tag-filter__list">
                <label
                  v-for="tag in filteredTagOptions"
                  :key="tag.value"
                  class="datapoint-tag-filter__item"
                >
                  <input
                    type="checkbox"
                    :checked="selectedTagSet.has(tag.value)"
                    @change="toggleTagFilter(tag.value)"
                  />
                  <span class="datapoint-tag-filter__name">
                    {{ tag.name }}
                  </span>
                  <button
                    type="button"
                    class="datapoint-tag-filter__delete"
                    :aria-label="`删除标签 ${tag.name}`"
                    @click.stop.prevent="handleDeleteTagOption(tag.value)"
                  >
                    <Delete />
                  </button>
                </label>
              </div>

              <p
                v-if="filteredTagOptions.length === 0"
                class="datapoint-tag-filter__empty"
              >
                暂无标签
              </p>

              <label class="datapoint-tag-filter__group-row">
                <input v-model="groupByTags" type="checkbox" />
                <span>按标签分组展示</span>
              </label>
            </div>
          </Transition>
        </div>
        <el-popover placement="bottom-start" :width="160" trigger="click">
          <template #reference>
            <button type="button" class="datapoint-list__pill-button">
              <span>{{ currentSortFieldLabel }}</span>
            </button>
          </template>
          <div class="datapoint-list__popover-menu">
            <button
              v-for="option in sortFieldOptions"
              :key="option.value"
              type="button"
              class="datapoint-list__popover-item"
              :class="{ 'is-active': sortField === option.value }"
              @click="sortField = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </el-popover>
        <button
          type="button"
          class="datapoint-list__pill-button"
          @click="toggleSortOrder"
        >
          <SortDown v-if="sortOrder === 'desc'" />
          <SortUp v-else />
          <span>{{ currentSortOrderLabel }}</span>
        </button>
      </div>
      <div class="datapoint-list__toolbar-right">
        <el-tooltip content="刷新数据点" placement="bottom">
          <button
            type="button"
            class="datapoint-list__icon-button"
            aria-label="刷新数据点"
            @click="handleRefresh"
          >
            <Refresh />
          </button>
        </el-tooltip>
        <el-tooltip content="清理选中的失效数据点" placement="bottom">
          <button
            type="button"
            class="datapoint-list__icon-button is-danger"
            :class="{ 'is-disabled': selectedInvalidRows.length === 0 }"
            :disabled="selectedInvalidRows.length === 0"
            aria-label="清理选中的失效数据点"
            @click="handleBatchDelete"
          >
            <Delete />
          </button>
        </el-tooltip>
      </div>
    </div>

    <div class="datapoint-list__panel">
      <div class="datapoint-list__content">
        <el-table
          v-loading="loading"
          :data="displayDataPoints"
          row-key="id"
          :size="isManagementMode ? 'default' : 'small'"
          class="datapoint-list__table"
          height="100%"
          @row-click="handleRowClick"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="56" fixed="left" align="center" />
          <el-table-column label="名称" width="220" fixed="left">
            <template #default="{ row }">
              <div class="datapoint-list__name-cell">
                <span class="datapoint-list__name" :title="row.name">
                  {{ row.name || "-" }}
                </span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="来源" width="110">
            <template #default="{ row }">
              <span class="datapoint-list__source-text">
                {{ formatSourceType(row.sourceType) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="类型" width="100">
            <template #default="{ row }">
              <span class="datapoint-list__mono">{{ row.dataType || "-" }}</span>
            </template>
          </el-table-column>
          <el-table-column label="标签" width="160">
            <template #default="{ row }">
              <div class="datapoint-list__tags-cell">
                <el-tag
                  v-for="tag in getVisibleTags(row)"
                  :key="tag"
                  size="small"
                  effect="plain"
                  class="datapoint-list__tag"
                >
                  {{ tag }}
                </el-tag>
                <el-tooltip
                  v-if="getHiddenTagCount(row) > 0"
                  :content="getHiddenTags(row).join('、')"
                  placement="top"
                >
                  <span class="datapoint-list__tag-more">
                    +{{ getHiddenTagCount(row) }}
                  </span>
                </el-tooltip>
                <button
                  v-if="normalizeTags(row.tags).length === 0"
                  type="button"
                  class="datapoint-list__add-tag"
                  @click.stop="openTagDialog(row)"
                >
                  + 标签
                </button>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="96" align="center" header-align="center">
            <template #default="{ row }">
              <span
                class="datapoint-list__status-badge"
                :class="{ 'is-invalid': row.status === 'invalid' }"
              >
                {{ formatStatus(row.status) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="写权限" width="240">
            <template #default="{ row }">
              <span class="datapoint-list__permission">
                {{ summarizeRuntimeGrant(getWriteRuntimeGrant(row)) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="190">
            <template #default="{ row }">
              <span class="datapoint-list__time">
                {{ formatTime(getCreatedAt(row)) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="190">
            <template #default="{ row }">
              <span class="datapoint-list__time">
                {{ formatTime(getUpdatedAt(row)) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="路径" min-width="280" show-overflow-tooltip>
            <template #default="{ row }">
              <div class="datapoint-list__path-cell">
                <el-tooltip :content="row.path || '-'" placement="top">
                  <span class="datapoint-list__path">{{ row.path || "-" }}</span>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="184" fixed="right" align="center" header-align="center">
            <template #default="{ row }">
              <div class="datapoint-list__row-actions">
                <el-tooltip content="查看详情" placement="top">
                  <button
                    type="button"
                    class="datapoint-list__table-action"
                    aria-label="查看数据点详情"
                    @click.stop="openDetailDrawer(row)"
                  >
                    <View />
                  </button>
                </el-tooltip>
                <el-tooltip content="标签管理" placement="top">
                  <button
                    type="button"
                    class="datapoint-list__table-action"
                    aria-label="标签管理"
                    @click.stop="openTagDialog(row)"
                  >
                    <PriceTag />
                  </button>
                </el-tooltip>
                <el-tooltip content="写权限" placement="top">
                  <button
                    type="button"
                    class="datapoint-list__table-action"
                    aria-label="编辑写权限"
                    @click.stop="openPermissionDialog(row)"
                  >
                    <User />
                  </button>
                </el-tooltip>
                <el-tooltip content="复制路径" placement="top">
                  <button
                    type="button"
                    class="datapoint-list__table-action"
                    aria-label="复制数据点路径"
                    @click.stop="copyPath(row.path)"
                  >
                    <CopyDocument />
                  </button>
                </el-tooltip>
                <el-tooltip
                  v-if="row.status === 'invalid'"
                  content="清理失效数据点"
                  placement="top"
                >
                  <button
                    type="button"
                    class="datapoint-list__table-action is-danger"
                    aria-label="清理失效数据点"
                    @click.stop="handleDelete(row)"
                  >
                    <Delete />
                  </button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
          <template #empty>
            <el-empty description="暂无匹配的数据点" />
          </template>
        </el-table>
      </div>

      <div v-if="isManagementMode" class="datapoint-list__pagination-bar">
        <p class="datapoint-list__pagination-summary">
          {{ paginationSummary }}
        </p>

        <div class="datapoint-list__pagination-controls">
          <span class="datapoint-list__pagination-label">每页</span>
          <el-popover placement="top" :width="100" trigger="click">
            <template #reference>
              <button type="button" class="datapoint-list__pagination-pill">
                {{ pagination.pageSize }}
                <ArrowDown />
              </button>
            </template>
            <div class="datapoint-list__page-size-menu">
              <button
                v-for="size in pageSizeOptions"
                :key="size"
                type="button"
                class="datapoint-list__page-size-item"
                :class="{ 'is-active': pagination.pageSize === size }"
                @click="handlePageSizeChange(size)"
              >
                {{ size }}
              </button>
            </div>
          </el-popover>

          <div class="datapoint-list__page-nav">
            <button
              type="button"
              class="datapoint-list__page-nav-button"
              :disabled="!canPrevPage"
              aria-label="上一页"
              @click="handlePrevPage"
            >
              <ArrowLeft />
            </button>
            <button
              type="button"
              class="datapoint-list__page-nav-button"
              :disabled="!canNextPage"
              aria-label="下一页"
              @click="handleNextPage"
            >
              <ArrowRight />
            </button>
          </div>

          <span class="datapoint-list__page-indicator">
            {{ pageIndicator }}
          </span>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="tagDialogVisible"
      :title="tagDialogTitle"
      width="520px"
      class="datapoint-list__tag-modal"
      destroy-on-close
      :show-close="false"
    >
      <template #header>
        <div class="datapoint-list__dialog-head">
          <h3>{{ tagDialogTitle }}</h3>
          <button
            type="button"
            class="datapoint-list__dialog-close"
            aria-label="关闭标签管理"
            @click="tagDialogVisible = false"
          >
            <Close />
          </button>
        </div>
      </template>
      <div class="datapoint-list__tag-dialog">
        <div class="datapoint-list__tag-form-row">
          <label>新建标签</label>
          <div class="datapoint-list__tag-create">
            <el-input
              v-model="tagCreateInput"
              placeholder="请输入标签名称"
              @keyup.enter="appendTagDraft"
            />
            <button
              type="button"
              class="datapoint-list__tag-create-button"
              @click="appendTagDraft"
            >
              新建标签
            </button>
          </div>
        </div>

        <div class="datapoint-list__tag-form-row">
          <label>标签筛选</label>
          <el-select
            v-model="tagDraft"
            multiple
            filterable
            allow-create
            default-first-option
            collapse-tags
            collapse-tags-tooltip
            placeholder="请选择或输入标签"
          >
            <el-option
              v-for="item in tagOptions"
              :key="item.value"
              :label="item.name"
              :value="item.value"
            />
          </el-select>
        </div>
      </div>
      <template #footer>
        <div class="datapoint-list__dialog-footer">
          <el-button @click="tagDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="tagSaving" @click="saveTags">
            保存
          </el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog
      v-model="permissionDialogVisible"
      :title="`写权限：${currentPermissionDatapoint?.name || '-'}`"
      width="520px"
      destroy-on-close
    >
      <div class="permission-dialog">
        <div class="permission-dialog__hint">
          运行态写权限只影响节点运行时是否允许写入该数据点，不改变开发态管理权限。
        </div>

        <el-form label-position="top">
          <el-form-item label="当前摘要">
            <el-tag type="info">
              {{ summarizeRuntimeGrant(draftWriteRuntimeGrant) }}
            </el-tag>
          </el-form-item>

          <el-form-item label="继承工程默认规则">
            <el-switch v-model="permissionForm.inherit" />
          </el-form-item>

          <el-form-item label="允许写入的角色">
            <el-input
              v-model="allowRolesInput"
              type="textarea"
              :rows="4"
              placeholder="每行一个角色，也可用逗号分隔"
            />
          </el-form-item>

          <el-form-item label="禁止写入的角色">
            <el-input
              v-model="denyRolesInput"
              type="textarea"
              :rows="4"
              placeholder="每行一个角色，也可用逗号分隔"
            />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="permissionDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="permissionSaving"
          @click="handlePermissionSave"
        >
          保存
        </el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="detailDrawerVisible"
      size="420px"
      title="数据点详情"
      destroy-on-close
    >
      <div v-if="detailDatapoint" class="datapoint-detail">
        <div class="datapoint-detail__header">
          <div class="datapoint-detail__name">
            {{ detailDatapoint.name || "-" }}
          </div>
          <el-tag
            size="small"
            :type="detailDatapoint.status === 'invalid' ? 'info' : 'success'"
          >
            {{ formatStatus(detailDatapoint.status) }}
          </el-tag>
        </div>
        <div class="datapoint-detail__path">
          <span>{{ detailDatapoint.path || "-" }}</span>
          <button
            type="button"
            class="datapoint-list__copy"
            aria-label="复制数据点路径"
            @click="copyPath(detailDatapoint.path)"
          >
            <CopyDocument />
          </button>
        </div>

        <dl class="datapoint-detail__grid">
          <div>
            <dt>来源</dt>
            <dd>{{ formatSourceType(detailDatapoint.sourceType) }}</dd>
          </div>
          <div>
            <dt>模块</dt>
            <dd>{{ resolveModuleName(detailDatapoint) }}</dd>
          </div>
          <div>
            <dt>数据类型</dt>
            <dd>{{ detailDatapoint.dataType || "-" }}</dd>
          </div>
          <div>
            <dt>质量</dt>
            <dd>{{ formatQuality(detailDatapoint.quality) }}</dd>
          </div>
          <div>
            <dt>当前值</dt>
            <dd>{{ formatLastValue(detailDatapoint.lastValue) }}</dd>
          </div>
          <div>
            <dt>写权限</dt>
            <dd>{{ summarizeRuntimeGrant(getWriteRuntimeGrant(detailDatapoint)) }}</dd>
          </div>
          <div>
            <dt>更新时间</dt>
            <dd>{{ formatTime(getUpdatedAt(detailDatapoint)) }}</dd>
          </div>
        </dl>

        <div class="datapoint-detail__section">
          <div class="datapoint-detail__section-title">标签</div>
          <div class="datapoint-detail__tags">
            <el-tag
              v-for="tag in normalizeTags(detailDatapoint.tags)"
              :key="tag"
              size="small"
              effect="plain"
            >
              {{ tag }}
            </el-tag>
            <span
              v-if="normalizeTags(detailDatapoint.tags).length === 0"
              class="datapoint-detail__muted"
            >
              未设置
            </span>
          </div>
        </div>

        <div class="datapoint-detail__section">
          <div class="datapoint-detail__section-title">来源配置</div>
          <pre class="datapoint-detail__code">{{
            formatSourceConfig(detailDatapoint.sourceConfig)
          }}</pre>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  Close,
  CollectionTag,
  CopyDocument,
  Delete,
  Filter,
  PriceTag,
  Refresh,
  Search,
  SortDown,
  SortUp,
  User,
  View,
} from "@element-plus/icons-vue";
import dayjs from "dayjs";
import { TIME_FORMAT } from "@/constants";
import dataAPI from "@/api/data.api";
import {
  normalizeRuntimeGrantPayload,
  summarizeRuntimeGrant,
} from "@/utils/runtime-permission-grants";
import { getApiErrorMessage } from "@/utils/request";

type RuntimeGrant = ReturnType<typeof normalizeRuntimeGrantPayload>;
type DataPointListMode = "embedded" | "management";
type SortField = "createdAt" | "updatedAt" | "name" | "path";
type SortOrder = "asc" | "desc";

interface DataPointRow {
  id: string;
  projectId?: string;
  path?: string;
  name?: string;
  description?: string | null;
  sourceType?: string;
  sourceId?: string | null;
  sourceConfig?: Record<string, unknown>;
  dataType?: string;
  runtimePermissions?: Record<string, unknown>;
  runtimePermissionGrants?: Record<string, unknown>;
  writePermission?: Record<string, unknown>;
  tags?: unknown[];
  status?: string;
  quality?: string;
  lastValue?: unknown;
  updatedAt?: string;
  updated_at?: string;
  createdAt?: string;
  created_at?: string;
}

const props = withDefaults(
  defineProps<{
    projectId: string;
    sourceType?: string;
    status?: string;
    search?: string;
    showToolbar?: boolean;
    mode?: DataPointListMode;
  }>(),
  {
    sourceType: undefined,
    status: undefined,
    search: undefined,
    showToolbar: true,
    mode: "embedded",
  },
);

const emit = defineEmits<{
  select: [row: DataPointRow];
}>();

const loading = ref(false);
const datapoints = ref<DataPointRow[]>([]);
const searchText = ref("");
const typeFilter = ref("");
const moduleFilter = ref("");
const tagFilterValues = ref<string[]>([]);
const tagFilterVisible = ref(false);
const tagFilterKeyword = ref("");
const groupByTags = ref(false);
const tagFilterRootRef = ref<HTMLElement | null>(null);
const statusFilter = ref("");
const sortField = ref<SortField>("createdAt");
const sortOrder = ref<SortOrder>("desc");
const debounceTimer = ref<number | null>(null);
const pagination = ref({
  page: 1,
  pageSize: 20,
});
const selectedRows = ref<DataPointRow[]>([]);
const permissionDialogVisible = ref(false);
const permissionSaving = ref(false);
const currentPermissionDatapoint = ref<DataPointRow | null>(null);
const permissionForm = ref<RuntimeGrant>(normalizeRuntimeGrantPayload());
const allowRolesInput = ref("");
const denyRolesInput = ref("");
const tagDialogVisible = ref(false);
const tagSaving = ref(false);
const tagDraft = ref<string[]>([]);
const tagCreateInput = ref("");
const tagEditMode = ref<"single" | "batch">("single");
const currentTagDatapoint = ref<DataPointRow | null>(null);
const detailDrawerVisible = ref(false);
const detailDatapoint = ref<DataPointRow | null>(null);

const sourceTypeOptions = [
  { label: "数据库查询", value: "db.query" },
  { label: "MQTT 变量", value: "mqtt.tag" },
  { label: "MQTT 订阅", value: "mqtt.subscription" },
  { label: "计算输出", value: "calc.output" },
  { label: "静态变量", value: "static.var" },
];

const statusOptions = [
  { label: "全部", value: "" },
  { label: "有效", value: "active" },
  { label: "失效", value: "invalid" },
];
const sortFieldOptions: Array<{ label: string; value: SortField }> = [
  { label: "创建时间", value: "createdAt" },
  { label: "更新时间", value: "updatedAt" },
  { label: "名称", value: "name" },
  { label: "路径", value: "path" },
];
const pageSizeOptions = [10, 20, 50, 100];

const isManagementMode = computed(() => props.mode === "management");

const sourceTypeLabels = sourceTypeOptions.reduce<Record<string, string>>(
  (result, item) => {
    result[item.value] = item.label;
    return result;
  },
  {},
);

const tagDialogTitle = computed(() =>
  tagEditMode.value === "batch"
    ? `标签管理（${selectedRows.value.length} 项）`
    : `标签管理：${currentTagDatapoint.value?.name || "-"}`,
);

const selectedInvalidRows = computed(() =>
  selectedRows.value.filter((item) => item.status === "invalid"),
);

const tagOptions = computed(() => {
  const tags = new Map<string, number>();
  datapoints.value.forEach((item) => {
    normalizeTags(item.tags).forEach((tag) => {
      tags.set(tag, (tags.get(tag) || 0) + 1);
    });
  });
  return Array.from(tags.entries())
    .sort((a, b) => a[0].localeCompare(b[0], "zh-Hans-CN"))
    .map(([value, count]) => ({
      value,
      name: value,
      count,
      label: `${value} (${count})`,
    }));
});

const filteredTagOptions = computed(() => {
  const keyword = tagFilterKeyword.value.trim().toLowerCase();
  if (!keyword) {
    return tagOptions.value;
  }
  return tagOptions.value.filter((tag) =>
    tag.name.toLowerCase().includes(keyword),
  );
});

const selectedTagSet = computed(() => new Set(tagFilterValues.value));

const tagFilterLabel = computed(() =>
  tagFilterValues.value.length > 0
    ? `标签筛选 (${tagFilterValues.value.length})`
    : "标签筛选",
);

const currentStatusLabel = computed(
  () =>
    statusOptions.find((option) => option.value === statusFilter.value)?.label ||
    "全部",
);

const currentSortFieldLabel = computed(
  () =>
    sortFieldOptions.find((option) => option.value === sortField.value)?.label ||
    "创建时间",
);

const currentSortOrderLabel = computed(() =>
  sortOrder.value === "desc" ? "降序" : "升序",
);

const moduleOptions = computed(() => {
  const modules = new Map<string, number>();
  datapoints.value.forEach((item) => {
    const moduleName = resolveModuleName(item);
    modules.set(moduleName, (modules.get(moduleName) || 0) + 1);
  });
  return Array.from(modules.entries())
    .sort((a, b) => a[0].localeCompare(b[0], "zh-Hans-CN"))
    .map(([value, count]) => ({
      value,
      label: `${value} (${count})`,
    }));
});

const getSortValue = (row: DataPointRow, field: SortField) => {
  if (field === "createdAt") {
    return row.createdAt || row.created_at || "";
  }
  if (field === "updatedAt") {
    return row.updatedAt || row.updated_at || row.createdAt || row.created_at || "";
  }
  return String(row[field] || "");
};

const compareDataPointRows = (left: DataPointRow, right: DataPointRow) => {
  const leftValue = getSortValue(left, sortField.value);
  const rightValue = getSortValue(right, sortField.value);
  let result = 0;

  if (sortField.value === "createdAt" || sortField.value === "updatedAt") {
    result = dayjs(leftValue).valueOf() - dayjs(rightValue).valueOf();
  } else {
    result = String(leftValue).localeCompare(String(rightValue), "zh-Hans-CN");
  }

  return sortOrder.value === "desc" ? -result : result;
};

const visibleDataPoints = computed(() => {
  const filtered = datapoints.value.filter((item) => {
    if (moduleFilter.value && resolveModuleName(item) !== moduleFilter.value) {
      return false;
    }
    if (
      tagFilterValues.value.length > 0 &&
      !tagFilterValues.value.every((tag) => normalizeTags(item.tags).includes(tag))
    ) {
      return false;
    }
    return true;
  });

  const sorted = [...filtered].sort(compareDataPointRows);

  if (!groupByTags.value) {
    return sorted;
  }

  return sorted.sort((left, right) => {
    const leftGroup = normalizeTags(left.tags)[0] || "未设置";
    const rightGroup = normalizeTags(right.tags)[0] || "未设置";
    const groupDelta = leftGroup.localeCompare(rightGroup, "zh-Hans-CN");
    if (groupDelta !== 0) return groupDelta;
    return String(left.name || left.path || "").localeCompare(
      String(right.name || right.path || ""),
      "zh-Hans-CN",
    );
  });
});

const totalVisibleCount = computed(() => visibleDataPoints.value.length);

const totalPages = computed(() => {
  if (!isManagementMode.value || totalVisibleCount.value === 0) return 0;
  return Math.ceil(totalVisibleCount.value / pagination.value.pageSize);
});

const currentPage = computed(() => {
  if (totalPages.value === 0) return 1;
  return Math.min(pagination.value.page, totalPages.value);
});

const pagedDataPoints = computed(() => {
  const start = (currentPage.value - 1) * pagination.value.pageSize;
  return visibleDataPoints.value.slice(start, start + pagination.value.pageSize);
});

const displayDataPoints = computed(() =>
  isManagementMode.value ? pagedDataPoints.value : visibleDataPoints.value,
);

const paginationSummary = computed(() => {
  if (totalVisibleCount.value === 0) return "共 0 条";
  const start = (currentPage.value - 1) * pagination.value.pageSize + 1;
  const end = Math.min(
    currentPage.value * pagination.value.pageSize,
    totalVisibleCount.value,
  );
  return `${start}-${end} / 共 ${totalVisibleCount.value} 条`;
});

const pageIndicator = computed(
  () => `${totalPages.value === 0 ? 0 : currentPage.value} / ${totalPages.value}`,
);

const canPrevPage = computed(
  () => totalPages.value > 0 && currentPage.value > 1,
);

const canNextPage = computed(
  () => totalPages.value > 0 && currentPage.value < totalPages.value,
);

/**
 * 外部工作区和列表内置工具栏共用同一个请求构造逻辑。
 * 搜索、来源和状态走后端过滤，模块与标签基于当前页做轻量前端过滤。
 */
const buildDataPointQueryParams = () => {
  const selectedSourceType = props.sourceType ?? typeFilter.value;
  const selectedStatus = props.status ?? statusFilter.value;
  const selectedSearch = props.search ?? searchText.value;
  const params: Record<string, unknown> = {
    page: 1,
    pageSize: 200,
  };

  if (selectedSourceType) {
    params.type = selectedSourceType;
  }
  if (selectedStatus) {
    params.status = selectedStatus;
  }
  if (selectedSearch) {
    params.search = selectedSearch;
  }

  return params;
};

const loadDataPoints = async () => {
  if (!props.projectId) return;
  loading.value = true;
  try {
    const response = await dataAPI.getDataPoints(props.projectId, {
      ...buildDataPointQueryParams(),
    });
    datapoints.value = response.data?.datapoints || [];
    selectedRows.value = [];
  } catch (error) {
    ElMessage.error(
      "加载数据点失败：" + getApiErrorMessage(error, "加载数据点失败"),
    );
  } finally {
    loading.value = false;
  }
};

const handleRefresh = async () => {
  await loadDataPoints();
  ElMessage.success("数据点已刷新");
};

const handleSelectionChange = (selection: DataPointRow[]) => {
  selectedRows.value = selection || [];
};

const handlePrevPage = () => {
  if (!canPrevPage.value) return;
  pagination.value.page = currentPage.value - 1;
};

const handleNextPage = () => {
  if (!canNextPage.value) return;
  pagination.value.page = currentPage.value + 1;
};

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize;
  pagination.value.page = 1;
};

const toggleSortOrder = () => {
  sortOrder.value = sortOrder.value === "desc" ? "asc" : "desc";
  pagination.value.page = 1;
};

const toggleTagFilterPanel = () => {
  tagFilterVisible.value = !tagFilterVisible.value;
  if (tagFilterVisible.value) {
    tagFilterKeyword.value = "";
  }
};

const toggleTagFilter = (tag: string) => {
  const next = new Set(tagFilterValues.value);
  if (next.has(tag)) {
    next.delete(tag);
  } else {
    next.add(tag);
  }
  tagFilterValues.value = [...next];
  pagination.value.page = 1;
};

const handleDocumentClick = (event: MouseEvent) => {
  if (!tagFilterVisible.value) return;
  const target = event.target as Node | null;
  if (target && tagFilterRootRef.value?.contains(target)) {
    return;
  }
  tagFilterVisible.value = false;
};

const openDetailDrawer = (row: DataPointRow) => {
  detailDatapoint.value = row;
  detailDrawerVisible.value = true;
  emit("select", row);
};

const handleRowClick = (row: DataPointRow) => {
  if (!isManagementMode.value) {
    emit("select", row);
  }
};

const handleDelete = async (datapoint: DataPointRow) => {
  try {
    await ElMessageBox.confirm(
      `确认清理失效数据点「${datapoint.name || datapoint.path || "-"}」？`,
      "清理失效数据点",
      {
        confirmButtonText: "清理",
        cancelButtonText: "取消",
        type: "warning",
      },
    );
    await dataAPI.deleteDataPoint(props.projectId, datapoint.id);
    ElMessage.success("数据点已清理");
    await loadDataPoints();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        "删除数据点失败：" + getApiErrorMessage(error, "删除数据点失败"),
      );
    }
  }
};

const handleBatchDelete = async () => {
  if (selectedInvalidRows.value.length === 0) return;
  try {
    await ElMessageBox.confirm(
      `确认清理 ${selectedInvalidRows.value.length} 个失效数据点？`,
      "批量清理失效项",
      {
        confirmButtonText: "批量清理",
        cancelButtonText: "取消",
        type: "warning",
      },
    );
    const response = await dataAPI.deleteDataPointsBatch(
      props.projectId,
      selectedInvalidRows.value.map((item) => item.id),
    );
    const deletedCount = response?.data?.deletedCount ?? 0;
    ElMessage.success(`已清理 ${deletedCount} 个失效数据点`);
    selectedRows.value = [];
    await loadDataPoints();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        "批量删除数据点失败：" +
          getApiErrorMessage(error, "批量删除数据点失败"),
      );
    }
  }
};

const openTagDialog = (row: DataPointRow) => {
  currentTagDatapoint.value = row;
  tagEditMode.value = "single";
  tagDraft.value = normalizeTags(row.tags);
  tagCreateInput.value = "";
  tagDialogVisible.value = true;
};

const openBatchTagDialog = () => {
  if (selectedRows.value.length === 0) return;
  currentTagDatapoint.value = null;
  tagEditMode.value = "batch";
  tagDraft.value = [];
  tagCreateInput.value = "";
  tagDialogVisible.value = true;
};

const appendTagDraft = () => {
  const tag = tagCreateInput.value.trim();
  if (!tag) return;
  tagDraft.value = mergeTags(tagDraft.value, [tag]);
  tagCreateInput.value = "";
};

const handleDeleteTagOption = async (tag: string) => {
  const affectedRows = datapoints.value.filter((row) =>
    normalizeTags(row.tags).includes(tag),
  );
  if (affectedRows.length === 0) return;

  try {
    await ElMessageBox.confirm(
      `确认从 ${affectedRows.length} 个数据点中移除标签「${tag}」？`,
      "删除标签",
      {
        confirmButtonText: "删除",
        cancelButtonText: "取消",
        type: "warning",
      },
    );
    await Promise.all(
      affectedRows.map((row) =>
        dataAPI.updateDataPoint(props.projectId, row.id, {
          tags: normalizeTags(row.tags).filter((item) => item !== tag),
        }),
      ),
    );
    tagFilterValues.value = tagFilterValues.value.filter((item) => item !== tag);
    tagDraft.value = tagDraft.value.filter((item) => item !== tag);
    ElMessage.success("标签已删除");
    await loadDataPoints();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        "删除标签失败：" + getApiErrorMessage(error, "删除标签失败"),
      );
    }
  }
};

const saveTags = async () => {
  const normalizedTags = normalizeTags(tagDraft.value);
  tagSaving.value = true;
  try {
    if (tagEditMode.value === "single") {
      if (!currentTagDatapoint.value?.id) return;
      await dataAPI.updateDataPoint(
        props.projectId,
        currentTagDatapoint.value.id,
        {
          tags: normalizedTags,
        },
      );
      ElMessage.success("标签已保存");
    } else {
      await Promise.all(
        selectedRows.value.map((row) => {
          const nextTags = mergeTags(normalizeTags(row.tags), normalizedTags);
          return dataAPI.updateDataPoint(props.projectId, row.id, {
            tags: nextTags,
          });
        }),
      );
      ElMessage.success(`已为 ${selectedRows.value.length} 个数据点更新标签`);
    }
    tagDialogVisible.value = false;
    await loadDataPoints();
  } catch (error) {
    ElMessage.error(
      "保存标签失败：" + getApiErrorMessage(error, "保存标签失败"),
    );
  } finally {
    tagSaving.value = false;
  }
};

const parseRoleInput = (value: string) => {
  return String(value || "")
    .split(/[\n,，]/)
    .map((item) => item.trim())
    .filter(Boolean);
};

const getWriteRuntimeGrant = (row: DataPointRow) => {
  const runtimePermissions = row?.runtimePermissions as
    | { write?: unknown }
    | undefined;
  const runtimePermissionGrants = row?.runtimePermissionGrants as
    | { write?: unknown }
    | undefined;
  return normalizeRuntimeGrantPayload(
    runtimePermissions?.write ||
      runtimePermissionGrants?.write ||
      row?.writePermission ||
      {},
  );
};

const draftWriteRuntimeGrant = computed(() =>
  normalizeRuntimeGrantPayload({
    allowRoles: parseRoleInput(allowRolesInput.value),
    denyRoles: parseRoleInput(denyRolesInput.value),
    inherit: permissionForm.value.inherit,
  }),
);

const openPermissionDialog = (row: DataPointRow) => {
  const currentGrant = getWriteRuntimeGrant(row);
  currentPermissionDatapoint.value = row;
  permissionForm.value = currentGrant;
  allowRolesInput.value = currentGrant.allowRoles.join("\n");
  denyRolesInput.value = currentGrant.denyRoles.join("\n");
  permissionDialogVisible.value = true;
};

const handlePermissionSave = async () => {
  if (!props.projectId || !currentPermissionDatapoint.value?.id) {
    return;
  }

  const writeGrant = draftWriteRuntimeGrant.value;
  permissionSaving.value = true;
  try {
    await dataAPI.updateDatapointRuntimePermissions(
      props.projectId,
      currentPermissionDatapoint.value.id,
      { write: writeGrant },
    );
    permissionForm.value = writeGrant;
    ElMessage.success("写权限已保存");
    permissionDialogVisible.value = false;
    await loadDataPoints();
  } catch (error) {
    ElMessage.error(
      "保存运行态权限失败：" + getApiErrorMessage(error, "保存运行态权限失败"),
    );
  } finally {
    permissionSaving.value = false;
  }
};

const copyPath = async (path?: string) => {
  if (!path) return;
  try {
    await navigator.clipboard.writeText(path);
    ElMessage.success("已复制数据点路径");
  } catch {
    ElMessage.error("复制失败，请手动复制");
  }
};

const formatTime = (value?: string) => {
  if (!value) return "-";
  return dayjs(value).format(TIME_FORMAT);
};

const getUpdatedAt = (row: DataPointRow) => {
  return (
    row.updatedAt || row.updated_at || row.createdAt || row.created_at || ""
  );
};

const getCreatedAt = (row: DataPointRow) => {
  return row.createdAt || row.created_at || "";
};

const formatStatus = (status?: string) => {
  switch (status) {
    case "invalid":
      return "失效";
    case "inactive":
      return "停用";
    default:
      return "有效";
  }
};

const formatSourceType = (sourceType?: string) => {
  if (!sourceType) return "-";
  return sourceTypeLabels[sourceType] || sourceType;
};

const formatQuality = (quality?: string) => {
  switch (quality) {
    case "good":
      return "良好";
    case "bad":
      return "异常";
    default:
      return "未知";
  }
};

const formatLastValue = (value: unknown) => {
  if (value === undefined || value === null || value === "") return "-";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
};

const formatSourceConfig = (value?: Record<string, unknown>) => {
  if (!value || Object.keys(value).length === 0) return "{}";
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return "{}";
  }
};

const resolveModuleName = (row: DataPointRow) => {
  const path = String(row.path || "");
  const parts = path.split(/[./:]/).filter(Boolean);
  if (parts.length >= 2) {
    return parts[1];
  }
  return parts[0] || "未分组";
};

const normalizeTags = (value: unknown): string[] => {
  if (!Array.isArray(value)) return [];
  const result: string[] = [];
  value.forEach((item) => {
    let label = "";
    if (typeof item === "string") {
      label = item;
    } else if (item && typeof item === "object") {
      const record = item as Record<string, unknown>;
      label = String(record.label || record.name || record.value || "");
    }
    const normalized = label.trim();
    if (normalized && !result.includes(normalized)) {
      result.push(normalized);
    }
  });
  return result;
};

const mergeTags = (left: string[], right: string[]) => {
  return Array.from(
    new Set([...left, ...right].map((item) => item.trim())),
  ).filter(Boolean);
};

const getVisibleTags = (row: DataPointRow) => normalizeTags(row.tags).slice(0, 3);

const getHiddenTags = (row: DataPointRow) => normalizeTags(row.tags).slice(3);

const getHiddenTagCount = (row: DataPointRow) => getHiddenTags(row).length;

watch([searchText, typeFilter, statusFilter], () => {
  pagination.value.page = 1;
  if (debounceTimer.value) {
    window.clearTimeout(debounceTimer.value);
  }
  debounceTimer.value = window.setTimeout(() => {
    void loadDataPoints();
  }, 300);
});

watch([moduleFilter, tagFilterValues, groupByTags, sortField, sortOrder], () => {
  pagination.value.page = 1;
});

watch([totalVisibleCount, totalPages], () => {
  if (totalPages.value === 0) {
    pagination.value.page = 1;
    return;
  }
  if (pagination.value.page > totalPages.value) {
    pagination.value.page = totalPages.value;
  }
});

watch(
  () => [props.sourceType, props.status, props.search],
  () => {
    pagination.value.page = 1;
    if (debounceTimer.value) {
      window.clearTimeout(debounceTimer.value);
    }
    debounceTimer.value = window.setTimeout(() => {
      void loadDataPoints();
    }, 300);
  },
);

onMounted(() => {
  document.addEventListener("click", handleDocumentClick);
  void loadDataPoints();
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleDocumentClick);
  if (debounceTimer.value) {
    window.clearTimeout(debounceTimer.value);
  }
});

defineExpose({
  refresh: loadDataPoints,
});
</script>

<style scoped>
.datapoint-list {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  box-shadow: var(--dc-shadow-surface);
}

.datapoint-list--management {
  gap: 12px;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.datapoint-list__toolbar {
  position: relative;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.datapoint-list--management .datapoint-list__toolbar {
  flex-shrink: 0;
  gap: 12px;
  border: 1px solid var(--dc-border);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.75);
  box-shadow: var(--dc-shadow-surface);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
}

.datapoint-list--management .datapoint-list__toolbar :deep(.el-input__wrapper) {
  border-radius: var(--dc-radius-md);
  padding: 6px 12px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  box-shadow: none;
  transition: all 0.2s ease;
}

.datapoint-list--management .datapoint-list__toolbar :deep(.el-input__wrapper:hover) {
  border-color: rgba(37, 99, 235, 0.22);
}

.datapoint-list--management .datapoint-list__toolbar :deep(.el-input__wrapper:focus-within),
.datapoint-list--management .datapoint-list__toolbar :deep(.el-input__wrapper.is-focus) {
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 3px var(--dc-primary-soft);
}

.datapoint-list--management .datapoint-list__toolbar :deep(.el-select .el-input__wrapper) {
  height: 32px;
  padding: 4px 10px;
  border: 0;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
  box-shadow: none;
  font-size: 13px;
}

.datapoint-list--management .datapoint-list__toolbar :deep(.el-select .el-input__wrapper:hover) {
  background: rgba(0, 0, 0, 0.08);
}

.datapoint-list--management .datapoint-list__search {
  width: 176px;
  flex: 0 0 176px;
  --el-input-bg-color: var(--dc-surface-muted);
  --el-input-border-color: var(--dc-border);
  --el-input-hover-border-color: var(--dc-primary);
  --el-input-focus-border-color: var(--dc-primary);
  --el-input-placeholder-color: var(--dc-text-muted);
}

.datapoint-list--management .datapoint-list__search :deep(.el-input__wrapper) {
  height: 32px;
  min-height: 32px;
  border-radius: var(--dc-radius-md);
  padding: 6px 12px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  box-shadow: none;
}

.datapoint-list--management .datapoint-list__search :deep(.el-input__inner) {
  color: var(--dc-text);
  font-size: 14px;
}

.datapoint-list--management .datapoint-list__search :deep(.el-input__prefix-inner) {
  color: var(--dc-text-muted);
}

.datapoint-list__toolbar-left,
.datapoint-list__toolbar-right {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  min-width: 0;
}

.datapoint-list__toolbar-left {
  flex: 1 1 auto;
}

.datapoint-list__toolbar-right {
  gap: 8px;
  margin-left: auto;
}

.datapoint-list__search {
  width: 176px;
  flex: 0 0 176px;
}

.datapoint-list__select {
  width: 126px;
  flex: 0 0 126px;
}

.datapoint-list__status {
  flex: 0 0 auto;
}

.datapoint-list__pill-button {
  height: 32px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border: 0;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 400;
  white-space: nowrap;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
}

.datapoint-list__pill-button:hover,
.datapoint-list__pill-button:focus-visible {
  background: rgba(0, 0, 0, 0.08);
  color: var(--dc-text);
  outline: 0;
}

.datapoint-list__pill-button svg {
  width: 16px;
  height: 16px;
}

.datapoint-list__popover-menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: -6px -8px;
}

.datapoint-list__popover-item {
  width: 100%;
  padding: 7px 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  text-align: left;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.datapoint-list__popover-item:hover {
  background: rgba(0, 0, 0, 0.04);
  color: var(--dc-text);
}

.datapoint-list__popover-item.is-active {
  background: rgba(29, 78, 216, 0.12);
  color: var(--dc-primary);
  font-weight: 600;
}

.datapoint-tag-filter {
  position: relative;
  display: inline-flex;
  flex: 0 0 auto;
}

.datapoint-tag-filter__trigger {
  height: 32px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 400;
  white-space: nowrap;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
}

.datapoint-tag-filter__trigger:hover,
.datapoint-tag-filter__trigger.is-active {
  background: rgba(0, 0, 0, 0.08);
  color: var(--dc-text);
}

.datapoint-tag-filter__trigger svg {
  width: 16px;
  height: 16px;
}

.datapoint-tag-filter__panel {
  position: absolute;
  top: calc(100% + 10px);
  left: 0;
  z-index: 200;
  width: 280px;
  padding: 12px 14px;
  border: 1px solid var(--dc-border);
  border-radius: 14px;
  background: var(--dc-surface-raised);
  box-shadow: 0 16px 36px rgba(15, 23, 42, 0.16);
}

.datapoint-tag-filter__search {
  height: 32px;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  padding: 0 9px;
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  color: var(--dc-text-muted);
}

.datapoint-tag-filter__search svg {
  width: 16px;
  height: 16px;
}

.datapoint-tag-filter__search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--dc-text);
  font-family: inherit;
  font-size: 13px;
}

.datapoint-tag-filter__list {
  max-height: 220px;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.datapoint-tag-filter__item,
.datapoint-tag-filter__group-row {
  min-height: 42px;
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-size: 13px;
}

.datapoint-tag-filter__item input,
.datapoint-tag-filter__group-row input {
  width: 15px;
  height: 15px;
  margin: 0;
  accent-color: var(--dc-primary);
}

.datapoint-tag-filter__name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-tag-filter__delete {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
}

.datapoint-tag-filter__delete:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.datapoint-tag-filter__delete svg {
  width: 15px;
  height: 15px;
}

.datapoint-tag-filter__empty {
  margin: 8px 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  text-align: center;
}

.datapoint-tag-filter__group-row {
  margin-top: 8px;
  padding-top: 10px;
  border-top: 1px solid var(--dc-border);
  font-size: 14px;
}

.datapoint-tag-filter-enter-active,
.datapoint-tag-filter-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.datapoint-tag-filter-enter-from,
.datapoint-tag-filter-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.datapoint-list__count {
  height: 32px;
  display: inline-flex;
  align-items: center;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: 10px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.datapoint-list--management .datapoint-list__count {
  border: 0;
  background: rgba(0, 0, 0, 0.04);
  color: var(--dc-text-secondary);
  font-size: 13px;
  font-weight: 400;
}

.datapoint-list__icon-button {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  transition:
    color 0.18s ease,
    background 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.datapoint-list--management .datapoint-list__icon-button {
  border-color: var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.datapoint-list--management .datapoint-list__icon-button:hover {
  transform: translateY(-1px);
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.datapoint-list__icon-button svg {
  width: 16px;
  height: 16px;
}

.datapoint-list__icon-button:hover {
  border-color: rgba(37, 99, 235, 0.24);
  background: var(--dc-primary-soft);
  color: var(--dc-text);
  box-shadow: var(--dc-shadow-surface);
}

.datapoint-list__icon-button.is-danger:hover {
  border-color: rgba(220, 38, 38, 0.26);
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger);
}

.datapoint-list__icon-button.is-disabled,
.datapoint-list__icon-button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
  box-shadow: none;
}

.datapoint-list--management .datapoint-list__icon-button:hover {
  transform: translateY(-1px);
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.datapoint-list--management .datapoint-list__icon-button.is-danger:hover {
  border-color: rgba(220, 38, 38, 0.26);
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger);
}

.datapoint-list__panel {
  position: relative;
  z-index: 1;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.datapoint-list--management .datapoint-list__panel {
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: 18px;
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.datapoint-list__content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background: var(--dc-surface-raised);
}

.datapoint-list--management .datapoint-list__content {
  padding: 20px 24px 0;
  background: transparent;
}

.datapoint-list__table {
  width: 100%;
  min-width: 1360px;
}

.datapoint-list__name-cell {
  min-width: 0;
  display: flex;
  align-items: center;
}

.datapoint-list__name-row,
.datapoint-list__tag-row,
.datapoint-list__path-cell,
.datapoint-list__module {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.datapoint-list__name {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-list--management .datapoint-list__name {
  color: #2563eb;
  font-size: 14px;
  font-weight: 500;
}

.datapoint-list__tag-row {
  min-height: 22px;
  flex-wrap: wrap;
}

.datapoint-list__inline-action,
.datapoint-list__copy {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  border: 0;
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
  font-size: 12px;
}

.datapoint-list__inline-action {
  height: 22px;
  padding: 0 6px;
  border-radius: 6px;
}

.datapoint-list__inline-action svg,
.datapoint-list__copy svg,
.datapoint-list__module svg {
  width: 14px;
  height: 14px;
}

.datapoint-list__inline-action:hover,
.datapoint-list__copy:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.datapoint-list__inline-action.is-danger:hover {
  color: var(--dc-danger);
}

.datapoint-list__tags-cell {
  min-height: 28px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px;
  overflow: hidden;
}

.datapoint-list__tag {
  max-width: 72px;
  --el-tag-bg-color: var(--dc-primary-soft);
  --el-tag-border-color: transparent;
  --el-tag-text-color: var(--dc-primary);
  border-radius: 999px;
  font-weight: 700;
}

.datapoint-list__tag :deep(.el-tag__content) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-list__tag-more {
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
}

.datapoint-list__tag-empty {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.datapoint-list__add-tag {
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px dashed var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface);
  color: var(--dc-text-muted);
  cursor: pointer;
  font-size: 11px;
  font-weight: 700;
}

.datapoint-list__add-tag:hover {
  border-color: rgba(37, 99, 235, 0.28);
  color: var(--dc-primary);
}

.datapoint-list__source-text {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  color: #64748b;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-list__module {
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-list__mono,
.datapoint-list__time,
.datapoint-list__path,
.datapoint-list__value {
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.datapoint-list--management .datapoint-list__mono,
.datapoint-list--management .datapoint-list__time,
.datapoint-list--management .datapoint-list__path,
.datapoint-list--management .datapoint-list__permission {
  color: #475569;
  font-family: inherit;
  font-size: 14px;
  font-weight: 400;
}

.datapoint-list__value {
  display: inline-block;
  max-width: 140px;
  overflow: hidden;
  color: var(--dc-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-list__value.is-muted {
  color: var(--dc-text-muted);
}

.datapoint-list__permission {
  display: inline-block;
  max-width: none;
  overflow: visible;
  color: var(--dc-text-secondary);
  font-size: 12px;
  text-align: left;
  text-overflow: clip;
  white-space: nowrap;
}

.datapoint-list__status-badge {
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: var(--dc-success-soft);
  color: var(--dc-success);
  font-size: 12px;
  font-weight: 600;
}

.datapoint-list--management .datapoint-list__status-badge {
  border-color: rgba(34, 197, 94, 0.2);
  background: rgba(34, 197, 94, 0.1);
  color: #15803d;
  font-weight: 500;
}

.datapoint-list__status-badge.is-invalid {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.datapoint-list--management .datapoint-list__status-badge.is-invalid {
  border-color: rgba(239, 68, 68, 0.22);
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
}

.datapoint-list__row-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
}

.datapoint-list__table-action {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
  transition:
    color 0.18s ease,
    background 0.18s ease,
    border-color 0.18s ease;
}

.datapoint-list__table-action svg {
  width: 15px;
  height: 15px;
}

.datapoint-list__table-action:hover {
  border-color: rgba(37, 99, 235, 0.2);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.datapoint-list--management .datapoint-list__table-action:hover {
  border-color: transparent;
  background: transparent;
  color: #64748b;
}

.datapoint-list__table-action.is-danger:hover,
.datapoint-list--management .datapoint-list__table-action.is-danger:hover {
  border-color: transparent;
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger);
}

.datapoint-list__path {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--dc-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-list--management .datapoint-list__path {
  max-width: 100%;
  color: #64748b;
}

.datapoint-list__copy {
  width: 24px;
  height: 24px;
  flex: 0 0 24px;
  border-radius: var(--dc-radius-sm);
}

.datapoint-list__copy:hover {
  background: var(--dc-primary-soft);
}

.datapoint-list__tag-dialog,
.permission-dialog {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

:deep(.datapoint-list__tag-modal) {
  border-radius: 14px;
}

:deep(.datapoint-list__tag-modal .el-dialog__header) {
  margin: 0;
  padding: 0;
}

:deep(.datapoint-list__tag-modal .el-dialog__body) {
  padding: 32px 36px 30px;
}

:deep(.datapoint-list__tag-modal .el-dialog__footer) {
  padding: 0 36px 34px;
}

.datapoint-list__dialog-head {
  min-height: 80px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 0 36px;
  border-radius: 14px 14px 0 0;
  background: rgba(248, 250, 252, 0.86);
}

.datapoint-list__dialog-head h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 18px;
  font-weight: 500;
}

.datapoint-list__dialog-close {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
}

.datapoint-list__dialog-close:hover {
  background: rgba(0, 0, 0, 0.04);
  color: var(--dc-text);
}

.datapoint-list__dialog-close svg {
  width: 18px;
  height: 18px;
}

.datapoint-list__tag-dialog {
  gap: 22px;
  padding-bottom: 34px;
  border-bottom: 1px solid var(--dc-border);
}

.datapoint-list__tag-form-row {
  display: grid;
  grid-template-columns: 92px minmax(0, 1fr);
  align-items: center;
  gap: 14px;
}

.datapoint-list__tag-form-row > label {
  color: var(--dc-text-secondary);
  font-size: 14px;
  font-weight: 600;
  text-align: right;
  white-space: nowrap;
}

.datapoint-list__tag-create {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 112px;
  gap: 10px;
}

.datapoint-list__tag-create-button {
  height: 32px;
  border: 1px solid rgba(64, 158, 255, 0.42);
  border-radius: var(--dc-radius-md);
  background: rgba(64, 158, 255, 0.08);
  color: #409eff;
  cursor: pointer;
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
}

.datapoint-list__tag-create-button:hover {
  background: rgba(64, 158, 255, 0.14);
}

.datapoint-list__tag-form-row :deep(.el-select),
.datapoint-list__tag-form-row :deep(.el-input) {
  width: 100%;
}

.datapoint-list__dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.datapoint-list__dialog-footer :deep(.el-button) {
  min-width: 76px;
  height: 32px;
  border-radius: var(--dc-radius-md);
  font-size: 14px;
  font-weight: 600;
}

.permission-dialog__hint {
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.datapoint-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: var(--dc-text);
}

.datapoint-detail__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--dc-border);
}

.datapoint-detail__name {
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 16px;
  font-weight: 700;
  line-height: 1.45;
}

.datapoint-detail__path {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.datapoint-detail__path span {
  min-width: 0;
  flex: 1;
}

.datapoint-detail__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin: 0;
}

.datapoint-detail__grid > div {
  min-width: 0;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface);
}

.datapoint-detail__grid dt {
  margin: 0 0 5px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.datapoint-detail__grid dd {
  min-width: 0;
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.45;
}

.datapoint-detail__section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.datapoint-detail__section-title {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.datapoint-detail__tags {
  min-height: 32px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.datapoint-detail__muted {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.datapoint-detail__code {
  max-height: 240px;
  overflow: auto;
  margin: 0;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.datapoint-list__pagination-bar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 28px;
  border-top: 1px solid rgba(15, 23, 42, 0.08);
  background: var(--dc-surface-raised);
}

.datapoint-list__pagination-summary {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  white-space: nowrap;
}

.datapoint-list__pagination-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.datapoint-list__pagination-label,
.datapoint-list__page-indicator {
  color: var(--dc-text-muted);
  font-size: 12px;
  white-space: nowrap;
}

.datapoint-list__page-indicator {
  min-width: 64px;
  text-align: right;
}

.datapoint-list__pagination-pill {
  height: 28px;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0 10px;
  border: 0;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.datapoint-list__pagination-pill:hover {
  background: rgba(0, 0, 0, 0.08);
  color: var(--dc-text);
}

.datapoint-list__pagination-pill svg {
  width: 12px;
  height: 12px;
}

.datapoint-list__page-nav {
  display: flex;
  padding: 2px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
}

.datapoint-list__page-nav-button {
  width: 28px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.datapoint-list__page-nav-button:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.6);
  color: var(--dc-text);
}

.datapoint-list__page-nav-button:disabled {
  cursor: not-allowed;
  opacity: 0.32;
}

.datapoint-list__page-nav-button svg {
  width: 14px;
  height: 14px;
}

.datapoint-list__page-size-menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: -6px -8px;
}

.datapoint-list__page-size-item {
  display: block;
  width: 100%;
  padding: 6px 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  text-align: center;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.datapoint-list__page-size-item:hover {
  background: rgba(0, 0, 0, 0.04);
  color: var(--dc-text);
}

.datapoint-list__page-size-item.is-active {
  background: rgba(29, 78, 216, 0.12);
  color: var(--dc-primary);
  font-weight: 600;
}

:deep(.el-input__wrapper),
:deep(.el-select .el-input__wrapper) {
  border-radius: var(--dc-radius-md);
  box-shadow: none;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
}

:deep(.el-input__wrapper:hover),
:deep(.el-select .el-input__wrapper:hover) {
  border-color: rgba(37, 99, 235, 0.22);
}

:deep(.el-input__wrapper.is-focus),
:deep(.el-input__wrapper:focus-within),
:deep(.el-select .el-input__wrapper.is-focus) {
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 3px var(--dc-primary-soft);
}

:deep(.el-segmented) {
  --el-segmented-bg-color: var(--dc-surface-muted);
  --el-segmented-item-selected-bg-color: var(--dc-surface);
  --el-segmented-item-selected-color: var(--dc-primary);
  --el-segmented-item-hover-bg-color: var(--dc-surface);
  border: 1px solid var(--dc-border);
  border-radius: 10px;
  padding: 3px;
}

:deep(.el-segmented__item) {
  min-width: 48px;
  height: 26px;
  border-radius: 8px;
}

:deep(.el-table) {
  --el-table-header-bg-color: var(--dc-surface-muted);
  --el-table-tr-bg-color: var(--dc-surface-raised);
  --el-table-row-hover-bg-color: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
}

:deep(.el-table th.el-table__cell) {
  color: var(--dc-text-secondary);
  font-weight: 800;
}

:deep(.el-table th.el-table__cell .cell) {
  height: 38px;
  display: flex;
  align-items: center;
}

:deep(.el-table .el-table__cell) {
  border-bottom-color: var(--dc-border);
}

:deep(.el-table__body .el-table__cell) {
  height: 48px;
  padding: 8px 0;
}

:deep(.el-table__row) {
  cursor: default;
}

:deep(.el-table .cell) {
  line-height: 1.45;
}

.datapoint-list--management :deep(.el-segmented) {
  --el-segmented-bg-color: rgba(0, 0, 0, 0.04);
  --el-segmented-item-selected-bg-color: #ffffff;
  --el-segmented-item-selected-color: var(--dc-primary);
  --el-segmented-item-hover-bg-color: rgba(255, 255, 255, 0.5);
  border: 0;
}

.datapoint-list--management :deep(.el-table) {
  --el-table-header-bg-color: #f4f6f9;
  --el-table-row-hover-bg-color: rgba(15, 23, 42, 0.04);
  --el-table-border-color: #e5e7eb;
  --el-table-fixed-left-column: 4px 0 10px rgba(15, 23, 42, 0.04);
  --el-table-fixed-right-column: -4px 0 10px rgba(15, 23, 42, 0.04);
  background: transparent;
  color: #475569;
  font-size: 14px;
}

.datapoint-list--management :deep(.el-table th.el-table__cell) {
  color: var(--dc-text-secondary);
  font-weight: 700;
}

.datapoint-list--management :deep(.el-table__header th) {
  background: #f4f6f9;
  color: #1f2937;
  font-weight: 700;
}

.datapoint-list--management :deep(.el-table th.el-table__cell .cell) {
  height: 52px;
  align-items: center;
  padding: 0 12px;
  font-size: 14px;
}

.datapoint-list--management :deep(.el-table__body .el-table__cell) {
  height: 72px;
  padding: 18px 0;
}

.datapoint-list--management :deep(.el-table__body .cell) {
  overflow: hidden;
  padding: 0 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-list--management :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.datapoint-list--management :deep(.el-table__header-wrapper) {
  border-radius: 8px 8px 0 0;
  overflow: hidden;
}

.datapoint-list--management :deep(.el-table__body td) {
  color: #475569;
}

.datapoint-list--management :deep(.el-table__fixed),
.datapoint-list--management :deep(.el-table__fixed-right),
.datapoint-list--management :deep(.el-table__fixed-left),
.datapoint-list--management :deep(.el-table-fixed-column--left),
.datapoint-list--management :deep(.el-table-fixed-column--right) {
  background: var(--dc-surface-raised);
}

.datapoint-list--management :deep(.el-table__body tr:hover > td),
.datapoint-list--management :deep(.el-table__body tr:hover > td.el-table-fixed-column--left),
.datapoint-list--management :deep(.el-table__body tr:hover > td.el-table-fixed-column--right) {
  background: rgba(15, 23, 42, 0.04);
}

.datapoint-list--management :deep(.el-table__header th.el-table-fixed-column--left),
.datapoint-list--management :deep(.el-table__header th.el-table-fixed-column--right) {
  background: #f4f6f9;
}

.datapoint-list--management :deep(.el-table__header th.el-table-fixed-column--right .cell),
.datapoint-list--management :deep(.el-table__body td.el-table-fixed-column--right .cell) {
  display: flex;
  justify-content: center;
  padding-right: 12px;
  padding-left: 12px;
}

.datapoint-list--management .datapoint-list__name-cell,
.datapoint-list--management .datapoint-list__path-cell,
.datapoint-list--management .datapoint-list__tags-cell,
.datapoint-list--management .datapoint-list__row-actions {
  flex-wrap: nowrap;
  overflow: hidden;
  white-space: nowrap;
}

.datapoint-list--management .datapoint-list__tags-cell {
  min-height: 24px;
}

.datapoint-list--management .datapoint-list__row-actions {
  width: 100%;
  justify-content: center;
}

.datapoint-list--management .datapoint-list__time {
  display: inline-block;
  min-width: 148px;
  white-space: nowrap;
}

.datapoint-list--management .datapoint-list__tag {
  max-width: 84px;
  --el-tag-bg-color: #f3f4f6;
  --el-tag-border-color: #e5e7eb;
  --el-tag-text-color: #64748b;
  border-radius: 6px;
  font-weight: 500;
}

.datapoint-list--management .datapoint-list__table-action {
  width: 28px;
  height: 28px;
  border-radius: 999px;
  color: #94a3b8;
}

.datapoint-list--management .datapoint-list__table-action svg {
  width: 16px;
  height: 16px;
}

@media (max-width: 900px) {
  .datapoint-list__toolbar-left,
  .datapoint-list__toolbar-right {
    width: 100%;
  }

  .datapoint-list__search,
  .datapoint-list__select,
  .datapoint-tag-filter {
    flex: 1 1 160px;
    width: auto;
  }

  .datapoint-tag-filter__trigger {
    width: 100%;
  }

  .datapoint-list__pagination-bar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
