/**
 * 工程运行态角色授权模型
 * @description 存储单个角色对资源类型、资源实例和动作的 allow / deny 授权，并保留 scopeConfig 供后续扩展。
 */
module.exports = (sequelize, DataTypes) => {
  const ProjectRoleGrant = sequelize.define(
    "ProjectRoleGrant",
    {
      id: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        comment: "授权ID",
      },
      projectId: {
        type: DataTypes.UUID,
        allowNull: false,
        references: {
          model: "projects",
          key: "id",
        },
        comment: "所属工程ID",
      },
      createdBy: {
        type: DataTypes.UUID,
        allowNull: true,
        references: {
          model: "users",
          key: "id",
        },
        comment: "创建者ID",
      },
      updatedBy: {
        type: DataTypes.UUID,
        allowNull: true,
        references: {
          model: "users",
          key: "id",
        },
        comment: "更新者ID",
      },
      roleId: {
        type: DataTypes.UUID,
        allowNull: false,
        references: {
          model: "project_roles",
          key: "id",
        },
        comment: "角色ID",
      },
      resourceType: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: "资源类型",
      },
      resourceId: {
        type: DataTypes.STRING(1000),
        allowNull: false,
        defaultValue: "*",
        comment: "资源实例ID，* 表示全部实例",
      },
      action: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: "动作名称",
      },
      effect: {
        type: DataTypes.ENUM("allow", "deny"),
        allowNull: false,
        defaultValue: "allow",
        comment: "授权效果",
      },
      scopeConfig: {
        type: DataTypes.JSON,
        allowNull: false,
        defaultValue: {},
        comment: "授权范围配置",
      },
      condition: {
        type: DataTypes.JSON,
        allowNull: true,
        comment: "条件表达式",
      },
      createdAt: {
        type: DataTypes.DATE,
        allowNull: false,
      },
      updatedAt: {
        type: DataTypes.DATE,
        allowNull: false,
      },
    },
    {
      tableName: "project_role_grants",
      comment: "工程运行态角色授权表",
      indexes: [
        {
          unique: true,
          fields: [
            "projectId",
            "roleId",
            "resourceType",
            "resourceId",
            "action",
            "effect",
          ],
        },
        {
          fields: ["projectId", "roleId"],
        },
        {
          fields: ["projectId", "resourceType", "resourceId", "action"],
        },
        {
          fields: ["createdBy"],
        },
        {
          fields: ["updatedBy"],
        },
      ],
    },
  );

  return ProjectRoleGrant;
};
