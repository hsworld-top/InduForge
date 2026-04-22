/**
 * 页面属性面板配置构建工具
 */
import type { BackgroundConfig, PageConfig, PageNode, RolePermission } from "@/editor-core/document/types";
import type {
  BackgroundRepeat,
  BackgroundSize,
  CacheMode,
  OpenMode,
  OverflowMode,
  PageInspectorFormState,
  PageRole,
  PreloadMode,
  RouteMode,
  TransitionType,
  ViewportPreset,
} from "./page-inspector/page-inspector-types";
import { sanitizeRoleGrant } from "@/ui/shared/permissions/role-grant-summary";

export type WindowStyle = "popup" | "cover" | "replace";
export type BackgroundKind = BackgroundConfig["kind"];

export interface PageInspectorConfigInput {
  title: string;
  description: string;
  role: PageRole;
  routeMode: RouteMode;
  routePath: string;
  routeSlug: string;
  viewportPreset: ViewportPreset;
  width: number;
  height: number;
  autoFit: boolean;
  lockAspectRatio: boolean;
  minWidth: number;
  minHeight: number;
  overflowMode: OverflowMode;
  backgroundType: BackgroundKind;
  backgroundValue: string;
  backgroundSize: BackgroundSize;
  backgroundPosition: string;
  backgroundRepeat: BackgroundRepeat;
  transitionType: TransitionType;
  openMode: OpenMode;
  popupWidth: number;
  popupHeight: number;
  popupCenter: boolean;
  popupMaskClosable: boolean;
  permissionSummary: string;
  pageViewPermission: RolePermission | undefined;
  cacheMode: CacheMode;
  preloadMode: PreloadMode;
}

export interface PageInspectorConfigPatch extends Partial<PageNode["config"]> {
  meta?: {
    title: string;
    description: string;
  };
  route?: {
    mode: RouteMode;
    path: string;
    slug: string;
  };
  viewport?: {
    preset: ViewportPreset;
    width: number;
    height: number;
    autoFit: boolean;
    lockAspectRatio: boolean;
    minWidth: number;
    minHeight: number;
    overflowMode: OverflowMode;
  };
  runtime?: {
    openMode: WindowStyle;
    popup: {
      width: number;
      height: number;
      center: boolean;
      maskClosable: boolean;
    };
    permission: {
      summary: string;
    };
    cacheMode: CacheMode;
    preloadMode: PreloadMode;
  };
  transition?: {
    type: TransitionType;
  };
  description?: string;
  lockAspectRatio?: boolean;
  enableMinSize?: boolean;
  windowStyle?: WindowStyle;
  permissionDesc?: string;
}

const DEFAULT_WIDTH = 1920;
const DEFAULT_HEIGHT = 1080;
const DEFAULT_POPUP_WIDTH = 960;
const DEFAULT_POPUP_HEIGHT = 540;
const DEFAULT_BACKGROUND_COLOR = "#ffffff";
const DEFAULT_PERMISSION_DESC = "0item";
const DEFAULT_BACKGROUND_POSITION = "center";
const DEFAULT_ROUTE_PATH = "/";

/**
 * 规范化窗口类型（兼容旧值 normal）
 * @param {unknown} value - 原始窗口类型
 * @returns {WindowStyle} 规范化后的窗口类型
 */
export function normalizeWindowStyle(value: unknown): WindowStyle {
  if (value === "popup" || value === "cover" || value === "replace") {
    return value;
  }
  if (value === "normal") {
    return "replace";
  }
  return "cover";
}

/**
 * 规范化背景类型
 * @param {unknown} value - 原始背景类型
 * @returns {BackgroundKind} 规范化后的背景类型
 */
function normalizeBackgroundKind(value: unknown): BackgroundKind {
  if (value === "color" || value === "image" || value === "gradient") {
    return value;
  }
  return "color";
}

function normalizeRouteMode(value: unknown): RouteMode {
  return value === "manual" ? "manual" : "auto";
}

function normalizeViewportPreset(value: unknown): ViewportPreset {
  if (
    value === "bigscreen"
    || value === "pc"
    || value === "tablet"
    || value === "phoneLandscape"
    || value === "phonePortrait"
    || value === "custom"
  ) {
    return value;
  }
  return "pc";
}

function normalizeOverflowMode(value: unknown): OverflowMode {
  if (value === "hidden" || value === "scroll") {
    return value;
  }
  return "auto";
}

function normalizeBackgroundSize(value: unknown): BackgroundSize {
  if (value === "contain" || value === "stretch" || value === "auto") {
    return value;
  }
  return "cover";
}

function normalizeBackgroundPosition(value: unknown): string {
  const text = String(value || "").trim();
  return text || DEFAULT_BACKGROUND_POSITION;
}

function normalizeBackgroundRepeat(value: unknown): BackgroundRepeat {
  if (value === "repeat" || value === "repeat-x" || value === "repeat-y") {
    return value;
  }
  return "no-repeat";
}

function normalizeTransitionType(value: unknown): TransitionType {
  if (value === "fade" || value === "slide" || value === "zoom") {
    return value;
  }
  return "none";
}

function normalizeCacheMode(value: unknown): CacheMode {
  if (value === "cache" || value === "no-cache") {
    return value;
  }
  return "default";
}

function normalizePreloadMode(value: unknown): PreloadMode {
  return value === "eager" ? "eager" : "lazy";
}

function normalizeOptionalNumber(value: unknown): number {
  const numeric = Number(value);
  return Number.isFinite(numeric) && numeric > 0 ? numeric : 0;
}

function normalizePath(value: unknown): string {
  const text = String(value || "").trim();
  if (!text) {
    return DEFAULT_ROUTE_PATH;
  }
  return text.startsWith("/") ? text : `/${text}`;
}

function normalizeSlug(value: unknown): string {
  return String(value || "").trim();
}

function normalizeRole(value: unknown): PageRole {
  if (value === "home" || value === "login" || value === "logout") {
    return value;
  }
  return "normal";
}

function normalizeOpenMode(value: unknown): OpenMode {
  if (value === "popup" || value === "cover" || value === "replace" || value === "normal") {
    return value;
  }
  return "cover";
}

/**
 * 归一化数字输入
 * @param {unknown} value - 原始数值
 * @param {number} fallback - 回退值
 * @returns {number} 归一化后的数字
 */
function normalizeNumber(value: unknown, fallback: number): number {
  const numeric = Number(value);
  return Number.isFinite(numeric) && numeric > 0 ? numeric : fallback;
}

export function createDefaultPageInspectorFormState(): PageInspectorFormState {
  return {
    name: "",
    title: "",
    description: "",
    role: "normal",
    routeMode: "auto",
    routePath: DEFAULT_ROUTE_PATH,
    routeSlug: "",
    parentRoutePath: "",
    viewportPreset: "pc",
    width: DEFAULT_WIDTH,
    height: DEFAULT_HEIGHT,
    autoFit: true,
    lockAspectRatio: false,
    minWidth: 0,
    minHeight: 0,
    overflowMode: "auto",
    backgroundType: "color",
    backgroundValue: DEFAULT_BACKGROUND_COLOR,
    backgroundSize: "cover",
    backgroundPosition: DEFAULT_BACKGROUND_POSITION,
    backgroundRepeat: "no-repeat",
    transitionType: "none",
    openMode: "cover",
    popupWidth: DEFAULT_POPUP_WIDTH,
    popupHeight: DEFAULT_POPUP_HEIGHT,
    popupCenter: true,
    popupMaskClosable: true,
    permissionSummary: DEFAULT_PERMISSION_DESC,
    pageViewPermission: undefined,
    cacheMode: "default",
    preloadMode: "lazy",
  };
}

export function hydratePageInspectorForm(input: {
  name?: string;
  role?: PageRole;
  routePath?: string;
  parentRoutePath?: string;
  config?: Partial<PageNode["config"]> | null;
}): PageInspectorFormState {
  const base = createDefaultPageInspectorFormState();
  const config = input.config || {};
  const viewport = (config.viewport || {}) as Partial<NonNullable<PageNode["config"]["viewport"]>>;
  const route = (config.route || {}) as Partial<NonNullable<PageNode["config"]["route"]>>;
  const runtime = (config.runtime || {}) as Partial<NonNullable<PageNode["config"]["runtime"]>>;
  const popup =
    (runtime.popup || {}) as Partial<NonNullable<NonNullable<PageNode["config"]["runtime"]>["popup"]>>;
  const permission = (runtime.permission || {}) as Partial<
    NonNullable<NonNullable<PageNode["config"]["runtime"]>["permission"]>
  >;
  const background = (config.background || {}) as Partial<NonNullable<PageNode["config"]["background"]>>;
  const transition = (config.transition || {}) as Partial<
    NonNullable<PageNode["config"]["transition"]>
  >;

  const width = normalizeNumber(viewport.width ?? config.width, DEFAULT_WIDTH);
  const height = normalizeNumber(viewport.height ?? config.height, DEFAULT_HEIGHT);
  const autoFit = Boolean(viewport.autoFit ?? config.autoFit ?? base.autoFit);
  const lockAspectRatio = Boolean(viewport.lockAspectRatio ?? config.lockAspectRatio ?? false);
  const pageViewPermission = sanitizeRoleGrant(config.runtimePermissions?.pageView);

  return {
    ...base,
    name: String(input.name || ""),
    title: String(config.meta?.title || ""),
    description: String(config.meta?.description ?? config.description ?? ""),
    role: normalizeRole(input.role),
    routeMode: normalizeRouteMode(route.mode),
    routePath: normalizePath(route.path ?? input.routePath),
    routeSlug: normalizeSlug(route.slug),
    parentRoutePath: String(input.parentRoutePath || ""),
    viewportPreset: normalizeViewportPreset(viewport.preset),
    width,
    height,
    autoFit,
    lockAspectRatio: autoFit && lockAspectRatio,
    /**
     * 旧页面只提供 enableMinSize 开关时，不擅自推导具体最小尺寸。
     * 表单层保持 0，避免把历史开关误写成新的硬编码尺寸。
     */
    minWidth: autoFit ? normalizeOptionalNumber(viewport.minWidth) : 0,
    minHeight: autoFit ? normalizeOptionalNumber(viewport.minHeight) : 0,
    overflowMode: normalizeOverflowMode(viewport.overflowMode),
    backgroundType: normalizeBackgroundKind(background.kind),
    backgroundValue: String(background.value || DEFAULT_BACKGROUND_COLOR),
    backgroundSize: normalizeBackgroundSize(background.size),
    backgroundPosition: normalizeBackgroundPosition(background.position),
    backgroundRepeat: normalizeBackgroundRepeat(background.repeat),
    transitionType: normalizeTransitionType(transition.type),
    openMode: normalizeOpenMode(runtime.openMode ?? config.windowStyle),
    popupWidth: normalizeNumber(popup.width, DEFAULT_POPUP_WIDTH),
    popupHeight: normalizeNumber(popup.height, DEFAULT_POPUP_HEIGHT),
    popupCenter: popup.center ?? true,
    popupMaskClosable: popup.maskClosable ?? true,
    permissionSummary: pageViewPermission
      ? "已配置页面访问权限"
      : String(permission.summary ?? config.permissionDesc ?? DEFAULT_PERMISSION_DESC),
    pageViewPermission,
    cacheMode: normalizeCacheMode(runtime.cacheMode),
    preloadMode: normalizePreloadMode(runtime.preloadMode),
  };
}

/**
 * 构建页面配置补丁（仅包含保留字段）
 * @param {PageInspectorConfigInput} input - 表单输入
 * @returns {PageInspectorConfigPatch} 页面配置补丁
 */
export function buildPageConfigPatch(input: PageInspectorConfigInput): PageInspectorConfigPatch {
  const autoFit = Boolean(input.autoFit);
  const normalizedWindowStyle = normalizeWindowStyle(input.openMode);
  const normalizedPermissionSummary = String(input.permissionSummary || DEFAULT_PERMISSION_DESC);
  const backgroundKind = normalizeBackgroundKind(input.backgroundType);
  const pageViewPermission = sanitizeRoleGrant(input.pageViewPermission);
  const patch: PageInspectorConfigPatch = {
    meta: {
      title: String(input.title || ""),
      description: String(input.description || ""),
    },
    route: {
      mode: normalizeRouteMode(input.routeMode),
      path: normalizePath(input.routePath),
      slug: normalizeSlug(input.routeSlug),
    },
    viewport: {
      preset: normalizeViewportPreset(input.viewportPreset),
      width: normalizeNumber(input.width, DEFAULT_WIDTH),
      height: normalizeNumber(input.height, DEFAULT_HEIGHT),
      autoFit,
      lockAspectRatio: autoFit && Boolean(input.lockAspectRatio),
      minWidth: autoFit ? normalizeOptionalNumber(input.minWidth) : 0,
      minHeight: autoFit ? normalizeOptionalNumber(input.minHeight) : 0,
      overflowMode: normalizeOverflowMode(input.overflowMode),
    },
    runtime: {
      openMode: normalizedWindowStyle,
      popup: {
        width: normalizeNumber(input.popupWidth, DEFAULT_POPUP_WIDTH),
        height: normalizeNumber(input.popupHeight, DEFAULT_POPUP_HEIGHT),
        center: Boolean(input.popupCenter),
        maskClosable: Boolean(input.popupMaskClosable),
      },
      permission: {
        summary: pageViewPermission ? "已配置页面访问权限" : normalizedPermissionSummary,
      },
      cacheMode: normalizeCacheMode(input.cacheMode),
      preloadMode: normalizePreloadMode(input.preloadMode),
    },
    transition: {
      type: normalizeTransitionType(input.transitionType),
    },
    description: String(input.description || ""),
    width: normalizeNumber(input.width, DEFAULT_WIDTH),
    height: normalizeNumber(input.height, DEFAULT_HEIGHT),
    autoFit,
    lockAspectRatio: autoFit && Boolean(input.lockAspectRatio),
    enableMinSize:
      autoFit
      && (normalizeOptionalNumber(input.minWidth) > 0 || normalizeOptionalNumber(input.minHeight) > 0),
    windowStyle: normalizedWindowStyle,
    permissionDesc: pageViewPermission ? "已配置页面访问权限" : normalizedPermissionSummary,
    background: {
      kind: backgroundKind,
      value: String(input.backgroundValue || DEFAULT_BACKGROUND_COLOR),
      size: normalizeBackgroundSize(input.backgroundSize),
      position: normalizeBackgroundPosition(input.backgroundPosition),
      repeat: normalizeBackgroundRepeat(input.backgroundRepeat),
    },
  };

  if (pageViewPermission) {
    patch.runtimePermissions = {
      ...(patch.runtimePermissions || {}),
      pageView: pageViewPermission,
    };
  }

  return patch;
}

/**
 * 合并页面配置补丁。对运行态权限使用显式移除语义，避免旧的 pageView 残留。
 * @param {PageConfig | undefined} baseConfig - 当前页面配置
 * @param {PageInspectorConfigPatch} patch - 页面属性面板构建出的补丁
 * @returns {PageConfig} 合并后的配置
 */
export function mergePageConfigPatch(
  baseConfig: PageConfig | undefined,
  patch: PageInspectorConfigPatch,
  options: {
    clearPageViewPermission?: boolean;
  } = {},
): PageConfig {
  const nextConfig = {
    ...(baseConfig || {}),
    ...patch,
    meta: {
      ...(baseConfig?.meta || {}),
      ...(patch.meta || {}),
    },
    route: {
      ...(baseConfig?.route || {}),
      ...(patch.route || {}),
    },
    viewport: {
      ...(baseConfig?.viewport || {}),
      ...(patch.viewport || {}),
    },
    runtime: {
      ...(baseConfig?.runtime || {}),
      ...(patch.runtime || {}),
      popup: {
        ...(baseConfig?.runtime?.popup || {}),
        ...(patch.runtime?.popup || {}),
      },
      permission: {
        ...(baseConfig?.runtime?.permission || {}),
        ...(patch.runtime?.permission || {}),
      },
    },
    transition: {
      ...(baseConfig?.transition || {}),
      ...(patch.transition || {}),
    },
    background: {
      ...(baseConfig?.background || {}),
      ...(patch.background || {}),
    },
  } as PageConfig;

  if ("runtimePermissions" in patch) {
    if (patch.runtimePermissions?.pageView) {
      nextConfig.runtimePermissions = {
        ...(baseConfig?.runtimePermissions || {}),
        ...patch.runtimePermissions,
      };
    } else {
      const remainingPermissions = { ...(baseConfig?.runtimePermissions || {}) };
      delete remainingPermissions.pageView;
      if (Object.keys(remainingPermissions).length > 0) {
        nextConfig.runtimePermissions = remainingPermissions;
      } else {
        delete nextConfig.runtimePermissions;
      }
    }
  } else if (options.clearPageViewPermission) {
    const remainingPermissions = { ...(baseConfig?.runtimePermissions || {}) };
    delete remainingPermissions.pageView;
    if (Object.keys(remainingPermissions).length > 0) {
      nextConfig.runtimePermissions = remainingPermissions;
    } else {
      delete nextConfig.runtimePermissions;
    }
  }

  return nextConfig;
}
