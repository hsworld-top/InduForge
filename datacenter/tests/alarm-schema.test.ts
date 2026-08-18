import { describe, expect, test } from 'vitest'
import { reactive } from 'vue'
import { AlarmPolicySaveSchema, AlarmPolicySchema } from '../src/api/schemas/alarm.schema'

const policy = {
  id: 'policy-1',
  projectId: 'project-1',
  groupId: null,
  groupName: null,
  name: '温度报警',
  description: null,
  mode: 'per_target',
  derivedExpression: '',
  bindings: [
    {
      datapointId: 'dp-1',
      path: 'temperature',
      name: '温度',
      dataType: 'number',
      role: 'target',
      inputKey: null,
    },
  ],
  conditions: [
    {
      id: 'condition-1',
      kind: 'threshold',
      operator: 'gt',
      label: '高于',
      severity: 'warning',
      params: { threshold: 80 },
      triggerDelayMs: 0,
      clearDelayMs: 0,
      deadband: 0,
    },
  ],
  notification: { mode: 'inherit', channelIds: [], messageTemplate: '' },
  isEnabled: true,
  revision: 1,
  contract: { schemaVersion: 'alarm.policy.v1' },
  createdAt: null,
  updatedAt: null,
}

describe('alarm policy schema', () => {
  test('解析规范化绑定、条件和通知', () => {
    const parsed = AlarmPolicySchema.parse(policy)
    expect(parsed.bindings[0].role).toBe('target')
    expect(parsed.conditions[0].kind).toBe('threshold')
    expect(parsed.notification.mode).toBe('inherit')
  })

  test('保存 payload 不接受旧 targets/suppression 模型', () => {
    expect(() =>
      AlarmPolicySaveSchema.parse({
        ...policy,
        targets: policy.bindings,
        suppression: {},
        bindings: undefined,
      }),
    ).toThrow()
  })

  test('保存契约将响应式草稿转换为可提交的普通对象', () => {
    const draft = reactive({
      groupId: policy.groupId,
      name: policy.name,
      description: policy.description,
      mode: policy.mode,
      bindings: policy.bindings,
      derivedExpression: policy.derivedExpression,
      conditions: policy.conditions,
      notification: policy.notification,
      isEnabled: policy.isEnabled,
    })

    const payload = AlarmPolicySaveSchema.parse(draft)

    expect(() => structuredClone(payload)).not.toThrow()
  })
})
