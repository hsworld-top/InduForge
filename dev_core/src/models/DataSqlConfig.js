'use strict';
const { Model } = require('sequelize');

module.exports = (sequelize, DataTypes) => {
  class DataSqlConfig extends Model {
    static associate(models) {
      // 与数据连接关联
      DataSqlConfig.belongsTo(models.DataConnection, {
        foreignKey: 'connectionId',
        as: 'connection'
      });

      // 与用户关联（创建者）
      DataSqlConfig.belongsTo(models.User, {
        foreignKey: 'createdBy',
        as: 'creator'
      });

      // 与用户关联（更新者）
      DataSqlConfig.belongsTo(models.User, {
        foreignKey: 'updatedBy',
        as: 'updater'
      });
    }
  }

  DataSqlConfig.init({
    id: {
      type: DataTypes.CHAR(36),
      primaryKey: true,
      defaultValue: DataTypes.UUIDV4
    },
    connectionId: {
      type: DataTypes.CHAR(36),
      allowNull: false,
      comment: '连接ID'
    },
    name: {
      type: DataTypes.STRING(100),
      allowNull: false,
      comment: 'SQL配置名称'
    },
    description: {
      type: DataTypes.TEXT,
      comment: '描述'
    },
    sqlStatement: {
      type: DataTypes.TEXT,
      allowNull: false,
      comment: 'SQL语句'
    },
    parameters: {
      type: DataTypes.JSON,
      comment: '参数定义',
      defaultValue: null
    },
    isEnabled: {
      type: DataTypes.BOOLEAN,
      allowNull: false,
      defaultValue: true,
      comment: '是否启用'
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
    modelName: 'DataSqlConfig',
    tableName: 'data_sql_configs',
    timestamps: true,
    paranoid: false,
    createdAt: 'createdAt',
    updatedAt: 'updatedAt'
  });

  return DataSqlConfig;
};
