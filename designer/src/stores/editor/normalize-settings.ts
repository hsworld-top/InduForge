/**
 * 工程设置与 API 负载规范化（变量、全局脚本、菜单默认值等）
 */

export interface GlobalScriptsNormalized {
  system: {
    startup: { code: string }
    shutdown: { code: string }
  }
  timers: { groups: unknown[]; items: unknown[] }
  variableChanges: { groups: unknown[]; items: unknown[] }
  custom: { groups: unknown[]; items: unknown[] }
}

export interface GlobalVariablesNormalized {
  definitions: Record<string, unknown>
  groups: unknown[]
}

export function getDefaultGlobalScripts(): GlobalScriptsNormalized {
  return {
    system: {
      startup: { code: '' },
      shutdown: { code: '' },
    },
    timers: { groups: [], items: [] },
    variableChanges: { groups: [], items: [] },
    custom: { groups: [], items: [] },
  }
}

export function normalizeVariableDef(detail: unknown): unknown {
  if (!detail || typeof detail !== 'object') return detail
  const next = { ...(detail as Record<string, unknown>) }
  if (typeof next.source === 'string') {
    next.source = { type: 'dataCenter', path: next.source }
    next.mapped = true
  }
  const mappedPath = next.mappedPath || next.sourcePath || next.path
  if (!next.source && mappedPath) {
    next.source = { type: 'dataCenter', path: mappedPath }
    next.mapped = true
  }
  if (next.source && typeof next.source === 'object') {
    const src = next.source as Record<string, unknown>
    if (!src.type && (next.mapped || src.path)) {
      src.type = 'dataCenter'
    }
    if (!next.mapped && src.type === 'dataCenter') {
      next.mapped = true
    }
  }
  return next
}

export function normalizeGlobalVariables(raw: Record<string, unknown>): GlobalVariablesNormalized {
  if (!raw || typeof raw !== 'object') {
    throw new Error('globalVariables 无效')
  }
  if (raw.definitions || raw.groups) {
    const definitions =
      raw.definitions && typeof raw.definitions === 'object'
        ? (raw.definitions as Record<string, unknown>)
        : {}
    const normalizedDefinitions: Record<string, unknown> = {}
    Object.entries(definitions).forEach(([name, detail]) => {
      normalizedDefinitions[name] = normalizeVariableDef(detail)
    })
    return {
      definitions: normalizedDefinitions,
      groups: Array.isArray(raw.groups) ? raw.groups : [],
    }
  }
  const normalizedDefinitions: Record<string, unknown> = {}
  Object.entries(raw).forEach(([name, detail]) => {
    normalizedDefinitions[name] = normalizeVariableDef(detail)
  })
  return { definitions: normalizedDefinitions, groups: [] }
}

export function normalizeGlobalScripts(
  raw: Record<string, unknown> | null | undefined,
): GlobalScriptsNormalized {
  const system =
    raw && typeof raw.system === 'object' && raw.system
      ? (raw.system as Record<string, unknown>)
      : {}
  const timers =
    raw && typeof raw.timers === 'object' && raw.timers
      ? (raw.timers as Record<string, unknown>)
      : {}
  const variableChanges =
    raw && typeof raw.variableChanges === 'object' && raw.variableChanges
      ? (raw.variableChanges as Record<string, unknown>)
      : {}
  const custom =
    raw && typeof raw.custom === 'object' && raw.custom
      ? (raw.custom as Record<string, unknown>)
      : {}

  const sysStartup =
    system.startup && typeof system.startup === 'object'
      ? (system.startup as { code?: string })
      : {}
  const sysShutdown =
    system.shutdown && typeof system.shutdown === 'object'
      ? (system.shutdown as { code?: string })
      : {}

  return {
    system: {
      startup: { code: sysStartup.code || '' },
      shutdown: { code: sysShutdown.code || '' },
    },
    timers: {
      groups: Array.isArray(timers.groups) ? timers.groups : [],
      items: Array.isArray(timers.items) ? timers.items : [],
    },
    variableChanges: {
      groups: Array.isArray(variableChanges.groups) ? variableChanges.groups : [],
      items: Array.isArray(variableChanges.items) ? variableChanges.items : [],
    },
    custom: {
      groups: Array.isArray(custom.groups) ? custom.groups : [],
      items: Array.isArray(custom.items) ? custom.items : [],
    },
  }
}

export function getMenuDefaultDetailConfig(): string {
  return (
    'this.menu({\n' +
    '  id: "menuNav",\n' +
    '  label: "菜单基础配置",\n' +
    '  type: "Menu",\n' +
    '  props: {\n' +
    '    defaultActive: "2",\n' +
    '    items: [\n' +
    '      { index: "1", label: "导航一", icon: "location" },\n' +
    '      { index: "2", label: "导航二", icon: "menu" },\n' +
    '      { index: "3", label: "导航三", icon: "document", disabled: true },\n' +
    '      { index: "4", label: "导航四", icon: "setting" },\n' +
    '    ],\n' +
    '  },\n' +
    '});'
  )
}

export function getMenuDefaultProps(): Record<string, unknown> {
  return {
    defaultActive: '2',
    items: [
      { index: '1', label: '导航一', icon: 'location' },
      { index: '2', label: '导航二', icon: 'menu' },
      { index: '3', label: '导航三', icon: 'document', disabled: true },
      { index: '4', label: '导航四', icon: 'setting' },
    ],
  }
}
