/**
 * DesignPage Model - 设计页面模型
 * 存储设计中心的页面数据，包括页面Schema
 * Requirements: 7.1, 7.2, 7.3
 */
const { DataTypes } = require('sequelize');
const { sequelize } = require('../config/database');

const DesignPage = sequelize.define('DesignPage', {
  id: {
    type: DataTypes.UUID,
    defaultValue: DataTypes.UUIDV4,
    primaryKey: true,
    comment: '页面唯一标识 (UUID v4)',
  },
  projectId: {
    type: DataTypes.UUID,
    allowNull: false,
    references: {
      model: 'projects',
      key: 'id',
    },
    comment: '所属工程ID',
  },
  parentId: {
    type: DataTypes.UUID,
    allowNull: true,
    references: {
      model: 'design_pages',
      key: 'id',
    },
    comment: '父页面/文件夹ID，用于层级结构',
  },
  name: {
    type: DataTypes.STRING(200),
    allowNull: false,
    comment: '页面名称',
  },
  type: {
    type: DataTypes.ENUM('page', 'folder', 'dialog'),
    defaultValue: 'page',
    allowNull: false,
    comment: '类型：page-页面, folder-文件夹, dialog-对话框',
  },
  schemaContent: {
    type: DataTypes.JSON,
    allowNull: true,
    comment: '完整的 Page Schema (DSL JSON)',
  },
  sortOrder: {
    type: DataTypes.INTEGER,
    defaultValue: 0,
    allowNull: false,
    comment: '排序顺序',
  },
  lockedBy: {
    type: DataTypes.UUID,
    allowNull: true,
    references: {
      model: 'users',
      key: 'id',
    },
    comment: '编辑锁定者ID',
  },
  lockedAt: {
    type: DataTypes.DATE,
    allowNull: true,
    comment: '锁定时间',
  },
  createdBy: {
    type: DataTypes.UUID,
    allowNull: false,
    references: {
      model: 'users',
      key: 'id',
    },
    comment: '创建者ID',
  },
  updatedBy: {
    type: DataTypes.UUID,
    allowNull: true,
    references: {
      model: 'users',
      key: 'id',
    },
    comment: '更新者ID',
  },
  createdAt: {
    type: DataTypes.DATE,
    allowNull: false,
  },
  updatedAt: {
    type: DataTypes.DATE,
    allowNull: false,
  },
}, {
  tableName: 'design_pages',
  comment: '设计页面表',
  indexes: [
    {
      fields: ['projectId'],
      name: 'idx_design_pages_project',
    },
    {
      fields: ['parentId'],
      name: 'idx_design_pages_parent',
    },
    {
      fields: ['projectId', 'parentId', 'sortOrder'],
      name: 'idx_design_pages_sort',
    },
    {
      fields: ['type'],
      name: 'idx_design_pages_type',
    },
    {
      fields: ['lockedBy'],
      name: 'idx_design_pages_locked',
    },
  ],
});

module.exports = DesignPage;
