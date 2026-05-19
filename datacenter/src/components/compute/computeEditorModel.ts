import type {
  ComputeUnitDetail,
  ComputeUnitSave,
} from "@/api/schemas/compute.schema";

export interface ComputeEditorTab {
  id: string;
  name: string;
  dirty: boolean;
}

export interface ComputeInputRow {
  uid: string;
  name: string;
  path: string;
  datapointId?: string;
}

export interface ComputeDependencyDraft {
  id: string;
}

export interface ComputeDraft {
  id: string;
  name: string;
  folderId: string | null;
  lang: string;
  status: string;
  code: string;
  triggerType: string;
  triggerConfig: Record<string, unknown>;
  inputBindings: Record<string, unknown>;
  outputBindings: Record<string, unknown>;
  timeoutMs: number;
  isEnabled: boolean;
  dependencies: ComputeDependencyDraft[];
  inputRows: ComputeInputRow[];
  dirty: boolean;
}

const defaultCode = (lang?: string) => {
  if (lang === "python") {
    return "result = input\n";
  }
  return "result = input;\n";
};

export function toComputeDraft(unit: ComputeUnitDetail): ComputeDraft {
  const inputBindings = asRecord(unit.inputBindings);
  return {
    id: String(unit.id),
    name: unit.name || "未命名计算单元",
    folderId: unit.folderId ? String(unit.folderId) : null,
    lang: String(unit.lang || unit.language || "javascript"),
    status: String(unit.status || (unit.isEnabled === false ? "disabled" : "enabled")),
    code:
      String(unit.code || unit.scriptCode || "") ||
      defaultCode(String(unit.lang || unit.language)),
    triggerType: String(unit.triggerType || "manual"),
    triggerConfig: asRecord(unit.triggerConfig),
    inputBindings,
    outputBindings: asRecord(unit.outputBindings),
    timeoutMs: Number(unit.timeoutMs || 3000),
    isEnabled: unit.isEnabled !== false,
    dependencies: toDependencyDrafts(unit.dependencies),
    inputRows: toInputRows(inputBindings),
    dirty: false,
  };
}

export function draftToSavePayload(draft: ComputeDraft): Partial<ComputeUnitSave> {
  return {
    name: draft.name,
    lang: draft.lang,
    code: draft.code,
    folderId: draft.folderId,
    triggerType: draft.triggerType,
    triggerConfig: draft.triggerConfig,
    inputBindings: inputRowsToBindings(draft.inputRows),
    outputBindings: draft.outputBindings,
    timeoutMs: draft.timeoutMs,
    isEnabled: draft.isEnabled,
    dependencies: draft.dependencies.map((item) => ({ id: item.id })),
  };
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return { ...(value as Record<string, unknown>) };
  }
  return {};
}

function toDependencyDrafts(value: unknown): ComputeDependencyDraft[] {
  if (!Array.isArray(value)) return [];
  return value
    .map((item) => {
      if (typeof item === "string") return { id: item };
      if (item && typeof item === "object" && "id" in item) {
        return { id: String((item as { id: unknown }).id) };
      }
      return null;
    })
    .filter((item): item is ComputeDependencyDraft => Boolean(item?.id));
}

function toInputRows(bindings: Record<string, unknown>): ComputeInputRow[] {
  const rawInputs = bindings.inputs;
  if (Array.isArray(rawInputs)) {
    const rows: Array<ComputeInputRow | null> = rawInputs.map((item) => {
        if (!item || typeof item !== "object") return null;
        const record = item as Record<string, unknown>;
        return {
          uid: String(crypto.randomUUID()),
          name: String(record.name || record.key || ""),
          path: String(record.path || record.datapointPath || ""),
          datapointId: record.datapointId ? String(record.datapointId) : undefined,
        };
      });
    return rows.filter(
      (item): item is ComputeInputRow => Boolean(item?.name || item?.path),
    );
  }

  return Object.entries(bindings).map(([name, value]) => ({
    uid: String(crypto.randomUUID()),
    name,
    path: String(value || ""),
  }));
}

function inputRowsToBindings(rows: ComputeInputRow[]): Record<string, unknown> {
  return {
    inputs: rows
      .filter((row) => row.name.trim() && row.path.trim())
      .map((row) => ({
        name: row.name.trim(),
        path: row.path.trim(),
        datapointId: row.datapointId,
      })),
  };
}
