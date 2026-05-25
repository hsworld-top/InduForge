export interface SelectionBreadcrumbItem {
  id: string
  label: string
  type: string
  current: boolean
}

interface SelectionBreadcrumbNodeLike {
  id: string
  label?: string
  type?: string
}

interface SelectionBreadcrumbDocLike {
  getNode?: (id: string) => SelectionBreadcrumbNodeLike | null | undefined
  getParent?: (id: string) => SelectionBreadcrumbNodeLike | null | undefined
}

interface BuildSelectionBreadcrumbOptions {
  doc: SelectionBreadcrumbDocLike | null | undefined
  selectedNodeId: string | null | undefined
  rootNodeId?: string | null | undefined
}

/**
 * 根据当前选中节点向上收集可点击层级路径。
 *
 * 输入是文档模型、当前节点 ID 和页面根节点 ID；输出从外层布局到当前节点的路径。
 * 页面根节点会被过滤，避免用户在选择父布局时看到没有操作意义的根节点。
 * 如果文档缺失、节点缺失或父链异常，会返回已确认安全的部分路径，避免 UI 报错。
 */
export function buildSelectionBreadcrumbItems({
  doc,
  selectedNodeId,
  rootNodeId,
}: BuildSelectionBreadcrumbOptions): SelectionBreadcrumbItem[] {
  if (!doc || !selectedNodeId) return []

  const selectedNode = doc.getNode?.(selectedNodeId)
  if (!selectedNode) return []

  const path: SelectionBreadcrumbNodeLike[] = []
  const visited = new Set<string>()
  let current: SelectionBreadcrumbNodeLike | null | undefined = selectedNode

  while (current && !visited.has(current.id)) {
    visited.add(current.id)
    if (current.id !== rootNodeId) {
      path.unshift(current)
    }
    current = doc.getParent?.(current.id)
  }

  return path.map((node) => ({
    id: node.id,
    label: String(node.label || node.type || node.id),
    type: String(node.type || ''),
    current: node.id === selectedNode.id,
  }))
}
