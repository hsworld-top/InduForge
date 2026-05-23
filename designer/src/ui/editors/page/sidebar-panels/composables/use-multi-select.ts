/**
 * 多选属性计算 Composable
 */

import type { History } from '@/editor-core/commands/History'
import { storeToRefs } from 'pinia'
import { computed, toRef } from 'vue'
import { UpdateGraphicCommand, UpdateNodeCommand } from '@/editor-core'
import { useEditorStore } from '@/stores/editor-store'

export type MultiSelectValue =
  | { type: 'same'; value: unknown }
  | { type: 'mixed'; values: unknown[] }

function getNestedValue(obj: Record<string, unknown> | null | undefined, path: string): unknown {
  if (!obj || !path) return undefined
  const keys = path.split('.')
  let value: unknown = obj
  for (const key of keys) {
    if (value === null || value === undefined) return undefined
    value = (value as Record<string, unknown>)[key]
  }
  return value
}

function setNestedValue(
  obj: Record<string, unknown>,
  path: string,
  value: unknown,
): Record<string, unknown> {
  if (!path) return obj
  const keys = path.split('.')
  const result = { ...obj }
  let current: Record<string, unknown> = result

  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i]!
    const next = current[key]
    current[key] = next && typeof next === 'object' ? { ...(next as object) } : {}
    current = current[key] as Record<string, unknown>
  }

  current[keys.at(-1)!] = value as never
  return result
}

export function getMultiSelectValueFromElements(
  elements: unknown[],
  path: string,
): MultiSelectValue {
  if (!elements || elements.length === 0) {
    return { type: 'same', value: undefined }
  }

  const values = elements.map((el) => getNestedValue(el as Record<string, unknown>, path))
  const uniqueValues: unknown[] = []
  const seen = new Set<string>()

  for (const val of values) {
    const key = JSON.stringify(val)
    if (!seen.has(key)) {
      seen.add(key)
      uniqueValues.push(val)
    }
  }

  if (uniqueValues.length === 1) {
    return { type: 'same', value: uniqueValues[0] }
  }

  return { type: 'mixed', values: uniqueValues }
}

export interface MultiSelectProps {
  elements: Array<{ id: string; kind: string }>
}

export function useMultiSelect(props: MultiSelectProps) {
  const editorStore = useEditorStore()
  const { doc, history } = storeToRefs(editorStore)

  const elementsRef = toRef(props, 'elements')

  const selectedCount = computed(() => elementsRef.value?.length || 0)

  const getMultiSelectValue = (path: string): MultiSelectValue => {
    if (!elementsRef.value?.length || !doc.value) {
      return { type: 'same', value: undefined }
    }

    const fullElements = elementsRef.value
      .map((el) => {
        if (el.kind === 'node') {
          return doc.value?.getNode?.(el.id)
        } else if (el.kind === 'graphic') {
          return doc.value?.getGraphic?.(el.id)
        }
        return el
      })
      .filter(Boolean)

    return getMultiSelectValueFromElements(fullElements, path)
  }

  const buildPatch = (path: string, value: unknown): Record<string, unknown> => {
    return setNestedValue({}, path, value)
  }

  const setUnifiedValue = (path: string, value: unknown): void => {
    if (!elementsRef.value?.length || !doc.value || !history.value) return

    const hist = history.value as History & {
      startBatch?: () => void
      endBatch?: () => void
    }
    hist.startBatch?.()

    try {
      for (const el of elementsRef.value) {
        if (el.kind === 'node') {
          const node = doc.value.getNode?.(el.id)
          if (node) {
            const patch = buildPatch(path, value)
            hist.execute(new UpdateNodeCommand(el.id, patch))
          }
        } else if (el.kind === 'graphic') {
          const graphic = doc.value.getGraphic?.(el.id)
          if (graphic) {
            const patch = buildPatch(path, value)
            hist.execute(new UpdateGraphicCommand(el.id, patch))
          }
        }
      }
    } finally {
      hist.endBatch?.()
    }
  }

  return {
    selectedCount,
    getMultiSelectValue,
    setUnifiedValue,
  }
}

export default useMultiSelect
