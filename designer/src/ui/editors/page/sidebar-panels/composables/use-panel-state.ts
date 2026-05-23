/**
 * 面板状态计算 Composable
 */

import type { ComponentNode, GraphicNode } from '@/editor-core/document/types'
import { storeToRefs } from 'pinia'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useEditorStore } from '@/stores/editor-store'

export type PanelState = 'page' | 'element' | 'multi'

export interface SelectableRef {
  id: string
  kind: string
}

export function computePanelState(count: number): PanelState {
  if (count === 0) return 'page'
  if (count === 1) return 'element'
  return 'multi'
}

function normalizeType(type: string | undefined): string | undefined {
  if (!type) return type
  if (type === 'Elayout' || type === 'EILayout') return 'ElLayout'
  if (type === 'ElayoutRow' || type === 'EILayoutRow') return 'ElLayoutRow'
  if (type === 'Elcol' || type === 'EICol') return 'ElCol'
  if (type.startsWith('EI')) return `El${type.slice(2)}`
  return type
}

export function usePanelState() {
  const editorStore = useEditorStore()
  const { doc, selection, docVersion, currentPage } = storeToRefs(editorStore)

  const selectedCount = ref(0)
  const selectedElements = ref<SelectableRef[]>([])
  const primaryElement = ref<SelectableRef | null>(null)
  const selectedNode = ref<ComponentNode | null>(null)
  const selectedGraphic = ref<GraphicNode | null>(null)

  let unsubscribeSelection: (() => void) | null = null

  const syncSelection = (payload?: {
    elements?: SelectableRef[]
    primary?: SelectableRef | null
  }) => {
    if (!selection.value) {
      selectedCount.value = 0
      selectedElements.value = []
      primaryElement.value = null
      selectedNode.value = null
      selectedGraphic.value = null
      return
    }

    const elements = selection.value.getSelectedElements?.() || []
    selectedCount.value = elements.length
    selectedElements.value = elements

    const primary = payload?.primary || selection.value.getPrimaryElement?.()
    primaryElement.value = primary || null

    if (primary && doc.value) {
      if (primary.kind === 'node') {
        const node = doc.value.getNode?.(primary.id) || null
        if (node?.type) {
          const normalizedType = normalizeType(node.type)
          if (normalizedType && normalizedType !== node.type) {
            editorStore.updateNode(node.id, { type: normalizedType })
            node.type = normalizedType
          }
        }
        selectedNode.value = node
          ? ({
              ...node,
              type: normalizeType(node.type) ?? node.type,
              props: { ...(node.props || {}) },
              style: { ...(node.style || {}) },
              children: Array.isArray(node.children) ? [...node.children] : node.children,
            } as ComponentNode)
          : null
        selectedGraphic.value = null
      } else if (primary.kind === 'graphic') {
        const graphic = doc.value.getGraphic?.(primary.id) || null
        selectedNode.value = null
        selectedGraphic.value = graphic
          ? ({
              ...graphic,
              props: { ...(graphic.props || {}) },
            } as GraphicNode)
          : null
      } else {
        selectedNode.value = null
        selectedGraphic.value = null
      }
    } else {
      selectedNode.value = null
      selectedGraphic.value = null
    }
  }

  syncSelection()

  const subscribeSelection = (model: { on?: (e: string, fn: () => void) => () => void } | null) => {
    if (!model) return
    unsubscribeSelection = model.on?.('change', () => syncSelection()) ?? null
    syncSelection()
  }

  watch(
    () => selection.value,
    (model) => {
      if (unsubscribeSelection) {
        unsubscribeSelection()
        unsubscribeSelection = null
      }
      if (model) {
        subscribeSelection(model)
      } else {
        syncSelection()
      }
    },
    { immediate: true },
  )

  watch(
    () => docVersion.value,
    () => {
      syncSelection()
    },
  )

  onBeforeUnmount(() => {
    if (unsubscribeSelection) {
      unsubscribeSelection()
      unsubscribeSelection = null
    }
  })

  const panelState = computed((): PanelState => {
    if (selectedCount.value === 1 && selectedNode.value) {
      const page = currentPage.value
      if (page && selectedNode.value.id === page.rootNodeId) {
        return 'page'
      }
    }
    return computePanelState(selectedCount.value)
  })

  return {
    panelState,
    selectedCount,
    selectedElements,
    primaryElement,
    selectedNode,
    selectedGraphic,
  }
}

export default usePanelState
