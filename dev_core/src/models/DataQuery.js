'use strict';
const { Model } = require('sequelize');

module.exports = (sequelize, DataTypes) => {
  class DataQuery extends Model {
    static associate(models) {
      // 与工程关联
      DataQuery.belongsTo(models.Project, {
        foreignKey: 'projectId',
        as: 'project'
      });

      // 与数据连接关联
      DataQuery.belongsTo(models.DataConnection, {
        foreignKey: 'connectionId',
        as: 'connection'
      });

      // 与用户关联（创建者）
      DataQuery.belongsTo(models.User, {
        foreignKey: 'createdBy',
        as: 'creator'
      });

      // 与用户关联（更新者）
      DataQuery.belongsTo(models.User, {
        foreignKey: 'updatedBy',
        as: 'updater'
      });

      // 与SQL配置关联
      DataQuery.hasOne(models.DataSqlConfig, {
        foreignKey: 'queryId',
        as: 'sqlConfig'
      });

      // 与查询日志关联
      DataQuery.hasMany(models.DataQueryLog, {
        foreignKey: 'queryId',
        as: 'logs'
      });
    }
  }

  DataQuery.init({
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
    connectionId: {
      type: DataTypes.CHAR(36),
      allowNull: false,
      comment: '数据连接ID'
    },
    name: {
      type: DataTypes.STRING(200),
      allowNull: false,
      comment: '查询名称'
    },
    description: {
      type: DataTypes.TEXT,
      comment: '查询描述'
    },
    category: {
      type: DataTypes.STRING(100),
      comment: '查询分类'
    },
    queryType: {
      type: DataTypes.ENUM('sql', 'http_get', 'http_post', 'mqtt_publish', 'mqtt_subscribe', 'websocket', 'opcua_read', 'opcua_write'),
      allowNull: false,
      comment: '查询类型'
    },
    isActive: {
      type: DataTypes.BOOLEAN,
      allowNull: false,
      defaultValue: true,
      comment: '是否激活'
    },
    cacheEnabled: {
      type: DataTypes.BOOLEAN,
      allowNull: false,
      defaultValue: false,
      comment: '是否启用缓存'
    },
    cacheTtl: {
      type: DataTypes.INTEGER,
      defaultValue: 300,
      comment: '缓存过期时间(秒)'
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
    modelName: 'DataQuery',
    tableName: 'data_queries',
    timestamps: true,
    paranoid: false,
    indexes: [
      {
        fields: ['projectId', 'connectionId'],
        name: 'data_queries_project_connection_idx'
      },
      {
        fields: ['queryType', 'isActive'],
        name: 'data_queries_type_active_idx'
      },
      {
        fields: ['projectId', 'name'],
        name: 'data_queries_project_name_uq',
        unique: true
      }
    ]
  });

  return DataQuery;
};
