// 导入 Sequelize 和数据库连接
const { Sequelize } = require("sequelize");
const { sequelize } = require("../config/database");

//===========================================
// 导入所有模型
//===========================================
const Tenant = require("./Tenant"); // 租户模型
const User = require("./User"); // 用户模型
const Project = require("./Project"); // 工程模型
const Log = require("./Log"); // 日志模型
const DataConnectionFn = require("./DataConnection"); // 数据连接模型
const DataRelationalConfigFn = require("./DataRelationalConfig"); // 关系库配置模型
const DataMqttConfigFn = require("./DataMqttConfig"); // MQTT配置模型
const DataMqttSubscriptionFn = require("./DataMqttSubscription"); // MQTT订阅模型
const DataMqttTagGroupFn = require("./DataMqttTagGroup"); // MQTT变量组模型
const DataMqttTagFn = require("./DataMqttTag"); // MQTT变量模型
// const DataMqttTagValueFn = require("./DataMqttTagValue"); // MQTT变量值模型 - 已废弃，改用 Redis
const DataQueryFn = require("./DataQuery"); // 数据查询模型
const DataSqlConfigFn = require("./DataSqlConfig"); // SQL配置模型
const DataQueryLogFn = require("./DataQueryLog"); // 查询日志模型
const DesignPage = require("./DesignPage"); // 设计页面模型
const DesignAssetFolder = require("./DesignAssetFolder"); // 资源文件夹模型
const DesignAsset = require("./DesignAsset"); // 资源文件模型
const DataPointFn = require("./DataPoint"); // 数据点模型

// 运维模块模型
const NodeFn = require("./Node"); // 节点模型
const NodeDeploymentFn = require("./NodeDeployment"); // 节点部署关系模型
const DeploymentFn = require("./Deployment"); // 发布版本模型
const NodeCommandFn = require("./NodeCommand"); // 节点命令模型

//===========================================
// 初始化数据中心模型
//===========================================
const DataConnection = DataConnectionFn(sequelize, Sequelize.DataTypes); // 数据连接模型
const DataRelationalConfig = DataRelationalConfigFn(
  sequelize,
  Sequelize.DataTypes
); // 关系库配置模型
const DataMqttConfig = DataMqttConfigFn(sequelize, Sequelize.DataTypes); // MQTT配置模型
const DataMqttSubscription = DataMqttSubscriptionFn(
  sequelize,
  Sequelize.DataTypes
); // MQTT订阅模型
const DataMqttTagGroup = DataMqttTagGroupFn(sequelize); // MQTT变量组模型
const DataMqttTag = DataMqttTagFn(sequelize, Sequelize.DataTypes); // MQTT变量模型
// const DataMqttTagValue = DataMqttTagValueFn(sequelize, Sequelize.DataTypes); // MQTT变量值模型 - 已废弃，改用 Redis
const DataQuery = DataQueryFn(sequelize, Sequelize.DataTypes); // 数据查询模型
const DataSqlConfig = DataSqlConfigFn(sequelize, Sequelize.DataTypes); // SQL配置模型
const DataQueryLog = DataQueryLogFn(sequelize, Sequelize.DataTypes); // 查询日志模型
const DataPoint = DataPointFn(sequelize, Sequelize.DataTypes); // 数据点模型

// 运维模块模型初始化
const Node = NodeFn(sequelize, Sequelize.DataTypes); // 节点模型
const NodeDeployment = NodeDeploymentFn(sequelize, Sequelize.DataTypes); // 节点部署关系模型
const Deployment = DeploymentFn(sequelize, Sequelize.DataTypes); // 发布版本模型
const NodeCommand = NodeCommandFn(sequelize, Sequelize.DataTypes); // 节点命令模型

//===========================================
// 定义模型关联关系
//===========================================

// 租户和用户：一对多
Tenant.hasMany(User, {
  foreignKey: "tenantId",
  as: "users",
  onDelete: "CASCADE",
});

User.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

// 租户和工程：一对多
Tenant.hasMany(Project, {
  foreignKey: "tenantId",
  as: "projects",
  onDelete: "CASCADE",
});

Project.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

// 用户和工程：多对多（创建者和更新者）
User.hasMany(Project, {
  foreignKey: "createdBy",
  as: "createdProjects",
});

User.hasMany(Project, {
  foreignKey: "updatedBy",
  as: "updatedProjects",
});

Project.belongsTo(User, {
  foreignKey: "createdBy",
  as: "creator",
});

Project.belongsTo(User, {
  foreignKey: "updatedBy",
  as: "updater",
});

// 租户和日志：一对多
Tenant.hasMany(Log, {
  foreignKey: "tenantId",
  as: "logs",
  onDelete: "CASCADE",
});

Log.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

// 用户和日志：一对多
User.hasMany(Log, {
  foreignKey: "userId",
  as: "logs",
});

Log.belongsTo(User, {
  foreignKey: "userId",
  as: "user",
});

// ===========================================
// 数据中心模型关联关系
// ===========================================

// 工程和数据连接：一对多
Project.hasMany(DataConnection, {
  foreignKey: "projectId",
  as: "dataConnections",
  onDelete: "CASCADE",
});

DataConnection.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 用户和数据连接：多对多（创建者和更新者）
User.hasMany(DataConnection, {
  foreignKey: "createdBy",
  as: "createdDataConnections",
});

User.hasMany(DataConnection, {
  foreignKey: "updatedBy",
  as: "updatedDataConnections",
});

DataConnection.belongsTo(User, {
  foreignKey: "createdBy",
  as: "creator",
});

DataConnection.belongsTo(User, {
  foreignKey: "updatedBy",
  as: "updater",
});

// 数据连接和关系库配置：一对一
DataConnection.hasOne(DataRelationalConfig, {
  foreignKey: "connectionId",
  as: "relationalConfig",
});

DataRelationalConfig.belongsTo(DataConnection, {
  foreignKey: "connectionId",
  as: "connection",
});

// 数据连接和MQTT配置：一对一
DataConnection.hasOne(DataMqttConfig, {
  foreignKey: "connectionId",
  as: "mqttConfig",
});

DataMqttConfig.belongsTo(DataConnection, {
  foreignKey: "connectionId",
  as: "connection",
});

// 数据连接和MQTT订阅：一对多
DataConnection.hasMany(DataMqttSubscription, {
  foreignKey: "connectionId",
  as: "mqttSubscriptions",
});

DataMqttSubscription.belongsTo(DataConnection, {
  foreignKey: "connectionId",
  as: "connection",
});

// 工程和MQTT订阅：一对多
Project.hasMany(DataMqttSubscription, {
  foreignKey: "projectId",
  as: "mqttSubscriptions",
});

DataMqttSubscription.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 用户和MQTT订阅：创建者和更新者
User.hasMany(DataMqttSubscription, {
  foreignKey: "createdBy",
  as: "createdMqttSubscriptions",
});

User.hasMany(DataMqttSubscription, {
  foreignKey: "updatedBy",
  as: "updatedMqttSubscriptions",
});

DataMqttSubscription.belongsTo(User, {
  foreignKey: "createdBy",
  as: "creator",
});

DataMqttSubscription.belongsTo(User, {
  foreignKey: "updatedBy",
  as: "updater",
});

// MQTT订阅和MQTT变量组：一对多
DataMqttSubscription.hasMany(DataMqttTagGroup, {
  foreignKey: "subscriptionId",
  as: "tagGroups",
});

DataMqttTagGroup.belongsTo(DataMqttSubscription, {
  foreignKey: "subscriptionId",
  as: "subscription",
});

// MQTT订阅和MQTT变量：一对多
DataMqttSubscription.hasMany(DataMqttTag, {
  foreignKey: "subscriptionId",
  as: "tags",
});

DataMqttTag.belongsTo(DataMqttSubscription, {
  foreignKey: "subscriptionId",
  as: "subscription",
});

// MQTT变量组和MQTT变量：一对多
DataMqttTagGroup.hasMany(DataMqttTag, {
  foreignKey: "groupId",
  as: "tags",
});

DataMqttTag.belongsTo(DataMqttTagGroup, {
  foreignKey: "groupId",
  as: "group",
});

// 工程和MQTT变量：一对多
Project.hasMany(DataMqttTag, {
  foreignKey: "projectId",
  as: "mqttTags",
});

DataMqttTag.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 用户和MQTT变量：创建者和更新者
User.hasMany(DataMqttTag, {
  foreignKey: "createdBy",
  as: "createdMqttTags",
});

User.hasMany(DataMqttTag, {
  foreignKey: "updatedBy",
  as: "updatedMqttTags",
});

DataMqttTag.belongsTo(User, {
  foreignKey: "createdBy",
  as: "creator",
});

DataMqttTag.belongsTo(User, {
  foreignKey: "updatedBy",
  as: "updater",
});

// MQTT变量和变量值：一对一 - 已废弃，改用 Redis
// DataMqttTag.hasOne(DataMqttTagValue, {
//   foreignKey: "tagId",
//   as: "currentValue",
// });

// DataMqttTagValue.belongsTo(DataMqttTag, {
//   foreignKey: "tagId",
//   as: "tag",
// });

// 数据连接和数据查询：一对多
DataConnection.hasMany(DataQuery, {
  foreignKey: "connectionId",
  as: "queries",
});

DataQuery.belongsTo(DataConnection, {
  foreignKey: "connectionId",
  as: "connection",
});

// 工程和数据查询：一对多
Project.hasMany(DataQuery, {
  foreignKey: "projectId",
  as: "dataQueries",
});

DataQuery.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 用户和数据查询：多对多（创建者和更新者）
User.hasMany(DataQuery, {
  foreignKey: "createdBy",
  as: "createdDataQueries",
});

User.hasMany(DataQuery, {
  foreignKey: "updatedBy",
  as: "updatedDataQueries",
});

DataQuery.belongsTo(User, {
  foreignKey: "createdBy",
  as: "creator",
});

DataQuery.belongsTo(User, {
  foreignKey: "updatedBy",
  as: "updater",
});

// 数据查询和查询日志：一对多
DataQuery.hasMany(DataQueryLog, {
  foreignKey: "queryId",
  as: "logs",
});

DataQueryLog.belongsTo(DataQuery, {
  foreignKey: "queryId",
  as: "query",
});

// 数据连接和查询日志：一对多
DataConnection.hasMany(DataQueryLog, {
  foreignKey: "connectionId",
  as: "queryLogs",
});

DataQueryLog.belongsTo(DataConnection, {
  foreignKey: "connectionId",
  as: "connection",
});

// 用户和查询日志：一对多
User.hasMany(DataQueryLog, {
  foreignKey: "executedBy",
  as: "dataQueryLogs",
});

DataQueryLog.belongsTo(User, {
  foreignKey: "executedBy",
  as: "executor",
});

// ===========================================
// 设计中心模型关联关系
// ===========================================

// 工程和设计页面：一对多
Project.hasMany(DesignPage, {
  foreignKey: "projectId",
  as: "designPages",
  onDelete: "CASCADE",
});

DesignPage.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 工程和资源文件夹：一对多
Project.hasMany(DesignAssetFolder, {
  foreignKey: "projectId",
  as: "assetFolders",
  onDelete: "CASCADE",
});

DesignAssetFolder.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 资源文件夹自关联
DesignAssetFolder.hasMany(DesignAssetFolder, {
  foreignKey: "parentId",
  as: "children",
  onDelete: "CASCADE",
});

DesignAssetFolder.belongsTo(DesignAssetFolder, {
  foreignKey: "parentId",
  as: "parent",
});

// 工程和资源文件：一对多
Project.hasMany(DesignAsset, {
  foreignKey: "projectId",
  as: "assets",
  onDelete: "CASCADE",
});

DesignAsset.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 文件夹和资源文件
DesignAssetFolder.hasMany(DesignAsset, {
  foreignKey: "folderId",
  as: "assets",
});

DesignAsset.belongsTo(DesignAssetFolder, {
  foreignKey: "folderId",
  as: "folder",
});

// 用户和资源文件：上传者
User.hasMany(DesignAsset, {
  foreignKey: "uploadedBy",
  as: "uploadedAssets",
});

DesignAsset.belongsTo(User, {
  foreignKey: "uploadedBy",
  as: "uploader",
});

// 设计页面自关联：父子关系（文件夹结构）
DesignPage.hasMany(DesignPage, {
  foreignKey: "parentId",
  as: "children",
  onDelete: "RESTRICT", // 防止删除有子页面的文件夹
});

DesignPage.belongsTo(DesignPage, {
  foreignKey: "parentId",
  as: "parent",
});

// 用户和设计页面：创建者
User.hasMany(DesignPage, {
  foreignKey: "createdBy",
  as: "createdDesignPages",
});

DesignPage.belongsTo(User, {
  foreignKey: "createdBy",
  as: "creator",
});

// 用户和设计页面：更新者
User.hasMany(DesignPage, {
  foreignKey: "updatedBy",
  as: "updatedDesignPages",
});

DesignPage.belongsTo(User, {
  foreignKey: "updatedBy",
  as: "updater",
});

// 工程和数据点：一对多
Project.hasMany(DataPoint, {
  foreignKey: "projectId",
  as: "dataPoints",
});

DataPoint.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 用户和数据点：创建者和更新者
User.hasMany(DataPoint, {
  foreignKey: "createdBy",
  as: "createdDataPoints",
});

User.hasMany(DataPoint, {
  foreignKey: "updatedBy",
  as: "updatedDataPoints",
});

DataPoint.belongsTo(User, {
  foreignKey: "createdBy",
  as: "creator",
});

DataPoint.belongsTo(User, {
  foreignKey: "updatedBy",
  as: "updater",
});

// 用户和设计页面：锁定者
User.hasMany(DesignPage, {
  foreignKey: "lockedBy",
  as: "lockedDesignPages",
});

DesignPage.belongsTo(User, {
  foreignKey: "lockedBy",
  as: "locker",
});

// ===========================================
// 运维模块关联关系
// ===========================================

// 租户和节点：一对多
Tenant.hasMany(Node, {
  foreignKey: "tenantId",
  as: "nodes",
  onDelete: "CASCADE",
});

Node.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

// 用户和节点：注册申请人
User.hasMany(Node, {
  foreignKey: "registeredBy",
  as: "registeredNodes",
});

Node.belongsTo(User, {
  foreignKey: "registeredBy",
  as: "registrant",
});

// 工程和发布版本：一对多
Project.hasMany(Deployment, {
  foreignKey: "projectId",
  as: "deployments",
  onDelete: "CASCADE",
});

Deployment.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 租户和发布版本：一对多
Tenant.hasMany(Deployment, {
  foreignKey: "tenantId",
  as: "deployments",
  onDelete: "CASCADE",
});

Deployment.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

// 用户和发布版本：发布者
User.hasMany(Deployment, {
  foreignKey: "deployedBy",
  as: "deployedVersions",
});

Deployment.belongsTo(User, {
  foreignKey: "deployedBy",
  as: "deployer",
});

// 节点和节点部署：一对多
Node.hasMany(NodeDeployment, {
  foreignKey: "nodeId",
  as: "deployments",
  onDelete: "CASCADE",
});

Node.hasMany(NodeDeployment, {
  foreignKey: "nodeId",
  as: "deploymentHistory",
  onDelete: "CASCADE",
});

NodeDeployment.belongsTo(Node, {
  foreignKey: "nodeId",
  as: "node",
});

// 发布版本和节点部署：一对多
Deployment.hasMany(NodeDeployment, {
  foreignKey: "deploymentId",
  as: "nodeDeployments",
  onDelete: "CASCADE",
});

NodeDeployment.belongsTo(Deployment, {
  foreignKey: "deploymentId",
  as: "deployment",
});

// 工程和节点部署：一对多（冗余关联，便于查询）
Project.hasMany(NodeDeployment, {
  foreignKey: "projectId",
  as: "nodeDeployments",
});

NodeDeployment.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 节点部署和命令：一对多
NodeDeployment.hasMany(NodeCommand, {
  foreignKey: "deploymentId",
  as: "commands",
  onDelete: "CASCADE",
});

NodeCommand.belongsTo(NodeDeployment, {
  foreignKey: "deploymentId",
  as: "deployment",
});

// 节点和命令：一对多
Node.hasMany(NodeCommand, {
  foreignKey: "nodeId",
  as: "commands",
  onDelete: "CASCADE",
});

NodeCommand.belongsTo(Node, {
  foreignKey: "nodeId",
  as: "node",
});

// 工程和命令：一对多
Project.hasMany(NodeCommand, {
  foreignKey: "projectId",
  as: "nodeCommands",
});

NodeCommand.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

// 用户和节点部署：部署者
User.hasMany(NodeDeployment, {
  foreignKey: "deployedBy",
  as: "nodeDeployments",
});

NodeDeployment.belongsTo(User, {
  foreignKey: "deployedBy",
  as: "deployer",
});

// 节点当前工程关联
Node.belongsTo(Project, {
  foreignKey: "currentProjectId",
  as: "currentProject",
});

// 节点当前部署关联
Node.belongsTo(NodeDeployment, {
  foreignKey: "currentDeploymentId",
  as: "currentDeployment",
});

module.exports = {
  Tenant, // 租户模型
  User, // 用户模型
  Project, // 工程模型
  Log, // 日志模型
  DataConnection, // 数据连接模型
  DataRelationalConfig, // 关系库配置模型
  DataMqttConfig, // MQTT配置模型
  DataMqttSubscription, // MQTT订阅模型
  DataMqttTagGroup, // MQTT变量组模型
  DataMqttTag, // MQTT变量模型
  // DataMqttTagValue, // MQTT变量值模型 - 已废弃，改用 Redis
  DataQuery, // 数据查询模型
  DataSqlConfig, // SQL配置模型
  DataQueryLog, // 查询日志模型
  DesignPage, // 设计页面模型
  DesignAssetFolder, // 资源文件夹模型
  DesignAsset, // 资源文件模型
  DataPoint, // 数据点模型
  // 运维模块模型
  Node, // 节点模型
  NodeDeployment, // 节点部署关系模型
  Deployment, // 发布版本模型
  NodeCommand, // 节点命令模型
};
