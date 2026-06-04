export type MqttSubscription = {
  id: string
  name?: string
  topic?: string
  qos?: number
  usageMode?: 'raw_datapoint' | 'single_variable' | 'batch_variable'
  groupId?: string | null
  description?: string | null
  messageRetention?: number
  defaultBatchParseRule?: Record<string, unknown> | null
  order?: number
}

export type MqttSubscriptionGroup = {
  id: string
  name: string
  parentId?: string | null
  children?: MqttSubscriptionGroup[]
}

export type MqttSubscriptionGroupNode = {
  id: string
  name: string
  parentId: string | null
  children: MqttSubscriptionGroupNode[]
  subscriptions: MqttSubscription[]
}

const toId = (value: unknown) =>
  value === null || value === undefined || value === '' ? null : String(value)

const subscriptionLabel = (subscription: MqttSubscription) =>
  subscription.name || subscription.topic || ''

const sortByLabel = <T>(items: T[], getter: (item: T) => string) =>
  [...items].sort((a, b) => getter(a).localeCompare(getter(b)))

export function buildMqttSubscriptionTree(
  groups: MqttSubscriptionGroup[],
  subscriptions: MqttSubscription[],
) {
  const groupMap = new Map<string, MqttSubscriptionGroupNode>()

  const visitGroup = (group: MqttSubscriptionGroup, parentId?: string | null) => {
    const id = String(group.id)
    const node = groupMap.get(id) ?? {
      id,
      name: group.name,
      parentId: toId(group.parentId) ?? parentId ?? null,
      children: [],
      subscriptions: [],
    }

    node.name = group.name
    node.parentId = toId(group.parentId) ?? parentId ?? null
    groupMap.set(id, node)

    group.children?.forEach((child) => visitGroup(child, id))
  }

  groups.forEach((group) => visitGroup(group))

  const rootSubscriptions: MqttSubscription[] = []
  const childIdSet = new Set<string>()
  groupMap.forEach((node) => {
    node.children = []
    node.subscriptions = []
  })

  groupMap.forEach((node) => {
    if (!node.parentId || !groupMap.has(node.parentId)) return
    groupMap.get(node.parentId)?.children.push(node)
    childIdSet.add(node.id)
  })

  subscriptions.forEach((subscription) => {
    const groupId = toId(subscription.groupId)
    if (groupId && groupMap.has(groupId)) {
      groupMap.get(groupId)?.subscriptions.push(subscription)
      return
    }
    rootSubscriptions.push(subscription)
  })

  const sortNode = (node: MqttSubscriptionGroupNode): MqttSubscriptionGroupNode => ({
    ...node,
    children: sortByLabel(node.children, (item) => item.name).map(sortNode),
    subscriptions: sortByLabel(node.subscriptions, subscriptionLabel),
  })

  return {
    rootGroups: sortByLabel(
      [...groupMap.values()].filter((node) => !childIdSet.has(node.id)),
      (item) => item.name,
    ).map(sortNode),
    rootSubscriptions: sortByLabel(rootSubscriptions, subscriptionLabel),
  }
}

export function filterMqttSubscriptionTree(
  groups: MqttSubscriptionGroupNode[],
  subscriptions: MqttSubscription[],
  keyword: string,
) {
  const lower = keyword.trim().toLowerCase()
  if (!lower) return { groups, subscriptions }

  const matchesSubscription = (subscription: MqttSubscription) =>
    [subscription.name, subscription.topic, subscription.description]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(lower))

  const matchesGroup = (group: MqttSubscriptionGroupNode) =>
    group.name.toLowerCase().includes(lower)

  const filterGroup = (group: MqttSubscriptionGroupNode): MqttSubscriptionGroupNode | null => {
    const groupMatched = matchesGroup(group)
    const children = group.children
      .map(filterGroup)
      .filter((child): child is MqttSubscriptionGroupNode => Boolean(child))
    const matchedSubscriptions = group.subscriptions.filter(matchesSubscription)

    if (groupMatched || children.length > 0 || matchedSubscriptions.length > 0) {
      return {
        ...group,
        children: groupMatched ? group.children : children,
        subscriptions: groupMatched ? group.subscriptions : matchedSubscriptions,
      }
    }
    return null
  }

  return {
    groups: groups
      .map(filterGroup)
      .filter((group): group is MqttSubscriptionGroupNode => Boolean(group)),
    subscriptions: subscriptions.filter(matchesSubscription),
  }
}

export function flattenMqttSubscriptionGroups(
  groups: MqttSubscriptionGroup[],
  blockedIds: Set<string> = new Set(),
  depth = 0,
): Array<{ id: string; label: string }> {
  return groups.flatMap((group) => {
    const id = String(group.id)
    const children = flattenMqttSubscriptionGroups(group.children || [], blockedIds, depth + 1)
    if (blockedIds.has(id)) return children
    return [{ id, label: `${'　'.repeat(depth)}${group.name}` }, ...children]
  })
}

export function collectMqttSubscriptionGroupIds(group?: MqttSubscriptionGroupNode | null) {
  const ids = new Set<string>()
  const visit = (node?: MqttSubscriptionGroupNode | null) => {
    if (!node) return
    ids.add(node.id)
    node.children.forEach(visit)
  }
  visit(group)
  return ids
}
