import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import vm from 'node:vm'
import { describe, expect, it, vi } from 'vitest'

const runtimeSource = readFileSync(
  resolve(process.cwd(), 'public/ht-editor/custom/libs/InduForgeSceneRuntime.js'),
  'utf8',
)

function createRuntime() {
  const messages = []
  const parent = { postMessage: vi.fn((message, origin) => messages.push({ message, origin })) }
  const listeners = new Map()
  const viewerWindow = {
    parent,
    addEventListener(type, listener) {
      listeners.set(type, listener)
    },
    setTimeout,
    clearTimeout,
  }
  const context = vm.createContext({
    window: viewerWindow,
    document: { referrer: 'https://preview.workspace.test/page/' },
    URL,
    Map,
    Set,
    Promise,
    Object,
    String,
    Number,
    TypeError,
    Error,
    setTimeout,
    clearTimeout,
  })
  vm.runInContext(runtimeSource, context)
  return {
    api: viewerWindow.InduForgeScene,
    sceneRuntime: viewerWindow.InduForgeSceneRuntime,
    messages,
    receive(data) {
      listeners.get('message')({ source: parent, origin: 'https://preview.workspace.test', data })
    },
  }
}

function protocolMessage(type, data = {}) {
  return {
    channel: 'induforge-scene-runtime',
    version: 1,
    sceneId: 'scene-a',
    type,
    ...data,
  }
}

describe('InduForgeScene Viewer API', () => {
  it('接收完整参数对象并通知监听器', () => {
    const runtime = createRuntime()
    runtime.api.configure({ sceneId: 'scene-a' })
    const listener = vi.fn()
    const unsubscribe = runtime.api.onParamsChange(listener)

    runtime.receive(protocolMessage('INITIALIZE', { params: { deviceId: 'A01' } }))
    expect(runtime.api.getParams()).toEqual({ deviceId: 'A01' })
    expect(listener).toHaveBeenCalledWith({ deviceId: 'A01' })

    unsubscribe()
    runtime.receive(protocolMessage('PARAMS_CHANGED', { params: { deviceId: 'A02' } }))
    expect(listener).toHaveBeenCalledTimes(1)
  })

  it('发送 READY 和公开场景事件到精确父窗口 origin', () => {
    const runtime = createRuntime()
    runtime.api.configure({ sceneId: 'scene-a' })

    runtime.api.markReady()
    runtime.api.emit('selected', { id: 'A01' })

    expect(runtime.messages).toEqual([
      { message: protocolMessage('READY'), origin: 'https://preview.workspace.test' },
      {
        message: protocolMessage('EVENT', { name: 'selected', payload: { id: 'A01' } }),
        origin: 'https://preview.workspace.test',
      },
    ])
  })

  it('执行已注册命令并拒绝重复或未注册命令', async () => {
    const runtime = createRuntime()
    runtime.api.configure({ sceneId: 'scene-a' })
    const unregister = runtime.api.registerCommand('focusDevice', async ({ id }) => ({ id }))
    expect(() => runtime.api.registerCommand('focusDevice', () => {})).toThrow('已注册')

    runtime.receive(protocolMessage('COMMAND_REQUEST', {
      requestId: 'command-1',
      name: 'focusDevice',
      payload: { id: 'A01' },
    }))
    await vi.waitFor(() => expect(runtime.messages.at(-1)?.message).toEqual(
      protocolMessage('COMMAND_RESULT', { requestId: 'command-1', result: { id: 'A01' } }),
    ))

    unregister()
    runtime.receive(protocolMessage('COMMAND_REQUEST', {
      requestId: 'command-2',
      name: 'focusDevice',
      payload: { id: 'A02' },
    }))
    expect(runtime.messages.at(-1)?.message).toEqual(protocolMessage('COMMAND_RESULT', {
      requestId: 'command-2',
      error: { code: 26005, msg: '场景未注册命令 focusDevice' },
    }))
  })

  it('控件启停命令调用 HT 原生 setEnabled', async () => {
    const runtime = createRuntime()
    runtime.api.configure({ sceneId: 'scene-a' })
    const setEnabled = vi.fn()
    const data = {
      a: (name) => name === 'induforge.interactions'
        ? { commands: [{ name: 'enableInput', action: 'enable' }, { name: 'disableInput', action: 'disable' }] }
        : null,
      setEnabled,
    }
    const dm = { each: (callback) => callback(data), addDataPropertyChangeListener: vi.fn() }
    runtime.sceneRuntime.bindGraphView({ dm, addInteractorListener: vi.fn() })

    runtime.receive(protocolMessage('COMMAND_REQUEST', { requestId: 'enable-1', name: 'enableInput', payload: {} }))
    runtime.receive(protocolMessage('COMMAND_REQUEST', { requestId: 'disable-1', name: 'disableInput', payload: {} }))

    await vi.waitFor(() => expect(setEnabled).toHaveBeenNthCalledWith(2, false))
    expect(setEnabled).toHaveBeenNthCalledWith(1, true)
  })

  it('双向写入失败后恢复最后一次服务端确认值', async () => {
    const runtime = createRuntime()
    runtime.api.configure({ sceneId: 'scene-a' })
    let currentValue = 0
    let propertyListener
    const setValue = vi.fn((value) => { currentValue = value })
    const binding = { pointPath: 'line.temperature', direction: 'twoWay', readMode: 'subscribe' }
    const data = {
      a: (name) => name === 'induforge.bindings' ? { value: binding } : null,
      getValue: () => currentValue,
      setValue,
    }
    const dm = {
      each: (callback) => callback(data),
      addDataPropertyChangeListener: (listener) => { propertyListener = listener },
    }
    runtime.sceneRuntime.bindGraphView({ dm, addInteractorListener: vi.fn() })
    const subscribeRequest = runtime.messages.find(({ message }) => message.type === 'DATA_REQUEST' && message.operation === 'sub').message
    runtime.receive(protocolMessage('DATA_RESULT', { requestId: subscribeRequest.requestId, data: { subscribed: true } }))
    runtime.receive(protocolMessage('DATA_PUSH', { path: 'line.temperature', data: { value: 10, quality: 'good' } }))
    await vi.waitFor(() => expect(currentValue).toBe(10))
    await new Promise((resolve) => setTimeout(resolve, 0))

    currentValue = 20
    propertyListener({ data, property: 'value' })
    const writeRequest = runtime.messages.findLast(({ message }) => message.type === 'DATA_REQUEST' && message.operation === 'set').message
    runtime.receive(protocolMessage('DATA_RESULT', { requestId: writeRequest.requestId, error: { code: 26011, msg: '写入失败' } }))

    await vi.waitFor(() => expect(currentValue).toBe(10))
    expect(setValue).toHaveBeenLastCalledWith(10)
  })
})
