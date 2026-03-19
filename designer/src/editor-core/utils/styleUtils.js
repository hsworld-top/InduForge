/**
 * 样式工具函数
 *
 * 从 NodeRenderer 抽取的样式处理辅助函数，供 composable 和组件复用。
 *
 * @module editor-core/utils/styleUtils
 */

/**
 * 判断是否需要追加 px 单位
 * @param {string} key - 样式键
 * @returns {boolean}
 */
export function needsPxUnit(key) {
  return [
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
  ].includes(key);
}

/**
 * 标准化样式值（补充单位）
 * @param {string} key - 样式键
 * @param {any} value - 样式值
 * @returns {any} 标准化结果
 */
export function normalizeStyleValue(key, value) {
  if (value === null || value === undefined) return value;
  if (typeof value === "string" && needsPxUnit(key)) {
    const trimmed = value.trim();
    if (trimmed && /^-?\d+(\.\d+)?$/.test(trimmed)) {
      return `${trimmed}px`;
    }
  }
  if (typeof value === "number" && needsPxUnit(key)) {
    return `${value}px`;
  }
  return value;
}

/**
 * 标准化样式对象
 * @param {Record<string, any>} rawStyle - 原始样式
 * @returns {Record<string, any>} 规范化样式
 */
export function normalizeStyleObject(rawStyle) {
  const style = {};
  for (const [key, value] of Object.entries(rawStyle)) {
    if (value && typeof value === "object" && key === "background") {
      if (value.value !== undefined) {
        style.background = value.value;
      }
      continue;
    }
    style[key] = normalizeStyleValue(key, value);
  }
  return style;
}

/**
 * 解析文本组件的样式属性
 * @param {Record<string, any>} props - 文本属性
 * @returns {Record<string, any>}
 */
export function resolveTextPropStyle(props) {
  if (!props || typeof props !== "object") return {};
  const style = {};
  if (props.fontSize !== undefined) style.fontSize = props.fontSize;
  if (props.fontWeight !== undefined) style.fontWeight = props.fontWeight;
  if (props.fontFamily) style.fontFamily = props.fontFamily;
  if (props.color) style.color = props.color;
  if (props.textAlign) style.textAlign = props.textAlign;
  if (props.textAlign === "justify") {
    style.textAlignLast = "justify";
  }
  if (props.lineHeight !== undefined) style.lineHeight = props.lineHeight;
  if (props.letterSpacing !== undefined)
    style.letterSpacing = props.letterSpacing;
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
