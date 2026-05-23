/**
 * 组件 Manifest 注册表：属性、样式、事件等元数据
 */

import { i18n } from '@/i18n'

export type PropType = 'string' | 'number' | 'boolean' | 'color' | 'enum' | 'object' | 'array'

export interface PropOption {
  label: string
  value: unknown
}

export interface PropDefinition {
  name: string
  type: PropType
  label: string
  group?: string
  defaultValue?: unknown
  options?: PropOption[]
  min?: number
  max?: number
  step?: number
  placeholder?: string
  /** 是否允许数据绑定 */
  bindable?: boolean
  /** 属性面板编辑器类型，如 code */
  editor?: string
  language?: string
  height?: string
  /** 是否隐藏在基础属性面板，但仍参与默认值与底层渲染 */
  hidden?: boolean
}

export interface EventDefinition {
  name: string
  label: string
  description?: string
}

/** 画布/属性面板消费的组件清单（与 editor-core registry 对齐的公共子集） */
export interface ComponentManifest {
  type: string
  name: string
  category: string
  props: PropDefinition[]
  defaultSize?: { width: number; height: number }
  isContainer?: boolean
  events?: EventDefinition[]
  defaultStyle?: Record<string, string | number>
  icon?: string
  description?: string
  allowedChildren?: string[]
  slots?: Record<string, unknown>
}

const manifestRegistry = new Map<string, ComponentManifest>()

const manifestLiteralKeyMap: Record<string, string> = {
  按钮: 'componentManifest.names.Button',
  水平布局: 'componentManifest.names.HorizontalLayout',
  垂直布局: 'componentManifest.names.VerticalLayout',
  折叠面板布局: 'componentManifest.names.Collapse',
  选项卡布局: 'componentManifest.names.Tabs',
  表单布局: 'componentManifest.names.FormLayout',
  区域布局: 'componentManifest.names.ElContainer',
  页面根布局: 'componentManifest.names.FreeContainer',
  布局: 'componentManifest.groups.layout',
  显示: 'componentManifest.groups.display',
  数据: 'componentManifest.groups.data',
  状态: 'componentManifest.groups.state',
  样式: 'componentManifest.groups.style',
  功能: 'componentManifest.groups.features',
  属性: 'componentManifest.groups.props',
  内容: 'componentManifest.groups.content',
  外观: 'componentManifest.groups.appearance',
  更多属性: 'componentManifest.groups.more',
  按钮文字: 'componentManifest.labels.buttonText',
  按钮类型: 'componentManifest.labels.buttonType',
  图标: 'componentManifest.labels.icon',
  尺寸: 'componentManifest.labels.size',
  视觉规格: 'componentManifest.labels.visualSpec',
  朴素按钮: 'componentManifest.labels.plainButton',
  圆角样式: 'componentManifest.labels.buttonShape',
  禁用: 'componentManifest.labels.disabled',
  加载中: 'componentManifest.labels.loading',
  圆角: 'componentManifest.labels.round',
  圆形: 'componentManifest.labels.circle',
  区域预设: 'componentManifest.labels.regionPreset',
  Header区域: 'componentManifest.labels.showHeader',
  Aside区域: 'componentManifest.labels.showAside',
  Main区域: 'componentManifest.labels.showMain',
  Footer区域: 'componentManifest.labels.showFooter',
  Header高度: 'componentManifest.labels.headerHeight',
  Aside宽度: 'componentManifest.labels.asideWidth',
  Footer高度: 'componentManifest.labels.footerHeight',
  水平排列: 'componentManifest.labels.horizontalJustify',
  垂直对齐: 'componentManifest.labels.verticalAlign',
  间距: 'componentManifest.labels.gap',
  垂直排列: 'componentManifest.labels.verticalJustify',
  水平对齐: 'componentManifest.labels.horizontalAlign',
  表单项间距: 'componentManifest.labels.itemGap',
  显示边框: 'componentManifest.labels.showBorder',
  面板: 'componentManifest.labels.panels',
  默认激活: 'componentManifest.labels.defaultActive',
  风格: 'componentManifest.labels.variant',
  可关闭: 'componentManifest.labels.closable',
  标签位置: 'componentManifest.labels.tabPosition',
  宽度自撑: 'componentManifest.labels.stretch',
  标签页: 'componentManifest.labels.tabs',
  起始: 'componentManifest.options.start',
  居中: 'componentManifest.options.center',
  末尾: 'componentManifest.options.end',
  两端: 'componentManifest.options.between',
  环绕: 'componentManifest.options.around',
  均匀: 'componentManifest.options.evenly',
  拉伸: 'componentManifest.options.stretch',
  顶部: 'componentManifest.options.top',
  底部: 'componentManifest.options.bottom',
  基线: 'componentManifest.options.baseline',
  左侧: 'componentManifest.options.left',
  右侧: 'componentManifest.options.right',
  默认: 'componentManifest.options.default',
  卡片: 'componentManifest.options.card',
  边框卡片: 'componentManifest.options.borderCard',
  上: 'componentManifest.options.topShort',
  右: 'componentManifest.options.rightShort',
  下: 'componentManifest.options.bottomShort',
  左: 'componentManifest.options.leftShort',
  上下: 'componentManifest.options.topMain',
  '上左下（左单独一列）': 'componentManifest.options.asideFullHeight',
  '上左下（左被上下夹着）': 'componentManifest.options.asideBetween',
  主要: 'componentManifest.options.primary',
  成功: 'componentManifest.options.success',
  警告: 'componentManifest.options.warning',
  危险: 'componentManifest.options.danger',
  信息: 'componentManifest.options.info',
  文本: 'componentManifest.options.text',
  大: 'componentManifest.options.large',
  小: 'componentManifest.options.small',
  按钮圆角选项: 'componentManifest.options.round',
  按钮圆形选项: 'componentManifest.options.circle',
  '图标-搜索': 'componentManifest.options.iconSearch',
  '图标-新增': 'componentManifest.options.iconPlus',
  '图标-下载': 'componentManifest.options.iconDownload',
  '图标-上传': 'componentManifest.options.iconUpload',
  '图标-删除': 'componentManifest.options.iconDelete',
  '图标-编辑': 'componentManifest.options.iconEdit',
  '图标-刷新': 'componentManifest.options.iconRefresh',
  '图标-关闭': 'componentManifest.options.iconClose',
  '图标-文档': 'componentManifest.options.iconDocument',
  '图标-文件夹': 'componentManifest.options.iconFolder',
  '如: el-icon-search': 'componentManifest.placeholders.icon',
  '如: Search / Plus / Download': 'componentManifest.placeholders.buttonIcon',
  选择或输入图标名: 'componentManifest.placeholders.buttonIconInput',
}

function translateManifestLiteral(value: string | undefined): string {
  const text = String(value || '')
  const key = manifestLiteralKeyMap[text]
  return key ? i18n.global.t(key) : text
}

function localizePropOption(option: PropOption): PropOption {
  return {
    ...option,
    label: translateManifestLiteral(option.label),
  }
}

function localizePropDefinition(prop: PropDefinition): PropDefinition {
  return {
    ...prop,
    label: translateManifestLiteral(prop.label),
    ...(prop.group !== undefined ? { group: translateManifestLiteral(prop.group) } : {}),
    ...(prop.placeholder !== undefined
      ? { placeholder: translateManifestLiteral(prop.placeholder) }
      : {}),
    ...(Array.isArray(prop.options) ? { options: prop.options.map(localizePropOption) } : {}),
  }
}

function localizeEventDefinition(event: EventDefinition): EventDefinition {
  return {
    ...event,
    label: translateManifestLiteral(event.label),
    ...(event.description !== undefined
      ? { description: translateManifestLiteral(event.description) }
      : {}),
  }
}

function localizeManifest(manifest: ComponentManifest): ComponentManifest {
  return {
    ...manifest,
    name: translateManifestLiteral(manifest.name),
    props: Array.isArray(manifest.props) ? manifest.props.map(localizePropDefinition) : [],
    ...(manifest.description !== undefined
      ? { description: translateManifestLiteral(manifest.description) }
      : {}),
    ...(Array.isArray(manifest.events)
      ? { events: manifest.events.map(localizeEventDefinition) }
      : {}),
  }
}

export function registerManifest(manifest: ComponentManifest): void {
  if (!manifest?.type) {
    throw new Error('Invalid manifest: missing type')
  }
  manifestRegistry.set(manifest.type, manifest)
}

export function getManifest(type: string): ComponentManifest | undefined {
  const manifest = manifestRegistry.get(type)
  return manifest ? localizeManifest(manifest) : undefined
}

export function getAllManifests(): ComponentManifest[] {
  return Array.from(manifestRegistry.values(), localizeManifest)
}

export function getManifestsByCategory(category: string): ComponentManifest[] {
  return getAllManifests().filter((m) => m.category === category)
}

export default {
  registerManifest,
  getManifest,
  getAllManifests,
  getManifestsByCategory,
}
