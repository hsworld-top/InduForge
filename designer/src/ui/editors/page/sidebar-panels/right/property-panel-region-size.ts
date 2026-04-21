/**
 * PropertyPanel 区域尺寸相关纯函数。
 * 只负责尺寸解析、上限计算、最小尺寸推导，不处理节点更新副作用。
 */

import { i18n } from "@/i18n";

export interface RegionSizeValue {
  value: string;
  unit: string;
}

export interface RegionSizeItemLike {
  key?: string;
}

export interface RegionSizeAbsolutePosLike {
  w?: number;
  h?: number;
}

export interface RegionSizeNodeLike {
  id?: string;
  type?: string;
  style?: Record<string, unknown>;
  props?: Record<string, unknown>;
  children?: string[];
  absolutePos?: RegionSizeAbsolutePosLike;
}

export interface RegionSizeRectLike {
  width?: number | undefined;
  height?: number | undefined;
}

export interface RegionSizeState {
  headerHeight: RegionSizeValue;
  asideWidth: RegionSizeValue;
  footerHeight: RegionSizeValue;
}

export const regionSizeDefaults: Record<string, string> = {
  headerHeight: "60px",
  asideWidth: "200px",
  footerHeight: "60px",
};
const SIZE_VALUE_RE = /^([0-9.]+)(px|%)?$/;

/**
 * 获取区域默认尺寸
 * @param {string} propName - 属性名
 * @returns {string | undefined}
 */
export function getRegionDefaultValue(propName: string): string | undefined {
  return regionSizeDefaults[propName];
}

/**
 * 解析尺寸字符串
 * @param {string | number | undefined} value - 原始值
 * @returns {RegionSizeValue}
 */
export function parseSize(value: string | number | undefined): RegionSizeValue {
  if (!value || value === "auto") {
    return { value: "", unit: "auto" };
  }
  const str = String(value);
  const match = str.match(SIZE_VALUE_RE);
  if (match) {
    return { value: match[1] ?? "", unit: match[2] || "px" };
  }
  return { value: "", unit: "auto" };
}

/**
 * 将尺寸转换为数字
 * @param {string | number | undefined} value - 原始值
 * @returns {number | undefined}
 */
export function parseSizeToNumber(value: string | number | undefined): number | undefined {
  const parsed = parseSize(value);
  if (!parsed.value || parsed.unit !== "px") return undefined;
  const num = Number.parseFloat(parsed.value);
  return Number.isFinite(num) ? num : undefined;
}

/**
 * 获取区域尺寸文案
 * @param {RegionSizeItemLike | null | undefined} item - 区域项
 * @returns {string}
 */
export function getRegionSizeText(item: RegionSizeItemLike | null | undefined): string {
  if (!item) return "";
  if (item.key === "aside") return i18n.global.t("componentManifest.labels.width");
  if (item.key === "header" || item.key === "footer") {
    return i18n.global.t("componentManifest.labels.height");
  }
  return "";
}

/**
 * 解析容器最大尺寸
 * @param {RegionSizeNodeLike | null | undefined} currentElement - 当前节点
 * @param {RegionSizeNodeLike | null | undefined} liveNode - 运行态节点
 * @param {RegionSizeRectLike | null | undefined} containerRect - DOM 尺寸
 * @returns {RegionSizeRectLike}
 */
export function resolveContainerMaxSize(
  currentElement: RegionSizeNodeLike | null | undefined,
  liveNode: RegionSizeNodeLike | null | undefined,
  containerRect?: RegionSizeRectLike | null,
): RegionSizeRectLike {
  const el = currentElement;
  if (!el || el.type !== "ElContainer") return {};
  const targetNode = liveNode || el;
  const styleWidth = parseSizeToNumber(targetNode.style?.width as string | number | undefined);
  const styleHeight = parseSizeToNumber(targetNode.style?.height as string | number | undefined);
  const absWidth =
    targetNode.absolutePos && Number.isFinite(targetNode.absolutePos.w)
      ? targetNode.absolutePos.w
      : undefined;
  const absHeight =
    targetNode.absolutePos && Number.isFinite(targetNode.absolutePos.h)
      ? targetNode.absolutePos.h
      : undefined;

  if (containerRect) {
    return {
      width: containerRect.width || styleWidth || absWidth,
      height: containerRect.height || styleHeight || absHeight,
    };
  }

  return {
    width: styleWidth ?? absWidth,
    height: styleHeight ?? absHeight,
  };
}

/**
 * 解析区域尺寸上限
 * @param {string} propName - 属性名
 * @param {RegionSizeNodeLike | null | undefined} currentElement - 当前节点
 * @param {RegionSizeRectLike | null | undefined} maxSize - 容器最大尺寸
 * @returns {number | undefined}
 */
export function resolveRegionMaxValue(
  propName: string,
  currentElement: RegionSizeNodeLike | null | undefined,
  maxSize: RegionSizeRectLike | null | undefined,
): number | undefined {
  const containerProps = currentElement?.props || {};
  const hasHeader = containerProps.showHeader !== false;
  const hasFooter = containerProps.showFooter !== false;
  const hasAside = containerProps.showAside !== false;
  const hasMain = containerProps.showMain !== false;
  const minBodySize = 40;

  const headerHeight =
    parseSizeToNumber(containerProps.headerHeight as string | number | undefined) ??
    parseSizeToNumber(regionSizeDefaults.headerHeight) ??
    60;
  const footerHeight =
    parseSizeToNumber(containerProps.footerHeight as string | number | undefined) ??
    parseSizeToNumber(regionSizeDefaults.footerHeight) ??
    60;
  if (propName === "asideWidth" && maxSize?.width) {
    const bodyMin = hasMain ? minBodySize : 0;
    return Math.max(0, maxSize.width - bodyMin);
  }
  if (propName === "headerHeight" && maxSize?.height) {
    const footer = hasFooter ? footerHeight : 0;
    const body = hasAside || hasMain ? minBodySize : 0;
    return Math.max(0, maxSize.height - footer - body);
  }
  if (propName === "footerHeight" && maxSize?.height) {
    const header = hasHeader ? headerHeight : 0;
    const body = hasAside || hasMain ? minBodySize : 0;
    return Math.max(0, maxSize.height - header - body);
  }
  return undefined;
}

/**
 * 获取区域尺寸上限文案
 * @param {string | null | undefined} propName - 属性名
 * @param {RegionSizeNodeLike | null | undefined} currentElement - 当前节点
 * @param {RegionSizeRectLike | null | undefined} maxSize - 容器尺寸
 * @returns {string}
 */
export function getRegionMaxLabel(
  propName: string | null | undefined,
  currentElement: RegionSizeNodeLike | null | undefined,
  maxSize: RegionSizeRectLike | null | undefined,
): string {
  if (!propName) return "";
  const maxValue = resolveRegionMaxValue(propName, currentElement, maxSize);
  if (!maxValue) return "";
  return `<= ${Math.round(maxValue)}px`;
}

/**
 * 限制区域尺寸输入值
 * @param {string} propName - 属性名
 * @param {string} value - 数值
 * @param {string} unit - 单位
 * @param {RegionSizeNodeLike | null | undefined} currentElement - 当前节点
 * @param {RegionSizeRectLike | null | undefined} maxSize - 容器尺寸
 * @returns {RegionSizeValue}
 */
export function clampRegionSize(
  propName: string,
  value: string,
  unit: string,
  currentElement: RegionSizeNodeLike | null | undefined,
  maxSize: RegionSizeRectLike | null | undefined,
): RegionSizeValue {
  const maxValue = resolveRegionMaxValue(propName, currentElement, maxSize);
  if (unit === "%") {
    const num = Number.parseFloat(value || "0");
    if (!Number.isFinite(num)) return { value, unit };
    if (maxValue !== undefined && (maxSize?.width || maxSize?.height)) {
      const base = propName === "asideWidth" ? maxSize?.width : maxSize?.height;
      if (base) {
        const maxPercent = Math.max(0, (maxValue / base) * 100);
        return {
          value: String(Math.min(num, Math.min(100, maxPercent))),
          unit,
        };
      }
    }
    return { value: String(Math.min(100, Math.max(0, num))), unit };
  }
  if (unit !== "px") return { value, unit };
  const num = Number.parseFloat(value || "0");
  if (!Number.isFinite(num)) return { value, unit };
  if (maxValue !== undefined) {
    return { value: String(Math.min(num, maxValue)), unit };
  }
  return { value, unit };
}

/**
 * 计算 ElContainer 最小尺寸
 * @param {RegionSizeNodeLike | null | undefined} containerNode - 容器节点
 * @param {(id: string) => RegionSizeNodeLike | null | undefined} getNodeById - 节点查询函数
 * @returns {RegionSizeRectLike | null}
 */
export function resolveElContainerMinSize(
  containerNode: RegionSizeNodeLike | null | undefined,
  getNodeById: (id: string) => RegionSizeNodeLike | null | undefined,
): RegionSizeRectLike | null {
  if (!containerNode || containerNode.type !== "ElContainer") return null;
  const children = containerNode.children || [];
  let hasHeader = false;
  let hasFooter = false;
  let hasAside = false;
  let hasMain = false;
  for (const childId of children) {
    const childNode = getNodeById(childId);
    if (!childNode) continue;
    if (childNode.type === "ElHeader") hasHeader = true;
    if (childNode.type === "ElFooter") hasFooter = true;
    if (childNode.type === "ElAside") hasAside = true;
    if (childNode.type === "ElMain") hasMain = true;
  }

  const props = containerNode.props || {};
  if (typeof props.showHeader === "boolean") hasHeader = props.showHeader;
  if (typeof props.showFooter === "boolean") hasFooter = props.showFooter;
  if (typeof props.showAside === "boolean") hasAside = props.showAside;
  if (typeof props.showMain === "boolean") hasMain = props.showMain;

  const headerHeight =
    parseSizeToNumber(props.headerHeight as string | number | undefined) ??
    parseSizeToNumber(regionSizeDefaults.headerHeight) ??
    60;
  const footerHeight =
    parseSizeToNumber(props.footerHeight as string | number | undefined) ??
    parseSizeToNumber(regionSizeDefaults.footerHeight) ??
    60;
  const asideWidth =
    parseSizeToNumber(props.asideWidth as string | number | undefined) ??
    parseSizeToNumber(regionSizeDefaults.asideWidth) ??
    200;
  const minBodySize = 40;

  const hasBody = hasAside || hasMain;
  let minWidth = 0;
  if (hasAside && hasMain) {
    minWidth = asideWidth + minBodySize;
  } else if (hasAside) {
    minWidth = asideWidth;
  } else if (hasMain) {
    minWidth = minBodySize;
  }

  let minHeight = 0;
  if (hasHeader) minHeight += headerHeight;
  if (hasFooter) minHeight += footerHeight;
  if (hasBody) minHeight += minBodySize;

  if (minWidth <= 0 && minHeight <= 0) return null;
  return { width: minWidth, height: minHeight };
}

/**
 * 构建区域尺寸状态
 * @param {Record<string, unknown> | undefined} props - 组件属性
 * @returns {RegionSizeState}
 */
export function buildRegionSizeState(props: Record<string, unknown> | undefined): RegionSizeState {
  const safeProps = props || {};
  return {
    headerHeight: parseSize(
      (safeProps.headerHeight as string | number | undefined) ??
        getRegionDefaultValue("headerHeight"),
    ),
    asideWidth: parseSize(
      (safeProps.asideWidth as string | number | undefined) ?? getRegionDefaultValue("asideWidth"),
    ),
    footerHeight: parseSize(
      (safeProps.footerHeight as string | number | undefined) ??
        getRegionDefaultValue("footerHeight"),
    ),
  };
}
