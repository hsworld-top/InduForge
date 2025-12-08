/**
 * Component Registration - 组件注册
 * 
 * 统一注册所有可用组件
 */

import { registerComponent } from '../index'
import basicComponents from './basic'

/**
 * 注册所有基础组件
 */
export function registerBasicComponents() {
  basicComponents.forEach(component => {
    registerComponent(component)
  })
  
  console.log(`✅ Registered ${basicComponents.length} basic components`)
}

/**
 * 注册所有组件
 */
export function registerAllComponents() {
  registerBasicComponents()
  // TODO: 注册其他类型的组件
  // registerUIComponents()
  // registerChartComponents()
  // registerIndustrialComponents()
}

export default {
  registerBasicComponents,
  registerAllComponents
}
