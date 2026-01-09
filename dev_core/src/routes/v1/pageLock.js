/**
 * Page Lock Routes - 页面锁路由
 */
const express = require("express");
const router = express.Router();
const { authenticateToken } = require("../../middlewares/auth");
const pageLockController = require("../../controllers/pageLockController");

router.use(authenticateToken);

router.get("/:pageId/lock", pageLockController.getPageLock);
router.post("/:pageId/lock", pageLockController.acquirePageLock);
router.delete("/:pageId/lock", pageLockController.releasePageLock);
router.post("/:pageId/lock/heartbeat", pageLockController.heartbeatPageLock);
router.post("/:pageId/lock/release", pageLockController.releasePageLockBeacon);
router.delete("/:pageId/lock/force", pageLockController.forceReleasePageLock);

module.exports = router;
