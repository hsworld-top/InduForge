/**
 * 工程运行态角色授权模型
 * @description 存储单个角色对资源/动作的 allow 或 deny 授权，后续由服务层做多角色合并。
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
          fields: ["projectId", "roleId", "resourceType", "action", "effect"],
        },
        {
          fields: ["projectId", "roleId"],
        },
        {
          fields: ["projectId", "resourceType", "action"],
        },
      ],
    },
  );

  return ProjectRoleGrant;
};
