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
      defaultValue: DataTypes.UUIDV4,
      comment: '主键ID'
    },
    connectionId: {
      type: DataTypes.CHAR(36),
      allowNull: false,
      unique: true,
      comment: '连接ID'
    },
    dbType: {
      type: DataTypes.ENUM('mysql', 'postgresql', 'sqlserver', 'oracle', 'sqlite', 'clickhouse'),
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
      comment: '密码(加密存储)'
    },
    schema: {
      type: DataTypes.STRING(100),
      comment: 'Schema名(PostgreSQL/Oracle)'
    },
    charset: {
      type: DataTypes.STRING(50),
      defaultValue: 'utf8mb4',
      comment: '字符集'
    },
    timezone: {
      type: DataTypes.STRING(50),
      comment: '时区'
    },
    ssl: {
      type: DataTypes.BOOLEAN,
      defaultValue: false,
      comment: '是否启用SSL'
    },
    sslConfig: {
      type: DataTypes.JSON,
      comment: 'SSL配置'
    },
    poolMin: {
      type: DataTypes.INTEGER,
      defaultValue: 2,
      comment: '连接池最小连接数'
    },
    poolMax: {
      type: DataTypes.INTEGER,
      defaultValue: 10,
      comment: '连接池最大连接数'
    },
    acquireTimeout: {
      type: DataTypes.INTEGER,
      defaultValue: 60000,
      comment: '获取连接超时(ms)'
    },
    idleTimeout: {
      type: DataTypes.INTEGER,
      defaultValue: 30000,
      comment: '空闲超时(ms)'
    },
    queryTimeout: {
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
