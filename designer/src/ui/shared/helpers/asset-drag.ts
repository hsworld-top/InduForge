/**
 * 资源拖拽协议与组件映射工具。
 * 统一资源面板与画布落点之间的数据格式，避免多处解析分叉。
 */

export const DESIGNER_ASSET_DRAG_MIME = "application/x-designer-asset";

export interface DesignerAssetDragPayload {
  id: string;
  name?: string;
  displayName?: string;
  url?: string;
  thumbnailUrl?: string;
  type?: string;
  mimeType?: string;
  ext?: string;
  size?: number;
}

export type DesignerAssetComponentType = "Image" | "Video" | "DownloadLink";

interface AssetLike {
  id?: string;
  name?: string;
  displayName?: string;
  originalName?: string;
  url?: string;
  thumbnailUrl?: string;
  type?: string;
  mimeType?: string;
  ext?: string;
  size?: number | string | null;
}

/**
 * 规范化资源显示名。
 * @param {DesignerAssetDragPayload | null | undefined} asset - 资源对象
 * @returns {string}
 */
export function resolveAssetDisplayName(asset: DesignerAssetDragPayload | null | undefined): string {
  const raw = asset?.displayName || asset?.name || "";
  const value = String(raw || "").trim();
  return value || "资源文件";
}

/**
 * 由资源对象构建拖拽 payload。
 * @param {AssetLike | null | undefined} asset - 资源对象
 * @returns {DesignerAssetDragPayload | null}
 */
export function buildAssetDragPayload(asset: AssetLike | null | undefined): DesignerAssetDragPayload | null {
  if (!asset?.id) return null;
  const numericSize = Number(asset.size ?? 0);
  const payload: Partial<DesignerAssetDragPayload> = {
    id: String(asset.id),
    size: Number.isFinite(numericSize) ? numericSize : 0,
  };
  const displayName = asset.displayName || asset.name || asset.originalName || "";
  const optionalMap: Record<string, string> = {
    name: asset.name || "",
    displayName,
    url: asset.url || "",
    thumbnailUrl: asset.thumbnailUrl || "",
    type: asset.type || "",
    mimeType: asset.mimeType || "",
    ext: asset.ext || "",
  };
  for (const [key, value] of Object.entries(optionalMap)) {
    const trimmed = String(value || "").trim();
    if (!trimmed) continue;
    if (key === "name") payload.name = trimmed;
    if (key === "displayName") payload.displayName = trimmed;
    if (key === "url") payload.url = trimmed;
    if (key === "thumbnailUrl") payload.thumbnailUrl = trimmed;
    if (key === "type") payload.type = trimmed;
    if (key === "mimeType") payload.mimeType = trimmed;
    if (key === "ext") payload.ext = trimmed;
  }
  return payload as DesignerAssetDragPayload;
}

/**
 * 序列化资源拖拽 payload。
 * @param {DesignerAssetDragPayload} payload - 资源拖拽数据
 * @returns {string}
 */
export function serializeAssetDragPayload(payload: DesignerAssetDragPayload): string {
  return JSON.stringify(payload);
}

/**
 * 解析资源拖拽 payload。
 * @param {string | null | undefined} raw - 原始字符串
 * @returns {DesignerAssetDragPayload | null}
 */
export function parseAssetDragPayload(raw: string | null | undefined): DesignerAssetDragPayload | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as DesignerAssetDragPayload;
    if (!parsed || typeof parsed !== "object") return null;
    if (!parsed.id) return null;
    return parsed;
  } catch {
    return null;
  }
}

/**
 * 解析资源应落成的组件类型。
 * - 图片类：Image
 * - 其它资源：DownloadLink
 * @param {DesignerAssetDragPayload | null | undefined} asset - 资源拖拽数据
 * @returns {DesignerAssetComponentType}
 */
export function resolveAssetComponentType(
  asset: DesignerAssetDragPayload | null | undefined,
): DesignerAssetComponentType {
  const type = String(asset?.type || "").toLowerCase();
  const mime = String(asset?.mimeType || "").toLowerCase();
  if (type === "image" || type === "svg" || mime.startsWith("image/")) {
    return "Image";
  }
  if (type === "video" || mime.startsWith("video/")) {
    return "Video";
  }
  return "DownloadLink";
}

/**
 * 由资源拖拽数据生成组件属性。
 * @param {DesignerAssetDragPayload | null | undefined} asset - 资源拖拽数据
 * @param {DesignerAssetComponentType} componentType - 组件类型
 * @returns {Record<string, unknown>}
 */
export function buildAssetNodeProps(
  asset: DesignerAssetDragPayload | null | undefined,
  componentType: DesignerAssetComponentType,
): Record<string, unknown> {
  const fileName = resolveAssetDisplayName(asset);
  const url = String(asset?.url || "").trim();
  if (componentType === "Image") {
    return {
      src: url,
      alt: fileName,
      fit: "contain",
    };
  }
  if (componentType === "Video") {
    return {
      src: url,
      controls: true,
      autoplay: false,
      loop: false,
      muted: false,
    };
  }
  return {
    text: fileName,
    href: url,
    displayMode: "button",
    actionMode: "download",
    triggerMode: "double",
    downloadFileName: fileName,
    target: "_blank",
  };
}
