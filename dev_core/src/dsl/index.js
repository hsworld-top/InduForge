/**
 * DSL Module - Schema definitions, constants, and validators
 * for InduForge low-code platform
 */

const constants = require('./constants')
const validators = require('./validators')

module.exports = {
  ...constants,
  ...validators,
}
