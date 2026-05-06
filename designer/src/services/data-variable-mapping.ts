/**
 * 数据点映射为工程变量的前端规则。
 *
 * 这里生成的是设计态工程变量定义，运行态仍只消费工程变量 `$global.xxx`，
 * 不让未映射的数据点直接进入脚本上下文。
 */

export interface DataCenterDataPointLike {
  id?: string;
  datapointId?: string;
  name?: string;
  path?: string;
  dataType?: string;
  type?: string;
  sourceType?: string;
  sourceId?: string;
  description?: string;
  status?: string;
}

export interface ProjectVariableSourceLike {
  type?: string;
  path?: string;
  sourceType?: string;
  sourceId?: string;
  datapointId?: string;
}

export interface ProjectVariableDefinitionLike {
  type?: string;
  default?: unknown;
  value?: unknown;
  description?: string;
  groupId?: string | null;
  mapped?: boolean;
  source?: ProjectVariableSourceLike;
}

export interface BuildProjectVariableOptions {
  existingNames?: string[];
  name?: string;
  type?: string;
  groupId?: string | null;
  defaultValue?: unknown;
  description?: string;
}

export interface BuiltProjectVariable {
  name: string;
  definition: {
    type: string;
    default: unknown;
    description: string;
    groupId?: string | null;
    mapped: true;
    source: {
      type: "dataCenter";
      path: string;
      sourceType: string;
      sourceId: string;
      datapointId: string;
    };
  };
}

const JS_IDENTIFIER_RE = /^[A-Za-z_$][\w$]*$/;

function normalizeAsciiIdentifier(input: string): string {
  return input
    .trim()
    .replace(/[^A-Za-z0-9_$]/g, "_")
    .replace(/_+/g, "_")
    .replace(/^_+|_+$/g, "");
}

export function normalizeProjectVariableName(input: string, existingNames: string[] = []): string {
  const existing = new Set(existingNames);
  let baseName = normalizeAsciiIdentifier(input || "");
  if (!baseName) {
    baseName = "datapoint";
  }
  if (/^\d/.test(baseName)) {
    baseName = `dp_${baseName}`;
  }
  let candidate = baseName;
  let index = 2;
  while (existing.has(candidate)) {
    candidate = `${baseName}_${index}`;
    index += 1;
  }
  return candidate;
}

export function resolveProjectVariableSnippet(name: string): string {
  if (JS_IDENTIFIER_RE.test(name)) {
    return `$global.${name}`;
  }
  return `$global[${JSON.stringify(name)}]`;
}

export function resolvePageVariableSnippet(name: string): string {
  if (JS_IDENTIFIER_RE.test(name)) {
    return `$vars.${name}`;
  }
  return `$vars[${JSON.stringify(name)}]`;
}

function normalizeVariableType(dataType: unknown): string {
  const value = String(dataType || "string").toLowerCase();
  if (["number", "int", "integer", "float", "double", "decimal"].some((item) => value.includes(item))) {
    return "number";
  }
  if (["bool", "boolean"].some((item) => value.includes(item))) {
    return "boolean";
  }
  if (value.includes("array") || value.includes("list")) {
    return "array";
  }
  if (value.includes("map")) {
    return "map";
  }
  if (value.includes("set")) {
    return "set";
  }
  if (value.includes("date") || value.includes("time")) {
    return "date";
  }
  if (value.includes("object") || value.includes("json") || value.includes("struct")) {
    return "object";
  }
  return "string";
}

function defaultValueForType(type: string): unknown {
  switch (type) {
    case "number":
      return 0;
    case "boolean":
      return false;
    case "array":
      return [];
    case "set":
      return [];
    case "map":
      return [];
    case "object":
      return {};
    case "date":
      return null;
    default:
      return "";
  }
}

export function buildProjectVariableFromDataPoint(
  datapoint: DataCenterDataPointLike,
  options: BuildProjectVariableOptions = {},
): BuiltProjectVariable {
  const rawName = options.name || datapoint.name || datapoint.path || datapoint.id || "datapoint";
  const name = normalizeProjectVariableName(rawName, options.existingNames || []);
  const type = normalizeVariableType(options.type || datapoint.dataType || datapoint.type);
  const path = String(datapoint.path || datapoint.name || datapoint.id || "");
  const datapointId = String(datapoint.datapointId || datapoint.id || path);
  const description = options.description ?? datapoint.description ?? "";
  const defaultValue =
    Object.prototype.hasOwnProperty.call(options, "defaultValue")
      ? options.defaultValue
      : defaultValueForType(type);

  return {
    name,
    definition: {
      type,
      default: defaultValue,
      description,
      groupId: options.groupId ?? null,
      mapped: true,
      source: {
        type: "dataCenter",
        path,
        sourceType: String(datapoint.sourceType || datapoint.type || ""),
        sourceId: String(datapoint.sourceId || ""),
        datapointId,
      },
    },
  };
}

export function findMappedProjectVariableName(
  datapoint: DataCenterDataPointLike,
  projectVariables: Record<string, ProjectVariableDefinitionLike> | null | undefined,
): string | null {
  const datapointId = String(datapoint.datapointId || datapoint.id || "");
  const path = String(datapoint.path || "");
  for (const [name, detail] of Object.entries(projectVariables || {})) {
    const source = detail?.source;
    if (source?.type !== "dataCenter" && detail?.mapped !== true) continue;
    if (datapointId && source?.datapointId === datapointId) return name;
    if (path && source?.path === path) return name;
  }
  return null;
}

export function resolveProjectVariableSourceLabel(
  detail: ProjectVariableDefinitionLike | null | undefined,
): string {
  const source = detail?.source;
  if (source?.type !== "dataCenter") return "";
  return source.path || source.datapointId || source.sourceId || "";
}
