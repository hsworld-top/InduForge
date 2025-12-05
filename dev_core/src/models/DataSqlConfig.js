'use strict';
const { Model } = require('sequelize');

module.exports = (sequelize, DataTypes) => {
  class DataSqlConfig extends Model {
    static associate(models) {
      // 与数据查询关联
      DataSqlConfig.belongsTo(models.DataQuery, {
        foreignKey: 'queryId',
        as: 'query'
      });
    }
  }

  DataSqlConfig.init({
    queryId: {
      type: DataTypes.CHAR(36),
      primaryKey: true,
      comment: '查询ID'
    },
    sql: {
      type: DataTypes.TEXT,
      allowNull: false,
      comment: 'SQL语句（使用?占位符）'
    },
    parameters: {
      type: DataTypes.JSON,
      comment: '参数定义',
      defaultValue: null
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
