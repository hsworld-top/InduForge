import { describe, expect, test } from 'vitest'
import { buildKafkaTopicTree, filterKafkaTopicTree } from '@/components/kafka/kafkaTopicTreeModel'
import type { KafkaTopicGroup, KafkaTopicMapping } from '@/components/kafka/types'

describe('kafkaTopicTreeModel', () => {
  test('按 parentId 组装 Topic 分组树并挂载映射', () => {
    const groups: KafkaTopicGroup[] = [
      { id: 'g1', connectionId: 'c1', parentId: null, name: '产线 A', sortOrder: 0 },
      { id: 'g2', connectionId: 'c1', parentId: 'g1', name: '炉区', sortOrder: 0 },
    ]
    const mappings: KafkaTopicMapping[] = [
      {
        id: 'm1',
        connectionId: 'c1',
        groupId: 'g2',
        name: '温度',
        topic: 'device.temperature',
        description: '',
        outputMode: 'field_mapping',
        rawOutputScope: 'value',
        partitionMode: 'all',
        startPosition: 'latest',
        decode: 'json',
        sampleLimit: 100,
        timeoutMs: 5000,
        sortOrder: 0,
      },
    ]

    const tree = buildKafkaTopicTree(groups, mappings)
    expect(tree.groups).toHaveLength(1)
    expect(tree.groups[0].children[0].mappings[0].topic).toBe('device.temperature')
  })

  test('按名称和 topic 过滤树', () => {
    const tree = buildKafkaTopicTree(
      [],
      [
        {
          id: 'm1',
          connectionId: 'c1',
          name: '报警事件',
          topic: 'alarm.events',
          description: '',
          outputMode: 'field_mapping',
          rawOutputScope: 'value',
          partitionMode: 'all',
          startPosition: 'latest',
          decode: 'json',
          sampleLimit: 100,
          timeoutMs: 5000,
          sortOrder: 0,
        },
      ],
    )
    const result = filterKafkaTopicTree(tree.groups, tree.rootMappings, 'alarm')
    expect(result.rootMappings).toHaveLength(1)
  })
})
