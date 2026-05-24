/**
 * Property-Based Test: Schema Validation
 * **Feature: design-center, Property 11: Schema Validation**
 * **Validates: Requirements 6.3**
 *
 * For any page schema submitted to the backend, the validator SHALL return
 * errors for missing required fields or invalid enum values.
 */

const fc = require('fast-check')
const {
  validatePageSchema,
  validateComponentSchema,
  validateDataSourceSchema,
  validateActionSchema,
} = require('../validators')
const { ScaleMode, Theme, ActionType, DataSourceType, DataSourceMode } = require('../constants')

// Valid enum values for reference
const validScaleModes = Object.values(ScaleMode)
const validThemes = Object.values(Theme)
const validActionTypes = Object.values(ActionType)
const validDataSourceTypes = Object.values(DataSourceType)
const validDataSourceModes = Object.values(DataSourceMode)

// Required fields for each schema type
const pageRequiredFields = [
  'version',
  'meta',
  'config',
  'variables',
  'dataSources',
  'components',
  'permissions',
]
const componentRequiredFields = [
  'id',
  'type',
  'label',
  'locked',
  'visible',
  'style',
  'props',
  'bindings',
  'events',
  'animations',
  'children',
]
const dataSourceRequiredFields = ['id', 'type', 'mode']

/**
 * Generate a valid base page schema
 */
const validPageSchemaArbitrary = fc.record({
  version: fc.constantFrom('1.0.0', '2.0.0'),
  meta: fc.record({
    id: fc.uuid(),
    name: fc.string({ minLength: 1, maxLength: 50 }),
  }),
  config: fc.record({
    width: fc.integer({ min: 320, max: 3840 }),
    height: fc.integer({ min: 240, max: 2160 }),
    scaleMode: fc.constantFrom(...validScaleModes),
    theme: fc.constantFrom(...validThemes),
  }),
  variables: fc.dictionary(fc.string({ minLength: 1, maxLength: 20 }), fc.jsonValue()),
  dataSources: fc.constant([]),
  components: fc.constant([]),
  permissions: fc.record({
    roles: fc.array(fc.string({ minLength: 1, maxLength: 20 }), { minLength: 0, maxLength: 3 }),
  }),
})

/**
 * Generate a valid component schema
 */
const validComponentSchemaArbitrary = fc.record({
  id: fc.uuid(),
  type: fc.constantFrom('Container', 'Text', 'Button', 'Image', 'Input'),
  label: fc.string({ minLength: 1, maxLength: 50 }),
  locked: fc.boolean(),
  visible: fc.boolean(),
  style: fc.record({
    position: fc.constantFrom('absolute', 'relative'),
    left: fc.integer({ min: 0, max: 2000 }),
    top: fc.integer({ min: 0, max: 2000 }),
    width: fc.integer({ min: 10, max: 1000 }),
    height: fc.integer({ min: 10, max: 1000 }),
  }),
  props: fc.dictionary(fc.string({ minLength: 1, maxLength: 20 }), fc.jsonValue()),
  bindings: fc.dictionary(
    fc.string({ minLength: 1, maxLength: 20 }),
    fc.string({ minLength: 1, maxLength: 50 }),
  ),
  events: fc.constant({}),
  animations: fc.constant([]),
  children: fc.constant([]),
})

/**
 * Generate a valid data source schema
 */
const validDataSourceSchemaArbitrary = fc.record({
  id: fc.uuid(),
  type: fc.constantFrom(...validDataSourceTypes),
  mode: fc.constantFrom(DataSourceMode.SUBSCRIPTION, DataSourceMode.REQUEST),
})

/**
 * Generate an invalid enum value (not in the valid set)
 */
const invalidEnumArbitrary = (validValues) => {
  return fc.string({ minLength: 1, maxLength: 30 }).filter((s) => !validValues.includes(s))
}

describe('Schema Validation Property Tests', () => {
  /**
   * Property 11.1: Missing required fields detection for Page Schema
   * For any page schema with a required field removed, the validator SHALL
   * return an error indicating the missing field.
   */
  test('validates missing required fields in page schema', () => {
    fc.assert(
      fc.property(
        validPageSchemaArbitrary,
        fc.constantFrom(...pageRequiredFields),
        (schema, fieldToRemove) => {
          // Create a copy and remove one required field
          const schemaWithMissingField = { ...schema }
          delete schemaWithMissingField[fieldToRemove]

          // Validate
          const result = validatePageSchema(schemaWithMissingField)

          // Should be invalid
          expect(result.valid).toBe(false)

          // Should have an error for the missing field
          const hasMissingFieldError = result.errors.some(
            (e) => e.path === fieldToRemove && e.code === 'MISSING_FIELD',
          )
          expect(hasMissingFieldError).toBe(true)

          return true
        },
      ),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.2: Invalid scaleMode enum detection
   * For any page schema with an invalid scaleMode value, the validator SHALL
   * return an error for the invalid enum.
   */
  test('validates invalid scaleMode enum in page schema', () => {
    fc.assert(
      fc.property(
        validPageSchemaArbitrary,
        invalidEnumArbitrary(validScaleModes),
        (schema, invalidScaleMode) => {
          // Create schema with invalid scaleMode
          const schemaWithInvalidEnum = {
            ...schema,
            config: {
              ...schema.config,
              scaleMode: invalidScaleMode,
            },
          }

          // Validate
          const result = validatePageSchema(schemaWithInvalidEnum)

          // Should be invalid
          expect(result.valid).toBe(false)

          // Should have an error for config.scaleMode
          const hasScaleModeError = result.errors.some((e) => e.path === 'config.scaleMode')
          expect(hasScaleModeError).toBe(true)

          return true
        },
      ),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.3: Invalid theme enum detection
   * For any page schema with an invalid theme value, the validator SHALL
   * return an error for the invalid enum.
   */
  test('validates invalid theme enum in page schema', () => {
    fc.assert(
      fc.property(
        validPageSchemaArbitrary,
        invalidEnumArbitrary(validThemes),
        (schema, invalidTheme) => {
          // Create schema with invalid theme
          const schemaWithInvalidEnum = {
            ...schema,
            config: {
              ...schema.config,
              theme: invalidTheme,
            },
          }

          // Validate
          const result = validatePageSchema(schemaWithInvalidEnum)

          // Should be invalid
          expect(result.valid).toBe(false)

          // Should have an error for config.theme
          const hasThemeError = result.errors.some((e) => e.path === 'config.theme')
          expect(hasThemeError).toBe(true)

          return true
        },
      ),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.4: Missing required fields detection for Component Schema
   * For any component schema with a required field removed, the validator SHALL
   * return an error indicating the missing field.
   */
  test('validates missing required fields in component schema', () => {
    fc.assert(
      fc.property(
        validComponentSchemaArbitrary,
        fc.constantFrom(...componentRequiredFields),
        (schema, fieldToRemove) => {
          // Create a copy and remove one required field
          const schemaWithMissingField = { ...schema }
          delete schemaWithMissingField[fieldToRemove]

          // Validate
          const result = validateComponentSchema(schemaWithMissingField)

          // Should be invalid
          expect(result.valid).toBe(false)

          // Should have an error for the missing field
          const hasMissingFieldError = result.errors.some(
            (e) => e.path === fieldToRemove && e.code === 'MISSING_FIELD',
          )
          expect(hasMissingFieldError).toBe(true)

          return true
        },
      ),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.5: Missing required fields detection for DataSource Schema
   * For any data source schema with a required field removed, the validator SHALL
   * return an error indicating the missing field.
   */
  test('validates missing required fields in data source schema', () => {
    fc.assert(
      fc.property(
        validDataSourceSchemaArbitrary,
        fc.constantFrom(...dataSourceRequiredFields),
        (schema, fieldToRemove) => {
          // Create a copy and remove one required field
          const schemaWithMissingField = { ...schema }
          delete schemaWithMissingField[fieldToRemove]

          // Validate
          const result = validateDataSourceSchema(schemaWithMissingField)

          // Should be invalid
          expect(result.valid).toBe(false)

          // Should have an error for the missing field
          const hasMissingFieldError = result.errors.some(
            (e) => e.path === fieldToRemove && e.code === 'MISSING_FIELD',
          )
          expect(hasMissingFieldError).toBe(true)

          return true
        },
      ),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.6: Invalid DataSource type enum detection
   * For any data source schema with an invalid type value, the validator SHALL
   * return an error for the invalid enum.
   */
  test('validates invalid type enum in data source schema', () => {
    fc.assert(
      fc.property(
        validDataSourceSchemaArbitrary,
        invalidEnumArbitrary(validDataSourceTypes),
        (schema, invalidType) => {
          // Create schema with invalid type
          const schemaWithInvalidEnum = {
            ...schema,
            type: invalidType,
          }

          // Validate
          const result = validateDataSourceSchema(schemaWithInvalidEnum)

          // Should be invalid
          expect(result.valid).toBe(false)

          // Should have an error for type
          const hasTypeError = result.errors.some((e) => e.path === 'type')
          expect(hasTypeError).toBe(true)

          return true
        },
      ),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.7: Invalid DataSource mode enum detection
   * For any data source schema with an invalid mode value, the validator SHALL
   * return an error for the invalid enum.
   */
  test('validates invalid mode enum in data source schema', () => {
    fc.assert(
      fc.property(
        validDataSourceSchemaArbitrary,
        invalidEnumArbitrary(validDataSourceModes),
        (schema, invalidMode) => {
          // Create schema with invalid mode
          const schemaWithInvalidEnum = {
            ...schema,
            mode: invalidMode,
          }

          // Validate
          const result = validateDataSourceSchema(schemaWithInvalidEnum)

          // Should be invalid
          expect(result.valid).toBe(false)

          // Should have an error for mode
          const hasModeError = result.errors.some((e) => e.path === 'mode')
          expect(hasModeError).toBe(true)

          return true
        },
      ),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.8: Invalid Action type enum detection
   * For any action schema with an invalid action type, the validator SHALL
   * return an error for the invalid enum.
   */
  test('validates invalid action type enum in action schema', () => {
    fc.assert(
      fc.property(fc.uuid(), invalidEnumArbitrary(validActionTypes), (id, invalidAction) => {
        // Create action schema with invalid action type
        const actionSchema = {
          id,
          action: invalidAction,
        }

        // Validate
        const result = validateActionSchema(actionSchema)

        // Should be invalid
        expect(result.valid).toBe(false)

        // Should have an error for action
        const hasActionError = result.errors.some((e) => e.path === 'action')
        expect(hasActionError).toBe(true)

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.9: Valid schemas pass validation
   * For any valid page schema, the validator SHALL return valid: true with no errors.
   */
  test('valid page schemas pass validation', () => {
    fc.assert(
      fc.property(validPageSchemaArbitrary, (schema) => {
        const result = validatePageSchema(schema)

        expect(result.valid).toBe(true)
        expect(result.errors).toHaveLength(0)

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.10: Valid component schemas pass validation
   * For any valid component schema, the validator SHALL return valid: true with no errors.
   */
  test('valid component schemas pass validation', () => {
    fc.assert(
      fc.property(validComponentSchemaArbitrary, (schema) => {
        const result = validateComponentSchema(schema)

        expect(result.valid).toBe(true)
        expect(result.errors).toHaveLength(0)

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property 11.11: Valid data source schemas pass validation
   * For any valid data source schema, the validator SHALL return valid: true with no errors.
   */
  test('valid data source schemas pass validation', () => {
    fc.assert(
      fc.property(validDataSourceSchemaArbitrary, (schema) => {
        const result = validateDataSourceSchema(schema)

        expect(result.valid).toBe(true)
        expect(result.errors).toHaveLength(0)

        return true
      }),
      { numRuns: 100 },
    )
  })
})
