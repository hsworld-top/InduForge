/**
 * 样式配置 CSS 工具。
 *
 * 设计器属性面板生成的样式多为 inline style，用户在样式编辑器中手写的
 * 选择器 CSS 需要具备更高优先级，因此注入前统一补充 !important。
 */

const KEYFRAME_SELECTOR_RE = /^(from|to|\d+(?:\.\d+)?%)$/i;
const DOM_TOKEN_RE = /[^a-zA-Z0-9_-]+/g;

function normalizeDomToken(value: string, fallback: string): string {
  const text = String(value || "")
    .trim()
    .replace(DOM_TOKEN_RE, "-")
    .replace(/^-+|-+$/g, "");
  return text || fallback;
}

export function buildDesignerPageDomId(pageId: string): string {
  return `page-${normalizeDomToken(pageId, "current")}`;
}

export function buildDesignerNodeDomId(nodeId: string): string {
  return `dom-${normalizeDomToken(nodeId, "node")}`;
}

export function replaceStyleConfigPlaceholders(
  css: string,
  options: { pageId?: string | null; nodeId?: string | null } = {},
): string {
  let text = String(css || "");
  if (options.pageId) {
    text = text.replace(/#pageId\b/g, `#${buildDesignerPageDomId(options.pageId)}`);
  }
  if (options.nodeId) {
    text = text.replace(/#domId\b/g, `#${buildDesignerNodeDomId(options.nodeId)}`);
  }
  return text;
}

function splitDeclarations(body: string): string[] {
  const declarations: string[] = [];
  let buffer = "";
  let quote: '"' | "'" | "" = "";
  let parenDepth = 0;

  for (const char of body) {
    if (quote) {
      buffer += char;
      if (char === quote) quote = "";
      continue;
    }
    if (char === '"' || char === "'") {
      quote = char;
      buffer += char;
      continue;
    }
    if (char === "(") {
      parenDepth += 1;
      buffer += char;
      continue;
    }
    if (char === ")") {
      parenDepth = Math.max(0, parenDepth - 1);
      buffer += char;
      continue;
    }
    if (char === ";" && parenDepth === 0) {
      declarations.push(buffer);
      buffer = "";
      continue;
    }
    buffer += char;
  }

  if (buffer.trim()) {
    declarations.push(buffer);
  }

  return declarations;
}

function elevateDeclaration(declaration: string): string {
  const text = declaration.trim();
  if (!text || !text.includes(":")) return text;
  if (/!\s*important\s*$/i.test(text)) return text;
  return `${text} !important`;
}

function elevateBlock(selector: string, body: string): string {
  const trimmedSelector = selector.trim();
  if (!trimmedSelector || trimmedSelector.startsWith("@")) {
    return `${selector}{${body}}`;
  }
  if (KEYFRAME_SELECTOR_RE.test(trimmedSelector)) {
    return `${selector}{${body}}`;
  }

  const declarations = splitDeclarations(body).map(elevateDeclaration).filter(Boolean);
  if (declarations.length === 0) {
    return `${selector}{${body}}`;
  }
  return `${selector}{${declarations.join("; ")}}`;
}

/**
 * 提升用户手写样式配置的声明优先级。
 *
 * 这是一个轻量 CSS 重写器，覆盖设计器样式配置的常见选择器块场景；
 * 不处理 @keyframes/@font-face 等声明块，避免破坏动画和字体描述符。
 */
export function elevateStyleConfigPriority(css: string): string {
  const text = String(css || "").trim();
  if (!text || !text.includes("{")) return text;
  return text.replace(/([^{}]+)\{([^{}]*)\}/g, (_match, selector: string, body: string) =>
    elevateBlock(selector, body),
  );
}
