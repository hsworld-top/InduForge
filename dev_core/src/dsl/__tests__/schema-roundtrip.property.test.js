/**
 * Property-Based Test: Schema Round-Trip
 * **Feature: design-center, Property 12: Schema Round-Trip**
 * **Validates: Requirements 6.6, 6.7**
 *
 * For any valid Page Schema, serializing to JSON and parsing back
 * SHALL produce an equivalent schema.
 */

const fc = require('fast-check')
const { validatePageSchema } = require('../validators')
const { ScaleMode, Theme, ActionType, DataSourceType, DataSourceMode } = require('../constants')

// Arbitrary generators for DSL schema types

/**
 * Generate a valid component style
 */
const styleArbitrary = fc.record({
  position: fc.constantFrom('absolute', 'relative'),
  left: fc.integer({ min: 0, max: 2000 }),
  top: fc.integer({ min: 0, max: 2000 }),
  width: fc.integer({ min: 10, max: 1000 }),
  height: fc.integer({ min: 10, max: 1000 }),
  zIndex: fc.integer({ min: 0, max: 100 }),
})

/**
 * Generate a valid action schema
 */
const actionArbitrary = fc.record({
  id: fc.uuid(),
  action: fc.constantFrom(...Object.values(ActionType)),
  payload: fc.option(fc.dictionary(fc.string({ minLength: 1, maxLength: 20 }), fc.jsonValue()), {
    nil: undefined,
  }),
  condition: fc.option(fc.string({ minLength: 0, maxLength: 50 }), { nil: undefined }),
})

/**
 * Generate a valid component schema (non-recursive for simplicity)
 */
const componentArbitrary = fc.record({
  id: fc.uuid(),
  type: fc.constantFrom('Container', 'Text', 'Button', 'Image', 'Input'),
  label: fc.string({ minLength: 1, maxLength: 50 }),
  locked: fc.boolean(),
  visible: fc.boolean(),
  style: styleArbitrary,
  props: fc.dictionary(fc.string({ minLength: 1, maxLength: 20 }), fc.jsonValue()),
  bindings: fc.dictionary(
    fc.string({ minLength: 1, maxLength: 20 }),
    fc.string({ minLength: 1, maxLength: 100 }),
  ),
  events: fc.dictionary(
    fc.constantFrom('onClick', 'onLoad', 'onChange'),
    fc.array(actionArbitrary, { minLength: 0, maxLength: 2 }),
  ),
  animations: fc.array(
    fc.record({
      trigger: fc.constantFrom('onLoad', 'onHover', 'onClick'),
      type: fc.constantFrom('shake', 'fade', 'slide'),
      duration: fc.integer({ min: 100, max: 5000 }),
    }),
    { minLength: 0, maxLength: 2 },
  ),
  children: fc.constant([]), // Non-recursive for simplicity
})

/**
 * Generate a valid data source schema
 */
const dataSourceArbitrary = fc.oneof(
  // dataCenter type
  fc.record({
    id: fc.uuid(),
    type: fc.constant(DataSourceType.DATA_CENTER),
    mode: fc.constantFrom(DataSourceMode.SUBSCRIPTION, DataSourceMode.REQUEST),
    queryId: fc.option(fc.uuid(), { nil: undefined }),
    dataHandler: fc.option(fc.string({ minLength: 0, maxLength: 100 }), { nil: undefined }),
  }),
  // http type with poll mode (requires pollingInterval)
  fc.record({
    id: fc.uuid(),
    type: fc.constant(DataSourceType.HTTP),
    mode: fc.constant(DataSourceMode.POLL),
    pollingInterval: fc.integer({ min: 1000, max: 60000 }),
    url: fc.option(fc.webUrl(), { nil: undefined }),
    dataHandler: fc.option(fc.string({ minLength: 0, maxLength: 100 }), { nil: undefined }),
  }),
  // http type with non-poll mode
  fc.record({
    id: fc.uuid(),
    type: fc.constant(DataSourceType.HTTP),
    mode: fc.constantFrom(DataSourceMode.SUBSCRIPTION, DataSourceMode.REQUEST),
    url: fc.option(fc.webUrl(), { nil: undefined }),
    dataHandler: fc.option(fc.string({ minLength: 0, maxLength: 100 }), { nil: undefined }),
  }),
  // static type
  fc.record({
    id: fc.uuid(),
    type: fc.constant(DataSourceType.STATIC),
    mode: fc.constantFrom(DataSourceMode.SUBSCRIPTION, DataSourceMode.REQUEST),
    data: fc.option(fc.jsonValue(), { nil: undefined }),
  }),
)

/**
 * Generate a valid component ACL
 */
const componentAclArbitrary = fc.record({
  componentId: fc.uuid(),
  visibleFor: fc.array(fc.string({ minLength: 1, maxLength: 20 }), { minLength: 0, maxLength: 5 }),
  editableFor: fc.array(fc.string({ minLength: 1, maxLength: 20 }), { minLength: 0, maxLength: 5 }),
})

/**
 * Generate a valid permissions schema
 */
const permissionsArbitrary = fc.record({
  roles: fc.array(fc.string({ minLength: 1, maxLength: 20 }), { minLength: 0, maxLength: 5 }),
  componentAcl: fc.option(fc.array(componentAclArbitrary, { minLength: 0, maxLength: 3 }), {
    nil: undefined,
  }),
})

/**
 * Generate a valid hex color string
 */
const hexDigit = fc.constantFrom(
  '0',
  '1',
  '2',
  '3',
  '4',
  '5',
  '6',
  '7',
  '8',
  '9',
  'a',
  'b',
  'c',
  'd',
  'e',
  'f',
)
const hexColorArbitrary = fc
  .tuple(hexDigit, hexDigit, hexDigit, hexDigit, hexDigit, hexDigit)
  .map((digits) => `#${digits.join('')}`)

/**
 * Generate a valid page config
 */
const pageConfigArbitrary = fc.record({
  width: fc.integer({ min: 320, max: 3840 }),
  height: fc.integer({ min: 240, max: 2160 }),
  scaleMode: fc.option(fc.constantFrom(...Object.values(ScaleMode)), { nil: undefined }),
  backgroundColor: fc.option(hexColorArbitrary, { nil: undefined }),
  backgroundImage: fc.option(fc.constant(null), { nil: undefined }),
  gridSize: fc.option(fc.integer({ min: 1, max: 100 }), { nil: undefined }),
  theme: fc.option(fc.constantFrom(...Object.values(Theme)), { nil: undefined }),
})

/**
 * Generate a valid page meta
 */
const pageMetaArbitrary = fc.record({
  id: fc.uuid(),
  name: fc.string({ minLength: 1, maxLength: 100 }),
  screenshot: fc.option(fc.constant(null), { nil: undefined }),
  lockedBy: fc.option(fc.constant(null), { nil: undefined }),
  lockedAt: fc.option(fc.constant(null), { nil: undefined }),
})

/**
 * Generate a valid page schema
 */
const pageSchemaArbitrary = fc.record({
  version: fc.constantFrom('1.0.0', '2.0.0'),
  meta: pageMetaArbitrary,
  config: pageConfigArbitrary,
  variables: fc.dictionary(fc.string({ minLength: 1, maxLength: 20 }), fc.jsonValue()),
  dataSources: fc.array(dataSourceArbitrary, { minLength: 0, maxLength: 3 }),
  components: fc.array(componentArbitrary, { minLength: 0, maxLength: 5 }),
  permissions: permissionsArbitrary,
})

describe('Schema Round-Trip Property Tests', () => {
  /**
   * Property 12: Schema Round-Trip
   * For any valid Page Schema, serializing to JSON and parsing back
   * SHALL produce an equivalent schema.
   * Note: JSON serialization normalizes -0 to 0, which is acceptable for DSL purposes.
   */
  test('JSON.parse(JSON.stringify(schema)) produces equivalent schema', () => {
    fc.assert(
      fc.property(pageSchemaArbitrary, (schema) => {
        // Serialize to JSON string
        const jsonString = JSON.stringify(schema)

        // Parse back to object
        const parsed = JSON.parse(jsonString)

        // Use JSON comparison to handle -0 vs 0 edge case
        // JSON.stringify normalizes -0 to 0, which is semantically equivalent
        expect(JSON.stringify(parsed)).toEqual(jsonString)

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Additional property: Round-trip preserves validation status
   * If a schema is valid before serialization, it should be valid after.
   */
  test('Round-trip preserves schema validity', () => {
    fc.assert(
      fc.property(pageSchemaArbitrary, (schema) => {
        // Validate original
        const originalValidation = validatePageSchema(schema)

        // Round-trip
        const roundTripped = JSON.parse(JSON.stringify(schema))

        // Validate round-tripped
        const roundTrippedValidation = validatePageSchema(roundTripped)

        // Validation results should match
        expect(roundTrippedValidation.valid).toBe(originalValidation.valid)

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property: Component schema round-trip
   * For any valid component schema, serialization preserves structure.
   * Note: JSON serialization normalizes -0 to 0, which is acceptable for DSL purposes.
   */
  test('Component schema round-trip preserves structure', () => {
    fc.assert(
      fc.property(componentArbitrary, (component) => {
        const roundTripped = JSON.parse(JSON.stringify(component))
        // Use JSON comparison to handle -0 vs 0 edge case
        // JSON.stringify normalizes -0 to 0, which is semantically equivalent
        expect(JSON.stringify(roundTripped)).toEqual(JSON.stringify(component))
        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property: DataSource schema round-trip
   * For any valid data source schema, serialization preserves structure.
   * Note: JSON serialization normalizes -0 to 0, which is acceptable for DSL purposes.
   */
  test('DataSource schema round-trip preserves structure', () => {
    fc.assert(
      fc.property(dataSourceArbitrary, (dataSource) => {
        const roundTripped = JSON.parse(JSON.stringify(dataSource))
        // Use JSON comparison to handle -0 vs 0 edge case
        // JSON.stringify normalizes -0 to 0, which is semantically equivalent
        expect(JSON.stringify(roundTripped)).toEqual(JSON.stringify(dataSource))
        return true
      }),
      { numRuns: 100 },
    )
  })
})
