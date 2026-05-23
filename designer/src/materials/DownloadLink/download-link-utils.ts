export type DownloadLinkDisplayMode = 'link' | 'button'
export type DownloadLinkActionMode = 'download' | 'open' | 'none'
export type DownloadLinkTriggerMode = 'single' | 'double'

export interface DownloadLinkConfig {
  text: string
  href: string
  displayMode: DownloadLinkDisplayMode
  actionMode: DownloadLinkActionMode
  triggerMode: DownloadLinkTriggerMode
  target: '_blank' | '_self'
  downloadFileName: string
  disabled: boolean
  underline: boolean
  fontSize: number
  fontWeight: number
  textColor: string
  backgroundColor: string
  borderColor: string
  borderRadius: number
  paddingX: number
  paddingY: number
}

function toStringValue(value: unknown, fallback = ''): string {
  const next = String(value ?? '').trim()
  return next || fallback
}

function toBooleanValue(value: unknown, fallback = false): boolean {
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') {
    if (value === 'true') return true
    if (value === 'false') return false
  }
  return fallback
}

function toNumberValue(value: unknown, fallback: number): number {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return fallback
  return parsed
}

/**
 * 解析下载组件配置。
 * @param {Record<string, unknown> | null | undefined} resolvedProps - 解析后的属性
 * @returns {DownloadLinkConfig}
 */
export function resolveDownloadLinkConfig(
  resolvedProps: Record<string, unknown> | null | undefined,
): DownloadLinkConfig {
  const props = resolvedProps || {}
  const rawDisplayMode = toStringValue(props.displayMode, 'button')
  const displayMode: DownloadLinkDisplayMode = rawDisplayMode === 'link' ? 'link' : 'button'

  const rawActionMode = toStringValue(props.actionMode, 'download')
  const actionMode: DownloadLinkActionMode =
    rawActionMode === 'download' || rawActionMode === 'open' || rawActionMode === 'none'
      ? rawActionMode
      : 'download'

  const rawTriggerMode = toStringValue(props.triggerMode, 'double')
  const triggerMode: DownloadLinkTriggerMode = rawTriggerMode === 'single' ? 'single' : 'double'

  const rawTarget = toStringValue(props.target, '_blank')
  const target: '_blank' | '_self' = rawTarget === '_self' ? '_self' : '_blank'

  const text = toStringValue(props.text, '下载文件')

  return {
    text,
    href: toStringValue(props.href),
    displayMode,
    actionMode,
    triggerMode,
    target,
    downloadFileName: toStringValue(props.downloadFileName, text),
    disabled: toBooleanValue(props.disabled, false),
    underline: toBooleanValue(props.underline, true),
    fontSize: Math.max(10, toNumberValue(props.fontSize, 14)),
    fontWeight: Math.max(300, toNumberValue(props.fontWeight, 500)),
    textColor: toStringValue(props.textColor, '#2563eb'),
    backgroundColor: toStringValue(props.backgroundColor, '#ffffff'),
    borderColor: toStringValue(props.borderColor, '#2563eb'),
    borderRadius: Math.max(0, toNumberValue(props.borderRadius, 6)),
    paddingX: Math.max(0, toNumberValue(props.paddingX, 12)),
    paddingY: Math.max(0, toNumberValue(props.paddingY, 6)),
  }
}

/**
 * 根据配置生成下载组件内联样式。
 * @param {DownloadLinkConfig} config - 下载组件配置
 * @returns {Record<string, string>}
 */
export function buildDownloadLinkStyle(config: DownloadLinkConfig): Record<string, string> {
  const isButton = config.displayMode === 'button'
  return {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    boxSizing: 'border-box',
    width: '100%',
    height: '100%',
    padding: isButton ? `${config.paddingY}px ${config.paddingX}px` : '0',
    borderRadius: isButton ? `${config.borderRadius}px` : '0',
    border: isButton ? `1px solid ${config.borderColor}` : 'none',
    backgroundColor: isButton ? config.backgroundColor : 'transparent',
    color: config.textColor,
    fontSize: `${config.fontSize}px`,
    fontWeight: String(config.fontWeight),
    textDecoration: config.displayMode === 'link' && config.underline ? 'underline' : 'none',
    cursor: config.disabled ? 'not-allowed' : 'pointer',
    opacity: config.disabled ? '0.6' : '1',
    userSelect: 'none',
    transition: 'all 0.18s ease',
  }
}

/**
 * 根据触发模式判断当前事件是否应触发动作。
 * @param {DownloadLinkTriggerMode} triggerMode - 触发模式
 * @param {"click" | "dblclick"} eventName - 事件名
 * @returns {boolean}
 */
export function shouldTriggerByEvent(
  triggerMode: DownloadLinkTriggerMode,
  eventName: 'click' | 'dblclick',
): boolean {
  if (triggerMode === 'double') return eventName === 'dblclick'
  return eventName === 'click'
}
