export interface SDKResult<T> {
  code: number
  msg: string
  data: T | null
  reqId?: string
}

export function createBrowserRuntime(): RuntimeConfiguration

export interface DataPointSample<T = unknown> {
  path: string
  value: T
  quality: string
  timestamp: string | null
  observedAt?: string | null
  sourceTimestamp?: string | null
  status: string | null
}

export interface DataPointCapabilities {
  get?: boolean
  read?: boolean
  peek?: boolean
  set?: boolean
  subscribe?: boolean
  history?: boolean
  refresh?: boolean
  run?: boolean
  execute?: boolean
  publish?: boolean
  [operation: string]: boolean | undefined
}

export interface DataPointContract<T = unknown> {
  id?: string | null
  path?: string
  name?: string | null
  displayName?: string | null
  dataType?: string | null
  schema?: Record<string, unknown> | null
  source?: { type?: string | null; id?: string | null }
  sourceType?: string | null
  sourceId?: string | null
  status?: string | null
  unit?: string | null
  precision?: number | null
  min?: number | null
  max?: number | null
  defaultValue?: T | null
  tags?: unknown[]
  attributes?: Record<string, string>
  attributeDefaults?: Record<string, string>
  capabilities?: DataPointCapabilities
}

export interface RuntimeAdapter {
  get?(path: string, options?: unknown): unknown
  read?(path: string, options?: unknown): unknown
  peek?(path: string): unknown
  set?(path: string, value: unknown, options?: unknown): unknown
  subscribe?(path: string, handler: (value: unknown) => void, options?: unknown): unknown
  history?(path: string, query?: unknown): unknown
  refresh?(path: string, options?: unknown): unknown
  run?(path: string, input?: unknown): unknown
  execute?(path: string, input?: unknown): unknown
  publish?(path: string, payload: unknown, options?: unknown): unknown
}

export interface AlarmAdapter {
  listItems?(query?: unknown): unknown
  getItem?(id: string): unknown
  getSettings?(): unknown
  updateSettings?(patch: unknown): unknown
  listCurrent?(query?: unknown): unknown
  getCurrent?(id: string): unknown
  subscribeChanges?(handler: (event: unknown) => void, options?: unknown): unknown
  acknowledge?(id: string, input?: unknown): unknown
  unacknowledge?(id: string, input?: unknown): unknown
  forceClear?(id: string, input?: unknown): unknown
  shelve?(id: string, input?: unknown): unknown
  unshelve?(id: string, input?: unknown): unknown
  listHistory?(query?: unknown): unknown
  getHistory?(id: string): unknown
}

export interface AlarmAcknowledgementInput {
  expectedVersion: number
  comment?: string
  /** 仅 HTTP Runtime 使用，用于传入 signal、headers、credentials 或 timeoutMs。 */
  requestOptions?: HttpRequestOptions
}

export interface AlarmAcknowledgement {
  version: number
  acknowledgedAt: string
}

export interface AlarmSourceDatapoint {
  datapointId: string
  path: string
}

export interface RuntimeAlarmState {
  alarmItemId: string
  revision: number
  state: unknown
  version: number
  updatedAt: string
  /** 来自当前 deployment 已校验发布工件的报警展示名称。 */
  name?: string
  /** 普通单点报警的来源路径；组合报警请使用 sourceDatapoints。 */
  datapointPath?: string
  sourceDatapoints?: AlarmSourceDatapoint[]
}

export interface RuntimeAlarmItem {
  id: string
  revision: number
  name: string
  displayName: string
  enabled: boolean
  mode: string
  inputs: AlarmSourceDatapoint[]
}

export interface ComputeAdapter {
  run?(ref: string, input?: ComputeRunInput): unknown
  describe?(ref: string): unknown
  status?(commandId: string, requestOptions?: HttpRequestOptions): unknown
}

export interface ComputeRunInput { idempotencyKey?: string }
export interface ComputeCommandStatus { commandId: string; status: 'queued' | 'running' | 'succeeded' | 'failed'; resultRefs?: string[]; resultVersion?: number; failureCode?: string }
export interface ComputeCommandHandle { commandId: string; status(options?: HttpRequestOptions): Promise<SDKResult<ComputeCommandStatus>>; wait(options?: { timeoutMs?: number; intervalMs?: number } & HttpRequestOptions): Promise<SDKResult<ComputeCommandStatus>> }

export interface NavigationAdapter {
  open2D(sceneId: string, options?: unknown): unknown
  open3D(sceneId: string, options?: unknown): unknown
}

export interface RuntimeConfiguration {
  adapter?: RuntimeAdapter
  alarmAdapter?: AlarmAdapter
  computeAdapter?: ComputeAdapter
  pointContracts?: Record<string, DataPointContract>
  navigation?: NavigationAdapter
  roles?: string[]
  access?: { roles?: string[] }
  sceneResolver?: {
    resolve(input: { sceneId: string; kind: '2d' | '3d' }): Promise<SceneResolution>
  }
}

/** Runtime API 请求可选项；除历史和报警列表外不参与 URL 查询串。 */
export interface HttpRequestOptions {
  signal?: AbortSignal
  timeoutMs?: number
  headers?: HeadersInit
  credentials?: RequestCredentials
}

export interface HttpRuntimeIdentity {
  deploymentId?: string
  projectId?: string
}

export interface PointSubscriptionOptions extends HttpRequestOptions {
  onError?: (error: Error) => void
  onClose?: (event: { code: number; reason: string; wasClean: boolean }) => void
}

export interface RuntimeCatalog {
  schemaVersion: string
  artifactDigest: string
  points: DataPointContract[]
  computes: RuntimeComputeUnit[]
  alarms: RuntimeAlarmItem[]
}

export interface RuntimeComputeUnit {
  id: string
  revision: number
  name: string
  description?: string | null
  enabled: boolean
  inputs?: unknown
  outputs?: unknown
  trigger?: unknown
}

export interface RuntimeSession {
  subjectId: string
  roles: string[]
  expiresAt?: string
}

export interface HttpRuntimeSession {
  establish(accessToken?: string, options?: HttpRequestOptions): Promise<SDKResult<RuntimeSession>>
  query(options?: HttpRequestOptions): Promise<SDKResult<RuntimeSession>>
  exit(options?: HttpRequestOptions): Promise<SDKResult<{ revoked: boolean }>>
}

export interface HttpRuntimeCatalog {
  get(options?: HttpRequestOptions): Promise<SDKResult<RuntimeCatalog>>
}

export interface HttpRuntimeOptions {
  /** 默认同源 /api/v1/runtime，由 Project Gateway 注入工程身份。 */
  baseUrl?: string
  /** 默认同源 /ws/v1/points。baseUrl 不规则时请显式指定。 */
  wsUrl?: string
  alarmWsUrl?: string
  /** 会话建立时使用的 Bearer Token；不会保存到 SDK 外部。 */
  accessToken?: string
  /** 仅直连 Runtime API 或测试时传入；经 Gateway 的浏览器请求通常不应设置。 */
  identity?: HttpRuntimeIdentity
  timeoutMs?: number
  credentials?: RequestCredentials
  fetch?: typeof globalThis.fetch
  WebSocket?: typeof globalThis.WebSocket
}

export interface HttpRuntimeConfiguration extends RuntimeConfiguration {
  pointContracts: Record<string, DataPointContract>
  session: HttpRuntimeSession
  catalog: HttpRuntimeCatalog
}

export interface SceneResolution {
  url: string
  expiresAt: string
  revision: number
  contract: Record<string, unknown>
}

export interface DataPoint<T = unknown> {
  readonly id: string | null
  readonly ref: string
  readonly path: string
  readonly name: string | null
  readonly displayName: string | null
  readonly dataType: string | null
  readonly schema: Record<string, unknown> | null
  readonly source: { type: string | null; id: string | null }
  readonly status: string | null
  readonly unit: string | null
  readonly precision: number | null
  readonly min: number | null
  readonly max: number | null
  readonly defaultValue: T | null
  readonly tags: readonly unknown[]
  readonly attributes: Readonly<Record<string, string>>
  readonly capabilities: Readonly<DataPointCapabilities>
  get(options?: unknown): Promise<SDKResult<T>>
  read(options?: unknown): Promise<SDKResult<DataPointSample<T>>>
  peek(): Promise<SDKResult<DataPointSample<T>>>
  set(value: T, options?: unknown): Promise<SDKResult<unknown>>
  subscribe(handler: (value: T) => void, options?: unknown): Promise<SDKResult<unknown>>
  history(query?: unknown): Promise<SDKResult<DataPointSample<T>[]>>
  refresh(options?: unknown): Promise<SDKResult<unknown>>
  run(input?: unknown): Promise<SDKResult<unknown>>
  execute(input?: unknown): Promise<SDKResult<unknown>>
  publish(payload: unknown, options?: unknown): Promise<SDKResult<unknown>>
  sub(handler: (value: T) => void, options?: unknown): Promise<SDKResult<unknown>>
  pub(payload: unknown, options?: unknown): Promise<SDKResult<unknown>>
}

export type PointPath<T = unknown> = DataPoint<T> & {
  /** 动态路径段无法在声明期枚举，因此保留链式调用的宽松类型。 */
  readonly [segment: string]: any
  byPath(path: string): DataPoint
  resolve(path: string): DataPoint
}

export interface AlarmSDK {
  items: {
    list(query?: unknown): Promise<SDKResult<unknown>>
    get(id: string): Promise<SDKResult<unknown>>
  }
  settings: {
    get(): Promise<SDKResult<unknown>>
    update(patch: unknown): Promise<SDKResult<unknown>>
  }
  current: {
    list(query?: unknown): Promise<SDKResult<{ items: RuntimeAlarmState[]; total: number }>>
    get(id: string): Promise<SDKResult<unknown>>
  }
  changes: {
    subscribe(handler: (event: unknown) => void, options?: unknown): Promise<SDKResult<unknown>>
  }
  actions: {
    acknowledge(id: string, input: AlarmAcknowledgementInput): Promise<SDKResult<AlarmAcknowledgement>>
    unacknowledge(id: string, input?: unknown): Promise<SDKResult<unknown>>
    forceClear(id: string, input?: unknown): Promise<SDKResult<unknown>>
    shelve(id: string, input?: unknown): Promise<SDKResult<unknown>>
    unshelve(id: string, input?: unknown): Promise<SDKResult<unknown>>
  }
  history: {
    list(query?: unknown): Promise<SDKResult<unknown>>
    get(id: string): Promise<SDKResult<unknown>>
  }
}

export type ComputePath = {
  /** 动态计算引用无法在声明期枚举，因此保留链式调用的宽松类型。 */
  readonly [segment: string]: any
  run(input?: unknown): Promise<SDKResult<unknown>>
  describe(): Promise<SDKResult<unknown>>
  byRef(ref: string): ComputePath
  resolve(ref: string): ComputePath
}

export interface RuntimeAccess {
  hasRole(role: string): boolean
  hasAnyRole(roles: string[]): boolean
}

export interface RuntimeScenes {
  open2D(sceneId: string, options?: unknown): unknown
  open3D(sceneId: string, options?: unknown): unknown
}

export interface RuntimeClient {
  points: PointPath
  alarms: AlarmSDK
  computes: ComputePath
  access: RuntimeAccess
  scenes: RuntimeScenes
}

declare global {
  interface Window {
    __INDUFORGE_RUNTIME__?: RuntimeConfiguration
  }
}

export const points: PointPath
export const alarms: AlarmSDK
export const computes: ComputePath
export const access: RuntimeAccess
export const scenes: RuntimeScenes

export function configureRuntime(runtime: RuntimeConfiguration): RuntimeClient
export function createRuntimeClient(runtime?: RuntimeConfiguration): RuntimeClient
/** 创建发布态 Runtime API 的 HTTP/WebSocket 适配器，再传给 configureRuntime 或 createRuntimeClient。 */
export function createHttpRuntime(options?: HttpRuntimeOptions): HttpRuntimeConfiguration
