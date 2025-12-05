/**
 * Property Test: Component Instantiation Defaults
 * **Feature: design-center, Property 16: Component Instantiation Defaults**
 * **Validates: Requirements 8.2**
 * 
 * *For any* component type in the registry, creating a new instance SHALL produce
 * a component with all defaultProps and defaultStyle values applied.
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import * as fc from 'fast-check'
import {
  registerComponent,
  getComponent,
  createComponentInstance,
  clearRegistry,
  getAllComponents
} from '../index.js'
import { basicComponents, registerBasicComponents } from '../components/index.js'

describe('Property 16: Component Instantiation Defaults', () => {
  beforeEach(() => {
    clearRegistry()
  })

  afterEach(() => {
    clearRegistry()
  })

  /**
   * Property: For any registered component type, creating an instance
   * produces a component with all defaultProps values applied
   */
  it('should apply all defaultProps when creating a component instance', () => {
    // Register all basic components
    registerBasicComponents()
    
    fc.assert(
      fc.property(
        fc.constantFrom(...basicComponents.map(c => c.type)),
        (componentType) => {
          const definition = getComponent(componentType)
          expect(definition).not.toBeNull()
          
          const instance = createComponentInstance(componentType)
          expect(instance).not.toBeNull()
          
          // Verify all defaultProps are applied
          for (const [key, value] of Object.entries(definition.defaultProps)) {
            expect(instance.props[key]).toEqual(value)
          }
          
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: For any registered component type, creating an instance
   * produces a component with all defaultStyle values applied
   */
  it('should apply all defaultStyle when creating a component instance', () => {
    registerBasicComponents()
    
    fc.assert(
      fc.property(
        fc.constantFrom(...basicComponents.map(c => c.type)),
        (componentType) => {
          const definition = getComponent(componentType)
          expect(definition).not.toBeNull()
          
          const instance = createComponentInstance(componentType)
          expect(instance).not.toBeNull()
          
          // Verify all defaultStyle are applied
          for (const [key, value] of Object.entries(definition.defaultStyle)) {
            expect(instance.style[key]).toEqual(value)
          }
          
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: For any component definition with arbitrary defaultProps,
   * creating an instance applies all those defaults
   */
  it('should apply arbitrary defaultProps from any component definition', () => {
    // Generate arbitrary component definitions
    const propValueArb = fc.oneof(
      fc.string({ minLength: 1, maxLength: 20 }),
      fc.integer({ min: 0, max: 1000 }),
      fc.boolean(),
      fc.constant(null)
    )
    
    const defaultPropsArb = fc.dictionary(
      fc.stringMatching(/^[a-zA-Z][a-zA-Z0-9]{0,9}$/),
      propValueArb,
      { minKeys: 1, maxKeys: 5 }
    )
    
    const componentTypeArb = fc.stringMatching(/^[A-Z][a-zA-Z]{2,15}$/)
    
    fc.assert(
      fc.property(
        componentTypeArb,
        defaultPropsArb,
        (type, defaultProps) => {
          clearRegistry()
          
          const definition = {
            type,
            name: type,
            category: 'Test',
            icon: 'test',
            defaultProps,
            defaultStyle: { position: 'absolute', left: 0, top: 0 },
            propsSchema: {}
          }
          
          registerComponent(definition)
          
          const instance = createComponentInstance(type)
          expect(instance).not.toBeNull()
          
          // Verify all defaultProps are applied
          for (const [key, value] of Object.entries(defaultProps)) {
            expect(instance.props[key]).toEqual(value)
          }
          
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: For any component definition with arbitrary defaultStyle,
   * creating an instance applies all those styles
   */
  it('should apply arbitrary defaultStyle from any component definition', () => {
    const styleValueArb = fc.oneof(
      fc.integer({ min: 0, max: 1000 }),
      fc.constantFrom('absolute', 'relative', 'fixed'),
      fc.stringMatching(/^#[0-9a-fA-F]{6}$/)
    )
    
    const defaultStyleArb = fc.dictionary(
      fc.stringMatching(/^[a-zA-Z][a-zA-Z0-9]{0,9}$/),
      styleValueArb,
      { minKeys: 1, maxKeys: 5 }
    )
    
    const componentTypeArb = fc.stringMatching(/^[A-Z][a-zA-Z]{2,15}$/)
    
    fc.assert(
      fc.property(
        componentTypeArb,
        defaultStyleArb,
        (type, defaultStyle) => {
          clearRegistry()
          
          const definition = {
            type,
            name: type,
            category: 'Test',
            icon: 'test',
            defaultProps: {},
            defaultStyle,
            propsSchema: {}
          }
          
          registerComponent(definition)
          
          const instance = createComponentInstance(type)
          expect(instance).not.toBeNull()
          
          // Verify all defaultStyle are applied
          for (const [key, value] of Object.entries(defaultStyle)) {
            expect(instance.style[key]).toEqual(value)
          }
          
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: Instance should have required structural properties
   * (id, type, label, locked, visible, style, props, bindings, events, animations, children)
   */
  it('should create instance with all required structural properties', () => {
    registerBasicComponents()
    
    fc.assert(
      fc.property(
        fc.constantFrom(...basicComponents.map(c => c.type)),
        (componentType) => {
          const instance = createComponentInstance(componentType)
          expect(instance).not.toBeNull()
          
          // Verify required structural properties exist
          expect(instance).toHaveProperty('id')
          expect(instance).toHaveProperty('type')
          expect(instance).toHaveProperty('label')
          expect(instance).toHaveProperty('locked')
          expect(instance).toHaveProperty('visible')
          expect(instance).toHaveProperty('style')
          expect(instance).toHaveProperty('props')
          expect(instance).toHaveProperty('bindings')
          expect(instance).toHaveProperty('events')
          expect(instance).toHaveProperty('animations')
          expect(instance).toHaveProperty('children')
          
          // Verify default values for structural properties
          expect(instance.type).toBe(componentType)
          expect(instance.locked).toBe(false)
          expect(instance.visible).toBe(true)
          expect(instance.bindings).toEqual({})
          expect(instance.events).toEqual({})
          expect(instance.animations).toEqual([])
          expect(instance.children).toEqual([])
          
          // Verify id is a valid UUID-like string
          expect(instance.id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
          
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: Overrides should be merged with defaults, not replace them entirely
   */
  it('should merge overrides with defaults, preserving non-overridden values', () => {
    registerBasicComponents()
    
    fc.assert(
      fc.property(
        fc.constantFrom(...basicComponents.map(c => c.type)),
        fc.record({
          props: fc.dictionary(
            fc.constantFrom('customProp1', 'customProp2'),
            fc.string({ minLength: 1, maxLength: 10 }),
            { minKeys: 0, maxKeys: 2 }
          ),
          style: fc.record({
            left: fc.integer({ min: 0, max: 500 }),
            top: fc.integer({ min: 0, max: 500 })
          })
        }),
        (componentType, overrides) => {
          const definition = getComponent(componentType)
          const instance = createComponentInstance(componentType, overrides)
          
          expect(instance).not.toBeNull()
          
          // Verify overridden style values
          expect(instance.style.left).toBe(overrides.style.left)
          expect(instance.style.top).toBe(overrides.style.top)
          
          // Verify non-overridden style values are preserved from defaults
          for (const [key, value] of Object.entries(definition.defaultStyle)) {
            if (key !== 'left' && key !== 'top') {
              expect(instance.style[key]).toEqual(value)
            }
          }
          
          // Verify overridden props values
          for (const [key, value] of Object.entries(overrides.props)) {
            expect(instance.props[key]).toBe(value)
          }
          
          // Verify non-overridden props values are preserved from defaults
          for (const [key, value] of Object.entries(definition.defaultProps)) {
            if (!(key in overrides.props)) {
              expect(instance.props[key]).toEqual(value)
            }
          }
          
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: Creating instance for unregistered type should return null
   */
  it('should return null for unregistered component types', () => {
    fc.assert(
      fc.property(
        fc.stringMatching(/^[A-Z][a-zA-Z]{5,15}$/),
        (unknownType) => {
          clearRegistry()
          // Don't register any components
          
          const instance = createComponentInstance(unknownType)
          expect(instance).toBeNull()
          
          return true
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property: Each instance should have a unique ID
   */
  it('should generate unique IDs for each instance', () => {
    registerBasicComponents()
    
    fc.assert(
      fc.property(
        fc.constantFrom(...basicComponents.map(c => c.type)),
        fc.integer({ min: 2, max: 10 }),
        (componentType, count) => {
          const ids = new Set()
          
          for (let i = 0; i < count; i++) {
            const instance = createComponentInstance(componentType)
            expect(instance).not.toBeNull()
            ids.add(instance.id)
          }
          
          // All IDs should be unique
          expect(ids.size).toBe(count)
          
          return true
        }
      ),
      { numRuns: 100 }
    )
  })
})
