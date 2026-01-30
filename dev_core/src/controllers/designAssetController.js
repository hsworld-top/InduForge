const designAssetService = require("../services/designAssetService");
const { Project } = require("../models");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");

/**
 * 检查用户是否有权限访问指定工程
 * @param {Object} req - Express 请求对象
 * @param {string} projectId - 工程ID
 */
async function checkProjectAccess(req, projectId) {
  if (req.user.role === "SUPER_ADMIN" || req.user.role === "SYSTEM_ADMIN") {
    return;
  }

  const project = await Project.findOne({
    where: { id: projectId, tenantId: req.user.tenantId },
  });

  if (!project) {
    throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
      message: "无权访问此工程",
    });
  }
}

async function getFolders(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const folders = await designAssetService.listFolders(projectId);
    res.json({ success: true, data: { folders } });
  } catch (error) {
    next(error);
  }
}

async function createFolder(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const { name, parentId } = req.body || {};
    if (!name || typeof name !== "string") {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "文件夹名称不能为空",
      });
    }

    const folder = await designAssetService.createFolder(
      projectId,
      parentId,
      name
    );
    res.status(201).json({
      success: true,
      message: "文件夹创建成功",
      data: folder,
    });
  } catch (error) {
    next(error);
  }
}

async function renameFolder(req, res, next) {
  try {
    const { projectId, folderId } = req.params;
    await checkProjectAccess(req, projectId);

    const { name, parentId } = req.body || {};
    if ((!name || typeof name !== "string") && typeof parentId === "undefined") {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "文件夹名称不能为空",
      });
    }

    const folder = await designAssetService.updateFolder(projectId, folderId, {
      name,
      parentId,
    });

    res.json({
      success: true,
      message: "文件夹已更新",
      data: folder,
    });
  } catch (error) {
    next(error);
  }
}

async function deleteFolder(req, res, next) {
  try {
    const { projectId, folderId } = req.params;
    await checkProjectAccess(req, projectId);

    await designAssetService.deleteFolder(projectId, folderId);
    res.json({ success: true, message: "文件夹已删除" });
  } catch (error) {
    next(error);
  }
}

async function getAssets(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const { folderId, keyword, type } = req.query || {};
    const assets = await designAssetService.listAssets(projectId, {
      folderId,
      keyword,
      type,
    });

    res.json({ success: true, data: { assets } });
  } catch (error) {
    next(error);
  }
}

async function uploadAssets(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const files = req.files || [];
    if (!files.length) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "请上传资源文件",
      });
    }

    const { folderId, conflictStrategy } = req.body || {};
    const assets = await designAssetService.createAssets(
      projectId,
      folderId,
      files,
      req.user.id,
      { conflictStrategy }
    );

    res.status(201).json({
      success: true,
      message: "资源上传成功",
      data: { assets },
    });
  } catch (error) {
    next(error);
  }
}

async function deleteAsset(req, res, next) {
  try {
    const { projectId, assetId } = req.params;
    await checkProjectAccess(req, projectId);

    await designAssetService.deleteAsset(projectId, assetId);
    res.json({ success: true, message: "资源已删除" });
  } catch (error) {
    next(error);
  }
}

async function updateAsset(req, res, next) {
  try {
    const { projectId, assetId } = req.params;
    await checkProjectAccess(req, projectId);

    const { name, folderId } = req.body || {};
    const asset = await designAssetService.updateAsset(projectId, assetId, {
      name,
      folderId,
    });

    res.json({
      success: true,
      message: "资源已更新",
      data: asset,
    });
  } catch (error) {
    next(error);
  }
}

async function copyAsset(req, res, next) {
  try {
    const { projectId, assetId } = req.params;
    await checkProjectAccess(req, projectId);

    const { folderId, name } = req.body || {};
    const asset = await designAssetService.copyAsset(
      projectId,
      assetId,
      folderId,
      name,
      req.user.id
    );

    res.status(201).json({
      success: true,
      message: "资源已复制",
      data: asset,
    });
  } catch (error) {
    next(error);
  }
}

async function getAssetFile(req, res, next) {
  try {
    const { projectId, assetId } = req.params;
    const { asset, stream } = await designAssetService.getAssetFile(
      projectId,
      assetId
    );

    res.setHeader("Content-Type", asset.mimeType || "application/octet-stream");
    res.setHeader("Cache-Control", "public, max-age=31536000");
    stream.on("error", (error) => next(error));
    stream.pipe(res);
  } catch (error) {
    next(error);
  }
}

module.exports = {
  getFolders,
  createFolder,
  renameFolder,
  deleteFolder,
  getAssets,
  uploadAssets,
  deleteAsset,
  updateAsset,
  copyAsset,
  getAssetFile,
};
