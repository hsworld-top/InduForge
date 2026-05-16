<template>
  <div class="datapoint-list" :class="`datapoint-list--${mode}`">
    <!-- 工具栏 -->
    <div v-if="showToolbar" class="datapoint-list__toolbar">
      <div class="datapoint-list__toolbar-left">
        <!-- 搜索：名称 / 路径，300ms 防抖 -->
        <el-input
          v-model="searchText"
          size="small"
          class="datapoint-list__search"
          clearable
          :prefix-icon="Search"
          placeholder="搜索名称、路径"
        />

        <!-- 状态筛选 pill -->
        <el-popover placement="bottom-start" :width="150" trigger="click">
          <template #reference>
            <PillButton :active="statusFilter !== ''">
              <template #icon><Filter /></template>
              {{ currentStatusLabel }}
            </PillButton>
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

        <!-- 来源类型筛选 pill（动态来自当前列表） -->
        <el-popover placement="bottom-start" :width="180" trigger="click">
          <template #reference>
            <PillButton :active="sourceFilter !== ''">
              <template #icon><Connection /></template>
              {{ currentSourceLabel }}
            </PillButton>
          </template>
          <div class="datapoint-list__popover-menu">
            <button
              type="button"
              class="datapoint-list__popover-item"
              :class="{ 'is-active': sourceFilter === '' }"
              @click="sourceFilter = ''"
            >
              全部来源
            </button>
            <button
              v-for="option in dynamicSourceOptions"
              :key="option.value"
              type="button"
              class="datapoint-list__popover-item"
              :class="{ 'is-active': sourceFilter === option.value }"
              @click="sourceFilter = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </el-popover>

        <!-- 标签筛选 popover（保留原有实现） -->
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

        <!-- 排序字段 pill -->
        <el-popover placement="bottom-start" :width="150" trigger="click">
          <template #reference>
            <PillButton>
              {{ currentSortFieldLabel }}
            </PillButton>
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

        <!-- 排序方向 pill -->
        <PillButton @click="toggleSortOrder">
          <template #icon>
            <SortDown v-if="sortOrder === 'desc'" />
            <SortUp v-else />
          </template>
          {{ currentSortOrderLabel }}
        </PillButton>
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
        <el-tooltip content="批量清理失效数据点" placement="bottom">
          <button
            type="button"
            class="datapoint-list__icon-button is-danger"
            :class="{ 'is-disabled': selectedInvalidRows.length === 0 }"
            :disabled="selectedInvalidRows.length === 0"
            aria-label="批量清理失效数据点"
            @click="handleBatchDelete"
          >
            <Delete />
          </button>
        </el-tooltip>
      </div>
    </div>

    <!-- 主面板 -->
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
          <!-- 选择列 -->
          <el-table-column type="selection" width="56" fixed="left" align="center" />

          <!-- 名称列 -->
          <el-table-column label="名称" width="220" fixed="left">
            <template #default="{ row }">
              <div class="datapoint-list__name-cell">
                <el-tooltip :content="row.name || '-'" placement="top" :show-after="400">
                  <span class="datapoint-list__name">
                    {{ row.name || "-" }}
                  </span>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>

          <!-- 路径列 -->
          <el-table-column label="路径" min-width="280">
            <template #default="{ row }">
              <div class="datapoint-list__path-cell">
                <el-tooltip :content="row.path || '-'" placement="top" :show-after="300">
                  <span class="datapoint-list__path">{{ row.path || "-" }}</span>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>

          <!-- 来源列：图标 + 文字 -->
          <el-table-column label="来源" width="130">
            <template #default="{ row }">
              <div class="datapoint-list__source-cell">
                <component
                  :is="getSourceIcon(row.sourceType)"
                  class="datapoint-list__source-icon"
                  aria-hidden="true"
                />
                <span class="datapoint-list__source-text">
                  {{ formatSourceType(row.sourceType) }}
                </span>
              </div>
            </template>
          </el-table-column>

          <!-- 类型列 -->
          <el-table-column label="类型" width="100">
            <template #default="{ row }">
              <span class="datapoint-list__mono">{{ row.dataType || "-" }}</span>
            </template>
          </el-table-column>

          <!-- 状态列：v2 StatusBadge -->
          <el-table-column label="状态" width="96" align="center" header-align="center">
            <template #default="{ row }">
              <StatusBadge
                :tone="resolveStatusTone(row.status)"
                :text="formatStatus(row.status)"
              />
            </template>
          </el-table-column>

          <!-- 引用列（D2 完整实现，D1 占位显示 `-`） -->
          <el-table-column label="引用" width="96" align="center" header-align="center">
            <template #default="{ row }">
              <span class="datapoint-list__ref-count">
                {{ resolveRefCount(row) }}
              </span>
            </template>
          </el-table-column>

          <!-- 标签列 -->
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

          <!-- 更新时间列 -->
          <el-table-column label="更新时间" width="170">
            <template #default="{ row }">
              <span class="datapoint-list__time">
                {{ formatTime(getUpdatedAt(row)) }}
              </span>
            </template>
          </el-table-column>

          <!-- 操作列（4 个按钮） -->
          <el-table-column label="操作" width="200" fixed="right" align="center" header-align="center">
            <template #default="{ row }">
              <div class="datapoint-list__row-actions">
                <!-- 查看详情 -->
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
                <!-- 跳转来源（D1 先做按钮，D2 完整实现 LinkChip 跳转） -->
                <el-tooltip content="跳转来源" placement="top">
                  <button
                    type="button"
                    class="datapoint-list__table-action"
                    aria-label="跳转来源"
                    @click.stop="handleJumpToSource(row)"
                  >
                    <Connection />
                  </button>
                </el-tooltip>
                <!-- 复制路径 -->
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
                <!-- 清理失效（仅 invalid 行可见） -->
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

      <!-- 批量操作条（选中 ≥ 1 时浮现，位于分页栏上方） -->
      <BulkActionBar
        :selected-count="selectedRows.length"
        @clear="selectedRows = []"
      >
        <button
          type="button"
          class="datapoint-list__bulk-action-btn"
          @click="openBatchTagDialog"
        >
          打标签
        </button>
        <button
          type="button"
          class="datapoint-list__bulk-action-btn is-danger"
          :disabled="selectedInvalidRows.length === 0"
          @click="handleBatchDelete"
        >
          清理失效
        </button>
      </BulkActionBar>

      <!-- 分页栏 -->
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

    <!-- 标签 dialog（保留 D2 前实现，不动） -->
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

    <!-- 写权限 dialog（保留 D2 前实现，不动） -->
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

    <!-- 详情抽屉（保留 D2 前实现，不动） -->
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
  Bell,
  Close,
  CollectionTag,
  Connection,
  CopyDocument,
  Cpu,
  DataLine,
  Delete,
  Filter,
  Refresh,
  Search,
  SortDown,
  SortUp,
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
import PillButton from "@/components/shared/PillButton.vue";
import StatusBadge from "@/components/shared/StatusBadge.vue";
import BulkActionBar from "@/components/shared/BulkActionBar.vue";

type RuntimeGrant = ReturnType<typeof normalizeRuntimeGrantPayload>;
type DataPointListMode = "embedded" | "management";
type SortField = "updatedAt" | "name" | "path";
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
  /** 引用数（后端暂未返回，D2 补全） */
  refCount?: number;
}

const props = withDefaults(
  defineProps<{
    projectId: string;
    sourceType?: string;
    status?: string;
    search?: string;
    showToolbar?: boolean;
    mode?: DataPointListMode;
    /** Workspace 通过 URL query 注入的搜索词 */
    filterQ?: string;
    /** Workspace 通过 URL query 注入的状态筛选 */
    filterStatus?: string;
    /** Workspace 通过 URL query 注入的来源类型筛选 */
    filterSource?: string;
    /** Workspace 通过 URL query 注入的标签筛选 */
    filterTags?: string[];
    /** Workspace 通过 URL query 注入的排序字段 */
    sortField?: string;
    /** Workspace 通过 URL query 注入的排序方向 */
    sortOrder?: string;
    /** Workspace 通过 URL query 注入的页码 */
    page?: number;
  }>(),
  {
    sourceType: undefined,
    status: undefined,
    search: undefined,
    showToolbar: true,
    mode: "embedded",
    filterQ: "",
    filterStatus: "",
    filterSource: "",
    filterTags: () => [],
    sortField: "updatedAt",
    sortOrder: "desc",
    page: 1,
  },
);

const emit = defineEmits<{
  select: [row: DataPointRow];
  /** 筛选 / 排序 / 搜索变化时通知 Workspace 更新 URL */
  "update:filter": [params: Record<string, unknown>];
}>();

// ── 本地状态 ──────────────────────────────────────────────────────────────

const loading = ref(false);
const datapoints = ref<DataPointRow[]>([]);

/* 搜索文本 */
const searchText = ref(props.filterQ || "");
/* 状态筛选（'' = 全部） */
const statusFilter = ref(props.filterStatus || "");
/* 来源类型筛选 */
const sourceFilter = ref(props.filterSource || "");

/* 标签筛选 */
const tagFilterValues = ref<string[]>(props.filterTags || []);
const tagFilterVisible = ref(false);
const tagFilterKeyword = ref("");
const groupByTags = ref(false);
const tagFilterRootRef = ref<HTMLElement | null>(null);

/* 排序 */
const sortFieldValue = ref<SortField>(
  (["updatedAt", "name", "path"].includes(props.sortField || "") ? props.sortField : "updatedAt") as SortField,
);
const sortOrderValue = ref<SortOrder>(
  (props.sortOrder === "asc" ? "asc" : "desc") as SortOrder,
);

/* 分页 */
const pagination = ref({
  page: props.page || 1,
  pageSize: 50,
});
const debounceTimer = ref<number | null>(null);

/* 选中行 */
const selectedRows = ref<DataPointRow[]>([]);

/* 写权限 dialog 状态（保留 D2 前实现） */
const permissionDialogVisible = ref(false);
const permissionSaving = ref(false);
const currentPermissionDatapoint = ref<DataPointRow | null>(null);
const permissionForm = ref<RuntimeGrant>(normalizeRuntimeGrantPayload());
const allowRolesInput = ref("");
const denyRolesInput = ref("");

/* 标签 dialog 状态（保留 D2 前实现） */
const tagDialogVisible = ref(false);
const tagSaving = ref(false);
const tagDraft = ref<string[]>([]);
const tagCreateInput = ref("");
const tagEditMode = ref<"single" | "batch">("single");
const currentTagDatapoint = ref<DataPointRow | null>(null);

/* 详情抽屉状态（保留 D2 前实现） */
const detailDrawerVisible = ref(false);
const detailDatapoint = ref<DataPointRow | null>(null);

// ── 静态配置 ─────────────────────────────────────────────────────────────

/** 状态选项（v2 规格：全部/活跃/失效/错误） */
const statusOptions = [
  { label: "全部", value: "" },
  { label: "活跃", value: "active" },
  { label: "失效", value: "invalid" },
  { label: "错误", value: "error" },
];

/** 排序字段选项（v2 规格：更新时间/名称/路径） */
const sortFieldOptions: Array<{ label: string; value: SortField }> = [
  { label: "更新时间", value: "updatedAt" },
  { label: "名称", value: "name" },
  { label: "路径", value: "path" },
];

const pageSizeOptions = [20, 50, 100];

/** 来源类型标签映射 */
const sourceTypeLabels: Record<string, string> = {
  "db.query": "数据库查询",
  "mqtt.tag": "MQTT 变量",
  "mqtt.subscription": "MQTT 订阅",
  "calc.output": "计算输出",
  "static.var": "静态变量",
};

// ── 来源图标映射 ──────────────────────────────────────────────────────────

function getSourceIcon(sourceType?: string) {
  if (!sourceType) return DataLine;
  if (sourceType.startsWith("mqtt")) return Connection;
  if (sourceType.startsWith("db")) return DataLine;
  if (sourceType.startsWith("calc")) return Cpu;
  if (sourceType.startsWith("alarm")) return Bell;
  return DataLine;
}

// ── 动态来源类型（从当前列表提取，去重） ─────────────────────────────────

const dynamicSourceOptions = computed(() => {
  const seen = new Set<string>();
  datapoints.value.forEach((item) => {
    if (item.sourceType) seen.add(item.sourceType);
  });
  return Array.from(seen).sort().map((v) => ({
    value: v,
    label: sourceTypeLabels[v] || v,
  }));
});

// ── 标签选项（从当前数据点列表提取） ─────────────────────────────────────

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
  if (!keyword) return tagOptions.value;
  return tagOptions.value.filter((tag) =>
    tag.name.toLowerCase().includes(keyword),
  );
});

const selectedTagSet = computed(() => new Set(tagFilterValues.value));

// ── 计算属性：label ───────────────────────────────────────────────────────

const isManagementMode = computed(() => props.mode === "management");

const currentStatusLabel = computed(
  () =>
    statusOptions.find((o) => o.value === statusFilter.value)?.label || "全部",
);

const currentSourceLabel = computed(() => {
  if (!sourceFilter.value) return "来源";
  return sourceTypeLabels[sourceFilter.value] || sourceFilter.value;
});

const currentSortFieldLabel = computed(
  () =>
    sortFieldOptions.find((o) => o.value === sortFieldValue.value)?.label || "更新时间",
);

const currentSortOrderLabel = computed(() =>
  sortOrderValue.value === "desc" ? "降序" : "升序",
);

const tagFilterLabel = computed(() =>
  tagFilterValues.value.length > 0
    ? `标签 (${tagFilterValues.value.length})`
    : "标签",
);

const tagDialogTitle = computed(() =>
  tagEditMode.value === "batch"
    ? `标签管理（${selectedRows.value.length} 项）`
    : `标签管理：${currentTagDatapoint.value?.name || "-"}`,
);

const selectedInvalidRows = computed(() =>
  selectedRows.value.filter((item) => item.status === "invalid"),
);

// ── 数据过滤 / 排序 / 分页（前端层，补充标签 / 分组排序） ────────────────

const visibleDataPoints = computed(() => {
  let list = datapoints.value;

  /* 标签过滤（前端层，后端只按搜索/状态/来源拉数据） */
  if (tagFilterValues.value.length > 0) {
    list = list.filter((item) =>
      tagFilterValues.value.every((tag) => normalizeTags(item.tags).includes(tag)),
    );
  }

  /* 来源筛选（前端二次过滤；后端已支持时可移除） */
  if (sourceFilter.value) {
    list = list.filter((item) => item.sourceType === sourceFilter.value);
  }

  /* 排序 */
  list = [...list].sort((a, b) => {
    let delta = 0;
    if (sortFieldValue.value === "updatedAt") {
      delta =
        dayjs(getUpdatedAt(a)).valueOf() - dayjs(getUpdatedAt(b)).valueOf();
    } else {
      delta = String(a[sortFieldValue.value] || "").localeCompare(
        String(b[sortFieldValue.value] || ""),
        "zh-Hans-CN",
      );
    }
    return sortOrderValue.value === "desc" ? -delta : delta;
  });

  /* 按标签分组 */
  if (groupByTags.value) {
    list = list.sort((a, b) => {
      const ag = normalizeTags(a.tags)[0] || "未设置";
      const bg = normalizeTags(b.tags)[0] || "未设置";
      const gd = ag.localeCompare(bg, "zh-Hans-CN");
      if (gd !== 0) return gd;
      return String(a.name || a.path || "").localeCompare(
        String(b.name || b.path || ""),
        "zh-Hans-CN",
      );
    });
  }

  return list;
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

// ── 数据加载 ──────────────────────────────────────────────────────────────

const buildQueryParams = () => {
  const params: Record<string, unknown> = {
    page: 1,
    pageSize: 500,
  };
  const q = props.search ?? searchText.value;
  const st = props.status ?? statusFilter.value;
  const src = props.sourceType ?? sourceFilter.value;
  if (q) params.search = q;
  if (st) params.status = st;
  if (src) params.type = src;
  return params;
};

const loadDataPoints = async () => {
  if (!props.projectId) return;
  loading.value = true;
  try {
    const response = await dataAPI.getDataPoints(props.projectId, buildQueryParams());
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

// ── 分页 ──────────────────────────────────────────────────────────────────

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

// ── 排序 ──────────────────────────────────────────────────────────────────

const toggleSortOrder = () => {
  sortOrderValue.value = sortOrderValue.value === "desc" ? "asc" : "desc";
  pagination.value.page = 1;
};

// ── 标签过滤面板 ──────────────────────────────────────────────────────────

const toggleTagFilterPanel = () => {
  tagFilterVisible.value = !tagFilterVisible.value;
  if (tagFilterVisible.value) tagFilterKeyword.value = "";
};

const toggleTagFilter = (tag: string) => {
  const next = new Set(tagFilterValues.value);
  if (next.has(tag)) next.delete(tag);
  else next.add(tag);
  tagFilterValues.value = [...next];
  pagination.value.page = 1;
};

const handleDocumentClick = (event: MouseEvent) => {
  if (!tagFilterVisible.value) return;
  const target = event.target as Node | null;
  if (target && tagFilterRootRef.value?.contains(target)) return;
  tagFilterVisible.value = false;
};

// ── 行操作 ────────────────────────────────────────────────────────────────

const openDetailDrawer = (row: DataPointRow) => {
  detailDatapoint.value = row;
  detailDrawerVisible.value = true;
  emit("select", row);
};

const handleRowClick = (row: DataPointRow) => {
  if (!isManagementMode.value) emit("select", row);
};

/** 跳转来源（D1 占位，D2 实现完整 LinkChip 跳转逻辑） */
const handleJumpToSource = (row: DataPointRow) => {
  if (!row.sourceType || !row.sourceId) {
    ElMessage.info("该数据点暂无来源信息");
    return;
  }
  ElMessage.info(`跳转来源：${formatSourceType(row.sourceType)}（D2 完整实现）`);
};

const handleDelete = async (datapoint: DataPointRow) => {
  try {
    await ElMessageBox.confirm(
      `确认清理失效数据点「${datapoint.name || datapoint.path || "-"}」？`,
      "清理失效数据点",
      { confirmButtonText: "清理", cancelButtonText: "取消", type: "warning" },
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
      { confirmButtonText: "批量清理", cancelButtonText: "取消", type: "warning" },
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

// ── 标签 dialog（保留 D2 前实现） ─────────────────────────────────────────

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
      { confirmButtonText: "删除", cancelButtonText: "取消", type: "warning" },
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
        { tags: normalizedTags },
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

// ── 写权限 dialog（保留 D2 前实现） ──────────────────────────────────────

const parseRoleInput = (value: string) => {
  return String(value || "")
    .split(/[\n,，]/)
    .map((item) => item.trim())
    .filter(Boolean);
};

const getWriteRuntimeGrant = (row: DataPointRow) => {
  const rp = row?.runtimePermissions as { write?: unknown } | undefined;
  const rpg = row?.runtimePermissionGrants as { write?: unknown } | undefined;
  return normalizeRuntimeGrantPayload(
    rp?.write || rpg?.write || row?.writePermission || {},
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
  if (!props.projectId || !currentPermissionDatapoint.value?.id) return;
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

// ── 工具函数 ──────────────────────────────────────────────────────────────

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

const getUpdatedAt = (row: DataPointRow) =>
  row.updatedAt || row.updated_at || row.createdAt || row.created_at || "";

/** 状态 badge tone 映射（v2 规格） */
function resolveStatusTone(
  status?: string,
): "success" | "danger" | "warning" | "muted" {
  switch (status) {
    case "active":
      return "success";
    case "invalid":
      return "danger";
    case "error":
      return "warning";
    default:
      return "muted";
  }
}

const formatStatus = (status?: string) => {
  switch (status) {
    case "active":
      return "活跃";
    case "invalid":
      return "失效";
    case "error":
      return "错误";
    case "inactive":
      return "停用";
    default:
      return "未知";
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
  if (typeof value === "number" || typeof value === "boolean") return String(value);
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
  if (parts.length >= 2) return parts[1];
  return parts[0] || "未分组";
};

/** 引用数（D2 完整实现；D1 全部显示 `-`） */
const resolveRefCount = (row: DataPointRow) => {
  const count = (row as DataPointRow & { refCount?: number }).refCount;
  if (typeof count === "number" && count > 0) return count;
  return "-";
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
    if (normalized && !result.includes(normalized)) result.push(normalized);
  });
  return result;
};

const mergeTags = (left: string[], right: string[]) =>
  Array.from(new Set([...left, ...right].map((item) => item.trim()))).filter(Boolean);

const getVisibleTags = (row: DataPointRow) => normalizeTags(row.tags).slice(0, 2);
const getHiddenTags = (row: DataPointRow) => normalizeTags(row.tags).slice(2);
const getHiddenTagCount = (row: DataPointRow) => getHiddenTags(row).length;

// ── URL 同步：向 Workspace 推送变化 ──────────────────────────────────────

function emitFilterUpdate() {
  emit("update:filter", {
    q: searchText.value,
    status: statusFilter.value,
    source: sourceFilter.value,
    tags: tagFilterValues.value,
    sort: sortFieldValue.value,
    order: sortOrderValue.value,
    page: pagination.value.page,
  });
}

// ── Watch ─────────────────────────────────────────────────────────────────

/* 搜索、状态、来源：300ms 防抖后重新拉数据 + 同步 URL */
watch([searchText, statusFilter, sourceFilter], () => {
  pagination.value.page = 1;
  if (debounceTimer.value) window.clearTimeout(debounceTimer.value);
  debounceTimer.value = window.setTimeout(() => {
    void loadDataPoints();
    emitFilterUpdate();
  }, 300);
});

/* 排序变化：重新排序 + 同步 URL */
watch([sortFieldValue, sortOrderValue], () => {
  pagination.value.page = 1;
  emitFilterUpdate();
});

/* 标签 / 分组：前端层，不触发请求，只同步 URL */
watch([tagFilterValues, groupByTags], () => {
  pagination.value.page = 1;
  emitFilterUpdate();
});

/* 页码变化：同步 URL */
watch(() => pagination.value.page, () => {
  emitFilterUpdate();
});

/* 翻页越界保护 */
watch([totalVisibleCount, totalPages], () => {
  if (totalPages.value === 0) {
    pagination.value.page = 1;
    return;
  }
  if (pagination.value.page > totalPages.value) {
    pagination.value.page = totalPages.value;
  }
});

/* 外部 props 注入（Workspace URL → 列表）：只在首次挂载前已通过默认值写入 */
watch(
  () => [props.sourceType, props.status, props.search],
  () => {
    pagination.value.page = 1;
    if (debounceTimer.value) window.clearTimeout(debounceTimer.value);
    debounceTimer.value = window.setTimeout(() => {
      void loadDataPoints();
    }, 300);
  },
);

// ── 生命周期 ──────────────────────────────────────────────────────────────

onMounted(() => {
  document.addEventListener("click", handleDocumentClick);
  void loadDataPoints();
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleDocumentClick);
  if (debounceTimer.value) window.clearTimeout(debounceTimer.value);
});

defineExpose({
  refresh: loadDataPoints,
  openPermissionDialog,
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

/* ── 工具栏 ── */

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

.datapoint-list--management .datapoint-list__search {
  width: 200px;
  flex: 0 0 200px;
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

.datapoint-list__search {
  width: 176px;
  flex: 0 0 176px;
}

.datapoint-list__toolbar-left,
.datapoint-list__toolbar-right {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.datapoint-list__toolbar-left {
  flex: 1 1 auto;
}

.datapoint-list__toolbar-right {
  gap: 8px;
  margin-left: auto;
}

/* ── 菜单 ── */

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

/* ── 标签过滤面板（保留原有实现） ── */

.datapoint-tag-filter {
  position: relative;
  display: inline-flex;
  flex: 0 0 auto;
}

.datapoint-tag-filter__trigger {
  height: 28px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: 12px;
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  transition:
    background-color 0.2s ease,
    color 0.2s ease,
    border-color 0.2s ease;
}

.datapoint-tag-filter__trigger:hover,
.datapoint-tag-filter__trigger.is-active {
  border-color: rgba(29, 78, 216, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.datapoint-tag-filter__trigger svg {
  width: 14px;
  height: 14px;
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

/* ── icon-button ── */

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

/* ── 主面板 ── */

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
  min-width: 1400px;
}

/* ── 单元格 ── */

.datapoint-list__name-cell {
  min-width: 0;
  display: flex;
  align-items: center;
}

.datapoint-list__name {
  min-width: 0;
  overflow: hidden;
  color: #2563eb;
  font-size: 14px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-list__path-cell {
  min-width: 0;
  display: flex;
  align-items: center;
}

.datapoint-list__path {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: #64748b;
  font-family: var(--dc-font-mono);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 来源：图标 + 文字 */
.datapoint-list__source-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.datapoint-list__source-icon {
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
  color: var(--dc-text-muted);
}

.datapoint-list__source-text {
  display: inline-block;
  min-width: 0;
  overflow: hidden;
  color: #64748b;
  font-size: 13px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-list__mono {
  color: #475569;
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.datapoint-list__time {
  color: #475569;
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

/* 引用数 */
.datapoint-list__ref-count {
  color: var(--dc-text-muted);
  font-size: 13px;
}

/* ── 标签单元格 ── */

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

/* ── 行操作 ── */

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
  border-color: transparent;
  background: transparent;
  color: #64748b;
}

.datapoint-list__table-action.is-danger:hover {
  border-color: transparent;
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger);
}

/* ── 批量操作按钮 ── */

.datapoint-list__bulk-action-btn {
  height: 28px;
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
  transition:
    background 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease;
}

.datapoint-list__bulk-action-btn:hover:not(:disabled) {
  border-color: rgba(29, 78, 216, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.datapoint-list__bulk-action-btn.is-danger:hover:not(:disabled) {
  border-color: rgba(220, 38, 38, 0.26);
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger);
}

.datapoint-list__bulk-action-btn:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

/* ── 复制按钮 ── */

.datapoint-list__copy {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex: 0 0 24px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
}

.datapoint-list__copy svg {
  width: 14px;
  height: 14px;
}

.datapoint-list__copy:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

/* ── 分页 ── */

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

/* ── 标签 dialog ── */

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
  display: flex;
  flex-direction: column;
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

/* ── 写权限 dialog ── */

.permission-dialog {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.permission-dialog__hint {
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

/* ── 详情抽屉 ── */

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

/* ── 全局 el-input 样式（非 management 模式） ── */

:deep(.el-input__wrapper),
:deep(.el-select .el-input__wrapper) {
  border-radius: var(--dc-radius-md);
  box-shadow: none;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
}
</style>
