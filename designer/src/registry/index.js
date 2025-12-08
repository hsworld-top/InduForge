/**
 * Component Registry - 组件注册中心
 * 
 * 管理所有可用组件的注册、检索和实例化
 * 支持组件分类、搜索、预览等功能
 */

// Internal registry storage
const componentRegistry = new Map()

// 组件分类索引
const categoryIndex = new Map()

// 组件标签索引
const tagIndex = new Map()

/**
 * Register a component definition
 * @param {Object} definition - Component definition
 * @param {string} definition.type - Unique component type identifier
 * @param {string} definition.name - Display name
 * @param {string} definition.category - Category for grouping
 * @param {string} definition.icon - Icon identifier
 * @param {string} definition.thumbnail - Thumbnail image URL
 * @param {Array<string>} definition.tags - Search tags
 * @param {Object} definition.defaultProps - Default property values
 * @param {Object} definition.defaultStyle - Default style values
 * @param {Object} definition.propsSchema - Schema for property editing
 * @param {Object} definition.eventsSchema - Schema for event configuration
 * @param {Function} definition.render - Custom render function
 * @param {boolean} definition.container - Whether component can contain children
 */
export function registerComponent(definition) {
  if (!definition || !definition.type) {
    throw new Error('Component definition must have a type')
  }
  
  if (componentRegistry.has(definition.type)) {
    console.warn(`Component type "${definition.type}" is already registered. Overwriting.`)
  }
  
  const component = {
    type: definition.type,
    name: definition.name || definition.type,
    category: definition.category || 'Other',
    icon: definition.icon || 'component',
    thumbnail: definition.thumbnail || null,
    tags: definition.tags || [],
    description: definition.description || '',
    defaultProps: definition.defaultProps || {},
    defaultStyle: definition.defaultStyle || {
      position: 'absolute',
      left: 0,
      top: 0,
      width: 100,
      height: 100
    },
    propsSchema: definition.propsSchema || {},
    eventsSchema: definition.eventsSchema || {},
    render: definition.render || null,
    container: definition.container || false,
    version: definition.version || '1.0.0'
  }
  
  componentRegistry.set(definition.type, component)
  
  // 更新分类索引
  updateCategoryIndex(component)
  
  // 更新标签索引
  updateTagIndex(component)
}

/**
 * 更新分类索引
 */
function updateCategoryIndex(component) {
  const category = component.category
  if (!categoryIndex.has(category)) {
    categoryIndex.set(category, [])
  }
  const components = categoryIndex.get(category)
  const existingIndex = components.findIndex(c => c.type === component.type)
  if (existingIndex >= 0) {
    components[existingIndex] = component
  } else {
    components.push(component)
  }
}

/**
 * 更新标签索引
 */
function updateTagIndex(component) {
  component.tags.forEach(tag => {
    if (!tagIndex.has(tag)) {
      tagIndex.set(tag, [])
    }
    const components = tagIndex.get(tag)
    if (!components.find(c => c.type === component.type)) {
      components.push(component)
    }
  })
}

/**
 * Get a component definition by type
 * @param {string} type - Component type identifier
 * @returns {Object|null} Component definition or null if not found
 */
export function getComponent(type) {
  return componentRegistry.get(type) || null
}

/**
 * Get all registered components
 * @returns {Object[]} Array of all component definitions
 */
export function getAllComponents() {
  return Array.from(componentRegistry.values())
}

/**
 * Get components grouped by category
 * @returns {Object} Object with category names as keys and arrays of components as values
 */
export function getComponentsByCategory() {
  return Object.fromEntries(categoryIndex.entries())
}

/**
 * Search components by keyword
 * @param {string} keyword - Search keyword
 * @returns {Object[]} Matching components
 */
export function searchComponents(keyword) {
  if (!keyword) {
    return getAllComponents()
  }
  
  const lowerKeyword = keyword.toLowerCase()
  return getAllComponents().filter(component => {
    return (
      component.name.toLowerCase().includes(lowerKeyword) ||
      component.type.toLowerCase().includes(lowerKeyword) ||
      component.description.toLowerCase().includes(lowerKeyword) ||
      component.tags.some(tag => tag.toLowerCase().includes(lowerKeyword))
    )
  })
}

/**
 * Get components by tag
 * @param {string} tag - Tag name
 * @returns {Object[]} Components with the specified tag
 */
export function getComponentsByTag(tag) {
  return tagIndex.get(tag) || []
}

/**
 * Get all categories
 * @returns {string[]} Array of category names
 */
export function getAllCategories() {
  return Array.from(categoryIndex.keys())
}

/**
 * Create a new component instance with default values
 * @param {string} type - Component type identifier
 * @param {Object} overrides - Optional property/style overrides
 * @returns {Object|null} New component instance or null if type not found
 */
export function createComponentInstance(type, overrides = {}) {
  const definition = getComponent(type)
  if (!definition) {
    return null
  }
  
  return {
    id: generateId(),
    type: definition.type,
    label: definition.name,
    locked: false,
    visible: true,
    style: {
      ...definition.defaultStyle,
      ...(overrides.style || {})
    },
    props: {
      ...definition.defaultProps,
      ...(overrides.props || {})
    },
    bindings: {},
    events: {},
    animations: [],
    children: []
  }
}

/**
 * Generate a unique ID for components
 * @returns {string} UUID v4
 */
function generateId() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = Math.random() * 16 | 0
    const v = c === 'x' ? r : (r & 0x3 | 0x8)
    return v.toString(16)
  })
}

/**
 * Clear all registered components (useful for testing)
 */
export function clearRegistry() {
  componentRegistry.clear()
}

/**
 * Check if a component type is registered
 * @param {string} type - Component type identifier
 * @returns {boolean} True if registered
 */
export function hasComponent(type) {
  return componentRegistry.has(type)
}
