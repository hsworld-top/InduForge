/**
 * 工程设置加载/保存（从 editor-store 拆出，供壳层委托）
 */

import type { Ref } from 'vue'
import type { ProjectI18nSettings } from '@/editor-core/document/types'
import { unwrapApiData } from '@/types/api'
import { normalizeGlobalScripts, normalizeGlobalVariables } from './normalize-settings'
import { normalizeProjectI18nSettings } from '@/editor-core/i18n/project-i18n'

export interface ProjectSettingsApi {
  getProjectSettings: (projectId: string) => Promise<unknown>
  updateProjectSettings: (projectId: string, payload: unknown) => Promise<unknown>
  updateProjectVariables: (projectId: string, variables: unknown) => Promise<unknown>
}

export interface ProjectSettingsStateRefs {
  projectVariables: Ref<Record<string, unknown>>
  projectVariableGroups: Ref<unknown[]>
  globalScripts: Ref<unknown>
  projectI18n: Ref<ProjectI18nSettings>
}

export interface ProjectSettingsSnapshot {
  projectVariables: Record<string, unknown>
  projectVariableGroups: unknown[]
  globalScripts: ReturnType<typeof normalizeGlobalScripts>
  projectI18n: ProjectI18nSettings
}

export async function fetchProjectSettingsForStore(
  projectId: string,
  api: Pick<ProjectSettingsApi, 'getProjectSettings'>,
): Promise<ProjectSettingsSnapshot> {
  if (!projectId) {
    return {
      projectVariables: {},
      projectVariableGroups: [],
      globalScripts: normalizeGlobalScripts({}),
      projectI18n: normalizeProjectI18nSettings(null),
    }
  }

  const settingsRaw = await api.getProjectSettings(projectId)
  const settingsResult = unwrapApiData(settingsRaw)
  if (!settingsResult || typeof settingsResult !== 'object') {
    throw new Error('工程设置响应无效')
  }
  const sr = settingsResult as Record<string, unknown>
  const gv = sr.globalVariables
  const normalizedVariables = normalizeGlobalVariables(
    gv && typeof gv === 'object'
      ? (gv as Record<string, unknown>)
      : { definitions: {}, groups: [] },
  )

  return {
    projectVariables: normalizedVariables.definitions as Record<string, unknown>,
    projectVariableGroups: normalizedVariables.groups,
    globalScripts: normalizeGlobalScripts((sr.globalScripts as Record<string, unknown>) || {}),
    projectI18n: normalizeProjectI18nSettings(sr.i18n),
  }
}

export async function loadProjectSettingsForStore(
  projectId: string,
  api: ProjectSettingsApi,
  out: ProjectSettingsStateRefs,
): Promise<void> {
  const snapshot = await fetchProjectSettingsForStore(projectId, api)
  out.projectVariables.value = snapshot.projectVariables
  out.projectVariableGroups.value = snapshot.projectVariableGroups
  out.globalScripts.value = snapshot.globalScripts
  out.projectI18n.value = snapshot.projectI18n
}

export async function saveProjectSettingsForStore(
  projectId: string,
  api: ProjectSettingsApi,
  state: ProjectSettingsStateRefs,
): Promise<{ ok: boolean; error?: Error }> {
  if (!projectId) {
    return { ok: false, error: new Error('缺少工程信息') }
  }

  const payload = {
    globalVariables: {
      definitions: state.projectVariables.value,
      groups: state.projectVariableGroups.value,
    },
    globalScripts: state.globalScripts.value,
    i18n: state.projectI18n.value,
  }

  try {
    const results = await Promise.allSettled([
      api.updateProjectSettings(projectId, payload),
      api.updateProjectVariables(projectId, state.projectVariables.value),
    ])
    const settingsResult = results[0].status === 'fulfilled' ? results[0].value : null
    const data = settingsResult != null ? unwrapApiData(settingsResult) : null
    const rejected = results.find((entry) => entry.status === 'rejected')
    if (rejected) {
      return {
        ok: false,
        error: rejected.reason instanceof Error ? rejected.reason : new Error('保存失败'),
      }
    }

    if (data && typeof data === 'object') {
      const body = data as Record<string, unknown>
      if (body.globalVariables) {
        const normalized = normalizeGlobalVariables(body.globalVariables as Record<string, unknown>)
        state.projectVariables.value = normalized.definitions as Record<string, unknown>
        state.projectVariableGroups.value = normalized.groups
      }
      if (body.globalScripts) {
        state.globalScripts.value = normalizeGlobalScripts(
          body.globalScripts as Record<string, unknown>,
        )
      }
      if (body.i18n) {
        state.projectI18n.value = normalizeProjectI18nSettings(body.i18n)
      }
    }

    return { ok: true }
  } catch (error) {
    return {
      ok: false,
      error: error instanceof Error ? error : new Error('保存失败'),
    }
  }
}
