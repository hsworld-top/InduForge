// 导入 Sequelize 和数据库连接
const { Sequelize } = require('sequelize');
const { sequelize } = require('../config/database');

//===========================================
// 导入所有模型
//===========================================
const Tenant = require('./Tenant'); // 租户模型
const User = require('./User'); // 用户模型
const Project = require('./Project'); // 工程模型
const Log = require('./Log'); // 日志模型 
const DataConnectionFn = require('./DataConnection'); // 数据连接模型
const DataRelationalConfigFn = require('./DataRelationalConfig'); // 关系库配置模型
const DataQueryFn = require('./DataQuery'); // 数据查询模型
const DataSqlConfigFn = require('./DataSqlConfig'); // SQL配置模型
const DataQueryLogFn = require('./DataQueryLog'); // 查询日志模型
const DesignPage = require('./DesignPage'); // 设计页面模型

//===========================================
// 初始化数据中心模型
//===========================================
const DataConnection = DataConnectionFn(sequelize, Sequelize.DataTypes); // 数据连接模型
const DataRelationalConfig = DataRelationalConfigFn(sequelize, Sequelize.DataTypes); // 关系库配置模型
const DataQuery = DataQueryFn(sequelize, Sequelize.DataTypes); // 数据查询模型
const DataSqlConfig = DataSqlConfigFn(sequelize, Sequelize.DataTypes); // SQL配置模型
const DataQueryLog = DataQueryLogFn(sequelize, Sequelize.DataTypes); // 查询日志模型

//===========================================
// 定义模型关联关系
//===========================================

// 租户和用户：一对多
Tenant.hasMany(User, {
  foreignKey: 'tenantId',
  as: 'users',
  onDelete: 'CASCADE',
});

User.belongsTo(Tenant, {
  foreignKey: 'tenantId',
  as: 'tenant',
});

// 租户和工程：一对多
Tenant.hasMany(Project, {
  foreignKey: 'tenantId',
  as: 'projects',
  onDelete: 'CASCADE',
});

Project.belongsTo(Tenant, {
  foreignKey: 'tenantId',
  as: 'tenant',
});

// 用户和工程：多对多（创建者和更新者）
User.hasMany(Project, {
  foreignKey: 'createdBy',
  as: 'createdProjects',
});

User.hasMany(Project, {
  foreignKey: 'updatedBy',
  as: 'updatedProjects',
});

Project.belongsTo(User, {
  foreignKey: 'createdBy',
  as: 'creator',
});

Project.belongsTo(User, {
  foreignKey: 'updatedBy',
  as: 'updater',
});


// 租户和日志：一对多
Tenant.hasMany(Log, {
  foreignKey: 'tenantId',
  as: 'logs',
  onDelete: 'CASCADE',
});

Log.belongsTo(Tenant, {
  foreignKey: 'tenantId',
  as: 'tenant',
});

// 用户和日志：一对多
User.hasMany(Log, {
  foreignKey: 'userId',
  as: 'logs',
});

Log.belongsTo(User, {
  foreignKey: 'userId',
  as: 'user',
});

// ===========================================
// 数据中心模型关联关系
// ===========================================

// 工程和数据连接：一对多
Project.hasMany(DataConnection, {
  foreignKey: 'projectId',
  as: 'dataConnections',
  onDelete: 'CASCADE',
});

DataConnection.belongsTo(Project, {
  foreignKey: 'projectId',
  as: 'project',
});

// 用户和数据连接：多对多（创建者和更新者）
User.hasMany(DataConnection, {
  foreignKey: 'createdBy',
  as: 'createdDataConnections',
});

User.hasMany(DataConnection, {
  foreignKey: 'updatedBy',
  as: 'updatedDataConnections',
});

DataConnection.belongsTo(User, {
  foreignKey: 'createdBy',
  as: 'creator',
});

DataConnection.belongsTo(User, {
  foreignKey: 'updatedBy',
  as: 'updater',
});

// 数据连接和关系库配置：一对一
DataConnection.hasOne(DataRelationalConfig, {
  foreignKey: 'connectionId',
  as: 'relationalConfig',
});

DataRelationalConfig.belongsTo(DataConnection, {
  foreignKey: 'connectionId',
  as: 'connection',
});

// 数据连接和数据查询：一对多
DataConnection.hasMany(DataQuery, {
  foreignKey: 'connectionId',
  as: 'queries',
});

DataQuery.belongsTo(DataConnection, {
  foreignKey: 'connectionId',
  as: 'connection',
});

// 工程和数据查询：一对多
Project.hasMany(DataQuery, {
  foreignKey: 'projectId',
  as: 'dataQueries',
});

DataQuery.belongsTo(Project, {
  foreignKey: 'projectId',
  as: 'project',
});

// 用户和数据查询：多对多（创建者和更新者）
User.hasMany(DataQuery, {
  foreignKey: 'createdBy',
  as: 'createdDataQueries',
});

User.hasMany(DataQuery, {
  foreignKey: 'updatedBy',
  as: 'updatedDataQueries',
});

DataQuery.belongsTo(User, {
  foreignKey: 'createdBy',
  as: 'creator',
});

DataQuery.belongsTo(User, {
  foreignKey: 'updatedBy',
  as: 'updater',
});

// 数据查询和查询日志：一对多
DataQuery.hasMany(DataQueryLog, {
  foreignKey: 'queryId',
  as: 'logs',
});

DataQueryLog.belongsTo(DataQuery, {
  foreignKey: 'queryId',
  as: 'query',
});

// 数据连接和查询日志：一对多
DataConnection.hasMany(DataQueryLog, {
  foreignKey: 'connectionId',
  as: 'queryLogs',
});

DataQueryLog.belongsTo(DataConnection, {
  foreignKey: 'connectionId',
  as: 'connection',
});

// 用户和查询日志：一对多
User.hasMany(DataQueryLog, {
  foreignKey: 'executedBy',
  as: 'dataQueryLogs',
});

DataQueryLog.belongsTo(User, {
  foreignKey: 'executedBy',
  as: 'executor',
});

// ===========================================
// 设计中心模型关联关系
// ===========================================

// 工程和设计页面：一对多
Project.hasMany(DesignPage, {
  foreignKey: 'projectId',
  as: 'designPages',
  onDelete: 'CASCADE',
});

DesignPage.belongsTo(Project, {
  foreignKey: 'projectId',
  as: 'project',
});

// 设计页面自关联：父子关系（文件夹结构）
DesignPage.hasMany(DesignPage, {
  foreignKey: 'parentId',
  as: 'children',
  onDelete: 'RESTRICT', // 防止删除有子页面的文件夹
});

DesignPage.belongsTo(DesignPage, {
  foreignKey: 'parentId',
  as: 'parent',
});

// 用户和设计页面：创建者
User.hasMany(DesignPage, {
  foreignKey: 'createdBy',
  as: 'createdDesignPages',
});

DesignPage.belongsTo(User, {
  foreignKey: 'createdBy',
  as: 'creator',
});

// 用户和设计页面：更新者
User.hasMany(DesignPage, {
  foreignKey: 'updatedBy',
  as: 'updatedDesignPages',
});

DesignPage.belongsTo(User, {
  foreignKey: 'updatedBy',
  as: 'updater',
});

// 用户和设计页面：锁定者
User.hasMany(DesignPage, {
  foreignKey: 'lockedBy',
  as: 'lockedDesignPages',
});

DesignPage.belongsTo(User, {
  foreignKey: 'lockedBy',
  as: 'locker',
});


module.exports = {
  Tenant, // 租户模型
  User, // 用户模型
  Project, // 工程模型
  Log, // 日志模型  
  DataConnection, // 数据连接模型
  DataRelationalConfig, // 关系库配置模型
  DataQuery, // 数据查询模型
  DataSqlConfig, // SQL配置模型
  DataQueryLog, // 查询日志模型
  DesignPage, // 设计页面模型
};
