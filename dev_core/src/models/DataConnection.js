'use strict';
const { Model } = require('sequelize');

module.exports = (sequelize, DataTypes) => {
  class DataConnection extends Model {
    static associate(models) {
      // 与工程关联
      DataConnection.belongsTo(models.Project, {
        foreignKey: 'projectId',
        as: 'project'
      });

      // 与用户关联（创建者）
      DataConnection.belongsTo(models.User, {
        foreignKey: 'createdBy',
        as: 'creator'
      });

      // 与用户关联（更新者）
      DataConnection.belongsTo(models.User, {
        foreignKey: 'updatedBy',
        as: 'updater'
      });

      // 与数据查询关联
      DataConnection.hasMany(models.DataQuery, {
        foreignKey: 'connectionId',
        as: 'queries'
      });

      // 与关系库配置关联
      DataConnection.hasOne(models.DataRelationalConfig, {
        foreignKey: 'connectionId',
        as: 'relationalConfig'
      });
    }
  }

  DataConnection.init({
    id: {
      type: DataTypes.CHAR(36),
      primaryKey: true,
      defaultValue: DataTypes.UUIDV4
    },
    projectId: {
      type: DataTypes.CHAR(36),
      allowNull: false,
      comment: '所属工程ID'
    },
    name: {
      type: DataTypes.STRING(100),
      allowNull: false,
      comment: '连接名称'
    },
    type: {
      type: DataTypes.ENUM('relational', 'mqtt', 'websocket', 'opcua', 'http'),
      allowNull: false,
      comment: '连接类型'
    },
    category: {
      type: DataTypes.ENUM('internal', 'external'),
      allowNull: false,
      defaultValue: 'external',
      comment: '连接类别'
    },
    status: {
      type: DataTypes.ENUM('active', 'inactive', 'error'),
      allowNull: false,
      defaultValue: 'active',
      comment: '连接状态'
    },
    lastConnected: {
      type: DataTypes.DATE,
      comment: '最后连接时间'
    },
    createdBy: {
      type: DataTypes.CHAR(36),
      allowNull: false,
      comment: '创建者ID'
    },
    updatedBy: {
      type: DataTypes.CHAR(36),
      comment: '更新者ID'
    }
  }, {
    sequelize,
    modelName: 'DataConnection',
    tableName: 'data_connections',
    timestamps: true,
    paranoid: false,
    indexes: [
      {
        fields: ['projectId', 'type'],
        name: 'data_connections_project_type_idx'
      },
      {
        fields: ['status'],
        name: 'data_connections_status_idx'
      }
    ]
  });

  return DataConnection;
};
