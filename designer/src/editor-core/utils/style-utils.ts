/**
 * 样式工具函数
 *
 * 从 NodeRenderer 抽取的样式处理辅助函数，供 composable 和组件复用。
 *
 * @module editor-core/utils/style-utils
 */

const PX_KEYS = new Set([
  "left",
  "top",
  "right",
  "bottom",
  "width",
  "height",
  "minWidth",
  "minHeight",
  "maxWidth",
  "maxHeight",
  "fontSize",
  "letterSpacing",
  "borderRadius",
  "gap",
  "padding",
  "paddingTop",
  "paddingRight",
  "paddingBottom",
  "paddingLeft",
  "margin",
  "marginTop",
  "marginRight",
  "marginBottom",
  "marginLeft",
]);

const NUMERIC_PX_RE = /^-?\d+(?:\.\d+)?$/;

export function needsPxUnit(key: string): boolean {
  return PX_KEYS.has(key);
}

export function normalizeStyleValue(key: string, value: unknown): unknown {
  if (value === null || value === undefined) return value;
  if (typeof value === "string" && needsPxUnit(key)) {
    const trimmed = value.trim();
    if (trimmed && NUMERIC_PX_RE.test(trimmed)) {
      return `${trimmed}px`;
    }
  }
  if (typeof value === "number" && needsPxUnit(key)) {
    return `${value}px`;
  }
  return value;
}

export function normalizeStyleObject(rawStyle: Record<string, unknown>): Record<string, unknown> {
  const style: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(rawStyle)) {
    if (value && typeof value === "object" && key === "background") {
      const bg = value as { value?: unknown };
      if (bg.value !== undefined) {
        style.background = bg.value;
      }
      continue;
    }
    style[key] = normalizeStyleValue(key, value);
  }
  return style;
}

export function resolveTextPropStyle(props: Record<string, unknown>): Record<string, unknown> {
  if (!props || typeof props !== "object") return {};
  const style: Record<string, unknown> = {};
  if (props.fontSize !== undefined) style.fontSize = props.fontSize;
  if (props.fontWeight !== undefined) style.fontWeight = props.fontWeight;
  if (props.fontFamily) style.fontFamily = props.fontFamily;
  if (props.color) style.color = props.color;
  if (props.textAlign) style.textAlign = props.textAlign;
  if (props.textAlign === "justify") {
    style.textAlignLast = "justify";
  }
  if (props.lineHeight !== undefined) style.lineHeight = props.lineHeight;
  if (props.letterSpacing !== undefined) style.letterSpacing = props.letterSpacing;
  if (props.wordBreak) style.wordBreak = props.wordBreak;

  const lineClamp = Number(props.lineClamp);
  if (Number.isFinite(lineClamp) && lineClamp > 0) {
    style.display = "-webkit-box";
    style.overflow = "hidden";
    style.WebkitLineClamp = lineClamp;
    style.WebkitBoxOrient = "vertical";
  } else if (props.truncate) {
    style.whiteSpace = "nowrap";
    style.overflow = "hidden";
    style.textOverflow = "ellipsis";
  }

  return style;
}
