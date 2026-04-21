const { Sequelize } = require("sequelize");
const { sequelize } = require("../config/database");

const Tenant = require("./Tenant");
const User = require("./User");
const Project = require("./Project");
const ProjectRuntimeUserFn = require("./ProjectRuntimeUser");
const ProjectRoleFn = require("./ProjectRole");
const ProjectUserRoleBindingFn = require("./ProjectUserRoleBinding");
const ProjectRoleGrantFn = require("./ProjectRoleGrant");
const Log = require("./Log");
const DesignPage = require("./DesignPage");
const DesignAssetFolder = require("./DesignAssetFolder");
const DesignAsset = require("./DesignAsset");
const NodeFn = require("./Node");
const NodeDeploymentFn = require("./NodeDeployment");
const DeploymentFn = require("./Deployment");
const NodeCommandFn = require("./NodeCommand");

// 仅保留控制面仍在使用的模型。
// dev_core 已下线本地数据域实现，Data* 模型与关联全部移除，避免启动期继续绑定旧数据域表。
const Node = NodeFn(sequelize, Sequelize.DataTypes);
const NodeDeployment = NodeDeploymentFn(sequelize, Sequelize.DataTypes);
const Deployment = DeploymentFn(sequelize, Sequelize.DataTypes);
const NodeCommand = NodeCommandFn(sequelize, Sequelize.DataTypes);
const ProjectRuntimeUser = ProjectRuntimeUserFn(sequelize, Sequelize.DataTypes);
const ProjectRole = ProjectRoleFn(sequelize, Sequelize.DataTypes);
const ProjectUserRoleBinding = ProjectUserRoleBindingFn(
  sequelize,
  Sequelize.DataTypes,
);
const ProjectRoleGrant = ProjectRoleGrantFn(sequelize, Sequelize.DataTypes);

// 租户与用户/工程/日志的基础关系。
Tenant.hasMany(User, {
  foreignKey: "tenantId",
  as: "users",
  onDelete: "CASCADE",
});
User.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

Tenant.hasMany(Project, {
  foreignKey: "tenantId",
  as: "projects",
  onDelete: "CASCADE",
});
Project.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

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

// 运行态授权模型：工程内的运行态用户、角色与授权关系。
Project.hasMany(ProjectRuntimeUser, {
  foreignKey: "projectId",
  as: "runtimeUsers",
  onDelete: "CASCADE",
});
ProjectRuntimeUser.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

Project.hasMany(ProjectRole, {
  foreignKey: "projectId",
  as: "runtimeRoles",
  onDelete: "CASCADE",
});
ProjectRole.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

Project.hasMany(ProjectUserRoleBinding, {
  foreignKey: "projectId",
  as: "runtimeRoleBindings",
  onDelete: "CASCADE",
});
ProjectUserRoleBinding.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

ProjectRuntimeUser.hasMany(ProjectUserRoleBinding, {
  foreignKey: "runtimeUserId",
  as: "roleBindings",
  onDelete: "CASCADE",
});
ProjectUserRoleBinding.belongsTo(ProjectRuntimeUser, {
  foreignKey: "runtimeUserId",
  as: "runtimeUser",
});

ProjectRole.hasMany(ProjectUserRoleBinding, {
  foreignKey: "roleId",
  as: "userBindings",
  onDelete: "CASCADE",
});
ProjectUserRoleBinding.belongsTo(ProjectRole, {
  foreignKey: "roleId",
  as: "role",
});

Project.hasMany(ProjectRoleGrant, {
  foreignKey: "projectId",
  as: "runtimeRoleGrants",
  onDelete: "CASCADE",
});
ProjectRoleGrant.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

ProjectRole.hasMany(ProjectRoleGrant, {
  foreignKey: "roleId",
  as: "grants",
  onDelete: "CASCADE",
});
ProjectRoleGrant.belongsTo(ProjectRole, {
  foreignKey: "roleId",
  as: "role",
});

Tenant.hasMany(Log, {
  foreignKey: "tenantId",
  as: "logs",
  onDelete: "CASCADE",
});
Log.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

User.hasMany(Log, {
  foreignKey: "userId",
  as: "logs",
});
Log.belongsTo(User, {
  foreignKey: "userId",
  as: "user",
});

// 设计中心关系：页面、资源目录、资源文件。
Project.hasMany(DesignPage, {
  foreignKey: "projectId",
  as: "designPages",
  onDelete: "CASCADE",
});
DesignPage.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

DesignPage.hasMany(DesignPage, {
  foreignKey: "parentId",
  as: "children",
  onDelete: "RESTRICT",
});
DesignPage.belongsTo(DesignPage, {
  foreignKey: "parentId",
  as: "parent",
});

User.hasMany(DesignPage, {
  foreignKey: "createdBy",
  as: "createdDesignPages",
});
DesignPage.belongsTo(User, {
  foreignKey: "createdBy",
  as: "creator",
});

User.hasMany(DesignPage, {
  foreignKey: "updatedBy",
  as: "updatedDesignPages",
});
DesignPage.belongsTo(User, {
  foreignKey: "updatedBy",
  as: "updater",
});

User.hasMany(DesignPage, {
  foreignKey: "lockedBy",
  as: "lockedDesignPages",
});
DesignPage.belongsTo(User, {
  foreignKey: "lockedBy",
  as: "locker",
});

Project.hasMany(DesignAssetFolder, {
  foreignKey: "projectId",
  as: "assetFolders",
  onDelete: "CASCADE",
});
DesignAssetFolder.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

DesignAssetFolder.hasMany(DesignAssetFolder, {
  foreignKey: "parentId",
  as: "children",
  onDelete: "CASCADE",
});
DesignAssetFolder.belongsTo(DesignAssetFolder, {
  foreignKey: "parentId",
  as: "parent",
});

Project.hasMany(DesignAsset, {
  foreignKey: "projectId",
  as: "assets",
  onDelete: "CASCADE",
});
DesignAsset.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

DesignAssetFolder.hasMany(DesignAsset, {
  foreignKey: "folderId",
  as: "assets",
});
DesignAsset.belongsTo(DesignAssetFolder, {
  foreignKey: "folderId",
  as: "folder",
});

User.hasMany(DesignAsset, {
  foreignKey: "uploadedBy",
  as: "uploadedAssets",
});
DesignAsset.belongsTo(User, {
  foreignKey: "uploadedBy",
  as: "uploader",
});

// 运维中心关系：节点、发布、部署、命令。
Tenant.hasMany(Node, {
  foreignKey: "tenantId",
  as: "nodes",
  onDelete: "CASCADE",
});
Node.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

User.hasMany(Node, {
  foreignKey: "registeredBy",
  as: "registeredNodes",
});
Node.belongsTo(User, {
  foreignKey: "registeredBy",
  as: "registrant",
});

Project.hasMany(Deployment, {
  foreignKey: "projectId",
  as: "deployments",
  onDelete: "CASCADE",
});
Deployment.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

Tenant.hasMany(Deployment, {
  foreignKey: "tenantId",
  as: "deployments",
  onDelete: "CASCADE",
});
Deployment.belongsTo(Tenant, {
  foreignKey: "tenantId",
  as: "tenant",
});

User.hasMany(Deployment, {
  foreignKey: "deployedBy",
  as: "deployedVersions",
});
Deployment.belongsTo(User, {
  foreignKey: "deployedBy",
  as: "deployer",
});

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

Deployment.hasMany(NodeDeployment, {
  foreignKey: "deploymentId",
  as: "nodeDeployments",
  onDelete: "CASCADE",
});
NodeDeployment.belongsTo(Deployment, {
  foreignKey: "deploymentId",
  as: "deployment",
});

Project.hasMany(NodeDeployment, {
  foreignKey: "projectId",
  as: "nodeDeployments",
});
NodeDeployment.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

NodeDeployment.hasMany(NodeCommand, {
  foreignKey: "deploymentId",
  as: "commands",
  onDelete: "CASCADE",
});
NodeCommand.belongsTo(NodeDeployment, {
  foreignKey: "deploymentId",
  as: "deployment",
});

Node.hasMany(NodeCommand, {
  foreignKey: "nodeId",
  as: "commands",
  onDelete: "CASCADE",
});
NodeCommand.belongsTo(Node, {
  foreignKey: "nodeId",
  as: "node",
});

Project.hasMany(NodeCommand, {
  foreignKey: "projectId",
  as: "nodeCommands",
});
NodeCommand.belongsTo(Project, {
  foreignKey: "projectId",
  as: "project",
});

User.hasMany(NodeDeployment, {
  foreignKey: "deployedBy",
  as: "nodeDeployments",
});
NodeDeployment.belongsTo(User, {
  foreignKey: "deployedBy",
  as: "deployer",
});

Node.belongsTo(Project, {
  foreignKey: "currentProjectId",
  as: "currentProject",
});
Node.belongsTo(NodeDeployment, {
  foreignKey: "currentDeploymentId",
  as: "currentDeployment",
});

module.exports = {
  sequelize,
  Sequelize,
  Tenant,
  User,
  Project,
  Log,
  ProjectRuntimeUser,
  ProjectRole,
  ProjectUserRoleBinding,
  ProjectRoleGrant,
  DesignPage,
  DesignAssetFolder,
  DesignAsset,
  Node,
  NodeDeployment,
  Deployment,
  NodeCommand,
};
