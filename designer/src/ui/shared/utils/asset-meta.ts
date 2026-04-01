interface AssetMetaLike {
  ext?: string | undefined;
  type?: string | undefined;
  mimeType?: string | undefined;
}

const MIME_EXT_MAP: Record<string, string> = {
  "application/x-zip-compressed": "zip",
  "application/zip": "zip",
  "application/x-rar-compressed": "rar",
  "application/x-7z-compressed": "7z",
};

/**
 * 规范化资源扩展名，避免展示 `x-zip-compressed` 这类 MIME 尾段。
 * @param {string | null | undefined} value - 原始扩展名或 MIME 片段
 * @returns {string}
 */
export function normalizeAssetExt(value: string | null | undefined): string {
  const raw = String(value || "").trim().toLowerCase();
  if (!raw) return "";
  if (MIME_EXT_MAP[raw]) return MIME_EXT_MAP[raw];
  if (raw.includes("zip")) return "zip";
  if (raw.includes("rar")) return "rar";
  if (raw.includes("7z")) return "7z";
  if (raw.startsWith("x-")) return raw.slice(2);
  return raw;
}

/**
 * 生成资源类型展示文案（统一大写）。
 * @param {AssetMetaLike | null | undefined} asset - 资源元信息
 * @returns {string}
 */
export function resolveAssetTypeLabel(asset: AssetMetaLike | null | undefined): string {
  const ext = normalizeAssetExt(asset?.ext);
  if (ext) return ext.toUpperCase();
  const mime = normalizeAssetExt(asset?.mimeType);
  if (mime) return mime.toUpperCase();
  const type = normalizeAssetExt(asset?.type);
  if (type) return type.toUpperCase();
  return "FILE";
}
