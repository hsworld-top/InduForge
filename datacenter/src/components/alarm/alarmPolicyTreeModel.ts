import type { AlarmPolicy, AlarmPolicyGroup } from '@/api/schemas/alarm.schema'

export type AlarmPolicyGroupNode = AlarmPolicyGroup & {
  parentId: string | null
  children: AlarmPolicyGroupNode[]
  policies: AlarmPolicy[]
  policyIds: string[]
}

const toId = (value: unknown) =>
  value === null || value === undefined || value === '' ? null : String(value)

export function buildAlarmPolicyGroupTree(groups: AlarmPolicyGroup[], policies: AlarmPolicy[]) {
  const groupMap = new Map<string, AlarmPolicyGroupNode>()

  for (const group of groups) {
    const id = String(group.id)
    groupMap.set(id, {
      ...group,
      id,
      parentId: toId(group.parentId),
      children: [],
      policies: [],
      policyIds: [],
    })
  }

  const childIdSet = new Set<string>()
  groupMap.forEach((node) => {
    const parentId = toId(node.parentId)
    if (!parentId || !groupMap.has(parentId)) return
    groupMap.get(parentId)?.children.push(node)
    childIdSet.add(node.id)
  })

  const rootPolicies: AlarmPolicy[] = []
  for (const policy of policies) {
    const groupId = toId(policy.groupId)
    if (groupId && groupMap.has(groupId)) {
      groupMap.get(groupId)?.policies.push(policy)
      continue
    }
    rootPolicies.push(policy)
  }

  const sortByName = <T extends { name?: string }>(items: T[]) =>
    [...items].sort((a, b) => (a.name || '').localeCompare(b.name || ''))

  const sortNode = (node: AlarmPolicyGroupNode): AlarmPolicyGroupNode => {
    const children = sortByName(node.children).map(sortNode)
    const childPolicyIds = children.flatMap((child) => child.policyIds)
    const ownPolicies = sortByName(node.policies)
    return {
      ...node,
      children,
      policies: ownPolicies,
      policyIds: [...ownPolicies.map((policy) => policy.id), ...childPolicyIds],
    }
  }

  const rootGroups = [...groupMap.values()].filter((node) => !childIdSet.has(node.id))

  return {
    groups: sortByName(rootGroups).map(sortNode),
    rootPolicies: sortByName(rootPolicies),
  }
}

export function flattenAlarmPolicyGroups(
  groups: AlarmPolicyGroupNode[],
  depth = 0,
): Array<{ id: string; label: string }> {
  return groups.flatMap((group) => [
    { id: group.id, label: `${'　'.repeat(depth)}${group.name}` },
    ...flattenAlarmPolicyGroups(group.children, depth + 1),
  ])
}
