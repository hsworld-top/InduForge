import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getAlarmLevelSettings } from '@/api/alarm.api'
import type { AlarmSeverityDefinition } from '@/api/schemas/alarm.schema'
import { defaultAlarmSeverityDefinitions } from '@/models/alarm-item'
import { getApiErrorMessage } from '@/utils/request'

export function useAlarmLevelDefinitions(projectId: () => string) {
  const definitions = ref<AlarmSeverityDefinition[]>(
    defaultAlarmSeverityDefinitions.map((item) => ({ ...item })),
  )
  let requestSequence = 0

  async function loadDefinitions() {
    const id = projectId()
    if (!id) return
    const sequence = ++requestSequence
    try {
      const result = await getAlarmLevelSettings(id)
      if (sequence === requestSequence)
        definitions.value = result.severityDefinitions.map((item) => ({ ...item }))
    } catch (error) {
      if (sequence === requestSequence)
        ElMessage.error(getApiErrorMessage(error, '加载报警级别失败'))
    }
  }

  return { definitions, loadDefinitions }
}
