/**
 * 工程设置加载/保存（从 editor-store 拆出，供壳层委托）
 */

import type { Ref } from "vue";
import { unwrapApiData } from "@/types/api";
import { normalizeGlobalScripts, normalizeGlobalVariables } from "./normalize-settings";

export interface ProjectSettingsApi {
  getProjectSettings: (projectId: string) => Promise<unknown>;
  updateProjectSettings: (projectId: string, payload: unknown) => Promise<unknown>;
  updateProjectVariables: (projectId: string, variables: unknown) => Promise<unknown>;
}

export interface ProjectSettingsStateRefs {
  projectVariables: Ref<Record<string, unknown>>;
  projectVariableGroups: Ref<unknown[]>;
  globalScripts: Ref<unknown>;
}

export async function loadProjectSettingsForStore(
  projectId: string,
  api: ProjectSettingsApi,
  out: ProjectSettingsStateRefs,
): Promise<void> {
  if (!projectId) return;

  const settingsRaw = await api.getProjectSettings(projectId);
  const settingsResult = unwrapApiData(settingsRaw);
  if (!settingsResult || typeof settingsResult !== "object") {
    throw new Error("工程设置响应无效");
  }
  const sr = settingsResult as Record<string, unknown>;
  const gv = sr.globalVariables;
  const normalizedVariables = normalizeGlobalVariables(
    gv && typeof gv === "object"
      ? (gv as Record<string, unknown>)
      : { definitions: {}, groups: [] },
  );
  out.projectVariables.value = normalizedVariables.definitions as Record<string, unknown>;
  out.projectVariableGroups.value = normalizedVariables.groups;
  out.globalScripts.value = normalizeGlobalScripts(
    (sr.globalScripts as Record<string, unknown>) || {},
  );
}

export async function saveProjectSettingsForStore(
  projectId: string,
  api: ProjectSettingsApi,
  state: ProjectSettingsStateRefs,
): Promise<{ ok: boolean; error?: Error }> {
  if (!projectId) {
    return { ok: false, error: new Error("缺少工程信息") };
  }

  const payload = {
    globalVariables: {
      definitions: state.projectVariables.value,
      groups: state.projectVariableGroups.value,
    },
    globalScripts: state.globalScripts.value,
  };

  try {
    const results = await Promise.allSettled([
      api.updateProjectSettings(projectId, payload),
      api.updateProjectVariables(projectId, state.projectVariables.value),
    ]);
    const settingsResult = results[0].status === "fulfilled" ? results[0].value : null;
    const data = settingsResult != null ? unwrapApiData(settingsResult) : null;
    const rejected = results.find((entry) => entry.status === "rejected");
    if (rejected) {
      return {
        ok: false,
        error: rejected.reason instanceof Error ? rejected.reason : new Error("保存失败"),
      };
    }

    if (data && typeof data === "object") {
      const body = data as Record<string, unknown>;
      if (body.globalVariables) {
        const normalized = normalizeGlobalVariables(
          body.globalVariables as Record<string, unknown>,
        );
        state.projectVariables.value = normalized.definitions as Record<string, unknown>;
        state.projectVariableGroups.value = normalized.groups;
      }
      if (body.globalScripts) {
        state.globalScripts.value = normalizeGlobalScripts(
          body.globalScripts as Record<string, unknown>,
        );
      }
    }

    return { ok: true };
  } catch (error) {
    return {
      ok: false,
      error: error instanceof Error ? error : new Error("保存失败"),
    };
  }
}
