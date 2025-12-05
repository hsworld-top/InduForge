const express = require('express');
const { logger } = require('../../utils/logger');
// Reuse existing route modules
const authRoutes = require('./auth');
const tenantRoutes = require('./tenant');
const userRoutes = require('./user');
const projectRoutes = require('./project');
const logRoutes = require('./log');
const roleRoutes = require('./role');
const dataRoutes = require('./data');
const designRoutes = require('./design');

function buildV1Router(options = {}) {
  const router = express.Router();
  const { authLimiter } = options;

  if (authLimiter) {
    router.use('/auth', authLimiter, authRoutes);
  } else {
    router.use('/auth', authRoutes);
  }
  router.use('/tenants', tenantRoutes);
  router.use('/users', userRoutes);
  router.use('/projects', projectRoutes);
  router.use('/logs', logRoutes);
  router.use('/roles', roleRoutes);
  router.use('/data', dataRoutes);
  router.use('/', designRoutes); // Design routes use /projects/:projectId/design/... pattern
  
  logger.info('V1 router mounted successfully');
  return router;
}

module.exports = { buildV1Router };


