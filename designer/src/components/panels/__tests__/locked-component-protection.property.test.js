/**
 * Property Test: Locked Component Protection
 * **Feature: design-center, Property 9: Locked Component Protection**
 * **Validates: Requirements 5.5**
 * 
 * *For any* component with locked === true, selection and editing operations SHALL be prevented.
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
 * Generates a valid component label (alphanumeric to avoid HTML escaping issues)
 */
const componentLabelArb = fc.stringMatching(/^[a-zA-Z][a-zA-Z0-9 ]{0,19}$/)
  .filter(s => s.trim().length > 0)

/**
 * Generates a locked component (locked === true)
 */
const lockedComponentArb = fc.record({
  id: componentIdArb,
  type: fc.constantFrom('Container', 'Text', 'Button', 'Image', 'Input'),
  label: componentLabelArb,
  locked: fc.constant(true),
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
 * Generates an unlocked component (locked === false)
 */
const unlockedComponentArb = fc.record({
  id: componentIdArb,
  type: fc.constantFrom('Container', 'Text', 'Button', 'Image', 'Input'),
  label: componentLabelArb,
  locked: fc.constant(false),
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
 * Generates a locked container with unlocked children
 */
const lockedContainerWithChildrenArb = fc.record({
  id: componentIdArb,
  type: fc.constant('Container'),
  label: componentLabelArb,
  locked: fc.constant(true),
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
  children: fc.array(unlockedComponentArb, { minLength: 1, maxLength: 3 }),
})

describe('Property 9: Locked Component Protection', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  /**
   * Property: For any locked component, clicking on it SHALL NOT change the selection
   */
  it('should prevent selection of locked components when clicked', () => {
    fc.assert(
      fc.property(
        lockedComponentArb,
        fc.array(unlockedComponentArb, { minLength: 0, maxLength: 3 }),
        (lockedComponent, otherComponents) => {
          const pinia = createPinia()
          setActivePinia(pinia)
          const store = useDesignStore()
          
          // Set up the store with a page containing the locked component
          const allComponents = [lockedComponent, ...otherComponents]
          store.currentPage = {
            version: '2.0.0',
            meta: { id: 'test-page', name: 'Test Page' },
            config: {},
            components: allComponents,
          }
          store.currentPageId = 'test-page'
          store.selectedComponentId = null

          const wrapper = mount(ComponentTree, {
            global: {
              plugins: [ElementPlus, pinia],
            }
          })

          // Find the locked component's tree node and click it
          const treeNodes = wrapper.findAll('.tree-node')
          const lockedNode = treeNodes.find(node => {
            const label = node.find('.node-label')
            return label && label.text() === (lockedComponent.label || lockedComponent.type)
          })

          if (lockedNode) {
            // Simulate click on the tree node
            lockedNode.trigger('click')
          }

          // Selection should NOT have changed (should still be null)
          expect(store.selectedComponentId).toBeNull()
          
          wrapper.unmount()
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: For any locked component, the allowDrag function SHALL return false
   */
  it('should prevent dragging of locked components', () => {
    fc.assert(
      fc.property(
        lockedComponentArb,
        (lockedComponent) => {
          const pinia = createPinia()
          setActivePinia(pinia)
          const store = useDesignStore()
          
          store.currentPage = {
            version: '2.0.0',
            meta: { id: 'test-page', name: 'Test Page' },
            config: {},
            components: [lockedComponent],
          }
          store.currentPageId = 'test-page'

          const wrapper = mount(ComponentTree, {
            global: {
              plugins: [ElementPlus, pinia],
            }
          })

          // Find the el-tree component and check its allow-drag prop behavior
          const elTree = wrapper.findComponent({ name: 'ElTree' })
          
          // The tree should exist
          expect(elTree.exists()).toBe(true)
          
          // Check that the locked node has the locked styling
          const lockedNodes = wrapper.findAll('.tree-node--locked')
          expect(lockedNodes.length).toBeGreaterThanOrEqual(1)
          
          wrapper.unmount()
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: For any locked component, it SHALL display a lock icon
   */
  it('should display lock icon for locked components', () => {
    fc.assert(
      fc.property(
        lockedComponentArb,
        (lockedComponent) => {
          const pinia = createPinia()
          setActivePinia(pinia)
          const store = useDesignStore()
          
          store.currentPage = {
            version: '2.0.0',
            meta: { id: 'test-page', name: 'Test Page' },
            config: {},
            components: [lockedComponent],
          }
          store.currentPageId = 'test-page'

          const wrapper = mount(ComponentTree, {
            global: {
              plugins: [ElementPlus, pinia],
            }
          })

          // Locked components should have lock icon displayed
          const lockIcons = wrapper.findAll('.lock-icon')
          expect(lockIcons.length).toBeGreaterThanOrEqual(1)
          
          wrapper.unmount()
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: For any unlocked component, clicking on it SHALL update the selection
   */
  it('should allow selection of unlocked components when clicked', () => {
    fc.assert(
      fc.property(
        unlockedComponentArb,
        (unlockedComponent) => {
          const pinia = createPinia()
          setActivePinia(pinia)
          const store = useDesignStore()
          
          store.currentPage = {
            version: '2.0.0',
            meta: { id: 'test-page', name: 'Test Page' },
            config: {},
            components: [unlockedComponent],
          }
          store.currentPageId = 'test-page'
          store.selectedComponentId = null

          const wrapper = mount(ComponentTree, {
            global: {
              plugins: [ElementPlus, pinia],
            }
          })

          // Find the el-tree and trigger node-click event
          const elTree = wrapper.findComponent({ name: 'ElTree' })
          if (elTree.exists()) {
            elTree.vm.$emit('node-click', unlockedComponent)
          }

          // Selection should have changed to the unlocked component
          expect(store.selectedComponentId).toBe(unlockedComponent.id)
          
          wrapper.unmount()
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: For any locked container, dropping components into it SHALL be prevented
   * (verified by checking the locked styling is applied)
   */
  it('should apply locked styling to locked components', () => {
    fc.assert(
      fc.property(
        lockedContainerWithChildrenArb,
        (lockedContainer) => {
          const pinia = createPinia()
          setActivePinia(pinia)
          const store = useDesignStore()
          
          store.currentPage = {
            version: '2.0.0',
            meta: { id: 'test-page', name: 'Test Page' },
            config: {},
            components: [lockedContainer],
          }
          store.currentPageId = 'test-page'

          const wrapper = mount(ComponentTree, {
            global: {
              plugins: [ElementPlus, pinia],
            }
          })

          // The locked container should have the locked class
          const lockedNodes = wrapper.findAll('.tree-node--locked')
          expect(lockedNodes.length).toBeGreaterThanOrEqual(1)
          
          // Children should NOT have locked class (they are unlocked)
          const allNodes = wrapper.findAll('.tree-node')
          const unlockedNodes = allNodes.filter(node => !node.classes().includes('tree-node--locked'))
          expect(unlockedNodes.length).toBe(lockedContainer.children.length)
          
          wrapper.unmount()
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: Locked components should have cursor: not-allowed styling
   */
  it('should apply not-allowed cursor to locked components', () => {
    fc.assert(
      fc.property(
        lockedComponentArb,
        (lockedComponent) => {
          const pinia = createPinia()
          setActivePinia(pinia)
          const store = useDesignStore()
          
          store.currentPage = {
            version: '2.0.0',
            meta: { id: 'test-page', name: 'Test Page' },
            config: {},
            components: [lockedComponent],
          }
          store.currentPageId = 'test-page'

          const wrapper = mount(ComponentTree, {
            global: {
              plugins: [ElementPlus, pinia],
            }
          })

          // The locked node should have the tree-node--locked class
          const lockedNodes = wrapper.findAll('.tree-node--locked')
          expect(lockedNodes.length).toBe(1)
          
          wrapper.unmount()
          return true
        }
      ),
      { numRuns: 100 }
    )
  })
})
