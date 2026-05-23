/**
 * MockDataProvider - 设计态 Mock 数据提供者
 * 在设计态为组件提供模拟数据
 */

import { EventEmitter } from '../editor-core/utils/EventEmitter.ts'

export interface DatapointStatusInfo {
  path: string
  status: string
  dataType: string
}

export interface DatapointValue {
  path: string
  value: unknown
  timestamp: number
}

export interface MockConfig {
  path: string
  value?: unknown
  dataType?: string
  autoAnimate?: boolean
  animateMin?: number
  animateMax?: number
  animateInterval?: number
}

function inferDefaultValue(dataType: string | undefined): unknown {
  switch (dataType?.toLowerCase()) {
    case 'number':
    case 'int':
    case 'float':
    case 'double':
    case 'integer':
      return 0
    case 'boolean':
    case 'bool':
      return false
    case 'string':
    case 'text':
      return ''
    case 'array':
    case 'list':
      return []
    case 'object':
    case 'json':
      return {}
    case 'date':
    case 'datetime':
    case 'timestamp':
      return new Date().toISOString()
    default:
      return null
  }
}

function inferDataType(path: string): string {
  const lowerPath = path.toLowerCase()

  if (
    lowerPath.includes('temp') ||
    lowerPath.includes('pressure') ||
    lowerPath.includes('speed') ||
    lowerPath.includes('level') ||
    lowerPath.includes('flow') ||
    lowerPath.includes('value') ||
    lowerPath.includes('count') ||
    lowerPath.includes('voltage') ||
    lowerPath.includes('current') ||
    lowerPath.includes('power')
  ) {
    return 'number'
  }

  if (
    lowerPath.includes('status') ||
    lowerPath.includes('state') ||
    lowerPath.includes('running') ||
    lowerPath.includes('enabled') ||
    lowerPath.includes('active') ||
    lowerPath.includes('alarm') ||
    lowerPath.includes('fault')
  ) {
    return 'boolean'
  }

  if (lowerPath.includes('time') || lowerPath.includes('date') || lowerPath.includes('timestamp')) {
    return 'datetime'
  }

  return 'string'
}

export class MockDataProvider extends EventEmitter {
  private readonly _configs = new Map<string, MockConfig>()
  private readonly _values = new Map<string, unknown>()
  private readonly _animationTimers = new Map<string, ReturnType<typeof setInterval>>()
  private readonly _subscriptions = new Set<string>()

  setMock(path: string, config: MockConfig | unknown): void {
    const mockConfig: MockConfig =
      config && typeof config === 'object' && 'path' in (config as object)
        ? (config as MockConfig)
        : { path, value: config }

    this._configs.set(path, mockConfig)
    this._values.set(path, mockConfig.value)

    if (mockConfig.autoAnimate) {
      this._startAnimation(path, mockConfig)
    }

    if (this._subscriptions.has(path)) {
      this._notifyChange(path, mockConfig.value)
    }
  }

  setMocks(configs: Record<string, MockConfig | unknown>): void {
    for (const [path, config] of Object.entries(configs)) {
      this.setMock(path, config)
    }
  }

  removeMock(path: string): void {
    this._stopAnimation(path)
    this._configs.delete(path)
    this._values.delete(path)
  }

  clearMocks(): void {
    for (const path of this._animationTimers.keys()) {
      this._stopAnimation(path)
    }
    this._configs.clear()
    this._values.clear()
  }

  getMockConfig(path: string): MockConfig | undefined {
    return this._configs.get(path)
  }

  getValue(path: string, binding?: { designMock?: unknown }): unknown {
    if (binding?.designMock !== undefined) {
      return binding.designMock
    }

    if (this._values.has(path)) {
      return this._values.get(path)
    }

    const config = this._configs.get(path)
    if (config?.value !== undefined) {
      return config.value
    }

    const dataType = config?.dataType || inferDataType(path)
    return inferDefaultValue(dataType)
  }

  getStatus(path: string): DatapointStatusInfo {
    const config = this._configs.get(path)
    return {
      path,
      status: 'active',
      dataType: config?.dataType || inferDataType(path),
    }
  }

  setValue(path: string, value: unknown): void {
    this._values.set(path, value)

    if (this._subscriptions.has(path)) {
      this._notifyChange(path, value)
    }
  }

  subscribe(path: string, callback: (data: DatapointValue) => void): () => void {
    this._subscriptions.add(path)

    const handler = (...args: unknown[]) => {
      const data = args[0] as { path: string; value: unknown; timestamp: number }
      if (data.path === path) {
        callback(data)
      }
    }

    this.on('change', handler)

    const currentValue = this.getValue(path)
    callback({
      path,
      value: currentValue,
      timestamp: Date.now(),
    })

    return () => {
      this.off('change', handler)
      this._subscriptions.delete(path)
    }
  }

  subscribeMany(paths: string[], callback: (data: DatapointValue) => void): () => void {
    const unsubscribes = paths.map((path) => this.subscribe(path, callback))
    return () => {
      unsubscribes.forEach((unsub) => unsub())
    }
  }

  private _notifyChange(path: string, value: unknown): void {
    this.emit('change', {
      path,
      value,
      timestamp: Date.now(),
    })
  }

  private _startAnimation(path: string, config: MockConfig): void {
    this._stopAnimation(path)

    const { animateMin = 0, animateMax = 100, animateInterval = 1000 } = config

    const timer = setInterval(() => {
      const value = animateMin + Math.random() * (animateMax - animateMin)
      const roundedValue = Math.round(value * 100) / 100

      this._values.set(path, roundedValue)

      if (this._subscriptions.has(path)) {
        this._notifyChange(path, roundedValue)
      }
    }, animateInterval)

    this._animationTimers.set(path, timer)
  }

  private _stopAnimation(path: string): void {
    const timer = this._animationTimers.get(path)
    if (timer) {
      clearInterval(timer)
      this._animationTimers.delete(path)
    }
  }

  loadPreset(sceneName: string): void {
    const presets: Record<string, Record<string, unknown>> = {
      factory: {
        'factory/line1/temp': { value: 45.5, dataType: 'number' },
        'factory/line1/pressure': { value: 1.2, dataType: 'number' },
        'factory/line1/status': { value: true, dataType: 'boolean' },
        'factory/line1/count': { value: 1250, dataType: 'number' },
        'factory/line2/temp': { value: 52.3, dataType: 'number' },
        'factory/line2/pressure': { value: 1.5, dataType: 'number' },
        'factory/line2/status': { value: false, dataType: 'boolean' },
        'factory/line2/count': { value: 980, dataType: 'number' },
      },
      energy: {
        'energy/voltage': { value: 220, dataType: 'number' },
        'energy/current': { value: 15.5, dataType: 'number' },
        'energy/power': { value: 3410, dataType: 'number' },
        'energy/consumption': { value: 12580, dataType: 'number' },
        'energy/pf': { value: 0.92, dataType: 'number' },
      },
      environment: {
        'env/temperature': { value: 24.5, dataType: 'number' },
        'env/humidity': { value: 65, dataType: 'number' },
        'env/pm25': { value: 35, dataType: 'number' },
        'env/co2': { value: 420, dataType: 'number' },
      },
      animated: {
        'demo/sine': {
          value: 50,
          dataType: 'number',
          autoAnimate: true,
          animateMin: 0,
          animateMax: 100,
          animateInterval: 500,
        },
        'demo/random': {
          value: 0,
          dataType: 'number',
          autoAnimate: true,
          animateMin: -50,
          animateMax: 50,
          animateInterval: 1000,
        },
      },
    }

    const preset = presets[sceneName]
    if (preset) {
      this.setMocks(preset)
    } else {
      console.warn(`Unknown preset: ${sceneName}`)
    }
  }

  destroy(): void {
    for (const path of this._animationTimers.keys()) {
      this._stopAnimation(path)
    }

    this._configs.clear()
    this._values.clear()
    this._subscriptions.clear()

    this.removeAllListeners()
  }
}

export function createMockDataProvider(): MockDataProvider {
  return new MockDataProvider()
}

export default MockDataProvider
