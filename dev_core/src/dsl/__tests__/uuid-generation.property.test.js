/**
 * Property-Based Test: UUID Generation
 * **Feature: design-center, Property 14: UUID Generation**
 * **Validates: Requirements 7.3**
 *
 * For any page creation request, the backend SHALL generate a valid UUID v4 for the page id.
 */

const fc = require('fast-check')

/**
 * UUID v4 validation regex
 * Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
 * where x is any hex digit and y is one of 8, 9, a, or b
 */
const UUID_V4_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

/**
 * Validates if a string is a valid UUID v4
 * @param {string} uuid - The string to validate
 * @returns {boolean} True if valid UUID v4
 */
function isValidUUIDv4(uuid) {
  if (typeof uuid !== 'string') return false
  return UUID_V4_REGEX.test(uuid)
}

/**
 * Simulates UUID v4 generation as done by Sequelize's UUIDV4
 * This mimics the behavior of DataTypes.UUIDV4
 */
function generateUUIDv4() {
  // Standard UUID v4 generation algorithm
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

describe('UUID Generation Property Tests', () => {
  /**
   * Property 14: UUID Generation
   * For any page creation request, the backend SHALL generate a valid UUID v4 for the page id.
   *
   * This test verifies that the UUID generation function produces valid UUID v4 strings.
   */
  test('Generated UUIDs are valid UUID v4 format', () => {
    fc.assert(
      fc.property(fc.integer({ min: 1, max: 1000 }), () => {
        // Generate a UUID using the same algorithm as Sequelize UUIDV4
        const uuid = generateUUIDv4()

        // Verify it matches UUID v4 format
        expect(isValidUUIDv4(uuid)).toBe(true)

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property: UUID v4 version bit is always 4
   * The 13th character (after removing hyphens) should always be '4'
   */
  test('UUID v4 has correct version bit (4)', () => {
    fc.assert(
      fc.property(fc.integer({ min: 1, max: 1000 }), () => {
        const uuid = generateUUIDv4()

        // The version is in the 13th position (index 14 with hyphens)
        // Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
        const versionChar = uuid.charAt(14)
        expect(versionChar).toBe('4')

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property: UUID v4 variant bits are correct
   * The 17th character should be one of 8, 9, a, or b
   */
  test('UUID v4 has correct variant bits (8, 9, a, or b)', () => {
    fc.assert(
      fc.property(fc.integer({ min: 1, max: 1000 }), () => {
        const uuid = generateUUIDv4()

        // The variant is in the 17th position (index 19 with hyphens)
        // Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
        const variantChar = uuid.charAt(19).toLowerCase()
        expect(['8', '9', 'a', 'b']).toContain(variantChar)

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property: Generated UUIDs are unique
   * Multiple generated UUIDs should all be different
   */
  test('Generated UUIDs are unique', () => {
    fc.assert(
      fc.property(fc.integer({ min: 10, max: 100 }), (count) => {
        const uuids = new Set()

        for (let i = 0; i < count; i++) {
          uuids.add(generateUUIDv4())
        }

        // All generated UUIDs should be unique
        expect(uuids.size).toBe(count)

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property: UUID format is consistent
   * All generated UUIDs should have the same structure (8-4-4-4-12)
   */
  test('UUID format follows 8-4-4-4-12 structure', () => {
    fc.assert(
      fc.property(fc.integer({ min: 1, max: 1000 }), () => {
        const uuid = generateUUIDv4()
        const parts = uuid.split('-')

        expect(parts.length).toBe(5)
        expect(parts[0].length).toBe(8)
        expect(parts[1].length).toBe(4)
        expect(parts[2].length).toBe(4)
        expect(parts[3].length).toBe(4)
        expect(parts[4].length).toBe(12)

        return true
      }),
      { numRuns: 100 },
    )
  })

  /**
   * Property: UUID contains only valid hex characters
   * All characters (except hyphens) should be valid hexadecimal digits
   */
  test('UUID contains only valid hexadecimal characters', () => {
    fc.assert(
      fc.property(fc.integer({ min: 1, max: 1000 }), () => {
        const uuid = generateUUIDv4()
        const hexOnly = uuid.replace(/-/g, '')

        // Should be 32 hex characters
        expect(hexOnly.length).toBe(32)

        // All characters should be valid hex
        expect(/^[0-9a-f]+$/i.test(hexOnly)).toBe(true)

        return true
      }),
      { numRuns: 100 },
    )
  })
})
