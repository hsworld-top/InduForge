import type { Ref } from 'vue'
import type { ComponentNode } from '@/editor-core/document/types'
import { watch } from 'vue'
import { elementTypeName } from './property-panel-utils'

/** 将画布节点 type 规范为 manifests 中的标准名（与 PropertyPanel 原逻辑一致） */
export function usePropertyPanelNormalizeNodeType(
  selectedNode: Ref<Pick<ComponentNode, 'id' | 'type'> | null | undefined>,
  updateNode: (nodeId: string, patch: Partial<ComponentNode>) => void,
): void {
  watch(
    () => selectedNode.value?.type,
    (type) => {
      if (!selectedNode.value || !type) return
      const normalized = elementTypeName(type)
      if (!normalized || normalized === type) return
      updateNode(selectedNode.value.id, { type: normalized })
    },
    { immediate: true },
  )
}
