/**
 * 框选点击守卫：用于吞掉框选结束后浏览器补发的一次 click，
 * 避免多选结果被节点 click 单选覆盖。
 */
export interface MarqueeClickGuard {
  /**
   * 标记下一次 click 需要被吞掉。
   */
  markShouldSuppressNextClick: () => void
  /**
   * 重置守卫状态。
   */
  reset: () => void
  /**
   * 消费一次 click 抑制状态。
   * @returns {boolean} true 表示本次 click 需要被阻止
   */
  consumeShouldSuppressNextClick: () => boolean
}

/**
 * 创建框选 click 守卫实例。
 * @returns {MarqueeClickGuard}
 */
export function createMarqueeClickGuard(): MarqueeClickGuard {
  let shouldSuppressNextClick = false

  return {
    markShouldSuppressNextClick() {
      shouldSuppressNextClick = true
    },
    reset() {
      shouldSuppressNextClick = false
    },
    consumeShouldSuppressNextClick() {
      if (!shouldSuppressNextClick) return false
      shouldSuppressNextClick = false
      return true
    },
  }
}
