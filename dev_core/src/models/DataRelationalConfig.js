'use strict';
const { Model } = require('sequelize');

module.exports = (sequelize, DataTypes) => {
  class DataRelationalConfig extends Model {
    static associate(models) {
      // 与数据连接关联
      DataRelationalConfig.belongsTo(models.DataConnection, {
        foreignKey: 'connectionId',
        as: 'connection'
      });
    }
  }

  DataRelationalConfig.init({
    id: {
      type: DataTypes.CHAR(36),
      primaryKey: true,
      defaultValue: DataTypes.UUIDV4
    },
    connectionId: {
      type: DataTypes.CHAR(36),
      allowNull: false,
      unique: true,
      comment: '连接ID'
    },
    dbType: {
      type: DataTypes.ENUM('mysql', 'postgresql', 'sqlserver', 'oracle', 'sqlite'),
      allowNull: false,
      comment: '数据库类型'
    },
    host: {
      type: DataTypes.STRING(255),
      allowNull: false,
      comment: '主机地址'
    },
    port: {
      type: DataTypes.INTEGER,
      allowNull: false,
      comment: '端口号'
    },
    database: {
      type: DataTypes.STRING(100),
      allowNull: false,
      comment: '数据库名'
    },
    username: {
      type: DataTypes.STRING(100),
      allowNull: false,
      comment: '用户名'
    },
    password: {
      type: DataTypes.TEXT,
      allowNull: false,
      comment: '密码（加密存储）'
    },
    charset: {
      type: DataTypes.STRING(50),
      defaultValue: 'utf8mb4',
      comment: '字符集'
    },
    connectionLimit: {
      type: DataTypes.INTEGER,
      defaultValue: 10,
      comment: '连接池大小'
    },
    acquireTimeout: {
      type: DataTypes.INTEGER,
      defaultValue: 60000,
      comment: '获取连接超时(ms)'
    },
    timeout: {
      type: DataTypes.INTEGER,
      defaultValue: 60000,
      comment: '查询超时(ms)'
    }
  }, {
    sequelize,
    modelName: 'DataRelationalConfig',
    tableName: 'data_relational_configs',
    timestamps: false,
    paranoid: false
  });

  return DataRelationalConfig;
};
