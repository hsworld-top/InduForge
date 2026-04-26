import type {
  PagePermissionScheme,
  RuntimeRoleRef,
} from "@/editor-core/document/types";

/**
 * 页面属性面板的 UI 层类型定义
 * 统一约束页面角色、路由模式和整张表单的字段结构，避免各小节组件重复声明。
 */
export type PageRole = "normal" | "home" | "login" | "logout";
export type RouteMode = "auto" | "manual";
export type ViewportPreset =
  | "bigscreen"
  | "pc"
  | "tablet"
  | "phoneLandscape"
  | "phonePortrait"
  | "custom";
export type OverflowMode = "auto" | "hidden" | "scroll";
export type BackgroundType = "color" | "image" | "gradient";
export type BackgroundSize = "cover" | "contain" | "stretch" | "auto";
export type BackgroundRepeat = "no-repeat" | "repeat" | "repeat-x" | "repeat-y";
export type TransitionType = "none" | "fade" | "slide" | "zoom";
export type OpenMode = "replace" | "cover" | "popup" | "normal";
export type CacheMode = "default" | "cache" | "no-cache";
export type PreloadMode = "lazy" | "eager";

export interface PageInspectorFormState {
  name: string;
  title: string;
  description: string;
  role: PageRole;
  routeMode: RouteMode;
  routePath: string;
  routeSlug: string;
  parentRoutePath: string;
  viewportPreset: ViewportPreset;
  width: number;
  height: number;
  autoFit: boolean;
  lockAspectRatio: boolean;
  minWidth: number;
  minHeight: number;
  overflowMode: OverflowMode;
  backgroundType: BackgroundType;
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
  runtimeAccessEnabled?: boolean;
  runtimeAccessAllowedRoles?: RuntimeRoleRef[];
  runtimePermissionSchemes?: PagePermissionScheme[];
  cacheMode: CacheMode;
  preloadMode: PreloadMode;
}

export interface PageInspectorSelectOption<T extends string = string> {
  label: string;
  value: T;
}
