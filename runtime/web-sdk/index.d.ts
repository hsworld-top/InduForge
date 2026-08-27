export interface SDKResult<T> {
  code: number
  msg: string
  data: T | null
  reqId?: string
}

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

export interface ComputeAdapter {
  run?(ref: string, input?: unknown): unknown
  describe?(ref: string): unknown
}

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
  readonly [segment: string]: PointPath
  byPath(path: string): DataPoint
  resolve(path: string): DataPoint
}

export interface AlarmSDK {
  items: { list(query?: unknown): Promise<SDKResult<unknown>>; get(id: string): Promise<SDKResult<unknown>> }
  settings: { get(): Promise<SDKResult<unknown>>; update(patch: unknown): Promise<SDKResult<unknown>> }
  current: { list(query?: unknown): Promise<SDKResult<unknown>>; get(id: string): Promise<SDKResult<unknown>> }
  changes: { subscribe(handler: (event: unknown) => void, options?: unknown): Promise<SDKResult<unknown>> }
  actions: {
    acknowledge(id: string, input?: unknown): Promise<SDKResult<unknown>>
    unacknowledge(id: string, input?: unknown): Promise<SDKResult<unknown>>
    forceClear(id: string, input?: unknown): Promise<SDKResult<unknown>>
    shelve(id: string, input?: unknown): Promise<SDKResult<unknown>>
    unshelve(id: string, input?: unknown): Promise<SDKResult<unknown>>
  }
  history: { list(query?: unknown): Promise<SDKResult<unknown>>; get(id: string): Promise<SDKResult<unknown>> }
}

export type ComputePath = {
  readonly [segment: string]: ComputePath
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
