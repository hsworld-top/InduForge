/**
 * 鑺傜偣鎷栨斁 Composable
 *
 * 浠?NodeRenderer 鎶藉彇鐨?handleDragOver銆乭andleDrop銆佹彃鍏ョ嚎鎸囩ず绛夐€昏緫銆?
 * 渚涢€掑綊娓叉煋鍣ㄥ鐢ㄣ€?
 *
 * @module ui/Canvas/composables/use-node-drop
 */

import type {
  CanvasDocLike,
  InsertLineStyleLike,
  LayoutInsertInfoLike,
  RowInsertInfoLike,
  UseNodeDropDeps,
} from './types'
import type { ComponentNode } from '@/editor-core/document/types'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import {
  canAcceptChildByDescriptor,
  getDefaultSize,
  getDescriptor,
  isContainerType,
  isLayoutContainerType,
  isRegionType,
} from '@/editor-core/descriptors/registry'
import { resolveElContainerMain } from '@/editor-core/utils/layout-utils'
import {
  clampPositionInContainer,
  eventToCanvasPosition,
} from '@/editor-core/utils/placement-utils'
import { calculateInsertIndexWithFallback } from '@/editor-core/utils/placement-resolver'
import {
  DESIGNER_ASSET_DRAG_MIME,
  buildAssetNodeProps,
  parseAssetDragPayload,
  resolveAssetComponentType,
  type DesignerAssetDragPayload,
} from '@/ui/shared/helpers/asset-drag'

/**
 * 鍒ゆ柇瀹瑰櫒鏄惁鍏佽瀛愮粍浠?
 * @param {import('@/editor-core').ComponentNode} parentNode - 鐖惰妭鐐?
 * @param {string} childType - 瀛愮粍浠剁被鍨?
 * @returns {boolean}
 */
function canAcceptChild(parentNode: ComponentNode, childType: string) {
  const currentChildCount = (parentNode.children || []).length
  const descriptor = getDescriptor(parentNode.type)
  if (!descriptor) return false
  return canAcceptChildByDescriptor(parentNode.type, childType, currentChildCount)
}

/**
 * 鍒ゆ柇鑺傜偣鏄惁涓哄彲鏀剧疆瀹瑰櫒
 * @param {import('@/editor-core').ComponentNode | null} targetNode - 鐩爣鑺傜偣
 * @returns {boolean}
 */
function isDroppableContainer(targetNode: ComponentNode | null | undefined) {
  if (!targetNode) return false
  return isContainerType(targetNode.type)
}

/**
 * 鍒ゆ柇鑺傜偣鏄惁涓衡€滄祦寮忓瓙椤光€濈殑瀹瑰櫒锛圓lt 鎷栨嫿鏃堕渶瑕佷笂鎻愬埌澶栧眰锛?
 * @param {import('@/editor-core').ComponentNode | null | undefined} targetNode - 鐩爣鑺傜偣
 * @returns {boolean}
 */
function isFlowDropContainer(targetNode: ComponentNode | null | undefined): boolean {
  if (!targetNode) return false
  if (
    targetNode.type === 'ElLayout' ||
    targetNode.type === 'ElLayoutRow' ||
    targetNode.type === 'ElCol'
  ) {
    return true
  }
  const descriptor = getDescriptor(targetNode.type)
  return Boolean(descriptor?.isContainer && descriptor.childPositioning === 'flow')
}

/** 区域布局自动生成内容区前，先按区域本身计算普通组件的自由落点。 */
function resolvePlainRegionDropPosition(
  event: DragEvent,
  targetNode: ComponentNode | null | undefined,
  targetElement: Element | null | undefined,
  childType: string,
  zoom: number,
) {
  if (
    !targetNode ||
    !isRegionType(targetNode.type) ||
    targetNode.type === 'ElCol' ||
    childType === 'FreeContainer' ||
    isLayoutContainerType(childType) ||
    isContainerType(childType)
  ) {
    return null
  }
  const targetHost = targetElement instanceof HTMLElement ? targetElement : null
  if (!targetHost) return null
  const rawPosition = eventToCanvasPosition(event, targetHost, zoom)
  const defaultSize = getDefaultSize(childType) ?? { width: 120, height: 40 }
  return clampPositionInContainer(rawPosition, targetHost, defaultSize, zoom)
}

function isPlainRegionContentType(childType: string): boolean {
  return (
    Boolean(childType) &&
    childType !== 'FreeContainer' &&
    !isLayoutContainerType(childType) &&
    !isContainerType(childType)
  )
}

function resolveRegionContentHost(
  targetNode: ComponentNode,
  doc: CanvasDocLike,
): { node: ComponentNode; element: Element | null } | null {
  const hostId = (targetNode.children || []).find((childId) => {
    const childNode = doc.getNode?.(childId)
    return childNode?.type === 'FreeContainer'
  })
  const hostNode = hostId ? doc.getNode?.(hostId) : null
  if (!hostNode) return null
  const hostElement = document.querySelector(`[data-node-id="${CSS.escape(hostNode.id)}"]`)
  return { node: hostNode, element: hostElement }
}

function resolveRegionContentHostFromFreeContainer(
  targetNode: ComponentNode,
  doc: CanvasDocLike,
): { node: ComponentNode; element: Element | null } | null {
  if (targetNode.type !== 'FreeContainer') return null
  const parentNode = doc.getParent?.(targetNode.id)
  if (!parentNode || !isRegionType(parentNode.type)) return null
  const hostElement = document.querySelector(`[data-node-id="${CSS.escape(targetNode.id)}"]`)
  return { node: targetNode, element: hostElement }
}

function resolvePlainRegionContentTarget(
  targetNode: ComponentNode | null | undefined,
  targetElement: Element | null | undefined,
  childType: string,
  doc: CanvasDocLike | null | undefined,
): { node: ComponentNode; element: Element | null | undefined } | null {
  if (
    !targetNode ||
    !doc ||
    !isRegionType(targetNode.type) ||
    !isPlainRegionContentType(childType)
  ) {
    return null
  }
  return resolveRegionContentHost(targetNode, doc) || { node: targetNode, element: targetElement }
}

function resolvePlainRegionInsertVisualTarget(
  targetNode: ComponentNode | null | undefined,
  targetElement: Element | null | undefined,
  childType: string,
  doc: CanvasDocLike | null | undefined,
): { node: ComponentNode; element: Element | null | undefined } | null {
  if (!targetNode || !doc || !isPlainRegionContentType(childType)) return null
  const existingHost = resolveRegionContentHostFromFreeContainer(targetNode, doc)
  if (existingHost) return existingHost
  if (!isRegionType(targetNode.type) || targetNode.type === 'ElCol') return null
  return resolveRegionContentHost(targetNode, doc) || { node: targetNode, element: targetElement }
}

/**
 * 浠庨紶鏍囦綅缃В鏋愬彲鏀剧疆瀹瑰櫒
 * @param {DragEvent} event - 鎷栨嫿浜嬩欢
 * @param {string} childType - 瀛愮粍浠剁被鍨?
 * @param {import('@/editor-core').Document} doc - 鏂囨。瀹炰緥
 * @returns {{ node: import('@/editor-core').ComponentNode, element: HTMLElement | null } | null}
 */
function resolveDropContainer(
  event: DragEvent,
  childType: string,
  doc: CanvasDocLike | null | undefined,
  preferOuterDropByAlt = false,
) {
  if (!doc || !event) return null
  const hitList = document.elementsFromPoint(event.clientX, event.clientY)
  let containerNode: ComponentNode | null = null

  const canAcceptByAutoInsert = (
    targetNode: ComponentNode | null | undefined,
    nextChildType: string,
  ) => {
    if (!targetNode || !nextChildType) return false
    if (targetNode.type === 'ElLayout') {
      return nextChildType !== 'ElLayoutRow'
    }
    if (targetNode.type === 'ElLayoutRow') {
      return nextChildType !== 'ElCol'
    }
    if (targetNode.type === 'ElCol') {
      return nextChildType !== 'ElCol'
    }
    return false
  }

  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue
    const nodeElement = hit.closest?.('[data-node-id][data-node-type]')
    if (!nodeElement) continue
    const nodeId = nodeElement.getAttribute('data-node-id')
    const targetNode = nodeId ? doc?.getNode?.(nodeId) : null
    if (!targetNode) continue
    if (preferOuterDropByAlt && isFlowDropContainer(targetNode)) {
      continue
    }
    if (isRegionType(targetNode.type)) {
      const isPlainRegionContent = isPlainRegionContentType(childType)
      if (isPlainRegionContent) {
        const hostTarget = resolveRegionContentHost(targetNode, doc)
        if (hostTarget) return hostTarget
      }
      if (childType && !canAcceptChild(targetNode, childType) && !isPlainRegionContent) continue
      return { node: targetNode, element: nodeElement }
    }
    if (targetNode.type === 'ElContainer') {
      if (!containerNode) {
        containerNode =
          resolveElContainerMain(
            doc as { getNode: (id: string) => ComponentNode | null },
            targetNode,
          ) || targetNode
      }
      continue
    }
    if (isDroppableContainer(targetNode)) {
      if (childType && !canAcceptChild(targetNode, childType)) {
        if (!canAcceptByAutoInsert(targetNode, childType)) continue
      }
      return { node: targetNode, element: nodeElement }
    }
  }
  if (containerNode && (!childType || canAcceptChild(containerNode, childType))) {
    const containerElement = document.querySelector(`[data-node-id="${containerNode.id}"]`)
    return { node: containerNode, element: containerElement }
  }
  return null
}

/**
 * 鍒涘缓鑺傜偣鎷栨斁閫昏緫
 * @param {object} deps - 渚濊禆
 * @param {import('vue').ComputedRef<object>} deps.node - 鑺傜偣 computed
 * @param {import('vue').ComputedRef<object>} deps.doc - 鏂囨。 computed
 * @param {import('vue').ComputedRef<object>} deps.dragState - 鎷栨嫿鐘舵€?computed
 * @param {object} deps.dragDropManager - 鎷栨嫿绠＄悊鍣ㄥ疄渚?
 * @param {import('vue').ComputedRef<boolean>} deps.isContainer - 鏄惁涓哄鍣?computed
 * @param {Function} deps.resolveFlexDirection - 瑙ｆ瀽 Flex 鏂瑰悜鍑芥暟
 * @param {Function} deps.isFlexContainer - 鍒ゆ柇鏄惁涓?Flex 瀹瑰櫒鍑芥暟
 * @param {import('vue').ComputedRef<boolean>} deps.readonly - 鍙妯″紡 computed
 * @param {object} deps.editorStore - 缂栬緫鍣?store 瀹炰緥
 * @param {import('vue').Ref} deps.canvasZoom - 鐢诲竷缂╂斁 ref
 * @param {Function} deps.endDrag - 缁撴潫鎷栨嫿鍑芥暟
 * @param {Function} deps.notifyInsertFailure - 閫氱煡鎻掑叆澶辫触鍑芥暟
 * @param {import('vue').Ref} deps.activeTabName - 褰撳墠婵€娲荤殑 tab 鍚嶇О ref
 * @param {import('vue').ComputedRef} deps.tabsList - tabs 鍒楄〃 computed
 * @returns {object} 杩斿洖 handleDragOver銆乭andleDrop銆乻howInsertLine 绛?
 */
export function useNodeDrop(deps: UseNodeDropDeps) {
  const {
    node,
    doc,
    dragState,
    dragDropManager,
    isContainer,
    resolveFlexDirection,
    isFlexContainer,
    readonly,
    editorStore,
    canvasZoom,
    endDrag,
    notifyInsertFailure,
    activeTabName,
    tabsList,
    activeCollapseName,
    collapseItems,
  } = deps

  /**
   * 鎻掑叆鑺傜偣锛堟嫋鍏ュ満鏅級锛氱姝㈣嚜鍔ㄩ€変腑鏂板缓鑺傜偣
   * @param {string} type - 缁勪欢绫诲瀷
   * @param {string | undefined} parentId - 鐖惰妭鐐?ID
   * @param {number | undefined} index - 鎻掑叆绱㈠紩
   * @param {{ dropPosition?: { x: number; y: number } }} [options] - 鎻掑叆鍙傛暟
   * @returns {import('@/editor-core').ComponentNode | null}
   */
  const insertNodeWithoutSelection = (
    type: string,
    parentId: string | undefined,
    index?: number,
    options: { dropPosition?: { x: number; y: number } | undefined } = {},
  ): ComponentNode | null => {
    const insertOptions: { dropPosition?: { x: number; y: number }; autoSelectInserted: false } = {
      autoSelectInserted: false,
    }
    if (options.dropPosition) {
      insertOptions.dropPosition = options.dropPosition
    }
    return editorStore.insertNode(type, parentId, index, insertOptions)
  }

  const resolveActiveTabKey = (): string => {
    const raw = activeTabName.value || tabsList.value?.[0]?.name || tabsList.value?.[0]?.label || ''
    return String(raw || '').trim()
  }

  const resolveActiveCollapseKey = (): string => {
    const raw =
      activeCollapseName.value ||
      collapseItems.value?.[0]?.name ||
      collapseItems.value?.[0]?.title ||
      collapseItems.value?.[0]?.label ||
      ''
    return String(raw || '').trim()
  }

  /**
   * 瑙ｆ瀽 Tabs/Collapse 娲诲姩鍐呭鍖虹殑钀界偣涓婁笅鏂?
   * @param {import('@/editor-core').ComponentNode | null | undefined} targetNode - 鐩爣瀹瑰櫒
   * @param {DragEvent} event - 鎷栨嫿浜嬩欢
   * @returns {{ key: string, keyProp: 'tabKey' | 'collapseKey', hostElement: Element | null, childIds: string[] } | null}
   */
  const resolveScopedSlotMeta = (
    targetNode: ComponentNode | null | undefined,
    event: DragEvent,
  ) => {
    if (!targetNode || (targetNode.type !== 'Tabs' && targetNode.type !== 'Collapse')) {
      return null
    }
    const isTabs = targetNode.type === 'Tabs'
    const keyProp = isTabs ? 'tabKey' : 'collapseKey'
    const attrName = isTabs ? 'data-tab-key' : 'data-collapse-key'
    const defaultKey = isTabs ? resolveActiveTabKey() : resolveActiveCollapseKey()
    const currentTarget = event.currentTarget instanceof Element ? event.currentTarget : null
    const slotElement = currentTarget?.closest?.(`[${attrName}]`) || null
    let resolvedElement: Element | null = slotElement || currentTarget
    const rawKey =
      slotElement?.getAttribute(attrName) ||
      currentTarget?.getAttribute(attrName) ||
      defaultKey ||
      ''
    const key = String(rawKey || '').trim()
    if (!slotElement && currentTarget && key) {
      const escapedKey =
        typeof CSS !== 'undefined' && typeof CSS.escape === 'function' ? CSS.escape(key) : key
      const scopedSelector = `[data-node-id="${targetNode.id}"][${attrName}="${escapedKey}"]`
      resolvedElement = currentTarget.querySelector(scopedSelector) || resolvedElement
    }
    const childIds = (targetNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId)
      if (!childNode) return false
      const slotKey = childNode.props?.[keyProp]
      if (!slotKey) {
        return Boolean(key)
      }
      return String(slotKey) === key
    })
    return {
      key,
      keyProp,
      hostElement: resolvedElement,
      childIds,
    }
  }

  /**
   * 灏嗘椿鍔ㄥ唴瀹瑰尯鍐呯殑鐩稿鎻掑叆绱㈠紩鏄犲皠涓哄鍣?children 鐨勭粷瀵圭储寮?
   * @param {import('@/editor-core').ComponentNode | null | undefined} targetNode - 鐩爣瀹瑰櫒
   * @param {string[]} scopedChildIds - 褰撳墠娲诲姩鍖哄唴瀛愯妭鐐?
   * @param {number} scopedInsertIndex - 娲诲姩鍖哄唴鎻掑叆绱㈠紩
   * @returns {number}
   */
  const mapScopedInsertIndex = (
    targetNode: ComponentNode | null | undefined,
    scopedChildIds: string[],
    scopedInsertIndex: number,
  ): number => {
    const allChildIds = targetNode?.children || []
    if (!scopedChildIds.length) return allChildIds.length
    const scopedIndices = scopedChildIds
      .map((childId) => allChildIds.indexOf(childId))
      .filter((value) => value >= 0)
      .sort((a, b) => a - b)
    if (!scopedIndices.length) return allChildIds.length
    if (scopedInsertIndex <= 0) return scopedIndices[0] ?? allChildIds.length
    if (scopedInsertIndex >= scopedIndices.length) {
      const lastIndex = scopedIndices[scopedIndices.length - 1]
      return (lastIndex ?? allChildIds.length - 1) + 1
    }
    return scopedIndices[scopedInsertIndex] ?? allChildIds.length
  }

  /**
   * 缁欒惤鍏?Tabs/Collapse 鐨勬柊鑺傜偣鍐欏叆褰掑睘 key
   * @param {import('@/editor-core').ComponentNode | null} inserted - 鏂版彃鍏ヨ妭鐐?
   * @param {import('@/editor-core').ComponentNode | null | undefined} targetNode - 鐩爣瀹瑰櫒
   */
  const applyScopedKeyForInsertedNode = (
    inserted: ComponentNode | null,
    targetNode: ComponentNode | null | undefined,
    scopedKey?: string | null,
  ) => {
    if (!inserted || !targetNode) return
    if (targetNode.type === 'Tabs') {
      const tabKey = String(scopedKey || resolveActiveTabKey() || '').trim()
      if (tabKey) {
        editorStore.updateNode(inserted.id, {
          props: { ...(inserted.props || {}), tabKey: String(tabKey) },
        })
      }
      return
    }
    if (targetNode.type === 'Collapse') {
      const collapseKey = String(scopedKey || resolveActiveCollapseKey() || '').trim()
      if (collapseKey) {
        editorStore.updateNode(inserted.id, {
          props: { ...(inserted.props || {}), collapseKey: String(collapseKey) },
        })
      }
    }
  }

  /**
   * 瑙ｆ瀽鎷栨嫿涓殑璧勬簮 payload銆?
   * @param {DragEvent} event - 鎷栨嫿浜嬩欢
   * @returns {DesignerAssetDragPayload | null}
   */
  const resolveDraggedAssetPayload = (event: DragEvent): DesignerAssetDragPayload | null => {
    const raw = event.dataTransfer?.getData(DESIGNER_ASSET_DRAG_MIME) || ''
    return parseAssetDragPayload(raw)
  }

  /**
   * 灏嗚祫婧愪俊鎭啓鍏ユ柊寤鸿妭鐐瑰睘鎬с€?
   * @param {ComponentNode | null} inserted - 鏂版彃鍏ヨ妭鐐?
   * @param {string} nodeType - 鑺傜偣绫诲瀷
   * @param {DesignerAssetDragPayload | null} assetPayload - 璧勬簮 payload
   */
  const applyAssetPayloadToInsertedNode = (
    inserted: ComponentNode | null,
    nodeType: string,
    assetPayload: DesignerAssetDragPayload | null,
  ): void => {
    if (!inserted || !assetPayload) return
    const componentType =
      nodeType === 'Image' ? 'Image' : nodeType === 'Video' ? 'Video' : 'DownloadLink'
    const patchProps = buildAssetNodeProps(assetPayload, componentType)
    if (!patchProps || Object.keys(patchProps).length === 0) return
    editorStore.updateNode(inserted.id, {
      props: { ...(inserted.props || {}), ...patchProps },
    })
  }

  // 鎷栨嫿鐘舵€?
  const isDragOver = ref(false)
  const showInsertLine = ref(false)
  const insertLineStyle = ref<InsertLineStyleLike | null>(null)
  const genericInsertLineBox = ref<{
    left: number
    top: number
    width: number
    height: number
  } | null>(null)
  const rowInsertInfo = ref<RowInsertInfoLike | null>(null)
  const layoutInsertInfo = ref<LayoutInsertInfoLike | null>(null)
  const rowInsertEdgeThreshold = 8
  const colInsertEdgeThreshold = 8

  const insertLineBox = computed(
    () =>
      rowInsertInfo.value?.lineBox ||
      layoutInsertInfo.value?.lineBox ||
      genericInsertLineBox.value ||
      null,
  )
  const altKeyPressed = ref(false)

  /**
   * 娓呯悊褰撳墠鑺傜偣鐨勬嫋鎷借瑙夌姸鎬侊紙楂樹寒銆佹彃鍏ョ嚎銆佸揩鐓э級
   */
  const clearDropVisualState = () => {
    isDragOver.value = false
    showInsertLine.value = false
    insertLineStyle.value = null
    genericInsertLineBox.value = null
    rowInsertInfo.value = null
    layoutInsertInfo.value = null
  }

  /**
   * 璁＄畻褰撳墠鏄惁搴斾紭鍏堣蛋 Alt 涓婂眰鏀剧疆璇箟
   * - 鏀寔鎷栨嫿杩囩▼涓腑閫旀寜涓?Alt锛堜笉鍙緷璧?DragEvent.altKey锛?
   */
  const shouldPreferOuterDropByAlt = (event: { altKey?: boolean } | null | undefined) => {
    return Boolean(event?.altKey || altKeyPressed.value)
  }

  const handleGlobalAltState = (event: KeyboardEvent) => {
    if (event.key !== 'Alt') return
    altKeyPressed.value = event.type === 'keydown'
  }
  const handleGlobalDragOverState = (event: DragEvent) => {
    if (!dragState.dragType) return
    altKeyPressed.value = Boolean(event.altKey)
  }
  const handleGlobalDragEndState = () => {
    altKeyPressed.value = false
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('keydown', handleGlobalAltState)
    window.addEventListener('keyup', handleGlobalAltState)
    window.addEventListener('dragover', handleGlobalDragOverState as EventListener)
    window.addEventListener('drop', handleGlobalDragEndState)
    window.addEventListener('dragend', handleGlobalDragEndState)
  }

  onBeforeUnmount(() => {
    if (typeof window === 'undefined') return
    window.removeEventListener('keydown', handleGlobalAltState)
    window.removeEventListener('keyup', handleGlobalAltState)
    window.removeEventListener('dragover', handleGlobalDragOverState as EventListener)
    window.removeEventListener('drop', handleGlobalDragEndState)
    window.removeEventListener('dragend', handleGlobalDragEndState)
  })

  watch(
    () => dragState.dragType,
    (value) => {
      if (!value) {
        altKeyPressed.value = false
        clearDropVisualState()
      }
    },
  )

  watch(
    [() => dragState.dragType, altKeyPressed, () => node.value],
    ([dragType, isAltActive, currentNode]) => {
      if (!dragType || !isAltActive) return
      if (!isFlowDropContainer(currentNode as ComponentNode | null | undefined)) return
      clearDropVisualState()
    },
  )

  const suppressDropByAlt = computed(() => {
    if (!dragState.dragType) return false
    if (!altKeyPressed.value) return false
    return isFlowDropContainer(node.value)
  })

  /**
   * 澶勭悊鎷栨嫿鎮仠
   * @param {DragEvent} event - 鎷栨嫿浜嬩欢
   */
  const handleDragOver = (event: DragEvent) => {
    if (readonly.value) return

    const resolveLayoutInsertFromPoint = () => {
      const hitList = document.elementsFromPoint(event.clientX, event.clientY)
      for (const hit of hitList) {
        const layoutElement = hit?.closest?.('[data-node-type="ElLayout"][data-node-id]')
        if (!layoutElement) continue
        const layoutId = layoutElement.getAttribute('data-node-id')
        const layoutNode = layoutId ? doc.value?.getNode?.(layoutId) : null
        if (!layoutNode || layoutNode.type !== 'ElLayout') continue

        const rowIds = (layoutNode.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId)
          return childNode?.type === 'ElLayoutRow'
        })
        if (rowIds.length === 0) return null

        const pointY = event.clientY
        const threshold = rowInsertEdgeThreshold
        for (let i = 0; i < rowIds.length; i += 1) {
          const rowId = rowIds[i]
          const rowElement = layoutElement.querySelector(`[data-node-id="${rowId}"]`)
          if (!rowElement) continue
          const rect = rowElement.getBoundingClientRect?.()
          if (!rect) continue
          if (Math.abs(pointY - rect.top) <= threshold) {
            return { layoutNode, layoutElement, index: i, lineY: rect.top }
          }
          if (Math.abs(pointY - rect.bottom) <= threshold) {
            return {
              layoutNode,
              layoutElement,
              index: i + 1,
              lineY: rect.bottom,
            }
          }
        }
      }
      return null
    }

    const resolveRowInsertFromPath = () => {
      const path = event.composedPath?.() || []
      for (const item of path) {
        if (!(item instanceof Element)) continue
        const nodeElement = item.closest?.('[data-node-id][data-node-type]')
        if (!nodeElement) continue
        const nodeType = nodeElement.getAttribute('data-node-type')
        if (nodeType !== 'ElCol') continue
        const nodeId = nodeElement.getAttribute('data-node-id')
        const colNode = nodeId ? doc.value?.getNode?.(nodeId) : null
        if (!colNode) continue
        const parentNode = doc.value?.getParent?.(colNode.id)
        if (parentNode?.type !== 'ElLayoutRow') continue
        const rowSelector = `[data-node-id="${parentNode.id}"]`
        const rowElement = document.querySelector(rowSelector) || nodeElement.closest?.(rowSelector)
        if (!rowElement) continue
        const colRect = nodeElement.getBoundingClientRect?.()
        const edgeThreshold = colInsertEdgeThreshold
        const nearEdge = colRect
          ? event.clientX - colRect.left <= edgeThreshold ||
            colRect.right - event.clientX <= edgeThreshold
          : false
        return { rowElement, parentNode, nearEdge, colRect }
      }
      return null
    }
    const resolveLayoutInsertFromPath = () => {
      const path = event.composedPath?.() || []
      for (const item of path) {
        if (!(item instanceof Element)) continue
        const nodeElement = item.closest?.('[data-node-id][data-node-type]')
        if (!nodeElement) continue
        const nodeType = nodeElement.getAttribute('data-node-type')
        if (nodeType !== 'ElLayoutRow') continue
        const nodeId = nodeElement.getAttribute('data-node-id')
        const rowNode = nodeId ? doc.value?.getNode?.(nodeId) : null
        if (!rowNode) continue
        const parentNode = doc.value?.getParent?.(rowNode.id)
        if (parentNode?.type !== 'ElLayout') continue
        const layoutSelector = `[data-node-id="${parentNode.id}"]`
        const layoutElement =
          document.querySelector(layoutSelector) || nodeElement.closest?.(layoutSelector)
        if (!layoutElement) continue
        const rowRect = nodeElement.getBoundingClientRect?.()
        const edgeThreshold = rowInsertEdgeThreshold
        const nearEdge = rowRect
          ? event.clientY - rowRect.top <= edgeThreshold ||
            rowRect.bottom - event.clientY <= edgeThreshold
          : false
        return { layoutElement, parentNode, rowElement: nodeElement, nearEdge }
      }
      return null
    }

    if (!isContainer.value) return
    const assetPayload = resolveDraggedAssetPayload(event)
    const hasComponent =
      event.dataTransfer?.types?.includes('application/x-designer-component') ||
      event.dataTransfer?.types?.includes('application/x-designer-node') ||
      event.dataTransfer?.types?.includes(DESIGNER_ASSET_DRAG_MIME) ||
      event.dataTransfer?.types?.includes('text/plain') ||
      Boolean(dragState.dragType)
    if (!hasComponent) return
    const preferOuterDropByAlt = shouldPreferOuterDropByAlt(event)
    if (preferOuterDropByAlt && isFlowDropContainer(node.value)) {
      clearDropVisualState()
      return
    }
    // 闃绘浜嬩欢鍐掓场
    event.stopPropagation()

    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'copy'
    }
    isDragOver.value = true

    // ElLayoutRow 鍐呮嫋鍏ョ粍浠舵椂锛屼紭鍏堟彁绀哄乏鍙虫彃鍏?
    const payload =
      event.dataTransfer?.getData('application/x-designer-component') ||
      event.dataTransfer?.getData('application/x-designer-node') ||
      event.dataTransfer?.getData('text/plain')
    const fallbackType = dragState.dragType || ''
    let dragType = ''
    if (payload) {
      try {
        const parsed = JSON.parse(payload)
        dragType = parsed?.type || ''
      } catch {
        dragType = payload
      }
    }
    if (!dragType && assetPayload) {
      dragType = resolveAssetComponentType(assetPayload)
    }
    dragType = dragType || fallbackType

    if (dragType && dragType !== 'ElLayoutRow') {
      const resolvedLayout = resolveLayoutInsertFromPoint()
      if (resolvedLayout) {
        const { layoutElement, layoutNode, index, lineY } = resolvedLayout
        const layoutRect = layoutElement.getBoundingClientRect?.()
        if (layoutRect) {
          showInsertLine.value = true
          insertLineStyle.value = {
            orientation: 'horizontal',
            offset: Math.max(0, lineY - layoutRect.top),
          }
          genericInsertLineBox.value = null
          rowInsertInfo.value = null
          layoutInsertInfo.value = {
            layoutId: layoutNode.id,
            index,
            lineBox: {
              left: layoutRect.left,
              top: layoutRect.top,
              width: layoutRect.width,
              height: layoutRect.height,
            },
          }
          return
        }
      } else {
        layoutInsertInfo.value = null
      }
    }

    if (dragType && dragType !== 'ElLayoutRow') {
      const resolvedLayout = resolveLayoutInsertFromPath()
      if (resolvedLayout) {
        const { layoutElement, parentNode, nearEdge } = resolvedLayout
        if (!nearEdge) {
          showInsertLine.value = false
          insertLineStyle.value = null
          genericInsertLineBox.value = null
          layoutInsertInfo.value = null
        } else {
          const layoutRect = layoutElement.getBoundingClientRect?.()
          if (layoutRect) {
            const direction = resolveFlexDirection('ElLayout', layoutElement)
            const insertInfo = dragDropManager.calculateFlexInsertPosition(
              layoutElement,
              event,
              direction,
            )
            const insertLine = insertInfo.insertLine || {
              orientation: 'horizontal',
              offset: layoutRect.bottom - layoutRect.top,
            }
            const adjustedLine = {
              ...insertLine,
              offset: Math.max(0, insertLine.offset),
            }
            showInsertLine.value = true
            insertLineStyle.value = adjustedLine
            genericInsertLineBox.value = null
            rowInsertInfo.value = null
            layoutInsertInfo.value = {
              layoutId: parentNode.id,
              index: insertInfo.index,
              lineBox: {
                left: layoutRect.left,
                top: layoutRect.top,
                width: layoutRect.width,
                height: layoutRect.height,
              },
            }
            return
          }
        }
      } else {
        layoutInsertInfo.value = null
      }
    }

    if (dragType && dragType !== 'ElCol') {
      const resolvedRow = resolveRowInsertFromPath()
      if (resolvedRow) {
        const { rowElement, parentNode, nearEdge } = resolvedRow
        if (!nearEdge) {
          showInsertLine.value = false
          insertLineStyle.value = null
          genericInsertLineBox.value = null
          rowInsertInfo.value = null
          layoutInsertInfo.value = null
          return
        }
        const rowRect = rowElement.getBoundingClientRect?.()
        if (rowRect) {
          const direction = resolveFlexDirection('ElLayoutRow', rowElement)
          const insertInfo = dragDropManager.calculateFlexInsertPosition(
            rowElement,
            event,
            direction,
          )
          const rowRectSnapshot = rowRect
          const insertLine = insertInfo.insertLine || {
            orientation: 'vertical',
            offset: rowRectSnapshot.right - rowRectSnapshot.left,
          }
          const hostRect = rowElement.getBoundingClientRect?.()
          if (hostRect && insertLine) {
            const rawOffset =
              insertLine.orientation === 'vertical' ? insertLine.offset : insertLine.offset
            const adjustedLine = {
              ...insertLine,
              offset: Math.max(0, rawOffset),
            }
            showInsertLine.value = true
            insertLineStyle.value = adjustedLine
            genericInsertLineBox.value = null
            rowInsertInfo.value = {
              rowId: parentNode.id,
              index: insertInfo.index,
              lineBox: {
                left: rowRectSnapshot.left,
                top: rowRectSnapshot.top,
                width: rowRectSnapshot.width,
                height: rowRectSnapshot.height,
              },
            }
            layoutInsertInfo.value = null
            return
          }
        }
      }
    }

    // ElCol 渚ц竟鎻掑叆鎻愮ず鐢变笂闈㈢殑 resolveRowInsertFromPath 澶勭悊

    const regionVisualTarget = resolvePlainRegionInsertVisualTarget(
      node.value,
      event.currentTarget instanceof Element ? event.currentTarget : null,
      dragType,
      doc.value,
    )
    if (regionVisualTarget) {
      rowInsertInfo.value = null
      layoutInsertInfo.value = null
      const hostElement =
        regionVisualTarget.element instanceof HTMLElement
          ? regionVisualTarget.element
          : document.querySelector(`[data-node-id="${CSS.escape(regionVisualTarget.node.id)}"]`)
      if (!hostElement) {
        showInsertLine.value = false
        insertLineStyle.value = null
        genericInsertLineBox.value = null
        return
      }
      const hostRect = hostElement.getBoundingClientRect?.()
      if (!hostRect) {
        showInsertLine.value = false
        insertLineStyle.value = null
        genericInsertLineBox.value = null
        return
      }
      const childIds = regionVisualTarget.node.children || []
      if (childIds.length === 0) {
        showInsertLine.value = false
        insertLineStyle.value = null
        genericInsertLineBox.value = null
        return
      }
      const insertInfo = dragDropManager.calculateFlexInsertPosition(hostElement, event, 'column')
      if (insertInfo.insertLine) {
        showInsertLine.value = true
        insertLineStyle.value = insertInfo.insertLine
        genericInsertLineBox.value = {
          left: hostRect.left,
          top: hostRect.top,
          width: hostRect.width,
          height: hostRect.height,
        }
      } else {
        showInsertLine.value = false
        insertLineStyle.value = null
        genericInsertLineBox.value = null
      }
      return
    }

    if (node.value?.type === 'Tabs' || node.value?.type === 'Collapse') {
      rowInsertInfo.value = null
      layoutInsertInfo.value = null
      const scopedMeta = resolveScopedSlotMeta(node.value, event)
      const hostElement = scopedMeta?.hostElement
      const scopedChildIds = scopedMeta?.childIds || []
      if (!hostElement) {
        showInsertLine.value = false
        insertLineStyle.value = null
        genericInsertLineBox.value = null
        return
      }
      if (!scopedChildIds.length) {
        showInsertLine.value = false
        insertLineStyle.value = null
        genericInsertLineBox.value = null
        return
      }
      const insertInfo = dragDropManager.calculateFlexInsertPosition(hostElement, event, 'column')
      if (insertInfo.insertLine) {
        showInsertLine.value = true
        insertLineStyle.value = insertInfo.insertLine
        const hostRect = hostElement.getBoundingClientRect?.()
        genericInsertLineBox.value = hostRect
          ? {
              left: hostRect.left,
              top: hostRect.top,
              width: hostRect.width,
              height: hostRect.height,
            }
          : null
      } else {
        showInsertLine.value = false
        insertLineStyle.value = null
        genericInsertLineBox.value = null
      }
      return
    }

    // 璁＄畻鎻掑叆浣嶇疆
    if (node.value?.type && isFlexContainer(node.value.type)) {
      rowInsertInfo.value = null
      layoutInsertInfo.value = null
      const hasExistingChildren = (node.value.children || []).length > 0
      if (!hasExistingChildren) {
        showInsertLine.value = false
        insertLineStyle.value = null
        genericInsertLineBox.value = null
        return
      }
      const outerElement = event.currentTarget instanceof Element ? event.currentTarget : null
      const contentElement =
        outerElement?.querySelector?.('[data-node-id]')?.parentElement || outerElement
      if (!contentElement) return
      const direction = resolveFlexDirection(node.value.type, contentElement)
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        contentElement,
        event,
        direction,
      )

      if (insertInfo.insertLine) {
        showInsertLine.value = true
        insertLineStyle.value = insertInfo.insertLine
        const contentRect = contentElement.getBoundingClientRect?.()
        genericInsertLineBox.value = contentRect
          ? {
              left: contentRect.left,
              top: contentRect.top,
              width: contentRect.width,
              height: contentRect.height,
            }
          : null
      } else {
        showInsertLine.value = false
        insertLineStyle.value = null
        genericInsertLineBox.value = null
      }
    }
  }

  /**
   * 澶勭悊鎷栨嫿鏀剧疆
   * @param {DragEvent} event - 鎷栨嫿浜嬩欢
   */
  const handleDrop = (event: DragEvent) => {
    if (readonly.value) return
    const preferOuterDropByAlt = shouldPreferOuterDropByAlt(event)
    if (preferOuterDropByAlt && isFlowDropContainer(node.value)) {
      clearDropVisualState()
      return
    }
    // 闃绘浜嬩欢鍐掓场锛岄伩鍏嶉噸澶嶆彃鍏?
    event.stopPropagation()

    const rowInsertSnapshot = rowInsertInfo.value
    const layoutInsertSnapshot = layoutInsertInfo.value
    isDragOver.value = false
    showInsertLine.value = false
    insertLineStyle.value = null
    genericInsertLineBox.value = null
    rowInsertInfo.value = null
    layoutInsertInfo.value = null

    const resolveLayoutInsertFromPoint = () => {
      const hit = event.target instanceof Element ? event.target : null
      const layoutElement = hit?.closest?.('[data-node-type="ElLayout"][data-node-id]')
      if (!layoutElement) return null
      const layoutId = layoutElement.getAttribute('data-node-id')
      const layoutNode = layoutId ? doc.value?.getNode?.(layoutId) : null
      if (!layoutNode || layoutNode.type !== 'ElLayout') return null

      const rowIds = (layoutNode.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId)
        return childNode?.type === 'ElLayoutRow'
      })
      if (rowIds.length === 0) return null

      const pointY = event.clientY
      const threshold = rowInsertEdgeThreshold
      for (let i = 0; i < rowIds.length; i += 1) {
        const rowId = rowIds[i]
        const rowElement = layoutElement.querySelector(`[data-node-id="${rowId}"]`)
        if (!rowElement) continue
        const rect = rowElement.getBoundingClientRect?.()
        if (!rect) continue
        if (Math.abs(pointY - rect.top) <= threshold) {
          return { layoutNode, layoutElement, index: i }
        }
        if (Math.abs(pointY - rect.bottom) <= threshold) {
          return { layoutNode, layoutElement, index: i + 1 }
        }
      }
      return null
    }
    /**
     * 鍦?ElLayout 涓寜琛屾彃鍏ョ粍浠?
     * @param {import('@/editor-core').ComponentNode} layoutNode - 甯冨眬鑺傜偣
     * @param {number} insertIndex - 琛屾彃鍏ヤ綅缃?
     * @param {string} componentType - 缁勪欢绫诲瀷
     */
    const insertIntoLayoutByRow = (
      layoutNode: ComponentNode | null | undefined,
      insertIndex: number,
      componentType: string,
    ) => {
      if (!layoutNode || layoutNode.type !== 'ElLayout') return null
      const rowNode = insertNodeWithoutSelection('ElLayoutRow', layoutNode.id, insertIndex)
      if (!rowNode) return null
      const latestLayout = doc.value?.getNode?.(layoutNode.id)
      const rowCount = (latestLayout?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId)
        return childNode?.type === 'ElLayoutRow'
      }).length
      editorStore.updateNode(layoutNode.id, {
        props: {
          ...(latestLayout?.props || layoutNode.props || {}),
          rows: Math.max(1, rowCount),
        },
      })
      editorStore.updateNode(rowNode.id, {
        props: { ...(rowNode.props || {}), columns: 1 },
      })
      const latestRow = doc.value?.getNode?.(rowNode.id)
      const colIds = (latestRow?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId)
        return childNode?.type === 'ElCol'
      })
      let colId = colIds[0]
      if (!colId) {
        const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, 0)
        if (!colNode) return null
        colId = colNode.id
      }
      return insertNodeWithoutSelection(componentType, colId)
    }

    const resolveRowInsertTarget = () => {
      const path = event.composedPath?.() || []
      for (const item of path) {
        if (!(item instanceof Element)) continue
        const nodeElement = item.closest?.('[data-node-id][data-node-type]')
        if (!nodeElement) continue
        const nodeType = nodeElement.getAttribute('data-node-type')
        if (nodeType !== 'ElCol') continue
        const nodeId = nodeElement.getAttribute('data-node-id')
        const colNode = nodeId ? doc.value?.getNode?.(nodeId) : null
        if (!colNode) continue
        const parentNode = doc.value?.getParent?.(colNode.id)
        if (parentNode?.type !== 'ElLayoutRow') continue
        const rowSelector = `[data-node-id="${parentNode.id}"]`
        const rowElement = document.querySelector(rowSelector) || nodeElement.closest?.(rowSelector)
        const colRect = nodeElement.getBoundingClientRect?.()
        const edgeThreshold = colInsertEdgeThreshold
        const nearEdge = colRect
          ? event.clientX - colRect.left <= edgeThreshold ||
            colRect.right - event.clientX <= edgeThreshold
          : false
        let index = null
        if (colRect && nearEdge) {
          const colIds = (parentNode.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          })
          const currentIndex = Math.max(0, colIds.indexOf(colNode.id))
          const nearLeft = event.clientX - colRect.left <= edgeThreshold
          index = nearLeft ? currentIndex : currentIndex + 1
        }
        return { rowNode: parentNode, rowElement, nearEdge, index }
      }
      return null
    }
    const resolveLayoutInsertTarget = () => {
      const path = event.composedPath?.() || []
      for (const item of path) {
        if (!(item instanceof Element)) continue
        const nodeElement = item.closest?.('[data-node-id][data-node-type]')
        if (!nodeElement) continue
        const nodeType = nodeElement.getAttribute('data-node-type')
        if (nodeType !== 'ElLayoutRow') continue
        const nodeId = nodeElement.getAttribute('data-node-id')
        const rowNode = nodeId ? doc.value?.getNode?.(nodeId) : null
        if (!rowNode) continue
        const parentNode = doc.value?.getParent?.(rowNode.id)
        if (parentNode?.type !== 'ElLayout') continue
        const layoutSelector = `[data-node-id="${parentNode.id}"]`
        const layoutElement =
          document.querySelector(layoutSelector) || nodeElement.closest?.(layoutSelector)
        const rowRect = nodeElement.getBoundingClientRect?.()
        const edgeThreshold = rowInsertEdgeThreshold
        const nearEdge = rowRect
          ? event.clientY - rowRect.top <= edgeThreshold ||
            rowRect.bottom - event.clientY <= edgeThreshold
          : false
        return { layoutNode: parentNode, layoutElement, nearEdge }
      }
      return null
    }

    const resolveElContainerTarget = () => {
      const currentType = node.value?.type
      const isElContainerScope =
        currentType === 'ElContainer' ||
        currentType === 'ElHeader' ||
        currentType === 'ElAside' ||
        currentType === 'ElMain' ||
        currentType === 'ElFooter'
      if (!isElContainerScope) return null
      const resolveLayoutTarget = (element: Element | null | undefined) => {
        if (!element) return null
        const nodeElement = element.closest?.('[data-node-id][data-node-type]')
        if (!nodeElement) return null
        const nodeType = nodeElement.getAttribute('data-node-type')
        if (nodeType !== 'ElLayout' && nodeType !== 'ElLayoutRow') return null
        const nodeId = nodeElement.getAttribute('data-node-id')
        const targetNode = nodeId ? doc.value?.getNode?.(nodeId) : null
        if (!targetNode) return null
        return { node: targetNode, element: nodeElement }
      }
      const resolveNodeFromId = (nodeId: string | null | undefined) => {
        if (!nodeId) return null
        const targetNode = doc.value?.getNode?.(nodeId)
        if (!targetNode) return null
        if (isRegionType(targetNode.type)) {
          return { node: targetNode, element: null }
        }
        if (targetNode.type === 'ElContainer') {
          const mainNode =
            resolveElContainerMain(
              doc.value as { getNode: (id: string) => ComponentNode | null } | null | undefined,
              targetNode,
            ) || targetNode
          return { node: mainNode, element: null }
        }
        return null
      }
      const resolveFromElement = (element: Element | null | undefined) => {
        if (!element) return null
        const layoutTarget = resolveLayoutTarget(element)
        if (layoutTarget) return layoutTarget
        const nodeElement = element.closest?.('[data-node-id]')
        if (!nodeElement) return null
        const nodeId = nodeElement.getAttribute('data-node-id')
        const resolved = resolveNodeFromId(nodeId)
        if (!resolved) return null
        return { node: resolved.node, element: nodeElement }
      }

      const hitList = document.elementsFromPoint(event.clientX, event.clientY)
      for (const item of hitList) {
        if (!(item instanceof Element)) continue
        const resolved = resolveFromElement(item)
        if (resolved) return resolved
      }

      const path = event.composedPath?.() || []
      for (const item of path) {
        if (!(item instanceof Element)) continue
        const resolved = resolveFromElement(item)
        if (resolved) return resolved
      }

      const hit = document.elementFromPoint(event.clientX, event.clientY)
      const resolvedHit = resolveFromElement(hit)
      if (resolvedHit) return resolvedHit

      return null
    }

    const assetPayload = resolveDraggedAssetPayload(event)
    const payload =
      event.dataTransfer?.getData('application/x-designer-component') ||
      event.dataTransfer?.getData('application/x-designer-node') ||
      event.dataTransfer?.getData('text/plain')
    const fallbackType = dragState.dragType || ''
    const assetComponentType = assetPayload ? resolveAssetComponentType(assetPayload) : ''
    const insertNodeByResolvedType = (
      type: string,
      parentId: string | undefined,
      index?: number,
      options: { dropPosition?: { x: number; y: number } | undefined } = {},
    ) => {
      // D-05: Fast drag-drop insertIndex staleness fix
      // 楠岃瘉 insertIndex 鏈夋晥鎬э紝濡傛灉鐩爣鑺傜偣涓嶅瓨鍦ㄥ垯 fallback 鍒?append
      let validIndex = index
      if (validIndex !== undefined && parentId) {
        const parentNode = doc.value?.getNode?.(parentId)
        const siblings = (parentNode?.children || []) as string[]
        if (validIndex < 0 || validIndex > siblings.length || !siblings[validIndex]) {
          // index 鏃犳晥锛堣秴鍑鸿寖鍥存垨鎸囧悜宸插垹闄よ妭鐐癸級锛宖allback 鍒?append
          validIndex = siblings.length
        }
      }
      const inserted = insertNodeWithoutSelection(type, parentId, validIndex, options)
      applyAssetPayloadToInsertedNode(inserted, type, assetPayload)
      return inserted
    }
    const insertIntoLayoutByRowAndApplyAsset = (
      layoutNode: ComponentNode | null | undefined,
      insertIndex: number,
      componentType: string,
    ) => {
      const inserted = insertIntoLayoutByRow(layoutNode, insertIndex, componentType)
      applyAssetPayloadToInsertedNode(inserted, componentType, assetPayload)
      return inserted
    }

    try {
      const parsed = JSON.parse(payload || '{}') as { type?: string }
      const type = parsed?.type || ''
      const resolvedType = type || assetComponentType || fallbackType
      if (!resolvedType) return
      if (!node.value) return
      if (!layoutInsertSnapshot) {
        const resolvedLayoutByPoint = resolveLayoutInsertFromPoint()
        if (resolvedLayoutByPoint && resolvedType !== 'ElLayoutRow') {
          const inserted = insertIntoLayoutByRowAndApplyAsset(
            resolvedLayoutByPoint.layoutNode,
            resolvedLayoutByPoint.index,
            resolvedType,
          )
          if (!inserted) {
            notifyInsertFailure()
          }
          endDrag()
          return
        }
      }
      if (
        layoutInsertSnapshot?.layoutId &&
        resolvedType !== 'ElLayoutRow' &&
        doc.value?.getNode?.(layoutInsertSnapshot.layoutId)?.type === 'ElLayout'
      ) {
        const layoutNode = doc.value.getNode(layoutInsertSnapshot.layoutId)
        const inserted = insertIntoLayoutByRowAndApplyAsset(
          layoutNode,
          layoutInsertSnapshot.index,
          resolvedType,
        )
        if (!inserted) notifyInsertFailure()
        endDrag()
        return
      }
      const resolvedTarget = resolveElContainerTarget()
      let targetNode = (resolvedTarget && resolvedTarget.node) || node.value
      let targetElement =
        (resolvedTarget && resolvedTarget.element) ||
        (event.currentTarget instanceof Element ? event.currentTarget : null)
      if (!isDroppableContainer(targetNode)) {
        const resolvedContainer = resolveDropContainer(
          event,
          resolvedType,
          doc.value,
          preferOuterDropByAlt,
        )
        if (resolvedContainer) {
          targetNode = resolvedContainer.node
          targetElement = resolvedContainer.element || targetElement
        } else {
          return
        }
      }
      const rowResolved = resolveRowInsertTarget()
      const layoutResolved = resolveLayoutInsertTarget()
      const layoutResolvedByPoint = resolveLayoutInsertFromPoint()
      if (
        rowResolved &&
        resolvedType !== 'ElCol' &&
        rowResolved.nearEdge &&
        rowInsertSnapshot?.rowId === rowResolved.rowNode.id
      ) {
        targetNode = rowResolved.rowNode
        targetElement = rowResolved.rowElement || targetElement
      }
      if (
        targetNode?.type === 'ElCol' &&
        resolvedType !== 'ElCol' &&
        doc.value?.getParent?.(targetNode.id)?.type === 'ElLayoutRow' &&
        rowInsertSnapshot?.rowId === doc.value.getParent(targetNode.id)?.id
      ) {
        const parentNode = doc.value.getParent(targetNode.id)
        if (parentNode) {
          targetNode = parentNode
          const rowSelector = `[data-node-id="${parentNode.id}"]`
          const currentTarget = event.currentTarget instanceof Element ? event.currentTarget : null
          targetElement =
            document.querySelector(rowSelector) ||
            currentTarget?.closest?.(rowSelector) ||
            targetElement
        }
      }
      if (
        targetNode?.type === 'ElCol' &&
        resolvedType !== 'ElCol' &&
        doc.value?.getParent?.(targetNode.id)?.type === 'ElLayoutRow'
      ) {
        const rowNode = doc.value.getParent(targetNode.id)
        const layoutNode = rowNode ? doc.value.getParent?.(rowNode.id) : null
        if (
          layoutInsertSnapshot?.layoutId &&
          layoutNode?.type === 'ElLayout' &&
          layoutInsertSnapshot.layoutId === layoutNode.id &&
          resolvedType !== 'ElLayoutRow'
        ) {
          const rowNodeInserted = insertNodeWithoutSelection(
            'ElLayoutRow',
            layoutNode.id,
            layoutInsertSnapshot.index,
          )
          if (rowNodeInserted) {
            const latestLayout = doc.value?.getNode?.(layoutNode.id)
            const rowCount = (latestLayout?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId)
              return childNode?.type === 'ElLayoutRow'
            }).length
            editorStore.updateNode(layoutNode.id, {
              props: {
                ...(latestLayout?.props || layoutNode.props || {}),
                rows: Math.max(1, rowCount),
              },
            })
            editorStore.updateNode(rowNodeInserted.id, {
              props: { ...(rowNodeInserted.props || {}), columns: 1 },
            })
            const latestRow = doc.value?.getNode?.(rowNodeInserted.id)
            const colIds = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId)
              return childNode?.type === 'ElCol'
            })
            let colId = colIds[0]
            if (!colId) {
              const colNode = insertNodeWithoutSelection('ElCol', rowNodeInserted.id, 0)
              if (!colNode) {
                notifyInsertFailure()
                endDrag()
                return
              }
              colId = colNode.id
            }
            const inserted = insertNodeByResolvedType(resolvedType, colId)
            if (!inserted) {
              notifyInsertFailure()
            }
          } else {
            notifyInsertFailure()
          }
          endDrag()
          return
        }
        const colHasChild = (targetNode.children || []).length > 0
        if (colHasChild) {
          const rowResolved = resolveRowInsertTarget()
          if (!rowNode) {
            notifyInsertFailure()
            endDrag()
            return
          }
          const colIds = (rowNode?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          })
          const currentIndex = Math.max(0, colIds.indexOf(targetNode.id))
          let insertIndex = currentIndex + 1
          if (rowInsertSnapshot?.rowId === rowNode.id) {
            insertIndex = rowInsertSnapshot.index
          } else if (Number.isInteger(rowResolved?.index)) {
            insertIndex = rowResolved?.index ?? insertIndex
          }
          const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, insertIndex)
          if (colNode) {
            const latestRow = doc.value?.getNode?.(rowNode.id)
            const colCount = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId)
              return childNode?.type === 'ElCol'
            }).length
            editorStore.updateNode(rowNode.id, {
              props: {
                ...(latestRow?.props || rowNode.props || {}),
                columns: colCount,
              },
            })
            const inserted = insertNodeByResolvedType(resolvedType, colNode.id)
            if (!inserted) {
              notifyInsertFailure()
            }
          } else {
            notifyInsertFailure()
          }
          endDrag()
          return
        }
      }
      if (
        layoutResolved &&
        resolvedType !== 'ElLayoutRow' &&
        layoutResolved.nearEdge &&
        layoutInsertSnapshot?.layoutId === layoutResolved.layoutNode.id
      ) {
        targetNode = layoutResolved.layoutNode
        targetElement = layoutResolved.layoutElement || targetElement
      }
      if (layoutResolvedByPoint && resolvedType !== 'ElLayoutRow' && !layoutInsertSnapshot) {
        targetNode = layoutResolvedByPoint.layoutNode
        targetElement = layoutResolvedByPoint.layoutElement || targetElement
      }
      const plainRegionContentTarget = resolvePlainRegionContentTarget(
        targetNode,
        targetElement,
        resolvedType,
        doc.value,
      )
      if (plainRegionContentTarget) {
        targetNode = plainRegionContentTarget.node
        targetElement = plainRegionContentTarget.element || targetElement
      }
      const allowRowAutoInsert = targetNode?.type === 'ElLayoutRow' && resolvedType !== 'ElCol'
      const allowLayoutAutoInsert =
        targetNode?.type === 'ElLayout' && resolvedType !== 'ElLayoutRow'
      const allowPlainRegionContentDrop = Boolean(
        targetNode && isRegionType(targetNode.type) && isPlainRegionContentType(resolvedType),
      )
      if (
        !allowRowAutoInsert &&
        !allowLayoutAutoInsert &&
        !allowPlainRegionContentDrop &&
        !canAcceptChild(targetNode, resolvedType)
      ) {
        return
      }
      let insertIndex = (targetNode?.children || []).length
      if (
        rowInsertSnapshot &&
        targetNode?.type === 'ElLayoutRow' &&
        rowInsertSnapshot.rowId === targetNode.id
      ) {
        insertIndex = rowInsertSnapshot.index
      }
      if (
        layoutInsertSnapshot &&
        targetNode?.type === 'ElLayout' &&
        layoutInsertSnapshot.layoutId === targetNode.id
      ) {
        insertIndex = layoutInsertSnapshot.index
      }
      if (
        layoutResolvedByPoint &&
        targetNode?.type === 'ElLayout' &&
        layoutResolvedByPoint.layoutNode.id === targetNode.id &&
        layoutInsertSnapshot?.layoutId !== targetNode.id
      ) {
        insertIndex = layoutResolvedByPoint.index
      }

      const skipFlexInsertForLayout =
        targetNode?.type === 'ElLayout' && (layoutInsertSnapshot || layoutResolvedByPoint)
      let scopedDropKey: string | null = null
      if (targetNode?.type === 'Tabs' || targetNode?.type === 'Collapse') {
        const scopedMeta = resolveScopedSlotMeta(targetNode, event)
        scopedDropKey = scopedMeta?.key || null
        if (scopedMeta?.hostElement) {
          targetElement = scopedMeta.hostElement
          if (scopedMeta.childIds.length > 0) {
            const scopedInsert = dragDropManager.calculateFlexInsertPosition(
              scopedMeta.hostElement,
              event,
              'column',
            )
            const scopedIndex =
              typeof scopedInsert.index === 'number'
                ? scopedInsert.index
                : scopedMeta.childIds.length
            insertIndex = mapScopedInsertIndex(targetNode, scopedMeta.childIds, scopedIndex)
          } else {
            insertIndex = (targetNode.children || []).length
          }
        }
      }
      if (targetNode?.type && isFlexContainer(targetNode.type) && !skipFlexInsertForLayout) {
        const outerEl = targetElement
        const contentEl = outerEl?.querySelector?.('[data-node-id]')?.parentElement || outerEl
        if (!contentEl) {
          endDrag()
          return
        }
        const direction = resolveFlexDirection(targetNode.type, contentEl)
        const insertInfo = dragDropManager.calculateFlexInsertPosition(contentEl, event, direction)
        insertIndex = insertInfo.index
      }
      let dropPosition = null
      if (targetNode?.type === 'FreeContainer' && targetElement) {
        const targetHost = targetElement instanceof HTMLElement ? targetElement : null
        if (!targetHost) {
          endDrag()
          return
        }
        const zoomValue = Number(canvasZoom?.value) || 1
        const rawPosition = eventToCanvasPosition(event, targetHost, zoomValue)
        const defaultSize = getDefaultSize(resolvedType) ?? {
          width: 120,
          height: 40,
        }
        dropPosition = clampPositionInContainer(rawPosition, targetHost, defaultSize, zoomValue)
      } else {
        const zoomValue = Number(canvasZoom?.value) || 1
        dropPosition = resolvePlainRegionDropPosition(
          event,
          targetNode,
          targetElement,
          resolvedType,
          zoomValue,
        )
      }

      // ElLayout 鍐呮嫋鍏ョ粍浠讹細鑷姩鏂板涓€琛屽苟灏嗙粍浠舵斁鍏ヨ琛岀殑鍒?
      if (allowLayoutAutoInsert) {
        const rowNode = insertNodeWithoutSelection('ElLayoutRow', targetNode.id, insertIndex)
        if (rowNode) {
          const latestLayout = doc.value?.getNode?.(targetNode.id)
          const rowCount = (latestLayout?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElLayoutRow'
          }).length
          const nextRows = Math.max(1, rowCount)
          if ((latestLayout?.props?.rows || 0) !== nextRows) {
            editorStore.updateNode(targetNode.id, {
              props: {
                ...(latestLayout?.props || targetNode.props || {}),
                rows: nextRows,
              },
            })
          }
          editorStore.updateNode(rowNode.id, {
            props: { ...(rowNode.props || {}), columns: 1 },
          })
          const latestRow = doc.value?.getNode?.(rowNode.id)
          const colIds = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          })
          let colId = colIds[0]
          if (!colId) {
            const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, 0)
            if (!colNode) {
              notifyInsertFailure()
              endDrag()
              return
            }
            colId = colNode.id
          }
          const inserted = insertNodeByResolvedType(resolvedType, colId)
          if (!inserted) {
            notifyInsertFailure()
          }
        } else {
          notifyInsertFailure()
        }
        endDrag()
        return
      }

      // ElLayoutRow 鍐呮嫋鍏ョ粍浠讹細鑷姩鏂板涓€鍒楀苟灏嗙粍浠舵斁鍏ヨ鍒?
      if (allowRowAutoInsert) {
        const colNode = insertNodeWithoutSelection('ElCol', targetNode.id, insertIndex)
        if (colNode) {
          const latestRow = doc.value?.getNode?.(targetNode.id)
          const colCount = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          }).length
          const nextColumns = Math.max(1, colCount)
          editorStore.updateNode(targetNode.id, {
            props: {
              ...(latestRow?.props || targetNode.props || {}),
              columns: nextColumns,
            },
          })
          const inserted = insertNodeByResolvedType(resolvedType, colNode.id)
          if (!inserted) {
            notifyInsertFailure()
          }
        } else {
          notifyInsertFailure()
        }
        endDrag()
        return
      }

      // 鎻掑叆鏂拌妭鐐?
      const inserted = insertNodeByResolvedType(resolvedType, targetNode?.id, insertIndex, {
        dropPosition: dropPosition || undefined,
      })
      if (!inserted) {
        notifyInsertFailure()
      }
      applyScopedKeyForInsertedNode(inserted, targetNode, scopedDropKey)
      endDrag()
    } catch {
      let type = assetComponentType || fallbackType
      if (!type && payload) {
        try {
          const parsed = JSON.parse(payload) as { type?: string }
          type = parsed?.type || payload
        } catch {
          type = payload
        }
      }
      if (!type) return
      if (!node.value) return
      if (!layoutInsertSnapshot) {
        const resolvedLayoutByPoint = resolveLayoutInsertFromPoint()
        if (resolvedLayoutByPoint && type !== 'ElLayoutRow') {
          const inserted = insertIntoLayoutByRowAndApplyAsset(
            resolvedLayoutByPoint.layoutNode,
            resolvedLayoutByPoint.index,
            type,
          )
          if (!inserted) {
            notifyInsertFailure()
          }
          endDrag()
          return
        }
      }
      if (
        layoutInsertSnapshot?.layoutId &&
        type !== 'ElLayoutRow' &&
        doc.value?.getNode?.(layoutInsertSnapshot.layoutId)?.type === 'ElLayout'
      ) {
        const layoutNode = doc.value.getNode(layoutInsertSnapshot.layoutId)
        const inserted = insertIntoLayoutByRowAndApplyAsset(
          layoutNode,
          layoutInsertSnapshot.index,
          type,
        )
        if (!inserted) notifyInsertFailure()
        endDrag()
        return
      }
      const resolvedTarget = resolveElContainerTarget()
      let targetNode = resolvedTarget?.node || node.value
      let targetElement =
        resolvedTarget?.element ||
        (event.currentTarget instanceof Element ? event.currentTarget : null)
      if (!isDroppableContainer(targetNode)) {
        const resolvedContainer = resolveDropContainer(event, type, doc.value, preferOuterDropByAlt)
        if (resolvedContainer) {
          targetNode = resolvedContainer.node
          targetElement = resolvedContainer.element || targetElement
        } else {
          return
        }
      }
      const rowResolved = resolveRowInsertTarget()
      const layoutResolved = resolveLayoutInsertTarget()
      const layoutResolvedByPoint = resolveLayoutInsertFromPoint()
      if (
        rowResolved &&
        type !== 'ElCol' &&
        rowResolved.nearEdge &&
        rowInsertSnapshot?.rowId === rowResolved.rowNode.id
      ) {
        targetNode = rowResolved.rowNode
        targetElement = rowResolved.rowElement || targetElement
      }
      if (
        targetNode?.type === 'ElCol' &&
        type !== 'ElCol' &&
        doc.value?.getParent?.(targetNode.id)?.type === 'ElLayoutRow' &&
        rowInsertSnapshot?.rowId === doc.value.getParent(targetNode.id)?.id
      ) {
        const parentNode = doc.value.getParent(targetNode.id)
        if (parentNode) {
          targetNode = parentNode
          const rowSelector = `[data-node-id="${parentNode.id}"]`
          const currentTarget = event.currentTarget instanceof Element ? event.currentTarget : null
          targetElement =
            document.querySelector(rowSelector) ||
            currentTarget?.closest?.(rowSelector) ||
            targetElement
        }
      }
      if (
        targetNode?.type === 'ElCol' &&
        type !== 'ElCol' &&
        doc.value?.getParent?.(targetNode.id)?.type === 'ElLayoutRow'
      ) {
        const rowNode = doc.value.getParent(targetNode.id)
        const layoutNode = rowNode ? doc.value.getParent?.(rowNode.id) : null
        if (
          layoutInsertSnapshot?.layoutId &&
          layoutNode?.type === 'ElLayout' &&
          layoutInsertSnapshot.layoutId === layoutNode.id &&
          type !== 'ElLayoutRow'
        ) {
          const rowNodeInserted = insertNodeWithoutSelection(
            'ElLayoutRow',
            layoutNode.id,
            layoutInsertSnapshot.index,
          )
          if (rowNodeInserted) {
            const latestLayout = doc.value?.getNode?.(layoutNode.id)
            const rowCount = (latestLayout?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId)
              return childNode?.type === 'ElLayoutRow'
            }).length
            editorStore.updateNode(layoutNode.id, {
              props: {
                ...(latestLayout?.props || layoutNode.props || {}),
                rows: Math.max(1, rowCount),
              },
            })
            editorStore.updateNode(rowNodeInserted.id, {
              props: { ...(rowNodeInserted.props || {}), columns: 1 },
            })
            const latestRow = doc.value?.getNode?.(rowNodeInserted.id)
            const colIds = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId)
              return childNode?.type === 'ElCol'
            })
            let colId = colIds[0]
            if (!colId) {
              const colNode = insertNodeWithoutSelection('ElCol', rowNodeInserted.id, 0)
              if (!colNode) {
                notifyInsertFailure()
                endDrag()
                return
              }
              colId = colNode.id
            }
            const inserted = insertNodeByResolvedType(type, colId)
            if (!inserted) {
              notifyInsertFailure()
            }
          } else {
            notifyInsertFailure()
          }
          endDrag()
          return
        }
        const colHasChild = (targetNode.children || []).length > 0
        if (colHasChild) {
          const rowResolved = resolveRowInsertTarget()
          if (!rowNode) {
            notifyInsertFailure()
            endDrag()
            return
          }
          const colIds = (rowNode?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          })
          const currentIndex = Math.max(0, colIds.indexOf(targetNode.id))
          let insertIndex = currentIndex + 1
          if (rowInsertSnapshot?.rowId === rowNode.id) {
            insertIndex = rowInsertSnapshot.index
          } else if (Number.isInteger(rowResolved?.index)) {
            insertIndex = rowResolved?.index ?? insertIndex
          }
          const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, insertIndex)
          if (colNode) {
            const latestRow = doc.value?.getNode?.(rowNode.id)
            const colCount = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId)
              return childNode?.type === 'ElCol'
            }).length
            editorStore.updateNode(rowNode.id, {
              props: {
                ...(latestRow?.props || rowNode.props || {}),
                columns: colCount,
              },
            })
            const inserted = insertNodeByResolvedType(type, colNode.id)
            if (!inserted) {
              notifyInsertFailure()
            }
          } else {
            notifyInsertFailure()
          }
          endDrag()
          return
        }
      }
      if (
        layoutResolved &&
        type !== 'ElLayoutRow' &&
        layoutResolved.nearEdge &&
        layoutInsertSnapshot?.layoutId === layoutResolved.layoutNode.id
      ) {
        targetNode = layoutResolved.layoutNode
        targetElement = layoutResolved.layoutElement || targetElement
      }
      if (layoutResolvedByPoint && type !== 'ElLayoutRow' && !layoutInsertSnapshot) {
        targetNode = layoutResolvedByPoint.layoutNode
        targetElement = layoutResolvedByPoint.layoutElement || targetElement
      }
      const plainRegionContentTarget = resolvePlainRegionContentTarget(
        targetNode,
        targetElement,
        type,
        doc.value,
      )
      if (plainRegionContentTarget) {
        targetNode = plainRegionContentTarget.node
        targetElement = plainRegionContentTarget.element || targetElement
      }
      const allowRowAutoInsert = targetNode?.type === 'ElLayoutRow' && type !== 'ElCol'
      const allowLayoutAutoInsert = targetNode?.type === 'ElLayout' && type !== 'ElLayoutRow'
      const allowPlainRegionContentDrop = Boolean(
        targetNode && isRegionType(targetNode.type) && isPlainRegionContentType(type),
      )
      if (
        !allowRowAutoInsert &&
        !allowLayoutAutoInsert &&
        !allowPlainRegionContentDrop &&
        !canAcceptChild(targetNode, type)
      ) {
        return
      }

      // 璁＄畻鎻掑叆浣嶇疆
      let insertIndex = (targetNode?.children || []).length
      if (
        rowInsertSnapshot &&
        targetNode?.type === 'ElLayoutRow' &&
        rowInsertSnapshot.rowId === targetNode.id
      ) {
        insertIndex = rowInsertSnapshot.index
      }
      if (
        layoutInsertSnapshot &&
        targetNode?.type === 'ElLayout' &&
        layoutInsertSnapshot.layoutId === targetNode.id
      ) {
        insertIndex = layoutInsertSnapshot.index
      }
      if (
        layoutResolvedByPoint &&
        targetNode?.type === 'ElLayout' &&
        layoutResolvedByPoint.layoutNode.id === targetNode.id &&
        layoutInsertSnapshot?.layoutId !== targetNode.id
      ) {
        insertIndex = layoutResolvedByPoint.index
      }

      const skipFlexInsertForLayout =
        targetNode?.type === 'ElLayout' && (layoutInsertSnapshot || layoutResolvedByPoint)
      let scopedDropKey: string | null = null
      if (targetNode?.type === 'Tabs' || targetNode?.type === 'Collapse') {
        const scopedMeta = resolveScopedSlotMeta(targetNode, event)
        scopedDropKey = scopedMeta?.key || null
        if (scopedMeta?.hostElement) {
          targetElement = scopedMeta.hostElement
          if (scopedMeta.childIds.length > 0) {
            const scopedInsert = dragDropManager.calculateFlexInsertPosition(
              scopedMeta.hostElement,
              event,
              'column',
            )
            const scopedIndex =
              typeof scopedInsert.index === 'number'
                ? scopedInsert.index
                : scopedMeta.childIds.length
            insertIndex = mapScopedInsertIndex(targetNode, scopedMeta.childIds, scopedIndex)
          } else {
            insertIndex = (targetNode.children || []).length
          }
        }
      }
      if (targetNode?.type && isFlexContainer(targetNode.type) && !skipFlexInsertForLayout) {
        const outerEl = targetElement
        const contentEl = outerEl?.querySelector?.('[data-node-id]')?.parentElement || outerEl
        if (!contentEl) {
          endDrag()
          return
        }
        const direction = resolveFlexDirection(targetNode.type, contentEl)
        const insertInfo = dragDropManager.calculateFlexInsertPosition(contentEl, event, direction)
        insertIndex = insertInfo.index
      }

      let dropPosition = null
      if (targetNode?.type === 'FreeContainer' && targetElement) {
        const targetHost = targetElement instanceof HTMLElement ? targetElement : null
        if (!targetHost) {
          endDrag()
          return
        }
        const zoomValue = Number(canvasZoom?.value) || 1
        const rawPosition = eventToCanvasPosition(event, targetHost, zoomValue)
        const defaultSize = getDefaultSize(type) ?? { width: 120, height: 40 }
        dropPosition = clampPositionInContainer(rawPosition, targetHost, defaultSize, zoomValue)
      } else {
        const zoomValue = Number(canvasZoom?.value) || 1
        dropPosition = resolvePlainRegionDropPosition(
          event,
          targetNode,
          targetElement,
          type,
          zoomValue,
        )
      }

      // ElLayout 鍐呮嫋鍏ョ粍浠讹細鑷姩鏂板涓€琛屽苟灏嗙粍浠舵斁鍏ヨ琛岀殑鍒?
      if (allowLayoutAutoInsert) {
        const rowNode = insertNodeWithoutSelection('ElLayoutRow', targetNode.id, insertIndex)
        if (rowNode) {
          const latestLayout = doc.value?.getNode?.(targetNode.id)
          const rowCount = (latestLayout?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElLayoutRow'
          }).length
          const nextRows = Math.max(1, rowCount)
          if ((latestLayout?.props?.rows || 0) !== nextRows) {
            editorStore.updateNode(targetNode.id, {
              props: {
                ...(latestLayout?.props || targetNode.props || {}),
                rows: nextRows,
              },
            })
          }
          editorStore.updateNode(rowNode.id, {
            props: { ...(rowNode.props || {}), columns: 1 },
          })
          const latestRow = doc.value?.getNode?.(rowNode.id)
          const colIds = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          })
          let colId = colIds[0]
          if (!colId) {
            const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, 0)
            if (!colNode) {
              notifyInsertFailure()
              endDrag()
              return
            }
            colId = colNode.id
          }
          const inserted = insertNodeByResolvedType(type, colId)
          if (!inserted) {
            notifyInsertFailure()
          }
        } else {
          notifyInsertFailure()
        }
        endDrag()
        return
      }

      // ElLayoutRow 鍐呮嫋鍏ョ粍浠讹細鑷姩鏂板涓€鍒楀苟灏嗙粍浠舵斁鍏ヨ鍒?
      if (allowRowAutoInsert) {
        const colNode = insertNodeWithoutSelection('ElCol', targetNode.id, insertIndex)
        if (colNode) {
          const latestRow = doc.value?.getNode?.(targetNode.id)
          const colCount = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          }).length
          const nextColumns = Math.max(1, colCount)
          editorStore.updateNode(targetNode.id, {
            props: {
              ...(latestRow?.props || targetNode.props || {}),
              columns: nextColumns,
            },
          })
          const inserted = insertNodeByResolvedType(type, colNode.id)
          if (!inserted) {
            notifyInsertFailure()
          }
        } else {
          notifyInsertFailure()
        }
        endDrag()
        return
      }

      // 鎻掑叆鏂拌妭鐐?
      const inserted = insertNodeByResolvedType(type, targetNode?.id, insertIndex, {
        dropPosition: dropPosition || undefined,
      })
      if (!inserted) {
        notifyInsertFailure()
      }
      applyScopedKeyForInsertedNode(inserted, targetNode, scopedDropKey)
      endDrag()
    }
  }

  return {
    handleDragOver,
    handleDrop,
    showInsertLine,
    insertLineStyle,
    genericInsertLineBox,
    rowInsertInfo,
    layoutInsertInfo,
    insertLineBox,
    isDragOver,
    suppressDropByAlt,
    canAcceptChild,
    isDroppableContainer,
    resolveDropContainer: (event: DragEvent, childType: string) =>
      resolveDropContainer(event, childType, doc.value),
  }
}

export default { useNodeDrop }
