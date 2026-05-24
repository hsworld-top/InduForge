const { DataTypes } = require('sequelize')
const { sequelize } = require('../config/database')

const DesignAsset = sequelize.define(
  'DesignAsset',
  {
    id: {
      type: DataTypes.UUID,
      primaryKey: true,
      defaultValue: DataTypes.UUIDV4,
    },
    projectId: {
      type: DataTypes.UUID,
      allowNull: false,
    },
    folderId: {
      type: DataTypes.UUID,
      allowNull: true,
    },
    name: {
      type: DataTypes.STRING(200),
      allowNull: false,
    },
    originalName: {
      type: DataTypes.STRING(200),
      allowNull: true,
    },
    type: {
      type: DataTypes.ENUM('image', 'svg', 'video', 'audio', 'model_3d', 'font', 'json', 'other'),
      allowNull: false,
    },
    mimeType: {
      type: DataTypes.STRING(100),
      allowNull: true,
    },
    url: {
      type: DataTypes.STRING(1000),
      allowNull: false,
    },
    thumbnailUrl: {
      type: DataTypes.STRING(1000),
      allowNull: true,
    },
    size: {
      type: DataTypes.BIGINT,
      allowNull: true,
      defaultValue: 0,
    },
    width: {
      type: DataTypes.INTEGER,
      allowNull: true,
    },
    height: {
      type: DataTypes.INTEGER,
      allowNull: true,
    },
    duration: {
      type: DataTypes.INTEGER,
      allowNull: true,
    },
    metadata: {
      type: DataTypes.JSON,
      allowNull: true,
    },
    tags: {
      type: DataTypes.JSON,
      allowNull: true,
    },
    usageCount: {
      type: DataTypes.INTEGER,
      allowNull: true,
      defaultValue: 0,
    },
    uploadedBy: {
      type: DataTypes.UUID,
      allowNull: false,
    },
  },
  {
    tableName: 'design_assets',
    timestamps: true,
    createdAt: 'createdAt',
    updatedAt: 'updatedAt',
  },
)

module.exports = DesignAsset
