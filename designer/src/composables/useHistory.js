/**
 * useHistory - 历史记录管理（撤销/重做）
 * 
 * 使用快照模式记录状态变化，支持撤销和重做操作
 */
import { ref, computed } from 'vue'

export function useHistory(maxSize = 50) {
  const history = ref([])
  const currentIndex = ref(-1)
  
  const canUndo = computed(() => currentIndex.value > 0)
  const canRedo = computed(() => currentIndex.value < history.value.length - 1)
  
  /**
   * 添加新的历史记录
   * @param {Object} state - 要保存的状态快照
   * @param {string} description - 操作描述
   */
  function push(state, description = '') {
    // 删除当前索引之后的所有历史记录
    if (currentIndex.value < history.value.length - 1) {
      history.value = history.value.slice(0, currentIndex.value + 1)
    }
    
    // 添加新状态（深拷贝）
    history.value.push({
      state: JSON.parse(JSON.stringify(state)),
      description,
      timestamp: Date.now()
    })
    
    // 限制历史记录大小
    if (history.value.length > maxSize) {
      history.value.shift()
    } else {
      currentIndex.value++
    }
  }
  
  /**
   * 撤销操作
   * @returns {Object|null} 上一个状态或 null
   */
  function undo() {
    if (canUndo.value) {
      currentIndex.value--
      return history.value[currentIndex.value].state
    }
    return null
  }
  
  /**
   * 重做操作
   * @returns {Object|null} 下一个状态或 null
   */
  function redo() {
    if (canRedo.value) {
      currentIndex.value++
      return history.value[currentIndex.value].state
    }
    return null
  }
  
  /**
   * 清空历史记录
   */
  function clear() {
    history.value = []
    currentIndex.value = -1
  }
  
  /**
   * 获取当前状态
   */
  function getCurrentState() {
    if (currentIndex.value >= 0 && currentIndex.value < history.value.length) {
      return history.value[currentIndex.value].state
    }
    return null
  }
  
  return {
    canUndo,
    canRedo,
    push,
    undo,
    redo,
    clear,
    getCurrentState,
    history: computed(() => history.value),
    currentIndex: computed(() => currentIndex.value)
  }
}
