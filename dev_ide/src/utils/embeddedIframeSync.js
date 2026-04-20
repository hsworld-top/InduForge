/**
 * 构造嵌入应用更新消息。
 * @param {string} type - 消息类型
 * @param {string} key - 消息字段名
 * @param {string} value - 消息值
 * @returns {object} 消息对象
 */
export const createEmbeddedUpdateMessage = (type, key, value) => ({
  type,
  [key]: value,
})

/**
 * 向嵌入 iframe 广播消息。
 * @param {Iterable<object>} iframes - iframe 列表
 * @param {object} message - 要广播的消息
 */
export const broadcastToEmbeddedIframes = (iframes, message) => {
  if (!iframes?.forEach) return

  iframes.forEach((iframe) => {
    iframe?.contentWindow?.postMessage?.(message, '*')
  })
}

/**
 * 判断 iframe 是否属于设计中心。
 * @param {object} iframe - iframe 节点
 * @returns {boolean} 是否为设计中心 iframe
 */
export const isDesignerEmbeddedIframe = (iframe) => {
  const src = iframe?.getAttribute?.('src') || iframe?.src || ''
  if (!src) return false

  try {
    const url = new URL(src, 'http://localhost/')
    return url.pathname === '/designer' || url.pathname.startsWith('/designer/')
  } catch (error) {
    return false
  }
}

/**
 * 仅向设计中心 iframe 广播消息。
 * @param {Iterable<object>} iframes - iframe 列表
 * @param {object} message - 要广播的消息
 */
export const broadcastToDesignerEmbeddedIframes = (iframes, message) => {
  if (!iframes?.forEach) return

  iframes.forEach((iframe) => {
    if (!isDesignerEmbeddedIframe(iframe)) return
    iframe?.contentWindow?.postMessage?.(message, '*')
  })
}

/**
 * 从文档中筛选嵌入 iframe。
 * @param {object} documentLike - 类文档对象
 * @returns {Array<object>} iframe 列表
 */
export const getEmbeddedIframes = (documentLike) => {
  if (!documentLike?.querySelectorAll) return []
  return documentLike.querySelectorAll('iframe.embedded-iframe') || []
}

/**
 * 同步设计中心语言到嵌入 iframe。
 * @param {object} documentLike - 类文档对象
 * @param {string} locale - 语言
 */
export const syncDesignerLocaleToEmbeddedIframes = (documentLike, locale) => {
  const iframes = getEmbeddedIframes(documentLike)
  broadcastToDesignerEmbeddedIframes(
    iframes,
    createEmbeddedUpdateMessage('LOCALE_UPDATE', 'locale', locale)
  )
}
