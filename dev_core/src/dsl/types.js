/**
 * DSL Type Definitions using JSDoc
 * Requirements: 1.1, 2.1, 3.1, 4.1, 5.1
 */

const { ScaleMode, Theme, ActionType, DataSourceType, DataSourceMode } = require('./constants')

/**
 * Page meta information
 * @typedef {Object} PageMeta
 * @property {string} id - Page unique identifier
 * @property {string} name - Page display name
 * @property {string|null} [screenshot] - Screenshot URL using assets:// protocol
 * @property {string|null} [lockedBy] - User ID who locked the page
 * @property {string|null} [lockedAt] - Lock timestamp in ISO format
 */

/**
 * Page configuration
 * @typedef {Object} PageConfig
 * @property {number} width - Page width in pixels
 * @property {number} height - Page height in pixels
 * @property {'fit'|'fill'|'fixed'} [scaleMode] - Scale mode for responsive display
 * @property {string} [backgroundColor] - Background color (hex format)
 * @property {string|null} [backgroundImage] - Background image using assets:// protocol
 * @property {number} [gridSize] - Grid size for alignment
 * @property {'dark'|'light'} [theme] - Page theme
 */

/**
 * Component style definition
 * @typedef {Object} ComponentStyle
 * @property {string} [position] - CSS position value
 * @property {number} [left] - Left position in pixels
 * @property {number} [top] - Top position in pixels
 * @property {number} [width] - Width in pixels
 * @property {number} [height] - Height in pixels
 * @property {number} [zIndex] - Z-index for layering
 * @property {string} [transform] - CSS transform value
 */

/**
 * Animation definition
 * @typedef {Object} Animation
 * @property {string} trigger - Animation trigger type
 * @property {string} [condition] - Condition expression using {{ }} syntax
 * @property {string} type - Animation type (e.g., 'shake', 'fade')
 * @property {number} duration - Animation duration in milliseconds
 */

/**
 * Action schema for event handlers
 * Requirements: 3.1
 * @typedef {Object} ActionSchema
 * @property {string} id - Action unique identifier
 * @property {'setVariable'|'executeQuery'|'navigate'|'openDialog'|'closeDialog'|'message'|'script'} action - Action type
 * @property {Object} [payload] - Action payload data
 * @property {string} [condition] - Condition expression using {{ $prevResult }}
 */

/**
 * Component schema definition
 * Requirements: 2.1
 * @typedef {Object} ComponentSchema
 * @property {string} id - Component unique identifier
 * @property {string} type - Component type name
 * @property {string|null} [refId] - Reference to custom component ID
 * @property {string} label - Component display label
 * @property {boolean} locked - Whether component is locked for editing
 * @property {boolean} visible - Whether component is visible
 * @property {ComponentStyle} style - Component style configuration
 * @property {Object} props - Component properties
 * @property {Object.<string, string>} [bindings] - Data bindings using {{ expression }} syntax
 * @property {Object.<string, ActionSchema[]>} [events] - Event handlers
 * @property {Animation[]} [animations] - Animation configurations
 * @property {ComponentSchema[]} [children] - Child components
 */

/**
 * DataSource schema definition
 * Requirements: 4.1, 4.2
 * @typedef {Object} DataSourceSchema
 * @property {string} id - DataSource unique identifier
 * @property {'dataCenter'|'http'|'static'} type - DataSource type
 * @property {string} [queryId] - Query ID for dataCenter type
 * @property {'subscription'|'poll'|'request'} mode - Data fetch mode
 * @property {number} [pollingInterval] - Polling interval in milliseconds (required when mode is 'poll')
 * @property {string} [dataHandler] - JavaScript function string for data transformation
 * @property {string} [url] - URL for http type
 * @property {Object} [headers] - HTTP headers for http type
 * @property {*} [data] - Static data for static type
 */

/**
 * Component ACL entry
 * Requirements: 5.2
 * @typedef {Object} ComponentAcl
 * @property {string} componentId - Component ID
 * @property {string[]} visibleFor - Roles that can see this component
 * @property {string[]} editableFor - Roles that can edit this component
 */

/**
 * Permissions schema definition
 * Requirements: 5.1, 5.2
 * @typedef {Object} PermissionsSchema
 * @property {string[]} roles - Roles allowed to access the page
 * @property {ComponentAcl[]} [componentAcl] - Component-level access control list
 */

/**
 * Page schema definition - the complete DSL structure
 * Requirements: 1.1
 * @typedef {Object} PageSchema
 * @property {string} version - DSL version number
 * @property {PageMeta} meta - Page meta information
 * @property {PageConfig} config - Page configuration
 * @property {Object.<string, *>} variables - Page variables (key-value pairs)
 * @property {DataSourceSchema[]} dataSources - Data source definitions
 * @property {ComponentSchema[]} components - Component tree
 * @property {PermissionsSchema} permissions - Permission configuration
 */

/**
 * Validation result
 * @typedef {Object} ValidationResult
 * @property {boolean} valid - Whether validation passed
 * @property {ValidationError[]} errors - List of validation errors
 */

/**
 * Validation error
 * @typedef {Object} ValidationError
 * @property {string} path - Path to the invalid field
 * @property {string} message - Error message
 * @property {string} [code] - Error code
 */

module.exports = {
  // Type exports are for documentation purposes
  // The actual validation is done in validators.js
}
