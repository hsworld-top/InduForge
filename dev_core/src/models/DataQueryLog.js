'use strict';
const { Model } = require('sequelize');

module.exports = (sequelize, DataTypes) => {
  class DataQueryLog extends Model {
    static associate(models) {
      // 与数据查询关联
      DataQueryLog.belongsTo(models.DataQuery, {
        foreignKey: 'queryId',
        as: 'query'
      });

      // 与数据连接关联
      DataQueryLog.belongsTo(models.DataConnection, {
        foreignKey: 'connectionId',
        as: 'connection'
      });

      // 与用户关联（执行者）
      DataQueryLog.belongsTo(models.User, {
        foreignKey: 'executedBy',
        as: 'executor'
      });
    }
  }

  DataQueryLog.init({
    id: {
      type: DataTypes.CHAR(36),
      primaryKey: true,
      defaultValue: DataTypes.UUIDV4
    },
    queryId: {
      type: DataTypes.CHAR(36),
      allowNull: false,
      comment: '查询ID'
    },
    connectionId: {
      type: DataTypes.CHAR(36),
      allowNull: false,
      comment: '连接ID'
    },
    executedBy: {
      type: DataTypes.CHAR(36),
      allowNull: false,
      comment: '执行者ID'
    },
    parameters: {
      type: DataTypes.JSON,
      comment: '执行参数',
      defaultValue: null
    },
    executionTime: {
      type: DataTypes.INTEGER,
      allowNull: false,
      comment: '执行时间(ms)'
    },
    resultCount: {
      type: DataTypes.INTEGER,
      comment: '结果行数'
    },
    status: {
      type: DataTypes.ENUM('success', 'error', 'timeout'),
      allowNull: false,
      comment: '执行状态'
    },
    errorMessage: {
      type: DataTypes.TEXT,
      comment: '错误信息'
    },
    executedAt: {
      type: DataTypes.DATE,
      allowNull: false,
      comment: '执行时间'
    }
  }, {
    sequelize,
    modelName: 'DataQueryLog',
    tableName: 'data_query_logs',
    timestamps: false,
    paranoid: false,
    indexes: [
      {
        fields: ['queryId', 'executedAt'],
        name: 'data_query_logs_query_time_idx'
      },
      {
        fields: ['status'],
        name: 'data_query_logs_status_idx'
      }
    ]
  });

  return DataQueryLog;
};
