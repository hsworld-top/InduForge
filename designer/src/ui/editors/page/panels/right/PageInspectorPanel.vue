<script setup lang="ts">
/**
 * 页面属性面板
 * 显示和编辑当前页面的配置信息
 */

import type {
  BackgroundConfig,
  ComponentNode,
  EntryConfig,
  PageNode,
} from "@/editor-core/document/types";
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, reactive, ref, watch } from "vue";
import FriendlyColorPicker from "@/components/common/FriendlyColorPicker.vue";
import MonacoEditor from "@/components/common/monaco-editor-async";
import { useEditorStore } from "@/stores/editor-store";

type PageType = "business" | "login" | "logout";
type SystemPageType = PageType | "home";

interface PageInspectorForm {
  name: string;
  path: string;
  description: string;
  pageType: PageType;
  x: number;
  y: number;
  windowWidth: number;
  windowHeight: number;
  width: number;
  height: number;
  showGrid: boolean;
  enableSnap: boolean;
  autoFit: boolean;
  lockAspectRatio: boolean;
  enableMinSize: boolean;
  fontAutoFit: boolean;
  windowStyle: string;
  permissionDesc: string;
  backgroundKind: BackgroundConfig["kind"];
  backgroundValue: string;
}

interface DiagnosticSummary {
  total: number;
  active: number;
  invalid: number;
}

interface SystemPageMeta {
  label: string;
  path: string;
}

interface CanvasStylePreset {
  id: string;
  label: string;
  content: string;
}

interface PageRecordLike {
  id: string;
  name?: string;
  path?: string;
  parentId?: string | null;
  type?: string;
  title?: string;
  config: PageNode["config"] & {
    description?: string;
    x?: number;
    y?: number;
    windowWidth?: number;
    windowHeight?: number;
    width?: number;
    height?: number;
    showGrid?: boolean;
    enableSnap?: boolean;
    autoFit?: boolean;
    lockAspectRatio?: boolean;
    enableMinSize?: boolean;
    fontAutoFit?: boolean;
    windowStyle?: string;
    permissionDesc?: string;
    background?: {
      kind?: BackgroundConfig["kind"];
      value?: string;
    };
  };
  styleConfig?: string;
}

interface EntryConfigLike extends EntryConfig {
  logoutPageId?: string | null;
}

function showWarningMessage(message: string): void {
  ElMessage.warning(message as never);
}

function showErrorMessage(message: string): void {
  ElMessage.error(message as never);
}

function showInfoMessage(message: string): void {
  ElMessage.info(message as never);
}

const editorStore = useEditorStore();
const { currentPage, currentPageId, pages, doc } = storeToRefs(editorStore);

const form = reactive<PageInspectorForm>({
  name: "",
  path: "",
  description: "",
  pageType: "business",
  x: 0,
  y: 0,
  windowWidth: 1990,
  windowHeight: 1100,
  width: 1920,
  height: 1080,
  showGrid: false,
  enableSnap: true,
  autoFit: true,
  lockAspectRatio: false,
  enableMinSize: false,
  fontAutoFit: false,
  windowStyle: "cover",
  permissionDesc: "0item",
  backgroundKind: "color",
  backgroundValue: "#ffffff",
});

const canvasStyleDialogVisible = ref(false);
const canvasStyleDraft = ref("");
const canvasStylePresets: CanvasStylePreset[] = [
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
const filteredCanvasPresetOptions = computed<CanvasStylePreset[]>(() => {
  const keyword = String(canvasPresetSearch.value || "")
    .trim()
    .toLowerCase();
  if (!keyword) return canvasStylePresets;
  return canvasStylePresets.filter((item) =>
    String(item.label || "")
      .toLowerCase()
      .includes(keyword),
  );
});

/**
 * 获取当前页面根节点
 */
const rootNode = computed<ComponentNode | null>(() => {
  if (!currentPage.value || !doc.value) return null;
  return doc.value.getNode?.(currentPage.value.rootNodeId) || null;
});

/**
 * 判断是否存在画布样式配置
 */
// 诊断统计
const diagnostic = reactive<DiagnosticSummary>({
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
const isSystemPage = computed<boolean>(
  () => isHomePage.value || form.pageType === "login" || form.pageType === "logout",
);
const BASIC_PAGE_META: Record<SystemPageType, SystemPageMeta> = {
  business: { label: "业务页", path: "" },
  home: { label: "首页", path: "/" },
  login: { label: "登录页", path: "/login" },
  logout: { label: "登出页", path: "/logout" },
};
const PAGE_NAME_DOTS_RE = /^\.+$/;
const PAGE_NAME_FORBIDDEN_RE = /[/?#\\%]/;
const NUMERIC_INPUT_RE = /^[0-9.\-]$/;
const PATH_SEGMENT_SPACE_RE = /\s+/g;
const PATH_SEGMENT_FORBIDDEN_RE = /[/?#\\]+/g;

function hasControlChars(value: string): boolean {
  for (let i = 0; i < value.length; i++) {
    const code = value.charCodeAt(i);
    if ((code >= 0 && code <= 31) || code === 127) {
      return true;
    }
  }
  return false;
}

/**
 * 获取基础页面类型
 * @param {Object | null | undefined} page - 页面对象
 * @returns {"home" | "login" | "logout" | null}
 */
function getFixedSystemType(page: PageRecordLike | null | undefined): PageType | "home" | null {
  if (!page) return null;
  const entry = doc.value?.entry as EntryConfigLike | undefined;
  if (entry?.homePageId === page.id) return "home";
  if (entry?.loginPageId === page.id || page.path === "/login") return "login";
  if (entry?.logoutPageId === page.id || page.path === "/logout") return "logout";
  return null;
}

/**
 * 规范化路由片段
 * @param {string} value - 名称
 * @returns {string}
 */
function toPathSegment(value: string): string {
  const normalized = value.trim().replace(PATH_SEGMENT_SPACE_RE, "-");
  const sanitized = normalized.replace(PATH_SEGMENT_FORBIDDEN_RE, "-");
  return sanitized || "page";
}

/**
 * 校验页面名称是否合法
 * @param {string} value - 页面名称
 * @returns {{ valid: boolean, message: string }}
 */
function validatePageName(value: string): { valid: boolean; message: string } {
  const name = String(value || "").trim();
  if (!name) {
    return { valid: false, message: "名称不能为空" };
  }
  if (PAGE_NAME_DOTS_RE.test(name)) {
    return { valid: false, message: "页面名称不能仅包含点号" };
  }
  if (PAGE_NAME_FORBIDDEN_RE.test(name)) {
    return { valid: false, message: "页面名称不能包含 / ? # % \\" };
  }
  if (hasControlChars(name)) {
    return { valid: false, message: "页面名称不能包含控制字符" };
  }
  return { valid: true, message: "" };
}

/**
 * 获取直属分组路径片段
 * @param {string | null | undefined} parentId - 分组 ID
 * @returns {string[]}
 */
function getFolderPathSegments(parentId: string | null | undefined): string[] {
  const folder = (pages.value as PageRecordLike[]).find(
    (page) => page.id === (parentId || null) && page.type === "folder",
  );
  if (!folder) {
    return [];
  }
  return [toPathSegment(folder.name || folder.title || folder.id)];
}

/**
 * 生成业务页面路由
 * @param {string} name - 页面名称
 * @param {string | null | undefined} parentId - 分组 ID
 * @returns {string}
 */
function buildBusinessPagePath(name: string, parentId: string | null | undefined): string {
  const segments = [...getFolderPathSegments(parentId), toPathSegment(name)];
  return `/${segments.filter(Boolean).join("/")}`;
}

/**
 * 格式化路由路径展示
 * @param {string} path - 路由路径
 * @returns {string}
 */
function formatPathForDisplay(path: string): string {
  if (!path) return "/";
  try {
    return decodeURIComponent(path);
  } catch {
    return path;
  }
}

/**
 * 获取页面类型
 * @param {Object} page - 页面对象
 * @returns {'business' | 'login' | 'logout'}
 */
function getPageType(page: PageRecordLike | null | undefined): PageType {
  if (!page || !doc.value) return "business";
  const entry = doc.value.entry as EntryConfigLike | undefined;
  if (entry?.loginPageId === page.id) return "login";
  if (entry?.logoutPageId === page.id) return "logout";
  if (page.path === "/login") return "login";
  if (page.path === "/logout") return "logout";
  return "business";
}

/**
 * 根据页面信息与名称生成路由
 * @param {Object} page - 页面对象
 * @param {string} name - 页面名称
 * @returns {string}
 */
function resolvePath(page: PageRecordLike | null | undefined, name: string): string {
  if (!page) return "/";
  const systemType = getFixedSystemType(page);
  if (systemType) {
    return BASIC_PAGE_META[systemType].path;
  }
  return buildBusinessPagePath(name, page.parentId || null);
}

/**
 * 判断页面名称是否唯一
 * @param {string} name - 页面名称
 * @param {string} excludeId - 排除的页面 ID
 * @returns {boolean}
 */
function isNameUnique(name: string, excludeId: string): boolean {
  const lowerName = name.trim().toLowerCase();
  return !pages.value.some(
    (page) => page.id !== excludeId && (page.name || "").trim().toLowerCase() === lowerName,
  );
}

/**
 * 同步表单状态
 * @param {Object} page - 页面对象
 */
function syncForm(page: PageRecordLike | null | undefined): void {
  // 优先使用列表中的页面数据，避免脏缓存
  const pageFromList = currentPageId.value
    ? (pages.value as PageRecordLike[]).find((p) => p.id === currentPageId.value) || null
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
  form.description = page?.config?.description ?? "";
  form.x = page?.config?.x ?? 0;
  form.y = page?.config?.y ?? 0;
  form.windowWidth = page?.config?.windowWidth ?? 1990;
  form.windowHeight = page?.config?.windowHeight ?? 1100;
  form.width = page?.config?.width ?? 1920;
  form.height = page?.config?.height ?? 1080;
  form.showGrid = page?.config?.showGrid ?? false;
  form.enableSnap = page?.config?.enableSnap ?? true;
  form.autoFit = page?.config?.autoFit ?? true;
  form.lockAspectRatio = page?.config?.lockAspectRatio ?? false;
  form.enableMinSize = page?.config?.enableMinSize ?? false;
  form.fontAutoFit = page?.config?.fontAutoFit ?? false;
  form.windowStyle = page?.config?.windowStyle ?? "cover";
  form.permissionDesc = page?.config?.permissionDesc ?? "0item";
  form.backgroundKind = page?.config?.background?.kind || "color";
  form.backgroundValue = page?.config?.background?.value || "#ffffff";

  // 更新诊断统计
  updateDiagnostic();
}

/**
 * 更新诊断统计
 */
function updateDiagnostic(): void {
  // 当前暂无实时诊断来源，先重置为 0
  diagnostic.total = 0;
  diagnostic.active = 0;
  diagnostic.invalid = 0;
}

/**
 * 规范化样式输出，保证包含选择器
 * @param {string} content - 样式内容
 * @returns {string}
 */
function formatCanvasStyleOutput(content: string): string {
  const text = String(content || "");
  if (text.includes("{")) return text;
  const trimmed = text.trim();
  if (!trimmed) return "";
  return `#domId {
${trimmed}
}`;
}

/**
 * 为样式补全 #domId 作用域
 * @param {string} content - 样式内容
 * @returns {string}
 */
function prefixCanvasStyleScope(content: string): string {
  const text = String(content || "").trim();
  if (!text || !text.includes("{")) return text;
  const blocks = text.split("}");
  const rebuilt = blocks
    .map((block) => {
      const [selector = "", body] = block.split("{");
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
}

// 监听页面切换，同步表单
watch(
  [() => currentPage.value, () => currentPageId.value, () => pages.value],
  () => {
    syncForm(currentPage.value);
  },
  { immediate: true },
);

/**
 * 更新页面名称并同步路由路径
 */
async function handleNameUpdate(): Promise<void> {
  const page = currentPage.value;
  if (!page) return;
  if (isSystemPage.value) {
    syncForm(page);
    return;
  }
  const name = form.name.trim();
  if (!name) {
    showWarningMessage("名称不能为空");
    syncForm(page);
    return;
  }
  const nameValidation = validatePageName(name);
  if (!nameValidation.valid) {
    showWarningMessage(nameValidation.message);
    syncForm(page);
    return;
  }
  if (!isNameUnique(name, page.id)) {
    showWarningMessage("页面名称已存在");
    syncForm(page);
    return;
  }
  const currentPageFromList =
    (pages.value as PageRecordLike[]).find((item) => item.id === page.id) ||
    (page as PageRecordLike);
  if (name === (page.name || "")) {
    const path = resolvePath(currentPageFromList, name);
    form.path = formatPathForDisplay(path);
    return;
  }
  const path = resolvePath(currentPageFromList, name);
  try {
    await editorStore.renamePage(page.id, name, path);
    syncForm(page);
  } catch {
    showErrorMessage("更新页面名称失败");
    syncForm(page);
  }
}

/**
 * 切换页面类型并更新入口配置
 * @param {string} type - 页面类型
 */
function handlePageTypeChange(type: PageType): void {
  if (!currentPage.value || !doc.value) return;

  const page = currentPage.value;
  if (!page) return;
  const pageId = page.id;
  let path = form.path;

  if (type === "login") {
    path = "/login";
    editorStore.updateEntry({
      loginPageId: pageId,
      logoutPageId: null,
    } as unknown as Partial<EntryConfig>);
    void editorStore.persistEntry();
  } else if (type === "logout") {
    path = "/logout";
    editorStore.updateEntry({
      logoutPageId: pageId,
      loginPageId: null,
    } as unknown as Partial<EntryConfig>);
    void editorStore.persistEntry();
  } else {
    // 普通业务页，清理入口配置
    const entry = (doc.value.entry || {}) as EntryConfigLike;
    if (entry.loginPageId === pageId) {
      editorStore.updateEntry({ loginPageId: null } as unknown as Partial<EntryConfig>);
      void editorStore.persistEntry();
    }
    if (entry.logoutPageId === pageId) {
      editorStore.updateEntry({ logoutPageId: null } as unknown as Partial<EntryConfig>);
      void editorStore.persistEntry();
    }
    path = buildBusinessPagePath(form.name, (page as PageRecordLike).parentId || null);
  }

  form.path = path;
  editorStore.updateCurrentPage({ path });
}

/**
 * 插入画布样式模板
 * @param {string} id - 模板 ID
 */
function handleCanvasPresetChange(id: string): void {
  const target = canvasStylePresets.find((item) => item.id === id);
  if (!target) return;
  const nextContent = formatCanvasStyleOutput(target.content || "");
  if (!nextContent) return;
  const current = String(canvasStyleDraft.value || "").trim();
  const separator = current ? "\\n\\n" : "";
  canvasStyleDraft.value = `${current}${separator}${nextContent}`;
}

/**
 * 更新画布配置
 */
function handleConfigUpdate(): void {
  if (!currentPage.value) return;
  const nextConfig = {
    ...(currentPage.value.config || {}),
    description: form.description,
    x: Number(form.x) || 0,
    y: Number(form.y) || 0,
    windowWidth: Number(form.windowWidth) || 1990,
    windowHeight: Number(form.windowHeight) || 1100,
    width: Number(form.width) || currentPage.value.config?.width || 1920,
    height: Number(form.height) || currentPage.value.config?.height || 1080,
    showGrid: form.showGrid,
    enableSnap: form.enableSnap,
    autoFit: form.autoFit,
    lockAspectRatio: form.lockAspectRatio,
    enableMinSize: form.enableMinSize,
    fontAutoFit: form.fontAutoFit,
    windowStyle: form.windowStyle,
    permissionDesc: form.permissionDesc,
  };
  editorStore.updateCurrentPage({ config: nextConfig });
}

/**
 * 过滤非数字输入（只允许数字、负号、小数点、退格、方向键等）
 * @param {KeyboardEvent} event - 键盘事件
 */
function filterNumericInput(event: KeyboardEvent): void {
  const allowed = [
    "Backspace",
    "Delete",
    "Tab",
    "Escape",
    "Enter",
    "ArrowLeft",
    "ArrowRight",
    "ArrowUp",
    "ArrowDown",
    "Home",
    "End",
  ];
  if (allowed.includes(event.key)) return;
  if ((event.ctrlKey || event.metaKey) && ["a", "c", "v", "x", "z"].includes(event.key)) {
    return;
  }
  if (NUMERIC_INPUT_RE.test(event.key)) return;
  event.preventDefault();
}

/**
 * 数字字段失焦时校验并回写
 * @param {string} field - 表单字段名
 * @param {FocusEvent} event - 失焦事件
 */
function handleNumericBlur(
  field: "x" | "y" | "windowWidth" | "windowHeight" | "width" | "height",
  event: FocusEvent,
): void {
  const raw = String(form[field] ?? "").trim();
  const num = Number(raw);
  if (raw === "" || !Number.isFinite(num)) {
    form[field] = 0;
  } else {
    form[field] = num;
  }
  const target = event.target as HTMLElement | null;
  target?.setAttribute("readonly", "");
  handleConfigUpdate();
}

/**
 * 打开权限描述配置（当前先提供占位入口）
 */
function handlePermissionConfig(): void {
  showInfoMessage("权限描述配置能力待接入");
}

/**
 * 保存样式配置
 */
function saveCanvasStyleDialog(): void {
  if (!rootNode.value) return;
  let content = String(canvasStyleDraft.value || "");
  content = formatCanvasStyleOutput(content);
  content = prefixCanvasStyleScope(content);
  editorStore.updateNode(rootNode.value.id, { styleConfig: content });
  void editorStore.saveCurrentPage?.();
  canvasStyleDialogVisible.value = false;
}

/**
 * 清空样式配置
 */
function clearCanvasStyleDialog(): void {
  if (!rootNode.value) return;
  canvasStyleDraft.value = "";
  editorStore.updateNode(rootNode.value.id, { styleConfig: "" });
  void editorStore.saveCurrentPage?.();
}

/**
 * 更新背景配置
 */
function handleBackgroundUpdate(): void {
  if (!currentPage.value) return;
  const nextConfig = {
    ...currentPage.value.config,
    background: {
      kind: form.backgroundKind,
      value: form.backgroundValue,
    },
  };
  editorStore.updateCurrentPage({ config: nextConfig });
}
</script>

<template>
  <div class="page-inspector-panel">
    <div class="page-prop-list">
      <div class="page-group">
        <div class="section-title">基本</div>
        <div class="page-prop-item">
          <div class="page-prop-label">名称</div>
          <div class="page-prop-editor">
            <el-input
              v-model="form.name"
              size="small"
              :disabled="isSystemPage"
              @blur="handleNameUpdate"
            />
          </div>
        </div>
        <div class="page-prop-item">
          <div class="page-prop-label">描述</div>
          <div class="page-prop-editor">
            <el-input v-model="form.description" size="small" @blur="handleConfigUpdate" />
          </div>
        </div>
        <div class="page-prop-item">
          <div class="page-prop-label">页面类型</div>
          <div class="page-prop-editor">
            <el-select
              v-model="form.pageType"
              size="small"
              :disabled="isHomePage"
              @change="handlePageTypeChange"
            >
              <el-option label="业务页面" value="business" />
              <el-option label="登录页" value="login" />
              <el-option label="登出页" value="logout" />
            </el-select>
          </div>
        </div>
        <div class="page-prop-item">
          <div class="page-prop-label">路由</div>
          <div class="page-prop-editor">
            <el-input v-model="form.path" size="small" disabled />
          </div>
        </div>
        <div class="page-prop-item">
          <div class="page-prop-label">位置</div>
          <div class="page-prop-editor">
            <div class="axis-inline-group">
              <div class="axis-inline-item">
                <span class="axis-inline-tag">X</span>
                <el-input v-model="form.x" size="small" disabled />
              </div>
              <div class="axis-inline-item">
                <span class="axis-inline-tag">Y</span>
                <el-input v-model="form.y" size="small" disabled />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="page-group">
        <div class="section-title">样式</div>
        <div class="page-prop-item">
          <div class="page-prop-label">背景类型</div>
          <div class="page-prop-editor">
            <el-select v-model="form.backgroundKind" size="small" @change="handleBackgroundUpdate">
              <el-option label="纯色" value="color" />
              <el-option label="图片" value="image" />
              <el-option label="渐变" value="gradient" />
            </el-select>
          </div>
        </div>
        <div class="page-prop-item page-prop-item--stacked">
          <div class="page-prop-label">背景值</div>
          <div class="page-prop-editor page-prop-editor-stacked">
            <FriendlyColorPicker
              v-if="form.backgroundKind === 'color'"
              v-model="form.backgroundValue"
              @change="handleBackgroundUpdate"
            />
            <el-input
              v-else
              v-model="form.backgroundValue"
              size="small"
              @blur="handleBackgroundUpdate"
            />
          </div>
        </div>
      </div>

      <div class="page-group">
        <div class="section-title">窗口大小</div>
        <div class="page-prop-item">
          <div class="page-prop-label">宽度</div>
          <div class="page-prop-editor">
            <el-input
              v-model="form.windowWidth"
              size="small"
              readonly
              class="num-readonly-input"
              @focus="$event.target.removeAttribute('readonly')"
              @blur="handleNumericBlur('windowWidth', $event)"
              @keydown="filterNumericInput"
            />
          </div>
        </div>
        <div class="page-prop-item">
          <div class="page-prop-label">高度</div>
          <div class="page-prop-editor">
            <el-input
              v-model="form.windowHeight"
              size="small"
              readonly
              class="num-readonly-input"
              @focus="$event.target.removeAttribute('readonly')"
              @blur="handleNumericBlur('windowHeight', $event)"
              @keydown="filterNumericInput"
            />
          </div>
        </div>
      </div>

      <div class="page-group">
        <div class="section-title">画面大小</div>
        <div class="page-prop-item">
          <div class="page-prop-label">宽度</div>
          <div class="page-prop-editor">
            <el-input
              v-model="form.width"
              size="small"
              readonly
              class="num-readonly-input"
              @focus="$event.target.removeAttribute('readonly')"
              @blur="handleNumericBlur('width', $event)"
              @keydown="filterNumericInput"
            />
          </div>
        </div>
        <div class="page-prop-item">
          <div class="page-prop-label">高度</div>
          <div class="page-prop-editor">
            <el-input
              v-model="form.height"
              size="small"
              readonly
              class="num-readonly-input"
              @focus="$event.target.removeAttribute('readonly')"
              @blur="handleNumericBlur('height', $event)"
              @keydown="filterNumericInput"
            />
          </div>
        </div>
      </div>

      <div class="page-group">
        <div class="section-title">窗口</div>
        <div class="page-prop-item page-prop-item--switch">
          <div class="page-prop-label">自适应</div>
          <div class="page-prop-editor page-prop-editor-switch">
            <el-switch v-model="form.autoFit" @change="handleConfigUpdate" />
          </div>
        </div>
        <div class="page-prop-item page-prop-item--switch">
          <div class="page-prop-label">启用锁定宽高比</div>
          <div class="page-prop-editor page-prop-editor-switch">
            <el-switch v-model="form.lockAspectRatio" @change="handleConfigUpdate" />
          </div>
        </div>
        <div class="page-prop-item page-prop-item--switch">
          <div class="page-prop-label">启用最小尺寸</div>
          <div class="page-prop-editor page-prop-editor-switch">
            <el-switch v-model="form.enableMinSize" @change="handleConfigUpdate" />
          </div>
        </div>
        <div class="page-prop-item page-prop-item--switch">
          <div class="page-prop-label">字体自适应</div>
          <div class="page-prop-editor page-prop-editor-switch">
            <el-switch v-model="form.fontAutoFit" @change="handleConfigUpdate" />
          </div>
        </div>
        <div class="page-prop-item">
          <div class="page-prop-label">窗口样式</div>
          <div class="page-prop-editor">
            <el-select v-model="form.windowStyle" size="small" @change="handleConfigUpdate">
              <el-option label="覆盖式" value="cover" />
              <el-option label="标准式" value="normal" />
            </el-select>
          </div>
        </div>
      </div>

      <div class="page-group">
        <div class="section-title">权限描述管理</div>
        <div class="page-prop-item">
          <div class="page-prop-label">配置</div>
          <div class="page-prop-editor">
            <el-button size="small" @click="handlePermissionConfig">
              {{ form.permissionDesc || "0item" }}
            </el-button>
          </div>
        </div>
      </div>
    </div>
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
      <MonacoEditor v-model="canvasStyleDraft" language="css" height="520px" />
    </div>
    <template #footer>
      <el-button @click="clearCanvasStyleDialog">清除</el-button>
      <el-button @click="canvasStyleDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="saveCanvasStyleDialog">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.page-inspector-panel {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-md);
}

.page-prop-list {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-md);
}

.page-group {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-sm);
  overflow: hidden;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
  padding: var(--designer-gap-sm);
}

.panel-title {
  display: flex;
  gap: var(--designer-gap-sm);
  font-size: var(--designer-font-lg);
  font-weight: 500;
}

.page-name {
  color: var(--designer-primary-text);
  font-weight: 600;
}

.section-title {
  display: flex;
  align-items: center;
  min-height: 34px;
  margin: calc(var(--designer-gap-sm) * -1) calc(var(--designer-gap-sm) * -1) 0;
  padding: 0 10px;
  background: var(--designer-group-surface);
  border-bottom: 1px solid var(--designer-border-soft);
  font-size: var(--designer-font-sm);
  font-weight: 600;
  color: var(--designer-text-secondary);
  letter-spacing: 0.02em;
}

.page-prop-item {
  display: flex;
  align-items: center;
  gap: var(--designer-gap-sm);
  min-height: 32px;
  padding: 4px 6px;
  border-radius: var(--designer-radius-sm);
  transition: background-color 0.15s ease;
}

.page-prop-item:hover {
  background: var(--designer-hover-surface);
}

.page-prop-item--stacked {
  align-items: flex-start;
}

.page-prop-label {
  width: 88px;
  min-width: 72px;
  max-width: 88px;
  flex-shrink: 0;
  font-size: var(--designer-font-sm);
  color: var(--designer-text-regular);
}

.page-prop-editor {
  flex: 1;
  min-width: 0;
}

.page-prop-editor-switch {
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

.page-prop-editor-inline {
  display: flex;
  align-items: center;
  gap: var(--designer-gap-sm);
}

.page-prop-editor-stacked {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: var(--designer-gap-xs);
}

.axis-inline-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--designer-gap-xs);
}

.axis-inline-item {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.axis-inline-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  min-width: 18px;
  height: 18px;
  border-radius: 999px;
  background: var(--designer-group-surface);
  color: var(--designer-text-secondary);
  font-size: 11px;
  font-weight: 600;
}

.page-prop-editor :deep(.el-input),
.page-prop-editor :deep(.el-input-number),
.page-prop-editor :deep(.el-select) {
  width: 100%;
}

.page-prop-item--switch {
  justify-content: space-between;
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
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-group-surface);
}

.config-toolbar-item {
  display: flex;
  gap: 6px;
}

.config-label {
  font-size: var(--designer-font-sm);
  color: var(--designer-text-secondary);
}

.config-select {
  width: 180px;
}

.config-editor {
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  overflow: hidden;
}

.is-active {
  border-color: var(--designer-primary-border);
  color: var(--designer-primary-text);
}
</style>
