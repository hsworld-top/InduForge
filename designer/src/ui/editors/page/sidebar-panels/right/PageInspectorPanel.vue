<script setup lang="ts">
/**
 * 页面属性主面板
 *
 * 设计目标：
 * - 主文件只负责把当前页面、入口配置和表单状态聚合起来
 * - 具体字段 UI 拆到 5 个小节组件，避免继续堆积在一个超长组件里
 * - 写回时优先生成新的 grouped page config，同时保留旧平铺字段双写兼容
 * - 页面运行态权限继续挂在 runtime 小节，避免再单独开一条全局权限面板链路
 */

import type { PageNode } from "@/editor-core/document/types";
import type { PageRole, RouteMode, ViewportPreset } from "./page-inspector/page-inspector-types";
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { VIEW_PRESETS } from "@/constants";
import { useEditorStore } from "@/stores/editor-store";
import PageIdentitySection from "./page-inspector/PageIdentitySection.vue";
import PageRouteSection from "./page-inspector/PageRouteSection.vue";
import PageRuntimeAccessSection from "./page-inspector/PageRuntimeAccessSection.vue";
import PageRuntimeSection from "./page-inspector/PageRuntimeSection.vue";
import PageViewportSection from "./page-inspector/PageViewportSection.vue";
import PageVisualSection from "./page-inspector/PageVisualSection.vue";
import {
  buildPageConfigPatch,
  createDefaultPageInspectorFormState,
  hydratePageInspectorForm,
  mergePageConfigPatch,
} from "./page-inspector-config";
import { resolveRouteByRole } from "./page-inspector/page-inspector-route";
import {
  canEditRuntimeConstraint,
  getPageInspectorSections,
  type PageInspectorSectionKey,
} from "./page-inspector-sections";

interface EntryConfigLike {
  homePageId?: string | null;
  loginPageId?: string | null;
  logoutPageId?: string | null;
}

interface PageRecordLike extends PageNode {
  parentId?: string | null;
  type?: string;
  title?: string;
}

const PAGE_NAME_DOTS_RE = /^\.+$/;
const PAGE_NAME_FORBIDDEN_RE = /[/?#\\%]/;
const PRESET_VIEWPORT_SIZES: Record<
  Exclude<ViewportPreset, "custom">,
  { width: number; height: number }
> = VIEW_PRESETS.reduce(
  (acc, preset) => {
    acc[preset.key as Exclude<ViewportPreset, "custom">] = {
      width: preset.width,
      height: preset.height,
    };
    return acc;
  },
  {} as Record<Exclude<ViewportPreset, "custom">, { width: number; height: number }>,
);

const editorStore = useEditorStore();
const editorRefs = storeToRefs(editorStore);
const { currentPage, currentPageId, pages, doc } = editorRefs;
const rawRuntimeRoles = (editorRefs as unknown as { runtimeRoles?: { value: any[] } }).runtimeRoles;
const { t, locale } = useI18n();

const form = reactive(createDefaultPageInspectorFormState());

function showWarningMessage(message: string): void {
  ElMessage.warning(message as never);
}

function showErrorMessage(message: string): void {
  ElMessage.error(message as never);
}

function getSectionTitle(key: PageInspectorSectionKey): string {
  return getPageInspectorSections().find((section) => section.key === key)?.title || "";
}

function hasControlChars(value: string): boolean {
  for (let i = 0; i < value.length; i += 1) {
    const code = value.charCodeAt(i);
    if ((code >= 0 && code <= 31) || code === 127) {
      return true;
    }
  }
  return false;
}

function validatePageName(value: string): { valid: boolean; message: string } {
  const name = String(value || "").trim();
  if (!name) {
    return { valid: false, message: t("pageInspector.messages.nameRequired") };
  }
  if (PAGE_NAME_DOTS_RE.test(name)) {
    return { valid: false, message: t("pageInspector.messages.nameDotsOnly") };
  }
  if (PAGE_NAME_FORBIDDEN_RE.test(name)) {
    return { valid: false, message: t("pageInspector.messages.nameForbiddenChars") };
  }
  if (hasControlChars(name)) {
    return { valid: false, message: t("pageInspector.messages.nameControlChars") };
  }
  return { valid: true, message: "" };
}

function getSystemRoleLabel(role: Exclude<PageRole, "normal">): string {
  return t(`pageInspector.roles.${role}`);
}

function toPathSegment(value: string): string {
  return (
    String(value || "")
      .trim()
      .replace(/\s+/g, "-")
      .replace(/[/?#\\]+/g, "-") || "page"
  );
}

/**
 * 递归拼接文件夹链路，确保页面自动路由能反映页面树中的父级目录。
 */
function getFolderPathSegments(parentId: string | null | undefined): string[] {
  if (!parentId) return [];
  const folder = (pages.value as PageRecordLike[]).find(
    (page) => page.id === parentId && page.type === "folder",
  );
  if (!folder) return [];
  return [
    ...getFolderPathSegments(folder.parentId || null),
    toPathSegment(folder.name || folder.title || folder.id),
  ];
}

function getParentRoutePath(page: PageRecordLike | null | undefined): string {
  if (!page) return "";
  const segments = getFolderPathSegments(page.parentId || null);
  return segments.length ? `/${segments.join("/")}` : "";
}

function getCurrentPageRecord(): PageRecordLike | null {
  if (!currentPageId.value) return null;
  return (pages.value as PageRecordLike[]).find((item) => item.id === currentPageId.value) || null;
}

function getFixedRole(page: PageRecordLike | null | undefined): PageRole | null {
  if (!page || !doc.value) return null;
  const entry = doc.value.entry as EntryConfigLike | undefined;
  if (entry?.homePageId === page.id) return "home";
  if (entry?.loginPageId === page.id || page.path === "/login") return "login";
  if (entry?.logoutPageId === page.id || page.path === "/logout") return "logout";
  return null;
}

const pageRole = computed<PageRole>(
  () =>
    getFixedRole(getCurrentPageRecord() || (currentPage.value as PageRecordLike | null)) ||
    "normal",
);
const isSystemPage = computed(() => pageRole.value !== "normal");
const canEditConstraintOptions = computed(() => canEditRuntimeConstraint(form.autoFit));
const availableRuntimeRoles = computed<any[]>(() => {
  const value = rawRuntimeRoles?.value;
  return Array.isArray(value) ? value : [];
});

function isNameUnique(name: string, excludeId: string): boolean {
  const lowerName = name.trim().toLowerCase();
  return !(pages.value as PageRecordLike[]).some(
    (page) => page.id !== excludeId && (page.name || "").trim().toLowerCase() === lowerName,
  );
}

function syncForm(page: PageRecordLike | null | undefined): void {
  const pageFromList = getCurrentPageRecord();
  const displayPage = pageFromList || page;
  if (!displayPage) {
    Object.assign(form, createDefaultPageInspectorFormState());
    return;
  }

  const role = getFixedRole(displayPage) || "normal";
  const name =
    role === "normal" ? pageFromList?.name || page?.name || "" : getSystemRoleLabel(role);
  const parentRoutePath = getParentRoutePath(displayPage);
  const nextForm = hydratePageInspectorForm({
    name,
    role,
    routePath: displayPage.path,
    parentRoutePath,
    config: page?.config ?? null,
  });
  nextForm.viewportPreset = resolveViewportPresetBySize(nextForm.width, nextForm.height);

  const resolvedRoute = resolveRouteByRole({
    role: nextForm.role,
    name: nextForm.name,
    routeMode: nextForm.routeMode,
    routePath: nextForm.routePath,
    routeSlug: nextForm.routeSlug,
    parentRoutePath: nextForm.parentRoutePath,
  });

  Object.assign(form, nextForm, resolvedRoute, {
    openMode: role === "normal" ? nextForm.openMode : "replace",
    permissionSummary: nextForm.permissionSummary,
  });
}

function resolveViewportPresetBySize(width: number, height: number): ViewportPreset {
  const matched = VIEW_PRESETS.find((preset) => preset.width === width && preset.height === height);
  return (matched?.key as ViewportPreset | undefined) || "custom";
}

watch(
  [() => currentPage.value, () => currentPageId.value, () => pages.value],
  () => {
    syncForm(currentPage.value as PageRecordLike | null);
  },
  { immediate: true },
);

watch(
  () => locale.value,
  () => {
    syncForm(currentPage.value as PageRecordLike | null);
  },
);

async function persistCurrentPagePath(nextPath: string): Promise<void> {
  const page = currentPage.value;
  if (!page) return;
  try {
    await editorStore.renamePage(page.id, page.name, nextPath);
  } catch {
    editorStore.updateCurrentPage({ path: nextPath });
  }
}

function updateCurrentPageConfig(): void {
  if (!currentPage.value) return;
  const configPatch = buildPageConfigPatch({
    title: form.title,
    description: form.description,
    role: form.role,
    routeMode: form.routeMode,
    routePath: form.routePath,
    routeSlug: form.routeSlug,
    viewportPreset: form.viewportPreset,
    width: form.width,
    height: form.height,
    autoFit: form.autoFit,
    lockAspectRatio: form.lockAspectRatio,
    minWidth: form.minWidth,
    minHeight: form.minHeight,
    overflowMode: form.overflowMode,
    backgroundType: form.backgroundType,
    backgroundValue: form.backgroundValue,
    backgroundSize: form.backgroundSize,
    backgroundPosition: form.backgroundPosition,
    backgroundRepeat: form.backgroundRepeat,
    transitionType: form.transitionType,
    openMode: isSystemPage.value ? "replace" : form.openMode,
    popupWidth: form.popupWidth,
    popupHeight: form.popupHeight,
    popupCenter: form.popupCenter,
    popupMaskClosable: form.popupMaskClosable,
    permissionSummary: form.permissionSummary,
    runtimeAccessEnabled: Boolean(form.runtimeAccessEnabled),
    runtimeAccessAllowedRoles: form.runtimeAccessAllowedRoles || [],
    runtimePermissionSchemes: form.runtimePermissionSchemes || [],
    cacheMode: form.cacheMode,
    preloadMode: form.preloadMode,
  });
  if (!configPatch.autoFit) {
    form.lockAspectRatio = false;
    form.minWidth = 0;
    form.minHeight = 0;
  }
  if (isSystemPage.value) {
    form.openMode = "replace";
  }
  const nextConfig = mergePageConfigPatch(currentPage.value.config, configPatch);
  editorStore.updateCurrentPage({
    path: form.routePath,
    config: nextConfig,
  });
}

async function handleNameUpdate(): Promise<void> {
  const page = currentPage.value;
  if (!page || isSystemPage.value) {
    syncForm(page as PageRecordLike | null);
    return;
  }
  const name = form.name.trim();
  const validation = validatePageName(name);
  if (!validation.valid) {
    showWarningMessage(validation.message);
    syncForm(page as PageRecordLike);
    return;
  }
  if (!isNameUnique(name, page.id)) {
    showWarningMessage(t("pageInspector.messages.duplicatedName"));
    syncForm(page as PageRecordLike);
    return;
  }

  const nextRoute = resolveRouteByRole({
    role: form.role,
    name,
    routeMode: form.routeMode,
    routePath: form.routePath,
    routeSlug: form.routeSlug,
    parentRoutePath: form.parentRoutePath,
  });

  try {
    await editorStore.renamePage(page.id, name, nextRoute.routePath);
    Object.assign(form, nextRoute, { name });
    updateCurrentPageConfig();
  } catch {
    showErrorMessage(t("pageInspector.messages.renameFailed"));
    syncForm(page as PageRecordLike);
  }
}

async function handleRouteModeChange(nextMode: RouteMode): Promise<void> {
  form.routeMode = nextMode;
  if (nextMode === "auto") {
    const nextRoute = resolveRouteByRole({
      role: form.role,
      name: form.name,
      routeMode: "auto",
      routeSlug: form.routeSlug,
      parentRoutePath: form.parentRoutePath,
    });
    Object.assign(form, nextRoute);
    await persistCurrentPagePath(nextRoute.routePath);
  }
  updateCurrentPageConfig();
}

async function handleRoutePathBlur(): Promise<void> {
  if (isSystemPage.value || form.routeMode !== "manual") {
    return;
  }
  const path = String(form.routePath || "").trim();
  if (!path) {
    showWarningMessage(t("pageInspector.messages.routePathRequired"));
    syncForm(currentPage.value as PageRecordLike | null);
    return;
  }
  if (!path.startsWith("/")) {
    showWarningMessage(t("pageInspector.messages.routePathInvalid"));
    syncForm(currentPage.value as PageRecordLike | null);
    return;
  }
  const nextRoute = resolveRouteByRole({
    role: form.role,
    name: form.name,
    routeMode: "manual",
    routePath: path,
    routeSlug: form.routeSlug,
    parentRoutePath: form.parentRoutePath,
  });
  Object.assign(form, nextRoute);
  await persistCurrentPagePath(nextRoute.routePath);
  updateCurrentPageConfig();
}

async function handleRouteSlugBlur(): Promise<void> {
  if (isSystemPage.value) return;
  if (!String(form.routeSlug || "").trim()) {
    showWarningMessage(t("pageInspector.messages.routeSlugRequired"));
    syncForm(currentPage.value as PageRecordLike | null);
    return;
  }
  const nextRoute = resolveRouteByRole({
    role: form.role,
    name: form.name,
    routeMode: form.routeMode,
    routePath: form.routePath,
    routeSlug: form.routeSlug,
    parentRoutePath: form.parentRoutePath,
  });
  Object.assign(form, nextRoute);
  await persistCurrentPagePath(nextRoute.routePath);
  updateCurrentPageConfig();
}

function handlePresetChange(nextPreset: ViewportPreset): void {
  form.viewportPreset = nextPreset;
  if (nextPreset !== "custom") {
    const preset = PRESET_VIEWPORT_SIZES[nextPreset];
    form.width = preset.width;
    form.height = preset.height;
  }
  updateCurrentPageConfig();
}

function handleConfigUpdate(): void {
  form.viewportPreset = resolveViewportPresetBySize(form.width, form.height);
  updateCurrentPageConfig();
}
</script>

<template>
  <div class="page-inspector-panel">
    <div class="page-group">
      <div class="section-title">{{ getSectionTitle("identity") }}</div>
      <PageIdentitySection
        :form="form"
        :is-system-page="isSystemPage"
        :page-id="currentPageId || ''"
        @update-name="handleNameUpdate"
        @update-config="handleConfigUpdate"
      />
    </div>

    <div class="page-group">
      <div class="section-title">{{ getSectionTitle("route") }}</div>
      <PageRouteSection
        :form="form"
        :is-system-page="isSystemPage"
        @route-mode-change="handleRouteModeChange"
        @route-path-blur="handleRoutePathBlur"
        @route-slug-blur="handleRouteSlugBlur"
      />
    </div>

    <div class="page-group">
      <div class="section-title">{{ getSectionTitle("viewport") }}</div>
      <PageViewportSection
        :form="form"
        :can-edit-constraint-options="canEditConstraintOptions"
        @preset-change="handlePresetChange"
        @update-config="handleConfigUpdate"
      />
    </div>

    <div class="page-group">
      <div class="section-title">{{ getSectionTitle("visual") }}</div>
      <PageVisualSection :form="form" @update-config="handleConfigUpdate" />
    </div>

    <div class="page-group">
      <div class="section-title">{{ getSectionTitle("runtime") }}</div>
      <PageRuntimeSection
        :form="form"
        :is-system-page="isSystemPage"
        @update-config="handleConfigUpdate"
      />
    </div>

    <div class="page-group">
      <div class="section-title">{{ getSectionTitle("runtimeAccess") }}</div>
      <PageRuntimeAccessSection
        :form="form"
        :runtime-roles="availableRuntimeRoles"
        @update-config="handleConfigUpdate"
      />
    </div>
  </div>
</template>

<style scoped>
.page-inspector-panel {
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
}
</style>
