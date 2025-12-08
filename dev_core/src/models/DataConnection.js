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

      // 与SQL配置关联
      DataConnection.hasMany(models.DataSqlConfig, {
        foreignKey: 'connectionId',
        as: 'sqlConfigs'
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
      type: DataTypes.ENUM('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7'),
      allowNull: false,
      comment: '连接类型'
    },
    category: {
      type: DataTypes.ENUM('database', 'message', 'protocol', 'api'),
      allowNull: false,
      defaultValue: 'api',
      comment: '连接类别'
    },
    status: {
      type: DataTypes.ENUM('connected', 'disconnected', 'error', 'unknown'),
      allowNull: false,
      defaultValue: 'unknown',
      comment: '连接状态'
    },
    isEnabled: {
      type: DataTypes.BOOLEAN,
      allowNull: false,
      defaultValue: true,
      comment: '是否启用'
    },
    retryCount: {
      type: DataTypes.INTEGER,
      allowNull: false,
      defaultValue: 3,
      comment: '重连次数'
    },
    retryInterval: {
      type: DataTypes.INTEGER,
      allowNull: false,
      defaultValue: 5000,
      comment: '重连间隔(ms)'
    },
    healthCheckInterval: {
      type: DataTypes.INTEGER,
      defaultValue: 30000,
      comment: '健康检查间隔(ms)'
    },
    lastConnectedAt: {
      type: DataTypes.DATE,
      comment: '最后连接时间'
    },
    lastErrorMessage: {
      type: DataTypes.TEXT,
      comment: '最后错误信息'
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
