/**
 * Component Registry
 * 
 * Manages registration and retrieval of design components.
 * Each component definition includes type, name, category, icon,
 * defaultProps, defaultStyle, and propsSchema.
 */

// Internal registry storage
const componentRegistry = new Map()

/**
 * Register a component definition
 * @param {Object} definition - Component definition
 * @param {string} definition.type - Unique component type identifier
 * @param {string} definition.name - Display name
 * @param {string} definition.category - Category for grouping
 * @param {string} definition.icon - Icon identifier
 * @param {Object} definition.defaultProps - Default property values
 * @param {Object} definition.defaultStyle - Default style values
 * @param {Object} definition.propsSchema - Schema for property editing
 */
export function registerComponent(definition) {
  if (!definition || !definition.type) {
    throw new Error('Component definition must have a type')
  }
  
  if (componentRegistry.has(definition.type)) {
    console.warn(`Component type "${definition.type}" is already registered. Overwriting.`)
  }
  
  componentRegistry.set(definition.type, {
    type: definition.type,
    name: definition.name || definition.type,
    category: definition.category || 'Other',
    icon: definition.icon || 'component',
    defaultProps: definition.defaultProps || {},
    defaultStyle: definition.defaultStyle || {},
    propsSchema: definition.propsSchema || {},
    render: definition.render || null
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
  const categories = {}
  
  for (const component of componentRegistry.values()) {
    const category = component.category
    if (!categories[category]) {
      categories[category] = []
    }
    categories[category].push(component)
  }
  
  return categories
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
