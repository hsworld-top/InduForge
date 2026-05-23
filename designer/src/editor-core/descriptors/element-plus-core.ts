import type { ComponentDescriptor } from './registry'
import CheckboxRenderer from '@/materials/ElementPlusCore/renderers/CheckboxRenderer.vue'
import InputNumberRenderer from '@/materials/ElementPlusCore/renderers/InputNumberRenderer.vue'
import PaginationRenderer from '@/materials/ElementPlusCore/renderers/PaginationRenderer.vue'
import RadioRenderer from '@/materials/ElementPlusCore/renderers/RadioRenderer.vue'
import SelectRenderer from '@/materials/ElementPlusCore/renderers/SelectRenderer.vue'
import SwitchRenderer from '@/materials/ElementPlusCore/renderers/SwitchRenderer.vue'
import TableRenderer from '@/materials/ElementPlusCore/renderers/TableRenderer.vue'

type LooseRecord = Record<string, unknown>

function stripProps(props: LooseRecord, keys: string[]): LooseRecord {
  const next = { ...props }
  keys.forEach((key) => {
    delete next[key]
  })
  return next
}

export const inputDescriptor: ComponentDescriptor = {
  renderTag: 'el-input',
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  propsFilter: (props) => props,
  defaultSize: { width: 220, height: 34 },
}

export const inputNumberDescriptor: ComponentDescriptor = {
  renderTag: 'div',
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  displayContent: () => null,
  customRenderer: InputNumberRenderer,
  defaultSize: { width: 160, height: 34 },
}

export const selectDescriptor: ComponentDescriptor = {
  renderTag: 'div',
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  displayContent: () => null,
  customRenderer: SelectRenderer,
  defaultSize: { width: 220, height: 34 },
}

export const radioDescriptor: ComponentDescriptor = {
  renderTag: 'div',
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  displayContent: () => null,
  customRenderer: RadioRenderer,
  defaultSize: { width: 260, height: 34 },
}

export const checkboxDescriptor: ComponentDescriptor = {
  renderTag: 'div',
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  displayContent: () => null,
  customRenderer: CheckboxRenderer,
  defaultSize: { width: 280, height: 34 },
}

export const switchDescriptor: ComponentDescriptor = {
  renderTag: 'div',
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  displayContent: () => null,
  customRenderer: SwitchRenderer,
  defaultSize: { width: 80, height: 32 },
}

export const tableDescriptor: ComponentDescriptor = {
  renderTag: 'div',
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  displayContent: () => null,
  customRenderer: TableRenderer,
  defaultSize: { width: 420, height: 180 },
}

export const paginationDescriptor: ComponentDescriptor = {
  renderTag: 'div',
  isContainer: false,
  defaultStyle: {},
  acceptChildren: false,
  isMovable: true,
  childResizable: true,
  propsFilter: (props) => stripProps(props, ['layout']),
  displayContent: () => null,
  customRenderer: PaginationRenderer,
  defaultSize: { width: 420, height: 36 },
}
