import type { ComponentDescriptor } from './registry'
import LanguageSwitcherRenderer from '@/materials/LanguageSwitcher/LanguageSwitcherRenderer.vue'

export const descriptor: ComponentDescriptor = {
  renderTag: 'div',
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  displayContent: () => null,
  customRenderer: LanguageSwitcherRenderer,
  defaultSize: { width: 160, height: 34 },
}

export default descriptor
