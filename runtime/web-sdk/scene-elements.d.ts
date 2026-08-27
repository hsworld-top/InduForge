export interface SceneContractMember {
  name: string
  description?: string
  schema: Record<string, unknown>
}

export interface SceneParameter extends SceneContractMember { required: boolean }
export interface SceneCommand {
  name: string
  description?: string
  inputSchema: Record<string, unknown>
  outputSchema: Record<string, unknown>
}

export interface SceneContract {
  description: string
  parameters: SceneParameter[]
  events: SceneContractMember[]
  commands: SceneCommand[]
}

export interface SceneReadyDetail { sceneId: string; kind: '2d' | '3d'; revision: number }
export interface SceneEventDetail { sceneId: string; kind: '2d' | '3d'; name: string; payload: unknown }
export interface SceneErrorDetail { sceneId: string; kind: '2d' | '3d'; code: number; msg: string; requestId: string }

export declare class SceneRuntimeError extends Error {
  readonly code: number
  readonly requestId: string
  constructor(code: number, message: string, requestId?: string)
}

export declare class InduForgeSceneElement extends HTMLElement {
  sceneId: string
  params: Record<string, unknown>
  commandTimeout: number
  setParams(value: Record<string, unknown>): Promise<void>
  invoke(name: string, payload?: unknown): Promise<unknown>
  reload(): Promise<void>
}

export declare class InduForgeScene2DElement extends InduForgeSceneElement {}
export declare class InduForgeScene3DElement extends InduForgeSceneElement {}
export declare const SceneErrorCode: Readonly<Record<string, number>>

declare global {
  interface HTMLElementTagNameMap {
    'induforge-scene-2d': InduForgeScene2DElement
    'induforge-scene-3d': InduForgeScene3DElement
  }
}

declare module 'react' {
  namespace JSX {
    interface IntrinsicElements {
      'induforge-scene-2d': Record<string, unknown>
      'induforge-scene-3d': Record<string, unknown>
    }
  }
}
