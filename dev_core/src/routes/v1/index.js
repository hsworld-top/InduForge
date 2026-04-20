const express = require("express");
const { logger } = require("../../utils/logger");
const { operationAuditLog } = require("../../middlewares/operationAuditLog");
// Reuse existing route modules
const authRoutes = require("./auth");
const tenantRoutes = require("./tenant");
const userRoutes = require("./user");
const projectRoutes = require("./project");
const logRoutes = require("./log");
const roleRoutes = require("./role");
const designRoutes = require("./design");
const designAssetsRoutes = require("./designAssets");
const pageLockRoutes = require("./pageLock");
// 运维模块路由
const nodeRoutes = require("./node");
const nodeRegisterRoutes = require("./node-register");
const deploymentRoutes = require("./deployment");
const publishRoutes = require("./publish");

function buildV1Router(options = {}) {
  const router = express.Router();
  const { authLimiter } = options;

  if (authLimiter) {
    router.use("/auth", authLimiter, authRoutes);
  } else {
    router.use("/auth", authRoutes);
  }
  router.use(operationAuditLog);
  router.use("/tenants", tenantRoutes);
  router.use("/users", userRoutes);
  router.use("/projects", projectRoutes);
  router.use("/logs", logRoutes);
  router.use("/roles", roleRoutes);
  router.use("/design", designRoutes);
  router.use("/design", designAssetsRoutes);
  router.use("/pages", pageLockRoutes);
  // 运维模块路由
  router.use("/nodes", nodeRoutes);
  router.use("/node-register", nodeRegisterRoutes); // 节点注册（无需认证）
  router.use("/deployments", deploymentRoutes);
  router.use("/publish", publishRoutes);

  logger.info("V1 router mounted successfully");

  return router;
}

module.exports = { buildV1Router };
