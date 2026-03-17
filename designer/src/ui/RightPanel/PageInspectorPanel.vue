<template>
  <div class="page-inspector-panel">
    <el-form label-width="72px" size="small">
      <!-- 基础信息 -->
      <div class="section-title">基础信息</div>

      <el-form-item label="页面名称">
        <el-input v-model="form.name" :disabled="isSystemPage" @blur="handleNameUpdate" />
      </el-form-item>

      <el-form-item label="路由路径">
        <el-input v-model="form.path" disabled />
      </el-form-item>

      <el-divider />

      <!-- 样式设置 -->
      <div class="section-title">样式设置</div>
      <el-form-item label="自适应">
        <el-switch
          v-model="form.autoFit"
          @change="handleConfigUpdate"
        />
      </el-form-item>
      <el-form-item label="样式配置">
        <el-button
          size="small"
          :disabled="!rootNode"
          :class="{ 'is-active': hasCanvasStyleConfig }"
          @click="openCanvasStyleDialog"
        >
          配置
        </el-button>
      </el-form-item>


      <el-divider />

      <!-- 背景设置 -->
      <div class="section-title">背景设置</div>

      <el-form-item label="背景类型">
        <el-select v-model="form.backgroundKind" @change="handleBackgroundUpdate">
          <el-option label="颜色" value="color" />
          <el-option label="图片" value="image" />
          <el-option label="渐变" value="gradient" />
        </el-select>
      </el-form-item>

      <el-form-item v-if="form.backgroundKind === 'color'" label="背景色">
        <FriendlyColorPicker
          v-model="form.backgroundValue"
          @change="handleBackgroundUpdate"
        />
      </el-form-item>

      <el-form-item v-else label="背景值">
        <el-input
          v-model="form.backgroundValue"
          @blur="handleBackgroundUpdate"
        />
      </el-form-item>

      <el-divider />

      <!-- 数据点状态 -->
      <div class="section-title">数据点状态</div>

      <el-form-item label="数据点">
        <div class="diagnostic-info">
          <el-tag type="success" size="small">
            正常 {{ diagnostic.active }}
          </el-tag>
          <el-tag v-if="diagnostic.invalid > 0" type="danger" size="small">
            失效 {{ diagnostic.invalid }}
          </el-tag>
          <el-tag type="info" size="small">
            共 {{ diagnostic.total }}
          </el-tag>
        </div>
      </el-form-item>
    </el-form>
  </div>

  <el-dialog
    v-model="canvasStyleDialogVisible"
    title="样式配置"
    width="980px"
    top="4vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <div class="config-toolbar">
      <div class="config-toolbar-item">
        <span class="config-label">样式模板：</span>
        <el-select
          v-model="selectedCanvasPresetId"
          size="small"
          class="config-select preset-select"
          placeholder="请选择"
          @change="handleCanvasPresetChange"
        >
          <el-option
            v-for="item in filteredCanvasPresetOptions"
            :key="item.id"
            :label="item.label"
            :value="item.id"
          />
        </el-select>
      </div>
      <div class="config-toolbar-item">
        <span class="config-label">筛选：</span>
        <el-input
          v-model="canvasPresetSearch"
          size="small"
          class="config-select"
          placeholder="搜索模板"
          clearable
        />
      </div>
    </div>
    <div class="config-editor">
      <MonacoEditor
        v-model="canvasStyleDraft"
        language="css"
        height="520px"
      />
    </div>
    <template #footer>
      <el-button @click="clearCanvasStyleDialog">清除</el-button>
      <el-button @click="canvasStyleDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="saveCanvasStyleDialog">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
/**
 * 页面属性面板
 * 显示和编辑当前页面的配置信息
 */

import { computed, reactive, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { ElMessage } from "element-plus";
import MonacoEditor from "@/components/common/MonacoEditor.vue";
import FriendlyColorPicker from "@/components/common/FriendlyColorPicker.vue";
import IconEpHomeFilled from "~icons/ep/home-filled";

const editorStore = useEditorStore();
const { currentPage, currentPageId, pages, doc } = storeToRefs(editorStore);

const form = reactive({
  name: "",
  path: "",
  pageType: "business",
  width: 1920,
  height: 1080,
  showGrid: false,
  enableSnap: true,
  autoFit: true,
  backgroundKind: "color",
  backgroundValue: "#ffffff",
});

const canvasStyleDialogVisible = ref(false);
const canvasStyleDraft = ref("");
const canvasStylePresets = [
  { id: "empty", label: "空模板", content: "" },
  {
    id: "center",
    label: "居中布局",
    content: "display: flex;\nalign-items: center;\njustify-content: center;",
  },
];
const selectedCanvasPresetId = ref("");
const canvasPresetSearch = ref("");

/**
 * 根据搜索关键字过滤样式模板
 */
const filteredCanvasPresetOptions = computed(() => {
  const keyword = String(canvasPresetSearch.value || "").trim().toLowerCase();
  if (!keyword) return canvasStylePresets;
  return canvasStylePresets.filter((item) =>
    String(item.label || "").toLowerCase().includes(keyword)
  );
});

/**
 * 获取当前页面根节点
 */
const rootNode = computed(() => {
  if (!currentPage.value || !doc.value) return null;
  return doc.value.getNode?.(currentPage.value.rootNodeId) || null;
});

/**
 * 判断是否存在画布样式配置
 */
const hasCanvasStyleConfig = computed(() => {
  const value = rootNode.value?.styleConfig;
  return Boolean(String(value || "").trim());
});

// 诊断统计
const diagnostic = reactive({
  total: 0,
  active: 0,
  invalid: 0,
});

/**
 * 判断当前页面是否为首页
 */
const isHomePage = computed(() => {
  if (!currentPageId.value || !doc.value) return false;
  return doc.value.entry?.homePageId === currentPageId.value;
});
const isSystemPage = computed(
  () => isHomePage.value || form.pageType === "login" || form.pageType === "logout"
);
const BASIC_PAGE_META = {
  home: { label: "首页", path: "/" },
  login: { label: "登录页", path: "/login" },
  logout: { label: "登出页", path: "/logout" },
};

/**
 * 获取基础页面类型
 * @param {Object | null | undefined} page - 页面对象
 * @returns {"home" | "login" | "logout" | null}
 */
const getFixedSystemType = (page) => {
  if (!page) return null;
  if (doc.value?.entry?.homePageId === page.id) return "home";
  if (doc.value?.entry?.loginPageId === page.id || page.path === "/login") return "login";
  if (doc.value?.entry?.logoutPageId === page.id || page.path === "/logout") return "logout";
  return null;
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
 * 获取直属分组路径片段
 * @param {string | null | undefined} parentId - 分组 ID
 * @returns {string[]}
 */
const getFolderPathSegments = (parentId) => {
  const folder = pages.value.find(
    (page) => page.id === (parentId || null) && page.type === "folder"
  );
  if (!folder) {
    return [];
  }
  return [toPathSegment(folder.name || folder.title || folder.id)];
};

/**
 * 生成业务页面路由
 * @param {string} name - 页面名称
 * @param {string | null | undefined} parentId - 分组 ID
 * @returns {string}
 */
const buildBusinessPagePath = (name, parentId) => {
  const segments = [...getFolderPathSegments(parentId), toPathSegment(name)];
  return `/${segments.filter(Boolean).join("/")}`;
};

/**
 * 格式化路由路径展示
 * @param {string} path - 路由路径
 * @returns {string}
 */
const formatPathForDisplay = (path) => {
  if (!path) return "/";
  try {
    return decodeURIComponent(path);
  } catch (error) {
    return path;
  }
};

/**
 * 获取页面类型
 * @param {Object} page - 页面对象
 * @returns {'business' | 'login' | 'logout'}
 */
const getPageType = (page) => {
  if (!page || !doc.value) return "business";
  if (doc.value.entry?.loginPageId === page.id) return "login";
  if (doc.value.entry?.logoutPageId === page.id) return "logout";
  if (page.path === "/login") return "login";
  if (page.path === "/logout") return "logout";
  return "business";
};

/**
 * 根据页面信息与名称生成路由
 * @param {Object} page - 页面对象
 * @param {string} name - 页面名称
 * @returns {string}
 */
const resolvePath = (page, name) => {
  if (!page) return "/";
  const systemType = getFixedSystemType(page);
  if (systemType) {
    return BASIC_PAGE_META[systemType].path;
  }
  return buildBusinessPagePath(name, page.parentId || null);
};

/**
 * 判断页面名称是否唯一
 * @param {string} name - 页面名称
 * @param {string} excludeId - 排除的页面 ID
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
 * 同步表单状态
 * @param {Object} page - 页面对象
 */
const syncForm = (page) => {
  // 优先使用列表中的页面数据，避免脏缓存
  const pageFromList = currentPageId.value
    ? pages.value.find((p) => p.id === currentPageId.value)
    : null;

  if (!page && !pageFromList) {
    return;
  }

  // 更新名称
  const displayPage = pageFromList || page;
  const systemType = getFixedSystemType(displayPage);
  const nextName = systemType
    ? BASIC_PAGE_META[systemType].label
    : pageFromList?.name || page?.name || "";
  form.name = nextName;

  // 更新路由路径
  const path = resolvePath(displayPage, nextName);
  form.path = formatPathForDisplay(path);

  form.pageType = getPageType(page || pageFromList);
  form.width = page?.config?.width ?? 1920;
  form.height = page?.config?.height ?? 1080;
  form.showGrid = page?.config?.showGrid ?? false;
  form.enableSnap = page?.config?.enableSnap ?? true;
  form.autoFit = page?.config?.autoFit ?? true;
  form.backgroundKind = page?.config?.background?.kind || "color";
  form.backgroundValue = page?.config?.background?.value || "#ffffff";

  // 更新诊断统计
  updateDiagnostic();
};

/**
 * 更新诊断统计
 */
const updateDiagnostic = () => {
  // 当前暂无实时诊断来源，先重置为 0
  diagnostic.total = 0;
  diagnostic.active = 0;
  diagnostic.invalid = 0;
};

/**
 * 规范化样式输出，保证包含选择器
 * @param {string} content - 样式内容
 * @returns {string}
 */
const formatCanvasStyleOutput = (content) => {
  const text = String(content || "");
  if (text.includes("{")) return text;
  const trimmed = text.trim();
  if (!trimmed) return "";
  return `#domId {
${trimmed}
}`;
};

/**
 * 为样式补全 #domId 作用域
 * @param {string} content - 样式内容
 * @returns {string}
 */
const prefixCanvasStyleScope = (content) => {
  const text = String(content || "").trim();
  if (!text || !text.includes("{")) return text;
  const blocks = text.split("}");
  const rebuilt = blocks
    .map((block) => {
      const [selector, body] = block.split("{");
      if (!body) return "";
      const trimmedSelector = selector.trim();
      if (!trimmedSelector) return "";
      if (trimmedSelector.startsWith("@")) {
        return `${trimmedSelector} {${body}`;
      }
      const selectors = trimmedSelector.split(",").map((item) => {
        const sel = item.trim();
        if (!sel) return "";
        if (sel.includes("#domId")) return sel;
        if (sel.startsWith(":")) return `#domId${sel}`;
        return `#domId ${sel}`;
      });
      return `${selectors.filter(Boolean).join(", ")} {${body}`;
    })
    .filter(Boolean)
    .join("}\n");
  return rebuilt ? `${rebuilt}}` : text;
};

// 监听页面切换，同步表单
watch(
  [() => currentPage.value, () => currentPageId.value, () => pages.value],
  () => {
    syncForm(currentPage.value);
  },
  { immediate: true }
);

/**
 * 更新页面名称并同步路由路径
 */
const handleNameUpdate = async () => {
  if (!currentPage.value) return;
  if (isSystemPage.value) {
    syncForm(currentPage.value);
    return;
  }
  const name = form.name.trim();
  if (!name) {
    ElMessage.warning("名称不能为空");
    syncForm(currentPage.value);
    return;
  }
  const nameValidation = validatePageName(name);
  if (!nameValidation.valid) {
    ElMessage.warning(nameValidation.message);
    syncForm(currentPage.value);
    return;
  }
  if (!isNameUnique(name, currentPage.value.id)) {
    ElMessage.warning("页面名称已存在");
    syncForm(currentPage.value);
    return;
  }
  const currentPageFromList = pages.value.find((page) => page.id === currentPage.value.id) || currentPage.value;
  if (name === (currentPage.value.name || "")) {
    const path = resolvePath(currentPageFromList, name);
    form.path = formatPathForDisplay(path);
    return;
  }
  const path = resolvePath(currentPageFromList, name);
  try {
    await editorStore.renamePage(currentPage.value.id, name, path);
    syncForm(currentPage.value);
  } catch (error) {
    ElMessage.error("更新页面名称失败");
    syncForm(currentPage.value);
  }
};

/**
 * 切换页面类型并更新入口配置
 * @param {string} type - 页面类型
 */
const handlePageTypeChange = (type) => {
  if (!currentPage.value || !doc.value) return;

  const pageId = currentPage.value.id;
  let path = form.path;

  if (type === "login") {
    path = "/login";
    editorStore.updateEntry({ loginPageId: pageId });
    void editorStore.persistEntry();
  } else if (type === "logout") {
    path = "/logout";
    editorStore.updateEntry({ logoutPageId: pageId });
    void editorStore.persistEntry();
  } else {
    // 普通业务页，清理入口配置
    const entry = doc.value.entry || {};
    if (entry.loginPageId === pageId) {
      editorStore.updateEntry({ loginPageId: null });
      void editorStore.persistEntry();
    }
    if (entry.logoutPageId === pageId) {
      editorStore.updateEntry({ logoutPageId: null });
      void editorStore.persistEntry();
    }
    path = buildBusinessPagePath(form.name, currentPage.value.parentId || null);
  }

  form.path = path;
  editorStore.updateCurrentPage({ path });
};

/**
 * 设置当前页面为首页
 */
const handleSetAsHome = () => {
  if (!currentPageId.value) return;
  editorStore.updateEntry({ homePageId: currentPageId.value });
  void editorStore.persistEntry();
  ElMessage.success("已设为首页");
};

/**
 * 插入画布样式模板
 * @param {any} id - 模板 ID
 */
const handleCanvasPresetChange = (id) => {
  const target = canvasStylePresets.find((item) => item.id === id);
  if (!target) return;
  const nextContent = formatCanvasStyleOutput(target.content || "");
  if (!nextContent) return;
  const current = String(canvasStyleDraft.value || "").trim();
  const separator = current ? "\\n\\n" : "";
  canvasStyleDraft.value = `${current}${separator}${nextContent}`;
};

/**
 * 更新画布配置
 */
const handleConfigUpdate = () => {
  if (!currentPage.value) return;
  const nextConfig = {
    ...currentPage.value.config,
    width: Number(form.width) || currentPage.value.config?.width || 1920,
    height: Number(form.height) || currentPage.value.config?.height || 1080,
    showGrid: form.showGrid,
    enableSnap: form.enableSnap,
    autoFit: form.autoFit,
  };
  editorStore.updateCurrentPage({ config: nextConfig });
};

/**
 * 打开样式配置弹窗
 */
const openCanvasStyleDialog = () => {
  if (!rootNode.value) return;
  canvasStyleDraft.value = rootNode.value.styleConfig || "";
  canvasStyleDialogVisible.value = true;
};

/**
 * 保存样式配置
 */
const saveCanvasStyleDialog = () => {
  if (!rootNode.value) return;
  let content = String(canvasStyleDraft.value || "");
  content = formatCanvasStyleOutput(content);
  content = prefixCanvasStyleScope(content);
  editorStore.updateNode(rootNode.value.id, { styleConfig: content });
  void editorStore.saveCurrentPage?.();
  canvasStyleDialogVisible.value = false;
};

/**
 * 清空样式配置
 */
const clearCanvasStyleDialog = () => {
  if (!rootNode.value) return;
  canvasStyleDraft.value = "";
  editorStore.updateNode(rootNode.value.id, { styleConfig: "" });
  void editorStore.saveCurrentPage?.();
};

/**
 * 更新背景配置
 */
const handleBackgroundUpdate = () => {
  if (!currentPage.value) return;
  const nextConfig = {
    ...currentPage.value.config,
    background: {
      kind: form.backgroundKind,
      value: form.backgroundValue,
    },
  };
  editorStore.updateCurrentPage({ config: nextConfig });
};
</script>



<style scoped>
.page-inspector-panel {
  padding: 12px;
}

.panel-title {
  display: flex;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
}

.page-name {
  color: var(--el-color-primary);
  font-weight: 600;
}

.section-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin: 12px 0 8px;
}

.diagnostic-info {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.config-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
  margin-bottom: 12px;
  padding: 8px 10px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: #fafafa;
}

.config-toolbar-item {
  display: flex;
  gap: 6px;
}

.config-label {
  font-size: 12px;
  color: #606266;
}

.config-select {
  width: 180px;
}

.config-editor {
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  overflow: hidden;
}

.is-active {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
</style>
