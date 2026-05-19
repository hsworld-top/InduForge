import type {
  ComponentNode,
  ProjectI18nResource,
  ProjectI18nResourceTarget,
  ProjectI18nSettings,
  ProjectSchema,
} from "@/editor-core/document/types";
import "@/materials/manifests";
import { getManifest } from "@/materials/manifests/manifest-registry";

const DEFAULT_LOCALE = "zh-CN";
const DEFAULT_EDIT_LOCALE = "en-US";
const EXCLUDED_PROP_KEYS = new Set([
  "id",
  "key",
  "class",
  "className",
  "style",
  "icon",
  "src",
  "url",
  "href",
  "path",
  "name",
  "type",
  "value",
  "modelValue",
]);
const TEXT_PROP_KEYS = new Set([
  "text",
  "label",
  "title",
  "placeholder",
  "emptyText",
  "content",
  "description",
  "prefix",
  "suffix",
]);
const MANIFEST_TYPE_ALIASES: Record<string, string> = {
  button: "Button",
  elbutton: "Button",
  "el-button": "Button",
};
const PLAIN_TEXT_FIELD_COMPONENTS = new Set(["Button", "DownloadLink"]);

export type I18nScanStatus =
  | "candidate"
  | "complete"
  | "missing-default"
  | "missing-current"
  | "source-changed"
  | "orphan"
  | "binding-conflict";

export interface I18nTextCandidate {
  pageId: string;
  pageName: string;
  nodeId: string;
  nodeLabel: string;
  nodeType: string;
  fieldPath: string;
  sourceText: string;
  resourceKey?: string;
  hasDynamicBinding: boolean;
}

export interface I18nResourceRow extends I18nTextCandidate {
  id: string;
  status: I18nScanStatus;
  resourceKey: string;
  defaultValue: string;
  currentValue: string;
  resource?: ProjectI18nResource;
}

export interface I18nScanPageInput {
  pageId: string;
  pageName?: string;
  schema: ProjectSchema;
}

export interface I18nScanSummary {
  totalResources: number;
  candidates: number;
  missingCurrent: number;
  missingDefault: number;
  sourceChanged: number;
  orphan: number;
  conflicts: number;
}

export interface I18nScanResult {
  rows: I18nResourceRow[];
  summary: I18nScanSummary;
}

interface MutableNodeRef {
  schema: ProjectSchema;
  node: ComponentNode;
}

export function createDefaultProjectI18nSettings(): ProjectI18nSettings {
  return {
    enabled: false,
    defaultLocale: DEFAULT_LOCALE,
    currentLocale: DEFAULT_EDIT_LOCALE,
    locales: [
      { code: DEFAULT_LOCALE, name: "简体中文", enabled: true },
      { code: DEFAULT_EDIT_LOCALE, name: "English", enabled: true },
    ],
    resources: {},
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function normalizeLocaleCode(value: unknown, fallback: string): string {
  const text = String(value || "").trim();
  return text || fallback;
}

export function normalizeProjectI18nSettings(raw: unknown): ProjectI18nSettings {
  const defaults = createDefaultProjectI18nSettings();
  if (!isRecord(raw)) return defaults;

  const rawLocales = Array.isArray(raw.locales) ? raw.locales : [];
  const locales = rawLocales
    .map((item) => {
      if (!isRecord(item)) return null;
      const code = String(item.code || "").trim();
      if (!code) return null;
      return {
        code,
        name: String(item.name || code).trim() || code,
        enabled: item.enabled !== false,
      };
    })
    .filter(Boolean) as ProjectI18nSettings["locales"];

  const resources: Record<string, ProjectI18nResource> = {};
  const rawResources = isRecord(raw.resources) ? raw.resources : {};
  Object.entries(rawResources).forEach(([key, value]) => {
    if (!isRecord(value)) return;
    const target = isRecord(value.target) ? value.target : {};
    const resourceKey = String(value.key || key || "").trim();
    if (!resourceKey) return;
    const resource: ProjectI18nResource = {
      key: resourceKey,
      sourceText: String(value.sourceText || ""),
      values: normalizeResourceValues(value.values),
      target: {
        pageId: String(target.pageId || ""),
        pageName: String(target.pageName || ""),
        nodeId: String(target.nodeId || ""),
        nodeLabel: String(target.nodeLabel || ""),
        nodeType: String(target.nodeType || ""),
        fieldPath: String(target.fieldPath || ""),
      },
    };
    if (typeof value.updatedAt === "string") {
      resource.updatedAt = value.updatedAt;
    }
    resources[resourceKey] = resource;
  });

  const defaultLocale = normalizeLocaleCode(raw.defaultLocale, defaults.defaultLocale);
  const currentLocale = normalizeLocaleCode(raw.currentLocale, defaults.currentLocale);
  const nextLocales = locales.length ? locales : defaults.locales;
  if (!nextLocales.some((item) => item.code === defaultLocale)) {
    nextLocales.unshift({ code: defaultLocale, name: defaultLocale, enabled: true });
  }
  if (!nextLocales.some((item) => item.code === currentLocale)) {
    nextLocales.push({ code: currentLocale, name: currentLocale, enabled: true });
  }

  return {
    enabled: raw.enabled === true,
    defaultLocale,
    currentLocale,
    locales: nextLocales,
    resources,
  };
}

function normalizeResourceValues(value: unknown): Record<string, string> {
  if (!isRecord(value)) return {};
  const result: Record<string, string> = {};
  Object.entries(value).forEach(([locale, text]) => {
    result[locale] = String(text ?? "");
  });
  return result;
}

export function cloneProjectI18nSettings(settings: ProjectI18nSettings): ProjectI18nSettings {
  return normalizeProjectI18nSettings(JSON.parse(JSON.stringify(settings)));
}

export function getValueByPath(source: unknown, path: string): unknown {
  if (!path) return source;
  return path.split(".").reduce((current: unknown, segment) => {
    if (Array.isArray(current)) {
      const index = Number(segment);
      return Number.isInteger(index) ? current[index] : undefined;
    }
    if (isRecord(current)) return current[segment];
    return undefined;
  }, source);
}

export function setValueByPath(source: unknown, path: string, value: unknown): unknown {
  if (!isRecord(source) && !Array.isArray(source)) return source;
  const segments = path.split(".").filter(Boolean);
  if (!segments.length) return source;
  let current = source as Record<string, unknown> | unknown[];
  segments.forEach((segment, index) => {
    const isLast = index === segments.length - 1;
    if (isLast) {
      if (Array.isArray(current)) current[Number(segment)] = value;
      else current[segment] = value;
      return;
    }
    const nextSegment = segments[index + 1];
    const shouldBeArray = Number.isInteger(Number(nextSegment));
    if (Array.isArray(current)) {
      const arrayIndex = Number(segment);
      if (!isRecord(current[arrayIndex]) && !Array.isArray(current[arrayIndex])) {
        current[arrayIndex] = shouldBeArray ? [] : {};
      }
      current = current[arrayIndex] as Record<string, unknown> | unknown[];
      return;
    }
    if (!isRecord(current[segment]) && !Array.isArray(current[segment])) {
      current[segment] = shouldBeArray ? [] : {};
    }
    current = current[segment] as Record<string, unknown> | unknown[];
  });
  return source;
}

export function deleteValueByPath(source: unknown, path: string): unknown {
  if (!isRecord(source) && !Array.isArray(source)) return source;
  const segments = path.split(".").filter(Boolean);
  const last = segments.pop();
  if (!last) return source;
  const parent = getValueByPath(source, segments.join("."));
  if (Array.isArray(parent)) {
    parent.splice(Number(last), 1);
  } else if (isRecord(parent)) {
    delete parent[last];
  }
  return source;
}

function isTextCandidateKey(key: string, depth: number): boolean {
  if (depth === 0 && EXCLUDED_PROP_KEYS.has(key)) return false;
  if (TEXT_PROP_KEYS.has(key)) return true;
  return depth === 0 && !EXCLUDED_PROP_KEYS.has(key);
}

function shouldKeepTextValue(value: string, key = "", allowPlainText = false): boolean {
  const text = value.trim();
  if (!text) return false;
  if (/^\{\{[\s\S]*\}\}$/.test(text)) return false;
  const isPlainToken = /^[\w.-]+$/.test(text) && !/[\u4E00-\u9FA5\s]/.test(text);
  if (isPlainToken) return allowPlainText || key !== "text";
  return true;
}

function collectTextFields(
  value: unknown,
  out: Array<{ path: string; text: string }>,
  prefix = "",
  depth = 0,
  allowPlainTextFields = false,
): void {
  if (typeof value === "string") {
    const key = prefix.split(".").pop() || "";
    if (isTextCandidateKey(key, depth) && shouldKeepTextValue(value, key, allowPlainTextFields)) {
      out.push({ path: prefix, text: value });
    }
    return;
  }
  if (Array.isArray(value)) {
    value.forEach((item, index) => {
      collectTextFields(
        item,
        out,
        prefix ? `${prefix}.${index}` : String(index),
        depth + 1,
        allowPlainTextFields,
      );
    });
    return;
  }
  if (!isRecord(value)) return;
  Object.entries(value).forEach(([key, child]) => {
    if (depth === 0 && EXCLUDED_PROP_KEYS.has(key)) return;
    collectTextFields(
      child,
      out,
      prefix ? `${prefix}.${key}` : key,
      depth + 1,
      allowPlainTextFields,
    );
  });
}

function cloneDefaultValue(value: unknown): unknown {
  if (!isRecord(value) && !Array.isArray(value)) return value;
  return JSON.parse(JSON.stringify(value));
}

function resolveManifestType(type: string): string {
  const normalized = String(type || "").trim();
  if (!normalized) return "";
  const alias = MANIFEST_TYPE_ALIASES[normalized.toLowerCase()];
  return alias || normalized;
}

function resolveNodePropsForScan(node: ComponentNode): Record<string, unknown> {
  const defaultProps: Record<string, unknown> = {};
  const manifest = getManifest(resolveManifestType(node.type));
  manifest?.props?.forEach((prop) => {
    if (prop.defaultValue !== undefined) {
      defaultProps[prop.name] = cloneDefaultValue(prop.defaultValue);
    }
  });
  return {
    ...defaultProps,
    ...(node.props || {}),
  };
}

export function buildI18nResourceKey(target: ProjectI18nResourceTarget): string {
  const normalize = (value: string) =>
    String(value || "")
      .trim()
      .replace(/[^\w.-]+/g, "_")
      .replace(/_+/g, "_")
      .replace(/^_+|_+$/g, "");
  return [
    normalize(target.pageId || "page"),
    normalize(target.nodeId || "node"),
    "props",
    normalize(target.fieldPath || "text"),
  ].join(".");
}

function resolveStatus(args: {
  candidate: I18nTextCandidate;
  resource?: ProjectI18nResource;
  settings: ProjectI18nSettings;
}): I18nScanStatus {
  const { candidate, resource, settings } = args;
  if (candidate.hasDynamicBinding) return "binding-conflict";
  if (!candidate.resourceKey) return "candidate";
  if (!resource) return "candidate";
  if (!String(resource.values?.[settings.currentLocale] ?? "").trim()) return "missing-current";
  if (resource.sourceText !== candidate.sourceText) return "source-changed";
  return "complete";
}

export function scanProjectI18nResources(
  pages: I18nScanPageInput[],
  settingsInput: ProjectI18nSettings,
): I18nScanResult {
  const settings = normalizeProjectI18nSettings(settingsInput);
  const rows: I18nResourceRow[] = [];
  const seenResourceKeys = new Set<string>();

  pages.forEach(({ pageId, pageName, schema }) => {
    const page = schema.pagesById?.[pageId];
    const resolvedPageName = pageName || page?.name || pageId;
    Object.values(schema.nodesById || {}).forEach((node) => {
      if (!node?.id) return;
      const propsForScan = resolveNodePropsForScan(node);
      if (!Object.keys(propsForScan).length) return;
      const fields: Array<{ path: string; text: string }> = [];
      collectTextFields(
        propsForScan,
        fields,
        "",
        0,
        PLAIN_TEXT_FIELD_COMPONENTS.has(resolveManifestType(node.type)),
      );
      fields.forEach((field) => {
        const topProp = field.path.split(".")[0] || field.path;
        const resourceKey = node.i18n?.props?.[field.path] || "";
        if (resourceKey) seenResourceKeys.add(resourceKey);
        const candidate: I18nTextCandidate = {
          pageId,
          pageName: resolvedPageName,
          nodeId: node.id,
          nodeLabel: node.label || node.type,
          nodeType: node.type,
          fieldPath: field.path,
          sourceText: field.text,
          resourceKey,
          hasDynamicBinding: Boolean(node.bindings?.[field.path] || node.bindings?.[topProp]),
        };
        const resource = resourceKey ? settings.resources[resourceKey] : undefined;
        const fallbackTarget: ProjectI18nResourceTarget = {
          pageId,
          pageName: resolvedPageName,
          nodeId: node.id,
          nodeType: node.type,
          fieldPath: field.path,
        };
        if (node.label) fallbackTarget.nodeLabel = node.label;
        const fallbackKey = buildI18nResourceKey(fallbackTarget);
        const scanRow: I18nResourceRow = {
          ...candidate,
          id: `${pageId}:${node.id}:${field.path}`,
          status: resolveStatus({
            candidate,
            ...(resource ? { resource } : {}),
            settings,
          }),
          resourceKey: resourceKey || fallbackKey,
          defaultValue: resource?.values?.[settings.defaultLocale] ?? field.text,
          currentValue: resource?.values?.[settings.currentLocale] ?? "",
        };
        if (resource) scanRow.resource = resource;
        rows.push(scanRow);
      });
    });
  });

  Object.values(settings.resources).forEach((resource) => {
    if (seenResourceKeys.has(resource.key)) return;
    rows.push({
      id: `orphan:${resource.key}`,
      status: "orphan",
      pageId: resource.target.pageId,
      pageName: resource.target.pageName || resource.target.pageId,
      nodeId: resource.target.nodeId,
      nodeLabel: resource.target.nodeLabel || resource.target.nodeType || resource.target.nodeId,
      nodeType: resource.target.nodeType || "",
      fieldPath: resource.target.fieldPath,
      sourceText: resource.sourceText,
      resourceKey: resource.key,
      defaultValue: resource.values?.[settings.defaultLocale] ?? "",
      currentValue: resource.values?.[settings.currentLocale] ?? "",
      hasDynamicBinding: false,
      resource,
    });
  });

  return {
    rows,
    summary: summarizeI18nRows(rows, settings),
  };
}

export function summarizeI18nRows(
  rows: I18nResourceRow[],
  settings: ProjectI18nSettings,
): I18nScanSummary {
  const resourceKeys = new Set<string>();
  rows.forEach((row) => {
    if (row.resource) resourceKeys.add(row.resource.key);
  });
  return {
    totalResources: Math.max(resourceKeys.size, Object.keys(settings.resources || {}).length),
    candidates: rows.filter((row) => row.status === "candidate").length,
    missingCurrent: rows.filter((row) => row.status === "missing-current").length,
    missingDefault: rows.filter((row) => row.status === "missing-default").length,
    sourceChanged: rows.filter((row) => row.status === "source-changed").length,
    orphan: rows.filter((row) => row.status === "orphan").length,
    conflicts: rows.filter((row) => row.status === "binding-conflict").length,
  };
}

export function joinI18nResource(
  settingsInput: ProjectI18nSettings,
  node: ComponentNode,
  row: I18nResourceRow,
  updatedAt: string,
): ProjectI18nSettings {
  const settings = cloneProjectI18nSettings(settingsInput);
  const key = row.resourceKey || buildI18nResourceKey(row);
  node.i18n = {
    ...(node.i18n || {}),
    props: {
      ...(node.i18n?.props || {}),
      [row.fieldPath]: key,
    },
  };
  settings.resources[key] = {
    key,
    sourceText: row.sourceText,
    values: {
      ...(settings.resources[key]?.values || {}),
      [settings.defaultLocale]: row.sourceText,
      ...(settings.currentLocale === settings.defaultLocale
        ? {}
        : { [settings.currentLocale]: row.currentValue || "" }),
    },
    target: {
      pageId: row.pageId,
      pageName: row.pageName,
      nodeId: row.nodeId,
      nodeLabel: row.nodeLabel,
      nodeType: row.nodeType,
      fieldPath: row.fieldPath,
    },
    updatedAt,
  };
  return settings;
}

export function updateI18nResourceValue(
  settingsInput: ProjectI18nSettings,
  key: string,
  locale: string,
  value: string,
  updatedAt: string,
): ProjectI18nSettings {
  const settings = cloneProjectI18nSettings(settingsInput);
  const resource = settings.resources[key];
  if (!resource) return settings;
  resource.values = {
    ...(resource.values || {}),
    [locale]: value,
  };
  resource.updatedAt = updatedAt;
  return settings;
}

export function removeI18nBinding(node: ComponentNode, fieldPath: string): void {
  if (!node.i18n?.props) return;
  const nextProps = { ...(node.i18n.props || {}) };
  delete nextProps[fieldPath];
  node.i18n = {
    ...(node.i18n || {}),
    props: nextProps,
  };
}

export function deleteI18nResource(
  settingsInput: ProjectI18nSettings,
  key: string,
): ProjectI18nSettings {
  const settings = cloneProjectI18nSettings(settingsInput);
  delete settings.resources[key];
  return settings;
}

export function findMutableNode(
  pages: I18nScanPageInput[],
  pageId: string,
  nodeId: string,
): MutableNodeRef | null {
  const page = pages.find((item) => item.pageId === pageId);
  const node = page?.schema.nodesById?.[nodeId];
  if (!page || !node) return null;
  return { schema: page.schema, node };
}

export function translatePropsWithI18n(
  props: Record<string, unknown>,
  node: ComponentNode,
  settingsInput: ProjectI18nSettings | null | undefined,
  localeInput: string | null | undefined,
): Record<string, unknown> {
  const settings = settingsInput ? normalizeProjectI18nSettings(settingsInput) : null;
  if (!settings?.enabled || !node.i18n?.props) return props;
  const locale = String(localeInput || settings.currentLocale || settings.defaultLocale);
  const next = JSON.parse(JSON.stringify(props || {})) as Record<string, unknown>;
  Object.entries(node.i18n.props).forEach(([fieldPath, key]) => {
    const resource = settings.resources[key];
    if (!resource) return;
    const value =
      resource.values?.[locale] ||
      resource.values?.[settings.defaultLocale] ||
      resource.sourceText ||
      getValueByPath(props, fieldPath);
    setValueByPath(next, fieldPath, value);
  });
  return next;
}
