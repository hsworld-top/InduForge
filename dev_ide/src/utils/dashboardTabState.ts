import { resolveDashboardTabTitle } from './dashboardTabTitle'

const SUPPORTED_MICRO_APP_TYPES = new Set(['designer', 'datacenter'])

type Translate = (key: string, params?: Record<string, unknown>) => string

interface MicroAppProps {
  appType: 'designer' | 'datacenter'
  project: Record<string, any> & { id: unknown; projectId?: unknown }
}

interface WorkspaceToolProps {
  target: '2d' | '3d'
  sceneId: string
  project: Record<string, any> & { id: unknown; projectId?: unknown }
}

interface DashboardTab {
  key: string
  title: string
  titleKey: string | null
  titlePrefix: string | null
  titleParams: Record<string, unknown> | null
  component: unknown
  icon: unknown
  props: MicroAppProps | WorkspaceToolProps | null
}

type PersistedTab =
  | {
      type: 'standard'
      key: string
    }
  | {
      type: 'micro-app'
      key: string
      titleKey: string | null
      titlePrefix: string
      titleParams: Record<string, unknown> | null
      title: string
      icon: string
      props: MicroAppProps
    }
  | {
      type: 'workspace-tool'
      key: string
      title: string
      icon: string
      props: WorkspaceToolProps
    }

interface SerializedDashboardTabState {
  tabs: PersistedTab[]
  activeTab: string
}

interface SerializeOptions {
  tabs?: Array<Record<string, any>>
  activeTab?: string
  tabConfigMap?: Record<string, Record<string, any>>
  hasTabPermission?: (key: string) => boolean
}

interface RestoreOptions {
  tabConfigMap?: Record<string, Record<string, any>>
  hasTabPermission?: (key: string) => boolean
  microAppComponent?: unknown
  workspaceToolComponent?: unknown
  translate?: Translate
}

interface SavedStateLike {
  tabs?: unknown[]
  activeTab?: unknown
}

/**
 * 判断值是否为普通对象。
 * 标签恢复只接受可序列化对象，避免把组件实例或响应式代理误写进 localStorage。
 *
 * @param {unknown} value - 待判断值
 * @returns {boolean} 是否为普通对象
 */
const isPlainObject = (value: unknown): value is Record<string, any> =>
  value !== null && typeof value === 'object' && !Array.isArray(value)

/**
 * 规范化标题参数。
 * i18n 参数需要保持对象结构；其他类型在恢复时没有意义，直接丢弃。
 *
 * @param {unknown} titleParams - 原始标题参数
 * @returns {Record<string, unknown>|null} 安全标题参数
 */
const normalizeTitleParams = (titleParams: unknown): Record<string, unknown> | null =>
  isPlainObject(titleParams) ? { ...titleParams } : null

/**
 * 校验并复制 Wujie 子应用所需的 props。
 * 当前仅设计中心与数据中心支持刷新后自动恢复，工程对象至少要保留 id。
 *
 * @param {unknown} props - 标签 props
 * @returns {{appType: string, project: Record<string, unknown>}|null} 安全 props
 */
const normalizeMicroAppProps = (props: unknown): MicroAppProps | null => {
  if (!isPlainObject(props)) return null

  const appType = typeof props.appType === 'string' ? props.appType : ''
  if (!SUPPORTED_MICRO_APP_TYPES.has(appType)) return null

  const project = isPlainObject(props.project) ? { ...props.project } : null
  const projectId = project?.id ?? project?.projectId ?? null
  if (!project || !projectId) return null

  if (!project.id) {
    project.id = projectId
  }

  return {
    appType: appType as MicroAppProps['appType'],
    project: project as MicroAppProps['project'],
  }
}

/**
 * 工具标签只持久化工程身份与目标类型，受控访问 URL 必须在恢复后重新获取。
 */
const normalizeWorkspaceToolProps = (props: unknown): WorkspaceToolProps | null => {
  if (
    !isPlainObject(props) ||
    !['2d', '3d'].includes(String(props.target)) ||
    typeof props.sceneId !== 'string' ||
    !props.sceneId.trim()
  )
    return null
  const project = isPlainObject(props.project) ? { ...props.project } : null
  const projectId = project?.id ?? project?.projectId ?? null
  if (!project || !projectId) return null
  project.id = projectId
  return {
    target: props.target as WorkspaceToolProps['target'],
    sceneId: props.sceneId.trim(),
    project: project as WorkspaceToolProps['project'],
  }
}

/**
 * 构造标准菜单标签的运行态对象。
 *
 * @param {string} key - 标签 key
 * @param {Record<string, any>} tabConfigMap - 标准标签配置映射
 * @param {(key: string, params?: Record<string, unknown>) => string} translate - 翻译函数
 * @returns {object|null} 可渲染标签
 */
const buildStandardTab = (
  key: string,
  tabConfigMap: Record<string, Record<string, any>>,
  translate: Translate,
): DashboardTab | null => {
  const config = tabConfigMap?.[key]
  if (!config) return null

  const titleKey = typeof config.titleKey === 'string' ? config.titleKey : null
  const normalizedTab: DashboardTab = {
    key,
    title: typeof config.title === 'string' ? config.title : '',
    titleKey,
    titlePrefix: null,
    titleParams: null,
    component: config.component,
    icon: config.icon,
    props: null,
  }

  normalizedTab.title = titleKey
    ? resolveDashboardTabTitle(normalizedTab, translate)
    : normalizedTab.title

  return normalizedTab
}

/**
 * 构造 Wujie 子应用工程标签的运行态对象。
 *
 * @param {Record<string, any>} record - 持久化记录
 * @param {any} microAppComponent - Wujie 子应用组件
 * @param {(key: string, params?: Record<string, unknown>) => string} translate - 翻译函数
 * @returns {object|null} 可渲染标签
 */
const buildMicroAppTab = (
  record: Record<string, any>,
  microAppComponent: unknown,
  translate: Translate,
): DashboardTab | null => {
  if (!isPlainObject(record) || !microAppComponent) return null

  const key = typeof record.key === 'string' ? record.key : ''
  const props = normalizeMicroAppProps(record.props)
  if (!key || !props) return null

  const normalizedTab: DashboardTab = {
    key,
    title: typeof record.title === 'string' ? record.title : '',
    titleKey: typeof record.titleKey === 'string' ? record.titleKey : null,
    titlePrefix: typeof record.titlePrefix === 'string' ? record.titlePrefix : '',
    titleParams: normalizeTitleParams(record.titleParams),
    component: microAppComponent,
    icon:
      typeof record.icon === 'string' && record.icon
        ? record.icon
        : props.appType === 'designer'
          ? 'design'
          : 'database',
    props,
  }

  normalizedTab.title = resolveDashboardTabTitle(normalizedTab, translate)

  return normalizedTab
}

const buildWorkspaceToolTab = (
  record: Record<string, any>,
  workspaceToolComponent: unknown,
): DashboardTab | null => {
  if (!isPlainObject(record) || !workspaceToolComponent) return null
  const key = typeof record.key === 'string' ? record.key : ''
  const props = normalizeWorkspaceToolProps(record.props)
  if (!key || !props) return null
  return {
    key,
    title: typeof record.title === 'string' ? record.title : '',
    titleKey: null,
    titlePrefix: null,
    titleParams: null,
    component: workspaceToolComponent,
    icon: typeof record.icon === 'string' && record.icon ? record.icon : 'design',
    props,
  }
}

/**
 * 序列化 Dashboard 标签状态。
 * 标准菜单页仍只保存 key；Wujie 工程页额外保存 appType、project 和标题元数据，
 * 这样刷新后能恢复子应用，并继续跟随宿主语言切换实时重算标题。
 *
 * @param {object} options - 序列化选项
 * @param {Array<object>} options.tabs - 当前标签列表
 * @param {string} options.activeTab - 当前激活标签 key
 * @param {Record<string, any>} options.tabConfigMap - 标准标签配置映射
 * @param {(key: string) => boolean} options.hasTabPermission - 权限校验函数
 * @returns {{tabs: Array<object>, activeTab: string}} 安全持久化结果
 */
export const serializeDashboardTabState = ({
  tabs = [],
  activeTab = '',
  tabConfigMap = {},
  hasTabPermission = () => true,
}: SerializeOptions = {}): SerializedDashboardTabState => {
  const persistedTabs: PersistedTab[] = []

  for (const tab of tabs) {
    if (!isPlainObject(tab) || typeof tab.key !== 'string') continue

    if (tabConfigMap[tab.key] && hasTabPermission(tab.key)) {
      persistedTabs.push({
        type: 'standard',
        key: tab.key,
      })
      continue
    }

    const microAppProps = normalizeMicroAppProps(tab.props)
    if (microAppProps) {
      persistedTabs.push({
        type: 'micro-app',
        key: tab.key,
        titleKey: typeof tab.titleKey === 'string' ? tab.titleKey : null,
        titlePrefix: typeof tab.titlePrefix === 'string' ? tab.titlePrefix : '',
        titleParams: normalizeTitleParams(tab.titleParams),
        title: typeof tab.title === 'string' ? tab.title : '',
        icon: typeof tab.icon === 'string' ? tab.icon : '',
        props: microAppProps,
      })
      continue
    }

    const workspaceToolProps = normalizeWorkspaceToolProps(tab.props)
    if (workspaceToolProps) {
      persistedTabs.push({
        type: 'workspace-tool',
        key: tab.key,
        title: typeof tab.title === 'string' ? tab.title : '',
        icon: typeof tab.icon === 'string' ? tab.icon : '',
        props: workspaceToolProps,
      })
    }
  }

  const persistedKeys = persistedTabs.map((tab) => tab.key).filter((key) => typeof key === 'string')

  return {
    tabs: persistedTabs,
    activeTab: persistedKeys.includes(activeTab) ? activeTab : persistedKeys[0] || '',
  }
}

/**
 * 从持久化结果恢复 Dashboard 标签状态。
 *
 * @param {object|null|undefined} savedState - localStorage 中的原始状态
 * @param {object} options - 恢复选项
 * @param {Record<string, any>} options.tabConfigMap - 标准标签配置映射
 * @param {(key: string) => boolean} options.hasTabPermission - 权限校验函数
 * @param {any} options.microAppComponent - Wujie 子应用组件
 * @param {(key: string, params?: Record<string, unknown>) => string} options.translate - 翻译函数
 * @returns {{tabs: Array<object>, activeTab: string}|null} 恢复结果
 */
export const restoreDashboardTabState = (
  savedState: SavedStateLike | null | undefined,
  {
    tabConfigMap = {},
    hasTabPermission = () => true,
    microAppComponent = null,
    workspaceToolComponent = null,
    translate = (key: string) => key,
  }: RestoreOptions = {},
): { tabs: DashboardTab[]; activeTab: string } | null => {
  if (!savedState || !Array.isArray(savedState.tabs)) return null

  const restoredTabs: DashboardTab[] = []

  for (const record of savedState.tabs) {
    if (typeof record === 'string') {
      if (!tabConfigMap[record] || !hasTabPermission(record)) continue
      const standardTab = buildStandardTab(record, tabConfigMap, translate)
      if (standardTab) restoredTabs.push(standardTab)
      continue
    }

    if (!isPlainObject(record)) continue

    if (
      (record.type === 'standard' || (!record.type && tabConfigMap[record.key])) &&
      typeof record.key === 'string'
    ) {
      if (!tabConfigMap[record.key] || !hasTabPermission(record.key)) continue
      const standardTab = buildStandardTab(record.key, tabConfigMap, translate)
      if (standardTab) restoredTabs.push(standardTab)
      continue
    }

    if (record.type === 'micro-app') {
      const microAppTab = buildMicroAppTab(record, microAppComponent, translate)
      if (microAppTab) restoredTabs.push(microAppTab)
      continue
    }

    if (record.type === 'workspace-tool') {
      const workspaceToolTab = buildWorkspaceToolTab(record, workspaceToolComponent)
      if (workspaceToolTab) restoredTabs.push(workspaceToolTab)
    }
  }

  if (restoredTabs.length === 0) return null

  const restoredKeys = restoredTabs.map((tab) => tab.key)
  const safeActiveTab =
    typeof savedState.activeTab === 'string' && restoredKeys.includes(savedState.activeTab)
      ? savedState.activeTab
      : restoredKeys[0]

  return {
    tabs: restoredTabs,
    activeTab: safeActiveTab,
  }
}
