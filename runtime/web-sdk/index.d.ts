export interface RuntimeAdapter {
  get(path: string, params?: unknown): unknown
  set(path: string, value: unknown): unknown
  subscribe(path: string, handler: (value: unknown) => void): unknown
  publish(path: string, payload: unknown): unknown
}

export interface NavigationAdapter {
  open2D(sceneId: string, options?: unknown): unknown
  open3D(sceneId: string, options?: unknown): unknown
}

export interface RuntimeConfiguration {
  adapter?: RuntimeAdapter
  navigation?: NavigationAdapter
  roles?: string[]
  access?: {
    roles?: string[]
  }
}

export interface PointOperations {
  get(params?: unknown): unknown
  set(value: unknown): unknown
  sub(handler: (value: unknown) => void): unknown
  pub(payload: unknown): unknown
}

export type PointPath = PointOperations & {
  readonly [segment: string]: PointPath
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
  access: RuntimeAccess
  scenes: RuntimeScenes
}

declare global {
  interface Window {
    __INDUFORGE_RUNTIME__?: RuntimeConfiguration
  }
}

export const points: PointPath
export const access: RuntimeAccess
export const scenes: RuntimeScenes

export function configureRuntime(runtime: RuntimeConfiguration): RuntimeClient
export function createRuntimeClient(runtime?: RuntimeConfiguration): RuntimeClient
