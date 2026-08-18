import { ElTooltip } from 'element-plus'

type PopperOptions = {
  modifiers?: unknown[]
  [key: string]: unknown
}

type TooltipSetup = (props: Record<PropertyKey, unknown>, context: unknown) => unknown

type TooltipWithMutableStrategy = {
  props?: {
    strategy?: {
      default?: 'absolute' | 'fixed'
    }
  }
  setup?: TooltipSetup
  __induForgePopperConfigured?: boolean
}

const fixedViewportModifier = {
  name: 'computeStyles',
  options: { adaptive: false },
}

/**
 * Wujie 子应用在 macOS 宿主窗口中可能存在视口原点偏移。Element Plus 的浮层统一改用
 * 视口坐标，并关闭基于子应用视口高度的自适应坐标换算，避免 Teleport 到宿主 body 后
 * 重复叠加顶部偏移。调用方传入的同名 modifier 排在最后，仍可覆盖这里的默认设置。
 */
export const configureElementPlusPopper = () => {
  const tooltip = ElTooltip as unknown as TooltipWithMutableStrategy
  if (tooltip.__induForgePopperConfigured) return

  const strategy = tooltip.props?.strategy
  if (strategy) {
    strategy.default = 'fixed'
  }

  const originalSetup = tooltip.setup
  if (originalSetup) {
    tooltip.setup = (props, context) =>
      originalSetup(
        new Proxy(props, {
          get(target, property, receiver) {
            const value = Reflect.get(target, property, receiver)
            if (property !== 'popperOptions') return value

            const options = (value ?? {}) as PopperOptions
            return {
              ...options,
              modifiers: [fixedViewportModifier, ...(options.modifiers ?? [])],
            }
          },
        }),
        context,
      )
  }

  tooltip.__induForgePopperConfigured = true
}
