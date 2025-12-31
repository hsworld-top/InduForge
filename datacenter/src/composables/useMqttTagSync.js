import { ref } from "vue";

/**
 * MQTT Tag 同步管理
 * 用于左右侧Tag列表和监控面板的数据同步
 */

// 全局状态：每个subscription的同步事件
const tagSyncEvents = ref(new Map());

/**
 * 获取或创建指定订阅的同步事件管理器
 * @param {string} subscriptionId - 订阅ID
 * @returns {object} 同步事件管理器
 */
export function useMqttTagSync(subscriptionId) {
  if (!tagSyncEvents.value.has(subscriptionId)) {
    tagSyncEvents.value.set(subscriptionId, {
      listeners: new Set(),
      lastUpdate: Date.now(),
    });
  }

  const syncManager = tagSyncEvents.value.get(subscriptionId);

  /**
   * 订阅Tag变化事件
   * @param {Function} callback - 回调函数
   */
  const subscribe = (callback) => {
    syncManager.listeners.add(callback);
    console.log(
      `[TagSync] Subscribed to ${subscriptionId}, total: ${syncManager.listeners.size}`,
    );
  };

  /**
   * 取消订阅Tag变化事件
   * @param {Function} callback - 回调函数
   */
  const unsubscribe = (callback) => {
    syncManager.listeners.delete(callback);
    console.log(
      `[TagSync] Unsubscribed from ${subscriptionId}, remaining: ${syncManager.listeners.size}`,
    );
  };

  /**
   * 通知Tag发生变化
   * @param {string} type - 变化类型：'created' | 'updated' | 'deleted' | 'toggled'
   * @param {object} data - 变化数据
   */
  const notify = (type, data) => {
    syncManager.lastUpdate = Date.now();
    console.log(
      `[TagSync] Notifying ${type} to ${syncManager.listeners.size} listeners`,
      data,
    );

    syncManager.listeners.forEach((callback) => {
      try {
        callback({ type, data, timestamp: syncManager.lastUpdate });
      } catch (error) {
        console.error("[TagSync] Error in listener callback:", error);
      }
    });
  };

  /**
   * 触发刷新事件（用于批量更新等场景）
   */
  const refresh = () => {
    notify("refresh", { subscriptionId });
  };

  return {
    subscribe,
    unsubscribe,
    notify,
    refresh,
  };
}
