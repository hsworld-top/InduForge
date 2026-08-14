const PREVIEW_DEVTOOLS_SOURCE = 'induforge-designer'
const PREVIEW_DEVTOOLS_TYPE = 'PREVIEW_DEVTOOLS_COMMAND'
const DRAWER_DEFAULT_RATIO = 0.34
const DRAWER_MAX_RATIO = 0.8
const DRAWER_MIN_HEIGHT = 160

let devtoolsDrawer = null
let devtoolsContent = null
let devtoolsOpen = false
let erudaInitialized = false

function resolveParentOrigin() {
  if (!document.referrer) return null
  try {
    return new URL(document.referrer).origin
  } catch {
    return null
  }
}

function isTrustedMessageSource(source) {
  if (source === window.parent) return true
  try {
    for (let index = 0; index < window.top.frames.length; index += 1) {
      if (source === window.top.frames[index]) return true
    }
  } catch {
    return false
  }
  return false
}

function clampDrawerHeight(height) {
  const maxHeight = Math.max(1, Math.floor(window.innerHeight * DRAWER_MAX_RATIO))
  const minHeight = Math.min(DRAWER_MIN_HEIGHT, maxHeight)
  return Math.min(Math.max(height, minHeight), maxHeight)
}

function setDrawerHeight(height) {
  if (devtoolsDrawer) devtoolsDrawer.style.height = `${clampDrawerHeight(height)}px`
}

function beginDrawerResize(event) {
  if (!devtoolsDrawer) return
  event.preventDefault()
  const startY = event.clientY
  const startHeight = devtoolsDrawer.getBoundingClientRect().height
  const move = (moveEvent) => setDrawerHeight(startHeight + startY - moveEvent.clientY)
  const end = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', end)
    window.removeEventListener('pointercancel', end)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', end)
  window.addEventListener('pointercancel', end)
}

function createDevtoolsDrawer() {
  if (devtoolsDrawer && devtoolsContent) return
  const drawer = document.createElement('section')
  drawer.id = 'induforge-preview-devtools'
  drawer.setAttribute('aria-label', '页面开发控制台')
  Object.assign(drawer.style, {
    position: 'fixed',
    zIndex: '2147483647',
    right: '0',
    bottom: '0',
    left: '0',
    display: 'none',
    minWidth: '0',
    overflow: 'hidden',
    borderTop: '1px solid #3b414b',
    background: '#202124',
    boxShadow: '0 -8px 24px rgba(15, 23, 42, 0.24)',
  })

  const resizeHandle = document.createElement('div')
  resizeHandle.setAttribute('role', 'separator')
  resizeHandle.setAttribute('aria-label', '调整控制台高度')
  resizeHandle.setAttribute('aria-orientation', 'horizontal')
  resizeHandle.tabIndex = 0
  Object.assign(resizeHandle.style, {
    position: 'absolute',
    zIndex: '2',
    top: '0',
    right: '0',
    left: '0',
    height: '7px',
    cursor: 'row-resize',
    touchAction: 'none',
    background: '#2b2d31',
  })

  const grip = document.createElement('span')
  Object.assign(grip.style, {
    position: 'absolute',
    top: '2px',
    left: '50%',
    width: '38px',
    height: '2px',
    borderRadius: '2px',
    background: '#727983',
    transform: 'translateX(-50%)',
  })
  resizeHandle.appendChild(grip)
  resizeHandle.addEventListener('pointerdown', beginDrawerResize)
  resizeHandle.addEventListener('keydown', (event) => {
    if (!devtoolsDrawer || !['ArrowUp', 'ArrowDown'].includes(event.key)) return
    event.preventDefault()
    setDrawerHeight(
      devtoolsDrawer.getBoundingClientRect().height + (event.key === 'ArrowUp' ? 24 : -24),
    )
  })

  const content = document.createElement('div')
  Object.assign(content.style, {
    position: 'absolute',
    top: '7px',
    right: '0',
    bottom: '0',
    left: '0',
    minHeight: '0',
    overflow: 'hidden',
  })
  drawer.append(resizeHandle, content)
  document.body.appendChild(drawer)
  devtoolsDrawer = drawer
  devtoolsContent = content
  setDrawerHeight(window.innerHeight * DRAWER_DEFAULT_RATIO)
}

function togglePreviewDevtools() {
  createDevtoolsDrawer()
  if (!devtoolsDrawer || !devtoolsContent || !window.eruda) return
  if (devtoolsOpen) {
    devtoolsDrawer.style.display = 'none'
    devtoolsOpen = false
    return
  }
  devtoolsDrawer.style.display = 'block'
  devtoolsOpen = true
  if (!erudaInitialized) {
    window.eruda.init({
      container: devtoolsContent,
      inline: true,
      defaults: { displaySize: 50, transparency: 1 },
    })
    erudaInitialized = true
  }
  window.eruda.show('console')
}

if (window.parent !== window) {
  const parentOrigin = resolveParentOrigin()
  if (parentOrigin) {
    window.addEventListener('message', (event) => {
      if (!isTrustedMessageSource(event.source) || event.origin !== parentOrigin) return
      const message = event.data
      if (
        message?.source === PREVIEW_DEVTOOLS_SOURCE &&
        message.type === PREVIEW_DEVTOOLS_TYPE &&
        message.command === 'toggle'
      ) {
        togglePreviewDevtools()
      }
    })
  }
}
