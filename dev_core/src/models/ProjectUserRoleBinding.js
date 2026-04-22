/**
 * 工程运行态用户角色绑定模型
 * @description 连接运行态用户与角色，支持同一用户绑定多个角色。
 */
module.exports = (sequelize, DataTypes) => {
  const ProjectUserRoleBinding = sequelize.define(
    "ProjectUserRoleBinding",
    {
      id: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        comment: "绑定ID",
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
      runtimeUserId: {
        type: DataTypes.UUID,
        allowNull: false,
        references: {
          model: "project_runtime_users",
          key: "id",
        },
        comment: "运行态用户ID",
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
      assignedBy: {
        type: DataTypes.UUID,
        allowNull: true,
        references: {
          model: "users",
          key: "id",
        },
        comment: "分配人ID",
      },
      assignedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "分配时间",
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
      tableName: "project_user_role_bindings",
      comment: "工程运行态用户角色绑定表",
      indexes: [
        {
          unique: true,
          fields: ["projectId", "runtimeUserId", "roleId"],
        },
        {
          fields: ["projectId", "runtimeUserId"],
        },
        {
          fields: ["projectId", "roleId"],
        },
        {
          fields: ["createdBy"],
        },
      ],
    },
  );

  return ProjectUserRoleBinding;
};
