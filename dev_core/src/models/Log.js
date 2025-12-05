const { DataTypes } = require('sequelize');
const { sequelize } = require('../config/database');

const Log = sequelize.define('Log', {
  id: {
    type: DataTypes.UUID,
    defaultValue: DataTypes.UUIDV4,
    primaryKey: true,
  },
  level: {
    type: DataTypes.ENUM('info', 'warning', 'error', 'debug'),
    defaultValue: 'info',
    comment: '日志级别',
  },
  message: {
    type: DataTypes.TEXT,
    allowNull: false,
    comment: '日志消息',
  },
  action: {
    type: DataTypes.STRING(100),
    allowNull: true,
    comment: '操作类型',
  },
  resource: {
    type: DataTypes.STRING(100),
    allowNull: true,
    comment: '资源类型',
  },
  resourceId: {
    type: DataTypes.UUID,
    allowNull: true,
    comment: '资源ID',
  },
  userId: {
    type: DataTypes.UUID,
    allowNull: true,
    references: {
      model: 'users',
      key: 'id',
    },
    comment: '操作用户ID',
  },
  tenantId: {
    type: DataTypes.UUID,
    allowNull: true,
    references: {
      model: 'tenants',
      key: 'id',
    },
    comment: '租户ID',
  },
  ip: {
    type: DataTypes.STRING(45),
    allowNull: true,
    comment: 'IP地址',
  },
  userAgent: {
    type: DataTypes.TEXT,
    allowNull: true,
    comment: '用户代理',
  },
  metadata: {
    type: DataTypes.JSON,
    allowNull: true,
    comment: '额外元数据',
  },
  createdAt: {
    type: DataTypes.DATE,
    allowNull: false,
  },
}, {
  tableName: 'logs',
  comment: '系统日志表',
  indexes: [
    {
      fields: ['level'],
    },
    {
      fields: ['action'],
    },
    {
      fields: ['userId'],
    },
    {
      fields: ['tenantId'],
    },
    {
      fields: ['createdAt'],
    },
  ],
});

module.exports = Log;
