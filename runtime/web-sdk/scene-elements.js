import Ajv2020 from 'ajv/dist/2020.js'

const CHANNEL = 'induforge-scene-runtime'
const PREVIEW_CHANNEL = 'induforge-preview-runtime'
const PROTOCOL_VERSION = 1
const DEFAULT_COMMAND_TIMEOUT = 5000
const RESOLVE_TIMEOUT = 10000
const VIEWER_READY_TIMEOUT = 15000

export const SceneErrorCode = Object.freeze({
  TOKEN_EXPIRED: 10003,
  NOT_COMMITTED: 26001,
  CONTRACT_INVALID: 26002,
  CONTRACT_VIOLATION: 26003,
  COMMAND_NOT_DECLARED: 26004,
  COMMAND_NOT_REGISTERED: 26005,
  COMMAND_TIMEOUT: 26006,
  DATAPOINT_NOT_DECLARED: 26007,
  DATAPOINT_READ_FAILED: 26008,
  DATAPOINT_SUBSCRIBE_FAILED: 26009,
  DATAPOINT_WRITE_FORBIDDEN: 26010,
  DATAPOINT_WRITE_FAILED: 26011,
})

export class SceneRuntimeError extends Error {
  constructor(code, message, requestId = '') {
    super(message)
    this.name = 'SceneRuntimeError'
    this.code = code
    this.requestId = requestId
  }
}

function requestId() {
  return globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function runtimeConfiguration() {
  return globalThis.window?.__INDUFORGE_RUNTIME__ ?? {}
}

function resolveWithPreviewHost(kind, sceneId) {
  const target = window.parent
  if (!target || target === window) {
    return Promise.reject(new SceneRuntimeError(26003, '当前页面不在 InduForge 受控预览宿主中'))
  }
  const id = requestId()
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      window.removeEventListener('message', handle)
      reject(new SceneRuntimeError(26003, '创建场景 Viewer 会话超时', id))
    }, RESOLVE_TIMEOUT)
    function handle(event) {
      const data = event.data
      if (
        event.source !== target ||
        !data ||
        data.channel !== PREVIEW_CHANNEL ||
        data.version !== PROTOCOL_VERSION ||
        data.requestId !== id ||
        data.type !== 'SCENE_VIEWER_RESULT'
      ) return
      window.clearTimeout(timer)
      window.removeEventListener('message', handle)
      if (data.error) {
        reject(new SceneRuntimeError(Number(data.error.code) || 26003, data.error.msg || '创建场景 Viewer 失败', id))
      } else {
        resolve(data.data)
      }
    }
    window.addEventListener('message', handle)
    target.postMessage({ channel: PREVIEW_CHANNEL, version: PROTOCOL_VERSION, type: 'CREATE_SCENE_VIEWER', requestId: id, sceneId, kind }, '*')
  })
}

async function resolveScene(kind, sceneId) {
  const resolver = runtimeConfiguration().sceneResolver
  if (resolver?.resolve) return resolver.resolve({ kind, sceneId })
  return resolveWithPreviewHost(kind, sceneId)
}

function compileContract(contract = {}) {
  const ajv = new Ajv2020({ allErrors: true, strict: true })
  const parameters = Array.isArray(contract.parameters) ? contract.parameters : []
  const properties = {}
  const required = []
  for (const parameter of parameters) {
    properties[parameter.name] = parameter.schema
    if (parameter.required) required.push(parameter.name)
  }
  const parameterValidator = ajv.compile({ type: 'object', properties, required, additionalProperties: false })
  const events = new Map()
  for (const event of Array.isArray(contract.events) ? contract.events : []) {
    events.set(event.name, ajv.compile(event.schema))
  }
  const commands = new Map()
  for (const command of Array.isArray(contract.commands) ? contract.commands : []) {
    commands.set(command.name, {
      input: ajv.compile(command.inputSchema),
      output: ajv.compile(command.outputSchema),
    })
  }
  return { parameterValidator, events, commands }
}

function validationMessage(validator) {
  return (validator.errors || []).map((error) => `${error.instancePath || '/'} ${error.message}`).join('；')
}

const HTMLElementBase = globalThis.HTMLElement ?? class {}

export class InduForgeSceneElement extends HTMLElementBase {
  static observedAttributes = ['scene-id', 'command-timeout']

  constructor(kind) {
    super()
    this.kind = kind
    this._params = {}
    this._session = null
    this._previousSession = null
    this._validators = null
    this._frame = null
    this._pendingCommands = new Map()
    this._loadSequence = 0
    this._ready = false
    this._readyTimer = null
    this.attachShadow({ mode: 'open' })
    this.shadowRoot.innerHTML = `
      <style>
        :host{display:block;position:relative;width:100%;height:100%;min-height:240px;background:#f4f6f5;color:#27322f;font:13px system-ui,sans-serif}
        iframe{position:absolute;inset:0;width:100%;height:100%;border:0;background:#fff}
        .state{position:absolute;z-index:2;inset:0;display:grid;place-content:center;gap:8px;padding:24px;background:#f4f6f5;text-align:center}
        .state[hidden]{display:none}.error{color:#a32626}.spinner{width:22px;height:22px;margin:auto;border:2px solid #c7ceca;border-top-color:#31403b;border-radius:50%;animation:spin .8s linear infinite}
        @keyframes spin{to{transform:rotate(360deg)}}
      </style>
      <div class="state loading"><span class="spinner"></span><span>正在加载场景</span></div>
      <div class="state empty" hidden>请选择场景</div>
      <div class="state error" hidden></div>`
    this._onMessage = this._onMessage.bind(this)
  }

  connectedCallback() {
    window.addEventListener('message', this._onMessage)
    this.reload()
  }

  disconnectedCallback() {
    window.removeEventListener('message', this._onMessage)
    this._loadSequence += 1
    this._clearReadyTimer()
    this._ready = false
    this._rejectPending(new SceneRuntimeError(26003, '场景组件已销毁'))
    this._releaseSession(this._session)
    this._releaseSession(this._previousSession)
    this._session = null
    this._previousSession = null
    this._frame?.remove()
    this._frame = null
  }

  attributeChangedCallback(name, oldValue, newValue) {
    if (oldValue === newValue || !this.isConnected) return
    if (name === 'scene-id') this.reload()
  }

  get sceneId() { return this.getAttribute('scene-id')?.trim() || '' }
  set sceneId(value) { value ? this.setAttribute('scene-id', value) : this.removeAttribute('scene-id') }
  get params() { return this._params }
  set params(value) { this.setParams(value).catch(() => {}) }
  get commandTimeout() {
    const value = Number(this.getAttribute('command-timeout') || DEFAULT_COMMAND_TIMEOUT)
    return Math.min(30000, Math.max(1000, Number.isFinite(value) ? value : DEFAULT_COMMAND_TIMEOUT))
  }
  set commandTimeout(value) { this.setAttribute('command-timeout', String(value)) }

  async setParams(value) {
    const next = value && typeof value === 'object' && !Array.isArray(value) ? structuredClone(value) : {}
    if (this._validators && !this._validators.parameterValidator(next)) {
      const error = new SceneRuntimeError(26003, `场景参数不符合契约：${validationMessage(this._validators.parameterValidator)}`)
      this._emitError(error)
      throw error
    }
    this._params = next
    this._postToViewer('PARAMS_CHANGED', { params: next })
  }

  async reload() {
    const sequence = ++this._loadSequence
    const sceneId = this.sceneId
    this._clearReadyTimer()
    this._ready = false
    this._rejectPending(new SceneRuntimeError(26003, '场景正在重新加载'))
    this._setState(sceneId ? 'loading' : 'empty')
    if (!sceneId) {
      this._releaseSession(this._session)
      this._releaseSession(this._previousSession)
      this._frame?.remove()
      this._previousFrame?.remove()
      this._frame = null
      this._previousFrame = null
      this._session = null
      this._previousSession = null
      this._validators = null
      return
    }
    if (!this.isConnected) return
    try {
      const session = await resolveScene(this.kind, sceneId)
      if (sequence !== this._loadSequence || !this.isConnected) return
      this._validators = compileContract(session.contract)
      if (!this._validators.parameterValidator(this._params)) {
        throw new SceneRuntimeError(26003, `场景参数不符合契约：${validationMessage(this._validators.parameterValidator)}`)
      }
      const frame = document.createElement('iframe')
      frame.title = `${this.kind.toUpperCase()} 场景`
      frame.src = session.url
      frame.style.opacity = '0'
      frame.addEventListener('error', () => {
        if (frame !== this._frame) return
        this._clearReadyTimer()
        const error = new SceneRuntimeError(26003, '场景 Viewer 加载失败')
        this._emitError(error)
        this._setState('error', error.message)
      }, { once: true })
      this._previousSession = this._session
      this._session = session
      this.shadowRoot.append(frame)
      const previous = this._frame
      this._frame = frame
      this._previousFrame = previous
      this._readyTimer = window.setTimeout(() => {
        if (frame !== this._frame || this._ready) return
        const error = new SceneRuntimeError(26003, '场景 Viewer 就绪超时')
        this._emitError(error)
        this._setState('error', error.message)
      }, VIEWER_READY_TIMEOUT)
    } catch (error) {
      if (sequence !== this._loadSequence) return
      this._emitError(error)
      this._setState('error', error instanceof Error ? error.message : String(error))
    }
  }

  invoke(name, payload) {
    const command = this._validators?.commands.get(name)
    if (!command) return this._rejectWithError(new SceneRuntimeError(26004, `场景未声明命令 ${name}`))
    if (!command.input(payload)) return this._rejectWithError(new SceneRuntimeError(26003, `命令 ${name} 输入不符合契约：${validationMessage(command.input)}`))
    if (!this._frame || !this._session || !this._ready) return this._rejectWithError(new SceneRuntimeError(26003, '场景尚未就绪'))
    const id = requestId()
    return new Promise((resolve, reject) => {
      const timer = window.setTimeout(() => {
        this._pendingCommands.delete(id)
        const error = new SceneRuntimeError(26006, `命令 ${name} 执行超时`, id)
        this._emitError(error)
        reject(error)
      }, this.commandTimeout)
      this._pendingCommands.set(id, { name, command, resolve, reject, timer })
      this._postToViewer('COMMAND_REQUEST', { requestId: id, name, payload })
    })
  }

  _onMessage(event) {
    const data = event.data
    if (
      event.source === window.parent &&
      data?.channel === PREVIEW_CHANNEL &&
      data.version === PROTOCOL_VERSION &&
      (data.type === 'SCENE_DATA_RESULT' || data.type === 'SCENE_DATA_PUSH')
    ) {
      if (data.viewerSessionId === this._session?.sessionId) {
        this._postToViewer(data.type === 'SCENE_DATA_RESULT' ? 'DATA_RESULT' : 'DATA_PUSH', {
          requestId: data.requestId,
          path: data.path,
          data: data.data,
          error: data.error,
        })
      }
      return
    }
    if (
      event.source === window.parent &&
      data?.channel === PREVIEW_CHANNEL &&
      data.version === PROTOCOL_VERSION &&
      data.type === 'SCENE_REVISION_CHANGED' &&
      data.sceneId === this.sceneId &&
      data.kind === this.kind
    ) {
      void this.reload()
      return
    }
    if (!this._frame || event.source !== this._frame.contentWindow) return
    if (!data || data.channel !== CHANNEL || data.version !== PROTOCOL_VERSION || data.sceneId !== this.sceneId) return
    if (data.type === 'DATA_REQUEST') {
      if (!this._session?.sessionId) return
      window.parent?.postMessage({
        channel: PREVIEW_CHANNEL,
        version: PROTOCOL_VERSION,
        type: 'SCENE_DATA_REQUEST',
        requestId: data.requestId,
        operation: data.operation,
        path: data.path,
        value: data.value,
        sceneId: this.sceneId,
        kind: this.kind,
        revision: this._session.revision,
        viewerSessionId: this._session.sessionId,
      }, '*')
      return
    }
    if (data.type === 'READY') {
      this._clearReadyTimer()
      this._ready = true
      this._postToViewer('INITIALIZE', { params: this._params })
      this._frame.style.opacity = '1'
      this._previousFrame?.remove()
      this._previousFrame = null
      this._releaseSession(this._previousSession)
      this._previousSession = null
      this._setState('ready')
      this.dispatchEvent(new CustomEvent('scene-ready', { detail: { sceneId: this.sceneId, kind: this.kind, revision: this._session?.revision }, bubbles: true, composed: true }))
      return
    }
    if (data.type === 'EVENT') {
      const validator = this._validators?.events.get(data.name)
      if (!validator || !validator(data.payload)) {
        this._emitError(new SceneRuntimeError(26003, `场景事件 ${data.name || ''} 不符合契约${validator ? `：${validationMessage(validator)}` : ''}`))
        return
      }
      this.dispatchEvent(new CustomEvent('scene-event', { detail: { sceneId: this.sceneId, kind: this.kind, name: data.name, payload: data.payload }, bubbles: true, composed: true }))
      return
    }
    if (data.type === 'COMMAND_RESULT') this._resolveCommand(data)
    if (data.type === 'ERROR') {
      const error = new SceneRuntimeError(Number(data.code) || 26003, data.msg || '场景运行错误', data.requestId || '')
      this._emitError(error)
      if (error.code === 10003) void this.reload()
    }
  }

  _resolveCommand(data) {
    const pending = this._pendingCommands.get(data.requestId)
    if (!pending) return
    window.clearTimeout(pending.timer)
    this._pendingCommands.delete(data.requestId)
    if (data.error) {
      const error = new SceneRuntimeError(Number(data.error.code) || 26003, data.error.msg || '命令执行失败', data.requestId)
      this._emitError(error)
      pending.reject(error)
    } else if (!pending.command.output(data.result)) {
      const error = new SceneRuntimeError(26003, `命令 ${pending.name} 返回值不符合契约：${validationMessage(pending.command.output)}`, data.requestId)
      this._emitError(error)
      pending.reject(error)
    } else pending.resolve(data.result)
  }

  _postToViewer(type, data = {}) {
    this._frame?.contentWindow?.postMessage({ channel: CHANNEL, version: PROTOCOL_VERSION, type, sceneId: this.sceneId, ...data }, new URL(this._frame.src).origin)
  }

  _releaseSession(session) {
    if (!session?.sessionId || !window.parent || window.parent === window) return
    window.parent.postMessage({
      channel: PREVIEW_CHANNEL,
      version: PROTOCOL_VERSION,
      type: 'RELEASE_SCENE_VIEWER',
      viewerSessionId: session.sessionId,
    }, '*')
  }

  _setState(state, message = '') {
    for (const name of ['loading', 'empty', 'error']) this.shadowRoot.querySelector(`.${name}`).hidden = name !== state
    if (state === 'error') this.shadowRoot.querySelector('.error').textContent = message
  }

  _emitError(error) {
    const detail = error instanceof SceneRuntimeError ? error : new SceneRuntimeError(26003, error instanceof Error ? error.message : String(error))
    this.dispatchEvent(new CustomEvent('scene-error', { detail: { sceneId: this.sceneId, kind: this.kind, code: detail.code, msg: detail.message, requestId: detail.requestId }, bubbles: true, composed: true }))
  }

  _rejectWithError(error) {
    this._emitError(error)
    return Promise.reject(error)
  }

  _clearReadyTimer() {
    if (this._readyTimer != null) window.clearTimeout(this._readyTimer)
    this._readyTimer = null
  }

  _rejectPending(error) {
    for (const pending of this._pendingCommands.values()) {
      window.clearTimeout(pending.timer)
      pending.reject(error)
    }
    this._pendingCommands.clear()
  }
}

export class InduForgeScene2DElement extends InduForgeSceneElement { constructor() { super('2d') } }
export class InduForgeScene3DElement extends InduForgeSceneElement { constructor() { super('3d') } }

if (globalThis.customElements) {
  if (!customElements.get('induforge-scene-2d')) customElements.define('induforge-scene-2d', InduForgeScene2DElement)
  if (!customElements.get('induforge-scene-3d')) customElements.define('induforge-scene-3d', InduForgeScene3DElement)
}
