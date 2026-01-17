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
      <!-- 分组列表 -->
      <template v-for="folder in filteredFolders" :key="folder.id">
        <div
          class="tree-node folder-node"
          :class="{ 'is-active': selectedNode?.id === folder.id }"
          @click="handleFolderClick(folder)"
          @contextmenu.prevent="
            handleNodeContextMenu($event, {
              id: folder.id,
              type: 'folder',
              label: folder.name,
            })
          "
        >
          <div class="node-indent" />
          <component
            :is="folderStates[folder.id] ? IconEpArrowDown : IconEpArrowRight"
            class="node-arrow"
            @click.stop="toggleFolder(folder.id)"
          />
          <IconEpFolder class="node-icon folder" />
          <span class="node-label">{{ folder.name }}</span>
          <span class="node-count">{{ getChildrenCount(folder.id) }}</span>
        </div>
        <div v-if="folderStates[folder.id]" class="tree-children">
          <div
            v-for="page in getFilteredChildren(folder.id)"
            :key="page.id"
            class="tree-node page-node level-1"
            :class="{ 'is-active': isPageActive(page.id) }"
            @click="handleNodeClick({ id: page.id, type: 'page' })"
            @dblclick="
              handleNodeDoubleClick({
                id: page.id,
                type: 'page',
                label: getPageLabel(page),
              })
            "
            @contextmenu.prevent="
              handleNodeContextMenu($event, {
                id: page.id,
                type: 'page',
                label: getPageLabel(page),
                parentId: folder.id,
              })
            "
          >
            <div class="node-indent">
              <span class="tree-line vertical" />
              <span class="tree-line horizontal" />
            </div>
            <IconEpDocument class="node-icon page" />
            <span class="node-label">{{ getPageLabel(page) }}</span>
            <el-tag
              v-for="badge in getPageBadges(page)"
              :key="badge.text"
              size="small"
              :type="badge.type"
              class="node-badge"
            >
              {{ badge.text }}
            </el-tag>
          </div>
        </div>
      </template>

      <!-- 根目录页面 -->
      <div
        v-for="page in filteredRootPages"
        :key="page.id"
        class="tree-node page-node"
        :class="{ 'is-active': isPageActive(page.id) }"
        @click="handleNodeClick({ id: page.id, type: 'page' })"
        @dblclick="
          handleNodeDoubleClick({
            id: page.id,
            type: 'page',
            label: getPageLabel(page),
          })
        "
        @contextmenu.prevent="
          handleNodeContextMenu($event, {
            id: page.id,
            type: 'page',
            label: getPageLabel(page),
          })
        "
      >
        <div class="node-indent" />
        <IconEpDocument class="node-icon page" />
        <span class="node-label">{{ getPageLabel(page) }}</span>
        <el-tag
          v-for="badge in getPageBadges(page)"
          :key="badge.text"
          size="small"
          :type="badge.type"
          class="node-badge"
        >
          {{ badge.text }}
        </el-tag>
      </div>

      <!-- 空状态 -->
      <div
        v-if="filteredAllPages.length === 0 && filteredFolders.length === 0"
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
  <!-- 右键菜单 -->
  <Teleport to="body">
    <div
      v-if="contextMenuVisible"
      class="context-menu-overlay"
      @click="contextMenuVisible = false"
      @contextmenu.prevent="contextMenuVisible = false"
    />
    <div
      v-if="contextMenuVisible"
      class="context-menu-popup"
      :style="{
        left: contextMenuPoint.x + 'px',
        top: contextMenuPoint.y + 'px',
      }"
    >
      <!-- 页面操作 -->
      <template v-if="contextMenuNode?.type === 'page'">
        <div class="menu-item" @click="handleCommand('open', contextMenuNode)">
          <IconEpView class="menu-icon" />
          <span>打开</span>
        </div>
        <div
          class="menu-item"
          @click="handleCommand('rename', contextMenuNode)"
        >
          <IconEpEdit class="menu-icon" />
          <span>重命名</span>
        </div>
        <div
          class="menu-item"
          @click="handleCommand('export', contextMenuNode)"
        >
          <IconEpDownload class="menu-icon" />
          <span>导出页面</span>
        </div>
        <div
          class="menu-item"
          @click="handleCommand('setHome', contextMenuNode)"
        >
          <IconEpHomeFilled class="menu-icon" />
          <span>设为首页</span>
        </div>
        <div class="menu-divider" />
        <div
          class="menu-submenu"
          :class="{ 'is-disabled': isHomePageNode(contextMenuNode) }"
        >
          <div class="menu-item has-submenu">
            <IconEpFolder class="menu-icon" />
            <span>{{
              isHomePageNode(contextMenuNode) ? "首页不可移动" : "移动到"
            }}</span>
            <IconEpArrowRight
              v-if="!isHomePageNode(contextMenuNode)"
              class="submenu-arrow"
            />
          </div>
          <div v-if="!isHomePageNode(contextMenuNode)" class="submenu-content">
            <div
              v-for="target in getMoveTargets(contextMenuNode)"
              :key="target.id ?? 'root'"
              class="menu-item"
              @click="handleMove(contextMenuNode.id, target.id)"
            >
              <component
                :is="target.id ? IconEpFolder : IconEpFolderOpened"
                class="menu-icon"
                :class="target.id ? 'text-yellow-500' : 'text-gray-400'"
              />
              <span>{{ target.label }}</span>
            </div>
            <div
              v-if="!getMoveTargets(contextMenuNode).length"
              class="menu-empty"
            >
              无可选分组
            </div>
          </div>
        </div>
        <div class="menu-divider" />
        <div
          class="menu-item"
          :class="{
            danger: !isHomePageNode(contextMenuNode),
            disabled: isHomePageNode(contextMenuNode),
          }"
          @click="
            !isHomePageNode(contextMenuNode) &&
            handleCommand('delete', contextMenuNode)
          "
        >
          <IconEpDelete class="menu-icon" />
          <span>{{
            isHomePageNode(contextMenuNode) ? "首页不可删除" : "删除"
          }}</span>
        </div>
      </template>

      <!-- 分组操作 -->
      <template v-else-if="contextMenuNode?.type === 'folder'">
        <div
          class="menu-item"
          @click="handleCommand('rename', contextMenuNode)"
        >
          <IconEpEdit class="menu-icon" />
          <span>重命名</span>
        </div>
        <div
          class="menu-item"
          @click="handleCreateInFolder(contextMenuNode.id)"
        >
          <IconEpPlus class="menu-icon" />
          <span>新建页面</span>
        </div>
        <div class="menu-divider" />
        <div
          class="menu-item danger"
          @click="handleCommand('delete', contextMenuNode)"
        >
          <IconEpDelete class="menu-icon" />
          <span>删除分组</span>
        </div>
      </template>
    </div>
  </Teleport>
</template>

<script setup>
import {
  computed,
  h,
  inject,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  watchEffect,
} from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { ElCheckbox, ElMessage, ElMessageBox } from "element-plus";
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
import IconEpView from "~icons/ep/view";
import IconEpHomeFilled from "~icons/ep/home-filled";
import IconEpDownload from "~icons/ep/download";

// 注入打开标签页的方法
const openPageTab = inject("openPageTab", null);

const editorStore = useEditorStore();
const { pages, currentPageId, entryConfig, canUndo } = storeToRefs(editorStore);
const selectedNode = ref(null);
const createDialogVisible = ref(false);
const createFormRef = ref(null);
const createForm = ref({ type: "page", name: "", parentId: null });
const creating = ref(false);
const skipSwitchPrompt = ref(false);
const pageSwitchPromptKey = "designer.pageSwitchPrompt.disabled";
const contextMenuVisible = ref(false);
const contextMenuNode = ref(null);
const contextMenuPoint = ref({ x: 0, y: 0 });

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
  {
    value: "login",
    label: "登录页",
    desc: "创建系统登录页面",
    icon: IconEpLock,
  },
  {
    value: "logout",
    label: "登出页",
    desc: "创建系统登出页面",
    icon: IconEpLock,
  },
];

// 新建弹窗标题
const createDialogTitle = computed(() => {
  const option = createTypeOptions.find(
    (o) => o.value === createForm.value.type
  );
  return `新建${option?.label || "页面"}`;
});

// 表单验证规则
const createFormRules = {
  name: [
    { required: true, message: "请输入名称", trigger: "blur" },
    { min: 1, max: 50, message: "名称长度在 1 到 50 个字符", trigger: "blur" },
  ],
};

/**
 * 在分组中创建页面
 * @param {string} folderId - 分组ID
 */
const handleCreateInFolder = (folderId) => {
  contextMenuVisible.value = false;
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

/**
 * 过滤后的分组
 */
const filteredFolders = computed(() => {
  return appPages.value.filter((page) => page.type === "folder");
});

/**
 * 获取分组下的子页面数量
 * @param {string} folderId - 分组ID
 * @returns {number}
 */
const getChildrenCount = (folderId) => {
  return appPages.value.filter(
    (page) => page.type === "page" && page.parentId === folderId
  ).length;
};

/**
 * 获取分组下过滤后的子页面
 * @param {string} folderId - 分组ID
 * @returns {Array}
 */
const getFilteredChildren = (folderId) => {
  const keyword = searchText.value.trim().toLowerCase();
  const children = appPages.value.filter(
    (page) => page.type === "page" && page.parentId === folderId
  );
  if (!keyword) return children;
  return children.filter(
    (page) =>
      (page.name || "").toLowerCase().includes(keyword) ||
      (page.path || "").toLowerCase().includes(keyword)
  );
};

/**
 * 页面排序权重（首页 > 登录/登出 > 其他）
 * @param {Object} page - 页面对象
 * @returns {number} 排序权重
 */
const getPageSortWeight = (page) => {
  const homePageId = entryConfig.value?.homePageId;
  const loginPageId = entryConfig.value?.loginPageId;
  const logoutPageId = entryConfig.value?.logoutPageId;

  if (page.id === homePageId) return 0; // 首页优先
  if (page.id === loginPageId) return 1; // 登录页次之
  if (page.id === logoutPageId) return 2; // 登出页再次
  return 3; // 其他页面
};

/**
 * 过滤后的根目录页面（无分组），按优先级排序
 */
const filteredRootPages = computed(() => {
  const keyword = searchText.value.trim().toLowerCase();
  let rootPages = appPages.value.filter(
    (page) => page.type === "page" && !page.parentId
  );

  // 按优先级排序：首页 > 登录/登出 > 其他
  rootPages = [...rootPages].sort(
    (a, b) => getPageSortWeight(a) - getPageSortWeight(b)
  );

  if (!keyword) return rootPages;
  return rootPages.filter(
    (page) =>
      (page.name || "").toLowerCase().includes(keyword) ||
      (page.path || "").toLowerCase().includes(keyword)
  );
});

/**
 * 过滤后的所有业务页（扁平化）
 */
const filteredAppPagesFlat = computed(() => {
  const keyword = searchText.value.trim().toLowerCase();
  const allPages = appPages.value.filter((page) => page.type === "page");
  if (!keyword) return allPages;
  return allPages.filter(
    (page) =>
      (page.name || "").toLowerCase().includes(keyword) ||
      (page.path || "").toLowerCase().includes(keyword)
  );
});

/**
 * 过滤后的所有页面（统一列表）
 */
const filteredAllPages = computed(() => {
  const keyword = searchText.value.trim().toLowerCase();
  const allPages = pages.value.filter((page) => page.type === "page");
  if (!keyword) return allPages;
  return allPages.filter(
    (page) =>
      (page.name || "").toLowerCase().includes(keyword) ||
      (page.path || "").toLowerCase().includes(keyword)
  );
});

const contextMenuVirtualRef = {
  getBoundingClientRect: () => {
    const { x, y } = contextMenuPoint.value;
    return {
      width: 0,
      height: 0,
      top: y,
      bottom: y,
      left: x,
      right: x,
    };
  },
};

watch(
  () => createForm.value.type,
  (type) => {
    if (type !== "page") {
      createForm.value.parentId = null;
    }
  }
);

/**
 * 获取页面显示名称
 * @param {import('@/editor-core').PageNode} page - 页面
 * @returns {string}
 */
const getPageLabel = (page) => {
  return page.name || page.title || page.id;
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
 * 获取页面标签数组
 * @param {import('@/editor-core').PageNode} page - 页面
 * @returns {Array<{text: string, type: string}>}
 */
const getPageBadges = (page) => {
  const badges = [];

  if (entryConfig.value?.homePageId === page.id) {
    badges.push({ text: "首页", type: "success" });
  }
  if (entryConfig.value?.loginPageId === page.id) {
    badges.push({ text: "登录", type: "warning" });
  }
  if (entryConfig.value?.logoutPageId === page.id) {
    badges.push({ text: "登出", type: "info" });
  }

  return badges;
};

/**
 * 页面标识（向后兼容，返回第一个标签）
 * @param {import('@/editor-core').PageNode} page - 页面
 * @returns {string}
 */
const getPageBadge = (page) => {
  const badges = getPageBadges(page);
  return badges.length > 0 ? badges[0].text : "";
};

/**
 * 获取页面标签类型（向后兼容，返回第一个标签类型）
 * @param {import('@/editor-core').PageNode} page - 页面
 * @returns {string}
 */
const getPageBadgeType = (page) => {
  const badges = getPageBadges(page);
  return badges.length > 0 ? badges[0].type : "info";
};

/**
 * 所有可见页面（包含登录页、登出页等系统页面）
 */
const appPages = computed(() =>
  pages.value.filter((page) => page.type !== "dialog")
);

const buildAppTree = (pageList) => {
  const nodeMap = new Map();
  const roots = [];

  for (const page of pageList) {
    nodeMap.set(page.id, {
      id: page.id,
      label: getPageLabel(page),
      type: page.type,
      parentId: page.parentId || null,
      badge: getPageBadge(page),
      children: [],
    });
  }

  for (const node of nodeMap.values()) {
    if (node.parentId && nodeMap.has(node.parentId)) {
      nodeMap.get(node.parentId).children.push(node);
    } else {
      roots.push(node);
    }
  }

  return roots;
};

const appTreeData = computed(() => buildAppTree(appPages.value));

/**
 * 选中分组名称
 * @returns {string}
 */
const selectedGroupLabel = computed(() => {
  if (selectedNode.value?.type === "folder") {
    return selectedNode.value.label || "未命名分组";
  }
  return "根目录";
});

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

const handleGlobalClick = () => {
  contextMenuVisible.value = false;
};

onMounted(() => {
  document.addEventListener("click", handleGlobalClick);
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleGlobalClick);
});

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
 * 处理页面双击切换
 * @param {{ id: string, type: string, label?: string }} node - 双击的节点
 * @returns {Promise<void>}
 */
const handleNodeDoubleClick = async (node) => {
  if (!node || node.type !== "page") return;

  // ✅ 使用标签页系统打开页面
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
 * 处理右键菜单
 * @param {MouseEvent} event - 鼠标事件
 * @param {{ id: string, type: string, label?: string }} node - 节点数据
 */
const handleNodeContextMenu = (event, node) => {
  if (!event) return;
  event.preventDefault();
  if (node) {
    selectedNode.value = node;
    contextMenuNode.value = node;
  }
  contextMenuPoint.value = { x: event.clientX, y: event.clientY };
  contextMenuVisible.value = true;
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
      () => "不再提示"
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
      }
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

/**
 * 右键菜单
 * @param {string} command - 命令
 * @param {{ id: string, type: string, label: string }} data - 节点
 */
const handleCommand = (command, data) => {
  selectedNode.value = data;
  contextMenuVisible.value = false;

  switch (command) {
    case "open":
      if (openPageTab && data.type === "page") {
        openPageTab(data.id);
      }
      break;
    case "rename":
      void handleRename();
      break;
    case "delete":
      void handleDelete();
      break;
    case "setHome":
      handleSetHome(data.id);
      break;
    case "export":
      handleExportPage(data.id, data.label);
      break;
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
 * 移动页面到分组
 * @param {string} pageId - 页面 ID
 * @param {string | null} parentId - 分组 ID
 */
const handleMove = async (pageId, parentId) => {
  try {
    await editorStore.movePageToGroup(pageId, parentId);
    contextMenuVisible.value = false;
    ElMessage.success("移动成功");
  } catch (error) {
    ElMessage.error("移动失败");
  }
};

/**
 * 设置首页
 * @param {string} pageId - 页面 ID
 */
const handleSetHome = (pageId) => {
  editorStore.updateEntry({ homePageId: pageId });
  void editorStore.persistEntry();
  ElMessage.success("已设置为首页");
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
      (page.name || "").trim().toLowerCase() === lowerName
  );
};

/**
 * 路由路径由名称生成
 * @param {string} name - 名称
 * @returns {string}
 */
const toRoutePath = (name) => {
  const normalized = name.trim().replace(/\s+/g, "-");
  const sanitized = normalized.replace(/[/?#\\]+/g, "-");
  return `/${sanitized || "page"}`;
};

/**
 * 获取登录/登出固定路径
 * @param {import('@/editor-core').PageNode | undefined} page - 页面
 * @returns {string | null}
 */
const getFixedSystemPath = (page) => {
  if (!page) return null;
  if (entryConfig.value?.loginPageId === page.id) return "/login";
  if (page.path === "/login") return "/login";
  if (page.path === "/logout") return "/logout";
  if (page.name?.includes("登录")) return "/login";
  if (page.name?.includes("登出")) return "/logout";
  return null;
};

/**
 * 创建页面/分组
 */
const handleCreateConfirm = async () => {
  const name = createForm.value.name?.trim();
  if (!name) {
    ElMessage.warning("名称不能为空");
    return;
  }
  if (!isNameUnique(name)) {
    ElMessage.warning("页面或分组名称已存在");
    return;
  }

  const type = createForm.value.type;
  const parentId =
    type === "page"
      ? createForm.value.parentId || null
      : selectedNode.value?.type === "folder"
        ? selectedNode.value.id
        : null;

  creating.value = true;
  try {
    if (type === "folder") {
      await editorStore.createPage({ name, type: "folder", parentId });
      ElMessage.success("分组创建成功");
      createDialogVisible.value = false;
      creating.value = false;
      return;
    }

    if (type === "login") {
      if (
        entryConfig.value?.loginPageId ||
        pages.value.some((page) => page.path === "/login")
      ) {
        ElMessage.warning("登录页已存在");
        creating.value = false;
        return;
      }
      // 一次性创建带 schema 的页面
      const schemaContent = editorStore.buildNewPageSchema({
        name,
        path: "/login",
      });
      const result = await editorStore.createPage({
        name,
        type: "page",
        parentId: null,
        schemaContent,
      });
      const pageId = result?.id;
      if (!pageId) throw new Error("登录页创建失败");
      editorStore.updateEntry({ loginPageId: pageId });
      await editorStore.persistEntry();
      ElMessage.success("登录页创建成功");
      createDialogVisible.value = false;
      return;
    }

    if (type === "logout") {
      if (pages.value.some((page) => page.path === "/logout")) {
        ElMessage.warning("登出页已存在");
        creating.value = false;
        return;
      }
      // 一次性创建带 schema 的页面
      const schemaContent = editorStore.buildNewPageSchema({
        name,
        path: "/logout",
      });
      const result = await editorStore.createPage({
        name,
        type: "page",
        parentId: null,
        schemaContent,
      });
      const pageId = result?.id;
      if (!pageId) throw new Error("登出页创建失败");
      editorStore.updateEntry({ logoutPageId: pageId });
      await editorStore.persistEntry();
      ElMessage.success("登出页创建成功");
      createDialogVisible.value = false;
      return;
    }

    // 普通页面：一次性创建带 schema 的页面
    const schemaContent = editorStore.buildNewPageSchema({
      name,
      path: toRoutePath(name),
    });
    const result = await editorStore.createPage({
      name,
      type: "page",
      parentId,
      schemaContent,
    });
    const pageId = result?.id;
    if (!pageId) throw new Error("页面创建失败");
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
    if (!isNameUnique(name, target.id)) {
      ElMessage.warning("页面或分组名称已存在");
      return;
    }
    const page = pages.value.find((item) => item.id === target.id);
    const fixedPath = getFixedSystemPath(page);
    const path =
      page?.type === "page" ? fixedPath || toRoutePath(name) : undefined;
    await editorStore.renamePage(target.id, name, path);
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
 * 判断右键菜单节点是否为首页
 * @param {Object} node - 节点
 * @returns {boolean}
 */
const isHomePageNode = (node) => {
  if (!node || node.type !== "page") return false;
  return isHomePage(node.id);
};

/**
 * 判断页面是否可删除
 * @param {string} pageId - 页面 ID
 * @returns {boolean}
 */
const canDeletePage = (pageId) => {
  // 首页不可删除
  if (isHomePage(pageId)) {
    return false;
  }
  return true;
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

  // ✅ 首页保护：不允许删除首页
  if (target.type === "page" && isHomePage(target.id)) {
    ElMessage.warning("首页不可删除，请先设置其他页面为首页");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `确定删除 "${target.label}" 吗？此操作不可恢复。`,
      "删除确认",
      {
        confirmButtonText: "删除",
        cancelButtonText: "取消",
        type: "warning",
        confirmButtonClass: "el-button--danger",
      }
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
  appPages.value
    .filter((page) => page.type === "folder")
    .map((page) => ({ id: page.id, name: page.name }))
);

/**
 * 可移动目标列表
 * @param {{ parentId?: string | null, id?: string }} node - 节点
 * @returns {Array<{ id: string | null, label: string }>}
 */
const getMoveTargets = (node) => {
  // 首页禁止移动到分组
  const homePageId = entryConfig.value?.homePageId;
  if (node?.id === homePageId) {
    return [];
  }

  const targets = [];
  if (node?.parentId) {
    targets.push({ id: null, label: "根目录" });
  }
  const groups = folderOptions.value.filter(
    (group) => group.id !== node.parentId
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
  padding: 8px 12px;
  flex-shrink: 0;
}

/* 树内容区 */
.page-tree-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px 8px;
}

/* ==================== 树节点通用样式 ==================== */
.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  margin: 1px 0;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  position: relative;
}

.tree-node:hover {
  background-color: rgba(59, 130, 246, 0.08);
}

.tree-node.is-active {
  background-color: rgba(59, 130, 246, 0.12);
}

.tree-node.is-active::before {
  content: "";
  position: absolute;
  left: 0;
  top: 4px;
  bottom: 4px;
  width: 3px;
  background-color: #409eff;
  border-radius: 0 2px 2px 0;
}

.dark .tree-node:hover {
  background-color: rgba(59, 130, 246, 0.15);
}

.dark .tree-node.is-active {
  background-color: rgba(59, 130, 246, 0.2);
}

/* 节点缩进区域 */
.node-indent {
  width: 16px;
  height: 24px;
  flex-shrink: 0;
  position: relative;
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
  width: 10px;
  left: 6px;
  top: 50%;
}

/* 子节点容器 */
.tree-children {
  position: relative;
  margin-left: 12px;
  padding-left: 8px;
}

.tree-children::before {
  content: "";
  position: absolute;
  left: 12px;
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
  width: 14px;
  height: 14px;
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
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.node-icon.folder {
  color: #e6a23c;
}

.node-icon.page {
  color: #909399;
}

.tree-node.is-active .node-icon.page {
  color: #409eff;
}

/* 节点标签 */
.node-label {
  flex: 1;
  font-size: 13px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dark .node-label {
  color: #d1d5db;
}

.tree-node.is-active .node-label {
  color: #409eff;
  font-weight: 500;
}

/* 节点计数 */
.node-count {
  font-size: 11px;
  color: #909399;
  background: #f0f2f5;
  padding: 1px 6px;
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
  background-color: rgba(230, 162, 60, 0.08);
}

.folder-node.is-active {
  background-color: rgba(230, 162, 60, 0.12);
}

.folder-node.is-active::before {
  background-color: #e6a23c;
}

.dark .folder-node:hover {
  background-color: rgba(230, 162, 60, 0.15);
}

/* ==================== 空状态 ==================== */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #909399;
}

.empty-icon {
  width: 48px;
  height: 48px;
  color: #dcdfe6;
  margin-bottom: 12px;
}

.dark .empty-icon {
  color: #4a4a4a;
}

.empty-text {
  font-size: 14px;
  color: #909399;
  margin: 0 0 4px 0;
}

.empty-hint {
  font-size: 12px;
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

/* ==================== 右键菜单样式 ==================== */
.context-menu-overlay {
  position: fixed;
  inset: 0;
  z-index: 999;
}

.context-menu-popup {
  position: fixed;
  z-index: 1000;
  min-width: 160px;
  padding: 6px 0;
  background: white;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border: 1px solid #e4e7ed;
}

.dark .context-menu-popup {
  background: #2a2a2a;
  border-color: #3a3a3a;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px;
  font-size: 13px;
  color: #606266;
  cursor: pointer;
  transition: all 0.15s;
}

.menu-item:hover {
  background-color: #f5f7fa;
  color: #409eff;
}

.dark .menu-item {
  color: #d1d5db;
}

.dark .menu-item:hover {
  background-color: #3a3a3a;
  color: #409eff;
}

.menu-item.danger {
  color: #f56c6c;
}

.menu-item.danger:hover {
  background-color: #fef0f0;
  color: #f56c6c;
}

.dark .menu-item.danger:hover {
  background-color: rgba(245, 108, 108, 0.1);
}

.menu-item.disabled {
  color: #c0c4cc;
  cursor: not-allowed;
}

.menu-item.disabled:hover {
  background-color: transparent;
  color: #c0c4cc;
}

.dark .menu-item.disabled {
  color: #5a5a5a;
}

.dark .menu-item.disabled:hover {
  background-color: transparent;
}

.menu-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.menu-divider {
  height: 1px;
  background-color: #e4e7ed;
  margin: 6px 0;
}

.dark .menu-divider {
  background-color: #3a3a3a;
}

.menu-submenu {
  position: relative;
}

.menu-submenu.is-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.menu-submenu.is-disabled .menu-item {
  pointer-events: none;
}

.menu-item.has-submenu {
  justify-content: space-between;
}

.submenu-arrow {
  width: 12px;
  height: 12px;
  color: #909399;
}

.submenu-content {
  display: none;
  position: absolute;
  left: 100%;
  top: -6px;
  min-width: 140px;
  padding: 6px 0;
  background: white;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border: 1px solid #e4e7ed;
}

.dark .submenu-content {
  background: #2a2a2a;
  border-color: #3a3a3a;
}

.menu-submenu:hover .submenu-content {
  display: block;
}

.menu-empty {
  padding: 8px 16px;
  font-size: 12px;
  color: #909399;
  text-align: center;
}
</style>
