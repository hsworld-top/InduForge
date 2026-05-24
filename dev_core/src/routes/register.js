const { buildV1Router } = require('./v1')
const { buildV2Router } = require('./v2')

function registerVersionedRoutes(app, options = {}) {
  // v1
  const v1Router = buildV1Router({ authLimiter: options.authLimiter })
  app.use('/api/v1', v1Router)
  // v2
  const v2Router = buildV2Router(options)
  app.use('/api/v2', v2Router)
}

module.exports = { registerVersionedRoutes }
