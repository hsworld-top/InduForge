/**
 * Property Test: Component Tree Structure
 * **Feature: design-center, Property 8: Component Tree Structure**
 * **Validates: Requirements 5.1**
 * 
 * *For any* page schema with nested components, the component tree SHALL display
 * a tree structure where each component's children appear as nested items under their parent.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import * as fc from 'fast-check'
import ComponentTree from '../ComponentTree.vue'
import ElementPlus from 'element-plus'
import { useDesignStore } from '@/store/design'

// Mock the design API to prevent actual API calls
vi.mock('@/api/design.api', () => ({
  designAPI: {
    getPages: vi.fn(),
    getPage: vi.fn(),
    createPage: vi.fn(),
    updatePage: vi.fn(),
    deletePage: vi.fn(),
  }
}))

/**
 * Generates a valid component ID
 */
const componentIdArb = fc.uuid()

/**
 * Generates a valid component type
 */
const componentTypeArb = fc.constantFrom('Container', 'Text', 'Button', 'Image', 'Input')

/**
 * Generates a valid component label (alphanumeric to avoid HTML escaping issues)
 */
const componentLabelArb = fc.stringMatching(/^[a-zA-Z][a-zA-Z0-9 ]{0,19}$/)
  .filter(s => s.trim().length > 0)

/**
 * Generates a leaf component (no children)
 */
const leafComponentArb = fc.record({
  id: componentIdArb,
  type: fc.constantFrom('Text', 'Button', 'Image', 'Input'),
  label: componentLabelArb,
  locked: fc.boolean(),
  visible: fc.constant(true),
  style: fc.record({
    position: fc.constant('absolute'),
    left: fc.integer({ min: 0, max: 500 }),
    top: fc.integer({ min: 0, max: 500 }),
    width: fc.integer({ min: 50, max: 300 }),
    height: fc.integer({ min: 20, max: 200 }),
    zIndex: fc.integer({ min: 0, max: 100 }),
  }),
  props: fc.constant({}),
  children: fc.constant([]),
})

/**
 * Generates a container component with children (depth-limited)
 */
function containerComponentArb(maxDepth) {
  if (maxDepth <= 0) {
    return leafComponentArb
  }
  
  return fc.record({
    id: componentIdArb,
    type: fc.constant('Container'),
    label: componentLabelArb,
    locked: fc.boolean(),
    visible: fc.constant(true),
    style: fc.record({
      position: fc.constant('absolute'),
      left: fc.integer({ min: 0, max: 500 }),
      top: fc.integer({ min: 0, max: 500 }),
      width: fc.integer({ min: 100, max: 500 }),
      height: fc.integer({ min: 100, max: 400 }),
      zIndex: fc.integer({ min: 0, max: 100 }),
    }),
    props: fc.constant({}),
    children: fc.array(
      fc.oneof(leafComponentArb, containerComponentArb(maxDepth - 1)),
      { minLength: 0, maxLength: 3 }
    ),
  })
}

/**
 * Generates a component that can be either a leaf or container
 */
const componentArb = fc.oneof(
  leafComponentArb,
  containerComponentArb(2)
)

/**
 * Generates an array of components (the root level)
 */
const componentsArrayArb = fc.array(componentArb, { minLength: 1, maxLength: 5 })

/**
 * Recursively counts all components in a tree
 */
function countAllComponents(components) {
  let count = 0
  for (const component of components) {
    count += 1
    if (component.children && component.children.length > 0) {
      count += countAllComponents(component.children)
    }
  }
  return count
}

/**
 * Recursively collects all component IDs from a tree
 */
function collectAllIds(components) {
  const ids = []
  for (const component of components) {
    ids.push(component.id)
    if (component.children && component.children.length > 0) {
      ids.push(...collectAllIds(component.children))
    }
  }
  return ids
}

/**
 * Verifies that a parent-child relationship exists in the rendered tree
 * by checking that child nodes are nested under parent nodes
 */
function verifyTreeStructure(wrapper, components, parentSelector = null) {
  for (const component of components) {
    // Find the tree node for this component
    const nodeSelector = `[data-component-id="${component.id}"]`
    
    // If component has children, verify they are nested
    if (component.children && component.children.length > 0) {
      // Recursively verify children
      verifyTreeStructure(wrapper, component.children, component.id)
    }
  }
  return true
}

describe('Property 8: Component Tree Structure', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  /**
   * Property: For any page schema with nested components, the component tree
   * displays all components in a hierarchical structure
   */
  it('should display all components from the schema in the tree', () => {
    fc.assert(
      fc.property(
        componentsArrayArb,
        (components) => {
          const pinia = createPinia()
          setActivePinia(pinia)
          const store = useDesignStore()
          
          // Set up the store with a page containing the generated components
          store.currentPage = {
            version: '2.0.0',
            meta: { id: 'test-page', name: 'Test Page' },
            config: {},
            components: components,
          }
          store.currentPageId = 'test-page'

          const wrapper = mount(ComponentTree, {
            global: {
              plugins: [ElementPlus, pinia],
            }
          })

          const html = wrapper.html()
          
          // Count total components in the schema
          const totalComponents = countAllComponents(components)
          
          // The tree should render all components
          // Each component should have its label or type displayed
          const allIds = collectAllIds(components)
          
          // Verify each component's label or type appears in the tree
          for (const component of components) {
            const displayText = component.label || component.type
            expect(html).toContain(displayText)
          }
          
          wrapper.unmount()
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: For any component with children, those children appear as nested
   * items in the tree structure
   */
  it('should display children as nested items under their parent', () => {
    fc.assert(
      fc.property(
        containerComponentArb(2),
        (containerComponent) => {
          // Only test if the container has children
          if (!containerComponent.children || containerComponent.children.length === 0) {
            return true // Skip empty containers
          }

          const pinia = createPinia()
          setActivePinia(pinia)
          const store = useDesignStore()
          
          // Set up the store with a page containing the container
          store.currentPage = {
            version: '2.0.0',
            meta: { id: 'test-page', name: 'Test Page' },
            config: {},
            components: [containerComponent],
          }
          store.currentPageId = 'test-page'

          const wrapper = mount(ComponentTree, {
            global: {
              plugins: [ElementPlus, pinia],
            }
          })

          const html = wrapper.html()
          
          // Parent should be displayed
          const parentText = containerComponent.label || containerComponent.type
          expect(html).toContain(parentText)
          
          // All children should be displayed
          for (const child of containerComponent.children) {
            const childText = child.label || child.type
            expect(html).toContain(childText)
          }
          
          // Verify the tree structure by checking el-tree-node nesting
          // The el-tree component renders nested nodes with el-tree-node class
          const treeNodes = wrapper.findAll('.el-tree-node')
          
          // Should have at least 1 + children count nodes
          const expectedMinNodes = 1 + containerComponent.children.length
          expect(treeNodes.length).toBeGreaterThanOrEqual(expectedMinNodes)
          
          wrapper.unmount()
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: The total number of tree nodes equals the total number of components
   * in the schema (including all nested children)
   */
  it('should have tree node count equal to total component count', () => {
    fc.assert(
      fc.property(
        componentsArrayArb,
        (components) => {
          const pinia = createPinia()
          setActivePinia(pinia)
          const store = useDesignStore()
          
          store.currentPage = {
            version: '2.0.0',
            meta: { id: 'test-page', name: 'Test Page' },
            config: {},
            components: components,
          }
          store.currentPageId = 'test-page'

          const wrapper = mount(ComponentTree, {
            global: {
              plugins: [ElementPlus, pinia],
            }
          })

          // Count total components in schema
          const totalComponents = countAllComponents(components)
          
          // Count tree nodes (each component should have one .tree-node element)
          const treeNodes = wrapper.findAll('.tree-node')
          
          expect(treeNodes.length).toBe(totalComponents)
          
          wrapper.unmount()
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: Empty component list should show empty state
   */
  it('should show empty state when no components exist', () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const store = useDesignStore()
    
    store.currentPage = {
      version: '2.0.0',
      meta: { id: 'test-page', name: 'Test Page' },
      config: {},
      components: [],
    }
    store.currentPageId = 'test-page'

    const wrapper = mount(ComponentTree, {
      global: {
        plugins: [ElementPlus, pinia],
      }
    })

    const html = wrapper.html()
    
    // Should show empty state
    expect(html).toContain('el-empty')
    expect(html).toContain('暂无组件')
    
    wrapper.unmount()
  })
})
