const { DataTypes } = require('sequelize')
const { sequelize } = require('../config/database')

const DesignAssetFolder = sequelize.define(
  'DesignAssetFolder',
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
    parentId: {
      type: DataTypes.UUID,
      allowNull: true,
    },
    name: {
      type: DataTypes.STRING(100),
      allowNull: false,
    },
    sortOrder: {
      type: DataTypes.INTEGER,
      allowNull: true,
      defaultValue: 0,
    },
  },
  {
    tableName: 'design_asset_folders',
    timestamps: true,
    createdAt: 'createdAt',
    updatedAt: 'updatedAt',
  },
)

module.exports = DesignAssetFolder
