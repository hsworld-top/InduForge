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

  const mappingMatches = (mapping: KafkaTopicMapping) =>
    `${mapping.name} ${mapping.topic}`.toLowerCase().includes(normalized)

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
