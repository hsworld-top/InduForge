const express = require("express");
const multer = require("multer");
const { authenticateToken } = require("../../middlewares/auth");
const designAssetController = require("../../controllers/designAssetController");

const router = express.Router();

const upload = multer({
  storage: multer.memoryStorage(),
  limits: {
    fileSize: 50 * 1024 * 1024,
  },
});

// 公开访问资源文件
router.get(
  "/projects/:projectId/assets/:assetId/file",
  designAssetController.getAssetFile
);

router.use(authenticateToken);

// 资源文件夹
router.get(
  "/projects/:projectId/asset-folders",
  designAssetController.getFolders
);
router.post(
  "/projects/:projectId/asset-folders",
  designAssetController.createFolder
);
router.patch(
  "/projects/:projectId/asset-folders/:folderId",
  designAssetController.renameFolder
);
router.delete(
  "/projects/:projectId/asset-folders/:folderId",
  designAssetController.deleteFolder
);

// 资源文件
router.get(
  "/projects/:projectId/assets",
  designAssetController.getAssets
);
router.post(
  "/projects/:projectId/assets",
  upload.array("files"),
  designAssetController.uploadAssets
);
router.patch(
  "/projects/:projectId/assets/:assetId",
  designAssetController.updateAsset
);
router.post(
  "/projects/:projectId/assets/:assetId/copy",
  designAssetController.copyAsset
);
router.delete(
  "/projects/:projectId/assets/:assetId",
  designAssetController.deleteAsset
);

module.exports = router;
