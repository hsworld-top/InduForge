/**
 * DSL Constants - Enum values for DSL schemas
 * Requirements: 1.1, 2.1, 3.1, 4.1, 5.1
 */

/**
 * Page scale mode options
 * @enum {string}
 */
const ScaleMode = {
  FIT: 'fit',
  FILL: 'fill',
  FIXED: 'fixed',
};

/**
 * Page theme options
 * @enum {string}
 */
const Theme = {
  DARK: 'dark',
  LIGHT: 'light',
};

/**
 * Action types for event handlers
 * Requirements: 3.1
 * @enum {string}
 */
const ActionType = {
  SET_VARIABLE: 'setVariable',
  EXECUTE_QUERY: 'executeQuery',
  NAVIGATE: 'navigate',
  OPEN_DIALOG: 'openDialog',
  CLOSE_DIALOG: 'closeDialog',
  MESSAGE: 'message',
  SCRIPT: 'script',
};

/**
 * DataSource types
 * Requirements: 4.1
 * @enum {string}
 */
const DataSourceType = {
  DATA_CENTER: 'dataCenter',
  HTTP: 'http',
  STATIC: 'static',
};


/**
 * DataSource mode options
 * Requirements: 4.2
 * @enum {string}
 */
const DataSourceMode = {
  SUBSCRIPTION: 'subscription',
  POLL: 'poll',
  REQUEST: 'request',
};

/**
 * Page type options
 * Requirements: 6.3
 * @enum {string}
 */
const PageType = {
  PAGE: 'page',
  FOLDER: 'folder',
  DIALOG: 'dialog',
};

/**
 * Data tag types
 * Requirements: 11.2
 * @enum {string}
 */
const DataTagType = {
  BOOLEAN: 'boolean',
  INT16: 'int16',
  UINT16: 'uint16',
  INT32: 'int32',
  FLOAT: 'float',
  DOUBLE: 'double',
  STRING: 'string',
  JSON: 'json',
};

/**
 * Project member roles
 * Requirements: 15.2
 * @enum {string}
 */
const ProjectMemberRole = {
  OWNER: 'OWNER',
  ADMIN: 'ADMIN',
  DEVELOPER: 'DEVELOPER',
  VIEWER: 'VIEWER',
};

/**
 * Page history change types
 * Requirements: 7.2
 * @enum {string}
 */
const ChangeType = {
  MANUAL: 'manual',
  AUTO: 'auto',
  PUBLISH: 'publish',
};

/**
 * Asset types
 * Requirements: 9.2
 * @enum {string}
 */
const AssetType = {
  IMAGE: 'image',
  VIDEO: 'video',
  MODEL_3D: 'model_3d',
  SVG: 'svg',
};

module.exports = {
  ScaleMode,
  Theme,
  ActionType,
  DataSourceType,
  DataSourceMode,
  PageType,
  DataTagType,
  ProjectMemberRole,
  ChangeType,
  AssetType,
};
