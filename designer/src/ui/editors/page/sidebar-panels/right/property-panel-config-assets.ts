/**
 * 属性面板「资源选择 / 配置」树构建等纯逻辑（从 PropertyPanel 抽出）
 */

export function unwrapApiData(response: unknown): unknown {
  const r = response as { data?: { data?: unknown; folders?: unknown; assets?: unknown } }
  return r?.data?.data ?? r?.data ?? response
}

export function decodeAssetName(value: string | undefined | null): string {
  if (!value) return ''
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

export function resolveConfigAssetUrl(
  asset: {
    url?: string
    src?: string
    path?: string
  } | null,
): string {
  if (!asset) return ''
  return asset.url || asset.src || asset.path || ''
}

export interface ConfigAssetTreeNode {
  id: string
  label: string
  type: 'folder' | 'asset'
  raw: unknown
  children?: ConfigAssetTreeNode[]
}

export function buildConfigAssetTree(
  folders: Array<{ id: string; name?: string; parentId?: string | null }> | null | undefined,
  assets:
    | Array<{
        id: string
        name?: string
        originalName?: string
        folderId?: string | null
      }>
    | null
    | undefined,
): ConfigAssetTreeNode[] {
  const folderNodes = (folders || []).map((folder) => ({
    id: folder.id,
    label: decodeAssetName(folder.name || '未命名文件夹'),
    type: 'folder' as const,
    raw: folder,
    children: [] as ConfigAssetTreeNode[],
  }))
  const folderMap = new Map(folderNodes.map((node) => [node.id, node]))
  const root: ConfigAssetTreeNode = {
    id: 'all',
    label: '全部资源',
    type: 'folder',
    raw: null,
    children: [],
  }
  folderNodes.forEach((node) => {
    const parentId = (node.raw as { parentId?: string | null }).parentId
    if (parentId && folderMap.has(parentId)) {
      folderMap.get(parentId)!.children!.push(node)
    } else {
      root.children!.push(node)
    }
  })
  ;(assets || []).forEach((asset) => {
    const node: ConfigAssetTreeNode = {
      id: asset.id,
      label: decodeAssetName(asset.name || asset.originalName || '未命名资源'),
      type: 'asset',
      raw: asset,
    }
    const folderId = asset.folderId
    if (folderId && folderMap.has(folderId)) {
      folderMap.get(folderId)!.children!.push(node)
    } else {
      root.children!.push(node)
    }
  })

  return [root]
}

export function filterConfigAssetNode(value: string, data: { label?: string }): boolean {
  if (!value) return true
  return String(data?.label || '')
    .toLowerCase()
    .includes(String(value).toLowerCase())
}
