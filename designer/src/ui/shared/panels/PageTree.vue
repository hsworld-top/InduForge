<!--
  PageTree - 页面树
  展示工程页面层级（基础页面、自定义页面），支持搜索、新建、重命名、删除、拖拽排序
-->
<template>
  <div class="page-tree-container">
    <!-- 搜索框 -->
    <div class="page-search">
      <el-input
        v-model="searchText"
        size="small"
        placeholder="搜索页面..."
        clearable
        :prefix-icon="IconEpSearch"
      />
    </div>

    <!-- 页面树 -->
    <div class="page-tree-content">
      <section class="page-section">
        <div class="page-section__header">
          <span class="page-section__title">基础页面</span>
        </div>
        <div v-if="filteredBasicSlots.length" class="page-section__body">
          <div
            v-for="slot in filteredBasicSlots"
            :key="slot.type"
            class="tree-node page-node page-node--basic"
            :class="{
              'is-active': slot.page ? isPageActive(slot.page.id) : false,
              'is-empty': !slot.page,
            }"
            @click="handleBasicSlotClick(slot)"
            @dblclick="handleBasicSlotDoubleClick(slot)"
          >
            <div class="node-indent node-indent--basic" />
            <IconEpDocument class="node-icon page" />
            <span class="node-label">{{ slot.label }}</span>
            <el-dropdown
              trigger="click"
              placement="bottom-end"
              @command="(command) => handleBasicRowAction(command, slot)"
            >
              <el-button class="node-action-btn" text @click.stop>
                <IconEpMoreFilled />
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item v-if="slot.page" command="open"
                    >打开</el-dropdown-item
                  >
                  <el-dropdown-item v-else command="create"
                    >创建</el-dropdown-item
                  >
                  <el-dropdown-item v-if="slot.page" command="export"
                    >导出页面</el-dropdown-item
                  >
                  <el-dropdown-item
                    v-if="slot.page && slot.type !== 'home'"
                    command="delete"
                    divided
                  >
                    删除
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
        <div v-else class="page-section__empty">暂无基础页面</div>
      </section>

      <section
        class="page-section"
        @dragover.prevent="handleContainerDragOver($event, null)"
        @drop.prevent="handleContainerDrop(null)"
      >
        <div class="page-section__header">
          <span class="page-section__title">普通页面</span>
        </div>
        <div v-if="filteredBusinessRootItems.length" class="page-section__body">
          <template v-for="item in filteredBusinessRootItems" :key="item.id">
            <div
              v-if="item.type === 'folder'"
              class="tree-node folder-node"
              :class="{
                'is-active': selectedNode?.id === item.id,
                'is-drop-target':
                  dragOverTarget?.id === item.id &&
                  dragOverTarget?.mode === 'append',
              }"
              draggable="true"
              @dragstart="handleDragStart(item, null)"
              @dragend="handleDragEnd"
              @dragover.prevent="handleFolderDragOver($event, item)"
              @dragleave="handleDragLeave(item.id)"
              @drop.prevent="handleFolderDrop(item)"
              @click="handleFolderClick(item)"
            >
              <div class="node-indent node-indent--folder" />
              <component
                :is="folderStates[item.id] ? IconEpArrowDown : IconEpArrowRight"
                class="node-arrow"
                @click.stop="toggleFolder(item.id)"
              />
              <IconEpFolder class="node-icon folder" />
              <span class="node-label">{{ item.name }}</span>
              <span class="node-count">{{ getChildrenCount(item.id) }}</span>
              <el-dropdown
                trigger="click"
                placement="bottom-end"
                @command="(command) => handleRowAction(command, item)"
              >
                <el-button class="node-action-btn" text @click.stop>
                  <IconEpMoreFilled />
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="rename">重命名</el-dropdown-item>
                    <el-dropdown-item command="createPage"
                      >新建页面</el-dropdown-item
                    >
                    <el-dropdown-item command="delete" divided
                      >删除分组</el-dropdown-item
                    >
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <div
              v-else
              class="tree-node page-node"
              :class="{
                'is-active': isPageActive(item.id),
                'is-drop-target':
                  dragOverTarget?.id === item.id &&
                  dragOverTarget?.mode === 'before',
              }"
              draggable="true"
              @dragstart="handleDragStart(item, null)"
              @dragend="handleDragEnd"
              @dragover.prevent="handleNodeDragOver($event, item, null)"
              @dragleave="handleDragLeave(item.id)"
              @drop.prevent="handleNodeDrop(item, null)"
              @click="handleNodeClick({ id: item.id, type: 'page' })"
              @dblclick="
                handleNodeDoubleClick({
                  id: item.id,
                  type: 'page',
                  label: getPageLabel(item),
                })
              "
            >
              <div class="node-indent" />
              <IconEpDocument class="node-icon page" />
              <span class="node-label">{{ getPageLabel(item) }}</span>
              <el-dropdown
                trigger="click"
                placement="bottom-end"
                @command="(command) => handleRowAction(command, item)"
              >
                <el-button class="node-action-btn" text @click.stop>
                  <IconEpMoreFilled />
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="open">打开</el-dropdown-item>
                    <el-dropdown-item command="rename">重命名</el-dropdown-item>
                    <el-dropdown-item command="export"
                      >导出页面</el-dropdown-item
                    >
                    <el-dropdown-item
                      v-if="item.parentId"
                      command="moveToRoot"
                      divided
                    >
                      移到根目录
                    </el-dropdown-item>
                    <el-dropdown-item
                      v-for="target in getMoveTargets(item)"
                      :key="`root-${item.id}-${target.id ?? 'root'}`"
                      :command="buildMoveCommand(target.id)"
                    >
                      {{ target.label }}
                    </el-dropdown-item>
                    <el-dropdown-item command="delete" divided
                      >删除</el-dropdown-item
                    >
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>

            <div
              v-if="item.type === 'folder' && folderStates[item.id]"
              class="tree-children"
            >
              <div
                v-for="page in getBusinessChildren(item.id)"
                :key="page.id"
                class="tree-node page-node level-1"
                :class="{
                  'is-active': isPageActive(page.id),
                  'is-drop-target':
                    dragOverTarget?.id === page.id &&
                    dragOverTarget?.mode === 'before',
                }"
                draggable="true"
                @dragstart="handleDragStart(page, item.id)"
                @dragend="handleDragEnd"
                @dragover.prevent="handleNodeDragOver($event, page, item.id)"
                @dragleave="handleDragLeave(page.id)"
                @drop.prevent="handleNodeDrop(page, item.id)"
                @click="handleNodeClick({ id: page.id, type: 'page' })"
                @dblclick="
                  handleNodeDoubleClick({
                    id: page.id,
                    type: 'page',
                    label: getPageLabel(page),
                  })
                "
              >
                <div class="node-indent">
                  <span class="tree-line horizontal" />
                </div>
                <IconEpDocument class="node-icon page" />
                <span class="node-label">{{ getPageLabel(page) }}</span>
                <el-dropdown
                  trigger="click"
                  placement="bottom-end"
                  @command="(command) => handleRowAction(command, page)"
                >
                  <el-button class="node-action-btn" text @click.stop>
                    <IconEpMoreFilled />
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="open">打开</el-dropdown-item>
                      <el-dropdown-item command="rename"
                        >重命名</el-dropdown-item
                      >
                      <el-dropdown-item command="export"
                        >导出页面</el-dropdown-item
                      >
                      <el-dropdown-item
                        v-if="page.parentId"
                        command="moveToRoot"
                        divided
                      >
                        移到根目录
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-for="target in getMoveTargets(page)"
                        :key="`${page.id}-${target.id ?? 'root'}`"
                        :command="buildMoveCommand(target.id)"
                      >
                        {{ target.label }}
                      </el-dropdown-item>
                      <el-dropdown-item command="delete" divided
                        >删除</el-dropdown-item
                      >
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
              <div
                v-if="
                  !getBusinessChildren(item.id).length && !searchText.trim()
                "
                class="folder-dropzone"
                @dragover.prevent="handleFolderDragOver($event, item)"
                @dragleave="handleDragLeave(item.id)"
                @drop.prevent="handleFolderDrop(item)"
              >
                拖拽页面到此分组
              </div>
            </div>
          </template>
        </div>
        <div v-else class="page-section__empty">暂无普通页面</div>
      </section>

      <div
        v-if="
          filteredBasicSlots.length === 0 &&
          filteredBusinessRootItems.length === 0
        "
        class="empty-state"
      >
        <IconEpDocument class="empty-icon" />
        <p class="empty-text">暂无页面</p>
        <p class="empty-hint">点击上方按钮新建页面</p>
      </div>
    </div>
  </div>
  <!-- 新建页面/分组弹窗 -->
  <el-dialog
    v-model="createDialogVisible"
    :title="createDialogTitle"
    width="420px"
    :close-on-click-modal="false"
    class="create-page-dialog"
  >
    <div class="create-type-selector">
      <div
        v-for="option in createTypeOptions"
        :key="option.value"
        class="type-option"
        :class="{ 'is-active': createForm.type === option.value }"
        @click="createForm.type = option.value"
      >
        <component :is="option.icon" class="type-icon" />
        <div class="type-info">
          <div class="type-name">{{ option.label }}</div>
          <div class="type-desc">{{ option.desc }}</div>
        </div>
      </div>
    </div>

    <el-form
      ref="createFormRef"
      :model="createForm"
      :rules="createFormRules"
      label-position="top"
      class="create-form"
    >
      <el-form-item label="名称" prop="name">
        <el-input
          v-model="createForm.name"
          :disabled="isFixedBasicCreateType(createForm.type)"
          placeholder="请输入名称"
          maxlength="50"
          show-word-limit
        />
      </el-form-item>

      <el-form-item v-if="createForm.type === 'page'" label="所属分组">
        <el-select
          v-model="createForm.parentId"
          clearable
          placeholder="选择分组（可选）"
        >
          <el-option label="根目录" :value="null">
            <div class="flex items-center gap-2">
              <IconEpFolderOpened class="text-gray-400" />
              <span>根目录</span>
            </div>
          </el-option>
          <el-option
            v-for="folder in folderOptions"
            :key="folder.id"
            :label="folder.name"
            :value="folder.id"
          >
            <div class="flex items-center gap-2">
              <IconEpFolder class="text-yellow-500" />
              <span>{{ folder.name }}</span>
            </div>
          </el-option>
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="creating"
          @click="handleCreateConfirm"
        >
          <IconEpPlus class="mr-1" />
          创建
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, h, inject, ref, watch, watchEffect } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { ElMessage, ElMessageBox } from "element-plus";
import IconEpFolder from "~icons/ep/folder";
import IconEpFolderOpened from "~icons/ep/folder-opened";
import IconEpDocument from "~icons/ep/document";
import IconEpSearch from "~icons/ep/search";
import IconEpArrowDown from "~icons/ep/arrow-down";
import IconEpArrowRight from "~icons/ep/arrow-right";
import IconEpLock from "~icons/ep/lock";
import IconEpPlus from "~icons/ep/plus";
import IconEpEdit from "~icons/ep/edit";
import IconEpDelete from "~icons/ep/delete";
import IconEpMoreFilled from "~icons/ep/more-filled";

// 注入打开标签页的方法
const openPageTab = inject("openPageTab", null);

const editorStore = useEditorStore();
const { pages, currentPageId, entryConfig, canUndo, projectId } =
  storeToRefs(editorStore);
const selectedNode = ref(null);
const createDialogVisible = ref(false);
const createFormRef = ref(null);
const createForm = ref({ type: "page", name: "", parentId: null });
const creating = ref(false);
const skipSwitchPrompt = ref(false);
const pageSwitchPromptKey = "designer.pageSwitchPrompt.disabled";
const PAGE_TREE_ORDER_PREFIX = "designer.pageTreeOrder";
const ROOT_CONTAINER_KEY = "__root__";
const pageOrderMap = ref({});
const dragState = ref(null);
const dragOverTarget = ref(null);
let dragOverFrameId = 0;
let pendingDragOverTarget = null;

// 新建弹窗类型选项
const createTypeOptions = [
  {
    value: "page",
    label: "业务页面",
    desc: "创建一个普通业务页面",
    icon: IconEpDocument,
  },
  {
    value: "folder",
    label: "分组",
    desc: "创建一个分组来组织页面",
    icon: IconEpFolder,
  },
];

// 新建弹窗标题
const createDialogTitle = computed(() => {
  const option = createTypeOptions.find(
    (o) => o.value === createForm.value.type,
  );
  return `新建${option?.label || "页面"}`;
});

// 表单验证规则
const createFormRules = {
  name: [
    { required: true, message: "请输入名称", trigger: "blur" },
    { min: 1, max: 50, message: "名称长度在 1 到 50 个字符", trigger: "blur" },
    {
      validator: (_rule, value, callback) => {
        const result = validatePageName(value);
        if (!result.valid) {
          callback(new Error(result.message));
          return;
        }
        callback();
      },
      trigger: "blur",
    },
  ],
};

const BASIC_PAGE_META = {
  home: { label: "首页", path: "/" },
  login: { label: "登录页", path: "/login" },
  logout: { label: "登出页", path: "/logout" },
};

/**
 * 获取基础页面固定配置
 * @param {"home" | "login" | "logout"} type - 基础页面类型
 * @returns {{ label: string, path: string }}
 */
const getBasicPageMeta = (type) =>
  BASIC_PAGE_META[type] || { label: "", path: "/" };

/**
 * 判断是否为固定基础页创建类型
 * @param {string} type - 创建类型
 * @returns {boolean}
 */
const isFixedBasicCreateType = (type) =>
  ["home", "login", "logout"].includes(type);

/**
 * 获取基础页面类型
 * @param {import('@/editor-core').PageNode | undefined | null} page - 页面
 * @returns {"home" | "login" | "logout" | null}
 */
const getFixedSystemType = (page) => {
  if (!page) return null;
  if (entryConfig.value?.homePageId === page.id) return "home";
  if (entryConfig.value?.loginPageId === page.id || page.path === "/login")
    return "login";
  if (entryConfig.value?.logoutPageId === page.id || page.path === "/logout")
    return "logout";
  return null;
};

/**
 * 校验页面名称是否合法
 * @param {string} value - 页面名称
 * @returns {{ valid: boolean, message: string }}
 */
const validatePageName = (value) => {
  const name = String(value || "").trim();
  if (!name) {
    return { valid: false, message: "名称不能为空" };
  }
  if (/^[.]+$/.test(name)) {
    return { valid: false, message: "页面名称不能仅包含点号" };
  }
  if (/[/?#\\%]/.test(name)) {
    return { valid: false, message: "页面名称不能包含 / ? # % \\" };
  }
  if (/[\u0000-\u001f\u007f]/.test(name)) {
    return { valid: false, message: "页面名称不能包含控制字符" };
  }
  return { valid: true, message: "" };
};

/**
 * 在分组中创建页面
 * @param {string} folderId - 分组ID
 */
const handleCreateInFolder = (folderId) => {
  createForm.value = { type: "page", name: "", parentId: folderId };
  createDialogVisible.value = true;
};

// 搜索和展开状态
const searchText = ref("");
const folderStates = ref({});

/**
 * 切换分组展开状态
 * @param {string} folderId - 分组ID
 */
const toggleFolder = (folderId) => {
  folderStates.value[folderId] = !folderStates.value[folderId];
};

/**
 * 点击分组
 * @param {Object} folder - 分组
 */
const handleFolderClick = (folder) => {
  selectedNode.value = { id: folder.id, type: "folder", label: folder.name };
  toggleFolder(folder.id);
};

const getProjectOrderStorageKey = () =>
  `${PAGE_TREE_ORDER_PREFIX}:${projectId.value || "default"}`;

const loadPageOrderMap = () => {
  try {
    const raw = localStorage.getItem(getProjectOrderStorageKey());
    pageOrderMap.value = raw ? JSON.parse(raw) || {} : {};
  } catch (error) {
    pageOrderMap.value = {};
  }
};

const persistPageOrderMap = () => {
  try {
    localStorage.setItem(
      getProjectOrderStorageKey(),
      JSON.stringify(pageOrderMap.value || {}),
    );
  } catch (error) {
    // ignore
  }
};

const getContainerOrderKey = (parentId = null) =>
  parentId || ROOT_CONTAINER_KEY;

const updateContainerOrder = (containerKey, orderedIds) => {
  const nextMap = { ...(pageOrderMap.value || {}) };
  nextMap[containerKey] = Array.from(
    new Set((orderedIds || []).filter(Boolean)),
  );
  pageOrderMap.value = nextMap;
  persistPageOrderMap();
};

const sortItemsByStoredOrder = (items, containerKey) => {
  const source = Array.isArray(items) ? [...items] : [];
  const stored = Array.isArray(pageOrderMap.value?.[containerKey])
    ? pageOrderMap.value[containerKey]
    : [];
  const orderIndexMap = new Map(stored.map((id, index) => [id, index]));
  return source.sort((a, b) => {
    const aIndex = orderIndexMap.has(a.id)
      ? orderIndexMap.get(a.id)
      : Number.MAX_SAFE_INTEGER;
    const bIndex = orderIndexMap.has(b.id)
      ? orderIndexMap.get(b.id)
      : Number.MAX_SAFE_INTEGER;
    if (aIndex !== bIndex) {
      return aIndex - bIndex;
    }
    return (a.name || "").localeCompare(b.name || "", "zh-CN");
  });
};

const ensureContainerOrder = (containerKey, items) => {
  const nextIds = (items || []).map((item) => item.id);
  const currentIds = Array.isArray(pageOrderMap.value?.[containerKey])
    ? pageOrderMap.value[containerKey].filter((id) => nextIds.includes(id))
    : [];
  const missingIds = nextIds.filter((id) => !currentIds.includes(id));
  if (missingIds.length === 0 && currentIds.length === nextIds.length) return;
  updateContainerOrder(containerKey, [...currentIds, ...missingIds]);
};

watch(
  () => createForm.value.type,
  (type) => {
    if (type !== "page") {
      createForm.value.parentId = null;
    }
  },
);

/**
 * 获取页面显示名称
 * @param {import('@/editor-core').PageNode} page - 页面
 * @returns {string}
 */
const getPageLabel = (page) => {
  return page.name || page.title || page.id;
};

const matchesKeyword = (page) => {
  const keyword = searchText.value.trim().toLowerCase();
  if (!keyword) return true;
  return (
    (page.name || "").toLowerCase().includes(keyword) ||
    (page.path || "").toLowerCase().includes(keyword)
  );
};

/**
 * 创建活跃页面ID映射（computed 确保响应式更新）
 */
const activePageMap = computed(() => {
  const map = new Map();
  if (currentPageId.value) {
    map.set(currentPageId.value, true);
  }
  return map;
});

/**
 * 判断页面是否选中
 * @param {string} pageId - 页面ID
 * @returns {boolean}
 */
const isPageActive = (pageId) => {
  return activePageMap.value.has(pageId);
};

/**
 * 同步页面选中状态（使用 watchEffect 确保响应式更新）
 */
watchEffect(() => {
  const pageId = currentPageId.value;
  const pageList = pages.value;

  if (!pageId || !pageList.length) return;

  const page = pageList.find((p) => p.id === pageId);
  if (page) {
    selectedNode.value = {
      id: page.id,
      type: "page",
      label: getPageLabel(page),
    };

    // 如果页面在分组中，自动展开该分组
    if (page.parentId) {
      folderStates.value[page.parentId] = true;
    }
  }
});

/**
 * 所有可见页面（包含登录页、登出页等系统页面）
 */
const appPages = computed(() =>
  pages.value.filter((page) => page.type !== "dialog"),
);
const systemPages = computed(() => ({
  home:
    appPages.value.find((page) => page.id === entryConfig.value?.homePageId) ||
    null,
  login:
    appPages.value.find((page) => page.id === entryConfig.value?.loginPageId) ||
    appPages.value.find((page) => getFixedSystemPath(page) === "/login") ||
    null,
  logout:
    appPages.value.find(
      (page) => page.id === entryConfig.value?.logoutPageId,
    ) ||
    appPages.value.find((page) => getFixedSystemPath(page) === "/logout") ||
    null,
}));
const basicPageIdSet = computed(
  () =>
    new Set(
      [
        systemPages.value.home?.id,
        systemPages.value.login?.id,
        systemPages.value.logout?.id,
      ].filter(Boolean),
    ),
);

const basicPageMap = computed(() => {
  const pageMap = new Map();
  appPages.value.forEach((page) => {
    pageMap.set(page.id, page);
  });
  return pageMap;
});

const basicPageSlots = computed(() => {
  const slots = [
    {
      type: "home",
      label: "首页",
      pageId: systemPages.value.home?.id || "",
    },
    {
      type: "login",
      label: "登录页",
      pageId: systemPages.value.login?.id || "",
    },
    {
      type: "logout",
      label: "登出页",
      pageId: systemPages.value.logout?.id || "",
    },
  ];
  return slots.map((slot) => ({
    ...slot,
    path: getBasicPageMeta(slot.type).path,
    page: slot.pageId ? basicPageMap.value.get(slot.pageId) || null : null,
  }));
});

const filteredBasicSlots = computed(() => {
  const keyword = searchText.value.trim().toLowerCase();
  if (!keyword) {
    return basicPageSlots.value;
  }
  return basicPageSlots.value.filter((slot) => {
    const label = slot.label.toLowerCase();
    const pageName = slot.page ? getPageLabel(slot.page).toLowerCase() : "";
    return label.includes(keyword) || pageName.includes(keyword);
  });
});

const businessPages = computed(() =>
  appPages.value.filter((page) => !basicPageIdSet.value.has(page.id)),
);

const businessFolders = computed(() =>
  businessPages.value.filter((page) => page.type === "folder"),
);

const businessRootItems = computed(() =>
  sortItemsByStoredOrder(
    businessPages.value.filter(
      (page) =>
        page.type === "folder" || (page.type === "page" && !page.parentId),
    ),
    ROOT_CONTAINER_KEY,
  ).sort((a, b) => {
    if (a.type === b.type) return 0;
    return a.type === "folder" ? -1 : 1;
  }),
);

const getChildrenCount = (folderId) =>
  businessPages.value.filter(
    (page) => page.type === "page" && page.parentId === folderId,
  ).length;

const getBusinessChildren = (folderId) =>
  sortItemsByStoredOrder(
    businessPages.value.filter(
      (page) =>
        page.type === "page" &&
        page.parentId === folderId &&
        matchesKeyword(page),
    ),
    getContainerOrderKey(folderId),
  );

const filteredBusinessRootItems = computed(() =>
  businessRootItems.value.filter((item) => {
    if (item.type === "folder") {
      return matchesKeyword(item) || getBusinessChildren(item.id).length > 0;
    }
    return matchesKeyword(item);
  }),
);

const moveIdBefore = (ids, sourceId, targetId) => {
  const nextIds = ids.filter((id) => id !== sourceId);
  const targetIndex = nextIds.indexOf(targetId);
  if (targetIndex === -1) {
    nextIds.push(sourceId);
    return nextIds;
  }
  nextIds.splice(targetIndex, 0, sourceId);
  return nextIds;
};

const appendIdToContainer = (containerKey, sourceId, visibleItems) => {
  const nextIds = (visibleItems || [])
    .map((item) => item.id)
    .filter((id) => id !== sourceId);
  nextIds.push(sourceId);
  updateContainerOrder(containerKey, nextIds);
};

/**
 * 仅在目标发生变化时更新拖拽高亮，避免 dragover 频繁触发整树重渲染
 * @param {{ id: string, mode: string, parentId: string | null }} nextTarget - 新目标
 */
const setDragOverTarget = (nextTarget) => {
  pendingDragOverTarget = nextTarget;
  if (dragOverFrameId) return;
  dragOverFrameId = requestAnimationFrame(() => {
    dragOverFrameId = 0;
    const current = dragOverTarget.value;
    const target = pendingDragOverTarget;
    if (!target) return;
    if (
      current?.id === target.id &&
      current?.mode === target.mode &&
      (current?.parentId || null) === (target.parentId || null)
    ) {
      return;
    }
    dragOverTarget.value = target;
  });
};

const handleDragStart = (item, parentId) => {
  dragState.value = {
    id: item.id,
    type: item.type,
    parentId: parentId || null,
  };
};

const handleDragEnd = () => {
  if (dragOverFrameId) {
    cancelAnimationFrame(dragOverFrameId);
    dragOverFrameId = 0;
  }
  pendingDragOverTarget = null;
  dragState.value = null;
  dragOverTarget.value = null;
};

const handleDragLeave = (targetId) => {
  if (dragOverTarget.value?.id === targetId) {
    dragOverTarget.value = null;
  }
  if (pendingDragOverTarget?.id === targetId) {
    pendingDragOverTarget = null;
  }
};

const handleNodeDragOver = (event, item, parentId) => {
  if (!dragState.value || dragState.value.id === item.id) return;
  event.dataTransfer.dropEffect = "move";
  setDragOverTarget({
    id: item.id,
    mode: "before",
    parentId: parentId || null,
  });
};

const handleFolderDragOver = (event, folder) => {
  if (!dragState.value || dragState.value.id === folder.id) return;
  event.dataTransfer.dropEffect = "move";
  setDragOverTarget({
    id: folder.id,
    mode: "append",
    parentId: folder.id,
  });
};

const handleContainerDragOver = (event, parentId) => {
  if (!dragState.value) return;
  event.dataTransfer.dropEffect = "move";
  setDragOverTarget({
    id: getContainerOrderKey(parentId),
    mode: "append",
    parentId: parentId || null,
  });
};

const handleNodeDrop = async (targetItem, parentId) => {
  if (!dragState.value || dragState.value.id === targetItem.id) return;
  const source = { ...dragState.value };
  const targetParentId = parentId || null;
  const sourceParentId = source.parentId || null;

  if (source.type === "page" && sourceParentId !== targetParentId) {
    await editorStore.movePageToGroup(source.id, targetParentId);
  }

  const visibleItems =
    targetParentId === null
      ? businessRootItems.value
      : businessPages.value.filter(
          (page) => page.type === "page" && page.parentId === targetParentId,
        );
  const nextOrder = moveIdBefore(
    visibleItems.map((item) => item.id),
    source.id,
    targetItem.id,
  );
  updateContainerOrder(getContainerOrderKey(targetParentId), nextOrder);

  if (sourceParentId !== targetParentId) {
    const sourceItems =
      sourceParentId === null
        ? businessRootItems.value.filter((item) => item.id !== source.id)
        : businessPages.value.filter(
            (page) =>
              page.type === "page" &&
              page.parentId === sourceParentId &&
              page.id !== source.id,
          );
    updateContainerOrder(
      getContainerOrderKey(sourceParentId),
      sourceItems.map((item) => item.id),
    );
  }
  handleDragEnd();
};

const handleFolderDrop = async (folder) => {
  if (!dragState.value || dragState.value.id === folder.id) return;
  const source = { ...dragState.value };
  if (source.type !== "page") {
    handleDragEnd();
    return;
  }
  if (source.parentId !== folder.id) {
    await editorStore.movePageToGroup(source.id, folder.id);
  }
  appendIdToContainer(
    getContainerOrderKey(folder.id),
    source.id,
    businessPages.value.filter(
      (page) => page.type === "page" && page.parentId === folder.id,
    ),
  );
  if (!folderStates.value[folder.id]) {
    folderStates.value[folder.id] = true;
  }
  if ((source.parentId || null) !== folder.id) {
    const sourceItems =
      source.parentId === null
        ? businessRootItems.value.filter((item) => item.id !== source.id)
        : businessPages.value.filter(
            (page) =>
              page.type === "page" &&
              page.parentId === source.parentId &&
              page.id !== source.id,
          );
    updateContainerOrder(
      getContainerOrderKey(source.parentId),
      sourceItems.map((item) => item.id),
    );
  }
  handleDragEnd();
};

const handleContainerDrop = async (parentId) => {
  if (!dragState.value) return;
  const source = { ...dragState.value };
  const targetParentId = parentId || null;

  if (source.type === "page" && (source.parentId || null) !== targetParentId) {
    await editorStore.movePageToGroup(source.id, targetParentId);
  }

  const visibleItems =
    targetParentId === null
      ? businessRootItems.value.filter((item) => item.id !== source.id)
      : businessPages.value.filter(
          (page) =>
            page.type === "page" &&
            page.parentId === targetParentId &&
            page.id !== source.id,
        );
  appendIdToContainer(
    getContainerOrderKey(targetParentId),
    source.id,
    visibleItems,
  );

  if ((source.parentId || null) !== targetParentId) {
    const sourceItems =
      source.parentId === null
        ? businessRootItems.value.filter((item) => item.id !== source.id)
        : businessPages.value.filter(
            (page) =>
              page.type === "page" &&
              page.parentId === source.parentId &&
              page.id !== source.id,
          );
    updateContainerOrder(
      getContainerOrderKey(source.parentId),
      sourceItems.map((item) => item.id),
    );
  }
  handleDragEnd();
};

watch(
  projectId,
  () => {
    loadPageOrderMap();
  },
  { immediate: true },
);

watch(
  [businessRootItems, businessFolders],
  () => {
    ensureContainerOrder(ROOT_CONTAINER_KEY, businessRootItems.value);
    businessFolders.value.forEach((folder) => {
      ensureContainerOrder(
        getContainerOrderKey(folder.id),
        businessPages.value.filter(
          (page) => page.type === "page" && page.parentId === folder.id,
        ),
      );
    });
  },
  { immediate: true, deep: true },
);

/**
 * 初始化切换提示配置
 * @returns {void}
 */
const initSwitchPromptState = () => {
  try {
    skipSwitchPrompt.value = localStorage.getItem(pageSwitchPromptKey) === "1";
  } catch (error) {
    skipSwitchPrompt.value = false;
  }
};

/**
 * 设置切换提示是否禁用
 * @param {boolean} disabled - 是否禁用
 * @returns {void}
 */
const setSwitchPromptDisabled = (disabled) => {
  skipSwitchPrompt.value = disabled;
  try {
    if (disabled) {
      localStorage.setItem(pageSwitchPromptKey, "1");
    } else {
      localStorage.removeItem(pageSwitchPromptKey);
    }
  } catch (error) {
    // 存储异常时保持内存状态
  }
};

initSwitchPromptState();

/**
 * 选中页面（只选中，不切换页面）
 * @param {{ id: string }} node - 点击的节点
 */
const handleNodeClick = (node) => {
  selectedNode.value = node;
  // 点击只选中节点，不切换页面
  // 页面切换通过双击 handleNodeDoubleClick 实现
};

/**
 * 处理基础页面槽位点击
 * @param {{ page?: import('@/editor-core').PageNode | null }} slot - 槽位
 * @returns {void}
 */
const handleBasicSlotClick = (slot) => {
  if (!slot?.page) {
    selectedNode.value = null;
    return;
  }
  handleNodeClick({ id: slot.page.id, type: "page" });
};

/**
 * 处理基础页面槽位双击
 * @param {{ page?: import('@/editor-core').PageNode | null }} slot - 槽位
 * @returns {void}
 */
const handleBasicSlotDoubleClick = (slot) => {
  if (!slot?.page) return;
  void handleNodeDoubleClick({
    id: slot.page.id,
    type: "page",
    label: slot.label,
  });
};

/**
 * 处理页面双击切换
 * @param {{ id: string, type: string, label?: string }} node - 双击的节点
 * @returns {Promise<void>}
 */
const handleNodeDoubleClick = async (node) => {
  if (!node || node.type !== "page") return;

  // 使用标签页系统打开页面
  if (openPageTab) {
    openPageTab(node.id);
    return;
  }

  // 兼容旧逻辑（如果没有注入 openPageTab）
  if (node.id === currentPageId.value) return;

  const targetPage = pages.value.find((page) => page.id === node.id);
  const targetName = targetPage?.name || node.label || node.id;

  const canSwitch = await confirmPageSwitch(targetName);
  if (!canSwitch) return;

  const safeToSwitch = await ensureUnsavedSwitch(targetName);
  if (!safeToSwitch) return;

  try {
    const result = await editorStore.loadPage(node.id);
    if (!result.ok) {
      ElMessage.error(result.error?.message || "切换页面失败");
    }
  } catch (error) {
    ElMessage.error("切换页面失败");
  }
};

/**
 * 确认是否切换页面
 * @param {string} targetName - 目标页面名称
 * @returns {Promise<boolean>}
 */
const confirmPageSwitch = async (targetName) => {
  if (skipSwitchPrompt.value) return true;

  const skipPrompt = ref(false);
  const message = h("div", { class: "flex flex-col gap-2" }, [
    h("div", `是否切换到${targetName}页面？`),
    h(
      ElCheckbox,
      {
        modelValue: skipPrompt.value,
        "onUpdate:modelValue": (value) => {
          skipPrompt.value = value;
        },
      },
      () => "不再提示",
    ),
  ]);

  try {
    await ElMessageBox({
      title: "切换页面",
      message,
      showCancelButton: true,
      confirmButtonText: "切换",
      cancelButtonText: "取消",
      distinguishCancelAndClose: true,
      closeOnClickModal: false,
    });
    if (skipPrompt.value) {
      setSwitchPromptDisabled(true);
    }
    return true;
  } catch (error) {
    return false;
  }
};

/**
 * 处理未保存切换提示
 * @param {string} targetName - 目标页面名称
 * @returns {Promise<boolean>}
 */
const ensureUnsavedSwitch = async (targetName) => {
  if (!canUndo.value) return true;
  try {
    const action = await ElMessageBox.confirm(
      `当前页面未保存，是否切换到${targetName}页面？`,
      "切换页面",
      {
        confirmButtonText: "保存并切换",
        cancelButtonText: "不保存切换",
        distinguishCancelAndClose: true,
        closeOnClickModal: false,
      },
    );
    if (action === "confirm") {
      await editorStore.saveCurrentPage();
    }
    return true;
  } catch (error) {
    if (error === "cancel") {
      return true;
    }
    if (error && error !== "close") {
      ElMessage.error("切换页面失败");
    }
    return false;
  }
};

const buildMoveCommand = (targetId) => `move:${targetId || "root"}`;

/**
 * 创建基础页面
 * @param {"home" | "login" | "logout"} basicType - 基础页面类型
 * @returns {Promise<void>}
 */
const createBasicPage = async (basicType) => {
  if (basicType === "home") {
    if (entryConfig.value?.homePageId) {
      ElMessage.warning("首页已存在");
      return;
    }
    if (!projectId.value) {
      ElMessage.error("缺少工程信息");
      return;
    }
    const result = await editorStore.createHomePage(projectId.value);
    if (!result?.ok || !result.pageId) {
      ElMessage.error(result?.error?.message || "首页创建失败");
      return;
    }
    await openCreatedPageTab(result.pageId);
    ElMessage.success("首页创建成功");
    return;
  }

  if (basicType === "login") {
    if (
      entryConfig.value?.loginPageId ||
      pages.value.some((page) => page.path === "/login")
    ) {
      ElMessage.warning("登录页已存在");
      return;
    }
    try {
      const meta = getBasicPageMeta("login");
      const schemaContent = editorStore.buildNewPageSchema({
        name: meta.label,
        path: meta.path,
      });
      const result = await editorStore.createPage({
        name: meta.label,
        type: "page",
        parentId: null,
        schemaContent,
      });
      const pageId = result?.id;
      if (!pageId) {
        throw new Error("登录页创建失败");
      }
      editorStore.updateEntry({ loginPageId: pageId });
      await editorStore.persistEntry();
      await openCreatedPageTab(pageId);
      ElMessage.success("登录页创建成功");
    } catch (error) {
      ElMessage.error(error?.message || "登录页创建失败");
    }
    return;
  }

  if (basicType === "logout") {
    if (
      entryConfig.value?.logoutPageId ||
      pages.value.some((page) => page.path === "/logout")
    ) {
      ElMessage.warning("登出页已存在");
      return;
    }
    try {
      const meta = getBasicPageMeta("logout");
      const schemaContent = editorStore.buildNewPageSchema({
        name: meta.label,
        path: meta.path,
      });
      const result = await editorStore.createPage({
        name: meta.label,
        type: "page",
        parentId: null,
        schemaContent,
      });
      const pageId = result?.id;
      if (!pageId) {
        throw new Error("登出页创建失败");
      }
      editorStore.updateEntry({ logoutPageId: pageId });
      await editorStore.persistEntry();
      await openCreatedPageTab(pageId);
      ElMessage.success("登出页创建成功");
    } catch (error) {
      ElMessage.error(error?.message || "登出页创建失败");
    }
  }
};

/**
 * 基础页面行尾操作
 * @param {string} command - 命令
 * @param {{ type: "home" | "login" | "logout", page?: import('@/editor-core').PageNode | null }} slot - 槽位
 * @returns {void}
 */
const handleBasicRowAction = (command, slot) => {
  if (!slot) return;
  if (command === "create") {
    void createBasicPage(slot.type);
    return;
  }
  if (!slot.page) return;
  selectedNode.value = {
    id: slot.page.id,
    type: "page",
    label: slot.label,
    parentId: null,
  };
  handleRowAction(command, slot.page);
};

/**
 * 行尾操作按钮
 * @param {string} command - 命令
 * @param {import('@/editor-core').PageNode} node - 节点
 * @returns {void}
 */
const handleRowAction = (command, node) => {
  if (!node) return;
  selectedNode.value = {
    id: node.id,
    type: node.type,
    label: getPageLabel(node),
    parentId: node.parentId || null,
  };

  if (command === "open" && node.type === "page") {
    void handleNodeDoubleClick({
      id: node.id,
      type: "page",
      label: getPageLabel(node),
    });
    return;
  }
  if (command === "rename") {
    if (getFixedSystemType(node)) {
      ElMessage.warning("基础页面名称固定，不支持重命名");
      return;
    }
    void handleRename();
    return;
  }
  if (command === "export" && node.type === "page") {
    handleExportPage(node.id, getPageLabel(node));
    return;
  }
  if (command === "createPage" && node.type === "folder") {
    handleCreateInFolder(node.id);
    return;
  }
  if (command === "moveToRoot" && node.type === "page") {
    void handleMove(node.id, null);
    return;
  }
  if (command.startsWith("move:") && node.type === "page") {
    const targetId = command.slice(5);
    void handleMove(node.id, targetId === "root" ? null : targetId);
    return;
  }
  if (command === "delete") {
    void handleDelete();
  }
};

/**
 * 导出页面 Schema
 * @param {string} pageId - 页面 ID
 * @param {string} label - 页面名称
 */
const handleExportPage = (pageId, label) => {
  if (!editorStore.doc || !pageId) {
    ElMessage.warning("暂无可导出的页面");
    return;
  }
  try {
    const payload = editorStore.serializer.exportPage(editorStore.doc, pageId);
    const json = JSON.stringify(payload, null, 2);
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const name = `${label || "page"}.json`;
    const link = document.createElement("a");
    link.href = url;
    link.download = name;
    link.click();
    URL.revokeObjectURL(url);
    ElMessage.success("已导出页面");
  } catch (error) {
    ElMessage.error("导出页面失败");
  }
};

/**
 * 打开新建弹窗
 */
const openCreateDialog = () => {
  createForm.value = {
    type: "page",
    name: "",
    parentId:
      selectedNode.value?.type === "folder" ? selectedNode.value.id : null,
  };
  createDialogVisible.value = true;
};

defineExpose({ openCreateDialog });

/**
 * 创建完成后打开对应标签并切换到页面
 * @param {string} pageId - 页面 ID
 */
const openCreatedPageTab = async (pageId) => {
  if (!pageId) return;
  selectedNode.value = { id: pageId, type: "page" };
  if (openPageTab) {
    openPageTab(pageId);
    return;
  }
  await editorStore.setCurrentPage(pageId);
};

/**
 * 移动页面到分组
 * @param {string} pageId - 页面 ID
 * @param {string | null} parentId - 分组 ID
 */
const handleMove = async (pageId, parentId) => {
  try {
    const page = pages.value.find((item) => item.id === pageId);
    const currentParentId = page?.parentId || null;
    if (currentParentId === (parentId || null)) {
      return;
    }
    if (getFixedSystemType(page) && parentId) {
      ElMessage.warning("基础页面不支持移动到分组");
      return;
    }
    const nextPath =
      page?.type === "page"
        ? getFixedSystemPath(page) ||
          buildBusinessPagePath(getPageLabel(page), parentId)
        : undefined;
    await editorStore.movePageToGroup(pageId, parentId, nextPath);
    const nextPage = pages.value.find((item) => item.id === pageId);
    if (nextPage?.type === "page") {
      await editorStore.renamePage(pageId, getPageLabel(nextPage), nextPath);
    }
    ElMessage.success("移动成功");
  } catch (error) {
    ElMessage.error("移动失败");
  }
};

/**
 * 校验名称唯一
 * @param {string} name - 名称
 * @param {string} [excludeId] - 排除的页面 ID
 * @returns {boolean}
 */
const isNameUnique = (name, excludeId) => {
  const lowerName = name.trim().toLowerCase();
  return !pages.value.some(
    (page) =>
      page.id !== excludeId &&
      (page.name || "").trim().toLowerCase() === lowerName,
  );
};

/**
 * 规范化路由片段
 * @param {string} value - 名称
 * @returns {string}
 */
const toPathSegment = (value) => {
  const normalized = value.trim().replace(/\s+/g, "-");
  const sanitized = normalized.replace(/[/?#\\]+/g, "-");
  return sanitized || "page";
};

/**
 * 获取直属分组路由片段
 * @param {string | null | undefined} parentId - 分组 ID
 * @returns {string[]}
 */
const getFolderPathSegments = (parentId) => {
  const folder = pages.value.find(
    (page) => page.id === (parentId || null) && page.type === "folder",
  );
  if (!folder) {
    return [];
  }
  return [toPathSegment(folder.name || folder.title || folder.id)];
};

/**
 * 生成业务页面路径
 * @param {string} name - 页面名称
 * @param {string | null | undefined} parentId - 分组 ID
 * @returns {string}
 */
const buildBusinessPagePath = (name, parentId) => {
  const segments = [...getFolderPathSegments(parentId), toPathSegment(name)];
  return `/${segments.filter(Boolean).join("/")}`;
};

/**
 * 同步分组直接子页面路径
 * @param {string} folderId - 分组 ID
 * @returns {Promise<void>}
 */
const syncFolderDescendantPaths = async (folderId) => {
  const descendants = pages.value.filter(
    (page) => page.type === "page" && page.parentId === folderId,
  );
  for (const page of descendants) {
    const fixedPath = getFixedSystemPath(page);
    if (fixedPath) continue;
    const nextPath = buildBusinessPagePath(
      getPageLabel(page),
      page.parentId || null,
    );
    if (page.path !== nextPath) {
      await editorStore.renamePage(page.id, getPageLabel(page), nextPath);
    }
  }
};

/**
 * 获取登录/登出固定路径
 * @param {import('@/editor-core').PageNode | undefined} page - 页面
 * @returns {string | null}
 */
function getFixedSystemPath(page) {
  const systemType = getFixedSystemType(page);
  return systemType ? getBasicPageMeta(systemType).path : null;
}

/**
 * 创建页面/分组
 */
const handleCreateConfirm = async () => {
  const name = createForm.value.name?.trim();
  if (!name) {
    ElMessage.warning("名称不能为空");
    return;
  }
  const nameValidation = validatePageName(name);
  if (!nameValidation.valid) {
    ElMessage.warning(nameValidation.message);
    return;
  }
  if (!isNameUnique(name)) {
    ElMessage.warning("页面或分组名称已存在");
    return;
  }

  const type = createForm.value.type;
  const parentId = type === "page" ? createForm.value.parentId || null : null;

  creating.value = true;
  try {
    if (type === "folder") {
      await editorStore.createPage({ name, type: "folder", parentId });
      ElMessage.success("分组创建成功");
      createDialogVisible.value = false;
      creating.value = false;
      return;
    }

    // 普通页面：一次性创建带 schema 的页面
    const schemaContent = editorStore.buildNewPageSchema({
      name,
      path: buildBusinessPagePath(name, parentId),
    });
    const result = await editorStore.createPage({
      name,
      type: "page",
      parentId,
      schemaContent,
    });
    const pageId = result?.id;
    if (!pageId) throw new Error("页面创建失败");
    await openCreatedPageTab(pageId);
    ElMessage.success("页面创建成功");
    createDialogVisible.value = false;
  } catch (error) {
    console.error("创建失败:", error);
    ElMessage.error(error?.message || "创建失败");
  } finally {
    creating.value = false;
  }
};

/**
 * 重命名页面/分组
 */
const handleRename = async () => {
  const target = selectedNode.value;
  if (!target) {
    ElMessage.warning("请先选择要重命名的页面或分组");
    return;
  }
  try {
    const result = await ElMessageBox.prompt("请输入新名称", "重命名", {
      inputValue: target.label || "",
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      closeOnClickModal: false,
    });
    const name = result?.value?.trim();
    if (!name) {
      ElMessage.warning("名称不能为空");
      return;
    }
    const nameValidation = validatePageName(name);
    if (!nameValidation.valid) {
      ElMessage.warning(nameValidation.message);
      return;
    }
    if (!isNameUnique(name, target.id)) {
      ElMessage.warning("页面或分组名称已存在");
      return;
    }
    const page = pages.value.find((item) => item.id === target.id);
    const fixedPath = getFixedSystemPath(page);
    const path =
      page?.type === "page"
        ? fixedPath || buildBusinessPagePath(name, page?.parentId || null)
        : undefined;
    await editorStore.renamePage(target.id, name, path);
    if (page?.type === "folder") {
      await syncFolderDescendantPaths(target.id);
    }
    ElMessage.success("重命名成功");
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error("重命名失败");
    }
  }
};

/**
 * 判断是否为首页
 * @param {string} pageId - 页面 ID
 * @returns {boolean}
 */
const isHomePage = (pageId) => {
  return entryConfig.value?.homePageId === pageId;
};

/**
 * 获取分组下的全部后代节点数量
 * @param {string} folderId - 分组 ID
 * @returns {number}
 */
const getFolderDescendantCount = (folderId) => {
  let count = 0;
  const stack = [folderId];
  while (stack.length) {
    const currentFolderId = stack.pop();
    const children = pages.value.filter(
      (page) => page.parentId === currentFolderId,
    );
    count += children.length;
    children
      .filter((page) => page.type === "folder")
      .forEach((folder) => stack.push(folder.id));
  }
  return count;
};

/**
 * 删除页面/分组
 */
const handleDelete = async () => {
  const target = selectedNode.value;
  if (!target) {
    ElMessage.warning("请先选择要删除的页面或分组");
    return;
  }

  // 首页保护：不允许删除首页
  if (target.type === "page" && isHomePage(target.id)) {
    ElMessage.warning("首页不可删除，请先设置其他页面为首页");
    return;
  }

  try {
    if (target.type === "folder") {
      const childCount = getFolderDescendantCount(target.id);
      try {
        await ElMessageBox.confirm(
          childCount > 0
            ? `分组 "${target.label}" 下还有 ${childCount} 个子项。建议优先选择“仅删除分组”，组下页面会自动移到根目录；“删除组和页面”会一并删除全部子项，且不可恢复。`
            : `确定删除分组 "${target.label}" 吗？`,
          "删除分组",
          {
            confirmButtonText: childCount > 0 ? "仅删除分组" : "删除分组",
            cancelButtonText: childCount > 0 ? "删除组和页面" : "取消",
            type: "warning",
            closeOnClickModal: false,
            distinguishCancelAndClose: true,
          },
        );
        await editorStore.deletePage(
          target.id,
          childCount > 0 ? "folder-only" : "single",
        );
      } catch (error) {
        if (error === "cancel" && childCount > 0) {
          await editorStore.deletePage(target.id, "cascade");
        } else if (error === "close") {
          return;
        } else if (error !== "cancel") {
          throw error;
        } else {
          return;
        }
      }

      selectedNode.value = null;
      ElMessage.success("删除成功");
      return;
    }

    await ElMessageBox.confirm(
      `确定删除 "${target.label}" 吗？此操作不可恢复。`,
      "删除确认",
      {
        confirmButtonText: "删除",
        cancelButtonText: "取消",
        type: "warning",
        confirmButtonClass: "el-button--danger",
      },
    );

    // 如果删除的是当前页面，先切换到首页
    if (target.type === "page" && target.id === currentPageId.value) {
      const homePageId = entryConfig.value?.homePageId;
      if (homePageId && homePageId !== target.id) {
        if (openPageTab) {
          openPageTab(homePageId);
        } else {
          await editorStore.setCurrentPage(homePageId);
        }
      }
    }

    await editorStore.deletePage(target.id);
    selectedNode.value = null;
    ElMessage.success("删除成功");
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error("删除失败");
    }
  }
};

/**
 * 分组选项
 */
const folderOptions = computed(() =>
  businessFolders.value.map((page) => ({ id: page.id, name: page.name })),
);

/**
 * 可移动目标列表
 * @param {{ parentId?: string | null, id?: string }} node - 节点
 * @returns {Array<{ id: string | null, label: string }>}
 */
const getMoveTargets = (node) => {
  if (getFixedSystemType(node)) {
    return [];
  }

  const targets = [];
  const groups = folderOptions.value.filter(
    (group) => group.id !== node.parentId,
  );
  for (const group of groups) {
    targets.push({ id: group.id, label: group.name });
  }
  return targets;
};
</script>

<style scoped>
/* ==================== 容器 ==================== */
.page-tree-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* 搜索框 */
.page-search {
  padding: 6px 10px 8px;
  flex-shrink: 0;
}

.page-search :deep(.el-input__wrapper) {
  min-height: 28px;
  padding-inline: 8px;
  box-shadow: 0 0 0 1px var(--designer-border-color, #e6e6e6) inset;
}

.page-search :deep(.el-input__inner) {
  font-size: 12px;
}

/* 树内容区 */
.page-tree-content {
  flex: 1;
  overflow-y: auto;
  padding: 2px 6px 8px;
}

.page-section {
  margin-bottom: 10px;
}

.page-section__header {
  display: flex;
  align-items: center;
  min-height: 22px;
  padding: 3px 8px 5px;
}

.page-section__title {
  font-size: 12px;
  font-weight: 600;
  color: var(--designer-text-primary, #303133);
}

.page-section__body {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.page-section__empty {
  padding: 4px 10px 2px;
  font-size: 11px;
  color: var(--designer-text-muted, #8c8c8c);
}

/* ==================== 树节点通用样式 ==================== */
.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  padding: 5px 8px;
  margin: 1px 0;
  border-radius: 4px;
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
  position: relative;
}

.tree-node:hover {
  background-color: rgba(15, 23, 42, 0.04);
}

.tree-node.is-active {
  background-color: rgba(15, 23, 42, 0.08);
}

.tree-node.is-active::before {
  content: none;
}

.tree-node.is-drop-target {
  background-color: rgba(59, 130, 246, 0.12);
  box-shadow: inset 0 0 0 1px rgba(59, 130, 246, 0.28);
}

.page-node--basic {
  background: rgba(15, 23, 42, 0.024);
}

.page-node--basic.is-empty {
  color: var(--designer-text-muted, #8c8c8c);
}

.folder-dropzone {
  margin: 2px 0 2px 26px;
  padding: 6px 8px;
  border: 1px dashed #d0d7de;
  border-radius: 4px;
  font-size: 11px;
  color: var(--designer-text-muted, #8c8c8c);
}

.dark .tree-node:hover {
  background-color: rgba(255, 255, 255, 0.06);
}

.dark .tree-node.is-active {
  background-color: rgba(255, 255, 255, 0.1);
}

/* 节点缩进区域 */
.node-indent {
  width: 16px;
  height: 20px;
  flex-shrink: 0;
  position: relative;
}

.node-indent--folder {
  width: 6px;
}

.node-indent--basic {
  width: 10px;
}

/* 层级连接线 */
.tree-line {
  position: absolute;
  background-color: #dcdfe6;
}

.dark .tree-line {
  background-color: #4a4a4a;
}

.tree-line.vertical {
  width: 1px;
  height: 100%;
  left: 6px;
  top: -12px;
}

.tree-line.horizontal {
  height: 1px;
  width: 14px;
  left: 2px;
  top: 50%;
}

/* 子节点容器 */
.tree-children {
  position: relative;
  margin-left: 4px;
  padding-left: 12px;
}

.tree-children::before {
  content: "";
  position: absolute;
  left: 20px;
  top: 0;
  bottom: 12px;
  width: 1px;
  background-color: #dcdfe6;
}

.dark .tree-children::before {
  background-color: #4a4a4a;
}

/* 节点箭头 */
.node-arrow {
  width: 12px;
  height: 12px;
  color: #909399;
  flex-shrink: 0;
  cursor: pointer;
  transition: transform 0.2s;
}

.node-arrow:hover {
  color: #606266;
}

/* 节点图标 */
.node-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.node-icon.folder {
  color: #e6a23c;
}

.node-icon.page {
  color: #909399;
}

.tree-node.is-active .node-icon.page {
  color: #606266;
}

/* 节点标签 */
.node-label {
  flex: 1;
  font-size: 12px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dark .node-label {
  color: #d1d5db;
}

.tree-node.is-active .node-label {
  color: #303133;
  font-weight: 500;
}

.node-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.node-main .node-label {
  flex: none;
}

.node-meta {
  font-size: 11px;
  line-height: 1.2;
  color: var(--designer-text-muted, #8c8c8c);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-action-btn {
  width: 18px;
  height: 18px;
  padding: 0;
  color: #909399;
  opacity: 0;
  transition:
    opacity 0.15s ease,
    color 0.15s ease,
    background-color 0.15s ease;
}

.tree-node:hover .node-action-btn,
.tree-node.is-active .node-action-btn,
.node-action-btn:focus-visible {
  opacity: 1;
}

.node-action-btn:hover {
  color: #606266;
  background: rgba(15, 23, 42, 0.05);
}

/* 节点计数 */
.node-count {
  font-size: 11px;
  color: #8c8c8c;
  background: rgba(15, 23, 42, 0.04);
  padding: 0 5px;
  line-height: 16px;
  border-radius: 10px;
}

.dark .node-count {
  background: #3a3a3a;
  color: #a0a0a0;
}

/* 节点标签 */
.node-badge {
  flex-shrink: 0;
  font-size: 10px;
  margin-left: 4px;
}

/* ==================== 分组节点 ==================== */
.folder-node:hover {
  background-color: rgba(15, 23, 42, 0.04);
}

.folder-node.is-active {
  background-color: rgba(15, 23, 42, 0.08);
}

.dark .folder-node:hover {
  background-color: rgba(255, 255, 255, 0.06);
}

.dark .folder-node.is-active {
  background-color: rgba(255, 255, 255, 0.1);
}

/* ==================== 空状态 ==================== */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 28px 16px;
  color: #909399;
}

.empty-icon {
  width: 32px;
  height: 32px;
  color: #dcdfe6;
  margin-bottom: 8px;
}

.dark .empty-icon {
  color: #4a4a4a;
}

.empty-text {
  font-size: 12px;
  color: #909399;
  margin: 0 0 2px 0;
}

.empty-hint {
  font-size: 11px;
  color: #c0c4cc;
  margin: 0;
}

/* ==================== 新建弹窗样式 ==================== */
.create-type-selector {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  margin-bottom: 20px;
}

.type-option {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border: 2px solid #e4e7ed;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.type-option:hover {
  border-color: #c0c4cc;
  background-color: #f5f7fa;
}

.type-option.is-active {
  border-color: #409eff;
  background-color: rgba(64, 158, 255, 0.05);
}

.dark .type-option {
  border-color: #3a3a3a;
}

.dark .type-option:hover {
  border-color: #4a4a4a;
  background-color: #2a2a2a;
}

.dark .type-option.is-active {
  border-color: #409eff;
  background-color: rgba(64, 158, 255, 0.1);
}

.type-icon {
  width: 24px;
  height: 24px;
  color: #909399;
  flex-shrink: 0;
}

.type-option.is-active .type-icon {
  color: #409eff;
}

.type-info {
  flex: 1;
  min-width: 0;
}

.type-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 4px;
}

.dark .type-name {
  color: #e5e7eb;
}

.type-desc {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}

.create-form {
  margin-top: 16px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
