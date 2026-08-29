import type { KafkaTopicGroup, KafkaTopicGroupNode, KafkaTopicMapping } from './types'

export function buildKafkaTopicTree(
  groups: KafkaTopicGroup[],
  mappings: KafkaTopicMapping[],
): { groups: KafkaTopicGroupNode[]; rootMappings: KafkaTopicMapping[] } {
  const nodes = new Map<string, KafkaTopicGroupNode>()
  groups.forEach((group) => {
    nodes.set(String(group.id), { ...group, children: [], mappings: [] })
  })

  const roots: KafkaTopicGroupNode[] = []
  nodes.forEach((node) => {
    const parentId = node.parentId ? String(node.parentId) : ''
    const parent = parentId ? nodes.get(parentId) : null
    if (parent) {
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  })

  const rootMappings: KafkaTopicMapping[] = []
  mappings.forEach((mapping) => {
    const groupId = mapping.groupId ? String(mapping.groupId) : ''
    const group = groupId ? nodes.get(groupId) : null
    if (group) {
      group.mappings.push(mapping)
    } else {
      rootMappings.push(mapping)
    }
  })

  const sortMappings = (items: KafkaTopicMapping[]) =>
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )

  const sortNodes = (items: KafkaTopicGroupNode[]) => {
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )
    items.forEach((item) => {
      sortNodes(item.children)
      sortMappings(item.mappings)
    })
  }

  sortNodes(roots)
  sortMappings(rootMappings)
  return { groups: roots, rootMappings }
}

export function filterKafkaTopicTree(
  groups: KafkaTopicGroupNode[],
  rootMappings: KafkaTopicMapping[],
  keyword: string,
): { groups: KafkaTopicGroupNode[]; rootMappings: KafkaTopicMapping[] } {
  const normalized = keyword.trim().toLowerCase()
  if (!normalized) {
    return { groups, rootMappings }
  }

  const mappingMatches = (mapping: KafkaTopicMapping) => {
    const modeLabels = mapping.outputMode === 'raw_message'
      ? ['整包数据点', 'raw message data point']
      : ['字段数据点', 'field data points']
    return [mapping.name, mapping.topic, mapping.consumerGroup, ...modeLabels]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized))
  }

  const filterNode = (node: KafkaTopicGroupNode): KafkaTopicGroupNode | null => {
    const children = node.children
      .map(filterNode)
      .filter((child): child is KafkaTopicGroupNode => Boolean(child))
    const mappings = node.mappings.filter(mappingMatches)
    const selfMatches = node.name.toLowerCase().includes(normalized)
    if (selfMatches || children.length > 0 || mappings.length > 0) {
      return { ...node, children, mappings: selfMatches ? node.mappings : mappings }
    }
    return null
  }

  return {
    groups: groups.map(filterNode).filter((node): node is KafkaTopicGroupNode => Boolean(node)),
    rootMappings: rootMappings.filter(mappingMatches),
  }
}

export function flattenKafkaTopicGroups(
  groups: KafkaTopicGroup[],
  blockedIds: Set<string> = new Set(),
): Array<{ id: string; label: string }> {
  const nodes = new Map<string, KafkaTopicGroupNode>()
  groups.forEach((group) => {
    nodes.set(String(group.id), { ...group, children: [], mappings: [] })
  })

  const roots: KafkaTopicGroupNode[] = []
  nodes.forEach((node) => {
    const parentId = node.parentId ? String(node.parentId) : ''
    const parent = parentId ? nodes.get(parentId) : null
    if (parent) {
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  })

  const visit = (
    group: KafkaTopicGroupNode,
    depth: number,
  ): Array<{ id: string; label: string }> => {
    const id = String(group.id)
    const children = group.children.flatMap((child) => visit(child, depth + 1))
    if (blockedIds.has(id)) return children
    return [{ id, label: `${'　'.repeat(depth)}${group.name}` }, ...children]
  }

  return roots.flatMap((group) => visit(group, 0))
}

export function collectKafkaTopicGroupIds(group?: KafkaTopicGroupNode | null) {
  const result = new Set<string>()
  const visit = (node?: KafkaTopicGroupNode | null) => {
    if (!node) return
    result.add(String(node.id))
    node.children.forEach(visit)
  }
  visit(group)
  return result
}
