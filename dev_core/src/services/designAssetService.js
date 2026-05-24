const path = require('path')
const { Readable } = require('stream')
const { Op } = require('sequelize')
const { randomUUID } = require('crypto')
const { DesignAsset, DesignAssetFolder } = require('../models')
const AppError = require('../utils/AppError')
const ErrorCodes = require('../constants/errorCodes')
const storageService = require('./storageService')

const buildAssetUrl = (projectId, assetId) =>
  `/api/v1/design/projects/${projectId}/assets/${assetId}/file`

const normalizeTypeFilter = (type) => {
  if (!type) return null
  if (Array.isArray(type)) return type
  if (typeof type === 'string' && type.includes(',')) {
    return type
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
  }
  return type
}

const detectAssetType = (mimeType, filename) => {
  if (mimeType === 'image/svg+xml') return 'svg'
  if (mimeType && mimeType.startsWith('image/')) return 'image'
  if (mimeType && mimeType.startsWith('video/')) return 'video'
  if (mimeType && mimeType.startsWith('audio/')) return 'audio'
  if (mimeType && mimeType.startsWith('font/')) return 'font'
  if (mimeType && mimeType.startsWith('model/')) return 'model_3d'
  if (mimeType === 'application/json') return 'json'

  const ext = path.extname(filename || '').toLowerCase()
  if (ext === '.svg') return 'svg'
  if (['.png', '.jpg', '.jpeg', '.gif', '.bmp', '.webp', '.avif'].includes(ext)) {
    return 'image'
  }
  if (['.mp4', '.mov', '.avi', '.mkv'].includes(ext)) return 'video'
  if (['.mp3', '.wav', '.ogg', '.flac'].includes(ext)) return 'audio'
  if (['.json'].includes(ext)) return 'json'
  if (['.ttf', '.otf', '.woff', '.woff2'].includes(ext)) return 'font'
  if (['.glb', '.gltf', '.fbx', '.obj'].includes(ext)) return 'model_3d'
  return 'other'
}

const hasCjk = (value) => /[\u4e00-\u9fff]/.test(value)

const decodeFilename = (name) => {
  if (!name || typeof name !== 'string') return name
  if (hasCjk(name)) return name
  const hasLatin1 = /[\u00a0-\u00ff]/.test(name)
  const looksGarbled = name.includes('\uFFFD') || /[ÃÂ]/.test(name) || hasLatin1
  if (!looksGarbled) return name
  try {
    const decoded = Buffer.from(name, 'latin1').toString('utf8')
    if (!decoded || decoded.includes('\uFFFD')) return name
    if (hasCjk(decoded)) return decoded
    return decoded
  } catch (error) {
    return name
  }
}

const normalizeBaseName = (name) => {
  const trimmed = String(name || '').trim()
  return trimmed || 'asset'
}

const ensureUniqueName = (baseName, existingNames) => {
  const normalized = baseName.toLowerCase()
  if (!existingNames.has(normalized)) return baseName
  let index = 1
  let candidate = `${baseName}(${index})`
  while (existingNames.has(candidate.toLowerCase())) {
    index += 1
    candidate = `${baseName}(${index})`
  }
  return candidate
}

const listFolders = async (projectId) => {
  return DesignAssetFolder.findAll({
    where: { projectId },
    order: [
      ['sortOrder', 'ASC'],
      ['createdAt', 'ASC'],
    ],
  })
}

const createFolder = async (projectId, parentId, name) => {
  return DesignAssetFolder.create({
    projectId,
    parentId: parentId || null,
    name: name.trim(),
  })
}

const renameFolder = async (projectId, folderId, name) => {
  const folder = await DesignAssetFolder.findOne({
    where: { id: folderId, projectId },
  })
  if (!folder) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: '资源文件夹不存在',
    })
  }
  await folder.update({ name: name.trim() })
  return folder
}

const updateFolder = async (projectId, folderId, payload = {}) => {
  const folder = await DesignAssetFolder.findOne({
    where: { id: folderId, projectId },
  })
  if (!folder) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: '资源文件夹不存在',
    })
  }
  const updates = {}
  if (typeof payload.name === 'string' && payload.name.trim()) {
    updates.name = payload.name.trim()
  }
  if (Object.prototype.hasOwnProperty.call(payload, 'parentId')) {
    updates.parentId = payload.parentId || null
  }
  if (Object.keys(updates).length === 0) {
    return folder
  }
  await folder.update(updates)
  return folder
}

const collectFolderIds = (folders, rootId) => {
  const map = new Map()
  folders.forEach((folder) => {
    if (!map.has(folder.parentId)) {
      map.set(folder.parentId, [])
    }
    map.get(folder.parentId).push(folder.id)
  })

  const result = []
  const stack = [rootId]
  while (stack.length) {
    const current = stack.pop()
    result.push(current)
    const children = map.get(current) || []
    children.forEach((childId) => stack.push(childId))
  }
  return result
}

const deleteAssetsByFolderIds = async (projectId, folderIds) => {
  if (!folderIds.length) return
  const assets = await DesignAsset.findAll({
    where: { projectId, folderId: { [Op.in]: folderIds } },
  })
  for (const asset of assets) {
    const objectKey = asset.metadata?.objectKey
    if (objectKey) {
      await storageService.removeObject('design', objectKey)
    }
  }
  await DesignAsset.destroy({
    where: { projectId, folderId: { [Op.in]: folderIds } },
  })
}

const deleteAssetsByProject = async (projectId) => {
  const assets = await DesignAsset.findAll({ where: { projectId } })
  for (const asset of assets) {
    const objectKey = asset.metadata?.objectKey
    if (objectKey) {
      await storageService.removeObject('design', objectKey)
    }
  }
  await DesignAsset.destroy({ where: { projectId } })
}

const deleteFoldersByProject = async (projectId) => {
  await DesignAssetFolder.destroy({ where: { projectId } })
}

const deleteFolder = async (projectId, folderId) => {
  const folder = await DesignAssetFolder.findOne({
    where: { id: folderId, projectId },
  })
  if (!folder) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: '资源文件夹不存在',
    })
  }
  const allFolders = await DesignAssetFolder.findAll({
    where: { projectId },
    attributes: ['id', 'parentId'],
  })
  const folderIds = collectFolderIds(allFolders, folderId)
  await deleteAssetsByFolderIds(projectId, folderIds)
  await DesignAssetFolder.destroy({ where: { id: { [Op.in]: folderIds } } })
}

const listAssets = async (projectId, options = {}) => {
  const { folderId, keyword, type } = options
  const where = { projectId }
  const typeFilter = normalizeTypeFilter(type)
  if (folderId) {
    where.folderId = folderId
  }
  if (typeFilter) {
    where.type = Array.isArray(typeFilter) ? { [Op.in]: typeFilter } : typeFilter
  }
  if (keyword) {
    where.name = { [Op.like]: `%${keyword}%` }
  }

  return DesignAsset.findAll({
    where,
    order: [['createdAt', 'DESC']],
  })
}

const createAssets = async (projectId, folderId, files, userId, options = {}) => {
  if (!Array.isArray(files) || files.length === 0) return []
  const results = []
  const conflictStrategy = String(options.conflictStrategy || 'rename')
  const targetFolderId = folderId || null

  const existingAssets = await DesignAsset.findAll({
    where: { projectId, folderId: targetFolderId },
  })
  const existingByName = new Map()
  const existingNames = new Set()
  existingAssets.forEach((asset) => {
    const key = String(asset.name || '').toLowerCase()
    if (key) {
      existingByName.set(key, asset)
      existingNames.add(key)
    }
  })

  for (const file of files) {
    const decodedOriginalName = decodeFilename(file.originalname || '')
    const parsed = path.parse(decodedOriginalName || 'asset')
    let baseName = normalizeBaseName(parsed.name)
    const ext = parsed.ext || path.extname(file.originalname || '') || ''
    const lowerKey = baseName.toLowerCase()
    const existingAsset = existingByName.get(lowerKey)

    if (existingAsset && conflictStrategy === 'replace') {
      const objectKey =
        existingAsset.metadata?.objectKey || `${projectId}/${existingAsset.id}${ext}`
      const type = detectAssetType(file.mimetype, decodedOriginalName)

      await storageService.uploadObject(
        'design',
        objectKey,
        Readable.from(file.buffer),
        file.size,
        {
          'Content-Type': file.mimetype || 'application/octet-stream',
        },
      )

      await existingAsset.update({
        originalName: decodedOriginalName || existingAsset.originalName,
        mimeType: file.mimetype || existingAsset.mimeType,
        type,
        size: file.size || 0,
        url: buildAssetUrl(projectId, existingAsset.id),
        metadata: {
          ...(existingAsset.metadata || {}),
          objectKey,
          bucket: 'design',
        },
      })

      results.push(existingAsset)
      continue
    }

    if (existingAsset && conflictStrategy === 'rename') {
      baseName = ensureUniqueName(baseName, existingNames)
    }

    const assetId = randomUUID()
    const objectKey = `${projectId}/${assetId}${ext}`
    const type = detectAssetType(file.mimetype, decodedOriginalName)

    await storageService.uploadObject('design', objectKey, Readable.from(file.buffer), file.size, {
      'Content-Type': file.mimetype || 'application/octet-stream',
    })

    const asset = await DesignAsset.create({
      id: assetId,
      projectId,
      folderId: targetFolderId,
      name: baseName,
      originalName: decodedOriginalName || null,
      type,
      mimeType: file.mimetype || null,
      url: buildAssetUrl(projectId, assetId),
      size: file.size || 0,
      metadata: {
        objectKey,
        bucket: 'design',
      },
      uploadedBy: userId,
    })

    existingNames.add(baseName.toLowerCase())
    results.push(asset)
  }

  return results
}

const updateAsset = async (projectId, assetId, payload = {}) => {
  const asset = await DesignAsset.findOne({
    where: { id: assetId, projectId },
  })
  if (!asset) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: '资源文件不存在',
    })
  }
  const updates = {}
  if (typeof payload.name === 'string' && payload.name.trim()) {
    updates.name = payload.name.trim()
  }
  if (Object.prototype.hasOwnProperty.call(payload, 'folderId')) {
    updates.folderId = payload.folderId || null
  }
  if (Object.keys(updates).length === 0) {
    return asset
  }
  await asset.update(updates)
  return asset
}

const copyAsset = async (projectId, assetId, targetFolderId, name, userId) => {
  const asset = await DesignAsset.findOne({
    where: { id: assetId, projectId },
  })
  if (!asset) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: '资源文件不存在',
    })
  }
  const sourceKey = asset.metadata?.objectKey
  if (!sourceKey) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: '资源文件不存在',
    })
  }
  const newId = randomUUID()
  const ext = path.extname(asset.originalName || asset.name || '') || ''
  const targetKey = `${projectId}/${newId}${ext}`
  await storageService.copyObject('design', sourceKey, targetKey)

  const nextName = (name && name.trim()) || `${asset.name}_副本`
  const newAsset = await DesignAsset.create({
    id: newId,
    projectId,
    folderId: targetFolderId || null,
    name: nextName,
    originalName: asset.originalName || asset.name,
    type: asset.type,
    mimeType: asset.mimeType,
    url: buildAssetUrl(projectId, newId),
    size: asset.size || 0,
    width: asset.width,
    height: asset.height,
    duration: asset.duration,
    metadata: {
      ...(asset.metadata || {}),
      objectKey: targetKey,
      bucket: 'design',
    },
    tags: asset.tags,
    usageCount: 0,
    uploadedBy: userId,
  })

  return newAsset
}

const deleteAsset = async (projectId, assetId) => {
  const asset = await DesignAsset.findOne({
    where: { id: assetId, projectId },
  })
  if (!asset) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: '资源文件不存在',
    })
  }
  const objectKey = asset.metadata?.objectKey
  if (objectKey) {
    await storageService.removeObject('design', objectKey)
  }
  await asset.destroy()
}

const getAssetFile = async (projectId, assetId) => {
  const asset = await DesignAsset.findOne({
    where: { id: assetId, projectId },
  })
  if (!asset) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: '资源文件不存在',
    })
  }
  const objectKey = asset.metadata?.objectKey
  if (!objectKey) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: '资源文件未找到',
    })
  }

  const stream = await storageService.getObjectStream('design', objectKey)
  return { asset, stream }
}

module.exports = {
  listFolders,
  createFolder,
  renameFolder,
  updateFolder,
  deleteFolder,
  listAssets,
  createAssets,
  updateAsset,
  copyAsset,
  deleteAsset,
  deleteAssetsByProject,
  deleteFoldersByProject,
  getAssetFile,
}
