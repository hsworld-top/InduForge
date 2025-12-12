/**
 * 连接状态管理 Composable
 * 处理连接状态的显示和更新
 */

import { computed } from 'vue'

/**
 * 获取状态显示信息
 * @param {string} status - 连接状态
 * @returns {Object} 状态显示信息
 */
export function getStatusInfo(status) {
  const statusMap = {
    connected: {
      label: '已连接',
      color: 'green',
      tagType: 'success',
      dotClass: 'bg-green-500'
    },
    disconnected: {
      label: '已断开',
      color: 'gray',
      tagType: 'info',
      dotClass: 'bg-gray-400'
    },
    error: {
      label: '连接错误',
      color: 'red',
      tagType: 'danger',
      dotClass: 'bg-red-500'
    },
    unknown: {
      label: '未知状态',
      color: 'gray',
      tagType: 'info',
      dotClass: 'bg-gray-400'
    }
  }

  return statusMap[status] || statusMap.unknown
}

/**
 * 连接状态管理
 */
export function useConnectionStatus(connection) {
  const statusInfo = computed(() => {
    return getStatusInfo(connection.value?.status)
  })

  const statusLabel = computed(() => statusInfo.value.label)
  const statusColor = computed(() => statusInfo.value.color)
  const statusTagType = computed(() => statusInfo.value.tagType)
  const statusDotClass = computed(() => statusInfo.value.dotClass)

  return {
    statusInfo,
    statusLabel,
    statusColor,
    statusTagType,
    statusDotClass,
  }
}
