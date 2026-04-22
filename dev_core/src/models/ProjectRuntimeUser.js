/**
 * 工程运行态用户模型
 * @description 保存单个工程内的运行态登录账号，用户名在工程维度唯一，密码只保存哈希值。
 */
module.exports = (sequelize, DataTypes) => {
  const ProjectRuntimeUser = sequelize.define(
    "ProjectRuntimeUser",
    {
      id: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        comment: "运行态用户ID",
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
      username: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: "运行态用户名",
      },
      passwordHash: {
        type: DataTypes.STRING(255),
        allowNull: false,
        comment: "密码哈希",
      },
      displayName: {
        type: DataTypes.STRING(100),
        allowNull: true,
        comment: "显示名称",
      },
      status: {
        type: DataTypes.ENUM("active", "inactive", "suspended"),
        allowNull: false,
        defaultValue: "active",
        comment: "账号状态",
      },
      lastLoginAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "最后登录时间",
      },
      lastLoginIp: {
        type: DataTypes.STRING(45),
        allowNull: true,
        comment: "最后登录IP",
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
      tableName: "project_runtime_users",
      comment: "工程运行态用户表",
      indexes: [
        {
          unique: true,
          fields: ["projectId", "username"],
        },
        {
          fields: ["projectId", "status"],
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

  return ProjectRuntimeUser;
};
